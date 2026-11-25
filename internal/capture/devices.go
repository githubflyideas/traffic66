package capture

import (
	"fmt"
	"strconv"
	"strings"
)

// Device is a capture device as offered to the user. On Windows, Name is
// Npcap's device name (\Device\NPF_{GUID}) and Friendly the connection name
// shown in Windows' network settings ("Wi-Fi", "Ethernet").
type Device struct {
	Num      int
	Name     string
	Friendly string
	Desc     string
	Addrs    []string
}

// formatDevices renders the list printed by "traffic66 interfaces".
func formatDevices(ds []Device) []string {
	w := 4
	for _, d := range ds {
		if len(d.Friendly) > w {
			w = len(d.Friendly)
		}
	}
	out := []string{fmt.Sprintf("%-3s %-*s  %-15s  %s", "#", w, "Name", "Address", "Adapter / device")}
	for _, d := range ds {
		addr := "-"
		if len(d.Addrs) > 0 {
			addr = d.Addrs[0]
		}
		desc := d.Desc
		if desc == "" {
			desc = d.Name
		} else if d.Name != d.Friendly {
			desc += "  " + d.Name
		}
		out = append(out, fmt.Sprintf("%-3d %-*s  %-15s  %s", d.Num, w, d.Friendly, addr, desc))
	}
	return out
}

// resolveDevice turns what the user typed into a device name: the number
// from the list, the connection name ("Wi-Fi", any case), the device name
// itself, or its GUID.
func resolveDevice(in string, ds []Device) (string, error) {
	in = strings.TrimSpace(in)
	if n, err := strconv.Atoi(in); err == nil {
		for _, d := range ds {
			if d.Num == n {
				return d.Name, nil
			}
		}
	}
	for _, d := range ds {
		if d.Name == in || strings.EqualFold(d.Friendly, in) {
			return d.Name, nil
		}
	}
	g := strings.ToUpper(strings.Trim(in, "{}"))
	for _, d := range ds {
		if g != "" && strings.Contains(strings.ToUpper(d.Name), g) {
			return d.Name, nil
		}
	}
	var names []string
	for _, d := range ds {
		names = append(names, fmt.Sprintf("%d %q", d.Num, d.Friendly))
	}
	return "", fmt.Errorf("no interface %q; use a number or name from 'traffic66 interfaces' (%s)", in, strings.Join(names, ", "))
}
