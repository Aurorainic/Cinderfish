import { ref } from "vue";

// In deck mode a note or a profile opens in the PageWindow overlay instead of
// a route. The state lives here so any column can raise it and DeckShell
// renders the single overlay layer.
export type DeckPage = { kind: "note"; id: string } | { kind: "profile"; handle: string };

export const deckPage = ref<DeckPage | null>(null);

export function openDeckPage(page: DeckPage): void {
  deckPage.value = page;
}

export function closeDeckPage(): void {
  deckPage.value = null;
}

// Which deck page a resolved route opens. `os.pageWindow`/`os.modalPageWindow`
// call this with the result of `router.resolve(path)`, so a path is accepted
// only when the main router actually knows it: the catch-all `not-found` route
// resolves every string, and silently opening a 404 window is worse than
// refusing. Everything else (settings, admin, the timelines) has no standalone
// in-window view and is opened by navigating the main surface instead.
export function deckPageFor(
  name: string | null | undefined,
  params: Record<string, string | string[]>,
): DeckPage | null {
  if (name === "note" && typeof params.id === "string" && params.id) {
    return { kind: "note", id: params.id };
  }
  if (name === "profile" && typeof params.handle === "string" && params.handle) {
    return { kind: "profile", handle: params.handle };
  }
  return null;
}
