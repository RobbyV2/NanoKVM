type WorkletPort = { postMessage: (message: Float32Array, transfer: Transferable[]) => void };

declare const sampleRate: number;
declare class AudioWorkletProcessor {
  readonly port: WorkletPort;
}
declare function registerProcessor(name: string, processor: typeof AudioWorkletProcessor): void;

class NanoKVMAudioProcessor extends AudioWorkletProcessor {
  private readonly chunkFrames = Math.round(sampleRate / 50);
  private chunk = new Float32Array(this.chunkFrames);
  private length = 0;

  process(inputs: Float32Array[][]) {
    const input = inputs[0]?.[0];
    if (!input) return true;
    let offset = 0;
    while (offset < input.length) {
      const count = Math.min(input.length - offset, this.chunk.length - this.length);
      this.chunk.set(input.subarray(offset, offset + count), this.length);
      this.length += count;
      offset += count;
      if (this.length === this.chunk.length) {
        const chunk = this.chunk;
        this.port.postMessage(chunk, [chunk.buffer]);
        this.chunk = new Float32Array(this.chunkFrames);
        this.length = 0;
      }
    }
    return true;
  }
}

registerProcessor('nanokvm-audio', NanoKVMAudioProcessor);
