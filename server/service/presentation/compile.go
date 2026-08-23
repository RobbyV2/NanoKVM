package presentation

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
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
	OpRmdir
)

var opKindNames = [...]string{"mkdir", "write", "symlink", "unlink", "bind", "unbind", "otg", "rmdir"}

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
	Ops       []Op           `json:"ops"`
	Profile   string         `json:"profile"`
	Endpoints EndpointUse    `json:"endpoints"`
	FIFOs     FIFOAssignment `json:"fifos,omitempty"`
}

const (
	OTGRoleDevice = "device"

	functionsDir = "functions"
	stringsDir   = "strings/0x409"
	configName   = "c.1"
	osDescDir    = "os_desc"
)

func Compile(p Profile, caps CapabilityTable) (Plan, error) {
	p.Normalize()
	if err := p.Validate(); err != nil {
		return Plan{}, fmt.Errorf("compile %s: %w", p.Name, err)
	}
	endpoints, err := AccountEndpoints(p.Functions, caps)
	if err != nil {
		return Plan{}, fmt.Errorf("compile %s: %w", p.Name, err)
	}
	fifos, err := SeatFIFOs(p.Functions, caps)
	if err != nil {
		return Plan{}, fmt.Errorf("compile %s: %w", p.Name, err)
	}

	c := &compiler{}
	c.device(p.Device)
	c.config(p.Config)

	osDesc := p.osDesc()
	osDescDone := false
	for _, function := range p.Functions {
		if !osDescDone && !function.Kind.isNet() {
			c.osDesc(osDesc)
			osDescDone = true
		}
		c.function(function)
	}
	if !osDescDone {
		c.osDesc(osDesc)
	}

	c.ops = append(c.ops,
		Op{Kind: OpBind, Path: udcAttr},
		Op{Kind: OpOTGRole, Data: []byte(OTGRoleDevice)},
	)

	for _, op := range c.ops {
		if op.Kind == OpOTGRole {
			continue
		}
		var err error
		switch op.Kind {
		case OpUnlink:
			err = validateRemove(op.Path)
		case OpRmdir:
			err = validateRmdir(op.Path)
		default:
			err = validateRel(op.Path)
		}
		if err != nil {
			return Plan{}, fmt.Errorf("compile %s: %s %s: %w", p.Name, op.Kind, op.Path, err)
		}
	}
	return Plan{Ops: c.ops, Profile: p.Name, Endpoints: endpoints, FIFOs: fifos}, nil
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

func (c *compiler) unlink(path string) {
	c.ops = append(c.ops, Op{Kind: OpUnlink, Path: path})
}

func (c *compiler) rmdir(path string) {
	c.ops = append(c.ops, Op{Kind: OpRmdir, Path: path})
}

func (c *compiler) device(d Device) {
	if d.BCDUSB != nil {
		c.write("bcdUSB", *d.BCDUSB)
	}
	if d.BCDDevice != nil {
		c.write(attrBCDDevice, *d.BCDDevice)
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

// H10, H11: the script triggers the MS-OS block off a /boot sentinel rather than
// off a function existing, and never clears os_desc/use, so a gadget that once
// had RNDIS keeps answering the 0xEE string request. The block follows the
// functions actually in the plan, in both directions.
func (p Profile) osDesc() *OSDesc {
	if !slices.ContainsFunc(p.Functions, func(f Function) bool { return f.Kind.isNet() }) {
		return nil
	}
	if p.OSDesc != nil {
		return p.OSDesc
	}
	return MSOSDesc()
}

func (c *compiler) osDesc(o *OSDesc) {
	if o == nil {
		c.write(osDescDir+"/use", "0")
		c.ops = append(c.ops, Op{Kind: OpUnlink, Path: osDescDir + "/" + configName})
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
	case FunctionFFS:
		c.mkdir(dir)
		c.link(configPrefix+"/"+name, dir)
	case FunctionUVC:
		c.uvc(dir, name, *f.Video)
	case FunctionUAC2:
		c.uac2(dir, name, *f.Audio)
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

	// H8: kstrtou8(page, 0, ...) reads the script's "01" and "03" as octal, so
	// those two land as 1 and 3, and rejects "e0" outright. Everything the
	// profile asks for is written 0x-prefixed, and the script's dead class write
	// is not in the profile at all.
	if n.Class != nil {
		c.write(dir+"/class", hexByte(*n.Class))
	}
	if n.SubClass != nil {
		c.write(dir+"/subclass", hexByte(*n.SubClass))
	}
	if n.Protocol != nil {
		c.write(dir+"/protocol", hexByte(*n.Protocol))
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

func (c *compiler) uvc(dir, name string, video VideoFunction) {
	for _, speed := range [...]string{"fs", "hs", "ss"} {
		c.unlink(dir + "/streaming/class/" + speed + "/h")
	}
	c.unlink(dir + "/streaming/header/h/m")
	for _, size := range [...]string{"1280x720", "640x480", "320x240", "160x120"} {
		c.rmdir(dir + "/streaming/mjpeg/m/" + size)
	}
	c.rmdir(dir + "/streaming/mjpeg/m")
	c.rmdir(dir + "/streaming/header/h")
	for _, speed := range [...]string{"fs", "ss"} {
		c.unlink(dir + "/control/class/" + speed + "/h")
	}
	c.rmdir(dir + "/control/header/h")
	c.mkdir(dir)
	c.write(dir+"/streaming_interval", strconv.Itoa(int(video.StreamingInterval)))
	c.write(dir+"/streaming_maxpacket", strconv.Itoa(int(video.StreamingMaxPacket)))
	c.write(dir+"/streaming_maxburst", strconv.Itoa(int(video.StreamingMaxBurst)))

	controlHeader := dir + "/control/header/h"
	c.mkdir(controlHeader)
	c.link(dir+"/control/class/fs/h", controlHeader)

	format := dir + "/streaming/mjpeg/m"
	c.mkdir(format)
	for _, frame := range video.Formats[0].Frames {
		frameDir := format + "/" + strconv.Itoa(int(frame.Width)) + "x" + strconv.Itoa(int(frame.Height))
		c.mkdir(frameDir)
		c.write(frameDir+"/wWidth", strconv.Itoa(int(frame.Width)))
		c.write(frameDir+"/wHeight", strconv.Itoa(int(frame.Height)))
		minRate, maxRate := videoRates(video.StreamingMaxPacket, frame.Intervals)
		c.write(frameDir+"/dwMinBitRate", strconv.FormatUint(uint64(minRate), 10))
		c.write(frameDir+"/dwMaxBitRate", strconv.FormatUint(uint64(maxRate), 10))
		bufferSize := uint64(frame.Width) * uint64(frame.Height) * 2
		c.write(frameDir+"/dwMaxVideoFrameBufferSize", strconv.FormatUint(bufferSize, 10))
		c.write(frameDir+"/dwDefaultFrameInterval", strconv.FormatUint(uint64(frame.Intervals[0]), 10))
		values := make([]string, len(frame.Intervals))
		for i, interval := range frame.Intervals {
			values[i] = strconv.FormatUint(uint64(interval), 10)
		}
		c.raw(frameDir+"/dwFrameInterval", []byte(strings.Join(values, "\n")+"\n"))
	}

	streamHeader := dir + "/streaming/header/h"
	c.mkdir(streamHeader)
	c.link(streamHeader+"/m", format)
	for _, speed := range [...]string{"fs", "hs"} {
		c.link(dir+"/streaming/class/"+speed+"/h", streamHeader)
	}
	c.link(configPrefix+"/"+name, dir)
}

func videoRates(packet uint16, intervals []uint32) (uint32, uint32) {
	minFPS, maxFPS := uint32(^uint32(0)), uint32(0)
	for _, interval := range intervals {
		fps := uint32(10_000_000 / interval)
		if fps < minFPS {
			minFPS = fps
		}
		if fps > maxFPS {
			maxFPS = fps
		}
	}
	capacity := uint64(packet) * 8000 * 8
	if minFPS == maxFPS {
		return uint32(capacity), uint32(capacity)
	}
	return uint32(capacity * uint64(minFPS) / uint64(maxFPS)), uint32(capacity)
}

func (c *compiler) uac2(dir, name string, audio AudioFunction) {
	c.mkdir(dir)
	attributes := []struct {
		name  string
		value uint32
	}{
		{"p_chmask", audio.PChannelMask},
		{"p_srate", audio.PSampleRate},
		{"p_ssize", uint32(audio.PSampleSize)},
		{"c_chmask", audio.CChannelMask},
		{"c_srate", audio.CSampleRate},
		{"c_ssize", uint32(audio.CSampleSize)},
		{"req_number", uint32(audio.RequestNumber)},
	}
	for _, attribute := range attributes {
		c.write(dir+"/"+attribute.name, strconv.FormatUint(uint64(attribute.value), 10))
	}
	c.link(configPrefix+"/"+name, dir)
}

func hexByte(v uint8) string {
	return fmt.Sprintf("0x%02X", v)
}

func boolBit(v bool) string {
	if v {
		return "1"
	}
	return "0"
}
