import assert from 'node:assert/strict';
import test from 'node:test';

import {
  detectPlatform,
  formatBytes,
  formatTime,
  formatUptime,
  isScriptServed,
  isValidDNS,
  parseOrigin,
  presentCommand,
  scriptErrorKey,
  wsScheme
} from './state.ts';

const shCommand =
  "curl -fsSLk -H 'Authorization: Bearer k7m2p9vx' https://nanokvm.local/exit/0/client.sh | sh";
const wstunnelCommand =
  'wstunnel client -R socks5://127.0.0.1:10820 -P exit/0 -H "Authorization: Bearer k7m2p9vx" --tls-verify-certificate wss://nanokvm.local';
const fingerprint = 'ab'.repeat(32);

test('a command templated for the address in the browser is shown as is', () => {
  const origin = { scheme: 'https', host: 'nanokvm.local' };

  const view = presentCommand('native', { command: shCommand }, origin, origin, fingerprint);
  assert.equal(view.text, shCommand);
  assert.equal(view.rewritten, false);
  assert.equal(view.schemeMismatch, false);
  assert.equal(view.cleartext, false);
  assert.equal(view.pinsFingerprint, true);
  assert.equal(view.pinUncertain, false);
});

test('behind a proxy on the same scheme only the host is readdressed', () => {
  const server = { scheme: 'https', host: '10.12.34.1' };
  const local = { scheme: 'https', host: 'kvm.example.org:8443' };

  const view = presentCommand(
    'native',
    { command: shCommand.replace('nanokvm.local', '10.12.34.1') },
    server,
    local,
    fingerprint
  );
  assert.equal(
    view.text,
    "curl -fsSLk -H 'Authorization: Bearer k7m2p9vx' https://kvm.example.org:8443/exit/0/client.sh | sh"
  );
  assert.equal(view.rewritten, true);
  assert.equal(view.schemeMismatch, false);
  // the script pins the device leaf; a proxy that terminates tls presents another
  assert.equal(view.pinsFingerprint, false);
  assert.equal(view.pinUncertain, true);
});

test('a scheme the server did not see is not templated in the browser', () => {
  const server = { scheme: 'http', host: '10.12.34.1' };
  const local = { scheme: 'https', host: 'kvm.example.org' };

  const command =
    "curl -fsSL -H 'Authorization: Bearer k7m2p9vx' http://10.12.34.1/exit/0/client.sh | sh";

  const view = presentCommand('native', { command }, server, local, '');
  // rewriting would drop -k and the powershell prefix: curl exit 60 on the device cert
  assert.equal(view.text, command);
  assert.equal(view.rewritten, false);
  assert.equal(view.schemeMismatch, true);
  assert.equal(view.scheme, 'http');
  // the command dials http whatever the page was served over
  assert.equal(view.cleartext, true);
  assert.equal(view.pinsFingerprint, false);
});

test('--tls-verify-certificate never lands on a ws:// url', () => {
  const server = { scheme: 'https', host: 'nanokvm.local' };
  const local = { scheme: 'http', host: '192.168.1.20' };

  const view = presentCommand('wstunnel', { command: wstunnelCommand }, server, local, fingerprint);
  assert.equal(view.text, wstunnelCommand);
  assert.equal(view.schemeMismatch, true);
  assert.equal(view.scheme, 'https');
  assert.equal(view.cleartext, false);
  assert.equal(view.wstunnelUnverified, false);
});

test('the websocket url of a wstunnel command follows the host on the same scheme', () => {
  const server = { scheme: 'https', host: 'nanokvm.local' };
  const local = { scheme: 'https', host: 'kvm.example.org' };

  const view = presentCommand('wstunnel', { command: wstunnelCommand }, server, local, fingerprint);
  assert.equal(
    view.text,
    'wstunnel client -R socks5://127.0.0.1:10820 -P exit/0 -H "Authorization: Bearer k7m2p9vx" --tls-verify-certificate wss://kvm.example.org'
  );
  assert.equal(view.rewritten, true);
});

test('a wstunnel command without the verify flag is called unauthenticated only over https', () => {
  const origin = { scheme: 'https', host: 'nanokvm.local' };
  const plain = { scheme: 'http', host: 'nanokvm.local' };
  const unverified = wstunnelCommand.replace(' --tls-verify-certificate', '');

  assert.equal(
    presentCommand('wstunnel', { command: unverified }, origin, origin, '').wstunnelUnverified,
    true
  );
  assert.equal(
    presentCommand('wstunnel', { command: unverified.replace('wss://', 'ws://') }, plain, plain, '')
      .wstunnelUnverified,
    false
  );
  assert.equal(
    presentCommand('native', { command: shCommand }, origin, origin, '').wstunnelUnverified,
    false
  );
});

test('a bracketed ipv6 host is replaced literally, not as a pattern', () => {
  const server = { scheme: 'https', host: '[fd00::1]:8443' };
  const local = { scheme: 'https', host: 'kvm.example.org' };

  assert.equal(
    presentCommand(
      'native',
      { command: 'irm https://[fd00::1]:8443/exit/0/client.ps1 | iex' },
      server,
      local,
      ''
    ).text,
    'irm https://kvm.example.org/exit/0/client.ps1 | iex'
  );
});

test('a host that is a prefix of another is not rewritten inside it', () => {
  const server = { scheme: 'https', host: 'kvm' };
  const local = { scheme: 'https', host: 'kvm2' };

  assert.equal(
    presentCommand(
      'native',
      { command: 'curl https://kvm/exit/0/client.sh https://kvm.example.org/x' },
      server,
      local,
      ''
    ).text,
    'curl https://kvm2/exit/0/client.sh https://kvm.example.org/x'
  );
});

test('an empty server origin means nothing to rewrite', () => {
  const local = { scheme: 'https', host: 'kvm.example.org' };

  const view = presentCommand(
    'native',
    { command: shCommand },
    { scheme: '', host: '' },
    local,
    ''
  );
  assert.equal(view.text, shCommand);
  assert.equal(view.rewritten, false);
  assert.equal(view.schemeMismatch, false);
});

test('no command for the platform presents as nothing', () => {
  const origin = { scheme: 'https', host: 'nanokvm.local' };

  const view = presentCommand('native', undefined, origin, origin, fingerprint);
  assert.equal(view.text, '');
  assert.equal(view.pinsFingerprint, false);
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

test('the script is asked for only while the gate would serve it', () => {
  // the gate answers a disabled or pending slot with the same 404 as a wrong
  // token and charges the source a failure (D10), so the panel must not ask
  assert.equal(isScriptServed({ enabled: true, pending: false }), true);
  assert.equal(isScriptServed({ enabled: false, pending: false }), false);
  assert.equal(isScriptServed({ enabled: true, pending: true }), false);
  assert.equal(isScriptServed({ enabled: false, pending: true }), false);
});

test('a 404 from the script route is named as the gate, anything else as a failure', () => {
  assert.equal(scriptErrorKey(404), 'scriptGated');
  assert.equal(scriptErrorKey(500), 'scriptFailed');
  assert.equal(scriptErrorKey(0), 'scriptFailed');
});
