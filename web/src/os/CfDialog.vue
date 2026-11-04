<template>
	<CfModal
		ref="modal"
		:close-on-backdrop="true"
		@click="onBackdrop"
		@closed="emit('closed')"
	>
		<div class="body _shadow" :class="{ mini: isYesNo }" @click.stop>
			<div v-if="iconName" class="icon" :class="type">
				<i :class="`ph-bold ph-${iconName}`"></i>
			</div>
			<div v-if="title" class="title">{{ title }}</div>
			<div v-if="text" class="text">
				<pre v-if="!isPlaintext" class="text-content">{{ text }}</pre>
				<template v-else>{{ text }}</template>
			</div>
			<input
				v-if="input && input.type !== 'paragraph'"
				ref="inputEl"
				v-model="inputValue"
				class="input"
				:type="input.type ?? 'text'"
				:placeholder="input.placeholder ?? undefined"
				:autocomplete="input.autocomplete"
				:minlength="input.minLength"
				:maxlength="input.maxLength"
				@keydown.enter.prevent="ok"
				@keydown.esc.prevent="cancel"
			/>
			<textarea
				v-else-if="input && input.type === 'paragraph'"
				ref="inputEl"
				v-model="inputValue"
				class="input textarea"
				:placeholder="input.placeholder ?? undefined"
				@keydown.esc.prevent="cancel"
			></textarea>
			<div class="buttons">
				<button v-if="showCancelButton" class="button _button" @click="cancel">
					{{ cancelText ?? t("cancel") }}
				</button>
				<button
					class="button primary"
					:class="{ _buttonPrimary: !isYesNo }"
					@click="ok"
				>
					{{ okText ?? (isYesNo ? t("yes") : t("ok")) }}
				</button>
			</div>
		</div>
	</CfModal>
</template>

<script lang="ts" setup>
import { computed, onMounted, ref } from "vue";
import { t } from "../i18n";
import CfModal from "./CfModal.vue";
import type { DialogInput, DialogResult, DialogType } from "./types";

const props = withDefaults(
	defineProps<{
		type?: DialogType;
		title?: string | null;
		text?: string | null;
		isPlaintext?: boolean;
		showCancelButton?: boolean;
		isYesNo?: boolean;
		okText?: string;
		cancelText?: string;
		input?: DialogInput | null;
	}>(),
	{
		type: "info",
		title: null,
		text: null,
		isPlaintext: true,
		showCancelButton: false,
		isYesNo: false,
		okText: undefined,
		cancelText: undefined,
		input: null,
	},
);

const emit = defineEmits<{
	done: [result: DialogResult];
	closed: [];
}>();

const modal = ref<InstanceType<typeof CfModal> | null>(null);
const inputEl = ref<HTMLInputElement | HTMLTextAreaElement | null>(null);
const inputValue = ref<string>(
	props.input?.default == null ? "" : String(props.input.default),
);

const iconName = computed<string | null>(() => {
	switch (props.type) {
		case "error":
			return "warning-circle";
		case "warning":
			return "warning";
		case "success":
			return "check-circle";
		case "question":
			return "question";
		case "waiting":
			return null;
		default:
			return "info";
	}
});

function finish(result: DialogResult): void {
	emit("done", result);
	modal.value?.close();
}

function ok(): void {
	if (props.input != null) {
		finish({ canceled: false, result: inputValue.value });
	} else {
		finish({ canceled: false });
	}
}

function cancel(): void {
	finish({ canceled: true });
}

function onBackdrop(): void {
	if (props.showCancelButton || props.input == null) cancel();
}

onMounted(() => {
	if (props.input != null) {
		window.setTimeout(() => inputEl.value?.focus(), 10);
	}
});
</script>

<style lang="scss" scoped>
.body {
	inline-size: min(420px, 100%);
	max-block-size: calc(100dvb - 64px);
	overflow: auto;
	padding: 32px;
	background: var(--panel);
	border-radius: var(--radius);
	box-shadow: 0 4px 32px var(--shadow);
	text-align: center;
	box-sizing: border-box;

	&.mini {
		inline-size: min(360px, 100%);
	}
}

.icon {
	margin-block-end: 16px;
	font-size: 2.4em;
	line-height: 1;

	&.error {
		color: var(--error);
	}
	&.warning {
		color: var(--warn);
	}
	&.success {
		color: var(--success);
	}
	&.question {
		color: var(--accent);
	}
}

.title {
	font-weight: bold;
	font-size: 1.1em;
	margin-block-end: 12px;
	color: var(--fg);
}

.text {
	color: var(--fgTransparentWeak);
	font-size: 0.95em;
	margin-block-end: 16px;
	overflow-wrap: anywhere;
}

.text-content {
	margin: 0;
	font: inherit;
	white-space: pre-wrap;
	text-align: start;
}

.input {
	inline-size: 100%;
	padding: 10px 12px;
	margin-block-end: 16px;
	background: var(--bg);
	color: var(--fg);
	border: solid 1px var(--inputBorder);
	border-radius: 6px;
	font: inherit;
	outline: none;

	&:focus {
		border-color: var(--accent);
	}
}

.textarea {
	min-block-size: 96px;
	resize: vertical;
}

.buttons {
	display: flex;
	justify-content: flex-end;
	gap: 8px;
}

.button {
	padding: 8px 16px;
	border-radius: 6px;
	font-weight: bold;
	background: var(--buttonBg);
	color: var(--fg);
	transition: background 0.2s;

	&:hover {
		background: var(--buttonHoverBg);
	}

	&.primary {
		background: var(--accent);
		color: var(--fgOnAccent);
	}
}
</style>
