<!-- Ported from firefish@b425f61:packages/client/src/components/MkFollowButton.vue (AGPL-3.0-only) -->
<template>
	<button
		v-if="me == null || me.id !== user.id"
		class="kpoogebi _button follow-button"
		:class="{
			wait,
			active: isFollowing || hasPendingFollowRequestFromYou,
			full,
			large,
		}"
		:disabled="wait"
		:aria-label="`${state} ${user.name || user.username}`"
		v-tooltip="full ? null : `${state} ${user.name || user.username}`"
		@click.stop="onClick"
	>
		<template v-if="!wait">
			<template v-if="hasPendingFollowRequestFromYou && user.isLocked">
				<span>{{ t("followRequestPending") }}</span>
				<i class="ph-hourglass-medium ph-bold ph-lg"></i>
			</template>
			<template v-else-if="hasPendingFollowRequestFromYou">
				<span>{{ t("processing") }}</span>
				<i class="ph-circle-notch ph-bold ph-lg fa-pulse"></i>
			</template>
			<template v-else-if="isFollowing">
				<span>{{ t("unfollow") }}</span>
				<i class="ph-minus ph-bold ph-lg"></i>
			</template>
			<template v-else-if="user.isLocked">
				<span>{{ t("followRequest") }}</span>
				<i class="ph-lock-open ph-bold ph-lg"></i>
			</template>
			<template v-else>
				<span>{{ t("follow") }}</span>
				<i class="ph-plus ph-bold ph-lg"></i>
			</template>
		</template>
		<template v-else>
			<span>{{ t("processing") }}</span>
			<i class="ph-circle-notch ph-bold ph-lg fa-pulse ph-fw ph-lg"></i>
		</template>
	</button>
</template>

<script lang="ts" setup>
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import { api } from "@/api/client";
import type { UserDetailed } from "@/api/types";
import { me } from "@/account";
import { t } from "@/i18n";
import * as os from "@/os";
import { pleaseLogin } from "@/scripts/please-login";
import { stream } from "@/stream";

// Firefish renders a "..." menu button here backed by `getUserMenu`. The user
// menu needs mute/block/user-relation endpoints this backend does not have
// (contract §5), so the menu button is dropped and the `hideMenu` prop is kept
// only for call-site compatibility.

const props = withDefaults(
	defineProps<{
		user: UserDetailed;
		full?: boolean;
		large?: boolean;
		hideMenu?: boolean;
	}>(),
	{
		full: false,
		large: false,
		hideMenu: false,
	},
);

const emit = defineEmits<{
	refresh: [];
}>();

const isFollowing = ref(props.user.isFollowing);
const hasPendingFollowRequestFromYou = ref(
	props.user.hasPendingFollowRequestFromYou,
);
const wait = ref(false);

const state = computed(() => {
	if (wait.value) return t("processing");
	if (hasPendingFollowRequestFromYou.value) {
		return props.user.isLocked ? t("followRequestPending") : t("processing");
	}
	if (isFollowing.value) return t("unfollow");
	if (props.user.isLocked) return t("followRequest");
	return t("follow");
});

let offFollowed: (() => void) | null = null;

async function refreshRelation(): Promise<void> {
	// `users/show` fills the relation flags only when a token is sent
	// (docs/research/api-contract/following-relations.md), so this is a no-op
	// for visitors.
	try {
		const user = await api("users/show", { userId: props.user.id });
		if (user.id !== props.user.id) return;
		isFollowing.value = user.isFollowing;
		hasPendingFollowRequestFromYou.value =
			user.hasPendingFollowRequestFromYou;
	} catch (error) {
		console.error(error);
	}
}

async function onClick(): Promise<void> {
	if (me.value == null) {
		pleaseLogin();
		return;
	}

	wait.value = true;

	try {
		if (isFollowing.value) {
			const { canceled } = await os.confirm({
				type: "warning",
				text: t("unfollowConfirm", {
					name: props.user.name || props.user.username,
				}),
			});

			if (canceled) return;

			await api("following/delete", { userId: props.user.id });
			isFollowing.value = false;
			hasPendingFollowRequestFromYou.value = false;
			emit("refresh");
		} else if (hasPendingFollowRequestFromYou.value) {
			// `following/delete` cannot undo a pending request; the cancel
			// endpoint is the right one for that state.
			await api("following/requests/cancel", { userId: props.user.id });
			hasPendingFollowRequestFromYou.value = false;
		} else {
			await api("following/create", { userId: props.user.id });
			// Local unlocked accounts are followed immediately; remote targets
			// also get an optimistic pending row (following-relations.md).
			if (props.user.host == null && !props.user.isLocked) {
				isFollowing.value = true;
			} else {
				hasPendingFollowRequestFromYou.value = true;
			}
			emit("refresh");
		}
	} catch (error) {
		console.error(error);
	} finally {
		wait.value = false;
	}
}

onMounted(() => {
	if (me.value == null || me.value.id === props.user.id) return;

	// `main` is a per-connection singleton: only this handler is removed on
	// unmount, never the shared subscription.
	const channel = stream.useChannel("main");
	offFollowed = channel.on("followed", (payload: { id: string }) => {
		if (payload.id === props.user.id) void refreshRelation();
	});

	void refreshRelation();
});

onBeforeUnmount(() => {
	offFollowed?.();
	offFollowed = null;
});
</script>

<style lang="scss" scoped>
.menu {
	width: 3em;
	height: 2em;
	vertical-align: middle;
}
.follow-button {
	position: relative;
	display: inline-flex;
	align-items: center;
	justify-content: center;
	font-weight: bold;
	color: var(--accent);
	border: solid 1px var(--accent);
	padding: 0;
	font-size: 16px;
	width: 2em;
	height: 2em;
	border-radius: 100px;
	background: var(--bg);
	vertical-align: middle;

	&.full {
		padding: 0.2em 0.7em;
		width: auto;
		font-size: 14px;
	}

	&.large {
		font-size: 16px;
		height: 38px;
		padding: 0 12px 0 16px;
	}

	&:not(.full) {
		width: 31px;
		span {
			display: none;
		}
	}

	&:focus-visible {
		&:after {
			content: "";
			pointer-events: none;
			position: absolute;
			top: -5px;
			right: -5px;
			bottom: -5px;
			left: -5px;
			border: 2px solid var(--focus);
			border-radius: 32px;
		}
	}

	&.active {
		color: var(--fgOnAccent);
		background: var(--accent);

		&:hover {
			background: var(--accentLighten);
			border-color: var(--accentLighten);
		}

		&:active {
			background: var(--accentDarken);
			border-color: var(--accentDarken);
		}
	}

	&.wait {
		cursor: wait;
		opacity: 0.7;
	}

	> span {
		margin-right: 6px;
	}
}
</style>
