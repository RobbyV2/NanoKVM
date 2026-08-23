# WebUSB relay

- `NKUF` v1 uses a 32-byte big-endian header and at most 64 KiB of payload.
- The browser is untrusted. Validate profile identity, selected interfaces, response IDs, lengths, status, and pending budgets server-side.
- WebUSB is Hybrid-only. It never attaches VHCI and never supports isochronous endpoints.
- Protected interfaces stay in the catalog but never enter the runtime FunctionFS projection.
