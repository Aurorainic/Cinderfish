<!-- Ported from firefish@b425f61:packages/client/src/pages/settings/account-info.vue (AGPL-3.0-only) -->
<template>
	<div class="_formRoot cf-settings-account-info">
		<CfLoading v-if="loading" />
		<CfInfo v-else-if="loadError != null" warn>
			<span class="errorText">{{ loadError }}</span>
			<CfButton inline @click="load">{{ t("retry") }}</CfButton>
		</CfInfo>
		<CfInfo v-else-if="account == null" warn>{{ t("signinRequired") }}</CfInfo>

		<template v-else>
			<CfKeyValue :copy="account.id">
				<template #key>ID</template>
				<template #value>
					<span class="monospace">{{ account.id }}</span>
				</template>
			</CfKeyValue>

			<section class="_formBlock">
				<CfKeyValue oneline>
					<template #key>{{ t("registeredDate") }}</template>
					<template #value>
						<CfTime :time="account.createdAt" mode="detail" />
					</template>
				</CfKeyValue>
			</section>

			<section class="_formBlock">
				<h2 class="sectionTitle">{{ t("statistics") }}</h2>
				<CfKeyValue oneline>
					<template #key>{{ t("notesCount") }}</template>
					<template #value>{{ number(account.notesCount) }}</template>
				</CfKeyValue>
				<CfKeyValue oneline>
					<template #key>{{ t("followingCount") }}</template>
					<template #value>{{ number(account.followingCount) }}</template>
				</CfKeyValue>
				<CfKeyValue oneline>
					<template #key>{{ t("followersCount") }}</template>
					<template #value>{{ number(account.followersCount) }}</template>
				</CfKeyValue>
			</section>

			<section class="_formBlock">
				<h2 class="sectionTitle">{{ t("other") }}</h2>
				<CfKeyValue oneline>
					<template #key>{{ t("administrator") }}</template>
					<template #value>{{
						account.isAdmin ? t("yes") : t("no")
					}}</template>
				</CfKeyValue>
				<CfKeyValue oneline>
					<template #key>{{ t("moderator") }}</template>
					<template #value>{{
						account.isModerator ? t("yes") : t("no")
					}}</template>
				</CfKeyValue>
				<CfKeyValue oneline>
					<template #key>emailVerified</template>
					<template #value>{{
						account.emailVerified === true ? t("yes") : t("no")
					}}</template>
				</CfKeyValue>
			</section>

			<section class="_formBlock">
				<ul class="unavailableList">
					<li v-for="item in unavailableItems" :key="item.title" class="item">
						<span class="name">{{ item.title }}</span>
						<span class="badge">{{ t("unavailable") }}</span>
					</li>
				</ul>
			</section>
		</template>
	</div>
</template>

<script lang="ts" setup>
// Firefish fetches `users/stats` for the statistics block. Cinderfish has no
// such endpoint (docs/research/api-contract/auth-users.md), but `i` already
// carries notesCount / followingCount / followersCount, so the block is filled
// from the profile and refreshed once on mount. Password change, 2FA and
// account deletion have no endpoints at all, so they are listed, not rendered.
import { computed, onMounted, ref } from "vue";
import { me, refreshAccount } from "@/account";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfKeyValue from "@/components/ui/CfKeyValue.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import CfTime from "@/components/ui/CfTime.vue";
import number from "@/filters/number";
import { t } from "@/i18n";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageSettingsAccountInfo" });

const account = computed(() => me.value);
const loading = ref(false);
const loadError = ref<string | null>(null);

interface UnavailableItem {
	title: string;
}

const unavailableItems = computed<UnavailableItem[]>(() => [
	{ title: t("changePassword") },
	{ title: t("twoStepAuthentication") },
	{ title: t("deleteAccount") },
]);

async function load(): Promise<void> {
	if (me.value == null) return;
	loading.value = true;
	loadError.value = null;
	try {
		// refreshAccount() posts `i` through the account layer, so the page never
		// touches the token itself; it resolves null on a lost session (which the
		// template then shows as signed-out) or on a transport failure.
		const fresh = await refreshAccount();
		if (fresh == null && me.value != null) {
			loadError.value = t("somethingHappened");
		}
	} finally {
		loading.value = false;
	}
}

onMounted(() => {
	void load();
});

definePageMetadata({ title: t("accountInfo"), icon: "ph-info ph-bold ph-lg" });
</script>

<style lang="scss" scoped>
.cf-settings-account-info {
	.monospace {
		font-family: monospace;
		overflow-wrap: anywhere;
	}

	.sectionTitle {
		margin: 0 0 4px;
		color: var(--fg);
		font-size: 1em;
		opacity: 0.85;
	}

	.errorText {
		flex: 1;
		min-inline-size: 0;
	}

	.unavailableList {
		margin: 0;
		padding: 0;
		list-style: none;
		border: solid 1px var(--divider);
		border-radius: var(--radius);

		> .item {
			display: flex;
			align-items: center;
			gap: 12px;
			padding: 10px 14px;

			&:not(:first-child) {
				border-block-start: solid 1px var(--divider);
			}
		}

		.name {
			flex: 1;
			min-inline-size: 0;
			color: var(--fg);
		}

		.badge {
			flex-shrink: 0;
			padding: 2px 8px;
			border-radius: 999px;
			background: var(--accentedBg);
			color: var(--fgTransparentWeak);
			font-size: 0.75em;
		}
	}
}
</style>
