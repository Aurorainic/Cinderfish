<!-- Ported from firefish@b425f61:packages/client/src/pages/user/reactions.vue (AGPL-3.0-only) -->
<template>
	<div class="reactions _panel">
		<header class="header">
			<i class="ph-smiley ph-bold ph-lg"></i>
			<span>{{ t("reaction") }}</span>
		</header>
		<div class="body">
			<CfInfo warn class="notice">
				该页签依赖 users/reactions 接口（某用户发出的回应列表），后端尚未提供，因此这里不列出回应记录。单条帖子的回应仍可在帖子详情页查看（notes/reactions）。
			</CfInfo>
		</div>
	</div>
</template>

<script lang="ts" setup>
// Firefish pages over `users/reactions` with MkPagination and MkReactionIcon.
// Cinderfish registers neither `users/reactions` nor `charts/*` (see
// internal/httpapi/server.go), and inventing rows is not acceptable, so the tab
// states what is missing (completion criterion 4). `notes/reactions` exists and
// is used by Note.vue for the per-note reaction list.
import type { UserDetailed } from "@/api/types";
import CfInfo from "@/components/ui/CfInfo.vue";
import { t } from "@/i18n";

defineProps<{
	user: UserDetailed;
}>();
</script>

<style lang="scss" scoped>
.reactions {
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

	> .body > .notice {
		font-size: 0.85em;
	}
}
</style>
