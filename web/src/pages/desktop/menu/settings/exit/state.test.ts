import assert from 'node:assert/strict';
import test from 'node:test';

import {
  advancedValuesKey,
  detectPlatform,
  formatBytes,
  formatTime,
  formatUptime,
  isScriptServed,
  isValidDNS,
  mergeSaved,
  parseOrigin,
  presentCommand,
  resolverProblem,
  scriptErrorKey,
  serialRefresher,
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

const latestLinuxCommand =
  'set -e; fail() { echo "could not fetch the latest wstunnel ($1); download it from https://github.com/erebe/wstunnel/releases" >&2; exit 1; }; d=$(mktemp -d); cd "$d"; case "$(uname -m)" in x86_64) a=amd64;; aarch64|arm64) a=arm64;; *) echo "unsupported architecture: $(uname -m)" >&2; exit 1;; esac; v=$(curl -fsSLo /dev/null -w \'%{url_effective}\' https://github.com/erebe/wstunnel/releases/latest) || fail resolve; v=${v##*/v}; f="wstunnel_${v}_linux_${a}.tar.gz"; curl -fsSLo wstunnel.tgz "https://github.com/erebe/wstunnel/releases/download/v$v/$f" || fail download; curl -fsSL "https://github.com/erebe/wstunnel/releases/download/v$v/checksums.txt" | grep " $f$" | sed \'s/  .*/  wstunnel.tgz/\' | sha256sum -c - || fail checksum; tar -xzf wstunnel.tgz wstunnel || fail extract; exec ./wstunnel client -P exit/0 -H \'Authorization: Bearer k7m2p9vx\' -R socks5://127.0.0.1:10820 \'wss://kvm.example.net\'';
const latestWindowsCommand =
  "try { $v=(Invoke-RestMethod -UseBasicParsing 'https://api.github.com/repos/erebe/wstunnel/releases/latest').tag_name.TrimStart('v'); $a=if ($env:PROCESSOR_ARCHITECTURE -eq 'ARM64') {'arm64'} else {'amd64'}; $n=\"wstunnel_${v}_windows_$a.tar.gz\"; $d=Join-Path $env:TEMP 'wstunnel-latest'; New-Item -Force -ItemType Directory $d | Out-Null; $f=Join-Path $d 'wstunnel.tar.gz'; Invoke-WebRequest -UseBasicParsing \"https://github.com/erebe/wstunnel/releases/download/v$v/$n\" -OutFile $f; $s=Join-Path $d 'checksums.txt'; Invoke-WebRequest -UseBasicParsing \"https://github.com/erebe/wstunnel/releases/download/v$v/checksums.txt\" -OutFile $s; $h=(Select-String -SimpleMatch -Pattern \"  $n\" -Path $s | Select-Object -First 1).Line; if (-not $h -or $h.Split(' ')[0] -ne (Get-FileHash $f -Algorithm SHA256).Hash) { throw 'wstunnel checksum mismatch' }; tar -xzf $f -C $d; & (Join-Path $d 'wstunnel.exe') client -P exit/0 -H 'Authorization: Bearer k7m2p9vx' -R socks5://127.0.0.1:10820 'wss://kvm.example.net' } catch { throw \"could not fetch the latest wstunnel: $_ Download it from https://github.com/erebe/wstunnel/releases\" }";

test('a latest-release wstunnel command readdresses the kvm url and no github url', () => {
  const server = { scheme: 'https', host: 'kvm.example.net' };
  const local = { scheme: 'https', host: 'kvm.example.org:8443' };

  for (const command of [latestLinuxCommand, latestWindowsCommand]) {
    const view = presentCommand('wstunnel', { command }, server, local, fingerprint);
    assert.equal(
      view.text,
      command.replace("'wss://kvm.example.net'", "'wss://kvm.example.org:8443'")
    );
    assert.equal(view.rewritten, true);
    // the github urls are hosts of their own and stay as the server wrote them
    assert.ok(view.text.includes('https://github.com/erebe/wstunnel/releases'));
    assert.equal(view.text.includes('kvm.example.net'), false);
  }
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

test('dns servers are literal ip addresses, parsed as the server parses them', () => {
  assert.equal(isValidDNS('1.1.1.1'), true);
  assert.equal(isValidDNS(' 1.1.1.1 '), true);
  assert.equal(isValidDNS('2606:4700:4700::1111'), true);
  assert.equal(isValidDNS('1:2:3:4:5:6:7:8'), true);
  // ipv4-mapped: net.ParseIP takes it, so must the panel
  assert.equal(isValidDNS('::ffff:1.2.3.4'), true);
  assert.equal(isValidDNS('dns.google'), false);
  assert.equal(isValidDNS('1.1.1'), false);
  assert.equal(isValidDNS('256.1.1.1'), false);
  assert.equal(isValidDNS('01.1.1.1'), false);
  assert.equal(isValidDNS(''), false);
  // malformed ipv6 the old charset check let through; `dive,ip` refuses them
  assert.equal(isValidDNS(':::'), false);
  assert.equal(isValidDNS(':1'), false);
  assert.equal(isValidDNS('1:'), false);
  assert.equal(isValidDNS('1:2:3:4:5:6:7:8:9'), false);
  assert.equal(isValidDNS('12345::1'), false);
  assert.equal(isValidDNS('fe80::1%eth0'), false);
});

test('a resolver the front door would always refuse is refused inline (D21)', () => {
  for (const denied of [
    '127.0.0.1',
    '0.0.0.0',
    '169.254.169.254',
    '198.18.0.53',
    '198.19.255.1',
    '224.0.0.251',
    '255.255.255.255',
    '::',
    '::1',
    'fe80::1',
    'febf::1',
    'ff02::fb',
    // mapped loopback is judged as ipv4
    '::ffff:127.0.0.1'
  ]) {
    assert.equal(resolverProblem(denied, true), 'dnsDenied', denied);
  }
});

test('a private resolver needs allowPrivate, a public one never does', () => {
  for (const priv of [
    '10.0.0.53',
    '100.64.0.1',
    '172.16.0.1',
    '172.31.255.254',
    '192.168.1.1',
    'fd00::53',
    'fc00::1'
  ]) {
    assert.equal(resolverProblem(priv, false), 'dnsPrivate', priv);
    assert.equal(resolverProblem(priv, true), '', priv);
  }
  for (const pub of [
    '1.1.1.1',
    '8.8.8.8',
    '172.32.0.1',
    '100.128.0.1',
    '2606:4700:4700::1111',
    '::ffff:9.9.9.9'
  ]) {
    assert.equal(resolverProblem(pub, false), '', pub);
  }
  assert.equal(resolverProblem('dns.google', true), 'dnsInvalid');
  assert.equal(resolverProblem('', true), 'dnsInvalid');
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

const settle = () => new Promise((resolve) => setImmediate(resolve));

test('a refresh asked for while a poll is in flight runs once the poll answers', async () => {
  const pending: (() => void)[] = [];
  let runs = 0;
  const status = serialRefresher(() => {
    runs++;
    return new Promise<void>((resolve) => pending.push(resolve));
  });

  // the interval tick: one request, a second tick while it is out is dropped
  status.poll();
  status.poll();
  assert.equal(runs, 1);

  // an action completed while that poll was out: its answer predates the
  // action, so a follow-up is owed, but only one however many actions asked
  status.refresh();
  status.refresh();
  assert.equal(runs, 1);

  pending[0]();
  await settle();
  assert.equal(runs, 2);

  pending[1]();
  await settle();
  assert.equal(runs, 2);

  // idle: a refresh goes out at once
  status.refresh();
  assert.equal(runs, 3);
});

test('a failed request does not wedge the refresher', async () => {
  let runs = 0;
  const status = serialRefresher(() => {
    runs++;
    return Promise.reject(new Error('down'));
  });

  status.poll();
  status.refresh();
  await settle();
  assert.equal(runs, 2);

  status.poll();
  assert.equal(runs, 3);
});

const config = {
  slot: '0',
  mode: 'native' as const,
  dns: ['1.1.1.1'],
  mtu: 1280,
  allowPrivate: false,
  pinPeer: false
};

test('a save lands on the config as it is by then, not as it was when the save started', () => {
  // the operator switched mode while the save was out and the switch answered first
  const switched = { ...config, mode: 'wstunnel' as const };
  const saved = { dns: ['9.9.9.9'], mtu: 1280, allowPrivate: false, pinPeer: false };

  assert.deepEqual(mergeSaved(switched, saved), { ...switched, dns: ['9.9.9.9'] });
  assert.equal(mergeSaved(undefined, saved), undefined);
});

test('the advanced form is re-seeded only when the values it edits change', () => {
  assert.equal(advancedValuesKey({ ...config, mode: 'wstunnel' }), advancedValuesKey(config));
  assert.notEqual(advancedValuesKey({ ...config, dns: ['9.9.9.9'] }), advancedValuesKey(config));
  assert.notEqual(advancedValuesKey({ ...config, mtu: 1400 }), advancedValuesKey(config));
  assert.notEqual(advancedValuesKey({ ...config, allowPrivate: true }), advancedValuesKey(config));
  assert.notEqual(advancedValuesKey({ ...config, pinPeer: true }), advancedValuesKey(config));
  assert.equal(advancedValuesKey(undefined), '');
});
