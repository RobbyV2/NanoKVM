import assert from 'node:assert/strict';
import test from 'node:test';

import { withBrowserOrigin } from './state.ts';

const serverJson = (address: unknown) =>
  JSON.stringify({ address, passcode: 'abc123', insecure: true, allowPrivate: false }, null, 2) +
  '\n';

const https = { protocol: 'https:', host: 'kvm.example.org' };

test('withBrowserOrigin replaces the LAN ip with the domain the page is on', () => {
  const out = JSON.parse(withBrowserOrigin(serverJson('https://192.168.8.210/exit/0'), https));
  assert.equal(out.address, 'https://kvm.example.org/exit/0');
});

test('withBrowserOrigin follows an http page to an http scheme', () => {
  const out = JSON.parse(
    withBrowserOrigin(serverJson('https://192.168.8.210/exit/0'), {
      protocol: 'http:',
      host: 'kvm.lan'
    })
  );
  assert.equal(out.address, 'http://kvm.lan/exit/0');
});

test('withBrowserOrigin keeps the browser port', () => {
  const out = JSON.parse(
    withBrowserOrigin(serverJson('http://192.168.8.210/exit/1'), {
      protocol: 'https:',
      host: 'kvm.example.org:8443'
    })
  );
  assert.equal(out.address, 'https://kvm.example.org:8443/exit/1');
});

test('withBrowserOrigin keeps the server path', () => {
  const out = JSON.parse(withBrowserOrigin(serverJson('https://10.0.0.5:444/exit/3'), https));
  assert.equal(out.address, 'https://kvm.example.org/exit/3');
});

test('withBrowserOrigin leaves the other fields and their order alone', () => {
  const out = withBrowserOrigin(serverJson('https://192.168.8.210/exit/0'), https);
  assert.equal(
    out,
    '{\n  "address": "https://kvm.example.org/exit/0",\n  "passcode": "abc123",\n' +
      '  "insecure": true,\n  "allowPrivate": false\n}\n'
  );
});

test('withBrowserOrigin returns invalid or missing addresses unchanged', () => {
  for (const input of [
    serverJson('not a url'),
    serverJson(42),
    JSON.stringify({ passcode: 'x' }),
    'not json',
    '[]'
  ]) {
    assert.equal(withBrowserOrigin(input, https), input);
  }
});
