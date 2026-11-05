<!-- Ported from firefish@b425f61:packages/client/src/components/MkEmojiPicker.vue (AGPL-3.0-only) -->
<template>
	<div
		class="omfetrab"
		:class="{ asDrawer }"
		:style="maxHeight != null ? { maxHeight: `${maxHeight}px` } : undefined"
		tabindex="-1"
	>
		<input
			ref="searchEl"
			v-model.trim="q"
			class="search"
			data-prevent-emoji-insert
			:class="{ filled: q !== '' }"
			:placeholder="t('search')"
			type="search"
			@paste.stop="onPaste"
			@keyup.enter="done()"
		/>
		<div ref="emojisEl" class="emojis">
			<section v-if="q !== ''" class="result">
				<div v-if="searchResultCustom.length > 0" class="body">
					<button
						v-for="emoji in searchResultCustom"
						:key="emoji.name"
						class="_button item"
						:title="emoji.name"
						@click="chosen(emoji, $event)"
					>
						<img
							class="emoji custom"
							:src="resolveMediaUrl(emoji.url)"
							:alt="emoji.name"
						/>
					</button>
				</div>
				<div v-if="searchResultUnicode.length > 0" class="body">
					<button
						v-for="emoji in searchResultUnicode"
						:key="emoji.slug"
						class="_button item"
						:title="emoji.slug"
						@click="chosen(emoji, $event)"
					>
						<MkEmoji class="emoji" :emoji="emoji.emoji" />
					</button>
				</div>
			</section>

			<div class="group index">
				<section v-if="showPinned && pinned.length > 0">
					<div class="body">
						<button
							v-for="emoji in pinned"
							:key="emoji"
							class="_button item"
							@click="chosen(emoji, $event)"
						>
							<MkEmoji class="emoji" :emoji="emoji" :normal="true" />
						</button>
					</div>
				</section>

				<section v-if="recentlyUsedEmojis.length > 0">
					<header class="_acrylic">
						<i class="ph-alarm ph-bold ph-fw ph-lg"></i>
						{{ t("recentUsed") }}
					</header>
					<div class="body">
						<button
							v-for="emoji in recentlyUsedEmojis"
							:key="emoji"
							class="_button item"
							@click="chosen(emoji, $event)"
						>
							<MkEmoji class="emoji" :emoji="emoji" :normal="true" />
						</button>
					</div>
				</section>
			</div>

			<div v-if="customEmojis.length > 0 || loadFailed" class="group">
				<header class="_acrylic">{{ t("customEmojis") }}</header>
				<CfInfo v-if="loadFailed" warn class="load-failed">
					{{ t("somethingHappened") }}
				</CfInfo>
				<XSection
					v-for="category in customEmojiCategories"
					:key="`custom:${category}`"
					:emojis="
						customEmojis
							.filter((e) => (e.category ?? '') === category)
							.map((e) => `:${e.name}:`)
					"
					@chosen="chosen"
				>
					{{ category === "" ? t("other") : category }}
				</XSection>
			</div>

			<div class="group">
				<header class="_acrylic">{{ t("emoji") }}</header>
				<XSection
					v-for="category in unicodeEmojiCategories"
					:key="category"
					:skin-tone-selector="category === 'people'"
					:skin-tones="unicodeEmojiSkinTones"
					:skin-tone-labels="unicodeEmojiSkinToneLabels"
					:emojis="
						emojilist
							.filter((e) => e.category === category)
							.map((e) => e.emoji)
					"
					@chosen="chosen"
				>
					{{ getNicelyLabeledCategory(category) }}
				</XSection>
			</div>
		</div>
	</div>
</template>

<script lang="ts" setup>
// Wave-3 degradation (contract §2): `focus-trap-vue`, `unicode-emoji-json` and
// `emojilib` are not dependencies of this port.
//   * the unicode list is the curated one from scripts/emojilist.ts;
//   * focus stays on the search field instead of being trapped in a ring;
//   * MkRipple is not ported, so the tap feedback is the CSS `:active` state;
//   * the category tabs are dropped (upstream hides them with `display: none`
//     anyway) — the picker is one scrollable list of sections.
import { computed, onMounted, ref, watch } from "vue";
import type { EmojiSimple } from "@/api/types";
import { api, resolveMediaUrl } from "@/api/client";
import { defaultStore } from "@/store/base";
import { t } from "@/i18n";
import {
	emojilist,
	getNicelyLabeledCategory,
	unicodeEmojiCategories,
	type UnicodeEmojiDef,
} from "@/scripts/emojilist";
import { deviceKind } from "@/scripts/device-kind";
import CfInfo from "@/components/ui/CfInfo.vue";
import XSection from "@/components/MkEmojiPicker.section.vue";

const props = withDefaults(
	defineProps<{
		showPinned?: boolean;
		asReactionPicker?: boolean;
		maxHeight?: number;
		asDrawer?: boolean;
	}>(),
	{
		showPinned: true,
		asReactionPicker: false,
		maxHeight: undefined,
		asDrawer: false,
	},
);

const emit = defineEmits<{
	chosen: [emoji: string, event: MouseEvent | null];
}>();

const searchEl = ref<HTMLInputElement | null>(null);
const emojisEl = ref<HTMLDivElement | null>(null);

const pinned = defaultStore.reactiveState.reactions;
const recentlyUsedEmojis = defaultStore.reactiveState.recentlyUsedEmojis;

const unicodeEmojiSkinTones = [
	"#FFDC5E",
	"#F7DFCF",
	"#F3D3A3",
	"#D6AE89",
	"#B17F56",
	"#7D523C",
];

const unicodeEmojiSkinToneLabels = [
	t("_skinTones.yellow"),
	t("_skinTones.light"),
	t("_skinTones.mediumLight"),
	t("_skinTones.medium"),
	t("_skinTones.mediumDark"),
	t("_skinTones.dark"),
];

const customEmojis = ref<EmojiSimple[]>([]);
const loadFailed = ref(false);
const q = ref<string>("");
const searchResultCustom = ref<EmojiSimple[]>([]);
const searchResultUnicode = ref<UnicodeEmojiDef[]>([]);

const customEmojiCategories = computed<string[]>(() => {
	const names = new Set<string>();
	for (const emoji of customEmojis.value) names.add(emoji.category ?? "");
	return Array.from(names).sort((a, b) =>
		a === "" ? 1 : b === "" ? -1 : a.localeCompare(b),
	);
});

async function loadCustomEmojis(): Promise<void> {
	try {
		const res = await api("emojis", {});
		customEmojis.value = res.emojis ?? [];
	} catch {
		// The picker still works with the unicode list; the custom-emoji group
		// says so instead of silently looking like an empty registry.
		loadFailed.value = true;
		customEmojis.value = [];
	}
}

function searchCustom(newQ: string): EmojiSimple[] {
	const max = 16;
	const matches = new Set<EmojiSimple>();
	const emojis = customEmojis.value;

	const exactMatch = emojis.find((emoji) => emoji.name === newQ);
	if (exactMatch != null) matches.add(exactMatch);

	const keywords = newQ.split(" ");
	const nameFirst = (emoji: EmojiSimple): boolean =>
		keywords.every((keyword) => emoji.name.includes(keyword));

	for (const emoji of emojis) {
		if (emoji.name.startsWith(newQ)) {
			matches.add(emoji);
			if (matches.size >= max) return Array.from(matches);
		}
	}
	for (const emoji of emojis) {
		if (emoji.aliases.some((alias) => alias.startsWith(newQ))) {
			matches.add(emoji);
			if (matches.size >= max) return Array.from(matches);
		}
	}
	for (const emoji of emojis) {
		if (nameFirst(emoji)) {
			matches.add(emoji);
			if (matches.size >= max) return Array.from(matches);
		}
	}
	for (const emoji of emojis) {
		if (
			emoji.name.includes(newQ) ||
			emoji.aliases.some((alias) => alias.includes(newQ))
		) {
			matches.add(emoji);
			if (matches.size >= max) return Array.from(matches);
		}
	}

	return Array.from(matches);
}

function searchUnicode(newQ: string): UnicodeEmojiDef[] {
	const max = 32;
	const matches = new Set<UnicodeEmojiDef>();

	const exactMatch = emojilist.find((emoji) => emoji.slug === newQ);
	if (exactMatch != null) matches.add(exactMatch);

	const keywords = newQ.split(" ");
	const matchesKeywords = (emoji: UnicodeEmojiDef): boolean =>
		keywords.every((keyword) => emoji.keywords.includes(keyword));

	for (const emoji of emojilist) {
		if (emoji.slug.startsWith(newQ)) {
			matches.add(emoji);
			if (matches.size >= max) return Array.from(matches);
		}
	}
	for (const emoji of emojilist) {
		if (emoji.keywords.some((keyword) => keyword.startsWith(newQ))) {
			matches.add(emoji);
			if (matches.size >= max) return Array.from(matches);
		}
	}
	for (const emoji of emojilist) {
		if (matchesKeywords(emoji)) {
			matches.add(emoji);
			if (matches.size >= max) return Array.from(matches);
		}
	}
	for (const emoji of emojilist) {
		if (
			emoji.slug.includes(newQ) ||
			emoji.keywords.some((keyword) => keyword.includes(newQ))
		) {
			matches.add(emoji);
			if (matches.size >= max) return Array.from(matches);
		}
	}

	return Array.from(matches);
}

watch(q, () => {
	if (emojisEl.value != null) emojisEl.value.scrollTop = 0;

	if (q.value === "") {
		searchResultCustom.value = [];
		searchResultUnicode.value = [];
		return;
	}

	const newQ = q.value.replace(/:/g, "").toLowerCase();
	searchResultCustom.value = searchCustom(newQ);
	searchResultUnicode.value = searchUnicode(newQ);
});

function focus(): void {
	if (deviceKind !== "smartphone" && deviceKind !== "tablet" && !props.asDrawer) {
		searchEl.value?.focus({ preventScroll: true });
	}
}

function reset(): void {
	if (emojisEl.value != null) emojisEl.value.scrollTop = 0;
	q.value = "";
}

function getKey(emoji: string | EmojiSimple | UnicodeEmojiDef): string {
	return typeof emoji === "string"
		? emoji
		: "emoji" in emoji
			? emoji.emoji
			: `:${emoji.name}:`;
}

/** Keeps scripts/reaction-picker.ts's recents list shared with the picker. */
function pushRecent(key: string): void {
	const recents = defaultStore.state.recentlyUsedEmojis.filter(
		(emoji) => emoji !== key,
	);
	recents.unshift(key);
	defaultStore.set("recentlyUsedEmojis", recents.slice(0, 32));
}

function chosen(
	emoji: string | EmojiSimple | UnicodeEmojiDef,
	ev?: MouseEvent | null,
): void {
	const key = getKey(emoji);
	emit("chosen", key, ev ?? null);
	if (!pinned.value.includes(key)) pushRecent(key);
}

function onPaste(event: ClipboardEvent): void {
	const text = event.clipboardData?.getData("text") ?? "";
	if (done(text)) event.preventDefault();
}

function done(query?: string): boolean | void {
	const value = query ?? q.value;
	if (value === "") return;

	const q2 = value.replaceAll(":", "");

	const exactCustom = customEmojis.value.find((emoji) => emoji.name === q2);
	if (exactCustom != null) {
		chosen(exactCustom);
		return true;
	}
	const exactUnicode = emojilist.find(
		(emoji) => emoji.emoji === q2 || emoji.slug === q2,
	);
	if (exactUnicode != null) {
		chosen(exactUnicode);
		return true;
	}
	if (searchResultCustom.value.length > 0) {
		chosen(searchResultCustom.value[0]);
		return true;
	}
	if (searchResultUnicode.value.length > 0) {
		chosen(searchResultUnicode.value[0]);
		return true;
	}
}

onMounted(() => {
	void loadCustomEmojis();
	focus();
});

defineExpose({ focus, reset });
</script>

<style lang="scss" scoped>
.omfetrab {
	--eachSize: 40px;
	--pad: 8px;

	display: flex;
	flex-direction: column;
	inline-size: calc(var(--eachSize) * 7 + var(--pad) * 2);
	box-sizing: border-box;
	overflow: hidden;

	&.asDrawer {
		inline-size: 100%;
	}

	> .search {
		inline-size: 100%;
		padding: 12px;
		box-sizing: border-box;
		font-size: 1em;
		outline: none;
		border: none;
		background: transparent;
		color: var(--fg);

		&:not(.filled) {
			order: 1;
			z-index: 2;
			box-shadow: 0 -1px 0 0 var(--divider);
		}

		&:focus {
			outline: none;
		}
	}

	> .emojis {
		block-size: 100%;
		overflow-y: auto;
		overflow-x: hidden;
		scrollbar-width: none;

		&::-webkit-scrollbar {
			display: none;
		}

		> .group {
			&:not(.index) {
				padding: 4px 0 8px;
				border-top: solid 0.5px var(--divider);
			}

			> header {
				block-size: 32px;
				line-height: 32px;
				z-index: 2;
				padding: 0 8px;
				font-size: 12px;
			}
		}

		> section {
			> header {
				position: sticky;
				inset-block-start: 0;
				block-size: 32px;
				line-height: 32px;
				z-index: 1;
				padding: 0 8px;
				font-size: 12px;
			}

			> .body {
				position: relative;
				padding: var(--pad);
				display: flex;
				flex-wrap: wrap;
			}
		}

		// The .item rules live here as well so the search result rows and the
		// pinned/recents rows share one look; the section component repeats them
		// for its own scoped tree.
		.item {
			padding: 0;
			inline-size: var(--eachSize);
			block-size: var(--eachSize);
			border-radius: 4px;
			font-size: 24px;

			&:hover {
				background: var(--listItemHoverBg);
			}

			&:active {
				background: var(--accent);
			}

			> .emoji {
				block-size: 1.25em;
				vertical-align: -0.25em;
				pointer-events: none;

				&.custom {
					block-size: 24px;
					inline-size: 24px;
					object-fit: contain;
				}
			}
		}

		.load-failed {
			margin: 8px;
		}

		> .result {
			border-bottom: solid 0.5px var(--divider);

			> .empty {
				padding: 16px 8px;
				font-size: 0.85em;
				opacity: 0.7;
				text-align: center;
			}
		}
	}
}
</style>
