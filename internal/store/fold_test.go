package store

import (
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"
)

// foldFixture is five hours of conversations: IPv4 and IPv6, two
// exporters, duplicates, threats, several applications. Every extra
// is a function of the conversation, so any_value over it is the same
// value in every implementation.
func foldFixture(base time.Time) []Row {
	type host struct {
		text string
		num  uint32
	}
	clients := []host{{"198.51.100.20", 0xC6336414}, {"198.51.100.21", 0xC6336415}, {"2001:db8:2::20", 0}}
	servers := []host{{"203.0.113.10", 0xCB00710A}, {"203.0.113.11", 0xCB00710B}, {"2001:db8:1::10", 0}}
	ports := []struct {
		port  uint16
		proto uint8
		app   string
	}{{443, 6, "HTTPS"}, {53, 17, "DNS"}}
	var rows []Row
	c := 0
	for m := 0; m < 300; m += 7 {
		for i := 0; i < 4; i++ {
			cl, sv, p := clients[(c+i)%3], servers[(c+i)%3], ports[(c+i)%2]
			if (cl.num == 0) != (sv.num == 0) {
				continue // a conversation's ends are one family
			}
			wire := uint64(1000*(c+i+1) + m)
			threat := ""
			if c%11 == 0 {
				threat = "scanner"
			}
			exp := "10.0.0.1"
			if (c+i)%2 == 1 {
				exp = "10.0.0.2"
			}
			rows = append(rows, Row{
				TS:  base.Add(time.Duration(m) * time.Minute),
				Src: cl.text, Dst: sv.text, Client: cl.text, Server: sv.text,
				Cli4: cl.num, Srv4: sv.num, SvcPort: p.port, Proto: p.proto, App: p.app,
				Bytes: wire * 3 / 4, Wire: wire, Pkts: uint64(10 + i), Flows: 1,
				Exporter: exp, InIf: 1, OutIf: 2, Dir: DirOutbound, PeerCC: "US",
				Threat: threat, Dup: c%13 == 0, CliInt: true,
			})
			c++
		}
	}
	return rows
}

// convDump is r_conv canonicalised: partial rows of one conversation
// summed, so two stores that folded the same records in different
// chunks compare equal exactly when their summaries agree.
func convDump(t *testing.T, s *Store) string {
	t.Helper()
	rows, err := s.DB.Query(`SELECT ts, exporter, dup, client, server, cli4, srv4, svc_port, proto, app, peer_cc, threat, dir,
		bool_or(cli_int), bool_or(srv_int), sum(bytes), sum(wire), sum(pkts), sum(flows)
		FROM r_conv GROUP BY ALL`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var lines []string
	for rows.Next() {
		var ts time.Time
		var exp, cl, sv, app, cc, threat string
		var dup, ci, si bool
		var cli4, srv4 uint32
		var port uint16
		var proto, dir uint8
		var b, w, p, f uint64
		if err := rows.Scan(&ts, &exp, &dup, &cl, &sv, &cli4, &srv4, &port, &proto, &app, &cc, &threat, &dir, &ci, &si, &b, &w, &p, &f); err != nil {
			t.Fatal(err)
		}
		lines = append(lines, fmt.Sprintf("%s|%s|%v|%s|%s|%d|%d|%d|%d|%s|%s|%s|%d|%v|%v|%d|%d|%d|%d",
			ts.Format("2006-01-02 15:04"), exp, dup, cl, sv, cli4, srv4, port, proto, app, cc, threat, dir, ci, si, b, w, p, f))
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n")
}

func openFoldStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(Options{Dir: t.TempDir(), Threads: 1})
	if err != nil {
		t.Fatal(err)
	}
	s.stopFold() // the tests drive the fold themselves
	return s
}

// The conversation summaries a store ends up with must not depend on
// how the work was split between the folder and the seals: folded a
// few minutes at a time as the data settled, folded by the seals
// alone, or folded in one pass over the hot table, the summaries of
// the same records are the same rows.
func TestFoldAndSealAgree(t *testing.T) {
	base := time.Date(2026, 1, 5, 12, 0, 0, 0, time.UTC)
	rows := foldFixture(base)
	end := base.Add(6 * time.Hour)

	// one pass over the hot table
	one := openFoldStore(t)
	defer one.Close()
	if err := one.AppendRows(rows); err != nil {
		t.Fatal(err)
	}
	tx, err := one.DB.Begin()
	if err != nil {
		t.Fatal(err)
	}
	if err := one.foldTx(tx, time.Time{}, end); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	want := convDump(t, one)

	// seals alone
	sealed := openFoldStore(t)
	defer sealed.Close()
	if err := sealed.AppendRows(rows); err != nil {
		t.Fatal(err)
	}
	for _, now := range []time.Time{base.Add(2 * time.Hour), base.Add(4 * time.Hour), end} {
		if err := sealed.Seal(now); err != nil {
			t.Fatal(err)
		}
	}
	if got := convDump(t, sealed); got != want {
		t.Fatalf("seal-only summaries differ from one fold:\n%s\n---\n%s", got, want)
	}

	// folds and seals interleaved; folding the same minutes twice and
	// sealing over folded minutes must not count anything twice
	both := openFoldStore(t)
	defer both.Close()
	if err := both.AppendRows(rows); err != nil {
		t.Fatal(err)
	}
	for _, now := range []time.Time{base.Add(time.Hour), base.Add(2 * time.Hour)} {
		if err := both.foldOnce(now); err != nil {
			t.Fatal(err)
		}
		if err := both.foldOnce(now); err != nil { // again: a no-op
			t.Fatal(err)
		}
	}
	if err := both.Seal(base.Add(3 * time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := both.foldOnce(base.Add(5 * time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := both.Seal(end); err != nil {
		t.Fatal(err)
	}
	if got := convDump(t, both); got != want {
		t.Fatalf("fold+seal summaries differ from one fold:\n%s\n---\n%s", got, want)
	}
	if w := both.foldWatermark(); !w.Equal(end.Truncate(time.Hour).Add(time.Hour)) && w.Before(base.Add(5*time.Hour)) {
		t.Fatalf("watermark %v, want it past the folded range", w)
	}
}

// A record that arrives after its minute was folded is counted when
// its hour is sealed: the seal rebuilds the summary of the hours it
// touches from the records.
func TestFoldCountsLateRowsAtSeal(t *testing.T) {
	base := time.Date(2026, 1, 5, 12, 0, 0, 0, time.UTC)
	s := openFoldStore(t)
	defer s.Close()
	rows := foldFixture(base)
	if err := s.AppendRows(rows); err != nil {
		t.Fatal(err)
	}
	if err := s.foldOnce(base.Add(2 * time.Hour)); err != nil {
		t.Fatal(err)
	}
	// a late record for an already folded minute
	late := rows[0]
	late.Wire, late.Bytes = 7777, 5555
	wireOf := func() uint64 {
		var w uint64
		err := s.DB.QueryRow(`SELECT coalesce(sum(wire), 0) FROM r_conv
			WHERE ts = ? AND client = ? AND server = ? AND svc_port = ? AND exporter = ? AND dup = ?`,
			late.TS.Truncate(time.Hour), late.Client, late.Server, late.SvcPort, late.Exporter, late.Dup).Scan(&w)
		if err != nil {
			t.Fatal(err)
		}
		return w
	}
	before := wireOf()
	if err := s.AppendRows([]Row{late}); err != nil {
		t.Fatal(err)
	}
	if err := s.foldOnce(base.Add(3 * time.Hour)); err != nil {
		t.Fatal(err)
	}
	if got := wireOf(); got != before {
		t.Fatalf("the fold counted a late record: wire %d, want %d (it is counted at the seal)", got, before)
	}
	if err := s.Seal(base.Add(6 * time.Hour)); err != nil {
		t.Fatal(err)
	}
	after := convDump(t, s)
	if got := wireOf(); got != before+7777 {
		t.Fatalf("the seal did not count the late record: wire %d, want %d", got, before+7777)
	}
	// the whole store, folded at the seal alone, agrees
	one := openFoldStore(t)
	defer one.Close()
	all := append(append([]Row{}, rows...), late)
	if err := one.AppendRows(all); err != nil {
		t.Fatal(err)
	}
	if err := one.Seal(base.Add(6 * time.Hour)); err != nil {
		t.Fatal(err)
	}
	if got := convDump(t, one); got != after {
		t.Fatalf("late record summaries differ:\n%s\n---\n%s", got, after)
	}
}

// The watermark is in the database, not the process: after a reopen
// the folder takes up where it left off and folds nothing twice.
func TestFoldWatermarkSurvivesReopen(t *testing.T) {
	dir := t.TempDir()
	base := time.Date(2026, 1, 5, 12, 0, 0, 0, time.UTC)
	rows := foldFixture(base)
	s, err := Open(Options{Dir: dir, Threads: 1})
	if err != nil {
		t.Fatal(err)
	}
	s.stopFold()
	if err := s.AppendRows(rows); err != nil {
		t.Fatal(err)
	}
	if err := s.foldOnce(base.Add(2 * time.Hour)); err != nil {
		t.Fatal(err)
	}
	want := convDump(t, s)
	w := s.foldWatermark()
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	s2, err := Open(Options{Dir: dir, Threads: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer s2.Close()
	s2.stopFold()
	if got := s2.foldWatermark(); !got.Equal(w) {
		t.Fatalf("watermark after reopen %v, want %v", got, w)
	}
	if err := s2.foldOnce(base.Add(2 * time.Hour)); err != nil {
		t.Fatal(err)
	}
	if got := convDump(t, s2); got != want {
		t.Fatalf("folding after a reopen counted rows twice:\n%s\n---\n%s", got, want)
	}
}
