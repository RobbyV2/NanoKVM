import { http } from '@/lib/http.ts';

// Import, enumeration, and relay setup can outlive a normal request timeout.
const startTimeout = 120 * 1000;
const listTimeout = 15 * 1000;

// The imported device as the exporter described it.
export type PassthroughDevice = {
  busId: string;
  idVendor: string;
  idProduct: string;
  speed: string;
  class: number;
};

export type PassthroughStatus = {
  active: boolean;
  mode: PassthroughMode;
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

export type PassthroughMode = 'hybrid' | 'exact';

// One entry of the exporter's device list. unsupported carries why the device
// cannot be relayed and is absent when it can.
export type PassthroughRemoteDevice = {
  busId: string;
  idVendor: string;
  idProduct: string;
  speed: string;
  class: number;
  unsupported?: string;
};

export type PassthroughDeviceList = {
  devices: PassthroughRemoteDevice[];
};

// the current session, or a zero status when none is running
export function getPassthrough() {
  return http.get('/api/vm/passthrough');
}

// Import busId from exporter and start the selected relay.
export function startPassthrough(exporter: string, busId: string, mode: PassthroughMode) {
  return http.post(
    '/api/vm/passthrough/start',
    { exporter, busId, mode },
    { timeout: startTimeout }
  );
}

// Enumerate the exporter's devices so a busid does not have to be typed.
export function listPassthroughDevices(exporter: string) {
  return http.post('/api/vm/passthrough/devices', { exporter }, { timeout: listTimeout });
}

// Stop the relay, detach the port, and restore the gadget.
export function stopPassthrough() {
  return http.post('/api/vm/passthrough/stop');
}
