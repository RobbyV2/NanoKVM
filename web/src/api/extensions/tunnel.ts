import { http } from '@/lib/http.ts';

// get tunnel status
export function getStatus(name: string) {
  return http.get(`/api/extensions/tunnel/${name}/status`);
}

// get tunnel config
export function getConfig(name: string) {
  return http.get(`/api/extensions/tunnel/${name}/config`);
}

// update tunnel config
export function setConfig(name: string, args: string, env: Record<string, string>) {
  return http.post(`/api/extensions/tunnel/${name}/config`, { args, env });
}

// start tunnel
export function start(name: string) {
  return http.post(`/api/extensions/tunnel/${name}/start`);
}

// stop tunnel
export function stop(name: string) {
  return http.post(`/api/extensions/tunnel/${name}/stop`);
}

// restart tunnel
export function restart(name: string) {
  return http.post(`/api/extensions/tunnel/${name}/restart`);
}

// get tunnel logs
export function getLogs(name: string) {
  return http.get(`/api/extensions/tunnel/${name}/logs`);
}

// get tunnel memory limit
export function getMemory(name: string) {
  return http.get(`/api/extensions/tunnel/${name}/memory`);
}

// set tunnel memory limit
export function setMemory(name: string, enabled: boolean) {
  return http.post(`/api/extensions/tunnel/${name}/memory`, { enabled });
}

// upload a custom tunnel binary
export function uploadBinary(name: string, file: File) {
  const formData = new FormData();
  formData.append('file', file);

  return file
    .arrayBuffer()
    .then((buffer) => crypto.subtle.digest('SHA-256', buffer))
    .then((digest) => {
      const checksum = Array.from(new Uint8Array(digest))
        .map((byte) => byte.toString(16).padStart(2, '0'))
        .join('');

      return http.post(`/api/extensions/tunnel/${name}/binary`, formData, {
        headers: { 'Content-Type': 'multipart/form-data', 'X-SHA256-Checksum': checksum }
      });
    });
}

// remove the custom tunnel binary
export function deleteBinary(name: string) {
  return http.delete(`/api/extensions/tunnel/${name}/binary`);
}
