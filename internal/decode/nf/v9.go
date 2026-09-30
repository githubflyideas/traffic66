package nf

import (
	"errors"
	"net/netip"
	"time"

	"github.com/githubflyideas/traffic66/internal/flow"
	"github.com/githubflyideas/traffic66/internal/wire"
)

// NetFlow v9 (RFC 3954): 20-byte header, then FlowSets.
func (d *Decoder) decodeV9(b []byte, addr netip.Addr, now time.Time) (Result, error) {
	r := wire.New(b)
	r.U16() // version
	count := r.U16()
	uptime := r.U32()
	secs := r.U32()
	seq := r.U32()
	srcID := r.U32()
	if r.Err() != nil {
		return Result{}, errors.New("nf: v9 header truncated")
	}
	h := msgHeader{src: flow.SrcNetFlow9, exportSecs: secs, uptime: uptime}

	d.mu.Lock()
	defer d.mu.Unlock()
	s := d.session(key{addr, srcID, flow.SrcNetFlow9})
	s.touch(now, time.Unix(int64(secs), 0))

	var res Result
	dataRecs := 0
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
		case setID == 0 || setID == 1:
			for _, id := range parseTemplateSet(s, body, flow.SrcNetFlow9, setID == 1) {
				d.replayHeld(s, id, addr, srcID, now, &res.Records)
			}
		case setID >= 256:
			t := s.templates[setID]
			if t == nil {
				s.holdSet(setID, body, h, now)
				continue
			}
			dataRecs += d.decodeData(s, t, body, h, addr, srcID, now, &res.Records)
		}
	}
	_ = count
	s.seq(seq, 1, uint32(dataRecs), now)
	return res, nil
}
