import { atom } from 'jotai';

import { cropLockTransform, type Rect } from './selection-math.ts';

export const cropLockAtom = atom(false);

// L: scale the selected region of #screen to fill the screen viewport. Mouse
// mapping keeps working because it reads #screen's rendered rect.
export function applyCropLock(sel: Rect): boolean {
  const screen = document.getElementById('screen');
  const area = document.getElementById('screen-viewport');
  if (!screen || !area || screen.offsetWidth === 0) return false;
  screen.style.transform = 'none';
  const rect = screen.getBoundingClientRect();
  const transform = cropLockTransform(
    sel,
    rect,
    area.getBoundingClientRect(),
    rect.width / screen.offsetWidth
  );
  if (!transform) return false;
  screen.style.transformOrigin = '0 0';
  screen.style.transform = transform;
  return true;
}

export function resetCropLock() {
  const screen = document.getElementById('screen');
  if (!screen) return;
  screen.style.transform = 'none';
  screen.style.transformOrigin = '';
}
