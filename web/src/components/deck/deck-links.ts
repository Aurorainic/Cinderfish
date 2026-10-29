import { openDeckPage, type DeckPage } from "./deck-page";

// Deck columns must not navigate: a click on a note or a profile anywhere in a
// column opens that page in the PageWindow overlay instead. The note and
// avatar components still render plain RouterLinks, so the interception runs
// in the capture phase above them — preventDefault stops the navigation and
// stopPropagation keeps the anchor's own click handler from firing.
const NOTE_PATH = /^\/notes\/([^/]+)\/?$/;
const PROFILE_PATH = /^\/@([^/]+)\/?$/;

// Clicks on these belong to the control itself (reactions, media, CW banners).
const CONTROL_SELECTOR = "button, input, textarea, select, label, summary, video, audio, [contenteditable='true']";

function pageForHref(href: string): DeckPage | null {
  let url: URL;
  try {
    url = new URL(href, window.location.origin);
  } catch {
    return null;
  }
  if (url.origin !== window.location.origin) return null;

  const note = NOTE_PATH.exec(url.pathname);
  if (note) return { kind: "note", id: decodeURIComponent(note[1]) };

  const profile = PROFILE_PATH.exec(url.pathname);
  if (profile) return { kind: "profile", handle: decodeURIComponent(profile[1]) };

  return null;
}

// Clicking a note's text opens the note; the note row has no id of its own in
// the DOM, so the id comes from the timestamp link every note carries.
function noteIdIn(article: Element): string {
  const link = article.querySelector('a[href^="/notes/"]');
  const href = link?.getAttribute("href") ?? "";
  const match = NOTE_PATH.exec(href);
  return match ? decodeURIComponent(match[1]) : "";
}

export function interceptDeckClick(event: MouseEvent): void {
  // Modified and middle clicks keep their browser behaviour (new tab etc.).
  if (event.defaultPrevented) return;
  if (event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return;
  if (!(event.target instanceof Element)) return;

  const anchor = event.target.closest("a[href]");
  if (anchor) {
    const page = pageForHref(anchor.getAttribute("href") ?? "");
    if (!page) return;
    event.preventDefault();
    event.stopPropagation();
    openDeckPage(page);
    return;
  }

  if (event.target.closest(CONTROL_SELECTOR)) return;
  // A drag-selected passage is being read, not clicked through.
  if ((window.getSelection()?.toString() ?? "") !== "") return;

  const body = event.target.closest(".body");
  const article = body?.closest("article");
  if (!article) return;
  const id = noteIdIn(article);
  if (!id) return;
  event.preventDefault();
  event.stopPropagation();
  openDeckPage({ kind: "note", id });
}
