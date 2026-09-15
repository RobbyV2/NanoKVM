import assert from 'node:assert/strict';
import test from 'node:test';

import {
  detectPlatform,
  formatBytes,
  formatTime,
  formatUptime,
  isValidDNS,
  parseOrigin,
  rewriteCommand,
  wsScheme
} from './state.ts';

const shCommand =
  "curl -fsSLk -H 'Authorization: Bearer k7m2p9vx' https://nanokvm.local/exit/0/client.sh | sh";
const wstunnelCommand =
  'wstunnel client -R socks5://127.0.0.1:10820 -P exit/0 -H "Authorization: Bearer k7m2p9vx" wss://nanokvm.local';

test('a command templated for the address in the browser is left alone', () => {
  const origin = { scheme: 'https', host: 'nanokvm.local' };

  assert.equal(rewriteCommand(shCommand, origin, origin), shCommand);
});

test('a command templated behind a proxy is readdressed to the host the browser reached', () => {
  const server = { scheme: 'http', host: '10.12.34.1' };
  const local = { scheme: 'https', host: 'kvm.example.org:8443' };

  const command =
    "curl -fsSL -H 'Authorization: Bearer k7m2p9vx' http://10.12.34.1/exit/0/client.sh | sh";

  assert.equal(
    rewriteCommand(command, server, local),
    "curl -fsSL -H 'Authorization: Bearer k7m2p9vx' https://kvm.example.org:8443/exit/0/client.sh | sh"
  );
});

test('the websocket url of a wstunnel command follows the http scheme', () => {
  const server = { scheme: 'https', host: 'nanokvm.local' };
  const local = { scheme: 'http', host: '192.168.1.20' };

  assert.equal(
    rewriteCommand(wstunnelCommand, server, local),
    'wstunnel client -R socks5://127.0.0.1:10820 -P exit/0 -H "Authorization: Bearer k7m2p9vx" ws://192.168.1.20'
  );
});

test('a bracketed ipv6 host is replaced literally, not as a pattern', () => {
  const server = { scheme: 'https', host: '[fd00::1]:8443' };
  const local = { scheme: 'https', host: 'kvm.example.org' };

  assert.equal(
    rewriteCommand('irm https://[fd00::1]:8443/exit/0/client.ps1 | iex', server, local),
    'irm https://kvm.example.org/exit/0/client.ps1 | iex'
  );
});

test('a host that is a prefix of another is not rewritten inside it', () => {
  const server = { scheme: 'https', host: 'kvm' };
  const local = { scheme: 'https', host: 'kvm2' };

  assert.equal(
    rewriteCommand('curl https://kvm/exit/0/client.sh https://kvm.example.org/x', server, local),
    'curl https://kvm2/exit/0/client.sh https://kvm.example.org/x'
  );
});

test('an empty server origin means nothing to rewrite', () => {
  const local = { scheme: 'https', host: 'kvm.example.org' };

  assert.equal(rewriteCommand(shCommand, { scheme: '', host: '' }, local), shCommand);
});

test('the websocket scheme pairs with the http one', () => {
  assert.equal(wsScheme('https'), 'wss');
  assert.equal(wsScheme('http'), 'ws');
});

test('a base url splits into scheme and host with port', () => {
  assert.deepEqual(parseOrigin('https://kvm.example.org:8443'), {
    scheme: 'https',
    host: 'kvm.example.org:8443'
  });
  assert.deepEqual(parseOrigin('http://[fd00::1]'), { scheme: 'http', host: '[fd00::1]' });
});

test('uptime reads as the two largest units', () => {
  assert.equal(formatUptime(0), '0s');
  assert.equal(formatUptime(59), '59s');
  assert.equal(formatUptime(61), '1m 1s');
  assert.equal(formatUptime(3600), '1h 0m');
  assert.equal(formatUptime(90061), '1d 1h');
});

test('bytes are shown in the unit that keeps them short', () => {
  assert.equal(formatBytes(0), '0 B');
  assert.equal(formatBytes(1023), '1023 B');
  assert.equal(formatBytes(1536), '1.5 KiB');
  assert.equal(formatBytes(5 * 1024 * 1024), '5.0 MiB');
  assert.equal(formatBytes(3 * 1024 ** 3), '3.00 GiB');
});

test("go's zero time and a missing time are shown as nothing", () => {
  assert.equal(formatTime(undefined), '');
  assert.equal(formatTime(null), '');
  assert.equal(formatTime('0001-01-01T00:00:00Z'), '');
  assert.notEqual(formatTime('2026-09-15T20:00:00Z'), '');
});

test('the platform tab follows the browser', () => {
  assert.equal(detectPlatform('Mozilla/5.0 (Windows NT 10.0; Win64; x64)'), 'windows');
  assert.equal(detectPlatform('Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)'), 'macos');
  assert.equal(detectPlatform('Mozilla/5.0 (X11; Linux x86_64)'), 'linux');
  assert.equal(detectPlatform(''), 'linux');
});

test('dns servers are literal ip addresses', () => {
  assert.equal(isValidDNS('1.1.1.1'), true);
  assert.equal(isValidDNS('2606:4700:4700::1111'), true);
  assert.equal(isValidDNS('dns.google'), false);
  assert.equal(isValidDNS('1.1.1'), false);
  assert.equal(isValidDNS('256.1.1.1'), false);
  assert.equal(isValidDNS(''), false);
});
