// SPDX-License-Identifier: AGPL-3.0-only
// Firefish's remaining built-in palettes, loaded from their upstream json5
// sources (firefish@02c31e0:packages/client/src/themes/).
//
// The sources carry Firefish's UUID `id` values, which the client cannot keep:
// like the two hand-ported defaults in `builtin.ts`, every palette is
// re-identified by its file slug so the id is stable and storable.

import { parseJson5, validateTheme } from "./json5";
import type { Theme } from "./types";

import dAstro from "./builtin/d-astro.json5?raw";
import dBotanical from "./builtin/d-botanical.json5?raw";
import dCatppuccinFrappe from "./builtin/d-catppuccin-frappe.json5?raw";
import dCatppuccinMocha from "./builtin/d-catppuccin-mocha.json5?raw";
import dCherry from "./builtin/d-cherry.json5?raw";
import dDark from "./builtin/d-dark.json5?raw";
import dFuture from "./builtin/d-future.json5?raw";
import dGreenLime from "./builtin/d-green-lime.json5?raw";
import dGreenOrange from "./builtin/d-green-orange.json5?raw";
import dGruvbox from "./builtin/d-gruvbox.json5?raw";
import dIce from "./builtin/d-ice.json5?raw";
import dNord from "./builtin/d-nord.json5?raw";
import dPersimmon from "./builtin/d-persimmon.json5?raw";
import dRosepineMoon from "./builtin/d-rosepinemoon.json5?raw";
import dU0 from "./builtin/d-u0.json5?raw";
import lApricot from "./builtin/l-apricot.json5?raw";
import lCatppuccinLatte from "./builtin/l-catppuccin-latte.json5?raw";
import lCherry from "./builtin/l-cherry.json5?raw";
import lCoffee from "./builtin/l-coffee.json5?raw";
import lGruvbox from "./builtin/l-gruvbox.json5?raw";
import lLight from "./builtin/l-light.json5?raw";
import lNord from "./builtin/l-nord.json5?raw";
import lRainy from "./builtin/l-rainy.json5?raw";
import lSushi from "./builtin/l-sushi.json5?raw";
import lU0 from "./builtin/l-u0.json5?raw";
import lVivid from "./builtin/l-vivid.json5?raw";

/**
 * Parses one palette and swaps its upstream UUID for the file slug. Bad data
 * throws instead of being skipped: a missing palette would otherwise only show
 * up as a hole in the picker.
 */
function palette(slug: string, raw: string): Theme {
	const parsed = parseJson5(raw);
	if (parsed === null || typeof parsed !== "object" || Array.isArray(parsed)) {
		throw new Error(`palette ${slug}: invalid theme`);
	}
	const theme = {
		...(parsed as Record<string, unknown>),
		id: slug,
		author: (parsed as Record<string, unknown>).author ?? "",
	} as unknown as Theme;
	if (!validateTheme(theme)) {
		throw new Error(`palette ${slug}: invalid theme`);
	}
	return theme;
}

const lightPalettes: Theme[] = [
	palette("l-apricot", lApricot),
	palette("l-catppuccin-latte", lCatppuccinLatte),
	palette("l-cherry", lCherry),
	palette("l-coffee", lCoffee),
	palette("l-gruvbox", lGruvbox),
	palette("l-light", lLight),
	palette("l-nord", lNord),
	palette("l-rainy", lRainy),
	palette("l-sushi", lSushi),
	palette("l-u0", lU0),
	palette("l-vivid", lVivid),
];

const darkPalettes: Theme[] = [
	palette("d-astro", dAstro),
	palette("d-botanical", dBotanical),
	palette("d-catppuccin-frappe", dCatppuccinFrappe),
	palette("d-catppuccin-mocha", dCatppuccinMocha),
	palette("d-cherry", dCherry),
	palette("d-dark", dDark),
	palette("d-future", dFuture),
	palette("d-green-lime", dGreenLime),
	palette("d-green-orange", dGreenOrange),
	palette("d-gruvbox", dGruvbox),
	palette("d-ice", dIce),
	palette("d-nord", dNord),
	palette("d-persimmon", dPersimmon),
	palette("d-rosepinemoon", dRosepineMoon),
	palette("d-u0", dU0),
];

function byId(a: Theme, b: Theme): number {
	if (a.id === b.id) return 0;
	return a.id < b.id ? -1 : 1;
}

/** Palettes in picker order: light first, then dark, each sorted by id. */
export const paletteThemes: Theme[] = [
	...lightPalettes.sort(byId),
	...darkPalettes.sort(byId),
];
