<!-- Ported from firefish@b425f61:packages/client/src/components/MkReactionIcon.vue (AGPL-3.0-only) -->
<template>
	<span class="mk-emoji" :class="{ noStyle }">
		<img
			v-if="emojiUrl != null"
			class="emoji"
			:src="emojiUrl"
			:alt="reaction"
			draggable="false"
		/>
		<span v-else class="unicode">{{ reaction }}</span>
	</span>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { resolveMediaUrl } from "@/api/client";

/**
 * Firefish delegates to the global `MkEmoji` component. Cinderfish's emoji
 * registry is always empty in note payloads (`note.emojis` / `reactionEmojis`
 * are `[]`), so this renders directly:
 *   - `☕` (a unicode reaction) is printed as-is;
 *   - `:name:` / `:name@host:` is looked up in the caller-supplied emoji list
 *     and rendered as an image when found, otherwise printed as plain text.
 */
const props = defineProps<{
	reaction: string;
	customEmojis?: unknown[];
	noStyle?: boolean;
}>();

interface CustomEmojiLike {
	name: string;
	url: string;
}

/** `:name:` or `:name@host:` — anything else is a unicode reaction. */
const shortcode = computed<{ name: string } | null>(() => {
	const match = props.reaction.match(/^:([^:\s@]+)(?:@([^:\s]+))?:$/);
	if (match == null) return null;
	return { name: match[1] };
});

const emojiUrl = computed<string | null>(() => {
	const wanted = shortcode.value;
	if (wanted == null) return null;
	for (const candidate of props.customEmojis ?? []) {
		if (candidate == null || typeof candidate !== "object") continue;
		const emoji = candidate as Partial<CustomEmojiLike>;
		if (typeof emoji.name !== "string" || typeof emoji.url !== "string") {
			continue;
		}
		if (emoji.name !== wanted.name) continue;
		return resolveMediaUrl(emoji.url);
	}
	return null;
});
</script>

<style lang="scss" scoped>
.mk-emoji {
	display: inline-flex;
	align-items: center;
	justify-content: center;
	line-height: 1;

	> .emoji {
		height: 1.25em;
		width: auto;
		vertical-align: middle;
		pointer-events: none;
	}

	&.noStyle {
		height: 1em;
	}
}
</style>
