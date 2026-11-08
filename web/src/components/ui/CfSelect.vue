<!-- wave-3: themed popup list replaces native <select>; native element kept hidden for form semantics. -->
<template>
	<div ref="rootEl" class="vblkjoeq">
		<div class="label"><slot name="label"></slot></div>
		<div
			ref="triggerEl"
			v-adaptive-border
			class="input"
			:class="{ inline, disabled, focused: isOpen }"
			role="combobox"
			:aria-expanded="isOpen"
			aria-haspopup="listbox"
			:tabindex="disabled ? -1 : 0"
			@click="toggle"
			@keydown.stop="onKeydown"
		>
			<div v-if="$slots.prefix" class="prefix"><slot name="prefix"></slot></div>
			<div class="display">
				<span class="selected-text">{{ selectedLabel }}</span>
			</div>
			<div class="suffix">
				<slot name="suffix"></slot>
				<i class="ph-bold ph-caret-down" :class="{ rotated: isOpen }"></i>
			</div>
		</div>
		<select
			ref="nativeEl"
			v-model="value"
			class="sr-only"
			aria-hidden="true"
			tabindex="-1"
			:disabled="disabled"
			:required="required"
		>
			<slot></slot>
		</select>
		<div class="caption"><slot name="caption"></slot></div>
		<CfButton v-if="manualSave && changed" primary class="save" @click="commit">
			{{ t("save") }}
		</CfButton>
		<Teleport to="body">
			<div
				v-if="isOpen"
				ref="listEl"
				class="cf-select-list _shadow"
				:style="listStyle"
				role="listbox"
			>
				<button
					v-for="opt in options"
					:key="opt.value ?? opt.text"
					role="option"
					class="item _button"
					:class="{ selected: opt.value === value }"
					:aria-selected="opt.value === value"
					:disabled="opt.disabled"
					@click="select(opt.value)"
				>
					<span class="text">{{ opt.text }}</span>
					<i v-if="opt.value === value" class="ph-bold ph-check"></i>
				</button>
			</div>
		</Teleport>
	</div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue";
import CfButton from "./CfButton.vue";
import { t } from "../../i18n";

interface SelectOption {
	value: string;
	text: string;
	disabled?: boolean;
}

const props = withDefaults(
	defineProps<{
		modelValue: string | null;
		required?: boolean;
		readonly?: boolean;
		disabled?: boolean;
		placeholder?: string;
		autofocus?: boolean;
		inline?: boolean;
		manualSave?: boolean;
		small?: boolean;
		large?: boolean;
	}>(),
	{
		required: false,
		readonly: false,
		disabled: false,
		placeholder: undefined,
		autofocus: false,
		inline: false,
		manualSave: false,
		small: false,
		large: false,
	},
);

const emit = defineEmits<{
	change: [event: Event];
	"update:modelValue": [value: string | null];
}>();

const value = ref<string | null>(props.modelValue);
const isOpen = ref(false);
const changed = ref(false);
const rootEl = ref<HTMLElement | null>(null);
const triggerEl = ref<HTMLElement | null>(null);
const nativeEl = ref<HTMLSelectElement | null>(null);
const listEl = ref<HTMLElement | null>(null);
const listStyle = ref<Record<string, string>>({});

// Collect <option> elements from the hidden native select after mount.
const options = ref<SelectOption[]>([]);

function syncOptions(): void {
	const sel = nativeEl.value;
	if (sel == null) return;
	options.value = Array.from(sel.options).map((o) => ({
		value: o.value,
		text: o.text,
		disabled: o.disabled,
	}));
}

const selectedLabel = computed(() => {
	const match = options.value.find((o) => o.value === value.value);
	return match?.text ?? props.placeholder ?? "";
});

function positionList(): void {
	const trigger = triggerEl.value;
	if (trigger == null) return;
	const rect = trigger.getBoundingClientRect();
	const spaceBelow = window.innerHeight - rect.bottom;
	const spaceAbove = rect.top;
	const openUp = spaceBelow < 200 && spaceAbove > spaceBelow;
	if (openUp) {
		listStyle.value = {
			position: "fixed",
			inlineSize: `${rect.width}px`,
			left: `${rect.left}px`,
			bottom: `${window.innerHeight - rect.top}px`,
			top: "auto",
		};
	} else {
		listStyle.value = {
			position: "fixed",
			inlineSize: `${rect.width}px`,
			left: `${rect.left}px`,
			top: `${rect.bottom}px`,
		};
	}
}

function open(): void {
	if (props.disabled || props.readonly) return;
	syncOptions();
	isOpen.value = true;
	nextTick(() => positionList());
}

function close(): void {
	isOpen.value = false;
}

function toggle(): void {
	isOpen.value ? close() : open();
}

function select(val: string): void {
	value.value = val;
	changed.value = true;
	close();
	if (!props.manualSave) {
		emit("update:modelValue", val);
	}
}

function commit(): void {
	changed.value = false;
	emit("update:modelValue", value.value);
}

function onKeydown(event: KeyboardEvent): void {
	if (event.key === "Enter" || event.key === " ") {
		event.preventDefault();
		toggle();
	} else if (event.key === "Escape") {
		close();
	} else if (event.key === "ArrowDown" || event.key === "ArrowUp") {
		event.preventDefault();
		if (!isOpen.value) { open(); return; }
		const idx = options.value.findIndex((o) => o.value === value.value);
		const next = event.key === "ArrowDown"
			? Math.min(idx + 1, options.value.length - 1)
			: Math.max(idx - 1, 0);
		const opt = options.value[next];
		if (opt && !opt.disabled) select(opt.value);
	}
}

function onOutsidePointer(event: PointerEvent): void {
	if (
		rootEl.value != null && !rootEl.value.contains(event.target as Node) &&
		listEl.value != null && !listEl.value.contains(event.target as Node)
	) {
		close();
	}
}

watch(() => props.modelValue, (next) => { value.value = next; });

onMounted(() => {
	nextTick(() => syncOptions());
	if (props.autofocus) triggerEl.value?.focus();
	window.addEventListener("pointerdown", onOutsidePointer, true);
});

onBeforeUnmount(() => {
	window.removeEventListener("pointerdown", onOutsidePointer, true);
});
</script>

<style lang="scss" scoped>
.vblkjoeq {
	.label {
		font-size: 0.85em;
		padding-block-end: 8px;
		user-select: none;
		&:empty { display: none; }
	}

	.input {
		position: relative;
		display: flex;
		align-items: center;
		border: solid 1px var(--inputBorder);
		// `v-adaptive-border` picks a visible edge on flush surfaces and the
		// field's own fill on contrasting ones; the token is the fallback.
		border-color: var(--cf-adaptive-border-color, var(--inputBorder));
		border-radius: 6px;
		background: var(--bg);
		cursor: pointer;
		transition: border-color 0.2s;
		user-select: none;

		&.focused { border-color: var(--accent); }
		&.disabled { opacity: 0.6; pointer-events: none; cursor: default; }
		&.inline { display: inline-flex; }

		&:focus-visible { outline: 2px solid var(--accent); outline-offset: 2px; }
	}

	.display {
		flex: 1;
		min-inline-size: 0;
		padding: 9px 0 9px 12px;
		color: var(--fg);
		font: inherit;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.prefix {
		padding-inline-start: 10px;
		color: var(--fgTransparentWeak);
	}

	.suffix {
		padding-inline-end: 10px;
		display: flex;
		align-items: center;
		gap: 4px;
		color: var(--fgTransparentWeak);
		flex-shrink: 0;

		> .ph-caret-down {
			transition: transform 0.2s;
			&.rotated { transform: rotate(180deg); }
		}
	}

	.sr-only {
		position: absolute;
		inset: 0;
		opacity: 0;
		pointer-events: none;
		inline-size: 100%;
		block-size: 100%;
	}

	.caption {
		padding-block-start: 6px;
		font-size: 0.85em;
		color: var(--fgTransparentWeak);
		&:empty { display: none; }
	}

	.save { margin-block-start: 8px; }
}
</style>

<style lang="scss">
.cf-select-list {
	z-index: 9999;
	min-inline-size: 120px;
	max-block-size: min(70dvb, 360px);
	overflow-y: auto;
	padding: 6px;
	background: var(--popup);
	border-radius: var(--radius);
	box-shadow: 0 4px 32px var(--shadow);
	animation: cf-select-in 0.1s ease;

	.item {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 10px;
		inline-size: 100%;
		min-block-size: 40px;
		padding: 8px 12px;
		border-radius: 6px;
		color: var(--fg);
		text-align: start;
		font-size: 1rem;

		&:hover { background: var(--listItemHoverBg); }

		&.selected {
			color: var(--accent);
			> .text { font-weight: bold; }
		}

		&:disabled { opacity: 0.45; pointer-events: none; }

		> .text {
			flex: 1;
			overflow: hidden;
			text-overflow: ellipsis;
			white-space: nowrap;
		}
	}
}

@keyframes cf-select-in {
	from { opacity: 0; transform: scale(0.97) translateY(-4px); }
	to   { opacity: 1; transform: scale(1)    translateY(0); }
}
</style>
