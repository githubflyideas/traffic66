package api

import (
	"fmt"
	"net/http"
	"runtime"
	"sort"
	"strings"
	"time"
)

// metrics serves counters in the Prometheus text format, for monitoring
// traffic66 itself (not the traffic it counts). It needs a sign-in like
// the rest of the API: a scraper uses HTTP Basic auth with a traffic66
// user (Prometheus: basic_auth in the scrape config).
func (s *Server) metrics(w http.ResponseWriter, r *http.Request) {
	var b strings.Builder
	m := func(name, typ, help string, v float64, labels ...string) {
		if !strings.Contains(b.String(), "# TYPE "+name+" ") {
			fmt.Fprintf(&b, "# HELP %s %s\n# TYPE %s %s\n", name, help, name, typ)
		}
		l := ""
		if len(labels) > 0 {
			var kv []string
			for i := 0; i+1 < len(labels); i += 2 {
				kv = append(kv, fmt.Sprintf("%s=%q", labels[i], labels[i+1]))
			}
			l = "{" + strings.Join(kv, ",") + "}"
		}
		fmt.Fprintf(&b, "%s%s %v\n", name, l, v)
	}
	m("traffic66_build_info", "gauge", "Version of traffic66 and of Go it was built with.", 1, "version", s.Version, "build", s.BuildKind, "go", runtime.Version())
	m("traffic66_uptime_seconds", "gauge", "Seconds since traffic66 started.", time.Since(s.Started).Seconds())
	if p := s.Pipe; p != nil {
		m("traffic66_records_total", "counter", "Flow records received and processed.", float64(p.Records.Load()))
		m("traffic66_records_refused_total", "counter", "Records refused for an impossible sampling rate, size or time.", float64(p.Implausible.Load()))
		m("traffic66_records_dropped_total", "counter", "Records dropped because processing fell behind.", float64(p.Dropped.Load()))
		m("traffic66_rows_written_total", "counter", "Rows written to storage (a record has a row per minute it spans).", float64(p.Rows.Load()))
		m("traffic66_duplicate_rows_total", "counter", "Rows reported by more than one device, counted once.", float64(p.DupRows.Load()))
		m("traffic66_write_errors_total", "counter", "Failed writes to storage.", float64(p.WriteErrs.Load()))
	}
	if c := s.Col; c != nil {
		m("traffic66_records_time_corrected_total", "counter", "Records whose time was implausible and was set to the time received.", float64(c.Skewed.Load()))
		ls := append(c.Listeners[:0:0], c.Listeners...)
		sort.Slice(ls, func(i, j int) bool { return ls[i].Addr < ls[j].Addr })
		for _, l := range ls {
			m("traffic66_udp_datagrams_total", "counter", "Datagrams received per collector port.", float64(l.Packets.Load()), "listener", l.Addr)
		}
		for _, l := range ls {
			m("traffic66_udp_undecoded_total", "counter", "Datagrams that could not be decoded per collector port.", float64(l.Undecoded.Load()), "listener", l.Addr)
		}
		for _, l := range ls {
			m("traffic66_udp_blocked_total", "counter", "Datagrams dropped from a blocked source per collector port.", float64(l.Dropped.Load()), "listener", l.Addr)
		}
		m("traffic66_sources", "gauge", "Exporting devices (and observation domains) seen.", float64(len(c.Sources())))
	}
	if st := s.Store; st != nil {
		u := st.Usage()
		m("traffic66_storage_bytes", "gauge", "Bytes the data directory takes on disk.", float64(u.DiskBytes))
		m("traffic66_storage_free_bytes", "gauge", "Free bytes on the disk of the data directory.", float64(diskFree(s.DataDir)))
		m("traffic66_storage_recent_rows", "gauge", "Rows not yet sealed into a file.", float64(u.HotRows))
		m("traffic66_storage_files", "gauge", "Sealed flow files.", float64(u.Segments))
		if !u.Oldest.IsZero() {
			m("traffic66_storage_oldest_timestamp_seconds", "gauge", "Time of the oldest flow detail kept.", float64(u.Oldest.Unix()))
		}
	}
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	w.Write([]byte(b.String()))
}
