package snmp

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"net"
	"net/netip"
	"sort"
	"sync"
	"time"

	"github.com/githubflyideas/traffic66/internal/flow"
)

// IF-MIB columns (RFC 2863).
var (
	OIDIfName          = MustOID("1.3.6.1.2.1.31.1.1.1.1")
	OIDIfHCInOctets    = MustOID("1.3.6.1.2.1.31.1.1.1.6")
	OIDIfHCInUcastPkt  = MustOID("1.3.6.1.2.1.31.1.1.1.7")
	OIDIfHCOutOctets   = MustOID("1.3.6.1.2.1.31.1.1.1.10")
	OIDIfHCOutUcastPkt = MustOID("1.3.6.1.2.1.31.1.1.1.11")
	OIDIfHighSpeed     = MustOID("1.3.6.1.2.1.31.1.1.1.15")
	OIDIfAlias         = MustOID("1.3.6.1.2.1.31.1.1.1.18")
)

// Client talks SNMPv2c to one agent.
type Client struct {
	Addr      netip.AddrPort
	Community string
	Timeout   time.Duration
	Retries   int
	// MaxRepetitions per GETBULK.
	MaxRepetitions int
}

// ErrTimeout means the agent did not answer (wrong community, ACL, down).
var ErrTimeout = errors.New("snmp: no response")

func (c *Client) exchange(ctx context.Context, req *packet) (*packet, error) {
	conn, err := net.ListenUDP("udp", nil)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	timeout := c.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	buf := make([]byte, 65535)
	for attempt := 0; attempt <= c.Retries; attempt++ {
		req.requestID = rand.Int32()
		b, err := req.marshal()
		if err != nil {
			return nil, err
		}
		if _, err := conn.WriteToUDPAddrPort(b, c.Addr); err != nil {
			return nil, err
		}
		deadline := time.Now().Add(timeout)
		if d, ok := ctx.Deadline(); ok && d.Before(deadline) {
			deadline = d
		}
		conn.SetReadDeadline(deadline)
		for {
			n, _, err := conn.ReadFromUDPAddrPort(buf)
			if err != nil {
				var ne net.Error
				if errors.As(err, &ne) && ne.Timeout() {
					break
				}
				return nil, err
			}
			resp, err := unmarshal(buf[:n])
			if err != nil || resp.pduType != pduResponse || resp.requestID != req.requestID {
				continue // stray or late answer to an earlier attempt
			}
			if resp.a != 0 {
				return nil, fmt.Errorf("snmp: agent error status %d at index %d", resp.a, resp.b)
			}
			return resp, nil
		}
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
	}
	return nil, ErrTimeout
}

// Walk returns every binding under root, using GETBULK.
func (c *Client) Walk(ctx context.Context, root OID) ([]VarBind, error) {
	reps := c.MaxRepetitions
	if reps <= 0 {
		reps = 25
	}
	var out []VarBind
	cur := root
	for guard := 0; guard < 10000; guard++ {
		resp, err := c.exchange(ctx, &packet{community: c.Community, pduType: pduGetBulk, a: 0, b: int64(reps),
			varBinds: []VarBind{{Name: cur, Value: Value{Kind: KindNull}}}})
		if err != nil {
			return out, err
		}
		if len(resp.varBinds) == 0 {
			return out, nil
		}
		for _, vb := range resp.varBinds {
			if vb.Value.Kind == KindEndOfMib || !vb.Name.HasPrefix(root) || len(vb.Name) == len(root) {
				return out, nil
			}
			if vb.Name.Compare(cur) <= 0 {
				return out, errors.New("snmp: agent returned OIDs out of order")
			}
			out = append(out, vb)
			cur = vb.Name
		}
	}
	return out, errors.New("snmp: walk did not terminate")
}

// column walks a table column and keys the values by the last sub-identifier.
func (c *Client) column(ctx context.Context, col OID) (map[uint32]Value, error) {
	vbs, err := c.Walk(ctx, col)
	m := make(map[uint32]Value, len(vbs))
	for _, vb := range vbs {
		if len(vb.Name) == len(col)+1 {
			m[vb.Name[len(col)]] = vb.Value
		}
	}
	return m, err
}

// Interface is one polled interface.
type Interface struct {
	Index uint32
	Name  string
	Speed uint64 // bits/s
}

// PollInterfaces reads the 64-bit counters of every interface.
func (c *Client) PollInterfaces(ctx context.Context, exporter netip.Addr, at time.Time) ([]flow.IfCounters, []Interface, error) {
	in, err := c.column(ctx, OIDIfHCInOctets)
	if err != nil {
		return nil, nil, err
	}
	if len(in) == 0 {
		return nil, nil, errors.New("snmp: agent has no 64-bit interface counters (IF-MIB ifHCInOctets)")
	}
	out, err := c.column(ctx, OIDIfHCOutOctets)
	if err != nil {
		return nil, nil, err
	}
	// optional columns
	speed, _ := c.column(ctx, OIDIfHighSpeed)
	names, _ := c.column(ctx, OIDIfName)
	alias, _ := c.column(ctx, OIDIfAlias)
	inP, _ := c.column(ctx, OIDIfHCInUcastPkt)
	outP, _ := c.column(ctx, OIDIfHCOutUcastPkt)
	var idx []uint32
	for i := range in {
		idx = append(idx, i)
	}
	sort.Slice(idx, func(a, b int) bool { return idx[a] < idx[b] })
	var ctrs []flow.IfCounters
	var ifs []Interface
	for _, i := range idx {
		sp := speed[i].Uint * 1_000_000
		ctrs = append(ctrs, flow.IfCounters{Time: at, Exporter: exporter, IfIndex: i, Speed: sp,
			InOctets: in[i].Uint, OutOctets: out[i].Uint, InPkts: inP[i].Uint, OutPkts: outP[i].Uint})
		name := string(names[i].Str)
		if a := string(alias[i].Str); a != "" {
			name = a // the operator's description says more than "Gi0/0/3"
		}
		ifs = append(ifs, Interface{Index: i, Name: name, Speed: sp})
	}
	return ctrs, ifs, nil
}

// Target is one device to poll. Exporter is the address its flows come from,
// Host is where to send SNMP (often the same).
type Target struct {
	Exporter  netip.Addr
	Host      netip.AddrPort
	Community string
}

// Status is the latest poll outcome for a target.
type Status struct {
	Exporter   string    `json:"exporter"`
	Host       string    `json:"host"`
	OK         bool      `json:"ok"`
	Error      string    `json:"error,omitempty"`
	Interfaces int       `json:"interfaces"`
	At         time.Time `json:"at"`
}

// Poller polls targets on a fixed interval.
type Poller struct {
	Interval time.Duration
	Targets  func() []Target
	Emit     func([]flow.IfCounters)
	Names    func(exporter netip.Addr, ifs []Interface)

	mu     sync.Mutex
	status map[string]Status
}

// Run polls until ctx ends.
func (p *Poller) Run(ctx context.Context) {
	if p.Interval <= 0 {
		p.Interval = time.Minute
	}
	p.pollAll(ctx)
	t := time.NewTicker(p.Interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			p.pollAll(ctx)
		}
	}
}

func (p *Poller) pollAll(ctx context.Context) {
	targets := p.Targets()
	var wg sync.WaitGroup
	for _, tg := range targets {
		wg.Add(1)
		go func(tg Target) {
			defer wg.Done()
			p.pollOne(ctx, tg)
		}(tg)
	}
	wg.Wait()
	// forget targets removed from the inventory
	keep := map[string]bool{}
	for _, tg := range targets {
		keep[tg.Exporter.String()] = true
	}
	p.mu.Lock()
	for k := range p.status {
		if !keep[k] {
			delete(p.status, k)
		}
	}
	p.mu.Unlock()
}

func (p *Poller) pollOne(ctx context.Context, tg Target) {
	cctx, cancel := context.WithTimeout(ctx, p.Interval/2)
	defer cancel()
	c := &Client{Addr: tg.Host, Community: tg.Community, Retries: 1}
	now := time.Now()
	ctrs, ifs, err := c.PollInterfaces(cctx, tg.Exporter, now)
	st := Status{Exporter: tg.Exporter.String(), Host: tg.Host.String(), At: now, OK: err == nil, Interfaces: len(ctrs)}
	if err != nil {
		st.Error = err.Error()
	}
	p.mu.Lock()
	if p.status == nil {
		p.status = map[string]Status{}
	}
	p.status[st.Exporter] = st
	p.mu.Unlock()
	if err != nil {
		return
	}
	if p.Names != nil {
		p.Names(tg.Exporter, ifs)
	}
	p.Emit(ctrs)
}

// Status returns the latest outcome per target.
func (p *Poller) Status() []Status {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]Status, 0, len(p.status))
	for _, s := range p.status {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Exporter < out[j].Exporter })
	return out
}
