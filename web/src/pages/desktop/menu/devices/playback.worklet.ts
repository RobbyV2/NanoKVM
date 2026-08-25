// This file is an ES module - an AudioWorklet module always is - which is also
// what keeps its ambient worklet declarations out of the global scope that
// audio.worklet.ts already declares them in.
export {};

type WorkletMessage = { samples?: Float32Array; reset?: boolean };
type WorkletPort = { onmessage: ((event: MessageEvent<WorkletMessage>) => void) | null };

declare const sampleRate: number;
declare class AudioWorkletProcessor {
  readonly port: WorkletPort;
}
declare function registerProcessor(name: string, processor: typeof AudioWorkletProcessor): void;

const sourceRate = 48000;
// Three 20 ms packets before the first sample is played. Below that a single
// late websocket frame is audible; above it the target host's audio lags.
const prefillFrames = sourceRate * 0.06;
// Anything past 400 ms is a browser that stopped keeping up. Dropping the
// oldest audio bounds both the delay and the memory this worklet can hold.
const maxFrames = sourceRate * 0.4;

// The gadget always sends 48 kHz mono; the output device runs at whatever rate
// the browser chose, so every output frame is interpolated from the source
// timeline rather than assuming the two agree.
class NanoKVMPlaybackProcessor extends AudioWorkletProcessor {
  private buffer = new Float32Array(0);
  private position = 0;
  private readonly step = sourceRate / sampleRate;
  private playing = false;

  constructor() {
    super();
    this.port.onmessage = ({ data }) => {
      if (data.reset) {
        this.buffer = new Float32Array(0);
        this.position = 0;
        this.playing = false;
        return;
      }
      if (!data.samples) return;
      const kept = this.buffer.subarray(Math.floor(this.position));
      this.position -= Math.floor(this.position);
      const merged = new Float32Array(kept.length + data.samples.length);
      merged.set(kept, 0);
      merged.set(data.samples, kept.length);
      this.buffer = merged.length > maxFrames ? merged.subarray(merged.length - maxFrames) : merged;
      if (merged.length > maxFrames) this.position = 0;
    };
  }

  process(_inputs: Float32Array[][], outputs: Float32Array[][]) {
    const output = outputs[0]?.[0];
    if (!output) return true;
    const available = this.buffer.length - this.position;
    if (!this.playing) {
      if (available < prefillFrames) {
        output.fill(0);
        return true;
      }
      this.playing = true;
    }
    for (let i = 0; i < output.length; i++) {
      const index = Math.floor(this.position);
      if (index + 1 >= this.buffer.length) {
        // Underrun: silence out the rest and wait for the jitter buffer to
        // refill rather than stuttering on the same few samples.
        output.fill(0, i);
        this.playing = false;
        this.buffer = new Float32Array(0);
        this.position = 0;
        return true;
      }
      const part = this.position - index;
      output[i] = this.buffer[index] + (this.buffer[index + 1] - this.buffer[index]) * part;
      this.position += this.step;
    }
    return true;
  }
}

registerProcessor('nanokvm-playback', NanoKVMPlaybackProcessor);
