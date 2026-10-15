package license

import (
	"fmt"
	"math"
	"time"
)

// Kinds of state an installation can be in.
const (
	KindUnregistered = "unregistered"
	KindLicensed     = "licensed"
	KindExpired      = "expired"
)

// State is what the rest of traffic66 asks: may this installation use a
// registered-only feature, how many accounts may it have, and do its
// reports carry the evaluation watermark.
//
// Every failure lands on the unregistered state, never on a more permissive
// one: a licence that cannot be read, cannot be verified, belongs to
// another installation or has run out grants nothing. Visibility rules
// (which group may see which pages and tags) are applied by the caller and
// must keep applying in that state, so losing a licence can never widen
// what anyone can see.
type State struct {
	Kind           string // KindUnregistered, KindLicensed, KindExpired
	Reason         string // why, when not licensed; for the footer and support
	Customer       string
	Product        string
	Type           string // TypeTrial, TypeSubscription, TypePerpetual
	Tier           string
	InstallationID string
	KeyID          string // which issuing key signed it, for support

	Days         int    // whole days left on the licence, -1 when it never expires
	Expires      string // as issued, empty when it never expires
	UpdatesUntil string

	maxUsers   int
	maxSites   int
	maxRacks   int
	maxDevices int
	features   map[string]bool
}

// Unregistered is the state of an installation with no licence.
func Unregistered(reason string) State {
	return State{Kind: KindUnregistered, Reason: reason, features: map[string]bool{}}
}

// Check turns a licence file into a State. installID is this
// installation's own number, used for licences that are pinned to one
// installation (trials are, paid licences are not, so they work in
// containers where the data directory is temporary).
func Check(ring Keyring, f File, installID string, now time.Time) State {
	key, ver, err := ring.Verify(f)
	if err != nil {
		s := Unregistered(err.Error())
		if key.Note != "" {
			s.Reason += " (" + key.Note + ")"
		}
		return s
	}
	if f.InstallationID != "" && installID != "" && f.InstallationID != installID {
		return Unregistered("license: issued for another installation")
	}

	s := State{
		Kind: KindLicensed, Customer: f.Customer, Product: f.Product,
		Type: f.License, Tier: f.Tier, InstallationID: f.InstallationID,
		KeyID: key.ID, Days: -1, features: map[string]bool{},
	}
	if ver == 1 {
		// A v1 signature covers no quotas and no features, so it can only
		// mean the accounts a registered installation always gets.
		s.maxUsers = FreeUsers
	} else {
		s.maxUsers, s.maxSites, s.maxRacks, s.maxDevices = f.MaxUsers, f.MaxSites, f.MaxRacks, f.MaxDevices
		for _, k := range CanonFeatures(f.Features) {
			s.features[k] = true
		}
	}

	if f.Expires != nil && *f.Expires != "" {
		end, err := Day(*f.Expires)
		if err != nil {
			return Unregistered(err.Error())
		}
		s.Expires = *f.Expires
		if now.After(end) {
			ex := s
			ex.Kind, ex.Reason = KindExpired, fmt.Sprintf("license: expired on %s", *f.Expires)
			ex.Days, ex.features = 0, map[string]bool{}
			ex.maxUsers, ex.maxSites, ex.maxRacks, ex.maxDevices = 0, 0, 0, 0
			return ex
		}
		s.Days = days(now, end)
	}
	if f.UpdatesUntil != nil {
		s.UpdatesUntil = *f.UpdatesUntil
	}
	return s
}

func days(now, end time.Time) int {
	d := end.Sub(now).Hours() / 24
	if d < 0 {
		return 0
	}
	return int(math.Ceil(d))
}

// Registered reports whether a licence is in force.
func (s State) Registered() bool { return s.Kind == KindLicensed }

// Watermark reports whether reports, Flow paths, Geo & networks, Findings,
// shared links and the other generated pages carry the evaluation mark. It
// depends only on whether the installation is registered, never on how old
// it is.
func (s State) Watermark() bool { return !s.Registered() }

// Allowed reports whether a registered-only feature may be used.
func (s State) Allowed(feature string) bool {
	return s.Registered() && s.features[feature]
}

// UserLimit is how many accounts this installation may have; 0 means no
// limit. An unregistered installation keeps FreeUsers accounts.
func (s State) UserLimit() int {
	if !s.Registered() {
		return FreeUsers
	}
	return s.maxUsers
}

// Quotas for the asset records: 0 means no limit, and NotAvailable means
// the asset pages are not open to this installation at all, so there is
// nothing to limit. Only a licence granting FeatDCIM has a real number
// here, which is why "no limit" and "nothing" have to be told apart.
const NotAvailable = -1

func (s State) SiteLimit() int   { return s.quota(s.maxSites) }
func (s State) RackLimit() int   { return s.quota(s.maxRacks) }
func (s State) DeviceLimit() int { return s.quota(s.maxDevices) }

func (s State) quota(v int) int {
	if !s.Allowed(FeatDCIM) {
		return NotAvailable
	}
	return v
}

// UpdatesCovered reports whether a release built at t is covered. A
// subscription covers releases while it runs; a perpetual licence covers
// them until UpdatesUntil and keeps working for ever on what it covered.
func (s State) UpdatesCovered(t time.Time) bool {
	if !s.Registered() {
		return false
	}
	if s.UpdatesUntil == "" {
		return s.Days != 0
	}
	end, err := Day(s.UpdatesUntil)
	if err != nil {
		return false
	}
	return !t.After(end)
}

// FeatureList is the granted features, sorted; for the licence page.
func (s State) FeatureList() []string {
	if !s.Registered() {
		return nil
	}
	out := make([]string, 0, len(s.features))
	for k := range s.features {
		out = append(out, k)
	}
	return CanonFeatures(out)
}

// Public is the licence state as /api/status and the web UI see it. It is
// spelled out rather than marshalled from State so that the shape the
// pages depend on is visible here, and so that what a licence grants
// travels with it: a page that must grey a tab out asks this, not the
// licence file.
type Public struct {
	Kind           string   `json:"kind"`
	Registered     bool     `json:"registered"`
	Watermark      bool     `json:"watermark"`
	Reason         string   `json:"reason,omitempty"`
	Customer       string   `json:"customer,omitempty"`
	Product        string   `json:"product,omitempty"`
	Type           string   `json:"type,omitempty"`
	Tier           string   `json:"tier,omitempty"`
	InstallationID string   `json:"installation_id,omitempty"`
	KeyID          string   `json:"key_id,omitempty"`
	Days           int      `json:"days"`
	Expires        string   `json:"expires,omitempty"`
	UpdatesUntil   string   `json:"updates_until,omitempty"`
	Users          int      `json:"users"`
	Sites          int      `json:"sites"`
	Racks          int      `json:"racks"`
	Devices        int      `json:"devices"`
	Features       []string `json:"features,omitempty"`
}

// Public returns the state for the API.
func (s State) Public() Public {
	return Public{
		Kind: s.Kind, Registered: s.Registered(), Watermark: s.Watermark(),
		Reason: s.Reason, Customer: s.Customer, Product: s.Product,
		Type: s.Type, Tier: s.Tier, InstallationID: s.InstallationID,
		KeyID: s.KeyID, Days: s.Days, Expires: s.Expires,
		UpdatesUntil: s.UpdatesUntil, Users: s.UserLimit(),
		Sites: s.SiteLimit(), Racks: s.RackLimit(), Devices: s.DeviceLimit(),
		Features: s.FeatureList(),
	}
}
