package nf

import (
	"net/netip"
	"time"
)

// octetTotalCount (85) and packetTotalCount (86) are not this report's
// traffic: RFC 7012 defines them as the counts for the flow since it
// started, so an exporter that re-reports a long flow on its active timeout
// sends a running total that grows with every report. Adding those up counts
// the same bytes once more on every report — a flow still running after ten
// reports was counted about fifty times over.
//
// What the collector needs is the increase since the last report of the same
// flow, which means remembering the last total per flow. The flow's start
// time is part of what identifies it, and that is what makes this safe for
// exporters that do not behave this way: an exporter that reports each
// interval separately gives each report its own start time, so every report
// is a flow we have not seen, and its total is taken whole, exactly as
// before. Only a report that repeats a start time we have already seen is
// treated as a running total.
type totKey struct {
	src, dst     netip.Addr
	sport, dport uint16
	proto        uint8
	inIf, outIf  uint32
	startMs      int64
}

type totState struct {
	bytes, pkts uint64
}

// maxTotals caps the flows one exporter's running totals are remembered for.
// At the cap the table rotates rather than growing: entries move to a
// previous generation that is still read, and the oldest generation is
// dropped. Memory is bounded at twice the cap per exporter whatever the
// exporter sends, which matters because the flows come off the network.
const maxTotals = 1 << 17 // about 7 MB per generation

// totalsIdle rotates the table when an exporter is slow enough that the cap
// is never reached, so a flow that stopped being reported is not remembered
// for ever. It is well past any usual active timeout, so a flow still being
// reported keeps its total across a rotation.
const totalsIdle = 30 * time.Minute

// totals holds the last total seen per flow, in two generations so that the
// table can be bounded without discarding what is still in use.
type totals struct {
	cur, prev map[totKey]totState
	rotated   time.Time
}

// delta returns the traffic to record for a report carrying running totals,
// and remembers the totals for the next report of the same flow.
func (t *totals) delta(k totKey, bytes, pkts uint64, now time.Time) (uint64, uint64) {
	if t.cur == nil {
		t.cur = make(map[totKey]totState)
		t.rotated = now
	}
	if len(t.cur) >= maxTotals || now.Sub(t.rotated) >= totalsIdle {
		t.prev, t.cur = t.cur, make(map[totKey]totState)
		t.rotated = now
	}
	prev, seen := t.cur[k]
	if !seen {
		prev, seen = t.prev[k]
	}
	t.cur[k] = totState{bytes: bytes, pkts: pkts}
	if !seen {
		// First report of this flow: its total so far is its traffic so far.
		return bytes, pkts
	}
	// A total that did not grow means the exporter restarted its metering,
	// or reused the flow's identity for something new. Either way the
	// difference is meaningless and the total is the best estimate.
	if bytes < prev.bytes || pkts < prev.pkts {
		return bytes, pkts
	}
	return bytes - prev.bytes, pkts - prev.pkts
}

// totalDelta turns a report's running totals into the traffic since the same
// flow was last reported. Without a start time from the exporter two reports
// cannot be told apart from one flow reported twice, so the total is taken
// whole, which is what this did before it tracked anything.
func (s *session) totalDelta(a *acc, now time.Time) (uint64, uint64) {
	if !a.startKnown {
		return a.totBytes, a.totPkts
	}
	r := &a.rec
	k := totKey{
		src: r.Src, dst: r.Dst, sport: r.SrcPort, dport: r.DstPort, proto: r.Proto,
		inIf: r.InIf, outIf: r.OutIf, startMs: r.Start.UnixMilli(),
	}
	return s.tot.delta(k, a.totBytes, a.totPkts, now)
}
