package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/githubflyideas/traffic66/internal/pcapfile"
	"github.com/githubflyideas/traffic66/internal/sandbox"
)

// offlineRun holds what "traffic66 file.pcap" passes to serve.
type offlineRun struct {
	files   []string
	browser bool
}

var offline *offlineRun

// looksLikeCapture tells "traffic66 file.pcap" from a mistyped command.
func looksLikeCapture(arg string) bool {
	ext := strings.ToLower(filepath.Ext(arg))
	if ext == ".pcap" || ext == ".pcapng" || ext == ".cap" {
		return true
	}
	fi, err := os.Stat(arg)
	return err == nil && fi.Mode().IsRegular()
}

// checkCaptures checks the files before anything starts: at most the local
// limits, and each a pcap or pcapng file.
func checkCaptures(files []string, lim sandbox.Limits) error {
	if len(files) > lim.Files {
		return fmt.Errorf("at most %d files at a time (got %d)", lim.Files, len(files))
	}
	var total int64
	for _, p := range files {
		fi, err := os.Stat(p)
		if err != nil {
			return err
		}
		if !fi.Mode().IsRegular() {
			return fmt.Errorf("%s is not a file", p)
		}
		f, err := os.Open(p)
		if err != nil {
			return err
		}
		head := make([]byte, 4)
		_, err = io.ReadFull(f, head)
		f.Close()
		if err != nil || pcapfile.Format(head) == "" {
			return fmt.Errorf("%s is not a capture file: use .pcap or .pcapng as saved by Wireshark or tcpdump (not compressed)", p)
		}
		total += fi.Size()
	}
	if total > lim.TotalSize {
		return fmt.Errorf("the files have %s; at most %s in all", humanBytes(total), humanBytes(lim.TotalSize))
	}
	return nil
}

func humanBytes(n int64) string {
	switch {
	case n >= 1e9:
		return fmt.Sprintf("%.1f GB", float64(n)/1e9)
	case n >= 1e6:
		return fmt.Sprintf("%.1f MB", float64(n)/1e6)
	}
	return fmt.Sprintf("%d bytes", n)
}

// runOffline analyses capture files: a private traffic66 on 127.0.0.1 with
// a temporary database that is deleted on exit; the files stay untouched.
func runOffline(args []string) {
	var files, rest []string
	for i, a := range args {
		if strings.HasPrefix(a, "-") {
			rest = args[i:]
			break
		}
		files = append(files, a)
	}
	fs := flag.NewFlagSet("traffic66 FILE.pcap", flag.ExitOnError)
	addr := fs.String("addr", "127.0.0.1:0", "web UI address (default: a free port on this computer only)")
	noBrowser := fs.Bool("no-browser", false, "do not open the browser")
	dns := fs.Bool("dns", false, "look up host names of the addresses (off: a capture's addresses are not sent to DNS)")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: traffic66 FILE.pcap [FILE2.pcapng FILE3.pcap] [flags]\n\nAnalyse capture files (at most %d, %s in all) in the web UI.\n\n", sandbox.LocalLimits.Files, humanBytes(sandbox.LocalLimits.TotalSize))
		fs.PrintDefaults()
	}
	fs.Parse(rest)
	files = append(files, fs.Args()...)
	if err := checkCaptures(files, sandbox.LocalLimits); err != nil {
		fmt.Fprintln(os.Stderr, "traffic66:", err)
		os.Exit(2)
	}
	tmp, err := os.MkdirTemp("", "traffic66-offline-")
	if err != nil {
		fatalf("temporary directory: %v", err)
	}
	defer os.RemoveAll(tmp)
	offline = &offlineRun{files: files, browser: !*noBrowser}
	sargs := []string{"-data", tmp, "-listen", "", "-addr", *addr, "-password", readablePassword(), "-memory", "0.15"}
	if !*dns {
		sargs = append(sargs, "-no-dns")
	}
	serve(sargs, false)
}

// readablePassword is 12 letters and digits without look-alikes.
func readablePassword() string {
	const set = "abcdefghjkmnpqrstuvwxyz23456789"
	b := make([]byte, 12)
	rand.Read(b)
	for i := range b {
		b[i] = set[int(b[i])%len(set)]
	}
	return string(b)
}
