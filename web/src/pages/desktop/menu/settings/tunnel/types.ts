export type TunnelService = 'wstunnel' | 'newt';

export type TunnelState =
  | 'notInstall'
  | 'notConfigured'
  | 'stopped'
  | 'running'
  | 'connected'
  | 'error';

// currentState can only reach connected for a service whose spec names a health
// file to stat. wstunnel's is empty: it ships no health file and has no status
// command, so running is all the server can ever report for it and connected is
// newt's alone.
export const reportsConnection: Record<TunnelService, boolean> = {
  wstunnel: false,
  newt: true
};

export type TunnelStatus = {
  state: TunnelState;
  message: string;
  pid: number;
  custom: boolean;
  enabled: boolean;
};

export type EnvEntry = {
  key: string;
  value: string;
  secret: boolean;
  configured: boolean;
};
