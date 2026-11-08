<!-- Ported from firefish@b425f61:packages/client/src/pages/user/index.activity.vue (AGPL-3.0-only) -->
<template>
	<div class="activity _panel">
		<header class="header">
			<i class="ph-chart-bar ph-bold ph-lg"></i>
			<span>{{ t("activity") }}</span>
		</header>
		<div class="body">
			<CfLoading v-if="fetching && chart == null" />
			<CfInfo v-else-if="failed" warn>
				活动统计加载失败，稍后会自动重试。
			</CfInfo>
			<NotesChart v-else :chart="chart" class="chart" />
			<div class="stats">
				<div class="stat">
					<b><MkNumber :value="user.notesCount" /></b>
					<span>{{ t("notes") }}</span>
				</div>
				<div class="stat">
					<b><MkNumber :value="user.followingCount" /></b>
					<span>{{ t("following") }}</span>
				</div>
				<div class="stat">
					<b><MkNumber :value="user.followersCount" /></b>
					<span>{{ t("followers") }}</span>
				</div>
			</div>
		</div>
	</div>
</template>

<script lang="ts" setup>
// Firefish renders <MkChart src="per-user-notes"> (chart.js). chart.js is not a
// dependency here, so the shared NotesChart component draws the same
// `charts/user/notes` arrays as stacked bars.
import { onMounted, ref, watch } from "vue";
import type { UserDetailed, UserNotesChart } from "@/api/types";
import { api } from "@/api/client";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import MkNumber from "@/components/MkNumber.vue";
import NotesChart from "@/components/NotesChart.vue";
import { t } from "@/i18n";

const props = defineProps<{
	user: UserDetailed;
}>();

const chart = ref<UserNotesChart | null>(null);
const fetching = ref(false);
const failed = ref(false);

function load(): void {
	fetching.value = true;
	failed.value = false;
	api<UserNotesChart>("charts/user/notes", {
		span: "day",
		limit: 30,
		userId: props.user.id,
	})
		.then((result) => {
			chart.value = result;
		})
		.catch(() => {
			failed.value = true;
		})
		.finally(() => {
			fetching.value = false;
		});
}

onMounted(load);
watch(() => props.user.id, load);
</script>

<style lang="scss" scoped>
.activity {
	overflow: hidden;

	> .header {
		display: flex;
		align-items: center;
		gap: 0.5em;
		padding: 12px 16px;
		border-bottom: solid 0.5px var(--divider);
		font-weight: bold;
	}

	> .body {
		padding: 16px;
	}

	.chart {
		margin-bottom: 16px;
	}

	.stats {
		display: flex;
		gap: 16px;

		> .stat {
			flex: 1;
			text-align: center;

			> b {
				display: block;
				font-size: 1.4em;
				color: var(--accent);
			}

			> span {
				font-size: 0.8em;
				color: var(--fgTransparentWeak);
			}
		}
	}
}
</style>
