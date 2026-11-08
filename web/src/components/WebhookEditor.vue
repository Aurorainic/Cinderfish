<!-- SPDX-License-Identifier: AGPL-3.0-only -->
<template>
	<form class="webhook-editor" @submit.prevent="submit">
		<CfInput v-model="name" class="field" :disabled="saving" required>
			<template #label>名称</template>
			<template #caption>最多 100 个字符，只用于你自己识别。</template>
		</CfInput>
		<CfInput v-model="url" class="field" type="url" :disabled="saving" required>
			<template #label>接收地址</template>
			<template #caption>必须是 http(s) 地址；实例会向它 POST JSON。</template>
		</CfInput>
		<CfInput v-model="secret" class="field" :disabled="saving" required>
			<template #label>签名密钥</template>
			<template #caption
				>每次请求放在 <code>X-Firefish-Hook-Secret</code> 头里，用来确认请求来自本实例。</template
			>
		</CfInput>

		<fieldset class="events" :disabled="saving">
			<legend>触发事件</legend>
			<CfSwitch
				v-for="event in supported"
				:key="event.value"
				:model-value="on.includes(event.value)"
				:disabled="saving"
				@update:model-value="(checked) => setEvent(event.value, checked)"
			>
				{{ event.label }}
				<template #caption>{{ event.caption }}</template>
			</CfSwitch>
		</fieldset>
		<p class="hint">
			上游还有 follow / reaction / mention 等事件；本实例只产生上面三种，
			其余事件即使写进配置也不会触发，所以这里不提供。
		</p>

		<CfSwitch v-if="initial != null" v-model="active" class="field">
			启用
			<template #caption>停用后不再投递，配置保留。</template>
		</CfSwitch>

		<CfInfo v-if="error !== null" warn class="field">{{ error }}</CfInfo>

		<div class="actions">
			<CfButton type="submit" primary :disabled="saving || on.length === 0">
				{{ saving ? "保存中…" : submitLabel }}
			</CfButton>
			<slot name="actions"></slot>
		</div>
	</form>
</template>

<script setup lang="ts">
// Shared form for creating and editing a webhook. The event checkboxes only
// list events this instance actually produces (`note`, `reply`, `renote`);
// listing the rest would be a switch that never fires.
import { ref } from "vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfInput from "@/components/ui/CfInput.vue";
import CfSwitch from "@/components/ui/CfSwitch.vue";
import type { Webhook } from "@/api/types";

const props = withDefaults(
	defineProps<{
		initial?: Webhook | null;
		saving?: boolean;
		error?: string | null;
		submitLabel?: string;
	}>(),
	{
		initial: null,
		saving: false,
		error: null,
		submitLabel: "保存",
	},
);

const emit = defineEmits<{
	submit: [
		payload: {
			name: string;
			url: string;
			secret: string;
			on: string[];
			active: boolean;
		},
	];
}>();

const supported = [
	{ value: "note", label: "新帖子", caption: "发布一条普通帖子时触发" },
	{ value: "reply", label: "回复", caption: "回复别人的帖子（含自己的）时触发" },
	{ value: "renote", label: "转帖", caption: "转帖或引用时触发" },
];

const name = ref<string | number | null>(props.initial?.name ?? "");
const url = ref<string | number | null>(props.initial?.url ?? "");
const secret = ref<string | number | null>(props.initial?.secret ?? "");
const on = ref<string[]>(
	props.initial?.on?.filter((event) => supported.some((s) => s.value === event)) ?? [
		"note",
	],
);
const active = ref(props.initial?.active ?? true);

function setEvent(value: string, checked: boolean): void {
	if (checked) {
		if (!on.value.includes(value)) on.value = [...on.value, value];
	} else {
		on.value = on.value.filter((event) => event !== value);
	}
}

function submit(): void {
	const events = on.value.filter((event) => supported.some((s) => s.value === event));
	if (events.length === 0) return;
	emit("submit", {
		name: String(name.value ?? "").trim(),
		url: String(url.value ?? "").trim(),
		secret: String(secret.value ?? "").trim(),
		on: events,
		active: active.value,
	});
}
</script>

<style lang="scss" scoped>
.webhook-editor {
	display: flex;
	flex-direction: column;
	gap: 12px;

	.field {
		margin: 0;
	}
}

.events {
	display: flex;
	flex-direction: column;
	gap: 8px;
	margin: 0;
	padding: 12px 14px;
	border: 1px solid var(--divider);
	border-radius: var(--radius);

	legend {
		padding: 0 6px;
		font-size: 0.85em;
		opacity: 0.75;
	}
}

.hint {
	margin: 0;
	font-size: 0.82em;
	opacity: 0.65;
}

.actions {
	display: flex;
	gap: 8px;
	flex-wrap: wrap;
}
</style>
