import { atom, useAtomValue } from 'jotai';

import type { Rect } from './selection-math.ts';

// P: a still of the target laid over the picture; pointer-events pass through.
export const freezeAtom = atom<{ src: string; rect: Rect } | null>(null);

export const FreezeOverlay = () => {
  const freeze = useAtomValue(freezeAtom);
  if (!freeze) return null;
  return (
    <div
      className="freeze-overlay"
      style={{
        position: 'fixed',
        left: `${freeze.rect.left}px`,
        top: `${freeze.rect.top}px`,
        width: `${freeze.rect.width}px`,
        height: `${freeze.rect.height}px`,
        zIndex: 10050,
        background: `rgba(255,255,255,0.01) url('${freeze.src}') center center / 100% 100% no-repeat`,
        pointerEvents: 'none',
        transition: 'opacity 0.2s'
      }}
    />
  );
};
