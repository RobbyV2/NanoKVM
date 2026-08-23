import type { Demand, SourceKind } from '@/api/sources.ts';

import { PcmPacketizer } from './pcm.ts';

type FrameHandler = (payload: ArrayBuffer) => void;
type ErrorHandler = (message: string) => void;

type CameraWorkerResponse = { id: number; payload?: ArrayBuffer; error?: string };

export function captureSupport(kind: SourceKind) {
  if (!navigator.mediaDevices?.getUserMedia) return 'Media capture is unavailable';
  if (
    kind === 'camera' &&
    (!window.Worker || !window.OffscreenCanvas || !window.createImageBitmap)
  ) {
    return 'This browser cannot encode camera frames';
  }
  if (
    kind === 'microphone' &&
    (!window.AudioContext ||
      !window.AudioWorkletNode ||
      !('audioWorklet' in AudioContext.prototype))
  ) {
    return 'This browser cannot process microphone audio';
  }
  return '';
}

export class CameraCapture {
  private worker?: Worker;
  private stream?: MediaStream;
  private video?: HTMLVideoElement;
  private timer?: number;
  private encoding = false;
  private frameID = 0;
  private stopped = true;

  async start(deviceID: string, demand: Demand, onFrame: FrameHandler, onError: ErrorHandler) {
    this.stop();
    const unsupported = captureSupport('camera');
    if (unsupported) throw new Error(unsupported);
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
      if (data.payload) onFrame(data.payload);
    };
    this.worker.onerror = () => {
      this.encoding = false;
      onError('Camera encoder stopped');
    };
    this.stopped = false;

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

  stop() {
    this.stopped = true;
    if (this.timer) window.clearTimeout(this.timer);
    this.timer = undefined;
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
    const unsupported = captureSupport('microphone');
    if (unsupported) throw new Error(unsupported);
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
    await this.context.audioWorklet.addModule(new URL('./audio.worklet.ts', import.meta.url));
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
