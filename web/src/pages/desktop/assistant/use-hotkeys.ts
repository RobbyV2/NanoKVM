import { useEffect } from 'react';

import type { AssistantActions } from './actions.ts';
import { HotkeyMatcher } from './hotkeys.ts';

// Window capture phase, so it runs before the desktop keyboard's document
// listener; a matched key never reaches the target. Ctrl itself still does.
export function useHotkeys(enabled: boolean, actions: AssistantActions) {
  useEffect(() => {
    if (!enabled) return;
    let matcher = new HotkeyMatcher();
    const onKeyDown = (e: KeyboardEvent) => {
      const result = matcher.keydown(e);
      if (!result.swallow) return;
      e.preventDefault();
      e.stopImmediatePropagation();
      if (result.action) actions.run(result.action);
    };
    const onKeyUp = (e: KeyboardEvent) => {
      if (!matcher.keyup(e)) return;
      e.preventDefault();
      e.stopImmediatePropagation();
    };
    // A keyup lost to a focus change must not swallow the next ordinary press.
    const onBlur = () => {
      matcher = new HotkeyMatcher();
    };
    window.addEventListener('keydown', onKeyDown, true);
    window.addEventListener('keyup', onKeyUp, true);
    window.addEventListener('blur', onBlur);
    return () => {
      window.removeEventListener('keydown', onKeyDown, true);
      window.removeEventListener('keyup', onKeyUp, true);
      window.removeEventListener('blur', onBlur);
    };
  }, [enabled, actions]);
}
