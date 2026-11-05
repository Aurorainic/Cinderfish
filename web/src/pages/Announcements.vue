<!-- Ported from firefish@b425f61:packages/client/src/pages/announcements.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :actions="headerActions" />
		</template>
		<MkSpacer :content-max="800">
			<div class="announcements _formRoot">
				<CfInfo class="_formBlock" icon="megaphone-simple">
					公告接口尚未提供：后端没有注册 <code>announcements</code> 与
					<code>i/read-announcement</code> 端点（见第三波施工契约 §2），所以这里没有列表、已读标记与「知道了」按钮。
				</CfInfo>

				<div class="empty _formBlock">
					<i class="ph-megaphone-simple ph-bold ph-lg icon"></i>
					<p class="title">{{ t("announcements") }}</p>
					<p class="caption">当前没有可显示的公告。</p>
				</div>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script lang="ts" setup>
// Firefish paginates `announcements` and marks each one read through
// `i/read-announcement`. Neither endpoint exists on Cinderfish (wave-3
// contract §2), so the page has no request to make: it renders the empty state
// by contract instead of faking a list. When the endpoints land, the body is
// replaced by the upstream MkPagination + MkTime loop and nothing else changes.
import { computed } from "vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import { t } from "@/i18n";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageAnnouncements" });

const headerActions = computed(() => []);

definePageMetadata({
	title: t("announcements"),
	icon: "ph-megaphone-simple ph-bold ph-lg",
});
</script>

<style lang="scss" scoped>
.announcements {
	padding: 16px;
}

.empty {
	display: flex;
	flex-direction: column;
	align-items: center;
	gap: 8px;
	padding: 48px 16px;
	border-radius: var(--radius);
	background: var(--panel);
	text-align: center;

	> .icon {
		font-size: 32px;
		opacity: 0.5;
	}

	> .title {
		margin: 0;
		font-weight: bold;
	}

	> .caption {
		margin: 0;
		font-size: 0.9em;
		color: var(--fgTransparentWeak);
	}
}
</style>
