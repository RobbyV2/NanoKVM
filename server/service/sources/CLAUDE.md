# sources

Owns browser source sessions, declared media slots, binding leases and the shared live
state feed. It does not open media devices or compile USB functions. Design:
`docs/superpowers/specs/2026-08-22-media-sources-design.md`.

One mutex owns the source, sink and binding maps. Keep claims, resumes, slot replacement
and event publication in that critical section, so observers never see a binding without
its sink and two claims cannot win one slot.

Every client field is bounded and validated before it enters the registry. Control
WebSockets accept JSON text only, cap messages at 64 KiB and close on sustained message
floods. Event subscribers have bounded queues; overflow closes the subscription so the
client reconnects and receives a fresh snapshot.

Lease tokens are returned once on claim, stay out of snapshots and events, and are
compared in constant time. A socket close orphans a binding for the refresh grace rather
than releasing its slot. Resume requires the same username, token, sink kind and a live
source stream.

`Sink.Output` is a state primitive. `black` and `silence` instruct the later media backend
to generate legal fallback data while host demand is active; this package never creates
frames or samples.
