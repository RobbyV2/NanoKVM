# sources

Owns browser source sessions, declared media slots, binding leases and the shared live
state feed. It does not open media devices or compile USB functions. Design:
`docs/superpowers/specs/2026-08-22-media-sources-design.md`.

One mutex owns the source, sink and binding maps. Keep claims, resumes, slot replacement
and event publication in that critical section, so observers never see a binding without
its sink and two claims cannot win one slot.

Every client field is bounded and validated before it enters the registry. Control
messages are JSON text capped at 64 KiB; media frames use the bounded binary format
below. Sustained message floods close the socket. Event subscribers have bounded queues;
overflow closes the subscription so the client reconnects and receives a fresh snapshot.

Lease tokens are returned once on claim, stay out of snapshots and events, and are
compared in constant time. A socket close orphans a binding for the refresh grace rather
than releasing its slot. Resume requires the same username, token, sink kind and a live
source stream.

`Sink.Output` is a state primitive. `black` and `silence` instruct the media backend to
generate legal fallback data while host demand is active; this package never creates
frames or samples.

The source socket accepts both control JSON and NKMF v1 binary frames. The 26-byte
big-endian header is `NKMF`, version, kind, flags, sequence, timestamp, sink length,
stream length and payload length. IDs are at most 64 bytes. MJPEG is at most 2 MiB.
The PCM envelope is at most 40 ms; the gadget accepts exact 20 ms mono S16LE 48 kHz
packets. Authenticate the binding before ingress, copy payloads before enqueueing, and
acknowledge only accepted frames.
