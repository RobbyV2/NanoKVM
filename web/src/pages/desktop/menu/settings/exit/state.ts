import type { ExitMode, ExitPlatform } from './types.ts';

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
  // the script pins the fingerprint of the certificate the exit will see
  pinsFingerprint: boolean;
  // the script pins the device leaf, but the exit dials a readdressed host that
  // may terminate tls with another certificate
  pinUncertain: boolean;
  // wstunnel dials https without --tls-verify-certificate (D16)
  wstunnelUnverified: boolean;
};

export function presentCommand(
  mode: ExitMode,
  command: { command: string } | undefined,
  server: Origin,
  local: Origin,
  fingerprint: string
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
  const pins = !!command && mode === 'native' && !cleartext && !!fingerprint;

  return {
    text,
    scheme,
    rewritten,
    schemeMismatch,
    cleartext,
    pinsFingerprint: pins && !rewritten,
    pinUncertain: pins && rewritten,
    // the server adds the flag only behind a CA-signed certificate, so its
    // absence is the tell
    wstunnelUnverified:
      !!command && mode === 'wstunnel' && !cleartext && !command.command.includes('--tls-verify')
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

const ipv4 = /^(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)(\.(25[0-5]|2[0-4]\d|1\d\d|[1-9]?\d)){3}$/;
const ipv6 = /^[0-9a-fA-F:]+$/;

// the resolvers are dialled as literals through the exit (D3), and the server
// validates the same way (`dive,ip`), so a name is refused before it is sent
export function isValidDNS(value: string): boolean {
  const address = value.trim();
  if (!address) return false;
  if (ipv4.test(address)) return true;

  if (!ipv6.test(address) || !address.includes(':')) return false;
  const groups = address.split('::');
  if (groups.length > 2) return false;

  const hextets = address.split(':').filter((part) => part !== '');
  return hextets.every((part) => part.length <= 4) && hextets.length <= 8;
}
