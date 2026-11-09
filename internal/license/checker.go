package license

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// InstallationFile holds this installation's number. It is kept apart from
// the licence because it identifies the installation, not the licence:
// installing a licence replaces license.json, and the number has to
// survive that.
const InstallationFile = "installation"

// Checker reads license.json now and every few hours, so a licence dropped
// into the data directory takes effect without a restart.
type Checker struct {
	Path    string
	Install string
	Ring    Keyring
	mu      sync.Mutex
	st      State
}

// Open prepares the licence state of a data directory, giving the
// installation a number on its first run.
func Open(dir string) (*Checker, error) {
	id, err := installation(dir)
	if err != nil {
		return nil, err
	}
	c := &Checker{Path: InData(dir), Install: id, Ring: Release()}
	c.Check(time.Now())
	return c, nil
}

// installation reads this installation's number, creating it on the first
// run. An installation that predates the separate file keeps the number it
// already had, which was written inside license.json.
func installation(dir string) (string, error) {
	path := filepath.Join(dir, InstallationFile)
	if b, err := os.ReadFile(path); err == nil {
		if id := trimSpace(string(b)); id != "" {
			return id, nil
		}
	}
	id := ""
	if b, err := os.ReadFile(InData(dir)); err == nil {
		var f File
		if json.Unmarshal(b, &f) == nil {
			id = f.InstallationID
		}
	}
	if id == "" {
		n, err := rand.Int(rand.Reader, big.NewInt(90000000))
		if err != nil {
			return "", err
		}
		id = fmt.Sprintf("%08d", n.Int64()+10000000)
	}
	if err := os.WriteFile(path, []byte(id+"\n"), 0o644); err != nil {
		return "", err
	}
	return id, nil
}

// State returns the last checked state.
func (c *Checker) State() State {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.st
}

// Check reads the licence again. A data directory with no licence is the
// normal case, not an error: the installation is unregistered, which costs
// it the registered-only features and marks its reports, and nothing else.
func (c *Checker) Check(now time.Time) State {
	st := Unregistered("license: not registered")
	if f, err := Load(c.Path); err == nil {
		st = Check(c.Ring, f, c.Install, now)
	} else if !os.IsNotExist(err) {
		st = Unregistered("license: " + err.Error())
	}
	st.InstallationID = c.Install
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
