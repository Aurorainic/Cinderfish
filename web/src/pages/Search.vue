<!-- Ported from firefish@b425f61:packages/client/src/pages/search.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :display-back-button="true" />
		</template>
		<MkSpacer :content-max="800">
			<div class="query _panel">
				<i class="ph-magnifying-glass ph-bold"></i>
				<span v-if="query.length > 0" class="text">{{ query }}</span>
				<span v-else class="text placeholder">{{ t("searchPlaceholder") }}</span>
			</div>

			<CfInfo class="notice">
				搜索接口尚未提供：后端没有实现 `notes/search` 与 `users/search`，因此这里不会返回帖子或用户结果。可以用「发现」页的热门标签进入标签页浏览本地帖子。
			</CfInfo>

			<div class="actions">
				<CfButton primary link to="/explore">
					<i class="ph-compass ph-bold"></i>
					{{ t("explore") }}
				</CfButton>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script lang="ts" setup>
// Firefish's search page paginates `notes/search` and `users/search`. Neither
// endpoint exists on Cinderfish (contract §2), and fabricating results from the
// timelines would misreport "no match" as "no such note". The page therefore
// states the gap and routes the visitor to the explore page's tag list, which is
// backed by a real endpoint (`hashtags/trend`).
import { computed } from "vue";
import { useRoute } from "vue-router";
import { CfButton, CfInfo } from "@/components/ui";
import { t } from "@/i18n";
import { definePageMetadata } from "@/scripts/page-metadata";

const route = useRoute();

const query = computed(() => {
	const raw = route.query.q;
	if (typeof raw === "string") return raw;
	if (Array.isArray(raw)) return raw.join(" ");
	return "";
});

definePageMetadata(
	computed(() => ({
		title:
			query.value.length > 0
				? t("searchWith", { q: query.value })
				: t("search"),
		icon: "ph-magnifying-glass ph-bold ph-lg",
	})),
);
</script>

<style lang="scss" scoped>
.query {
	display: flex;
	align-items: center;
	gap: 10px;
	padding: 14px 16px;
	border-radius: var(--radius);
	color: var(--fgTransparentWeak);

	> .text {
		color: var(--fg);
		word-break: break-all;
	}

	> .placeholder {
		color: var(--fgTransparentWeak);
	}
}

.notice {
	margin-block: var(--margin);
}

.actions {
	display: flex;
	justify-content: center;
}
</style>
