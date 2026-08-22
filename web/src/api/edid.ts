import { http } from '@/lib/http.ts';

// a flash holds the i2c bus far longer than a normal request, so it gets its own timeout
const applyTimeout = 180 * 1000;

export type EdidSummary = {
  sha256: string;
  manufacturer: string;
  model: string;
  productCode: number;
  serial: number;
  week: number;
  year: number;
  version: string;
  preferredMode: string;
  pixelClockKhz: number;
  extensions: number;
  audio: boolean;
};

export type EdidPreflight = {
  chip: string;
  product: string;
  supported: boolean;
  requiresPowerCycle: boolean;
  toolAvailable: boolean;
  reason?: string;
};

export type EdidBackup = {
  id: string;
  sha256: string;
  appliedAt: string;
  size: number;
};

export type EdidStatus = {
  active?: EdidSummary;
  source?: string;
  appliedAt?: string;
  unverifiedSinceBoot: boolean;
  preflight: EdidPreflight;
  backups: EdidBackup[];
  factoryAvailable: boolean;
};

export type EdidProfile = {
  id: string;
  manufacturer: string;
  model: string;
  preferredMode: string;
  source: string;
};

export type EdidResult = {
  state: string;
  verified: boolean;
  retryable: boolean;
  requiresPowerCycle: boolean;
  message: string;
  summary?: EdidSummary;
};

// get the active edid, the preflight and the backup history
export function getEdid() {
  return http.get('/api/vm/edid');
}

// get the shipped profile library
export function getProfiles() {
  return http.get('/api/vm/edid/profiles');
}

// decode a blob without touching the chip
export function decode(data: string) {
  return http.post('/api/vm/edid/decode', { data });
}

// flash a shipped profile or an uploaded blob
export function apply(profile: string, data: string) {
  return http.post('/api/vm/edid/apply', { profile, data }, { timeout: applyTimeout });
}

// download the bytes of the last verified flash
export function downloadEdid() {
  return http
    .request({ method: 'get', url: '/api/vm/edid/download', responseType: 'blob' })
    .then((rsp) => save(rsp as unknown as Blob, 'nanokvm-edid.bin'));
}

// download one history entry
export function downloadBackup(id: string) {
  return http
    .request({ method: 'get', url: '/api/vm/edid/backup', params: { id }, responseType: 'blob' })
    .then((rsp) => save(rsp as unknown as Blob, `nanokvm-edid-${id}.bin`));
}

// read a file as the base64 the decode and apply endpoints take
export function toBase64(file: File) {
  return file.arrayBuffer().then((buffer) => {
    const bytes = new Uint8Array(buffer);

    let binary = '';
    for (let i = 0; i < bytes.length; i++) {
      binary += String.fromCharCode(bytes[i]);
    }

    return btoa(binary);
  });
}

// the download routes answer with raw bytes, or with the usual json envelope when they fail
function save(blob: Blob, name: string) {
  if (blob.type.includes('json')) {
    return blob.text().then((text) => {
      throw new Error(JSON.parse(text)?.msg || 'download failed');
    });
  }

  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = name;

  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);

  return Promise.resolve();
}
