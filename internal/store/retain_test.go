package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// walSize is how much of the write-ahead log is waiting to be folded into
// the database file. A delete lands there and stays until a checkpoint.
func walSize(t *testing.T, dir string) int64 {
	t.Helper()
	fi, err := os.Stat(filepath.Join(dir, "traffic66.duckdb.wal"))
	if os.IsNotExist(err) {
		return 0
	}
	if err != nil {
		t.Fatal(err)
	}
	return fi.Size()
}

// Retention deletes what is past its retention and then checkpoints, so
// the deletes do not sit in the write-ahead log growing for as long as the
// installation runs. Sealing, the manual cleanup and deleting by interface
// all did this already; the pass that runs on its own every day did not.
func TestRetainCheckpoints(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(Options{Dir: dir, Threads: 1, RawDays: 30, RollupDays: 400})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	now := time.Now().UTC().Truncate(time.Hour)
	// counters old enough to be past RawDays, and enough of them that the
	// delete is something a checkpoint has to fold away
	for i := 0; i < 4000; i++ {
		ts := now.AddDate(0, 0, -40).Add(time.Duration(i) * time.Minute)
		if err := s.AppendCounter(ts, "192.0.2.1", uint32(i%8), 1e9, uint64(i)*1e6, uint64(i)*2e6, uint64(i), uint64(i)*2); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.DB.Exec(`CHECKPOINT`); err != nil { // a clean slate to measure from
		t.Fatal(err)
	}
	// Measured, so that the assertion below is not looser than the thing it
	// is looking for: after a checkpoint the log file is gone, and the
	// delete this test provokes puts about 32 KB back into it.
	if n := walSize(t, dir); n != 0 {
		t.Fatalf("the log is %d bytes right after a checkpoint; this test cannot tell anything apart", n)
	}

	var left int
	if err := s.DB.QueryRow(`SELECT count(*) FROM ifc`).Scan(&left); err != nil {
		t.Fatal(err)
	}
	if left != 4000 {
		t.Fatalf("ifc holds %d rows before retention, want 4000", left)
	}

	if err := s.Retain(now); err != nil {
		t.Fatalf("Retain: %v", err)
	}

	if err := s.DB.QueryRow(`SELECT count(*) FROM ifc`).Scan(&left); err != nil {
		t.Fatal(err)
	}
	if left != 0 {
		t.Errorf("ifc still holds %d rows past their retention", left)
	}
	// The point of the change: after Retain the deletion is in the database
	// file and the log is folded away, rather than holding the delete for
	// as long as the installation runs. Without the checkpoint this reads
	// about 32 KB; with it the file is gone.
	if after := walSize(t, dir); after != 0 {
		t.Errorf("the write-ahead log still holds %d bytes: the delete was not checkpointed", after)
	}
}

// Retention reports a failure instead of returning nil over it. Every
// Exec in this pass used to have its result dropped, so a delete that
// failed looked exactly like one that worked.
func TestRetainReportsFailure(t *testing.T) {
	dir := t.TempDir()
	s, err := Open(Options{Dir: dir, Threads: 1, RawDays: 30, RollupDays: 400})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	now := time.Now().UTC().Truncate(time.Hour)
	if err := s.AppendCounter(now.AddDate(0, 0, -40), "192.0.2.1", 1, 1e9, 1, 2, 3, 4); err != nil {
		t.Fatal(err)
	}
	// Take a table out from under it: the delete then cannot work, and the
	// pass has to say so.
	if _, err := s.DB.Exec(`DROP TABLE r_dim`); err != nil {
		t.Fatal(err)
	}
	if err := s.Retain(now); err == nil {
		t.Error("Retain returned nil although a delete could not run")
	} else if !contains(err.Error(), "r_dim") {
		t.Errorf("the error does not name the table that failed: %v", err)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
