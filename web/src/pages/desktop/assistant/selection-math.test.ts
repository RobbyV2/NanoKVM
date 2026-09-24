import assert from 'node:assert/strict';
import test from 'node:test';

import { cropLockTransform, selectionToCrop } from './selection-math.ts';

const media = { left: 100, top: 0, width: 800, height: 600 };

test('selection maps to frame fractions', () => {
  assert.deepEqual(selectionToCrop({ left: 300, top: 150, width: 400, height: 300 }, media), {
    x: 0.25,
    y: 0.25,
    w: 0.5,
    h: 0.5
  });
});

test('selection over the letterbox is clamped to the picture', () => {
  assert.deepEqual(selectionToCrop({ left: 0, top: 0, width: 300, height: 600 }, media), {
    x: 0,
    y: 0,
    w: 0.25,
    h: 1
  });
});

test('selection outside the picture is null', () => {
  assert.equal(selectionToCrop({ left: 0, top: 0, width: 90, height: 600 }, media), null);
  assert.equal(selectionToCrop({ left: 300, top: 150, width: 0, height: 0 }, media), null);
});

test('crop-lock scales the selection to fill the area', () => {
  const el = { left: 0, top: 0, width: 800, height: 600 };
  assert.equal(
    cropLockTransform({ left: 0, top: 0, width: 400, height: 300 }, el, el, 1),
    'translate(0px, 0px) scale(2)'
  );
  assert.equal(
    cropLockTransform({ left: 400, top: 300, width: 400, height: 300 }, el, el, 1),
    'translate(-800px, -600px) scale(2)'
  );
});

test('crop-lock translate is divided by the layout scale', () => {
  const el = { left: 0, top: 0, width: 800, height: 600 };
  assert.equal(
    cropLockTransform({ left: 400, top: 300, width: 400, height: 300 }, el, el, 2),
    'translate(-400px, -300px) scale(2)'
  );
});
