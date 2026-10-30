import {
  effectScope,
  onBeforeUnmount,
  onMounted,
  toValue,
  watch,
  type EffectScope,
  type MaybeRefOrGetter,
} from "vue";
import { instance } from "../stores/instance";

const FALLBACK_SITE_NAME = "Cinderfish";

interface TitleEntry {
  id: number;
  title: string;
}

const stack: TitleEntry[] = [];
let nextId = 1;
let siteScope: EffectScope | null = null;

function siteName(): string {
  const meta = instance.meta;
  const name = meta?.name?.trim() || meta?.shortName?.trim();
  return name || FALLBACK_SITE_NAME;
}

function render(): void {
  const top = stack[stack.length - 1];
  const title = top?.title.trim() ?? "";
  document.title = title ? `${title} · ${siteName()}` : siteName();
}

function refresh(entry: TitleEntry): void {
  if (stack[stack.length - 1] === entry) render();
}

function ensureSiteWatch(): void {
  if (siteScope) return;
  siteScope = effectScope(true);
  siteScope.run(() => {
    watch(
      () => instance.meta?.name,
      () => render(),
    );
  });
}

export function usePageTitle(source: MaybeRefOrGetter<string | null | undefined>): void {
  const entry: TitleEntry = { id: nextId++, title: toValue(source) ?? "" };

  const stopTitleWatch = watch(
    () => toValue(source) ?? "",
    (title) => {
      entry.title = title;
      refresh(entry);
    },
  );

  onMounted(() => {
    stack.push(entry);
    ensureSiteWatch();
    render();
  });

  onBeforeUnmount(() => {
    stopTitleWatch();
    const index = stack.findIndex((item) => item.id === entry.id);
    if (index >= 0) stack.splice(index, 1);
    render();
  });
}
