package collector

import (
	"os"
	"runtime"
	"testing"
)

// A listener gets the receive buffer it asks for: past the system's cap
// when it may raise it (as root, which is how the service runs), else as
// much as the cap allows.
func TestReceiveBuffer(t *testing.T) {
	c := New(nil)
	l, err := c.Listen("127.0.0.1:0", "test")
	if err != nil {
		t.Fatal(err)
	}
	defer l.conn.Close()
	if runtime.GOOS == "linux" && os.Geteuid() == 0 && l.RcvBuf < 16<<20 {
		t.Errorf("receive buffer %d KB as root, want 16 MB", l.RcvBuf/1024)
	}
	if l.RcvBuf <= 0 && runtime.GOOS != "windows" {
		t.Errorf("receive buffer %d", l.RcvBuf)
	}
}
