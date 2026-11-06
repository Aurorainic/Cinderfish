<!-- Ported from firefish@b425f61:packages/client/src/widgets/job-queue.vue (AGPL-3.0-only) -->
<template>
	<WidgetCard
		:title="t('_widgets.jobQueue')"
		icon="ph-bold ph-tray"
		:show-header="widgetProps.showHeader === true"
	>
		<CfLoading v-if="fetching && info == null && forbidden === false" />
		<div v-else-if="forbidden" class="state">需要管理员权限才能查看投递队列。</div>
		<div v-else-if="failed" class="state">
			<div>{{ t("somethingHappened") }}</div>
			<CfButton @click="loadQueue">{{ t("retry") }}</CfButton>
		</div>
		<template v-else-if="info != null">
			<div class="rows">
				<div class="row">
					<span class="label">待投递</span>
					<span class="value">{{ info.pending }}</span>
				</div>
				<div class="row">
					<span class="label">到期可投</span>
					<span class="value">{{ info.dueNow }}</span>
				</div>
				<div class="row">
					<span class="label">已放弃</span>
					<span class="value" :class="{ bad: info.failed > 0 }">{{ info.failed }}</span>
				</div>
			</div>
			<div v-if="info.recent.length > 0" class="recent">
				<div v-for="entry in info.recent.slice(0, 5)" :key="entry.id" class="entry">
					<span class="host" :title="entry.host">{{ entry.host }}</span>
					<span class="attempts">#{{ entry.attempts }}</span>
					<span v-if="entry.lastError" class="error" :title="entry.lastError">
						{{ entry.lastError }}
					</span>
				</div>
			</div>
		</template>
	</WidgetCard>
</template>

<script setup lang="ts">
// Firefish subscribes to the `queueStats` Streaming channel and shows every
// queue (inbox, deliver, …). Cinderfish has one delivery queue in PostgreSQL and
// no queueStats channel, so the widget polls the admin endpoint and shows the
// same three counters plus the newest entries. Non-admins see a notice instead
// of an empty chart.
import { ref } from "vue";
import WidgetCard from "./WidgetCard.vue";
import { CfButton, CfLoading } from "@/components/ui";
import { api } from "@/api/client";
import type { DeliverQueueInfo } from "@/api/types";
import { useInterval } from "@/composables/use-interval";
import { t } from "@/i18n";
import { useWidgetPropsManager, type WidgetComponentExpose, type WidgetField } from "./index";
import type { Widget } from "@/store/widgets";

const name = "jobQueue";

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

const info = ref<DeliverQueueInfo | null>(null);
const fetching = ref(true);
const failed = ref(false);
const forbidden = ref(false);

function loadQueue(): void {
	fetching.value = true;
	failed.value = false;
	api<DeliverQueueInfo>("admin/deliver-queue")
		.then((result) => {
			info.value = result;
			forbidden.value = false;
		})
		.catch((error: unknown) => {
			const code =
				error != null && typeof error === "object" && "code" in error
					? (error as { code?: unknown }).code
					: null;
			forbidden.value = code === "FORBIDDEN" || code === "PERMISSION_DENIED";
			if (!forbidden.value) failed.value = true;
		})
		.finally(() => {
			fetching.value = false;
		});
}

useInterval(loadQueue, 1000 * 30, { immediate: true, afterMounted: true });

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
	text-align: center;
}

.rows {
	display: flex;
	flex-direction: column;
	gap: 8px;
	padding: 12px 14px;

	> .row {
		display: flex;
		justify-content: space-between;
		font-size: 0.9em;

		> .label {
			opacity: 0.7;
		}

		> .value.bad {
			color: var(--error);
		}
	}
}

.recent {
	border-top: solid 0.5px var(--divider);
	padding: 8px 14px;
	font-size: 0.8em;

	> .entry {
		display: flex;
		gap: 6px;
		white-space: nowrap;
		overflow: hidden;

		> .host {
			flex: 1;
			overflow: hidden;
			text-overflow: ellipsis;
		}

		> .attempts {
			opacity: 0.6;
		}

		> .error {
			max-width: 45%;
			overflow: hidden;
			text-overflow: ellipsis;
			opacity: 0.6;
		}
	}
}
</style>
