# Media gadget

- A vendor-5.10 UVC function consumes two IN endpoints: status interrupt, then stream.
- Host microphone audio uses UAC2 `p_*`: USB IN is local ALSA playback. `c_*` is USB OUT.
- Resolve every node after bind. Where the kernel publishes `function_name` that is the identity; this vendor 5.10 does not, so a gadget video node is identified by its controller (the node is named for the UDC and shares its platform device) and two of them are ambiguous, never guessed apart.
- Never infer ALSA cards or video nodes from minors or enumeration order. A display name may only tie a node to its controller, never one function to another.
- A linked UVC function keeps the whole gadget deactivated (`bind_deactivated`) until its video node is open, and `cdev->deactivations` does not survive overlapping opens. Hold exactly one `open(2)` per linked function for the function's lifetime and dup it for streaming; release the holds before gadget teardown, or the unlink returns `-EBUSY`.
- Every wait on a node is bounded. A node that will not open is an error the caller reports, never a request left hanging.
- Audio dlopens the packaged tinyalsa and probes readiness with zero-timeout `pcm_wait`; stop paths must never wait on PCM drain.
- Keep the C ABI assertions equal to headers installed from the vendored 5.10 kernel.
- Pace UVC submissions at the host-negotiated interval; the UDC drain rate must never set the frame rate.
- The UVC MJPEG payload spec requires YCbCr; the black frame is a cached three-component JPEG, never grayscale.
- Browser frames are untrusted. Keep payloads and queues bounded and reject undeclared formats.
- Passthrough owns the same UDC. Suspend all outputs before surrender and reopen only after the presentation gadget binds.
