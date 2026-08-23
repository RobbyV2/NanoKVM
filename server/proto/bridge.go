package proto

import "time"

// The durable outcome of the last apply, read back by a client whose connection
// the apply itself cut: enabling the bridge moves the management address, so
// losing the response is the expected case and GET is how it learns what
// happened.
type BridgeState string

const (
	// The stock device: no br0, the address on eth0, no l2-uplink file. A device
	// that never bridged and one that was disabled are byte-identical here.
	BridgeDisabled BridgeState = "disabled"

	BridgeEnabled BridgeState = "enabled"

	// A gate failed or the deadline expired and the snapshot was restored.
	BridgeRolledBack BridgeState = "rolledBack"

	// The restore itself did not complete. The device may need the wlan0 AP or
	// a serial console.
	BridgeFailed BridgeState = "failed"

	// A pending.json is armed: an apply is in flight, or the device booted into
	// an unfinished one.
	BridgePending BridgeState = "pending"
)

// All three gates must be true before the dead-man is disarmed. Address and
// Gateway together are only L3 reachability; Inbound is what proves the
// management plane answers, which is a different property on a device whose
// S95nanokvm installs DROP OUTPUT tcp --sport 8000.
type BridgeChecks struct {
	Address bool `json:"address"`
	Gateway bool `json:"gateway"`
	Inbound bool `json:"inbound"`

	// Inbound was satisfied by the self-connect fallback rather than a real
	// client, which proves the listener and local delivery, not the wire.
	InboundWeak bool `json:"inboundWeak"`
}

func (c BridgeChecks) Passed() bool {
	return c.Address && c.Gateway && c.Inbound
}

type BridgePort struct {
	Name  string `json:"name"`
	State string `json:"state"`
	Up    bool   `json:"up"`

	// Carrier is IFF_LOWER_UP: the cable, as opposed to Up's administrative
	// state, which is set on a port with nothing plugged into it too.
	Carrier bool `json:"carrier"`
}

// A second path between the uplink's segment and a port of the same bridge.
// STP is off by design, because its listening delay costs the DHCP lease, so
// nothing in the kernel breaks such a loop and this warning is the compensating
// control. Non-nil is evidence the condition exists; nil is not evidence that
// it does not, since the entry it is read from moves between ports.
type BridgeLoop struct {
	Port   string `json:"port"`
	MAC    string `json:"mac"`
	Reason string `json:"reason"`
}

type BridgeArmed struct {
	Operation    string    `json:"operation"`
	SnapshotPath string    `json:"snapshotPath"`
	ArmedAt      time.Time `json:"armedAt"`
	Deadline     time.Time `json:"deadline"`
}

type BridgeApply struct {
	State     BridgeState  `json:"state"`
	Uplink    string       `json:"uplink"`
	Enabled   bool         `json:"enabled"`
	Checks    BridgeChecks `json:"checks"`
	Message   string       `json:"message"`
	AppliedAt time.Time    `json:"appliedAt"`
}

type GetBridgeRsp struct {
	State   BridgeState  `json:"state"`
	Uplink  string       `json:"uplink"`
	Exists  bool         `json:"exists"`
	MAC     string       `json:"mac"`
	Ports   []BridgePort `json:"ports"`
	Address string       `json:"address"`
	Gateway string       `json:"gateway"`

	// Carrier on the device Uplink names, which is br0 once the bridge holds
	// the address and eth0 otherwise. A false here with the bridge enabled is
	// the state the enable preflight exists to refuse to create.
	Carrier bool `json:"carrier"`

	Loop      *BridgeLoop  `json:"loop"`
	Pending   *BridgeArmed `json:"pending"`
	LastApply *BridgeApply `json:"lastApply"`

	// ncm, rndis, or empty for a gadget presenting no network. Reported from
	// the presentation snapshot, because the protocol the gadget presents is a
	// property of the USB profile and not of the bridge. The control for it
	// lives on the Virtual Network switch under Settings, Device.
	Protocol string `json:"protocol"`
}

type SetBridgeReq struct {
	Enabled bool `json:"enabled"`
}

type SetBridgeRsp struct {
	State   BridgeState  `json:"state"`
	Uplink  string       `json:"uplink"`
	Checks  BridgeChecks `json:"checks"`
	Message string       `json:"message"`
}
