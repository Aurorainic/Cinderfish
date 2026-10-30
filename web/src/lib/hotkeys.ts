// Per-row keyboard shortcuts. A row binds the returned handler to its own
// keydown event, so a note only answers while it holds focus; the handler
// swallows every key it matches so the browser does not also act on it.

export type HotkeyMap = Record<string, () => void>;

interface Action {
  codes: string[];
  allowRepeat: boolean;
  run: () => void;
}

const ALIASES: Record<string, string[]> = {
  up: ["ArrowUp"],
  down: ["ArrowDown"],
  left: ["ArrowLeft"],
  right: ["ArrowRight"],
  esc: ["Escape"],
  enter: ["Enter"],
  space: ["Space"],
  plus: ["Equal", "NumpadAdd", "+"],
  minus: ["Minus", "NumpadSubtract", "-"],
};

function codesFor(key: string): string[] {
  const alias = ALIASES[key];
  if (alias) return alias;
  if (/^[a-z]$/.test(key)) return [`Key${key.toUpperCase()}`, key];
  if (/^[0-9]$/.test(key)) return [`Digit${key}`, `Numpad${key}`, key];
  return [key];
}

function parse(map: HotkeyMap): Action[] {
  return Object.entries(map).map(([pattern, run]) => {
    let allowRepeat = true;
    let value = pattern;
    if (value.startsWith("(") && value.endsWith(")")) {
      allowRepeat = false;
      value = value.slice(1, -1);
    }
    const codes = value
      .split("|")
      .flatMap((part) => codesFor(part.trim().toLowerCase()))
      .filter(Boolean);
    return { codes, allowRepeat, run };
  });
}

function isTyping(element: Element | null): boolean {
  if (!element) return false;
  const tag = element.tagName.toLowerCase();
  if (tag === "input" || tag === "textarea" || tag === "select") return true;
  return element instanceof HTMLElement && element.isContentEditable;
}

export function makeHotkey(map: HotkeyMap): (event: KeyboardEvent) => void {
  const actions = parse(map);

  return (event: KeyboardEvent) => {
    if (event.ctrlKey || event.metaKey || event.altKey) return;
    if (isTyping(document.activeElement)) return;

    const pressed = [(event.code || "").toLowerCase(), (event.key || "").toLowerCase()];
    for (const action of actions) {
      if (!action.codes.some((code) => pressed.includes(code.toLowerCase()))) continue;
      if (!action.allowRepeat && event.repeat) return;
      event.preventDefault();
      event.stopPropagation();
      action.run();
      return;
    }
  };
}
