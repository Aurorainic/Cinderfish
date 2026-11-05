<!-- Ported from firefish@b425f61:packages/client/src/pages/user/index.activity.vue (AGPL-3.0-only) -->
<template>
	<div class="activity _panel">
		<header class="header">
			<i class="ph-chart-bar ph-bold ph-lg"></i>
			<span>{{ t("activity") }}</span>
		</header>
		<div class="body">
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
			<CfInfo warn class="notice">
				活动图表需要 charts/user/notes 接口，后端尚未提供，因此这里只显示
				users/show 可直接读到的计数，不绘制折线或柱状图。
			</CfInfo>
		</div>
	</div>
</template>

<script lang="ts" setup>
// Firefish renders <MkChart src="per-user-notes"> (chart.js). Cinderfish has no
// `charts/*` endpoint and chart.js is not a dependency, so the panel keeps the
// real counters from `users/show` and states plainly what is missing instead of
// drawing a fake chart (contract §2 / completion criterion 4).
import type { UserDetailed } from "@/api/types";
import CfInfo from "@/components/ui/CfInfo.vue";
import MkNumber from "@/components/MkNumber.vue";
import { t } from "@/i18n";

defineProps<{
	user: UserDetailed;
}>();
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

	.stats {
		display: flex;
		gap: 16px;
		margin-bottom: 16px;

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
				opacity: 0.7;
			}
		}
	}

	> .body > .notice {
		font-size: 0.85em;
	}
}
</style>
