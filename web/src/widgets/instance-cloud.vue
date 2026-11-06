<!-- Ported from firefish@b425f61:packages/client/src/widgets/instance-cloud.vue (AGPL-3.0-only) -->
<template>
	<WidgetCard
		:title="t('_widgets.instanceCloud')"
		icon="ph-bold ph-cloud"
		:show-header="widgetProps.showHeader === true"
	>
		<CfLoading v-if="fetching && instances.length === 0" />
		<div v-else-if="failed" class="state">
			<div>{{ t("somethingHappened") }}</div>
			<CfButton @click="loadInstances">{{ t("retry") }}</CfButton>
		</div>
		<div v-else-if="instances.length === 0" class="state">
			{{ t("nothing") }}
		</div>
		<div v-else class="cloud">
			<a
				v-for="instance in instances"
				:key="instance.host"
				class="node"
				:href="`https://${instance.host}`"
				target="_blank"
				rel="noopener noreferrer"
				:title="`${instance.usersCount} ${t('users')} · ${instance.notesCount} ${t('notes')}`"
			>{{ instance.host }}</a>
		</div>
	</WidgetCard>
</template>

<script setup lang="ts">
// Firefish renders the 25 instances with the newest `lastCommunicatedAt` as an
// icon cloud, using the icon URL in the response. `federation/instances` here
// orders by user count and carries no icon, so the cloud becomes a wrapped list
// of host names with the real counts in the tooltip.
import { ref } from "vue";
import WidgetCard from "./WidgetCard.vue";
import { CfButton, CfLoading } from "@/components/ui";
import { api } from "@/api/client";
import type { FederationInstance } from "@/api/types";
import { useInterval } from "@/composables/use-interval";
import { t } from "@/i18n";
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

const instances = ref<FederationInstance[]>([]);
const fetching = ref(true);
const failed = ref(false);

function loadInstances(): void {
	fetching.value = true;
	failed.value = false;
	api<FederationInstance[]>("federation/instances", { limit: 25 })
		.then((result) => {
			instances.value = result;
		})
		.catch(() => {
			failed.value = true;
		})
		.finally(() => {
			fetching.value = false;
		});
}

useInterval(loadInstances, 1000 * 60, { immediate: true, afterMounted: true });

defineExpose<WidgetComponentExpose>({
	name,
	configure,
	id: props.widget?.id ?? null,
});
</script>

<style lang="scss" scoped>
.state {
	display: flex;
	flex-direction: column;
	align-items: center;
	gap: 8px;
	padding: 16px 14px;
	opacity: 0.8;
}

.cloud {
	display: flex;
	flex-wrap: wrap;
	gap: 6px;
	padding: 12px 14px;

	> .node {
		display: inline-block;
		max-width: 100%;
		padding: 2px 8px;
		border: solid 0.5px var(--divider);
		border-radius: 999px;
		font-size: 0.82em;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
}
</style>
