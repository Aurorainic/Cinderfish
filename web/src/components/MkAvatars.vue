<!-- Ported from firefish@b425f61:packages/client/src/components/MkAvatars.vue (AGPL-3.0-only) -->
<template>
	<div class="defgtij">
		<div v-for="user in users" :key="user.id" class="avatar-holder">
			<MkAvatar :user="user" :show-indicator="true" class="avatar" />
		</div>
	</div>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue";
import type { UserDetailed } from "@/api/types";
import { api } from "@/api/client";
import { ENDPOINTS } from "@/api/endpoints";

/**
 * Ported from Firefish. users/show only accepts a single userId in Cinderfish
 * (docs/research/api-contract/auth-users.md), so the list is fetched with one
 * request per user; unresolved ids are skipped instead of rendering an empty
 * avatar. The list is capped so a large group cannot fan out unbounded.
 */
const props = defineProps<{
	userIds: string[];
}>();

const MAX_AVATARS = 24;

const users = ref<UserDetailed[]>([]);

onMounted(async () => {
	const ids = props.userIds.slice(0, MAX_AVATARS);
	const fetched = await Promise.all(
		ids.map((userId) => api(ENDPOINTS.usersShow, { userId }).catch(() => null)),
	);
	users.value = fetched.filter((user): user is UserDetailed => user != null);
});
</script>

<style lang="scss">
.defgtij {
	padding: 12px;
	grid-gap: 12px;
	display: grid;
	grid-template-columns: repeat(auto-fill, minmax(30px, 40px));
	place-content: center;

	> .avatar-holder {
		aspect-ratio: 1;

		> .avatar {
			width: 100%;
			height: 100%;
			aspect-ratio: 1;
		}
	}
}
</style>