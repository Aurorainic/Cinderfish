// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/components/MkNotification.vue
//
// Adaptations:
// - firefish-js entities.Notification is replaced by the local api/types
//   Notification (same Misskey field names; groupInvited/app/pollVote types
//   are not emitted by the Cinderfish backend and are dropped).
// - Mfm renders as plain text via note/get-note-summary; MkReactionIcon
//   becomes the curated emoji catalog (lib/emoji-catalog); reaction emoji
//   resolution and the default-reaction fallback keep the original's shape.
// - The follow request accept/reject buttons, MkFollowButton and the read
//   IntersectionObserver (marking rows read on sight) are dropped:
//   TODO no following/requests/accept endpoint and no read-marking endpoint
//   in Cinderfish; unread state is store-driven.
// - useTooltip for the reaction preview is dropped; the reaction keeps a
//   native title attribute instead.
<template>
	<div class="notification" :class="[notification.type, { unread: !notification.isRead }]">
		<div class="head">
			<MkAvatar
				v-if="notification.note && notification.type === 'pollEnded'"
				class="icon"
				:user="notification.note.user"
			/>
			<MkAvatar v-else-if="notification.user" class="icon" :user="notification.user" />
			<span v-else class="icon icon-fallback" aria-hidden="true">
				<AppIcon name="bell" />
			</span>
			<span class="sub-icon" :class="notification.type">
				<AppIcon v-if="!(notification.type === 'reaction' && reaction)" :name="subIcon" />
				<span v-else-if="notification.type === 'reaction' && reaction" class="reaction-emoji" :title="reactionTitle">
					{{ reaction }}
				</span>
			</span>
		</div>
		<div class="tail">
			<header>
				<template v-if="notification.type === 'pollEnded'">
					<span class="name">投票已结束</span>
				</template>
				<MkA v-else-if="notification.user" class="name" :to="userPage(notification.user)">
					<MkUserName :user="notification.user" />
				</MkA>
				<span v-else class="name">通知</span>
				<MkTime :time="notification.createdAt" class="time" />
			</header>
			<MkA
				v-if="notification.note && noteTypes.has(notification.type)"
				class="text"
				:to="notePage(notification.note)"
				:title="getNoteSummary(notification.type === 'renote' && notification.note.renote ? notification.note.renote : notification.note)"
			>
				<span class="text-body">{{ noteSummary }}</span>
			</MkA>
			<span v-else-if="notification.type === 'follow'" class="text muted">
				{{ i18n.ts._notification.youWereFollowed }}
			</span>
			<span v-else-if="notification.type === 'followRequestAccepted'" class="text muted">
				{{ i18n.ts._notification.yourFollowRequestAccepted }}
			</span>
			<span v-else-if="notification.type === 'receiveFollowRequest'" class="text muted">
				{{ i18n.ts._notification.youReceivedFollowRequest }}
			</span>
			<span v-else-if="notification.type === 'app' && notification.body" class="text muted">
				{{ notification.body }}
			</span>
		</div>
	</div>
</template>

<script lang="ts" setup>
import { computed } from "vue";
import type { Note, Notification, UserLite } from "../api/types";
import { emojiChar, emojiName } from "../lib/emoji-catalog";
import { i18n } from "../i18n";
import { getNoteSummary } from "./note/get-note-summary";
import AppIcon from "./AppIcon.vue";
import MkA from "./ui/MkA.vue";
import MkAvatar from "./ui/MkAvatar.vue";
import MkTime from "./ui/MkTime.vue";
import MkUserName from "./ui/MkUserName.vue";

// Types whose text line links to the related note, mirroring the original's
// per-type MkA blocks (reaction/renote/reply/mention/quote/pollVote/pollEnded).
const noteTypes = new Set([
	"reaction",
	"renote",
	"reply",
	"mention",
	"quote",
	"pollVote",
	"pollEnded",
]);

const props = defineProps<{ notification: Notification }>();

// The sub-icon maps one-to-one onto the original's ph-* icon chain.
// The sub-icon maps one-to-one onto the original's ph-* icon chain. For a
// reaction notification the badge shows the reaction emoji itself instead of
// an icon, like the original's XReactionIcon slot.
const subIcon = computed(() => {
	switch (props.notification.type) {
		case "follow":
			return "user";
		case "receiveFollowRequest":
			return "clock";
		case "followRequestAccepted":
			return "check";
		case "renote":
			return "renote";
		case "reply":
			return "reply";
		case "mention":
			return "bell";
		case "quote":
			return "quote";
		case "pollVote":
		case "pollEnded":
			return "question";
		default:
			return "smiley";
	}
});

// Firefish resolves :shortcode@.: against the note's customEmojis; the
// curated catalog is Cinderfish's equivalent. A raw unicode reaction shows
// as-is; the default-reaction fallback of the original is kept ("⭐").
const reaction = computed(() => {
	const raw = props.notification.reaction?.trim();
	if (!raw) return "";
	const shortcode = /^:([\w+-]+)(?:@[\w.]*)?:$/.exec(raw);
	if (shortcode) return emojiChar(shortcode[1]) ?? raw;
	return raw;
});

const reactionTitle = computed(() => emojiName(reaction.value));

const noteSummary = computed(() => {
	const note = props.notification.note;
	if (!note) return "";
	// A renote notification shows the boosted note, as the original does with
	// getNoteSummary(notification.note.renote).
	const target =
		props.notification.type === "renote" && note.renote ? note.renote : note;
	return getNoteSummary(target);
});

// Cinderfish's router serves /notes/:id and /@username (filters/note and
// filters/user of the original collapse to these two builders).
function notePage(note: Note): string {
	return `/notes/${note.id}`;
}

function userPage(user: UserLite): string {
	return `/@${user.username}`;
}
</script>

<style scoped>
.notification {
	display: flex;
	gap: 12px;
	padding: 14px 16px;
	border-bottom: 1px solid var(--divider);
	background: var(--panel);
	transition: background 0.15s ease;
}

.notification:hover {
	background: var(--panelHighlight);
}

.notification.unread {
	background: color-mix(in srgb, var(--accent) 8%, var(--panel));
}

/* The type badge overlaps the avatar's corner like the original's .sub-icon. */
.head {
	position: relative;
	flex: none;
	width: 42px;
}

.icon {
	width: 42px;
	height: 42px;
	border-radius: 50%;
}

.icon-fallback {
	display: grid;
	place-items: center;
	background: var(--panel-2);
	color: var(--muted);
}

.icon-fallback .icon {
	width: 18px;
	height: 18px;
}

.sub-icon {
	position: absolute;
	right: -3px;
	bottom: -3px;
	display: grid;
	place-items: center;
	width: 20px;
	height: 20px;
	border-radius: 50%;
	background: var(--panel);
	color: var(--muted);
	font-size: 12px;
	line-height: 1;
}

.sub-icon .icon {
	width: 12px;
	height: 12px;
}

.sub-icon.reaction {
	background: var(--panel);
}

.reaction-emoji {
	max-width: 20px;
	overflow: hidden;
}

.tail {
	flex: 1;
	min-width: 0;
}

header {
	display: flex;
	align-items: baseline;
	gap: 8px;
}

.name {
	font-weight: 700;
	color: var(--text-strong);
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;
}

.time {
	margin-left: auto;
	color: var(--muted);
	font-size: 0.8rem;
	white-space: nowrap;
}

.text {
	display: block;
	margin-top: 2px;
	color: var(--text);
	font-size: 0.92rem;
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;
}

.text-body {
	overflow: hidden;
	text-overflow: ellipsis;
}

.muted {
	opacity: 0.7;
}

/* Narrow viewport: the original's v-size max:[500,450] collapses the layout
   to a tighter row. */
@container (max-width: 500px) {
	.notification {
		gap: 10px;
		padding: 12px;
	}

	.head {
		width: 34px;
	}

	.icon {
		width: 34px;
		height: 34px;
	}

	.sub-icon {
		right: -4px;
		bottom: -4px;
		width: 18px;
		height: 18px;
	}
}
</style>
