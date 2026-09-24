// Ctrl-Ctrl prefix from the chaice content script: two non-repeat Control
// keydowns within 500 ms, then an action key matched on `code`. The action
// key is swallowed from its keydown until its keyup so the target never sees it.

export const HOTKEY_CODES = [
  'KeyA',
  'KeyC',
  'KeyT',
  'KeyF',
  'KeyS',
  'KeyP',
  'KeyL',
  'KeyR',
  'KeyM',
  'ArrowUp',
  'ArrowDown'
] as const;

export type HotkeyCode = (typeof HOTKEY_CODES)[number];

const PREFIX_WINDOW_MS = 500;

export function isHotkeyCode(code: string): code is HotkeyCode {
  return (HOTKEY_CODES as readonly string[]).includes(code);
}

export class HotkeyMatcher {
  private ctrlTimes: number[] = [];
  private held: string | null = null;
  private readonly now: () => number;

  // Plain field, not a parameter property: Node's strip-only TS mode rejects those.
  constructor(now: () => number = () => Date.now()) {
    this.now = now;
  }

  keydown(e: { code: string; repeat: boolean }): { action: HotkeyCode | null; swallow: boolean } {
    if (this.held !== null && e.code === this.held) {
      return { action: null, swallow: true };
    }
    if ((e.code === 'ControlLeft' || e.code === 'ControlRight') && !e.repeat) {
      const t = this.now();
      this.ctrlTimes.push(t);
      this.ctrlTimes = this.ctrlTimes.filter((x) => t - x < PREFIX_WINDOW_MS);
      if (this.ctrlTimes.length > 2) this.ctrlTimes.shift();
    }
    if (this.ctrlTimes.length === 2 && isHotkeyCode(e.code)) {
      this.ctrlTimes = [];
      this.held = e.code;
      return { action: e.code, swallow: true };
    }
    return { action: null, swallow: false };
  }

  keyup(e: { code: string }): boolean {
    if (this.held !== null && e.code === this.held) {
      this.held = null;
      return true;
    }
    return false;
  }
}
