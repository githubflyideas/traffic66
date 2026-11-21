package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// TokensFileName is the API token file inside the data directory.
const TokensFileName = "tokens.json"

// Token is one API token: a program's credential, standing in for a
// user with a scope — "read" may only look, "write" may do everything
// the user may. Only the secret's hash is stored; the secret itself is
// shown once, when the token is made, and never again.
type Token struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	User     string    `json:"user"`
	Scope    string    `json:"scope"` // "read" or "write"
	Hash     string    `json:"hash"`  // hex SHA-256 of the secret
	Created  time.Time `json:"created"`
	LastUsed time.Time `json:"last_used,omitempty"`
}

// TokenHash is what the token file stores of a secret.
func TokenHash(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}

// Matches reports whether secret is this token's secret.
func (t Token) Matches(secret string) bool {
	return subtle.ConstantTimeCompare([]byte(t.Hash), []byte(TokenHash(secret))) == 1
}

var tokenMu sync.Mutex

// NewToken makes a token for user and returns it with its secret — the
// one moment the secret exists outside the caller's hands.
func NewToken(user, name, scope string) (Token, string, error) {
	if scope != "read" && scope != "write" {
		return Token{}, "", fmt.Errorf("bad scope %q", scope)
	}
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return Token{}, "", err
	}
	secret := "t66_" + base64.RawURLEncoding.EncodeToString(raw)
	id := make([]byte, 6)
	if _, err := rand.Read(id); err != nil {
		return Token{}, "", err
	}
	return Token{
		ID: hex.EncodeToString(id), Name: name, User: user, Scope: scope,
		Hash: TokenHash(secret), Created: time.Now().UTC(),
	}, secret, nil
}

// LoadTokens reads the token file in dir. It returns nil and no error
// when the file does not exist.
func LoadTokens(dir string) ([]Token, error) {
	b, err := os.ReadFile(filepath.Join(dir, TokensFileName))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var f struct {
		Tokens []Token `json:"tokens"`
	}
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, fmt.Errorf("%s: %w", TokensFileName, err)
	}
	return f.Tokens, nil
}

// SaveTokens replaces the token file in dir, atomically and readable
// only by the owner: the hashes in it are password-equivalent for
// whoever can crack one, and there is no reason for anyone else to read
// even those.
func SaveTokens(dir string, toks []Token) error {
	tokenMu.Lock()
	defer tokenMu.Unlock()
	defer lockDir(dir)()
	b, err := json.MarshalIndent(struct {
		Tokens []Token `json:"tokens"`
	}{toks}, "", "  ")
	if err != nil {
		return err
	}
	f, err := os.CreateTemp(dir, TokensFileName+".*.tmp")
	if err != nil {
		return err
	}
	tmp := f.Name()
	_, err = f.Write(append(b, '\n'))
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	if err == nil {
		err = os.Chmod(tmp, 0o600)
	}
	if err == nil {
		err = os.Rename(tmp, filepath.Join(dir, TokensFileName))
	}
	if err != nil {
		os.Remove(tmp)
	}
	return err
}
