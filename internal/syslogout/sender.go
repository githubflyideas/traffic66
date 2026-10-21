package syslogout

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

// Config is what a customer sets. It is deliberately small: an address, a
// format, and the two knobs that stop a feed from being turned off again
// (what is worth sending, and how much of it at most).
type Config struct {
	Enabled  bool   `json:"enabled"`
	Address  string `json:"address"`    // host:port
	Network  string `json:"network"`    // udp, tcp, tls
	Format   Format `json:"format"`     // cef, leef, json, text
	MinSev   int    `json:"min_sev"`    // 1 low, 2 medium, 3 high
	Facility int    `json:"facility"`   // syslog facility, default 13 (log audit)
	Hostname string `json:"hostname"`   // what to put in the header; empty is this machine's
	ASCII    bool   `json:"ascii"`      // escape everything outside ASCII
	PerMin   int    `json:"per_minute"` // hard ceiling, 0 takes the default
	CAFile   string `json:"ca_file"`    // trust for a TLS collector with its own CA
	Insecure bool   `json:"insecure"`   // accept any certificate; for a lab, not for a customer
	BaseURL  string `json:"base_url"`   // where the web UI is reachable, for the evidence link
}

// Defaults for anything left empty.
const (
	DefaultFacility = 13 // log audit
	DefaultPerMin   = 60
	dialTimeout     = 5 * time.Second
	writeTimeout    = 5 * time.Second
)

func (c *Config) defaults() {
	if c.Network == "" {
		c.Network = "udp"
	}
	if c.Format == "" {
		c.Format = FormatCEF
	}
	if c.Facility == 0 {
		c.Facility = DefaultFacility
	}
	if c.PerMin == 0 {
		c.PerMin = DefaultPerMin
	}
	if c.MinSev == 0 {
		c.MinSev = 1
	}
}

// Check reports what is wrong with a configuration, so the settings page
// can say so before anything is saved.
func (c Config) Check() error {
	c.defaults()
	if !c.Enabled {
		return nil
	}
	if c.Address == "" {
		return errors.New("syslog: no collector address")
	}
	if _, _, err := net.SplitHostPort(c.Address); err != nil {
		return fmt.Errorf("syslog: address should be host:port, not %q", c.Address)
	}
	switch c.Network {
	case "udp", "tcp", "tls":
	default:
		return fmt.Errorf("syslog: %q is not udp, tcp or tls", c.Network)
	}
	if !c.Format.Valid() {
		return fmt.Errorf("syslog: %q is not one of %v", c.Format, Formats)
	}
	if c.MinSev < 1 || c.MinSev > 3 {
		return fmt.Errorf("syslog: the least severity to send is 1, 2 or 3, not %d", c.MinSev)
	}
	return nil
}

// Sender holds the connection and the throttle.
type Sender struct {
	mu   sync.Mutex
	cfg  Config
	conn net.Conn

	minute  time.Time
	sent    int
	dropped int

	st Stats
}

// Stats is what the settings page shows about the feed: whether messages
// are going out, and if not, why.
type Stats struct {
	Sent      int64     `json:"sent"`
	Failed    int64     `json:"failed"`
	Dropped   int       `json:"dropped"`
	LastSent  time.Time `json:"last_sent,omitzero"`
	LastError string    `json:"last_error,omitempty"`
	LastErrAt time.Time `json:"last_error_at,omitzero"`
}

// Stats returns the counts since this process started.
func (s *Sender) Stats() Stats {
	s.mu.Lock()
	defer s.mu.Unlock()
	st := s.st
	st.Dropped = s.dropped
	return st
}

// note records how one message went.
func (s *Sender) note(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	if err != nil {
		s.st.Failed++
		s.st.LastError, s.st.LastErrAt = err.Error(), now
		return
	}
	s.st.Sent++
	s.st.LastSent = now
}

// New prepares a sender. It does not connect: a collector that is down
// when traffic66 starts is not a reason for traffic66 not to start.
func New(cfg Config) *Sender {
	cfg.defaults()
	return &Sender{cfg: cfg}
}

// Config returns the configuration in use.
func (s *Sender) Config() Config {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cfg
}

// SetConfig swaps the configuration, closing any connection to the old
// collector.
func (s *Sender) SetConfig(cfg Config) {
	cfg.defaults()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg = cfg
	s.close()
}

// Dropped is how many messages the ceiling has thrown away, for the
// settings page: a feed that is silently losing events is worse than one
// that says so.
func (s *Sender) Dropped() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.dropped
}

// Send reports one event. Anything below the configured severity, and
// anything over the ceiling, is dropped here rather than at the collector.
func (s *Sender) Send(e Event) {
	s.mu.Lock()
	cfg := s.cfg
	if !cfg.Enabled || cfg.Address == "" {
		s.mu.Unlock()
		return
	}
	if e.Phase != PhaseTest && e.Severity < cfg.MinSev {
		s.mu.Unlock()
		return
	}
	if !s.allow(time.Now()) {
		s.dropped++
		s.mu.Unlock()
		return
	}
	s.mu.Unlock()

	err := s.write(Message(cfg, e, time.Now()))
	s.note(err)
	if err != nil {
		log.Printf("syslog: %v", err)
	}
}

// Test sends one message and reports whether it went, for the button that
// says so on the settings page.
func (s *Sender) Test() error {
	cfg := s.Config()
	if err := cfg.Check(); err != nil {
		return err
	}
	if !cfg.Enabled {
		return errors.New("syslog: not switched on")
	}
	now := time.Now()
	e := Event{
		Phase: PhaseTest, ID: "0", Kind: "test", Severity: 1,
		First: now, Last: now, Extra: map[string]string{},
	}
	e.Text = Describe(e)
	err := s.write(Message(cfg, e, now))
	s.note(err)
	return err
}

// allow is the ceiling: a flood of findings must not become a flood of
// syslog, or the customer turns the feed off and nobody hears about the
// next one either.
func (s *Sender) allow(now time.Time) bool {
	m := now.Truncate(time.Minute)
	if m != s.minute {
		s.minute, s.sent = m, 0
	}
	if s.sent >= s.cfg.PerMin {
		return false
	}
	s.sent++
	return true
}

// write sends one line, reconnecting once if the connection has gone. A
// collector that restarts should cost one message at most, not the feed.
func (s *Sender) write(msg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for attempt := 0; attempt < 2; attempt++ {
		if s.conn == nil {
			c, err := s.dial()
			if err != nil {
				return err
			}
			s.conn = c
		}
		s.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
		var err error
		if s.cfg.Network == "udp" {
			_, err = s.conn.Write([]byte(msg))
		} else {
			// Octet counting (RFC 6587): a stream needs to know where one
			// message ends, and a length is the framing collectors agree on.
			_, err = fmt.Fprintf(s.conn, "%d %s", len(msg), msg)
		}
		if err == nil {
			return nil
		}
		s.close()
		if attempt == 1 {
			return fmt.Errorf("sending to %s: %w", s.cfg.Address, err)
		}
	}
	return nil
}

func (s *Sender) dial() (net.Conn, error) {
	switch s.cfg.Network {
	case "tls":
		cfg := &tls.Config{InsecureSkipVerify: s.cfg.Insecure, MinVersion: tls.VersionTLS12}
		if s.cfg.CAFile != "" {
			pem, err := os.ReadFile(s.cfg.CAFile)
			if err != nil {
				return nil, fmt.Errorf("syslog: %w", err)
			}
			pool := x509.NewCertPool()
			if !pool.AppendCertsFromPEM(pem) {
				return nil, fmt.Errorf("syslog: %s holds no certificate", s.cfg.CAFile)
			}
			cfg.RootCAs = pool
		}
		if host, _, err := net.SplitHostPort(s.cfg.Address); err == nil {
			cfg.ServerName = host
		}
		return tls.DialWithDialer(&net.Dialer{Timeout: dialTimeout}, "tcp", s.cfg.Address, cfg)
	case "tcp":
		return net.DialTimeout("tcp", s.cfg.Address, dialTimeout)
	}
	return net.DialTimeout("udp", s.cfg.Address, dialTimeout)
}

// close drops the connection; the next write dials again.
func (s *Sender) close() {
	if s.conn != nil {
		s.conn.Close()
		s.conn = nil
	}
}

// Close stops using the collector.
func (s *Sender) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.close()
}

// title upper-cases the first letter of a field name for CEF, where the
// custom field names are conventionally camel case.
func title(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// LoadConfig reads the configuration saved from the settings page. ok is
// false when nothing has been saved yet.
func LoadConfig(path string) (cfg Config, ok bool, err error) {
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Config{}, false, nil
	}
	if err != nil {
		return Config{}, false, err
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return Config{}, false, fmt.Errorf("syslog: %s: %w", path, err)
	}
	return cfg, true, nil
}

// SaveConfig writes the configuration so it outlives a restart.
func SaveConfig(path string, cfg Config) error {
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
