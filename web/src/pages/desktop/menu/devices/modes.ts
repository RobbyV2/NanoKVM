import type { PresentationProfile, USBFunction } from '@/api/presentation.ts';

export type CameraMode = { width: number; height: number };

// Largest first. The host picks the biggest mode on offer, so capping the list
// is the only way to make it ask for less: there is no "preferred" flag a UVC
// host has to honour.
export const cameraModes: CameraMode[] = [
  { width: 1280, height: 720 },
  { width: 640, height: 480 },
  { width: 320, height: 240 },
  { width: 160, height: 120 }
];

export const modeKey = (mode: CameraMode) => `${mode.width}x${mode.height}`;

const pixels = (mode: CameraMode) => mode.width * mode.height;

function cameraFunction(profile: PresentationProfile, instance: string): USBFunction | undefined {
  return profile.functions.find((fn) => fn.kind === 'uvc' && fn.instance === instance);
}

function framesOf(profile: PresentationProfile, instance: string): CameraMode[] {
  const format = cameraFunction(profile, instance)?.video?.formats?.[0];
  return (format?.frames ?? []).map(({ width, height }) => ({ width, height }));
}

// What the camera currently offers as its largest mode, which is what the host
// will have picked. Undefined when the profile has no such camera.
export function cameraCap(profile: PresentationProfile, instance: string): string | undefined {
  const frames = framesOf(profile, instance);
  if (frames.length === 0) return undefined;
  const largest = frames.reduce((a, b) => (pixels(b) > pixels(a) ? b : a));
  return modeKey(largest);
}

// The modes worth offering: the standard ladder, plus anything the profile
// already carries. A profile that was set to 1280x720 by hand keeps that as a
// choice rather than having it silently disappear from the menu.
export function cameraOptions(profile: PresentationProfile, instance: string): CameraMode[] {
  const present = framesOf(profile, instance);
  const known = new Set(cameraModes.map(modeKey));
  const extra = present.filter((mode) => !known.has(modeKey(mode)));
  return [...cameraModes, ...extra].sort((a, b) => pixels(b) - pixels(a));
}

// A copy of the profile whose camera offers nothing larger than `cap`. Frames
// already in the profile keep their own intervals; a cap the profile does not
// carry is added at the rates every other mode uses, so choosing it is not
// silently ignored. Returns the profile untouched if there is no such camera or
// the cap is not a mode we know.
export function withCameraCap(
  profile: PresentationProfile,
  instance: string,
  cap: string
): PresentationProfile {
  const target = [...cameraModes, ...framesOf(profile, instance)].find(
    (mode) => modeKey(mode) === cap
  );
  const format = cameraFunction(profile, instance)?.video?.formats?.[0];
  if (!target || !format) return profile;

  const kept = (format.frames ?? []).filter((frame) => pixels(frame) <= pixels(target));
  const intervals = format.frames?.[0]?.intervals ?? [333333, 666666];
  const frames = kept.some((frame) => modeKey(frame) === cap)
    ? kept
    : [...kept, { width: target.width, height: target.height, intervals }];

  return {
    ...profile,
    functions: profile.functions.map((fn) =>
      fn.kind === 'uvc' && fn.instance === instance && fn.video
        ? {
            ...fn,
            video: {
              ...fn.video,
              formats: fn.video.formats?.map((entry, index) =>
                index === 0 ? { ...entry, frames } : entry
              )
            }
          }
        : fn
    )
  };
}
