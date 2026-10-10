import { useEffect, useRef } from 'react';
import { useAtomValue } from 'jotai';

import { getOperatingSystem } from '@/lib/browser.ts';
import { KeyboardReport } from '@/lib/keyboard.ts';
import { isModifier } from '@/lib/keymap';
import { client, MessageEvent } from '@/lib/websocket.ts';
import { isKeyboardEnableAtom } from '@/jotai/keyboard.ts';
import { picoclawTakeoverStateAtom } from '@/jotai/picoclaw.ts';

import { isCompositionKey, releasedModifiers } from './input-state.ts';
import { Recorder } from './recorder.tsx';
import { useAltGr } from './useAltGr.ts';
import { useLeaderKey } from './useLeaderKey.ts';
import { normalizeKeyCode } from './utils.ts';

export const Keyboard = () => {
  const os = getOperatingSystem();

  const isKeyboardEnabled = useAtomValue(isKeyboardEnableAtom);
  const picoclawTakeoverState = useAtomValue(picoclawTakeoverStateAtom);

  const keyboardRef = useRef(new KeyboardReport());
  const pressedKeys = useRef(new Set<string>());
  const isComposing = useRef(false);

  // Send key event helper
  const sendKeyEvent = (type: 'keydown' | 'keyup', code: string) => {
    const kb = keyboardRef.current;
    const report = type === 'keydown' ? kb.keyDown(code) : kb.keyUp(code);
    sendReport(report);
  };

  // Init leader key handler
  const leaderKey = useLeaderKey(pressedKeys, sendKeyEvent);
  const leaderKeyRef = useRef(leaderKey);
  leaderKeyRef.current = leaderKey;

  // Init AltGr key handler
  const altGr = useAltGr(os, pressedKeys, sendKeyEvent);
  const altGrRef = useRef(altGr);
  altGrRef.current = altGr;

  useEffect(() => {
    if (!picoclawTakeoverState.active) {
      return;
    }

    pressedKeys.current.clear();
    leaderKeyRef.current.reset();
    altGrRef.current.reset();
    sendReport(keyboardRef.current.reset());
  }, [picoclawTakeoverState.active]);

  useEffect(() => {
    document.addEventListener('keydown', handleKeyDown);
    document.addEventListener('keyup', handleKeyUp);
    document.addEventListener('compositionstart', handleCompositionStart);
    document.addEventListener('compositionend', handleCompositionEnd);
    window.addEventListener('blur', handleBlur);
    window.addEventListener('focus', handleFocus);
    document.addEventListener('visibilitychange', handleVisibilityChange);

    function handleKeyDown(event: KeyboardEvent) {
      if (!isKeyboardEnabled) return;
      if (isCompositionKey(isComposing.current, event)) return;
      isComposing.current = false;

      event.preventDefault();
      event.stopPropagation();

      const code = normalizeKeyCode(event, os);
      if (!code) return;

      releaseStaleModifiers(event, code);

      if (pressedKeys.current.has(code)) {
        if (event.repeat) return;
        // A fresh press of a key still recorded as held means its keyup was
        // lost. Release it so the host sees a new keystroke.
        pressedKeys.current.delete(code);
        sendKeyEvent('keyup', code);
      }

      // Handle leader key
      const leaderHandled = leaderKeyRef.current.handleKeyDown(code);
      if (leaderHandled) {
        return;
      }

      // Handle AltGr key
      altGrRef.current.handleKeyDown(code, event.timeStamp);

      pressedKeys.current.add(code);
      sendKeyEvent('keydown', code);
    }

    function handleKeyUp(event: KeyboardEvent) {
      if (!isKeyboardEnabled) return;
      if (isComposing.current || event.isComposing) return;

      event.preventDefault();
      event.stopPropagation();

      const code = normalizeKeyCode(event, os);
      if (!code) return;

      releaseStaleModifiers(event, code);

      // Handle leader key release
      const leaderHandled = leaderKeyRef.current.handleKeyUp(code);
      if (leaderHandled) {
        return;
      }

      // Handle AltGr key release
      const altGrHandled = altGrRef.current.handleKeyUp(code);
      if (altGrHandled) {
        return;
      }

      // Release Meta combinations (macOS compatibility)
      if (code === 'MetaLeft' || code === 'MetaRight') {
        releaseMetaCombinations();
      }

      pressedKeys.current.delete(code);
      sendKeyEvent('keyup', code);
    }

    function handleCompositionStart() {
      isComposing.current = true;
    }

    function handleCompositionEnd() {
      isComposing.current = false;
    }

    // Leaving the page can interrupt a composition before compositionend is
    // delivered, so focus changes end it as well as releasing held keys.
    function handleBlur() {
      isComposing.current = false;
      releaseKeys();
    }

    function handleFocus() {
      isComposing.current = false;
      releaseKeys();
    }

    function handleVisibilityChange() {
      isComposing.current = false;
      if (document.hidden) {
        releaseKeys();
      }
    }

    function releaseStaleModifiers(event: KeyboardEvent, code: string) {
      for (const modifier of releasedModifiers(pressedKeys.current, event, code)) {
        pressedKeys.current.delete(modifier);
        sendKeyEvent('keyup', modifier);
      }
    }

    function releaseMetaCombinations() {
      const keysToRelease = Array.from(pressedKeys.current).filter((code) => !isModifier(code));
      for (const key of keysToRelease) {
        sendKeyEvent('keyup', key);
        pressedKeys.current.delete(key);
      }
    }

    function releaseKeys() {
      pressedKeys.current.forEach((code) => {
        sendKeyEvent('keyup', code);
      });
      pressedKeys.current.clear();

      leaderKeyRef.current.reset();
      altGrRef.current.reset();

      const report = keyboardRef.current.reset();
      sendReport(report);
    }

    return () => {
      document.removeEventListener('keydown', handleKeyDown);
      document.removeEventListener('keyup', handleKeyUp);
      document.removeEventListener('compositionstart', handleCompositionStart);
      document.removeEventListener('compositionend', handleCompositionEnd);
      window.removeEventListener('blur', handleBlur);
      window.removeEventListener('focus', handleFocus);
      document.removeEventListener('visibilitychange', handleVisibilityChange);

      releaseKeys();
    };
  }, [isKeyboardEnabled]);

  function sendReport(report: Uint8Array) {
    const data = new Uint8Array([MessageEvent.Keyboard, ...report]);
    client.send(data);
  }

  return (
    <>
      <Recorder recordMode={leaderKey.recordMode} recordedKeys={leaderKey.recordedKeys} />
    </>
  );
};
