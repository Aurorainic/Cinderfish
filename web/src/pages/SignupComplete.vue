<!-- Ported from firefish@b425f61:packages/client/src/pages/signup-complete.vue (AGPL-3.0-only) -->
<template>
	<MkSpacer :content-max="700" :margin-min="16" :margin-max="32">
		<CfFeatureGap
			title="邮箱确认注册未启用"
			:summary="summary"
			:available="available"
			:missing="missing"
		/>
	</MkSpacer>
</template>

<script lang="ts" setup>
// Misskey's /signup-complete/:code finishes an email-gated signup: the code is a
// pending-user row created by signup when emailRequiredForSignup is on.
// Cinderfish's registration modes are open, invite-only and admin-created; there
// is no email-gated mode, no pending-user table and no signup-pending endpoint,
// so such a link cannot belong to this instance.
import CfFeatureGap from "@/components/CfFeatureGap.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageSignupComplete" });

const summary =
	"Cinderfish 的注册模式是开放注册、邀请码注册与管理员建号三种，没有「注册需邮箱确认」这一档。" +
	"因此这个链接不会属于本站：要么来自其它实例的邮件，要么是旧版本遗留的链接。你的账号没有被创建，" +
	"可以直接用正常入口注册或登录。";

const available = [
	{ label: "登录", note: "已有账号", to: "/signin" },
	{ label: "注册", note: "按本站当前注册模式", to: "/signup" },
];

const missing = [
	"邮箱确认注册（emailRequiredForSignup）",
	"待确认账号表与 30 分钟有效期",
	"确认后自动登录",
];

definePageMetadata({ title: "注册确认", icon: "ph-envelope-simple ph-bold ph-lg" });
</script>
