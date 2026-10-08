// Package capture turns packets seen on a local interface into flow
// records, like a small software flow probe. Packets are not deduplicated.
package capture

import (
	"context"
	"hash/fnv"
	"net/netip"
	"sync"
	"sync/atomic"
	"time"

	"github.com/githubflyideas/traffic66/internal/flow"
	"github.com/githubflyideas/traffic66/internal/packet"
)

// Sink receives flow records.
type Sink interface{ Submit([]flow.Record) }

// source is an OS packet source returning Ethernet frames.
type source interface {
	// Read returns frames (header bytes) with their full wire lengths.
	Read(buf []byte, fn func(frame []byte, wireLen int)) error
	Close() error
	Method() string
	Drops() uint64
}

type key struct {
	src, dst     netip.Addr
	sport, dport uint16
	proto        uint8
	vlan         uint16
	encap        flow.Encap
}

type entry struct {
	first, last time.Time
	pkts, bytes uint64
	flags       uint8
	opener      int8 // kept across active-timeout exports
}

// Capture is one running interface capture.
type Capture struct {
	Iface   string
	Method  string
	Packets atomic.Uint64
	Dropped atomic.Uint64
	src     source
	sink    Sink
	domain  uint32

	mu     sync.Mutex
	table  map[key]*entry
	errMsg atomic.Value
}

const (
	activeTimeout = 60 * time.Second
	idleTimeout   = 15 * time.Second
	maxEntries    = 200000
)

// Exporter is the address recorded for locally captured flows.
var Exporter = netip.MustParseAddr("127.0.0.1")

// Start opens iface and runs until ctx ends.
func Start(ctx context.Context, iface string, sink Sink) (*Capture, error) {
	src, err := open(iface)
	if err != nil {
		return nil, err
	}
	h := fnv.New32a()
	h.Write([]byte(iface))
	c := &Capture{Iface: iface, Method: src.Method(), src: src, sink: sink, table: map[key]*entry{}, domain: h.Sum32()}
	go func() { <-ctx.Done(); src.Close() }()
	go c.readLoop(ctx)
	go c.expireLoop(ctx)
	return c, nil
}

func (c *Capture) Err() string {
	if v, ok := c.errMsg.Load().(string); ok {
		return v
	}
	return ""
}

func (c *Capture) Active() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.table)
}

func (c *Capture) readLoop(ctx context.Context) {
	buf := make([]byte, 1<<20)
	for ctx.Err() == nil {
		err := c.src.Read(buf, func(frame []byte, wire int) {
			c.Packets.Add(1)
			var r flow.Record
			if !packet.Ethernet(frame, &r) {
				return
			}
			k := key{r.Src, r.Dst, r.SrcPort, r.DstPort, r.Proto, r.VLAN, r.Encap}
			now := time.Now()
			c.mu.Lock()
			e := c.table[k]
			if e == nil {
				if len(c.table) >= maxEntries {
					c.mu.Unlock()
					c.Dropped.Add(1)
					return
				}
				e = &entry{first: now, opener: opener(r.TCPFlags)}
				c.table[k] = e
			}
			e.last = now
			e.pkts++
			e.bytes += uint64(wire) + 4 // captured frames exclude the FCS
			e.flags |= r.TCPFlags
			c.mu.Unlock()
		})
		if err != nil && ctx.Err() == nil {
			c.errMsg.Store(err.Error())
			time.Sleep(time.Second)
		}
	}
}

func (c *Capture) expireLoop(ctx context.Context) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			c.flush(time.Now(), true)
			return
		case now := <-t.C:
			c.flush(now, false)
			c.Dropped.Store(c.src.Drops())
		}
	}
}

func (c *Capture) flush(now time.Time, all bool) {
	var out []flow.Record
	c.mu.Lock()
	// first learn who opened each connection from either direction, before
	// any entry is removed below
	for k, e := range c.table {
		if e.opener == 0 && e.pkts > 0 {
			if re := c.table[key{k.dst, k.src, k.dport, k.sport, k.proto, k.vlan, k.encap}]; re != nil {
				e.opener = -re.opener
			}
		}
	}
	for k, e := range c.table {
		idle := now.Sub(e.last) >= idleTimeout
		active := now.Sub(e.first) >= activeTimeout
		if !all && !idle && !active {
			continue
		}
		if e.pkts == 0 {
			if idle || all {
				delete(c.table, k)
			}
			continue
		}
		op := e.opener
		out = append(out, flow.Record{Start: e.first, End: e.last, Src: k.src, Dst: k.dst, SrcPort: k.sport, DstPort: k.dport,
			Proto: k.proto, VLAN: k.vlan, Encap: k.encap, TCPFlags: e.flags, Bytes: e.bytes, Packets: e.pkts, L2: true, Opener: op,
			Sampling: 1, Mult: 1, SamplingKnown: true, Exporter: Exporter, Domain: c.domain, Source: flow.SrcCapture, Direction: 255})
		if idle || all {
			delete(c.table, k)
		} else {
			*e = entry{first: now, last: now, opener: op}
		}
	}
	c.mu.Unlock()
	c.sink.Submit(out)
}

// opener reads the first packet of a TCP connection seen in one direction:
// a SYN alone opens it (1), a SYN+ACK answers (-1); anything else says
// nothing (0).
func opener(flags uint8) int8 {
	switch flags & 0x12 {
	case 0x02:
		return 1
	case 0x12:
		return -1
	}
	return 0
}
