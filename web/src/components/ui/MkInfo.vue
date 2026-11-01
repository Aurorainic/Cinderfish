// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/components/MkInfo.vue
//
// Firefish's icon helper emits Phosphor icons; Cinderfish has no Phosphor
// dependency (TODO: icon runtime), so the port renders Unicode glyphs.
<template>
	<div v-if="visible" class="cinderfish-info" :class="{ warn, card }">
		<span v-if="warn" class="glyph">⚠</span>
		<span v-else class="glyph">{{ glyph }}</span>
		<slot></slot>
		<button
			v-if="closeable"
			class="close"
			:aria-label="strings.close"
			@click.stop="close"
		>
			✕
		</button>
	</div>
</template>

<script lang="ts" setup>
import { computed, ref } from "vue";

// Local UI strings (hardcoded Chinese until a shared i18n layer exists).
const strings = {
	close: "关闭",
};

const props = defineProps<{
	icon?: string;
	warn?: boolean;
	card?: boolean;
	closeable?: boolean;
}>();

const emit = defineEmits<{
	(ev: "close"): void;
}>();

const visible = ref(true);

// Only a small glyph subset is mapped; unknown icons fall back to ℹ.
const glyphs: Record<string, string> = {
	"ph-info": "ℹ",
	"ph-warning": "⚠",
	"ph-bell": "🔔",
	"ph-megaphone": "📣",
};

const glyph = computed(() => (props.icon && glyphs[props.icon]) || "ℹ");

function close() {
	visible.value = false;
	emit("close");
}
</script>

<style scoped>
.cinderfish-info {
	padding: 16px;
	font-size: 90%;
	background: var(--accent-soft);
	color: inherit;
	border-radius: var(--radius);
	display: flex;
	align-items: center;
	gap: 0.4em;
	white-space: pre-line;
}

.cinderfish-info.warn {
	background: rgba(235, 111, 146, 0.15);
	color: #eb6f92;
}

.cinderfish-info.card {
	display: block;
	background: var(--panel);
	color: inherit;
	padding: 48px;
	font-size: 1em;
	text-align: center;
}

.cinderfish-info.card > .glyph {
	display: block;
	font-size: 4em;
	margin: 0;
}

.cinderfish-info.card > :deep(*) {
	margin-inline: auto;
}

.cinderfish-info.card > :deep(:not(:last-child)) {
	margin-block-end: 20px;
}

.cinderfish-info.card > :deep(p) {
	max-inline-size: 30em;
}

.cinderfish-info > .glyph {
	margin-inline-end: 4px;
}

.cinderfish-info > .close {
	margin-inline-start: auto;
	background: none;
	border: none;
	cursor: pointer;
	color: inherit;
	font-size: 1em;
	padding: 4px;
}
</style>
