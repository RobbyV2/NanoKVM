import { atom } from 'jotai';

import type { AssistantConfig } from '@/api/assistant.ts';

// Loaded once on the desktop page for admins; the settings tab replaces it on save.
export const assistantConfigAtom = atom<AssistantConfig | null>(null);
export const assistantContextCountAtom = atom(0);
// The LLM loading dot.
export const assistantBusyAtom = atom(false);
