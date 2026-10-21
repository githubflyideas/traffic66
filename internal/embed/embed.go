// Package embed keeps the links other platforms use to show one address's
// traffic: an asset platform puts a "traffic" button beside each of its
// service addresses (VIPs), and the button opens, or frames, a page here
// with that address's bandwidth and nothing else.
//
// One key covers a whole platform, so its links are built by putting the
// address into a fixed URL rather than by keeping one secret per address.
// What keeps that safe is everything around the key: it opens only the
// addresses in its ranges, only from the sites it names, optionally only
// for viewers on given networks, and only so many times a minute. It is
// stored as a hash, shown once when it is made, and can be replaced (the
// old one keeps working for a day, while the platform changes over) or
// withdrawn at once.
package embed

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/netip"
	"net/url"
	"os"
	"slices"
	"strings"
	"sync"
	"time"
)

const (
	// KeyPrefix starts every key, so one pasted somewhere it should not be
	// is recognisable for what it is.
	KeyPrefix = "t66e_"
	// Grace is how long a replaced key keeps working.
	Grace = 24 * time.Hour
	// PerMinute is how many requests one key may make in a minute from
	// one viewer: enough for a page that redraws every minute to be open
	// in several tabs, and nowhere near enough to read the data out fast.
	PerMinute = 60
	// MaxSources keeps the file a size a person can read.
	MaxSources = 200
)

// Errors a request can be refused with. The page says which, so whoever
// set the platform up can tell a typo from a block.
var (
	ErrKey     = errors.New("unknown or withdrawn key")
	ErrScope   = errors.New("this address is not one the key may show")
	ErrOrigin  = errors.New("not opened from a site the key allows")
	ErrViewer  = errors.New("not opened from a network the key allows")
	ErrRate    = errors.New("too many requests; try again in a minute")
	ErrUnknown = errors.New("no such link")
)

// Source is one platform allowed to show traffic.
type Source struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Ranges  []string `json:"ranges"`            // addresses and prefixes it may show
	Origins []string `json:"origins"`           // sites it may be opened or framed from; "*" for any
	Clients []string `json:"clients,omitempty"` // viewer networks; empty for any

	Prefix   string    `json:"prefix"` // the start of the key, to tell keys apart
	Hash     string    `json:"hash"`
	OldHash  string    `json:"old_hash,omitempty"`
	OldUntil time.Time `json:"old_until,omitzero"`
	Created  time.Time `json:"created"`
	LastUsed time.Time `json:"last_used,omitzero"`
	Hits     int64     `json:"hits"`
}

// Store is the list of sources, kept in one file.
type Store struct {
	mu    sync.Mutex
	path  string
	list  []*Source
	saved time.Time
	rate  map[string]*window
	now   func() time.Time
}

type window struct {
	min time.Time
	n   int
}

// Open reads the list; a missing file is an empty one.
func Open(path string) (*Store, error) {
	s := &Store{path: path, rate: map[string]*window{}, now: time.Now}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, &s.list); err != nil {
		return nil, fmt.Errorf("embed: %s: %w", path, err)
	}
	return s, nil
}

// List returns the sources without their key hashes.
func (s *Store) List() []Source {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Source, 0, len(s.list))
	for _, x := range s.list {
		c := *x
		c.Hash, c.OldHash = "", ""
		if !c.OldUntil.After(s.now()) {
			c.OldUntil = time.Time{}
		}
		out = append(out, c)
	}
	return out
}

// Put adds a source, or changes the name and limits of one. A new source
// gets a key, returned this once; changing one keeps its key.
func (s *Store) Put(in Source) (key string, err error) {
	if err := in.clean(); err != nil {
		return "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if in.ID == "" {
		if len(s.list) >= MaxSources {
			return "", fmt.Errorf("at most %d links", MaxSources)
		}
		key = newKey()
		x := &Source{ID: randHex(6), Name: in.Name, Ranges: in.Ranges, Origins: in.Origins, Clients: in.Clients,
			Prefix: key[:len(KeyPrefix)+4], Hash: hash(key), Created: s.now().UTC()}
		s.list = append(s.list, x)
		return key, s.save()
	}
	x := s.find(in.ID)
	if x == nil {
		return "", ErrUnknown
	}
	x.Name, x.Ranges, x.Origins, x.Clients = in.Name, in.Ranges, in.Origins, in.Clients
	return "", s.save()
}

// Rotate gives a source a new key. The old one works for Grace more, so
// the platform can change over without its buttons failing meanwhile.
func (s *Store) Rotate(id string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	x := s.find(id)
	if x == nil {
		return "", ErrUnknown
	}
	key := newKey()
	x.OldHash, x.OldUntil = x.Hash, s.now().UTC().Add(Grace)
	x.Hash, x.Prefix = hash(key), key[:len(KeyPrefix)+4]
	return key, s.save()
}

// Delete withdraws a source: its key stops working at once.
func (s *Store) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := slices.IndexFunc(s.list, func(x *Source) bool { return x.ID == id })
	if i < 0 {
		return ErrUnknown
	}
	s.list = slices.Delete(s.list, i, i+1)
	return s.save()
}

// Request is what one page or data request carries.
type Request struct {
	Key     string
	Addr    netip.Addr // the address to show
	Viewer  netip.Addr // who is asking
	Referer string     // the page it was opened from; checked when set to
	Page    bool       // the page itself, which is what a site opens or frames
}

// Check decides one request, and counts it. The page is checked against
// the sites the key allows; the data it then fetches comes from this
// server's own page, so for that the key, the address and the viewer are
// what is checked.
func (s *Store) Check(r Request) (Source, error) {
	h := hash(r.Key)
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.now()
	var x *Source
	for _, c := range s.list {
		if eq(c.Hash, h) || (c.OldHash != "" && now.Before(c.OldUntil) && eq(c.OldHash, h)) {
			x = c
		}
	}
	if x == nil || !strings.HasPrefix(r.Key, KeyPrefix) {
		return Source{}, ErrKey
	}
	if !s.allow(x.ID+"|"+r.Viewer.String(), now) {
		return Source{}, ErrRate
	}
	if !inAny(r.Addr, x.Ranges) {
		return Source{}, ErrScope
	}
	if len(x.Clients) > 0 && !inAny(r.Viewer, x.Clients) {
		return Source{}, ErrViewer
	}
	if r.Page && !originOK(r.Referer, x.Origins) {
		return Source{}, ErrOrigin
	}
	x.Hits++
	x.LastUsed = now.UTC()
	// counts are kept, but not by writing the file at every request
	if now.Sub(s.saved) > time.Minute {
		s.save()
	}
	c := *x
	c.Hash, c.OldHash = "", ""
	return c, nil
}

// Flush writes the counts kept since the last save.
func (s *Store) Flush() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.save()
}

func (s *Store) allow(who string, now time.Time) bool {
	m := now.Truncate(time.Minute)
	if len(s.rate) > 10000 {
		for k, w := range s.rate {
			if w.min != m {
				delete(s.rate, k)
			}
		}
	}
	w := s.rate[who]
	if w == nil || w.min != m {
		w = &window{min: m}
		s.rate[who] = w
	}
	if w.n >= PerMinute {
		return false
	}
	w.n++
	return true
}

func (s *Store) find(id string) *Source {
	for _, x := range s.list {
		if x.ID == id {
			return x
		}
	}
	return nil
}

func (s *Store) save() error {
	s.saved = s.now()
	if s.path == "" {
		return nil
	}
	b, err := json.MarshalIndent(s.list, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// clean checks a source as typed into the settings page and puts it in
// one canonical form.
func (in *Source) clean() error {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" || len([]rune(in.Name)) > 80 {
		return errors.New("a name of up to 80 characters is needed")
	}
	var err error
	if in.Ranges, err = prefixes(in.Ranges); err != nil {
		return err
	}
	if len(in.Ranges) == 0 {
		return errors.New("at least one address or range to show is needed")
	}
	if in.Clients, err = prefixes(in.Clients); err != nil {
		return err
	}
	var origins []string
	for _, o := range in.Origins {
		o = strings.ToLower(strings.TrimSpace(o))
		if o == "" {
			continue
		}
		// a pasted URL is taken for its host
		if u, err := url.Parse(o); err == nil && u.Host != "" {
			o = u.Hostname()
		}
		if o != "*" && !validHost(strings.TrimPrefix(o, "*.")) {
			return fmt.Errorf("%q is not a site name", o)
		}
		if !slices.Contains(origins, o) {
			origins = append(origins, o)
		}
	}
	if len(origins) == 0 {
		return errors.New("at least one site the link may be opened from is needed, or * for any")
	}
	in.Origins = origins
	return nil
}

func prefixes(ss []string) ([]string, error) {
	var out []string
	for _, x := range ss {
		x = strings.TrimSpace(x)
		if x == "" {
			continue
		}
		p, err := netip.ParsePrefix(x)
		if err != nil {
			a, aerr := netip.ParseAddr(x)
			if aerr != nil {
				return nil, fmt.Errorf("%q is neither an address nor a range", x)
			}
			p = netip.PrefixFrom(a.Unmap(), a.Unmap().BitLen())
		}
		s := p.Masked().String()
		if !slices.Contains(out, s) {
			out = append(out, s)
		}
	}
	if len(out) > 1000 {
		return nil, errors.New("at most 1000 ranges")
	}
	return out, nil
}

func inAny(a netip.Addr, ranges []string) bool {
	if !a.IsValid() {
		return false
	}
	a = a.Unmap()
	for _, r := range ranges {
		if p, err := netip.ParsePrefix(r); err == nil && p.Contains(a) {
			return true
		}
	}
	return false
}

// originOK: the page was opened from, or framed by, one of the sites.
// "*.example.com" covers example.com's subdomains.
func originOK(referer string, origins []string) bool {
	if slices.Contains(origins, "*") {
		return true
	}
	u, err := url.Parse(referer)
	if err != nil || u.Host == "" {
		return false
	}
	h := strings.ToLower(u.Hostname())
	for _, o := range origins {
		if h == o || (strings.HasPrefix(o, "*.") && strings.HasSuffix(h, o[1:])) {
			return true
		}
	}
	return false
}

// FrameAncestors is the Content-Security-Policy value that lets only the
// source's sites frame the page.
func FrameAncestors(origins []string) string {
	if slices.Contains(origins, "*") {
		return "frame-ancestors *"
	}
	return "frame-ancestors " + strings.Join(origins, " ")
}

func validHost(h string) bool {
	if h == "" || len(h) > 253 {
		return false
	}
	if _, err := netip.ParseAddr(h); err == nil {
		return true
	}
	for _, l := range strings.Split(h, ".") {
		if l == "" || len(l) > 63 {
			return false
		}
		for _, c := range l {
			if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-') {
				return false
			}
		}
	}
	return true
}

func newKey() string {
	b := make([]byte, 24)
	rand.Read(b)
	return KeyPrefix + base64.RawURLEncoding.EncodeToString(b)
}

func randHex(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func hash(key string) string {
	h := sha256.Sum256([]byte(key))
	return hex.EncodeToString(h[:])
}

func eq(a, b string) bool { return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1 }
