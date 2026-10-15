package assets

import (
	"strings"
	"testing"
)

// A ledger exported, edited and imported again updates what it came from
// rather than duplicating it: the id column ties a row to its record.
func TestCSVRoundTrip(t *testing.T) {
	s := open(t)
	site, _ := s.SaveSite(Site{Name: "Tokyo DC", Address: "Koto-ku"})
	rack, _ := s.SaveRack(Rack{SiteID: site.ID, Name: "A-01", Units: 42})
	if _, err := s.SaveDevice(Device{RackID: rack.ID, Name: "core-sw1", PosU: 5, HeightU: 2,
		Vendor: "Generic", Model: "GS-48", MgmtIP: "10.0.0.1", Tags: []string{"core", "prod"}}); err != nil {
		t.Fatal(err)
	}
	var out strings.Builder
	if err := s.ExportCSV(&out, "device"); err != nil {
		t.Fatal(err)
	}
	text := out.String()
	for _, want := range []string{"core-sw1", "A-01", "10.0.0.1", "core prod", "Generic"} {
		if !strings.Contains(text, want) {
			t.Fatalf("the export does not mention %q:\n%s", want, text)
		}
	}
	// edit a field and put it back
	edited := strings.Replace(text, "Generic", "Acme", 1)
	p, err := s.ImportCSV(strings.NewReader(edited), "device", true)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Changes) != 1 || p.Changes[0].Action != "update" || p.Changes[0].Error != "" {
		t.Fatalf("import: %+v", p.Changes)
	}
	ds, _ := s.Devices(DeviceQuery{})
	if len(ds) != 1 {
		t.Fatalf("the import duplicated the device: %+v", ds)
	}
	if ds[0].Vendor != "Acme" || ds[0].RackID != rack.ID || ds[0].PosU != 5 {
		t.Fatalf("after the import: %+v", ds[0])
	}
}

// A spreadsheet somebody already keeps: no ids, references by name, one
// bad line among good ones.
func TestCSVImportFromScratch(t *testing.T) {
	s := open(t)
	site, _ := s.SaveSite(Site{Name: "Tokyo DC"})
	if _, err := s.SaveRack(Rack{SiteID: site.ID, Name: "A-01", Units: 42}); err != nil {
		t.Fatal(err)
	}
	text := "name,rack,pos_u,height_u,model,tags\n" +
		"core-sw1,A-01,5,2,GS-48,core prod\n" +
		"edge-sw1,Nowhere,1,1,GS-24,edge\n" + // no such rack
		"spare-sw,,0,1,GS-24,\n" // not racked at all, which is fine
	p, err := s.ImportCSV(strings.NewReader(text), "device", true)
	if err != nil {
		t.Fatal(err)
	}
	var errs int
	for _, c := range p.Changes {
		if c.Error != "" {
			errs++
			if !strings.Contains(c.Error, "line 3") {
				t.Fatalf("the failing line is not named: %q", c.Error)
			}
		}
	}
	if errs != 1 {
		t.Fatalf("errors %d: %+v", errs, p.Changes)
	}
	ds, _ := s.Devices(DeviceQuery{})
	if len(ds) != 2 { // the good lines landed; the bad one did not
		t.Fatalf("devices: %+v", ds)
	}
}

// A preview writes nothing.
func TestCSVPreview(t *testing.T) {
	s := open(t)
	p, err := s.ImportCSV(strings.NewReader("name,address\nTokyo DC,Koto-ku\n"), "site", false)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Changes) != 1 || p.Changes[0].Action != "add" {
		t.Fatalf("%+v", p.Changes)
	}
	if ss, _ := s.Sites(); len(ss) != 0 {
		t.Fatalf("a preview wrote %d sites", len(ss))
	}
}

// A file that is not this kind of export says so instead of filling the
// ledger with nonsense.
func TestCSVRefuses(t *testing.T) {
	s := open(t)
	if _, err := s.ImportCSV(strings.NewReader("a,b\n1,2\n"), "device", true); err == nil {
		t.Fatal("a file with no name column was accepted")
	}
	if _, err := s.ImportCSV(strings.NewReader(""), "device", true); err == nil {
		t.Fatal("an empty file was accepted")
	}
	if _, err := s.ImportCSV(strings.NewReader("name\nx\n"), "nonsense", true); err == nil {
		t.Fatal("an unknown kind was accepted")
	}
}

// Every kind exports with a header, so the pages can offer them all and
// an operator can see the columns before filling one in.
func TestCSVEveryKind(t *testing.T) {
	s := open(t)
	for _, k := range CSVKinds {
		var out strings.Builder
		if err := s.ExportCSV(&out, k); err != nil {
			t.Fatalf("%s: %v", k, err)
		}
		head := strings.SplitN(out.String(), "\n", 2)[0]
		if !strings.HasPrefix(head, "id,") {
			t.Fatalf("%s header %q", k, head)
		}
	}
}
