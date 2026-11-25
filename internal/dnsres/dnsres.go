// Package dnsres resolves addresses to names for display, with a cache and
// a query budget so the upstream server is never flooded.
package dnsres

import (
	"context"
	"net"
	"net/netip"
	"strings"
	"sync"
	"time"
)

type entry struct {
	name string
	exp  time.Time
}

// Resolver does rate-limited reverse lookups.
type Resolver struct {
	r        *net.Resolver
	ttl      time.Duration
	rate     float64
	maxConc  int
	upstream string
	private  bool // allow PTR queries for private addresses

	mu       sync.Mutex
	cache    map[string]entry
	inflight map[string]chan struct{}
	tokens   float64
	last     time.Time
	running  int

	Queries, Hits, Skipped uint64
}

// Options configure a Resolver. Zero values take the defaults.
type Options struct {
	Upstream   string // host:port; empty = system resolver
	PerSecond  float64
	TTL        time.Duration
	MaxConcurr int
}

func New(o Options) *Resolver {
	if o.PerSecond <= 0 {
		o.PerSecond = 20
	}
	if o.TTL <= 0 {
		o.TTL = 120 * time.Second
	}
	if o.MaxConcurr <= 0 {
		o.MaxConcurr = 8
	}
	res := &Resolver{ttl: o.TTL, rate: o.PerSecond, maxConc: o.MaxConcurr, cache: map[string]entry{}, inflight: map[string]chan struct{}{},
		tokens: o.PerSecond, last: time.Now(), private: true}
	res.r = net.DefaultResolver
	if o.Upstream != "" {
		up := o.Upstream
		if _, err := netip.ParseAddrPort(up); err != nil {
			up = net.JoinHostPort(strings.Trim(up, "[]"), "53")
		}
		res.upstream = up
		if h, _, err := net.SplitHostPort(up); err == nil {
			if a, err := netip.ParseAddr(h); err == nil && !a.IsPrivate() && !a.IsLoopback() {
				// Do not tell a public resolver which internal hosts we look at.
				res.private = false
			}
		}
		d := net.Dialer{Timeout: 2 * time.Second}
		res.r = &net.Resolver{PreferGo: true, Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
			return d.DialContext(ctx, network, up)
		}}
	}
	return res
}

// Upstream describes where queries go.
func (r *Resolver) Upstream() string {
	if r.upstream == "" {
		return "system"
	}
	return r.upstream
}

func (r *Resolver) take(now time.Time) bool {
	r.tokens += now.Sub(r.last).Seconds() * r.rate
	if r.tokens > r.rate {
		r.tokens = r.rate
	}
	r.last = now
	if r.tokens < 1 || r.running >= r.maxConc {
		return false
	}
	r.tokens--
	return true
}

// Lookup returns names for the given addresses, waiting at most wait for
// fresh lookups. Addresses over the budget are left out and may be asked
// again later.
func (r *Resolver) Lookup(ips []string, wait time.Duration) map[string]string {
	out := map[string]string{}
	var waits []chan struct{}
	now := time.Now()
	r.mu.Lock()
	for _, ip := range ips {
		a, err := netip.ParseAddr(ip)
		if err != nil {
			continue
		}
		if e, ok := r.cache[ip]; ok && now.Before(e.exp) {
			r.Hits++
			if e.name != "" {
				out[ip] = e.name
			}
			continue
		}
		if ch, ok := r.inflight[ip]; ok {
			waits = append(waits, ch)
			continue
		}
		if !r.private && (a.IsPrivate() || a.IsLoopback() || a.IsLinkLocalUnicast()) {
			continue
		}
		if !r.take(now) {
			r.Skipped++
			continue
		}
		ch := make(chan struct{})
		r.inflight[ip] = ch
		r.running++
		r.Queries++
		waits = append(waits, ch)
		go r.resolve(ip, ch)
	}
	r.mu.Unlock()
	deadline := time.After(wait)
	for _, ch := range waits {
		select {
		case <-ch:
		case <-deadline:
			goto done
		}
	}
done:
	r.mu.Lock()
	for _, ip := range ips {
		if e, ok := r.cache[ip]; ok && e.name != "" {
			out[ip] = e.name
		}
	}
	r.mu.Unlock()
	return out
}

func (r *Resolver) resolve(ip string, ch chan struct{}) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	names, err := r.r.LookupAddr(ctx, ip)
	cancel()
	name := ""
	if err == nil && len(names) > 0 {
		name = strings.TrimSuffix(names[0], ".")
	}
	r.mu.Lock()
	r.cache[ip] = entry{name, time.Now().Add(r.ttl)}
	delete(r.inflight, ip)
	r.running--
	if len(r.cache) > 200000 {
		now := time.Now()
		for k, e := range r.cache {
			if now.After(e.exp) {
				delete(r.cache, k)
			}
		}
	}
	r.mu.Unlock()
	close(ch)
}
