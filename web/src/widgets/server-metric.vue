<!-- Ported from firefish@b425f61:packages/client/src/widgets/server-metric/index.vue (AGPL-3.0-only) -->
<template>
	<WidgetCard
		:title="t('_widgets.serverMetric')"
		icon="ph-bold ph-gauge"
		:show-header="widgetProps.showHeader === true"
	>
		<CfLoading v-if="fetching && info == null" />
		<div v-else-if="signedOut" class="state">登录后可以查看服务器指标。</div>
		<div v-else-if="failed" class="state">
			<div>{{ t("somethingHappened") }}</div>
			<CfButton @click="loadMetrics">{{ t("retry") }}</CfButton>
		</div>
		<template v-else-if="info != null">
			<div class="metrics">
				<div class="metric">
					<div class="head">
						<span class="label">CPU</span>
						<span class="value">{{ percent(info.cpu.usage) }}</span>
					</div>
					<div class="track">
						<div class="fill cpu" :style="{ width: barWidth(info.cpu.usage) }"></div>
					</div>
					<div class="note" :title="info.cpu.model">
						{{ info.cpu.cores }} 核 · {{ info.cpu.model }}
					</div>
				</div>
				<div class="metric">
					<div class="head">
						<span class="label">内存</span>
						<span class="value">{{ ratio(info.mem.used, info.mem.total) }}</span>
					</div>
					<div class="track">
						<div class="fill mem" :style="{ width: barWidth(memPercent) }"></div>
					</div>
					<div class="note">{{ bytes(info.mem.used) }} / {{ bytes(info.mem.total) }}</div>
				</div>
				<div class="metric">
					<div class="head">
						<span class="label">磁盘</span>
						<span class="value">{{ ratio(info.fs.used, info.fs.total) }}</span>
					</div>
					<div class="track">
						<div class="fill fs" :style="{ width: barWidth(fsPercent) }"></div>
					</div>
					<div class="note">{{ bytes(info.fs.used) }} / {{ bytes(info.fs.total) }}</div>
				</div>
				<div class="metric">
					<div class="head">
						<span class="label">负载</span>
						<span class="value">{{ info.load.one.toFixed(2) }}</span>
					</div>
					<div class="note">
						1 分钟 {{ info.load.one.toFixed(2) }} · 5 分钟
						{{ info.load.five.toFixed(2) }} · 15 分钟
						{{ info.load.fifteen.toFixed(2) }}
					</div>
				</div>
			</div>
			<div v-if="!info.mem.known || !info.fs.known || !info.load.known" class="state small">
				当前平台无法读取全部指标。
			</div>
		</template>
	</WidgetCard>
</template>

<script setup lang="ts">
// Firefish draws CPU/MEM/disk/network curves from the `serverStats` Streaming
// channel. Cinderfish has no such channel, so the widget polls `server-info`
// and renders the same figures as bars; the endpoint is credential-only and
// reports `known=false` where the platform cannot read a value.
import { computed, ref } from "vue";
import WidgetCard from "./WidgetCard.vue";
import { CfButton, CfLoading } from "@/components/ui";
import { api } from "@/api/client";
import type { ServerInfo } from "@/api/types";
import { useInterval } from "@/composables/use-interval";
import { t } from "@/i18n";
import { useWidgetPropsManager, type WidgetComponentExpose, type WidgetField } from "./index";
import type { Widget } from "@/store/widgets";

const name = "serverMetric";

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

const info = ref<ServerInfo | null>(null);
const fetching = ref(true);
const failed = ref(false);
const signedOut = ref(false);

const memPercent = computed(() =>
	info.value == null || info.value.mem.total <= 0
		? 0
		: (info.value.mem.used / info.value.mem.total) * 100,
);
const fsPercent = computed(() =>
	info.value == null || info.value.fs.total <= 0
		? 0
		: (info.value.fs.used / info.value.fs.total) * 100,
);

function loadMetrics(): void {
	fetching.value = true;
	failed.value = false;
	api<ServerInfo>("server-info")
		.then((result) => {
			info.value = result;
			signedOut.value = false;
		})
		.catch((error: unknown) => {
			const status =
				error != null && typeof error === "object" && "status" in error
					? (error as { status?: number }).status
					: null;
			signedOut.value = status === 401;
			if (!signedOut.value) failed.value = true;
		})
		.finally(() => {
			fetching.value = false;
		});
}

function percent(value: number): string {
	return `${value.toFixed(1)}%`;
}

function barWidth(value: number): string {
	return `${Math.max(0, Math.min(100, value))}%`;
}

function ratio(used: number, total: number): string {
	if (total <= 0) return "—";
	return `${((used / total) * 100).toFixed(1)}%`;
}

function bytes(value: number): string {
	if (value <= 0) return "—";
	const units = ["B", "KiB", "MiB", "GiB", "TiB"];
	let n = value;
	let i = 0;
	while (n >= 1024 && i < units.length - 1) {
		n /= 1024;
		i++;
	}
	return `${n.toFixed(i === 0 ? 0 : 1)} ${units[i]}`;
}

useInterval(loadMetrics, 1000 * 5, { immediate: true, afterMounted: true });

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
	text-align: center;
	opacity: 0.8;

	&.small {
		padding: 4px 14px 10px;
		font-size: 0.75em;
	}
}

.metrics {
	display: flex;
	flex-direction: column;
	gap: 10px;
	padding: 12px 14px;

	> .metric {
		> .head {
			display: flex;
			justify-content: space-between;
			font-size: 0.85em;
			margin-bottom: 3px;

			> .label {
				opacity: 0.7;
			}
		}

		> .track {
			height: 5px;
			border-radius: 3px;
			background: var(--divider);
			overflow: hidden;

			> .fill {
				height: 100%;
				background: var(--accent);
				transition: width 0.4s ease;
			}
		}

		> .note {
			margin-top: 3px;
			font-size: 0.72em;
			opacity: 0.6;
			white-space: nowrap;
			overflow: hidden;
			text-overflow: ellipsis;
		}
	}
}
</style>
