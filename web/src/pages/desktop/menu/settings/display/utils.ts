import type { EdidPreflight, EdidResult, EdidSummary } from '@/api/edid.ts';

export type CheckLevel = 'error' | 'warning' | 'info';

export type Check = {
  level: CheckLevel;
  key?: string;
  text?: string;
};

export type Mode = {
  width: number;
  height: number;
  refresh: number;
  interlaced: boolean;
};

// the tool writes modes as 1920x1080p60
const modePattern = /^(\d+)x(\d+)([pi])(\d+)$/;

// the server formats the edid version as major.minor
const versionPattern = /^(\d+)\.(\d+)$/;

// /etc/kvm/hw names the board, and the tool's own labels for it are these three
const hardwareNames: Record<string, string> = {
  CUBE_A: 'NanoKVM Cube (alpha)',
  CUBE_B: 'NanoKVM Cube (beta)',
  PCIE_A: 'NanoKVM PCIe'
};

export function parseMode(mode?: string): Mode | undefined {
  const matched = mode ? modePattern.exec(mode) : null;
  if (!matched) return undefined;

  return {
    width: Number(matched[1]),
    height: Number(matched[2]),
    interlaced: matched[3] === 'i',
    refresh: Number(matched[4])
  };
}

export function formatMode(mode?: string): string {
  const parsed = parseMode(mode);
  if (!parsed) return mode ?? '';

  return `${parsed.width} × ${parsed.height} @ ${parsed.refresh} Hz`;
}

// compared part by part, because as strings '1.10' sorts below '1.4'
export function isOlderVersion(version: string | undefined, major: number, minor: number): boolean {
  const matched = version ? versionPattern.exec(version) : null;
  if (!matched) return false;

  const gotMajor = Number(matched[1]);
  const gotMinor = Number(matched[2]);

  return gotMajor === major ? gotMinor < minor : gotMajor < major;
}

// the board and chip the preflight detected, empty when it identified neither
export function hardwareName(preflight?: EdidPreflight): string {
  const product = preflight?.product ? hardwareNames[preflight.product] : undefined;
  if (!product) return '';

  return preflight?.chip ? `${product} · ${preflight.chip}` : product;
}

// errors block the apply, warnings and information do not
export function buildChecks(summary?: Partial<EdidSummary>, preflight?: EdidPreflight): Check[] {
  const checks: Check[] = [];

  if (preflight && !preflight.supported) {
    checks.push({ level: 'error', key: 'unsupported', text: preflight.reason });
  }
  if (preflight && !preflight.toolAvailable) {
    checks.push({ level: 'error', key: 'toolMissing' });
  }

  if (summary) {
    const mode = parseMode(summary.preferredMode);

    if (mode?.interlaced) {
      checks.push({ level: 'warning', key: 'interlaced' });
    }
    if (mode && (mode.height > 1080 || mode.refresh > 60)) {
      checks.push({ level: 'warning', key: 'tooLarge' });
    }
    if (summary.audio === false) {
      checks.push({ level: 'warning', key: 'noAudio' });
    }
    if (isOlderVersion(summary.version, 1, 4)) {
      checks.push({ level: 'warning', key: 'oldVersion' });
    }

    checks.push({ level: 'info', key: 'hdmiNotice' });
    if (preflight?.requiresPowerCycle) {
      checks.push({ level: 'info', key: 'powerCycleNotice' });
    }
  }

  return checks;
}

// the flash region was touched on every outcome the server sets this on, not
// only on the ones that verified, and a half written region is where the trip
// to the device stops being optional
export function powerCycleNotice(result?: EdidResult): string {
  if (!result?.requiresPowerCycle) return '';

  return result.verified ? 'powerCycleNotice' : 'powerCycleUnverified';
}
