<template>
	<component
		:is="disableLink ? 'span' : CfA"
		class="eiwwqkts _noSelect"
		:class="{ square, cat: user.isCat }"
		:to="disableLink ? undefined : userPage(user)"
		:title="acct(user)"
		@click.stop="emit('click', $event)"
	>
		<img v-if="!noAvatar" class="inner" :src="src" decoding="async" alt="" />
		<span v-else class="inner no-avatar ph-bold ph-user"></span>
		<span
			v-if="showIndicator"
			class="indicator"
			:class="{ online: user.onlineStatus === 'online' }"
		></span>
	</component>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { resolveMediaUrl } from "../../api/client";
import type { UserLite } from "../../api/types";
import { defaultStore } from "../../store/store";
import CfA from "./CfA.vue";
import { acct, userPage } from "./icon";

const props = withDefaults(
	defineProps<{
		user: UserLite;
		target?: string | null;
		disableLink?: boolean;
		showLightBox?: boolean;
		disablePreview?: boolean;
		showIndicator?: boolean;
	}>(),
	{
		target: null,
		disableLink: false,
		showLightBox: false,
		disablePreview: false,
		showIndicator: false,
	},
);

const emit = defineEmits<{
	click: [event: MouseEvent];
}>();

// Cinderfish accounts have no avatar upload yet (avatarUrl is always null), so
// render a neutral placeholder instead of the browser's broken-image box.
const src = computed(() => resolveMediaUrl(props.user.avatarUrl) || "/favicon.svg");
const noAvatar = computed(() => props.user.avatarUrl == null);

const square = computed(() =>
	props.user.isCat
		? defaultStore.state.squareCatAvatars
		: defaultStore.state.squareAvatars,
);
</script>

<style lang="scss" scoped>
.eiwwqkts {
	position: relative;
	display: inline-block;
	flex-shrink: 0;
	inline-size: var(--avatarSize);
	block-size: var(--avatarSize);
	border-radius: 50%;
	overflow: hidden;
	background: var(--panelHighlight);
	vertical-align: bottom;

	&.square {
		border-radius: var(--radius);
	}

	.inner {
		display: block;
		inline-size: 100%;
		block-size: 100%;
		object-fit: cover;
	}

	.indicator {
		position: absolute;
		inset-inline-end: 2px;
		inset-block-end: 2px;
		inline-size: 10px;
		block-size: 10px;
		border-radius: 50%;
		background: var(--fgTransparent);
		border: solid 2px var(--panel);

		&.online {
			background: var(--success);
		}
	}
}
</style>
