package license

import (
	"crypto/elliptic"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestOpenGivesAnInstallationNumber(t *testing.T) {
	dir := t.TempDir()
	c, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Install) != 32 {
		t.Fatalf("installation number %q", c.Install)
	}
	if _, err := hex.DecodeString(c.Install); err != nil {
		t.Fatalf("installation number %q is not hex: %v", c.Install, err)
	}
	if c.State().Registered() {
		t.Fatal("a fresh data directory is not registered")
	}
	if !c.State().Watermark() {
		t.Fatal("and its reports are marked")
	}
	if c.State().InstallationID != c.Install {
		t.Fatal("the state should carry the installation number, for support")
	}

	// The number must not change between runs.
	again, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if again.Install != c.Install {
		t.Fatalf("the number changed: %s then %s", c.Install, again.Install)
	}
}

// Installations made before the number had its own file keep the number
// they already had, which lived inside license.json — and the first_run
// date those files also carry is ignored rather than refused, because the
// local trial countdown it fed is gone.
func TestInstallationNumberIsMigrated(t *testing.T) {
	dir := t.TempDir()
	old := `{"installation_id":"12345678","first_run":"2026-01-01T00:00:00Z"}`
	if err := os.WriteFile(InData(dir), []byte(old), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if c.Install != "12345678" {
		t.Fatalf("got %q, want the number it already had", c.Install)
	}
	if _, err := os.Stat(filepath.Join(dir, InstallationFile)); err != nil {
		t.Fatal("the number should now have its own file, which installing a licence does not replace")
	}
}

// A licence dropped into the data directory takes effect on the next
// check, without a restart — and replacing license.json must not cost the
// installation its number.
func TestCheckPicksUpALicence(t *testing.T) {
	dir := t.TempDir()
	c, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	id := c.Install

	priv, pub := testKey(t)
	c.Ring = SingleKey(pub) // this build does not ship the test key

	f := File{
		KeyID: KeyID(pub), Customer: "Example KK", Product: "traffic66",
		License: TypeSubscription, Tier: TierStandard, Expires: str("2027-10-31"),
		MaxUsers: 25, Features: []string{FeatSyslog, FeatDCIM},
	}
	f.Signature = signMsg(t, priv, Message(f))
	if err := Save(c.Path, f); err != nil {
		t.Fatal(err)
	}

	st := c.Check(time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC))
	if !st.Registered() {
		t.Fatalf("the licence was not picked up: %s", st.Reason)
	}
	if st.Watermark() {
		t.Fatal("a registered installation must not watermark")
	}
	if st.InstallationID != id {
		t.Fatalf("the installation number changed with the licence: %s then %s", id, st.InstallationID)
	}
	if st.UserLimit() != 25 || !st.Allowed(FeatDCIM) {
		t.Fatalf("the licence terms did not come through: %+v", st.Public())
	}
}

// The keys this build ships must be usable, or no licence could ever
// verify.
func TestReleaseKeysParse(t *testing.T) {
	r := Release()
	if len(r.Keys) != len(releaseKeys) {
		t.Fatalf("got %d usable keys, want %d — a malformed key in keys.go", len(r.Keys), len(releaseKeys))
	}
	seen := map[string]bool{}
	for _, k := range r.Keys {
		if !isP521(k.Pub) || k.Pub.Curve != elliptic.P521() {
			t.Fatalf("key %s is not an ECDSA P-521 key", k.ID)
		}
		if seen[k.ID] {
			t.Fatalf("key %s appears twice", k.ID)
		}
		seen[k.ID] = true
	}
	if r.Keys[0].Disabled {
		t.Fatal("the first key is the one in use and must not be disabled")
	}
	if len(r.Allow) != 0 {
		t.Fatal("a normal release carries no allow list; see keys.go")
	}
}
