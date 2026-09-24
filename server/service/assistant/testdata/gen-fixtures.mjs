// Regenerate fixtures.json by running the chaice extension's own llm.js.
// Usage (from the NanoKVM root):
//   node server/service/assistant/testdata/gen-fixtures.mjs ../../chaice/chaice/chaice-extension/llm.js
import { readFileSync, writeFileSync } from "node:fs";
import { dirname, join } from "node:path";
import { fileURLToPath } from "node:url";
import vm from "node:vm";

const here = dirname(fileURLToPath(import.meta.url));
const ctx = { window: {} };
vm.createContext(ctx);
vm.runInContext(readFileSync(process.argv[2], "utf8"), ctx);
const L = ctx.window.ChaiceLLM;

const cases = JSON.parse(readFileSync(join(here, "cases.json"), "utf8"));
const out = {};
for (const c of cases) {
  const turns = c.turns.map((t) => ({
    role: t.role,
    text: t.text,
    images: t.images || [],
    attachments: (t.attachments || []).map((a) => {
      const cls = L.classifyFile(a.name);
      const att = { name: a.name, kind: cls.kind, mime: cls.mime, audioFormat: cls.audioFormat, b64: a.b64 };
      if (cls.kind === "text") att.text = a.text;
      return att;
    }),
  }));
  const req = L.buildConversation({ provider: c.provider, settings: c.settings, turns, thinking: c.thinking });
  out[c.name] = { url: req.url, headers: req.headers, body: JSON.parse(req.body) };
}
writeFileSync(join(here, "fixtures.json"), JSON.stringify(out, null, 2) + "\n");
console.log(`wrote ${Object.keys(out).length} fixtures`);
