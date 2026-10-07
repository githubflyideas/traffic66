package license

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestTrialAndLicence(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	old := publicKey
	publicKey = base64.StdEncoding.EncodeToString(pub)
	defer func() { publicKey = old }()

	dir := t.TempDir()
	c, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	st := c.State()
	if st.Kind != "trial" || st.Days != TrialDays || len(st.InstallationID) != 8 {
		t.Fatalf("first run: %+v", st)
	}
	var f File
	b, _ := os.ReadFile(c.Path)
	json.Unmarshal(b, &f)
	first, _ := time.Parse(time.RFC3339, f.FirstRun)
	if st := c.Check(first.Add(29*24*time.Hour + time.Hour)); st.Kind != "trial" || st.Days != 1 {
		t.Errorf("day 30: %+v", st)
	}
	if st := c.Check(first.Add(31 * 24 * time.Hour)); st.Kind != "trial_over" {
		t.Errorf("day 31: %+v", st)
	}
	if st := c.Check(first.Add(-48 * time.Hour)); st.Kind != "trial_over" {
		t.Errorf("clock set back: %+v", st)
	}

	write := func(f File) {
		b, _ := json.MarshalIndent(f, "", "  ")
		os.WriteFile(c.Path, b, 0o644)
	}
	sign := func(f File) File {
		f.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(priv, Message(f)))
		return f
	}
	exp := "2027-12-31"
	lic := sign(File{InstallationID: f.InstallationID, FirstRun: f.FirstRun, Customer: "ABC Corp", Product: "traffic66", License: "commercial", Expires: &exp})
	write(lic)
	now := time.Date(2027, 12, 1, 0, 0, 0, 0, time.UTC)
	if st := c.Check(now); st.Kind != "licensed" || st.Days != 31 || st.Customer != "ABC Corp" {
		t.Errorf("licensed: %+v", st)
	}
	if st := c.Check(time.Date(2028, 1, 2, 0, 0, 0, 0, time.UTC)); st.Kind != "trial_over" {
		t.Errorf("expired: %+v", st)
	}
	// a changed field breaks the signature
	tampered := lic
	tampered.Customer = "Someone else"
	write(tampered)
	if st := c.Check(now); st.Kind == "licensed" {
		t.Errorf("tampered: %+v", st)
	}
	later := "2099-01-01"
	tampered = lic
	tampered.Expires = &later
	write(tampered)
	if st := c.Check(now); st.Kind == "licensed" {
		t.Errorf("extended by hand: %+v", st)
	}
	// expires null: not licensed
	nl := sign(File{InstallationID: f.InstallationID, FirstRun: f.FirstRun, Customer: "ABC Corp", Product: "traffic66", License: "commercial"})
	write(nl)
	if st := c.Check(first.Add(time.Hour)); st.Kind != "trial" {
		t.Errorf("expires null: %+v", st)
	}
	// Open keeps an existing file
	if _, err := Open(dir); err != nil {
		t.Fatal(err)
	}
	b2, _ := os.ReadFile(filepath.Join(dir, "license.json"))
	if string(b2) == string(b) {
		t.Error("file rewritten")
	}
}
