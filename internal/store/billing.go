package store

import (
	"fmt"
	"net/netip"
	"sort"
	"strings"
	"time"
)

// Billing is the traffic of a set of service addresses (VIPs) over a
// range: what arrived at them (In), what they sent (Out), and the two
// added together (Total), which is the number a bill is written from.
//
// The figures are judged flow by flow, by both ends of each flow: a
// flow between two addresses of the same set is traffic inside the
// set, counted once in Total, where adding the per-address figures
// would count it twice — once as one address's Out and once as the
// other's In. In and Out keep it on both sides, so the split still
// reconciles: Total = In + Out − Inside. No summary by address can
// give this: it has already lost which flow a byte belonged to.
//
// While the flow records cover the range, they answer. The records
// are kept a week and a bill is written for a month, so past them
// the billing summaries answer (billfold.go): one row per hour and
// pair of addresses, folded from the records as the minutes settle,
// judged by the same two ends when the set is resolved at the time
// of the asking. The whole range comes from the one source, never a
// splice of the two, and the summaries are read by the whole hour
// buckets that cover the range, the way Top reads its summaries —
// for the hour-aligned ranges a bill is written over (a calendar
// month, a preset) the figures are the records' figures exactly.
//
// The figures are flow figures: sampled, therefore estimates, as
// Totals and every other page's are. SettledTo and DataFrom say how
// much of the range the data covers, the same marks the charts
// carry; the totals and the rates are taken over the settled part
// only, so a bill is never written from a half-arrived last minute.
type Billing struct {
	Step      int64       `json:"step"` // bucket width, seconds
	Times     []int64     `json:"times"`
	In        []float64   `json:"in"`    // bits per second arriving at the set
	Out       []float64   `json:"out"`   // bits per second sent by the set
	Total     []float64   `json:"total"` // In + Out with inside traffic counted once
	InBytes   uint64      `json:"in_bytes"`
	OutBytes  uint64      `json:"out_bytes"`
	Bytes     uint64      `json:"bytes"` // the billing figure: InBytes + OutBytes − inside bytes
	SettledTo int64       `json:"settled_to,omitempty"`
	DataFrom  int64       `json:"data_from,omitempty"`
	Members   []MemberRow `json:"members,omitempty"`
}

// MemberRow is one address of the set and its own in and out, for the
// split under the product line's total. A flow between two members is
// in one member's Out and the other's In, so the members' totals add up
// to more than the set's Bytes: the difference is the inside traffic.
type MemberRow struct {
	Addr     string `json:"addr"`
	InBytes  uint64 `json:"in_bytes"`
	OutBytes uint64 `json:"out_bytes"`
	Bytes    uint64 `json:"bytes"` // InBytes + OutBytes of this address alone
}

// Billing returns the traffic of the given addresses over q's range.
// The addresses are the whole selection: one VIP, or every VIP of a
// product line. Filters on q other than its range and zone are ignored;
// the set is the filter.
func (s *Store) Billing(q Query, addrs []netip.Addr) (*Billing, error) {
	if len(addrs) == 0 {
		return nil, fmt.Errorf("no addresses")
	}
	if len(addrs) > 256 {
		return nil, fmt.Errorf("too many addresses: %d", len(addrs))
	}
	set := make(map[string]bool, len(addrs))
	names := make([]string, 0, len(addrs))
	for _, a := range addrs {
		a = a.Unmap()
		if !set[a.String()] {
			set[a.String()] = true
			names = append(names, a.String())
		}
	}
	sort.Strings(names)

	step := StepFor(q.span())
	summary := s.billingSummary(q)
	if summary && step < time.Hour {
		// the summaries are kept by the hour: a range they answer
		// is bucketed by the hour at finest, where the records
		// bucket a short range by the minute
		step = time.Hour
	}
	start := q.localTrunc(q.From, step)
	settled := settledTo(q.To, false)
	n := bucketCount(start, q.To, step)
	b := &Billing{Step: int64(step.Seconds())}
	for i := 0; i < n; i++ {
		b.Times = append(b.Times, start.Add(time.Duration(i)*step).UnixMilli())
	}
	inB := make([]float64, n)   // bytes per bucket
	outB := make([]float64, n)  // bytes per bucket
	insB := make([]float64, n)  // bytes per bucket with both ends in the set
	mIn := map[string]uint64{}  // settled bytes per member
	mOut := map[string]uint64{} // settled bytes per member

	var err error
	if summary {
		err = s.billingSummaryRows(q, names, start, step, inB, outB, insB, mIn, mOut)
	} else {
		err = s.billingRecordRows(q, names, start, step, settled, inB, outB, insB, mIn, mOut)
	}
	if err != nil {
		return nil, err
	}

	dataFrom := s.DataStart()
	b.In = make([]float64, n)
	b.Out = make([]float64, n)
	b.Total = make([]float64, n)
	for i := 0; i < n; i++ {
		secs := bucketSecs(start, i, step, q.To, dataFrom)
		b.In[i] = inB[i] * 8 / secs
		b.Out[i] = outB[i] * 8 / secs
		b.Total[i] = (inB[i] + outB[i] - insB[i]) * 8 / secs
		// the totals are the settled part only: a bucket that starts at
		// or after the settled boundary is still filling
		bt := start.Add(time.Duration(i) * step)
		if bt.Before(settled) {
			b.InBytes += uint64(inB[i])
			b.OutBytes += uint64(outB[i])
			b.Bytes += uint64(inB[i] + outB[i] - insB[i])
		}
	}
	b.SettledTo = unsettledMark(settled, q.To)
	b.DataFrom = dataFromMark(start, dataFrom)
	for _, nm := range names {
		b.Members = append(b.Members, MemberRow{Addr: nm, InBytes: mIn[nm], OutBytes: mOut[nm], Bytes: mIn[nm] + mOut[nm]})
	}
	return b, nil
}

// billingSummary reports whether q is answered from the billing
// summaries: it starts before the flow records do, and the
// summaries reach back at least as far as the records. Before an
// upgraded installation's backfill is done, and on a store too
// young for retention to have bitten, the records answer, as they
// did before there were summaries.
func (s *Store) billingSummary(q Query) bool {
	if !s.billReady.Load() {
		return false
	}
	bs := s.billStart()
	if bs.IsZero() {
		return false
	}
	ds := s.DetailStart()
	if ds.IsZero() {
		return true // no records left: the summaries are all there is
	}
	return q.From.Before(ds) && !bs.After(ds)
}

// billingInList is the SQL address list of the set and its arguments,
// built once per appearance in a query's text.
func billingInList(names []string) (string, []any) {
	ph := make([]string, len(names))
	one := make([]any, len(names))
	for i, nm := range names {
		ph[i] = "?"
		one[i] = nm
	}
	return "(" + strings.Join(ph, ", ") + ")", one
}

// billingRecordRows fills the buckets and the member split from the
// flow records over q's range.
func (s *Store) billingRecordRows(q Query, names []string, start time.Time, step time.Duration, settled time.Time, inB, outB, insB []float64, mIn, mOut map[string]uint64) error {
	n := len(inB)
	inList, one := billingInList(names)
	// one row per bucket and direction class: 1 = arriving at the set
	// from outside it, 2 = sent by the set to outside it, 3 = inside the
	// set (both ends in it). The classes are disjoint, so nothing is
	// counted twice within a class and the three add up to every byte
	// that touched the set, once.
	bucket := q.bucketSQL(step)
	src := s.Source(q.From, q.To)
	dup := ""
	if !q.IncludeDup {
		dup = " AND NOT dup"
	}
	sqlq := fmt.Sprintf(`SELECT %s AS b,
		CASE WHEN src IN %s AND dst IN %s THEN 3 WHEN dst IN %s THEN 1 ELSE 2 END AS cls,
		CAST(sum(wire) AS DOUBLE)
		FROM %s WHERE ts >= ? AND ts < ?
		AND (src IN %s OR dst IN %s)%s GROUP BY 1, 2`,
		bucket, inList, inList, inList, src, inList, inList, dup)
	// the IN lists take the address arguments once per appearance, in
	// the order the placeholders appear in the text: the CASE's three
	// lists come before the WHERE's time bounds and its two lists
	full := []any{}
	for i := 0; i < 3; i++ {
		full = append(full, one...)
	}
	full = append(full, q.From.UTC(), q.To.UTC())
	for i := 0; i < 2; i++ {
		full = append(full, one...)
	}
	rows, err := s.DB.Query(sqlq, full...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var t time.Time
		var cls int
		var w float64
		if err := rows.Scan(&t, &cls, &w); err != nil {
			return err
		}
		ti := int(t.UTC().Sub(start) / step)
		if ti < 0 || ti >= n {
			continue
		}
		switch cls {
		case 1:
			inB[ti] += w
		case 2:
			outB[ti] += w
		case 3:
			insB[ti] += w
			inB[ti] += w
			outB[ti] += w
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	// the per-member split, over the settled part only, like the totals
	mq := fmt.Sprintf(`SELECT dst, CAST(sum(wire) AS UBIGINT) FROM %s
		WHERE ts >= ? AND ts < ? AND dst IN %s%s GROUP BY 1`, src, inList, dup)
	margs := []any{q.From.UTC(), settled}
	margs = append(margs, one...)
	mrows, err := s.DB.Query(mq, margs...)
	if err != nil {
		return err
	}
	defer mrows.Close()
	for mrows.Next() {
		var a string
		var w uint64
		if err := mrows.Scan(&a, &w); err != nil {
			return err
		}
		mIn[a] = w
	}
	if err := mrows.Err(); err != nil {
		return err
	}
	mq = fmt.Sprintf(`SELECT src, CAST(sum(wire) AS UBIGINT) FROM %s
		WHERE ts >= ? AND ts < ? AND src IN %s%s GROUP BY 1`, src, inList, dup)
	mrows2, err := s.DB.Query(mq, margs...)
	if err != nil {
		return err
	}
	defer mrows2.Close()
	for mrows2.Next() {
		var a string
		var w uint64
		if err := mrows2.Scan(&a, &w); err != nil {
			return err
		}
		mOut[a] = w
	}
	return mrows2.Err()
}

// billingSummaryRows fills the buckets and the member split from the
// billing summaries, over the whole hour buckets that cover q's
// range. The direction classes are the records' own, judged by both
// ends of each folded pair of addresses; the member split covers
// the same buckets, and the totals above take the settled part of
// them, as they do on the records path.
func (s *Store) billingSummaryRows(q Query, names []string, start time.Time, step time.Duration, inB, outB, insB []float64, mIn, mOut map[string]uint64) error {
	n := len(inB)
	ef, et := hourRange(q.From, q.To)
	inList, one := billingInList(names)
	dup := ""
	if !q.IncludeDup {
		dup = " AND NOT dup"
	}
	bucket := q.bucketSQL(step)
	sqlq := fmt.Sprintf(`SELECT %s AS b,
		CASE WHEN src IN %s AND dst IN %s THEN 3 WHEN dst IN %s THEN 1 ELSE 2 END AS cls,
		CAST(sum(wire) AS DOUBLE)
		FROM r_bill WHERE ts >= ? AND ts < ?
		AND (src IN %s OR dst IN %s)%s GROUP BY 1, 2`,
		bucket, inList, inList, inList, inList, inList, dup)
	full := []any{}
	for i := 0; i < 3; i++ {
		full = append(full, one...)
	}
	full = append(full, ef, et)
	for i := 0; i < 2; i++ {
		full = append(full, one...)
	}
	rows, err := s.DB.Query(sqlq, full...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var t time.Time
		var cls int
		var w float64
		if err := rows.Scan(&t, &cls, &w); err != nil {
			return err
		}
		ti := int(t.UTC().Sub(start) / step)
		if ti < 0 || ti >= n {
			continue
		}
		switch cls {
		case 1:
			inB[ti] += w
		case 2:
			outB[ti] += w
		case 3:
			insB[ti] += w
			inB[ti] += w
			outB[ti] += w
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	mq := fmt.Sprintf(`SELECT dst, CAST(sum(wire) AS UBIGINT) FROM r_bill
		WHERE ts >= ? AND ts < ? AND dst IN %s%s GROUP BY 1`, inList, dup)
	margs := []any{ef, et}
	margs = append(margs, one...)
	mrows, err := s.DB.Query(mq, margs...)
	if err != nil {
		return err
	}
	defer mrows.Close()
	for mrows.Next() {
		var a string
		var w uint64
		if err := mrows.Scan(&a, &w); err != nil {
			return err
		}
		mIn[a] = w
	}
	if err := mrows.Err(); err != nil {
		return err
	}
	mq = fmt.Sprintf(`SELECT src, CAST(sum(wire) AS UBIGINT) FROM r_bill
		WHERE ts >= ? AND ts < ? AND src IN %s%s GROUP BY 1`, inList, dup)
	mrows2, err := s.DB.Query(mq, margs...)
	if err != nil {
		return err
	}
	defer mrows2.Close()
	for mrows2.Next() {
		var a string
		var w uint64
		if err := mrows2.Scan(&a, &w); err != nil {
			return err
		}
		mOut[a] = w
	}
	return mrows2.Err()
}
