import assert from 'node:assert/strict';
import test from 'node:test';

import type { SourceSink } from '../../../../api/sources.ts';
import { mediaSlots, reduceSources } from './state.ts';
import { decodeMediaFrame, encodeMediaFrame } from './transport.ts';

// A speaker sends frames the other way, so the browser has to read the same
// header it writes. If the two ever disagree the audio is silently unplayable.
test('a speaker frame survives the round trip through the NKMF header', () => {
  const payload = new Uint8Array(1920);
  payload[0] = 0x7f;
  payload[1919] = 0x80;
  const encoded = encodeMediaFrame({
    kind: 'pcm_s16le_mono_48k',
    sequence: 0x0a0b0c0d,
    timestampUS: 0x0102030405,
    sinkID: 'uac2.spk0',
    streamID: 'spk_browser',
    payload: payload.buffer
  });
  const frame = decodeMediaFrame(encoded);
  assert.ok(frame);
  assert.equal(frame.kind, 'pcm_s16le_mono_48k');
  assert.equal(frame.sequence, 0x0a0b0c0d);
  assert.equal(frame.timestampUS, 0x0102030405);
  assert.equal(frame.sinkID, 'uac2.spk0');
  assert.equal(frame.streamID, 'spk_browser');
  assert.deepEqual([...new Uint8Array(frame.payload)], [...payload]);
});

test('a frame the gadget could not have sent is refused rather than played', () => {
  const good = new Uint8Array(
    encodeMediaFrame({
      kind: 'pcm_s16le_mono_48k',
      sequence: 1,
      timestampUS: 1,
      sinkID: 'uac2.spk0',
      streamID: 'spk_browser',
      payload: new ArrayBuffer(1920)
    })
  );
  assert.ok(decodeMediaFrame(good.buffer));

  const wrongMagic = good.slice();
  wrongMagic[0] = 0x4e;
  wrongMagic[1] = 0x4b;
  wrongMagic[2] = 0x55;
  wrongMagic[3] = 0x46;
  assert.equal(decodeMediaFrame(wrongMagic.buffer), undefined, 'a WebUSB transfer is not audio');

  const wrongVersion = good.slice();
  wrongVersion[4] = 2;
  assert.equal(decodeMediaFrame(wrongVersion.buffer), undefined);

  const wrongKind = good.slice();
  wrongKind[5] = 9;
  assert.equal(decodeMediaFrame(wrongKind.buffer), undefined);

  const flagged = good.slice();
  flagged[6] = 1;
  assert.equal(decodeMediaFrame(flagged.buffer), undefined);

  assert.equal(decodeMediaFrame(good.slice(0, 20).buffer), undefined);
  assert.equal(decodeMediaFrame(good.slice(0, good.length - 1).buffer), undefined);
});

test('speaker slots renumber apart from microphones', () => {
  const rows = [
    { key: 'a', kind: 'speaker' as const, label: 'Desk speaker', hostName: '' },
    { key: 'b', kind: 'microphone' as const, label: 'Podcast', hostName: '' },
    { key: 'c', kind: 'speaker' as const, label: 'Rack speaker', hostName: '' }
  ];
  assert.deepEqual(mediaSlots(rows), [
    { id: 'uac2.spk0', kind: 'speaker', label: 'Desk speaker' },
    { id: 'uac2.mic0', kind: 'microphone', label: 'Podcast' },
    { id: 'uac2.spk1', kind: 'speaker', label: 'Rack speaker' }
  ]);
});

// A speaker generates nothing, so it never reports the silence a microphone
// slot falls back to when the host is reading and no browser is bound.
test('a demanded speaker with no listener is idle, never silence', () => {
  const speaker: SourceSink = {
    id: 'uac2.spk0',
    kind: 'speaker',
    label: 'Speaker 1',
    slot: 0,
    demand: { streaming: true },
    output: 'idle',
    binding: null
  };
  const microphone: SourceSink = { ...speaker, id: 'uac2.mic0', kind: 'microphone' };
  const snapshot = reduceSources(
    { sinks: [], sources: [], bindings: [] },
    { type: 'snapshot', snapshot: { sinks: [speaker, microphone], sources: [], bindings: [] } }
  );
  const withDemand = reduceSources(snapshot, {
    type: 'demand',
    sink_id: 'uac2.mic0',
    demand: { streaming: true }
  });
  assert.equal(withDemand.sinks.find((sink) => sink.id === 'uac2.mic0')?.output, 'silence');

  const speakerDemand = reduceSources(withDemand, {
    type: 'demand',
    sink_id: 'uac2.spk0',
    demand: { streaming: true }
  });
  assert.equal(speakerDemand.sinks.find((sink) => sink.id === 'uac2.spk0')?.output, 'idle');
});
