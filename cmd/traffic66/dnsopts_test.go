package main

import (
	"testing"
	"time"
)

// Tuning reverse lookups is a registered feature: unregistered, the
// upstream, rate and cache flags give way to the defaults and are named
// for the log; registered, they are used as given.
func TestDNSOptionsByRegistration(t *testing.T) {
	f := serveFlags{dnsUpstream: "10.0.0.53:53", dnsRate: 100, dnsTTL: time.Hour}
	o, ignored := dnsOptions(f, true)
	if o.Upstream != "10.0.0.53:53" || o.PerSecond != 100 || o.TTL != time.Hour || len(ignored) != 0 {
		t.Errorf("registered: %+v %v", o, ignored)
	}
	o, ignored = dnsOptions(f, false)
	if o.Upstream != "" || o.PerSecond != dnsDefaultRate || o.TTL != dnsDefaultTTL || len(ignored) != 3 {
		t.Errorf("unregistered: %+v %v", o, ignored)
	}
	// the defaults themselves are not "ignored": nothing to say
	if _, ignored := dnsOptions(serveFlags{dnsRate: dnsDefaultRate, dnsTTL: dnsDefaultTTL}, false); len(ignored) != 0 {
		t.Errorf("defaults reported as ignored: %v", ignored)
	}
}
