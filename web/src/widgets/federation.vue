<!-- Ported from firefish@b425f61:packages/client/src/widgets/federation.vue (AGPL-3.0-only) -->
<template>
	<WidgetCard
		:title="t('_widgets.federation')"
		icon="ph-bold ph-planet"
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
		<div v-else class="instances">
			<div v-for="instance in instances" :key="instance.host" class="instance">
				<div class="icon placeholder">
					<i class="ph-bold ph-planet"></i>
				</div>
				<div class="body">
					<a
						class="host"
						:href="`https://${instance.host}`"
						target="_blank"
						rel="noopener noreferrer"
						:title="instance.host"
					>{{ instance.host }}</a>
					<p>
						{{ instance.softwareName || "?" }}
						{{ instance.softwareVersion || "" }}
					</p>
				</div>
			</div>
		</div>
	</WidgetCard>
</template>

<script setup lang="ts">
// Firefish draws a per-instance request chart from `charts/instance`. That
// endpoint has no time-series store here, and `federation/instances` carries no
// icon URL, so the widget shows the list only (host and software), which is
// real data derived from the stored remote users and notes.
import { ref } from "vue";
import WidgetCard from "./WidgetCard.vue";
import { CfButton, CfLoading } from "@/components/ui";
import { api } from "@/api/client";
import type { FederationInstance } from "@/api/types";
import { useInterval } from "@/composables/use-interval";
import { t } from "@/i18n";
import { useWidgetPropsManager, type WidgetComponentExpose, type WidgetField } from "./index";
import type { Widget } from "@/store/widgets";

const name = "federation";

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
	api<FederationInstance[]>("federation/instances", { limit: 5 })
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

.instances {
	> .instance {
		display: flex;
		align-items: center;
		padding: 10px 14px;
		border-bottom: solid 0.5px var(--divider);

		&:last-child {
			border-bottom: none;
		}

		> .icon {
			display: flex;
			align-items: center;
			justify-content: center;
			width: 28px;
			height: 28px;
			border-radius: 4px;
			margin-right: 8px;
			background: var(--panelHighlight);
			opacity: 0.7;
		}

		> .body {
			flex: 1;
			min-width: 0;
			font-size: 0.9em;

			> .host {
				display: block;
				white-space: nowrap;
				overflow: hidden;
				text-overflow: ellipsis;
			}

			> p {
				margin: 0;
				font-size: 75%;
				opacity: 0.7;
				white-space: nowrap;
				overflow: hidden;
				text-overflow: ellipsis;
			}
		}
	}
}
</style>
