// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/components/form/switch.vue
//
// Scoped CSS replaces the SCSS block; the visual contract (33px track, knob
// travel, disabled dimming) is kept.
<template>
	<label class="switch">
		<input
			type="checkbox"
			:checked="checked"
			:disabled="disabled"
			@change="(x) => toggle(x)"
		/>
		<span class="track" aria-hidden="true">
			<span class="knob"></span>
		</span>
		<span class="text">
			<span class="label"><slot name="label"></slot><slot></slot></span>
			<span v-if="$slots.caption" class="caption"><slot name="caption"></slot></span>
		</span>
	</label>
</template>

<script lang="ts" setup>
import { computed, toValue, type Ref } from "vue";

const props = defineProps<{
	modelValue: boolean | Ref<boolean> | { get: () => boolean; set: (value: boolean) => void };
	disabled?: boolean;
}>();

const emit = defineEmits<{
	"update:modelValue": [v: boolean];
}>();

const checked = computed(() =>
	typeof props.modelValue === "object" && "get" in props.modelValue
		? props.modelValue.get()
		: toValue(props.modelValue as boolean | Ref<boolean>),
);

function toggle(x: Event) {
	if (props.disabled) return;
	const next = (x.target as HTMLInputElement).checked;
	if (typeof props.modelValue === "object" && "set" in props.modelValue) {
		props.modelValue.set(next);
	} else {
		emit("update:modelValue", next);
	}
}
</script>

<style scoped>
.switch {
	position: relative;
	display: flex;
	align-items: flex-start;
	gap: 10px;
	padding: 8px 0;
	cursor: pointer;
	transition: opacity 0.2s ease;
}

.switch:has(input:disabled) {
	opacity: 0.5;
	cursor: default;
}

.switch > input {
	position: absolute;
	inline-size: 32px;
	block-size: 23px;
	margin: 0;
	opacity: 0;
}

.track {
	position: relative;
	flex-shrink: 0;
	display: inline-flex;
	margin-top: 2px;
	inline-size: 32px;
	block-size: 23px;
	outline: none;
	border-radius: 999px;
	background: var(--panel-2);
	background-size: 100% 200%;
	box-shadow: inset 0 0 0 1px var(--border);
	transition: background 0.2s ease;
}

.knob {
	position: absolute;
	inset-block-start: 3px;
	inset-inline-start: 3px;
	inline-size: 17px;
	block-size: 17px;
	border-radius: 999px;
	background: var(--text-weak);
	box-shadow: 0 1px 3px rgb(0 0 0 / 0.2);
	transition: inset-inline-start 0.2s cubic-bezier(0.4, 0, 0.6, 1), background 0.2s ease;
}

input:checked + .track {
	background: var(--accent);
	box-shadow: inset 0 0 0 1px var(--accent);
}

input:checked + .track .knob {
	inset-inline-start: 12px;
	background: var(--on-accent, #fff);
}

input:focus-visible + .track {
	outline: var(--focus-outline);
	outline-offset: 2px;
}

.text {
	display: flex;
	flex-direction: column;
	min-width: 0;
}

.label {
	color: var(--text);
	font-size: var(--fs-sm);
	line-height: 1.5;
}

.caption {
	color: var(--text-faint);
	font-size: var(--fs-xs);
	line-height: 1.5;
}
</style>
