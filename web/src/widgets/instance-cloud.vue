<!-- Ported from firefish@b425f61:packages/client/src/widgets/instance-cloud.vue (AGPL-3.0-only) -->
<template>
	<WidgetCard
		:title="t('_widgets.instanceCloud')"
		icon="ph-bold ph-cloud"
		:show-header="widgetProps.showHeader === true"
	>
		<div class="rows">
			<div class="row">
				<span class="label">{{ t("federation") }}</span>
				<span class="value">{{ federationLabel }}</span>
			</div>
			<div class="row">
				<span class="label">{{ t("host") }}</span>
				<span class="value">{{ instance.host }}</span>
			</div>
		</div>
		<CfInfo>
			实例云未提供：服务器没有 `federation/instances` 接口，无法列出已交换通讯的服务器。
		</CfInfo>
	</WidgetCard>
</template>

<script setup lang="ts">
// Firefish lists the 25 instances with the newest `lastCommunicatedAt` from
// `federation/instances`. That endpoint does not exist on Cinderfish and the
// contract forbids inventing entries, so the widget reports the real federation
// mode from `meta` and says the cloud itself is unavailable.
import WidgetCard from "./WidgetCard.vue";
import { CfInfo } from "@/components/ui";
import { instance } from "@/instance";
import { t } from "@/i18n";
import { computed } from "vue";
import { useWidgetPropsManager, type WidgetComponentExpose, type WidgetField } from "./index";
import type { Widget } from "@/store/widgets";

const name = "instanceCloud";

const props = defineProps<{ widget?: Widget }>();
const emit = defineEmits<{ updateProps: [data: Record<string, unknown>] }>();

const fields: WidgetField[] = [
	{ key: "showHeader", label: "显示标题栏", type: "boolean", default: true },
];

const { widgetProps, configure } = useWidgetPropsManager(
	name,
	fields,
	props.widget,
	(data) => emit("updateProps", data),
);

const federationLabel = computed(() => {
	switch (instance.federation) {
		case "all":
			return "公开";
		case "specified":
			return "仅白名单";
		case "none":
			return "关闭";
		default:
			return "?";
	}
});

defineExpose<WidgetComponentExpose>({
	name,
	configure,
	id: props.widget?.id ?? null,
});
</script>

<style lang="scss" scoped>
.rows {
	display: flex;
	flex-direction: column;
	gap: 8px;
	padding: 12px 14px;

	> .row {
		display: flex;
		justify-content: space-between;
		gap: 8px;
		font-size: 0.9em;

		> .label {
			opacity: 0.7;
		}

		> .value {
			overflow: hidden;
			white-space: nowrap;
			text-overflow: ellipsis;
		}
	}
}
</style>
