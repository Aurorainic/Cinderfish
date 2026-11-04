<template>
	<span class="cf-user-name" :class="{ nowrap }" :title="user.name ?? user.username">
		{{ user.name ?? user.username }}
	</span>
</template>

<script setup lang="ts">
import type { UserLite } from "../../api/types";

withDefaults(
	defineProps<{
		user: Pick<UserLite, "name" | "username">;
		nowrap?: boolean;
	}>(),
	{
		nowrap: true,
	},
);
</script>

<style lang="scss" scoped>
.cf-user-name {
	unicode-bidi: isolate;
	overflow-wrap: anywhere;

	&.nowrap {
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}
}
</style>
