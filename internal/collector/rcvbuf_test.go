package collector

import (
	"os"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// netAdmin reports whether this process may raise buffers past the cap
// (CAP_NET_ADMIN); root in a container often may not.
func netAdmin() bool {
	b, err := os.ReadFile("/proc/self/status")
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(b), "\n") {
		if v, ok := strings.CutPrefix(line, "CapEff:"); ok {
			caps, err := strconv.ParseUint(strings.TrimSpace(v), 16, 64)
			return err == nil && caps&(1<<12) != 0
		}
	}
	return false
}

// A listener gets the receive buffer it asks for: past the system's cap
// when it may raise it (CAP_NET_ADMIN, as the service runs), else as
// much as the cap allows.
func TestReceiveBuffer(t *testing.T) {
	c := New(nil)
	l, err := c.Listen("127.0.0.1:0", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer l.conn.Close()
	if runtime.GOOS == "linux" && netAdmin() && l.RcvBuf < 16<<20 {
		t.Errorf("receive buffer %d KB with CAP_NET_ADMIN, want 16 MB", l.RcvBuf/1024)
	}
	if l.RcvBuf <= 0 && runtime.GOOS != "windows" {
		t.Errorf("receive buffer %d", l.RcvBuf)
	}
}
