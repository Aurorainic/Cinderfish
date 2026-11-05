<!-- Ported from firefish@b425f61:packages/client/src/pages/user-info.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :display-back-button="true" />
		</template>
		<MkSpacer :content-max="600" :margin-min="16" :margin-max="32">
			<div v-if="user" class="aeakzknw">
				<div class="card _panel">
					<MkAvatar class="avatar" :user="user" :show-indicator="true" />
					<div class="body">
						<span class="name"><MkUserName :user="user" /></span>
						<span class="sub"
							><span class="acct _monospace">@{{ acct(user) }}</span></span
						>
						<span class="state">
							<span v-if="user.isSuspended" class="suspended">{{
								t("suspend")
							}}</span>
							<span v-if="user.isSilenced" class="silenced">{{
								t("silence")
							}}</span>
							<span v-if="user.isAdmin" class="moderator">{{
								t("isAdmin")
							}}</span>
							<span v-else-if="user.isModerator" class="moderator">{{
								t("isModerator")
							}}</span>
						</span>
					</div>
				</div>

				<div class="links">
					<CfA class="link _panel" :to="userPage(user)">
						<i class="ph-user ph-bold ph-lg"></i>
						{{ t("profile") }}
					</CfA>
				</div>

				<div class="card _panel">
					<CfKeyValue :copy="user.id" oneline class="kv">
						<template #key>ID</template>
						<template #value
							><span class="_monospace">{{ user.id }}</span></template
						>
					</CfKeyValue>
					<CfKeyValue oneline class="kv">
						<template #key>{{ t("createdAt") }}</template>
						<template #value
							><MkTime :time="user.createdAt" mode="detail"
						/></template>
					</CfKeyValue>
					<CfKeyValue oneline class="kv">
						<template #key>{{ t("host") }}</template>
						<template #value
							><span class="_monospace">{{ user.host ?? host }}</span></template
						>
					</CfKeyValue>
				</div>

				<div v-if="canModerate" class="card _panel">
					<CfButton
						v-if="!user.isSuspended"
						class="action"
						@click="setSuspended(true)"
					>
						<i class="ph-snowflake ph-bold ph-lg"></i>
						{{ t("suspend") }}
					</CfButton>
					<CfButton v-else class="action" @click="setSuspended(false)">
						<i class="ph-sun ph-bold ph-lg"></i>
						{{ t("unsuspend") }}
					</CfButton>
				</div>
			</div>
			<MkError v-else-if="error != null" @retry="fetchUser" />
			<CfLoading v-else />
		</MkSpacer>
	</MkStickyContainer>
</template>

<script lang="ts" setup>
// Firefish's user-info page is a three-tab admin console (overview / moderation
// / chart) built on `admin/user-ips`, `admin/user`, moderation notes and the
// charts API. None of those endpoints exist on Cinderfish, so the page keeps the
// part `users/show` can answer — identity, id, creation date, host and the
// admin's suspend toggle (`admin/suspend-user` / `admin/unsuspend-user`) — and
// drops the tab bar instead of rendering dead controls.
import { computed, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { api } from "@/api/client";
import type { UserDetailed } from "@/api/types";
import { me } from "@/account";
import { host } from "@/config";
import CfA from "@/components/ui/CfA.vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfKeyValue from "@/components/ui/CfKeyValue.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import { acct, userPage } from "@/filters/user";
import { t } from "@/i18n";
import * as os from "@/os";
import { definePageMetadata } from "@/scripts/page-metadata";

const route = useRoute();

const userId = computed<string>(() => {
	const value = route.params.userId;
	return typeof value === "string" ? value : "";
});

const user = ref<UserDetailed | null>(null);
const error = ref<unknown>(null);

const canModerate = computed<boolean>(
	() =>
		me.value?.isAdmin === true &&
		user.value != null &&
		user.value.id !== me.value.id,
);

async function fetchUser(): Promise<void> {
	if (userId.value === "") return;
	error.value = null;
	user.value = null;
	try {
		user.value = await api("users/show", { userId: userId.value });
	} catch (cause) {
		error.value = cause;
	}
}

async function setSuspended(suspended: boolean): Promise<void> {
	if (user.value == null) return;
	const confirmed = await os.confirm({
		type: "warning",
		text: suspended ? t("suspendConfirm") : t("unsuspendConfirm"),
	});
	if (confirmed.canceled) return;
	await os.apiWithDialog(
		suspended ? "admin/suspend-user" : "admin/unsuspend-user",
		{ userId: user.value.id },
	);
	await fetchUser();
}

watch(userId, fetchUser, { immediate: true });

definePageMetadata(
	computed(() => ({
		title: t("userInfo"),
		icon: "ph-user ph-bold ph-lg",
		userName: user.value,
	})),
);
</script>

<style lang="scss" scoped>
.aeakzknw {
	display: flex;
	flex-direction: column;
	gap: var(--margin);

	> .card {
		padding: 16px;
	}

	> .card:first-child {
		display: flex;
		align-items: center;
		gap: 12px;

		> .avatar {
			width: 48px;
			height: 48px;
		}

		> .body {
			display: flex;
			flex-direction: column;
			min-width: 0;

			> .name {
				font-weight: bold;
				overflow: hidden;
				text-overflow: ellipsis;
				white-space: nowrap;
			}

			> .sub {
				font-size: 0.85em;
				opacity: 0.7;
			}

			> .state {
				display: flex;
				gap: 8px;
				font-size: 0.75em;

				> .suspended {
					color: var(--error);
				}

				> .silenced {
					color: var(--warn);
				}

				> .moderator {
					color: var(--badge);
				}
			}
		}
	}

	> .links {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
		gap: 8px;

		> .link {
			display: flex;
			align-items: center;
			gap: 8px;
			padding: 12px 16px;
		}
	}

	> .card > .kv {
		margin-block: 4px;
	}

	> .card > .action {
		width: 100%;
	}
}
</style>
