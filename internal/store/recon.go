package store

import (
	"fmt"
	"math"
	"sort"
	"time"
)

// IfaceInfo is one interface known from counters or flows.
type IfaceInfo struct {
	Exporter string `json:"exporter"`
	IfIndex  uint32 `json:"ifindex"`
	Speed    uint64 `json:"speed"`
	Counters bool   `json:"counters"`
	InWire   uint64 `json:"in_wire"`
	OutWire  uint64 `json:"out_wire"`
}

// Interfaces lists interfaces seen in the range, busiest first.
func (s *Store) Interfaces(q Query) ([]IfaceInfo, error) {
	m := map[string]*IfaceInfo{}
	key := func(e string, i uint32) string { return fmt.Sprintf("%s/%d", e, i) }
	rows, err := s.DB.Query(`SELECT exporter, ifindex, max(speed) FROM ifc WHERE ts >= ? AND ts < ? GROUP BY 1,2`, q.From.UTC(), q.To.UTC())
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var f IfaceInfo
		if err := rows.Scan(&f.Exporter, &f.IfIndex, &f.Speed); err != nil {
			rows.Close()
			return nil, err
		}
		f.Counters = true
		m[key(f.Exporter, f.IfIndex)] = &f
	}
	rows.Close()
	src := s.Source(q.From, q.To)
	for _, side := range []string{"in_if", "out_if"} {
		rows, err := s.DB.Query(fmt.Sprintf(`SELECT exporter, %s, sum(wire) FROM %s WHERE ts >= ? AND ts < ? AND %s <> 0 GROUP BY 1,2 ORDER BY 3 DESC LIMIT 200`, side, src, side), q.From.UTC(), q.To.UTC())
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var e string
			var idx uint32
			var w uint64
			if err := rows.Scan(&e, &idx, &w); err != nil {
				rows.Close()
				return nil, err
			}
			f := m[key(e, idx)]
			if f == nil {
				f = &IfaceInfo{Exporter: e, IfIndex: idx}
				m[key(e, idx)] = f
			}
			if side == "in_if" {
				f.InWire = w
			} else {
				f.OutWire = w
			}
		}
		rows.Close()
	}
	out := make([]IfaceInfo, 0, len(m))
	for _, f := range m {
		out = append(out, *f)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Counters != out[j].Counters {
			return out[i].Counters
		}
		return out[i].InWire+out[i].OutWire > out[j].InWire+out[j].OutWire
	})
	return out, nil
}

// Recon compares one interface's counters with the flow estimate.
type Recon struct {
	Exporter   string    `json:"exporter"`
	IfIndex    uint32    `json:"ifindex"`
	Step       int64     `json:"step"`
	Times      []int64   `json:"times"`
	InCounter  []float64 `json:"in_counter"` // bits/s, NaN-free: -1 = no counter data
	InFlow     []float64 `json:"in_flow"`
	OutCounter []float64 `json:"out_counter"`
	OutFlow    []float64 `json:"out_flow"`
	// Summary over buckets where both sides have data.
	InDev       float64 `json:"in_dev"` // (flow-counter)/counter
	OutDev      float64 `json:"out_dev"`
	InCtrBytes  uint64  `json:"in_ctr_bytes"`
	InFlowBytes uint64  `json:"in_flow_bytes"`
	InPkts      uint64  `json:"in_pkts"`
	InSamples   float64 `json:"in_samples"`
	InStatErr   float64 `json:"in_stat_err"` // approx. 95% relative error of the flow estimate
	OutStatErr  float64 `json:"out_stat_err"`
	HasCounters bool    `json:"has_counters"`
}

func (s *Store) Recon(q Query, exporter string, ifindex uint32) (*Recon, error) {
	step := StepFor(q.span())
	if step < 5*time.Minute {
		step = 5 * time.Minute
	}
	start := q.From.UTC().Truncate(step)
	q.From = start
	n := int(q.To.UTC().Sub(start)/step) + 1
	// the last bucket counts only if it is complete
	complete := func(i int) bool { return !start.Add(time.Duration(i+1) * step).After(q.To.UTC()) }
	r := &Recon{Exporter: exporter, IfIndex: ifindex, Step: int64(step.Seconds())}
	r.Times = make([]int64, n)
	for i := range r.Times {
		r.Times[i] = start.Add(time.Duration(i) * step).UnixMilli()
	}
	inC, outC := make([]float64, n), make([]float64, n)
	covered := make([]float64, n) // seconds of each bucket covered by counters

	// Counter deltas spread over each polling interval.
	rows, err := s.DB.Query(`SELECT ts, in_oct, out_oct FROM ifc WHERE exporter = ? AND ifindex = ? AND ts >= ? AND ts < ? ORDER BY ts`,
		exporter, ifindex, q.From.UTC().Add(-10*time.Minute), q.To.UTC())
	if err != nil {
		return nil, err
	}
	var pt time.Time
	var pin, pout uint64
	first := true
	for rows.Next() {
		var t time.Time
		var in, out uint64
		if err := rows.Scan(&t, &in, &out); err != nil {
			rows.Close()
			return nil, err
		}
		t = t.UTC()
		if !first && in >= pin && out >= pout && t.After(pt) && t.Sub(pt) < time.Hour {
			span := t.Sub(pt).Seconds()
			din, dout := float64(in-pin), float64(out-pout)
			for b := pt.Truncate(step); b.Before(t); b = b.Add(step) {
				lo, hi := maxT(b, pt), minT(b.Add(step), t)
				ov := hi.Sub(lo).Seconds()
				i := int(b.Sub(start) / step)
				if ov <= 0 || i < 0 || i >= n {
					continue
				}
				inC[i] += din * ov / span
				outC[i] += dout * ov / span
				covered[i] += ov
			}
		}
		pt, pin, pout, first = t, in, out, false
	}
	rows.Close()

	inF, outF := make([]float64, n), make([]float64, n)
	var inPk, inSamp, outSamp float64
	src := s.Source(q.From, q.To)
	bucket := fmt.Sprintf("time_bucket(INTERVAL '%d seconds', ts)", int64(step.Seconds()))
	for side, dst := range map[string][]float64{"in_if": inF, "out_if": outF} {
		rows, err := s.DB.Query(fmt.Sprintf(`SELECT %s, sum(wire), sum(pkts), sum(pkts / greatest(mult, 1)) FROM %s
			WHERE ts >= ? AND ts < ? AND exporter = ? AND %s = ? GROUP BY 1`, bucket, src, side),
			q.From.UTC(), q.To.UTC(), exporter, ifindex)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var b time.Time
			var w, pk uint64
			var samp float64
			if err := rows.Scan(&b, &w, &pk, &samp); err != nil {
				rows.Close()
				return nil, err
			}
			i := int(b.UTC().Sub(start) / step)
			if i < 0 || i >= n {
				continue
			}
			dst[i] = float64(w)
			if !complete(i) {
				continue
			}
			if side == "in_if" && covered[i] >= step.Seconds()*0.9 {
				inPk += float64(pk)
				inSamp += samp
			}
			if side == "out_if" && covered[i] >= step.Seconds()*0.9 {
				outSamp += samp
			}
		}
		rows.Close()
	}
	var ic, iflw, oc, oflw float64
	r.InCounter, r.OutCounter = make([]float64, n), make([]float64, n)
	r.InFlow, r.OutFlow = make([]float64, n), make([]float64, n)
	for i := 0; i < n; i++ {
		bits := 8 / step.Seconds()
		r.InFlow[i], r.OutFlow[i] = inF[i]*bits, outF[i]*bits
		if covered[i] < step.Seconds()*0.9 || !complete(i) {
			r.InCounter[i], r.OutCounter[i] = -1, -1
			if !complete(i) {
				r.InFlow[i], r.OutFlow[i] = -1, -1
			}
			continue
		}
		r.HasCounters = true
		r.InCounter[i], r.OutCounter[i] = inC[i]*bits, outC[i]*bits
		ic += inC[i]
		iflw += inF[i]
		oc += outC[i]
		oflw += outF[i]
	}
	if ic > 0 {
		r.InDev = (iflw - ic) / ic
	}
	if oc > 0 {
		r.OutDev = (oflw - oc) / oc
	}
	r.InCtrBytes, r.InFlowBytes, r.InPkts, r.InSamples = uint64(ic), uint64(iflw), uint64(inPk), inSamp
	if inSamp > 0 {
		r.InStatErr = 1.96 / math.Sqrt(inSamp)
	}
	if outSamp > 0 {
		r.OutStatErr = 1.96 / math.Sqrt(outSamp)
	}
	return r, nil
}

func maxT(a, b time.Time) time.Time {
	if a.After(b) {
		return a
	}
	return b
}
func minT(a, b time.Time) time.Time {
	if a.Before(b) {
		return a
	}
	return b
}
