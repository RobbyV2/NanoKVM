import assert from 'node:assert/strict';
import test from 'node:test';

import { isCompositionKey, releasedModifiers } from './input-state.ts';

function modifiers(held: { ctrl?: boolean; shift?: boolean; alt?: boolean; meta?: boolean; altGraph?: boolean }) {
  return {
    ctrlKey: held.ctrl ?? false,
    shiftKey: held.shift ?? false,
    altKey: held.alt ?? false,
    metaKey: held.meta ?? false,
    getModifierState: (key: string) => key === 'AltGraph' && (held.altGraph ?? false)
  };
}

test('modifiers whose keyup was lost are released by the next key event', () => {
  const pressed = ['AltLeft', 'ControlLeft', 'ShiftRight'];
  assert.deepEqual(releasedModifiers(pressed, modifiers({ shift: true }), 'KeyA'), [
    'ControlLeft',
    'AltLeft'
  ]);
});

test('held modifiers and ordinary keys are kept', () => {
  const pressed = ['ControlLeft', 'MetaRight', 'KeyC'];
  assert.deepEqual(releasedModifiers(pressed, modifiers({ ctrl: true, meta: true }), 'KeyV'), []);
});

test('the event key family is not reconciled from its own event', () => {
  assert.deepEqual(releasedModifiers(['ControlLeft'], modifiers({}), 'ControlRight'), []);
  assert.deepEqual(releasedModifiers(['AltLeft'], modifiers({}), 'AltLeft'), []);
});

test('AltGraph keeps the right Alt key held', () => {
  assert.deepEqual(releasedModifiers(['AltRight'], modifiers({ altGraph: true }), 'KeyQ'), []);
  assert.deepEqual(releasedModifiers(['AltRight'], modifiers({}), 'KeyQ'), ['AltRight']);
});

test('composition suppresses only input-method keys', () => {
  assert.equal(isCompositionKey(false, { isComposing: true, keyCode: 65, key: 'a' }), true);
  assert.equal(isCompositionKey(true, { isComposing: false, keyCode: 229, key: 'Process' }), true);
  assert.equal(isCompositionKey(false, { isComposing: false, keyCode: 229, key: 'Process' }), false);
});

test('a composition that never ended does not suppress later typing', () => {
  assert.equal(isCompositionKey(true, { isComposing: false, keyCode: 65, key: 'a' }), false);
});
