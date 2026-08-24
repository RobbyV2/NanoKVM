package proto

type GetVersionRsp struct {
	Current      string `json:"current"`
	Latest       string `json:"latest"`
	LatestKernel string `json:"latestKernel,omitempty"`
}

type UpdateRsp struct {
	Reboot bool `json:"reboot"`
}

type GetKernelRsp struct {
	Slot       string `json:"slot,omitempty"`
	Installed  string `json:"installed,omitempty"`
	RolledBack string `json:"rolledBack,omitempty"`
}

type GetPreviewRsp struct {
	Enabled bool `json:"enabled"`
}

type SetPreviewReq struct {
	Enable bool `validate:"omitempty"`
}

type GetUpdateServerRsp struct {
	Enabled bool   `json:"enabled"`
	URL     string `json:"url"`
}

type SetUpdateServerReq struct {
	Enabled *bool  `json:"enabled" form:"enabled" validate:"required"`
	URL     string `json:"url" form:"url"`
}

type StartupStatus struct {
	Name  string `json:"name"`
	Error string `json:"error,omitempty"`
}

type GetStartupRsp struct {
	Steps []StartupStatus `json:"steps"`
}
