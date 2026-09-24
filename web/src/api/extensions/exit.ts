import { http } from '@/lib/http.ts';
import { getBaseUrl } from '@/lib/service.ts';
import { nexitDownloadRequest } from '@/pages/desktop/menu/settings/exit/state.ts';
import type { NexitDownload } from '@/pages/desktop/menu/settings/exit/state.ts';
import type { SetExitConfigReq } from '@/pages/desktop/menu/settings/exit/types.ts';

// Admin API of the exit tunnel, JWT and admin role, {code,msg,data} envelope.
// The token-gated /exit/:slot/* surface is what the exit device calls; the
// panel reaches it only to download nexit and its nexit.json (downloadNexit).

const base = '/api/extensions/exit';

// list every slot with its status
export function getSlots() {
  return http.get(`${base}/slots`);
}

// get one slot's status (polled every 3 s while the panel is open)
export function getStatus(slot: string) {
  return http.get(`${base}/${slot}/status`);
}

// get the editable config without the token
export function getConfig(slot: string) {
  return http.get(`${base}/${slot}/config`);
}

// update the editable config; absent fields are left alone
export function setConfig(slot: string, req: SetExitConfigReq) {
  return http.post(`${base}/${slot}/config`, req);
}

// run the enable transaction (D23)
export function enable(slot: string) {
  return http.post(`${base}/${slot}/enable`);
}

// run the disable transaction (D23)
export function disable(slot: string) {
  return http.post(`${base}/${slot}/disable`);
}

// issue a new token and drop every connected exit (D11)
export function regenerateToken(slot: string) {
  return http.post(`${base}/${slot}/token/regenerate`);
}

// drop the active exit without changing anything else
export function disconnect(slot: string) {
  return http.post(`${base}/${slot}/disconnect`);
}

// templated commands for both modes and three platforms, plus the fingerprint
export function getCommands(slot: string) {
  return http.get(`${base}/${slot}/commands`);
}

// hev and wstunnel log tails, token-shaped values redacted
export function getLogs(slot: string) {
  return http.get(`${base}/${slot}/logs`);
}

// fetch nexit.exe or nexit.json from the token-gated surface with the slot's
// token and save it under the name nexit expects. The gate answers every
// refusal (disabled slot, wrong token, locked source) with the same 404.
export async function downloadNexit(slot: string, token: string, download: NexitDownload) {
  const { url, init } = nexitDownloadRequest(getBaseUrl('http'), slot, token, download.asset);
  const rsp = await fetch(url, init);
  if (!rsp.ok) {
    throw new Error(`HTTP ${rsp.status}`);
  }
  const blob = await rsp.blob();

  const href = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = href;
  link.download = download.file;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(href);
}
