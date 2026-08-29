// A capture can die without anything reporting an error. The camera is
// unplugged and its track ends; the encoder worker takes a frame and never
// answers, so the encode loop keeps firing and nothing ever leaves it; the
// machine sleeps and the camera's clock stops. In every one of those the
// browser sits there believing it is streaming while the target host sees a
// frozen picture, which is the failure an operator cannot diagnose from either
// end. Delivered frames are the only honest signal - an error that is never
// raised proves nothing - so the watchdog counts frames rather than errors.

export const captureStallTimeoutMS = 3000;
export const captureWatchdogIntervalMS = 1000;
// A capture that ran this long before failing was working, not thrashing, so
// its next failure starts the backoff over rather than continuing it.
export const captureHealthyMS = 15000;
export const captureRestartDelaysMS = [500, 1000, 2000, 4000, 8000];

export type CaptureSample = {
  at: number;
  // Payloads the encoder has handed back since this capture opened.
  frames: number;
  // Whether the browser still considers the underlying track able to produce.
  live: boolean;
};

// The most recent sample in which a frame arrived. Measuring against this rather
// than against the previous sample means a capture running slower than the
// watchdog ticks is never mistaken for a dead one.
export function captureProgress(progress: CaptureSample, latest: CaptureSample): CaptureSample {
  return latest.frames !== progress.frames ? latest : progress;
}

export function captureStalled(
  progress: CaptureSample,
  latest: CaptureSample,
  timeoutMS = captureStallTimeoutMS
) {
  // A track the browser has already ended will never produce another frame, so
  // there is nothing left to wait out.
  if (!latest.live) return true;
  return latest.at - progress.at >= timeoutMS;
}

export function captureRestartDelay(restarts: number) {
  const index = Math.min(Math.max(restarts, 0), captureRestartDelaysMS.length - 1);
  return captureRestartDelaysMS[index];
}
