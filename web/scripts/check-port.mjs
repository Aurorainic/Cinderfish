#!/usr/bin/env node
// Porting gate: scans web/src for the conventions the wave-2 contract fixes.
// Usage: node web/scripts/check-port.mjs [--quiet]
// Exit code 1 when a hard rule is violated.
import { readdirSync, readFileSync, statSync } from "node:fs";
import { join, relative, resolve, dirname } from "node:path";
import { fileURLToPath } from "node:url";

const here = dirname(fileURLToPath(import.meta.url));
const src = resolve(here, "..", "src");
const dict = JSON.parse(readFileSync(join(src, "i18n/locales/zh-CN.json"), "utf8"));

const files = [];
function walk(dir) {
  for (const entry of readdirSync(dir)) {
    const p = join(dir, entry);
    const st = statSync(p);
    if (st.isDirectory()) walk(p);
    else if (/\.(ts|vue|mts)$/.test(entry)) files.push(p);
  }
}
walk(src);

const problems = { forbiddenDeps: [], i18nStyle: [], reactivity: [], directFetch: [], tokenRead: [], missingKeys: [], missingHeader: [] };
const allowFetch = new Set(["api/client.ts", "account/transport.ts"]);
const allowTokenRead = [/(^|\/)api\/token\.ts$/, /(^|\/)account\//, /(^|\/)store\//];
const allowedExternal = new Set(["vue", "vue-router", "mfm-js", "@phosphor-icons/web"]);

for (const file of files) {
  const rel = relative(src, file).replace(/\\/g, "/");
  const text = readFileSync(file, "utf8");
  const lines = text.split("\n");

  for (const line of lines) {
    const importMatch = line.match(/from\s+"([^".][^"]*)"/);
    if (importMatch) {
      const spec = importMatch[1];
      const pkg = spec.startsWith("@") ? spec.split("/").slice(0, 2).join("/") : spec.split("/")[0];
      if (!allowedExternal.has(pkg) && !pkg.startsWith("@/") && !pkg.startsWith(".")) {
        problems.forbiddenDeps.push(rel + ": " + spec);
      }
    }
  }
  if (/\bi18n\.ts\b/.test(text) && rel !== "i18n/index.ts") problems.i18nStyle.push(rel);
  if (/\$(ref|computed|shallowRef)\(/.test(text)) problems.reactivity.push(rel);
  if (/\bfetch\s*\(/.test(text) && !allowFetch.has(rel)) problems.directFetch.push(rel);
  const tokenRead = /localStorage\.getItem\(\s*["'](?:account|cinderfish\.token)["']/.test(text);
  if (tokenRead && !allowTokenRead.some((re) => re.test(rel))) problems.tokenRead.push(rel);

  const declared = text.match(/Ported from firefish@/);
  if (!declared && /^(components\/(Mk|global)|scripts|filters|directives)\//.test(rel) && rel !== "scripts/index.ts") {
    problems.missingHeader.push(rel);
  }

  const calls = text.matchAll(/\bt\(\s*"([^"]+)"/g);
  for (const m of calls) {
    const key = m[1];
    if (dict[key] === undefined && rel !== "i18n/index.ts") problems.missingKeys.push(rel + ": " + key);
  }
}


// ---- rule: every <Mk...>/<Cf...> tag used in a template must be imported,
// registered globally, or a known auto-registered global. vue-tsc catches a
// missing import only when the component lives under components/global/; a page
// that renders <MkNoteDetailed> without importing it type-checks cleanly and
// silently renders an empty element (this happened once in wave 2).
import { readdirSync as rd } from "node:fs";
const globals = new Set(
  rd(join(src, "components/global"))
    .filter((n) => n.endsWith(".vue"))
    .map((n) => n.replace(/\.vue$/, "")),
);
globals.add("Mfm");
const htmlTags = new Set([
  "a","abbr","audio","b","blockquote","br","button","canvas","code","col","colgroup","details","div","dl","dt","dd",
  "em","fieldset","figure","figcaption","footer","form","h1","h2","h3","h4","h5","h6","header","hr","i","iframe",
  "img","input","label","legend","li","main","mark","nav","ol","option","optgroup","p","pre","progress","q","s",
  "section","select","small","source","span","strong","sub","summary","sup","table","tbody","td","textarea","tfoot",
  "th","thead","tr","u","ul","video","wbr","svg","path","circle","rect","g","defs","linearGradient","stop","template","slot","component","transition","keep-alive","router-view","router-link","teleport","suspense",
]);
const unresolved = [];
for (const file of files) {
  if (!file.endsWith(".vue")) continue;
  const rel = relative(src, file).replace(/\\/g, "/");
  const text = readFileSync(file, "utf8");
  const template = text.match(/<template>([\s\S]*)<\/template>/);
  if (!template) continue;
  const used = new Set();
  const selfName = (text.match(/defineOptions\(\s*\{\s*name:\s*"([^"]+)"/) ?? [])[1];
  for (const m of template[1].matchAll(/<([A-Z][A-Za-z0-9]*)/g)) used.add(m[1]);
  for (const tag of used) {
    if (globals.has(tag)) continue;
    if (["RouterView", "RouterLink"].includes(tag)) continue;
    if (selfName != null && selfName.endsWith(tag)) continue; // recursive self-reference
    const basename = rel.split("/").pop().replace(/\.vue$/, "");
    if (basename === tag) continue; // <script setup>'s implicit self-reference by filename
    const imported = new RegExp("import\\s+" + tag + "\\b").test(text) ||
      new RegExp("\\b" + tag + "\\s*,").test(text) ||
      new RegExp("\\b" + tag + "\\s*\\}").test(text);
    const declared = /components\s*:/.test(text) || /defineOptions\s*\(/.test(text);
    if (!imported && !declared) unresolved.push(rel + ": <" + tag + ">");
  }
}

const report = {
  files: files.length,
  forbiddenDeps: [...new Set(problems.forbiddenDeps)],
  i18nStyle: problems.i18nStyle,
  reactivity: problems.reactivity,
  directFetch: problems.directFetch,
  tokenRead: problems.tokenRead,
  missingKeys: [...new Set(problems.missingKeys)],
  missingHeader: problems.missingHeader,
};
const _hard0 = report.forbiddenDeps.length + report.i18nStyle.length + report.reactivity.length + report.directFetch.length + report.tokenRead.length;
report.unresolvedComponents = [...new Set(unresolved)];
report.hardTotal = report.forbiddenDeps.length + report.i18nStyle.length + report.reactivity.length + report.directFetch.length + report.tokenRead.length + report.unresolvedComponents.length;
const hard = report.hardTotal;
console.log(JSON.stringify(report, null, 2));
process.exit(hard > 0 ? 1 : 0);