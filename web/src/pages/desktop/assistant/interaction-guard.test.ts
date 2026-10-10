import assert from 'node:assert/strict';
import test from 'node:test';

import { allowAssistantKeyboard, hasLocalOverlay, isRemoteScreen } from './interaction-guard.ts';

const screen = { closest: (selector: string) => (selector === '#screen' ? screen : null) };
const localControl = { closest: () => null };
function page() {
  const body = {};
  return {
    body,
    documentElement: {},
    activeElement: body as unknown,
    querySelectorAll: (): { getClientRects: () => { length: number } }[] => []
  };
}

test('assistant gestures recognize only the remote screen, not local controls or portals', () => {
  assert.equal(isRemoteScreen(screen), true);
  for (const target of [localControl, null, {}, { closest: () => null }]) {
    assert.equal(isRemoteScreen(target), false);
  }
});

test('desktop keyboard shortcuts work with body or screen focus', () => {
  const doc = page();
  assert.equal(allowAssistantKeyboard(doc, doc.body, false), true);
  doc.activeElement = screen;
  assert.equal(allowAssistantKeyboard(doc, screen, false), true);
});

test('local UI keyboard events and focus never trigger assistant shortcuts', () => {
  const doc = page();
  assert.equal(allowAssistantKeyboard(doc, localControl, false), false);
  doc.activeElement = localControl;
  assert.equal(allowAssistantKeyboard(doc, doc.body, false), false);
  assert.equal(allowAssistantKeyboard(doc, screen, false), false);
});

test('clicking non-focusable UI suspends keyboard shortcuts until returning to the screen', () => {
  const doc = page();
  assert.equal(allowAssistantKeyboard(doc, doc.body, true), false);
  assert.equal(allowAssistantKeyboard(doc, doc.body, false), true);
});

test('visible local overlays suspend shortcuts even when keyboard focus remains on the desktop', () => {
  const doc = page();
  doc.querySelectorAll = () => [{ getClientRects: () => ({ length: 1 }) }];
  assert.equal(hasLocalOverlay(doc), true);
  assert.equal(allowAssistantKeyboard(doc, doc.body, false), false);
});

test('retained hidden overlay DOM does not disable desktop shortcuts', () => {
  const doc = page();
  doc.querySelectorAll = () => [{ getClientRects: () => ({ length: 0 }) }];
  assert.equal(hasLocalOverlay(doc), false);
  assert.equal(allowAssistantKeyboard(doc, screen, false), true);
});
