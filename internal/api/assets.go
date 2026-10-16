package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/githubflyideas/traffic66/internal/assets"
	"github.com/githubflyideas/traffic66/internal/license"
)

// The asset ledger. Reading it is open to everyone signed in, minus what
// an administrator has hidden; writing it is the administrator's, because
// a ledger everyone can edit is not a record of anything.
//
// Two separate checks guard every call and neither stands in for the
// other: the licence decides whether the ledger exists at all on this
// installation, and the account decides who may change it. A licence that
// cannot be read grants nothing, and losing one never widens what a
// non-administrator can see.

// ledger answers the request and returns nil when the ledger is not
// available: no licence for it, or the installation has none open.
func (s *Server) ledger(w http.ResponseWriter) *assets.Store {
	if !s.allowed(w, license.FeatDCIM) {
		return nil
	}
	if s.Assets == nil {
		fail(w, errors.New("the asset ledger is not open"))
		return nil
	}
	return s.Assets
}

// isAdmin reports whether the signed-in user manages the others. With no
// password file there is no administrator, so nothing is hidden from and
// nothing may be written by anyone — the ledger is read-only rather than
// open to all.
func (s *Server) isAdmin(r *http.Request) bool {
	// This machine's own token counts: the TUI and the local subcommands
	// carry it, and whoever can read it from the data directory could
	// have changed the files there directly.
	if local(r) {
		return true
	}
	if s.Accounts == nil {
		return false
	}
	u := userOf(r)
	return u != "" && u == s.Accounts.Admin()
}

// requireAdmin answers 403 unless the caller may change the installation
// itself, and is the gate behind every call that deletes stored data or
// reconfigures the installation for everybody.
//
// It is deliberately not admin(): that one asks for a password file it
// can write and answers 409 "fixed" without one, which is right when the
// call is about the account list and wrong here. An installation started
// with -password, or without a data directory, still has an
// administrator — it simply cannot add users — so gating these calls on
// admin() would lock such an installation out of its own settings.
// adminOnly guards h with requireAdmin.
func (s *Server) adminOnly(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.requireAdmin(w, r) {
			h(w, r)
		}
	}
}

func (s *Server) requireAdmin(w http.ResponseWriter, r *http.Request) bool {
	if s.isAdmin(r) {
		return true
	}
	writeJSON(w, http.StatusForbidden, map[string]string{"error": "not_admin"})
	return false
}

// ledgerAdmin is ledger plus the administrator check, for the writes.
func (s *Server) ledgerAdmin(w http.ResponseWriter, r *http.Request) *assets.Store {
	a := s.ledger(w)
	if a == nil {
		return nil
	}
	if !s.isAdmin(r) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "not_admin"})
		return nil
	}
	return a
}

// getAssets returns the whole ledger. It is small by design — a few
// thousand rows at the quotas we sell — so the pages get it in one call
// and do their own joining, rather than asking per rack.
func (s *Server) getAssets(w http.ResponseWriter, r *http.Request) {
	a := s.ledger(w)
	if a == nil {
		return
	}
	admin := s.isAdmin(r)
	sites, err := a.Sites()
	if err != nil {
		fail(w, err)
		return
	}
	racks, err := a.Racks(0)
	if err != nil {
		fail(w, err)
		return
	}
	devices, err := a.Devices(assets.DeviceQuery{})
	if err != nil {
		fail(w, err)
		return
	}
	ports, err := a.Ports(0)
	if err != nil {
		fail(w, err)
		return
	}
	links, err := a.Links()
	if err != nil {
		fail(w, err)
		return
	}
	circuits, err := a.Circuits()
	if err != nil {
		fail(w, err)
		return
	}
	if !admin {
		sites = visible(sites, func(v assets.Site) bool { return v.Hidden })
		racks = visible(racks, func(v assets.Rack) bool { return v.Hidden })
		devices = visible(devices, func(v assets.Device) bool { return v.Hidden })
		circuits = visible(circuits, func(v assets.Circuit) bool { return v.Hidden })
		ports = ownedBy(ports, devices)
		links = cabled(links, ports)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"sites": sites, "racks": racks, "devices": devices, "ports": ports,
		"links": links, "circuits": circuits,
		"counts": a.Counts(), "quota": s.assetQuota(), "admin": admin,
		// the height a new rack is offered with, so the page and the
		// ledger cannot drift apart on it
		"default_units": assets.DefaultUnits,
	})
}

// getIPAM returns the networks with how full they are, and what does not
// add up in the assignments.
func (s *Server) getIPAM(w http.ResponseWriter, r *http.Request) {
	a := s.ledger(w)
	if a == nil {
		return
	}
	use, err := a.Use()
	if err != nil {
		fail(w, err)
		return
	}
	if !s.isAdmin(r) {
		use = visible(use, func(v assets.SubnetUse) bool { return v.Hidden })
	}
	conflicts, err := a.Conflicts()
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"subnets": use, "conflicts": conflicts})
}

// getAddrs lists assignments, filtered by network, device, tag or text.
func (s *Server) getAddrs(w http.ResponseWriter, r *http.Request) {
	a := s.ledger(w)
	if a == nil {
		return
	}
	q := r.URL.Query()
	list, err := a.Addrs(assets.AddrQuery{
		SubnetID: idParam(q.Get("subnet")),
		DeviceID: idParam(q.Get("device")),
		Tag:      q.Get("tag"),
		Search:   q.Get("q"),
		Limit:    limitParam(r, 5000),
	})
	if err != nil {
		fail(w, err)
		return
	}
	if !s.isAdmin(r) {
		list = visible(list, func(v assets.Addr) bool { return v.Hidden })
	}
	writeJSON(w, http.StatusOK, map[string]any{"addrs": list})
}

// getFreeAddrs suggests addresses nothing is assigned to.
func (s *Server) getFreeAddrs(w http.ResponseWriter, r *http.Request) {
	a := s.ledger(w)
	if a == nil {
		return
	}
	n, _ := strconv.Atoi(r.URL.Query().Get("n"))
	free, err := a.FreeAddrs(idParam(r.URL.Query().Get("subnet")), n)
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"free": free})
}

// putAsset saves one record of the kind named in the path.
func (s *Server) putAsset(w http.ResponseWriter, r *http.Request) {
	a := s.ledgerAdmin(w, r)
	if a == nil {
		return
	}
	kind := r.PathValue("kind")
	b, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<16))
	if err != nil {
		fail(w, errors.New("bad request"))
		return
	}
	// the quota is read off the body before it is decoded, because only a
	// new record (no id) adds to the count
	var head struct {
		ID int64 `json:"id"`
	}
	json.Unmarshal(b, &head)
	if err := s.quotaRoom(a, kind, head.ID); err != nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "quota", "detail": err.Error()})
		return
	}
	var out any
	switch kind {
	case "site":
		var v assets.Site
		if err = json.Unmarshal(b, &v); err == nil {
			out, err = a.SaveSite(v)
		}
	case "rack":
		var v assets.Rack
		if err = json.Unmarshal(b, &v); err == nil {
			out, err = a.SaveRack(v)
		}
	case "device":
		var v assets.Device
		if err = json.Unmarshal(b, &v); err == nil {
			out, err = a.SaveDevice(v)
		}
	case "port":
		var v assets.Port
		if err = json.Unmarshal(b, &v); err == nil {
			out, err = a.SavePort(v)
		}
	case "link":
		var v assets.Link
		if err = json.Unmarshal(b, &v); err == nil {
			out, err = a.SaveLink(v)
		}
	case "circuit":
		var v assets.Circuit
		if err = json.Unmarshal(b, &v); err == nil {
			out, err = a.SaveCircuit(v)
		}
	case "subnet":
		var v assets.Subnet
		if err = json.Unmarshal(b, &v); err == nil {
			out, err = a.SaveSubnet(v)
		}
	case "addr":
		var v assets.Addr
		if err = json.Unmarshal(b, &v); err == nil {
			out, err = a.SaveAddr(v)
		}
	default:
		fail(w, fmt.Errorf("%q is not a kind of asset", kind))
		return
	}
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// deleteAsset removes one record.
func (s *Server) deleteAsset(w http.ResponseWriter, r *http.Request) {
	a := s.ledgerAdmin(w, r)
	if a == nil {
		return
	}
	id := idParam(r.URL.Query().Get("id"))
	if id == 0 {
		fail(w, errors.New("id missing"))
		return
	}
	var err error
	switch kind := r.PathValue("kind"); kind {
	case "site":
		err = a.DeleteSite(id)
	case "rack":
		err = a.DeleteRack(id)
	case "device":
		err = a.DeleteDevice(id)
	case "port":
		err = a.DeletePort(id)
	case "link":
		err = a.DeleteLink(id)
	case "circuit":
		err = a.DeleteCircuit(id)
	case "subnet":
		err = a.DeleteSubnet(id)
	case "addr":
		err = a.DeleteAddr(id)
	default:
		fail(w, fmt.Errorf("%q is not a kind of asset", kind))
		return
	}
	if err != nil {
		fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"ok": "deleted"})
}

// postPaste reads a pasted command output. Without "apply" nothing is
// written and the plan only says what it would do, so an operator sees
// what a paste of the wrong buffer would have done before it does it.
func (s *Server) postPaste(w http.ResponseWriter, r *http.Request) {
	a := s.ledgerAdmin(w, r)
	if a == nil {
		return
	}
	var in struct {
		Text     string `json:"text"`
		DeviceID int64  `json:"device_id"`
		Apply    bool   `json:"apply"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&in); err != nil {
		fail(w, errors.New("bad request"))
		return
	}
	plan, err := a.Paste(assets.PasteOptions{Text: in.Text, DeviceID: in.DeviceID})
	if err != nil {
		fail(w, err)
		return
	}
	if in.Apply {
		// A neighbour paste adds devices, so it is held to the same quota.
		// It asks for room for as many as the preview above found: asking
		// only whether the ledger was already full let a paste of fifty
		// through at forty-nine of fifty, and a paste that adds none is
		// still never refused.
		if err := s.quotaForPlan(a, plan); err != nil {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "quota", "detail": err.Error()})
			return
		}
		if plan, err = a.Paste(assets.PasteOptions{Text: in.Text, DeviceID: in.DeviceID, Apply: true}); err != nil {
			fail(w, err)
			return
		}
	}
	writeJSON(w, http.StatusOK, plan)
}

// getAssetCSV hands the ledger back as a spreadsheet, because an asset
// list that cannot leave the tool is a hostage.
//
// It is the administrator's, like the other whole-table operations. The
// pages filter hidden records one by one on the way out; a file of whole
// tables cannot be filtered that way without quietly handing someone a
// shorter ledger than the one they asked for, and a ledger that is quietly
// short is worse than one they cannot have.
func (s *Server) getAssetCSV(w http.ResponseWriter, r *http.Request) {
	a := s.ledgerAdmin(w, r)
	if a == nil {
		return
	}
	kind := r.URL.Query().Get("kind")
	var buf bytes.Buffer
	// An unregistered installation's file says so in leading comment lines;
	// the import below skips them, so the file still round-trips.
	buf.WriteString(commentLines(s.notice(), "# "))
	if err := a.ExportCSV(&buf, kind); err != nil {
		fail(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"traffic66-%s.csv\"", kind))
	w.Header().Set("Cache-Control", "no-store")
	w.Write(buf.Bytes())
}

// postAssetCSV reads a spreadsheet back in. Like a paste, it previews by
// default: "apply" is what writes.
func (s *Server) postAssetCSV(w http.ResponseWriter, r *http.Request) {
	a := s.ledgerAdmin(w, r)
	if a == nil {
		return
	}
	var in struct {
		Kind  string `json:"kind"`
		Text  string `json:"text"`
		Apply bool   `json:"apply"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<20)).Decode(&in); err != nil {
		fail(w, errors.New("bad request"))
		return
	}
	// Preview first, always: the preview is what the quota is decided on,
	// so an import that does not fit is refused before it has written
	// half of itself. This path had no quota check at all, which made a
	// spreadsheet the way round every number the licence sells.
	plan, err := a.ImportCSV(strings.NewReader(in.Text), in.Kind, false)
	if err != nil {
		fail(w, err)
		return
	}
	if in.Apply {
		if err := s.quotaForPlan(a, plan); err != nil {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "quota", "detail": err.Error()})
			return
		}
		if plan, err = a.ImportCSV(strings.NewReader(in.Text), in.Kind, true); err != nil {
			fail(w, err)
			return
		}
	}
	writeJSON(w, http.StatusOK, plan)
}

// assetQuota is how many of each kind this licence allows; 0 means no
// limit. It is reported so the page can say "41 of 50" before the save
// that would fail.
func (s *Server) assetQuota() map[string]int {
	if s.License == nil {
		return map[string]int{"sites": license.NotAvailable, "racks": license.NotAvailable, "devices": license.NotAvailable}
	}
	st := s.License.State()
	return map[string]int{"sites": st.SiteLimit(), "racks": st.RackLimit(), "devices": st.DeviceLimit()}
}

// quotaRoom refuses a new record that would take the ledger past what the
// licence allows. Editing an existing record (a non-zero id) is never
// refused: a quota that stopped an operator correcting a typo would be a
// trap, and the count does not change.
func (s *Server) quotaRoom(a *assets.Store, kind string, id int64) error {
	if id != 0 {
		return nil
	}
	return s.quotaFor(a, kind, 1)
}

// quotaFor refuses n new records of a kind when the ledger has not got
// room for all of them.
//
// Asking for the number rather than "are we at the limit yet" is the
// whole point: an import arrives as a batch, and a check that only
// refuses a ledger already at its limit lets a batch of fifty through at
// forty-nine of fifty. It is told what the batch would add and answers
// for the batch.
func (s *Server) quotaFor(a *assets.Store, kind string, n int) error {
	if n <= 0 {
		return nil
	}
	var limit, have int
	c := a.Counts()
	q := s.assetQuota()
	switch kind {
	case "site":
		limit, have = q["sites"], c.Sites
	case "rack":
		limit, have = q["racks"], c.Racks
	case "device":
		limit, have = q["devices"], c.Devices
	default:
		return nil // ports, links, circuits, subnets and addresses are not counted
	}
	if limit <= 0 || have+n <= limit { // 0: no limit
		return nil
	}
	if n == 1 {
		return fmt.Errorf("this licence records %d %ss, and the ledger holds %d", limit, kind, have)
	}
	return fmt.Errorf("this licence records %d %ss: the ledger holds %d and this would add %d", limit, kind, have, n)
}

// quotaForPlan refuses an import whose additions would not fit. The plan
// comes from a run that wrote nothing, so this is decided before anything
// is saved rather than discovered half way through.
func (s *Server) quotaForPlan(a *assets.Store, plan assets.Plan) error {
	adds := map[string]int{}
	for _, c := range plan.Changes {
		if c.Action == "add" {
			adds[c.Kind]++
		}
	}
	for _, kind := range []string{"site", "rack", "device"} {
		if err := s.quotaFor(a, kind, adds[kind]); err != nil {
			return err
		}
	}
	return nil
}

// visible drops the records an administrator has hidden.
func visible[T any](in []T, hidden func(T) bool) []T {
	out := make([]T, 0, len(in))
	for _, v := range in {
		if !hidden(v) {
			out = append(out, v)
		}
	}
	return out
}

// ownedBy keeps the ports of the devices that are left, so that hiding a
// device hides its ports with it.
func ownedBy(ports []assets.Port, devices []assets.Device) []assets.Port {
	seen := make(map[int64]bool, len(devices))
	for _, d := range devices {
		seen[d.ID] = true
	}
	return visible(ports, func(p assets.Port) bool { return !seen[p.DeviceID] })
}

// cabled keeps the links whose both ends are still visible.
func cabled(links []assets.Link, ports []assets.Port) []assets.Link {
	seen := make(map[int64]bool, len(ports))
	for _, p := range ports {
		seen[p.ID] = true
	}
	return visible(links, func(l assets.Link) bool { return !seen[l.APort] || !seen[l.BPort] })
}

// idParam reads a record identifier; anything unreadable is 0, which
// every query treats as "not given".
func idParam(s string) int64 {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil || n < 0 {
		return 0
	}
	return n
}
