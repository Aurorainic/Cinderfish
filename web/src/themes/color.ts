// SPDX-License-Identifier: AGPL-3.0-only
// Minimal colour engine covering exactly the value language Firefish themes
// use. Firefish depends on tinycolor2; porting it wholesale would add ~30 KB
// for six operations, so this file reimplements the same operations with
// tinycolor-compatible output (`rgb(r, g, b)` / `rgba(r, g, b, a)`).
//
// Fidelity notes (verified against tinycolor2@1.6.0, see PORTING-NOTES):
//  - HSL uses tinycolor's representation: hue in degrees, saturation and
//    lightness in 0-1.
//  - Channels stay fractional until `formatColor`; rounding mid-chain drifts
//    by one step per derivation.
//  - tinycolor quantizes HSL input to two decimals of a percent by truncation
//    (`parseInt(n * 100) / 100`), which `quantizePercent` reproduces.

export interface Rgba {
	r: number;
	g: number;
	b: number;
	a: number;
}

/** tinycolor's HSL representation: hue in degrees, saturation/lightness 0-1. */
export interface Hsla {
	h: number;
	s: number;
	l: number;
	a: number;
}

const clamp = (value: number, min: number, max: number): number =>
	Math.min(max, Math.max(min, value));

function quantizePercent(value: number): number {
	return Math.floor(value * 10000) / 10000;
}

function hueToRgb(p: number, q: number, t: number): number {
	let tt = t;
	if (tt < 0) tt += 1;
	if (tt > 1) tt -= 1;
	if (tt < 1 / 6) return p + (q - p) * 6 * tt;
	if (tt < 1 / 2) return q;
	if (tt < 2 / 3) return p + (q - p) * (2 / 3 - tt) * 6;
	return p;
}

export function rgbaToHsla({ r, g, b, a }: Rgba): Hsla {
	const rn = r / 255;
	const gn = g / 255;
	const bn = b / 255;
	const max = Math.max(rn, gn, bn);
	const min = Math.min(rn, gn, bn);
	const l = (max + min) / 2;
	let h = 0;
	let s = 0;

	if (max !== min) {
		const d = max - min;
		s = l > 0.5 ? d / (2 - max - min) : d / (max + min);
		if (max === rn) h = (gn - bn) / d + (gn < bn ? 6 : 0);
		else if (max === gn) h = (bn - rn) / d + 2;
		else h = (rn - gn) / d + 4;
		h /= 6;
	}

	return { h: h * 360, s, l, a };
}

export function hslaToRgba({ h, s, l, a }: Hsla): Rgba {
	const hn = (((h % 360) + 360) % 360) / 360;
	const sn = quantizePercent(clamp(s, 0, 1));
	const ln = quantizePercent(clamp(l, 0, 1));

	if (sn === 0) {
		const v = ln * 255;
		return { r: v, g: v, b: v, a };
	}

	const q = ln < 0.5 ? ln * (1 + sn) : ln + sn - ln * sn;
	const p = 2 * ln - q;

	return {
		r: hueToRgb(p, q, hn + 1 / 3) * 255,
		g: hueToRgb(p, q, hn) * 255,
		b: hueToRgb(p, q, hn - 1 / 3) * 255,
		a,
	};
}

function parseHex(value: string): Rgba | null {
	const hex = value.slice(1);
	if (!/^[0-9a-fA-F]+$/.test(hex)) return null;
	if (hex.length === 3 || hex.length === 4) {
		const [r, g, b, a] = hex.split("").map((ch) => parseInt(ch + ch, 16));
		return { r, g, b, a: a == null ? 1 : a / 255 };
	}
	if (hex.length === 6 || hex.length === 8) {
		const r = parseInt(hex.slice(0, 2), 16);
		const g = parseInt(hex.slice(2, 4), 16);
		const b = parseInt(hex.slice(4, 6), 16);
		const a = hex.length === 8 ? parseInt(hex.slice(6, 8), 16) / 255 : 1;
		return { r, g, b, a };
	}
	return null;
}

function parseNumber(value: string, percentBase: number): number {
	return value.trim().endsWith("%")
		? (parseFloat(value) / 100) * percentBase
		: parseFloat(value);
}

export function parseColor(value: string): Rgba | null {
	const input = value.trim().toLowerCase();

	if (input === "transparent") return { r: 0, g: 0, b: 0, a: 0 };
	if (input === "white") return { r: 255, g: 255, b: 255, a: 1 };
	if (input === "black") return { r: 0, g: 0, b: 0, a: 1 };

	if (input.startsWith("#")) return parseHex(input);

	const rgb = /^rgba?\(([^)]+)\)$/.exec(input);
	if (rgb) {
		const parts = rgb[1].split(",").map((part) => part.trim());
		if (parts.length < 3) return null;
		return {
			r: clamp(parseNumber(parts[0], 255), 0, 255),
			g: clamp(parseNumber(parts[1], 255), 0, 255),
			b: clamp(parseNumber(parts[2], 255), 0, 255),
			a: parts[3] == null ? 1 : clamp(parseFloat(parts[3]), 0, 1),
		};
	}

	const hsl = /^hsla?\(([^)]+)\)$/.exec(input);
	if (hsl) {
		const parts = hsl[1].split(",").map((part) => part.trim());
		if (parts.length < 3) return null;
		return hslaToRgba({
			h: parseFloat(parts[0]),
			s: clamp(parseFloat(parts[1]), 0, 100) / 100,
			l: clamp(parseFloat(parts[2]), 0, 100) / 100,
			a: parts[3] == null ? 1 : clamp(parseFloat(parts[3]), 0, 1),
		});
	}

	return null;
}

export function formatColor({ r, g, b, a }: Rgba): string {
	const rr = Math.round(clamp(r, 0, 255));
	const gg = Math.round(clamp(g, 0, 255));
	const bb = Math.round(clamp(b, 0, 255));
	if (a >= 1) return `rgb(${rr}, ${gg}, ${bb})`;
	const alpha = Math.round(clamp(a, 0, 1) * 100) / 100;
	return `rgba(${rr}, ${gg}, ${bb}, ${alpha})`;
}

function withHsla(color: Rgba, transform: (hsla: Hsla) => Hsla): Rgba {
	return hslaToRgba(transform(rgbaToHsla(color)));
}

export const darken = (color: Rgba, amount: number): Rgba =>
	withHsla(color, (h) => ({ ...h, l: clamp(h.l - amount / 100, 0, 1) }));

export const lighten = (color: Rgba, amount: number): Rgba =>
	withHsla(color, (h) => ({ ...h, l: clamp(h.l + amount / 100, 0, 1) }));

export const saturate = (color: Rgba, amount: number): Rgba =>
	withHsla(color, (h) => ({ ...h, s: clamp(h.s + amount / 100, 0, 1) }));

export const spin = (color: Rgba, amount: number): Rgba =>
	withHsla(color, (h) => ({ ...h, h: (h.h + amount) % 360 }));

export const setAlpha = (color: Rgba, alpha: number): Rgba => ({
	...color,
	a: clamp(alpha, 0, 1),
});
