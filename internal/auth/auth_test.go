package auth

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetLoadCheck(t *testing.T) {
	dir := t.TempDir()
	if err := Set(dir, "admin", "first"); err != nil {
		t.Fatal(err)
	}
	if err := Set(dir, "ops", "other"); err != nil {
		t.Fatal(err)
	}
	if err := Set(dir, "admin", "second"); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(dir, FileName))
	if strings.Contains(string(b), "second") || strings.Contains(string(b), "first") {
		t.Fatal("password stored in clear text")
	}
	if fi, _ := os.Stat(filepath.Join(dir, FileName)); fi.Mode().Perm()&0o077 != 0 && os.PathSeparator == '/' {
		t.Errorf("file mode %v, want owner only", fi.Mode().Perm())
	}
	es, err := Load(dir)
	if err != nil || len(es) != 2 {
		t.Fatalf("Load: %v %v", es, err)
	}
	c := NewChecker(nil, es)
	for _, tc := range []struct {
		u, p string
		ok   bool
	}{{"admin", "second", true}, {"admin", "second", true}, {"admin", "first", false}, {"ops", "other", true}, {"nobody", "x", false}, {"admin", "", false}} {
		if got := c.Check(tc.u, tc.p); got != tc.ok {
			t.Errorf("Check(%q, %q) = %v", tc.u, tc.p, got)
		}
	}
}

// TestUnknownUserCostsTheSameWork: a check for a user that does not
// exist runs the same PBKDF2 verification a wrong password for a real
// user runs — against the dummy hash — so the time a login takes does
// not reveal whether the account exists.
func TestUnknownUserCostsTheSameWork(t *testing.T) {
	c := NewChecker(nil, []Entry{{User: "admin", Hash: Hash("right")}})
	dummies := 0
	orig := verifyDummy
	verifyDummy = func(pw string) bool { dummies++; return Verify(dummyHash, pw) }
	defer func() { verifyDummy = orig }()

	if c.Check("nosuchuser", "whatever") {
		t.Fatal("an unknown user checked out")
	}
	if dummies != 1 {
		t.Fatalf("unknown user: dummy verification ran %d times, want 1", dummies)
	}
	if c.Check("admin", "wrong") {
		t.Fatal("a wrong password checked out")
	}
	if dummies != 1 {
		t.Fatalf("known user, wrong password: dummy verification ran %d times total, want still 1", dummies)
	}
	// Both failures are the same answer: false, nothing else to tell
	// them apart.
	if c.Check("nosuchuser", "whatever") != c.Check("admin", "wrong") {
		t.Fatal("unknown user and wrong password answered differently")
	}
	// The dummy hash is a real hash at the full cost: a verification
	// against it must not simply fail fast on a malformed string.
	if Iterations(dummyHash) != iterations {
		t.Fatalf("dummy hash has %d iterations, want %d", Iterations(dummyHash), iterations)
	}
}

func TestPlainOverrides(t *testing.T) {
	c := NewChecker(map[string]string{"admin": "flag"}, []Entry{{"admin", Hash("file")}})
	if !c.Check("admin", "flag") || c.Check("admin", "file") {
		t.Fatal("a password given at start should replace the stored one")
	}
}

func TestLoadMissing(t *testing.T) {
	es, err := Load(t.TempDir())
	if es != nil || err != nil {
		t.Fatal(es, err)
	}
}

func TestFileCheckerSeesChanges(t *testing.T) {
	dir := t.TempDir()
	Set(dir, "admin", "one")
	fc := NewFileChecker(dir, nil)
	if !fc.Check("admin", "one") {
		t.Fatal("stored password rejected")
	}
	os.Remove(filepath.Join(dir, FileName))
	Set(dir, "admin", "two")
	if fc.Check("admin", "one") || !fc.Check("admin", "two") {
		t.Fatal("change of the password file not picked up")
	}
}

func TestDelete(t *testing.T) {
	dir := t.TempDir()
	Set(dir, "admin", "a")
	Set(dir, "alice", "b")
	if err := Delete(dir, "nobody"); err == nil {
		t.Fatal("deleting an unknown user succeeded")
	}
	if err := Delete(dir, "alice"); err != nil {
		t.Fatal(err)
	}
	if err := Delete(dir, "admin"); err != ErrLastUser {
		t.Fatalf("deleting the last user: %v", err)
	}
	es, _ := Load(dir)
	if len(es) != 1 || es[0].User != "admin" || !NewChecker(nil, es).Check("admin", "a") {
		t.Fatalf("%+v", es)
	}
}
