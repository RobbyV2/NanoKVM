import { http } from '@/lib/http.ts';
import type { SetExitConfigReq } from '@/pages/desktop/menu/settings/exit/types.ts';

// Admin API of the exit tunnel, JWT and admin role, {code,msg,data} envelope.
// The token-gated /exit/:slot/* surface is not here: it is what the exit device
// calls, and the panel only reaches it to show a script (see commands.tsx).

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
