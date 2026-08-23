const targetRate = 48000;
const packetSamples = 960;
const maxInputSamples = 4096;

export class PcmPacketizer {
  private readonly step: number;
  private input: number[] = [];
  private read = 0;
  private packet = new ArrayBuffer(packetSamples * 2);
  private packetView = new DataView(this.packet);
  private packetLength = 0;

  constructor(inputRate: number) {
    if (!Number.isFinite(inputRate) || inputRate < 8000 || inputRate > 192000) {
      throw new Error('audio sample rate');
    }
    this.step = inputRate / targetRate;
  }

  push(samples: Float32Array) {
    const packets: ArrayBuffer[] = [];
    for (const sample of samples) this.input.push(sample);
    if (this.input.length > maxInputSamples) {
      const dropped = this.input.length - maxInputSamples;
      this.input.splice(0, dropped);
      this.read = Math.max(0, this.read - dropped);
    }

    while (this.read + 1 < this.input.length) {
      const index = Math.floor(this.read);
      const part = this.read - index;
      const sample = this.input[index] + (this.input[index + 1] - this.input[index]) * part;
      const bounded = Math.max(-1, Math.min(1, sample));
      this.packetView.setInt16(
        this.packetLength++ * 2,
        Math.round(bounded * (bounded < 0 ? 32768 : 32767)),
        true
      );
      this.read += this.step;
      if (this.packetLength === packetSamples) {
        packets.push(this.packet);
        this.packet = new ArrayBuffer(packetSamples * 2);
        this.packetView = new DataView(this.packet);
        this.packetLength = 0;
      }
    }

    const consumed = Math.floor(this.read);
    if (consumed > 0) {
      this.input.splice(0, consumed);
      this.read -= consumed;
    }
    return packets;
  }

  reset() {
    this.input = [];
    this.read = 0;
    this.packet = new ArrayBuffer(packetSamples * 2);
    this.packetView = new DataView(this.packet);
    this.packetLength = 0;
  }
}
