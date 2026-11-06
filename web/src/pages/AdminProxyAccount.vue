<!-- Ported from firefish@b425f61:packages/client/src/pages/admin/proxy-account.vue (AGPL-3.0-only) -->
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
// Firefish's proxy-account page configures the instance actor used to fetch
// remote content for anonymous visitors. Cinderfish signs fetches with the
// instance key and has no separate proxy account, so the page explains that
// rather than offering a name field that nothing would read.
import { computed } from "vue";
import { isAdmin } from "@/account";
import AdminUnavailable from "@/components/admin/AdminUnavailable.vue";
import CfA from "@/components/ui/CfA.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageAdminProxyAccount" });

const admin = computed<boolean>(() => isAdmin());

const state = {
	title: "代理账号",
	summary: "代理账号是实例用来代替匿名访客抓取远端内容的 Fediverse 身份。Cinderfish 的联邦抓取由实例自己的签名密钥完成（第一个管理员），没有单独的代理账号配置。",
	available: [{"label": "联邦模式", "note": "public / allowlist / none 三档运行期切换", "to": "/admin/federation"}],
	missing: ["代理账号（proxyAccount）配置", "用于抓取的独立 actor 文档"],
};

definePageMetadata({ title: "代理账号", icon: "ph-info ph-bold ph-lg" });
</script>

<style lang="scss" scoped>
.wrap {
	max-width: 760px;
}
</style>
