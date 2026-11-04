<template>
	<CfA v-if="user" class="cf-account-button" :class="{ compact }" :to="userPage(user)">
		<CfAvatar class="avatar" :user="user" disable-link />
		<div class="body">
			<div class="name">
				<CfUserName :user="user" />
			</div>
			<div class="acct">{{ acct(user) }}</div>
		</div>
	</CfA>

	<CfA v-else class="cf-account-button guest" to="/">
		<div class="guest-icon ph-bold ph-user-circle"></div>
		<div class="body">
			<div class="name">{{ t("login") }}</div>
			<div class="acct">{{ t("enterUsername") }}</div>
		</div>
	</CfA>
</template>

<script setup lang="ts">
import { computed } from "vue";
import type { UserLite } from "../api/types";
import { me } from "../account/me";
import CfA from "../components/ui/CfA.vue";
import CfAvatar from "../components/ui/CfAvatar.vue";
import CfUserName from "../components/ui/CfUserName.vue";
import { acct, userPage } from "../components/ui/icon";
import { useLocale } from "../i18n";

const { t } = useLocale();

withDefaults(
	defineProps<{
		compact?: boolean;
	}>(),
	{
		compact: false,
	},
);

const user = computed<UserLite | null>(() => me.value);
</script>

<style lang="scss" scoped>
.cf-account-button {
	display: flex;
	align-items: center;
	gap: 10px;
	padding: 8px;
	border-radius: var(--radius);
	transition: background 0.2s;

	&:hover {
		background: var(--buttonBg);
	}

	.avatar {
		--avatarSize: 36px;
	}

	.body {
		min-inline-size: 0;
		line-height: 1.25;
	}

	.name {
		font-weight: bold;
		font-size: 0.9em;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.acct {
		font-size: 0.75em;
		color: var(--fgTransparentWeak);
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.guest-icon {
		font-size: 32px;
		color: var(--fgTransparentWeak);
	}
}
</style>
