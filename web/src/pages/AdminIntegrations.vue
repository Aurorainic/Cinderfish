<!-- Ported from firefish@b425f61:packages/client/src/pages/admin/integrations.vue (AGPL-3.0-only) -->
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
// Firefish's integrations page wires third-party sign-in and notification
// relays. Cinderfish deliberately has no third-party login; the only
// delegation paths are MiAuth and OAuth, which have their own pages.
import { computed } from "vue";
import { isAdmin } from "@/account";
import AdminUnavailable from "@/components/admin/AdminUnavailable.vue";
import CfA from "@/components/ui/CfA.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageAdminIntegrations" });

const admin = computed<boolean>(() => isAdmin());

const state = {
	title: "外部整合",
	summary: "上游的整合页用来配置 Twitter/GitHub 登录与外部通知推送。Cinderfish 明确不做第三方账号登录：登录方式只有密码（可选 TOTP），以及对第三方客户端的 MiAuth/OAuth 授权。",
	available: [{"label": "MiAuth / OAuth 授权", "note": "第三方客户端用这两种协议接入（/miauth/:session、/oauth/authorize）", "to": "/settings/apps"}],
	missing: ["Twitter / GitHub / Discord 账号登录", "外部通知转发（Slack、Discord webhook）", "外部图床"],
};

definePageMetadata({ title: "外部整合", icon: "ph-info ph-bold ph-lg" });
</script>

<style lang="scss" scoped>
.wrap {
	max-width: 760px;
}
</style>
