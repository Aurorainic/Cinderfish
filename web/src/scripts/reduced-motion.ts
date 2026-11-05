// Ported from firefish@b425f61:packages/client/src/scripts/reduced-motion.ts (AGPL-3.0-only)

export function reducedMotion(): boolean {
	if (typeof window === "undefined" || typeof window.matchMedia !== "function") {
		return false;
	}
	return window.matchMedia("(prefers-reduced-motion: reduce)").matches;
}
