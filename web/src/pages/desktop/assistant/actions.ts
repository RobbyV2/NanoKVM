import { useMemo } from 'react';
import { useStore } from 'jotai';

import * as api from '@/api/assistant.ts';
import type { AskKind, Crop } from '@/api/assistant.ts';
import {
  assistantBusyAtom,
  assistantConfigAtom,
  assistantContextCountAtom
} from '@/jotai/assistant.ts';

import { getMediaSize, getRenderedMediaRect } from '../screen/geometry.ts';
import { applyCropLock, cropLockAtom, resetCropLock } from './crop-lock.ts';
import { freezeAtom } from './freeze.tsx';
import { frqContentAtom, frqVisibleAtom } from './frq-box.tsx';
import type { HotkeyCode } from './hotkeys.ts';
import { selectionToCrop, type Rect } from './selection-math.ts';
import { setStatus, STATUS } from './status.tsx';
import { showToast } from './toast.tsx';

function renderedMediaRect(): Rect | null {
  const screen = document.getElementById('screen');
  const media = screen && getMediaSize(screen);
  return screen && media ? getRenderedMediaRect(screen.getBoundingClientRect(), media) : null;
}

export function useAssistantActions(requestSelection: () => Promise<Rect | null>) {
  const store = useStore();

  return useMemo(() => {
    const config = () => store.get(assistantConfigAtom)!;
    const status = (color: string) => setStatus(store, color);

    // Selection -> crop, or 'cancel' (Escape) / null (missed the picture).
    async function selectCrop(color: string): Promise<Crop | null | 'cancel'> {
      status(color);
      const sel = await requestSelection();
      if (!sel) return 'cancel';
      const media = renderedMediaRect();
      return media ? selectionToCrop(sel, media) : null;
    }

    function copyAnswer(answer: string) {
      if (!navigator.clipboard) {
        console.warn('[Assistant] Clipboard API not available');
      } else if (document.hasFocus()) {
        navigator.clipboard
          .writeText(answer)
          .catch((err) => console.warn('[Assistant] Clipboard write failed:', err));
      } else {
        console.warn('[Assistant] Clipboard write skipped: document not focused');
      }
    }

    async function answer(kind: Exclude<AskKind, 'custom'>) {
      const useSel = kind === 'mcq' ? config().answerSel : config().frqSel;
      store.set(assistantBusyAtom, true);
      try {
        let crop: Crop | undefined;
        if (useSel) {
          const r = await selectCrop(kind === 'mcq' ? STATUS.loading : STATUS.frq);
          if (r === 'cancel') return status(STATUS.idle);
          if (r === null) return status(STATUS.error);
          crop = r;
        }
        const rsp = await api.ask(kind, crop);
        if (rsp.code === api.ASSISTANT_NO_ANSWER) return status(STATUS.warning);
        if (rsp.code !== 0) {
          console.error('[Assistant]', rsp.msg, rsp.data?.route);
          return status(STATUS.error);
        }
        const text: string = rsp.data.answer;
        if (kind === 'mcq') {
          showToast(store, text);
          status(STATUS.success);
          if (config().copyClipboard) copyAnswer(text);
        } else {
          store.set(frqContentAtom, (c) => ({ text, rev: c.rev + 1 }));
          status(STATUS.success);
        }
      } catch (err) {
        console.error('[Assistant]', err);
        status(STATUS.error);
      } finally {
        store.set(assistantBusyAtom, false);
      }
    }

    async function context() {
      let crop: Crop | undefined;
      if (config().contextSel) {
        const r = await selectCrop(STATUS.context);
        if (r === 'cancel') return status(STATUS.idle);
        if (r === null) return status(STATUS.error);
        crop = r;
      }
      try {
        const rsp = await api.addContext(crop);
        if (typeof rsp.data?.count === 'number')
          store.set(assistantContextCountAtom, rsp.data.count);
        if (rsp.code !== 0) status(STATUS.error);
      } catch {
        status(STATUS.error);
      }
    }

    async function clear() {
      try {
        const rsp = await api.clearContexts();
        if (rsp.code !== 0) return status(STATUS.error);
        store.set(assistantContextCountAtom, 0);
        status(STATUS.cleared);
      } catch {
        status(STATUS.error);
      }
    }

    async function custom() {
      if (!store.get(frqVisibleAtom)) store.set(frqVisibleAtom, true);
      store.set(assistantBusyAtom, true);
      status(STATUS.frq);
      try {
        const current = store.get(frqContentAtom).text;
        if (!current.trim()) return status(STATUS.warning);
        const rsp = await api.ask('custom', null, current);
        if (rsp.code === api.ASSISTANT_NO_ANSWER) return status(STATUS.warning);
        if (rsp.code !== 0) {
          console.error('[Assistant]', rsp.msg);
          return status(STATUS.error);
        }
        const updated = current + '\n\n' + rsp.data.answer;
        store.set(frqContentAtom, (c) => ({ text: updated, rev: c.rev + 1 }));
        status(STATUS.success);
      } catch (err) {
        console.error('[Assistant]', err);
        status(STATUS.error);
      } finally {
        store.set(assistantBusyAtom, false);
      }
    }

    async function freeze() {
      const current = store.get(freezeAtom);
      if (current) {
        URL.revokeObjectURL(current.src);
        store.set(freezeAtom, null);
        return;
      }
      try {
        const blob = await api.getScreenshot();
        const rect = renderedMediaRect();
        if (!config().ui || !rect) return;
        store.set(freezeAtom, { src: URL.createObjectURL(blob), rect });
      } catch (err) {
        console.error('[Assistant] freeze capture failed', err);
        status(STATUS.error);
      }
    }

    async function cropLock() {
      if (store.get(cropLockAtom)) {
        resetCropLock();
        store.set(cropLockAtom, false);
        return status(STATUS.crop);
      }
      status(STATUS.cropSelect);
      const sel = await requestSelection();
      if (!sel) return status(STATUS.idle);
      if (!config().ui) return;
      if (applyCropLock(sel)) {
        store.set(cropLockAtom, true);
        status(STATUS.crop);
      }
    }

    async function reasoning(up: boolean) {
      try {
        const rsp = await api.adjustReasoning(up ? 'up' : 'down');
        if (rsp.code !== 0) return status(STATUS.error);
        const result = rsp.data as api.ReasoningResult;
        showToast(store, result.label);
        if (result.changed) {
          store.set(assistantConfigAtom, result.config);
          status(STATUS.success);
        }
      } catch {
        status(STATUS.error);
      }
    }

    function run(code: HotkeyCode) {
      switch (code) {
        case 'KeyA':
          return void answer('mcq');
        case 'KeyF':
          return void answer('frq');
        case 'KeyC':
          return void context();
        case 'KeyT':
          return void clear();
        case 'KeyS':
          return store.set(frqVisibleAtom, (v) => !v);
        case 'KeyM':
          return void custom();
        case 'KeyP':
          return void freeze();
        case 'KeyL':
          return void cropLock();
        case 'KeyR':
          return location.reload();
        case 'ArrowUp':
        case 'ArrowDown':
          return void reasoning(code === 'ArrowUp');
      }
    }

    function quadClick() {
      status(STATUS.loading);
      void answer('mcq');
    }

    function teardown() {
      if (store.get(cropLockAtom)) resetCropLock();
      store.set(cropLockAtom, false);
      const current = store.get(freezeAtom);
      if (current) URL.revokeObjectURL(current.src);
      store.set(freezeAtom, null);
    }

    return { run, quadClick, teardown };
  }, [store, requestSelection]);
}

export type AssistantActions = ReturnType<typeof useAssistantActions>;
