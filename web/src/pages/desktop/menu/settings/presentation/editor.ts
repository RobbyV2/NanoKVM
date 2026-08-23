import type { PresentationProfile } from '@/api/presentation.ts';

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
