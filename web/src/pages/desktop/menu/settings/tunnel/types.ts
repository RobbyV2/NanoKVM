export type TunnelService = 'wstunnel' | 'newt';

export type TunnelState =
  | 'notInstall'
  | 'notConfigured'
  | 'stopped'
  | 'running'
  | 'connected'
  | 'error';

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
