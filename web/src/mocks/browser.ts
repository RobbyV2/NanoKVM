import { http, HttpResponse, ws } from 'msw';
import { setupWorker } from 'msw/browser';

import type { Binding, MediaSource, SourceSlot, SourcesSnapshot } from '@/api/sources.ts';

let isLoggedIn = false;
let activeProfile = 'standard';

const standardProfile = {
  schema_version: 1,
  name: 'standard',
  built_in: true,
  device: {
    vendor_id: '0x3346',
    product_id: '0x1009',
    bcd_device: '0x0510',
    class: 239,
    subclass: 2,
    protocol: 1,
    serial: '0123456789ABCDEF',
    manufacturer: 'sipeed',
    product: 'NanoKVM'
  },
  config: { bm_attributes: 224, max_power: 120, configuration: 'NanoKVM' },
  functions: [
    { kind: 'hid', instance: 'GS0' },
    { kind: 'hid', instance: 'GS1' },
    { kind: 'hid', instance: 'GS2' }
  ]
};

const presentationProfiles = new Map([[standardProfile.name, standardProfile]]);

const aliceBinding: Binding = {
  sink_id: 'uvc.cam1',
  source_id: 'src_alice',
  stream_id: 'cam_phone',
  owner: 'alice',
  source_label: 'Phone',
  stream_label: 'Back camera',
  state: 'streaming',
  started_at: '2026-08-22T11:04:19Z'
};
const aliceSource: MediaSource = {
  id: 'src_alice',
  owner: 'alice',
  agent: 'browser',
  label: 'Phone',
  streams: [{ id: 'cam_phone', kind: 'camera', label: 'Back camera' }],
  connected_at: '2026-08-22T11:04:11Z'
};
let sourcesSnapshot: SourcesSnapshot = {
  sinks: [
    {
      id: 'uvc.cam0',
      kind: 'camera',
      label: 'NanoKVM Camera 1',
      slot: 0,
      demand: { streaming: true, width: 640, height: 480, fps: 30 },
      output: 'black',
      binding: null
    },
    {
      id: 'uvc.cam1',
      kind: 'camera',
      label: 'NanoKVM Camera 2',
      slot: 1,
      demand: { streaming: true, width: 640, height: 480, fps: 30 },
      output: 'source',
      binding: aliceBinding
    },
    {
      id: 'uac2.mic0',
      kind: 'microphone',
      label: 'NanoKVM Microphone 1',
      slot: 0,
      demand: { streaming: true },
      output: 'silence',
      binding: null
    }
  ],
  sources: [aliceSource],
  bindings: [aliceBinding]
};

const sourceEventSenders = new Set<(data: string) => void>();

function broadcastSourceEvent(event: Record<string, unknown>) {
  const data = JSON.stringify(event);
  for (const send of sourceEventSenders) send(data);
}

const sourceEvents = ws.link(/\/api\/sources\/events$/);
const sourceEventsHandler = sourceEvents.addEventListener('connection', ({ client }) => {
  const send = (data: string) => client.send(data);
  sourceEventSenders.add(send);
  send(JSON.stringify({ type: 'snapshot', snapshot: sourcesSnapshot }));
  client.addEventListener('close', () => sourceEventSenders.delete(send));
});

const sourceControl = ws.link(/\/api\/sources\/ws$/);
const sourceControlHandler = sourceControl.addEventListener('connection', ({ client }) => {
  let streams: MediaSource['streams'] = [];
  client.addEventListener('close', () => {
    const source = sourcesSnapshot.sources.find((item) => item.id === 'src_browser');
    if (!source) return;
    sourcesSnapshot.sources = sourcesSnapshot.sources.filter((item) => item.id !== source.id);
    broadcastSourceEvent({ type: 'source_removed', source });
  });
  client.addEventListener('message', ({ data }) => {
    if (data instanceof ArrayBuffer) {
      const frame = decodeMockFrame(data);
      const sink = sourcesSnapshot.sinks.find((item) => item.id === frame.sinkID);
      if (sink?.binding && sink.binding.state !== 'streaming') {
        sink.binding.state = 'streaming';
        sink.output = 'source';
        broadcastSourceEvent({ type: 'binding_state', binding: sink.binding });
      }
      client.send(
        JSON.stringify({
          type: 'frame_ack',
          sink_id: frame.sinkID,
          stream_id: frame.streamID,
          sequence: frame.sequence
        })
      );
      return;
    }
    if (typeof data !== 'string') return;
    const message = JSON.parse(data) as Record<string, any>;
    if (message.type === 'hello') {
      streams = message.streams as MediaSource['streams'];
      const source: MediaSource = {
        ...aliceSource,
        id: 'src_browser',
        owner: 'admin',
        label: 'Browser',
        streams
      };
      const sourceIndex = sourcesSnapshot.sources.findIndex((item) => item.id === source.id);
      if (sourceIndex === -1) {
        sourcesSnapshot.sources.push(source);
        broadcastSourceEvent({ type: 'source_added', source });
      } else {
        sourcesSnapshot.sources[sourceIndex] = source;
      }
      client.send(
        JSON.stringify({
          type: 'source_ready',
          source,
          snapshot: sourcesSnapshot
        })
      );
      return;
    }
    if (message.type === 'claim') {
      const sink = sourcesSnapshot.sinks.find((item) => item.id === message.sink_id);
      if (!sink || sink.binding) {
        client.send(
          JSON.stringify({
            type: 'claim_refused',
            sink_id: message.sink_id,
            owner: sink?.binding?.owner || 'unknown',
            source_label: sink?.binding?.source_label || 'unknown',
            since: sink?.binding?.started_at || '',
            message: 'slot_occupied'
          })
        );
        return;
      }
      const stream = streams.find((item) => item.id === message.stream_id)!;
      const binding: Binding = {
        sink_id: sink.id,
        source_id: 'src_browser',
        stream_id: stream.id,
        owner: 'admin',
        source_label: 'Browser',
        stream_label: stream.label,
        state: 'claimed',
        started_at: new Date().toISOString()
      };
      sink.binding = binding;
      sourcesSnapshot.bindings.push(binding);
      broadcastSourceEvent({ type: 'binding_added', binding });
      client.send(JSON.stringify({ type: 'claimed', binding, token: 'mock-token' }));
      return;
    }
    if (message.type === 'resume') {
      const binding = sourcesSnapshot.bindings.find(
        (candidate) =>
          candidate.sink_id === message.sink_id && candidate.stream_id === message.stream_id
      );
      if (!binding || message.token !== 'mock-token') {
        client.send(JSON.stringify({ type: 'error', message: 'lease expired' }));
        return;
      }
      client.send(JSON.stringify({ type: 'resumed', binding }));
      return;
    }
    if (message.type === 'release') {
      const binding = sourcesSnapshot.bindings.find(
        (candidate) => candidate.sink_id === message.sink_id
      );
      releaseMockBinding(message.sink_id);
      if (binding) broadcastSourceEvent({ type: 'binding_removed', binding, reason: 'released' });
      client.send(JSON.stringify({ type: 'released', sink_id: message.sink_id }));
    }
  });
});

function profileSummary(profile: typeof standardProfile) {
  return {
    name: profile.name,
    built_in: profile.built_in,
    active: profile.name === activeProfile,
    manufacturer: profile.device.manufacturer,
    product: profile.device.product,
    functions: profile.functions.map((item) => `${item.kind}.${item.instance}`),
    provenance: {
      origin: profile.built_in ? 'built-in' : 'user',
      descriptors: false,
      imported: false
    }
  };
}

export const handlers = [
  http.post('/api/auth/login', () => {
    isLoggedIn = true;
    return HttpResponse.json({
      code: 0,
      data: {}
    });
  }),
  http.get('/api/auth/account', () => {
    if (!isLoggedIn) {
      return HttpResponse.json('unauthorized', { status: 401 });
    }
    return HttpResponse.json({
      code: 0,
      data: { username: 'admin', role: 'admin' }
    });
  }),
  http.post('/api/auth/logout', () => {
    isLoggedIn = false;
    return HttpResponse.json({ code: 0 });
  }),
  http.get('/api/sources', () => HttpResponse.json({ code: 0, data: sourcesSnapshot })),
  http.put('/api/sources/sinks', async ({ request }) => {
    const body = (await request.json()) as { slots: SourceSlot[] };
    sourcesSnapshot = {
      sinks: body.slots.map((slot) => ({
        ...slot,
        slot: Number(slot.id.match(/[0-9]+$/)?.[0] || 0),
        demand: { streaming: false },
        output: 'idle',
        binding: sourcesSnapshot.bindings.find((binding) => binding.sink_id === slot.id) || null
      })),
      sources: sourcesSnapshot.sources,
      bindings: sourcesSnapshot.bindings.filter((binding) =>
        body.slots.some((slot) => slot.id === binding.sink_id)
      )
    };
    broadcastSourceEvent({ type: 'sinks_changed', sinks: sourcesSnapshot.sinks });
    return HttpResponse.json({ code: 0, data: sourcesSnapshot });
  }),
  http.delete('/api/sources/bindings/:sink', ({ params }) => {
    const sinkID = String(params.sink);
    const binding = sourcesSnapshot.bindings.find((candidate) => candidate.sink_id === sinkID);
    releaseMockBinding(sinkID);
    if (binding) broadcastSourceEvent({ type: 'binding_removed', binding, reason: 'released' });
    return HttpResponse.json({ code: 0 });
  }),
  http.delete('/api/sources/bindings', () => {
    for (const sink of sourcesSnapshot.sinks) sink.binding = null;
    sourcesSnapshot.bindings = [];
    return HttpResponse.json({ code: 0 });
  }),
  http.get('/api/presentation/status', () => {
    const profile = presentationProfiles.get(activeProfile)!;
    return HttpResponse.json({
      code: 0,
      data: {
        snapshot: {
          active: activeProfile,
          mode: 'normal',
          linked: profile.functions.map((item) => `${item.kind}.${item.instance}`),
          endpoints: { in: 3, out: 3 },
          headroom: { in: 3, out: 2 }
        },
        profile: profileSummary(profile),
        last_known_good: activeProfile
      }
    });
  }),
  http.get('/api/presentation/profiles', () =>
    HttpResponse.json({
      code: 0,
      data: { profiles: [...presentationProfiles.values()].map(profileSummary) }
    })
  ),
  http.post('/api/presentation/profiles/import', () => {
    const profile = structuredClone(standardProfile);
    profile.name = 'imported-profile';
    profile.built_in = false;
    presentationProfiles.set(profile.name, profile);
    return HttpResponse.json({ code: 0, data: profile });
  }),
  http.get('/api/presentation/profiles/:name', ({ params }) => {
    const profile = presentationProfiles.get(String(params.name));
    return HttpResponse.json(
      profile ? { code: 0, data: profile } : { code: -2, msg: 'profile not found' }
    );
  }),
  http.post('/api/presentation/profiles/:name/clone', async ({ params, request }) => {
    const source = presentationProfiles.get(String(params.name));
    const body = (await request.json()) as { name: string };
    if (!source) return HttpResponse.json({ code: -2, msg: 'profile not found' });
    const profile = structuredClone(source);
    profile.name = body.name;
    profile.built_in = false;
    presentationProfiles.set(profile.name, profile);
    return HttpResponse.json({ code: 0, data: profile });
  }),
  http.put('/api/presentation/profiles/:name', async ({ request }) => {
    const profile = (await request.json()) as typeof standardProfile;
    presentationProfiles.set(profile.name, profile);
    return HttpResponse.json({ code: 0, data: profile });
  }),
  http.delete('/api/presentation/profiles/:name', ({ params }) => {
    presentationProfiles.delete(String(params.name));
    return HttpResponse.json({ code: 0 });
  }),
  http.get(
    '/api/presentation/profiles/:name/export',
    () =>
      new HttpResponse(new Uint8Array([0x50, 0x4b, 0x05, 0x06]), {
        headers: { 'Content-Type': 'application/vnd.nanokvm.presentation+zip' }
      })
  ),
  http.put('/api/presentation/config/preview', async ({ request }) => {
    const profile = (await request.json()) as typeof standardProfile;
    return HttpResponse.json({
      code: 0,
      data: {
        valid: true,
        errors: [],
        warnings: [],
        profile: profile.name,
        functions: profile.functions.map((item) => `${item.kind}.${item.instance}`),
        endpoints: { in: 3, out: 3 },
        headroom: { in: 3, out: 2 },
        operations: 24
      }
    });
  }),
  http.put('/api/presentation/config/apply', async ({ request }) => {
    const body = (await request.json()) as { name: string };
    activeProfile = body.name;
    return HttpResponse.json({ code: 0 });
  })
];
export const worker = setupWorker(...handlers, sourceEventsHandler, sourceControlHandler);

function releaseMockBinding(sinkID: string) {
  const sink = sourcesSnapshot.sinks.find((item) => item.id === sinkID);
  if (sink) sink.binding = null;
  sourcesSnapshot.bindings = sourcesSnapshot.bindings.filter(
    (binding) => binding.sink_id !== sinkID
  );
}

function decodeMockFrame(data: ArrayBuffer) {
  const view = new DataView(data);
  const sinkLength = view.getUint8(20);
  const streamLength = view.getUint8(21);
  const text = new TextDecoder();
  return {
    sequence: view.getUint32(8),
    sinkID: text.decode(new Uint8Array(data, 26, sinkLength)),
    streamID: text.decode(new Uint8Array(data, 26 + sinkLength, streamLength))
  };
}
