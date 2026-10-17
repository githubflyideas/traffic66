package main

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/githubflyideas/traffic66/internal/sandbox"
)

func TestCheckCaptures(t *testing.T) {
	dir := t.TempDir()
	pcap := []byte{0xd4, 0xc3, 0xb2, 0xa1, 2, 0, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0xff, 0xff, 0, 0, 1, 0, 0, 0}
	write := func(name string, b []byte) string {
		p := filepath.Join(dir, name)
		os.WriteFile(p, b, 0o644)
		return p
	}
	a, b, c, d := write("a.pcap", pcap), write("b.pcapng", append([]byte{0x0a, 0x0d, 0x0d, 0x0a}, make([]byte, 100)...)), write("c.cap", pcap), write("d.pcap", pcap)
	txt := write("notes.txt", []byte("hello world"))
	lim := sandbox.Limits{Files: 3, FileSize: 1000, TotalSize: 140}
	for _, tc := range []struct {
		files []string
		err   string
	}{
		{[]string{a, b}, ""},
		{[]string{a, b, c, d}, "at most 3 files"},
		{[]string{txt}, "not a capture file"},
		{[]string{a, b, c}, "in all"},
	} {
		err := checkCaptures(tc.files, lim)
		if (tc.err == "") != (err == nil) || err != nil && !strings.Contains(err.Error(), tc.err) {
			t.Errorf("%v: %v, want %q", tc.files, err, tc.err)
		}
	}
	// the message differs between systems
	if err := checkCaptures([]string{filepath.Join(dir, "missing.pcap")}, lim); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("missing file: %v", err)
	}
	if !looksLikeCapture("x.PCAPNG") || looksLikeCapture("serv") || !looksLikeCapture(txt) {
		t.Error("looksLikeCapture")
	}
}
