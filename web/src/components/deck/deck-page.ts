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
