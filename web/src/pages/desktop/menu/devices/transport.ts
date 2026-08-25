export type MediaFrameKind = 'mjpeg' | 'pcm_s16le_mono_48k';

export type MediaFrame = {
  kind: MediaFrameKind;
  sequence: number;
  timestampUS: number;
  sinkID: string;
  streamID: string;
  payload: ArrayBuffer;
};

export type FrameAck = {
  type: 'frame_ack';
  sink_id: string;
  stream_id: string;
  sequence: number;
};

export type FrameError = {
  type: 'frame_error';
  sink_id: string;
  stream_id: string;
  sequence: number;
  message: string;
};

const headerBytes = 26;
const textEncoder = new TextEncoder();
const maxPayload = { mjpeg: 2 << 20, pcm_s16le_mono_48k: 3840 } satisfies Record<
  MediaFrameKind,
  number
>;
const frameKind = { mjpeg: 1, pcm_s16le_mono_48k: 2 } satisfies Record<MediaFrameKind, number>;

export function encodeMediaFrame(frame: MediaFrame) {
  const sink = textEncoder.encode(frame.sinkID);
  const stream = textEncoder.encode(frame.streamID);
  const payload = new Uint8Array(frame.payload);
  if (sink.length === 0 || sink.length > 64 || stream.length === 0 || stream.length > 64) {
    throw new Error('media frame id length');
  }
  if (payload.length === 0 || payload.length > maxPayload[frame.kind]) {
    throw new Error('media frame payload length');
  }
  if (!Number.isSafeInteger(frame.timestampUS) || frame.timestampUS < 0) {
    throw new Error('media frame timestamp');
  }

  const bytes = new Uint8Array(headerBytes + sink.length + stream.length + payload.length);
  const view = new DataView(bytes.buffer);
  bytes.set([0x4e, 0x4b, 0x4d, 0x46], 0);
  view.setUint8(4, 1);
  view.setUint8(5, frameKind[frame.kind]);
  view.setUint16(6, 0);
  view.setUint32(8, frame.sequence);
  view.setBigUint64(12, BigInt(frame.timestampUS));
  view.setUint8(20, sink.length);
  view.setUint8(21, stream.length);
  view.setUint32(22, payload.length);
  bytes.set(sink, headerBytes);
  bytes.set(stream, headerBytes + sink.length);
  bytes.set(payload, headerBytes + sink.length + stream.length);
  return bytes.buffer;
}

const textDecoder = new TextDecoder('utf-8', { fatal: true });
const kindNames = { 1: 'mjpeg', 2: 'pcm_s16le_mono_48k' } as Record<number, MediaFrameKind>;

// The exact inverse of encodeMediaFrame, and the mirror of parseMediaFrame in
// server/service/sources/frame.go. A speaker slot sends frames the other way,
// so the browser has to read the same header it writes. Anything malformed
// returns undefined; a stream that cannot be trusted is never played.
export function decodeMediaFrame(data: ArrayBuffer): MediaFrame | undefined {
  if (data.byteLength < headerBytes) return undefined;
  const bytes = new Uint8Array(data);
  if (bytes[0] !== 0x4e || bytes[1] !== 0x4b || bytes[2] !== 0x4d || bytes[3] !== 0x46) {
    return undefined;
  }
  const view = new DataView(data);
  if (view.getUint8(4) !== 1 || view.getUint16(6) !== 0) return undefined;
  const kind = kindNames[view.getUint8(5)];
  if (!kind) return undefined;
  const sinkLength = view.getUint8(20);
  const streamLength = view.getUint8(21);
  const payloadLength = view.getUint32(22);
  if (sinkLength === 0 || sinkLength > 64 || streamLength === 0 || streamLength > 64) {
    return undefined;
  }
  if (payloadLength === 0 || payloadLength > maxPayload[kind]) return undefined;
  if (data.byteLength !== headerBytes + sinkLength + streamLength + payloadLength) {
    return undefined;
  }
  const sinkEnd = headerBytes + sinkLength;
  const streamEnd = sinkEnd + streamLength;
  try {
    return {
      kind,
      sequence: view.getUint32(8),
      timestampUS: Number(view.getBigUint64(12)),
      sinkID: textDecoder.decode(bytes.subarray(headerBytes, sinkEnd)),
      streamID: textDecoder.decode(bytes.subarray(sinkEnd, streamEnd)),
      payload: data.slice(streamEnd)
    };
  } catch {
    return undefined;
  }
}

export function mediaTimestampUS() {
  return Math.round((performance.timeOrigin + performance.now()) * 1000);
}
