<!-- Ported from firefish@b425f61:packages/client/src/components/global/MkAvatar.vue (AGPL-3.0-only) -->
<template>
	<span
		v-if="disableLink"
		v-user-preview="disablePreview ? undefined : user.id"
		class="eiwwqkts _noSelect"
		:class="{ cat: user.isCat, square, noAvatar }"
		:style="{ color }"
		:title="acct(user)"
		@click="onClick"
	>
		<img v-if="!noAvatar" class="inner" :src="url" decoding="async" alt="" />
		<span
			v-if="showIndicator && user.instance == null"
			class="indicator"
			:class="{ online: user.onlineStatus === 'online' }"
		></span>
	</span>
	<MkA
		v-else
		v-user-preview="disablePreview ? undefined : user.id"
		class="eiwwqkts _noSelect"
		:class="{ cat: user.isCat, square, noAvatar }"
		:style="{ color }"
		:to="userPage(user)"
		:title="acct(user)"
		:target="target"
		@click.stop
	>
		<img v-if="!noAvatar" class="inner" :src="url" decoding="async" alt="" />
		<span
			v-if="showIndicator && user.instance == null"
			class="indicator"
			:class="{ online: user.onlineStatus === 'online' }"
		></span>
	</MkA>
</template>

<script setup lang="ts">
import { computed } from "vue";
import type { UserLite } from "../../api/types";
import { resolveMediaUrl } from "../../api/client";
import { acct, userPage } from "../ui/user";
import { defaultStore } from "../../store";
import MkA from "./MkA.vue";

// Degraded versus Firefish: MkUserOnlineIndicator and the blurhash average
// colour are not ported. The online dot is drawn from user.onlineStatus;
// avatarBlurhash is always null on Cinderfish, so no placeholder colour is
// derived. The hover preview is provided by v-user-preview.
const props = withDefaults(
	defineProps<{
		user: UserLite;
		target?: string | null;
		disableLink?: boolean;
		/** Accepted for template compatibility; no preview popup exists yet. */
		disablePreview?: boolean;
		/** Accepted for template compatibility; no light box exists yet. */
		showLightBox?: boolean;
		showIndicator?: boolean;
	}>(),
	{
		target: null,
		disableLink: false,
		disablePreview: false,
		showLightBox: false,
		showIndicator: false,
	},
);

const emit = defineEmits<{
	click: [event: MouseEvent];
}>();

// avatarUrl comes from user_profiles.avatar_url (i/update with avatarId); an
// account without one would render a broken-image box, so it falls back to a
// placeholder asset and the `noAvatar` flag hides the <img>.
const url = computed<string>(() =>
	resolveMediaUrl(props.user.avatarUrl) || "/favicon.svg",
);
const noAvatar = computed<boolean>(() => props.user.avatarUrl == null);

// Firefish reads defaultStore.state.squareAvatars; cat ears stay on top of it.
const square = computed<boolean>(() => defaultStore.state.squareAvatars);

const color = computed<string | undefined>(() => undefined);

function onClick(ev: MouseEvent): void {
	emit("click", ev);
}
</script>

<style lang="scss" scoped>
// No avatar image on this instance: draw a neutral user glyph instead of the
// browser's broken-image placeholder.
.eiwwqkts.noAvatar {
	background: var(--panelHighlight);
	border: solid 1px var(--divider);
	color: var(--fgTransparentWeak);
}
.eiwwqkts.noAvatar::before {
	content: "";
	position: absolute;
	inset: 0;
	border-radius: inherit;
	background: url("/static-assets/avatar.svg") center / 58% no-repeat;
	opacity: 0.55;
}
@keyframes earwiggleleft {
	from {
		transform: rotate(37.6deg) skew(30deg);
	}
	25% {
		transform: rotate(10deg) skew(30deg);
	}
	50% {
		transform: rotate(20deg) skew(30deg);
	}
	75% {
		transform: rotate(0deg) skew(30deg);
	}
	to {
		transform: rotate(37.6deg) skew(30deg);
	}
}

@keyframes earwiggleright {
	from {
		transform: rotate(-37.6deg) skew(-30deg);
	}
	30% {
		transform: rotate(-10deg) skew(-30deg);
	}
	55% {
		transform: rotate(-20deg) skew(-30deg);
	}
	75% {
		transform: rotate(0deg) skew(-30deg);
	}
	to {
		transform: rotate(-37.6deg) skew(-30deg);
	}
}

.eiwwqkts {
	position: relative;
	display: inline-block;
	vertical-align: bottom;
	flex-shrink: 0;
	border-radius: 100%;
	line-height: 16px;
	inline-size: var(--avatarSize);
	block-size: var(--avatarSize);

	> .inner {
		position: absolute;
		bottom: 0;
		left: 0;
		right: 0;
		top: 0;
		border-radius: 100%;
		z-index: 1;
		overflow: hidden;
		object-fit: cover;
		width: 100%;
		height: 100%;
	}

	> .indicator {
		position: absolute;
		z-index: 1;
		bottom: 0;
		left: 0;
		width: 18%;
		height: 18%;
		border-radius: 100%;
		background: var(--fgTransparent);
		border: solid 2px var(--panel);

		&.online {
			background: var(--success);
		}
	}

	&.square {
		border-radius: 20%;

		> .inner {
			border-radius: 20%;
		}
	}

	&.cat {
		&:before,
		&:after {
			background: #ebbcba;
			border: solid 4px currentColor;
			box-sizing: border-box;
			content: "";
			display: inline-block;
			height: 50%;
			width: 50%;
			position: absolute;
			z-index: 2;
		}

		&:before {
			border-radius: 0 75% 75%;
			transform: rotate(37.5deg) skew(30deg);
		}

		&:after {
			border-radius: 75% 0 75% 75%;
			transform: rotate(-37.5deg) skew(-30deg);
		}

		&:hover {
			&:before {
				animation: earwiggleleft 1s infinite;
			}
		}
	}
}
</style>
