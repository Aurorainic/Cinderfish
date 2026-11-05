<!-- Cinderfish: replaces firefish@b425f61:packages/client/src/scripts/form.ts + MkFormDialog (AGPL-3.0-only) -->
<!--
	Widget settings dialog. Firefish builds it from the widget's form definition
	with `os.form()`; Cinderfish has neither the form renderer nor `os.form`, so
	the same field definition is rendered here with the shared Cf components.
-->
<template>
	<CfModal v-if="open" @closed="close">
		<div class="_panel config">
			<div class="header">{{ title }}</div>
			<div class="_monolithic_">
				<div class="_section">
					<template v-for="field in fields" :key="field.key">
						<CfSwitch
							v-if="field.type === 'boolean'"
							:model-value="boolValue(field)"
							@update:model-value="(value: boolean) => setValue(field, value)"
						>
							{{ field.label }}
						</CfSwitch>
						<CfInput
							v-else-if="field.type === 'number'"
							:model-value="numberValue(field)"
							type="number"
							:step="field.step ?? 1"
							@update:model-value="
								(value: string | number | null) => onNumber(field, value)
							"
						>
							<template #label>{{ field.label }}</template>
						</CfInput>
						<CfSelect
							v-else-if="field.type === 'enum'"
							:model-value="stringValue(field)"
							@update:model-value="
								(value: string | null) => onEnum(field, value)
							"
						>
							<template #label>{{ field.label }}</template>
							<option
								v-for="option in field.options ?? []"
								:key="String(option.value ?? '')"
								:value="String(option.value ?? '')"
							>
								{{ option.label }}
							</option>
						</CfSelect>
						<CfInput
							v-else
							:model-value="stringValue(field)"
							@update:model-value="
								(value: string | number | null) =>
									setValue(field, value == null ? '' : String(value))
							"
						>
							<template #label>{{ field.label }}</template>
						</CfInput>
					</template>
				</div>
			</div>
			<div class="footer">
				<CfButton @click="close">{{ t("cancel") }}</CfButton>
				<CfButton primary @click="ok">{{ t("ok") }}</CfButton>
			</div>
		</div>
	</CfModal>
</template>

<script setup lang="ts">
import { reactive, ref } from "vue";
import { CfButton, CfInput, CfSelect, CfSwitch } from "@/components/ui";
import { CfModal } from "@/os";
import { t } from "@/i18n";
import type { WidgetData } from "@/store/widgets";
import type { WidgetField } from "./index";

const props = withDefaults(
	defineProps<{
		title?: string;
		fields?: WidgetField[];
	}>(),
	{
		title: "",
		fields: () => [],
	},
);

const emit = defineEmits<{
	done: [values: WidgetData];
	closed: [];
}>();

const open = ref(true);

const values = reactive<WidgetData>({});
for (const field of props.fields) values[field.key] = field.default;

function boolValue(field: WidgetField): boolean {
	return values[field.key] === true;
}

function numberValue(field: WidgetField): number {
	const value = values[field.key];
	return typeof value === "number" && Number.isFinite(value)
		? value
		: Number(value ?? 0);
}

function stringValue(field: WidgetField): string {
	const value = values[field.key];
	return value == null ? "" : String(value);
}

function setValue(field: WidgetField, value: unknown): void {
	values[field.key] = value;
}

function onNumber(field: WidgetField, value: string | number | null): void {
	const parsed = typeof value === "number" ? value : Number(value ?? 0);
	values[field.key] = Number.isFinite(parsed) ? parsed : numberValue(field);
}

function onEnum(field: WidgetField, value: string | null): void {
	const options = field.options ?? [];
	const match = options.find((option) => String(option.value ?? "") === (value ?? ""));
	values[field.key] = match != null ? match.value : value;
}

function close(): void {
	open.value = false;
	emit("closed");
}

function ok(): void {
	emit("done", { ...values });
	close();
}
</script>

<style lang="scss" scoped>
.config {
	display: flex;
	flex-direction: column;
	width: min(400px, calc(100vw - 32px));
	max-height: min(520px, calc(100vh - 64px));
	overflow: auto;

	> .header {
		padding: 16px;
		font-weight: bold;
		border-bottom: solid 1px var(--divider);
	}

	> .footer {
		display: flex;
		gap: 8px;
		justify-content: flex-end;
		padding: 16px;
		border-top: solid 1px var(--divider);
	}
}
</style>
