import assert from 'node:assert/strict';
import { readdirSync } from 'node:fs';
import path from 'node:path';
import test from 'node:test';
import { fileURLToPath, pathToFileURL } from 'node:url';

// Every string the panel asks for by key must exist in every locale: i18next
// falls back to the raw key, so a locale missing one shows
// "settings.exit.commands.scriptGated" in the UI. The placeholders must match
// too, or the interpolation silently prints nothing.

const dir = fileURLToPath(new URL('../../../../../i18n/locales/', import.meta.url));

type Leaf = { key: string; placeholders: string[] };

function leaves(value: unknown, prefix = ''): Leaf[] {
  if (typeof value === 'string') {
    const placeholders = [...value.matchAll(/\{\{\s*(\w+)\s*\}\}/g)].map((m) => m[1]).sort();
    return [{ key: prefix, placeholders }];
  }
  if (value && typeof value === 'object') {
    return Object.entries(value).flatMap(([name, child]) =>
      leaves(child, prefix ? `${prefix}.${name}` : name)
    );
  }
  return [];
}

async function exitStrings(file: string): Promise<unknown> {
  const mod = await import(pathToFileURL(path.join(dir, file)).href);
  return mod.default?.translation?.settings?.exit;
}

test('every locale carries the exit strings of en, with the same placeholders', async () => {
  const files = readdirSync(dir).filter((file) => file.endsWith('.ts'));
  assert.ok(files.length >= 24, `expected the 24 locales, found ${files.length}`);

  const en = leaves(await exitStrings('en.ts')).sort((a, b) => a.key.localeCompare(b.key));
  assert.ok(en.length > 0, 'en has no settings.exit block');

  for (const file of files) {
    const exit = await exitStrings(file);
    assert.ok(exit, `${file} has no settings.exit block`);

    const found = leaves(exit).sort((a, b) => a.key.localeCompare(b.key));
    assert.deepEqual(
      found.map((leaf) => leaf.key),
      en.map((leaf) => leaf.key),
      `${file}: exit keys differ from en`
    );
    for (let i = 0; i < en.length; i++) {
      assert.deepEqual(
        found[i].placeholders,
        en[i].placeholders,
        `${file}: placeholders of ${en[i].key} differ from en`
      );
    }
  }
});
