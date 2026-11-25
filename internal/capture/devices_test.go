package capture

import (
	"strings"
	"testing"
)

func TestResolveDevice(t *testing.T) {
	ds := []Device{
		{Num: 1, Name: `\Device\NPF_{4B8A2C1E-1111-2222-3333-444455556666}`, Friendly: "Ethernet", Desc: "Intel(R) Ethernet I219-V"},
		{Num: 2, Name: `\Device\NPF_{9F00AA11-AAAA-BBBB-CCCC-DDDDEEEEFFFF}`, Friendly: "Wi-Fi", Desc: "Intel(R) Wi-Fi 6 AX201", Addrs: []string{"192.168.1.23"}},
		{Num: 3, Name: `\Device\NPF_Loopback`, Friendly: "Loopback", Desc: "Adapter for loopback traffic capture"},
	}
	wifi := ds[1].Name
	for _, in := range []string{"2", "Wi-Fi", "wi-fi", " Wi-Fi ", wifi, "{9F00AA11-AAAA-BBBB-CCCC-DDDDEEEEFFFF}", "9f00aa11-aaaa-bbbb-cccc-ddddeeeeffff"} {
		got, err := resolveDevice(in, ds)
		if err != nil || got != wifi {
			t.Errorf("%q: %q %v", in, got, err)
		}
	}
	if _, err := resolveDevice("WLAN 2", ds); err == nil || !strings.Contains(err.Error(), `2 "Wi-Fi"`) {
		t.Errorf("unknown name: %v", err)
	}
	lines := formatDevices(ds)
	if len(lines) != 4 || !strings.Contains(lines[2], "Wi-Fi") || !strings.Contains(lines[2], "192.168.1.23") {
		t.Errorf("%q", lines)
	}
	for _, l := range lines {
		t.Log(l)
	}
}
