// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0a2804de48c9d73028df525e197d8df622:packages/client/src/i18n.ts
//
// Firefish-shaped i18n over Cinderfish's single zh-CN dictionary.
// The class is backend-agnostic; only the dictionary behind it is
// Cinderfish's. Two access shapes are kept because ported components use both:
//
//   i18n.ts._postForm.replyPlaceholder   // direct property access
//   t("_postForm.replyPlaceholder", {})  // dot-path lookup with {arg} fills
//
// A missing key warns and returns the key itself, so an untranslated string is
// visible during development instead of rendering as an empty label.
import { markRaw } from "vue";
import zhCN from "./zh-CN";

type Dictionary = Record<string, any>;

class I18n<T extends Dictionary> {
  public ts: T;

  constructor(locale: T) {
    this.ts = locale;
    this.t = this.t.bind(this);
  }

  public t(key: string, args?: Record<string, string | number>): string {
    try {
      const found = key.split(".").reduce<unknown>((node, part) => {
        if (node === null || typeof node !== "object") throw new Error("missing");
        return (node as Record<string, unknown>)[part];
      }, this.ts);
      if (typeof found !== "string") throw new Error("not a string");
      let text = found;
      if (args) {
        for (const [name, value] of Object.entries(args)) {
          text = text.replaceAll(`{${name}}`, String(value));
        }
      }
      return text;
    } catch {
      console.warn(`missing localization '${key}'`);
      return key;
    }
  }
}

export const i18n = markRaw(new I18n(zhCN));

/** Dot-path lookup; bound to the singleton so `t("ok")` works on its own. */
export const t = i18n.t;

/** The dictionary itself, for callers that need to enumerate keys. */
export const locale = zhCN;
