import { ReactNode, useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useAuth } from '@/contexts/auth.ts';
import type { TFunction } from 'i18next';
import { useTranslation } from 'react-i18next';

import { getPassthrough } from '@/api/passthrough.ts';
import type { PassthroughStatus } from '@/api/passthrough.ts';
import * as api from '@/api/sources.ts';
import type { ClaimRefusal, SourceKind, SourceSink, SourcesSnapshot } from '@/api/sources.ts';
import { notifyAuthExpired } from '@/lib/auth-events.ts';

import { CameraCapture, captureSupport, CaptureUnsupported, MicrophoneCapture } from './capture.ts';
import { BrowserSourceClient, type DeviceOffer, type SourceConnection } from './client.ts';
import {
  DevicesContext,
  type DevicesState,
  type MediaPermission,
  type SourceEventsConnection
} from './context.ts';
import { SpeakerPlayback } from './playback.ts';
import { emptySnapshot, reduceSources } from './state.ts';
import { WebUSBRelay } from './webusb.ts';

const mediaGrace = 5000;

// The browser is the only device a speaker slot can use, so it needs no picker
// and no permission: one synthetic offer stands for "this page will play it".
const speakerStreamID = 'spk_browser';
const speakerOffer: DeviceOffer = {
  id: speakerStreamID,
  deviceID: speakerStreamID,
  kind: 'speaker',
  label: 'Browser playback',
  formats: [{ codec: 'pcm_s16le', sample_rate: 48000, channels: 1 }]
};

type Capture = CameraCapture | MicrophoneCapture | SpeakerPlayback;
type RunningCapture = {
  capture: Capture;
  deviceID: string;
  streamID: string;
  kind: SourceKind;
  demand: string;
};

export const SourcesProvider = ({ children }: { children: ReactNode }) => {
  const { account } = useAuth();
  const { t } = useTranslation();
  const [snapshot, setSnapshot] = useState<SourcesSnapshot>(emptySnapshot);
  const [eventsConnection, setEventsConnection] = useState<SourceEventsConnection>('connecting');
  const [sourceConnection, setSourceConnection] = useState<SourceConnection>('idle');
  const [devices, setDevices] = useState<Record<SourceKind, DeviceOffer[]>>({
    camera: [],
    microphone: [],
    speaker: [speakerOffer],
    usb_device: []
  });
  const [permissions, setPermissions] = useState<Record<'camera' | 'microphone', MediaPermission>>({
    camera: 'unknown',
    microphone: 'unknown'
  });
  const [passthrough, setPassthrough] = useState<PassthroughStatus | null>(null);
  const [owned, setOwned] = useState(new Set<string>());
  const [active, setActive] = useState(new Set<string>());
  const [busy, setBusy] = useState(new Set<string>());
  const [muted, setMuted] = useState(new Set<string>());
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [refusals, setRefusals] = useState<Record<string, ClaimRefusal>>({});
  const [revoked, setRevoked] = useState<Record<string, string>>({});
  const eventsSocket = useRef<WebSocket>();
  const eventsRetry = useRef<number>();
  const eventsConnect = useRef<() => void>();
  const captures = useRef(new Map<string, RunningCapture>());
  const stopTimers = useRef(new Map<string, number>());
  // Which sinks the host is actually reading right now. A capture outlives its
  // demand by mediaGrace so a brief blip does not tear the camera down, and for
  // those seconds the encoder keeps producing frames the sink is guaranteed to
  // refuse. Sending them anyway cost a websocket round trip and a rejection per
  // frame - 150 of them per blip at 30fps - and every rejection surfaced as
  // "media sink is not demanded", which is not a fault the operator can act on.
  // Holding them here keeps the grace period doing its job (an instant resume)
  // without spending the device's CPU on frames it will throw away.
  const demanded = useRef(new Set<string>());
  const mounted = useRef(true);
  const usbRelay = useRef<WebUSBRelay>();

  const stopCapture = useCallback(async (sinkID: string) => {
    window.clearTimeout(stopTimers.current.get(sinkID));
    stopTimers.current.delete(sinkID);
    const running = captures.current.get(sinkID);
    if (!running) return;
    captures.current.delete(sinkID);
    await running.capture.stop();
    setActive((current) => {
      const next = new Set(current);
      next.delete(sinkID);
      return next;
    });
  }, []);

  const closeRelay = useCallback(() => {
    const relay = usbRelay.current;
    usbRelay.current = undefined;
    void relay?.close();
    setDevices((current) => ({ ...current, usb_device: [] }));
    setActive((current) => {
      const next = new Set(current);
      next.delete('ffs.hybrid');
      return next;
    });
  }, []);

  const client = useMemo(
    () =>
      new BrowserSourceClient(account.username, {
        onConnection: setSourceConnection,
        onOwned: setOwned,
        onError: (sinkID, message) => {
          setErrors((current) => ({ ...current, [sinkID]: message }));
          if (sinkID === 'ffs.hybrid') closeRelay();
        },
        onRefused: (refusal) =>
          setRefusals((current) => ({ ...current, [refusal.sink_id]: refusal })),
        onRevoked: (sinkID, reason) => {
          setRevoked((current) => ({ ...current, [sinkID]: reason }));
          void stopCapture(sinkID);
          if (sinkID === 'ffs.hybrid') closeRelay();
        },
        onSnapshot: setSnapshot,
        onBinary: async (data) => usbRelay.current?.handle(data),
        onAudio: (frame) => {
          const running = captures.current.get(frame.sinkID);
          if (running?.kind === 'speaker') (running.capture as SpeakerPlayback).push(frame.payload);
        }
      }),
    [account.username, closeRelay, stopCapture]
  );

  const setError = useCallback((sinkID: string, message: string) => {
    setErrors((current) => ({ ...current, [sinkID]: message }));
  }, []);

  const clearError = useCallback((sinkID: string) => {
    setErrors((current) => {
      const next = { ...current };
      delete next[sinkID];
      return next;
    });
  }, []);

  useEffect(() => {
    mounted.current = true;
    let retryDelay = 1000;

    const connect = () => {
      window.clearTimeout(eventsRetry.current);
      setEventsConnection('connecting');
      const socket = new WebSocket(api.sourcesSocket('events'));
      eventsSocket.current = socket;
      socket.onopen = () => {
        retryDelay = 1000;
        setEventsConnection('connected');
      };
      socket.onmessage = (message) => {
        if (typeof message.data !== 'string') return;
        try {
          const event = JSON.parse(message.data) as api.SourcesEvent;
          setSnapshot((current) => reduceSources(current, event));
        } catch {
          setEventsConnection('disconnected');
        }
      };
      socket.onclose = (event) => {
        if (eventsSocket.current !== socket || !mounted.current) return;
        eventsSocket.current = undefined;
        setEventsConnection('disconnected');
        if (event.code === 4401) {
          notifyAuthExpired();
          return;
        }
        eventsRetry.current = window.setTimeout(connect, retryDelay);
        retryDelay = Math.min(retryDelay * 2, 10000);
      };
      socket.onerror = () => setEventsConnection('disconnected');
    };
    eventsConnect.current = connect;

    void api
      .getSources()
      .then((response) => {
        if (response.code === 0 && mounted.current) setSnapshot(response.data as SourcesSnapshot);
      })
      .catch(() => {
        if (mounted.current) setEventsConnection('disconnected');
      });
    connect();

    return () => {
      mounted.current = false;
      window.clearTimeout(eventsRetry.current);
      eventsSocket.current?.close(1000);
      eventsSocket.current = undefined;
    };
  }, []);

  const discover = useCallback(async (kind: SourceKind, requestPermission: boolean) => {
    if (kind === 'usb_device') throw new Error('Use the USB device picker');
    if (kind === 'speaker') {
      const blocked = captureSupport(kind);
      if (blocked) throw new CaptureUnsupported(blocked);
      return [speakerOffer];
    }
    const blocked = captureSupport(kind);
    if (blocked) throw new CaptureUnsupported(blocked);
    if (requestPermission) {
      const permission = await navigator.mediaDevices.getUserMedia({
        video: kind === 'camera',
        audio: kind === 'microphone'
      });
      permission.getTracks().forEach((track) => track.stop());
    }
    const entries = (await navigator.mediaDevices.enumerateDevices()).filter(
      (device) => device.kind === (kind === 'camera' ? 'videoinput' : 'audioinput')
    );
    const next = entries.map((device, index) => ({
      id: streamID(kind, device.deviceId, index),
      deviceID: device.deviceId,
      kind,
      label: device.label || `${kind === 'camera' ? 'Camera' : 'Microphone'} ${index + 1}`,
      formats:
        kind === 'camera'
          ? [
              { codec: 'mjpeg', width: 1280, height: 720, fps: 30 },
              { codec: 'mjpeg', width: 1280, height: 720, fps: 15 },
              { codec: 'mjpeg', width: 640, height: 480, fps: 30 },
              { codec: 'mjpeg', width: 320, height: 240, fps: 30 },
              { codec: 'mjpeg', width: 160, height: 120, fps: 30 }
            ]
          : [{ codec: 'pcm_s16le', sample_rate: 48000, channels: 1 }]
    }));
    if (next.length === 0) throw new Error(`No ${kind} input found`);
    setDevices((current) => ({ ...current, [kind]: next }));
    return next;
  }, []);

  useEffect(() => {
    const leases = client.selections();
    if (leases.length === 0 || !navigator.mediaDevices?.enumerateDevices) return;
    const kinds = [...new Set(leases.map((lease) => lease.kind))].filter(
      (kind) => kind !== 'usb_device'
    );
    void Promise.all(kinds.map((kind) => discover(kind, false)))
      .then((offers) => client.setOffers(offers.flat()))
      .catch((error) => setError('connection', mediaError(error, t)));
  }, [client, discover, setError, t]);

  const startCapture = useCallback(
    async (sink: SourceSink, lease: ReturnType<BrowserSourceClient['selections']>[number]) => {
      if (captures.current.has(sink.id)) return;
      const offer = devices[sink.kind].find((device) => device.deviceID === lease.deviceID);
      if (!offer) {
        setError(sink.id, 'Selected device is unavailable');
        return;
      }
      clearError(sink.id);
      try {
        if (sink.kind === 'speaker') {
          const playback = new SpeakerPlayback();
          captures.current.set(sink.id, {
            capture: playback,
            deviceID: offer.deviceID,
            streamID: offer.id,
            kind: sink.kind,
            demand: demandKey(sink)
          });
          await playback.start(muted.has(sink.id));
        } else if (sink.kind === 'camera') {
          const capture = new CameraCapture();
          captures.current.set(sink.id, {
            capture,
            deviceID: offer.deviceID,
            streamID: offer.id,
            kind: sink.kind,
            demand: demandKey(sink)
          });
          await capture.start(
            offer.deviceID,
            sink.demand,
            (payload) => {
              if (!demanded.current.has(sink.id)) return false;
              return client.sendFrame(sink.id, offer.id, 'mjpeg', payload);
            },
            (message) => setError(sink.id, message)
          );
        } else {
          const capture = new MicrophoneCapture();
          captures.current.set(sink.id, {
            capture,
            deviceID: offer.deviceID,
            streamID: offer.id,
            kind: sink.kind,
            demand: demandKey(sink)
          });
          await capture.start(
            offer.deviceID,
            (payload) => {
              if (!demanded.current.has(sink.id)) return false;
              return client.sendFrame(sink.id, offer.id, 'pcm_s16le_mono_48k', payload);
            },
            muted.has(sink.id)
          );
        }
        setActive((current) => new Set(current).add(sink.id));
      } catch (error) {
        await stopCapture(sink.id);
        setError(sink.id, mediaError(error, t));
      }
    },
    [clearError, client, devices, muted, setError, stopCapture, t]
  );

  useEffect(() => {
    const leaseBySink = new Map(client.selections().map((lease) => [lease.sinkID, lease]));
    const wanted = new Set<string>();
    for (const sink of snapshot.sinks) {
      const lease = leaseBySink.get(sink.id);
      // A speaker's renderer has to exist before the gadget's first packet
      // arrives, so it runs on the claim rather than on host demand.
      const canRun =
        sourceConnection === 'connected' &&
        owned.has(sink.id) &&
        !!lease &&
        (sink.demand.streaming || sink.kind === 'speaker') &&
        sink.binding?.state !== 'orphaned' &&
        sink.binding?.state !== 'suspended';
      if (sink.demand.streaming) demanded.current.add(sink.id);
      else demanded.current.delete(sink.id);
      if (canRun) {
        wanted.add(sink.id);
        window.clearTimeout(stopTimers.current.get(sink.id));
        stopTimers.current.delete(sink.id);
        const running = captures.current.get(sink.id);
        if (running && running.demand !== demandKey(sink)) {
          void stopCapture(sink.id).then(() => startCapture(sink, lease));
        } else {
          void startCapture(sink, lease);
        }
      } else if (captures.current.has(sink.id)) {
        if (sourceConnection !== 'connected' || !owned.has(sink.id)) {
          void stopCapture(sink.id);
        } else if (!stopTimers.current.has(sink.id)) {
          const timer = window.setTimeout(() => void stopCapture(sink.id), mediaGrace);
          stopTimers.current.set(sink.id, timer);
        }
      }
    }
    for (const sinkID of captures.current.keys()) {
      if (!wanted.has(sinkID) && !snapshot.sinks.some((sink) => sink.id === sinkID)) {
        void stopCapture(sinkID);
      }
    }
    // A sink that has left the snapshot demands nothing.
    for (const sinkID of [...demanded.current]) {
      if (!snapshot.sinks.some((sink) => sink.id === sinkID)) demanded.current.delete(sinkID);
    }
  }, [client, owned, snapshot.sinks, sourceConnection, startCapture, stopCapture]);

  useEffect(
    () => () => {
      client.close();
      void usbRelay.current?.close();
      for (const timer of stopTimers.current.values()) window.clearTimeout(timer);
      for (const sinkID of captures.current.keys()) void stopCapture(sinkID);
    },
    [client, stopCapture]
  );

  useEffect(() => {
    const query = navigator.permissions?.query?.bind(navigator.permissions);
    if (!query) return;
    const statuses: PermissionStatus[] = [];
    for (const kind of ['camera', 'microphone'] as const) {
      void query({ name: kind } as unknown as PermissionDescriptor)
        .then((status) => {
          if (!mounted.current) return;
          statuses.push(status);
          const apply = () =>
            setPermissions((current) => ({ ...current, [kind]: status.state as MediaPermission }));
          apply();
          status.onchange = apply;
        })
        .catch(() => undefined);
    }
    return () => {
      for (const status of statuses) status.onchange = null;
    };
  }, []);

  const refreshPassthrough = useCallback(() => {
    if (account.role !== 'admin') return;
    void getPassthrough()
      .then((response) => {
        if (response.code === 0 && mounted.current)
          setPassthrough(response.data as PassthroughStatus);
      })
      .catch(() => undefined);
  }, [account.role]);

  useEffect(refreshPassthrough, [refreshPassthrough]);

  const refresh = useCallback(() => {
    refreshPassthrough();
    client.reconnect();
    if (!eventsSocket.current) eventsConnect.current?.();
  }, [client, refreshPassthrough]);

  useEffect(() => {
    const onVisible = () => {
      if (document.visibilityState === 'visible') refresh();
    };
    document.addEventListener('visibilitychange', onVisible);
    return () => document.removeEventListener('visibilitychange', onVisible);
  }, [refresh]);

  // A reload does not unmount React, so nothing here would otherwise run on the
  // way out: the sockets die with the page and the registry only learns of it
  // when the connection times out. By then the next page has already said hello
  // and is told its own lease token is invalid, because the binding is still
  // held by a source the server thinks is alive - which is the refresh that
  // comes back with everything refused and a slot nobody can claim. Closing
  // here puts the goodbye ahead of the next page's hello. The captures stop
  // too, or the camera light stays on across the reload.
  useEffect(() => {
    const teardown = () => {
      client.close();
      window.clearTimeout(eventsRetry.current);
      eventsSocket.current?.close(1000);
      eventsSocket.current = undefined;
      void usbRelay.current?.close();
      for (const sinkID of [...captures.current.keys()]) void stopCapture(sinkID);
    };
    const onPageShow = (event: PageTransitionEvent) => {
      // Restored from the back/forward cache: the React tree survived, the
      // sockets did not, because we closed them on the way out.
      if (event.persisted) refresh();
    };

    window.addEventListener('pagehide', teardown);
    window.addEventListener('pageshow', onPageShow);
    return () => {
      window.removeEventListener('pagehide', teardown);
      window.removeEventListener('pageshow', onPageShow);
    };
  }, [client, refresh, stopCapture]);

  const clearAttempt = useCallback((sinkID: string) => {
    setRefusals((current) => {
      const next = { ...current };
      delete next[sinkID];
      return next;
    });
    setRevoked((current) => {
      const next = { ...current };
      delete next[sinkID];
      return next;
    });
  }, []);

  const pickOffer = useCallback(
    (available: DeviceOffer[], sinkID: string, selectedDeviceID?: string) => {
      const saved = localStorage.getItem(deviceKey(account.username, sinkID));
      return (
        available.find((device) => device.deviceID === selectedDeviceID) ||
        available.find((device) => device.deviceID === saved) ||
        available[0]
      );
    },
    [account.username]
  );

  const share = useCallback(
    async (sink: SourceSink, selectedDeviceID?: string) => {
      setBusy((current) => new Set(current).add(sink.id));
      clearError(sink.id);
      clearAttempt(sink.id);
      try {
        if (sink.kind === 'usb_device') {
          if (account.role !== 'admin') throw new Error('Administrator access is required');
          await usbRelay.current?.close();
          const relay = await WebUSBRelay.select(() => {
            closeRelay();
            void client.release('ffs.hybrid').catch(() => client.forget('ffs.hybrid'));
          });
          usbRelay.current = relay;
          setDevices((current) => ({ ...current, usb_device: [relay.offer] }));
          await client.setOffers(withOffers(devices, 'usb_device', [relay.offer]));
          await client.claim(sink.id, relay.offer);
          setActive((current) => new Set(current).add(sink.id));
          return;
        }
        let available = devices[sink.kind];
        if (available.length === 0) available = await discover(sink.kind, sink.kind !== 'speaker');
        const offer = pickOffer(available, sink.id, selectedDeviceID);
        await client.setOffers(withOffers(devices, sink.kind, available));
        await client.claim(sink.id, offer);
        localStorage.setItem(deviceKey(account.username, sink.id), offer.deviceID);
      } catch (error) {
        setError(sink.id, mediaError(error, t));
      } finally {
        setBusy((current) => {
          const next = new Set(current);
          next.delete(sink.id);
          return next;
        });
      }
    },
    [
      account.role,
      account.username,
      clearAttempt,
      clearError,
      client,
      closeRelay,
      devices,
      discover,
      pickOffer,
      setError,
      t
    ]
  );

  const takeover = useCallback(
    async (sink: SourceSink, selectedDeviceID?: string) => {
      setBusy((current) => new Set(current).add(sink.id));
      clearError(sink.id);
      try {
        let available = devices[sink.kind];
        if (available.length === 0) available = await discover(sink.kind, sink.kind !== 'speaker');
        const offer = pickOffer(available, sink.id, selectedDeviceID);
        await client.setOffers(withOffers(devices, sink.kind, available));
        const sourceID = client.sourceId();
        if (!sourceID) throw new Error('Media source disconnected');
        const response = await api.takeoverSource(sink.id, sourceID, offer.id);
        if (response.code === -3 && response.data) {
          setRefusals((current) => ({ ...current, [sink.id]: response.data as ClaimRefusal }));
          return;
        }
        if (response.code !== 0) throw new Error(response.msg);
        client.adopt(sink.id, offer, (response.data as api.ClaimGrant).token);
        localStorage.setItem(deviceKey(account.username, sink.id), offer.deviceID);
        clearAttempt(sink.id);
      } catch (error) {
        setError(sink.id, mediaError(error, t));
      } finally {
        setBusy((current) => {
          const next = new Set(current);
          next.delete(sink.id);
          return next;
        });
      }
    },
    [account.username, clearAttempt, clearError, client, devices, discover, pickOffer, setError, t]
  );

  const release = useCallback(
    async (sinkID: string) => {
      setBusy((current) => new Set(current).add(sinkID));
      clearError(sinkID);
      clearAttempt(sinkID);
      await stopCapture(sinkID);
      if (sinkID === 'ffs.hybrid') closeRelay();
      try {
        if (sourceConnection === 'connected' && owned.has(sinkID)) await client.release(sinkID);
        else {
          const response = await api.releaseSource(sinkID);
          if (response.code !== 0) throw new Error(response.msg);
          client.forget(sinkID);
        }
      } catch (error) {
        setError(sinkID, mediaError(error, t));
      } finally {
        setBusy((current) => {
          const next = new Set(current);
          next.delete(sinkID);
          return next;
        });
      }
    },
    [
      clearAttempt,
      clearError,
      client,
      closeRelay,
      owned,
      setError,
      sourceConnection,
      stopCapture,
      t
    ]
  );

  const choose = useCallback(
    (sinkID: string, deviceID: string) => {
      localStorage.setItem(deviceKey(account.username, sinkID), deviceID);
    },
    [account.username]
  );

  const setSlots = useCallback(async (slots: api.SourceSlot[]) => {
    const response = await api.setSourceSlots(slots);
    if (response.code !== 0) throw new Error(response.msg);
    setSnapshot(response.data as SourcesSnapshot);
  }, []);

  const setSinkMuted = useCallback((sinkID: string, value: boolean) => {
    setMuted((current) => {
      const next = new Set(current);
      if (value) next.add(sinkID);
      else next.delete(sinkID);
      return next;
    });
  }, []);

  useEffect(() => {
    for (const [sinkID, running] of captures.current) {
      if (running.kind === 'microphone') {
        (running.capture as MicrophoneCapture).setMuted(muted.has(sinkID));
      }
      if (running.kind === 'speaker') {
        (running.capture as SpeakerPlayback).setMuted(muted.has(sinkID));
      }
    }
  }, [muted]);

  const disconnectAll = useCallback(async () => {
    const response = await api.disconnectSources();
    if (response.code !== 0) throw new Error(response.msg);
    setSnapshot(response.data as SourcesSnapshot);
    for (const sinkID of captures.current.keys()) await stopCapture(sinkID);
    closeRelay();
  }, [closeRelay, stopCapture]);

  const value: DevicesState = {
    snapshot,
    eventsConnection,
    sourceConnection,
    devices,
    permissions,
    passthrough,
    owned,
    active,
    busy,
    muted,
    errors,
    refusals,
    revoked,
    share,
    takeover,
    release,
    choose,
    setMuted: setSinkMuted,
    setSlots,
    disconnectAll,
    refresh
  };

  return <DevicesContext.Provider value={value}>{children}</DevicesContext.Provider>;
};

// Every offer this browser still stands behind, with one kind replaced. The
// hello has to carry all of them, or claiming one slot revokes another.
function withOffers(
  devices: Record<SourceKind, DeviceOffer[]>,
  kind: SourceKind,
  available: DeviceOffer[]
) {
  const kinds: SourceKind[] = ['camera', 'microphone', 'speaker', 'usb_device'];
  return kinds.flatMap((candidate) => (candidate === kind ? available : devices[candidate]));
}

function streamID(kind: SourceKind, deviceID: string, index: number) {
  let hash = 2166136261;
  for (let i = 0; i < deviceID.length; i++) {
    hash ^= deviceID.charCodeAt(i);
    hash = Math.imul(hash, 16777619);
  }
  return `${kind === 'camera' ? 'cam' : 'mic'}_${(hash >>> 0).toString(36)}_${index}`;
}

function deviceKey(owner: string, sinkID: string) {
  return `nanokvm-media-device:${owner}:${sinkID}`;
}

function mediaError(error: unknown, t: TFunction) {
  if (error instanceof CaptureUnsupported) {
    return error.reason === 'insecure'
      ? t('devices.permission.insecure')
      : t(`devices.capture.${error.reason}`);
  }
  if (error instanceof DOMException && error.name === 'NotAllowedError') return 'Permission denied';
  if (error instanceof DOMException && error.name === 'NotFoundError')
    return 'Input device not found';
  return error instanceof Error ? error.message : 'Media operation failed';
}

function demandKey(sink: SourceSink) {
  const { width = 0, height = 0, fps = 0 } = sink.demand;
  return `${width}x${height}@${fps}`;
}
