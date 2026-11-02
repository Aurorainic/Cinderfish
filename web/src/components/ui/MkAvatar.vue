// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/components/global/MkAvatar.vue
//
// Simplifications versus the original:
// - no PhotoSwipe lightbox (TODO Cinderfish has no photoswipe dependency;
//   showLightBox degrades to plain display)
// - no online-status indicator dot (TODO Cinderfish has no
//   MkUserOnlineIndicator equivalent component yet)
// - blurhash avg colour accent kept, static-image toggle dropped (Cinderfish
//   serves a single avatar URL variant)
// - cat-avatar / square-avatar shape options kept via CSS
<template>
	<span
		v-if="disableLink"
		class="cinderfish-avatar"
		:class="{ cat: user.isCat, square: squareAvatars }"
		:style="{ color: color }"
		:title="title"
		@click="emit('click', $event)"
	>
		<img v-if="url" class="inner" :src="url" decoding="async" />
		<span v-else class="inner placeholder" aria-hidden="true">{{ initial }}</span>
	</span>
	<MkA
		v-else
		class="cinderfish-avatar"
		:class="{ cat: user.isCat, square: squareAvatars }"
		:style="{ color: color }"
		:to="to"
		:title="title"
		@click.stop
	>
		<img v-if="url" class="inner" :src="url" decoding="async" />
		<span v-else class="inner placeholder" aria-hidden="true">{{ initial }}</span>
	</MkA>
</template>

<script lang="ts" setup>
import { computed, ref, watch } from "vue";
import type { UserLite } from "../../api/types";
import MkA from "./MkA.vue";

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
	click: [v: MouseEvent];
}>();

// TODO: Cinderfish has no defaultStore yet; square avatars default off and
// the "show LightBox in gallery" mode is not wired to a lightbox runtime.
const squareAvatars = false;

const url = computed(() => props.user.avatarUrl);

// Accounts without an uploaded avatar get Firefish's blank-avatar treatment:
// a panel-coloured disc with the first character of the name, instead of a
// broken <img>.
const initial = computed(() => (props.user.name || props.user.username || "?").slice(0, 1).toUpperCase());

const to = computed(() => `/@${props.user.username}${props.user.host ? `@${props.user.host}` : ""}`);
const title = computed(
	() => `@${props.user.username}${props.user.host ? `@${props.user.host}` : ""}`,
);

const color = ref<string | undefined>();

watch(
	() => props.user.avatarBlurhash,
	() => {
		color.value = extractAvgColorFromBlurhash(props.user.avatarBlurhash);
	},
	{ immediate: true },
);

// Ported from firefish@02c31e0:packages/client/src/scripts/extract-avg-color-from-blurhash
// (small enough to inline; Cinderfish has no blurhash npm dependency).
function extractAvgColorFromBlurhash(hash: string | null): string | undefined {
	if (!hash) return undefined;
	try {
		const bin = atob(hash);
		const quantMaxA = bin[1].charCodeAt(0) + 1;
		const maxA = ((quantMaxA + 1) * 63) / 64;
		const quantR = bin[2].charCodeAt(0);
		const quantG = bin[3].charCodeAt(0);
		const quantB = bin[4].charCodeAt(0);
		const r = Math.floor((quantR / 255) * maxA);
		const g = Math.floor((quantG / 255) * maxA);
		const b = Math.floor((quantB / 255) * maxA);
		return `rgb(${r},${g},${b})`;
	} catch {
		return undefined;
	}
}
</script>

<style scoped>
.cinderfish-avatar {
	position: relative;
	display: inline-block;
	vertical-align: bottom;
	border-radius: 100%;
	overflow: clip;
	flex-shrink: 0;
	/* The box doubles as the placeholder's size container, so the initial
	   scales with whatever the caller sizes the avatar to. */
	container-type: size;
}

/* Zero specificity: Firefish sizes avatars from the call site (1.2em in a
   renote bar, 48px in a note header), so a parent class must win over our
   default. */
:where(.cinderfish-avatar) {
	inline-size: var(--avatar-size, 48px);
	block-size: var(--avatar-size, 48px);
}

.cinderfish-avatar.square {
	border-radius: 15%;
}

.cinderfish-avatar > .inner {
	inline-size: 100%;
	block-size: 100%;
	object-fit: cover;
	-webkit-user-drag: none;
	user-select: none;
}

.cinderfish-avatar > .placeholder {
	display: grid;
	place-items: center;
	background: var(--panelHighlight, var(--panel));
	color: var(--fgTransparentWeak, var(--fg));
	font-size: 45cqw;
	font-weight: bold;
}

/* Firefish draws "cat ears" on cat accounts; the simplified Cinderfish port
   renders the same rounded clipping only. */
.cinderfish-avatar.cat {
	border-radius: 100%;
}
</style>
