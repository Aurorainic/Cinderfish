// The overlay layer stack: an ordered list of "things that an Escape closes".
//
// It exists because the fixed Escape order os.ts used before (dialog → menu →
// picker → composer) cannot express a popup or a deck page window opened on top
// of another layer, and because several components listen for Escape on their
// own (nav drawer, widgets drawer, note menus). One stack plus one document
// capture listener means exactly one layer closes per Escape, in opening order.
//
// Deliberately free of Vue imports: the stack is imperative state, and keeping
// it pure lets the acceptance check in tmp/ exercise it in plain Node.

export interface LayerEntry {
  id: number;
  zIndex: number;
  close: () => void;
}

export interface LayerStack {
  entries(): readonly LayerEntry[];
  top(): LayerEntry | null;
  size(): number;
  push(close: () => void, zIndex: number): LayerEntry;
  remove(target: LayerEntry | number | null | undefined): boolean;
  /** Closes the highest visible layer; ties close last-opened first. */
  closeTop(): boolean;
  close(id: number): boolean;
}

export function createLayerStack(): LayerStack {
  const stack: LayerEntry[] = [];
  let seq = 0;

  function remove(target: LayerEntry | number | null | undefined): boolean {
    if (!target) return false;
    const at =
      typeof target === "number" ? stack.findIndex((entry) => entry.id === target) : stack.indexOf(target);
    if (at < 0) return false;
    stack.splice(at, 1);
    return true;
  }

  function push(close: () => void, zIndex: number): LayerEntry {
    const entry: LayerEntry = { id: ++seq, zIndex, close };
    stack.push(entry);
    return entry;
  }

  return {
    entries: () => stack,
    top: () => stack.reduce<LayerEntry | null>((top, entry) => {
      if (!top || entry.zIndex > top.zIndex || (entry.zIndex === top.zIndex && entry.id > top.id)) {
        return entry;
      }
      return top;
    }, null),
    size: () => stack.length,
    push,
    remove,
    close(id: number): boolean {
      const entry = stack.find((candidate) => candidate.id === id);
      if (!entry) return false;
      remove(entry);
      entry.close();
      return true;
    },
    closeTop(): boolean {
      const entry = this.top();
      if (!entry) return false;
      // Remove before running the closer: a closer that re-enters the stack
      // (a layer that closes another one on the way out) must not see itself.
      remove(entry);
      entry.close();
      return true;
    },
  };
}
