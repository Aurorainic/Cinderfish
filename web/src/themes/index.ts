// SPDX-License-Identifier: AGPL-3.0-only
// Public surface of the theme engine. The reactive light/dark/auto controller
// (`useTheme()`) lives in `web/src/composables/use-theme.ts` and builds on the
// pieces exported here.

export { applyTheme, prefersDarkMode, type ApplyThemeOptions } from "./apply";
export {
	baseThemes,
	builtinThemeMap,
	builtinThemes,
	defaultThemeIds,
	findTheme,
	getBuiltinThemes,
	getDefaultTheme,
	getTheme,
} from "./builtin";
export { compileTheme, resolveTheme, themeVariableNames } from "./compile";
export {
	darken,
	formatColor,
	lighten,
	parseColor,
	saturate,
	setAlpha,
	spin,
	type Rgba,
} from "./color";
export { Json5ParseError, parseJson5, parseThemeJson5, validateTheme } from "./json5";
export {
	themeMode,
	themeProps,
	type ColorSchema,
	type CompiledTheme,
	type Theme,
	type ThemeController,
	type ThemeDefinition,
	type ThemeMode,
} from "./types";
// `lightTheme` / `darkTheme` are the *default selectable* palettes, matching
// Firefish's `ColdDeviceStorage.default` (Rosé Pine Dawn / Rosé Pine). The
// token tables a theme can be layered on are `baseThemes`.
export { rosePineDawnTheme as lightTheme } from "./builtin/l-rosepinedawn";
export { rosePineTheme as darkTheme } from "./builtin/d-rosepine";
export { rosePineDawnTheme } from "./builtin/l-rosepinedawn";
export { rosePineTheme } from "./builtin/d-rosepine";
