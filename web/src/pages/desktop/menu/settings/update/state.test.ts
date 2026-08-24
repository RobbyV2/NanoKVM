import assert from 'node:assert/strict';
import test from 'node:test';

import { rollbackWarning, updateStatus } from './state.ts';

test('a kernel that failed its trial is named', () => {
  assert.equal(rollbackWarning({ slot: 'good', rolledBack: '2.9.0' }), '2.9.0');
});

test('a device that never rolled back raises nothing', () => {
  assert.equal(rollbackWarning({ slot: 'good', installed: '2.9.0' }), '');
  assert.equal(rollbackWarning(null), '');
});

test('a kernel offer is decided against the confirmed kernel', () => {
  const version = { current: '2.8.0', latest: '2.9.0', latestKernel: '2.9.0' };

  assert.equal(updateStatus(version, { installed: '2.8.0' }), 'outdated');
  assert.equal(updateStatus(version, { installed: '2.9.0' }), 'latest');
  assert.equal(updateStatus(version, null), 'outdated');
});

test('an application offer ignores the kernel entirely', () => {
  assert.equal(
    updateStatus({ current: '2.8.0', latest: '2.9.0' }, { installed: '2.9.0' }),
    'outdated'
  );
  assert.equal(
    updateStatus({ current: '2.9.0', latest: '2.9.0' }, { installed: '2.8.0' }),
    'latest'
  );
  assert.equal(updateStatus({ current: '2.9.0' }, null), 'latest');
});
