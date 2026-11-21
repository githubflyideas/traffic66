package api

import (
	"encoding/csv"
	"math"
	"net/http"
	"net/netip"
	"strconv"
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
func (s *Server) exportRollups(w http.ResponseWriter, r *http.Request) {
	q, err := parseQuery(r)
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
			row.Exporter, iface,
			strconv.FormatUint(row.InOct, 10), strconv.FormatUint(row.OutOct, 10),
			strconv.FormatUint(row.InPkts, 10), strconv.FormatUint(row.OutPkts, 10),
			avg(row.InOct), avg(row.OutOct),
		})
	}
	cw.Flush()
}
