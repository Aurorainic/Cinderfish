// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/components/MkCwButton.vue
//
// The toggle under a content warning. The original counts the hidden body
// with `stringz.length`; Cinderfish has no grapheme-count dependency, so the
// UTF-16 length is used — it can overcount astral emoji by a few characters.
<template>
	<button ref="el" type="button" class="cw-button" :class="{ showLess: modelValue }" @click.stop="toggle">
		<span>
			{{ modelValue ? i18n.ts._cw.hide : i18n.ts._cw.show }}
			<span v-if="!modelValue" class="label">{{ label }}</span>
		</span>
	</button>
</template>

<script lang="ts" setup>
import { computed, ref } from "vue";
import type { Note } from "../../api/types";
import { i18n } from "../../i18n";

const props = defineProps<{
	modelValue: boolean;
	note: Note;
}>();

const emit = defineEmits<{
	(ev: "update:modelValue", v: boolean): void;
}>();

const el = ref<HTMLButtonElement | null>(null);

const label = computed(() => {
	const parts: string[] = [];
	if (props.note.text) {
		parts.push(i18n.t("_cw.chars", { count: props.note.text.length }));
	}
	if (props.note.files && props.note.files.length !== 0) {
		parts.push(i18n.t("_cw.files", { count: props.note.files.length }));
	}
	// Firefish also lists 投票 (poll) and 引用 (quote) here; Cinderfish has no
	// poll i18n section and the quote case is rare in a CW label, so both are
	// left out for now. TODO: add _cw sections once polls exist.
	return parts.join(", ");
});

function toggle() {
	emit("update:modelValue", !props.modelValue);
}

function focus() {
	el.value?.focus();
}

defineExpose({ focus });
</script>

<style scoped>
.cw-button {
	font-weight: 700;
	z-index: 5;
	border: 0;
	background: transparent;
	padding: 0;
	cursor: pointer;
}

.cw-button > span {
	background: var(--cwBg);
	color: var(--cwFg);
	padding: 0.3em 0.7em;
	border-radius: 999px;
	transition: background 0.2s, color 0.2s;
}

.cw-button .label {
	font-weight: 500;
}

.cw-button .label::before {
	content: "(";
}

.cw-button .label::after {
	content: ")";
}

.cw-button:hover > span,
.cw-button:focus > span {
	background: var(--cwFg);
	color: var(--cwBg);
}
</style>
