// Ported from firefish@b425f61:packages/client/src/components/global/MkEmoji.vue (AGPL-3.0-only)
import { reactive } from "vue";
import { api } from "../../api/client";
import type { EmojiSimple } from "../../api/types";

// Firefish resolves custom emojis from instance.emojis, which is filled from the
// metadata response. Cinderfish has no emoji data in meta, so the registry is
// fetched once from the documented emojis endpoint (api-contract: "表情注册表")
// and cached for the lifetime of the page.
const state = reactive<{
	loaded: boolean;
	byName: Record<string, EmojiSimple>;
}>({
	loaded: false,
	byName: {},
});

// Non-reactive single-flight guard: lookupEmoji() is called from render code, so
// it must never write to reactive state while a component is rendering.
let started = false;

/** Synchronous lookup for render code; starts the single fetch in the background. */
export function lookupEmoji(name: string): EmojiSimple | null {
	if (!started) {
		started = true;
		void load();
	}
	return state.byName[name] ?? null;
}

async function load(): Promise<void> {
	try {
		const res = await api("emojis", {});
		const byName: Record<string, EmojiSimple> = {};
		for (const emoji of res.emojis ?? []) {
			byName[emoji.name] = emoji;
		}
		state.byName = byName;
	} catch {
		// A failed registry request degrades to "no custom emojis": shortcodes
		// render as plain text instead of retrying on every note.
	} finally {
		state.loaded = true;
	}
}

/** Test/refresh hook: forget the cached registry and allow one more fetch. */
export function resetEmojiRegistry(): void {
	state.loaded = false;
	state.byName = {};
	started = false;
}
