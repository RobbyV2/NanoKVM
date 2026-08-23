export const webUSBHeaderBytes = 32;
export const webUSBMaxPayload = 64 << 10;

const protectedClasses = new Set([0x01, 0x03, 0x08, 0x09, 0x0b, 0x0d, 0x0e, 0x10, 0xe0]);
const supportedClasses = new Set([0x02, 0x07, 0x0a, 0xff]);

export type WirePacket = {
  type: number;
  status: number;
  id: number;
  endpoint: number;
  requestType: number;
  request: number;
  transfer: number;
  value: number;
  index: number;
  declaredLength: number;
  timeoutMS: number;
  payload: Uint8Array;
};

export function decodeWebUSBFrame(buffer: ArrayBuffer): WirePacket {
  if (buffer.byteLength < webUSBHeaderBytes) throw new Error('Short USB frame');
  const data = new Uint8Array(buffer);
  if (String.fromCharCode(...data.subarray(0, 4)) !== 'NKUF' || data[4] !== 1)
    throw new Error('Invalid USB frame');
  const view = new DataView(buffer);
  const length = view.getUint32(24);
  if (length > webUSBMaxPayload || webUSBHeaderBytes + length !== buffer.byteLength)
    throw new Error('Invalid USB frame length');
  return {
    type: data[5],
    status: data[6],
    id: view.getUint32(8),
    endpoint: data[12],
    requestType: data[13],
    request: data[14],
    transfer: data[15],
    value: view.getUint16(16),
    index: view.getUint16(18),
    declaredLength: view.getUint32(20),
    timeoutMS: view.getUint32(28),
    payload: data.slice(webUSBHeaderBytes)
  };
}

export function encodeWebUSBFrame(packet: WirePacket) {
  if (packet.payload.length > webUSBMaxPayload || packet.declaredLength > webUSBMaxPayload)
    throw new Error('USB frame is too large');
  const buffer = new ArrayBuffer(webUSBHeaderBytes + packet.payload.length);
  const data = new Uint8Array(buffer);
  data.set([0x4e, 0x4b, 0x55, 0x46, 1, packet.type, packet.status, 0]);
  const view = new DataView(buffer);
  view.setUint32(8, packet.id);
  data[12] = packet.endpoint;
  data[13] = packet.requestType;
  data[14] = packet.request;
  data[15] = packet.transfer;
  view.setUint16(16, packet.value);
  view.setUint16(18, packet.index);
  view.setUint32(20, packet.declaredLength);
  view.setUint32(24, packet.payload.length);
  view.setUint32(28, packet.timeoutMS);
  data.set(packet.payload, webUSBHeaderBytes);
  return buffer;
}

export function classifyWebUSBConfiguration(config: Uint8Array) {
  const selected = new Set<number>();
  const alternates = new Set<number>();
  walk(config, (item) => {
    if (item[1] !== 4 || item.length < 9) return;
    if (item[3] !== 0) {
      alternates.add(item[2]);
      return;
    }
    if (!protectedClasses.has(item[5]) && supportedClasses.has(item[5])) selected.add(item[2]);
  });
  for (const number of selected) {
    if (alternates.has(number)) throw new Error(`Interface ${number} uses alternate settings`);
  }
  return [...selected].sort((a, b) => a - b);
}

function walk(config: Uint8Array, callback: (item: Uint8Array) => void) {
  for (let offset = 0; offset < config.length; ) {
    const length = config[offset];
    if (length < 2 || offset + length > config.length)
      throw new Error(`Malformed descriptor at ${offset}`);
    callback(config.subarray(offset, offset + length));
    offset += length;
  }
}
