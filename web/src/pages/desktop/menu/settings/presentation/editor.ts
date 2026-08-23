import type {
  FIFOAssignment,
  HIDRole,
  PresentationProfile,
  Preset,
  ProfileSummary
} from '@/api/presentation.ts';

export type IdentityFields = {
  vendorId: string;
  productId: string;
  bcdUSB: string;
  bcdDevice: string;
  manufacturer: string;
  product: string;
  serial: string;
  configuration: string;
};

const profileNamePattern = /^[a-z0-9][a-z0-9._-]{0,63}$/;

export function isProfileName(name: string) {
  return profileNamePattern.test(name.trim());
}

export function identityFields(profile: PresentationProfile): IdentityFields {
  return {
    vendorId: profile.device.vendor_id,
    productId: profile.device.product_id,
    bcdUSB: profile.device.bcd_usb || '',
    bcdDevice: profile.device.bcd_device || '',
    manufacturer: profile.device.manufacturer,
    product: profile.device.product,
    serial: profile.device.serial || '',
    configuration: profile.config.configuration
  };
}

export function editIdentity(
  profile: PresentationProfile,
  fields: IdentityFields
): PresentationProfile {
  return {
    ...profile,
    device: {
      ...profile.device,
      vendor_id: fields.vendorId.trim(),
      product_id: fields.productId.trim(),
      bcd_usb: fields.bcdUSB.trim() || undefined,
      bcd_device: fields.bcdDevice.trim() || undefined,
      manufacturer: fields.manufacturer.trim(),
      product: fields.product.trim(),
      serial: fields.serial.trim() || undefined
    },
    config: { ...profile.config, configuration: fields.configuration.trim() }
  };
}

export function applyPreset(fields: IdentityFields, preset: Preset): IdentityFields {
  return {
    ...fields,
    vendorId: preset.vendor_id,
    productId: preset.product_id,
    manufacturer: preset.manufacturer,
    product: preset.product
  };
}

// Mirrors Preset.matches on the server, which is what demotes a profile's
// preset provenance once the four fields stop agreeing with the entry.
export function matchPreset(presets: Preset[], fields: IdentityFields) {
  return presets.find(
    (preset) =>
      preset.vendor_id.toLowerCase() === fields.vendorId.trim().toLowerCase() &&
      preset.product_id.toLowerCase() === fields.productId.trim().toLowerCase() &&
      preset.manufacturer === fields.manufacturer.trim() &&
      preset.product === fields.product.trim()
  );
}

export function descriptorCount(profile: PresentationProfile) {
  const set = profile.descriptors;
  if (!set) return 0;
  return (
    Number(!!set.device) +
    (set.configurations?.length || 0) +
    Number(!!set.bos) +
    Object.keys(set.strings || {}).length +
    Object.keys(set.hid_reports || {}).length
  );
}

export function identityChanged(profile: PresentationProfile, fields: IdentityFields) {
  const saved = identityFields(profile);
  return (Object.keys(saved) as (keyof IdentityFields)[]).some(
    (key) => saved[key] !== fields[key].trim()
  );
}

export function formatFIFOs(fifos?: FIFOAssignment) {
  return Object.entries(fifos || {})
    .sort(([a], [b]) => a.localeCompare(b))
    .map(([name, slots]) => `${name} ${slots.join(',')}`)
    .join('; ');
}

export function provenanceTags(provenance: ProfileSummary['provenance']) {
  const tags: string[] = [];
  if (provenance.descriptors) tags.push('settings.presentation.descriptors');
  if (provenance.imported) tags.push('settings.presentation.imported');
  return tags;
}

export function recoveryKey(action: string) {
  switch (action) {
    case 'power-cycle':
      return 'settings.presentation.recoveryPowerCycle';
    case 'host-reboot':
      return 'settings.presentation.recoveryReboot';
    case 'hdmi-reset':
      return 'settings.presentation.recoveryHdmiReset';
    default:
      return 'settings.presentation.recoveryReconnect';
  }
}

export const HID_ROLES: HIDRole[] = ['keyboard', 'relative', 'absolute'];

// Slot index per role, or null when the role is not present at all. Two roles
// sharing a slot share one HID interface and therefore one IN endpoint.
export type HIDAssignment = Record<HIDRole, number | null>;

export function hidAssignment(profile: PresentationProfile): HIDAssignment {
  const assignment: HIDAssignment = { keyboard: null, relative: null, absolute: null };
  let slot = 0;
  for (const item of profile.functions) {
    if (item.kind !== 'hid') continue;
    for (const role of item.hid?.roles || []) assignment[role] = slot;
    slot += 1;
  }
  return assignment;
}

export function hidGroups(assignment: HIDAssignment): HIDRole[][] {
  const slots = [...new Set(HID_ROLES.map((role) => assignment[role]).filter((s) => s !== null))];
  slots.sort((a, b) => a - b);
  return slots.map((slot) => HID_ROLES.filter((role) => assignment[role] === slot));
}

export function hidSlotCount(assignment: HIDAssignment) {
  return hidGroups(assignment).length;
}

// Renumbering after every edit keeps the slots contiguous, which is what makes
// them a prefix of hid.GS0,GS1,GS2 on the server.
export function setHidSlot(
  assignment: HIDAssignment,
  role: HIDRole,
  slot: number | null
): HIDAssignment {
  const next = { ...assignment, [role]: slot };
  const used = [...new Set(HID_ROLES.map((item) => next[item]).filter((s) => s !== null))];
  used.sort((a, b) => a - b);
  for (const item of HID_ROLES) {
    if (next[item] !== null) next[item] = used.indexOf(next[item]);
  }
  return next;
}

export function hidRoleKey(role: HIDRole) {
  switch (role) {
    case 'keyboard':
      return 'settings.presentation.hidRoleKeyboard';
    case 'relative':
      return 'settings.presentation.hidRoleRelative';
    default:
      return 'settings.presentation.hidRoleAbsolute';
  }
}

export function hidKeyboardShares(assignment: HIDAssignment) {
  const slot = assignment.keyboard;
  if (slot === null) return false;
  return HID_ROLES.some((role) => role !== 'keyboard' && assignment[role] === slot);
}
