import { useEffect } from 'react';

import type { AssistantActions } from './actions.ts';
import { HotkeyMatcher } from './hotkeys.ts';
import { allowAssistantKeyboard, isRemoteScreen } from './interaction-guard.ts';

// Window capture phase, so it runs before the desktop keyboard's document
// listener; a matched key never reaches the target. Ctrl itself still does.
export function useHotkeys(enabled: boolean, actions: AssistantActions) {
  useEffect(() => {
    if (!enabled) return;
    let matcher = new HotkeyMatcher();
    let localInteraction = false;
    const reset = () => {
      matcher = new HotkeyMatcher();
    };
    const onPointerDown = (e: PointerEvent) => {
      localInteraction = !isRemoteScreen(e.target);
      reset();
    };
    const onPointerOver = (e: PointerEvent) => {
      const next = !isRemoteScreen(e.target);
      if (next !== localInteraction) reset();
      localInteraction = next;
    };
    const allowed = (e: KeyboardEvent) => {
      if (allowAssistantKeyboard(document, e.target, localInteraction)) return true;
      reset();
      return false;
    };
    const onKeyDown = (e: KeyboardEvent) => {
      if (!allowed(e) || e.isComposing) return;
      const result = matcher.keydown(e);
      if (!result.swallow) return;
      e.preventDefault();
      e.stopImmediatePropagation();
      if (result.action) actions.run(result.action);
    };
    const onKeyUp = (e: KeyboardEvent) => {
      if (!allowed(e)) return;
      if (!matcher.keyup(e)) return;
      e.preventDefault();
      e.stopImmediatePropagation();
    };
    // A keyup lost to a focus change must not swallow the next ordinary press.
    const onBlur = reset;
    window.addEventListener('keydown', onKeyDown, true);
    window.addEventListener('keyup', onKeyUp, true);
    window.addEventListener('blur', onBlur);
    window.addEventListener('pointerdown', onPointerDown, true);
    window.addEventListener('pointerover', onPointerOver, true);
    window.addEventListener('focusin', reset, true);
    return () => {
      window.removeEventListener('keydown', onKeyDown, true);
      window.removeEventListener('keyup', onKeyUp, true);
      window.removeEventListener('blur', onBlur);
      window.removeEventListener('pointerdown', onPointerDown, true);
      window.removeEventListener('pointerover', onPointerOver, true);
      window.removeEventListener('focusin', reset, true);
    };
  }, [enabled, actions]);
}
