package embed

import (
	"net/netip"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLinks(t *testing.T) {
	path := filepath.Join(t.TempDir(), "embeds.json")
	s, _ := Open(path)
	now := time.Date(2026, 10, 21, 9, 0, 0, 0, time.UTC)
	s.now = func() time.Time { return now }

	key, err := s.Put(Source{Name: "Asset platform", Ranges: []string{"203.0.113.0/24", "2001:db8::10"},
		Origins: []string{"https://assets.example.com/vips", "*.portal.example.net"}, Clients: nil})
	if err != nil || !strings.HasPrefix(key, KeyPrefix) {
		t.Fatalf("put: %q %v", key, err)
	}
	vip, from := netip.MustParseAddr("203.0.113.10"), "https://assets.example.com/vip/42"
	ok := func(r Request) error { _, err := s.Check(r); return err }
	req := Request{Key: key, Addr: vip, Viewer: netip.MustParseAddr("192.0.2.1"), Referer: from, Page: true}
	if err := ok(req); err != nil {
		t.Fatalf("the platform's own link: %v", err)
	}
	for name, c := range map[string]struct {
		r    Request
		want error
	}{
		"another address":    {Request{Key: key, Addr: netip.MustParseAddr("198.51.100.1"), Referer: from, Page: true}, ErrScope},
		"another site":       {Request{Key: key, Addr: vip, Referer: "https://evil.example.org/", Page: true}, ErrOrigin},
		"no referer":         {Request{Key: key, Addr: vip, Page: true}, ErrOrigin},
		"a made-up key":      {Request{Key: KeyPrefix + "guess", Addr: vip, Referer: from, Page: true}, ErrKey},
		"subdomain allowed":  {Request{Key: key, Addr: vip, Referer: "https://a.portal.example.net/x", Page: true}, nil},
		"IPv6 single":        {Request{Key: key, Addr: netip.MustParseAddr("2001:db8::10"), Referer: from, Page: true}, nil},
		"data, not the page": {Request{Key: key, Addr: vip}, nil},
	} {
		if err := ok(c.r); err != c.want {
			t.Errorf("%s: %v, want %v", name, err, c.want)
		}
	}
	if list := s.List(); len(list) != 1 || list[0].Hash != "" || list[0].Hits == 0 {
		t.Fatalf("list %+v", list)
	}

	// viewers limited to the platform's office
	id := s.List()[0].ID
	if _, err := s.Put(Source{ID: id, Name: "Asset platform", Ranges: []string{"203.0.113.0/24"}, Origins: []string{"*"}, Clients: []string{"192.0.2.0/24"}}); err != nil {
		t.Fatal(err)
	}
	if err := ok(Request{Key: key, Addr: vip, Viewer: netip.MustParseAddr("198.51.100.9")}); err != ErrViewer {
		t.Errorf("a viewer outside: %v", err)
	}

	// a new key: the old one keeps working for a day, then stops
	key2, err := s.Rotate(id)
	if err != nil {
		t.Fatal(err)
	}
	req = Request{Key: key, Addr: vip, Viewer: netip.MustParseAddr("192.0.2.1"), Page: true}
	if err := ok(req); err != nil {
		t.Errorf("old key during the changeover: %v", err)
	}
	now = now.Add(Grace + time.Minute)
	if err := ok(req); err != ErrKey {
		t.Errorf("old key after the changeover: %v", err)
	}
	req.Key = key2
	if err := ok(req); err != nil {
		t.Errorf("new key: %v", err)
	}

	// so many a minute, then a pause
	var last error
	for range PerMinute + 1 {
		last = ok(req)
	}
	if last != ErrRate {
		t.Errorf("over the ceiling: %v", last)
	}

	// kept on disk without the key itself
	again, err := Open(path)
	if err != nil || len(again.list) != 1 || again.list[0].Hash == "" || strings.Contains(again.list[0].Hash, key2) {
		t.Fatalf("reopened %+v %v", again.list, err)
	}

	// withdrawn: stops at once
	if err := s.Delete(id); err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Minute)
	if err := ok(req); err != ErrKey {
		t.Errorf("withdrawn: %v", err)
	}
}

func TestSourceChecks(t *testing.T) {
	s, _ := Open("")
	for _, bad := range []Source{
		{Name: "", Ranges: []string{"10.0.0.1"}, Origins: []string{"*"}},
		{Name: "x", Origins: []string{"*"}},
		{Name: "x", Ranges: []string{"not an address"}, Origins: []string{"*"}},
		{Name: "x", Ranges: []string{"10.0.0.1"}},
		{Name: "x", Ranges: []string{"10.0.0.1"}, Origins: []string{"bad host!"}},
	} {
		if _, err := s.Put(bad); err == nil {
			t.Errorf("accepted %+v", bad)
		}
	}
}
