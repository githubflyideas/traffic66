package license

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"testing"
	"time"
)

func sign(t *testing.T, priv *ecdsa.PrivateKey, f File) File {
	t.Helper()
	f.Signature = signMsg(t, priv, Message(f))
	return f
}

func paid(expires string) File {
	return File{
		Customer: "Example KK", Product: "traffic66", License: TypeSubscription,
		Tier: TierStandard, Expires: str(expires),
		MaxUsers: 25, MaxSites: 3, MaxRacks: 50, MaxDevices: 500,
		Features: []string{FeatSyslog, FeatDCIM, FeatTLS, FeatUsers},
	}
}

func TestTamper(t *testing.T) {
	priv, pub := testKey(t)
	good := sign(t, priv, paid("2027-10-31"))
	if _, err := Verify(pub, good); err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		what string
		edit func(*File)
	}{
		{"customer", func(f *File) { f.Customer = "Someone Else" }},
		{"licensee", func(f *File) { f.Licensee = LicenseeOrganization }},
		{"tier", func(f *File) { f.Tier = TierEnterprise }},
		{"users", func(f *File) { f.MaxUsers = 1000 }},
		{"sites", func(f *File) { f.MaxSites = 99 }},
		{"racks", func(f *File) { f.MaxRacks = 99 }},
		{"devices", func(f *File) { f.MaxDevices = 99 }},
		{"expiry", func(f *File) { f.Expires = str("2099-01-01") }},
		{"updates", func(f *File) { f.UpdatesUntil = str("2099-01-01") }},
		{"a feature added", func(f *File) { f.Features = append(f.Features, FeatWhiteLabel) }},
		{"installation pinned", func(f *File) { f.InstallationID = "deadbeef" }},
		{"product", func(f *File) { f.Product = "traffic66-ultra" }},
		{"licence type", func(f *File) { f.License = TypePerpetual }},
	} {
		bad := good
		bad.Features = append([]string(nil), good.Features...)
		c.edit(&bad)
		if _, err := Verify(pub, bad); err == nil {
			t.Errorf("%s: a changed licence verified", c.what)
		}
	}
}

// A note is not signed: it is for the issuer's own records and changing it
// must not invalidate a licence.
func TestNoteNotSigned(t *testing.T) {
	priv, pub := testKey(t)
	f := sign(t, priv, paid("2027-10-31"))
	f.Note = "renewal of #12"
	if _, err := Verify(pub, f); err != nil {
		t.Fatalf("a note invalidated the licence: %v", err)
	}
}

// The retired message versions stay retired: a signature over the same
// fields under a v3 head — the Ed25519-era format — must not verify, so
// nothing signed for the old algorithm can be replayed into this build.
func TestRetiredMessageVersionRefused(t *testing.T) {
	priv, pub := testKey(t)
	f := paid("2027-10-31")
	old := bytes.Replace(Message(f), []byte("traffic66-license-v4"), []byte("traffic66-license-v3"), 1)
	f.Signature = signMsg(t, priv, old)
	if _, err := Verify(pub, f); err == nil {
		t.Fatal("a signature over a retired message version verified")
	}
}

func TestNewlineRejected(t *testing.T) {
	priv, pub := testKey(t)
	f := paid("2027-10-31")
	f.Customer = "Example KK\ntier"
	f.Signature = signMsg(t, priv, Message(f))
	if _, err := Verify(pub, f); err != ErrBadField {
		t.Fatalf("got %v, want ErrBadField", err)
	}
}

func TestWrongKey(t *testing.T) {
	priv, _ := testKey(t)
	_, otherKey := pair(t)
	f := sign(t, priv, paid("2027-10-31"))
	if _, err := Verify(otherKey.Pub, f); err == nil {
		t.Fatal("another key verified the licence")
	}
	p256, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Verify(&p256.PublicKey, f); err != ErrBadKey {
		t.Fatalf("got %v, want ErrBadKey", err)
	}
	if _, err := Verify(nil, f); err != ErrBadKey {
		t.Fatalf("got %v, want ErrBadKey", err)
	}
	f.Signature = ""
	if _, err := Verify(otherKey.Pub, f); err != ErrNoSignature {
		t.Fatalf("got %v, want ErrNoSignature", err)
	}
}

func TestCanonFeatures(t *testing.T) {
	got := CanonFeatures([]string{" syslog ", "dcim", "syslog", "", "tls"})
	want := []string{"dcim", "syslog", "tls"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestDayIsEndOfDay(t *testing.T) {
	end, err := Day("2026-10-31")
	if err != nil {
		t.Fatal(err)
	}
	if !end.After(time.Date(2026, 10, 31, 23, 59, 0, 0, time.UTC)) {
		t.Fatalf("a plain date should last all day, got %s", end)
	}
	if end.After(time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("a plain date should not reach the next day, got %s", end)
	}
	if _, err := Day("not a date"); err == nil {
		t.Fatal("accepted a non-date")
	}
}
