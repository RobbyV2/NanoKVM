import assert from 'node:assert/strict';
import test from 'node:test';

import type { EdidResult } from '../../../../../api/edid.ts';
import { powerCycleNotice } from './utils.ts';

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
