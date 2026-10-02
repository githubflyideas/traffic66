package sandbox

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/githubflyideas/traffic66/internal/enrich"
	"github.com/githubflyideas/traffic66/internal/store"
)

func wait(t *testing.T, sb *Sandbox) Info {
	t.Helper()
	for i := 0; i < 600; i++ {
		in := sb.Info()
		pending := false
		for _, f := range in.Files {
			if f.Status != "done" && f.Status != "error" {
				pending = true
			}
		}
		if !pending && !in.Busy {
			return in
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("import did not finish")
	return Info{}
}

// The example capture is imported into its own database, the attack in it
// is found, and deleting removes everything.
func TestSampleImport(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sandbox")
	inv := enrich.NewInventory()
	inv.Parse("net 10.10.0.0/16 Office\nnet 10.20.0.0/24 Servers\nhost 10.20.0.15 File server\n")
	sb := New(dir, inv, enrich.NewASNDB(), enrich.NewThreats())
	defer sb.Close()
	var buf bytes.Buffer
	end := time.Date(2026, 9, 20, 11, 0, 0, 0, time.UTC)
	if err := Sample(&buf, end); err != nil {
		t.Fatal(err)
	}
	t.Logf("sample: %d bytes", buf.Len())
	if _, err := sb.Add(SampleName, bytes.NewReader(buf.Bytes()), true); err != nil {
		t.Fatal(err)
	}
	in := wait(t, sb)
	f := in.Files[0]
	if f.Status != "done" || f.Flows == 0 || !in.Ready {
		t.Fatalf("%+v", f)
	}
	if f.First.Before(end.Add(-16*time.Minute)) || f.Last.After(end) {
		t.Errorf("span %v – %v", f.First, f.Last)
	}
	st, _, _ := sb.Store()
	fs, err := st.Findings(store.FindingQuery{From: f.First.Add(-time.Hour), To: f.Last.Add(time.Hour)})
	if err != nil {
		t.Fatal(err)
	}
	kinds := map[string]bool{}
	for _, x := range fs {
		kinds[x.Kind] = true
		t.Logf("%s %s -> %s %s sev %d", x.Kind, x.Src, x.Dst, x.Port, x.Sev)
	}
	for _, k := range []string{"scan", "portscan", "brute"} {
		if !kinds[k] {
			t.Errorf("no %s finding", k)
		}
	}
	var wire float64
	st.DB.QueryRow(`SELECT sum(wire) FROM hot`).Scan(&wire)
	if wire < 20e6 {
		t.Errorf("traffic %.0f bytes", wire)
	}
	if err := sb.Delete(""); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("sandbox directory left: %v", err)
	}
	if st, _, _ := sb.Store(); st != nil {
		t.Error("database still open")
	}
}

func TestLimits(t *testing.T) {
	sb := NewWith(t.TempDir(), enrich.NewInventory(), enrich.NewASNDB(), enrich.NewThreats(), Limits{Files: 3, FileSize: 1000, TotalSize: 3000}, 0.05)
	defer sb.Close()
	var small bytes.Buffer
	Sample(&small, time.Now())
	head := small.Bytes()[:800]
	if _, err := sb.Add("x.pcap", bytes.NewReader(small.Bytes()), false); !errors.Is(err, ErrLimit) {
		t.Fatalf("big file: %v", err)
	}
	if _, err := sb.Add("x.txt", strings.NewReader("hello world"), false); err == nil || errors.Is(err, ErrLimit) {
		t.Fatalf("text file: %v", err)
	}
	for i := 0; i < 3; i++ {
		if _, err := sb.Add("a.pcap", bytes.NewReader(head), false); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := sb.Add("d.pcap", bytes.NewReader(head), false); !errors.Is(err, ErrLimit) {
		t.Fatalf("fourth file: %v", err)
	}
	in := wait(t, sb)
	names := []string{}
	for _, f := range in.Files {
		names = append(names, f.Name+":"+f.Exporter)
	}
	if strings.Join(names, " ") != "a.pcap:127.0.1.1 a-2.pcap:127.0.1.2 a-3.pcap:127.0.1.3" {
		t.Errorf("files %v", names)
	}
	if err := sb.Delete("a-2.pcap"); err != nil {
		t.Fatal(err)
	}
	if in := wait(t, sb); len(in.Files) != 2 {
		t.Errorf("%d files after delete", len(in.Files))
	}
	if _, err := sb.Add("../../etc/passwd.pcap", bytes.NewReader(head), false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(sb.Dir, "files", "passwd.pcap")); err != nil {
		t.Errorf("name not cleaned: %v", err)
	}
}

// Files opened in place count against the total and are never deleted.
func TestAddPath(t *testing.T) {
	dir := t.TempDir()
	var buf bytes.Buffer
	Sample(&buf, time.Now())
	p := filepath.Join(dir, "x.pcap")
	os.WriteFile(p, buf.Bytes()[:800], 0o644)
	sb := NewWith(filepath.Join(dir, "sb"), enrich.NewInventory(), enrich.NewASNDB(), enrich.NewThreats(), Limits{Files: 3, FileSize: 1000, TotalSize: 1500}, 0.05)
	defer sb.Close()
	if _, err := sb.AddPath(p); err != nil {
		t.Fatal(err)
	}
	if _, err := sb.AddPath(p); !errors.Is(err, ErrLimit) {
		t.Fatalf("over the total: %v", err)
	}
	wait(t, sb)
	if err := sb.Delete(""); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p); err != nil {
		t.Errorf("the original file was deleted: %v", err)
	}
}
