<!-- Ported from firefish@b425f61:packages/client/src/components/MkNotification.vue (AGPL-3.0-only) -->
<template>
	<div
		ref="elRef"
		v-size="{ max: [500, 450] }"
		class="qglefbjs notification"
		:class="notification.type"
	>
		<div class="head">
			<MkAvatar
				v-if="notification.user"
				class="icon"
				:user="notification.user"
			/>
			<div class="sub-icon" :class="notification.type">
				<i
					v-if="notification.type === 'renote'"
					class="ph-repeat ph-bold"
				></i>
				<i
					v-else-if="notification.type === 'reply'"
					class="ph-arrow-bend-up-left ph-bold"
				></i>
				<i
					v-else-if="notification.type === 'mention'"
					class="ph-at ph-bold"
				></i>
				<i
					v-else-if="notification.type === 'quote'"
					class="ph-quotes ph-bold"
				></i>
				<MkReactionIcon
					v-else-if="
						showEmojiReactions && notification.type === 'reaction'
					"
					ref="reactionRef"
					:reaction="reactionToShow"
					:no-style="true"
				/>
				<MkReactionIcon
					v-else-if="notification.type === 'reaction'"
					:reaction="defaultReaction"
					:no-style="true"
				/>
			</div>
		</div>
		<div class="tail">
			<header>
				<MkA
					v-if="notification.user"
					v-user-preview="notification.user.id"
					class="name"
					:to="userPage(notification.user)"
					><MkUserName :user="notification.user"
				/></MkA>
				<span v-else class="name">{{ t("unknown") }}</span>
				<MkTime
					v-if="withTime"
					:time="notification.createdAt"
					class="time"
				/>
			</header>
			<MkA
				v-if="notification.type === 'reaction' && notification.note != null"
				class="text"
				:to="notePage(notification.note)"
				:title="summary"
			>
				<span>{{ t("_notification.reacted") }}</span>
				<i class="ph-quotes ph-fill ph-lg"></i>
				<Mfm :text="summary" :plain="true" :nowrap="!full" />
				<i class="ph-quotes ph-fill ph-lg"></i>
			</MkA>
			<MkA
				v-if="notification.type === 'renote' && notification.note != null"
				class="text"
				:to="notePage(notification.note)"
				:title="summary"
			>
				<span>{{ t("_notification.renoted") }}</span>
				<i class="ph-quotes ph-fill ph-lg"></i>
				<Mfm :text="summary" :plain="true" :nowrap="!full" />
				<i class="ph-quotes ph-fill ph-lg"></i>
			</MkA>
			<MkA
				v-if="notification.type === 'reply' && notification.note != null"
				class="text"
				:to="notePage(notification.note)"
				:title="summary"
			>
				<Mfm :text="summary" :plain="true" :nowrap="!full" />
			</MkA>
			<MkA
				v-if="notification.type === 'mention' && notification.note != null"
				class="text"
				:to="notePage(notification.note)"
				:title="summary"
			>
				<Mfm :text="summary" :plain="true" :nowrap="!full" />
			</MkA>
			<MkA
				v-if="notification.type === 'quote' && notification.note != null"
				class="text"
				:to="notePage(notification.note)"
				:title="summary"
			>
				<Mfm :text="summary" :plain="true" :nowrap="!full" />
			</MkA>
			<!-- Cinderfish keeps the note row only while it still exists, so a
			     pruned or soft-deleted target degrades to the bare phrase. -->
			<span
				v-if="notification.note == null && missingNoteText !== ''"
				class="text"
				style="opacity: 0.7"
				>{{ missingNoteText }}</span
			>
		</div>
	</div>
</template>

<script lang="ts" setup>
import { computed, onMounted, onUnmounted, ref, watch } from "vue";
import MkReactionIcon from "@/components/MkReactionIcon.vue";
import MkReactionTooltip from "@/components/MkReactionTooltip.vue";
import { notePage } from "@/filters/note";
import { userPage } from "@/filters/user";
import { t } from "@/i18n";
import * as os from "@/os";
import { getNoteSummary } from "@/scripts/get-note-summary";
import { useTooltip } from "@/scripts/use-tooltip";
import { defaultStore } from "@/store/base";
import { stream } from "@/stream";
import type { Notification } from "@/api/types";

const props = withDefaults(
	defineProps<{
		notification: Notification;
		withTime?: boolean;
		full?: boolean;
	}>(),
	{
		withTime: false,
		full: false,
	},
);

const elRef = ref<HTMLElement | null>(null);

const showEmojiReactions =
	defaultStore.state.enableEmojiReactions ||
	defaultStore.state.showEmojisInReactionNotifications;

// `instance.defaultReaction` exists upstream but the Cinderfish meta payload
// does not carry it, so the Firefish fallback constant is used directly.
const defaultReaction = "⭐";

const reactionToShow = computed<string>(
	() => props.notification.reaction ?? defaultReaction,
);

const summary = computed<string>(() => {
	const note = props.notification.note;
	if (note == null) return "";
	if (props.notification.type === "renote") {
		return note.renote == null ? "" : getNoteSummary(note.renote);
	}
	return getNoteSummary(note);
});

const missingNoteText = computed<string>(() => {
	switch (props.notification.type) {
		case "reaction":
			return t("_notification.reacted");
		case "renote":
			return t("_notification.renoted");
		default:
			return "";
	}
});

const reactionRef = ref<InstanceType<typeof MkReactionIcon> | null>(null);

useTooltip(reactionRef, (showing) => {
	void os.popup(
		MkReactionTooltip,
		{
			showing,
			reaction: reactionToShow.value,
			emojis: props.notification.note?.emojis,
			targetElement: reactionRef.value?.$el,
		},
		{},
		"closed",
	);
});

let readObserver: IntersectionObserver | undefined;
// `main` is a per-connection singleton: disposing one wrapper would tear the
// subscription down for every other listener, so only this handler is removed.
let offReadAll: (() => void) | null = null;

onMounted(() => {
	if (props.notification.isRead) return;

	readObserver = new IntersectionObserver((entries, observer) => {
		if (!entries.some((entry) => entry.isIntersecting)) return;
		stream.send("readNotification", { id: props.notification.id });
		observer.disconnect();
	});
	if (elRef.value != null) readObserver.observe(elRef.value);

	const channel = stream.useChannel("main");
	offReadAll = channel.on("readAllNotifications", () => readObserver?.disconnect());
});

watch(
	() => props.notification.isRead,
	(isRead) => {
		if (isRead) readObserver?.disconnect();
	},
);

onUnmounted(() => {
	readObserver?.disconnect();
	offReadAll?.();
	offReadAll = null;
});
</script>

<style lang="scss" scoped>
.qglefbjs {
	position: relative;
	box-sizing: border-box;
	padding: 24px 32px;
	font-size: 0.9em;
	overflow-wrap: break-word;
	display: flex;
	contain: content;

	&.max-width_500px {
		padding-block: 16px;
		font-size: 0.9em;
	}
	&.max-width_450px {
		padding: 12px 16px;
	}

	> .head {
		position: sticky;
		top: 0;
		flex-shrink: 0;
		width: 42px;
		height: 42px;
		margin-right: 8px;

		> .icon {
			display: block;
			width: 100%;
			height: 100%;
			border-radius: 6px;
		}

		> .sub-icon {
			position: absolute;
			z-index: 1;
			bottom: -2px;
			right: -2px;
			width: 20px;
			height: 20px;
			box-sizing: border-box;
			border-radius: 100%;
			background: var(--panel);
			box-shadow: 0 0 0 3px var(--panel);
			font-size: 12px;
			text-align: center;

			&:empty {
				display: none;
			}

			> * {
				color: #fff;
				width: 100%;
				height: 100%;
			}

			&.renote {
				padding: 3px;
				background: #31748f;
				pointer-events: none;
			}

			&.quote {
				padding: 3px;
				background: #31748f;
				pointer-events: none;
			}

			&.reply {
				padding: 3px;
				background: #c4a7e7;
				pointer-events: none;
			}

			&.mention {
				padding: 3px;
				background: #908caa;
				pointer-events: none;
			}
		}
	}

	> .tail {
		flex: 1;
		min-width: 0;

		> header {
			display: flex;
			align-items: baseline;
			white-space: nowrap;

			> .name {
				text-overflow: ellipsis;
				white-space: nowrap;
				min-width: 0;
				overflow: hidden;
			}

			> .time {
				margin-left: auto;
				font-size: 0.9em;
			}
		}

		> .text {
			white-space: nowrap;
			display: -webkit-box;
			-webkit-line-clamp: 3;
			-webkit-box-orient: vertical;
			overflow: hidden;
			text-overflow: ellipsis;

			> span:first-child {
				opacity: 0.7;
				&::after {
					content: ": ";
				}
			}

			> i {
				vertical-align: super;
				font-size: 50%;
				opacity: 0.5;
			}

			> i:first-child {
				margin-right: 4px;
			}

			> i:last-child {
				margin-left: 4px;
			}
		}
	}
}
</style>
