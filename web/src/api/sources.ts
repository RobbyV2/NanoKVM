import { http } from '@/lib/http.ts';
import { getBaseUrl } from '@/lib/service.ts';

export type SourceKind = 'camera' | 'microphone' | 'usb_device';
export type BindingState = 'claimed' | 'streaming' | 'orphaned' | 'suspended';
export type OutputState = 'idle' | 'source' | 'black' | 'silence';

export type SourceFormat = {
  codec: string;
  width?: number;
  height?: number;
  fps?: number;
  sample_rate?: number;
  channels?: number;
};

export type SourceStream = {
  id: string;
  kind: SourceKind;
  label: string;
  formats?: SourceFormat[];
  usb?: {
    profile: string;
    configuration: number;
    interfaces: number[];
  };
};

export type MediaSource = {
  id: string;
  owner: string;
  agent: string;
  label: string;
  streams: SourceStream[];
  connected_at: string;
};

export type Demand = {
  streaming: boolean;
  width?: number;
  height?: number;
  fps?: number;
  since?: string;
};

export type Binding = {
  sink_id: string;
  source_id: string;
  stream_id: string;
  owner: string;
  source_label: string;
  stream_label: string;
  state: BindingState;
  started_at: string;
  expires_at?: string;
};

export type ClaimGrant = {
  binding: Binding;
  token: string;
};

export type ClaimRefusal = {
  sink_id: string;
  owner: string;
  source_label: string;
  since: string;
  takeover: 'immediate' | 'refused';
};

// browser-to-gadget skew over the last window. avg_ms and peak_ms are relative
// to base_ms, the smallest skew yet seen, because the two clocks need not agree.
export type SinkLatency = {
  frames: number;
  avg_ms: number;
  peak_ms: number;
  base_ms: number;
  updated_at: string;
};

export type SourceSink = {
  id: string;
  kind: SourceKind;
  label: string;
  slot: number;
  demand: Demand;
  output: OutputState;
  binding: Binding | null;
  latency?: SinkLatency;
};

export type SourcesSnapshot = {
  sinks: SourceSink[];
  sources: MediaSource[];
  bindings: Binding[];
};

export type SourcesEvent = {
  type: string;
  snapshot?: SourcesSnapshot;
  sink?: SourceSink;
  sinks?: SourceSink[];
  source?: MediaSource;
  binding?: Binding;
  sink_id?: string;
  source_id?: string;
  reason?: string;
  demand?: Demand;
};

export type SourceSlot = Pick<SourceSink, 'id' | 'kind' | 'label'>;

export function getSources() {
  return http.get('/api/sources');
}

export function setSourceSlots(slots: SourceSlot[]) {
  return http.request({ method: 'put', url: '/api/sources/sinks', data: { slots } });
}

export function releaseSource(sinkID: string) {
  return http.delete(`/api/sources/bindings/${encodeURIComponent(sinkID)}`);
}

export function disconnectSources() {
  return http.delete('/api/sources/bindings');
}

export function sourcesSocket(path: 'events' | 'ws') {
  return `${getBaseUrl('ws')}/api/sources/${path}`;
}
