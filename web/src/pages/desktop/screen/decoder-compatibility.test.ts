import assert from 'node:assert/strict';
import test from 'node:test';

import { directDecoderAcceleration } from './decoder-compatibility.ts';

const linuxFirefox = (version: string) =>
  `Mozilla/5.0 (X11; Linux x86_64; rv:${version}) Gecko/20100101 Firefox/${version}`;

test('Linux Firefox versions with the VA-API readback defect use software decoding', () => {
  for (const version of ['140.0', '153.3.0', '155.0', '156.0.1']) {
    assert.equal(directDecoderAcceleration(linuxFirefox(version)), 'prefer-software');
  }
});

test('fixed Linux Firefox versions retain hardware decoding', () => {
  for (const version of ['157.0', '157.0.1', '158.0', '200.0']) {
    assert.equal(directDecoderAcceleration(linuxFirefox(version)), 'prefer-hardware');
  }
});

test('other browser platforms and engines retain hardware decoding', () => {
  for (const userAgent of [
    linuxFirefox('156.0').replace('X11; Linux x86_64', 'Windows NT 10.0; Win64; x64'),
    linuxFirefox('156.0').replace('X11; Linux x86_64', 'Macintosh; Intel Mac OS X 14.0'),
    linuxFirefox('156.0').replace('X11; Linux x86_64', 'Android 16; Linux armv8l'),
    'Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 Chrome/156.0.0.0 Safari/537.36',
    'Mozilla/5.0 (iPhone; CPU iPhone OS 19_0 like Mac OS X) FxiOS/156.0',
    '',
    'Linux Firefox/unknown'
  ]) {
    assert.equal(directDecoderAcceleration(userAgent), 'prefer-hardware');
  }
});
