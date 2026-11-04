// Locale runtime. Firefish's i18n.ts exposes an I18n instance with
// `locale` (the nested message object) and `t(key, args)`. Cinderfish stores
// messages as flat dot-key JSON produced by web/scripts/gen-locale.mjs, so
// `t()` resolves the same dot paths against a flat dictionary and fails soft
// (returns the key) instead of throwing.
import { computed, ref, type ComputedRef, type Ref } from "vue";
import zhCN from "./locales/zh-CN.json";
import jaJP from "./locales/ja-JP.json";

export type LocaleCode = "zh-CN" | "ja-JP";
export type LocaleMessages = Record<string, string>;
export type I18nArgs = Record<string, string | number>;

export const messages: Record<LocaleCode, LocaleMessages> = {
  "zh-CN": zhCN as LocaleMessages,
  "ja-JP": jaJP as LocaleMessages,
};

export const availableLocales: readonly LocaleCode[] = ["zh-CN", "ja-JP"];
export const DEFAULT_LOCALE: LocaleCode = "zh-CN";

function isLocaleCode(value: string | null | undefined): value is LocaleCode {
  return value != null && (availableLocales as readonly string[]).includes(value);
}

function detectLocale(): LocaleCode {
  try {
    const stored = localStorage.getItem("lang");
    if (isLocaleCode(stored)) return stored;
  } catch {
    // ignore
  }
  if (typeof navigator !== "undefined") {
    const candidates: string[] = [
      ...(navigator.languages ?? []),
      navigator.language,
    ].filter((tag): tag is string => typeof tag === "string" && tag.length > 0);

    for (const tag of candidates) {
      if (isLocaleCode(tag)) return tag;
      const prefix = tag.slice(0, 2).toLowerCase();
      if (prefix === "ja") return "ja-JP";
      if (prefix === "zh") return "zh-CN";
    }
  }
  return DEFAULT_LOCALE;
}

export const locale: Ref<LocaleCode> = ref<LocaleCode>(detectLocale());

let syncChannel: BroadcastChannel | null = null;

function ensureSync(): void {
  if (syncChannel != null || typeof BroadcastChannel === "undefined") return;
  syncChannel = new BroadcastChannel("cinderfish:locale");
  syncChannel.addEventListener("message", (event: MessageEvent<{ locale?: string }>) => {
    const next = event.data?.locale;
    if (isLocaleCode(next)) locale.value = next;
  });
}

export function setLocale(next: LocaleCode): void {
  ensureSync();
  locale.value = next;
  try {
    localStorage.setItem("lang", next);
  } catch {
    // ignore
  }
  syncChannel?.postMessage({ locale: next });
}

function lookup(dictionary: LocaleMessages, key: string): string | null {
  const direct = dictionary[key];
  if (typeof direct === "string") return direct;

  // Also accept a nested object shape, for callers that build messages by hand.
  const segments = key.split(".");
  let cursor: unknown = dictionary;
  for (const segment of segments) {
    if (cursor == null || typeof cursor !== "object") return null;
    cursor = (cursor as Record<string, unknown>)[segment];
  }
  return typeof cursor === "string" ? cursor : null;
}

export function interpolate(template: string, args?: I18nArgs): string {
  if (args == null) return template;
  let result = template;
  for (const [name, value] of Object.entries(args)) {
    result = result.split(`{${name}}`).join(String(value));
  }
  return result;
}

export function translate(key: string, args?: I18nArgs): string {
  const active = messages[locale.value];
  const value = lookup(active, key) ?? lookup(messages[DEFAULT_LOCALE], key);
  if (value == null) {
    if (import.meta.env.DEV) console.warn(`[i18n] missing key: ${key}`);
    return key;
  }
  return interpolate(value, args);
}

/** Firefish-compatible singleton: `i18n.ts.someKey` and `i18n.t("someKey")`. */
export const i18n = {
  t: translate,
  get ts(): LocaleMessages {
    return messages[locale.value];
  },
  get locale(): LocaleCode {
    return locale.value;
  },
};

export interface UseLocaleReturn {
  locale: Ref<LocaleCode>;
  t: (key: string, args?: I18nArgs) => string;
  ts: ComputedRef<LocaleMessages>;
  setLocale: (next: LocaleCode) => void;
  availableLocales: readonly LocaleCode[];
}

export function useLocale(): UseLocaleReturn {
  ensureSync();
  return {
    locale,
    t: translate,
    ts: computed(() => messages[locale.value]),
    setLocale,
    availableLocales,
  };
}

/**
 * Compatibility shorthands used by ported Firefish components
 * (`import { t } from "@/i18n"`). They are plain aliases of the reactive
 * functions above; prefer `useLocale()` when a component must re-render on
 * locale change.
 */
export const t = translate;

export function getLocale(): LocaleCode {
	return locale.value;
}

/** Permissive nested/leaf tree used by older ported modules. */
export type I18nTree = Record<string, unknown>;

export type InterpolationArgs = I18nArgs;
