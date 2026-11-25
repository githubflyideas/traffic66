package store

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"
)

// The conversation summaries, folded in as the data settles.
//
// The summaries by time, address and dimension (r_ts, r_host, r_dim)
// are accumulated by the pipeline as the rows arrive and written once
// a minute, so they are never more than a minute or two behind. The
// summaries by device and interface are written when rows are sealed,
// and the queries that read them add the rows still in the hot table
// on the fly, so their answers are current too. Conversations had no
// summary at all: a Top of conversations over a range past the flow
// records was a scan of every record in it. r_conv is that summary,
// one row per hour and conversation, and because no query patches it
// up from the hot table it is folded in as the minutes settle, under
// a watermark, rather than waiting for the hour's seal.
//
// The watermark is the moment up to which hot rows have been folded.
// It lives in the meta table and moves only in the transaction that
// writes the summaries, so a stop anywhere leaves the two in step:
// minutes at or after it are folded by whoever gets there first (the
// folder below, or the seal), minutes before it are never folded
// again from the hot table. A row that arrives after its minute was
// folded is not lost: it is sealed with its hour, and sealing rebuilds
// the summary of every hour it touches from the records themselves,
// which is also what the crash recovery does for the recent hours.

var convSchema = []string{
	`CREATE TABLE IF NOT EXISTS r_conv (ts TIMESTAMP, exporter VARCHAR, dup BOOLEAN, client VARCHAR, server VARCHAR,
		cli4 UINTEGER, srv4 UINTEGER, svc_port USMALLINT, proto UTINYINT, app VARCHAR, peer_cc VARCHAR, threat VARCHAR,
		dir UTINYINT, cli_int BOOLEAN, srv_int BOOLEAN, bytes UBIGINT, wire UBIGINT, pkts UBIGINT, flows UBIGINT)`,
	`CREATE TABLE IF NOT EXISTS r_conv_day (ts TIMESTAMP, exporter VARCHAR, dup BOOLEAN, client VARCHAR, server VARCHAR,
		cli4 UINTEGER, srv4 UINTEGER, svc_port USMALLINT, proto UTINYINT, app VARCHAR, peer_cc VARCHAR, threat VARCHAR,
		dir UTINYINT, cli_int BOOLEAN, srv_int BOOLEAN, bytes UBIGINT, wire UBIGINT, pkts UBIGINT, flows UBIGINT)`,
}

// foldKey is the meta key of the fold watermark.
const foldKey = "fold_until"

// convKey is the meta key set once r_conv covers every flow file.
const convKey = "conv_v1"

// convDays is how long the hourly conversation summaries are kept.
// Older conversations are kept as the daily summaries below.
const convDays = 30

// convDayTop is how many of a day's conversations its daily summary
// keeps, the largest by wire bytes.
const convDayTop = 2000

// convDayRollup makes the daily conversation summary of every day
// whose hourly summaries end before cut and that has none yet: the
// day's conversations summed, the largest convDayTop of them kept.
// Retention calls it before the hourly rows of those days are
// deleted — afterwards the daily table is the only copy — and a day
// that has a summary already is not made again.
func (s *Store) convDayRollup(cut time.Time) error {
	rows, err := s.DB.Query(`SELECT DISTINCT CAST(ts AS DATE) FROM r_conv WHERE ts < ? ORDER BY 1`, cut.UTC())
	if err != nil {
		return fmt.Errorf("daily conversation summaries: %w", err)
	}
	var days []time.Time
	for rows.Next() {
		var d time.Time
		if err := rows.Scan(&d); err != nil {
			rows.Close()
			return fmt.Errorf("daily conversation summaries: %w", err)
		}
		days = append(days, d.UTC())
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return fmt.Errorf("daily conversation summaries: %w", err)
	}
	for _, d := range days {
		var n int64
		if err := s.DB.QueryRow(`SELECT count(*) FROM r_conv_day WHERE ts = ?`, d).Scan(&n); err != nil {
			return fmt.Errorf("daily conversation summaries: %w", err)
		}
		if n > 0 {
			continue
		}
		if _, err := s.DB.Exec(fmt.Sprintf(`INSERT INTO r_conv_day
			SELECT ts, exporter, dup, client, server, cli4, srv4, svc_port, proto, app, peer_cc, threat, dir, cli_int, srv_int, bytes, wire, pkts, flows
			FROM (SELECT CAST(ts AS DATE)::TIMESTAMP AS ts, exporter, dup, client, server, cli4, srv4, svc_port, proto, app, peer_cc, threat, dir,
				bool_or(cli_int) AS cli_int, bool_or(srv_int) AS srv_int,
				sum(bytes)::UBIGINT AS bytes, sum(wire)::UBIGINT AS wire, sum(pkts)::UBIGINT AS pkts, sum(flows)::UBIGINT AS flows,
				row_number() OVER (ORDER BY sum(wire) DESC) AS rn
				FROM r_conv WHERE ts >= ? AND ts < ? GROUP BY ALL) WHERE rn <= %d`, convDayTop),
			d, d.AddDate(0, 0, 1)); err != nil {
			return fmt.Errorf("daily conversation summaries: %w", err)
		}
	}
	return nil
}

// convInsert adds the conversations of the flow rows in src (a table
// or subquery with the flow columns) to r_conv, one row per hour,
// conversation and the columns a Top groups or filters by. Partial
// rows are expected — the folder folds a few minutes at a time — and
// the queries sum them, as they do for the other summaries.
func convInsert(src string) string {
	sum := func(c string) string {
		return fmt.Sprintf("sum(CASE WHEN %s > %d THEN 0 ELSE %s END)::UBIGINT", c, wrapLimit, c)
	}
	sums := sum("bytes") + ", " + sum("wire") + ", " + sum("pkts") + ", " + sum("flows")
	return `INSERT INTO r_conv SELECT date_trunc('hour', ts), exporter, dup, client, server, cli4, srv4,
		svc_port, proto, app, peer_cc, threat, dir, bool_or(cli_int), bool_or(srv_int), ` + sums + `
		FROM ` + src + ` GROUP BY ALL`
}

func tsLiteral(t time.Time) string {
	return "TIMESTAMP '" + t.UTC().Format("2006-01-02 15:04:05.999999") + "'"
}

// foldTx folds the hot rows in [from, to) into r_conv and r_bill
// and moves the watermark to to, in the caller's transaction. The
// billing summaries are folded in the same transaction, so the two
// tables and the watermark can never disagree about which minutes
// are counted. s.wmu must be held.
func (s *Store) foldTx(tx *sql.Tx, from, to time.Time) error {
	src := fmt.Sprintf("(SELECT * FROM hot WHERE ts >= %s AND ts < %s)", tsLiteral(from), tsLiteral(to))
	if _, err := tx.Exec(convInsert(src)); err != nil {
		return fmt.Errorf("fold conversations: %w", err)
	}
	if _, err := tx.Exec(billInsert(src)); err != nil {
		return fmt.Errorf("fold billing summaries: %w", err)
	}
	if _, err := tx.Exec(`INSERT INTO meta VALUES (?, ?) ON CONFLICT (k) DO UPDATE SET v = excluded.v`,
		foldKey, to.UTC().Format(time.RFC3339Nano)); err != nil {
		return fmt.Errorf("fold watermark: %w", err)
	}
	return nil
}

// convStart is the earliest hour the conversation summaries reach
// back to, zero while there are none.
func (s *Store) convStart() time.Time {
	var t sql.NullTime
	if err := s.DB.QueryRow(`SELECT min(ts) FROM r_conv`).Scan(&t); err != nil || !t.Valid {
		return time.Time{}
	}
	return t.Time.UTC()
}

// convDayStart is the earliest day the daily conversation summaries
// reach back to, zero while there are none.
func (s *Store) convDayStart() time.Time {
	var t sql.NullTime
	if err := s.DB.QueryRow(`SELECT min(ts) FROM r_conv_day`).Scan(&t); err != nil || !t.Valid {
		return time.Time{}
	}
	return t.Time.UTC()
}

// foldWatermark is the moment up to which hot rows have been folded
// into r_conv, zero before the first fold.
func (s *Store) foldWatermark() time.Time {
	n := s.foldWM.Load()
	if n == 0 {
		return time.Time{}
	}
	return time.Unix(0, n).UTC()
}

// loadFoldWatermark reads the watermark back at Open.
func (s *Store) loadFoldWatermark() {
	v := s.Meta(foldKey)
	if v == "" {
		return
	}
	if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
		s.foldWM.Store(t.UTC().UnixNano())
	}
}

// foldOnce folds the settled minutes the watermark has not reached.
// It is the folder's step; tests drive it directly.
func (s *Store) foldOnce(now time.Time) error {
	if !s.convReady.Load() || !s.billReady.Load() {
		return nil // a backfill owns the summaries until it is done
	}
	target := settledTo(now, true)
	s.wmu.Lock()
	defer s.wmu.Unlock()
	w := s.foldWatermark()
	if !target.After(w) {
		return nil
	}
	if err := s.appHot.Flush(); err != nil {
		return err
	}
	tx, err := s.DB.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := s.foldTx(tx, w, target); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	s.foldWM.Store(target.UTC().UnixNano())
	return nil
}

// startFold runs the folder in the background: every half minute it
// folds the minutes that have settled since the last fold, so the
// conversation summaries trail the data by the settle delay and one
// flush, a minute or two in all.
func (s *Store) startFold() {
	ctx, stop := context.WithCancel(context.Background())
	s.foldStop = stop
	s.foldWG.Add(1)
	go func() {
		defer s.foldWG.Done()
		t := time.NewTicker(30 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case now := <-t.C:
				if err := s.foldOnce(now); err != nil {
					log.Printf("store: fold: %v", err)
				}
			}
		}
	}()
}

// stopFold stops the background folder, if it is running.
func (s *Store) stopFold() {
	if s.foldStop != nil {
		s.foldStop()
		s.foldWG.Wait()
		s.foldStop = nil
	}
}

// rebuildConvHours rebuilds r_conv for every hour in [from, to] from
// the flow records, deleting what is there first. Sealing calls it
// for the hours it touched: a record that arrived after its minute
// was folded is in the file the seal wrote, and the rebuild is what
// counts it. s.wmu must be held, and the segments must be loaded.
func (s *Store) rebuildConvHours(from, to time.Time) error {
	for h := from.UTC().Truncate(time.Hour); !h.After(to.UTC().Truncate(time.Hour)); h = h.Add(time.Hour) {
		if _, err := s.DB.Exec(`DELETE FROM r_conv WHERE ts = ?`, h); err != nil {
			return fmt.Errorf("rebuild conversations: %w", err)
		}
		// Source names every file the hour touches; the rows of the
		// hour are picked out of them, the way the queries do.
		src := fmt.Sprintf("(SELECT * FROM %s WHERE ts >= %s AND ts < %s)",
			s.Source(h, h.Add(time.Hour)), tsLiteral(h), tsLiteral(h.Add(time.Hour)))
		if _, err := s.DB.Exec(convInsert(src)); err != nil {
			return fmt.Errorf("rebuild conversations: %w", err)
		}
	}
	return nil
}

// buildConv fills r_conv from the flow files sealed before it
// existed, once per data directory, a file at a time like buildViews.
// Until it is done, conversation Tops of old ranges read the flow
// records, as they did before there was a summary.
func (s *Store) buildConv(ctx context.Context) {
	defer func() {
		s.vmu.Lock()
		s.cStop = nil
		s.vmu.Unlock()
		s.bg.Done()
	}()
	// Start over, with sealing held off while the list is taken, the
	// way buildViews does: the files sealed from here on are folded
	// by the seal itself.
	s.wmu.Lock()
	if _, err := s.DB.Exec(`DELETE FROM r_conv`); err != nil {
		s.wmu.Unlock()
		log.Printf("store: conversation summaries: %v", err)
		return
	}
	s.segMu.RLock()
	segs := append([]segment(nil), s.segCache...)
	s.segMu.RUnlock()
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
		if _, err := s.DB.Exec(convInsert(src)); err != nil {
			if ctx.Err() == nil {
				log.Printf("store: conversation summaries of %s: %v", g.path, err)
			}
			return
		}
	}
	if err := s.SetMeta(convKey, time.Now().UTC().Format(time.RFC3339)); err != nil {
		log.Printf("store: conversation summaries: %v", err)
		return
	}
	s.convReady.Store(true)
	if len(segs) > 0 {
		log.Printf("store: summarised %d flow files by conversation in %s", len(segs), time.Since(t0).Round(time.Second))
	}
}

// startConv builds the conversation summaries of the flow files in
// the background.
func (s *Store) startConv() {
	ctx, stop := context.WithCancel(context.Background())
	s.vmu.Lock()
	s.cStop = stop
	s.vmu.Unlock()
	s.bg.Add(1)
	go s.buildConv(ctx)
}

// stopConv stops a build in progress, waits for it, and reports
// whether there was one — the way stopViews does, for the same
// caller: a purge deletes what a build would write back.
func (s *Store) stopConv() bool {
	s.vmu.Lock()
	stop := s.cStop
	s.vmu.Unlock()
	if stop == nil {
		return false
	}
	stop()
	s.bg.Wait()
	return true
}
