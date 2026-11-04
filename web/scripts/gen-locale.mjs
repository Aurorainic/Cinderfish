#!/usr/bin/env node
// Build-time locale generator.
//
// Firefish ships locales as YAML (`locales/<code>.yml`): top-level keys plus a
// handful of one-level nested maps (`_ago:`, `_time:`, `_themeNames:` ...).
// The client imports JS modules, so this script flattens the nested mapping
// into dot-separated keys and writes one JSON module per locale:
//
//   _ago:
//     justNow: "刚刚"
//   => { "_ago.justNow": "刚刚" }
//
// Usage (from the repository root):
//   node web/scripts/gen-locale.mjs
//   node web/scripts/gen-locale.mjs <sourceDir> [outDir]
//
// Requires the `yaml` package (NOT added to package.json on purpose: the
// generated JSON is committed, so only a locale update needs the parser; see
// web/src/i18n/PORTING-NOTES.md).
//
// `yaml` parses YAML 1.2, where `yes` / `no` are strings. That matters: the
// locale files use them as keys (`yes: "是"`), and YAML 1.1 parsers (PyYAML's
// default loader) turn them into booleans and silently drop the keys.

import { readFileSync, writeFileSync, mkdirSync, existsSync } from "node:fs";
import { dirname, join, resolve } from "node:path";
import { fileURLToPath } from "node:url";
import { parse } from "yaml";

const here = dirname(fileURLToPath(import.meta.url));
const projectRoot = resolve(here, "..");

const candidates = [
	"../tmp/refcheck/firefish-jclaudan-mirror/locales",
	"../tmp/refcheck/firefish/locales",
	"../tmp/refcheck/firefish-full/locales",
];
const defaultSource = candidates
	.map((candidate) => resolve(projectRoot, candidate))
	.find((candidate) => existsSync(candidate));

const sourceDir = process.argv[2] ? resolve(process.argv[2]) : defaultSource;
const outDir = process.argv[3]
	? resolve(process.argv[3])
	: join(projectRoot, "src/i18n/locales");

if (sourceDir == null || !existsSync(sourceDir)) {
	console.error("Locale source directory not found. Pass it as the first argument.");
	process.exit(1);
}

const TARGETS = ["zh-CN", "ja-JP"];

/** @param {Record<string, unknown>} node @param {string} prefix */
function flatten(node, prefix = "") {
	/** @type {Record<string, string>} */
	const out = {};
	for (const [key, value] of Object.entries(node)) {
		const path = prefix === "" ? key : `${prefix}.${key}`;
		if (value !== null && typeof value === "object" && !Array.isArray(value)) {
			Object.assign(out, flatten(/** @type {Record<string, unknown>} */ (value), path));
		} else {
			out[path] = value == null ? "" : String(value);
		}
	}
	return out;
}

mkdirSync(outDir, { recursive: true });

for (const code of TARGETS) {
	const source = join(sourceDir, `${code}.yml`);
	if (!existsSync(source)) {
		console.warn(`skip ${code}: ${source} not found`);
		continue;
	}
	const parsed = parse(readFileSync(source, "utf8"));
	const flat = flatten(parsed ?? {});
	writeFileSync(
		join(outDir, `${code}.json`),
		`${JSON.stringify(flat, null, "\t")}\n`,
		"utf8",
	);
	console.log(`${code}: ${Object.keys(flat).length} keys -> ${join(outDir, `${code}.json`)}`);
}
