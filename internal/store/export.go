package store

import (
	"math"
	"sort"
	"time"
)

// IfaceHour is one hour of one interface's device counters: how much the
// device itself says passed through it in that hour, in each direction.
// It is the row of the rollup export — the counters are the one
// per-interface record kept for the whole rollup retention (180 days),
// long after the flow detail (30 days) is gone.
type IfaceHour struct {
	Start    time.Time // hour start, an instant; the caller renders it in the viewer's zone
	Exporter string
	IfIndex  uint32
	InOct    uint64
	OutOct   uint64
	InPkts   uint64
	OutPkts  uint64
}

// IfaceHours adds up the interface counters into whole hours over the
// query's window. Counter snapshots are differences waiting to happen:
// the traffic of an interval between two snapshots belongs to the hours
// the interval overlaps, in proportion. An interval counts only when
// every counter rose (a fall is a reset or a wrap, whose size is
// unknowable) and the gap is under an hour — the same rule the
// reconciliation page applies, so the export and the page agree. The
// read reaches one snapshot past both ends of the window, so the
// intervals the window cuts into are clipped, not dropped.
func (s *Store) IfaceHours(q Query) ([]IfaceHour, error) {
	from, to := q.From.UTC(), q.To.UTC()
	rows, err := s.DB.Query(`SELECT ts, exporter, ifindex, in_oct, out_oct, in_pkts, out_pkts FROM ifc
		WHERE ts >= ? AND ts < ? ORDER BY exporter, ifindex, ts`,
		from.Add(-time.Hour), to.Add(time.Hour))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	type acc struct {
		inOct, outOct, inPkts, outPkts float64
	}
	m := map[IfaceHour]*acc{}
	key := func(b time.Time, e string, i uint32) IfaceHour { return IfaceHour{Start: b, Exporter: e, IfIndex: i} }
	var pe string
	var pi uint32
	var pt time.Time
	var pin, pout, pinp, poutp uint64
	have := false
	for rows.Next() {
		var t time.Time
		var e string
		var idx uint32
		var in, out, inp, outp uint64
		if err := rows.Scan(&t, &e, &idx, &in, &out, &inp, &outp); err != nil {
			return nil, err
		}
		t = t.UTC()
		if have && e == pe && idx == pi && in >= pin && out >= pout && inp >= pinp && outp >= poutp &&
			t.After(pt) && t.Sub(pt) < time.Hour {
			lo, hi := pt, t
			if lo.Before(from) {
				lo = from
			}
			if hi.After(to) {
				hi = to
			}
			if hi.After(lo) {
				span := t.Sub(pt).Seconds()
				din, dout := float64(in-pin), float64(out-pout)
				dinp, doutp := float64(inp-pinp), float64(outp-poutp)
				for b := q.localTrunc(lo, time.Hour); b.Before(hi); b = b.Add(time.Hour) {
					o := minT(b.Add(time.Hour), hi).Sub(maxT(b, lo)).Seconds()
					if o <= 0 {
						continue
					}
					k := key(b, e, idx)
					a := m[k]
					if a == nil {
						a = &acc{}
						m[k] = a
					}
					a.inOct += din * o / span
					a.outOct += dout * o / span
					a.inPkts += dinp * o / span
					a.outPkts += doutp * o / span
				}
			}
		}
		pe, pi, pt, pin, pout, pinp, poutp, have = e, idx, t, in, out, inp, outp, true
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	out := make([]IfaceHour, 0, len(m))
	for k, a := range m {
		r := k
		r.InOct = uint64(math.Round(a.inOct))
		r.OutOct = uint64(math.Round(a.outOct))
		r.InPkts = uint64(math.Round(a.inPkts))
		r.OutPkts = uint64(math.Round(a.outPkts))
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].Start.Equal(out[j].Start) {
			return out[i].Start.Before(out[j].Start)
		}
		if out[i].Exporter != out[j].Exporter {
			return out[i].Exporter < out[j].Exporter
		}
		return out[i].IfIndex < out[j].IfIndex
	})
	return out, nil
}
