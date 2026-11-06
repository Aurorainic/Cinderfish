<!-- Ported from firefish@b425f61:packages/client/src/pages/settings/plugin.vue (AGPL-3.0-only) -->
<template>
	<div class="_formRoot cf-settings-plugin">
		<CfFeatureGap
			title="插件运行时未实现"
			:summary="summary"
			:available="available"
			:missing="missing"
		/>
	</div>
</template>

<script lang="ts" setup>
// Firefish plugins are AiScript scripts that hook the client's UI at runtime
// (post form actions, note actions, user actions, note view/post interruptors).
// Cinderfish has the hook arrays defined in store/base.ts but no AiScript
// interpreter, so plugins cannot be installed or run.
import CfFeatureGap from "@/components/CfFeatureGap.vue";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageSettingsPlugin" });

const summary =
	"Firefish 的插件是 AiScript 脚本，安装后能挂到发帖栏、笔记菜单与用户菜单上。" +
	"Cinderfish 没有 AiScript 运行时（客户端依赖里也没有这个解释器），因此无法安装、启用或卸载插件。" +
	"下面的自定义 CSS 与 KaTeX 宏是本站已有的同类扩展手段。";

const available = [
	{ label: "自定义 CSS", note: "全站样式覆盖", to: "/settings/custom-css" },
	{ label: "自定义 KaTeX 宏", note: "公式输入快捷方式", to: "/settings/custom-katex-macro" },
	{ label: "API", note: "访问令牌与第三方应用授权", to: "/settings/api" },
];

const missing = ["插件安装（.aiscript 文件）", "插件启用 / 停用 / 卸载列表", "AiScript 运行时与插件 API"];

definePageMetadata({ title: "插件", icon: "ph-puzzle-piece ph-bold ph-lg" });
</script>

<style lang="scss" scoped>
.cf-settings-plugin {
	max-width: 760px;
}
</style>
