// Package license keeps the trial and licence state of an installation.
//
// On first start license.json is written to the data directory with an
// installation number and the time of the first run. A licence from the
// author is the same file with the customer, product, licence type, end
// date and a signature (Ed25519 over the fields, see Message). Nothing is
// ever switched off: the state only changes what the page footer says.
package license

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// publicKey checks licence signatures; the private key stays with the author
// and is not in this repository.
var publicKey = "QCHDB+akGoDVYVcQ9PiQmApccIABaexHMbxB/MtpY2c="

// TrialDays is the length of the trial.
const TrialDays = 30

// File is license.json.
type File struct {
	InstallationID string  `json:"installation_id"`
	FirstRun       string  `json:"first_run"`
	Customer       string  `json:"customer,omitempty"`
	Product        string  `json:"product,omitempty"`
	License        string  `json:"license,omitempty"`
	Expires        *string `json:"expires,omitempty"` // YYYY-MM-DD or RFC 3339; null: not licensed
	Signature      string  `json:"signature,omitempty"`
}

// Message is what a signature covers.
func Message(f File) []byte {
	exp := "null"
	if f.Expires != nil {
		exp = *f.Expires
	}
	return []byte(fmt.Sprintf("traffic66-license-v1\n%s\n%s\n%s\n%s\n%s\n", f.InstallationID, f.Customer, f.Product, f.License, exp))
}

// State is what the footer shows.
type State struct {
	Kind           string `json:"kind"` // trial, trial_over, licensed
	Days           int    `json:"days"` // days left of the trial or the licence
	Customer       string `json:"customer,omitempty"`
	InstallationID string `json:"installation_id"`
	Expires        string `json:"expires,omitempty"` // RFC 3339
}

// Checker reads license.json now and every few hours.
type Checker struct {
	Path string
	mu   sync.Mutex
	st   State
}

// Open creates license.json in dir on the first run and checks it.
func Open(dir string) (*Checker, error) {
	c := &Checker{Path: filepath.Join(dir, "license.json")}
	if _, err := os.Stat(c.Path); os.IsNotExist(err) {
		n, _ := rand.Int(rand.Reader, big.NewInt(90000000))
		f := File{InstallationID: fmt.Sprintf("%08d", n.Int64()+10000000), FirstRun: time.Now().UTC().Format(time.RFC3339)}
		b, _ := json.MarshalIndent(f, "", "  ")
		if err := os.WriteFile(c.Path, append(b, '\n'), 0o644); err != nil {
			return nil, err
		}
	}
	c.Check(time.Now())
	return c, nil
}

// State returns the last checked state.
func (c *Checker) State() State {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.st
}

// Check reads the file again.
func (c *Checker) Check(now time.Time) State {
	st := Evaluate(c.Path, now)
	c.mu.Lock()
	c.st = st
	c.mu.Unlock()
	return st
}

// Loop checks every interval until stop is closed.
func (c *Checker) Loop(stop <-chan struct{}, every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-stop:
			return
		case now := <-t.C:
			c.Check(now)
		}
	}
}

// Evaluate works out the state of the file at path.
func Evaluate(path string, now time.Time) State {
	var f File
	b, err := os.ReadFile(path)
	if err == nil {
		err = json.Unmarshal(b, &f)
	}
	st := State{InstallationID: f.InstallationID}
	if exp, ok := valid(f); ok && exp.After(now) {
		st.Kind, st.Customer, st.Expires = "licensed", f.Customer, exp.UTC().Format(time.RFC3339)
		st.Days = ceilDays(exp.Sub(now))
		return st
	}
	first, perr := time.Parse(time.RFC3339, f.FirstRun)
	if err != nil || perr != nil || first.After(now) {
		// unreadable, or the clock was set back: no trial left to count
		st.Kind = "trial_over"
		return st
	}
	left := first.Add(TrialDays * 24 * time.Hour).Sub(now)
	if left <= 0 {
		st.Kind = "trial_over"
		return st
	}
	st.Kind = "trial"
	st.Days = ceilDays(left)
	return st
}

// ceilDays counts a started day as a day.
func ceilDays(d time.Duration) int {
	return int((d + 24*time.Hour - 1) / (24 * time.Hour))
}

// valid checks the signature and returns the end of the licence; a licence
// without an end date (null) counts as none.
func valid(f File) (time.Time, bool) {
	if f.Signature == "" || f.Expires == nil || f.Product != "traffic66" {
		return time.Time{}, false
	}
	pub, _ := base64.StdEncoding.DecodeString(publicKey)
	sig, err := base64.StdEncoding.DecodeString(f.Signature)
	if err != nil || len(pub) != ed25519.PublicKeySize || !ed25519.Verify(pub, Message(f), sig) {
		return time.Time{}, false
	}
	if t, err := time.Parse(time.RFC3339, *f.Expires); err == nil {
		return t, true
	}
	if t, err := time.Parse("2006-01-02", *f.Expires); err == nil {
		return t.Add(24 * time.Hour), true // to the end of that day (UTC)
	}
	return time.Time{}, false
}
