import assert from 'node:assert/strict';
import test from 'node:test';

import {
  getMenuDisabledItems,
  getResolution,
  getSkipUpdate,
  setResolution
} from './localstorage.ts';

// Every one of these values is read while the desktop is mounting. A value that
// throws on the way in takes the whole route down, and because it is still in
// localStorage the next refresh takes it down in exactly the same place - the
// operator is locked out of their own KVM by a setting they cannot see or
// clear. Recovery has to start with the page refusing to trust what it stored.
function installStorage(entries: Record<string, string> = {}) {
  const values = new Map(Object.entries(entries));
  Object.defineProperty(globalThis, 'localStorage', {
    configurable: true,
    value: {
      getItem: (key: string) => (values.has(key) ? values.get(key)! : null),
      setItem: (key: string, value: string) => values.set(key, value),
      removeItem: (key: string) => values.delete(key)
    }
  });
  Object.defineProperty(globalThis, 'window', {
    configurable: true,
    value: {
      atob: (value: string) => Buffer.from(value, 'base64').toString('binary'),
      btoa: (value: string) => Buffer.from(value, 'binary').toString('base64')
    }
  });
  return values;
}

test('a stored resolution that cannot be decoded is discarded, not thrown', () => {
  const values = installStorage({ 'nano-kvm-web-resolution': 'not base64 at all' });
  assert.equal(getResolution(), null);
  assert.equal(values.has('nano-kvm-web-resolution'), false, 'want the bad value gone for good');
});

test('a stored resolution that decodes to the wrong shape is discarded', () => {
  const encoded = Buffer.from(JSON.stringify({ width: 'wide' }), 'utf8').toString('base64');
  const values = installStorage({ 'nano-kvm-web-resolution': encoded });
  assert.equal(getResolution(), null);
  assert.equal(values.has('nano-kvm-web-resolution'), false);
});

test('a readable resolution still round-trips', () => {
  installStorage();
  setResolution({ width: 1920, height: 1080 });
  assert.deepEqual(getResolution(), { width: 1920, height: 1080 });
});

test('a corrupt menu configuration falls back to every item enabled', () => {
  const values = installStorage({ 'nano-kvm-menu-disabled-items': '{"image":true}' });
  assert.deepEqual(getMenuDisabledItems(), []);
  assert.equal(values.has('nano-kvm-menu-disabled-items'), false);
});

test('a corrupt expiring value reads as unset', () => {
  const values = installStorage({ 'nano-kvm-check-update': 'true' });
  assert.equal(getSkipUpdate(), false);
  assert.equal(values.has('nano-kvm-check-update'), false);
});
