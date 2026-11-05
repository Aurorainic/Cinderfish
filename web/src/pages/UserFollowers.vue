<!-- Ported from firefish@b425f61:packages/client/src/pages/user/followers.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :display-back-button="true" />
		</template>
		<MkSpacer :content-max="1000">
			<transition name="fade" mode="out-in">
				<div v-if="user != null" :key="user.id">
					<FollowList :user="user" type="followers" />
				</div>
				<CfInfo v-else-if="remote" warn icon="globe" class="remote">
					<div>
						<b>{{ t("unavailable") }}</b>
						<div>{{ t("remoteUserCaution") }}</div>
					</div>
				</CfInfo>
				<MkError v-else-if="error != null" @retry="fetchUser" />
				<MkLoading v-else />
			</transition>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script lang="ts" setup>
// `users/show` resolves local accounts by `username` only (non-empty `host` is
// rejected, docs/research/api-contract/auth-users.md), so the federated
// `alice@remote.example` form gets a stated limitation instead of a load error.
// The page also reads its `acct` from the route: the `/@:acct/followers` record
// in router/routes.ts passes no props, unlike Firefish's nirax wiring.
import { computed, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { api } from "@/api/client";
import type { UserDetailed } from "@/api/types";
import CfInfo from "@/components/ui/CfInfo.vue";
import FollowList from "@/components/user/FollowList.vue";
import { t } from "@/i18n";
import { definePageMetadata } from "@/scripts/page-metadata";

const route = useRoute();

const acct = computed<string>(() => {
	const value = route.params.acct;
	return typeof value === "string" ? value : "";
});

const remote = computed<boolean>(() => acct.value.includes("@"));

const user = ref<UserDetailed | null>(null);
const error = ref<unknown>(null);

async function fetchUser(): Promise<void> {
	user.value = null;
	error.value = null;
	if (remote.value) return;
	try {
		user.value = await api("users/show", { username: acct.value });
	} catch (cause) {
		error.value = cause;
	}
}

watch(acct, () => void fetchUser(), { immediate: true });

definePageMetadata(
	computed(() => {
		if (user.value != null) {
			const name = user.value.name;
			return {
				icon: "ph-user ph-bold ph-lg",
				title:
					name != null && name !== ""
						? `${name} (@${user.value.username})`
						: `@${user.value.username}`,
				subtitle: t("followers"),
				userName: user.value,
				avatar: user.value,
			};
		}
		if (remote.value) {
			return {
				icon: "ph-user ph-bold ph-lg",
				title: `@${acct.value}`,
				subtitle: t("followers"),
			};
		}
		return null;
	}),
);
</script>

<style lang="scss" scoped>
.remote {
	align-items: flex-start;
}

.fade-enter-active,
.fade-leave-active {
	transition: opacity 0.125s ease;
}

.fade-enter-from,
.fade-leave-to {
	opacity: 0;
}
</style>
