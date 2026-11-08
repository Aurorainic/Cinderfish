<!-- Ported from firefish@b425f61:packages/client/src/pages/messaging/messaging-room.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :display-back-button="true" />
		</template>
		<MkSpacer :content-max="800" :margin-min="16" :margin-max="32">
			<!-- Group messaging is a recorded backend deferral: the route still
			     exists, so it renders an explicit state instead of crashing. -->
			<CfInfo v-if="isGroupRoute" warn class="_formBlock">
				{{ t("messagingWithGroup") }}：本实例仅支持一对一私聊。
			</CfInfo>

			<div v-else ref="rootEl" class="cf-messaging-room">
				<CfLoading v-if="resolving" />
				<CfInfo v-else-if="resolveError != null" warn>
					{{ errorText(resolveError) }}
				</CfInfo>
				<template v-else-if="peer != null">
					<div class="messages" aria-live="polite">
						<div v-if="hasMore" class="more">
							<CfButton
								v-if="!moreFetching"
								primary
								@click="loadOlder"
							>
								{{ t("loadMore") }}
							</CfButton>
							<CfLoading v-else mini />
						</div>

						<CfLoading v-if="fetching && items.length === 0" />
						<CfInfo
							v-else-if="pageError != null && items.length === 0"
							warn
						>
							{{ errorText(pageError) }}
						</CfInfo>
						<div v-else-if="orderedMessages.length === 0" class="_fullinfo">
							<div>{{ t("noMessagesYet") }}</div>
						</div>
						<div
							v-for="message in orderedMessages"
							:key="message.id"
							class="message"
							:class="{ isMe: isMine(message) }"
						>
							<CfAvatar
								v-if="!isMine(message)"
								class="avatar"
								:user="message.user"
								:show-indicator="true"
								disable-link
							/>
							<div class="content">
								<div
									class="balloon"
									:class="{
										noText:
											message.text == null ||
											message.text === '',
									}"
								>
									<button
										v-if="isMine(message)"
										class="delete-button"
										:title="t('delete')"
										@click.stop="del(message)"
									>
										<i class="ph-x-circle ph-fill"></i>
									</button>
									<div
										v-if="
											message.text != null &&
											message.text !== ''
										"
										class="text"
									>
										{{ message.text }}
									</div>
								</div>
								<div v-if="message.file != null" class="file">
									<MkMediaList
										v-if="isPreviewable(message.file)"
										:in-dm="true"
										:media-list="[message.file]"
									/>
									<a
										v-else
										:href="resolveMediaUrl(message.file.url)"
										rel="noopener"
										target="_blank"
										:title="message.file.name"
									>
										<p>{{ message.file.name }}</p>
									</a>
								</div>
								<footer>
									<span
										v-if="isMine(message) && message.isRead"
										class="read"
										>{{ t("messageRead") }}</span
									>
									<CfTime :time="message.createdAt" />
								</footer>
							</div>
						</div>
					</div>

					<footer class="composer">
						<div v-if="typers.length > 0" class="typers" aria-live="polite">
							<span v-for="typer in typers" :key="typer.id" class="user">{{
								typer.username
							}}</span>
							<CfEllipsis />
						</div>
						<div v-if="showIndicator" class="new-message">
							<CfButton primary @click="onIndicatorClick">
								<i class="ph-arrow-down ph-bold ph-lg"></i>
								{{ t("newMessageExists") }}
							</CfButton>
						</div>
						<div v-if="file != null" class="attached" @click="file = null">
							<i class="ph-paperclip ph-bold ph-lg"></i>
							{{ file.name }}
						</div>
						<CfTextarea
							v-model="draft"
							:placeholder="t('inputMessageHere')"
							@keydown="onKeydown"
						/>
						<div class="buttons">
							<button
								class="_button"
								:aria-label="t('attachFile')"
								@click="chooseFile"
							>
								<i class="ph-upload ph-bold ph-lg"></i>
							</button>
							<button
								class="send _button"
								:disabled="!canSend || sending"
								:title="t('send')"
								:aria-label="t('send')"
								@click="send"
							>
								<CfLoading v-if="sending" inline mini />
								<i v-else class="ph-paper-plane-tilt ph-bold ph-lg"></i>
							</button>
						</div>
					</footer>
				</template>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script setup lang="ts">
// Firefish's messaging-room reduced to its 1:1 branch. The group branches
// (`users/groups/*`, groupId params, read receipts keyed by `reads`) are gone
// because the backend keeps group messaging as a recorded deferral.
//
// Read marking: the thread is listed with `markAsRead: false`, then every
// unread incoming message is read explicitly through
// `messaging/messages/read`. This keeps `isRead` per row and matches
// Firefish's "mark what is on screen" behaviour; re-listing with
// `markAsRead: true` would silently read the whole conversation on open,
// including messages the user has not scrolled to yet.
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { me } from "@/account";
import { api, resolveMediaUrl } from "@/api/client";
import { ENDPOINTS } from "@/api/endpoints";
import type { DriveFile, MessagingMessage, UserDetailed, UserLite } from "@/api/types";
import type { MessagingReadEvent, StreamChannel } from "@/api/stream";
import { host as instanceHost } from "@/config";
import { parseAcct } from "@/filters/user";
import { t } from "@/i18n";
import { toast } from "@/os";
import { isBottomVisible, onScrollBottom, scrollToBottom } from "@/scripts/scroll";
import { selectFile } from "@/scripts/select-file";
import { definePageMetadata } from "@/scripts/page-metadata";
import { defaultStore } from "@/store";
import { stream } from "@/stream";
import { usePagination } from "@/composables/use-pagination";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";
import MkMediaList from "@/components/MkMediaList.vue";
import CfAvatar from "@/components/ui/CfAvatar.vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfEllipsis from "@/components/ui/CfEllipsis.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import CfTextarea from "@/components/ui/CfTextarea.vue";
import CfTime from "@/components/ui/CfTime.vue";

defineOptions({ name: "CfPageMessagingRoom" });

const route = useRoute();

const rootEl = ref<HTMLElement | null>(null);
const peer = ref<UserDetailed | null>(null);
const resolving = ref(true);
const resolveError = ref<unknown>(null);

const draft = ref("");
const file = ref<DriveFile | null>(null);
const sending = ref(false);
const typers = ref<UserLite[]>([]);
const showIndicator = ref(false);

const userAcct = computed(() => String(route.params.userAcct ?? ""));
const groupId = computed(() => String(route.params.groupId ?? ""));
const isGroupRoute = computed(
	() => groupId.value !== "" || route.path.startsWith("/my/messaging/group/"),
);

const {
	items,
	fetching,
	moreFetching,
	hasMore,
	error: pageError,
	reload,
	loadMore,
	prepend,
	updateById,
	removeById,
} = usePagination<MessagingMessage>({
	endpoint: ENDPOINTS.messagingMessages,
	params: () => ({
		userId: peer.value?.id ?? "",
		markAsRead: false,
	}),
	limit: 20,
	immediate: false,
});

/** The endpoint answers newest first; chats read oldest first. */
const orderedMessages = computed(() => [...items.value].reverse());

const canSend = computed(
	() => draft.value.trim() !== "" || file.value != null,
);

const headerTitle = computed(() => {
	const target = peer.value;
	const name = target == null ? null : target.name ?? target.username;
	return name == null ? t("messaging") : `${t("messaging")} · ${name}`;
});

definePageMetadata(
	computed(() => ({
		title: headerTitle.value,
		icon: "ph-chats-teardrop ph-bold",
	})),
);

function errorText(cause: unknown): string {
	return cause instanceof Error ? cause.message : String(cause);
}

function isMine(message: MessagingMessage): boolean {
	return me.value != null && message.userId === me.value.id;
}

function isPreviewable(target: DriveFile): boolean {
	return target.type.startsWith("image/") || target.type.startsWith("video/");
}

async function resolvePeer(): Promise<UserDetailed> {
	const parsed = parseAcct(userAcct.value);
	if (parsed == null) {
		throw new Error("Invalid account handle.");
	}
	// A bare name is local; a host equal to this instance is a local lookup.
	const localHost = instanceHost.toLowerCase().replace(/\.$/, "");
	const host = parsed.host === localHost ? undefined : parsed.host ?? undefined;
	return await api(
		ENDPOINTS.usersShow,
		host === undefined
			? { username: parsed.username }
			: { username: parsed.username, host },
	);
}

async function markRead(message: MessagingMessage): Promise<void> {
	if (me.value == null || message.userId === me.value.id || message.isRead) {
		return;
	}
	try {
		await api(ENDPOINTS.messagingMessagesRead, { messageId: message.id });
		// The `read` stream event also flips the row; doing it here keeps the
		// UI correct when the event is delayed or the connection is down.
		updateById(message.id, (old) => ({ ...old, isRead: true }));
	} catch {
		// The next listing reconciles the flag; a failed receipt is not fatal.
	}
}

function markLoadedRead(loaded: MessagingMessage[]): void {
	if (me.value == null) return;
	for (const message of loaded) {
		if (message.userId !== me.value.id && !message.isRead) {
			void markRead(message);
		}
	}
}

/** Fetch the next older page, then read what it brought into view. */
async function loadOlder(): Promise<void> {
	await loadMore();
	markLoadedRead(items.value);
}

/** Returning to the tab reads everything the visible thread now shows. */
function onVisibilitychange(): void {
	if (document.hidden) return;
	markLoadedRead(items.value);
}

function scrollBottom(): void {
	if (rootEl.value != null) scrollToBottom(rootEl.value, { behavior: "smooth" });
}

function onIndicatorClick(): void {
	showIndicator.value = false;
	scrollBottom();
}

function onMessage(message: MessagingMessage): void {
	const mine = me.value?.id;
	const atBottom = rootEl.value != null && isBottomVisible(rootEl.value, 64);

	prepend(message);

	if (mine != null && message.userId !== mine) {
		if (document.visibilityState === "visible") {
			void markRead(message);
		} else {
			showIndicator.value = true;
		}
	}

	if (atBottom) {
		void nextTick(() => scrollBottom());
	} else if (mine == null || message.userId !== mine) {
		showIndicator.value = true;
		if (rootEl.value != null) {
			onScrollBottom(rootEl.value, () => {
				showIndicator.value = false;
			});
		}
	}
}

function onRead(event: MessagingReadEvent): void {
	const mine = me.value?.id;
	if (mine == null) return;
	if (event.readerId === mine) {
		// The peer's messages are now read for this viewer.
		for (const message of items.value) {
			if (message.userId !== mine && !message.isRead) {
				updateById(message.id, (old) => ({ ...old, isRead: true }));
			}
		}
	} else {
		// The peer read the viewer's messages.
		for (const message of items.value) {
			if (message.userId === mine && !message.isRead) {
				updateById(message.id, (old) => ({ ...old, isRead: true }));
			}
		}
	}
}

function onDeleted(payload: { id: string } | null): void {
	if (payload != null) removeById(payload.id);
}

let typersTimer: number | null = null;

function onTypers(users: UserLite[]): void {
	const mine = me.value?.id;
	typers.value = users.filter((user) => user.id !== mine);
	// The backend only emits a non-empty aggregate and has no "stopped typing"
	// event, so expire the indicator after the five-second window.
	if (typersTimer != null) window.clearTimeout(typersTimer);
	typersTimer = window.setTimeout(() => {
		typers.value = [];
		typersTimer = null;
	}, 6000);
}

// The server aggregates typing over a five-second window, so a 3s local
// throttle is enough to keep the frame rate reasonable.
let lastTypingAt = 0;
function announceTyping(): void {
	const target = peer.value;
	if (target == null) return;
	const now = Date.now();
	if (now - lastTypingAt < 3000) return;
	lastTypingAt = now;
	stream.typingOnMessaging(target.id);
}

function onKeydown(event: KeyboardEvent): void {
	if (event.isComposing) return;
	announceTyping();

	const sendOnEnter = defaultStore.state.enterSendsMessage;
	if (sendOnEnter) {
		if (event.key === "Enter" && (event.ctrlKey || event.metaKey)) {
			// Modifier + Enter keeps a newline when Enter sends.
			return;
		}
		if (
			event.key === "Enter" &&
			!event.shiftKey &&
			!("ontouchstart" in document.documentElement) &&
			canSend.value
		) {
			event.preventDefault();
			void send();
		}
	} else if (
		event.key === "Enter" &&
		(event.ctrlKey || event.metaKey) &&
		canSend.value
	) {
		event.preventDefault();
		void send();
	}
}

async function send(): Promise<void> {
	const target = peer.value;
	if (target == null || sending.value || !canSend.value) return;
	sending.value = true;
	const text = draft.value.trim() === "" ? undefined : draft.value;
	try {
		await api(ENDPOINTS.messagingMessagesCreate, {
			userId: target.id,
			...(text === undefined ? {} : { text }),
			...(file.value == null ? {} : { fileId: file.value.id }),
		});
		draft.value = "";
		file.value = null;
	} catch (cause) {
		toast(errorText(cause), "error");
	} finally {
		sending.value = false;
	}
}

async function chooseFile(event: MouseEvent): Promise<void> {
	const picked = await selectFile({ label: t("selectFile"), src: event });
	if (picked != null) file.value = picked;
}

async function del(message: MessagingMessage): Promise<void> {
	try {
		await api(ENDPOINTS.messagingMessagesDelete, { messageId: message.id });
		removeById(message.id);
	} catch (cause) {
		toast(errorText(cause), "error");
	}
}

let connection: StreamChannel | null = null;
const offHandlers: (() => void)[] = [];

function teardownChannel(): void {
	for (const off of offHandlers.splice(0)) off();
	connection?.dispose();
	connection = null;
	if (typersTimer != null) {
		window.clearTimeout(typersTimer);
		typersTimer = null;
	}
}

async function init(): Promise<void> {
	teardownChannel();
	typers.value = [];
	showIndicator.value = false;
	resolveError.value = null;
	peer.value = null;

	if (isGroupRoute.value) {
		resolving.value = false;
		return;
	}

	resolving.value = true;
	let loaded: MessagingMessage[] = [];
	try {
		const resolved = await resolvePeer();
		peer.value = resolved;

		// Subscribe before the first listing so a message arriving during the
		// fetch is not lost; `reload` reconciles ids it already contains.
		const channel = stream.useChannel("messaging", { otherparty: resolved.id });
		connection = channel;
		offHandlers.push(channel.on("message", onMessage));
		offHandlers.push(channel.on("read", onRead));
		offHandlers.push(channel.on("deleted", onDeleted));
		offHandlers.push(channel.on("typers", onTypers));

		loaded = await reload();
	} catch (cause) {
		resolveError.value = cause;
	} finally {
		resolving.value = false;
	}

	if (resolveError.value == null && peer.value != null) {
		await nextTick();
		scrollBottom();
		markLoadedRead(loaded);
	}
}

watch([userAcct, isGroupRoute], () => {
	void init();
}, { immediate: true });

onMounted(() => {
	document.addEventListener("visibilitychange", onVisibilitychange);
});

onUnmounted(() => {
	document.removeEventListener("visibilitychange", onVisibilitychange);
	teardownChannel();
});
</script>

<style lang="scss" scoped>
.cf-messaging-room {
	position: relative;

	.messages {
		padding-block: 8px;

		.more {
			display: flex;
			justify-content: center;
			padding-block: 8px;
		}
	}

	.message {
		display: flex;
		position: relative;
		margin-block-end: 16px;

		> .avatar {
			position: sticky;
			inset-block-start: calc(var(--stickyTop, 0px) + 20px);
			display: block;
			inline-size: 45px;
			block-size: 45px;
			margin-inline-end: 8px;
		}

		> .content {
			min-inline-size: 0;
			max-inline-size: 100%;

			> .balloon {
				position: relative;
				display: inline-flex;
				align-items: center;
				min-block-size: 38px;
				max-inline-size: 100%;
				border-radius: 16px;
				background: var(--X4);

				&.noText {
					background: transparent;
				}

				&:hover > .delete-button {
					display: block;
				}

				> .delete-button {
					display: none;
					position: absolute;
					z-index: 1;
					inset-block-start: -4px;
					inset-inline-end: -4px;
					margin: 0;
					padding: 0;
					border: none;
					background: transparent;
					cursor: pointer;
					color: var(--accentLighten);

					> i {
						inline-size: 16px;
						block-size: 16px;
					}
				}

				> .text {
					margin: 0;
					padding-block: 12px;
					padding-inline: 18px;
					overflow: hidden;
					overflow-wrap: break-word;
					word-break: break-word;
					font-size: 1em;
					color: var(--fg);
				}
			}

			> .file {
				margin-block-start: 4px;

				> a {
					display: block;
					max-inline-size: 100%;
					border-radius: 16px;
					overflow: hidden;
					text-decoration: none;

					> p {
						margin: 0;
						padding: 16px;
						text-align: center;
						color: var(--fg);
						background: var(--panelHighlight);
					}
				}
			}

			> footer {
				display: block;
				margin-block-start: 2px;
				font-size: 0.65em;
				color: var(--fgTransparentWeak);

				> .read {
					margin-inline: 8px;
				}
			}
		}

		&.isMe {
			flex-direction: row-reverse;

			> .content {
				text-align: end;

				> .balloon {
					background: var(--accent);
					text-align: start;

					&.noText {
						background: transparent;
					}

					> .text {
						color: var(--fgOnAccent);
					}
				}

				> footer {
					text-align: end;
				}
			}
		}
	}

	.composer {
		position: sticky;
		z-index: 2;
		inset-block-end: 0;
		padding-block-start: 8px;
		background: var(--bg);
		border-block-start: solid 0.5px var(--divider);

		> .typers {
			padding-block-end: 4px;
			font-size: 0.9em;
			color: var(--fgTransparentWeak);

			> .user + .user::before {
				content: ", ";
			}
		}

		> .new-message {
			padding-block-end: 8px;
			text-align: center;
		}

		> .attached {
			padding: 6px 8px;
			cursor: pointer;
			color: var(--fgTransparentWeak);
			overflow: hidden;
			text-overflow: ellipsis;
			white-space: nowrap;
		}

		> .buttons {
			display: flex;

			._button {
				margin: 0;
				padding: 12px 16px;
				font-size: 1em;
				transition: color 0.1s ease;

				&:hover {
					color: var(--accent);
				}
			}

			> .send {
				margin-inline-start: auto;
				color: var(--accent);

				&:disabled {
					opacity: 0.5;
				}
			}
		}
	}
}
</style>
