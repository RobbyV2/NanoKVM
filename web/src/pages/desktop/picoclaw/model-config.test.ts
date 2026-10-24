import assert from 'node:assert/strict';
import test from 'node:test';

import {
  modelConfigErrorFromException,
  modelConfigErrorFromResponse,
  toModelConfigRequest
} from './model-config.ts';

test('the request carries trimmed values and leaves optional fields to the server', () => {
  assert.deepEqual(
    toModelConfigRequest({ model: ' openai/gpt-5.4\n', apiBase: '  ', apiKey: ' sk-test ' }),
    { model: 'openai/gpt-5.4', api_base: '', api_key: 'sk-test' }
  );
});

test('a rejected save shows the server message and the field to correct', () => {
  const response = {
    code: 'INVALID_ACTION',
    message: 'PicoClaw does not support the provider "openao"; did you mean openai?',
    data: { field: 'model' }
  };
  assert.deepEqual(modelConfigErrorFromResponse(response, 'fallback'), {
    message: response.message,
    field: 'model'
  });
});

test('an unknown field is not attached to an input', () => {
  assert.deepEqual(
    modelConfigErrorFromResponse({ code: 'X', message: 'boom', data: { field: 'gateway' } }, 'f'),
    { message: 'boom', field: undefined }
  );
});

test('a response without a message falls back to the generic text', () => {
  assert.deepEqual(modelConfigErrorFromResponse({ code: 'X', message: ' ' }, 'Failed to save'), {
    message: 'Failed to save',
    field: undefined
  });
  assert.deepEqual(modelConfigErrorFromResponse({ code: 'X', msg: 'from msg' }, 'f'), {
    message: 'from msg',
    field: undefined
  });
  assert.deepEqual(modelConfigErrorFromResponse(null, 'Failed to save'), {
    message: 'Failed to save'
  });
});

test('an HTTP error keeps the API error in its body', () => {
  const error = Object.assign(new Error('Request failed with status code 403'), {
    response: { status: 403, data: { code: 'FORBIDDEN', message: 'administrator role required' } }
  });
  assert.deepEqual(modelConfigErrorFromException(error, 'f'), {
    message: 'administrator role required',
    field: undefined
  });
});

test('a timeout or network failure shows the transport error', () => {
  assert.deepEqual(modelConfigErrorFromException(new Error('timeout of 60000ms exceeded'), 'f'), {
    message: 'timeout of 60000ms exceeded'
  });
  assert.deepEqual(modelConfigErrorFromException('weird', 'Failed to save'), {
    message: 'Failed to save'
  });
});
