package proto

type IP struct {
	Name    string `json:"name"`
	Addr    string `json:"addr"`
	Version string `json:"version"`
	Type    string `json:"type"`
}

type GetInfoRsp struct {
	IPs         []IP   `json:"ips"`
	Mdns        string `json:"mdns"`
	Image       string `json:"image"`
	Application string `json:"application"`
	DeviceKey   string `json:"deviceKey"`
}

type GetHardwareRsp struct {
	Version string `json:"version"`
}

type SetGpioReq struct {
	Type     string `validate:"required"`  // reset / power
	Duration uint   `validate:"omitempty"` // press time (unit: milliseconds)
}

type GetGpioRsp struct {
	PWR bool `json:"pwr"` // power led
	HDD bool `json:"hdd"` // hdd led
}

type SetScreenReq struct {
	Type  string `validate:"required"` // resolution / fps / quality
	Value int    `validate:"number"`   // value
}

type GetScriptsRsp struct {
	Files []string `json:"files"`
}

type UploadScriptRsp struct {
	File string `json:"file"`
}

type RunScriptReq struct {
	Name string `validate:"required"`
	Type string `validate:"required"` // foreground | background
}

type RunScriptRsp struct {
	Log string `json:"log"`
}

type DeleteScriptReq struct {
	Name string `validate:"required"`
}

// autostart
type GetAutostartRsp struct {
	Files []string `json:"files"`
}

type UploadAutostartReq struct {
	Content string `json:"content"`
}

// EndpointUsage is the dwc2 endpoint budget the linked functions consume. A
// controller runs out of endpoints long before it runs out of anything else, so
// this is what decides whether another function can be added at all.
type EndpointUsage struct {
	In  int `json:"in"`
	Out int `json:"out"`
}

type GetVirtualDeviceRsp struct {
	Network bool `json:"network"`
	Media   bool `json:"media"`
	Disk    bool `json:"disk"`

	// ncm or rndis, empty when no network function is linked. The gadget layer
	// builds these two and no others, so the selector offers these two.
	Protocol string `json:"protocol"`

	// What the active profile spends of the endpoint budget, and what is left.
	Endpoints EndpointUsage `json:"endpoints"`
	Headroom  EndpointUsage `json:"headroom"`
}

// Protocol names which network function the gadget should present. Empty keeps
// the toggle a toggle; set, it selects the protocol and leaves the function on.
type UpdateVirtualDeviceReq struct {
	Device   string `validate:"required"`
	Protocol string `validate:"omitempty"`
}

type UpdateVirtualDeviceRsp struct {
	On bool `json:"on"`
}

type SetMemoryLimitReq struct {
	Enabled bool  `validate:"omitempty"`
	Limit   int64 `validate:"omitempty"`
}

type GetMemoryLimitRsp struct {
	Enabled bool  `json:"enabled"`
	Limit   int64 `json:"limit"`
}

type SetOledReq struct {
	Sleep int `validate:"omitempty"`
}

type GetOLEDRsp struct {
	Exist bool `json:"exist"`
	Sleep int  `json:"sleep"`
}

type GetGetHdmiStateRsp struct {
	Enabled     bool `json:"enabled"`
	Signal      bool `json:"signal"`
	IdleTimeout int  `json:"idleTimeout"`
}

type SetHdmiIdleTimeoutReq struct {
	Minutes int `validate:"gte=0,lte=10080"`
}

type GetSSHStateRsp struct {
	Enabled bool `json:"enabled"`
}

type GetSwapRsp struct {
	Size int64 `json:"size"` // unit: MB
}

type SetSwapReq struct {
	Size int64 `validate:"omitempty"` // unit: MB
}

type GetMouseJigglerRsp struct {
	Enabled bool   `json:"enabled"`
	Mode    string `json:"mode"`
}

type SetMouseJigglerReq struct {
	Enabled bool   `validate:"omitempty"`
	Mode    string `validate:"omitempty"`
}

type GetMdnsStateRsp struct {
	Enabled bool `json:"enabled"`
}

type SetHostnameReq struct {
	Hostname string `validate:"required"`
}

type GetHostnameRsp struct {
	Hostname string `json:"hostname"`
}

type SetWebTitleReq struct {
	Title string `validate:"omitempty"`
}

type GetWebTitleRsp struct {
	Title string `json:"title"`
}

type SetFaviconReq struct {
	// Empty resets to the device default, mirroring SetWebTitleReq.
	Url string `json:"url" form:"url" validate:"omitempty"`
}

type GetFaviconRsp struct {
	Source      string `json:"source"` // custom | boot | default
	ContentType string `json:"contentType"`
	Size        int    `json:"size"`
	Version     string `json:"version"` // cache key for the icon URL
	BootLogo    bool   `json:"bootLogo"`
}

type SetTlsReq struct {
	Enabled bool `validate:"omitempty"`
}

type GetVNCRsp struct {
	Enabled bool `json:"enabled"`
	Port    int  `json:"port"`
}

type SetVNCReq struct {
	Enabled bool `validate:"omitempty"`
}

type InputRegion struct {
	Mode               string               `json:"mode"`
	FrameWidth         int                  `json:"frameWidth"`
	FrameHeight        int                  `json:"frameHeight"`
	Left               int                  `json:"left"`
	Top                int                  `json:"top"`
	Width              int                  `json:"width"`
	Height             int                  `json:"height"`
	Resolutions        []OriginalResolution `json:"resolutions,omitempty"`
	SelectedResolution string               `json:"selectedResolution"`
	Regions            []ManualRegion       `json:"regions,omitempty"`
	SelectedRegion     string               `json:"selectedRegion"`
}

type ManualRegion struct {
	FrameWidth  int `json:"frameWidth"`
	FrameHeight int `json:"frameHeight"`
	Left        int `json:"left"`
	Top         int `json:"top"`
	Width       int `json:"width"`
	Height      int `json:"height"`
}

type OriginalResolution struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

type SetInputRegionReq struct {
	Mode               string                `json:"mode"`
	FrameWidth         *int                  `json:"frameWidth,omitempty"`
	FrameHeight        *int                  `json:"frameHeight,omitempty"`
	Left               *int                  `json:"left,omitempty"`
	Top                *int                  `json:"top,omitempty"`
	Width              *int                  `json:"width,omitempty"`
	Height             *int                  `json:"height,omitempty"`
	Resolutions        *[]OriginalResolution `json:"resolutions,omitempty"`
	SelectedResolution *string               `json:"selectedResolution,omitempty"`
	Regions            *[]ManualRegion       `json:"regions,omitempty"`
	SelectedRegion     *string               `json:"selectedRegion,omitempty"`
}

type GetInputRegionRsp struct {
	InputRegion
}

type GetInputResolutionRsp struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}
