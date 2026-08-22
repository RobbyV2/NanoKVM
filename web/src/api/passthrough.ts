import { http } from '@/lib/http.ts';

// a start dials the exporter, imports the device, waits for it to enumerate on
// the local host stack and only then spawns the proxy, so it outruns a plain
// request more often than not
const startTimeout = 120 * 1000;

// the imported device as the exporter described it. Class is the USB device
// class byte: audio and video devices stream over isochronous endpoints, which
// this raw-gadget cannot carry, so the UI names them rather than letting
// someone find out.
export type PassthroughDevice = {
  busId: string;
  idVendor: string;
  idProduct: string;
  speed: string;
  class: number;
};

// hidSurrendered is the whole cost of a session: the SoC has one device
// controller and udc->driver is a single pointer, so while the proxy holds it
// there is no keyboard, no mouse and no virtual media.
export type PassthroughStatus = {
  active: boolean;
  exporter: string;
  udc: string;
  port: number;
  hub: string;
  bus: number;
  address: number;
  pid: number;
  hidSurrendered: boolean;
  startedAt: string;
  device?: PassthroughDevice | null;
};

// the current session, or a zero status when none is running
export function getPassthrough() {
  return http.get('/api/vm/passthrough');
}

// import busId from exporter and take the udc; the reply repeats the status the
// refetch would return
export function startPassthrough(exporter: string, busId: string) {
  return http.post('/api/vm/passthrough/start', { exporter, busId }, { timeout: startTimeout });
}

// stop the proxy, detach the port and give the gadget back
export function stopPassthrough() {
  return http.post('/api/vm/passthrough/stop');
}
