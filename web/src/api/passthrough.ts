import { http } from '@/lib/http.ts';

// Import, enumeration, and relay setup can outlive a normal request timeout.
const startTimeout = 120 * 1000;

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

// Stop the relay, detach the port, and restore the gadget.
export function stopPassthrough() {
  return http.post('/api/vm/passthrough/stop');
}
