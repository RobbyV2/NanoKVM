import type {
  FIFOAssignment,
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
