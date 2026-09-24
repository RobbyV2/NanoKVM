import { http } from '@/lib/http.ts';

export type Provider = 'gemini' | 'openrouter';
export type ReasoningEffort = 'low' | 'medium' | 'high';

export type AssistantConfig = {
  enabled: boolean;
  ui: boolean;
  provider: Provider;
  geminiBaseUrl: string;
  geminiModel: string;
  geminiThinking: boolean;
  thinkingBudget: number;
  orBaseUrl: string;
  orModel: string;
  orReasoning: boolean;
  orReasoningEffort: ReasoningEffort;
  proxyUrl: string;
  copyClipboard: boolean;
  answerSel: boolean;
  contextSel: boolean;
  frqSel: boolean;
  quadClickMCQ: boolean;
  hasGeminiApiKey: boolean;
  hasOrApiKey: boolean;
  hasProxyPass: boolean;
};

export type AssistantConfigUpdate = Partial<
  Omit<AssistantConfig, 'hasGeminiApiKey' | 'hasOrApiKey' | 'hasProxyPass'>
> & {
  geminiApiKey?: string;
  orApiKey?: string;
  proxyPass?: string;
  clearGeminiApiKey?: boolean;
  clearOrApiKey?: boolean;
  clearProxyPass?: boolean;
};

export type AskKind = 'mcq' | 'frq' | 'custom';
export type Crop = { x: number; y: number; w: number; h: number };
export type AttachmentInfo = { name: string; size: number };
export type ReasoningResult = { label: string; changed: boolean; config: AssistantConfig };

export const ASSISTANT_DISABLED = -2;
export const ASSISTANT_NO_ANSWER = -3;

export function getAssistantConfig() {
  return http.get('/api/assistant/config');
}

export function setAssistantConfig(update: AssistantConfigUpdate) {
  return http.post('/api/assistant/config', update);
}

// The server may spend 2 x 300 s on the relay, 5 s between, then 300 s direct.
export function ask(kind: AskKind, crop?: Crop | null, text?: string) {
  return http.post('/api/assistant/ask', { kind, crop: crop ?? undefined, text }, { timeout: 0 });
}

export function addContext(crop?: Crop | null) {
  return http.post('/api/assistant/context', { crop: crop ?? undefined });
}

export function clearContexts() {
  return http.delete('/api/assistant/context');
}

export function getContextCount() {
  return http.get('/api/assistant/context');
}

export async function getScreenshot(): Promise<Blob> {
  const blob = (await http.request({
    method: 'get',
    url: '/api/assistant/screenshot',
    responseType: 'blob'
  })) as unknown as Blob;
  if (blob.type !== 'image/jpeg') {
    throw new Error(await blob.text());
  }
  return blob;
}

export function adjustReasoning(direction: 'up' | 'down') {
  return http.post('/api/assistant/reasoning', { direction });
}

export function listAttachments() {
  return http.get('/api/assistant/attachments');
}

export function uploadAttachment(file: File) {
  const form = new FormData();
  form.append('file', file);
  return http.post('/api/assistant/attachments', form, { timeout: 0 });
}

export function deleteAttachment(name: string) {
  return http.delete(`/api/assistant/attachments?name=${encodeURIComponent(name)}`);
}
