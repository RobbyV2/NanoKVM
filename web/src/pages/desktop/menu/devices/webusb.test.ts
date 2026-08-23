import assert from 'node:assert/strict';
import test from 'node:test';

import {
  classifyWebUSBConfiguration,
  decodeWebUSBFrame,
  encodeWebUSBFrame
} from './webusb-wire.ts';
import {
  captureWebUSBDevice,
  WebUSBRelay,
  type USBDeviceLike,
  type USBProvider
} from './webusb.ts';

test('NKUF rejects truncated and oversized frames', () => {
  assert.throws(() => decodeWebUSBFrame(new ArrayBuffer(31)), /Short USB frame/);
  const buffer = new ArrayBuffer(32);
  const bytes = new Uint8Array(buffer);
  bytes.set([0x4e, 0x4b, 0x55, 0x46, 1, 2]);
  new DataView(buffer).setUint32(24, 1);
  assert.throws(() => decodeWebUSBFrame(buffer), /length/);
});

test('NKUF round trips bounded payloads', () => {
  const packet = {
    type: 2,
    status: 0,
    id: 7,
    endpoint: 0x81,
    requestType: 0,
    request: 0,
    transfer: 2,
    value: 0,
    index: 0,
    declaredLength: 4,
    timeoutMS: 5000,
    payload: new Uint8Array([1, 2, 3, 4])
  };
  assert.deepEqual(decodeWebUSBFrame(encodeWebUSBFrame(packet)), packet);
});

test('protected interfaces stay out of the WebUSB projection', () => {
  const config = new Uint8Array([
    9, 2, 34, 0, 2, 1, 0, 0x80, 50, 9, 4, 0, 0, 1, 0xff, 0, 0, 0, 7, 5, 0x81, 2, 64, 0, 0, 9, 4, 1,
    0, 0, 0x03, 1, 1, 0
  ]);
  assert.deepEqual(classifyWebUSBConfiguration(config), [0]);
});

test('alternate settings are refused for selected interfaces', () => {
  const config = new Uint8Array([
    9, 2, 27, 0, 1, 1, 0, 0x80, 50, 9, 4, 0, 0, 0, 0xff, 0, 0, 0, 9, 4, 0, 1, 0, 0xff, 0, 0, 0
  ]);
  assert.throws(() => classifyWebUSBConfiguration(config), /alternate/);
});

test('mock WebUSB capture preserves descriptors and selects vendor bulk', async () => {
  const deviceDescriptor = new Uint8Array([
    18, 1, 0, 2, 0, 0, 0, 64, 0x34, 0x12, 0x78, 0x56, 0, 1, 0, 0, 0, 1
  ]);
  const configuration = new Uint8Array([
    9, 2, 25, 0, 1, 1, 0, 0x80, 50, 9, 4, 0, 0, 1, 0xff, 0, 0, 0, 7, 5, 0x81, 2, 64, 0, 0
  ]);
  const mock = {
    vendorId: 0x1234,
    productId: 0x5678,
    manufacturerName: 'Fixture',
    productName: 'Debug adapter',
    opened: false,
    configurations: [{ configurationValue: 1, interfaces: [] }],
    async open() {
      this.opened = true;
    },
    async close() {},
    async reset() {},
    async selectConfiguration(value: number) {
      this.configuration = { configurationValue: value, interfaces: [] };
    },
    async claimInterface() {},
    async releaseInterface() {},
    async clearHalt() {},
    async controlTransferIn(setup: { value: number }, length: number) {
      const type = setup.value >> 8;
      const source = type === 1 ? deviceDescriptor : configuration;
      const bytes = source.slice(0, length);
      return { status: 'ok' as const, data: new DataView(bytes.buffer) };
    },
    async controlTransferOut() {
      return { status: 'ok' as const, bytesWritten: 0 };
    },
    async transferIn() {
      return { status: 'ok' as const, data: new DataView(new ArrayBuffer(0)) };
    },
    async transferOut(_endpoint: number, data: BufferSource) {
      return { status: 'ok' as const, bytesWritten: data.byteLength };
    }
  } as USBDeviceLike;
  const captured = await captureWebUSBDevice(mock);
  assert.deepEqual(captured.interfaces, [0]);
  assert.equal(captured.profile.device.vendor_id, '0x1234');
  assert.equal(
    captured.profile.descriptors?.device,
    Buffer.from(deviceDescriptor).toString('base64')
  );
});

test('mock WebUSB capture rejects excessive configuration counts', async () => {
  const deviceDescriptor = new Uint8Array([
    18, 1, 0, 2, 0, 0, 0, 64, 0x34, 0x12, 0x78, 0x56, 0, 1, 0, 0, 0, 9
  ]);
  const mock = {
    vendorId: 0x1234,
    productId: 0x5678,
    opened: true,
    configurations: [],
    async open() {},
    async close() {},
    async reset() {},
    async selectConfiguration() {},
    async claimInterface() {},
    async releaseInterface() {},
    async clearHalt() {},
    async controlTransferIn() {
      return { status: 'ok' as const, data: new DataView(deviceDescriptor.buffer) };
    },
    async controlTransferOut() {
      return { status: 'ok' as const, bytesWritten: 0 };
    },
    async transferIn() {
      return { status: 'ok' as const, data: new DataView(new ArrayBuffer(0)) };
    },
    async transferOut(_endpoint: number, data: BufferSource) {
      return { status: 'ok' as const, bytesWritten: data.byteLength };
    }
  } as USBDeviceLike;
  await assert.rejects(() => captureWebUSBDevice(mock), /1 to 8 configurations/);
});

const relayDescriptor = new Uint8Array([
  18, 1, 0, 2, 0, 0, 0, 64, 0x34, 0x12, 0x78, 0x56, 0, 1, 0, 0, 0, 1
]);
const relayConfiguration = new Uint8Array([
  9, 2, 41, 0, 2, 1, 0, 0x80, 50, 9, 4, 0, 0, 1, 0xff, 0, 0, 0, 7, 5, 0x81, 2, 64, 0, 0, 9, 4, 1, 0,
  1, 0xff, 0, 0, 0, 7, 5, 0x02, 2, 64, 0, 0
]);

function frame(overrides: Partial<Parameters<typeof encodeWebUSBFrame>[0]>) {
  return encodeWebUSBFrame({
    type: 1,
    status: 0,
    id: 1,
    endpoint: 0,
    requestType: 0,
    request: 0,
    transfer: 0,
    value: 0,
    index: 0,
    declaredLength: 0,
    timeoutMS: 5000,
    payload: new Uint8Array(),
    ...overrides
  });
}

function controlFrame(id: number) {
  return frame({ type: 1, id, requestType: 0x80, request: 6, value: 1 << 8, declaredLength: 18 });
}

function transferFrame(id: number) {
  return frame({ type: 2, id, endpoint: 0x81, declaredLength: 8 });
}

async function relayFixture() {
  const state = {
    opened: 0,
    closed: false,
    released: [] as number[],
    releaseFails: new Set<number>(),
    pendingIn: [] as ((result: { status: 'ok'; data: DataView }) => void)[]
  };
  const device: USBDeviceLike = {
    vendorId: 0x1234,
    productId: 0x5678,
    productName: 'Debug adapter',
    opened: false,
    configurations: [],
    async open() {
      state.opened++;
      device.opened = true;
    },
    async close() {
      state.closed = true;
      device.opened = false;
    },
    async reset() {},
    async selectConfiguration(value) {
      device.configuration = {
        configurationValue: value,
        interfaces: [
          {
            interfaceNumber: 0,
            alternates: [
              {
                alternateSetting: 0,
                interfaceClass: 0xff,
                endpoints: [{ endpointNumber: 1, direction: 'in', type: 'bulk' }]
              }
            ]
          },
          {
            interfaceNumber: 1,
            alternates: [
              {
                alternateSetting: 0,
                interfaceClass: 0xff,
                endpoints: [{ endpointNumber: 2, direction: 'out', type: 'bulk' }]
              }
            ]
          }
        ]
      };
    },
    async claimInterface() {},
    async releaseInterface(number) {
      if (state.releaseFails.has(number)) throw new Error('interface is busy');
      state.released.push(number);
    },
    async clearHalt() {},
    async controlTransferIn(setup, length) {
      const source = setup.value >> 8 === 1 ? relayDescriptor : relayConfiguration;
      const bytes = source.slice(0, length);
      return { status: 'ok', data: new DataView(bytes.buffer) };
    },
    async controlTransferOut() {
      return { status: 'ok', bytesWritten: 0 };
    },
    transferIn() {
      return new Promise((resolve) => state.pendingIn.push(resolve));
    },
    async transferOut(_endpoint, data) {
      return { status: 'ok', bytesWritten: data.byteLength };
    }
  };
  const usb: USBProvider = {
    async requestDevice() {
      throw new Error('unused');
    },
    async getDevices() {
      return [];
    },
    addEventListener() {},
    removeEventListener() {}
  };
  const captured = await captureWebUSBDevice(device);
  const relay = await WebUSBRelay.attach(usb, device, captured, 'fixture');
  return { relay, device, state };
}

test('a canceled transfer does not close the relay', async () => {
  const { relay, state } = await relayFixture();
  const inflight = relay.handle(transferFrame(1));
  assert.equal(await relay.handle(frame({ type: 6, id: 1 })), undefined);
  assert.equal(state.closed, false);
  state.pendingIn.pop()!({ status: 'ok', data: new DataView(new ArrayBuffer(0)) });
  assert.equal(await inflight, undefined);
  const response = decodeWebUSBFrame((await relay.handle(controlFrame(2)))!);
  assert.equal(response.status, 0);
  assert.equal(state.closed, false);
});

test('a disconnect frame with id 0 closes the relay', async () => {
  const { relay, state } = await relayFixture();
  assert.equal(await relay.handle(frame({ type: 7, id: 0 })), undefined);
  assert.equal(state.closed, true);
  assert.deepEqual(state.released, [0, 1]);
});

test('an unknown frame type is refused without ending the session', async () => {
  const { relay, state } = await relayFixture();
  const refused = decodeWebUSBFrame((await relay.handle(frame({ type: 9, id: 3 })))!);
  assert.equal(refused.type, 9 | 0x80);
  assert.equal(refused.status, 5);
  assert.equal(state.closed, false);
  const response = decodeWebUSBFrame((await relay.handle(controlFrame(4)))!);
  assert.equal(response.status, 0);
});

test('excess transfers are refused without ending the session', async () => {
  const { relay, state } = await relayFixture();
  const inflight = [];
  for (let id = 1; id <= 8; id++) inflight.push(relay.handle(transferFrame(id)));
  const refused = decodeWebUSBFrame((await relay.handle(transferFrame(9)))!);
  assert.equal(refused.status, 5);
  assert.equal(state.closed, false);
  for (const resolve of state.pendingIn.splice(0))
    resolve({ status: 'ok', data: new DataView(new ArrayBuffer(0)) });
  await Promise.all(inflight);
});

test('close releases every interface after one release fails', async () => {
  const { relay, state } = await relayFixture();
  state.releaseFails.add(0);
  await relay.close();
  assert.deepEqual(state.released, [1]);
  assert.equal(state.closed, true);
});

test('a reset that lands after close does not reopen the device', async () => {
  const { relay, device, state } = await relayFixture();
  let finishReset = () => {};
  device.reset = () => new Promise<void>((resolve) => (finishReset = resolve));
  const resetting = relay.handle(frame({ type: 5, id: 1 }));
  await relay.close();
  finishReset();
  const response = decodeWebUSBFrame((await resetting)!);
  assert.equal(response.status, 5);
  assert.equal(state.opened, 1);
});
