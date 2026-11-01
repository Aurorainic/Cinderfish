// Layer priorities mirror Firefish's os.claimZIndex: every call returns the
// next slot inside its band, so a stack of overlays keeps its insertion order
// without two of them sharing a z-index. The values sit far above the fixed
// z-indexes in main.css; only ported overlays use this, and the numbers must
// not be folded into the theme tokens (they encode order, not appearance).

export type ZIndexPriority = "low" | "middle" | "high";

const zIndexes: Record<ZIndexPriority, number> = {
  low: 1_000_000,
  middle: 2_000_000,
  high: 3_000_000,
};

const STEP = 100;

export function claimZIndex(priority: ZIndexPriority = "low"): number {
  zIndexes[priority] += STEP;
  return zIndexes[priority];
}
