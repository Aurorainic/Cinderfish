<!-- Ported from firefish@b425f61:packages/client/src/components/global/MkEmoji.vue (AGPL-3.0-only) -->
<template>
	<img
		v-if="customEmoji"
		class="mk-emoji custom"
		:class="{ normal, noStyle }"
		:src="url"
		:alt="alt"
		:title="alt"
		decoding="async"
	/>
	<span v-else-if="char" class="mk-emoji">{{ char }}</span>
	<span v-else class="mk-emoji plain">{{ emoji }}</span>
</template>

<script setup lang="ts">
import { computed, watchEffect } from "vue";
import type { EmojiSimple } from "../../api/types";
import { resolveMediaUrl } from "../../api/client";
import { lookupEmoji } from "./emoji-registry";

// Degraded versus Firefish: unicode emojis are rendered with the platform font
// instead of the bundled twemoji images (no twemoji assets are shipped), and
// animated-image downscaling is not applied. Custom emojis come from the
// customEmojis prop first and from the registered emoji endpoint second.
const props = defineProps<{
	emoji: string;
	normal?: boolean;
	noStyle?: boolean;
	customEmojis?: EmojiSimple[];
	isReaction?: boolean;
}>();

const isCustom = computed<boolean>(() => props.emoji.startsWith(":"));

const name = computed<string>(() =>
	isCustom.value ? props.emoji.substr(1, props.emoji.length - 2) : "",
);

const char = computed<string | null>(() =>
	isCustom.value ? null : props.emoji,
);

const customEmoji = computed<EmojiSimple | null>(() => {
	if (!isCustom.value) return null;
	const local = (props.customEmojis ?? []).find((x) => x.name === name.value);
	return local ?? lookupEmoji(name.value);
});

watchEffect(() => {
	// Touch the lookup while a shortcode is unresolved so the registry fetch is
	// started once; the reactive cache re-renders the component when it lands.
	if (isCustom.value && customEmoji.value == null) void lookupEmoji(name.value);
});

const url = computed<string>(() =>
	customEmoji.value == null ? "" : resolveMediaUrl(customEmoji.value.url),
);

const alt = computed<string>(() =>
	customEmoji.value ? ":" + customEmoji.value.name + ":" : (char.value ?? ""),
);
</script>

<style lang="scss" scoped>
.mk-emoji {
	height: 1.25em;
	vertical-align: -0.25em;

	&.custom {
		height: 2em;
		vertical-align: middle;
		transition: transform 0.2s ease;

		&:hover {
			transform: scale(1.2);
		}

		&.normal {
			height: 1.25em;
			vertical-align: -0.25em;

			&:hover {
				transform: none;
			}
		}
	}

	&.noStyle {
		height: auto;
	}
}
</style>
