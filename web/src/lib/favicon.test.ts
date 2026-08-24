import assert from 'node:assert/strict';
import { describe, it } from 'node:test';

import {
  applyFavicon,
  FAVICON_ENDPOINT,
  faviconErrorKey,
  faviconHref,
  faviconSourceKey,
  isFaviconUrl
} from './favicon.ts';

// A stand-in for the handful of Element operations applyFavicon uses, so the
// behaviour is testable without a DOM.
type FakeLink = {
  attributes: Record<string, string>;
  removed: number;
  setAttribute(name: string, value: string): void;
  removeAttribute(name: string): void;
  remove(): void;
};

function fakeLink(attributes: Record<string, string> = {}): FakeLink {
  const link: FakeLink = {
    attributes: { ...attributes },
    removed: 0,
    setAttribute(name: string, value: string) {
      link.attributes[name] = value;
    },
    removeAttribute(name: string) {
      delete link.attributes[name];
    },
    remove() {
      link.removed += 1;
    }
  };
  return link;
}

function fakeDocument(links: FakeLink[]) {
  const appended: FakeLink[] = [];
  const created: FakeLink[] = [];

  const doc = {
    head: {
      appendChild(node: FakeLink) {
        appended.push(node);
      }
    },
    querySelectorAll() {
      return links;
    },
    createElement() {
      const link = fakeLink();
      created.push(link);
      return link;
    }
  } as unknown as Document;

  return { doc, appended, created };
}

describe('faviconHref', () => {
  it('serves the bare endpoint when there is no version yet', () => {
    assert.equal(faviconHref(), FAVICON_ENDPOINT);
    assert.equal(faviconHref(''), FAVICON_ENDPOINT);
  });

  it('carries the version so a changed icon is a changed url', () => {
    assert.equal(faviconHref('abc123'), `${FAVICON_ENDPOINT}?v=abc123`);
    assert.notEqual(faviconHref('abc123'), faviconHref('def456'));
  });

  it('escapes the version rather than pasting it into the query', () => {
    assert.equal(faviconHref('a&b=c'), `${FAVICON_ENDPOINT}?v=a%26b%3Dc`);
  });
});

describe('applyFavicon', () => {
  it('repoints the existing link and re-inserts it so the tab re-reads it', () => {
    const existing = fakeLink({ rel: 'icon', type: 'image/svg+xml', href: FAVICON_ENDPOINT });
    const { doc, appended } = fakeDocument([existing]);

    const href = applyFavicon(doc, 'deadbeef');

    assert.equal(href, `${FAVICON_ENDPOINT}?v=deadbeef`);
    assert.equal(existing.attributes.href, href);
    assert.equal(existing.removed, 1, 'the link must be detached before it is re-appended');
    assert.deepEqual(appended, [existing]);
  });

  it('drops a stale type attribute, since the response declares the type', () => {
    const existing = fakeLink({ rel: 'icon', type: 'image/svg+xml' });
    const { doc } = fakeDocument([existing]);

    applyFavicon(doc, 'v1');

    assert.equal(existing.attributes.type, undefined);
  });

  it('creates a link when the document has none', () => {
    const { doc, appended, created } = fakeDocument([]);

    applyFavicon(doc, 'v2');

    assert.equal(created.length, 1);
    assert.equal(created[0].attributes.rel, 'icon');
    assert.equal(created[0].attributes.href, `${FAVICON_ENDPOINT}?v=v2`);
    assert.deepEqual(appended, created);
  });

  it('updates every icon link, not just the first', () => {
    const first = fakeLink({ rel: 'icon' });
    const second = fakeLink({ rel: 'shortcut icon' });
    const { doc, appended } = fakeDocument([first, second]);

    applyFavicon(doc, 'v3');

    assert.equal(first.attributes.href, `${FAVICON_ENDPOINT}?v=v3`);
    assert.equal(second.attributes.href, `${FAVICON_ENDPOINT}?v=v3`);
    assert.equal(appended.length, 2);
  });
});

describe('faviconErrorKey', () => {
  it('maps each server code to its own message', () => {
    const codes = [-2, -3, -4, -5];
    const keys = codes.map(faviconErrorKey);

    assert.equal(new Set(keys).size, codes.length, 'codes must not collapse onto one message');
    for (const key of keys) {
      assert.match(key, /^settings\.appearance\.faviconErr/);
    }
  });

  it('falls back to a generic failure for anything unrecognised', () => {
    assert.equal(faviconErrorKey(-1), 'settings.appearance.faviconErrSave');
    assert.equal(faviconErrorKey(-99), 'settings.appearance.faviconErrSave');
  });
});

describe('faviconSourceKey', () => {
  it('names each source distinctly so the panel cannot claim the wrong one', () => {
    assert.equal(faviconSourceKey('custom'), 'settings.appearance.faviconCustom');
    assert.equal(faviconSourceKey('boot'), 'settings.appearance.faviconBoot');
    assert.equal(faviconSourceKey('default'), 'settings.appearance.faviconDefault');
    assert.equal(faviconSourceKey(''), 'settings.appearance.faviconDefault');
  });
});

describe('isFaviconUrl', () => {
  it('accepts http and https', () => {
    assert.equal(isFaviconUrl('http://example.com/icon.png'), true);
    assert.equal(isFaviconUrl('https://example.com/icon.svg'), true);
    assert.equal(isFaviconUrl('  https://example.com/icon.ico  '), true);
  });

  it('rejects anything the device cannot fetch', () => {
    assert.equal(isFaviconUrl(''), false);
    assert.equal(isFaviconUrl('   '), false);
    assert.equal(isFaviconUrl('example.com/icon.png'), false);
    assert.equal(isFaviconUrl('file:///etc/shadow'), false);
    assert.equal(isFaviconUrl('data:image/png;base64,AAAA'), false);
    assert.equal(isFaviconUrl('javascript:alert(1)'), false);
  });
});
