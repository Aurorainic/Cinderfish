// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/components/MkFollowButton.vue
//
// Adaptations:
// - blocking/* and following/requests/* are dropped: the backend has neither
//   endpoint and UserDetailed.isBlocking/isLocked are fixed false, so the
//   blocked / follow-request branches can never trigger.
// - The "main" streaming channel subscription is dropped; the parent refreshes
//   the user through the `refresh` event instead.
// - os.api endpoint names map to the typed methods of web/src/api/client.ts.
<script lang="ts" setup>
import { ref, watch } from "vue";
import { api, ApiError } from "../../api/client";
import type { UserDetailed } from "../../api/types";
import MkButton from "../../components/ui/MkButton.vue";
import { i18n } from "../../i18n";
import { isSignedIn, me } from "../../me";
import { alert, confirm } from "../../os";

const props = withDefaults(
	defineProps<{
		user: UserDetailed;
		full?: boolean;
		large?: boolean;
	}>(),
	{ full: false, large: false },
);

const emit = defineEmits<{ refresh: [] }>();

const isFollowing = ref(props.user.isFollowing ?? false);
const wait = ref(false);

watch(
	() => props.user,
	(user) => {
		isFollowing.value = user.isFollowing ?? false;
	},
);

const label = ref("");

async function onClick() {
	if (!isSignedIn(me) || wait.value) return;
	if (me.id === props.user.id) return;
	wait.value = true;
	try {
		if (isFollowing.value) {
			const { canceled } = await confirm({
				type: "warning",
				text: i18n.t("unfollowConfirm", {
					name: props.user.name || props.user.username,
				}),
			});
			if (canceled) return;
			await api.unfollow(props.user.id);
			isFollowing.value = false;
			emit("refresh");
		} else {
			await api.follow(props.user.id);
			isFollowing.value = true;
			emit("refresh");
		}
	} catch (err) {
		const message =
			err instanceof ApiError && err.message ? err.message : i18n.ts.somethingHappened;
		void alert({ type: "error", text: message });
	} finally {
		wait.value = false;
	}
}
</script>

<template>
	<MkButton
		v-if="isSignedIn(me) && me.id !== user.id"
		class="follow-button"
		:class="{ full, large, active: isFollowing }"
		:primary="!isFollowing"
		rounded
		:disabled="wait"
		@click.stop="onClick"
	>
		<template v-if="wait">
			<span>{{ (label = i18n.ts.processing) }}</span>
		</template>
		<template v-else-if="isFollowing">
			<span>{{ (label = i18n.ts.unfollow) }}</span>
		</template>
		<template v-else>
			<span>{{ (label = i18n.ts.follow) }}</span>
		</template>
	</MkButton>
</template>

<style scoped>
.follow-button {
	display: inline-flex;
	align-items: center;
	font-weight: bold;
}

.follow-button.full {
	inline-size: 100%;
}

.follow-button.active :deep(.content) {
	color: var(--accent);
}
</style>
