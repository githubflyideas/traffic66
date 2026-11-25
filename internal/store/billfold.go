package store

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"
)

// The billing summaries: what the traffic statistics are answered
// from once the flow records of a range are gone.
//
// Billing judges every flow by both its ends — arriving at the set
// from outside it, sent by the set to outside it, or inside the set,
// where it counts once — and no summary by address keeps that: the
// per-address figures have already lost which flow a byte belonged
// to, and the inside traffic cannot be had back from them. The flow
// records keep it, but they are kept a week and a bill is written
// for a month, so the judgement's material is folded into r_bill as
// the minutes settle, under the same watermark as the conversation
// summaries: one row per hour and pair of addresses, the wire bytes
// of the flows between them, duplicates kept apart as the records
// keep them. The set itself is resolved when the bill is asked for,
// as it is against the records, so a product line edited today bills
// its whole history by its new membership, exactly as it did when
// the records answered.
//
// The fold, the seal's rebuild, the crash recovery and the upgrade
// backfill are the conversation summaries' own, shared: r_bill is
// written in the same transactions as r_conv, so the two can never
// disagree about which minutes are counted, and the billing figures
// of a range the summaries answer are the records' figures, neither
// more nor less.

var billSchema = []string{
	`CREATE TABLE IF NOT EXISTS r_bill (ts TIMESTAMP, dup BOOLEAN, src VARCHAR, dst VARCHAR, wire UBIGINT)`,
}

// billKey is the meta key set once r_bill covers every flow file.
const billKey = "bill_v1"

// billInsert adds the flows in src (a table or subquery with the
// flow columns) to r_bill, one row per hour and pair of addresses.
// Partial rows are expected — the folder folds a few minutes at a
// time — and the queries sum them, as they do for the other
// summaries.
func billInsert(src string) string {
	return fmt.Sprintf(`INSERT INTO r_bill SELECT date_trunc('hour', ts), dup, src, dst,
		sum(CASE WHEN wire > %d THEN 0 ELSE wire END)::UBIGINT FROM %s GROUP BY ALL`, wrapLimit, src)
}

// billStart is the earliest hour the billing summaries reach back
// to, zero while there are none.
func (s *Store) billStart() time.Time {
	var t sql.NullTime
	if err := s.DB.QueryRow(`SELECT min(ts) FROM r_bill`).Scan(&t); err != nil || !t.Valid {
		return time.Time{}
	}
	return t.Time.UTC()
}

// rebuildBillHours rebuilds r_bill for every hour in [from, to] from
// the flow records, deleting what is there first. Sealing calls it
// for the hours it touched, beside rebuildConvHours: a record that
// arrived after its minute was folded is in the file the seal wrote,
// and the rebuild is what counts it. s.wmu must be held, and the
// segments must be loaded.
func (s *Store) rebuildBillHours(from, to time.Time) error {
	for h := from.UTC().Truncate(time.Hour); !h.After(to.UTC().Truncate(time.Hour)); h = h.Add(time.Hour) {
		if _, err := s.DB.Exec(`DELETE FROM r_bill WHERE ts = ?`, h); err != nil {
			return fmt.Errorf("rebuild billing summaries: %w", err)
		}
		src := fmt.Sprintf("(SELECT * FROM %s WHERE ts >= %s AND ts < %s)",
			s.Source(h, h.Add(time.Hour)), tsLiteral(h), tsLiteral(h.Add(time.Hour)))
		if _, err := s.DB.Exec(billInsert(src)); err != nil {
			return fmt.Errorf("rebuild billing summaries: %w", err)
		}
	}
	return nil
}

// buildBill fills r_bill from the flow files sealed before it
// existed, once per data directory, a file at a time like buildConv,
// and folds the hot rows behind the watermark with them: the folder
// folds from the watermark on and will not see those again, while
// the seal rebuilds their hours from the records either way. Until
// it is done, billing reads the flow records, as it did before there
// was a summary.
func (s *Store) buildBill(ctx context.Context) {
	defer func() {
		s.vmu.Lock()
		s.bStop = nil
		s.vmu.Unlock()
		s.bg.Done()
	}()
	// Start over, with sealing held off while the list is taken, the
	// way buildConv does: the files sealed from here on are folded
	// by the seal itself.
	s.wmu.Lock()
	if _, err := s.DB.Exec(`DELETE FROM r_bill`); err != nil {
		s.wmu.Unlock()
		log.Printf("store: billing summaries: %v", err)
		return
	}
	s.segMu.RLock()
	segs := append([]segment(nil), s.segCache...)
	s.segMu.RUnlock()
	w := s.foldWatermark()
	s.wmu.Unlock()
	t0 := time.Now()
	for _, g := range segs {
		if ctx.Err() != nil {
			return
		}
		if _, err := os.Stat(g.path); err != nil {
			continue
		}
		src := "(SELECT * FROM read_parquet('" + sqlPath(g.path) + "'))"
		if _, err := s.DB.Exec(billInsert(src)); err != nil {
			if ctx.Err() == nil {
				log.Printf("store: billing summaries of %s: %v", g.path, err)
			}
			return
		}
	}
	if !w.IsZero() {
		src := fmt.Sprintf("(SELECT * FROM hot WHERE ts < %s)", tsLiteral(w))
		if _, err := s.DB.Exec(billInsert(src)); err != nil {
			if ctx.Err() == nil {
				log.Printf("store: billing summaries of the hot table: %v", err)
			}
			return
		}
	}
	if err := s.SetMeta(billKey, time.Now().UTC().Format(time.RFC3339)); err != nil {
		log.Printf("store: billing summaries: %v", err)
		return
	}
	s.billReady.Store(true)
	if len(segs) > 0 {
		log.Printf("store: summarised %d flow files for billing in %s", len(segs), time.Since(t0).Round(time.Second))
	}
}

// startBill builds the billing summaries of the flow files in the
// background.
func (s *Store) startBill() {
	ctx, stop := context.WithCancel(context.Background())
	s.vmu.Lock()
	s.bStop = stop
	s.vmu.Unlock()
	s.bg.Add(1)
	go s.buildBill(ctx)
}

// stopBill stops a build in progress, waits for it, and reports
// whether there was one — the way stopViews does, for the same
// caller: a purge deletes what a build would write back.
func (s *Store) stopBill() bool {
	s.vmu.Lock()
	stop := s.bStop
	s.vmu.Unlock()
	if stop == nil {
		return false
	}
	stop()
	s.bg.Wait()
	return true
}
