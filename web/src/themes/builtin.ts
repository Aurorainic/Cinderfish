// SPDX-License-Identifier: AGPL-3.0-only
// Port of the theme table in firefish@02c31e0:packages/client/src/scripts/theme.ts
// (`getBuiltinThemes`) for the palettes Cinderfish ships.

import { darkTheme } from "./dark";
import { lightTheme } from "./light";
import { rosePineDawnTheme } from "./builtin/l-rosepinedawn";
import { rosePineTheme } from "./builtin/d-rosepine";
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

/** Selectable built-in themes, in picker order. */
export const builtinThemes: Theme[] = [rosePineDawnTheme, rosePineTheme];

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
export const builtinThemeMap: Record<string, Theme> = {
  [rosePineDawnTheme.id]: rosePineDawnTheme,
  [rosePineTheme.id]: rosePineTheme,
  [lightTheme.id]: rosePineDawnTheme,
  [darkTheme.id]: rosePineTheme,
};

export function getTheme(idOrMode: string): Theme | undefined {
  return builtinThemeMap[idOrMode];
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
