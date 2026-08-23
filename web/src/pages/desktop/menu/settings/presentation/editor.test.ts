import assert from 'node:assert/strict';
import test from 'node:test';

import type { Preset, PresentationProfile } from '../../../../../api/presentation.ts';
import {
  applyPreset,
  descriptorCount,
  editIdentity,
  formatFIFOs,
  identityChanged,
  identityFields,
  isProfileName,
  matchPreset,
  recoveryKey
} from './editor.ts';

const profile: PresentationProfile = {
  schema_version: 1,
  name: 'desk',
  built_in: false,
  device: {
    vendor_id: '0x3346',
    product_id: '0x1009',
    bcd_device: '0x0510',
    manufacturer: 'sipeed',
    product: 'NanoKVM',
    serial: '0123'
  },
  config: { bm_attributes: 224, max_power: 120, configuration: 'NanoKVM' },
  functions: [
    {
      kind: 'hid',
      instance: 'GS0',
      hid: {
        protocol: 1,
        subclass: 0,
        report_length: 8,
        wakeup_on_write: true,
        report_desc: 'AA=='
      }
    }
  ],
  descriptors: {
    device: 'AA==',
    configurations: ['AA=='],
    bos: 'AA==',
    strings: { '1': 'sipeed' },
    hid_reports: { GS0: 'AA==' }
  }
};

test('safe identity edits preserve functions and descriptor assets', () => {
  const fields = identityFields(profile);
  fields.manufacturer = ' RobbyV2 ';
  fields.product = ' Desk KVM ';
  fields.serial = '';
  const edited = editIdentity(profile, fields);

  assert.equal(edited.device.manufacturer, 'RobbyV2');
  assert.equal(edited.device.product, 'Desk KVM');
  assert.equal(edited.device.serial, undefined);
  assert.strictEqual(edited.functions, profile.functions);
  assert.strictEqual(edited.descriptors, profile.descriptors);
});

const presets: Preset[] = [
  {
    id: 'logitech-unifying-receiver',
    vendor_id: '0x046d',
    product_id: '0xc52b',
    manufacturer: 'Logitech, Inc.',
    product: 'Unifying Receiver',
    source: 'usb.ids 046d:c52b'
  }
];

test('a preset fills the identity it can state and leaves the rest alone', () => {
  const filled = applyPreset(identityFields(profile), presets[0]);

  assert.equal(filled.vendorId, '0x046d');
  assert.equal(filled.productId, '0xc52b');
  assert.equal(filled.manufacturer, 'Logitech, Inc.');
  assert.equal(filled.product, 'Unifying Receiver');
  assert.equal(filled.serial, '0123');
  assert.equal(filled.bcdDevice, '0x0510');
  assert.equal(filled.configuration, 'NanoKVM');
});

test('a preset stays selected only while all four fields still agree', () => {
  const filled = applyPreset(identityFields(profile), presets[0]);
  assert.equal(matchPreset(presets, filled)?.id, 'logitech-unifying-receiver');
  assert.equal(matchPreset(presets, { ...filled, vendorId: '0x046D' })?.id, presets[0].id);
  assert.equal(matchPreset(presets, { ...filled, product: 'Desk KVM' }), undefined);
  assert.equal(matchPreset(presets, identityFields(profile)), undefined);
});

test('descriptor summary counts every stored asset', () => {
  assert.equal(descriptorCount(profile), 5);
  assert.equal(descriptorCount({ ...profile, descriptors: undefined }), 0);
});

test('profile names match the persisted filename contract', () => {
  assert.equal(isProfileName('desk-kvm.1'), true);
  assert.equal(isProfileName('../desk'), false);
  assert.equal(isProfileName('Desk'), false);
});

test('pending edits are the trimmed difference from the saved profile', () => {
  const fields = identityFields(profile);
  assert.equal(identityChanged(profile, fields), false);
  assert.equal(identityChanged(profile, { ...fields, product: ' NanoKVM ' }), false);
  assert.equal(identityChanged(profile, { ...fields, product: 'Desk KVM' }), true);
  assert.equal(identityChanged(profile, { ...fields, serial: '' }), true);
});

test('fifo assignment reads as one line per function', () => {
  assert.equal(formatFIFOs({ 'ncm.usb0': [1, 2], 'hid.GS0': [0] }), 'hid.GS0 0; ncm.usb0 1,2');
  assert.equal(formatFIFOs(undefined), '');
});

test('every recovery action names its own sentence', () => {
  const keys = ['power-cycle', 'host-reboot', 'hdmi-reset', 'usb-reconnect'].map(recoveryKey);
  assert.equal(new Set(keys).size, 4);
  assert.equal(recoveryKey('power-cycle'), 'settings.presentation.recoveryPowerCycle');
});
