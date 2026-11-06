<!-- Cinderfish: replaces firefish@b425f61:packages/client/src/components/MkChart.vue (AGPL-3.0-only) -->
<!--
	Firefish draws `charts/user/notes` with chart.js. chart.js is not a
	dependency here, so this component renders the same data as a 30-day
	stacked bar chart (normal / reply / renote) from the API arrays.
-->
<template>
	<div class="notes-chart">
		<div class="chart" role="img" :aria-label="`近 ${days.length} 天的发帖量`">
			<div
				v-for="(day, index) in days"
				:key="index"
				class="bar"
				:title="`${day.label}：${day.total} 帖`"
			>
				<div
					class="stack"
					:style="{ height: `${Math.max(day.total === 0 ? 0 : 6, (day.total / maxDay) * 100)}%` }"
				>
					<span
						v-if="day.normal > 0"
						class="seg normal"
						:style="{ flexGrow: day.normal }"
					></span>
					<span
						v-if="day.reply > 0"
						class="seg reply"
						:style="{ flexGrow: day.reply }"
					></span>
					<span
						v-if="day.renote > 0"
						class="seg renote"
						:style="{ flexGrow: day.renote }"
					></span>
				</div>
			</div>
		</div>
		<div class="legend">
			<span class="entry normal">{{ t("note") }}</span>
			<span class="entry reply">{{ t("reply") }}</span>
			<span class="entry renote">{{ t("renote") }}</span>
		</div>
	</div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import type { UserNotesChart } from "@/api/types";
import { t } from "@/i18n";

const props = defineProps<{ chart: UserNotesChart | null }>();

const days = computed(() => {
	const data = props.chart;
	if (data == null) return [];
	const now = new Date();
	const today = new Date(
		Date.UTC(now.getUTCFullYear(), now.getUTCMonth(), now.getUTCDate()),
	);
	return data.diffs.normal.map((normal, index) => {
		const reply = data.diffs.reply[index] ?? 0;
		const renote = data.diffs.renote[index] ?? 0;
		const date = new Date(today);
		date.setUTCDate(date.getUTCDate() - (data.diffs.normal.length - 1 - index));
		return {
			normal,
			reply,
			renote,
			total: normal + reply + renote,
			label: `${date.getUTCMonth() + 1}/${date.getUTCDate()}`,
		};
	});
});
const maxDay = computed(() => Math.max(1, ...days.value.map((day) => day.total)));
</script>

<style lang="scss" scoped>
.chart {
	display: flex;
	align-items: flex-end;
	gap: 2px;
	height: 80px;

	> .bar {
		flex: 1;
		height: 100%;
		display: flex;
		align-items: flex-end;

		> .stack {
			width: 100%;
			display: flex;
			flex-direction: column;
			border-radius: 1px;
			overflow: hidden;

			> .seg {
				display: block;
				min-height: 1px;

				&.normal {
					background: var(--accent);
				}

				&.reply {
					background: var(--accentedBg);
				}

				&.renote {
					background: var(--divider);
				}
			}
		}
	}
}

.legend {
	display: flex;
	gap: 10px;
	margin-top: 6px;
	font-size: 0.75em;
	opacity: 0.75;

	> .entry::before {
		content: "";
		display: inline-block;
		width: 8px;
		height: 8px;
		margin-right: 4px;
		border-radius: 2px;
	}

	> .normal::before {
		background: var(--accent);
	}

	> .reply::before {
		background: var(--accentedBg);
	}

	> .renote::before {
		background: var(--divider);
	}
}
</style>
