package license

import (
	"testing"
	"time"
)

var now = time.Date(2026, 10, 14, 12, 0, 0, 0, time.UTC)

func TestUnregistered(t *testing.T) {
	_, pub := testKey(t)
	s := Check(SingleKey(pub), File{}, "7f3a1c9e", now)
	if s.Registered() {
		t.Fatal("an empty licence registered the installation")
	}
	if !s.Watermark() {
		t.Fatal("an unregistered installation must watermark its reports")
	}
	if s.UserLimit() != FreeUsers {
		t.Fatalf("free accounts: got %d, want %d", s.UserLimit(), FreeUsers)
	}
	for _, f := range Features {
		if s.Allowed(f) {
			t.Errorf("%s was allowed without a licence", f)
		}
	}
	if s.SiteLimit() != NotAvailable || s.RackLimit() != NotAvailable || s.DeviceLimit() != NotAvailable {
		t.Fatal("without a licence the asset pages are not available at all, which must not read as unlimited")
	}
}

func TestLicensed(t *testing.T) {
	priv, pub := testKey(t)
	s := Check(SingleKey(pub), sign(t, priv, paid("2027-10-31")), "7f3a1c9e", now)
	if !s.Registered() {
		t.Fatalf("a good licence did not register: %s", s.Reason)
	}
	if s.Watermark() {
		t.Fatal("a registered installation must not watermark")
	}
	if s.UserLimit() != 25 || s.SiteLimit() != 3 || s.RackLimit() != 50 || s.DeviceLimit() != 500 {
		t.Fatalf("quotas not taken from the licence: %+v", s)
	}
	for _, f := range []string{FeatSyslog, FeatDCIM, FeatTLS, FeatUsers} {
		if !s.Allowed(f) {
			t.Errorf("%s was not granted", f)
		}
	}
	for _, f := range []string{FeatLDAP, FeatWhiteLabel, FeatMultiLogo, FeatPurgeIface} {
		if s.Allowed(f) {
			t.Errorf("%s was granted but not in the licence", f)
		}
	}
	if s.Days < 1 {
		t.Fatalf("days left: got %d", s.Days)
	}
}

// Running out must land on the unregistered side of every question, never
// on a more permissive one.
func TestExpiredFailsClosed(t *testing.T) {
	priv, pub := testKey(t)
	s := Check(SingleKey(pub), sign(t, priv, paid("2026-10-13")), "7f3a1c9e", now)
	if s.Kind != KindExpired {
		t.Fatalf("got %s, want %s", s.Kind, KindExpired)
	}
	if s.Registered() {
		t.Fatal("an expired licence must not count as registered")
	}
	if !s.Watermark() {
		t.Fatal("an expired licence must bring the watermark back")
	}
	if s.UserLimit() != FreeUsers {
		t.Fatalf("got %d accounts, want %d", s.UserLimit(), FreeUsers)
	}
	for _, f := range Features {
		if s.Allowed(f) {
			t.Errorf("%s survived expiry", f)
		}
	}
	if s.SiteLimit() != NotAvailable || s.DeviceLimit() != NotAvailable {
		t.Fatal("an expired licence must not leave the asset quotas reading as unlimited")
	}
	if s.UpdatesCovered(now) {
		t.Fatal("an expired licence covers no releases")
	}
	if s.Customer == "" {
		t.Fatal("the customer should still be shown, for the footer and support")
	}
}

// The last day of a licence is still a licensed day.
func TestExpiresToday(t *testing.T) {
	priv, pub := testKey(t)
	s := Check(SingleKey(pub), sign(t, priv, paid("2026-10-14")), "", now)
	if !s.Registered() {
		t.Fatalf("the last day should still be licensed: %s", s.Reason)
	}
}

func TestPinnedToInstallation(t *testing.T) {
	priv, pub := testKey(t)
	trial := File{
		InstallationID: "7f3a1c9e", Customer: "Example KK", Product: "traffic66",
		License: TypeTrial, Tier: TierTrial, Expires: str("2026-11-13"),
		MaxUsers: 3, Features: Features,
	}
	f := sign(t, priv, trial)
	if s := Check(SingleKey(pub), f, "7f3a1c9e", now); !s.Registered() {
		t.Fatalf("the right installation was refused: %s", s.Reason)
	}
	s := Check(SingleKey(pub), f, "ffffffff", now)
	if s.Registered() {
		t.Fatal("a trial licence worked on another installation")
	}
	if !s.Watermark() {
		t.Fatal("a refused licence must watermark")
	}
}

// A paid licence carries no installation number, so it keeps working where
// the data directory is temporary.
func TestPaidIsNotPinned(t *testing.T) {
	priv, pub := testKey(t)
	f := sign(t, priv, paid("2027-10-31"))
	for _, id := range []string{"7f3a1c9e", "ffffffff", ""} {
		if s := Check(SingleKey(pub), f, id, now); !s.Registered() {
			t.Fatalf("installation %q: %s", id, s.Reason)
		}
	}
}

func TestPerpetual(t *testing.T) {
	priv, pub := testKey(t)
	f := sign(t, priv, File{
		Customer: "Perpetual GmbH", Product: "traffic66", License: TypePerpetual,
		Tier: TierEnterprise, UpdatesUntil: str("2027-10-31"),
		Features: Features,
	})
	s := Check(SingleKey(pub), f, "7f3a1c9e", time.Date(2031, 1, 1, 0, 0, 0, 0, time.UTC))
	if !s.Registered() {
		t.Fatalf("a perpetual licence stopped working: %s", s.Reason)
	}
	if s.Days != -1 {
		t.Fatalf("days left: got %d, want -1 (never expires)", s.Days)
	}
	if s.UserLimit() != 0 {
		t.Fatalf("enterprise accounts should be unlimited, got %d", s.UserLimit())
	}
	if s.DeviceLimit() != 0 {
		t.Fatalf("enterprise devices should be unlimited, got %d", s.DeviceLimit())
	}
	if !s.UpdatesCovered(time.Date(2027, 10, 31, 0, 0, 0, 0, time.UTC)) {
		t.Fatal("a release inside the update window is covered")
	}
	if s.UpdatesCovered(time.Date(2027, 11, 1, 12, 0, 0, 0, time.UTC)) {
		t.Fatal("a release after the update window is not covered")
	}
}

// A v1 licence registers the installation but grants no quotas or features.
func TestLegacyGrantsNothingExtra(t *testing.T) {
	_, pub := testKey(t)
	v := vectorNamed(t, "legacy v1")
	f := v.file
	f.Signature = v.sig
	s := Check(SingleKey(pub), f, "7f3a1c9e", now)
	if !s.Registered() {
		t.Fatalf("a v1 licence should still register: %s", s.Reason)
	}
	if s.Watermark() {
		t.Fatal("a v1 licence is registered, so no watermark")
	}
	if s.UserLimit() != FreeUsers {
		t.Fatalf("got %d accounts, want %d", s.UserLimit(), FreeUsers)
	}
	for _, feat := range Features {
		if s.Allowed(feat) {
			t.Errorf("v1 granted %s", feat)
		}
	}
	if s.DeviceLimit() != NotAvailable {
		t.Fatal("a v1 licence does not open the asset pages")
	}
}

func TestReasonIsSaid(t *testing.T) {
	_, pub := testKey(t)
	f := paid("2027-10-31")
	f.Signature = "not base64 at all!!"
	if s := Check(SingleKey(pub), f, "", now); s.Reason == "" {
		t.Fatal("an unusable licence must say why, for the footer and support")
	}
}

// The API shape is what the pages depend on, so it is checked here rather
// than discovered by a page that greys the wrong tab out.
func TestPublicShape(t *testing.T) {
	priv, pub := testKey(t)
	p := Check(SingleKey(pub), sign(t, priv, paid("2027-10-31")), "", now).Public()
	if !p.Registered || p.Watermark {
		t.Fatalf("a licensed installation is registered and unmarked, got %+v", p)
	}
	if p.Users != 25 || p.Devices != 500 {
		t.Fatalf("quotas not carried: %+v", p)
	}
	if len(p.Features) == 0 || p.KeyID == "" {
		t.Fatalf("features and the signing key should travel with the state: %+v", p)
	}

	u := Unregistered("no licence").Public()
	if u.Registered || !u.Watermark {
		t.Fatal("an unregistered installation is unregistered and marked")
	}
	if u.Users != FreeUsers {
		t.Fatalf("got %d accounts, want %d", u.Users, FreeUsers)
	}
	if u.Devices != NotAvailable {
		t.Fatal("the asset pages are not available, which must not read as unlimited")
	}
	if len(u.Features) != 0 {
		t.Fatal("no features without a licence")
	}
}

// The licensee is signed, so it cannot be written into a file after it was
// issued: a licence sold to one person must not be able to call itself a
// company's by editing a line.
func TestLicenseeIsSigned(t *testing.T) {
	_, pub := testKey(t)
	v := vectorNamed(t, "individual")
	f := v.file
	f.Signature = v.sig
	s := Check(SingleKey(pub), f, "7f3a1c9e", now)
	if !s.Registered() || s.Licensee != LicenseeIndividual || s.Customer != "Ada Lovelace" {
		t.Fatalf("licensed to %q as %q: %s", s.Customer, s.Licensee, s.Reason)
	}
	if p := s.Public(); p.Licensee != LicenseeIndividual {
		t.Fatalf("the API says %q", p.Licensee)
	}
	// the same file, now claiming to be a company's
	f.Licensee = LicenseeOrganization
	if s := Check(SingleKey(pub), f, "7f3a1c9e", now); s.Registered() {
		t.Fatal("an edited licensee was accepted")
	}
	// and a licence issued before v3 carries none, rather than a guess
	v2 := vectorNamed(t, "subscription")
	f2 := v2.file
	f2.Signature = v2.sig
	s2 := Check(SingleKey(pub), f2, "", now)
	if !s2.Registered() || s2.Licensee != "" {
		t.Fatalf("a v2 licence says %q: %s", s2.Licensee, s2.Reason)
	}
	// nor can one be added to it
	f2.Licensee = LicenseeOrganization
	if s := Check(SingleKey(pub), f2, "", now); s.Registered() {
		t.Fatal("a licensee added to a v2 licence was accepted")
	}
}
