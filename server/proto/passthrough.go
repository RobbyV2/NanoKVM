package proto

import "time"

// The imported device as the exporter described it. Isochronous endpoints are
// out: this raw-gadget exposes no frame number, allows one in-flight request per
// endpoint and caps a transfer at one page, so webcams and audio devices cannot
// pass through and the UI says so rather than letting someone find out.
type PassthroughDevice struct {
	BusID     string `json:"busId"`
	IDVendor  string `json:"idVendor"`
	IDProduct string `json:"idProduct"`
	Speed     string `json:"speed"`
	Class     uint8  `json:"class"`
}

// HIDSurrendered is the whole cost of a session: udc->driver is a single
// pointer, so while raw-gadget holds the UDC there is no keyboard, no mouse and
// no virtual media.
type GetPassthroughRsp struct {
	Active         bool               `json:"active"`
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

// Exporter is dialled and never spawned; BusID is matched against the usbip
// busid grammar before it reaches the wire.
type StartPassthroughReq struct {
	Exporter string `validate:"required,hostname_port|hostname|ip"`
	BusID    string `validate:"required,max=31"`
}
