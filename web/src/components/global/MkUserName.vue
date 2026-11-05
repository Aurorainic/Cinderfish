<!-- Ported from firefish@b425f61:packages/client/src/components/global/MkUserName.vue (AGPL-3.0-only) -->
<template>
	<Mfm
		class="mk-user-name"
		:text="user.name || user.username"
		:plain="true"
		:nowrap="nowrap"
		:custom-emojis="customEmojis"
	/>
</template>

<script setup lang="ts">
import { computed } from "vue";
import type { EmojiSimple, UserLite } from "../../api/types";
import Mfm from "./MkMisskeyFlavoredMarkdown.vue";

const props = withDefaults(
	defineProps<{
		user: Pick<UserLite, "name" | "username" | "emojis">;
		nowrap?: boolean;
	}>(),
	{
		nowrap: true,
	},
);

const customEmojis = computed<EmojiSimple[]>(
	() => (props.user.emojis ?? []) as unknown as EmojiSimple[],
);
</script>

<style lang="scss" scoped>
.mk-user-name {
	unicode-bidi: isolate;
}
</style>
