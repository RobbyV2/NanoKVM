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

test('a context that is not 48 kHz still yields 48 kHz packets at 50 a second', () => {
  // The gadget only accepts 20 ms mono S16LE 48000 Hz packets, and an
  // AudioContext is entitled to run at whatever its output device runs at. Four
  // seconds of 44100 Hz input therefore has to come back out as four seconds of
  // 48000 Hz packets, or the microphone drifts against the host's clock until
  // it starves. One packet of pipeline lag is allowed; a drifting rate is not.
  const packetizer = new PcmPacketizer(44100);
  const chunks = 200;
  let packets = 0;
  for (let chunk = 0; chunk < chunks; chunk++) {
    const samples = new Float32Array(882);
    for (let i = 0; i < samples.length; i++) {
      samples[i] = Math.sin((2 * Math.PI * 1000 * (chunk * 882 + i)) / 44100);
    }
    for (const packet of packetizer.push(samples)) {
      assert.equal(packet.byteLength, 1920);
      packets++;
    }
  }
  assert.ok(
    packets === chunks || packets === chunks - 1,
    `emitted ${packets} packets, want ${chunks}`
  );
});

test('each packet owns its bytes', () => {
  // The packetizer writes into one buffer and hands it on. If the next packet
  // were written into the same buffer, every packet already queued for the
  // socket would be overwritten with the newest audio and the host would hear
  // one 20 ms slice repeated.
  const packetizer = new PcmPacketizer(48000);
  const emitted: ArrayBuffer[] = [];
  for (const level of [0.25, -0.5, 0.75]) {
    for (const packet of packetizer.push(new Float32Array(960).fill(level))) emitted.push(packet);
  }
  assert.equal(emitted.length, 2, 'want one packet per whole 20 ms, less one of lag');
  assert.notEqual(emitted[0], emitted[1]);
  assert.equal(new DataView(emitted[0]).getInt16(0, true), 8192);
  assert.equal(new DataView(emitted[1]).getInt16(0, true), -16384);
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

const microphoneOffer: DeviceOffer = {
  id: 'mic_a',
  deviceID: 'microphone-a',
  kind: 'microphone',
  label: 'Microphone'
};

type Callbacks = ConstructorParameters<typeof BrowserSourceClient>[1];

const flush = () => new Promise((resolve) => setTimeout(resolve, 0));

async function connectedClient(
  socket: ControlSocket,
  callbacks: Partial<Callbacks> = {},
  offers: DeviceOffer[] = [cameraOffer]
) {
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
  const ready = client.setOffers(offers);
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

// A reload is the recovery path the operator is told to use, so it has to work
// even when the new page beats its own goodbye to the server. The registry only
// accepts a resume once it has seen the previous socket go; until then the
// binding is still claimed and the token it holds is refused outright. Giving up
// on that first refusal used to strand the slot on a source that no longer
// exists, and every later claim was refused until the whole lease grace ran out.
class ResumeSocket extends ControlSocket {
  attempts = 0;
  succeedOn: number;
  refusal: string;

  constructor(succeedOn: number, refusal = 'invalid lease token') {
    super();
    this.succeedOn = succeedOn;
    this.refusal = refusal;
  }

  send(data: string) {
    super.send(data);
    const message = JSON.parse(data) as { type: string; sink_id?: string; stream_id?: string };
    if (message.type !== 'resume') return;
    this.attempts++;
    const attempt = this.attempts;
    queueMicrotask(() => {
      if (attempt < this.succeedOn) {
        this.deliver({ type: 'error', message: this.refusal });
        return;
      }
      this.deliver({
        type: 'resumed',
        binding: { sink_id: message.sink_id, stream_id: message.stream_id }
      });
    });
  }
}

function storeLease() {
  sessionStorage.setItem(
    'nanokvm-media-leases:alice',
    JSON.stringify([
      {
        sinkID: 'uvc.cam0',
        streamID: cameraOffer.id,
        token: 'lease',
        deviceID: cameraOffer.deviceID,
        kind: 'camera'
      }
    ])
  );
}

test(
  'a lease refused while the server still holds the old socket is resumed',
  { timeout: 5000 },
  async () => {
    installBrowserGlobals();
    storeLease();
    const socket = new ResumeSocket(2);
    const owned: string[][] = [];
    const client = await connectedClient(socket, {
      onOwned: (sinks: Set<string>) => owned.push([...sinks])
    });

    assert.equal(socket.attempts, 2, 'want the refusal retried across the detach window');
    assert.deepEqual(owned[owned.length - 1], ['uvc.cam0']);
    assert.deepEqual(
      client.selections().map((lease) => lease.sinkID),
      ['uvc.cam0'],
      'want the lease kept once it resumed'
    );
    client.close();
  }
);

test('a refusal that will not change gives the lease up at once', { timeout: 5000 }, async () => {
  // Waiting out a refusal only helps while the server is still catching up with
  // a socket that has gone. "Forbidden" is an answer, not a race, and retrying
  // it would delay the clean session the operator refreshed for.
  installBrowserGlobals();
  storeLease();
  const socket = new ResumeSocket(Number.POSITIVE_INFINITY, 'forbidden');
  const errors: Array<[string, string]> = [];
  const client = await connectedClient(socket, {
    onError: (sinkID: string, message: string) => errors.push([sinkID, message])
  });

  assert.equal(socket.attempts, 1);
  assert.deepEqual(client.selections(), [], 'want the dead lease dropped so the page starts clean');
  assert.ok(errors.some(([sinkID, message]) => sinkID === 'uvc.cam0' && message === 'forbidden'));
  client.close();
});

test(
  'a client closed for the page unload connects again when the page returns',
  { timeout: 1000 },
  async () => {
    // The page closes this socket on its way out so the server hears the goodbye
    // before the next page says hello. A page that comes back from the back and
    // forward cache still holds the offers that justified the connection, so the
    // close must not be the end of it.
    installBrowserGlobals();
    const sockets: ControlSocket[] = [];
    const client = new BrowserSourceClient(
      'alice',
      {
        onConnection() {},
        onOwned() {},
        onError() {},
        onRefused() {},
        onRevoked() {},
        onSnapshot() {}
      },
      () => {
        const socket = new ControlSocket();
        sockets.push(socket);
        return socket as unknown as WebSocket;
      }
    );

    const ready = client.setOffers([cameraOffer]);
    sockets[0].onopen?.();
    sockets[0].deliver({
      type: 'source_ready',
      source: {
        id: 'src_browser',
        owner: 'alice',
        agent: 'browser',
        label: 'Browser',
        streams: []
      },
      snapshot: { sinks: [], sources: [], bindings: [] }
    });
    await ready;

    client.close();
    assert.equal(sockets.length, 1);
    client.reconnect();
    assert.equal(sockets.length, 2, 'want a fresh socket once the page is back');
    client.close();
  }
);

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

// The in-flight window is a throughput knob, not an implementation detail. At
// one frame the camera waits out a whole round trip - encode, send, the
// device's work, the ack - before capturing the next. Measured against the
test(
  'audio keeps sending while a whole gadget queue is still unacknowledged',
  { timeout: 1000 },
  async () => {
    // An unacknowledged-frame window caps a continuous stream at window/RTT.
    // Audio is 50 packets a second, so a window of four let the browser send no
    // more than 4/RTT of them and silently dropped the rest on the floor -
    // frames the capture chain had already produced and nobody was ever told
    // about. The window has to cover an ordinary wireless round trip; real
    // backpressure is bufferedAmount, which sendFrame checks first.
    installBrowserGlobals();
    const socket = new ControlSocket();
    const client = await connectedClient(socket, {}, [cameraOffer, microphoneOffer]);

    const claiming = client.claim('uac2.mic0', microphoneOffer);
    await flush();
    socket.deliver({
      type: 'claimed',
      binding: { sink_id: 'uac2.mic0', stream_id: 'mic_a' },
      token: 'lease'
    });
    await claiming;

    const payload = new ArrayBuffer(1920);
    const accepted: boolean[] = [];
    for (let i = 0; i < 9; i++) {
      accepted.push(client.sendFrame('uac2.mic0', 'mic_a', 'pcm_s16le_mono_48k', payload));
    }
    assert.deepEqual(
      accepted,
      [true, true, true, true, true, true, true, true, false],
      'want eight 20 ms packets - 160 ms of round trip - in flight before backpressure'
    );
  }
);

// hardware with 135KB frames: window 1 gave 6.6 fps / 0.89 MB/s, window 2 gave
// 9.3 fps / 1.25 MB/s, window 3 gave nothing more because the device is the
// limit by then. Pin it so a change back to one is a decision, not an accident.
test(
  'a second video frame may be in flight before the first is acknowledged',
  { timeout: 1000 },
  async () => {
    installBrowserGlobals();
    const socket = new ControlSocket();
    const client = await connectedClient(socket);

    const claiming = client.claim('uvc.cam0', cameraOffer);
    await flush();
    socket.deliver({
      type: 'claimed',
      binding: { sink_id: 'uvc.cam0', stream_id: 'cam_a' },
      token: 'lease'
    });
    await claiming;

    const payload = new ArrayBuffer(16);
    const accepted = [
      client.sendFrame('uvc.cam0', 'cam_a', 'mjpeg', payload),
      client.sendFrame('uvc.cam0', 'cam_a', 'mjpeg', payload),
      client.sendFrame('uvc.cam0', 'cam_a', 'mjpeg', payload)
    ];

    assert.deepEqual(accepted, [true, true, false], 'want two frames in flight, then backpressure');
  }
);
