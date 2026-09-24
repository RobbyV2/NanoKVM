import { useEffect } from 'react';

// Four clicks within 2 s anywhere but the assistant's own UI -> MCQ.
export function useQuadClick(enabled: boolean, onQuad: () => void) {
  useEffect(() => {
    if (!enabled) return;
    let stamps: number[] = [];
    const handler = (e: MouseEvent) => {
      const target = e.target as Element | null;
      if (target?.closest?.('.chaice-overlay, #llm-answer, #frq-box, #context-count')) return;
      const now = Date.now();
      stamps.push(now);
      stamps = stamps.filter((t) => now - t < 2000);
      if (stamps.length >= 4) {
        stamps = [];
        onQuad();
      }
    };
    document.addEventListener('click', handler, true);
    return () => document.removeEventListener('click', handler, true);
  }, [enabled, onQuad]);
}
