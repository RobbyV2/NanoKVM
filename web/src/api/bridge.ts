import { http } from '@/lib/http.ts';

// the transaction runs inside the request and holds the management address for
// the whole verification window, so it outlives the client's default timeout
const applyTimeout = 90 * 1000;

export type BridgeState = 'disabled' | 'enabled' | 'rolledBack' | 'failed' | 'pending';

// the three verification gates, plus a note that inbound was only self-tested
export type BridgeChecks = {
  address: boolean;
  gateway: boolean;
  inbound: boolean;
  inboundWeak: boolean;
};

export type BridgePort = {
  name: string;
  state: string;
  up: boolean;
  // the cable, as opposed to up, which is set on a port with nothing in it too
  carrier: boolean;
};

// a second path between the uplink's segment and another port of the bridge.
// Non-null is evidence the condition exists; null is not evidence it does not.
export type BridgeLoop = {
  port: string;
  mac: string;
  reason: string;
};

// an armed dead-man marker: an apply is in flight, or the device booted into
// one that never finished
export type BridgeArmed = {
  operation: string;
  snapshotPath: string;
  armedAt: string;
  deadline: string;
};

export type BridgeApply = {
  state: BridgeState;
  uplink: string;
  enabled: boolean;
  checks: BridgeChecks;
  message: string;
  appliedAt: string;
};

export type BridgeStatus = {
  state: BridgeState;
  uplink: string;
  exists: boolean;
  mac: string;
  ports: BridgePort[];
  address: string;
  gateway: string;
  carrier: boolean;
  loop?: BridgeLoop | null;
  pending?: BridgeArmed | null;
  lastApply?: BridgeApply | null;

  // ncm, rndis, or empty for a gadget presenting no network. Reported only: the
  // control for it is the Virtual Network one under Settings, Device, because
  // the protocol is a property of the USB profile rather than of the bridge.
  protocol: string;
};

// get the bridge state, its ports, the armed marker and the last transaction
export function getBridge() {
  return http.get('/api/network/bridge');
}

// enable or disable the bridge; the reply repeats the state, the uplink, the
// gates and a message, but the refetch is what a caller who lost it reads back
export function setBridge(enabled: boolean) {
  return http.post('/api/network/bridge', { enabled }, { timeout: applyTimeout });
}

// put the snapshot back after an apply that never finished
export function revertBridge() {
  return http.post('/api/network/bridge/revert', {}, { timeout: applyTimeout });
}
