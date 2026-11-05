<!-- Ported from firefish@b425f61:packages/client/src/pages/user/index.vue (AGPL-3.0-only) -->
<template>
	<UserIndex v-if="!remote" :acct="acct" :page="page" />
	<MkStickyContainer v-else>
		<template #header>
			<MkPageHeader :display-back-button="true" :title="`@${acct}`" />
		</template>
		<MkSpacer :content-max="800">
			<div class="remote _panel">
				<header class="header">
					<i class="ph-user ph-bold ph-lg"></i>
					<span>@{{ acct }}</span>
				</header>
				<div class="body">
					<CfInfo warn>{{ t("remoteUserCaution") }}</CfInfo>
					<p class="note">
						远程账号的资料需要联邦侧的 profile 接口，后端 users/show
						只接受本地用户名（远程用户返回错误），因此这里不显示简介、关注按钮或帖子列表，也不会用占位数据充数。
					</p>
				</div>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script lang="ts" setup>
// The route is `/@:acct/:page?`, so `acct` is "alice" locally and
// "alice@remote.example" for a federated account. Firefish asks
// `users/show` with `Acct.parse(acct)` and lets the server resolve the remote
// actor; Cinderfish's users/show rejects non-empty hosts (contract §5), so the
// remote case is answered with an explicit explanation instead of a load error.
import { computed } from "vue";
import { useRoute } from "vue-router";
import CfInfo from "@/components/ui/CfInfo.vue";
import { t } from "@/i18n";
import { definePageMetadata } from "@/scripts/page-metadata";
import UserIndex from "./UserIndex.vue";

const route = useRoute();

const acct = computed<string>(() => {
	const value = route.params.acct;
	return typeof value === "string" ? value : "";
});

const page = computed<string | undefined>(() => {
	const value = route.params.page;
	return typeof value === "string" && value !== "" ? value : undefined;
});

const remote = computed<boolean>(() => acct.value.includes("@"));

// Only the remote branch owns page metadata here; the local branch renders
// UserIndex, which provides its own (user name, avatar, follow button).
definePageMetadata(
	computed(() =>
		remote.value
			? { title: `@${acct.value}`, icon: "ph-user ph-bold ph-lg" }
			: null,
	),
);
</script>

<style lang="scss" scoped>
.remote {
	overflow: hidden;

	> .header {
		display: flex;
		align-items: center;
		gap: 0.5em;
		padding: 12px 16px;
		border-bottom: solid 0.5px var(--divider);
		font-weight: bold;
	}

	> .body {
		padding: 16px;

		> .note {
			margin: 12px 0 0 0;
			font-size: 0.9em;
			opacity: 0.8;
		}
	}
}
</style>
