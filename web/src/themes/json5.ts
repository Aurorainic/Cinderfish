// SPDX-License-Identifier: AGPL-3.0-only
// JSON5 subset parser for theme files.
//
// Firefish parses installed themes with the `json5` package
// (firefish@02c31e0:packages/client/src/pages/settings/theme.install.vue).
// Cinderfish does not ship that dependency: the built-in themes are pre-
// converted TypeScript modules, and this parser covers the syntax a theme
// actually uses (comments, unquoted keys, single quotes, trailing commas).
//
// Not supported on purpose: multi-line strings, Unicode identifiers.

import type { Theme } from "./types";

export type Json5Value =
	| string
	| number
	| boolean
	| null
	| Json5Value[]
	| { [key: string]: Json5Value };

export class Json5ParseError extends Error {
	constructor(
		message: string,
		readonly position: number,
	) {
		super(`${message} (at offset ${position})`);
		this.name = "Json5ParseError";
	}
}

const IDENTIFIER_START = /[A-Za-z_$]/;
const IDENTIFIER_PART = /[A-Za-z0-9_$]/;
const DIGIT = /[0-9]/;
const HEX_DIGIT = /[0-9a-fA-F]/;

function isIdentifierStart(char: string | undefined): boolean {
	return char !== undefined && IDENTIFIER_START.test(char);
}

export function parseJson5(text: string): Json5Value {
	let position = 0;

	function fail(message: string): never {
		throw new Json5ParseError(message, position);
	}

	function skipIgnorable(): void {
		for (;;) {
			const char = text[position];
			if (
				char === " " ||
				char === "\t" ||
				char === "\n" ||
				char === "\r" ||
				char === "\uFEFF"
			) {
				position += 1;
				continue;
			}
			if (char === "/" && text[position + 1] === "/") {
				position += 2;
				while (position < text.length && text[position] !== "\n") {
					position += 1;
				}
				continue;
			}
			if (char === "/" && text[position + 1] === "*") {
				const end = text.indexOf("*/", position + 2);
				if (end === -1) fail("Unterminated block comment");
				position = end + 2;
				continue;
			}
			return;
		}
	}

	function parseString(): string {
		const quote = text[position];
		position += 1;
		let result = "";
		for (;;) {
			const char = text[position];
			if (char === undefined) fail("Unterminated string");
			if (char === quote) {
				position += 1;
				return result;
			}
			if (char === "\\") {
				const escape = text[position + 1];
				position += 2;
				switch (escape) {
					case "n":
						result += "\n";
						break;
					case "r":
						result += "\r";
						break;
					case "t":
						result += "\t";
						break;
					case "b":
						result += "\b";
						break;
					case "f":
						result += "\f";
						break;
					case "u": {
						const hex = text.slice(position, position + 4);
						if (!/^[0-9a-fA-F]{4}$/.test(hex)) {
							fail("Invalid unicode escape");
						}
						result += String.fromCharCode(Number.parseInt(hex, 16));
						position += 4;
						break;
					}
					case undefined:
						fail("Unterminated escape sequence");
						break;
					default:
						result += escape;
						break;
				}
				continue;
			}
			result += char;
			position += 1;
		}
	}

	function parseIdentifier(): string {
		if (!isIdentifierStart(text[position])) {
			fail("Expected an identifier");
		}
		const start = position;
		while (
			position < text.length &&
			IDENTIFIER_PART.test(text[position] ?? "")
		) {
			position += 1;
		}
		return text.slice(start, position);
	}

	function parseNumber(): number {
		const start = position;
		if (text[position] === "+" || text[position] === "-") position += 1;

		if (text[position] === "0" && /[xX]/.test(text[position + 1] ?? "")) {
			position += 2;
			const hexStart = position;
			while (HEX_DIGIT.test(text[position] ?? "")) position += 1;
			if (hexStart === position) fail("Invalid hexadecimal number");
			const value = Number.parseInt(text.slice(hexStart, position), 16);
			return text[start] === "-" ? -value : value;
		}

		while (DIGIT.test(text[position] ?? "")) position += 1;
		if (text[position] === ".") {
			position += 1;
			while (DIGIT.test(text[position] ?? "")) position += 1;
		}
		if (text[position] === "e" || text[position] === "E") {
			position += 1;
			if (text[position] === "+" || text[position] === "-") position += 1;
			while (DIGIT.test(text[position] ?? "")) position += 1;
		}

		const raw = text.slice(start, position);
		const value = Number(raw);
		if (Number.isNaN(value)) fail(`Invalid number "${raw}"`);
		return value;
	}

	function parseArray(): Json5Value[] {
		position += 1; // [
		const result: Json5Value[] = [];
		for (;;) {
			skipIgnorable();
			if (text[position] === "]") {
				position += 1;
				return result;
			}
			result.push(parseValue());
			skipIgnorable();
			if (text[position] === ",") {
				position += 1;
				continue;
			}
			if (text[position] === "]") {
				position += 1;
				return result;
			}
			fail('Expected "," or "]"');
		}
	}

	function parseObject(): { [key: string]: Json5Value } {
		position += 1; // {
		const result: { [key: string]: Json5Value } = {};
		for (;;) {
			skipIgnorable();
			if (text[position] === "}") {
				position += 1;
				return result;
			}
			const key =
				text[position] === '"' || text[position] === "'"
					? parseString()
					: parseIdentifier();
			skipIgnorable();
			if (text[position] !== ":") fail('Expected ":"');
			position += 1;
			result[key] = parseValue();
			skipIgnorable();
			if (text[position] === ",") {
				position += 1;
				continue;
			}
			if (text[position] === "}") {
				position += 1;
				return result;
			}
			fail('Expected "," or "}"');
		}
	}

	function parseValue(): Json5Value {
		skipIgnorable();
		const char = text[position];
		if (char === "{") return parseObject();
		if (char === "[") return parseArray();
		if (char === '"' || char === "'") return parseString();
		if (
			char === "+" ||
			char === "-" ||
			char === "." ||
			DIGIT.test(char ?? "")
		) {
			return parseNumber();
		}
		if (isIdentifierStart(char)) {
			const word = parseIdentifier();
			if (word === "true") return true;
			if (word === "false") return false;
			if (word === "null") return null;
			if (word === "Infinity") return Number.POSITIVE_INFINITY;
			if (word === "NaN") return Number.NaN;
			return fail(`Unexpected identifier "${word}"`);
		}
		return fail(`Unexpected character "${char ?? "<end of input>"}"`);
	}

	skipIgnorable();
	const value = parseValue();
	skipIgnorable();
	if (position < text.length) fail("Unexpected trailing input");
	return value;
}

/**
 * Validates an arbitrary value as a theme. Stricter than Firefish's
 * `validateTheme` in one way (`author` is required because the `Theme` type
 * requires it) and more permissive in another (`base` may be given as `kind`,
 * which is how the built-in base themes are written).
 */
export function validateTheme(candidate: unknown): candidate is Theme {
	if (candidate === null || typeof candidate !== "object") return false;
	if (Array.isArray(candidate)) return false;

	const theme = candidate as Record<string, unknown>;
	if (typeof theme.id !== "string" || theme.id.length === 0) return false;
	if (typeof theme.name !== "string") return false;
	if (typeof theme.author !== "string") return false;
	if (theme.desc !== undefined && typeof theme.desc !== "string") return false;
	if (
		theme.base !== undefined &&
		theme.base !== "light" &&
		theme.base !== "dark"
	) {
		return false;
	}
	if (
		theme.kind !== undefined &&
		theme.kind !== "light" &&
		theme.kind !== "dark"
	) {
		return false;
	}
	if (theme.base === undefined && theme.kind === undefined) return false;

	const props = theme.props;
	if (props === null || typeof props !== "object" || Array.isArray(props)) {
		return false;
	}
	for (const value of Object.values(props)) {
		if (typeof value !== "string") return false;
	}
	return true;
}

/** Parses a theme pasted into the theme-install dialog; throws when invalid. */
export function parseThemeJson5(text: string): Theme {
	const value = parseJson5(text);
	if (!validateTheme(value)) {
		throw new Json5ParseError("Parsed value is not a theme", 0);
	}
	return value;
}
