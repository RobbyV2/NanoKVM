package proto

import "time"

// The imported device as the exporter described it. Both modes refuse
// isochronous endpoints.
type PassthroughDevice struct {
	BusID     string `json:"busId"`
	IDVendor  string `json:"idVendor"`
	IDProduct string `json:"idProduct"`
	Speed     string `json:"speed"`
	Class     uint8  `json:"class"`
}

// HIDSurrendered distinguishes Exact from Hybrid sessions.
type GetPassthroughRsp struct {
	Active         bool               `json:"active"`
	Mode           string             `json:"mode"`
	Exporter       string             `json:"exporter"`
	UDC            string             `json:"udc"`
	Port           uint32             `json:"port"`
	Hub            string             `json:"hub"`
	Bus            uint32             `json:"bus"`
	Address        uint32             `json:"address"`
	Pid            int                `json:"pid"`
	HIDSurrendered bool               `json:"hidSurrendered"`
	StartedAt      time.Time          `json:"startedAt"`
	Device         *PassthroughDevice `json:"device"`
}

// One entry of the exporter's device list. Unsupported carries why the device
// cannot be relayed and is empty when it can.
type PassthroughRemoteDevice struct {
	BusID       string `json:"busId"`
	IDVendor    string `json:"idVendor"`
	IDProduct   string `json:"idProduct"`
	Speed       string `json:"speed"`
	Class       uint8  `json:"class"`
	Unsupported string `json:"unsupported,omitempty"`
}

type ListPassthroughRsp struct {
	Devices []PassthroughRemoteDevice `json:"devices"`
}

// Only addresses the exporter policy allows are dialled.
type ListPassthroughReq struct {
	Exporter string `validate:"required,hostname_port|hostname|ip"`
}

// Exporter is dialled and never spawned; BusID is matched against the usbip
// busid grammar before it reaches the wire.
type StartPassthroughReq struct {
	Exporter string `validate:"required,hostname_port|hostname|ip"`
	BusID    string `validate:"required,max=31"`
	Mode     string `validate:"omitempty,oneof=hybrid exact"`
}
