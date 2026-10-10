export class QuadClickMatcher {
  private stamps: number[] = [];

  reset() {
    this.stamps = [];
  }

  click(allowed: boolean, now = Date.now()): boolean {
    if (!allowed) {
      this.reset();
      return false;
    }
    this.stamps = this.stamps.filter((stamp) => now - stamp < 2000);
    this.stamps.push(now);
    if (this.stamps.length < 4) return false;
    this.reset();
    return true;
  }
}
