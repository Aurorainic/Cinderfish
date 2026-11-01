// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/components/global/MkError.vue
//
// The original shows the instance's error badge image; Cinderfish serves no
// static-assets badges, so the port renders a warning glyph instead.
<template>
	<transition name="zoom" appear>
		<div class="cinderfish-error">
			<div class="badge" aria-hidden="true">!</div>
			<p>{{ strings.somethingHappened }}</p>
			<MkButton class="button" @click.stop="emit('retry')">
				{{ strings.retry }}
			</MkButton>
		</div>
	</transition>
</template>

<script lang="ts" setup>
import MkButton from "./MkButton.vue";

// Local UI strings (hardcoded Chinese until a shared i18n layer exists).
const strings = {
	somethingHappened: "发生了一些问题",
	retry: "重试",
};

const emit = defineEmits<{
	(ev: "retry"): void;
}>();
</script>

<style scoped>
.cinderfish-error {
	padding: 32px;
	text-align: center;
	align-items: center;
}

.cinderfish-error > p {
	margin: 0 0 8px;
}

.cinderfish-error > .button {
	margin-block: 0;
	margin-inline: auto;
}

.cinderfish-error > .badge {
	inline-size: 128px;
	block-size: 128px;
	margin-block-end: 16px;
	margin-inline: auto;
	border-radius: 16px;
	background: var(--accent-soft);
	color: var(--accent);
	font-size: 64px;
	font-weight: bold;
	line-height: 128px;
}

.zoom-enter-active {
	transition: opacity 0.2s ease, transform 0.2s ease;
}

.zoom-enter-from {
	opacity: 0;
	transform: scale(0.95);
}
</style>
