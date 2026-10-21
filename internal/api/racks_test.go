package api

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/githubflyideas/traffic66/internal/enrich"
	"github.com/githubflyideas/traffic66/internal/store"
)

// The page lists every device that sent flows, with the interfaces flows
// were sampled on, and what was written down about where they are.
func TestRacks(t *testing.T) {
	st, err := store.Open(store.Options{Dir: t.TempDir(), Threads: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ts := time.Now().UTC().Add(-time.Hour)
	if err := st.AppendRows([]store.Row{
		{TS: ts, Exporter: "10.0.0.1", InIf: 3, OutIf: 0, Wire: 900, Bytes: 900, Pkts: 1, Flows: 1},
		{TS: ts, Exporter: "10.0.0.1", InIf: 7, OutIf: 0, Wire: 100, Bytes: 100, Pkts: 1, Flows: 1},
	}); err != nil {
		t.Fatal(err)
	}
	inv := enrich.NewInventory()
	inv.Save("device 10.0.0.9 Spare switch\niface 10.0.0.9 2 Backup line isp=Other_Carrier\n")
	s := &Server{Store: st, Inv: inv}

	put := func(handler string, body string) int {
		w := httptest.NewRecorder()
		r := httptest.NewRequest("POST", "/api/racks/"+handler, strings.NewReader(body))
		if handler == "device" {
			s.putRackDevice(w, r)
		} else {
			s.putRackIface(w, r)
		}
		return w.Code
	}
	if c := put("device", `{"exporter":"10.0.0.1","name":"Core router","room":"Room 1","rack":"A03"}`); c != 200 {
		t.Fatal(c)
	}
	if c := put("iface", `{"exporter":"10.0.0.1","ifindex":3,"name":"Uplink","isp":"China Telecom","circuit":"SH-12345"}`); c != 200 {
		t.Fatal(c)
	}
	if c := put("iface", `{"exporter":"not an address","ifindex":3}`); c != 400 {
		t.Fatalf("a bad exporter was accepted: %d", c)
	}

	w := httptest.NewRecorder()
	s.getRacks(w, httptest.NewRequest("GET", "/api/racks", nil))
	var out struct {
		Devices []rackDevice `json:"devices"`
	}
	json.Unmarshal(w.Body.Bytes(), &out)
	if len(out.Devices) != 2 {
		t.Fatalf("devices %+v", out.Devices)
	}
	core := out.Devices[1]
	if core.Room != "Room 1" {
		core = out.Devices[0]
	}
	if core.Exporter != "10.0.0.1" || core.Name != "Core router" || core.Rack != "A03" {
		t.Fatalf("core %+v", core)
	}
	if len(core.Ifaces) == 0 || core.Ifaces[0].IfIndex != 3 || core.Ifaces[0].ISP != "China Telecom" || core.Ifaces[0].Circuit != "SH-12345" {
		t.Fatalf("core interfaces %+v", core.Ifaces)
	}
	// a device that sent nothing still shows when the inventory names it,
	// with the interface that has a carrier written down
	var spare rackDevice
	for _, d := range out.Devices {
		if d.Exporter == "10.0.0.9" {
			spare = d
		}
	}
	if spare.Name != "Spare switch" || len(spare.Ifaces) != 1 || spare.Ifaces[0].ISP != "Other Carrier" {
		t.Fatalf("spare %+v", spare)
	}
}
