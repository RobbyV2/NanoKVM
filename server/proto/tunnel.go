package proto

type TunnelName string

const (
	TunnelWstunnel TunnelName = "wstunnel"
	TunnelNewt     TunnelName = "newt"
)

type TunnelState string

const (
	TunnelNotInstall    TunnelState = "notInstall"
	TunnelNotConfigured TunnelState = "notConfigured"
	TunnelStopped       TunnelState = "stopped"
	TunnelRunning       TunnelState = "running"
	TunnelConnected     TunnelState = "connected"
	TunnelError         TunnelState = "error"
)

type TunnelEnvEntry struct {
	Key        string `json:"key"`
	Value      string `json:"value"`
	Secret     bool   `json:"secret"`
	Configured bool   `json:"configured"`
}

type GetTunnelStatusRsp struct {
	State   TunnelState `json:"state"`
	Message string      `json:"message"`
	Pid     int         `json:"pid"`
	Custom  bool        `json:"custom"`
	Enabled bool        `json:"enabled"`
}

type GetTunnelConfigRsp struct {
	Args string           `json:"args"`
	Env  []TunnelEnvEntry `json:"env"`
}

type GetTunnelLogsRsp struct {
	Lines []string `json:"lines"`
}

type GetTunnelMemoryRsp struct {
	Supported bool  `json:"supported"`
	Enabled   bool  `json:"enabled"`
	Limit     int64 `json:"limit"`
}

type SetTunnelMemoryReq struct {
	Enabled bool `validate:"omitempty"`
}

type SetTunnelConfigReq struct {
	Args string            `validate:"omitempty"`
	Env  map[string]string `validate:"omitempty"`
}
