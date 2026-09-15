import type { ExitPlatform } from './types.ts';

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
// readdresses every http(s):// and ws(s):// url to window.location.
export function rewriteCommand(command: string, server: Origin, local: Origin): string {
  if (!server.scheme || !server.host) return command;
  if (server.scheme === local.scheme && server.host === local.host) return command;

  const http = replaceUrl(
    command,
    `${server.scheme}://${server.host}`,
    `${local.scheme}://${local.host}`
  );

  return replaceUrl(
    http,
    `${wsScheme(server.scheme)}://${server.host}`,
    `${wsScheme(local.scheme)}://${local.host}`
  );
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
