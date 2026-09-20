import type { ExitConfig, ExitMode, ExitPlatform } from './types.ts';

// scheme is http or https, host is host[:port], both exactly as the server
// templated them (D15) or as the browser reached this page
export type Origin = {
  scheme: string;
  host: string;
};

export function wsScheme(scheme: string): string {
  return scheme === 'https' ? 'wss' : 'ws';
}

// getBaseUrl() gives scheme://host[:port]; the commands need the two apart
export function parseOrigin(baseUrl: string): Origin {
  const separator = baseUrl.indexOf('://');
  if (separator < 0) return { scheme: '', host: baseUrl };

  return {
    scheme: baseUrl.slice(0, separator),
    host: baseUrl.slice(separator + 3).replace(/\/+$/, '')
  };
}

// a url boundary: the host is followed by a path, a space, a quote or the end,
// so nanokvm does not match inside nanokvm.example.org
function replaceUrl(command: string, from: string, to: string): string {
  let out = '';
  let index = 0;

  for (;;) {
    const found = command.indexOf(from, index);
    if (found < 0) break;

    const after = command.charAt(found + from.length);
    const isBoundary = after === '' || after === '/' || /[\s'"`)\]]/.test(after);

    out += command.slice(index, found) + (isBoundary ? to : from);
    index = found + from.length;
  }

  return out + command.slice(index);
}

// The server templates the commands from the request it saw (D15). Behind a
// proxy that is not the address the operator's browser has, so the panel
// readdresses every http(s):// and ws(s):// url to window.location, but only
// the host: -k, the PowerShell certificate callback, --tls-verify-certificate,
// the notes and the fingerprint all follow the scheme, and re-templating them
// here would have to duplicate the server. A scheme the server did not see is
// therefore shown as the server templated it, with a warning.
export type CommandView = {
  text: string;
  // the scheme the shown command dials with
  scheme: string;
  // the host was swapped for the one the browser reached
  rewritten: boolean;
  // the server and the browser disagree on the scheme: nothing was rewritten
  schemeMismatch: boolean;
  // the command carries the token over plain http
  cleartext: boolean;
  // https, but the command trusts the transport instead of verifying it: the
  // device's certificate is self-signed, so the token is what authenticates
  // the session. Both modes follow the same rule.
  trustsTransport: boolean;
};

export function presentCommand(
  mode: ExitMode,
  command: { command: string } | undefined,
  server: Origin,
  local: Origin
): CommandView {
  const known = !!server.scheme && !!server.host;
  const schemeMismatch = known && !!local.scheme && server.scheme !== local.scheme;
  const rewritten = known && !schemeMismatch && server.host !== local.host;
  const scheme = known ? server.scheme : local.scheme;

  let text = command?.command ?? '';
  if (rewritten) {
    text = replaceUrl(text, `${scheme}://${server.host}`, `${scheme}://${local.host}`);
    text = replaceUrl(
      text,
      `${wsScheme(scheme)}://${server.host}`,
      `${wsScheme(scheme)}://${local.host}`
    );
  }

  const cleartext = scheme !== 'https';

  return {
    text,
    scheme,
    rewritten,
    schemeMismatch,
    cleartext,
    // The server only emits the verifying form behind a CA-signed certificate,
    // so in each mode the tell is what it had to add to get past a certificate
    // no CA can vouch for.
    trustsTransport:
      !!command &&
      !cleartext &&
      (mode === 'wstunnel'
        ? !command.command.includes('--tls-verify')
        : command.command.includes('-fsSLk') || command.command.includes('NanoKVMExitTrust'))
  };
}

// The token gate answers a disabled or pending slot with the same 404 as a
// wrong token and charges the source a failure (D10), so the panel asks for
// the script only while the gate would serve it: a few clicks on a disabled
// slot would otherwise lock the operator's own address out of /exit/:slot/*.
export function isScriptServed(status: { enabled: boolean; pending: boolean }): boolean {
  return status.enabled && !status.pending;
}

// a 404 is the gate (disabled slot, stale token, rate-limited source), not a
// broken server, and the message has to say so
export function scriptErrorKey(httpStatus: number): 'scriptGated' | 'scriptFailed' {
  return httpStatus === 404 ? 'scriptGated' : 'scriptFailed';
}

// One status request in flight at a time: a slow device must not pile up
// polls. An interval tick during a request is dropped; a refresh after an
// action is queued behind the request in flight, because that one was sent
// before the action and answers with the state before it.
export function serialRefresher(run: () => Promise<unknown>): {
  poll: () => void;
  refresh: () => void;
} {
  let inFlight = false;
  let queued = false;

  function start() {
    inFlight = true;
    run()
      .catch(() => undefined)
      .then(() => {
        inFlight = false;
        if (queued) {
          queued = false;
          start();
        }
      });
  }

  return {
    poll() {
      if (!inFlight) start();
    },
    refresh() {
      if (inFlight) queued = true;
      else start();
    }
  };
}

// the fields the advanced form edits; the mode has its own selector
export type ExitAdvancedValues = Pick<ExitConfig, 'dns' | 'mtu' | 'allowPrivate' | 'pinPeer'>;

// A save lands on the config as it is when the save answers, not as it was
// when it started: a mode switch that resolved in between must not be undone
// by the save writing back the mode it captured.
export function mergeSaved(
  current: ExitConfig | undefined,
  saved: ExitAdvancedValues
): ExitConfig | undefined {
  return current ? { ...current, ...saved } : current;
}

// The form is re-seeded from the server only when the values it edits change.
// A mode switch replaces the config object; keying on identity would discard
// an unsaved DNS or MTU edit without a word.
export function advancedValuesKey(config?: ExitConfig): string {
  if (!config) return '';
  return JSON.stringify([config.dns ?? [], config.mtu, config.allowPrivate, config.pinPeer]);
}

export function formatUptime(seconds: number): string {
  const total = Math.max(0, Math.floor(seconds));
  const days = Math.floor(total / 86400);
  const hours = Math.floor((total % 86400) / 3600);
  const minutes = Math.floor((total % 3600) / 60);
  const secs = total % 60;

  if (days > 0) return `${days}d ${hours}h`;
  if (hours > 0) return `${hours}h ${minutes}m`;
  if (minutes > 0) return `${minutes}m ${secs}s`;
  return `${secs}s`;
}

export function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`;

  const kib = bytes / 1024;
  if (kib < 1024) return `${kib.toFixed(1)} KiB`;

  const mib = kib / 1024;
  if (mib < 1024) return `${mib.toFixed(1)} MiB`;

  return `${(mib / 1024).toFixed(2)} GiB`;
}

// Go's zero time reaches the browser as year 1, which is not a time to show
export function formatTime(value?: string | null): string {
  if (!value) return '';

  const time = new Date(value);
  if (Number.isNaN(time.getTime()) || time.getUTCFullYear() < 2000) return '';

  return time.toLocaleString();
}

export function detectPlatform(userAgent: string): ExitPlatform {
  if (/Windows/i.test(userAgent)) return 'windows';
  if (/Mac OS X|Macintosh/i.test(userAgent)) return 'macos';
  return 'linux';
}

// dotted quad, no leading zeros, no shorthand: what net.ParseIP takes
const ipv4 = /^(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}$/;

function parseIPv4(address: string): number[] | null {
  if (!ipv4.test(address)) return null;
  return address.split('.').map(Number);
}

// The URL parser is the one full IPv6 parser a browser ships: it refuses
// what net.ParseIP refuses (':::', ':1', nine hextets, a zone) and normalises
// an embedded IPv4 tail into hextets, so the eight groups can be read back.
function parseIPv6(address: string): number[] | null {
  if (!address.includes(':') || address.includes('%')) return null;

  let canonical: string;
  try {
    canonical = new URL(`http://[${address}]`).hostname;
  } catch {
    return null;
  }
  if (!canonical.startsWith('[') || !canonical.endsWith(']')) return null;

  const [head, tail = ''] = canonical.slice(1, -1).split('::');
  const left = head ? head.split(':') : [];
  const right = tail ? tail.split(':') : [];
  const missing = 8 - left.length - right.length;
  if (missing < 0) return null;

  return [...left, ...Array<string>(missing).fill('0'), ...right].map((part) => parseInt(part, 16));
}

// the resolvers are dialled as literals through the exit (D3), and the server
// validates the same way (`dive,ip`), so a name is refused before it is sent
export function isValidDNS(value: string): boolean {
  const address = value.trim();
  return parseIPv4(address) !== null || parseIPv6(address) !== null;
}

// D21's prefixes, mirrored so the same inline hint fires here as the front
// door would refuse the resolver; the always-denied set can never answer,
// the private set only with allowPrivate
type Prefix4 = [number, number, number, number, number]; // a.b.c.d/bits

const alwaysDenied4: Prefix4[] = [
  [0, 0, 0, 0, 8],
  [127, 0, 0, 0, 8],
  [169, 254, 0, 0, 16],
  [198, 18, 0, 0, 15],
  [224, 0, 0, 0, 3]
];
const privateDenied4: Prefix4[] = [
  [10, 0, 0, 0, 8],
  [100, 64, 0, 0, 10],
  [172, 16, 0, 0, 12],
  [192, 168, 0, 0, 16]
];

function inPrefix4(octets: number[], prefix: Prefix4): boolean {
  const value = ((octets[0] << 24) | (octets[1] << 16) | (octets[2] << 8) | octets[3]) >>> 0;
  const base = ((prefix[0] << 24) | (prefix[1] << 16) | (prefix[2] << 8) | prefix[3]) >>> 0;
  const mask = prefix[4] === 0 ? 0 : (0xffffffff << (32 - prefix[4])) >>> 0;
  return (value & mask) >>> 0 === (base & mask) >>> 0;
}

function isMapped6(hextets: number[]): boolean {
  return hextets.slice(0, 5).every((part) => part === 0) && hextets[5] === 0xffff;
}

function alwaysDenied6(h: number[]): boolean {
  const zero = h.every((part) => part === 0);
  const loopback = h.slice(0, 7).every((part) => part === 0) && h[7] === 1;
  const linkLocal = (h[0] & 0xffc0) === 0xfe80;
  const multicast = (h[0] & 0xff00) === 0xff00;
  return zero || loopback || linkLocal || multicast;
}

function privateDenied6(h: number[]): boolean {
  return (h[0] & 0xfe00) === 0xfc00;
}

export type ResolverProblem = '' | 'dnsInvalid' | 'dnsDenied' | 'dnsPrivate';

export function resolverProblem(value: string, allowPrivate: boolean): ResolverProblem {
  const address = value.trim();

  let octets = parseIPv4(address);
  if (!octets) {
    const hextets = parseIPv6(address);
    if (!hextets) return 'dnsInvalid';

    if (isMapped6(hextets)) {
      // judged as ipv4, as Policy.Allow unmaps
      octets = [hextets[6] >> 8, hextets[6] & 0xff, hextets[7] >> 8, hextets[7] & 0xff];
    } else {
      if (alwaysDenied6(hextets)) return 'dnsDenied';
      if (!allowPrivate && privateDenied6(hextets)) return 'dnsPrivate';
      return '';
    }
  }

  if (alwaysDenied4.some((prefix) => inPrefix4(octets, prefix))) return 'dnsDenied';
  if (!allowPrivate && privateDenied4.some((prefix) => inPrefix4(octets, prefix))) {
    return 'dnsPrivate';
  }
  return '';
}
