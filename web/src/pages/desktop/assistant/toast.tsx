import { useEffect, useState } from 'react';
import { atom, useAtomValue, useStore } from 'jotai';

import { assistantConfigAtom } from '@/jotai/assistant.ts';

type Store = ReturnType<typeof useStore>;

const toastAtom = atom<{ text: string; id: number } | null>(null);
let toastId = 0;

// The answer toast: first 40 words, gone after 1.75 s (content.js showLLMAnswer).
export function showToast(store: Store, answer: string) {
  if (!store.get(assistantConfigAtom)?.ui) return;
  const text = answer.trim().split(/\s+/).slice(0, 40).join(' ');
  store.set(toastAtom, { text, id: ++toastId });
}

export const Toast = () => {
  const toast = useAtomValue(toastAtom);
  const [visible, setVisible] = useState(false);

  useEffect(() => {
    if (!toast) return;
    setVisible(true);
    const timer = setTimeout(() => setVisible(false), 1750);
    return () => clearTimeout(timer);
  }, [toast]);

  if (!toast) return null;
  return (
    <div
      id="llm-answer"
      style={{
        position: 'fixed',
        left: '50%',
        bottom: '28px',
        transform: 'translateX(-50%)',
        color: '#800080',
        fontSize: '13px',
        fontFamily: 'sans-serif',
        fontWeight: 400,
        zIndex: 10003,
        pointerEvents: 'none',
        background: 'none',
        textAlign: 'center',
        opacity: visible ? 1 : 0
      }}
    >
      {toast.text}
    </div>
  );
};
