import assert from 'node:assert/strict';
import test from 'node:test';

import type { ClaimRefusal, SourceSink, SourcesSnapshot } from '../../../../api/sources.ts';
import { BrowserSourceClient, type DeviceOffer } from './client.ts';
import { PcmPacketizer } from './pcm.ts';
import { mediaSlotRows, mediaSlots, reduceSources } from './state.ts';
import { encodeMediaFrame } from './transport.ts';

test('NKMF frame uses network byte order and bounded identifiers', () => {
  const payload = Uint8Array.from([0xff, 0xd8, 0xff, 0xd9]).buffer;
  const encoded = encodeMediaFrame({
    kind: 'mjpeg',
    sequence: 0x01020304,
    timestampUS: 0x010203040506,
    sinkID: 'uvc.cam0',
    streamID: 'cam_a',
    payload
  });
  const bytes = new Uint8Array(encoded);
  const view = new DataView(encoded);
  assert.deepEqual([...bytes.subarray(0, 4)], [0x4e, 0x4b, 0x4d, 0x46]);
  assert.equal(view.getUint8(4), 1);
  assert.equal(view.getUint8(5), 1);
  assert.equal(view.getUint16(6), 0);
  assert.equal(view.getUint32(8), 0x01020304);
  assert.equal(view.getBigUint64(12), 0x010203040506n);
  assert.equal(view.getUint8(20), 8);
  assert.equal(view.getUint8(21), 5);
  assert.equal(view.getUint32(22), 4);
  assert.equal(new TextDecoder().decode(bytes.subarray(26, 34)), 'uvc.cam0');
  assert.equal(new TextDecoder().decode(bytes.subarray(34, 39)), 'cam_a');
  assert.deepEqual([...bytes.subarray(39)], [0xff, 0xd8, 0xff, 0xd9]);
});

test('NKMF rejects oversized audio and identifiers', () => {
  const base = {
    kind: 'pcm_s16le_mono_48k' as const,
    sequence: 1,
    timestampUS: 1,
    sinkID: 'uac2.mic0',
    streamID: 'mic_a'
  };
  assert.throws(() => encodeMediaFrame({ ...base, payload: new ArrayBuffer(3841) }));
  assert.throws(() =>
    encodeMediaFrame({ ...base, sinkID: 'x'.repeat(65), payload: new ArrayBuffer(2) })
  );
});

test('audio resamples and emits exact 20 millisecond packets', () => {
  const packetizer = new PcmPacketizer(48000);
  const samples = new Float32Array(961).fill(0.5);
  const packets = packetizer.push(samples);
  assert.equal(packets.length, 1);
  assert.equal(packets[0].byteLength, 1920);
  assert.equal(new DataView(packets[0]).getInt16(0, true), 16384);
});

test('event deltas keep shared sink state current', () => {
  const sink: SourceSink = {
    id: 'uvc.cam0',
    kind: 'camera',
    label: 'NanoKVM Camera 1',
    slot: 0,
    demand: { streaming: false },
    output: 'idle',
    binding: null
  };
  const initial: SourcesSnapshot = { sinks: [sink], sources: [], bindings: [] };
  const binding = {
    sink_id: sink.id,
    source_id: 'src_a',
    stream_id: 'cam_a',
    owner: 'alice',
    source_label: 'Browser',
    stream_label: 'Camera',
    state: 'claimed' as const,
    started_at: '2026-08-22T00:00:00Z'
  };
  const claimed = reduceSources(initial, { type: 'binding_added', binding });
  const demanded = reduceSources(claimed, {
    type: 'demand',
    sink_id: sink.id,
    demand: { streaming: true, width: 640, height: 480, fps: 30 }
  });
  assert.equal(demanded.sinks[0].binding?.owner, 'alice');
  assert.equal(demanded.sinks[0].output, 'black');
  const streaming = reduceSources(demanded, {
    type: 'binding_state',
    binding: { ...binding, state: 'streaming' }
  });
  assert.equal(streaming.sinks[0].output, 'source');
  const released = reduceSources(streaming, { type: 'binding_removed', binding });
  assert.equal(released.sinks[0].binding, null);
  assert.equal(released.sinks[0].output, 'black');
});

test('source reconnect replaces its live entry', () => {
  const source = {
    id: 'src_a',
    owner: 'alice',
    agent: 'browser',
    label: 'Phone',
    streams: [],
    connected_at: '2026-08-22T00:00:00Z'
  };
  const initial: SourcesSnapshot = { sinks: [], sources: [source], bindings: [] };
  const replaced = reduceSources(initial, {
    type: 'source_added',
    source: { ...source, label: 'Tablet' }
  });
  assert.equal(replaced.sources.length, 1);
  assert.equal(replaced.sources[0].label, 'Tablet');
});

test('source close rejects readiness and reports unavailable', { timeout: 1000 }, async () => {
  const values = new Map<string, string>();
  Object.defineProperty(globalThis, 'window', {
    configurable: true,
    value: {
      clearTimeout,
      dispatchEvent() {},
      location: { hostname: 'localhost', port: '80', protocol: 'http:' },
      setTimeout
    }
  });
  Object.defineProperty(globalThis, 'sessionStorage', {
    configurable: true,
    value: {
      getItem: (key: string) => values.get(key) || null,
      removeItem: (key: string) => values.delete(key),
      setItem: (key: string, value: string) => values.set(key, value)
    }
  });

  class ClosingSocket {
    static readonly OPEN = 1;
    bufferedAmount = 0;
    readyState = 0;
    binaryType = '';
    onclose?: (event: { code: number }) => void;
    onerror?: () => void;
    onmessage?: (event: MessageEvent) => void;
    onopen?: () => void;

    constructor() {
      queueMicrotask(() => this.onclose?.({ code: 1011 }));
    }

    close() {}
    send() {}
  }

  Object.defineProperty(globalThis, 'WebSocket', { configurable: true, value: ClosingSocket });
  const states: string[] = [];
  const errors: string[] = [];
  const client = new BrowserSourceClient(
    'alice',
    {
      onConnection: (state) => states.push(state),
      onError: (_sink, message) => errors.push(message),
      onOwned() {},
      onRefused() {},
      onRevoked() {},
      onSnapshot() {}
    },
    () => new ClosingSocket() as unknown as WebSocket
  );

  await assert.rejects(
    client.setOffers([{ id: 'cam_a', deviceID: 'camera-a', kind: 'camera', label: 'Camera' }]),
    /Media source unavailable/
  );
  client.close();
  assert.ok(states.includes('disconnected'));
  assert.ok(errors.includes('Media source unavailable'));
});

class ControlSocket {
  static readonly OPEN = 1;
  bufferedAmount = 0;
  readyState = 1;
  binaryType = '';
  sent: string[] = [];
  onclose?: (event: { code: number }) => void;
  onerror?: () => void;
  onmessage?: (event: MessageEvent) => void;
  onopen?: () => void;

  close() {}

  send(data: string) {
    this.sent.push(data);
  }

  deliver(payload: unknown) {
    this.onmessage?.({ data: JSON.stringify(payload) } as MessageEvent);
  }
}

function installBrowserGlobals() {
  const values = new Map<string, string>();
  Object.defineProperty(globalThis, 'window', {
    configurable: true,
    value: { clearTimeout, dispatchEvent() {}, setTimeout }
  });
  Object.defineProperty(globalThis, 'sessionStorage', {
    configurable: true,
    value: {
      getItem: (key: string) => values.get(key) || null,
      removeItem: (key: string) => values.delete(key),
      setItem: (key: string, value: string) => values.set(key, value)
    }
  });
  Object.defineProperty(globalThis, 'WebSocket', { configurable: true, value: ControlSocket });
}

const cameraOffer: DeviceOffer = {
  id: 'cam_a',
  deviceID: 'camera-a',
  kind: 'camera',
  label: 'Camera'
};

type Callbacks = ConstructorParameters<typeof BrowserSourceClient>[1];

const flush = () => new Promise((resolve) => setTimeout(resolve, 0));

async function connectedClient(socket: ControlSocket, callbacks: Partial<Callbacks> = {}) {
  const client = new BrowserSourceClient(
    'alice',
    {
      onConnection() {},
      onOwned() {},
      onError() {},
      onRefused() {},
      onRevoked() {},
      onSnapshot() {},
      ...callbacks
    },
    () => socket as unknown as WebSocket
  );
  const ready = client.setOffers([cameraOffer]);
  socket.onopen?.();
  socket.deliver({
    type: 'source_ready',
    source: { id: 'src_browser', owner: 'alice', agent: 'browser', label: 'Browser', streams: [] },
    snapshot: { sinks: [], sources: [], bindings: [] }
  });
  await ready;
  return client;
}

test('a revoked binding reaches the browser with its reason', { timeout: 1000 }, async () => {
  installBrowserGlobals();
  const socket = new ControlSocket();
  const owned: string[][] = [];
  const revocations: Array<[string, string]> = [];
  const client = await connectedClient(socket, {
    onOwned: (sinks: Set<string>) => owned.push([...sinks]),
    onRevoked: (sinkID: string, reason: string) => revocations.push([sinkID, reason])
  });

  const claiming = client.claim('uvc.cam0', cameraOffer);
  await flush();
  socket.deliver({
    type: 'claimed',
    binding: { sink_id: 'uvc.cam0', stream_id: 'cam_a' },
    token: 'lease'
  });
  await claiming;
  assert.deepEqual(owned[owned.length - 1], ['uvc.cam0']);
  assert.equal(client.sourceId(), 'src_browser');

  socket.deliver({ type: 'released', sink_id: 'uvc.cam0', reason: 'admin_disconnect' });
  assert.deepEqual(revocations, [['uvc.cam0', 'admin_disconnect']]);
  assert.deepEqual(owned[owned.length - 1], []);
  client.close();
});

test('a refused claim keeps the holder that the server named', { timeout: 1000 }, async () => {
  installBrowserGlobals();
  const socket = new ControlSocket();
  const refusals: ClaimRefusal[] = [];
  const client = await connectedClient(socket, {
    onRefused: (refusal: ClaimRefusal) => refusals.push(refusal)
  });

  const claiming = client.claim('uvc.cam0', cameraOffer);
  await flush();
  socket.deliver({
    type: 'claim_refused',
    message: 'slot_occupied',
    sink_id: 'uvc.cam0',
    owner: 'bob',
    source_label: 'Pixel',
    since: '2026-08-23T00:00:00Z',
    takeover: 'immediate'
  });
  await assert.rejects(claiming, /In use by bob from Pixel/);
  assert.deepEqual(refusals, [
    {
      sink_id: 'uvc.cam0',
      owner: 'bob',
      source_label: 'Pixel',
      since: '2026-08-23T00:00:00Z',
      takeover: 'immediate'
    }
  ]);
  client.close();
});

test('an owned release is not reported as a revocation', { timeout: 1000 }, async () => {
  installBrowserGlobals();
  const socket = new ControlSocket();
  const revocations: string[] = [];
  const client = await connectedClient(socket, {
    onRevoked: (sinkID: string) => revocations.push(sinkID)
  });

  const claiming = client.claim('uvc.cam0', cameraOffer);
  await flush();
  socket.deliver({
    type: 'claimed',
    binding: { sink_id: 'uvc.cam0', stream_id: 'cam_a' },
    token: 'lease'
  });
  await claiming;
  const releasing = client.release('uvc.cam0');
  await flush();
  socket.deliver({ type: 'released', sink_id: 'uvc.cam0' });
  await releasing;
  assert.deepEqual(revocations, []);
  client.close();
});

test('slot rows carry the name the host reads apart from the editable label', () => {
  const sinks: SourceSink[] = [
    {
      id: 'uvc.cam0',
      kind: 'camera',
      label: 'NanoKVM Camera 1',
      host_name: 'UVC Camera',
      slot: 0,
      demand: { streaming: false },
      output: 'idle',
      binding: null
    },
    {
      id: 'uac2.mic0',
      kind: 'microphone',
      label: 'NanoKVM Microphone 1',
      slot: 0,
      demand: { streaming: false },
      output: 'idle',
      binding: null
    },
    {
      id: 'usb.hybrid',
      kind: 'usb_device',
      label: 'Browser USB',
      slot: 0,
      demand: { streaming: false },
      output: 'idle',
      binding: null
    }
  ];
  const rows = mediaSlotRows(sinks);
  assert.deepEqual(
    rows.map((row) => [row.kind, row.label, row.hostName]),
    [
      ['camera', 'NanoKVM Camera 1', 'UVC Camera'],
      ['microphone', 'NanoKVM Microphone 1', '']
    ]
  );
});

test('slot rows renumber contiguously from zero within each kind', () => {
  const rows = [
    { key: 'a', kind: 'microphone' as const, label: 'Podcast', hostName: '' },
    { key: 'b', kind: 'camera' as const, label: 'Desk', hostName: '' },
    { key: 'c', kind: 'camera' as const, label: 'Rack', hostName: '' }
  ];
  assert.deepEqual(mediaSlots(rows), [
    { id: 'uac2.mic0', kind: 'microphone', label: 'Podcast' },
    { id: 'uvc.cam0', kind: 'camera', label: 'Desk' },
    { id: 'uvc.cam1', kind: 'camera', label: 'Rack' }
  ]);
});
