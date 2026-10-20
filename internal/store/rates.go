package store

import (
	"fmt"
	"time"
)

// Interface rates: a bill needs a month, the snapshots do not last one.
//
// ifc holds cumulative counter snapshots and is deleted with the flow
// detail at RawDays (30 by default). Thirty days cannot cover a calendar
// month, let alone "last month" asked for on the second of this one. What
// a 95th-percentile bill is made of is small, though: one row per interface
// per five minutes is 288 a day. So the snapshots are turned into rates
// while they are still here, and the rates are what is kept — for
// RollupDays (400 by default), which covers thirteen months.
//
// What the derivation drops rather than guesses:
//
//   - A counter that went backwards. Device counters are often 32 bits and
//     wrap; a device that reboots starts again from zero. Both look the
//     same from here, so adding 2^32 to the delta would invent traffic
//     every time the cause was a reboot. One interval is half a minute of a
//     five-minute bucket, so dropping it costs little and claims nothing.
//   - An interval longer than fifteen minutes. That is a hole in collection,
//     not a measurement of it.
//   - A rate above the interface's own speed. Those come from a counter
//     reset the backwards test cannot see, or from two devices reporting
//     under one ifIndex. The line sits a tenth over the speed so that a
//     snapshot timestamp a second out does not cost the whole bucket.
//
// samples says how many intervals a bucket was made of, so a report can say
// how much of the period it is speaking for. Nothing here invents a zero:
// an interface that reported nothing has no row, which is not the same as
// an interface that reported no traffic. A report that treats the two alike
// moves the 95th percentile, and in the customer's favour, so the choice
// belongs where it can be written down next to the number.
const (
	rateBucket = 5 * time.Minute

	// How far back each pass recomputes. The pass runs every five minutes,
	// so this is only reach for late snapshots and for passes that failed:
	// a process that was not running wrote no snapshots either, which is
	// why there is no watermark to catch up from.
	rateLookback = 2 * time.Hour

	// Longer than this between two snapshots is a collection gap.
	rateMaxGap = 15 * time.Minute
)

// ifrInsert derives one span of buckets. The reach back of rateBucket on
// the read is for lag(): the first snapshot of a partition has nothing to
// subtract from, so without an earlier row the span would lose its opening
// interval. Buckets before the span are filtered out again after the join.
const ifrInsert = `INSERT INTO ifr
WITH s AS (
	SELECT ts, exporter, ifindex, speed, in_oct, out_oct,
		lag(ts) OVER w AS pts, lag(in_oct) OVER w AS pin, lag(out_oct) OVER w AS pout
	FROM ifc WHERE ts >= ? AND ts < ?
	WINDOW w AS (PARTITION BY exporter, ifindex ORDER BY ts)
), d AS (
	SELECT time_bucket(INTERVAL 5 MINUTE, ts) AS bucket, exporter, ifindex, speed,
		epoch(ts) - epoch(pts) AS dt,
		CAST(in_oct AS HUGEINT) - CAST(pin AS HUGEINT) AS din,
		CAST(out_oct AS HUGEINT) - CAST(pout AS HUGEINT) AS dout
	FROM s WHERE pts IS NOT NULL
)
SELECT bucket, exporter, ifindex,
	8 * CAST(sum(din) AS DOUBLE) / sum(dt), 8 * CAST(sum(dout) AS DOUBLE) / sum(dt), count(*)
FROM d
WHERE dt > 0 AND dt <= ? AND din >= 0 AND dout >= 0
	AND (speed = 0 OR 8 * CAST(greatest(din, dout) AS DOUBLE) <= CAST(speed AS DOUBLE) * dt * 1.1)
	AND bucket >= ?
GROUP BY bucket, exporter, ifindex`

// RollRates turns counter snapshots into five-minute rate buckets. It is
// safe to run again over a span it has already done: the span is deleted
// first, so a second pass replaces its own work rather than doubling it.
func (s *Store) RollRates(now time.Time) error {
	to := now.UTC().Truncate(rateBucket) // the bucket in progress is not finished
	from := to.Add(-rateLookback)
	tx, err := s.DB.Begin()
	if err != nil {
		return fmt.Errorf("rates: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`DELETE FROM ifr WHERE ts >= ? AND ts < ?`, from, to); err != nil {
		return fmt.Errorf("rates: clear %s..%s: %w", from.Format(time.RFC3339), to.Format(time.RFC3339), err)
	}
	if _, err := tx.Exec(ifrInsert, from.Add(-rateBucket), to, rateMaxGap.Seconds(), from); err != nil {
		return fmt.Errorf("rates: derive: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("rates: %w", err)
	}
	return nil
}

// Bill is what one interface would be charged for over a period, and
// enough of the working to argue with.
type Bill struct {
	Exporter string  `json:"exporter"`
	IfIndex  uint32  `json:"ifindex"`
	In95     float64 `json:"in95"`
	Out95    float64 `json:"out95"`
	Billed   float64 `json:"billed"` // the greater of the two
	Dir      string  `json:"dir"`    // which of them it is
	InMax    float64 `json:"in_max"`
	OutMax   float64 `json:"out_max"`
	InAvg    float64 `json:"in_avg"`
	OutAvg   float64 `json:"out_avg"`
	Buckets  int     `json:"buckets"`  // five-minute buckets that hold a measurement
	Possible int     `json:"possible"` // buckets the period could have held
	Samples  int64   `json:"samples"`  // counter intervals behind those buckets
}

// Coverage is the share of the period that was actually measured. A bill
// taken over half a month is not wrong, but it is not a month either, and
// the report says so rather than leaving the reader to assume.
func (b Bill) Coverage() float64 {
	if b.Possible <= 0 {
		return 0
	}
	return float64(b.Buckets) / float64(b.Possible)
}

// Bills computes the 95th percentile per interface over [from, to).
//
// The number is the usual one: sort the five-minute samples, discard the
// top five per cent, charge the highest that is left. quantile_disc returns
// a sample that was actually measured rather than an interpolation between
// two, which is what makes the figure arguable against the device's own
// counters. In and out are percentiled separately and the greater is
// charged, which is how carriers word it; the percentile of the larger of
// each pair would be a different, higher number, and not the common term.
//
// Buckets with no measurement are absent, not zero. Counting them as zero
// would push the percentile down, in the paying customer's favour, which is
// exactly why it cannot be done quietly: Coverage is here so the report can
// show what the number rests on.
func (s *Store) Bills(from, to time.Time, exporter string) ([]Bill, error) {
	if !to.After(from) {
		return nil, fmt.Errorf("bills: %s is not before %s", from.Format(time.RFC3339), to.Format(time.RFC3339))
	}
	q := `SELECT exporter, ifindex,
		quantile_disc(in_bps, 0.95), quantile_disc(out_bps, 0.95),
		max(in_bps), max(out_bps), avg(in_bps), avg(out_bps), count(*), sum(samples)
		FROM ifr WHERE ts >= ? AND ts < ?`
	args := []any{from.UTC(), to.UTC()}
	if exporter != "" {
		q += ` AND exporter = ?`
		args = append(args, exporter)
	}
	q += ` GROUP BY exporter, ifindex ORDER BY greatest(quantile_disc(in_bps, 0.95), quantile_disc(out_bps, 0.95)) DESC`
	rows, err := s.DB.Query(q, args...)
	if err != nil {
		return nil, fmt.Errorf("bills: %w", err)
	}
	defer rows.Close()
	possible := int(to.Sub(from) / rateBucket)
	var out []Bill
	for rows.Next() {
		b := Bill{Possible: possible}
		if err := rows.Scan(&b.Exporter, &b.IfIndex, &b.In95, &b.Out95, &b.InMax, &b.OutMax, &b.InAvg, &b.OutAvg, &b.Buckets, &b.Samples); err != nil {
			return nil, fmt.Errorf("bills: %w", err)
		}
		b.Billed, b.Dir = b.In95, "in"
		if b.Out95 > b.In95 {
			b.Billed, b.Dir = b.Out95, "out"
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

// BillPoint is one five-minute bucket of an interface's own counters.
type BillPoint struct {
	TS      time.Time `json:"ts"`
	In      float64   `json:"in"`  // bits/s
	Out     float64   `json:"out"` // bits/s
	Samples int       `json:"samples"`
}

// BillSeries returns the buckets one interface's bill was computed from,
// oldest first. It is the working behind the figure: the same rows Bills
// takes its percentile of, so a customer who disputes a bill can be shown
// the line it was drawn across rather than asked to take the number.
//
// Buckets with no measurement are absent here as they are there, rather
// than being sent as zero: a chart that draws them flat at the bottom
// claims the circuit was idle when nothing was collected.
func (s *Store) BillSeries(from, to time.Time, exporter string, ifindex uint32) ([]BillPoint, error) {
	if !to.After(from) {
		return nil, fmt.Errorf("bill series: %s is not before %s", from.Format(time.RFC3339), to.Format(time.RFC3339))
	}
	rows, err := s.DB.Query(`SELECT ts, in_bps, out_bps, samples FROM ifr
		WHERE ts >= ? AND ts < ? AND exporter = ? AND ifindex = ? ORDER BY ts`,
		from.UTC(), to.UTC(), exporter, ifindex)
	if err != nil {
		return nil, fmt.Errorf("bill series: %w", err)
	}
	defer rows.Close()
	out := []BillPoint{}
	for rows.Next() {
		var p BillPoint
		if err := rows.Scan(&p.TS, &p.In, &p.Out, &p.Samples); err != nil {
			return nil, fmt.Errorf("bill series: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
