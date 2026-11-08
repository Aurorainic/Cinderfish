<!-- Ported from firefish@b425f61:packages/client/src/components/MkChatPreview.vue (AGPL-3.0-only) -->
<template>
	<CfA
		class="cf-chat-preview"
		:class="{
			isMe: isMe,
			isRead: message.isRead,
		}"
		:to="`/my/messaging/${peerAcct}`"
	>
		<div class="message _panel">
			<CfAvatar
				class="avatar"
				:user="peer"
				:show-indicator="true"
				disable-link
			/>
			<header>
				<span class="name"><CfUserName :user="peer" /></span>
				<span class="username">@{{ peerAcct }}</span>
				<CfTime :time="message.createdAt" class="time" />
			</header>
			<div class="body">
				<p class="text">
					<span v-if="isMe" class="me">{{ t("you") }}: </span>
					<template v-if="message.text != null && message.text.length > 0">
						{{ message.text }}
					</template>
					<i v-else class="ph-paperclip ph-bold ph-lg"></i>
				</p>
			</div>
		</div>
	</CfA>
</template>

<script setup lang="ts">
// Firefish's MkChatPreview with the group branch removed: Cinderfish only has
// the 1:1 conversations the legacy messaging contract supports. The row opens
// the peer's room at `/my/messaging/:acct`; `filters/user` keeps the handle
// without the leading "@" the route parser accepts either way.
import { computed } from "vue";
import { me } from "@/account";
import type { MessagingMessage } from "@/api/types";
import CfA from "@/components/ui/CfA.vue";
import CfAvatar from "@/components/ui/CfAvatar.vue";
import CfTime from "@/components/ui/CfTime.vue";
import CfUserName from "@/components/ui/CfUserName.vue";
import { acct } from "@/filters/user";
import { t } from "@/i18n";

defineOptions({ name: "MkChatPreview" });

const props = defineProps<{
	message: MessagingMessage;
}>();

const isMe = computed(
	() => me.value != null && props.message.userId === me.value.id,
);

/**
 * `messaging/history` embeds `recipient`; a live `messagingIndex` event does
 * not, so the index page refreshes after each event instead of inserting the
 * raw payload. The fallback keeps the row renderable if that ever changes.
 */
const peer = computed(() => {
	if (isMe.value && props.message.recipient != null) {
		return props.message.recipient;
	}
	return props.message.user;
});

const peerAcct = computed(() => acct(peer.value));
</script>

<style lang="scss" scoped>
.cf-chat-preview {
	display: block;
	text-decoration: none;

	> .message {
		display: block;
		margin-block-end: var(--margin);
		padding-block: 16px;
		padding-inline: 20px;
		border-radius: var(--radius);

		&:hover {
			border-color: var(--accent);
		}

		&:after {
			content: "";
			display: block;
			clear: both;
		}
	}

	&.isRead,
	&.isMe {
		opacity: 0.8;
	}

	&:not(.isMe):not(.isRead) > .message {
		background-color: var(--accentedBg);
	}

	> .message > header {
		display: flex;
		align-items: center;
		margin-block-end: 2px;
		white-space: nowrap;
		overflow: hidden;

		> .name {
			margin: 0;
			padding: 0;
			overflow: hidden;
			text-overflow: ellipsis;
			font-weight: bold;
		}

		> .username {
			margin-block: 0;
			margin-inline: 8px;
			color: var(--fgTransparentWeak);
		}

		> .time {
			margin-block: 0;
			margin-inline-start: auto;
			color: var(--fgTransparentWeak);
			font-size: 0.85em;
		}
	}

	> .message > .avatar {
		float: inline-start;
		inline-size: 54px;
		block-size: 54px;
		margin-block-start: 0;
		margin-inline-end: 16px;
		margin-block-end: 0;
		margin-inline-start: 0;
		border-radius: 8px;
	}

	> .message > .body > .text {
		display: block;
		margin: 0;
		padding: 0;
		overflow: hidden;
		overflow-wrap: break-word;
		font-size: 1.05em;
		color: var(--fg);

		> .me {
			opacity: 0.7;
		}
	}

	@media (max-width: 500px) {
		> .message {
			padding-block: 12px;
			padding-inline: 16px;

			> .avatar {
				inline-size: 48px;
				block-size: 48px;
				margin-inline-end: 12px;
			}
		}
	}
}
</style>
