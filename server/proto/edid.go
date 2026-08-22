package proto

type EdidState string

const (
	EdidStateSuccess       EdidState = "success"
	EdidStateUnverified    EdidState = "unverified"
	EdidStateNeedsRecovery EdidState = "needs_recovery"
	EdidStateChipRefused   EdidState = "chip_refused"
	EdidStateBusContention EdidState = "bus_contention"
	EdidStatePreflight     EdidState = "preflight"
	EdidStateInvalidInput  EdidState = "invalid_input"
	EdidStateTimeout       EdidState = "timeout"
	EdidStateGeneric       EdidState = "generic"
)

type ApplyEdidReq struct {
	Profile string `validate:"omitempty"` // sha256 id of a shipped profile
	Data    string `validate:"omitempty"` // base64 blob, exclusive with Profile
}

type DecodeEdidReq struct {
	Data string `validate:"omitempty"`
}

type RestoreEdidReq struct {
	Source string `validate:"required,oneof=factory history"`
	ID     string `validate:"omitempty"` // empty means the most recent entry
}

type GetEdidBackupReq struct {
	ID string `form:"id" validate:"required"`
}

type EdidSummary struct {
	SHA256        string `json:"sha256"`
	Manufacturer  string `json:"manufacturer"`
	Model         string `json:"model"`
	ProductCode   uint16 `json:"productCode"`
	Serial        uint32 `json:"serial"`
	Week          uint8  `json:"week"`
	Year          int    `json:"year"`
	Version       string `json:"version"`
	PreferredMode string `json:"preferredMode"`
	PixelClockKHz int    `json:"pixelClockKhz"`
	Extensions    uint8  `json:"extensions"`
	Audio         bool   `json:"audio"`
}

type EdidPreflight struct {
	Chip               string `json:"chip"`
	Product            string `json:"product"`
	Supported          bool   `json:"supported"`
	RequiresPowerCycle bool   `json:"requiresPowerCycle"`
	ToolAvailable      bool   `json:"toolAvailable"`
	Reason             string `json:"reason,omitempty"`
}

type EdidBackup struct {
	ID        string `json:"id"`
	SHA256    string `json:"sha256"`
	AppliedAt string `json:"appliedAt"`
	Size      int    `json:"size"`
}

type GetEdidRsp struct {
	Active              *EdidSummary  `json:"active"`
	Source              string        `json:"source,omitempty"`
	AppliedAt           string        `json:"appliedAt,omitempty"`
	UnverifiedSinceBoot bool          `json:"unverifiedSinceBoot"`
	Preflight           EdidPreflight `json:"preflight"`
	Backups             []EdidBackup  `json:"backups"`
	FactoryAvailable    bool          `json:"factoryAvailable"`
}

type EdidProfile struct {
	ID            string `json:"id"`
	Manufacturer  string `json:"manufacturer"`
	Model         string `json:"model"`
	PreferredMode string `json:"preferredMode"`
	Source        string `json:"source"`
}

type GetEdidProfilesRsp struct {
	Profiles []EdidProfile `json:"profiles"`
}

type DecodeEdidRsp struct {
	Summary EdidSummary `json:"summary"`
	Detail  any         `json:"detail"`
}

type ApplyEdidRsp struct {
	State              EdidState    `json:"state"`
	Verified           bool         `json:"verified"`
	Retryable          bool         `json:"retryable"`
	RequiresPowerCycle bool         `json:"requiresPowerCycle"`
	Message            string       `json:"message"`
	Summary            *EdidSummary `json:"summary,omitempty"`
	WrittenHex         string       `json:"writtenHex,omitempty"` // needs_recovery only
	ReadHex            string       `json:"readHex,omitempty"`    // needs_recovery only
}
