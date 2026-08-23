import assert from 'node:assert/strict';
import test from 'node:test';

import type { PresentationProfile } from '../../../../../api/presentation.ts';
import { descriptorCount, editIdentity, identityFields, isProfileName } from './editor.ts';

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

test('descriptor summary counts every stored asset', () => {
  assert.equal(descriptorCount(profile), 5);
  assert.equal(descriptorCount({ ...profile, descriptors: undefined }), 0);
});

test('profile names match the persisted filename contract', () => {
  assert.equal(isProfileName('desk-kvm.1'), true);
  assert.equal(isProfileName('../desk'), false);
  assert.equal(isProfileName('Desk'), false);
});
