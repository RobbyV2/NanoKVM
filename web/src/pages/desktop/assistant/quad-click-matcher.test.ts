import assert from 'node:assert/strict';
import test from 'node:test';

import { QuadClickMatcher } from './quad-click-matcher.ts';

test('four remote clicks within two seconds trigger once', () => {
  const matcher = new QuadClickMatcher();
  assert.deepEqual(
    [0, 300, 600, 900, 1200].map((now) => matcher.click(true, now)),
    [false, false, false, true, false]
  );
});

test('local UI clicks cannot trigger or complete a partial remote gesture', () => {
  const matcher = new QuadClickMatcher();
  for (let now = 0; now < 3; now++) assert.equal(matcher.click(true, now), false);
  for (let now = 3; now < 7; now++) assert.equal(matcher.click(false, now), false);
  assert.equal(matcher.click(true, 7), false);
});

test('moving focus to local UI or losing window focus discards a partial gesture', () => {
  const matcher = new QuadClickMatcher();
  for (let now = 0; now < 3; now++) matcher.click(true, now);
  matcher.reset();
  assert.equal(matcher.click(true, 3), false);
});

test('clicks two seconds apart cannot complete the gesture', () => {
  const matcher = new QuadClickMatcher();
  matcher.click(true, 0);
  matcher.click(true, 500);
  matcher.click(true, 1000);
  assert.equal(matcher.click(true, 2000), false);
  assert.equal(matcher.click(true, 2100), true);
});
