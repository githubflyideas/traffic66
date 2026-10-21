package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/netip"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/githubflyideas/traffic66/internal/store"
)

// The rooms-and-racks page is a picture, not an asset ledger: for every
// device that sends flows it shows the room and rack it stands in, and for
// its interfaces the carrier and circuit they go out to, so that whoever
// looks at a flow can see which room, rack, device, port and line it came
// through. It is written down once, in the inventory file beside the
// names, and nothing is looked up in it.

type rackIface struct {
	IfIndex uint32 `json:"ifindex"`
	Name    string `json:"name,omitempty"`
	Speed   uint64 `json:"speed,omitempty"`
	ISP     string `json:"isp,omitempty"`
	Circuit string `json:"circuit,omitempty"`
	Wire    uint64 `json:"wire"` // bytes in the last day, busiest first
}

type rackDevice struct {
	Exporter string      `json:"exporter"`
	Name     string      `json:"name,omitempty"`
	Room     string      `json:"room,omitempty"`
	Rack     string      `json:"rack,omitempty"`
	Seen     bool        `json:"seen"` // sent flows recently
	Ifaces   []rackIface `json:"ifaces"`
}

// getRacks lists the devices that send flows, and those the inventory
// names, with their interfaces: the ones flows were sampled on in the last
// day, and any other that has a carrier written down.
func (s *Server) getRacks(w http.ResponseWriter, r *http.Request) {
	devs := map[string]*rackDevice{}
	dev := func(exp string) *rackDevice {
		d, ok := devs[exp]
		if !ok {
			d = &rackDevice{Exporter: exp, Ifaces: []rackIface{}}
			devs[exp] = d
		}
		return d
	}
	if s.Col != nil {
		for _, si := range s.Col.Sources() {
			dev(si.Exporter).Seen = true
		}
	}
	for _, a := range s.Inv.Devices() {
		dev(a.String())
	}
	have := map[string]bool{}
	if s.Store != nil {
		now := time.Now().UTC()
		list, err := s.Store.Interfaces(store.Query{From: now.Add(-24 * time.Hour), To: now})
		if err != nil {
			fail(w, err)
			return
		}
		for _, f := range list {
			if f.Peer || f.IfIndex == 0 {
				continue
			}
			d := dev(f.Exporter)
			d.Ifaces = append(d.Ifaces, rackIface{IfIndex: f.IfIndex, Speed: f.Speed, Wire: f.InWire + f.OutWire})
			have[ifKey(f.Exporter, f.IfIndex)] = true
		}
	}
	for k := range s.Inv.Circuits() {
		if have[k] {
			continue
		}
		exp, idx, _ := strings.Cut(k, "/")
		n, err := strconv.ParseUint(idx, 10, 32)
		if err != nil {
			continue
		}
		d := dev(exp)
		d.Ifaces = append(d.Ifaces, rackIface{IfIndex: uint32(n)})
	}
	out := make([]rackDevice, 0, len(devs))
	for _, d := range devs {
		if a, err := netip.ParseAddr(d.Exporter); err == nil {
			d.Name = s.Inv.Device(a)
			pl := s.Inv.PlaceOf(a)
			d.Room, d.Rack = pl.Room, pl.Rack
			for i := range d.Ifaces {
				it := &d.Ifaces[i]
				ifc := s.Inv.Iface(a, it.IfIndex)
				it.Name, it.ISP, it.Circuit = ifc.Name, ifc.ISP, ifc.Circuit
				if it.Speed == 0 {
					it.Speed = ifc.Speed
				}
			}
		}
		sort.SliceStable(d.Ifaces, func(i, j int) bool {
			if (d.Ifaces[i].ISP != "") != (d.Ifaces[j].ISP != "") {
				return d.Ifaces[i].ISP != ""
			}
			if d.Ifaces[i].Wire != d.Ifaces[j].Wire {
				return d.Ifaces[i].Wire > d.Ifaces[j].Wire
			}
			return d.Ifaces[i].IfIndex < d.Ifaces[j].IfIndex
		})
		out = append(out, *d)
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Room != b.Room {
			return a.Room < b.Room
		}
		if a.Rack != b.Rack {
			return a.Rack < b.Rack
		}
		return a.Exporter < b.Exporter
	})
	writeJSON(w, http.StatusOK, map[string]any{"devices": out, "admin": s.isAdmin(r)})
}

// putRackDevice names a device and says which room and rack it is in.
func (s *Server) putRackDevice(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Exporter, Name, Room, Rack string
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10)).Decode(&in); err != nil {
		fail(w, errors.New("bad request"))
		return
	}
	a, err := netip.ParseAddr(in.Exporter)
	if err != nil {
		fail(w, errors.New("bad exporter address"))
		return
	}
	if err := s.Inv.SetDevice(a, clip(in.Name, 80), clip(in.Room, 60), clip(in.Rack, 60)); err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// putRackIface names an interface and says which carrier and circuit it
// goes out to.
func (s *Server) putRackIface(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Exporter     string
		IfIndex      uint32
		Name         string
		ISP, Circuit string
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4<<10)).Decode(&in); err != nil {
		fail(w, errors.New("bad request"))
		return
	}
	a, err := netip.ParseAddr(in.Exporter)
	if err != nil || in.IfIndex == 0 {
		fail(w, errors.New("bad interface"))
		return
	}
	if err := s.Inv.SetCircuit(a, in.IfIndex, clip(in.Name, 80), clip(in.ISP, 60), clip(in.Circuit, 80)); err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// clip trims a value typed into a form to a sane length.
func clip(s string, n int) string {
	s = strings.TrimSpace(s)
	if r := []rune(s); len(r) > n {
		s = string(r[:n])
	}
	return s
}
