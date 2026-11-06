<!-- Ported from firefish@b425f61:packages/client/src/pages/admin/experiments.vue (AGPL-3.0-only) -->
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
// Firefish's admin/experiments exposes feature flags for unfinished behaviour.
// Cinderfish has no experimental branches to toggle, so the page states that
// instead of rendering switches that nothing reads.
import { computed } from "vue";
import { isAdmin } from "@/account";
import AdminUnavailable from "@/components/admin/AdminUnavailable.vue";
import CfA from "@/components/ui/CfA.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageAdminExperiments" });

const admin = computed<boolean>(() => isAdmin());

const state = {
	title: "实验性功能",
	summary: "上游在这里放一组开关，用来打开尚未稳定的行为（新的时间线算法、前端实验特性等）。Cinderfish 没有这些实验开关：实例的行为由 config.toml 与运行期实例设置决定，没有隐藏的实验分支。",
	available: [{"label": "实例设置", "note": "名称、描述、注册模式、合规链接、隐藏标签、主题色", "to": "/admin/settings"}],
	missing: ["实验性时间线算法", "前端实验特性开关", "逐步放量（rollout）百分比"],
};

definePageMetadata({ title: "实验性功能", icon: "ph-info ph-bold ph-lg" });
</script>

<style lang="scss" scoped>
.wrap {
	max-width: 760px;
}
</style>
