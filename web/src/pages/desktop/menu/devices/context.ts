import { createContext, useContext } from 'react';

import type { PassthroughStatus } from '@/api/passthrough.ts';
import type { ClaimRefusal, SourceKind, SourceSink, SourcesSnapshot } from '@/api/sources.ts';

import type { DeviceOffer, SourceConnection } from './client.ts';

export type SourceEventsConnection = 'connecting' | 'connected' | 'disconnected';
export type MediaPermission = 'granted' | 'denied' | 'prompt' | 'unknown';

export type DevicesState = {
  snapshot: SourcesSnapshot;
  eventsConnection: SourceEventsConnection;
  sourceConnection: SourceConnection;
  devices: Record<SourceKind, DeviceOffer[]>;
  permissions: Record<'camera' | 'microphone', MediaPermission>;
  passthrough: PassthroughStatus | null;
  owned: Set<string>;
  active: Set<string>;
  busy: Set<string>;
  muted: Set<string>;
  errors: Record<string, string>;
  refusals: Record<string, ClaimRefusal>;
  revoked: Record<string, string>;
  share: (sink: SourceSink, deviceID?: string) => Promise<void>;
  takeover: (sink: SourceSink, deviceID?: string) => Promise<void>;
  release: (sinkID: string) => Promise<void>;
  choose: (sinkID: string, deviceID: string) => void;
  setMuted: (sinkID: string, muted: boolean) => void;
  setCounts: (cameras: number, microphones: number) => Promise<void>;
  disconnectAll: () => Promise<void>;
  refresh: () => void;
};

export const DevicesContext = createContext<DevicesState | null>(null);

export function useDevices() {
  const value = useContext(DevicesContext);
  if (!value) throw new Error('useDevices must be used inside SourcesProvider');
  return value;
}
