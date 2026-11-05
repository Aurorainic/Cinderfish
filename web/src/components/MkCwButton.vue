<!-- Ported from firefish@b425f61:packages/client/src/components/MkCwButton.vue (AGPL-3.0-only) -->
<template>
	<button
		ref="el"
		class="_button"
		:class="{ showLess: modelValue, fade: !modelValue }"
		@click.stop="toggle"
	>
		<span
			>{{ modelValue ? t("_cw.hide") : t("_cw.show") }}
			<span v-if="!modelValue">{{ label }}</span>
		</span>
	</button>
</template>

<script lang="ts" setup>
// Degradation: Firefish measures the text with `stringz` (grapheme aware).
// stringz is not a dependency of this port, so code points are counted instead.
import { computed, ref } from "vue";
import type { Note } from "@/api/types";
import { concat } from "@/scripts/array";
import { t } from "@/i18n";

const props = defineProps<{
	modelValue: boolean;
	note: Note & { poll?: unknown; renote?: unknown };
}>();

const emit = defineEmits<{
	(ev: "update:modelValue", v: boolean): void;
}>();

const el = ref<HTMLElement>();

const label = computed(() => {
	return concat([
		props.note.text
			? [t("_cw.chars", { count: [...props.note.text].length })]
			: [],
		props.note.files && props.note.files.length !== 0
			? [t("_cw.files", { count: props.note.files.length })]
			: [],
		props.note.poll != null ? [t("poll")] : [],
		props.note.renote != null ? [t("quoteAttached")] : [],
	] as string[][]).join(", ");
});

const toggle = () => {
	emit("update:modelValue", !props.modelValue);
};

function focus() {
	el.value?.focus();
}

defineExpose({
	focus,
});
</script>

<style lang="scss" scoped>
._button {
	font-weight: 700;
	z-index: 5;
	> span {
		background: var(--cwBg) !important;
		color: var(--cwFg);
		transition:
			background 0.2s,
			color 0.2s;
		> span {
			font-weight: 500;
			&::before {
				content: "(";
			}
			&::after {
				content: ")";
			}
		}
	}
	&:hover > span,
	&:focus > span {
		background: var(--cwFg) !important;
		color: var(--cwBg) !important;
	}

	&.fade {
		display: block;
		position: absolute;
		bottom: 0;
		left: 0;
		width: 100%;
		> span {
			display: inline-block;
			background: var(--panel);
			padding: 0.4em 1em;
			font-size: 0.8em;
			border-radius: 999px;
			box-shadow: 0 2px 6px rgb(0 0 0 / 20%);
		}
		&:hover {
			> span {
				background: var(--panelHighlight);
			}
		}
	}
	&.showLess {
		width: 100%;
		position: sticky;
		bottom: calc(var(--stickyBottom) - 1em);
		padding: 20px;

		> span {
			display: inline-block;
			background: var(--panel);
			padding: 6px 10px;
			font-size: 0.8em;
			border-radius: 999px;
			box-shadow: 0 0 7px 7px var(--bg);
		}
	}
}
</style>
