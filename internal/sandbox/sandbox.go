// Package sandbox imports packet capture files into a database of their own,
// apart from the live data: the same pages and detection rules then work on
// the capture, and deleting the sandbox removes both the files and the
// database.
package sandbox

import (
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"net/netip"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/githubflyideas/traffic66/internal/detect"
	"github.com/githubflyideas/traffic66/internal/enrich"
	"github.com/githubflyideas/traffic66/internal/flow"
	"github.com/githubflyideas/traffic66/internal/pcapfile"
	"github.com/githubflyideas/traffic66/internal/pipeline"
	"github.com/githubflyideas/traffic66/internal/store"
)

// Limits bound what a sandbox takes in.
type Limits struct {
	Files     int   `json:"max_files"`
	FileSize  int64 `json:"max_file_size"`
	TotalSize int64 `json:"max_total"`
}

// UploadLimits apply to files uploaded in the web UI; LocalLimits to files
// opened from the command line (traffic66 capture.pcap).
var (
	UploadLimits = Limits{Files: 3, FileSize: 50 << 20, TotalSize: 150 << 20}
	LocalLimits  = Limits{Files: 3, FileSize: 3e9, TotalSize: 3e9}
)

// File is one capture file in the sandbox.
type File struct {
	Name     string    `json:"name"`
	Size     int64     `json:"size"`
	Added    time.Time `json:"added"`
	Sample   bool      `json:"sample,omitempty"` // the demo's example; not counted against the limits
	Packets  uint64    `json:"packets"`
	Skipped  uint64    `json:"skipped"` // packets without an IP header
	Flows    uint64    `json:"flows"`
	First    time.Time `json:"first"`
	Last     time.Time `json:"last"`
	Status   string    `json:"status"` // waiting, importing, done, error
	Error    string    `json:"error,omitempty"`
	Exporter string    `json:"exporter"`       // the "device" the file's flows appear under
	Path     string    `json:"path,omitempty"` // a file opened in place (never deleted); else it is in Dir/files
}

// Sandbox holds the uploaded files and their database.
type Sandbox struct {
	Dir string
	Inv *enrich.Inventory // the live inventory: names of networks and hosts
	ASN *enrich.ASNDB
	Thr *enrich.Threats
	Lim Limits
	Mem float64 // share of memory for the sandbox database

	mu    sync.Mutex
	files []*File
	st    *store.Store
	det   *detect.Detector
	inv   *enrich.Inventory // live inventory plus a device per file
	busy  bool
	gen   int // bumped by every rebuild; stale imports stop
	wake  chan struct{}
}

// ErrLimit is returned when a file would exceed the limits.
var ErrLimit = errors.New("limit")

// ErrNotCapture is returned for files that are not pcap or pcapng.
var ErrNotCapture = errors.New("not a capture file: use .pcap or .pcapng as saved by Wireshark or tcpdump (not compressed)")

// New opens the sandbox in dir (creating nothing until a file is added) and
// starts its import worker.
func New(dir string, inv *enrich.Inventory, asn *enrich.ASNDB, thr *enrich.Threats) *Sandbox {
	return NewWith(dir, inv, asn, thr, UploadLimits, 0.05)
}

// NewWith is New with other limits and a share of memory for the database.
func NewWith(dir string, inv *enrich.Inventory, asn *enrich.ASNDB, thr *enrich.Threats, lim Limits, mem float64) *Sandbox {
	sb := &Sandbox{Dir: dir, Inv: inv, ASN: asn, Thr: thr, Lim: lim, Mem: mem, wake: make(chan struct{}, 1)}
	sb.load()
	go sb.worker()
	if len(sb.files) > 0 {
		sb.mu.Lock()
		if sb.st == nil { // database missing: rebuild from the files
			for _, f := range sb.files {
				f.Status = "waiting"
			}
		}
		sb.mu.Unlock()
		sb.kick()
	}
	return sb
}

func (sb *Sandbox) pcapDir() string { return filepath.Join(sb.Dir, "files") }

// filePath is where a file's packets are read from.
func (sb *Sandbox) filePath(f *File) string {
	if f.Path != "" {
		return f.Path
	}
	return filepath.Join(sb.pcapDir(), f.Name)
}
func (sb *Sandbox) dbDir() string { return filepath.Join(sb.Dir, "db") }
func (sb *Sandbox) indexPath() string {
	return filepath.Join(sb.Dir, "files.json")
}

func (sb *Sandbox) load() {
	b, err := os.ReadFile(sb.indexPath())
	if err != nil {
		return
	}
	var fs []*File
	if json.Unmarshal(b, &fs) != nil {
		return
	}
	for _, f := range fs {
		if _, err := os.Stat(sb.filePath(f)); err == nil {
			sb.files = append(sb.files, f)
		}
	}
	if _, err := os.Stat(filepath.Join(sb.dbDir(), "traffic66.duckdb")); err == nil {
		if err := sb.open(); err != nil {
			log.Printf("sandbox: %v", err)
		}
	}
	for _, f := range sb.files {
		if f.Status != "done" && f.Status != "error" {
			f.Status = "waiting"
		}
	}
}

func (sb *Sandbox) save() {
	b, _ := json.MarshalIndent(sb.files, "", " ")
	os.MkdirAll(sb.Dir, 0o755)
	tmp := sb.indexPath() + ".tmp"
	if os.WriteFile(tmp, b, 0o644) == nil {
		os.Rename(tmp, sb.indexPath())
	}
}

// open opens (or creates) the database; sb.mu held or not yet shared.
func (sb *Sandbox) open() error {
	threads := 2 // beside live collection
	if sb.Mem > 0.1 {
		threads = 0 // on its own: all but one core
	}
	st, err := store.Open(store.Options{Dir: sb.dbDir(), MemoryFraction: sb.Mem, Threads: threads})
	if err != nil {
		return err
	}
	sb.st = st
	sb.refreshInventory()
	sb.det = detect.New(st, sb.inv, detect.Config{})
	return nil
}

// refreshInventory names each file's exporter after the file.
func (sb *Sandbox) refreshInventory() {
	inv := enrich.NewInventory()
	var b strings.Builder
	if sb.Inv != nil {
		b.WriteString(sb.Inv.Text())
		b.WriteString("\n")
	}
	for _, f := range sb.files {
		fmt.Fprintf(&b, "device %s %s\n", f.Exporter, f.Name)
	}
	if err := inv.Parse(b.String()); err != nil {
		log.Printf("sandbox: inventory: %v", err)
	}
	sb.inv = inv
	if sb.det != nil {
		sb.det.Inv = inv
	}
}

// Store returns the sandbox database, or nil when there is none.
func (sb *Sandbox) Store() (*store.Store, *detect.Detector, *enrich.Inventory) {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	return sb.st, sb.det, sb.inv
}

// Info describes the sandbox for the UI.
type Info struct {
	Files []File    `json:"files"`
	Busy  bool      `json:"busy"`
	First time.Time `json:"first"`
	Last  time.Time `json:"last"`
	Limits
	Ready bool `json:"ready"` // has data to look at
}

func (sb *Sandbox) Info() Info {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	in := Info{Files: []File{}, Busy: sb.busy, Limits: sb.Lim}
	for _, f := range sb.files {
		in.Files = append(in.Files, *f)
		if f.Flows == 0 {
			continue
		}
		if in.First.IsZero() || f.First.Before(in.First) {
			in.First = f.First
		}
		if f.Last.After(in.Last) {
			in.Last = f.Last
		}
		in.Ready = sb.st != nil
	}
	return in
}

var unsafeName = regexp.MustCompile(`[^\p{L}\p{N}._ -]+`)

// cleanName makes an uploaded file name safe and unique.
func (sb *Sandbox) cleanName(name string) string {
	name = filepath.Base(strings.ReplaceAll(name, "\\", "/"))
	name = strings.TrimSpace(unsafeName.ReplaceAllString(name, "_"))
	if name == "" || name == "." || name == ".." || strings.HasPrefix(name, ".") {
		name = "capture.pcap"
	}
	if len(name) > 100 {
		name = name[len(name)-100:]
	}
	base, ext := name, filepath.Ext(name)
	base = strings.TrimSuffix(base, ext)
	for i := 2; ; i++ {
		taken := false
		for _, f := range sb.files {
			if f.Name == name {
				taken = true
			}
		}
		if !taken {
			return name
		}
		name = fmt.Sprintf("%s-%d%s", base, i, ext)
	}
}

// Add stores an uploaded capture file and queues it for import. The file is
// checked while it streams to disk: at most MaxFileSize bytes, and it must
// start like a pcap or pcapng file.
func (sb *Sandbox) Add(name string, r io.Reader, sample bool) (*File, error) {
	sb.mu.Lock()
	room, err := sb.room(sample)
	if err != nil {
		sb.mu.Unlock()
		return nil, err
	}
	name = sb.cleanName(name)
	// reserve the name while the upload runs
	f := &File{Name: name, Added: time.Now().UTC(), Status: "uploading", Sample: sample}
	sb.files = append(sb.files, f)
	sb.mu.Unlock()
	fail := func(err error) (*File, error) {
		sb.mu.Lock()
		sb.files = slicesDelete(sb.files, f)
		sb.mu.Unlock()
		os.Remove(filepath.Join(sb.pcapDir(), name))
		return nil, err
	}
	if err := os.MkdirAll(sb.pcapDir(), 0o755); err != nil {
		return fail(err)
	}
	head := make([]byte, 4)
	if _, err := io.ReadFull(r, head); err != nil || pcapfile.Format(head) == "" {
		return fail(ErrNotCapture)
	}
	path := filepath.Join(sb.pcapDir(), name)
	out, err := os.Create(path)
	if err != nil {
		return fail(err)
	}
	out.Write(head)
	written, err := io.Copy(out, io.LimitReader(r, room-4+1))
	out.Close()
	if err != nil {
		return fail(err)
	}
	if written+4 > room {
		return fail(sb.sizeErr())
	}
	sb.mu.Lock()
	f.Size = written + 4
	f.Status = "waiting"
	f.Exporter = sb.exporterFor(f)
	sb.save()
	sb.refreshInventory()
	sb.mu.Unlock()
	sb.kick()
	return f, nil
}

// room returns how many bytes the next file may have; sb.mu held.
func (sb *Sandbox) room(sample bool) (int64, error) {
	if sample {
		return 1 << 30, nil
	}
	n, used := 0, int64(0)
	for _, f := range sb.files {
		if !f.Sample {
			n++
			used += f.Size
		}
	}
	if n >= sb.Lim.Files {
		return 0, fmt.Errorf("%w: at most %d files; delete one first", ErrLimit, sb.Lim.Files)
	}
	room := min(sb.Lim.FileSize, sb.Lim.TotalSize-used)
	if room <= 0 {
		return 0, sb.sizeErr()
	}
	return room, nil
}

func (sb *Sandbox) sizeErr() error {
	return fmt.Errorf("%w: each file at most %d MB, %d MB in all", ErrLimit, sb.Lim.FileSize>>20, sb.Lim.TotalSize>>20)
}

// AddPath imports a capture file where it is, without copying it; the file
// is never deleted.
func (sb *Sandbox) AddPath(path string) (*File, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	fi, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	if !fi.Mode().IsRegular() {
		return nil, fmt.Errorf("%s is not a file", path)
	}
	in, err := os.Open(abs)
	if err != nil {
		return nil, err
	}
	head := make([]byte, 4)
	_, err = io.ReadFull(in, head)
	in.Close()
	if err != nil || pcapfile.Format(head) == "" {
		return nil, fmt.Errorf("%s: %w", path, ErrNotCapture)
	}
	sb.mu.Lock()
	defer sb.mu.Unlock()
	room, err := sb.room(false)
	if err != nil {
		return nil, err
	}
	if fi.Size() > room {
		return nil, sb.sizeErr()
	}
	f := &File{Name: sb.cleanName(filepath.Base(abs)), Path: abs, Size: fi.Size(), Added: time.Now().UTC(), Status: "waiting"}
	sb.files = append(sb.files, f)
	f.Exporter = sb.exporterFor(f)
	sb.save()
	sb.refreshInventory()
	sb.kick()
	return f, nil
}

// exporterFor picks a free address 127.0.1.n for a file's flows.
func (sb *Sandbox) exporterFor(f *File) string {
	used := map[string]bool{}
	for _, o := range sb.files {
		used[o.Exporter] = true
	}
	for i := 1; ; i++ {
		a := fmt.Sprintf("127.0.1.%d", i)
		if !used[a] {
			return a
		}
	}
}

func slicesDelete(fs []*File, f *File) []*File {
	out := fs[:0]
	for _, x := range fs {
		if x != f {
			out = append(out, x)
		}
	}
	return out
}

// Delete removes one file (name) or everything (name ""). The database is
// rebuilt from the remaining files.
func (sb *Sandbox) Delete(name string) error {
	sb.mu.Lock()
	var keep []*File
	found := name == ""
	for _, f := range sb.files {
		if name == "" || f.Name == name {
			found = true
			if f.Status == "uploading" {
				keep = append(keep, f)
				continue
			}
			if f.Path == "" {
				os.Remove(filepath.Join(sb.pcapDir(), f.Name))
			}
			continue
		}
		keep = append(keep, f)
	}
	if !found {
		sb.mu.Unlock()
		return errors.New("no such file")
	}
	sb.files = keep
	sb.gen++
	st := sb.st
	sb.st, sb.det = nil, nil
	for _, f := range sb.files {
		if f.Status != "uploading" {
			f.Status, f.Error, f.Packets, f.Skipped, f.Flows, f.First, f.Last = "waiting", "", 0, 0, 0, time.Time{}, time.Time{}
		}
	}
	sb.save()
	sb.refreshInventory()
	sb.mu.Unlock()
	if st != nil {
		st.Close()
	}
	if err := os.RemoveAll(sb.dbDir()); err != nil {
		return err
	}
	if len(keep) == 0 {
		os.RemoveAll(sb.Dir)
	}
	sb.kick()
	return nil
}

func (sb *Sandbox) kick() {
	select {
	case sb.wake <- struct{}{}:
	default:
	}
}

func (sb *Sandbox) worker() {
	for range sb.wake {
		for sb.importNext() {
		}
	}
}

// importNext imports one waiting file; it reports whether it did.
func (sb *Sandbox) importNext() bool {
	sb.mu.Lock()
	var f *File
	for _, x := range sb.files {
		if x.Status == "waiting" {
			f = x
			break
		}
	}
	if f == nil {
		sb.mu.Unlock()
		return false
	}
	if sb.st == nil {
		if err := sb.open(); err != nil {
			f.Status, f.Error = "error", err.Error()
			sb.save()
			sb.mu.Unlock()
			return true
		}
	}
	f.Status = "importing"
	sb.busy = true
	gen, st, det, inv := sb.gen, sb.st, sb.det, sb.inv
	sb.mu.Unlock()

	res, err := sb.importFile(gen, f, st, inv)
	if err == nil {
		// the rules over the whole capture, window by window
		every := det.Cfg.Every
		for t := res.first.Truncate(every).Add(every); !t.After(res.last.Add(every)); t = t.Add(every) {
			if _, err := det.Run(t); err != nil {
				log.Printf("sandbox: detect: %v", err)
				break
			}
		}
	}
	sb.mu.Lock()
	defer sb.mu.Unlock()
	sb.busy = false
	if gen != sb.gen { // deleted meanwhile
		return true
	}
	f.Packets, f.Skipped, f.Flows, f.First, f.Last = res.packets, res.skipped, res.flows, res.first, res.last
	switch {
	case err != nil:
		f.Status, f.Error = "error", err.Error()
	case res.flows == 0:
		f.Status, f.Error = "error", "no IP packets found"
	default:
		f.Status = "done"
	}
	sb.save()
	log.Printf("sandbox: %s: %d packets, %d flows, %s – %s (%v)", f.Name, res.packets, res.flows,
		res.first.Format(time.RFC3339), res.last.Format(time.RFC3339), err)
	return true
}

type result struct {
	packets, skipped, flows uint64
	first, last             time.Time
}

func (sb *Sandbox) importFile(gen int, f *File, st *store.Store, inv *enrich.Inventory) (result, error) {
	var res result
	in, err := os.Open(sb.filePath(f))
	if err != nil {
		return res, err
	}
	defer in.Close()
	rd, err := pcapfile.NewReader(in)
	if err != nil {
		return res, err
	}
	p := pipeline.New(pipeline.Config{}, st, inv, sb.ASN, sb.Thr)
	h := fnv.New32a()
	h.Write([]byte(f.Name))
	exp, _ := netip.ParseAddr(f.Exporter)
	n := 0
	b := newBuilder(exp, h.Sum32(), func(recs []flow.Record) {
		res.flows += uint64(len(recs))
		p.Ingest(recs)
		if n++; n%50 == 0 {
			p.FlushRows()
		}
	})
	var pk pcapfile.Packet
	for {
		err := rd.Next(&pk)
		if err == io.EOF {
			break
		}
		if err != nil {
			// keep what was read before the damage
			log.Printf("sandbox: %s: %v", f.Name, err)
			break
		}
		if pk.Time.Year() < 1995 || pk.Time.After(time.Now().Add(24*time.Hour)) {
			res.skipped++
			continue
		}
		if res.first.IsZero() || pk.Time.Before(res.first) {
			res.first = pk.Time
		}
		if pk.Time.After(res.last) {
			res.last = pk.Time
		}
		b.add(&pk)
		if rd.Count%100000 == 0 {
			sb.mu.Lock()
			stale := gen != sb.gen
			f.Packets = rd.Count
			sb.mu.Unlock()
			if stale {
				return res, errors.New("deleted")
			}
		}
	}
	b.sweep(res.last.Add(time.Hour), true)
	p.FlushRows()
	p.FlushRollups()
	res.packets, res.skipped = rd.Count, res.skipped+b.skipped
	return res, nil
}

// Close closes the database.
func (sb *Sandbox) Close() {
	sb.mu.Lock()
	st := sb.st
	sb.st = nil
	sb.mu.Unlock()
	if st != nil {
		st.Close()
	}
}
