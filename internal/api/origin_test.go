package api

import (
	"net/http/httptest"
	"net/url"
	"testing"
)

// The origin check behind the usual reverse proxy set-ups, and against a
// page on another site.
func TestSameOrigin(t *testing.T) {
	pub, _ := url.Parse("https://t66.example.org")
	for _, c := range []struct {
		name, origin, host string
		hdr                map[string]string
		public             *url.URL
		site               string
		want               bool
	}{
		{"direct", "http://10.0.0.5:8066", "10.0.0.5:8066", nil, nil, "", true},
		{"no origin (scripts)", "", "127.0.0.1:8066", nil, nil, "", true},
		{"nginx $http_host", "https://t66.example.org:8443", "t66.example.org:8443", nil, nil, "", true},
		{"nginx $host, port 443", "https://t66.example.org", "t66.example.org", nil, nil, "", true},
		{"nginx $host:$server_port", "https://t66.example.org", "t66.example.org:443", nil, nil, "", true},
		{"plain http :80", "http://t66.example.org", "t66.example.org:80", nil, nil, "", true},
		{"upper case", "https://T66.Example.org", "t66.example.org", nil, nil, "", true},
		{"IPv6", "https://[2001:db8::1]", "[2001:db8::1]:443", nil, nil, "", true},
		{"X-Forwarded-Host", "https://t66.example.org:8443", "127.0.0.1:8066", map[string]string{"X-Forwarded-Host": "t66.example.org:8443"}, nil, "", true},
		{"X-Forwarded-Host :443", "https://t66.example.org", "127.0.0.1:8066", map[string]string{"X-Forwarded-Host": "t66.example.org:443, other"}, nil, "", true},
		{"Forwarded", "https://t66.example.org", "127.0.0.1:8066", map[string]string{"Forwarded": `for=192.0.2.1;host="t66.example.org";proto=https`}, nil, "", true},
		{"-public-url", "https://t66.example.org", "127.0.0.1:8066", nil, pub, "", true},
		{"-public-url, direct still works", "http://127.0.0.1:8066", "127.0.0.1:8066", nil, pub, "", true},
		// refused
		{"nginx default (Host = upstream)", "https://t66.example.org", "127.0.0.1:8066", nil, nil, "", false},
		{"$host drops a port", "https://t66.example.org:8443", "t66.example.org", nil, nil, "", false},
		{"other port", "https://t66.example.org:8443", "t66.example.org:9443", nil, nil, "", false},
		{"other site", "https://evil.example", "t66.example.org", nil, nil, "", false},
		{"other site, -public-url", "https://evil.example", "127.0.0.1:8066", nil, pub, "", false},
		{"scheme of -public-url", "http://t66.example.org", "127.0.0.1:8066", nil, pub, "", false},
		{"cross-site fetch", "https://t66.example.org", "t66.example.org", nil, nil, "cross-site", false},
		{"null origin", "null", "t66.example.org", nil, nil, "", false},
	} {
		r := httptest.NewRequest("POST", "/api/login", nil)
		r.Host = c.host
		if c.origin != "" {
			r.Header.Set("Origin", c.origin)
		}
		if c.site != "" {
			r.Header.Set("Sec-Fetch-Site", c.site)
		}
		for k, v := range c.hdr {
			r.Header.Set(k, v)
		}
		s := &Server{PublicURL: c.public}
		if got := s.sameOrigin(r); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}
