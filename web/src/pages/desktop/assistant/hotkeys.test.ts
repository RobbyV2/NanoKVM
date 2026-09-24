import assert from 'node:assert/strict';
import test from 'node:test';

import { HotkeyMatcher } from './hotkeys.ts';

function matcher() {
  let now = 0;
  const m = new HotkeyMatcher(() => now);
  return { m, advance: (ms: number) => (now += ms) };
}
const down = (code: string, repeat = false) => ({ code, repeat });

test('two Ctrl presses within 500 ms arm the action key', () => {
  const { m, advance } = matcher();
  m.keydown(down('ControlLeft'));
  advance(300);
  m.keydown(down('ControlRight'));
  assert.deepEqual(m.keydown(down('KeyA')), { action: 'KeyA', swallow: true });
  assert.deepEqual(m.keydown(down('KeyA')), { action: null, swallow: true });
});

test('Ctrl presses 500 ms apart do not arm', () => {
  const { m, advance } = matcher();
  m.keydown(down('ControlLeft'));
  advance(500);
  m.keydown(down('ControlLeft'));
  assert.deepEqual(m.keydown(down('KeyA')), { action: null, swallow: false });
});

test('an auto-repeated Ctrl does not count', () => {
  const { m } = matcher();
  m.keydown(down('ControlLeft'));
  m.keydown(down('ControlLeft', true));
  assert.equal(m.keydown(down('KeyA')).action, null);
});

test('held key repeats and keyup are swallowed', () => {
  const { m } = matcher();
  m.keydown(down('ControlLeft'));
  m.keydown(down('ControlLeft'));
  m.keydown(down('KeyC'));
  assert.equal(m.keydown(down('KeyC', true)).swallow, true);
  assert.equal(m.keyup({ code: 'KeyC' }), true);
  assert.equal(m.keyup({ code: 'KeyC' }), false);
  assert.deepEqual(m.keydown(down('KeyC')), { action: null, swallow: false });
});

test('a non-action key leaves the prefix armed (content.js)', () => {
  const { m } = matcher();
  m.keydown(down('ControlLeft'));
  m.keydown(down('ControlLeft'));
  assert.equal(m.keydown(down('KeyB')).swallow, false);
  assert.equal(m.keydown(down('ArrowUp')).action, 'ArrowUp');
});

test('Escape and D/Q are not action keys', () => {
  for (const code of ['Escape', 'KeyD', 'KeyQ']) {
    const { m } = matcher();
    m.keydown(down('ControlLeft'));
    m.keydown(down('ControlLeft'));
    assert.equal(m.keydown(down(code)).action, null);
  }
});
