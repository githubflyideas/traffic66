package store

import (
	"fmt"
	"net/netip"
	"sort"
	"strings"
	"time"
)

// Billing by subnet: the traffic of every address of one CIDR range,
// judged exactly as a product line's traffic is judged — a flow with
// both ends inside the range is inside traffic, counted once in the
// total, with the in and out split kept for the reconciliation. The
// set is a range, not a list: a /64 holds more addresses than any
// list could name, so the summaries are aggregated by a range
// predicate on each flow's two ends, never by expanding the range
// into addresses and reusing the list path.
//
// The figures come from the billing summaries alone, read by the
// whole hour buckets that cover the range, the way the summaries
// answer a product line whose records are gone. The records are not
// read: the summaries are folded from them as the minutes settle and
// backfilled on upgrade, and a subnet is a billing question, written
// over the same spans a bill is. The coverage marks say where the
// summaries reach, as they do on the other path.

// subnetMemberCap is how many member addresses a subnet answer
// carries. A wide range can hold thousands of live addresses; the
// split under the total names the largest of them, and MembersTotal
// says how many there were.
const subnetMemberCap = 256

// BillingSubnet returns the traffic of the addresses of p over q's
// range, in the shape of Billing. The prefix is masked: 10.0.0.9/24
// asks for 10.0.0.0/24, the range the address stands in.
func (s *Store) BillingSubnet(q Query, p netip.Prefix) (*Billing, error) {
	p = p.Masked()
	// the summaries are kept by the hour: a range they answer is
	// bucketed by the hour at finest, where the records bucket a
	// short range by the minute
	step := StepFor(q.span())
	if step < time.Hour {
		step = time.Hour
	}
	start := q.localTrunc(q.From, step)
	settled := settledTo(q.To, false)
	n := bucketCount(start, q.To, step)
	b := &Billing{Step: int64(step.Seconds())}
	for i := 0; i < n; i++ {
		b.Times = append(b.Times, start.Add(time.Duration(i)*step).UnixMilli())
	}
	inB := make([]float64, n)   // bytes per bucket
	outB := make([]float64, n)  // bytes per bucket
	insB := make([]float64, n)  // bytes per bucket with both ends in the range
	mIn := map[string]uint64{}  // settled bytes per member address
	mOut := map[string]uint64{} // settled bytes per member address
	if err := s.billingSubnetRows(q, p, start, step, inB, outB, insB, mIn, mOut); err != nil {
		return nil, err
	}

	dataFrom := s.billStart()
	b.In = make([]float64, n)
	b.Out = make([]float64, n)
	b.Total = make([]float64, n)
	for i := 0; i < n; i++ {
		secs := bucketSecs(start, i, step, q.To, dataFrom)
		b.In[i] = inB[i] * 8 / secs
		b.Out[i] = outB[i] * 8 / secs
		b.Total[i] = (inB[i] + outB[i] - insB[i]) * 8 / secs
		// the totals are the settled part only, as on the list path
		bt := start.Add(time.Duration(i) * step)
		if bt.Before(settled) {
			b.InBytes += uint64(inB[i])
			b.OutBytes += uint64(outB[i])
			b.Bytes += uint64(inB[i] + outB[i] - insB[i])
			b.addDay(bt, q.ZoneOffset, inB[i], outB[i], inB[i]+outB[i]-insB[i])
		}
	}
	b.SettledTo = unsettledMark(settled, q.To)
	b.DataFrom = dataFromMark(start, dataFrom)

	// the member split: every address of the range that carried
	// traffic, the largest first, capped — the full count rides
	// along so a capped list is never mistaken for the whole set
	type member struct {
		addr string
		row  MemberRow
	}
	members := make([]member, 0, len(mIn)+len(mOut))
	seen := map[string]bool{}
	for a, w := range mIn {
		seen[a] = true
		members = append(members, member{a, MemberRow{Addr: a, InBytes: w, OutBytes: mOut[a], Bytes: w + mOut[a]}})
	}
	for a, w := range mOut {
		if !seen[a] {
			members = append(members, member{a, MemberRow{Addr: a, OutBytes: w, Bytes: w}})
		}
	}
	sort.Slice(members, func(i, j int) bool {
		if members[i].row.Bytes != members[j].row.Bytes {
			return members[i].row.Bytes > members[j].row.Bytes
		}
		return members[i].addr < members[j].addr
	})
	b.MembersTotal = len(members)
	if len(members) > subnetMemberCap {
		members = members[:subnetMemberCap]
	}
	for _, m := range members {
		b.Members = append(b.Members, m.row)
	}
	return b, nil
}

// billingSubnetRows fills the buckets and the member split from the
// billing summaries, over the whole hour buckets that cover q's
// range. The direction classes are the list path's own — 1 arriving
// at the range from outside it, 2 sent by the range to outside it,
// 3 inside it — judged by both ends of each folded pair, here by
// whether each end's key falls in the prefix's key range.
func (s *Store) billingSubnetRows(q Query, p netip.Prefix, start time.Time, step time.Duration, inB, outB, insB []float64, mIn, mOut map[string]uint64) error {
	n := len(inB)
	ef, et := hourRange(q.From, q.To)
	lo, hi := prefixKeyRange(p)
	srcK, dstK := addrKeySQL("src"), addrKeySQL("dst")
	dup := ""
	if !q.IncludeDup {
		dup = " AND NOT dup"
	}
	bucket := q.bucketSQL(step)
	// each row's two keys are rendered once, in the inner select:
	// the rendering is string surgery, and the class tests and the
	// filter would otherwise pay it five times a row
	sqlq := fmt.Sprintf(`SELECT %s AS b,
		CASE WHEN sk BETWEEN ? AND ? AND dk BETWEEN ? AND ? THEN 3 WHEN dk BETWEEN ? AND ? THEN 1 ELSE 2 END AS cls,
		CAST(sum(wire) AS DOUBLE)
		FROM (SELECT ts, dup, wire, %s AS sk, %s AS dk FROM r_bill WHERE ts >= ? AND ts < ?)
		WHERE (sk BETWEEN ? AND ? OR dk BETWEEN ? AND ?)%s GROUP BY 1, 2`,
		bucket, srcK, dstK, dup)
	// the arguments take the placeholders in the order they appear
	// in the text: the CASE's three membership tests first (the
	// select list precedes the FROM), then the inner time bounds,
	// then the filter's two tests
	full := []any{lo, hi, lo, hi, lo, hi, ef, et, lo, hi, lo, hi}
	rows, err := s.DB.Query(sqlq, full...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var t time.Time
		var cls int
		var w float64
		if err := rows.Scan(&t, &cls, &w); err != nil {
			return err
		}
		ti := int(t.UTC().Sub(start) / step)
		if ti < 0 || ti >= n {
			continue
		}
		switch cls {
		case 1:
			inB[ti] += w
		case 2:
			outB[ti] += w
		case 3:
			insB[ti] += w
			inB[ti] += w
			outB[ti] += w
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	mq := fmt.Sprintf(`SELECT dst, CAST(sum(wire) AS UBIGINT)
		FROM (SELECT dst, dup, wire, %s AS dk FROM r_bill WHERE ts >= ? AND ts < ?)
		WHERE dk BETWEEN ? AND ?%s GROUP BY 1`, dstK, dup)
	margs := []any{ef, et, lo, hi}
	mrows, err := s.DB.Query(mq, margs...)
	if err != nil {
		return err
	}
	defer mrows.Close()
	for mrows.Next() {
		var a string
		var w uint64
		if err := mrows.Scan(&a, &w); err != nil {
			return err
		}
		mIn[a] = w
	}
	if err := mrows.Err(); err != nil {
		return err
	}
	mq = fmt.Sprintf(`SELECT src, CAST(sum(wire) AS UBIGINT)
		FROM (SELECT src, dup, wire, %s AS sk FROM r_bill WHERE ts >= ? AND ts < ?)
		WHERE sk BETWEEN ? AND ?%s GROUP BY 1`, srcK, dup)
	mrows2, err := s.DB.Query(mq, margs...)
	if err != nil {
		return err
	}
	defer mrows2.Close()
	for mrows2.Next() {
		var a string
		var w uint64
		if err := mrows2.Scan(&a, &w); err != nil {
			return err
		}
		mOut[a] = w
	}
	return mrows2.Err()
}

// prefixKeyRange is the pair of address keys bounding a masked
// prefix: the key of its first address and of its last. An address
// belongs to the prefix exactly when its key lies between the two —
// the keys are fixed-width and family-tagged, so the string order
// within a family is the address order, and no address of the other
// family can fall inside the range.
func prefixKeyRange(p netip.Prefix) (lo, hi string) {
	p = p.Masked()
	first := p.Addr()
	last := first
	b := first.As16()
	host := 128 - p.Bits()
	if first.Is4() {
		host = 32 - p.Bits()
	}
	for i := 0; i < 16; i++ {
		bit := 128 - 8*i // the bit position just past this byte's top
		switch {
		case host >= bit:
			b[i] = 0xff
		case host > bit-8:
			b[i] |= byte(0xff >> (bit - host))
		}
	}
	if first.Is4() {
		last = netip.AddrFrom4([4]byte{b[12], b[13], b[14], b[15]})
		return "4" + addrKey4(first), "4" + addrKey4(last)
	}
	last = netip.AddrFrom16(b)
	return "6" + addrKey6(first), "6" + addrKey6(last)
}

// addrKey4 renders an IPv4 address as its four octets, each padded
// to three digits: the fixed-width form addrKeySQL builds in SQL.
func addrKey4(a netip.Addr) string {
	b := a.As4()
	return fmt.Sprintf("%03d%03d%03d%03d", b[0], b[1], b[2], b[3])
}

// addrKey6 renders an IPv6 address as its sixteen bytes in hex: the
// fixed-width form addrKeySQL builds in SQL.
func addrKey6(a netip.Addr) string {
	b := a.As16()
	const digits = "0123456789abcdef"
	var sb strings.Builder
	for _, c := range b {
		sb.WriteByte(digits[c>>4])
		sb.WriteByte(digits[c&0xf])
	}
	return sb.String()
}

// addrKeySQL is a SQL expression turning the address column c — a
// canonical address string, the form the flow records keep — into
// the address's key: a family tag ('4' or '6') and the fixed-width
// rendering prefixKeyRange bounds a prefix with. DuckDB's INET type
// lives in an extension that is not loaded, so the rendering is
// built from the string itself: the octets padded for IPv4, and for
// IPv6 the compressed '::' first expanded — the part before it, as
// many zero groups as it stands for, the part after it — then every
// group padded to four hex digits.
func addrKeySQL(c string) string {
	v4 := "'4'"
	for i := 1; i <= 4; i++ {
		v4 += fmt.Sprintf(" || lpad(split_part(%s, '.', %d), 3, '0')", c, i)
	}
	// the two halves around '::', and how many groups each names
	h := fmt.Sprintf("split_part(%s, '::', 1)", c)
	t := fmt.Sprintf("split_part(%s, '::', 2)", c)
	groups := func(x string) string {
		return fmt.Sprintf("CASE WHEN %s = '' THEN 0 ELSE length(%s) - length(replace(%s, ':', '')) + 1 END", x, x, x)
	}
	expanded := fmt.Sprintf("CASE WHEN contains(%s, '::') THEN trim(%s || ':' || repeat('0:', 8 - %s - %s) || %s, ':') ELSE %s END",
		c, h, groups(h), groups(t), t, c)
	v6 := "'6'"
	for i := 1; i <= 8; i++ {
		v6 += fmt.Sprintf(" || lpad(split_part(%s, ':', %d), 4, '0')", expanded, i)
	}
	return fmt.Sprintf("CASE WHEN contains(%s, ':') THEN %s ELSE %s END", c, v6, v4)
}
