<!-- Ported from firefish@b425f61:packages/client/src/pages/admin/ads.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :display-back-button="true" />
		</template>
		<MkSpacer :content-max="760" :margin-min="16" :margin-max="32">
			<div v-if="!admin" class="wrap">
				<CfInfo warn>
					需要管理员权限。
					<CfA to="/" class="_link">返回首页</CfA>
				</CfInfo>
			</div>
			<div v-else class="wrap">
				<AdminUnavailable
					:title="state.title"
					:summary="state.summary"
					:available="state.available"
					:missing="state.missing"
				/>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script lang="ts" setup>
// Firefish's admin/ads manages timeline promotions. Cinderfish has no ad
// system: meta.ads is always empty and nothing is inserted into the timeline,
// so the page points at announcements instead.
import { computed } from "vue";
import { isAdmin } from "@/account";
import AdminUnavailable from "@/components/admin/AdminUnavailable.vue";
import CfA from "@/components/ui/CfA.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageAdminPromotions" });

const admin = computed<boolean>(() => isAdmin());

const state = {
	title: "广告与推广",
	summary: "上游的广告页管理时间线里插入的推广条目（admin/ad/*，客户端读 meta.ads）。Cinderfish 没有广告系统：meta.ads 恒为空数组，时间线也不会插入推广内容。",
	available: [{"label": "实例公告", "note": "给所有用户的公告，可设有效期与上下线", "to": "/admin/announcements"}],
	missing: ["广告条目 CRUD（admin/ad/*）", "按比例插入时间线（notesPerOneAd）", "广告展示与点击统计"],
};

definePageMetadata({ title: "广告与推广", icon: "ph-info ph-bold ph-lg" });
</script>

<style lang="scss" scoped>
.wrap {
	max-width: 760px;
}
</style>
