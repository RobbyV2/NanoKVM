package exit

import (
	"time"

	"NanoKVM-Server/proto"
)

// newComponents wires one slot's dataplane from the real constructors (plan,
// "Contracts between packages"). The manager owns the result and stops every
// part of it on disable.
//
// Byte accounting: the mux counts DATA payload in Mode A and the front door
// counts relayed payload in Mode B, so those two share the slot's counter. The
// WSProxy would count WebSocket transport bytes on the same Mode B connection
// a second time, so it gets no counter.
func newComponents(slot Slot, deps componentDeps) components {
	mux := NewMux(slot, MuxHooks{
		OnSession: deps.OnSession,
		OnClose:   deps.OnClose,
		Policy:    deps.Policy,
		Bytes:     deps.Bytes,
	})
	door := NewFrontDoor(slot, deps.Policy, deps.Bytes)
	return components{
		Door:  &wiredDoor{FrontDoor: door, mux: mux},
		Mux:   mux,
		Proxy: NewWSProxy(slot, nil, deps.OnPeerChange),
		DNS:   NewForwarder(slot, deps.Upstreams, deps.Connected),
		Probe: NewProber(slot, deps.Connected),
	}
}

// wiredDoor is the front door as the manager sees it, with the one state the
// front door cannot know on its own: a Mode A upgrade has been accepted but
// the HELLO has not arrived yet. The front door reports that window for Mode B
// (peer tracked, reverse listener unproven); the mux reports it for Mode A.
type wiredDoor struct {
	*FrontDoor
	mux *Mux
}

func (d *wiredDoor) Attached() (proto.ExitTunnelState, *proto.ExitPeer, *time.Time) {
	state, peer, since := d.FrontDoor.Attached()
	if state == proto.ExitDisconnected && d.mux.Connecting() {
		return proto.ExitConnecting, nil, nil
	}
	return state, peer, since
}
