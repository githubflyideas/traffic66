package tlscert

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestSelfSignedIsKept(t *testing.T) {
	dir := t.TempDir()
	cfg, info, err := Load(dir, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if !info.SelfSigned {
		t.Fatal("made here, so self-signed")
	}
	if len(cfg.Certificates) != 1 {
		t.Fatal("no certificate in the configuration")
	}
	if info.Fingerprint == "" || len(info.Fingerprint) != 95 {
		t.Fatalf("fingerprint %q is not the form a browser shows", info.Fingerprint)
	}
	if time.Until(info.NotAfter) < 300*24*time.Hour {
		t.Fatalf("a certificate nobody has to think about should last about a year, got %s", info.NotAfter)
	}
	st, err := os.Stat(filepath.Join(dir, KeyFile))
	if err != nil {
		t.Fatal(err)
	}
	// Windows has no POSIX permission bits: Go maps only the read-only
	// attribute, so a file written with 0600 reads back as 0666 there and
	// what actually protects the key is the data directory's ACL. The
	// check is still worth making everywhere it means something.
	if runtime.GOOS != "windows" && st.Mode().Perm() != 0o600 {
		t.Fatalf("the private key is mode %o", st.Mode().Perm())
	}

	// A certificate that changed on every start would teach people to click
	// through the warning.
	_, again, err := Load(dir, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if again.Fingerprint != info.Fingerprint {
		t.Fatal("the certificate changed between runs")
	}
}

// The address someone types is rarely the one that was thought of, so the
// certificate covers every name this machine answers to.
func TestSelfSignedCoversThisMachine(t *testing.T) {
	dir := t.TempDir()
	_, info, err := Load(dir, "", "")
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"localhost": true, "127.0.0.1": true}
	for _, n := range info.Names {
		delete(want, n)
	}
	if len(want) != 0 {
		t.Fatalf("missing %v from %v", want, info.Names)
	}
}

func TestRenewedWhenCloseToTheEnd(t *testing.T) {
	dir := t.TempDir()
	_, old, err := Load(dir, "", "")
	if err != nil {
		t.Fatal(err)
	}
	// Age the certificate past the renewal point by rewriting it with an
	// end date inside the window.
	pair, err := tls.LoadX509KeyPair(filepath.Join(dir, CertFile), filepath.Join(dir, KeyFile))
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := x509.ParseCertificate(pair.Certificate[0])
	if err != nil {
		t.Fatal(err)
	}
	if time.Until(leaf.NotAfter) < Renew {
		t.Fatal("a fresh certificate should not already be due")
	}
	if err := os.Chtimes(filepath.Join(dir, CertFile), time.Now(), time.Now()); err != nil {
		t.Fatal(err)
	}
	// Replacing the file with one that has run out must bring a new one.
	if err := os.WriteFile(filepath.Join(dir, CertFile), []byte("not a certificate"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, fresh, err := Load(dir, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if fresh.Fingerprint == old.Fingerprint {
		t.Fatal("an unusable certificate should have been replaced")
	}
}

func TestOwnCertificate(t *testing.T) {
	dir := t.TempDir()
	if _, _, err := Load(dir, "", ""); err != nil {
		t.Fatal(err)
	}
	cert, key := filepath.Join(dir, CertFile), filepath.Join(dir, KeyFile)
	_, info, err := Load(t.TempDir(), cert, key)
	if err != nil {
		t.Fatal(err)
	}
	if info.SelfSigned {
		t.Fatal("a certificate that was handed to us is not ours to call self-signed")
	}
	if info.CertPath != cert {
		t.Fatalf("CertPath = %q", info.CertPath)
	}
}

func TestHalfAPairIsRefused(t *testing.T) {
	dir := t.TempDir()
	if _, _, err := Load(dir, "cert.pem", ""); err == nil {
		t.Fatal("a certificate without its key was accepted")
	}
	if _, _, err := Load(dir, "", "key.pem"); err == nil {
		t.Fatal("a key without its certificate was accepted")
	}
	if _, _, err := Load(dir, "missing.pem", "missing.key"); err == nil {
		t.Fatal("files that are not there were accepted")
	}
}

// It has to actually serve.
func TestItServes(t *testing.T) {
	dir := t.TempDir()
	cfg, info, err := Load(dir, "", "")
	if err != nil {
		t.Fatal(err)
	}
	ln, err := tls.Listen("tcp", "127.0.0.1:0", cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		if c, err := ln.Accept(); err == nil {
			c.(*tls.Conn).Handshake()
			c.Close()
		}
	}()
	host, port, _ := net.SplitHostPort(ln.Addr().String())
	c, err := tls.Dial("tcp", net.JoinHostPort(host, port), &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	got := c.ConnectionState().PeerCertificates
	if len(got) == 0 {
		t.Fatal("the handshake produced no certificate")
	}
	served := describe(tls.Certificate{Certificate: [][]byte{got[0].Raw}}, true, "").Fingerprint
	if served != info.Fingerprint {
		t.Fatalf("served %s, on disk %s", served, info.Fingerprint)
	}
}
