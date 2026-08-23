import assert from 'node:assert/strict';
import test from 'node:test';

import { applyView, emptyView, switchService } from './state.ts';
import type { TunnelStatus } from './types.ts';

const running: TunnelStatus = {
  state: 'running',
  message: '',
  pid: 42,
  custom: false,
  enabled: true
};

function loadedWstunnel() {
  return applyView(emptyView('wstunnel'), 'wstunnel', {
    status: running,
    args: 'client wss://tunnel.example.org',
    env: [{ key: 'RUST_LOG', value: 'INFO', secret: false, configured: true }],
    logs: ['connected'],
    isSaved: true
  });
}

test('switching services shows nothing of the service left behind', () => {
  const view = switchService(loadedWstunnel(), 'newt');

  assert.equal(view.service, 'newt');
  assert.equal(view.args, '');
  assert.deepEqual(view.env, []);
  assert.deepEqual(view.logs, []);
  assert.equal(view.status, undefined);
  assert.equal(view.isSaved, false);
});

test('a response that lands after the switch cannot write the other service onto the panel', () => {
  let view = switchService(loadedWstunnel(), 'newt');

  view = applyView(view, 'wstunnel', {
    status: running,
    args: 'client wss://tunnel.example.org',
    logs: ['connected']
  });

  assert.equal(view.service, 'newt');
  assert.equal(view.args, '');
  assert.deepEqual(view.logs, []);
});

test('switching back to the service already on screen keeps what it loaded', () => {
  const loaded = loadedWstunnel();

  assert.equal(switchService(loaded, 'wstunnel'), loaded);
});

test('a response for the service on screen is applied', () => {
  const view = applyView(emptyView('newt'), 'newt', { args: '--foreground' });

  assert.equal(view.args, '--foreground');
});
