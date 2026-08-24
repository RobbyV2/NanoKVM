import type { PresentationProfile } from '@/api/presentation.ts';

import type { DeviceOffer } from './client.ts';
import {
  classifyWebUSBConfiguration,
  decodeWebUSBFrame,
  encodeWebUSBFrame,
  webUSBMaxPayload,
  type WirePacket
} from './webusb-wire.ts';

type USBSetup = {
  requestType: 'standard' | 'class' | 'vendor';
  recipient: 'device' | 'interface' | 'endpoint' | 'other';
  request: number;
  value: number;
  index: number;
};

type USBInResult = { status: 'ok' | 'stall' | 'babble'; data?: DataView };
type USBOutResult = { status: 'ok' | 'stall' | 'babble'; bytesWritten: number };
type USBEndpointLike = { endpointNumber: number; direction: 'in' | 'out'; type: string };
type USBAlternateLike = {
  alternateSetting: number;
  interfaceClass: number;
  endpoints: USBEndpointLike[];
};
type USBInterfaceLike = { interfaceNumber: number; alternates: USBAlternateLike[] };
type USBConfigurationLike = { configurationValue: number; interfaces: USBInterfaceLike[] };

export type USBDeviceLike = {
  vendorId: number;
  productId: number;
  manufacturerName?: string;
  productName?: string;
  serialNumber?: string;
  opened: boolean;
  configuration?: USBConfigurationLike;
  configurations: USBConfigurationLike[];
  open(): Promise<void>;
  close(): Promise<void>;
  reset(): Promise<void>;
  selectConfiguration(value: number): Promise<void>;
  claimInterface(number: number): Promise<void>;
  releaseInterface(number: number): Promise<void>;
  clearHalt(direction: 'in' | 'out', endpoint: number): Promise<void>;
  controlTransferIn(setup: USBSetup, length: number): Promise<USBInResult>;
  controlTransferOut(setup: USBSetup, data?: BufferSource): Promise<USBOutResult>;
  transferIn(endpoint: number, length: number): Promise<USBInResult>;
  transferOut(endpoint: number, data: BufferSource): Promise<USBOutResult>;
};

export type USBProvider = {
  requestDevice(options: { filters: never[] }): Promise<USBDeviceLike>;
  getDevices(): Promise<USBDeviceLike[]>;
  addEventListener(type: 'disconnect', callback: (event: { device: USBDeviceLike }) => void): void;
  removeEventListener(
    type: 'disconnect',
    callback: (event: { device: USBDeviceLike }) => void
  ): void;
};

export type Captured = {
  device: Uint8Array;
  configurations: Uint8Array[];
  bos?: Uint8Array;
  strings: Record<string, string>;
  profile: PresentationProfile;
  configuration: number;
  interfaces: number[];
};

const maxConfigurations = 8;

export class WebUSBRelay {
  readonly offer: DeviceOffer;
  private readonly usb: USBProvider;
  private readonly device: USBDeviceLike;
  private readonly configuration: number;
  private readonly interfaces: number[];
  private endpoints = new Map<number, USBEndpointLike>();
  private pending = new Set<number>();
  private canceled = new Set<number>();
  private closed = false;
  private disconnected: (event: { device: USBDeviceLike }) => void;

  private constructor(
    usb: USBProvider,
    device: USBDeviceLike,
    captured: Captured,
    deviceID: string,
    onDisconnect?: () => void
  ) {
    this.usb = usb;
    this.device = device;
    this.configuration = captured.configuration;
    this.interfaces = captured.interfaces;
    this.offer = {
      id: `usb_${deviceID}`,
      deviceID,
      kind: 'usb_device',
      label: device.productName || `USB ${hex(device.vendorId)}:${hex(device.productId)}`,
      usb: {
        profile: captured.profile.name,
        configuration: captured.configuration,
        interfaces: captured.interfaces
      }
    };
    this.disconnected = (event) => {
      if (event.device === this.device) {
        this.closed = true;
        this.usb.removeEventListener('disconnect', this.disconnected);
        onDisconnect?.();
      }
    };
    this.usb.addEventListener('disconnect', this.disconnected);
  }

  static supported() {
    return !!webUSB();
  }

  // navigator.usb is absent both on a browser without WebUSB and on a Chromium
  // that has it but withholds it outside a secure context, and only the second
  // is fixable by the user.
  static unavailable(): 'insecure' | 'unsupported' | '' {
    if (webUSB()) return '';
    return window.isSecureContext === false ? 'insecure' : 'unsupported';
  }

  static async select(onDisconnect?: () => void) {
    const usb = webUSB();
    if (!usb)
      throw new Error(
        WebUSBRelay.unavailable() === 'insecure'
          ? 'This page is not a secure context, so the browser withholds WebUSB. Enable HTTPS in Settings, Network.'
          : 'WebUSB requires a Chromium browser'
      );
    const device = await usb.requestDevice({ filters: [] });
    const captured = await capture(device);
    await saveProfile(captured.profile);
    if (captured.bos && captured.bos[4] !== 0)
      throw new Error('This device has BOS capabilities that Hybrid mode cannot reproduce');
    if (captured.interfaces.length === 0)
      throw new Error('No control, bulk, or interrupt interface is safe to forward');
    const deviceID = await identity(device, captured.device);
    return WebUSBRelay.attach(usb, device, captured, deviceID, onDisconnect);
  }

  static async attach(
    usb: USBProvider,
    device: USBDeviceLike,
    captured: Captured,
    deviceID: string,
    onDisconnect?: () => void
  ) {
    const relay = new WebUSBRelay(usb, device, captured, deviceID, onDisconnect);
    await relay.prepare();
    return relay;
  }

  async handle(data: ArrayBuffer) {
    const packet = decodeWebUSBFrame(data);
    if (packet.type & 0x80) throw new Error('Invalid USB request');
    if (packet.type === 7) {
      await this.close();
      return undefined;
    }
    if (packet.id === 0) throw new Error('Invalid USB request');
    if (packet.type === 6) {
      if (this.pending.has(packet.id)) this.canceled.add(packet.id);
      return undefined;
    }
    if (this.closed)
      return encodeWebUSBFrame({
        ...packet,
        type: packet.type | 0x80,
        status: 3,
        payload: new Uint8Array()
      });
    if (this.pending.has(packet.id) || this.pending.size >= 8)
      return encodeWebUSBFrame({
        ...packet,
        type: packet.type | 0x80,
        status: 5,
        payload: new Uint8Array()
      });
    this.pending.add(packet.id);
    try {
      const payload = await this.execute(packet);
      if (this.canceled.delete(packet.id)) return undefined;
      return encodeWebUSBFrame({ ...packet, type: packet.type | 0x80, status: 0, payload });
    } catch (error) {
      if (this.canceled.delete(packet.id)) return undefined;
      const status = error instanceof DOMException && error.name === 'NetworkError' ? 1 : 5;
      return encodeWebUSBFrame({
        ...packet,
        type: packet.type | 0x80,
        status,
        payload: new Uint8Array()
      });
    } finally {
      this.pending.delete(packet.id);
    }
  }

  async close() {
    if (this.closed) return;
    this.closed = true;
    this.usb.removeEventListener('disconnect', this.disconnected);
    for (const number of this.interfaces) {
      try {
        await this.device.releaseInterface(number);
      } catch {
        continue;
      }
    }
    await this.device.close();
  }

  private async prepare() {
    if (!this.device.opened) await this.device.open();
    if (this.device.configuration?.configurationValue !== this.configuration)
      await this.device.selectConfiguration(this.configuration);
    const selected = new Set(this.interfaces);
    for (const iface of this.device.configuration?.interfaces || []) {
      if (!selected.has(iface.interfaceNumber)) continue;
      const alternate = iface.alternates.find((item) => item.alternateSetting === 0);
      if (!alternate)
        throw new Error(`Interface ${iface.interfaceNumber} has no alternate setting 0`);
      await this.device.claimInterface(iface.interfaceNumber);
      for (const endpoint of alternate.endpoints) {
        if (endpoint.type === 'isochronous')
          throw new Error('WebUSB has no isochronous data path');
        const address = endpoint.endpointNumber | (endpoint.direction === 'in' ? 0x80 : 0);
        this.endpoints.set(address, endpoint);
      }
    }
  }

  private async execute(packet: WirePacket) {
    if (packet.declaredLength > webUSBMaxPayload) throw new Error('USB transfer is too large');
    if (packet.type === 1) {
      const setup = controlSetup(packet);
      this.validateControl(packet, setup);
      if (packet.requestType & 0x80) {
        const result = await this.device.controlTransferIn(setup, packet.declaredLength);
        return transferData(result, packet.declaredLength);
      }
      const result = await this.device.controlTransferOut(setup, packet.payload.slice().buffer);
      if (result.status !== 'ok') throw new DOMException(result.status, 'NetworkError');
      return new Uint8Array();
    }
    const endpoint = this.endpoints.get(packet.endpoint);
    if ((packet.type === 2 || packet.type === 3) && !endpoint)
      throw new Error(`Endpoint 0x${packet.endpoint.toString(16)} is not selected`);
    if (packet.type === 2) {
      const result = await this.device.transferIn(endpoint!.endpointNumber, packet.declaredLength);
      return transferData(result, packet.declaredLength);
    }
    if (packet.type === 3) {
      const result = await this.device.transferOut(
        endpoint!.endpointNumber,
        packet.payload.slice().buffer
      );
      if (result.status !== 'ok') throw new DOMException(result.status, 'NetworkError');
      return new Uint8Array();
    }
    if (packet.type === 4) {
      if (!this.endpoints.has(packet.endpoint))
        throw new Error(`Endpoint 0x${packet.endpoint.toString(16)} is not selected`);
      await this.device.clearHalt(packet.endpoint & 0x80 ? 'in' : 'out', packet.endpoint & 0x0f);
      return new Uint8Array();
    }
    if (packet.type === 5) {
      await this.device.reset();
      if (this.closed) throw new Error('The USB relay is closed');
      await this.prepare();
      return new Uint8Array();
    }
    throw new Error('Unsupported USB operation');
  }

  private validateControl(packet: WirePacket, setup: USBSetup) {
    if (setup.requestType === 'standard') {
      if (!(packet.requestType & 0x80))
        throw new Error('WebUSB forbids standard control OUT transfers');
      if (![0, 6, 8, 10, 12].includes(packet.request))
        throw new Error('WebUSB forbids this standard control request');
    }
    if (setup.requestType !== 'standard' && setup.recipient === 'device')
      throw new Error('Device-wide class and vendor control transfers are refused');
    if (setup.recipient === 'interface' && !this.interfaces.includes(packet.index & 0xff))
      throw new Error('Control transfer targets an excluded interface');
    if (setup.recipient === 'endpoint' && !this.endpoints.has(packet.index & 0xff))
      throw new Error('Control transfer targets an excluded endpoint');
    if (setup.recipient === 'other')
      throw new Error('Other-recipient control transfers are refused');
  }
}

async function capture(device: USBDeviceLike): Promise<Captured> {
  if (!device.opened) await device.open();
  const rawDevice = await descriptor(device, 1, 0, 0, 18);
  if (rawDevice.length !== 18 || rawDevice[4] !== 0)
    throw new Error('Device-level USB classes require Exact mode');
  if (rawDevice[17] === 0 || rawDevice[17] > maxConfigurations)
    throw new Error(`WebUSB supports 1 to ${maxConfigurations} configurations`);
  const configurations: Uint8Array[] = [];
  for (let index = 0; index < rawDevice[17]; index++) {
    const header = await descriptor(device, 2, index, 0, 9);
    const total = header[2] | (header[3] << 8);
    if (total < 9 || total > 64 << 10) throw new Error('Invalid USB configuration length');
    configurations.push(await descriptor(device, 2, index, 0, total));
  }
  let bos: Uint8Array | undefined;
  if (((rawDevice[3] << 8) | rawDevice[2]) >= 0x0201) {
    const header = await descriptor(device, 15, 0, 0, 5);
    const total = header[2] | (header[3] << 8);
    if (total < 5 || total > 4096) throw new Error('Invalid BOS length');
    bos = await descriptor(device, 15, 0, 0, total);
  }
  const strings = await captureStrings(device, rawDevice, configurations);
  const selected = device.configuration?.configurationValue || configurations[0]?.[5];
  if (!selected) throw new Error('The device has no selectable configuration');
  if (device.configuration?.configurationValue !== selected)
    await device.selectConfiguration(selected);
  const active = configurations.find((item) => item[5] === selected);
  if (!active) throw new Error('Active USB configuration was not captured');
  const interfaces = classifyWebUSBConfiguration(active);
  const profile = profileFrom(device, rawDevice, configurations, bos, strings);
  return {
    device: rawDevice,
    configurations,
    bos,
    strings,
    profile,
    configuration: selected,
    interfaces
  };
}

function profileFrom(
  device: USBDeviceLike,
  raw: Uint8Array,
  configs: Uint8Array[],
  bos: Uint8Array | undefined,
  strings: Record<string, string>
): PresentationProfile {
  const selected =
    configs.find(
      (config) => config[5] === (device.configuration?.configurationValue || configs[0][5])
    ) || configs[0];
  const bcdUSB = `0x${hexByte(raw[3])}${hexByte(raw[2])}`;
  const bcdDevice = `0x${hexByte(raw[13])}${hexByte(raw[12])}`;
  const serial = device.serialNumber || strings[String(raw[16])];
  return {
    schema_version: 1,
    name: `webusb-${hex(device.vendorId)}-${hex(device.productId)}-${shortName(serial || device.productName || 'device')}`,
    built_in: false,
    device: {
      vendor_id: `0x${hex(device.vendorId)}`,
      product_id: `0x${hex(device.productId)}`,
      bcd_usb: bcdUSB,
      bcd_device: bcdDevice,
      class: raw[4],
      subclass: raw[5],
      protocol: raw[6],
      serial: serial || undefined,
      manufacturer: device.manufacturerName || strings[String(raw[14])] || 'Captured USB device',
      product:
        device.productName ||
        strings[String(raw[15])] ||
        `USB ${hex(device.vendorId)}:${hex(device.productId)}`
    },
    config: {
      bm_attributes: selected[7],
      max_power: Math.max(2, selected[8] * 2),
      configuration: strings[String(selected[6])] || 'Captured configuration'
    },
    functions: [],
    descriptors: {
      device: base64(raw),
      configurations: configs.map(base64),
      bos: bos ? base64(bos) : undefined,
      strings
    }
  };
}

async function saveProfile(profile: PresentationProfile) {
  const presentation = await import('@/api/presentation.ts');
  const existing = await presentation.getProfile(profile.name);
  const response =
    existing.code === 0
      ? await presentation.updateProfile(profile)
      : await presentation.createProfile(profile);
  if (response.code !== 0) throw new Error(response.msg || 'Could not save captured USB profile');
}

async function captureStrings(device: USBDeviceLike, raw: Uint8Array, configs: Uint8Array[]) {
  const indices = new Set<number>([raw[14], raw[15], raw[16]]);
  for (const config of configs) {
    indices.add(config[6]);
    walk(config, (item) => {
      if (item[1] === 4 && item.length >= 9) indices.add(item[8]);
      if (item[1] === 11 && item.length >= 8) indices.add(item[7]);
      if (item[1] === 0x24 && item.length >= 4 && item[2] === 0x0f) indices.add(item[3]);
    });
  }
  indices.delete(0);
  if (indices.size === 0) return {};
  const languages = await descriptor(device, 3, 0, 0, 255);
  if (languages.length < 4) throw new Error('USB string language table is missing');
  const language = languages[2] | (languages[3] << 8);
  const strings: Record<string, string> = {};
  for (const index of indices) {
    const value = await descriptor(device, 3, index, language, 255);
    strings[String(index)] = decodeString(value);
  }
  return strings;
}

async function descriptor(
  device: USBDeviceLike,
  type: number,
  index: number,
  language: number,
  length: number
) {
  const result = await device.controlTransferIn(
    {
      requestType: 'standard',
      recipient: 'device',
      request: 6,
      value: (type << 8) | index,
      index: language
    },
    length
  );
  if (result.status !== 'ok' || !result.data)
    throw new DOMException(`Descriptor ${type}:${index} ${result.status}`, 'NetworkError');
  return new Uint8Array(result.data.buffer, result.data.byteOffset, result.data.byteLength).slice();
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

function decodeString(data: Uint8Array) {
  if (data.length < 2 || data[0] !== data.length || data[1] !== 3 || data.length % 2)
    throw new Error('Malformed USB string');
  const units: number[] = [];
  for (let index = 2; index < data.length; index += 2)
    units.push(data[index] | (data[index + 1] << 8));
  return String.fromCharCode(...units);
}

function controlSetup(packet: WirePacket): USBSetup {
  const requestType = (packet.requestType >> 5) & 3;
  const recipient = packet.requestType & 0x1f;
  if (requestType > 2 || recipient > 3) throw new Error('Invalid USB control request type');
  return {
    requestType: ['standard', 'class', 'vendor'][requestType] as USBSetup['requestType'],
    recipient: ['device', 'interface', 'endpoint', 'other'][recipient] as USBSetup['recipient'],
    request: packet.request,
    value: packet.value,
    index: packet.index
  };
}

function transferData(result: USBInResult, limit: number) {
  if (result.status !== 'ok' || !result.data) throw new DOMException(result.status, 'NetworkError');
  if (result.data.byteLength > limit) throw new Error('Browser returned an oversized USB transfer');
  return new Uint8Array(result.data.buffer, result.data.byteOffset, result.data.byteLength).slice();
}

function webUSB() {
  return (navigator as Navigator & { usb?: USBProvider }).usb;
}

async function identity(device: USBDeviceLike, raw: Uint8Array) {
  const digest = await crypto.subtle.digest(
    'SHA-256',
    new Uint8Array([...raw, ...new TextEncoder().encode(device.serialNumber || '')])
  );
  return Array.from(new Uint8Array(digest).subarray(0, 8), (byte) => hexByte(byte)).join('');
}

function base64(data: Uint8Array) {
  let binary = '';
  for (const byte of data) binary += String.fromCharCode(byte);
  return btoa(binary);
}

function hex(value: number) {
  return value.toString(16).padStart(4, '0');
}

function hexByte(value: number) {
  return value.toString(16).padStart(2, '0');
}

function shortName(value: string) {
  const normalized = value
    .toLowerCase()
    .replace(/[^a-z0-9._-]+/g, '-')
    .replace(/^-+|-+$/g, '');
  return (normalized || 'device').slice(0, 24);
}

export const captureWebUSBDevice = capture;
