import type { EnvEntry, TunnelService, TunnelStatus } from './types.ts';

export type TunnelAction = '' | 'start' | 'stop' | 'restart' | 'save';

export type TunnelView = {
  service: TunnelService;
  status?: TunnelStatus;
  args: string;
  env: EnvEntry[];
  logs: string[];
  isLoading: boolean;
  action: TunnelAction;
  isSaved: boolean;
  errMsg: string;
};

export function emptyView(service: TunnelService): TunnelView {
  return {
    service,
    args: '',
    env: [],
    logs: [],
    isLoading: false,
    action: '',
    isSaved: false,
    errMsg: ''
  };
}

export function switchService(view: TunnelView, service: TunnelService): TunnelView {
  return view.service === service ? view : emptyView(service);
}

// One component serves both services, so every update names the service it was
// requested for. A response that arrives after the panel has moved on belongs
// to the service that asked for it, not the one on screen: without this guard
// newt shows wstunnel's arguments, and saving writes them to newt.
export function applyView(
  view: TunnelView,
  service: TunnelService,
  next: Partial<TunnelView>
): TunnelView {
  return view.service === service ? { ...view, ...next } : view;
}
