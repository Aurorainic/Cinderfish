<!-- Ported from firefish@b425f61:packages/client/src/pages/admin/security.vue (AGPL-3.0-only) -->
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
// Firefish's admin/security gathers registration limits, invite codes, IP
// bans and signin audit. Only the registration mode exists here, and it lives
// on the instance settings page; the rest has no storage or endpoint.
import { computed } from "vue";
import { isAdmin } from "@/account";
import AdminUnavailable from "@/components/admin/AdminUnavailable.vue";
import CfA from "@/components/ui/CfA.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageAdminSecurity" });

const admin = computed<boolean>(() => isAdmin());

const state = {
	title: "安全设置",
	summary: "上游的安全页集中管理注册限制、邀请码、IP 封禁与登录审计。Cinderfish 把注册模式放在实例设置里，其余几项没有对应的存储与端点。",
	available: [{"label": "注册模式", "note": "open / invite / admin 三档，运行期可切换", "to": "/admin/settings"}, {"label": "封禁用户", "note": "在用户列表里冻结/解冻账号", "to": "/admin/users"}, {"label": "两步验证", "note": "由每个账号在安全设置里自行开启", "to": "/settings/security"}],
	missing: ["IP / 网段封禁", "邀请码批量生成与管理界面", "登录失败审计与自动封禁"],
};

definePageMetadata({ title: "安全设置", icon: "ph-info ph-bold ph-lg" });
</script>

<style lang="scss" scoped>
.wrap {
	max-width: 760px;
}
</style>
