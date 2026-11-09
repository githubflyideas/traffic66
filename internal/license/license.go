// Package license is the single source of truth for what a traffic66
// licence says and what an installation is therefore allowed to do.
//
// A licence is license.json in the data directory: the customer, the tier,
// the quotas, the features and an Ed25519 signature over all of them. The
// private key stays with the author; the public key is compiled into
// traffic66. Nothing is ever switched off by this package: an installation
// without a licence keeps collecting, keeps its data and keeps every page
// of flow analysis. What it loses is the registered-only features and the
// clean, unwatermarked reports.
//
// Three versions of the signed message exist. v1 is what traffic66 shipped
// first (installation, customer, product, licence type, expiry). v2 adds
// the signing key's id, the tier, the update window, the quotas and the
// feature list. v3 adds who the licence is made out to — a person or an
// organisation — so the page can say "licensed to" rather than only
// "registered", and so a licence bought by one person cannot be passed off
// as a company's.
//
// An older signature can only ever authorise what its own version covers:
// a file carrying quotas or features is checked against v2 or later, and
// one naming a licensee against v3 alone, so an old signature cannot be
// replayed to grant something it never said.
//
// Verification goes against a Keyring rather than one key, because an
// offline licence cannot be withdrawn over the network: if an issuing key
// ever leaks, the only way to stop licences signed with it is a release
// that no longer accepts it. The keyring makes that a release rather than
// a redesign — it carries several keys, any of them can be marked disabled,
// the key id is signed into the licence so a leaked key cannot sign on
// another key's behalf, and an allow list of fingerprints can be switched
// on so that only the licences actually in the issuing ledger are
// accepted. With those, answering a leak costs customers an upgrade
// instead of an upgrade plus a new licence file.
package license

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// FileName is the licence inside the data directory.
const FileName = "license.json"

// FreeUsers is how many accounts an unregistered installation may have.
const FreeUsers = 3

// Licence types.
const (
	TypeTrial        = "trial"
	TypeSubscription = "subscription"
	TypePerpetual    = "perpetual"
)

// Who a licence is made out to. It is signed, so it cannot be edited into
// a file after issuing, and it is what the licence page shows beside the
// name: a licence bought by one person says so.
const (
	LicenseeIndividual   = "individual"
	LicenseeOrganization = "organization"
)

// Licensees is every licensee kind, for validation and for the issuing
// tool.
var Licensees = []string{LicenseeIndividual, LicenseeOrganization}

// Tiers.
const (
	TierTrial      = "trial"
	TierTeam       = "team"
	TierStandard   = "standard"
	TierEnterprise = "enterprise"
	TierMSP        = "msp"
)

// Features a licence can grant. Everything not listed here is free and
// stays free; these are the registered-only parts.
const (
	FeatSyslog = "syslog" // syslog output, including several targets
	// FeatDCIM gates nothing either: the asset ledger it opened is gone,
	// and the rooms-and-racks picture that replaced it is free. The name
	// stays in Features so every licence already issued verifies as signed.
	FeatDCIM = "dcim" // free since; granted by older licences
	// FeatTLS gates nothing: HTTPS is free, because the alternative is
	// passwords and session cookies crossing the network in the clear, and
	// that is not a feature to sell. The name stays, and stays in Features,
	// so every licence already issued verifies byte for byte as signed.
	FeatTLS = "tls" // free since; granted by older licences
	// FeatUsers is not consulted either: how many accounts an installation
	// may have comes from MaxUsers through UserLimit, which is the thing
	// actually enforced. Groups and per-page or per-tag permissions do not
	// exist; the comment here used to promise them.
	FeatUsers      = "users"       // see UserLimit; groups and permissions are NOT IMPLEMENTED
	FeatPurgeIface = "purge-iface" // deleting stored data by interface

	// These three are names and nothing else: no code anywhere consults
	// them, so granting one gives a customer exactly nothing. They are
	// kept because the signed-message test vectors sign the whole Features
	// list, and that file is shared with the issuing repository — removing
	// a name is a change to a signed format on both sides, which is not a
	// thing to do for a name that was never built.
	//
	// Do not issue a licence that grants them, and do not put them in a
	// price list, until the feature exists and the gate is in the code.
	FeatLDAP       = "ldap"        // NOT IMPLEMENTED: LDAP/AD sign-in
	FeatWhiteLabel = "white-label" // NOT IMPLEMENTED: reports without the traffic66 line
	FeatMultiLogo  = "multi-logo"  // NOT IMPLEMENTED: a logo per site or customer
)

// Features is every feature name, for validation and for the issuing tool.
var Features = []string{
	FeatSyslog, FeatDCIM, FeatTLS, FeatUsers,
	FeatLDAP, FeatPurgeIface, FeatWhiteLabel, FeatMultiLogo,
}

// File is license.json. A quota of 0 means no limit. Expires nil on a
// perpetual licence means it never expires; UpdatesUntil then says how long
// new releases are covered.
//
// Files written before the local trial countdown was removed carry a
// first_run date. It is ignored, as any unknown field is: nothing decides
// anything from a date the installation wrote for itself.
type File struct {
	KeyID          string   `json:"key_id,omitempty"`
	InstallationID string   `json:"installation_id,omitempty"`
	Customer       string   `json:"customer,omitempty"`
	Licensee       string   `json:"licensee,omitempty"` // LicenseeIndividual or LicenseeOrganization
	Product        string   `json:"product,omitempty"`
	License        string   `json:"license,omitempty"`
	Tier           string   `json:"tier,omitempty"`
	Expires        *string  `json:"expires,omitempty"`
	UpdatesUntil   *string  `json:"updates_until,omitempty"`
	MaxUsers       int      `json:"max_users,omitempty"`
	MaxSites       int      `json:"max_sites,omitempty"`
	MaxRacks       int      `json:"max_racks,omitempty"`
	MaxDevices     int      `json:"max_devices,omitempty"`
	Features       []string `json:"features,omitempty"`
	Note           string   `json:"note,omitempty"`
	Signature      string   `json:"signature,omitempty"`
}

var b64 = base64.StdEncoding

// Fingerprint identifies one licence in a short, quotable way: the start
// of the SHA-256 of its signature. It is what to ask a customer for when
// they report a problem, and what an allow list is made of.
func Fingerprint(f File) string {
	sum := sha256.Sum256([]byte(f.Signature))
	return hex.EncodeToString(sum[:8])
}

// KeyID names an issuing key by its public half, so the issuer cannot
// mislabel it and two keys cannot claim the same id.
func KeyID(pub ed25519.PublicKey) string {
	sum := sha256.Sum256(pub)
	return hex.EncodeToString(sum[:4])
}

// errors returned when a licence cannot be trusted.
var (
	ErrNoSignature = errors.New("license: no signature")
	ErrBadKey      = errors.New("license: public key is not an Ed25519 key")
	ErrBadField    = errors.New("license: a field contains a newline")
	ErrSignature   = errors.New("license: signature does not match")
	ErrNoKeys      = errors.New("license: no public key to check against")
	ErrUnknownKey  = errors.New("license: signed by a key this release does not know")
	ErrKeyDisabled = errors.New("license: signed by a key this release no longer accepts")
	ErrNotAllowed  = errors.New("license: not in this release's list of issued licences")
)

// CanonFeatures returns the feature list as it is signed: trimmed, without
// duplicates or empties, sorted.
func CanonFeatures(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, f := range in {
		f = strings.TrimSpace(f)
		if f == "" || seen[f] {
			continue
		}
		seen[f] = true
		out = append(out, f)
	}
	sort.Strings(out)
	return out
}

// fields are the free-text parts of a message; a newline in any of them
// would let one field pose as another, so they are rejected.
func fields(f File) []string {
	return []string{f.KeyID, f.InstallationID, f.Customer, f.Licensee, f.Product, f.License, f.Tier,
		strings.Join(CanonFeatures(f.Features), ",")}
}

func printable(f File) error {
	for _, s := range fields(f) {
		if strings.ContainsAny(s, "\n\r") {
			return ErrBadField
		}
	}
	for _, p := range []*string{f.Expires, f.UpdatesUntil} {
		if p != nil && strings.ContainsAny(*p, "\n\r") {
			return ErrBadField
		}
	}
	return nil
}

func orNull(p *string) string {
	if p == nil || *p == "" {
		return "null"
	}
	return *p
}

// Message is what the current signature covers, and what the issuing tool
// signs. It is v3: v2's fields with the licensee after the customer.
func Message(f File) []byte { return message("traffic66-license-v3", f, true) }

// MessageV2 is what traffic66 signed before a licence said who it was made
// out to. It is kept so licences issued then keep working.
func MessageV2(f File) []byte { return message("traffic66-license-v2", f, false) }

func message(head string, f File, licensee bool) []byte {
	var b strings.Builder
	b.WriteString(head)
	b.WriteByte('\n')
	parts := []string{f.KeyID, f.InstallationID, f.Customer}
	if licensee {
		parts = append(parts, f.Licensee)
	}
	parts = append(parts, f.Product, f.License, f.Tier)
	for _, s := range parts {
		b.WriteString(s)
		b.WriteByte('\n')
	}
	b.WriteString(orNull(f.Expires))
	b.WriteByte('\n')
	b.WriteString(orNull(f.UpdatesUntil))
	b.WriteByte('\n')
	fmt.Fprintf(&b, "%d\n%d\n%d\n%d\n", f.MaxUsers, f.MaxSites, f.MaxRacks, f.MaxDevices)
	b.WriteString(strings.Join(CanonFeatures(f.Features), ","))
	b.WriteByte('\n')
	return []byte(b.String())
}

// MessageV1 is what the first released version of traffic66 signed. It is
// kept so licences issued before v2 keep working.
func MessageV1(f File) []byte {
	return []byte(fmt.Sprintf("traffic66-license-v1\n%s\n%s\n%s\n%s\n%s\n",
		f.InstallationID, f.Customer, f.Product, f.License, orNull(f.Expires)))
}

// grants reports whether the file carries anything only a v2 message or
// later covers. Such a file is never accepted on a v1 signature.
func grants(f File) bool {
	return f.KeyID != "" || f.Tier != "" || f.UpdatesUntil != nil || len(f.Features) > 0 ||
		f.MaxUsers != 0 || f.MaxSites != 0 || f.MaxRacks != 0 || f.MaxDevices != 0
}

// namesLicensee reports whether the file says who it is made out to. Only
// a v3 signature covers that, so such a file is never accepted on an older
// one: otherwise a licence issued to a person could have "organization"
// written into it afterwards and still verify.
func namesLicensee(f File) bool { return f.Licensee != "" }

// Verify checks f against one key and returns the message version that
// matched. Keyring.Verify is what traffic66 uses; this is the single-key
// form, for issuing and for tests.
func Verify(pub ed25519.PublicKey, f File) (int, error) {
	if len(pub) != ed25519.PublicKeySize {
		return 0, ErrBadKey
	}
	_, ver, err := SingleKey(pub).Verify(f)
	return ver, err
}

// ParsePublicKey reads a base64 Ed25519 public key.
func ParsePublicKey(s string) (ed25519.PublicKey, error) {
	b, err := b64.DecodeString(strings.TrimSpace(s))
	if err != nil {
		return nil, fmt.Errorf("license: public key: %w", err)
	}
	if len(b) != ed25519.PublicKeySize {
		return nil, ErrBadKey
	}
	return ed25519.PublicKey(b), nil
}

// sortStrings and trimSpace keep keys.go and checker.go free of imports
// that would differ from the issuing copy of this package.
func sortStrings(s []string) { sort.Strings(s) }

func trimSpace(s string) string { return strings.TrimSpace(s) }

// Day parses a YYYY-MM-DD or RFC 3339 date. A plain date means the end of
// that day in UTC, so a licence that expires on the 31st is good all day.
func Day(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t.Add(24*time.Hour - time.Nanosecond), nil
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("license: %q is not a date", s)
	}
	return t.UTC(), nil
}
