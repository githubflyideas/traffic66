// Package auth stores the login password as a salted PBKDF2 hash in the
// data directory, so a password survives restarts without being kept in
// clear text or passed on the command line.
package auth

import (
	"bufio"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
)

// FileName is the password file inside the data directory.
const FileName = "password"

const iterations = 210000

var b64 = base64.RawStdEncoding

// Hash returns "pbkdf2-sha256$<iterations>$<salt>$<key>" for pw.
func Hash(pw string) string {
	salt := make([]byte, 16)
	rand.Read(salt)
	key, _ := pbkdf2.Key(sha256.New, pw, salt, iterations, 32)
	return fmt.Sprintf("pbkdf2-sha256$%d$%s$%s", iterations, b64.EncodeToString(salt), b64.EncodeToString(key))
}

// Verify reports whether pw matches a hash made by Hash.
func Verify(hash, pw string) bool {
	parts := strings.Split(hash, "$")
	if len(parts) != 4 || parts[0] != "pbkdf2-sha256" {
		return false
	}
	iter, err := strconv.Atoi(parts[1])
	if err != nil || iter < 1 {
		return false
	}
	salt, err1 := b64.DecodeString(parts[2])
	want, err2 := b64.DecodeString(parts[3])
	if err1 != nil || err2 != nil || len(want) == 0 {
		return false
	}
	got, err := pbkdf2.Key(sha256.New, pw, salt, iter, len(want))
	return err == nil && subtle.ConstantTimeCompare(got, want) == 1
}

// Generate returns a random password.
func Generate() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// Entry is one line of the password file: "user:hash".
type Entry struct{ User, Hash string }

// Load reads the password file in dir. It returns nil and no error when
// the file does not exist.
func Load(dir string) ([]Entry, error) {
	f, err := os.Open(filepath.Join(dir, FileName))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []Entry
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		u, h, ok := strings.Cut(line, ":")
		if !ok || u == "" {
			return nil, fmt.Errorf("%s: malformed line", FileName)
		}
		out = append(out, Entry{u, h})
	}
	return out, sc.Err()
}

// Set stores pw for user in dir, replacing any earlier password of that user.
func Set(dir, user, pw string) error {
	if user == "" || strings.ContainsAny(user, ":\r\n") {
		return fmt.Errorf("invalid user name %q", user)
	}
	entries, err := Load(dir)
	if err != nil {
		return err
	}
	var out []Entry
	for _, e := range entries {
		if e.User != user {
			out = append(out, e)
		}
	}
	return write(dir, append(out, Entry{user, Hash(pw)}))
}

// ErrLastUser is returned when deleting the only remaining user.
var ErrLastUser = errors.New("this is the only user; add another user first")

// Delete removes user from the password file in dir.
func Delete(dir, user string) error {
	entries, err := Load(dir)
	if err != nil {
		return err
	}
	var out []Entry
	for _, e := range entries {
		if e.User != user {
			out = append(out, e)
		}
	}
	if len(out) == len(entries) {
		return fmt.Errorf("no user %q", user)
	}
	if len(out) == 0 {
		return ErrLastUser
	}
	return write(dir, out)
}

func write(dir string, entries []Entry) error {
	var b strings.Builder
	b.WriteString("# traffic66 login, one user per line; change with: traffic66 passwd\n")
	for _, e := range entries {
		fmt.Fprintf(&b, "%s:%s\n", e.User, e.Hash)
	}
	tmp := filepath.Join(dir, FileName+".tmp")
	if err := os.WriteFile(tmp, []byte(b.String()), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(dir, FileName))
}

// Checker verifies logins against fixed passwords or stored hashes. A
// verified hash check is remembered so that later requests with the same
// credentials do not pay for PBKDF2 again.
type Checker struct {
	plain  map[string]string
	hashes map[string]string
	mu     sync.Mutex
	ok     map[[32]byte]bool
}

// NewChecker accepts plain (user → password) and hashed entries.
func NewChecker(plain map[string]string, hashed []Entry) *Checker {
	c := &Checker{plain: plain, hashes: map[string]string{}, ok: map[[32]byte]bool{}}
	for _, e := range hashed {
		c.hashes[e.User] = e.Hash
	}
	return c
}

// Check reports whether user and pw are valid.
func (c *Checker) Check(user, pw string) bool {
	if want, ok := c.plain[user]; ok {
		return subtle.ConstantTimeCompare([]byte(want), []byte(pw)) == 1
	}
	h, ok := c.hashes[user]
	if !ok {
		return false
	}
	k := sha256.Sum256([]byte(user + "\x00" + pw + "\x00" + h))
	c.mu.Lock()
	hit := c.ok[k]
	c.mu.Unlock()
	if hit {
		return true
	}
	if !Verify(h, pw) {
		return false
	}
	c.mu.Lock()
	if len(c.ok) > 1000 {
		c.ok = map[[32]byte]bool{}
	}
	c.ok[k] = true
	c.mu.Unlock()
	return true
}

// FileChecker checks logins against the password file in a directory and
// picks up changes made by "traffic66 passwd" without a restart. When
// fixed passwords are given, the file is not used.
type FileChecker struct {
	dir   string
	fixed map[string]string
	mu    sync.Mutex
	stamp string
	c     *Checker
}

// NewFileChecker returns a checker for dir; fixed may be nil.
func NewFileChecker(dir string, fixed map[string]string) *FileChecker {
	f := &FileChecker{dir: dir, fixed: fixed}
	if fixed != nil {
		f.c = NewChecker(fixed, nil)
	}
	return f
}

// Check reports whether user and pw are valid.
func (f *FileChecker) Check(user, pw string) bool {
	if f.fixed != nil {
		return f.c.Check(user, pw)
	}
	stamp := ""
	if fi, err := os.Stat(filepath.Join(f.dir, FileName)); err == nil {
		stamp = fmt.Sprint(fi.ModTime().UnixNano(), fi.Size())
	}
	f.mu.Lock()
	if f.c == nil || stamp != f.stamp {
		es, err := Load(f.dir)
		if err != nil {
			es = nil
		}
		f.c, f.stamp = NewChecker(nil, es), stamp
	}
	c := f.c
	f.mu.Unlock()
	return c.Check(user, pw)
}

// Exists reports whether user can currently sign in.
func (f *FileChecker) Exists(user string) bool {
	if f.fixed != nil {
		_, ok := f.fixed[user]
		return ok
	}
	f.Check("", "") // reload the file if it changed
	f.mu.Lock()
	c := f.c
	f.mu.Unlock()
	_, ok := c.hashes[user]
	return ok
}
