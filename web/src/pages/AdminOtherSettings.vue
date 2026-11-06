<!-- Ported from firefish@b425f61:packages/client/src/pages/admin/other-settings.vue (AGPL-3.0-only) -->
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
// Firefish's other-settings page collects the instance switches that fit
// nowhere else. Here they are constants or config.toml values, so the page
// lists what is adjustable at runtime and links to those pages.
import { computed } from "vue";
import { isAdmin } from "@/account";
import AdminUnavailable from "@/components/admin/AdminUnavailable.vue";
import CfA from "@/components/ui/CfA.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageAdminOtherSettings" });

const admin = computed<boolean>(() => isAdmin());

const state = {
	title: "其它设置",
	summary: "上游把无法归类的实例开关放在这里（保留用户名、远程文件缓存、字数上限、媒体代理等）。Cinderfish 的这些值要么是常量，要么在 config.toml 里，运行期不可改。",
	available: [{"label": "实例设置", "note": "名称、描述、注册模式、合规链接、隐藏标签、主题色、实例图标与横幅", "to": "/admin/settings"}, {"label": "实例 CSS", "note": "全站样式，保存在 instance_meta", "to": "/admin/custom-css"}, {"label": "对象存储", "note": "当前后端与配置状态（只读）", "to": "/admin/object-storage"}],
	missing: ["保留用户名 / 禁用用户名列表", "远程文件缓存开关（恒为 true）", "笔记字数上限调整（常量 3000）", "媒体代理地址"],
};

definePageMetadata({ title: "其它设置", icon: "ph-info ph-bold ph-lg" });
</script>

<style lang="scss" scoped>
.wrap {
	max-width: 760px;
}
</style>
