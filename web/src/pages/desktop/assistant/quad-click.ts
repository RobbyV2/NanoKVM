import { useEffect } from 'react';

import { hasLocalOverlay, isRemoteScreen } from './interaction-guard.ts';
import { QuadClickMatcher } from './quad-click-matcher.ts';

// Four clicks within 2 s on the remote picture -> MCQ.
export function useQuadClick(enabled: boolean, onQuad: () => void) {
  useEffect(() => {
    if (!enabled) return;
    const matcher = new QuadClickMatcher();
    const reset = () => matcher.reset();
    const onPointerDown = (e: PointerEvent) => {
      if (!isRemoteScreen(e.target) || hasLocalOverlay(document)) reset();
    };
    const handler = (e: MouseEvent) => {
      const allowed = e.button === 0 && isRemoteScreen(e.target) && !hasLocalOverlay(document);
      if (matcher.click(allowed)) {
        onQuad();
      }
    };
    document.addEventListener('click', handler, true);
    document.addEventListener('pointerdown', onPointerDown, true);
    document.addEventListener('focusin', reset, true);
    window.addEventListener('blur', reset);
    return () => {
      document.removeEventListener('click', handler, true);
      document.removeEventListener('pointerdown', onPointerDown, true);
      document.removeEventListener('focusin', reset, true);
      window.removeEventListener('blur', reset);
    };
  }, [enabled, onQuad]);
}
