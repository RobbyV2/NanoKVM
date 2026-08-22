package presentation

import (
	"fmt"
	"strconv"
)

type OpKind int

const (
	OpMkdir OpKind = iota
	OpWrite
	OpSymlink
	OpUnlink
	OpBind
	OpUnbind
	OpOTGRole
)

var opKindNames = [...]string{"mkdir", "write", "symlink", "unlink", "bind", "unbind", "otg"}

func (k OpKind) String() string {
	if k < 0 || int(k) >= len(opKindNames) {
		return "op(" + strconv.Itoa(int(k)) + ")"
	}
	return opKindNames[k]
}

type Op struct {
	Kind   OpKind `json:"kind"`
	Path   string `json:"path,omitempty"`
	Target string `json:"target,omitempty"`
	Data   []byte `json:"data,omitempty"`
}

type Plan struct {
	Ops       []Op        `json:"ops"`
	Endpoints EndpointUse `json:"endpoints"`
	Profile   string      `json:"profile"`
}

const (
	OTGRoleDevice = "device"
	OTGRoleHost   = "host"

	functionsDir = "functions"
	stringsDir   = "strings/0x409"
	configName   = "c.1"
	osDescDir    = "os_desc"
)

// Phase A reproduces S03usbdev/S03usbhid write for write, including the writes
// the kernel rejects (H8) and the os_desc block the script applies from a
// sentinel rather than from the function list (H10, H11).
func Compile(p Profile, caps CapabilityTable) (Plan, error) {
	p.Normalize()
	if err := p.Validate(); err != nil {
		return Plan{}, fmt.Errorf("compile %s: %w", p.Name, err)
	}
	endpoints, err := AccountEndpoints(p.Functions, caps)
	if err != nil {
		return Plan{}, fmt.Errorf("compile %s: %w", p.Name, err)
	}

	c := &compiler{}
	c.device(p.Device)
	c.config(p.Config)

	osDescDone := false
	for _, function := range p.Functions {
		if !osDescDone && !function.Kind.isNet() {
			c.osDesc(p.OSDesc)
			osDescDone = true
		}
		c.function(function)
	}
	if !osDescDone {
		c.osDesc(p.OSDesc)
	}

	c.ops = append(c.ops,
		Op{Kind: OpBind, Path: udcAttr},
		Op{Kind: OpOTGRole, Data: []byte(OTGRoleDevice)},
	)

	for _, op := range c.ops {
		if op.Kind == OpOTGRole {
			continue
		}
		if err := validateRel(op.Path); err != nil {
			return Plan{}, fmt.Errorf("compile %s: %s %s: %w", p.Name, op.Kind, op.Path, err)
		}
	}
	return Plan{Ops: c.ops, Endpoints: endpoints, Profile: p.Name}, nil
}

func (k FunctionKind) isNet() bool {
	return k == FunctionNCM || k == FunctionRNDIS
}

type compiler struct {
	ops []Op
}

func (c *compiler) mkdir(path string) {
	c.ops = append(c.ops, Op{Kind: OpMkdir, Path: path})
}

func (c *compiler) raw(path string, data []byte) {
	c.ops = append(c.ops, Op{Kind: OpWrite, Path: path, Data: data})
}

func (c *compiler) write(path, value string) {
	c.raw(path, []byte(value+"\n"))
}

func (c *compiler) link(path, target string) {
	c.ops = append(c.ops, Op{Kind: OpSymlink, Path: path, Target: target})
}

func (c *compiler) device(d Device) {
	if d.BCDUSB != nil {
		c.write("bcdUSB", *d.BCDUSB)
	}
	if d.BCDDevice != nil {
		c.write("bcdDevice", *d.BCDDevice)
	}
	c.write("idVendor", d.VendorID)
	c.write("idProduct", d.ProductID)
	if d.Class != nil {
		c.write("bDeviceClass", hexByte(*d.Class))
	}
	if d.SubClass != nil {
		c.write("bDeviceSubClass", hexByte(*d.SubClass))
	}
	if d.Protocol != nil {
		c.write("bDeviceProtocol", hexByte(*d.Protocol))
	}

	c.mkdir(stringsDir)
	if d.Serial != nil {
		c.write(stringsDir+"/serialnumber", *d.Serial)
	}
	c.write(stringsDir+"/manufacturer", d.Manufacturer)
	c.write(stringsDir+"/product", d.Product)
}

func (c *compiler) config(cfg ConfigDesc) {
	c.mkdir(configPrefix)
	c.write(configPrefix+"/bmAttributes", hexByte(cfg.BMAttributes))
	c.write(configPrefix+"/MaxPower", strconv.Itoa(int(cfg.MaxPower)))
	c.mkdir(configPrefix + "/" + stringsDir)
	c.write(configPrefix+"/"+stringsDir+"/configuration", cfg.Configuration)
}

func (c *compiler) osDesc(o *OSDesc) {
	if o == nil {
		return
	}
	c.write(osDescDir+"/use", "1")
	c.write(osDescDir+"/b_vendor_code", o.VendorCode)
	c.write(osDescDir+"/qw_sign", o.QwSign)
	c.link(osDescDir+"/"+configName, configPrefix)
}

func (c *compiler) function(f Function) {
	name := string(f.Kind) + "." + f.Instance
	dir := functionsDir + "/" + name

	switch f.Kind {
	case FunctionHID:
		c.hid(dir, name, *f.HID)
	case FunctionNCM, FunctionRNDIS:
		c.net(dir, name, f.Kind, *f.Net)
	case FunctionMassStorage:
		c.storage(dir, name, *f.Storage)
	}
}

// H2, ordering constraint 2: every f_hid option store returns -EBUSY once the
// function is linked, and a reordered link enumerates a 64-byte device carrying
// the kernel's default descriptor with no error anywhere. The link stays last.
func (c *compiler) hid(dir, name string, h HIDFunction) {
	c.mkdir(dir)
	if h.SubClass != 0 {
		c.write(dir+"/subclass", strconv.Itoa(int(h.SubClass)))
	}
	if h.WakeupOnWrite {
		c.write(dir+"/wakeup_on_write", "1")
	}
	c.write(dir+"/protocol", strconv.Itoa(int(h.Protocol)))
	c.write(dir+"/report_length", strconv.Itoa(int(h.ReportLength)))
	c.raw(dir+"/report_desc", h.ReportDesc)
	c.link(configPrefix+"/"+name, dir)
}

func (c *compiler) net(dir, name string, kind FunctionKind, n NetFunction) {
	c.mkdir(dir)
	if n.DevAddr != nil {
		c.write(dir+"/dev_addr", *n.DevAddr)
	}
	if n.HostAddr != nil {
		c.write(dir+"/host_addr", *n.HostAddr)
	}

	// H8: S03usbdev:66-68 writes e0/01/03 unprefixed, f_rndis parses them with
	// kstrtou8(page, 0, ...) which rejects unprefixed hex, and the values equal
	// the RNDIS IAD defaults, so the -EINVAL leaves no trace. Phase A emits the
	// rejected bytes verbatim; prefixing them is a Phase B behaviour change.
	if n.Class != nil {
		c.write(dir+"/class", bareByte(*n.Class))
	}
	if n.SubClass != nil {
		c.write(dir+"/subclass", bareByte(*n.SubClass))
	}
	if n.Protocol != nil {
		c.write(dir+"/protocol", bareByte(*n.Protocol))
	}

	iface := dir + "/" + osDescDir + "/interface." + string(kind)
	c.write(iface+"/compatible_id", n.CompatibleID)
	if n.SubCompatibleID != "" {
		c.write(iface+"/sub_compatible_id", n.SubCompatibleID)
	}
	c.link(configPrefix+"/"+name, dir)
}

// Ordering constraint 3: the LUN attributes carry no refcnt check, so
// mass_storage is the one function whose link may precede its attributes, and
// S03usbdev:135 relies on that. H7: ro and cdrom are written only when
// /boot/usb.disk0.ro exists, leaving whatever storage/image.go last set.
func (c *compiler) storage(dir, name string, s StorageFunction) {
	c.mkdir(dir)
	c.link(configPrefix+"/"+name, dir)

	lun := dir + "/" + lunDir
	c.write(lun+"/removable", boolBit(s.Removable))
	if s.ReadOnly {
		c.write(lun+"/ro", "1")
		c.write(lun+"/cdrom", "0")
	}
	if s.InquiryString != "" {
		c.write(lun+"/inquiry_string", s.InquiryString)
	}
	c.write(lun+"/file", s.File)
}

func hexByte(v uint8) string {
	return fmt.Sprintf("0x%02X", v)
}

func bareByte(v uint8) string {
	return fmt.Sprintf("%02x", v)
}

func boolBit(v bool) string {
	if v {
		return "1"
	}
	return "0"
}
