import type {
  Binding,
  ClaimRefusal,
  MediaSource,
  SourceKind,
  SourcesSnapshot,
  SourceStream
} from '@/api/sources.ts';

import { notifyAuthExpired } from '../../../../lib/auth-events.ts';
import { getBaseUrl } from '../../../../lib/service.ts';
import {
  decodeMediaFrame,
  encodeMediaFrame,
  mediaTimestampUS,
  type FrameAck,
  type FrameError,
  type MediaFrame,
  type MediaFrameKind
} from './transport.ts';

export type DeviceOffer = SourceStream & { deviceID: string };
export type SourceConnection = 'idle' | 'connecting' | 'connected' | 'disconnected';

export type StoredLease = {
  sinkID: string;
  streamID: string;
  token: string;
  deviceID: string;
  kind: SourceKind;
};

type SourceReady = {
  type: 'source_ready';
  source: MediaSource;
  snapshot: SourcesSnapshot;
};

type ControlResponse =
  | SourceReady
  | { type: 'claimed'; binding: Binding; token: string }
  | { type: 'resumed'; binding: Binding }
  | { type: 'released'; sink_id: string; reason?: string }
  | ({ type: 'claim_refused'; message: string } & ClaimRefusal)
  | FrameAck
  | FrameError
  | { type: 'error'; message: string; sink_id?: string };

type PendingRequest = {
  resolve: (binding?: Binding) => void;
  reject: (error: Error) => void;
};

type PendingFrame = { key: string; kind: MediaFrameKind; timer: number };

type ClientCallbacks = {
  onConnection: (state: SourceConnection) => void;
  onOwned: (sinks: Set<string>) => void;
  onError: (sinkID: string, message: string) => void;
  onRefused: (refusal: ClaimRefusal) => void;
  onRevoked: (sinkID: string, reason: string) => void;
  onSnapshot: (snapshot: SourcesSnapshot) => void;
  onBinary?: (data: ArrayBuffer) => Promise<ArrayBuffer | undefined>;
  // A speaker slot sends audio the other way, so the same socket that carries
  // browser frames out carries gadget frames in.
  onAudio?: (frame: MediaFrame) => void;
};

const maxBufferedBytes = (2 << 20) + 256;
const frameAckTimeout = 3000;

function sourceSocket() {
  return `${getBaseUrl('ws')}/api/sources/ws`;
}

export class BrowserSourceClient {
  private readonly callbacks: ClientCallbacks;
  private readonly leaseKey: string;
  private readonly openSocket: () => WebSocket;
  private socket?: WebSocket;
  private offers: DeviceOffer[] = [];
  private leases: StoredLease[];
  private owned = new Set<string>();
  private requests = new Map<string, PendingRequest>();
  private readyWaiters: Array<{ resolve: () => void; reject: (error: Error) => void }> = [];
  private reconnectTimer?: number;
  private reconnectDelay = 1000;
  private sequence = 0;
  private pendingFrames = new Map<number, PendingFrame>();
  private blockedUntil = new Map<string, number>();
  private shouldConnect = false;
  private restarting = false;
  private isReady = false;
  private sourceID = '';

  constructor(
    owner: string,
    callbacks: ClientCallbacks,
    openSocket = () => new WebSocket(sourceSocket())
  ) {
    this.callbacks = callbacks;
    this.leaseKey = `nanokvm-media-leases:${owner}`;
    this.leases = readLeases(this.leaseKey);
    this.openSocket = openSocket;
  }

  selections() {
    return this.leases.map((lease) => ({ ...lease }));
  }

  sourceId() {
    return this.sourceID;
  }

  adopt(sinkID: string, offer: DeviceOffer, token: string) {
    this.saveLease({
      sinkID,
      streamID: offer.id,
      token,
      deviceID: offer.deviceID,
      kind: offer.kind
    });
    this.owned.add(sinkID);
    this.emitOwned();
  }

  reconnect() {
    if (!this.shouldConnect || this.socket) return;
    this.reconnectDelay = 1000;
    this.connect();
  }

  async setOffers(offers: DeviceOffer[]) {
    const changed = offerKey(this.offers) !== offerKey(offers);
    this.offers = offers;
    this.shouldConnect = offers.length > 0 || this.leases.length > 0;
    if (!this.shouldConnect) {
      this.closeSocket();
      this.callbacks.onConnection('idle');
      return;
    }
    if (!this.socket || changed) this.restart();
    await this.ready();
  }

  async claim(sinkID: string, offer: DeviceOffer) {
    if (!this.offers.some((candidate) => candidate.id === offer.id)) {
      await this.setOffers([...this.offers, offer]);
    } else {
      await this.ready();
    }
    await this.request(sinkID, { type: 'claim', sink_id: sinkID, stream_id: offer.id });
    const response = this.lastBinding(sinkID);
    if (!response) throw new Error('Media lease was not stored');
  }

  async release(sinkID: string) {
    if (this.socket?.readyState !== WebSocket.OPEN) throw new Error('Media source disconnected');
    await this.request(sinkID, { type: 'release', sink_id: sinkID });
  }

  forget(sinkID: string) {
    this.removeLease(sinkID);
  }

  sendFrame(sinkID: string, streamID: string, kind: MediaFrameKind, payload: ArrayBuffer) {
    const socket = this.socket;
    if (
      !socket ||
      socket.readyState !== WebSocket.OPEN ||
      socket.bufferedAmount > maxBufferedBytes
    ) {
      return false;
    }
    const key = `${sinkID}\u0000${streamID}`;
    if ((this.blockedUntil.get(key) || 0) > Date.now()) return false;
    let inFlight = 0;
    for (const pending of this.pendingFrames.values()) {
      if (pending.key === key && pending.kind === kind) inFlight++;
    }
    if (inFlight >= (kind === 'mjpeg' ? 1 : 4)) return false;

    this.sequence = (this.sequence + 1) >>> 0;
    const sequence = this.sequence;
    const data = encodeMediaFrame({
      kind,
      sequence,
      timestampUS: mediaTimestampUS(),
      sinkID,
      streamID,
      payload
    });
    socket.send(data);
    const timer = window.setTimeout(() => {
      const pending = this.pendingFrames.get(sequence);
      if (!pending) return;
      this.pendingFrames.delete(sequence);
      this.blockedUntil.set(key, Date.now() + frameAckTimeout);
      this.callbacks.onError(sinkID, 'Media receiver stopped acknowledging frames');
    }, frameAckTimeout);
    this.pendingFrames.set(sequence, { key, kind, timer });
    return true;
  }

  close() {
    this.shouldConnect = false;
    this.closeSocket();
    this.rejectReady(new Error('Media source closed'));
    this.rejectRequests(new Error('Media source closed'));
    this.callbacks.onConnection('idle');
  }

  private restart() {
    this.isReady = false;
    this.restarting = true;
    this.closeSocket();
    this.restarting = false;
    this.connect();
  }

  private connect() {
    if (!this.shouldConnect || this.socket) return;
    window.clearTimeout(this.reconnectTimer);
    this.callbacks.onConnection('connecting');
    const socket = this.openSocket();
    this.socket = socket;
    socket.binaryType = 'arraybuffer';
    socket.onopen = () => {
      socket.send(
        JSON.stringify({
          type: 'hello',
          label: 'Browser',
          streams: this.offers.map(({ id, kind, label, formats, usb }) => ({
            id,
            kind,
            label,
            formats,
            usb
          }))
        })
      );
    };
    socket.onmessage = (event) => void this.onMessage(event);
    socket.onerror = () => this.callbacks.onConnection('disconnected');
    socket.onclose = (event) => this.onClose(socket, event);
  }

  private async onMessage(event: MessageEvent) {
    if (event.data instanceof ArrayBuffer) {
      // Two binary protocols share this socket. NKMF is media, and for a
      // speaker slot it arrives rather than leaves; everything else is a
      // WebUSB transfer for the relay.
      const audio = decodeMediaFrame(event.data);
      if (audio) {
        this.callbacks.onAudio?.(audio);
        return;
      }
      try {
        const response = await this.callbacks.onBinary?.(event.data);
        if (response && this.socket?.readyState === WebSocket.OPEN) this.socket.send(response);
      } catch (error) {
        this.callbacks.onError(
          'ffs.hybrid',
          error instanceof Error ? error.message : 'USB transfer failed'
        );
      }
      return;
    }
    if (typeof event.data !== 'string') return;
    let response: ControlResponse;
    try {
      response = JSON.parse(event.data) as ControlResponse;
    } catch {
      return;
    }
    if (response.type === 'source_ready') {
      this.sourceID = response.source?.id || '';
      this.callbacks.onSnapshot(response.snapshot);
      this.callbacks.onError('connection', '');
      this.reconnectDelay = 1000;
      await this.resumeLeases();
      this.isReady = true;
      this.callbacks.onConnection('connected');
      this.resolveReady();
      return;
    }
    if (response.type === 'frame_ack') {
      this.finishFrame(response.sequence);
      return;
    }
    if (response.type === 'frame_error') {
      this.finishFrame(response.sequence);
      this.callbacks.onError(response.sink_id, response.message);
      return;
    }
    if (response.type === 'claimed') {
      const offer = this.offers.find((candidate) => candidate.id === response.binding.stream_id);
      if (!offer) {
        this.finishRequest(response.binding.sink_id, new Error('Claimed device is unavailable'));
        return;
      }
      this.saveLease({
        sinkID: response.binding.sink_id,
        streamID: response.binding.stream_id,
        token: response.token,
        deviceID: offer.deviceID,
        kind: offer.kind
      });
      this.owned.add(response.binding.sink_id);
      this.emitOwned();
      this.finishRequest(response.binding.sink_id, undefined, response.binding);
      return;
    }
    if (response.type === 'resumed') {
      this.owned.add(response.binding.sink_id);
      this.emitOwned();
      this.finishRequest(response.binding.sink_id, undefined, response.binding);
      return;
    }
    if (response.type === 'released') {
      const requested = this.requests.has(response.sink_id);
      this.removeLease(response.sink_id);
      this.finishRequest(response.sink_id);
      if (!requested) this.callbacks.onRevoked(response.sink_id, response.reason || 'released');
      return;
    }
    if (response.type === 'claim_refused') {
      const { sink_id, owner, source_label, since, takeover } = response;
      this.callbacks.onRefused({ sink_id, owner, source_label, since, takeover });
      this.finishRequest(sink_id, new Error(`In use by ${owner} from ${source_label}`));
      return;
    }
    if (response.type === 'error') {
      const sinkID = response.sink_id || (this.requests.keys().next().value as string | undefined);
      if (sinkID) this.finishRequest(sinkID, new Error(response.message));
      if (sinkID === 'ffs.hybrid') this.removeLease(sinkID);
      if (sinkID) this.callbacks.onError(sinkID, response.message);
    }
  }

  private onClose(socket: WebSocket, event: CloseEvent) {
    if (this.socket !== socket) return;
    this.socket = undefined;
    this.isReady = false;
    this.sourceID = '';
    this.clearFrames();
    const error = new Error('Media source unavailable');
    this.rejectReady(error);
    this.rejectRequests(error);
    this.owned.clear();
    this.emitOwned();
    if (event.code === 4401) {
      this.shouldConnect = false;
      notifyAuthExpired();
      return;
    }
    if (!this.shouldConnect || this.restarting) return;
    this.callbacks.onConnection('disconnected');
    this.callbacks.onError('connection', error.message);
    this.reconnectTimer = window.setTimeout(() => this.connect(), this.reconnectDelay);
    this.reconnectDelay = Math.min(this.reconnectDelay * 2, 10000);
  }

  private async resumeLeases() {
    for (const lease of [...this.leases]) {
      const offer = this.offers.find(
        (candidate) => candidate.kind === lease.kind && candidate.deviceID === lease.deviceID
      );
      const fallback = this.offers.find((candidate) => candidate.kind === lease.kind);
      const selected = offer || fallback;
      if (!selected) continue;
      if (!offer) {
        lease.deviceID = selected.deviceID;
        lease.streamID = selected.id;
        this.writeLeases();
        this.callbacks.onError(lease.sinkID, 'Previous device missing; using the current default');
      }
      try {
        await this.request(lease.sinkID, {
          type: 'resume',
          sink_id: lease.sinkID,
          stream_id: selected.id,
          token: lease.token
        });
      } catch (error) {
        this.removeLease(lease.sinkID);
        this.callbacks.onError(
          lease.sinkID,
          error instanceof Error ? error.message : 'Media lease could not resume'
        );
      }
    }
  }

  private request(sinkID: string, message: Record<string, string>) {
    if (!this.socket || this.socket.readyState !== WebSocket.OPEN) {
      return Promise.reject(new Error('Media source disconnected'));
    }
    if (this.requests.has(sinkID))
      return Promise.reject(new Error('Media request already pending'));
    return new Promise<Binding | undefined>((resolve, reject) => {
      this.requests.set(sinkID, { resolve, reject });
      this.socket?.send(JSON.stringify(message));
    });
  }

  private ready() {
    if (this.isReady) return Promise.resolve();
    return new Promise<void>((resolve, reject) => this.readyWaiters.push({ resolve, reject }));
  }

  private resolveReady() {
    for (const waiter of this.readyWaiters.splice(0)) waiter.resolve();
  }

  private rejectReady(error: Error) {
    for (const waiter of this.readyWaiters.splice(0)) waiter.reject(error);
  }

  private finishRequest(sinkID: string, error?: Error, binding?: Binding) {
    const request = this.requests.get(sinkID);
    if (!request) return;
    this.requests.delete(sinkID);
    if (error) request.reject(error);
    else request.resolve(binding);
  }

  private rejectRequests(error: Error) {
    for (const request of this.requests.values()) request.reject(error);
    this.requests.clear();
  }

  private finishFrame(sequence: number) {
    const pending = this.pendingFrames.get(sequence);
    if (!pending) return;
    window.clearTimeout(pending.timer);
    this.pendingFrames.delete(sequence);
    this.blockedUntil.delete(pending.key);
  }

  private clearFrames() {
    for (const frame of this.pendingFrames.values()) window.clearTimeout(frame.timer);
    this.pendingFrames.clear();
    this.blockedUntil.clear();
  }

  private closeSocket() {
    window.clearTimeout(this.reconnectTimer);
    this.reconnectTimer = undefined;
    const socket = this.socket;
    this.socket = undefined;
    this.isReady = false;
    socket?.close(1000);
    this.clearFrames();
  }

  private saveLease(lease: StoredLease) {
    this.leases = [...this.leases.filter((item) => item.sinkID !== lease.sinkID), lease];
    this.writeLeases();
  }

  private removeLease(sinkID: string) {
    this.leases = this.leases.filter((lease) => lease.sinkID !== sinkID);
    this.writeLeases();
    this.owned.delete(sinkID);
    this.emitOwned();
  }

  private writeLeases() {
    sessionStorage.setItem(this.leaseKey, JSON.stringify(this.leases));
  }

  private lastBinding(sinkID: string) {
    return this.leases.find((lease) => lease.sinkID === sinkID);
  }

  private emitOwned() {
    this.callbacks.onOwned(new Set(this.owned));
  }
}

function readLeases(key: string): StoredLease[] {
  try {
    const parsed = JSON.parse(sessionStorage.getItem(key) || '[]') as StoredLease[];
    if (!Array.isArray(parsed)) return [];
    return parsed.filter(
      (lease) =>
        typeof lease.sinkID === 'string' &&
        typeof lease.streamID === 'string' &&
        typeof lease.token === 'string' &&
        typeof lease.deviceID === 'string' &&
        (lease.kind === 'camera' || lease.kind === 'microphone' || lease.kind === 'usb_device')
    );
  } catch {
    return [];
  }
}

function offerKey(offers: DeviceOffer[]) {
  return offers
    .map(
      (offer) =>
        `${offer.id}:${offer.kind}:${offer.usb?.profile || ''}:${offer.usb?.configuration || ''}:${offer.usb?.interfaces.join(',') || ''}`
    )
    .sort()
    .join('|');
}
