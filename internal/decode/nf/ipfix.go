package nf

import (
	"errors"
	"net/netip"
	"time"

	"github.com/githubflyideas/traffic66/internal/flow"
	"github.com/githubflyideas/traffic66/internal/wire"
)

// IPFIX (RFC 7011): 16-byte message header, then Sets.
func (d *Decoder) decodeIPFIX(b []byte, addr netip.Addr, now time.Time) (Result, error) {
	r := wire.New(b)
	r.U16() // version
	length := r.U16()
	exportTime := r.U32()
	seq := r.U32()
	domain := r.U32()
	if r.Err() != nil {
		return Result{}, errors.New("nf: ipfix header truncated")
	}
	if int(length) < 16 || int(length) > len(b) {
		return Result{}, errors.New("nf: ipfix length mismatch")
	}
	r = wire.New(b[16:length])
	h := msgHeader{src: flow.SrcIPFIX, exportSecs: exportTime}

	d.mu.Lock()
	defer d.mu.Unlock()
	s := d.session(key{addr, domain, flow.SrcIPFIX})
	if s == nil {
		return Result{}, ErrTooManyExporters
	}
	s.touch(now, time.Unix(int64(exportTime), 0))

	var res Result
	dataRecs := 0
	held := false
	for r.Len() >= 4 {
		setID := r.U16()
		setLen := r.U16()
		if setLen < 4 {
			break
		}
		body := r.Bytes(int(setLen) - 4)
		if r.Err() != nil {
			break
		}
		switch {
		case setID == 2 || setID == 3:
			for _, id := range parseTemplateSet(s, body, flow.SrcIPFIX, setID == 3) {
				d.replayHeld(s, id, addr, domain, now, &res.Records)
			}
		case setID >= 256:
			t := s.templates[setID]
			if t == nil {
				s.holdSet(setID, body, h, now)
				held = true
				continue
			}
			dataRecs += d.decodeData(s, t, body, h, addr, domain, now, &res.Records)
		}
	}
	// IPFIX sequence numbers count data records (including options data).
	s.seq(seq, uint32(dataRecs), uint32(dataRecs), now)
	if held {
		// Records in held sets are uncounted, so the next gap is unknowable.
		s.seqInit = false
	}
	return res, nil
}
