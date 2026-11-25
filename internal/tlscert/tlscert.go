// Package tlscert gives the web UI a certificate.
//
// Two ways in. An organisation with its own certificate authority passes
// the certificate and key it already issued, which is what a company that
// cares about this will want. Everyone else gets one traffic66 makes
// itself, kept in the data directory so it survives a restart — a
// certificate that changes on every start would teach people to click
// through the warning, which is worse than no certificate at all.
//
// A self-signed certificate is not trusted by anything, and a browser says
// so. What it does do is stop the password crossing the network in the
// clear, which is the thing worth fixing first. The fingerprint is printed
// at startup and the certificate can be downloaded, so it can be checked
// once by hand or pushed to a company's trust store.
package tlscert

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// File names inside the data directory.
const (
	CertFile = "tls.crt"
	KeyFile  = "tls.key"
)

// Life is how long a certificate traffic66 makes itself is good for, and
// Renew is how close to the end it is replaced. A year is long enough that
// nobody has to think about it and short enough to be ordinary.
const (
	Life  = 365 * 24 * time.Hour
	Renew = 30 * 24 * time.Hour
)

// Info describes the certificate in use, for the startup line and the
// settings page.
type Info struct {
	SelfSigned  bool
	Names       []string // what the certificate is valid for
	NotAfter    time.Time
	Fingerprint string // SHA-256 of the certificate, as a browser shows it
	CertPath    string
}

// Load returns the TLS configuration for the web UI. With cert and key
// given, those are used as they are. With neither, one is made and kept in
// dir, and replaced when it is close to running out.
func Load(dir, cert, key string) (*tls.Config, Info, error) {
	if (cert == "") != (key == "") {
		return nil, Info{}, fmt.Errorf("tls: give both a certificate and a key, or neither")
	}
	if cert != "" {
		pair, err := tls.LoadX509KeyPair(cert, key)
		if err != nil {
			return nil, Info{}, fmt.Errorf("tls: %w", err)
		}
		return config(pair), describe(pair, false, cert), nil
	}

	cert, key = filepath.Join(dir, CertFile), filepath.Join(dir, KeyFile)
	pair, err := tls.LoadX509KeyPair(cert, key)
	if err == nil && !expiring(pair) {
		return config(pair), describe(pair, true, cert), nil
	}
	if pair, err = generate(cert, key); err != nil {
		return nil, Info{}, err
	}
	return config(pair), describe(pair, true, cert), nil
}

func config(pair tls.Certificate) *tls.Config {
	return &tls.Config{Certificates: []tls.Certificate{pair}, MinVersion: tls.VersionTLS12}
}

func expiring(pair tls.Certificate) bool {
	leaf, err := leafOf(pair)
	return err != nil || time.Until(leaf.NotAfter) < Renew
}

func leafOf(pair tls.Certificate) (*x509.Certificate, error) {
	if pair.Leaf != nil {
		return pair.Leaf, nil
	}
	if len(pair.Certificate) == 0 {
		return nil, fmt.Errorf("tls: no certificate")
	}
	return x509.ParseCertificate(pair.Certificate[0])
}

func describe(pair tls.Certificate, self bool, path string) Info {
	in := Info{SelfSigned: self, CertPath: path}
	leaf, err := leafOf(pair)
	if err != nil {
		return in
	}
	in.NotAfter = leaf.NotAfter
	in.Names = append(append([]string{}, leaf.DNSNames...), ips(leaf)...)
	sum := sha256.Sum256(leaf.Raw)
	var b strings.Builder
	for i, c := range hex.EncodeToString(sum[:]) {
		if i > 0 && i%2 == 0 {
			b.WriteByte(':')
		}
		b.WriteRune(c)
	}
	in.Fingerprint = strings.ToUpper(b.String())
	return in
}

func ips(c *x509.Certificate) []string {
	out := make([]string, 0, len(c.IPAddresses))
	for _, ip := range c.IPAddresses {
		out = append(out, ip.String())
	}
	return out
}

// generate writes a new self-signed certificate for every name this
// machine answers to, because the address someone types is rarely the one
// that was thought of when the certificate was made.
func generate(certPath, keyPath string) (tls.Certificate, error) {
	k, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, err
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return tls.Certificate{}, err
	}
	host, _ := os.Hostname()
	names, addrs := subject(host)
	now := time.Now()
	tpl := x509.Certificate{
		SerialNumber:          serial,
		Subject:               pkix.Name{CommonName: first(names, "traffic66"), Organization: []string{"traffic66"}},
		NotBefore:             now.Add(-time.Hour), // a clock a little behind ours still accepts it
		NotAfter:              now.Add(Life),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true, // so it can be put in a trust store as it is
		DNSNames:              names,
		IPAddresses:           addrs,
	}
	der, err := x509.CreateCertificate(rand.Reader, &tpl, &tpl, &k.PublicKey, k)
	if err != nil {
		return tls.Certificate{}, err
	}
	keyDER, err := x509.MarshalECPrivateKey(k)
	if err != nil {
		return tls.Certificate{}, err
	}
	if err := write(certPath, "CERTIFICATE", der, 0o644); err != nil {
		return tls.Certificate{}, err
	}
	// 0600 on anything with POSIX permissions. Windows has none — Go maps
	// only the read-only attribute — so there the key is as private as the
	// data directory is, which is the same thing that protects the
	// password file next to it.
	if err := write(keyPath, "EC PRIVATE KEY", keyDER, 0o600); err != nil {
		return tls.Certificate{}, err
	}
	return tls.LoadX509KeyPair(certPath, keyPath)
}

func write(path, kind string, der []byte, mode os.FileMode) error {
	b := pem.EncodeToMemory(&pem.Block{Type: kind, Bytes: der})
	tmp := path + ".new"
	if err := os.WriteFile(tmp, b, mode); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}

// subject lists what the certificate should be valid for: this machine's
// name, every address it has, and localhost.
func subject(host string) ([]string, []net.IP) {
	names := []string{}
	seen := map[string]bool{}
	add := func(n string) {
		if n != "" && !seen[n] {
			seen[n] = true
			names = append(names, n)
		}
	}
	add(host)
	if host != "" && !strings.Contains(host, ".") {
		add(host + ".local")
	}
	add("localhost")

	addrs := []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback}
	ifaces, _ := net.InterfaceAddrs()
	for _, a := range ifaces {
		ipn, ok := a.(*net.IPNet)
		if !ok || ipn.IP.IsLoopback() || ipn.IP.IsLinkLocalUnicast() {
			continue
		}
		addrs = append(addrs, ipn.IP)
	}
	return names, addrs
}

func first(s []string, def string) string {
	if len(s) > 0 {
		return s[0]
	}
	return def
}
