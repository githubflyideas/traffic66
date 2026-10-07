// Package nf decodes NetFlow v5, NetFlow v9 (RFC 3954) and IPFIX (RFC 7011).
//
// A Decoder keeps per-exporter template and sampling state. Records whose
// sampling rate is not yet known are held until the exporter declares it or
// the hold time runs out.
package nf

import (
	"errors"
	"net/netip"
	"sort"
	"sync"
	"time"

	"github.com/githubflyideas/traffic66/internal/flow"
)

// Stats are cumulative counters for one exporter session.
type Stats struct {
	Source        flow.Source
	Exporter      netip.Addr
	Domain        uint32
	Packets       uint64
	Records       uint64
	LostRecords   uint64 // estimated from sequence numbers
	LostPackets   uint64
	NoTemplate    uint64   // data sets that arrived before their template (held)
	DroppedSets   uint64   // held data sets discarded because the template never came
	Pending       int      // records waiting for a sampling rate
	SamplingState string   // "known", "waiting", "assumed1"
	Sampling      uint32   // exporter-wide rate if known
	Rates         []uint32 // every declared rate (exporter, sampler, interface)
	Templates     int
	TemplateInfo  []string // the templates seen and their field ids
	// DirField: "yes" when the data templates carry flowDirection (61), so
	// the interface each flow was sampled on is known; "no" when they do
	// not; "" before any data template.
	DirField string
	Manual        bool     // the rate was set by hand (sampling= in the names)
	// LossComp is the multiplier currently applied to make up for records
	// lost in transit (1 = no loss).
	LossComp  float64
	LastSeen  time.Time
	ClockSkew time.Duration
}

type key struct {
	addr   netip.Addr
	domain uint32
	src    flow.Source
}

type fieldSpec struct {
	id    uint16
	ent   uint32
	len   uint16 // 0xFFFF = variable (IPFIX)
	scope bool
}

type template struct {
	fields  []fieldSpec
	options bool
	// NetFlow v9 option scope fields use their own type space.
	nf9Scope bool
}

type heldSet struct {
	body []byte
	hdr  msgHeader
	at   time.Time
}

type pendingRec struct {
	rec flow.Record
	ref samplerRef
	at  time.Time
}

type samplerRef struct {
	samplerID  uint32
	hasSampler bool
	inIf       uint32
	outIf      uint32
}

type session struct {
	stats     Stats
	templates map[uint16]*template
	held      map[uint16][]heldSet
	heldBytes int

	// sampling tables
	bySampler map[uint32]uint32
	byIf      map[uint32]uint32
	def       uint32
	assumed   bool

	pending []pendingRec

	tmplDesc map[uint16]string // templates seen, for the sources page
	tmplDir  map[uint16]bool   // data templates: whether they carry flowDirection

	initMs int64

	seqInit    bool
	nextSeq    uint32
	lastRecs   uint32
	recsPerPkt float64

	// loss compensation window
	winRecv, winLost float64
	winAt            time.Time
	comp             float64
}

func newSession(k key) *session {
	return &session{
		stats:     Stats{Source: k.src, Exporter: k.addr, Domain: k.domain, LossComp: 1},
		comp:      1,
		templates: map[uint16]*template{},
		held:      map[uint16][]heldSet{},
		bySampler: map[uint32]uint32{},
		byIf:      map[uint32]uint32{},
	}
}

// Decoder holds state for all NetFlow/IPFIX exporters.
type Decoder struct {
	mu       sync.Mutex
	sessions map[key]*session
	// HoldFor is how long records wait for a sampling rate.
	HoldFor time.Duration
	// MaxPending caps held records per session.
	MaxPending int
	// MaxSessions caps the number of exporter sessions, so that datagrams
	// from many (spoofed) source addresses cannot grow memory without bound.
	MaxSessions int
	// Unsampled lists exporters known to export every packet.
	Unsampled map[netip.Addr]bool
	// Manual is the sampling rate of exporters that do not declare it,
	// from sampling= in the names; a declared rate takes precedence.
	Manual map[netip.Addr]uint32
	// byAddr is the last rate an exporter declared, in any domain.
	byAddr map[netip.Addr]uint32
}

func NewDecoder() *Decoder {
	return &Decoder{
		sessions:    map[key]*session{},
		HoldFor:     5 * time.Minute,
		MaxPending:  200000,
		MaxSessions: 10000,
		Unsampled:   map[netip.Addr]bool{},
		Manual:      map[netip.Addr]uint32{},
		byAddr:      map[netip.Addr]uint32{},
	}
}

var ErrUnknownVersion = errors.New("nf: unknown version")

// ErrTooManyExporters is returned when MaxSessions exporters are tracked
// already and a datagram arrives from a new one.
var ErrTooManyExporters = errors.New("nf: too many exporters")

// sessionIdle is how long a silent exporter's session is kept.
const sessionIdle = 24 * time.Hour

// Result is the output of one Decode or Flush call.
type Result struct {
	Records []flow.Record
}

// Decode parses one UDP payload from exporter addr.
func (d *Decoder) Decode(b []byte, addr netip.Addr, now time.Time) (Result, error) {
	if len(b) < 2 {
		return Result{}, ErrUnknownVersion
	}
	switch uint16(b[0])<<8 | uint16(b[1]) {
	case 5:
		return d.decodeV5(b, addr, now)
	case 9:
		return d.decodeV9(b, addr, now)
	case 10:
		return d.decodeIPFIX(b, addr, now)
	}
	return Result{}, ErrUnknownVersion
}

func (d *Decoder) session(k key) *session {
	s := d.sessions[k]
	if s == nil {
		if d.MaxSessions > 0 && len(d.sessions) >= d.MaxSessions {
			return nil
		}
		s = newSession(k)
		d.sessions[k] = s
	}
	return s
}

// Stats returns a snapshot of every exporter session.
func (d *Decoder) Stats() []Stats {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := make([]Stats, 0, len(d.sessions))
	for _, s := range d.sessions {
		st := s.stats
		st.Pending = len(s.pending)
		st.Templates = len(s.templates)
		for _, v := range s.tmplDesc {
			st.TemplateInfo = append(st.TemplateInfo, v)
		}
		sort.Strings(st.TemplateInfo)
		for _, d := range s.tmplDir {
			if d {
				st.DirField = "yes"
				break
			}
			st.DirField = "no"
		}
		if s.def == 0 && len(s.bySampler) == 0 && len(s.byIf) == 0 && d.Manual[s.stats.Exporter] > 0 {
			st.Manual = true
			st.Sampling = d.Manual[s.stats.Exporter]
		}
		if !st.Manual {
			st.Sampling = s.def
		}
		st.LossComp = s.comp
		seen := map[uint32]bool{}
		add := func(r uint32) {
			if r > 0 && !seen[r] {
				seen[r] = true
				st.Rates = append(st.Rates, r)
			}
		}
		add(s.def)
		if st.Manual {
			add(st.Sampling)
		}
		for _, r := range s.bySampler {
			add(r)
		}
		for _, r := range s.byIf {
			add(r)
		}
		sort.Slice(st.Rates, func(i, j int) bool { return st.Rates[i] < st.Rates[j] })
		switch {
		case st.Manual:
			st.SamplingState = "manual"
		case s.assumed:
			st.SamplingState = "assumed1"
		case len(s.pending) > 0:
			st.SamplingState = "waiting"
		default:
			st.SamplingState = "known"
		}
		out = append(out, st)
	}
	return out
}

// Flush releases held records older than HoldFor, drops held data sets
// whose template never arrived, and forgets exporters silent for a day.
// After the first timeout an exporter that never declared a rate is treated
// as unsampled.
func (d *Decoder) Flush(now time.Time) Result {
	d.mu.Lock()
	defer d.mu.Unlock()
	var res Result
	for k, s := range d.sessions {
		// Held data sets whose template never arrived are dropped.
		for id, hs := range s.held {
			keep := hs[:0]
			for _, h := range hs {
				if now.Sub(h.at) < d.HoldFor {
					keep = append(keep, h)
				} else {
					s.stats.DroppedSets++
					s.heldBytes -= len(h.body)
				}
			}
			if len(keep) == 0 {
				delete(s.held, id)
			} else {
				s.held[id] = keep
			}
		}
		if len(s.pending) > 0 && now.Sub(s.pending[0].at) >= d.HoldFor {
			s.assumed = true
			for _, p := range s.pending {
				r := p.rec
				r.Sampling = 1
				r.Mult = s.comp
				r.SamplingKnown = false
				res.Records = append(res.Records, r)
			}
			s.pending = s.pending[:0]
		}
		if len(s.pending) == 0 && len(s.held) == 0 && now.Sub(s.stats.LastSeen) > sessionIdle {
			delete(d.sessions, k)
		}
	}
	return res
}

func (d *Decoder) rateFor(s *session, rec *flow.Record, ref samplerRef, inRecordRate uint32) uint32 {
	rate := inRecordRate
	if rate == 0 && ref.hasSampler {
		rate = s.bySampler[ref.samplerID]
	}
	if rate == 0 {
		rate = s.byIf[ref.inIf]
	}
	if rate == 0 && ref.outIf != 0 {
		// sampling configured on the egress interface
		rate = s.byIf[ref.outIf]
	}
	if rate == 0 {
		rate = s.def
	}
	if rate == 0 {
		// one rate declared for a sampler or interface the record does not
		// name: it is the only one there is
		rate = s.onlyRate()
	}
	if rate == 0 {
		// declared by the same device in another observation domain (line
		// cards export under their own ids)
		rate = d.byAddr[rec.Exporter]
	}
	if rate == 0 {
		rate = d.Manual[rec.Exporter]
	}
	return rate
}

// onlyRate is the session's rate when all its declared rates agree.
func (s *session) onlyRate() uint32 {
	var r uint32
	for _, m := range []map[uint32]uint32{s.bySampler, s.byIf} {
		for _, x := range m {
			if r != 0 && x != r {
				return 0
			}
			r = x
		}
	}
	return r
}

// resolve fills in the sampling multiplier or queues the record.
func (d *Decoder) resolve(s *session, rec flow.Record, ref samplerRef, inRecordRate uint32, now time.Time, out *[]flow.Record) {
	rate := d.rateFor(s, &rec, ref, inRecordRate)
	if rate == 0 && (s.assumed || d.Unsampled[rec.Exporter]) {
		rec.Sampling, rec.Mult, rec.SamplingKnown = 1, s.comp, !s.assumed
		*out = append(*out, rec)
		return
	}
	if rate == 0 {
		if len(s.pending) >= d.MaxPending {
			// Oldest record leaves unscaled rather than silently vanishing.
			old := s.pending[0]
			old.rec.Sampling, old.rec.Mult = 1, 1
			*out = append(*out, old.rec)
			s.pending = s.pending[1:]
		}
		s.pending = append(s.pending, pendingRec{rec: rec, ref: ref, at: now})
		return
	}
	rec.Sampling = rate
	rec.Mult = float64(rate) * s.comp
	rec.SamplingKnown = true
	*out = append(*out, rec)
}

// retryPending re-resolves held records after new sampling info arrives.
func (d *Decoder) retryPending(s *session, out *[]flow.Record) {
	keep := s.pending[:0]
	for _, p := range s.pending {
		rate := d.rateFor(s, &p.rec, p.ref, 0)
		if rate == 0 {
			keep = append(keep, p)
			continue
		}
		p.rec.Sampling, p.rec.Mult, p.rec.SamplingKnown = rate, float64(rate)*s.comp, true
		*out = append(*out, p.rec)
	}
	s.pending = keep
}

// seq tracks sequence continuity. unit is how many sequence steps this
// message consumed (1 for v9 packets, record count for v5 and IPFIX).
// Gaps are counted as lost; a message that arrives late after a gap was
// counted takes its share back out, so reordering is not reported as loss.
func (s *session) seq(seq uint32, unit uint32, recs uint32, now time.Time) {
	var lostRecs float64
	if s.seqInit {
		diff := int32(seq - s.nextSeq)
		switch {
		case diff > 0 && diff < 1<<20:
			gap := uint64(diff)
			if s.stats.Source == flow.SrcNetFlow9 {
				s.stats.LostPackets += gap
				lostRecs = float64(gap) * s.recsPerPkt
			} else {
				lostRecs = float64(gap)
				if s.recsPerPkt > 0 {
					s.stats.LostPackets += uint64(float64(gap) / s.recsPerPkt)
				}
			}
			s.stats.LostRecords += uint64(lostRecs)
		case diff < 0 && -diff < 1<<20:
			// late arrival: it was counted as lost when the gap opened
			back := float64(recs)
			if s.stats.Source == flow.SrcNetFlow9 && s.stats.LostPackets > 0 {
				s.stats.LostPackets--
			}
			if back > float64(s.stats.LostRecords) {
				back = float64(s.stats.LostRecords)
			}
			s.stats.LostRecords -= uint64(back)
			lostRecs = -back
		}
	}
	if !s.seqInit || int32(seq-s.nextSeq) >= 0 {
		s.nextSeq = seq + unit
	}
	s.seqInit = true
	if recs > 0 {
		if s.recsPerPkt == 0 {
			s.recsPerPkt = float64(recs)
		} else {
			s.recsPerPkt = s.recsPerPkt*0.95 + float64(recs)*0.05
		}
	}
	s.updateComp(float64(recs), lostRecs, now)
}

// updateComp keeps a decaying window of received and lost records and
// derives the multiplier that restores the lost share of the totals.
func (s *session) updateComp(recv, lost float64, now time.Time) {
	s.winRecv += recv
	s.winLost += lost
	if s.winLost < 0 {
		s.winLost = 0
	}
	if s.winAt.IsZero() {
		s.winAt = now
	}
	if now.Sub(s.winAt) < 30*time.Second {
		return
	}
	if s.winRecv >= 100 {
		c := (s.winRecv + s.winLost) / s.winRecv
		if c > 2 { // more than half missing: do not guess
			c = 2
		}
		if c < 1.005 { // below 0.5% loss (or a reorder straddling windows) do nothing
			c = 1
		}
		s.comp = c
	}
	s.winRecv *= 0.5
	s.winLost *= 0.5
	s.winAt = now
}

func (s *session) touch(now time.Time, exportTime time.Time) {
	s.stats.Packets++
	s.stats.LastSeen = now
	if !exportTime.IsZero() {
		s.stats.ClockSkew = exportTime.Sub(now)
	}
}

// SetUnsampled replaces the list of exporters known to export every packet.
func (d *Decoder) SetUnsampled(m map[netip.Addr]bool) {
	d.mu.Lock()
	d.Unsampled = m
	d.mu.Unlock()
}

// SetManual replaces the hand-set sampling rates; records waiting for a
// rate are released with them.
func (d *Decoder) SetManual(m map[netip.Addr]uint32) Result {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.Manual = m
	var res Result
	for _, s := range d.sessions {
		d.retryPending(s, &res.Records)
	}
	return res
}
