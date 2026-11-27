package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/githubflyideas/traffic66/internal/auth"
)

func TestDefaultDirAndMemoryLimitAndBytes(t *testing.T) {
	d := defaultDir("traffic66-data")
	if d == "" || d == "traffic66-data" && !strings.Contains(d, string(os.PathSeparator)) {
		// defaultDir returns just the name only if Executable fails
		if !filepath.IsAbs(d) && !strings.Contains(d, "traffic66-data") {
			t.Errorf("defaultDir = %q", d)
		}
	}
	if !strings.HasSuffix(d, "traffic66-data") {
		t.Errorf("defaultDir = %q, want a traffic66-data suffix", d)
	}
	os.Unsetenv("GOMEMLIMIT")
	setGoMemoryLimit(0.10)
	os.Setenv("GOMEMLIMIT", "256MiB")
	setGoMemoryLimit(0.10) // should no-op
	os.Unsetenv("GOMEMLIMIT")

	if got := humanBytes(2e9); !strings.Contains(got, "GB") {
		t.Errorf("humanBytes GB = %s", got)
	}
	if got := humanBytes(3e6); !strings.Contains(got, "MB") {
		t.Errorf("humanBytes MB = %s", got)
	}
	if got := humanBytes(400); got != "400 bytes" {
		t.Errorf("humanBytes small = %s", got)
	}
}

func TestLoginCheckerEnvPassword(t *testing.T) {
	defer func() { fixedPassword, firstStart = false, false }()
	dir := t.TempDir()
	t.Setenv("TRAFFIC66_PASSWORD", "from-env-99")
	fixedPassword = false
	c := loginChecker(dir, "admin", "", false)
	if !fixedPassword {
		t.Fatal("TRAFFIC66_PASSWORD did not mark the run as fixed-password")
	}
	if !c.Check("admin", "from-env-99") || c.Check("admin", "traffic66") {
		t.Fatal("env password not honoured")
	}
}

// helperProcess is the in-process child used by the CLI tests that
// would otherwise call os.Exit or wait for a signal.
func TestHelperProcess(t *testing.T) {
	if os.Getenv("T66_HELPER") == "" {
		t.Skip("child only")
	}
	var args []string
	if s := os.Getenv("T66_HELPER_ARGS"); s != "" {
		if err := json.Unmarshal([]byte(s), &args); err != nil {
			t.Fatalf("T66_HELPER_ARGS: %v", err)
		}
	}
	switch os.Getenv("T66_HELPER") {
	case "serve":
		serve(args, false)
	case "demo":
		serve(args, true)
	case "passwd":
		passwd(args)
	case "gen-token":
		genToken(args)
	case "main-version":
		os.Args = append([]string{"traffic66"}, args...)
		main()
	case "main-help":
		os.Args = append([]string{"traffic66"}, args...)
		main()
	default:
		t.Fatalf("unknown helper %s", os.Getenv("T66_HELPER"))
	}
}

func startHelper(t *testing.T, kind string, args []string, extraEnv ...string) *exec.Cmd {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestHelperProcess$")
	b, err := json.Marshal(args)
	if err != nil {
		t.Fatal(err)
	}
	env := append(os.Environ(),
		"T66_HELPER="+kind,
		"T66_HELPER_ARGS="+string(b),
	)
	cmd.Env = append(env, extraEnv...)
	return cmd
}

// stopHelper asks the child to exit: SIGTERM on Unix (serve listens for
// it), Kill on Windows (Signal is not delivered to NotifyContext).
func stopHelper(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	if runtime.GOOS == "windows" {
		return cmd.Process.Kill()
	}
	return cmd.Process.Signal(syscall.SIGTERM)
}

func waitHelper(t *testing.T, cmd *exec.Cmd, timeout time.Duration) error {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		return err
	case <-time.After(timeout):
		_ = cmd.Process.Kill()
		select {
		case err := <-done:
			if err != nil {
				return err
			}
			return nil
		case <-time.After(5 * time.Second):
			return fmt.Errorf("child did not exit after %s", timeout)
		}
	}
}

func TestPasswdGenerateAndList(t *testing.T) {
	dir := t.TempDir()
	cmd := startHelper(t, "passwd", []string{"-data", dir, "-user", "admin", "-generate"})
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("passwd -generate: %v\n%s", err, out)
	}
	if !bytes.Contains(out, []byte("user admin, password ")) {
		t.Fatalf("passwd -generate output: %s", out)
	}
	cmd = startHelper(t, "passwd", []string{"-data", dir, "-list"})
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("passwd -list: %v\n%s", err, out)
	}
	if !bytes.Contains(out, []byte("admin\n")) {
		t.Fatalf("passwd -list: %s", out)
	}
}

func TestGenTokenCreateListRevoke(t *testing.T) {
	dir := t.TempDir()
	if err := auth.Set(dir, "admin", "admin-pass-123"); err != nil {
		t.Fatal(err)
	}
	cmd := startHelper(t, "gen-token", []string{"-data", dir, "-user", "admin", "-name", "probe", "-scope", "read"})
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gen-token: %v\n%s", err, out)
	}
	if !bytes.Contains(out, []byte("token ")) || !bytes.Contains(out, []byte("t66_")) {
		t.Fatalf("gen-token output: %s", out)
	}
	cmd = startHelper(t, "gen-token", []string{"-data", dir, "-list"})
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("gen-token -list: %v\n%s", err, out)
	}
	if !bytes.Contains(out, []byte("probe")) {
		t.Fatalf("list: %s", out)
	}
	// pull the id (first field)
	fields := strings.Fields(string(out))
	if len(fields) < 1 {
		t.Fatalf("no id in %s", out)
	}
	id := fields[0]
	cmd = startHelper(t, "gen-token", []string{"-data", dir, "-revoke", id})
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("revoke: %v\n%s", err, out)
	}
	if !bytes.Contains(out, []byte("revoked")) {
		t.Fatalf("revoke output: %s", out)
	}
}

func TestMainVersionAndHelp(t *testing.T) {
	cmd := startHelper(t, "main-version", []string{"version"})
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("version: %v\n%s", err, out)
	}
	if !bytes.Contains(out, []byte("traffic66 ")) || !bytes.Contains(out, []byte("build)")) {
		t.Fatalf("version: %s", out)
	}
	cmd = startHelper(t, "main-help", []string{"help"})
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("help: %v\n%s", err, out)
	}
	if !bytes.Contains(out, []byte("Usage:")) {
		t.Fatalf("help: %s", out)
	}
	cmd = startHelper(t, "main-help", []string{"help", "version"})
	out, err = cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("help version: %v\n%s", err, out)
	}
	if !bytes.Contains(out, []byte("version")) {
		t.Fatalf("help version: %s", out)
	}
}

func freeListen(t *testing.T, network string) string {
	t.Helper()
	if network == "udp" {
		c, err := net.ListenPacket("udp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		addr := c.LocalAddr().String()
		c.Close()
		return addr
	}
	l, err := net.Listen(network, "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	addr := l.Addr().String()
	l.Close()
	return addr
}

func TestServeStartsAndStops(t *testing.T) {
	dir := t.TempDir()
	addr := freeListen(t, "tcp")
	sflow := freeListen(t, "udp")
	nf := freeListen(t, "udp")
	ipfix := freeListen(t, "udp")
	args := []string{
		"-addr", addr,
		"-data", dir,
		"-listen", "sflow=" + sflow + ",netflow=" + nf + ",ipfix=" + ipfix,
		"-memory", "0.10",
	}
	cmd := startHelper(t, "serve", args)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	stopped := false
	defer func() {
		if stopped {
			return
		}
		// Kill, not Signal: on Windows SIGTERM is not delivered and Wait would hang.
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	}()
	url := "http://" + addr + "/healthz"
	client := &http.Client{Timeout: 2 * time.Second}
	deadline := time.Now().Add(30 * time.Second)
	var last error
	for time.Now().Before(deadline) {
		res, err := client.Get(url)
		if err == nil {
			body, _ := io.ReadAll(res.Body)
			res.Body.Close()
			if res.StatusCode == 200 && bytes.Contains(body, []byte(`"status":"ok"`)) {
				if err := stopHelper(cmd); err != nil {
					t.Fatalf("stop: %v", err)
				}
				err := waitHelper(t, cmd, 15*time.Second)
				stopped = true
				if err != nil && runtime.GOOS != "windows" {
					t.Fatalf("serve exit: %v\n%s%s", err, stdout.String(), stderr.String())
				}
				return
			}
			last = fmt.Errorf("GET %s: %d %s", url, res.StatusCode, body)
		} else {
			last = err
		}
		time.Sleep(100 * time.Millisecond)
	}
	_ = cmd.Process.Kill()
	t.Fatalf("serve did not become healthy: %v\n%s%s", last, stdout.String(), stderr.String())
}

func TestServeRejectsBadFlags(t *testing.T) {
	cmd := startHelper(t, "serve", []string{"-retention-days", "0", "-data", t.TempDir(), "-addr", "127.0.0.1:0", "-listen", ""})
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("bad -retention-days started:\n%s", out)
	}
	if !bytes.Contains(out, []byte("-retention-days")) {
		t.Fatalf("error did not name the flag:\n%s", out)
	}
}
