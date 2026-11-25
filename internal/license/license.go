// Package license is the single source of truth for what a traffic66
// licence says and what an installation is therefore allowed to do.
//
// A licence is license.json in the data directory: the customer, the tier,
// the quotas, the features and an ECDSA P-521 signature over all of them.
// The private key stays with the author; the public key is compiled into
// traffic66. Nothing is ever switched off by this package: an installation
// without a licence keeps collecting, keeps its data and keeps every page
// of flow analysis. What it loses is the registered-only features and the
// clean, unwatermarked reports.
//
// The signed message is v4: the key id, the installation, the customer,
// who the licence is made out to — a person or an organisation — the
// product, the licence type, the tier, the expiry, the update window, the
// quotas and the feature list, one field per line under a
// "traffic66-license-v4" head. The signature is ECDSA over the SHA-512
// digest of that message, ASN.1 DER encoded. v1 to v3 were signed with
// Ed25519 keys; those keys and message versions were retired with the
// move to P-521, and no licence in the field depends on them.
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
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
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
// mislabel it and two keys cannot claim the same id. The id is taken from
// the key's canonical encoding, the PKIX form the key is embedded in.
func KeyID(pub *ecdsa.PublicKey) string {
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(der)
	return hex.EncodeToString(sum[:4])
}

// errors returned when a licence cannot be trusted.
var (
	ErrNoSignature = errors.New("license: no signature")
	ErrBadKey      = errors.New("license: public key is not an ECDSA P-521 key")
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

// Message is what the signature covers, and what the issuing tool signs.
// It is v4, the only message version this build verifies: a
// "traffic66-license-v4" head, then one field per line — key id,
// installation id, customer, licensee, product, licence type, tier,
// expiry, updates until, the four quotas in decimal, and the canonical
// feature list — every line newline-terminated, dates spelled "null" when
// absent. The field layout is v3's, unchanged; v4 is the move from
// Ed25519 to ECDSA P-521, and the new head is what keeps a signature made
// for one algorithm from ever being read as the other's.
func Message(f File) []byte {
	var b strings.Builder
	b.WriteString("traffic66-license-v4")
	b.WriteByte('\n')
	parts := []string{f.KeyID, f.InstallationID, f.Customer, f.Licensee, f.Product, f.License, f.Tier}
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

// Verify checks f against one key and returns the message version that
// matched. Keyring.Verify is what traffic66 uses; this is the single-key
// form, for issuing and for tests.
func Verify(pub *ecdsa.PublicKey, f File) (int, error) {
	if !isP521(pub) {
		return 0, ErrBadKey
	}
	_, ver, err := SingleKey(pub).Verify(f)
	return ver, err
}

// isP521 reports whether pub is a usable ECDSA P-521 public key: the
// right curve, coordinates in range, and the point on the curve
// (y² = x³ − 3x + b). The check is spelled out because the standard
// library's point helpers are deprecated without a public replacement
// that takes coordinates.
func isP521(pub *ecdsa.PublicKey) bool {
	if pub == nil || pub.Curve != elliptic.P521() || pub.X == nil || pub.Y == nil {
		return false
	}
	p := elliptic.P521().Params()
	if pub.X.Sign() < 0 || pub.Y.Sign() < 0 || pub.X.Cmp(p.P) >= 0 || pub.Y.Cmp(p.P) >= 0 {
		return false
	}
	lhs := new(big.Int).Mul(pub.Y, pub.Y)
	lhs.Mod(lhs, p.P)
	rhs := new(big.Int).Mul(pub.X, pub.X)
	rhs.Mul(rhs, pub.X)
	rhs.Sub(rhs, new(big.Int).Lsh(pub.X, 1))
	rhs.Sub(rhs, pub.X) // x³ − 3x
	rhs.Add(rhs, p.B)
	rhs.Mod(rhs, p.P)
	return lhs.Cmp(rhs) == 0
}

// ParsePublicKey reads a base64 ECDSA P-521 public key in its PKIX form —
// the same bytes as the PEM public key files the issuer keeps.
func ParsePublicKey(s string) (*ecdsa.PublicKey, error) {
	b, err := b64.DecodeString(strings.TrimSpace(s))
	if err != nil {
		return nil, fmt.Errorf("license: public key: %w", err)
	}
	k, err := x509.ParsePKIXPublicKey(b)
	if err != nil {
		return nil, ErrBadKey
	}
	pub, ok := k.(*ecdsa.PublicKey)
	if !ok || !isP521(pub) {
		return nil, ErrBadKey
	}
	return pub, nil
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
