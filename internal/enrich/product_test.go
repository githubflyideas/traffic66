package enrich

import (
	"net/netip"
	"strings"
	"testing"
)

// Product lines group service addresses for billing: the line is a name
// and its VIPs, an old inventory without product lines still loads, and
// one address can be claimed by one product only — two claims would put
// the same traffic on two bills.
func TestProductLines(t *testing.T) {
	inv := NewInventory()
	text := "# names\nhost 10.0.0.5 Finance PC\n" +
		"product Shop vip=203.0.113.10,203.0.113.11\n" +
		"product Mail Service vip=203.0.113.20\n"
	if err := inv.Parse(text); err != nil {
		t.Fatal(err)
	}
	ps := inv.Products()
	if len(ps) != 2 || ps[0].Name != "Shop" || ps[1].Name != "Mail Service" {
		t.Fatalf("products = %+v", ps)
	}
	if len(ps[0].VIPs) != 2 || ps[0].VIPs[0] != netip.MustParseAddr("203.0.113.10") {
		t.Fatalf("shop VIPs = %v", ps[0].VIPs)
	}
	if p, ok := inv.ProductOf(netip.MustParseAddr("203.0.113.20")); !ok || p.Name != "Mail Service" {
		t.Errorf("ProductOf(203.0.113.20) = %v, %v", p.Name, ok)
	}
	if _, ok := inv.ProductOf(netip.MustParseAddr("203.0.113.99")); ok {
		t.Error("an unclaimed address has a product")
	}
	if _, ok := inv.Product("Shop"); !ok {
		t.Error("Product(Shop) not found")
	}
}

func TestProductLineRefusals(t *testing.T) {
	for name, text := range map[string]string{
		"claimed twice":  "product A vip=203.0.113.10\nproduct B vip=203.0.113.10\n",
		"listed twice":   "product A vip=203.0.113.10,203.0.113.10\n",
		"product twice":  "product A vip=203.0.113.10\nproduct A vip=203.0.113.11\n",
		"no vip field":   "product A 203.0.113.10\n",
		"vip not last":   "product A vip=203.0.113.10 extra\n",
		"bad address":    "product A vip=203.0.113.999\n",
		"no name":        "product vip=203.0.113.10\n",
		"empty vip list": "product A vip=\n",
	} {
		inv := NewInventory()
		if err := inv.Parse(text); err == nil {
			t.Errorf("%s: parsed without an error", name)
		} else if name == "claimed twice" && !strings.Contains(err.Error(), "already belongs to product A") {
			t.Errorf("%s: error %q does not name the first claim", name, err)
		}
	}
	// a refused inventory leaves the previous one in place
	inv := NewInventory()
	if err := inv.Parse("product A vip=203.0.113.10\n"); err != nil {
		t.Fatal(err)
	}
	if err := inv.Parse("product B vip=203.0.113.10\nproduct C vip=203.0.113.10\n"); err == nil {
		t.Fatal("a double claim parsed")
	}
	if ps := inv.Products(); len(ps) != 1 || ps[0].Name != "A" {
		t.Errorf("after a refused parse, products = %+v", ps)
	}
}
