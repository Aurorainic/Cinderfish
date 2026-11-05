<!-- Ported from firefish@b425f61:packages/client/src/components/MkShowMoreButton.vue (AGPL-3.0-only) -->
<template>
	<button
		ref="el"
		class="_button"
		:class="{ fade: modelValue, showLess: !modelValue }"
		@click.stop="toggle"
	>
		<span>{{ modelValue ? t("showMore") : t("showLess") }}</span>
	</button>
</template>

<script lang="ts" setup>
import { ref } from "vue";
import { t } from "@/i18n";

const props = defineProps<{
	modelValue: boolean;
}>();

const el = ref<HTMLElement | null>(null);

const emit = defineEmits<{
	"update:modelValue": [value: boolean];
}>();

function toggle(): void {
	emit("update:modelValue", !props.modelValue);
}

function focus(): void {
	el.value?.focus();
}

defineExpose({ focus });
</script>

<style lang="scss" scoped>
.fade {
	display: block;
	position: absolute;
	bottom: 0;
	left: 0;
	width: 100%;
	padding: 20px;
	margin-bottom: -10px;
	z-index: 5;

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

.showLess {
	width: 100%;
	position: sticky;
	bottom: calc(var(--stickyBottom) - 1em);
	padding: 20px;
	z-index: 5;

	> span {
		display: inline-block;
		background: var(--panel);
		padding: 6px 10px;
		font-size: 0.8em;
		border-radius: 999px;
		box-shadow: 0 0 7px 7px var(--bg);
	}
}
</style>
