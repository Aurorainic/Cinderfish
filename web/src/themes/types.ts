import type { ComputedRef, Ref } from "vue";

export type ThemeMode = "light" | "dark";

/**
 * A Firefish theme definition. `props` uses Firefish's value language:
 *
 *   "#86b300"                literal colour
 *   "@accent"                reference to another prop (recursive)
 *   "$const"                 reference to a `$`-prefixed constant prop
 *   ":darken<10<@accent"     colour function applied to a resolved colour
 *   '" solid 1px var(...)'   a quoted literal emitted verbatim
 */
export interface Theme {
	id: string;
	name: string;
	author: string;
	desc?: string;
	/** Firefish stores this as `kind` on the built-in base themes. */
	kind?: ThemeMode;
	/** Legacy themes use `base`; kept for imported user themes. */
	base?: ThemeMode;
	props: Record<string, string>;
}

export type CompiledTheme = Record<string, string>;

export const themeProps = (theme: Theme): string[] =>
	Object.keys(theme.props).filter((key) => !key.startsWith("X"));

export function themeMode(theme: Theme): ThemeMode {
	if (theme.kind === "dark" || theme.base === "dark") return "dark";
	if (theme.kind === "light" || theme.base === "light") return "light";
	return "light";
}

/** Alias used by the ported theme controllers (`composables/use-theme.ts`). */
export type ThemeDefinition = Theme;

/** `light`/`dark` schema name used for `data-theme` and by theme controllers. */
export type ColorSchema = ThemeMode;

/**
 * The hook shape this engine expects the reactive layer to provide
 * (`useTheme()` in `web/src/composables/`):
 *
 *   const { themeName, setTheme, resolvedMode } = useTheme();
 *
 * `setTheme` takes either a mode (`"light"` / `"dark"` / `"auto"`) or a theme
 * id resolvable through `findTheme()`.
 */
export interface ThemeController {
  themeName: Ref<string>;
  resolvedMode: ComputedRef<ColorSchema>;
  setTheme(name: string): void;
}
