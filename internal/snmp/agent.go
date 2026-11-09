package snmp

import (
	"context"
	"net"
	"sort"
)

// Agent answers GET, GETNEXT and GETBULK from a snapshot of bindings. It is
// read-only and serves one community; it exists for the demo and tests.
type Agent struct {
	Community string
	// View returns the current bindings; they are sorted by the agent.
	View func() []VarBind
}

// InterfaceView builds IF-MIB bindings for interfaces and their counters.
func InterfaceView(rows []InterfaceRow) []VarBind {
	var out []VarBind
	col := func(c OID, idx uint32, v Value) {
		out = append(out, VarBind{Name: append(append(OID{}, c...), idx), Value: v})
	}
	for _, r := range rows {
		col(OIDIfName, r.Index, Value{Kind: KindString, Str: []byte(r.Name)})
		col(OIDIfHCInOctets, r.Index, Value{Kind: KindCounter64, Uint: r.InOctets})
		col(OIDIfHCInUcastPkt, r.Index, Value{Kind: KindCounter64, Uint: r.InPkts})
		col(OIDIfHCOutOctets, r.Index, Value{Kind: KindCounter64, Uint: r.OutOctets})
		col(OIDIfHCOutUcastPkt, r.Index, Value{Kind: KindCounter64, Uint: r.OutPkts})
		col(OIDIfHighSpeed, r.Index, Value{Kind: KindGauge32, Uint: r.SpeedMbps})
		col(OIDIfAlias, r.Index, Value{Kind: KindString, Str: []byte(r.Alias)})
	}
	return out
}

// InterfaceRow is one interface in an agent view.
type InterfaceRow struct {
	Index                                uint32
	Name, Alias                          string
	SpeedMbps                            uint64
	InOctets, OutOctets, InPkts, OutPkts uint64
}

// Serve answers requests on conn until ctx ends.
func (a *Agent) Serve(ctx context.Context, conn *net.UDPConn) {
	go func() { <-ctx.Done(); conn.Close() }()
	buf := make([]byte, 65535)
	for {
		n, from, err := conn.ReadFromUDPAddrPort(buf)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			continue
		}
		req, err := unmarshal(buf[:n])
		if err != nil || req.community != a.Community {
			continue // wrong community: silence, like real agents
		}
		resp := a.answer(req)
		if b, err := resp.marshal(); err == nil {
			conn.WriteToUDPAddrPort(b, from)
		}
	}
}

func (a *Agent) answer(req *packet) *packet {
	view := a.View()
	sort.Slice(view, func(i, j int) bool { return view[i].Name.Compare(view[j].Name) < 0 })
	next := func(o OID) VarBind {
		i := sort.Search(len(view), func(i int) bool { return view[i].Name.Compare(o) > 0 })
		if i == len(view) {
			return VarBind{Name: o, Value: Value{Kind: KindEndOfMib}}
		}
		return view[i]
	}
	resp := &packet{community: req.community, pduType: pduResponse, requestID: req.requestID}
	switch req.pduType {
	case pduGet:
		for _, vb := range req.varBinds {
			i := sort.Search(len(view), func(i int) bool { return view[i].Name.Compare(vb.Name) >= 0 })
			if i < len(view) && view[i].Name.Compare(vb.Name) == 0 {
				resp.varBinds = append(resp.varBinds, view[i])
			} else {
				resp.varBinds = append(resp.varBinds, VarBind{Name: vb.Name, Value: Value{Kind: KindNoSuchInstance}})
			}
		}
	case pduGetNext:
		for _, vb := range req.varBinds {
			resp.varBinds = append(resp.varBinds, next(vb.Name))
		}
	case pduGetBulk:
		nonRep, reps := int(req.a), int(req.b)
		if nonRep < 0 {
			nonRep = 0
		}
		if nonRep > len(req.varBinds) {
			nonRep = len(req.varBinds)
		}
		for _, vb := range req.varBinds[:nonRep] {
			resp.varBinds = append(resp.varBinds, next(vb.Name))
		}
		cur := make([]OID, 0, len(req.varBinds)-nonRep)
		for _, vb := range req.varBinds[nonRep:] {
			cur = append(cur, vb.Name)
		}
		for r := 0; r < reps && r < 200; r++ {
			done := true
			for i := range cur {
				vb := next(cur[i])
				resp.varBinds = append(resp.varBinds, vb)
				cur[i] = vb.Name
				if vb.Value.Kind != KindEndOfMib {
					done = false
				}
			}
			if done {
				break
			}
		}
	}
	return resp
}
