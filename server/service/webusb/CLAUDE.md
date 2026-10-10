# WebUSB relay

- `NKUF` v1 uses a 32-byte big-endian header and at most 64 KiB of payload.
- The browser is untrusted. Validate profile identity, selected interfaces, response IDs, lengths, status, and pending budgets server-side.
- WebUSB is Hybrid-only. It never attaches VHCI and never supports isochronous endpoints.
- Protected interfaces stay in the catalog but never enter the runtime FunctionFS projection.

## Browser webcam sharing

Devices → Share camera uses `getUserMedia`, JPEG encoding and the NKMF source
transport. It does not use WebUSB or this relay. Route webcam capture failures to
`web/src/pages/desktop/menu/devices/` and `service/media`; this package's
isochronous restrictions apply only to Share USB.
