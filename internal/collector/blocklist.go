// A blocklist of exporter addresses whose datagrams are dropped on
// arrival. It exists because a UDP collector accepts whatever reaches the
// port: one misconfigured device, or one host replaying captured flow
// records, is enough to fill the database with traffic that was never
// seen and to raise findings against hosts that did nothing.
//
// It is a blocklist and not an allow-list on purpose, and that choice has
// a cost worth stating plainly: whoever is sending can change their
// source address, and a UDP source address can be forged outright. So
// this stops a known nuisance — a device that will not stop exporting, a
// test rig left running — and it does not stop somebody who does not want
// to be stopped. For that, the collector ports belong behind a firewall
// that admits only your own devices, which is what the page says when a
// block is added.
package collector

import (
	"bufio"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// BlockFile is the name of the blocklist inside the data directory. It is
// a text file for the same reason the inventory is: it can be read,
// diffed, put in a repository and written by a script, and it needs no
// database to be consulted on the receive path.
const BlockFile = "blocked.txt"

// Block is one blocked source.
type Block struct {
	Addr    netip.Addr `json:"-"`
	Address string     `json:"address"`
	Note    string     `json:"note,omitempty"`
	Since   time.Time  `json:"since"`
	Dropped uint64     `json:"dropped"`
}

// Blocklist answers, for every datagram, whether its sender is blocked.
// Has is on the receive path, so it is a map read under a read lock and
// nothing more.
type Blocklist struct {
	mu   sync.RWMutex
	m    map[netip.Addr]*Block
	path string
	// Set by Has, cleared by Flush. The drop counts are written back to
	// the file periodically rather than per datagram: the receive path
	// cannot afford a write, and a count that survives a restart is worth
	// more than one that is exact to the last minute.
	dirty atomic.Bool
}

// blockable reports whether a can be the source address of a datagram at
// all. The unspecified address and any multicast address cannot, so
// blocking one only puts a line in the file that will never match.
func blockable(a netip.Addr) error {
	switch {
	case !a.IsValid():
		return errors.New("not an address")
	case a.IsUnspecified():
		return errors.New("the unspecified address is never a sender")
	case a.IsMulticast():
		return errors.New("a multicast address is never a sender")
	}
	return nil
}

// NewBlocklist reads the blocklist from dir. A missing file is not an
// error: it is what a fresh installation has. An unreadable line is
// skipped rather than refused, because a collector that will not start
// over a typo in a blocklist is worse than one that drops a line of it.
func NewBlocklist(dir string) (*Blocklist, error) {
	b := &Blocklist{m: map[netip.Addr]*Block{}}
	if dir != "" {
		b.path = filepath.Join(dir, BlockFile)
	}
	if b.path == "" {
		return b, nil
	}
	f, err := os.Open(b.path)
	if os.IsNotExist(err) {
		return b, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// "<address> [RFC3339 time] [note...]"
		f := strings.Fields(line)
		a, err := netip.ParseAddr(f[0])
		if err != nil || blockable(a.Unmap()) != nil {
			continue
		}
		e := &Block{Addr: a.Unmap(), Address: a.Unmap().String(), Since: time.Time{}}
		rest := f[1:]
		if len(rest) > 0 {
			if t, err := time.Parse(time.RFC3339, rest[0]); err == nil {
				e.Since = t
				rest = rest[1:]
			}
		}
		if len(rest) > 0 && strings.HasPrefix(rest[0], "dropped=") {
			if n, err := strconv.ParseUint(strings.TrimPrefix(rest[0], "dropped="), 10, 64); err == nil {
				e.Dropped = n
				rest = rest[1:]
			}
		}
		e.Note = strings.Join(rest, " ")
		b.m[e.Addr] = e
	}
	return b, sc.Err()
}

// Has reports whether datagrams from a are to be dropped, and counts the
// drop. It is called once per datagram.
func (b *Blocklist) Has(a netip.Addr) bool {
	b.mu.RLock()
	e := b.m[a]
	b.mu.RUnlock()
	if e == nil {
		return false
	}
	b.mu.Lock()
	e.Dropped++
	b.mu.Unlock()
	b.dirty.Store(true)
	return true
}

// List returns the blocked sources, oldest block first.
func (b *Blocklist) List() []Block {
	b.mu.RLock()
	out := make([]Block, 0, len(b.m))
	for _, e := range b.m {
		out = append(out, *e)
	}
	b.mu.RUnlock()
	sort.Slice(out, func(i, j int) bool {
		if out[i].Since.Equal(out[j].Since) {
			return out[i].Address < out[j].Address
		}
		return out[i].Since.Before(out[j].Since)
	})
	return out
}

// Add blocks an address. Blocking one that is already blocked keeps the
// original time and drop count, so a second click does not look like a
// fresh block.
func (b *Blocklist) Add(a netip.Addr, note string, now time.Time) error {
	a = a.Unmap()
	if err := blockable(a); err != nil {
		return err
	}
	b.mu.Lock()
	if b.m[a] == nil {
		b.m[a] = &Block{Addr: a, Address: a.String(), Note: note, Since: now.UTC()}
	} else if note != "" {
		b.m[a].Note = note
	}
	b.mu.Unlock()
	return b.save()
}

// Remove unblocks an address, and reports whether it was blocked.
func (b *Blocklist) Remove(a netip.Addr) (bool, error) {
	a = a.Unmap()
	b.mu.Lock()
	_, had := b.m[a]
	delete(b.m, a)
	b.mu.Unlock()
	if !had {
		return false, nil
	}
	return true, b.save()
}

// Flush writes the drop counts back if any changed. Call it periodically:
// Has cannot write on the receive path, so without this the counts would
// start again from zero at every restart, which is exactly when somebody
// wants to know how much a blocked source has been sending.
func (b *Blocklist) Flush() error {
	if !b.dirty.Swap(false) {
		return nil
	}
	return b.save()
}

// save writes the file, through a temporary file so a crash halfway
// cannot leave a half-written blocklist behind.
func (b *Blocklist) save() error {
	if b.path == "" {
		return nil
	}
	var sb strings.Builder
	sb.WriteString("# Sources whose datagrams traffic66 drops on arrival.\n")
	sb.WriteString("# One address per line, then the time it was blocked, the datagrams\n")
	sb.WriteString("# dropped from it so far, and an optional note.\n")
	sb.WriteString("# A blocked sender can change address, and a UDP source address can be\n")
	sb.WriteString("# forged, so this stops a known nuisance rather than an attacker: allow\n")
	sb.WriteString("# only your own devices to the collector ports in your firewall.\n")
	for _, e := range b.List() {
		since := e.Since
		if since.IsZero() {
			since = time.Now().UTC()
		}
		line := e.Address + " " + since.UTC().Format(time.RFC3339)
		if e.Dropped > 0 {
			line += " dropped=" + strconv.FormatUint(e.Dropped, 10)
		}
		if n := strings.TrimSpace(strings.ReplaceAll(e.Note, "\n", " ")); n != "" {
			line += " " + n
		}
		sb.WriteString(line + "\n")
	}
	tmp := b.path + ".tmp"
	if err := os.WriteFile(tmp, []byte(sb.String()), 0o644); err != nil {
		return fmt.Errorf("blocklist: %w", err)
	}
	return os.Rename(tmp, b.path)
}
