import assert from 'node:assert/strict';
import test from 'node:test';

import type { SourceSink, SourcesSnapshot } from '../../../../api/sources.ts';
import { BrowserSourceClient } from './client.ts';
import { PcmPacketizer } from './pcm.ts';
import { reduceSources } from './state.ts';
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
