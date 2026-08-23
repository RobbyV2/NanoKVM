import { createContext, useContext } from 'react';

import type { SourceKind, SourceSink, SourcesSnapshot } from '@/api/sources.ts';

import type { DeviceOffer, SourceConnection } from './client.ts';

export type SourceEventsConnection = 'connecting' | 'connected' | 'disconnected';

export type DevicesState = {
  snapshot: SourcesSnapshot;
  eventsConnection: SourceEventsConnection;
  sourceConnection: SourceConnection;
  devices: Record<SourceKind, DeviceOffer[]>;
  owned: Set<string>;
  active: Set<string>;
  busy: Set<string>;
  errors: Record<string, string>;
  share: (sink: SourceSink, deviceID?: string) => Promise<void>;
  release: (sinkID: string) => Promise<void>;
  choose: (sinkID: string, deviceID: string) => void;
  allow: (kind: SourceKind) => Promise<void>;
  setCounts: (cameras: number, microphones: number) => Promise<void>;
  disconnectAll: () => Promise<void>;
};

export const DevicesContext = createContext<DevicesState | null>(null);

export function useDevices() {
  const value = useContext(DevicesContext);
  if (!value) throw new Error('useDevices must be used inside SourcesProvider');
  return value;
}
