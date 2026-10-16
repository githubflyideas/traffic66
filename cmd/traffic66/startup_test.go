package main

import (
	"bytes"
	"net"
	"strings"
	"testing"

	"github.com/githubflyideas/traffic66/internal/api"
)

func TestPrintStartup(t *testing.T) {
	var b bytes.Buffer
	web := &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 8066}
	printStartup(&b, "http", web, []api.ListenInfo{{Name: "sflow", Port: "6343"}, {Name: "netflow", Port: "2055"}}, false, "admin", true)
	out := b.String()
	for _, want := range []string{"http://127.0.0.1:8066", "admin / traffic66", "new password", "UDP sflow 6343, netflow 2055"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	b.Reset()
	printStartup(&b, "http", web, nil, true, "admin", false)
	if strings.Contains(b.String(), "Devices") || !strings.Contains(b.String(), "demo") {
		t.Errorf("demo summary:\n%s", b.String())
	}
}
