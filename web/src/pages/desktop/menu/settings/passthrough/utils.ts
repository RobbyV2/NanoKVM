import type { PassthroughDevice } from '@/api/passthrough.ts';

// the port the server's dialer fills in when the exporter carries none
export const exporterPort = 3240;

// the loopback end of the reverse tunnel, which is the only address usbipd
// should be reachable on
export const defaultExporter = `127.0.0.1:${exporterPort}`;

// what `usbip list -l` prints for a device on the first bus
export const busIdExample = '1-2';

// the usbip busid grammar: a bus, a port, then a hub path
const busIdPattern = /^\d+-\d+(\.\d+)*$/;

// a hostname, an ipv4 literal or a bracketed ipv6 one, each with an optional port
const exporterPattern = /^(\[[0-9a-fA-F:.]+]|[0-9A-Za-z.-]+)(:\d{1,5})?$/;

// audio and video devices stream over isochronous endpoints
const isochronousClasses = [0x01, 0x0e];

export type Commands = {
  modprobe: string;
  list: string;
  bind: string;
  serve: string;
  tunnel: string;
  exporter: string;
  unbind: string;
};

export function isValidExporter(value: string): boolean {
  const exporter = value.trim();
  if (!exporter || exporter.length > 253) return false;

  const matched = exporterPattern.exec(exporter);
  if (!matched) return false;

  const port = matched[2] ? Number(matched[2].slice(1)) : exporterPort;
  return port > 0 && port <= 65535;
}

export function isValidBusId(value: string): boolean {
  const busId = value.trim();
  return busId.length > 0 && busId.length <= 31 && busIdPattern.test(busId);
}

// the port the exporter names, or the one the server would have assumed
export function exporterPortOf(exporter: string): number {
  const matched = exporterPattern.exec(exporter.trim());
  if (!matched || !matched[2]) return exporterPort;

  const port = Number(matched[2].slice(1));
  return port > 0 && port <= 65535 ? port : exporterPort;
}

// whatever this page was loaded from is the address the tunnel has to reach
export function deviceHost(): string {
  return window.location.hostname || 'nanokvm.local';
}

// vendor:product, the pair `usbip list -l` prints beside the busid
export function deviceId(device?: PassthroughDevice | null): string {
  if (!device?.idVendor || !device?.idProduct) return '';
  return `${device.idVendor}:${device.idProduct}`;
}

export function isIsochronous(device?: PassthroughDevice | null): boolean {
  return !!device && isochronousClasses.includes(device.class);
}

// Go's zero time reaches the browser as year 1, which is not a start time
export function formatTime(value?: string): string {
  if (!value) return '';

  const time = new Date(value);
  if (Number.isNaN(time.getTime()) || time.getUTCFullYear() < 2000) return '';

  return time.toLocaleString();
}

// The commands are stock usbip, run on the machine that owns the device: there
// is no client agent to install, by design. The port comes from the exporter
// field so the tunnel and the daemon always name the same one.
export function buildCommands(exporter: string, busId: string, host: string): Commands {
  const port = exporterPortOf(exporter);
  const id = busId.trim() || busIdExample;

  return {
    modprobe: 'sudo modprobe usbip-host',
    list: 'usbip list -l',
    bind: `sudo usbip bind -b ${id}`,
    serve: `sudo usbipd -4 --tcp-port ${port}`,
    tunnel: `ssh -N -R ${port}:127.0.0.1:${port} root@${host}`,
    exporter: `127.0.0.1:${port}`,
    unbind: `sudo usbip unbind -b ${id}`
  };
}

// the page is served over plain http on most deployments, where the async
// clipboard is unavailable, so the textarea path is the one that usually runs
function legacyCopy(text: string): Promise<void> {
  const textArea = document.createElement('textarea');
  textArea.value = text;
  textArea.setAttribute('readonly', '');
  textArea.style.position = 'fixed';
  textArea.style.left = '-9999px';
  textArea.style.top = '0';

  document.body.appendChild(textArea);
  textArea.focus();
  textArea.select();
  textArea.setSelectionRange(0, text.length);

  const copied = document.execCommand('copy');
  document.body.removeChild(textArea);

  return copied ? Promise.resolve() : Promise.reject(new Error('copy command was rejected'));
}

export function copyText(text: string): Promise<void> {
  if (window.isSecureContext === true && navigator.clipboard?.writeText) {
    return navigator.clipboard.writeText(text).catch(() => legacyCopy(text));
  }

  return legacyCopy(text);
}
