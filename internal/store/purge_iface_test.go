package store

import (
	"testing"
	"time"
)

// Deleting one interface's data takes its flow records out of the hot
// table and out of the flow files — rewriting a file it shares with
// another interface, deleting one it does not — and takes its counter
// samples with it. Everything else stays.
func TestPurgeIface(t *testing.T) {
	s, err := Open(Options{Dir: t.TempDir(), Threads: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	// Half past the hour, not the minute the test happens to run in: Seal
	// rounds down to the hour after subtracting five minutes, so a base time
	// in the first five minutes of an hour put the first Seal's boundary an
	// hour earlier than this test needs and sealed nothing. That made the
	// test fail for five minutes in every sixty with nothing wrong.
	now := time.Now().UTC().Truncate(time.Hour).Add(-30 * time.Minute)
	old := now.Add(-3 * time.Hour)
	row := func(ts time.Time, in, out uint32, exp string) Row {
		return Row{TS: ts, Src: "10.0.0.1", Dst: "10.0.0.2", Client: "10.0.0.1", Server: "10.0.0.2",
			Exporter: exp, InIf: in, OutIf: out, Bytes: 100, Wire: 100, Pkts: 1, Flows: 1}
	}
	// a file of its own, then a file shared with interface 7
	if err := s.AppendRows([]Row{row(old, 3, 0, "127.0.0.1"), row(old, 0, 3, "127.0.0.1")}); err != nil {
		t.Fatal(err)
	}
	if err := s.Seal(old.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	mid := now.Add(-2 * time.Hour)
	if err := s.AppendRows([]Row{row(mid, 3, 0, "127.0.0.1"), row(mid, 7, 0, "127.0.0.1"), row(mid, 3, 0, "127.0.0.2")}); err != nil {
		t.Fatal(err)
	}
	if err := s.Seal(mid.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	// and rows still in the hot table
	if err := s.AppendRows([]Row{row(now, 3, 0, "127.0.0.1"), row(now, 7, 0, "127.0.0.1")}); err != nil {
		t.Fatal(err)
	}
	for _, ts := range []time.Time{old, mid, now} {
		if err := s.AppendCounter(ts, "127.0.0.1", 3, 1e9, 10, 10, 1, 1); err != nil {
			t.Fatal(err)
		}
		if err := s.AppendCounter(ts, "127.0.0.1", 7, 1e9, 10, 10, 1, 1); err != nil {
			t.Fatal(err)
		}
	}

	est := s.PurgeSizeIface("127.0.0.1", 3, time.Time{})
	if est.Rows != 4 { // two in the first file, one in the second, one hot
		t.Fatalf("estimated %d rows, want 4", est.Rows)
	}
	if est.Counters != 3 {
		t.Fatalf("estimated %d counter samples, want 3", est.Counters)
	}
	if est.Files != 2 {
		t.Fatalf("estimated %d files, want 2", est.Files)
	}

	done, err := s.PurgeIface("127.0.0.1", 3, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if done.Rows != 4 {
		t.Fatalf("deleted %d rows, want 4", done.Rows)
	}
	if done.Counters != 3 {
		t.Fatalf("deleted %d counter samples, want 3", done.Counters)
	}

	from, to := old.Add(-time.Hour), now.Add(time.Hour)
	var left int
	if err := s.DB.QueryRow(`SELECT count(*) FROM ` + s.Source(from, to) +
		` WHERE exporter = '127.0.0.1' AND (in_if = 3 OR out_if = 3)`).Scan(&left); err != nil {
		t.Fatal(err)
	}
	if left != 0 {
		t.Fatalf("%d records of the interface are left", left)
	}
	// interface 7 of the same exporter, and interface 3 of another, stay
	for _, c := range []struct {
		where string
		want  int
	}{
		{"exporter = '127.0.0.1' AND in_if = 7", 2},
		{"exporter = '127.0.0.2' AND in_if = 3", 1},
	} {
		var n int
		if err := s.DB.QueryRow(`SELECT count(*) FROM ` + s.Source(from, to) + ` WHERE ` + c.where).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n != c.want {
			t.Fatalf("%s: %d records, want %d", c.where, n, c.want)
		}
	}
	var ctr int
	s.DB.QueryRow(`SELECT count(*) FROM ifc`).Scan(&ctr)
	if ctr != 3 {
		t.Fatalf("%d counter samples left, want 3 (interface 7's)", ctr)
	}
	// the rewritten file's recorded range and row count must match it
	rows, err := s.DB.Query(`SELECT path, min_ts, max_ts, rows FROM segments`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	files := 0
	for rows.Next() {
		var path string
		var mn, mx time.Time
		var n int64
		if err := rows.Scan(&path, &mn, &mx, &n); err != nil {
			t.Fatal(err)
		}
		files++
		var real int64
		var rmn, rmx time.Time
		if err := s.DB.QueryRow(`SELECT count(*), min(ts), max(ts) FROM read_parquet('`+sqlPath(path)+`')`).Scan(&real, &rmn, &rmx); err != nil {
			t.Fatal(err)
		}
		if n != real || !mn.Equal(rmn) || !mx.Equal(rmx) {
			t.Fatalf("%s: recorded %d rows %v..%v, file has %d rows %v..%v", path, n, mn, mx, real, rmn, rmx)
		}
	}
	if files != 1 { // the interface's own file is gone, the shared one is not
		t.Fatalf("%d flow files left, want 1", files)
	}
}

// Deleting by age as well as by interface leaves the interface's newer
// records alone.
func TestPurgeIfaceBefore(t *testing.T) {
	s, err := Open(Options{Dir: t.TempDir(), Threads: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	now := time.Now().UTC().Truncate(time.Minute)
	row := func(ts time.Time) Row {
		return Row{TS: ts, Src: "10.0.0.1", Dst: "10.0.0.2", Client: "10.0.0.1", Server: "10.0.0.2",
			Exporter: "127.0.0.1", InIf: 3, Bytes: 100, Wire: 100, Pkts: 1, Flows: 1}
	}
	if err := s.AppendRows([]Row{row(now.AddDate(0, 0, -10)), row(now)}); err != nil {
		t.Fatal(err)
	}
	done, err := s.PurgeIface("127.0.0.1", 3, now.AddDate(0, 0, -5))
	if err != nil {
		t.Fatal(err)
	}
	if done.Rows != 1 {
		t.Fatalf("deleted %d rows, want 1", done.Rows)
	}
	var n int
	s.DB.QueryRow(`SELECT count(*) FROM hot`).Scan(&n)
	if n != 1 {
		t.Fatalf("%d rows left, want 1", n)
	}
}

// An interface that was never seen is not an error and deletes nothing.
func TestPurgeIfaceUnknown(t *testing.T) {
	s, err := Open(Options{Dir: t.TempDir(), Threads: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	done, err := s.PurgeIface("127.0.0.1", 99, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if done.Rows != 0 || done.Files != 0 {
		t.Fatalf("deleted %+v", done)
	}
	if _, err := s.PurgeIface("", 1, time.Time{}); err == nil {
		t.Fatal("an empty exporter was accepted")
	}
}
