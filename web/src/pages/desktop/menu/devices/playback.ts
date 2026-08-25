import { captureSupport, CaptureUnsupported } from './capture.ts';

export class SpeakerPlayback {
  private context?: AudioContext;
  private worklet?: AudioWorkletNode;
  private gain?: GainNode;
  private muted = false;
  private frames = 0;

  async start(muted = false) {
    await this.stop();
    const blocked = captureSupport('speaker');
    if (blocked) throw new CaptureUnsupported(blocked);
    const context = new AudioContext({ latencyHint: 'interactive', sampleRate: 48000 });
    this.context = context;
    try {
      await context.audioWorklet.addModule(new URL('./playback.worklet.ts', import.meta.url));
      await context.resume();
      if (context.state !== 'running') throw new Error('Click Listen again to start playback');
      this.worklet = new AudioWorkletNode(context, 'nanokvm-playback', {
        numberOfInputs: 0,
        numberOfOutputs: 1,
        outputChannelCount: [1]
      });
      this.gain = context.createGain();
      this.gain.gain.value = muted ? 0 : 1;
      this.muted = muted;
      this.worklet.connect(this.gain).connect(context.destination);
    } catch (error) {
      await this.stop();
      throw error;
    }
  }

  // One 20 ms packet from the gadget. Conversion is cheap and the samples are
  // transferred, so nothing here copies audio a second time.
  push(payload: ArrayBuffer) {
    const worklet = this.worklet;
    if (!worklet) return false;
    const view = new DataView(payload);
    const count = Math.floor(payload.byteLength / 2);
    const samples = new Float32Array(count);
    for (let i = 0; i < count; i++) {
      const sample = view.getInt16(i * 2, true);
      samples[i] = sample / (sample < 0 ? 32768 : 32767);
    }
    worklet.port.postMessage({ samples }, [samples.buffer]);
    this.frames += count;
    return true;
  }

  reset() {
    this.worklet?.port.postMessage({ reset: true });
  }

  setMuted(muted: boolean) {
    this.muted = muted;
    if (this.gain) this.gain.gain.value = muted ? 0 : 1;
  }

  isMuted() {
    return this.muted;
  }

  playedFrames() {
    return this.frames;
  }

  async stop() {
    this.worklet?.disconnect();
    this.gain?.disconnect();
    const context = this.context;
    this.worklet = undefined;
    this.gain = undefined;
    this.context = undefined;
    this.frames = 0;
    if (context && context.state !== 'closed') await context.close();
  }
}
