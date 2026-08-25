import assert from 'node:assert/strict';
import test from 'node:test';

import type { PresentationProfile } from '@/api/presentation.ts';
import { cameraCap, cameraOptions, modeKey, withCameraCap } from './modes.ts';

const profile = (frames: Array<{ width: number; height: number; intervals?: number[] }>) =>
  ({
    schema_version: 1,
    name: 'current',
    built_in: false,
    device: {} as never,
    config: { bm_attributes: 224, max_power: 120, configuration: 'NanoKVM' },
    functions: [
      { kind: 'hid', instance: 'GS0' },
      { kind: 'uvc', instance: 'cam0', video: { formats: [{ codec: 'mjpeg', frames }] } }
    ]
  }) as unknown as PresentationProfile;

// The host takes the largest mode on offer, so the cap is the largest frame.
test('the cap is the largest mode the camera offers', () => {
  const p = profile([
    { width: 320, height: 240 },
    { width: 640, height: 480 },
    { width: 160, height: 120 }
  ]);
  assert.equal(cameraCap(p, 'cam0'), '640x480');
  assert.equal(cameraCap(p, 'cam1'), undefined);
});

// Capping is what makes the host ask for less; anything larger has to go.
test('capping drops every larger mode and keeps the rest with their intervals', () => {
  const p = profile([
    { width: 1280, height: 720, intervals: [333333] },
    { width: 640, height: 480, intervals: [333333, 666666] },
    { width: 320, height: 240, intervals: [333333, 666666] }
  ]);
  const capped = withCameraCap(p, 'cam0', '640x480');
  const frames = capped.functions[1].video!.formats![0].frames!;
  assert.deepEqual(
    frames.map((f) => modeKey(f)),
    ['640x480', '320x240']
  );
  assert.deepEqual(frames[0].intervals, [333333, 666666], 'existing intervals are preserved');
  assert.equal(cameraCap(capped, 'cam0'), '640x480');
});

// Choosing a mode the profile does not carry must not silently do nothing.
test('capping to a mode the profile lacks adds it', () => {
  const p = profile([{ width: 1280, height: 720, intervals: [333333, 666666] }]);
  const capped = withCameraCap(p, 'cam0', '320x240');
  const frames = capped.functions[1].video!.formats![0].frames!;
  assert.deepEqual(
    frames.map((f) => modeKey(f)),
    ['320x240']
  );
  assert.deepEqual(frames[0].intervals, [333333, 666666]);
});

// A profile set to 720p by hand keeps it as a choice rather than losing it.
test('options include a mode the profile carries outside the standard ladder', () => {
  const p = profile([{ width: 800, height: 600 }]);
  assert.ok(cameraOptions(p, 'cam0').some((mode) => modeKey(mode) === '800x600'));
});

// Editing one camera must not disturb the rest of the profile.
test('capping leaves other functions and unnamed fields alone', () => {
  const p = profile([{ width: 640, height: 480 }]);
  const capped = withCameraCap(p, 'cam0', '320x240');
  assert.deepEqual(capped.functions[0], { kind: 'hid', instance: 'GS0' });
  assert.equal(capped.name, 'current');
  assert.equal(withCameraCap(p, 'cam0', 'nonsense'), p, 'an unknown cap changes nothing');
});
