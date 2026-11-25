package api

import (
	"encoding/csv"
	"math"
	"net/http"
	"net/netip"
	"strconv"
	"strings"
	"time"
)

// exportRollups downloads the hourly interface counters as a spreadsheet
// file: one row per hour, exporter and interface, in and out in separate
// columns, with the average rates worked out so a report does not have
// to. The window is the one the page is showing (range or from/to, and
// the viewer's zone, whose offset is in the file's times). The counters
// are kept for the whole rollup retention, so this reaches back far past
// the flow detail. UTF-8 with a BOM, or Excel shows the interface names
// as mojibake.
// summarySpan is the widest window a query over the summaries may cover:
// the 180 days they are kept, and a few days over so a half-year export
// that starts on the first of a month is not refused.
const summarySpan = 186 * 24 * time.Hour

func (s *Server) exportRollups(w http.ResponseWriter, r *http.Request) {
	q, err := parseQuerySpan(r, summarySpan)
	if err != nil {
		fail(w, err)
		return
	}
	rows, err := s.Store.IfaceHours(q)
	if err != nil {
		fail(w, err)
		return
	}
	zone := time.FixedZone("", int(q.ZoneOffset.Seconds()))
	name := "traffic66-rollup-" + q.From.In(zone).Format("20060102") + "_" + q.To.In(zone).Format("20060102") + ".csv"
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	w.Write([]byte{0xEF, 0xBB, 0xBF})
	cw := csv.NewWriter(w)
	cw.Write([]string{"bucket_start", "bucket_end", "exporter", "interface",
		"bytes_in", "bytes_out", "packets_in", "packets_out", "avg_in_bps", "avg_out_bps"})
	avg := func(b uint64) string { return strconv.FormatUint(uint64(math.Round(float64(b)*8/3600)), 10) }
	for _, row := range rows {
		iface := strconv.FormatUint(uint64(row.IfIndex), 10)
		if a, err := netip.ParseAddr(row.Exporter); err == nil {
			if n := s.Inv.Iface(a, row.IfIndex).Name; n != "" {
				iface = n
			}
		}
		start := row.Start.In(zone)
		cw.Write([]string{
			start.Format(time.RFC3339), start.Add(time.Hour).Format(time.RFC3339),
			csvCell(row.Exporter), csvCell(iface),
			strconv.FormatUint(row.InOct, 10), strconv.FormatUint(row.OutOct, 10),
			strconv.FormatUint(row.InPkts, 10), strconv.FormatUint(row.OutPkts, 10),
			avg(row.InOct), avg(row.OutOct),
		})
	}
	cw.Flush()
}

// csvCell keeps a text field from being read as a formula when the file
// is opened in a spreadsheet: interface names come from the devices
// (SNMP ifName) and the inventory, and a name such as "=HYPERLINK(...)"
// would otherwise run as one. Such a cell is written with a leading
// apostrophe, which spreadsheets show as plain text.
func csvCell(v string) string {
	if v != "" && strings.ContainsRune("=+-@\t\r", rune(v[0])) {
		return "'" + v
	}
	return v
}
