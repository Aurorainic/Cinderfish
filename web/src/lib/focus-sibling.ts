// Focus movement between adjacent rows of a feed. A feed interleaves notes
// with non-interactive rows (date separators, notices), so the walk skips
// any sibling that is not programmatically focusable.

function neighbour(start: Element | null, direction: "previous" | "next"): HTMLElement | null {
  let current: Element | null = start;
  while (current) {
    const candidate: Element | null =
      direction === "previous" ? current.previousElementSibling : current.nextElementSibling;
    if (!candidate) return null;
    if (candidate instanceof HTMLElement && candidate.hasAttribute("tabindex")) return candidate;
    current = candidate;
  }
  return null;
}

function focusNeighbour(element: Element | null, direction: "previous" | "next"): HTMLElement | null {
  if (!element) return null;
  const target = neighbour(element, direction);
  target?.focus();
  return target;
}

export function focusPrev(element: Element | null): HTMLElement | null {
  return focusNeighbour(element, "previous");
}

export function focusNext(element: Element | null): HTMLElement | null {
  return focusNeighbour(element, "next");
}
