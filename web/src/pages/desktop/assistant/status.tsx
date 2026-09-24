import type { CSSProperties } from 'react';
import { atom, useAtomValue, useStore } from 'jotai';

import {
  assistantBusyAtom,
  assistantConfigAtom,
  assistantContextCountAtom
} from '@/jotai/assistant.ts';

import { freezeAtom } from './freeze.tsx';

type Store = ReturnType<typeof useStore>;

// Colours verbatim from the chaice content script.
export const STATUS = {
  idle: 'rgb(40,62,159)',
  loading: '#b3cfff',
  frq: '#e0cfff',
  context: '#0a1333',
  success: 'rgb(89, 105, 192)',
  error: 'red',
  warning: 'yellow',
  cleared: '#444',
  crop: '#285e9f',
  cropSelect: 'orange'
} as const;

const statusColorAtom = atom<string>(STATUS.idle);
let resetTimer: ReturnType<typeof setTimeout> | undefined;

export function setStatus(store: Store, color: string) {
  if (!store.get(assistantConfigAtom)?.ui) return;
  clearTimeout(resetTimer);
  store.set(statusColorAtom, color);
  resetTimer = setTimeout(() => store.set(statusColorAtom, STATUS.idle), 3000);
}

const dot = (background: string, top: string): CSSProperties => ({
  position: 'fixed',
  left: '4px',
  top,
  width: '8px',
  height: '8px',
  borderRadius: '50%',
  background,
  zIndex: 10051,
  boxShadow: `0 0 2px ${background}`,
  pointerEvents: 'none'
});

export const StatusBar = () => {
  const color = useAtomValue(statusColorAtom);
  const count = useAtomValue(assistantContextCountAtom);
  const busy = useAtomValue(assistantBusyAtom);
  const frozen = useAtomValue(freezeAtom);

  return (
    <>
      <div
        id="status-bar"
        style={{
          position: 'fixed',
          bottom: 0,
          left: 0,
          height: '2px',
          width: '100%',
          background: color,
          zIndex: 9999,
          transition: 'background 0.2s'
        }}
      />
      <div
        id="context-count"
        style={{
          position: 'fixed',
          left: '18px',
          bottom: '28px',
          color: '#800080',
          fontSize: '13px',
          fontFamily: 'sans-serif',
          fontWeight: 400,
          zIndex: 10003,
          pointerEvents: 'none',
          background: 'none',
          textAlign: 'left'
        }}
      >
        {count}
      </div>
      {busy && <div className="llm-loading-dot" style={dot('rgb(40,62,159)', '18px')} />}
      {frozen && <div className="freeze-dot" style={dot('#ffe600', '4px')} />}
    </>
  );
};
