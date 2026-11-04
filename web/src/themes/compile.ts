// SPDX-License-Identifier: AGPL-3.0-only
// Compiler for Firefish's theme value language, ported from
// firefish@02c31e0:packages/client/src/scripts/theme.ts (`compile()`).

import {
	darken,
	formatColor,
	lighten,
	parseColor,
	saturate,
	setAlpha,
	spin,
	type Rgba,
} from "./color";
import type { CompiledTheme, Theme } from "./types";
import { baseThemes } from "./builtin";

const TRANSPARENT_BLACK: Rgba = { r: 0, g: 0, b: 0, a: 1 };

/** Merge a theme over its base theme, exactly like Firefish's `applyTheme`. */
export function resolveTheme(theme: Theme): Theme {
	if (theme.base == null) return theme;
	const base = baseThemes[theme.base];
	if (base == null) return theme;
	return {
		...theme,
		props: { ...base.props, ...theme.props },
	};
}

function applyFunction(name: string, amount: number, color: Rgba): Rgba {
	switch (name) {
		case "darken":
			return darken(color, amount);
		case "lighten":
			return lighten(color, amount);
		case "alpha":
			return setAlpha(color, amount);
		case "hue":
			return spin(color, amount);
		case "saturate":
			return saturate(color, amount);
		default:
			// Firefish silently ignores unknown functions; keep the operand.
			return color;
	}
}

/**
 * Compile Firefish's theme value language into concrete CSS values.
 *
 * Values are resolved as floating-point colours and formatted only at the end:
 * tokens chain (`panel` → `panelHighlight`), and rounding mid-chain drifts by
 * one step per link compared with Firefish's tinycolor2 output.
 */
export function compileTheme(theme: Theme): CompiledTheme {
	const props = resolveTheme(theme).props;
	const colors = new Map<string, Rgba | null>();
	const resolving = new Set<string>();

	function resolveValue(raw: string): Rgba | null {
		// A leading quote is a literal CSS fragment, not a colour.
		if (raw.startsWith('"')) return null;

		// Reference to another prop.
		if (raw.startsWith("@")) return colorOf(raw.slice(1));
		// Reference to a `$`-prefixed constant (the `$` is part of the key).
		if (raw.startsWith("$")) return colorOf(raw);

		if (raw.startsWith(":")) {
			const parts = raw.slice(1).split("<");
			const name = parts[0] ?? "";
			const amount = Number.parseFloat(parts[1] ?? "");
			const color = resolveValue(parts.slice(2).join("<"));
			if (color == null || Number.isNaN(amount)) return null;
			return applyFunction(name, amount, color);
		}

		return parseColor(raw);
	}

	function colorOf(key: string): Rgba | null {
		if (colors.has(key)) return colors.get(key) ?? null;

		const raw = props[key];
		if (raw == null) {
			throw new Error(`[theme] unknown theme property "${key}"`);
		}
		if (resolving.has(key)) {
			throw new Error(`[theme] circular theme reference at "${key}"`);
		}

		resolving.add(key);
		const value = resolveValue(raw);
		resolving.delete(key);
		colors.set(key, value);
		return value;
	}

	const out: CompiledTheme = {};
	for (const [key, raw] of Object.entries(props)) {
		if (key.startsWith("$")) continue;
		out[key] = raw.startsWith('"')
			? raw.replace(/^"\s*/, "")
			: formatColor(colorOf(key) ?? TRANSPARENT_BLACK);
	}
	return out;
}

/** Names of the CSS custom properties produced by `compileTheme`, with `--`. */
export function themeVariableNames(theme: Theme): string[] {
	return Object.keys(theme.props)
		.filter((key) => !key.startsWith("$"))
		.map((key) => `--${key}`);
}
