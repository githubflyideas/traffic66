package auth

import (
	"crypto/pbkdf2"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Tokens are stored as hashes only, the file is private, and the secret
// matches exactly one token.
func TestTokenRoundTrip(t *testing.T) {
	dir := t.TempDir()
	tok, secret, err := NewToken("admin", "zabbix", "read")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(secret, "t66_") {
		t.Fatalf("secret shape: %q", secret[:8])
	}
	if err := SaveTokens(dir, []Token{tok}); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(filepath.Join(dir, TokensFileName))
	if err != nil {
		t.Fatal(err)
	}
	// Windows does not carry Unix permission bits (a plain file reports
	// 0666 there); the 0600 the save asked for is a Unix property.
	if runtime.GOOS != "windows" && fi.Mode().Perm() != 0o600 {
		t.Fatalf("tokens file mode %v, want 0600", fi.Mode().Perm())
	}
	b, _ := os.ReadFile(filepath.Join(dir, TokensFileName))
	if strings.Contains(string(b), secret) {
		t.Fatal("the secret itself is stored")
	}
	got, err := LoadTokens(dir)
	if err != nil || len(got) != 1 {
		t.Fatalf("load: %v, %d tokens", err, len(got))
	}
	if !got[0].Matches(secret) {
		t.Fatal("the saved token does not match its secret")
	}
	if got[0].Matches(secret + "x") {
		t.Fatal("a longer secret matches")
	}
	if _, _, err := NewToken("admin", "x", "root"); err == nil {
		t.Fatal("an unknown scope was accepted")
	}
}

// A password hash made at the old iteration count still verifies, and
// reports its count so a successful sign-in can remake it stronger.
func TestIterationsAndUpgrade(t *testing.T) {
	salt := []byte("0123456789abcdef")
	key, err := pbkdf2.Key(sha256.New, "pw", salt, 210000, 32)
	if err != nil {
		t.Fatal(err)
	}
	old := fmt.Sprintf("pbkdf2-sha256$210000$%s$%s", b64.EncodeToString(salt), b64.EncodeToString(key))
	if got := Iterations(old); got != 210000 {
		t.Fatalf("Iterations(old) = %d", got)
	}
	if !Verify(old, "pw") {
		t.Fatal("an old-count hash no longer verifies")
	}
	if Iterations("not-a-hash") != 0 {
		t.Fatal("a foreign string reports iterations")
	}
	// and a sign-in through the file checker remakes it at the new count
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte("# traffic66 login\n# admin: admin\nadmin:"+old+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	fc := NewFileChecker(dir, nil)
	if !fc.Check("admin", "pw") {
		t.Fatal("sign-in with the old hash failed")
	}
	es, err := Load(dir)
	if err != nil || len(es) != 1 {
		t.Fatalf("reload: %v, %d entries", err, len(es))
	}
	if got := Iterations(es[0].Hash); got != iterations {
		t.Fatalf("after sign-in the hash is at %d, want %d", got, iterations)
	}
	if !Verify(es[0].Hash, "pw") {
		t.Fatal("the remade hash does not verify")
	}
}
