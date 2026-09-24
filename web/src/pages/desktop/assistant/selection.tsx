import { useCallback, useEffect, useRef, useState } from 'react';
import { atom, useAtom, useAtomValue, useSetAtom } from 'jotai';

import { assistantConfigAtom } from '@/jotai/assistant.ts';

import type { Rect } from './selection-math.ts';

type Pending = { resolve: (rect: Rect | null) => void };

const pendingSelectionAtom = atom<Pending | null>(null);

// Resolves with the dragged rectangle (viewport CSS px), or null on Escape.
export function useRequestSelection() {
  const setPending = useSetAtom(pendingSelectionAtom);
  return useCallback(
    () => new Promise<Rect | null>((resolve) => setPending({ resolve })),
    [setPending]
  );
}

function swallowKeyupOnce(code: string) {
  const handler = (e: KeyboardEvent) => {
    if (e.code !== code) return;
    e.preventDefault();
    e.stopImmediatePropagation();
    window.removeEventListener('keyup', handler, true);
  };
  window.addEventListener('keyup', handler, true);
}

export const Selection = () => {
  const ui = useAtomValue(assistantConfigAtom)?.ui ?? true;
  const [pending, setPending] = useAtom(pendingSelectionAtom);
  const start = useRef<{ x: number; y: number } | null>(null);
  const [rect, setRect] = useState<Rect | null>(null);

  const finish = useCallback(
    (result: Rect | null) => {
      pending?.resolve(result);
      start.current = null;
      setRect(null);
      setPending(null);
    },
    [pending, setPending]
  );

  useEffect(() => {
    if (!pending) return;
    const onKeyDown = (e: KeyboardEvent) => {
      if (e.key !== 'Escape') return;
      e.preventDefault();
      e.stopImmediatePropagation();
      swallowKeyupOnce(e.code);
      finish(null);
    };
    const oldUserSelect = document.body.style.userSelect;
    document.body.style.userSelect = 'none';
    window.addEventListener('keydown', onKeyDown, true);
    return () => {
      window.removeEventListener('keydown', onKeyDown, true);
      document.body.style.userSelect = oldUserSelect;
    };
  }, [pending, finish]);

  if (!pending) return null;

  return (
    <>
      <div
        className="chaice-overlay"
        style={{
          position: 'fixed',
          left: 0,
          top: 0,
          width: '100vw',
          height: '100vh',
          zIndex: 10000,
          background: 'rgba(0,0,0,0)',
          pointerEvents: 'all'
        }}
        onMouseDown={(e) => {
          if (e.button !== 0) return;
          e.preventDefault();
          e.stopPropagation();
          start.current = { x: e.clientX, y: e.clientY };
          setRect({ left: e.clientX, top: e.clientY, width: 0, height: 0 });
        }}
        onMouseMove={(e) => {
          const s = start.current;
          if (!s) return;
          e.preventDefault();
          const w = e.clientX - s.x;
          const h = e.clientY - s.y;
          setRect({
            left: w < 0 ? s.x + w : s.x,
            top: h < 0 ? s.y + h : s.y,
            width: Math.abs(w),
            height: Math.abs(h)
          });
        }}
        onMouseUp={(e) => {
          if (!start.current) return;
          e.preventDefault();
          e.stopPropagation();
          finish(rect ?? { left: e.clientX, top: e.clientY, width: 0, height: 0 });
        }}
      />
      {rect && (
        <div
          className="selection-rect"
          style={{
            position: 'fixed',
            left: `${rect.left}px`,
            top: `${rect.top}px`,
            width: `${rect.width}px`,
            height: `${rect.height}px`,
            pointerEvents: 'none',
            zIndex: 10001,
            background: 'none',
            // UI off: still laid out and dragged, never seen.
            border: ui ? '2px dashed #e0cfff' : 'none',
            opacity: ui ? 0.45 : 0
          }}
        />
      )}
    </>
  );
};
