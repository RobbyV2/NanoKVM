# Media gadget

- A vendor-5.10 UVC function consumes two IN endpoints: status interrupt, then stream.
- Host microphone audio uses UAC2 `p_*`: USB IN is local ALSA playback. `c_*` is USB OUT.
- A speaker is the same function driven the other way: `c_chmask` is what enables the USB OUT endpoint (`EPOUT_EN`), and on the gadget it appears as an ALSA **capture** substream. The channel masks are the only direction marker; nothing infers direction from an instance name without checking them.
- A speaker spends an OUT endpoint and no IN one, so it fits where a microphone would not. Endpoint accounting and FIFO seating read the same two masks the compiler writes, so a plan the compiler accepts is one the kernel binds.
- Capture never blocks: the PCM descriptor is forced non-blocking and gated on `poll(POLLIN, 0)`, because this tinyalsa's `pcm_wait` polls `POLLOUT`, which a capture stream never raises. An overrun is reset with stop/prepare/start and is bounded, never retried forever.
- Frames leave a speaker slot through a bounded per-listener queue with its own goroutine. A browser that stops reading loses packets; it never back-pressures the gadget.
- Resolve every node after bind. Where the kernel publishes `function_name` that is the identity; this vendor 5.10 does not, so a gadget video node is identified by its controller (the node is named for the UDC and shares its platform device) and two of them are ambiguous, never guessed apart.
- Never infer ALSA cards or video nodes from minors or enumeration order. A display name may only tie a node to its controller, never one function to another.
- A linked UVC function keeps the whole gadget deactivated (`bind_deactivated`) until its video node is open, and `cdev->deactivations` does not survive overlapping opens. Hold exactly one `open(2)` per linked function for the function's lifetime and dup it for streaming; release the holds before gadget teardown, or the unlink returns `-EBUSY`.
- Every wait on a node is bounded. A node that will not open is an error the caller reports, never a request left hanging.
- `Suspend` reports what it could not release, and a caller about to unlink must refuse on that error. configfs does not fail an unlink whose video node is open, it blocks it in the kernel past every context and deadline. If a worker will not stop, the holds are kept rather than given up, because re-taking one means a second `uvc_v4l2_open()` and a leaked deactivation.
- Audio dlopens the packaged tinyalsa and probes readiness with zero-timeout `pcm_wait`; stop paths must never wait on PCM drain.
- Keep the C ABI assertions equal to headers installed from the vendored 5.10 kernel.
- Pace UVC submissions at the host-negotiated interval; the UDC drain rate must never set the frame rate.
- The UVC MJPEG payload spec requires YCbCr; the black frame is a cached three-component JPEG, never grayscale.
- Browser frames are untrusted. Keep payloads and queues bounded and reject undeclared formats.
- Passthrough owns the same UDC. Suspend all outputs before surrender and reopen only after the presentation gadget binds.
