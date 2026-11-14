package main

import (
	"fmt"
	"io"
	"net"
	"net/netip"
	"strings"

	"github.com/githubflyideas/traffic66/internal/api"
	"github.com/githubflyideas/traffic66/internal/auth"
)

// firstStart is set when this start created the password file, so the
// startup summary can say which password to sign in with.
var firstStart bool

// publicURL is -public-url, shown as the address to open.
var publicURL string

// fixedPassword is set when -password or TRAFFIC66_PASSWORD gives the password.
var fixedPassword bool

// printStartup prints what somebody starting traffic66 for the first time
// needs and otherwise has to piece together from the log: where the web UI
// is, which address and UDP ports to point the devices at, and how to sign
// in. It goes to stdout, apart from the log, so it stays readable.
func printStartup(w io.Writer, scheme string, web net.Addr, listens []api.ListenInfo, demo bool, user string, first bool) {
	_, port, _ := net.SplitHostPort(web.String())
	wildcard := isWildcard(web)
	ips := api.LocalIPs()
	fmt.Fprintf(w, "\ntraffic66 %s is running%s\n", version, map[bool]string{true: " (demo: simulated devices)"}[demo])
	if publicURL != "" {
		fmt.Fprintf(w, "  Web UI     %s  (through the reverse proxy)\n", publicURL)
	}
	fmt.Fprintf(w, "  Web UI     %s://%s\n", scheme, displayAddr(web))
	if wildcard {
		for _, ip := range ips {
			fmt.Fprintf(w, "             %s://%s\n", scheme, net.JoinHostPort(ip, port))
		}
	}
	switch {
	case fixedPassword:
		fmt.Fprintf(w, "  Sign in    %s, with the password given at start\n", user)
	case demo:
		fmt.Fprintf(w, "  Sign in    %s / %s\n", user, auth.DefaultPassword)
	case first:
		fmt.Fprintf(w, "  Sign in    %s / %s  (you choose a new password at the first sign-in)\n", user, auth.DefaultPassword)
	default:
		fmt.Fprintf(w, "  Sign in    %s, with your password (reset: traffic66 passwd)\n", user)
	}
	if len(listens) > 0 && !demo {
		var ps []string
		for _, l := range listens {
			ps = append(ps, l.Name+" "+l.Port)
		}
		to := "this machine's address"
		if len(ips) > 0 {
			to = strings.Join(ips, " or ")
		}
		local := true // every collector bound to loopback: only this machine can send
		hosts := map[string]bool{}
		for _, l := range listens {
			if a, err := netip.ParseAddr(l.Host); err != nil || !a.IsLoopback() {
				local = false
			} else {
				hosts[a.String()] = true
			}
		}
		if local {
			at := "127.0.0.1"
			if len(hosts) == 1 {
				for h := range hosts {
					at = h
				}
			}
			to = at + " (this machine only)"
		}
		fmt.Fprintf(w, "  Devices    export to %s, UDP %s (any port takes any protocol)\n", to, strings.Join(ps, ", "))
	}
	fmt.Fprintf(w, "  Stop       Ctrl+C\n\n")
}

func isWildcard(a net.Addr) bool {
	s := a.String()
	return strings.HasPrefix(s, "[::]:") || strings.HasPrefix(s, "0.0.0.0:")
}
