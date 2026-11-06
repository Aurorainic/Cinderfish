// SPDX-License-Identifier: AGPL-3.0-only
// Port of the theme table in firefish@02c31e0:packages/client/src/scripts/theme.ts
// (`getBuiltinThemes`) for the palettes Cinderfish ships.
//
// Built-in themes come from two sources: the two hand-ported defaults
// (`l-rosepinedawn` / `d-rosepine`) and the 26 upstream palettes loaded from
// their json5 sources by `palettes.ts`. Installed themes are looked up through
// `installed.ts` after the built-ins, so a user theme can never shadow one.

import { darkTheme } from "./dark";
import { lightTheme } from "./light";
import { rosePineDawnTheme } from "./builtin/l-rosepinedawn";
import { rosePineTheme } from "./builtin/d-rosepine";
import { findInstalledTheme } from "./installed";
import { paletteThemes } from "./palettes";
import type { Theme, ThemeMode } from "./types";

/**
 * The token tables a concrete theme is layered on. Firefish calls these
 * `_light` / `_dark` and never lists them in the picker: they exist so a theme
 * only has to override the props it cares about (see `resolveTheme`).
 */
export const baseThemes: Readonly<Record<ThemeMode, Theme>> = {
  light: lightTheme,
  dark: darkTheme,
};

function dedupeById(themes: Theme[]): Theme[] {
  const seen = new Set<string>();
  const out: Theme[] = [];
  for (const theme of themes) {
    if (seen.has(theme.id)) continue;
    seen.add(theme.id);
    out.push(theme);
  }
  return out;
}

/** Selectable built-in themes, in picker order (defaults first). */
export const builtinThemes: Theme[] = dedupeById([
  rosePineDawnTheme,
  rosePineTheme,
  ...paletteThemes,
]);

/**
 * Firefish's shipped defaults: Rosé Pine Dawn for light mode, Rosé Pine for
 * dark mode (see `ColdDeviceStorage.default` in Firefish's `store.ts`).
 */
export const defaultThemeIds: Readonly<Record<ThemeMode, string>> = {
  light: rosePineDawnTheme.id,
  dark: rosePineTheme.id,
};

/**
 * Theme lookup by id. The mode names `light` / `dark` are aliases for the
 * default theme of that mode so Firefish's `darkMode` style of caller
 * (`getTheme(resolvedMode)`) keeps working while real theme ids stay stable
 * slugs.
 */
export const builtinThemeMap: Record<string, Theme> = Object.fromEntries(
  builtinThemes.map((theme) => [theme.id, theme]),
);
builtinThemeMap[lightTheme.id] = rosePineDawnTheme;
builtinThemeMap[darkTheme.id] = rosePineTheme;

export function getTheme(idOrMode: string): Theme | undefined {
  return builtinThemeMap[idOrMode] ?? findInstalledTheme(idOrMode);
}

/** Name used by the ported theme controllers; same lookup as `getTheme`. */
export function findTheme(idOrMode: string): Theme | undefined {
  return getTheme(idOrMode);
}

export function getBuiltinThemes(): Theme[] {
  return builtinThemes;
}

/** Default palette for a resolved mode; never undefined. */
export function getDefaultTheme(mode: ThemeMode): Theme {
  return mode === "dark" ? rosePineTheme : rosePineDawnTheme;
}
