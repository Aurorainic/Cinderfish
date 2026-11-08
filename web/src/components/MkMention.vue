<!-- Ported from firefish@b425f61:packages/client/src/components/MkMention.vue (AGPL-3.0-only) -->
<template>
	<MkA class="mention" :class="{ isMe }" :to="url" @click.stop>
		<img class="icon" :src="avatarUrl" alt="" />
		<span class="main">
			<span class="username">@{{ username }}</span>
			<!-- An empty host means a local account: it has no @host part, and
			     rendering one would leave a dangling "@" (the moved banner asks
			     for a local target this way). -->
			<span v-if="host !== '' && (host != localHost || showFullAcct)" class="host"
				>@{{ host }}</span
			>
		</span>
	</MkA>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { host as localHost } from "../config";
import { me } from "../account";
import { defaultStore } from "../store";
import MkA from "./global/MkA.vue";

// Degraded versus Firefish: punycode decoding of the host, the user-preview
// popup and the per-instance avatar cache are not ported. Firefish's
// "url.startsWith('/')" test is always true (the mention path is local), so the
// dead external-anchor branch is dropped.
const props = defineProps<{
	username: string;
	host: string;
}>();

const url = computed<string>(
	() => "/@" + props.username + (props.host === localHost ? "" : "@" + props.host),
);

const avatarUrl = computed<string>(
	() => "/avatar/@" + props.username + "@" + props.host,
);

const showFullAcct = computed<boolean>(() => defaultStore.state.showFullAcct);

const isMe = computed<boolean>(() => {
	const user = me.value;
	if (user == null) return false;
	return (
		"@" + props.username + "@" + props.host ===
		("@" + user.username + "@" + localHost).toLowerCase()
	);
});
</script>

<style lang="scss" scoped>
.mention {
	position: relative;
	display: inline-block;
	padding: 2px 8px 2px 2px;
	margin-block: 2px;
	border-radius: 999px;
	max-width: 100%;
	white-space: nowrap;
	text-overflow: ellipsis;
	color: var(--mention);
	isolation: isolate;

	&::before {
		content: "";
		position: absolute;
		inset: 0;
		border-radius: inherit;
		background: var(--mention);
		opacity: 0.1;
		z-index: -1;
	}

	&.isMe {
		--mention: var(--mentionMe);
	}

	> .icon {
		width: 1.5em;
		height: 1.5em;
		object-fit: cover;
		margin: 0 0.2em 0 0;
		vertical-align: bottom;
		border-radius: 100%;
	}

	> .main > .host {
		color: var(--fgTransparentWeak);
	}
}
</style>
