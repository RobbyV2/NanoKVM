# Media and device sources

Status: design only. No code written, nothing has run on hardware, and the program is blocked on hardware evidence nobody has gathered. See "Gating".
Date: 2026-08-22.

## Goal

Let several clients supply media and USB devices to one controlled host at the same time, with no relationship between who supplies a stream and who holds keyboard and mouse control. The scenario that drives the design is a phone that opens the NanoKVM web UI, grants its camera and its microphone and forwards both, while a laptop separately drives the desktop through the same web UI, and a second laptop shares its own camera. The controlled host sees two cameras and one microphone in its own device list, next to the keyboard and the pointer that were always there.

Program 4 builds on `server/service/presentation`, which exists today. It adds two function kinds to the profile schema, a registry of sources, sinks and bindings, a media transport, and the demand-driven generators that keep the USB functions standards compliant while nothing is bound. It changes nothing about the HID path, nothing about the shape of the endpoint allocator, and nothing about the apply transaction.

Program 4 supersedes sections 1.4, 2.7 and 7 of the original plan wherever they disagree, and section 12's `POST /api/presentation/media/lease` in particular.

## The hardware ceiling

This comes first because it decides the shape of everything below.

`build/boards/default/dts/sg200x/soph_base.dtsi` in the LicheeRV-Nano-Build tree declares `g-rx-fifo-size = <536>`, `g-np-tx-fifo-size = <32>` and `g-tx-fifo-size = <768 512 512 384 128 128>`. That is six dedicated IN FIFOs plus EP0, and 536 + 32 + 768 + 512 + 512 + 384 + 128 + 128 = 3000 words of controller RAM allocated in total.

Each UVC function costs one isochronous IN for video, each UAC2 capture function one isochronous IN for audio. The three HID functions hold three interrupt INs and are retained in every configuration that keeps a working console. One camera plus one microphone plus HID reaches five of the six. Two cameras plus two microphones plus HID is seven and cannot exist. The realistic ceilings, stated concretely: two video streams with nothing else optional enabled, or one camera plus one microphone plus HID plus mass storage, or one camera plus HID plus a network function. The existing `staticV0` table at `capability.go:39-52` gives `ncm` and `rndis` two IN endpoints each, one bulk and one interrupt notify, so a network function and a microphone do not fit together alongside a camera.

| Configuration | IN if a UVC function costs 1 | IN if it costs 2 |
|---|---|---|
| HID | 3 | 3 |
| HID + 1 camera | 4 | 5 |
| HID + 1 camera + 1 microphone | 5 | 6 |
| HID + 1 camera + 1 microphone + storage | 6 | refused |
| HID + 1 camera + network | 6 | refused |
| HID + 2 cameras | 5 | refused |
| HID + 2 cameras + 1 microphone | 6 | refused |
| HID + 2 cameras + 2 microphones | refused | refused |

The right-hand column exists because `f_uvc` allocates a control status interrupt IN in `uvc_function_bind` ahead of the video endpoint, unconditionally in the 5.10 sources. If that endpoint is really allocated on this UDC then a camera costs two IN endpoints, one camera plus one microphone plus HID exactly fills all six, and every multi-camera configuration is dead before the FIFO question is asked. Which column is true is the first number the hardware run has to return.

**Decided: the right-hand column.** `staticV1` at `capability.go:40-55` spends two IN endpoints on `uvc` with `INPackets` `{16, 768}` — the control status interrupt first, then the video stream, in the order `uvc_function_bind` allocates them. Every row of the table above that says "refused" in the right-hand column is refused by `AccountEndpoints` today, at compile time, before anything reaches the kernel. The consequence worth stating plainly: **two cameras cannot coexist with the three HID functions on this silicon**, and since `Plan.Outcome` keeps all three `hid.*` links up across every apply, because a gadget that drops HID leaves a wedged host with nothing left to drive it, that means multi-camera does not exist on a NanoKVM at all. One camera plus one microphone plus HID fills all six IN endpoints exactly and is the ceiling.

This is read from the 5.10 sources, not measured. Adopting it is the conservative direction — it refuses configurations that might turn out to bind — so a wrong reading costs a refused profile, never a gadget that fails to enumerate. Gating steps 2 and 3 still owe the `lsusb -v` transcript.

### The FIFOs are not interchangeable

One FIFO is 768 words deep, two are 512, one is 384 and two are 128. dwc2 in dedicated-FIFO mode assigns one to each IN endpoint at enable time in `dwc2_hsotg_ep_enable`, scanning `fifo_map` for a free FIFO deep enough for the endpoint and taking the smallest that qualifies. Best fit, so the deep FIFO survives for the endpoint that needs it as long as the small endpoints are enabled first, and the interface order that `Profile.Functions` already pins at `profile.go:87-95` is also the order in which endpoints are enabled.

The comparison deserves care. The size the code demands is `maxpacket * mc` in bytes, and the value it compares against is the depth field of `DPTXFSIZN(i)` in 32-bit words, so a FIFO of depth 768 accepts an endpoint whose `wMaxPacketSize` is at most 768 bytes and holds four times that many bytes of RAM. On this device tree that caps an isochronous IN at 768 bytes per microframe with no multiplier, which rules out the 1024-byte high-bandwidth isochronous packets a UVC gadget normally asks for. `streaming_maxpacket` is therefore a validated profile field rather than a constant, and `streaming_maxburst` stays 0 because high speed has no bursts. Read from the 5.10 sources, unverified on the device.

The allocator assigns depths to functions rather than counting endpoints. It simulates the best fit over the IN endpoints in link order and records which depth each stream lands on, and the recorded depth is what caps the sustainable bitrate of the second camera. For the standard profile plus one camera and one microphone the simulation gives:

| Function | Endpoint | wMaxPacketSize | FIFO |
|---|---|---|---|
| `hid.GS0` | interrupt IN | 8 | 128 |
| `hid.GS1` | interrupt IN | 4 | 128 |
| `hid.GS2` | interrupt IN | 6 | 384 |
| `uvc.cam0` | isochronous IN | 768 | 768 |
| `uac2.mic0` | isochronous IN | 96 | 512 |
| free | | | 512 |

The last 512 seats a mass storage bulk IN at exactly its 512-byte high-speed maximum and nothing larger. Add a second camera in place of storage and it takes that 512, so `uvc.cam1` runs at two thirds of `uvc.cam0`'s packet size and around two thirds of its ceiling bitrate, which the UI states rather than hides.

Failure is late and nearly silent. Isochronous endpoints live in altsetting 1 and are enabled at SET_INTERFACE, so the FIFO is claimed when the host starts streaming and not when the gadget binds. Two declared cameras both enumerate, the first STREAMON succeeds, and the second returns `-ENOMEM` from `set_alt`, which the host sees as a stalled SET_INTERFACE and reports as a camera that will not start. On the NanoKVM side the only trace is a `dwc2` line in `dmesg`. That timing is the reason slots are validated at apply against the worst case of every declared slot streaming at once.

### One gadget, one address

The composite gadget carries one `idVendor` and one `idProduct`, so two cameras are two VideoControl and VideoStreaming interface pairs under a single USB device, each pair joined by an interface association descriptor. Every operating system lists two cameras and both carry the device's identity strings. The names come from each function's `function_name` attribute, fixed at apply, so the host shows the slot rather than the person, and the mapping from `NanoKVM Camera 1` to whoever is supplying it lives in the binding table in the web UI.

Separate per-camera vendor identity needs a real hub. Section 2.12 of the plan settled that: one DWC2 UDC exposes one USB address, common UDCs cannot emulate hubs, and a hub descriptor whose downstream ports cannot enumerate is not shipped. A topology abstraction stays in the design for hardware that has a real hub, and nothing in Program 4 depends on it.

## Why the single-lease model fails

The plan's model is one media lease per device, held by default by whoever holds active KVM control, revocable by an administrator. Six ways that fails.

The driving scenario has two clients and one controller. The phone supplies the camera and the microphone and holds no control at all, so a lease that follows control can never be held by the phone.

Control changes hands, and under the plan's rule the media lease moves with it, so the camera stream drops in the middle of whatever it was showing because somebody else picked up the keyboard. Nothing about a keyboard handover implies anything about a camera.

Camera and microphone are separate physical devices, frequently on separate machines. One lease per device makes "phone camera with laptop microphone" unrepresentable, and two leases per device is the general model with an arbitrary limit written into it.

USB device forwarding needs identical bookkeeping: an offer, a slot, a claim, an owner, an expiry, a takeover path. A media-only lease means writing all of it twice and having the two copies disagree in the third month.

A single lease has no place to record demand. The lease exists whether or not the host has the camera open, so the browser has no signal telling it when to open a real camera, and the plan's own section 7.5 has to bolt demand tracking on beside the lease.

A lease attached to the controller is invisible to everyone else sharing the device. The browser's own recording indicator tells the person doing the sharing and nobody else, so a camera forwarded from somebody's phone is a camera the other people looking at that desktop cannot see.

## Decisions

| # | Decision |
|---|---|
| D1 | Three primitives, with no camera special case. A `Source` is a client offering one or more streams, a browser tab or a native agent. A `Sink` is a USB function slot compiled into the active profile, with its endpoint and FIFO cost already known. A `Binding` joins one source stream to one sink for as long as both live. USB device forwarding uses the same three, so nothing in the model is media specific. |
| D2 | Slots are declared in the profile by an administrator. The active profile states how many camera and microphone slots exist, `Compile` validates that against the capability table at apply time and refuses with a named reason, so the failure happens when somebody configures the device rather than when somebody plugs in a phone. |
| D3 | Sources claim free slots first come, first served. When every slot is taken a new source is refused and told which slot is occupied and by whom, and may request a takeover. Nothing already streaming is interrupted by somebody merely connecting. |
| D4 | Media supply is decoupled from KVM control. Holding control confers no claim on a slot and supplying a stream confers no control. Any authenticated user may supply media once an administrator has declared the slots. This supersedes plan sections 1.4 and 2.7. |
| D5 | The binding table is visible to every authenticated user: which slots exist, which are filled, by whom, and whether the host currently has them open, streamed over the events socket. |
| D6 | Leases are per binding rather than one per device. Each carries its own owner, its own resume token for a page refresh, and its own expiry. |
| D7 | Streams start on host demand and stop on a grace interval after demand ceases. The browser does not open a real camera because a slot exists. |
| D8 | Black frames and digital silence are generated on the device, next to the V4L2 and ALSA nodes, so the host sees a legal stream during the milliseconds when no client is attached and during the minutes when none is. |
| D9 | Slots are entries in `Profile.Functions` and nothing else. No parallel media configuration file, no second store, no second notion of what the gadget is. The existing endpoint allocator already accounts per function. |
| D10 | The media path runs inside the server process. Program 1's D1 argument holds unchanged: `S95nanokvm`'s `stop_process` is `killall -INT NanoKVM-Server` by name and its ordering against `kvm_system` is load-bearing, so a second long-lived process has to be added to that script, to `system_init.cpp`'s copy list and to the OTA install path or a restart orphans it. |
| D11 | The capability table gains `uvc` and `uac2` entries and a FIFO depth list. The allocator gains FIFO feasibility. It still refuses plans and still never sizes them. |
| D12 | A binding never outlives its sink. A reapply that removes a slot, or changes any field that changes the descriptors, terminates the binding with a named reason and invalidates its resume token. |
| D13 | Video is MJPEG in v1, with one shipped frame list identical across slots: 1280x720, 640x480, 320x240 and 160x120 at 30 and 15 fps. Audio is 48 kHz, 16-bit, mono capture with no playback path. |

## The three primitives

Two new packages. `server/service/sources` holds the registry, the leases, the events socket and the handlers, and imports `presentation` for the function types and the manager. `server/service/media` holds the V4L2 and ALSA plumbing, the demand watcher and the generators, and implements the backend interface the registry drives. Neither package is imported by `presentation`, so the dependency runs one way and Program 1 stays testable without either.

```go
// sources/types.go
type Kind string

const (
    KindCamera     Kind = "camera"
    KindMicrophone Kind = "microphone"
    KindUSBDevice  Kind = "usb_device"
)

// A Source is one connected client offering streams. Its identity is the
// authenticated principal from middleware.CurrentPrincipal, so a phone and a
// laptop logged in as the same user are two sources with one owner.
type Source struct {
    ID          string    `json:"id"`     // server assigned, opaque, per connection
    Owner       string    `json:"owner"`  // authn username
    Agent       string    `json:"agent"`  // "browser" | "native"
    Label       string    `json:"label"`  // client supplied, sanitized, length capped
    Streams     []Stream  `json:"streams"`
    ConnectedAt time.Time `json:"connected_at"`
}

type Stream struct {
    ID      string   `json:"id"`
    Kind    Kind     `json:"kind"`
    Formats []Format `json:"formats"` // what this client can actually produce
}

// A Sink is a compiled USB function slot. Everything in it is derived from the
// active profile plus the capability table, so a sink exists only after an
// apply that bound.
type Sink struct {
    ID        string                    `json:"id"`    // "uvc.cam0", the function name
    Kind      Kind                      `json:"kind"`
    Label     string                    `json:"label"` // the function_name the host shows
    Slot      int                       `json:"slot"`
    Function  presentation.Function     `json:"-"`
    Endpoints presentation.EndpointUse  `json:"endpoints"`
    FIFOWords int                       `json:"fifo_words"` // assigned by the simulation
    MaxPacket int                       `json:"max_packet"`
    Formats   []Format                  `json:"formats"`
    Node      string                    `json:"-"` // /dev/videoN or hw:CARD, re-resolved per bind
}

type Binding struct {
    SinkID    string    `json:"sink_id"`
    SourceID  string    `json:"source_id"`
    StreamID  string    `json:"stream_id"`
    Owner     string    `json:"owner"`
    State     State     `json:"state"` // claimed | streaming | orphaned | suspended
    StartedAt time.Time `json:"started_at"`
    lease     Lease
}

type Lease struct {
    Token     [32]byte  // returned once at claim, never in a snapshot
    Owner     string
    ExpiresAt time.Time // set when the source disconnects, zero while connected
}
```

The registry owns all three maps under one mutex, and at most one binding exists per sink, keyed by sink id.

```go
// sources/registry.go
type Registry struct {
    mu       sync.Mutex
    sinks    map[string]*Sink
    sources  map[string]*Source
    bindings map[string]*Binding // key is sink id
    subs     map[uint64]chan Event
    backend  Backend
    now      func() time.Time // injected, so expiry is testable without sleeping
}

func (r *Registry) Claim(principal middleware.Principal, sourceID, streamID, sinkID string) (Lease, error)
func (r *Registry) Release(principal middleware.Principal, sinkID string) error
func (r *Registry) RequestTakeover(principal middleware.Principal, sinkID, reason string) (string, error)
func (r *Registry) ResolveTakeover(principal middleware.Principal, requestID string, grant bool) error
func (r *Registry) Resume(principal middleware.Principal, sinkID string, token []byte, sourceID string) error
func (r *Registry) Sinks() []Sink
func (r *Registry) Subscribe() (<-chan Event, func())
```

The backend interface is the boundary between bookkeeping and hardware, and the registry's tests use a fake that never opens a device node.

```go
// sources/backend.go
type Backend interface {
    Discover(profile presentation.Profile, snapshot presentation.Snapshot) ([]Sink, error)
    Attach(ctx context.Context, sinkID string, stream StreamReader) error
    Detach(sinkID string) error
    Demand(sinkID string) Demand
    Events() <-chan BackendEvent // demand edges, node loss, encode errors
}

type Demand struct {
    Streaming bool   `json:"streaming"`
    Width     int    `json:"width,omitempty"`
    Height    int    `json:"height,omitempty"`
    FPS       int    `json:"fps,omitempty"`
    Since     time.Time `json:"since,omitempty"`
}
```

## Slots in the profile

Two kinds join the four at `profile.go:30-35`, and two payloads join the three at `profile.go:116-122`.

```go
const (
    FunctionUVC  FunctionKind = "uvc"
    FunctionUAC2 FunctionKind = "uac2"
)

type Function struct {
    Kind     FunctionKind     `json:"kind"`
    Instance string           `json:"instance"` // "GS0", "usb0", "disk0", "cam0", "mic0"
    HID      *HIDFunction     `json:"hid,omitempty"`
    Net      *NetFunction     `json:"net,omitempty"`
    Storage  *StorageFunction `json:"storage,omitempty"`
    Video    *VideoFunction   `json:"video,omitempty"`
    Audio    *AudioFunction   `json:"audio,omitempty"`
}

type VideoFunction struct {
    FunctionName    string        `json:"function_name"`     // IAD string: "NanoKVM Camera 1"
    Formats         []VideoFormat `json:"formats"`           // mjpeg only in v1
    StreamingMaxPacket uint16     `json:"streaming_maxpacket"` // 256 | 512 | 768
    StreamingMaxBurst  uint8      `json:"streaming_maxburst"`  // 0 at high speed
    StreamingInterval  uint8      `json:"streaming_interval"`  // 1
}

type VideoFormat struct {
    Codec  string       `json:"codec"` // "mjpeg"
    Frames []VideoFrame `json:"frames"`
}

type VideoFrame struct {
    Width     uint16   `json:"width"`
    Height    uint16   `json:"height"`
    Intervals []uint32 `json:"intervals"` // 100 ns units, 333333 = 30 fps, 666666 = 15 fps
}

type AudioFunction struct {
    FunctionName     string `json:"function_name"`
    CaptureChannels  uint32 `json:"c_chmask"` // 1, mono
    CaptureRate      uint32 `json:"c_srate"`  // 48000
    CaptureSampleSize uint8 `json:"c_ssize"`  // 2
    PlaybackChannels uint32 `json:"p_chmask"` // 0, no host-to-gadget path in v1
    RequestNumber    uint8  `json:"req_number"`
}
```

`Function.validate()` at `profile.go:270-293` gains two arms enforcing exactly one payload per kind, as the four existing arms do. The new rules: an instance matching `cam[0-9]` or `mic[0-9]`, numbered contiguously from zero in profile order; at least one format with at least one frame; every frame drawn from the shipped list, so a profile cannot invent a mode nobody tested; every interval in `{333333, 666666}`; `StreamingMaxPacket` in `{256, 512, 768}`; `CaptureRate` 48000, `CaptureSampleSize` 2, `CaptureChannels` 1 and `PlaybackChannels` 0. Whether a given packet size can actually be seated is a whole-profile question and belongs to the allocator, not to `Validate`.

Media slots sort after the HID functions in `Profile.Functions`, which fixes two things at once: `bInterfaceNumber` assignment stays stable for the HID interfaces the host already knows, and the three small interrupt endpoints are enabled before the isochronous ones so best fit hands them the 128 and 384 FIFOs.

### What `Compile` emits for a camera slot

Relative to the gadget root, in order, following the shape of `compile.go:198-211`:

```
mkdir   functions/uvc.cam0
write   functions/uvc.cam0/function_name        "NanoKVM Camera 1"
write   functions/uvc.cam0/streaming_maxpacket  "768"
write   functions/uvc.cam0/streaming_maxburst   "0"
write   functions/uvc.cam0/streaming_interval   "1"
mkdir   functions/uvc.cam0/control/header/h
symlink functions/uvc.cam0/control/class/fs/h -> functions/uvc.cam0/control/header/h
symlink functions/uvc.cam0/control/class/hs/h -> functions/uvc.cam0/control/header/h
mkdir   functions/uvc.cam0/streaming/mjpeg/m
mkdir   functions/uvc.cam0/streaming/mjpeg/m/720p
write   .../720p/wWidth "1280"   wHeight "720"   dwFrameInterval "333333\n666666"
write   .../720p/dwMinBitRate, dwMaxBitRate, dwMaxVideoFrameBufferSize, dwDefaultFrameInterval
mkdir   .../480p, .../240p, .../120p and the same seven writes each
mkdir   functions/uvc.cam0/streaming/header/h
symlink functions/uvc.cam0/streaming/header/h/m   -> functions/uvc.cam0/streaming/mjpeg/m
symlink functions/uvc.cam0/streaming/class/fs/h   -> functions/uvc.cam0/streaming/header/h
symlink functions/uvc.cam0/streaming/class/hs/h   -> functions/uvc.cam0/streaming/header/h
symlink configs/c.1/uvc.cam0                      -> functions/uvc.cam0
```

`dwMaxVideoFrameBufferSize` is computed rather than configured, at width times height times two, which is the ceiling a 4:2:2 uncompressed frame would need and a bound no MJPEG frame of that size exceeds. `dwMinBitRate` and `dwMaxBitRate` are derived from the packet size and the interval, so a host that honours them asks for something the FIFO can carry.

A microphone slot is shorter:

```
mkdir functions/uac2.mic0
write functions/uac2.mic0/c_chmask   "1"
write functions/uac2.mic0/c_srate    "48000"
write functions/uac2.mic0/c_ssize    "2"
write functions/uac2.mic0/p_chmask   "0"
write functions/uac2.mic0/req_number "4"
write functions/uac2.mic0/function_name "NanoKVM Microphone"
symlink configs/c.1/uac2.mic0 -> functions/uac2.mic0
```

`p_chmask=0` is what removes the host-to-gadget playback path, and with it the isochronous OUT and the explicit feedback endpoint. A speaker on the controlled host is a second program and a second endpoint pair, and the budget table above says where it would have to come from.

### Ordering and kernel contracts, continuing Program 1's list

11. `f_uvc`'s configfs stores check `opts->refcnt` exactly as `f_hid`'s do, so every attribute and every internal symlink is written before `configs/c.1/uvc.cam0` exists. The link goes last, which is constraint 2 applied to a fifth function kind.
12. The internal link tree needs no change to `Ops`. `ConfigFSOps.Symlink` at `ops_configfs.go:173-187` absolutizes a relative target against the gadget root before `symlinkat`, and `validateRel` at `ops_configfs.go:64-79` already admits any path whose first segment is `functions`, so `functions/uvc.cam0/streaming/class/hs/h -> functions/uvc.cam0/streaming/header/h` is expressible with the code as written.
13. The internal tree is add-only like everything else. `validateRemove` at `ops_configfs.go:82-97` limits `Ops.Remove` to symlinks under `configs/c.1/` and to `os_desc/c.1`, so no op can remove a frame directory or a class link. Taking a camera out of a profile unlinks it from the config and leaves `functions/uvc.cam0` standing, the same treatment `hid.GS*` gets, which is what makes putting it back cheap.
14. `/dev/videoN` and the ALSA card are created at gadget bind rather than at mkdir. `f_uvc` registers the V4L2 device in `uvc_function_bind` and unregisters it in unbind, so every apply destroys and recreates every media node, and with two cameras the two minors can swap places. Nothing may cache a node index across an apply. This is the media analogue of the `/dev/hidgN` invariant with the opposite conclusion, since the HID minors are stable precisely because those functions are never removed, while the video nodes are recreated on every bind whatever anyone does.
15. Resolve nodes through sysfs rather than by index. `/sys/class/video4linux/video*/device` resolves into the gadget function directory, so the mapping from `uvc.cam0` to a node is read back after every bind and asserted before a single buffer is queued, and the audio equivalent is the card whose `/sys/class/sound/card*/device` resolves into `functions/uac2.mic0`. Both readings need confirming on hardware before anything depends on them.
16. Isochronous endpoints live in altsetting 1 and are enabled at SET_INTERFACE, so a FIFO shortage surfaces as a failed STREAMON on the host long after a successful apply, and never as an error from `Apply`.

## Validation at apply

The capability table at `capability.go:22-37` gains two function entries and one field.

```go
type CapabilityTable struct {
    Source          string                        `json:"source"`       // "static-v1" now
    GeneratedAt     time.Time                     `json:"generated_at"`
    MaxInEndpoints  int                           `json:"max_in_endpoints"`
    MaxOutEndpoints int                           `json:"max_out_endpoints"`
    INFifoWords     []int                         `json:"in_fifo_words"` // {768,512,512,384,128,128}
    Functions       map[FunctionKind]FunctionCaps `json:"functions"`
}

type FunctionCaps struct {
    Available  bool            `json:"available"`
    InEPs      int             `json:"in_eps"`
    OutEPs     int             `json:"out_eps"`
    INPackets  []int           `json:"in_packets,omitempty"` // bytes per IN ep, largest first
    Attributes map[string]bool `json:"attributes,omitempty"`
}
```

`staticV0` becomes `staticV1` with `uvc` at one IN and no OUT, `uac2` at one IN and no OUT, `INPackets` filled in for every kind, and the FIFO list copied from the device tree. The name change is what makes every refusal message say which table refused, as Program 1's D7 requires. `static-v0` stays in the file and stays loadable, so a device that never gains media slots keeps the table it was tested against.

`AccountEndpoints` at `endpoint.go:27-52` keeps its signature and its two existing refusals, and gains a third pass that seats packets into FIFOs by the same best fit dwc2 uses, in function order:

```go
var ErrFIFOBudget = errors.New("fifo budget exceeded")

func seatFIFOs(functions []Function, table CapabilityTable) (map[string][]int, error)
```

The refusal, in full, is the string an administrator reads:

```
compile media-2cam: fifo budget exceeded: uvc.cam1 needs a fifo of at least 768 words
for a 768 byte isochronous IN packet, the free fifos are 384 128 128, rejected by
capability table static-v1
```

and the apply handler returns it structured, so the settings page renders the numbers rather than parsing prose:

```json
{"code": -4, "msg": "fifo budget exceeded: ...", "data": {
  "reason": "fifo_budget", "function": "uvc.cam1", "needed_words": 768,
  "free_words": [384, 128, 128], "capabilities": "static-v1"}}
```

The endpoint-count refusal keeps the wording it already has at `endpoint.go:44-50` and gains the same structured envelope with `"reason": "endpoint_budget"`.

Both checks live inside `Compile`, a pure function that touches no filesystem, so the preview endpoint and the apply endpoint return the same refusal, and an administrator adding a second camera learns it will not fit at the moment of typing the number rather than at the moment somebody's phone tries to claim it.

## The binding lifecycle

### Claim

A client opens `/api/sources/ws`, authenticated by the same JWT cookie as `/api/ws` through `middleware.CheckToken()`, with `middleware.CheckWebSocketOrigin` on the upgrade and `middleware.WatchWebSocket` on the connection so a revoked session closes the socket the way it already closes the HID one. The client sends `hello` describing the streams it can produce. The server assigns a source id, replies with the sink table, and the client sends `claim {sink_id, stream_id}`.

`Registry.Claim` takes the mutex, checks the sink exists and is free, writes the binding, mints a 32-byte token from `crypto/rand`, returns it once, and broadcasts `binding_added` before releasing. Concurrent claims serialize on the mutex and exactly one wins, asserted by a `go test -race` case with N goroutines.

Claiming requires authentication and nothing else. No control check, no admin check, no seniority. Holding the keyboard is not a claim on a camera slot and supplying a camera is not a claim on the keyboard.

### Refusal

A claim on an occupied sink is refused with everything the person needs to decide what to do next, and disturbs nothing:

```json
{"type": "claim_refused", "sink_id": "uvc.cam0", "reason": "slot_occupied",
 "owner": "alice", "source_label": "Pixel 8 back camera",
 "since": "2026-08-22T11:04:19Z", "takeover": "allowed"}
```

`takeover` is `allowed` for any authenticated user, and `immediate` when the requester is an administrator.

### Takeover

`takeover_request {sink_id, reason}` creates a pending request, delivers `takeover_requested` to the incumbent's source and to every connected administrator, and starts a 30 second timer. The incumbent's browser shows a prompt naming the requester. An administrator may resolve it without the incumbent. On a grant the registry detaches the incumbent and binds the requester inside one critical section, so nobody else can take the slot in between, and the generator supplies black for the handover. On a denial or a timeout the requester gets `takeover_denied` and may not ask again for that sink for sixty seconds.

### Resume after a refresh

A page refresh closes the socket. The binding does not end. It moves to `orphaned` with `ExpiresAt` set to now plus the lease grace, default sixty seconds, the backend detaches the stream and the generator resumes black or silence within one frame interval, and the slot stays unavailable to everyone else until it expires. The reloaded page presents the token it kept in `sessionStorage` to `POST /api/sources/bindings/:sink/resume`, the registry compares it with `subtle.ConstantTimeCompare` and checks that the presenting principal is the same username, and on a match the binding returns to `claimed` under the new source id. A token presented by a different user is refused whatever it contains.

The browser resumes the media itself only when it can reacquire the same device. It calls `getUserMedia` with the previous exact `deviceId`, and where that device is gone it falls back to the current default and says on screen that the device changed rather than silently sharing a different camera.

### Expiry and other terminations

Every termination carries a reason, and each one is a string the UI can render:

| Reason | Cause |
|---|---|
| `released` | the owner pressed stop sharing |
| `source_closed` | the socket closed and the lease expired without a resume |
| `lease_expired` | orphaned past `ExpiresAt` |
| `taken_over` | a takeover was granted |
| `admin_disconnect` | an administrator disconnected all media |
| `session_revoked` | the owner's session was revoked, seen as the socket closing |
| `slot_removed` | a reapply removed the sink |
| `slot_changed` | a reapply changed the sink's descriptors |
| `apply_failed` | an apply left the gadget unbound, and the next apply did not restore the sink |

Demand disappearing never terminates a binding. The host closing its camera application is not a reason to give somebody else the slot.

### A reapply underneath live bindings

Applies run under the presentation manager's gadget lock at `manager.go:248-252`. The registry subscribes to apply completion and re-reads the sink table from the freshly applied profile plus a snapshot, and reconciles.

A sink whose id is still present with a byte-identical function payload survives. The backend re-resolves its node from sysfs, re-attaches the stream and emits `sink_rebound`. The stream is interrupted for the duration of the unbind either way, because the host's own STREAMON went away with the enumeration and comes back when the host reopens the device.

A sink whose id is gone terminates its binding with `slot_removed`. A sink whose payload changed in any field that changes the descriptors, meaning formats, frames, intervals, packet size or sample rate, terminates with `slot_changed`. Both invalidate the resume token immediately, so a page that refreshes into a different gadget cannot reattach to a slot that no longer means what it meant.

A failed apply leaves the gadget in whatever state the failing op left it, since Program 1's `apply` returns the error without unwinding. Every binding moves to `suspended` rather than terminating, holds its lease, and is either restored by the next successful apply that brings its sink back or terminated with `apply_failed` when the lease grace runs out.

## Demand tracking

The browser must not open a real camera because a slot exists. Demand is read from the host.

For video it is exact. The media backend opens `/dev/videoN`, subscribes to the UVC events, and treats `UVC_EVENT_STREAMON` and `UVC_EVENT_STREAMOFF` as the two edges. `UVC_EVENT_SETUP` and `UVC_EVENT_DATA` carry the probe and commit negotiation, which yields the frame size and interval the host actually chose, and the backend forwards those to the bound client so the browser scales once to the right size instead of scaling to a guess and letting the device rescale.

For audio there is no equivalent event on 5.10. The gadget side of `f_uac2` is an ALSA playback device that the host's capture stream drains, and the underlying requests complete only while the host has altsetting 1 selected, so writes stop being consumed when the host stops capturing. The backend writes with a bounded timeout and treats the first successful drain as the rising edge and a timeout as the falling one. The rate controls later kernels expose are the cleaner mechanism and are not in this tree. Which of the two is used is a hardware question, listed in Gating, and the registry sees only a boolean either way.

On the rising edge the registry emits `demand`, and a bound client starts capture. Where no client is bound, the generator carries the stream and the UI shows the slot as open with nothing behind it, which is the state that prompts somebody to share.

On the falling edge the client stops its tracks after `mediaGrace`, default five seconds, so an application that closes and reopens the camera does not re-prompt, does not restart the laptop's camera indicator, and does not lose a second to `getUserMedia`. Losing authentication, losing the socket, or a granted takeover stops tracks immediately with no grace.

Demand state rides in the events payload, so every authenticated user can see whether the host has the camera open, not merely that somebody is sharing one.

## Black frames and digital silence

A declared slot is a device the host may open at any moment, including when nothing is bound, and it stays standards compliant.

At apply the backend renders one all-black baseline JPEG per declared frame size and caches it, four small buffers per camera slot. While STREAMON is active with nothing attached, the queue loop hands the cached frame to the V4L2 output queue at exactly the negotiated `dwFrameInterval`, driven by a monotonic ticker, so the host sees a legal stream at the rate it asked for rather than a stalled one.

Nothing stale is ever replayed. Every buffer carries the generation counter of the binding it was produced under, the counter increments on every attach and every detach, and the queue loop drops any buffer whose generation is not current. On attach the queue is drained before the first real frame goes in, so no black frame lands after a real one. On detach every unqueued buffer is dropped, the timestamp base resets, and the black generator resumes from the current instant.

Audio works the same way. While the host is capturing with nothing attached, the writer supplies zeroed PCM periods at the UAC2 rate. On attach the jitter buffer starts empty. An underrun inserts a silent period rather than repeating the last one, and an overrun drops or shortens buffered audio rather than growing latency, which is the plan's section 7.4 rule kept intact.

Turning a slot off entirely is a profile change and a re-enumeration, so the host sees the camera disappear rather than watching a permanently black one.

## Events

`GET /api/sources/events` is a websocket open to every authenticated user with no admin gate, which is D5. It sends a snapshot on connect and deltas after.

```json
{
  "type": "snapshot",
  "capabilities": "static-v1",
  "sinks": [
    {
      "id": "uvc.cam0", "kind": "camera", "label": "NanoKVM Camera 1", "slot": 0,
      "fifo_words": 768, "max_packet": 768,
      "formats": [{"codec": "mjpeg", "frames": [
        {"width": 1280, "height": 720, "fps": [30, 15]},
        {"width": 640, "height": 480, "fps": [30, 15]}]}],
      "demand": {"streaming": true, "width": 1280, "height": 720, "fps": 30,
                 "since": "2026-08-22T11:03:58Z"},
      "binding": {"source_id": "src_7f2a", "owner": "alice",
                  "label": "Pixel 8 back camera", "agent": "browser",
                  "state": "streaming", "started_at": "2026-08-22T11:04:19Z"}
    },
    {
      "id": "uvc.cam1", "kind": "camera", "label": "NanoKVM Camera 2", "slot": 1,
      "fifo_words": 512, "max_packet": 512,
      "demand": {"streaming": false}, "binding": null
    },
    {
      "id": "uac2.mic0", "kind": "microphone", "label": "NanoKVM Microphone", "slot": 0,
      "fifo_words": 512, "max_packet": 96,
      "demand": {"streaming": false}, "binding": null
    }
  ],
  "sources": [
    {"id": "src_7f2a", "owner": "alice", "agent": "browser", "label": "Pixel 8",
     "connected_at": "2026-08-22T11:04:11Z",
     "streams": [{"id": "s1", "kind": "camera"}, {"id": "s2", "kind": "microphone"}]}
  ]
}
```

Deltas are `binding_added`, `binding_removed` with its reason, `binding_state`, `demand`, `takeover_requested`, `takeover_resolved`, `sinks_changed` after an apply, and `sink_rebound`. Owner usernames appear in all of them for every authenticated viewer. Resume tokens appear in none of them, and are returned exactly once to the claimant over its own socket.

Nothing in this package broadcasts today. `ws.Manager` at `service/ws/manager.go` keeps a client registry and fans out nothing, and the keyboard LED path is per-client polling, so the subscriber set, the per-subscriber buffered channel and the slow-consumer drop are new code in `sources/events.go`.

## API

```
GET    /api/sources                            authenticated  sinks, sources, bindings
GET    /api/sources/events                     authenticated  websocket, the table above
GET    /api/sources/ws                         authenticated  websocket, control plus media frames
POST   /api/sources/bindings                   authenticated  claim, for native agents
DELETE /api/sources/bindings/:sink             authenticated  owner or admin
POST   /api/sources/bindings/:sink/takeover    authenticated
POST   /api/sources/bindings/:sink/resume      authenticated  {token}
DELETE /api/sources/bindings                   admin          disconnect all media
GET    /api/presentation/profile               admin          the active profile including slots
PUT    /api/presentation/profile/preview       admin          compile only, returns the refusal
PUT    /api/presentation/profile/apply         admin
```

Browser clients claim over the socket they will stream on, so the claim and the transport share a lifetime and a disconnect is unambiguous. The REST claim exists for native agents that stream over their own channel. Both call the same `Registry.Claim`.

Program 1 deliberately added no routes. Program 4 is the first program that needs them, because slot counts have to be edited somewhere, and the three presentation routes above are the minimum for that. `/api/vm/device/virtual` keeps the frozen wire contract Program 1 left it with, including the `media` field that nothing assigns, and gains no knowledge of slots.

Admin gating uses `middleware.RequireRole(authn.RoleAdmin)` as `router/vm.go:16-19` does. Everything a non-admin can reach is a claim, a release, a takeover request, a resume, and reading the table.

## UI

The settings Device tab gains an admin-only block beside `virtual-devices.tsx`: a camera slot count, a microphone slot count, the FIFO depth and packet size each declared slot would get, and a preview that shows the refusal text when the budget says no. Applying warns that the USB device re-enumerates and that live bindings for changed slots end. No fidelity badges and no taxonomy, since Program 1's D11 still holds.

The desktop toolbar gains a Devices popover listing every slot with its demand state, its owner and the source label, fed by the events socket. The buttons are "Share this camera" on a free slot, "Request takeover" on somebody else's, and "Stop sharing" on your own. A slot the host currently has open is marked as such whether or not anybody is sharing into it, because that is the state that tells somebody their camera is wanted.

Permission is requested once per browser origin behind an explicit click, since browsers require a user gesture for a first-time grant. The chosen device ids persist per account and origin in `localStorage`, and a resume reuses the exact `deviceId` where it still enumerates.

Everybody sharing the device sees the same table, including the people supplying nothing. That is the privacy requirement, and the browser's own indicator does not satisfy it.

## Files

| Path | Action |
|---|---|
| `server/service/presentation/profile.go` | modify: two function kinds, two payloads, two validate arms |
| `server/service/presentation/compile.go` | modify: `uvc` and `uac2` arms in `function()`, the internal link tree |
| `server/service/presentation/capability.go` | modify: `staticV1`, `INFifoWords`, `INPackets`, the two new entries |
| `server/service/presentation/endpoint.go` | modify: `seatFIFOs`, `ErrFIFOBudget` |
| `server/service/presentation/snapshot.go` | modify: probe attributes for the two kinds in `functionProbeAttr` at `:30-35` |
| `server/service/sources/*.go` | new: types, registry, leases, events, handlers, fake backend |
| `server/service/media/*.go` | new: V4L2 and ALSA nodes, demand watcher, generators, JPEG validation |
| `server/router/sources.go` | new |
| `server/router/presentation.go` | new: the three admin routes |
| `web/src/pages/desktop/menu/settings/device/media-slots.tsx` | new |
| `web/src/pages/desktop/menu/devices/*.tsx` | new: the toolbar popover |
| `web/src/api/sources.ts` | new |
| `server/service/hid/*` | unchanged |

## Gating

Nothing below has run on a device. `CONFIG_USB_CONFIGFS_F_UVC` was already enabled and `CONFIG_USB_CONFIGFS_F_UAC2` was added at `f74b732` on `nanokvm-custom`, so the kernel side is ready and untested. The endpoint count comes from the device tree, and the real `num_dev_ep` is readable only from the controller's `GHWCFG` registers rather than from anything under `/sys/class/udc/`, so even the six is inference.

The proofs, in the order they have to be gathered:

1. `mkdir functions/uvc.cam0` and `mkdir functions/uac2.mic0` in the scratch `g_probe` gadget succeed, so the modules load. `probeAvailability()` at `ops_configfs.go:281` already does exactly this for the other kinds and extends to these two, never to `hid`, since a `hid` probe consumes a `/dev/hidgN` minor.
2. Two UVC functions instantiate, link into one config, and the gadget binds with the three HID functions still present. If this fails, D2's ceiling drops to one camera and every multi-camera question closes.
3. `lsusb -v` from the attached host: how many endpoints each UVC interface actually has, whether the control status interrupt endpoint is there, and what interface numbering results. This is the column of the ceiling table that is currently unknown.
4. `dmesg` during STREAMON on each camera, capturing the exact `dwc2` string emitted when the FIFO best fit fails, since that string is the only signal a failure produces on this side.
5. Sustained streaming: two cameras at 640x480 30 fps MJPEG for ten minutes, measuring frames the host dropped and CPU on the device.
6. Audio: the card appears, the host records 48 kHz mono, and demand detection through write draining behaves as described or does not.
7. Concurrency: HID keeps working throughout, and `ls -l /dev/hidg*` shows identical minors across ten applies that add and remove media slots.

Steps 2 and 3 have not run. The endpoint cost was settled from the 5.10 sources instead (see the ceiling section), which closes the two-camera question in the refusing direction; what the transcript can still do is relax it.

## Risks

**R4.1 Two UVC functions cannot coexist on this silicon.** Accepted rather than retired. `f_uvc` allocates a control status interrupt IN in `uvc_function_bind`, so two cameras plus HID needs seven IN endpoints against six FIFOs; `staticV1` encodes that and the compiler refuses the configuration. The multi-source premise survives as one camera plus one microphone. The open half is only whether the reading is right, which would relax a limit rather than break one, and it retires when a transcript from a device flashed with the `f74b732` image shows two `uvc.*` functions linked into `configs/c.1` alongside `hid.GS0` through `hid.GS2`, a successful UDC bind, `lsusb -v` from the attached host listing two camera functions with their endpoint counts, and both cameras opened simultaneously by two applications on that host.

**R4.2 A FIFO shortage appears only at STREAMON, as a camera that will not start.** The allocator's seating is a simulation of `dwc2_hsotg_ep_enable` read from source, the byte-against-word comparison means a 768-word FIFO refuses a 1024-byte packet, and the failure path produces one `dev_err` on the device and a stalled SET_INTERFACE on the host. Retires when a deliberately over-committed profile is applied on hardware, the exact `dmesg` line is captured, the host-side symptom is recorded, and `seatFIFOs` is shown to refuse that same profile at compile time with the message quoted above.

**R4.3 A node identity moves under a live binding and frames go to the wrong camera.** `/dev/videoN` and the ALSA card are created at bind and destroyed at unbind, two cameras can swap minors across an apply, and a cached index would silently send one person's camera to the other slot. This is the same class of failure as the `/dev/hidgN` renumbering in Program 1's R1.1 with none of the same protection, since these nodes genuinely are recreated. Retires when the sysfs resolution is proven on hardware for both kinds, and when twenty applies with two cameras bound show the `uvc.cam0` to node mapping re-derived and asserted every time, with a test that fails the attach rather than streaming into an unverified node.

**R4.4 Capture, encode and transport cannot sustain the declared modes.** The browser encodes JPEG in a worker, the frames cross an authenticated websocket, and the device validates and queues them into V4L2 while it is also encoding the HDMI capture stream for the remote desktop. Nothing has measured what two 720p30 MJPEG ingests do to an SG2002 already running the capture pipeline. Retires when the ten-minute two-camera run in Gating step 5 records host-side dropped frames, device CPU and memory, and the desktop stream's own frame rate during the run, and when the shipped frame list is cut to whatever that run sustains.

**R4.5 The registry and the gadget disagree about what exists.** Bindings live in Go memory, sinks come from a profile, and the gadget is configfs state that an apply, a failed apply, a PHY reset, a `SetLUN` or a reboot can move underneath both. A binding pointing at a sink that no longer exists streams into a closed node or, worse, into a reused one. Retires when reconciliation runs after every path that takes the gadget lock, when the fuzz test that interleaves applies with claims and releases under `go test -race` reports no binding whose sink is absent from the post-apply snapshot, and when a mid-apply power cut is shown to leave no resumable token for a slot the reboot did not restore.

**R4.6 A camera is forwarded and the binding table does not show it.** The privacy guarantee is the events socket, so a dropped subscription, a slow consumer that gets dropped silently, or a stale UI turns "everyone can see who is sharing" into "the person sharing can see it in their own browser". Retires when the events socket has an explicit disconnected state that the popover renders instead of showing stale rows, when a dropped slow consumer causes a client-side resubscribe with a fresh snapshot rather than a silent gap, and when a manual test with three browsers confirms all three see a binding added by the fourth within one second.

## Testing

The compiler stays pure and testable on a development machine with no device, so most of what follows runs without hardware.

Golden traces extend the existing matrix in `compile_test.go` with camera and microphone slot counts from zero to two, asserting the exact op order, the exact bytes including trailing newlines, and the exact symlink targets of the internal link tree, since a mistyped target there produces a gadget that binds and streams the kernel's idea of a format rather than the profile's.

The allocator gets a table-driven test with one case per row of the ceiling table, run against both the one-IN and two-IN readings of `f_uvc`, asserting which rows compile and the exact refusal string for the rest, plus a seating test asserting the FIFO assignment table above element by element.

The registry is pure Go over a fake backend, so no test touches V4L2 or ALSA. Cases: N goroutines claiming one sink under `go test -race` with exactly one winner and N-1 refusals naming the winner; takeover granted, denied and timed out; resume with the right token and the right user, the right token and the wrong user, an expired token, and a token for a sink that a reapply removed; expiry driven by the injected clock with no sleeps; and reconciliation across an apply that keeps a sink, changes a sink and removes a sink, asserting the three reasons.

The generation counter gets its own test: attach, detach, attach, asserting that no buffer produced under the first generation is queued after the second begins, which is the mechanical form of "no stale frame after a source disconnects".

On the device, the seven proofs in Gating, plus the ten-apply HID minor check from Program 1's R1.1 rerun with media slots being added and removed, since that is the invariant most likely to be broken by a program that adds functions.

No CI workflow in this repo runs `go test` or `go vet`, so all of it runs locally and in review until that changes.
