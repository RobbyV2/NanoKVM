import { http } from '@/lib/http.ts';

export type EndpointUse = { in: number; out: number };

export type FIFOAssignment = Record<string, number[]>;

export type DeviceIdentity = {
  vendor_id: string;
  product_id: string;
  bcd_device?: string;
  serial?: string;
  manufacturer: string;
  product: string;
};

export type RecoveryAction = 'power-cycle' | 'host-reboot' | 'hdmi-reset' | 'usb-reconnect';

export type PresentationOutcome = {
  profile: string;
  linked: string[];
  removes: string[];
  hid: boolean;
  recovery: RecoveryAction;
};

export type UDCStatus = {
  name: string;
  bound: boolean;
  state?: string;
  speed?: string;
};

export type ApplyFailure = {
  profile: string;
  message: string;
  at: string;
};

export type USBDevice = {
  vendor_id: string;
  product_id: string;
  bcd_usb?: string;
  bcd_device?: string;
  class?: number;
  subclass?: number;
  protocol?: number;
  serial?: string;
  manufacturer: string;
  product: string;
};

export type USBFunction = {
  kind: string;
  instance: string;
  hid?: {
    protocol: number;
    subclass: number;
    report_length: number;
    wakeup_on_write: boolean;
    report_desc: string;
  };
  net?: Record<string, unknown>;
  storage?: Record<string, unknown>;
};

export type DescriptorSet = {
  device?: string;
  configurations?: string[];
  bos?: string;
  strings?: Record<string, string>;
  hid_reports?: Record<string, string>;
};

export type PresentationProfile = {
  schema_version: number;
  name: string;
  built_in: boolean;
  device: USBDevice;
  config: {
    bm_attributes: number;
    max_power: number;
    configuration: string;
  };
  functions: USBFunction[];
  os_desc?: { b_vendor_code: string; qw_sign: string };
  descriptors?: DescriptorSet;
};

export type ProfileSummary = {
  name: string;
  built_in: boolean;
  active: boolean;
  manufacturer: string;
  product: string;
  functions: string[];
  has_descriptors: boolean;
};

export type PresentationPreview = {
  valid: boolean;
  errors: string[];
  warnings: string[];
  profile: string;
  functions: string[];
  endpoints: EndpointUse;
  headroom: EndpointUse;
  fifos?: FIFOAssignment;
  operations: number;
  device: DeviceIdentity;
  apply?: PresentationOutcome;
  rollback?: PresentationOutcome;
};

export type PresentationStatus = {
  snapshot: {
    active: string;
    mode: string;
    linked: string[];
    udc: UDCStatus;
    pending_power_cycle: boolean;
    last_error?: ApplyFailure;
    endpoints: EndpointUse;
    headroom: EndpointUse;
    fifos?: FIFOAssignment;
  };
  profile?: ProfileSummary;
  last_known_good: string;
  rollback_target: string;
};

export function getStatus() {
  return http.get('/api/presentation/status');
}

export function getProfiles() {
  return http.get('/api/presentation/profiles');
}

export function getProfile(name: string) {
  return http.get(`/api/presentation/profiles/${encodeURIComponent(name)}`);
}

export function createProfile(profile: PresentationProfile) {
  return http.post('/api/presentation/profiles', profile);
}

export function updateProfile(profile: PresentationProfile) {
  return http.request({
    method: 'put',
    url: `/api/presentation/profiles/${encodeURIComponent(profile.name)}`,
    data: profile
  });
}

export function cloneProfile(name: string, clone: string) {
  return http.post(`/api/presentation/profiles/${encodeURIComponent(name)}/clone`, {
    name: clone
  });
}

export function deleteProfile(name: string) {
  return http.delete(`/api/presentation/profiles/${encodeURIComponent(name)}`);
}

export function validateProfile(name: string) {
  return http.post(`/api/presentation/profiles/${encodeURIComponent(name)}/validate`);
}

export function previewProfile(profile: PresentationProfile) {
  return http.request({ method: 'put', url: '/api/presentation/config/preview', data: profile });
}

export function applyProfile(name: string) {
  return http.request({ method: 'put', url: '/api/presentation/config/apply', data: { name } });
}

export function rollbackProfile() {
  return http.post('/api/presentation/rollback');
}

export function importProfile(file: File) {
  const data = new FormData();
  data.append('package', file);
  return http.post('/api/presentation/profiles/import', data);
}

export function exportProfile(name: string) {
  return http
    .request({
      method: 'get',
      url: `/api/presentation/profiles/${encodeURIComponent(name)}/export`,
      responseType: 'blob'
    })
    .then((rsp) => save(rsp as unknown as Blob, `${name}.nanokvm-profile.zip`));
}

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
  link.remove();
  URL.revokeObjectURL(url);
}
