// Mirrors server/proto/exit.go. Field names and json tags are the contract.

export type ExitMode = 'native' | 'wstunnel';

export const exitModes: ExitMode[] = ['native', 'wstunnel'];

export type ExitTunnelState = 'disconnected' | 'connecting' | 'connected';

export type ExitPlatform = 'windows' | 'macos' | 'linux';

export const exitPlatforms: ExitPlatform[] = ['windows', 'macos', 'linux'];

export type ExitPeer = {
  addr: string;
  hostname?: string;
  os?: string;
  transport: ExitMode;
};

export type ExitNIC = {
  ifname: string;
  up: boolean;
  // ncm, rndis or "" when the profile links no network function
  protocol: string;
  // 10.x.y.1/24 or ""
  address: string;
};

// the boolean vector `S94exit status <slot>` reports, plus dns from the forwarder
export type ExitDownstream = {
  forward: boolean;
  routing: boolean;
  tun: boolean;
  hev: boolean;
  wstunnel: boolean;
  dns: boolean;
  nat: boolean;
};

export const downstreamKeys: (keyof ExitDownstream)[] = [
  'forward',
  'routing',
  'tun',
  'hev',
  'wstunnel',
  'dns',
  'nat'
];

export type ExitUpstream = {
  reachable: boolean;
  latencyMs: number;
  checkedAt: string | null;
};

export type ExitDNSStats = {
  queries: number;
  failures: number;
  redirected: number;
};

export type ExitBytes = {
  up: number;
  down: number;
};

// GetExitStatusRsp
export type ExitStatus = {
  slot: string;
  enabled: boolean;
  pending: boolean;
  mode: ExitMode;
  token: string;
  tunnel: ExitTunnelState;
  peer: ExitPeer | null;
  previousPeer: ExitPeer | null;
  peerChangedAt: string | null;
  connectedAt: string | null;
  lastConnectedAt: string | null;
  uptimeSeconds: number;
  nic: ExitNIC;
  downstream: ExitDownstream;
  upstream: ExitUpstream;
  dns: ExitDNSStats;
  bytes: ExitBytes;
  message: string;
};

// GetExitSlotsRsp
export type ExitSlots = {
  slots: ExitStatus[];
};

// GetExitConfigRsp: the editable part; the token travels with status
export type ExitConfig = {
  slot: string;
  mode: ExitMode;
  dns: string[];
  mtu: number;
  allowPrivate: boolean;
  pinPeer: boolean;
};

// SetExitConfigReq: an absent field leaves the server's value alone
export type SetExitConfigReq = {
  mode?: ExitMode;
  dns?: string[];
  mtu?: number;
  allowPrivate?: boolean;
  pinPeer?: boolean;
};

export type ExitCommand = {
  platform: ExitPlatform;
  shell: 'powershell' | 'bash' | string;
  command: string;
  notes?: string;
};

// GetExitCommandsRsp
export type ExitCommands = {
  // http or https
  scheme: string;
  // host[:port] as the operator reached the UI
  host: string;
  // sha256 hex of the serving certificate, "" on http
  fingerprint: string;
  wstunnelVersion: string;
  native: ExitCommand[];
  wstunnel: ExitCommand[];
};

// GetExitLogsRsp
export type ExitLogs = {
  hev: string[];
  wstunnel: string[];
};

// the server validates MTU with min=576,max=1400 and dns with max=4
export const mtuMin = 576;
export const mtuMax = 1400;
export const dnsMax = 4;
