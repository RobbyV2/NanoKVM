import type { Demand, SourceKind } from '@/api/sources.ts';

import {
  captureHealthyMS,
  captureProgress,
  captureRestartDelay,
  captureStalled,
  captureWatchdogIntervalMS,
  type CaptureSample
} from './health.ts';
import { PcmPacketizer } from './pcm.ts';
// See playback.ts: the bundler only emits the worklet for the `?worker&url`
// form, so a production build served nothing for the bare new URL() version.
import audioWorkletUrl from './audio.worklet.ts?worker&url';

type FrameHandler = (payload: ArrayBuffer) => void;
type ErrorHandler = (message: string) => void;

type CameraWorkerResponse = { id: number; payload?: ArrayBuffer; error?: string };

export type CaptureBlock = 'insecure' | 'unsupported' | 'camera' | 'microphone' | 'speaker';

// getUserMedia is absent both on a browser that never had it and on a browser
// that has it but withholds it outside a secure context, and only the second is
// fixable by the user.
export function captureSupport(kind: SourceKind): CaptureBlock | '' {
  // A speaker is played, not captured: it needs neither getUserMedia nor a
  // secure context, only the worklet that renders what the gadget sends.
  if (kind === 'speaker') {
    return audioWorkletMissing() ? 'speaker' : '';
  }
  if (!navigator.mediaDevices?.getUserMedia) {
    return window.isSecureContext === false ? 'insecure' : 'unsupported';
  }
  if (
    kind === 'camera' &&
    (!window.Worker || !window.OffscreenCanvas || !window.createImageBitmap)
  ) {
    return 'camera';
  }
  if (kind === 'microphone' && audioWorkletMissing()) {
    return 'microphone';
  }
  return '';
}

function audioWorkletMissing() {
  return (
    !window.AudioContext || !window.AudioWorkletNode || !('audioWorklet' in AudioContext.prototype)
  );
}

// Carries the reason as a code so the view can translate it; the message is only
// a fallback for logs.
export class CaptureUnsupported extends Error {
  readonly reason: CaptureBlock;

  constructor(reason: CaptureBlock) {
    super(`capture unsupported: ${reason}`);
    this.name = 'CaptureUnsupported';
    this.reason = reason;
  }
}

type CameraParams = {
  deviceID: string;
  demand: Demand;
  onFrame: FrameHandler;
  onError: ErrorHandler;
};

export class CameraCapture {
  private worker?: Worker;
  private stream?: MediaStream;
  private video?: HTMLVideoElement;
  private timer?: number;
  private watchdog?: number;
  private encoding = false;
  private frameID = 0;
  private stopped = true;
  private frames = 0;
  private restarts = 0;
  private openedAt = 0;
  // Every attempt carries the generation it belongs to, so a stop or a newer
  // restart silently retires the timers and promises of the old one instead of
  // letting two pipelines fight over one camera.
  private generation = 0;
  private params?: CameraParams;
  private progress: CaptureSample = { at: 0, frames: 0, live: false };

  async start(deviceID: string, demand: Demand, onFrame: FrameHandler, onError: ErrorHandler) {
    this.params = { deviceID, demand, onFrame, onError };
    this.restarts = 0;
    await this.open(++this.generation);
  }

  stop() {
    this.generation++;
    this.params = undefined;
    this.teardown();
  }

  // A capture that stops producing is re-established rather than left sitting
  // there: the operator gets their picture back without touching anything, and
  // the backoff keeps a camera that is genuinely gone from being reopened in a
  // tight loop.
  private async restart(generation: number) {
    if (this.generation !== generation || !this.params) return;
    const params = this.params;
    if (this.openedAt && Date.now() - this.openedAt >= captureHealthyMS) this.restarts = 0;
    const delay = captureRestartDelay(this.restarts);
    this.restarts++;
    const attempt = ++this.generation;
    this.teardown();
    await new Promise((resolve) => window.setTimeout(resolve, delay));
    if (this.generation !== attempt || this.params !== params) return;
    try {
      await this.open(attempt);
    } catch (error) {
      if (this.generation !== attempt || this.params !== params) return;
      params.onError(error instanceof Error ? error.message : 'Camera restart failed');
      void this.restart(attempt);
    }
  }

  private sample(): CaptureSample {
    const track = this.stream?.getVideoTracks()[0];
    return {
      at: Date.now(),
      frames: this.frames,
      live: !!track && track.readyState === 'live'
    };
  }

  private watch(generation: number) {
    this.progress = this.sample();
    this.watchdog = window.setInterval(() => {
      if (this.generation !== generation || this.stopped) return;
      const latest = this.sample();
      this.progress = captureProgress(this.progress, latest);
      if (!captureStalled(this.progress, latest)) return;
      this.params?.onError('Camera stopped producing frames; reconnecting');
      void this.restart(generation);
    }, captureWatchdogIntervalMS);
  }

  private async open(generation: number) {
    const params = this.params;
    if (!params || this.generation !== generation) return;
    const { deviceID, demand, onFrame, onError } = params;
    this.teardown();
    const blocked = captureSupport('camera');
    if (blocked) throw new CaptureUnsupported(blocked);
    const width = demand.width || 640;
    const height = demand.height || 480;
    const fps = demand.fps || 15;
    this.stream = await navigator.mediaDevices.getUserMedia({
      video: {
        deviceId: deviceID ? { exact: deviceID } : undefined,
        width: { ideal: width },
        height: { ideal: height },
        frameRate: { ideal: fps, max: fps }
      }
    });
    this.video = document.createElement('video');
    this.video.muted = true;
    this.video.playsInline = true;
    this.video.srcObject = this.stream;
    await this.video.play();
    this.worker = new Worker(new URL('./camera.worker.ts', import.meta.url), { type: 'module' });
    this.worker.onmessage = ({ data }: MessageEvent<CameraWorkerResponse>) => {
      this.encoding = false;
      if (data.error) onError(data.error);
      if (data.payload) {
        this.frames++;
        onFrame(data.payload);
      }
    };
    this.worker.onerror = () => {
      this.encoding = false;
      onError('Camera encoder stopped');
    };
    this.stopped = false;
    this.frames = 0;
    this.openedAt = Date.now();
    this.watch(generation);

    const interval = 1000 / fps;
    const encode = async () => {
      if (this.stopped || !this.video) return;
      if (!this.encoding && this.video.readyState >= HTMLMediaElement.HAVE_CURRENT_DATA) {
        this.encoding = true;
        try {
          const bitmap = await createImageBitmap(this.video);
          if (this.stopped) {
            bitmap.close();
            return;
          }
          this.worker?.postMessage({ id: ++this.frameID, bitmap, width, height }, [bitmap]);
        } catch (error) {
          this.encoding = false;
          onError(error instanceof Error ? error.message : 'Camera frame failed');
        }
      }
      this.timer = window.setTimeout(encode, interval);
    };
    await encode();
  }

  private teardown() {
    this.stopped = true;
    if (this.timer) window.clearTimeout(this.timer);
    this.timer = undefined;
    if (this.watchdog) window.clearInterval(this.watchdog);
    this.watchdog = undefined;
    this.worker?.terminate();
    this.worker = undefined;
    this.stream?.getTracks().forEach((track) => track.stop());
    this.stream = undefined;
    if (this.video) this.video.srcObject = null;
    this.video = undefined;
    this.encoding = false;
  }
}

export class MicrophoneCapture {
  private stream?: MediaStream;
  private context?: AudioContext;
  private source?: MediaStreamAudioSourceNode;
  private worklet?: AudioWorkletNode;
  private mute?: GainNode;

  setMuted(muted: boolean) {
    this.stream?.getAudioTracks().forEach((track) => (track.enabled = !muted));
  }

  async start(deviceID: string, onFrame: FrameHandler, muted = false) {
    await this.stop();
    const blocked = captureSupport('microphone');
    if (blocked) throw new CaptureUnsupported(blocked);
    this.stream = await navigator.mediaDevices.getUserMedia({
      audio: {
        deviceId: deviceID ? { exact: deviceID } : undefined,
        channelCount: 1,
        echoCancellation: false,
        noiseSuppression: false,
        autoGainControl: false
      }
    });
    this.context = new AudioContext({ latencyHint: 'interactive' });
    await this.context.audioWorklet.addModule(audioWorkletUrl);
    await this.context.resume();
    if (this.context.state !== 'running') throw new Error('Click Share again to start microphone');
    const packetizer = new PcmPacketizer(this.context.sampleRate);
    this.source = this.context.createMediaStreamSource(this.stream);
    this.worklet = new AudioWorkletNode(this.context, 'nanokvm-audio', {
      numberOfInputs: 1,
      numberOfOutputs: 1,
      outputChannelCount: [1]
    });
    this.mute = this.context.createGain();
    this.mute.gain.value = 0;
    this.worklet.port.onmessage = ({ data }: MessageEvent<Float32Array>) => {
      for (const packet of packetizer.push(data)) onFrame(packet);
    };
    this.source.connect(this.worklet).connect(this.mute).connect(this.context.destination);
    this.setMuted(muted);
  }

  async stop() {
    this.source?.disconnect();
    this.worklet?.disconnect();
    this.mute?.disconnect();
    this.stream?.getTracks().forEach((track) => track.stop());
    if (this.context && this.context.state !== 'closed') await this.context.close();
    this.stream = undefined;
    this.context = undefined;
    this.source = undefined;
    this.worklet = undefined;
    this.mute = undefined;
  }
}
