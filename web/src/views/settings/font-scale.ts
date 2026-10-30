import { ref, watch } from "vue";

// The type scale is built from rem, so the root font size is the one knob that
// resizes every page. The value is a per-device preference, stored next to the
// theme but under its own key.
export type FontScale = "small" | "standard" | "large";

export const FONT_SCALE_STORAGE_KEY = "cinderfish.fontScale";

export const FONT_SCALES: readonly { id: FontScale; label: string; sample: string; size: string }[] = [
  { id: "small", label: "小", sample: "小", size: "15px" },
  { id: "standard", label: "标准", sample: "标准", size: "16px" },
  { id: "large", label: "大", sample: "大", size: "17px" },
];

const DEFAULT_SCALE: FontScale = "standard";

function isFontScale(value: unknown): value is FontScale {
  return value === "small" || value === "standard" || value === "large";
}

function storedFontScale(): FontScale {
  try {
    const value = localStorage.getItem(FONT_SCALE_STORAGE_KEY);
    return isFontScale(value) ? value : DEFAULT_SCALE;
  } catch {
    return DEFAULT_SCALE;
  }
}

export const fontScale = ref<FontScale>(storedFontScale());

// Exported so the boot path can apply the stored scale before the first paint
// of a page other than the appearance settings.
export function applyFontScale(scale: FontScale = fontScale.value): void {
  const entry = FONT_SCALES.find((item) => item.id === scale) ?? FONT_SCALES[1];
  document.documentElement.style.fontSize = entry.size;
}

watch(fontScale, (scale) => {
  try {
    localStorage.setItem(FONT_SCALE_STORAGE_KEY, scale);
  } catch {
    // Storage may be unavailable; the choice simply does not persist.
  }
  applyFontScale(scale);
});

export function setFontScale(scale: FontScale): void {
  fontScale.value = scale;
}
