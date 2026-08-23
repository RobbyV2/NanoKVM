import assert from 'node:assert/strict';
import test from 'node:test';

import type { EdidPending, EdidResult } from '../../../../../api/edid.ts';
import { pendingNotice, powerCycleNotice } from './utils.ts';

const result: EdidResult = {
  state: 'success',
  verified: true,
  retryable: false,
  requiresPowerCycle: true,
  message: ''
};

test('a verified flash on hardware that reloads out of reset asks for the power cycle', () => {
  assert.equal(powerCycleNotice(result), 'powerCycleNotice');
});

test('a half written region asks for the power cycle too, with its own wording', () => {
  const mismatch: EdidResult = {
    ...result,
    state: 'needs_recovery',
    verified: false,
    message: 'EDID data mismatch after write/read cycle'
  };

  assert.equal(powerCycleNotice(mismatch), 'powerCycleUnverified');
});

test('pcie resets over gpio, so nothing is asked of the operator', () => {
  assert.equal(powerCycleNotice({ ...result, requiresPowerCycle: false }), '');
  assert.equal(powerCycleNotice({ ...result, verified: false, requiresPowerCycle: false }), '');
  assert.equal(powerCycleNotice(undefined), '');
});

const pending: EdidPending = {
  sha256: 'd0f1',
  source: 'profile',
  state: 'success',
  appliedAt: '2026-08-22T10:00:00Z'
};

test('a flash still waiting for its power cycle asks for it again after a reload', () => {
  assert.equal(pendingNotice(pending), 'powerCycleNotice');
});

test('a pending flash that never verified keeps the unverified wording', () => {
  assert.equal(pendingNotice({ ...pending, state: 'needs_recovery' }), 'powerCycleUnverified');
});

test('nothing pending asks nothing of the operator', () => {
  assert.equal(pendingNotice(undefined), '');
  assert.equal(pendingNotice(null), '');
});
