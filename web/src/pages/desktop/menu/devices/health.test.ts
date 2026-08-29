import assert from 'node:assert/strict';
import test from 'node:test';

import { captureProgress, captureRestartDelay, captureStalled } from './health.ts';

const sample = (at: number, frames: number, live = true) => ({ at, frames, live });

test('a capture that has delivered nothing for the timeout is stalled', () => {
  // The encoder worker that takes a frame and never answers leaves the encode
  // loop running and the socket silent. Nothing throws, nothing closes, and the
  // target host holds the last picture it was sent until someone notices.
  const progress = sample(0, 12);
  assert.equal(captureStalled(progress, sample(2999, 12)), false);
  assert.equal(captureStalled(progress, sample(3000, 12)), true);
});

test('a capture slower than the watchdog ticks is not mistaken for a dead one', () => {
  // Progress is measured from the last frame that arrived, not from the last
  // time we looked, so a camera producing one frame every two seconds keeps
  // resetting the clock instead of being restarted underneath the operator.
  let progress = sample(0, 1);
  for (let tick = 1; tick <= 10; tick++) {
    const latest = sample(tick * 2000, 1 + tick);
    progress = captureProgress(progress, latest);
    assert.equal(captureStalled(progress, latest), false, `restarted at tick ${tick}`);
  }
});

test('a track the browser has ended is stalled without waiting out the timeout', () => {
  // An unplugged camera will never produce another frame, so waiting three more
  // seconds only delays the reconnection.
  const progress = sample(0, 4);
  assert.equal(captureStalled(progress, sample(10, 4, false)), true);
});

test('restart backoff climbs and then holds at its longest delay', () => {
  // A camera that is genuinely gone must not be reopened in a tight loop, and a
  // camera that comes back must not wait minutes for the next attempt.
  assert.deepEqual(
    [0, 1, 2, 3, 4, 5, 40].map(captureRestartDelay),
    [500, 1000, 2000, 4000, 8000, 8000, 8000]
  );
});
