// Package snmp is a minimal SNMPv2c implementation: enough to walk the
// interface counter columns of IF-MIB (RFC 2863) from exporters, and a
// small agent used by the demo and tests. Encoding follows RFC 3416 and
// X.690 BER.
package snmp

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

const (
	tagInteger  = 0x02
	tagOctets   = 0x04
	tagNull     = 0x05
	tagOID      = 0x06
	tagSequence = 0x30
	tagIPAddr   = 0x40
	tagCounter  = 0x41
	tagGauge    = 0x42
	tagTicks    = 0x43
	tagCounter6 = 0x46

	tagNoSuchObject   = 0x80
	tagNoSuchInstance = 0x81
	tagEndOfMib       = 0x82

	pduGet      = 0xA0
	pduGetNext  = 0xA1
	pduResponse = 0xA2
	pduGetBulk  = 0xA5
)

const version2c = 1

// OID is an object identifier.
type OID []uint32

// ParseOID reads dotted notation.
func ParseOID(s string) (OID, error) {
	s = strings.TrimPrefix(s, ".")
	parts := strings.Split(s, ".")
	o := make(OID, 0, len(parts))
	for _, p := range parts {
		v, err := strconv.ParseUint(p, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("snmp: bad OID %q", s)
		}
		o = append(o, uint32(v))
	}
	if len(o) < 2 {
		return nil, fmt.Errorf("snmp: OID %q too short", s)
	}
	return o, nil
}

// MustOID panics on a bad OID; for constants.
func MustOID(s string) OID {
	o, err := ParseOID(s)
	if err != nil {
		panic(err)
	}
	return o
}

func (o OID) String() string {
	var b strings.Builder
	for i, v := range o {
		if i > 0 {
			b.WriteByte('.')
		}
		b.WriteString(strconv.FormatUint(uint64(v), 10))
	}
	return b.String()
}

// HasPrefix reports whether p is an ancestor of (or equal to) o.
func (o OID) HasPrefix(p OID) bool {
	if len(o) < len(p) {
		return false
	}
	for i := range p {
		if o[i] != p[i] {
			return false
		}
	}
	return true
}

// Compare orders OIDs lexicographically.
func (o OID) Compare(p OID) int {
	for i := 0; i < len(o) && i < len(p); i++ {
		if o[i] != p[i] {
			if o[i] < p[i] {
				return -1
			}
			return 1
		}
	}
	switch {
	case len(o) < len(p):
		return -1
	case len(o) > len(p):
		return 1
	}
	return 0
}

// Kind is the type of a variable binding value.
type Kind uint8

const (
	KindNull Kind = iota
	KindInt
	KindString
	KindOID
	KindCounter32
	KindGauge32
	KindTicks
	KindCounter64
	KindIPAddr
	KindNoSuchObject
	KindNoSuchInstance
	KindEndOfMib
)

// Value is a decoded variable binding value.
type Value struct {
	Kind Kind
	Int  int64
	Uint uint64
	Str  []byte
	OID  OID
}

// VarBind is one name/value pair.
type VarBind struct {
	Name  OID
	Value Value
}

// ------------------------------------------------------------------ encode

func appendLen(b []byte, n int) []byte {
	switch {
	case n < 0x80:
		return append(b, byte(n))
	case n < 0x100:
		return append(b, 0x81, byte(n))
	case n < 0x10000:
		return append(b, 0x82, byte(n>>8), byte(n))
	default:
		return append(b, 0x83, byte(n>>16), byte(n>>8), byte(n))
	}
}

func tlv(tag byte, body []byte) []byte {
	b := make([]byte, 0, len(body)+4)
	b = append(b, tag)
	b = appendLen(b, len(body))
	return append(b, body...)
}

// encInt returns the shortest two's-complement encoding of v.
func encInt(v int64) []byte {
	n := 8
	for n > 1 {
		hi, next := byte(v>>(8*(n-1))), byte(v>>(8*(n-2)))
		if (hi == 0 && next&0x80 == 0) || (hi == 0xff && next&0x80 != 0) {
			n--
			continue
		}
		break
	}
	out := make([]byte, n)
	for i := 0; i < n; i++ {
		out[i] = byte(v >> (8 * (n - 1 - i)))
	}
	return out
}

func encUint(v uint64) []byte {
	var body []byte
	for {
		body = append([]byte{byte(v)}, body...)
		v >>= 8
		if v == 0 {
			break
		}
	}
	if body[0]&0x80 != 0 {
		body = append([]byte{0}, body...)
	}
	return body
}

func encOID(o OID) ([]byte, error) {
	if len(o) < 2 || o[0] > 2 || (o[0] < 2 && o[1] >= 40) {
		return nil, fmt.Errorf("snmp: cannot encode OID %s", o)
	}
	body := base128(nil, o[0]*40+o[1])
	for _, v := range o[2:] {
		body = base128(body, v)
	}
	return body, nil
}

func base128(b []byte, v uint32) []byte {
	var tmp [5]byte
	i := len(tmp) - 1
	tmp[i] = byte(v & 0x7f)
	for v >>= 7; v > 0; v >>= 7 {
		i--
		tmp[i] = byte(v&0x7f) | 0x80
	}
	return append(b, tmp[i:]...)
}

func encValue(v Value) ([]byte, error) {
	switch v.Kind {
	case KindNull:
		return tlv(tagNull, nil), nil
	case KindInt:
		return tlv(tagInteger, encInt(v.Int)), nil
	case KindString:
		return tlv(tagOctets, v.Str), nil
	case KindOID:
		b, err := encOID(v.OID)
		if err != nil {
			return nil, err
		}
		return tlv(tagOID, b), nil
	case KindCounter32:
		return tlv(tagCounter, encUint(v.Uint&0xffffffff)), nil
	case KindGauge32:
		return tlv(tagGauge, encUint(v.Uint&0xffffffff)), nil
	case KindTicks:
		return tlv(tagTicks, encUint(v.Uint&0xffffffff)), nil
	case KindCounter64:
		return tlv(tagCounter6, encUint(v.Uint)), nil
	case KindIPAddr:
		return tlv(tagIPAddr, v.Str), nil
	case KindNoSuchObject:
		return tlv(tagNoSuchObject, nil), nil
	case KindNoSuchInstance:
		return tlv(tagNoSuchInstance, nil), nil
	case KindEndOfMib:
		return tlv(tagEndOfMib, nil), nil
	}
	return nil, fmt.Errorf("snmp: cannot encode value kind %d", v.Kind)
}

// packet is a v2c message.
type packet struct {
	community string
	pduType   byte
	requestID int32
	// errStatus / nonRepeaters share the slot, as do errIndex / maxRepetitions.
	a, b     int64
	varBinds []VarBind
}

func (p *packet) marshal() ([]byte, error) {
	var vbs []byte
	for _, vb := range p.varBinds {
		name, err := encOID(vb.Name)
		if err != nil {
			return nil, err
		}
		val, err := encValue(vb.Value)
		if err != nil {
			return nil, err
		}
		vbs = append(vbs, tlv(tagSequence, append(tlv(tagOID, name), val...))...)
	}
	pdu := tlv(tagInteger, encInt(int64(p.requestID)))
	pdu = append(pdu, tlv(tagInteger, encInt(p.a))...)
	pdu = append(pdu, tlv(tagInteger, encInt(p.b))...)
	pdu = append(pdu, tlv(tagSequence, vbs)...)
	msg := tlv(tagInteger, encInt(version2c))
	msg = append(msg, tlv(tagOctets, []byte(p.community))...)
	msg = append(msg, tlv(p.pduType, pdu)...)
	return tlv(tagSequence, msg), nil
}

// ------------------------------------------------------------------ decode

var errShort = errors.New("snmp: truncated message")

type reader struct{ b []byte }

func (r *reader) next() (tag byte, body []byte, err error) {
	if len(r.b) < 2 {
		return 0, nil, errShort
	}
	tag = r.b[0]
	n := int(r.b[1])
	off := 2
	if n&0x80 != 0 {
		k := n & 0x7f
		if k == 0 || k > 3 || len(r.b) < 2+k {
			return 0, nil, errors.New("snmp: bad length")
		}
		n = 0
		for i := 0; i < k; i++ {
			n = n<<8 | int(r.b[2+i])
		}
		off = 2 + k
	}
	if len(r.b) < off+n {
		return 0, nil, errShort
	}
	body = r.b[off : off+n]
	r.b = r.b[off+n:]
	return tag, body, nil
}

func (r *reader) expect(want byte) ([]byte, error) {
	tag, body, err := r.next()
	if err != nil {
		return nil, err
	}
	if tag != want {
		return nil, fmt.Errorf("snmp: expected tag 0x%02x, got 0x%02x", want, tag)
	}
	return body, nil
}

func decInt(b []byte) (int64, error) {
	if len(b) == 0 || len(b) > 8 {
		return 0, errors.New("snmp: bad integer")
	}
	v := int64(int8(b[0]))
	for _, c := range b[1:] {
		v = v<<8 | int64(c)
	}
	return v, nil
}

func decUint(b []byte) (uint64, error) {
	if len(b) == 0 || len(b) > 9 || (len(b) == 9 && b[0] != 0) {
		return 0, errors.New("snmp: bad unsigned")
	}
	var v uint64
	for _, c := range b {
		v = v<<8 | uint64(c)
	}
	return v, nil
}

func decOID(b []byte) (OID, error) {
	if len(b) == 0 {
		return nil, errors.New("snmp: empty OID")
	}
	var subs []uint32
	var v uint64
	for i, c := range b {
		v = v<<7 | uint64(c&0x7f)
		if v > 0xffffffff {
			return nil, errors.New("snmp: OID component overflow")
		}
		if c&0x80 == 0 {
			subs = append(subs, uint32(v))
			v = 0
		} else if i == len(b)-1 {
			return nil, errors.New("snmp: unterminated OID")
		}
	}
	first := subs[0]
	var o OID
	switch {
	case first < 40:
		o = OID{0, first}
	case first < 80:
		o = OID{1, first - 40}
	default:
		o = OID{2, first - 80}
	}
	return append(o, subs[1:]...), nil
}

func decValue(tag byte, b []byte) (Value, error) {
	switch tag {
	case tagNull:
		return Value{Kind: KindNull}, nil
	case tagInteger:
		v, err := decInt(b)
		return Value{Kind: KindInt, Int: v}, err
	case tagOctets:
		return Value{Kind: KindString, Str: append([]byte(nil), b...)}, nil
	case tagOID:
		o, err := decOID(b)
		return Value{Kind: KindOID, OID: o}, err
	case tagIPAddr:
		return Value{Kind: KindIPAddr, Str: append([]byte(nil), b...)}, nil
	case tagCounter, tagGauge, tagTicks, tagCounter6:
		v, err := decUint(b)
		k := map[byte]Kind{tagCounter: KindCounter32, tagGauge: KindGauge32, tagTicks: KindTicks, tagCounter6: KindCounter64}[tag]
		return Value{Kind: k, Uint: v}, err
	case tagNoSuchObject:
		return Value{Kind: KindNoSuchObject}, nil
	case tagNoSuchInstance:
		return Value{Kind: KindNoSuchInstance}, nil
	case tagEndOfMib:
		return Value{Kind: KindEndOfMib}, nil
	}
	return Value{}, fmt.Errorf("snmp: unsupported value tag 0x%02x", tag)
}

func unmarshal(b []byte) (*packet, error) {
	top := reader{b}
	msg, err := top.expect(tagSequence)
	if err != nil {
		return nil, err
	}
	r := reader{msg}
	vb, err := r.expect(tagInteger)
	if err != nil {
		return nil, err
	}
	if v, err := decInt(vb); err != nil || v != version2c {
		return nil, errors.New("snmp: not an SNMPv2c message")
	}
	comm, err := r.expect(tagOctets)
	if err != nil {
		return nil, err
	}
	pduType, pdu, err := r.next()
	if err != nil {
		return nil, err
	}
	p := &packet{community: string(comm), pduType: pduType}
	pr := reader{pdu}
	var ints [3]int64
	for i := range ints {
		ib, err := pr.expect(tagInteger)
		if err != nil {
			return nil, err
		}
		if ints[i], err = decInt(ib); err != nil {
			return nil, err
		}
	}
	p.requestID, p.a, p.b = int32(ints[0]), ints[1], ints[2]
	list, err := pr.expect(tagSequence)
	if err != nil {
		return nil, err
	}
	lr := reader{list}
	for len(lr.b) > 0 {
		one, err := lr.expect(tagSequence)
		if err != nil {
			return nil, err
		}
		or := reader{one}
		nb, err := or.expect(tagOID)
		if err != nil {
			return nil, err
		}
		name, err := decOID(nb)
		if err != nil {
			return nil, err
		}
		vt, vbody, err := or.next()
		if err != nil {
			return nil, err
		}
		val, err := decValue(vt, vbody)
		if err != nil {
			return nil, err
		}
		p.varBinds = append(p.varBinds, VarBind{Name: name, Value: val})
	}
	return p, nil
}
