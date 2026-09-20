package proto

import "time"

// ExitMode selects how the exit device reaches the slot's SOCKS front door.
type ExitMode string

const (
	// ExitModeNative is Mode A: the exit runs one of the embedded nexit/1
	// clients (PowerShell, Perl, Python) against /exit/<slot>/native.
	ExitModeNative ExitMode = "native"
	// ExitModeWstunnel is Mode B: the exit runs a pinned wstunnel binary as a
	// reverse-SOCKS client against a wstunnel server behind /exit/<slot>/.
	ExitModeWstunnel ExitMode = "wstunnel"
)

// ExitTunnelState is the state of the exit-to-NanoKVM channel.
type ExitTunnelState string

const (
	ExitDisconnected ExitTunnelState = "disconnected"
	ExitConnecting   ExitTunnelState = "connecting"
	ExitConnected    ExitTunnelState = "connected"
)

// ExitPeer identifies the exit device currently (or previously) attached.
type ExitPeer struct {
	Addr      string   `json:"addr"`
	Hostname  string   `json:"hostname,omitempty"`
	OS        string   `json:"os,omitempty"`
	Transport ExitMode `json:"transport"`
}

// ExitNIC describes the consumer-facing gadget NIC as last resolved.
type ExitNIC struct {
	Ifname   string `json:"ifname"`
	Up       bool   `json:"up"`
	Protocol string `json:"protocol"` // ncm, rndis or "" when the profile links no network function
	Address  string `json:"address"`  // 10.x.y.1/24 or ""
}

// ExitDownstream is the boolean vector `S94exit status <slot>` reports.
type ExitDownstream struct {
	Forward  bool `json:"forward"`  // net.ipv4.ip_forward=1
	Routing  bool `json:"routing"`  // ip rules by pref + fence route present
	Tun      bool `json:"tun"`      // persistent tun exists and has carrier (hev attached)
	Hev      bool `json:"hev"`      // hev-socks5-tunnel pid alive
	Wstunnel bool `json:"wstunnel"` // wstunnel server pid alive (always true in native mode)
	DNS      bool `json:"dns"`      // in-process forwarder bound to the gadget address
	NAT      bool `json:"nat"`      // chains present and jumped
}

// ExitUpstream is the result of the last reachability probe through the tunnel.
type ExitUpstream struct {
	Reachable bool       `json:"reachable"`
	LatencyMs int64      `json:"latencyMs"`
	CheckedAt *time.Time `json:"checkedAt"`
}

// ExitDNSStats are counters of the in-process forwarder.
type ExitDNSStats struct {
	Queries    uint64 `json:"queries"`
	Failures   uint64 `json:"failures"`
	Redirected uint64 `json:"redirected"` // conntrack DNAT hits, best effort
}

// ExitBytes are relayed byte counters for the slot.
type ExitBytes struct {
	Up   uint64 `json:"up"`   // consumer -> exit
	Down uint64 `json:"down"` // exit -> consumer
}

// GetExitStatusRsp is the per-slot status the UI polls every 3 s.
type GetExitStatusRsp struct {
	Slot            string          `json:"slot"`
	Enabled         bool            `json:"enabled"`
	Pending         bool            `json:"pending"`
	Mode            ExitMode        `json:"mode"`
	Token           string          `json:"token"`
	Tunnel          ExitTunnelState `json:"tunnel"`
	Peer            *ExitPeer       `json:"peer"`
	PreviousPeer    *ExitPeer       `json:"previousPeer"`
	PeerChangedAt   *time.Time      `json:"peerChangedAt"`
	ConnectedAt     *time.Time      `json:"connectedAt"`
	LastConnectedAt *time.Time      `json:"lastConnectedAt"`
	UptimeSeconds   int64           `json:"uptimeSeconds"`
	NIC             ExitNIC         `json:"nic"`
	Downstream      ExitDownstream  `json:"downstream"`
	Upstream        ExitUpstream    `json:"upstream"`
	DNS             ExitDNSStats    `json:"dns"`
	Bytes           ExitBytes       `json:"bytes"`
	Message         string          `json:"message"`
}

// GetExitSlotsRsp lists every slot with its status.
type GetExitSlotsRsp struct {
	Slots []GetExitStatusRsp `json:"slots"`
}

// GetExitConfigRsp is the editable part of the slot config. The token is not
// here; it travels with status because the panel shows both together.
type GetExitConfigRsp struct {
	Slot         string   `json:"slot"`
	Mode         ExitMode `json:"mode"`
	DNS          []string `json:"dns"`
	MTU          int      `json:"mtu"`
	AllowPrivate bool     `json:"allowPrivate"`
	PinPeer      bool     `json:"pinPeer"`
}

// SetExitConfigReq updates the editable fields; nil pointers leave a field alone.
type SetExitConfigReq struct {
	Mode         *ExitMode `json:"mode" validate:"omitempty,oneof=native wstunnel"`
	DNS          *[]string `json:"dns" validate:"omitempty,max=4,dive,ip"`
	MTU          *int      `json:"mtu" validate:"omitempty,min=576,max=1400"`
	AllowPrivate *bool     `json:"allowPrivate" validate:"omitempty"`
	PinPeer      *bool     `json:"pinPeer" validate:"omitempty"`
}

// ExitCommand is one ready-to-paste command for one platform.
type ExitCommand struct {
	Platform string `json:"platform"` // windows, macos, linux
	Shell    string `json:"shell"`    // powershell, bash
	Command  string `json:"command"`
	Notes    string `json:"notes,omitempty"`
}

// GetExitCommandsRsp carries both modes' commands plus what they were templated from.
type GetExitCommandsRsp struct {
	Scheme          string        `json:"scheme"`      // http or https
	Host            string        `json:"host"`        // host[:port] as the operator reached the UI
	Fingerprint     string        `json:"fingerprint"` // sha256 hex of the serving certificate, "" on http
	WstunnelVersion string        `json:"wstunnelVersion"`
	WstunnelRepo    string        `json:"wstunnelRepo"` // upstream repository, the manual fallback for the latest commands
	Native          []ExitCommand `json:"native"`
	// Nexit is the third way onto a Windows machine: the client this project
	// ships as a binary, for a host with no usable scripting host. It speaks
	// the same protocol as Native, so the slot stays in the native mode.
	Nexit          []ExitCommand `json:"nexit"`
	Wstunnel       []ExitCommand `json:"wstunnel"`
	WstunnelLatest []ExitCommand `json:"wstunnelLatest"` // windows, macos, linux; release resolved at run time
}

// GetExitLogsRsp is the tail of the slot's daemon logs, token-shaped values redacted.
type GetExitLogsRsp struct {
	Hev      []string `json:"hev"`
	Wstunnel []string `json:"wstunnel"`
}
