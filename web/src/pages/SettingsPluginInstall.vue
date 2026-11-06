<!-- Ported from firefish@b425f61:packages/client/src/pages/settings/plugin.install.vue (AGPL-3.0-only) -->
<template>
	<MkSpacer :content-max="700" :margin-min="16" :margin-max="32">
		<CfFeatureGap
			title="无法安装插件"
			:summary="summary"
			:available="available"
			:missing="missing"
		/>
	</MkSpacer>
</template>

<script lang="ts" setup>
// Uploading a plugin would store an AiScript source and hand it to the client's
// interpreter. Neither the storage nor the interpreter exists here; the page
// exists only so a saved link does not 404.
import CfFeatureGap from "@/components/CfFeatureGap.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageSettingsPluginInstall" });

const summary =
	"安装插件需要先有插件表与 AiScript 运行时，Cinderfish 两者都没有，" +
	"所以这里不接受任何 .aiscript 文件。如果你是从其它实例的分享链接过来，那个插件只能在那边使用。";

const available = [
	{ label: "插件说明", note: "为什么没有插件运行时", to: "/settings/plugin" },
	{ label: "自定义 CSS", note: "不改运行时的样式扩展", to: "/settings/custom-css" },
];

const missing = ["插件文件上传与校验", "插件表（启用状态、作者、版本）", "AiScript 解释器"];

definePageMetadata({ title: "安装插件", icon: "ph-puzzle-piece ph-bold ph-lg" });
</script>
