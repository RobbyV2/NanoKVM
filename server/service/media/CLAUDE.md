# Media gadget

- A vendor-5.10 UVC function consumes two IN endpoints: status interrupt, then stream.
- Host microphone audio uses UAC2 `p_*`: USB IN is local ALSA playback. `c_*` is USB OUT.
- Resolve every node after bind. UVC exposes `function_name`; the vendor UAC2 driver needs an equivalent sysfs attribute before audio may start.
- Never infer ALSA cards or video nodes from minors, enumeration order, or display names.
- Audio dlopens the packaged tinyalsa and probes readiness with zero-timeout `pcm_wait`; stop paths must never wait on PCM drain.
- Keep the C ABI assertions equal to headers installed from the vendored 5.10 kernel.
- Browser frames are untrusted. Keep payloads and queues bounded and reject undeclared formats.
- Passthrough owns the same UDC. Suspend all outputs before surrender and reopen only after the presentation gadget binds.
