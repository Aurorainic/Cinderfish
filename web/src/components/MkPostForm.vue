<!-- Ported from firefish@b425f61:packages/client/src/components/MkPostForm.vue (AGPL-3.0-only) -->
<template>
	<section
		v-size="{ max: [310, 500] }"
		class="gafaadew"
		:class="{ draghover }"
		:aria-label="t('note')"
		@dragover.stop="onDragover"
		@dragenter="onDragenter"
		@dragleave="onDragleave"
		@drop.stop="onDrop"
	>
		<header>
			<button
				v-if="!fixed"
				class="cancel _button"
				:aria-label="t('close')"
				@click="cancel"
			>
				<i class="ph-x ph-bold ph-lg"></i>
			</button>
			<div v-if="selfUser != null" class="account">
				<MkAvatar :user="selfUser" class="avatar" />
			</div>
			<div class="right">
				<span class="text-count" :class="{ over: textLength > maxTextLength }">{{
					maxTextLength - textLength
				}}</span>
				<span v-if="localOnly" class="local-only">
					<i class="ph-hand-fist ph-bold ph-lg"></i>
				</span>
				<button
					ref="visibilityButton"
					v-tooltip="t('visibility')"
					class="_button visibility"
					@click="setVisibility"
				>
					<i :class="visibilityIcon"></i>
				</button>
				<button
					v-tooltip="t('previewNoteText')"
					class="_button preview"
					:class="{ active: showPreview }"
					@click="showPreview = !showPreview"
				>
					<i class="ph-binoculars ph-bold ph-lg"></i>
				</button>
				<button
					class="submit _buttonGradate"
					data-cy-open-post-form-submit
					:disabled="!canPost"
					@click="post"
				>
					{{ submitText
					}}<i :class="submitIcon"></i>
				</button>
			</div>
		</header>
		<div class="form" :class="{ fixed }">
			<MkNoteSimple v-if="reply" class="preview" :note="reply" />
			<MkNoteSimple v-if="renote" class="preview" :note="renote" />
			<div v-if="quoteId" class="with-quote">
				<i class="ph-quotes ph-bold ph-lg"></i>
				{{ t("quoteAttached")
				}}<button
					class="_button"
					:aria-label="t('removeQuote')"
					@click="quoteId = null"
				>
					<i class="ph-x ph-bold ph-lg"></i>
				</button>
			</div>
			<div v-if="visibility === 'specified'" class="to-specified">
				<span class="recipient-label">{{ t("recipient") }}</span>
				<div class="visibleUsers">
					<span v-for="u in visibleUsers" :key="u.id" class="chip">
						<MkAcct :user="u" />
						<button
							class="_button"
							:aria-label="t('removeRecipient')"
							@click="removeVisibleUser(u)"
						>
							<i class="ph-x ph-bold ph-lg"></i>
						</button>
					</span>
					<button
						class="_button add-user"
						:aria-label="t('add')"
						@click="addVisibleUser"
					>
						<i class="ph-plus ph-bold ph-lg"></i>
					</button>
				</div>
			</div>
			<CfInfo v-if="hasNotSpecifiedMentions" warn class="hasNotSpecifiedMentions">
				{{ t("notSpecifiedMentionWarning") }} -
				<button class="_textButton" @click="addMissingMention">
					{{ t("add") }}
				</button>
			</CfInfo>
			<input
				v-show="useCw"
				ref="cwInputEl"
				v-model="cw"
				class="cw"
				:placeholder="t('annotation')"
				@keydown="onKeydown"
			/>
			<textarea
				ref="textareaEl"
				v-model="text"
				class="text"
				:class="{ withCw: useCw }"
				:disabled="posting"
				:placeholder="placeholder"
				@keydown="onKeydown"
				@paste="onPaste"
				@compositionupdate="onCompositionUpdate"
				@compositionend="onCompositionEnd"
			></textarea>
			<input
				v-show="withHashtags"
				ref="hashtagsInputEl"
				v-model="hashtags"
				class="hashtags"
				:placeholder="t('hashtags')"
				list="hashtags"
			/>
			<MkPostFormAttaches
				class="attaches"
				:files="files"
				@detach="detachFile"
			/>
			<div v-if="poll" class="poll">
				<MkPollEditor v-model="poll" />
				<button
					class="_button remove-poll"
					:aria-label="t('remove')"
					@click="poll = null"
				>
					<i class="ph-x ph-bold ph-lg"></i>
				</button>
			</div>
			<MkNotePreview v-if="showPreview" class="preview" :text="text" />
			<footer>
				<button
					v-tooltip="t('attachFile')"
					class="_button"
					:disabled="files.length >= 16"
					@click="chooseFileFrom"
				>
					<i class="ph-upload ph-bold ph-lg"></i>
				</button>
				<button
					v-tooltip="t('poll')"
					class="_button"
					:class="{ active: poll }"
					@click="togglePoll"
				>
					<i class="ph-microphone-stage ph-bold ph-lg"></i>
				</button>
				<button
					v-tooltip="t('useCw')"
					class="_button"
					:class="{ active: useCw }"
					@click="useCw = !useCw"
				>
					<i class="ph-eye-slash ph-bold ph-lg"></i>
				</button>
				<button v-tooltip="t('mention')" class="_button" @click="insertMention">
					<i class="ph-at ph-bold ph-lg"></i>
				</button>
				<button
					v-tooltip="t('hashtags')"
					class="_button"
					:class="{ active: withHashtags }"
					@click="withHashtags = !withHashtags"
				>
					<i class="ph-hash ph-bold ph-lg"></i>
				</button>
				<button v-tooltip="t('emoji')" class="_button" @click="insertEmoji">
					<i class="ph-smiley ph-bold ph-lg"></i>
				</button>
				<button
					v-tooltip="t('_mfm.cheatSheet')"
					class="_button right"
					@click="openCheatSheet"
				>
					<i class="ph-question ph-bold ph-lg"></i>
				</button>
			</footer>
			<datalist id="hashtags">
				<option v-for="tag in recentHashtags" :key="tag" :value="tag" />
			</datalist>
		</div>
	</section>
</template>

<script lang="ts" setup>
// Firefish parity notes for this port:
//   * every mutation goes through `api()`; the session comes from `@/account`;
//   * `inject("modal")` is gone — `fixed` alone decides whether the form renders
//     its own close button (contract §3.1);
//   * multi-account posting (`getAccounts`), channel posting, plugins
//     (`postFormActions` stays empty), the user picker (`os.selectUser`) and the
//     "attach from URL" entry do not exist here and are not faked.
import * as mfm from "mfm-js";
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue";
import type {
	DriveFile,
	NoteDetailed,
	NoteVisibility,
	UserLite,
} from "@/api/types";
import type { NotesCreateParams, PollParam } from "@/api/endpoints";
import { api, ApiError } from "@/api/client";
import { fetchMeta, me } from "@/account";
import { defaultStore, notePostInterruptors } from "@/store/base";
import { t } from "@/i18n";
import { alert, inputText, yesno } from "@/os/dialogs";
import { popup } from "@/os/state";
import { openEmojiPicker } from "@/os/open-emoji-picker";
import { selectFiles } from "@/scripts/select-file";
import { uploadFile } from "@/scripts/upload";
import { preprocess } from "@/scripts/preprocess";
import { Autocomplete } from "@/scripts/autocomplete";
import { extractMentions } from "@/scripts/extract-mentions";
import { unique } from "@/scripts/array";
import { formatTimeString } from "@/scripts/format-time-string";
import CfInfo from "@/components/ui/CfInfo.vue";
import MkNoteSimple from "@/components/MkNoteSimple.vue";
import MkNotePreview from "@/components/MkNotePreview.vue";
import MkPostFormAttaches from "@/components/MkPostFormAttaches.vue";
import MkPollEditor from "@/components/MkPollEditor.vue";
import MkVisibilityPicker from "@/components/MkVisibilityPicker.vue";
import MkCheatSheetDialog from "@/components/MkCheatSheetDialog.vue";

/** Same shape MkPollEditor models (it does not export the type). */
interface PollEditorValue {
	choices: string[];
	multiple: boolean;
	expiresAt?: number;
	expiredAfter?: number;
}

const props = withDefaults(
	defineProps<{
		reply?: NoteDetailed | null;
		renote?: NoteDetailed | null;
		initialText?: string;
		initialVisibility?: NoteVisibility;
		initialFiles?: DriveFile[];
		initialLocalOnly?: boolean;
		initialNote?: NoteDetailed | null;
		fixed?: boolean;
		autofocus?: boolean;
	}>(),
	{
		reply: null,
		renote: null,
		initialText: "",
		initialVisibility: undefined,
		initialFiles: () => [],
		initialLocalOnly: undefined,
		initialNote: null,
		fixed: false,
		autofocus: true,
	},
);

const emit = defineEmits<{
	posted: [];
	cancel: [];
}>();

/** Firefish's `$i`; the composer only needs the avatar and the self-mention check. */
const selfUser = computed<UserLite | null>(() => me.value);

const textareaEl = ref<HTMLTextAreaElement | null>(null);
const cwInputEl = ref<HTMLInputElement | null>(null);
const hashtagsInputEl = ref<HTMLInputElement | null>(null);
const visibilityButton = ref<HTMLElement | null>(null);

const posting = ref(false);
const text = ref(props.initialText);
const files = ref<DriveFile[]>([...props.initialFiles]);
const poll = ref<PollEditorValue | null>(null);
const useCw = ref(false);
const showPreview = ref(false);
const cw = ref<string>("");
const quoteId = ref<string | null>(null);
const hasNotSpecifiedMentions = ref(false);
const imeText = ref("");
const visibleUsers = ref<UserLite[]>([]);
const recentHashtags = ref<string[]>(readHashtagHistory());
const maxTextLength = ref(3000);
const draghover = ref(false);

const defaultVisibility = defaultStore.state.rememberNoteVisibility
	? defaultStore.state.visibility
	: defaultStore.state.defaultNoteVisibility;
const defaultLocalOnly = defaultStore.state.rememberNoteVisibility
	? defaultStore.state.localOnly
	: defaultStore.state.defaultNoteLocalOnly;

const visibility = ref<NoteVisibility>(
	props.initialVisibility ?? defaultVisibility,
);
const localOnly = ref<boolean>(props.initialLocalOnly ?? defaultLocalOnly);

const withHashtags = ref(defaultStore.state.postFormWithHashtags);
const hashtags = ref(defaultStore.state.postFormHashtags);

watch(withHashtags, (value) => {
	defaultStore.set("postFormWithHashtags", value);
});
watch(hashtags, (value) => {
	defaultStore.set("postFormHashtags", value);
});

const draftKey = computed<string>(() => {
	if (props.renote != null) return `renote:${props.renote.id}`;
	if (props.reply != null) return `reply:${props.reply.id}`;
	return "note";
});

const placeholder = computed<string>(() => {
	if (props.renote != null) return t("_postForm.quotePlaceholder");
	if (props.reply != null) return t("_postForm.replyPlaceholder");
	const xs = [
		t("_postForm._placeholders.a"),
		t("_postForm._placeholders.b"),
		t("_postForm._placeholders.c"),
		t("_postForm._placeholders.d"),
		t("_postForm._placeholders.e"),
		t("_postForm._placeholders.f"),
	];
	return xs[Math.floor(Math.random() * xs.length)];
});

const submitText = computed<string>(() =>
	props.renote != null ? t("quote") : props.reply != null ? t("reply") : t("note"),
);

const submitIcon = computed<string>(() =>
	props.reply != null
		? "ph-arrow-u-up-left ph-bold ph-lg"
		: props.renote != null
			? "ph-quotes ph-bold ph-lg"
			: "ph-paper-plane-tilt ph-bold ph-lg",
);

const visibilityIcon = computed<string>(() => {
	switch (visibility.value) {
		case "home":
			return "ph-house ph-bold ph-lg";
		case "followers":
			return "ph-lock ph-bold ph-lg";
		case "specified":
			return "ph-envelope-simple-open ph-bold ph-lg";
		default:
			return "ph-planet ph-bold ph-lg";
	}
});

// The `stringz` dependency is gone, and the server counts UTF-8 bytes rather
// than code points (`len(text) > 3000` -> NOTE_TOO_LONG, notes-core.md), so this
// counter measures the same thing the server enforces.
const textLength = computed<number>(() =>
	new TextEncoder().encode(
		(preprocess(text.value) + imeText.value).trim(),
	).length,
);

const postPoll = computed<PollParam | null>(() => {
	if (poll.value == null) return null;
	const choices = poll.value.choices
		.map((choice) => choice.trim())
		.filter((choice) => choice !== "");
	if (choices.length < 2) return null;
	const param: PollParam = { choices, multiple: poll.value.multiple };
	if (poll.value.expiresAt != null) param.expiresAt = poll.value.expiresAt;
	else if (poll.value.expiredAfter != null)
		param.expiredAfter = poll.value.expiredAfter;
	return param;
});

const canPost = computed<boolean>(() => {
	if (posting.value) return false;
	if (
		textLength.value < 1 &&
		files.value.length < 1 &&
		poll.value == null &&
		props.renote == null
	)
		return false;
	if (textLength.value > maxTextLength.value) return false;
	if (poll.value != null && postPoll.value == null) return false;
	// `visibleUserIds` must be non-empty for specified visibility, otherwise the
	// server answers INVALID_VISIBILITY (notes-core.md).
	if (visibility.value === "specified" && visibleUsers.value.length === 0)
		return false;
	return true;
});

function readHashtagHistory(): string[] {
	try {
		const raw = localStorage.getItem("hashtags");
		const parsed: unknown = raw == null ? [] : JSON.parse(raw);
		return Array.isArray(parsed)
			? parsed.filter((tag): tag is string => typeof tag === "string")
			: [];
	} catch {
		return [];
	}
}

function accountHandle(user: UserLite): string {
	return `@${user.username}${user.host != null ? `@${user.host}` : ""}`;
}

function pushVisibleUser(user: UserLite): void {
	if (
		!visibleUsers.value.some(
			(u) => u.username === user.username && u.host === user.host,
		)
	) {
		visibleUsers.value.push(user);
	}
}

function removeVisibleUser(user: UserLite): void {
	visibleUsers.value = visibleUsers.value.filter((u) => u.id !== user.id);
	checkMissingMention();
}

function initFromReply(note: NoteDetailed): void {
	const self = me.value;

	if (note.user.username !== self?.username || note.user.host != null) {
		text.value += `${accountHandle(note.user)} `;
	}

	if (note.text != null) {
		const mentions = extractMentions(mfm.parse(note.text));
		for (const mention of mentions) {
			if (self?.username === mention.username && mention.host == null) continue;
			const handle = `@${mention.username}${mention.host != null ? `@${mention.host}` : ""}`;
			if (text.value.includes(`${handle} `)) continue;
			text.value += `${handle} `;
		}
	}

	if (note.visibility !== "public") {
		visibility.value = note.visibility;
		if (note.visibility === "specified") void loadSpecifiedUsers(note);
	}
}

async function loadSpecifiedUsers(note: NoteDetailed): Promise<void> {
	const self = me.value;
	for (const userId of note.visibleUserIds ?? []) {
		if (userId === self?.id || userId === note.userId) continue;
		try {
			pushVisibleUser(await api("users/show", { userId }));
		} catch {
			// A deleted or remote user cannot be resolved; the composer stays usable.
		}
	}
	if (note.userId !== self?.id) {
		try {
			pushVisibleUser(await api("users/show", { userId: note.userId }));
		} catch {
			// ignored, see above
		}
	}
}

function checkMissingMention(): void {
	if (visibility.value !== "specified") {
		hasNotSpecifiedMentions.value = false;
		return;
	}
	const mentions = extractMentions(mfm.parse(text.value));
	hasNotSpecifiedMentions.value = mentions.some(
		(mention) =>
			!visibleUsers.value.some(
				(u) => u.username === mention.username && u.host === mention.host,
			),
	);
}

async function addMissingMention(): Promise<void> {
	const mentions = extractMentions(mfm.parse(text.value));
	for (const mention of mentions) {
		if (
			visibleUsers.value.some(
				(u) => u.username === mention.username && u.host === mention.host,
			)
		)
			continue;
		if (mention.host != null) continue; // users/show only resolves local users
		try {
			pushVisibleUser(await api("users/show", { username: mention.username }));
		} catch {
			// unknown user; nothing to add
		}
	}
	checkMissingMention();
}

function togglePoll(): void {
	poll.value =
		poll.value == null
			? { choices: ["", ""], multiple: false }
			: null;
}

function focus(): void {
	const el = textareaEl.value;
	if (el == null) return;
	el.focus();
	el.setSelectionRange(el.value.length, el.value.length);
}

/** `insert-text-at-cursor` replacement: setRangeText + an input event. */
function insertAtCursor(
	el: HTMLTextAreaElement | HTMLInputElement | null,
	value: string,
): void {
	if (el == null) return;
	const start = el.selectionStart ?? el.value.length;
	const end = el.selectionEnd ?? start;
	el.setRangeText(value, start, end, "end");
	el.dispatchEvent(new Event("input", { bubbles: true }));
	el.focus();
}

function insertMention(): void {
	void inputText({
		title: t("mention"),
		placeholder: "username@host",
	}).then(({ canceled, result }) => {
		if (canceled || result == null || result.trim() === "") return;
		insertAtCursor(textareaEl.value, `@${result.trim()} `);
	});
}

function insertEmoji(ev: MouseEvent): void {
	const target = (ev.currentTarget ?? ev.target) as HTMLElement | null;
	if (target == null) return;
	openEmojiPicker(target, {}, textareaEl.value);
}

function openCheatSheet(): void {
	popup(MkCheatSheetDialog, {}, {}, "closed");
}

function chooseFileFrom(ev: MouseEvent): void {
	void selectFiles({
		multiple: true,
		label: t("attachFile"),
		src: ev,
	}).then((picked) => {
		for (const file of picked) {
			if (files.value.length >= 16) break;
			files.value.push(file);
		}
	});
}

function detachFile(id: string): void {
	files.value = files.value.filter((file) => file.id !== id);
}

function upload(file: File, name?: string): void {
	void uploadFile(file, {
		folderId: defaultStore.state.uploadFolder,
		name,
	}).then((created) => {
		files.value.push(created);
	});
}

function addVisibleUser(): void {
	void inputText({
		title: t("recipient"),
		placeholder: "username",
	}).then(async ({ canceled, result }) => {
		if (canceled || result == null || result.trim() === "") return;
		try {
			pushVisibleUser(await api("users/show", { username: result.trim() }));
			checkMissingMention();
		} catch (cause) {
			void alert({
				type: "error",
				text: cause instanceof ApiError ? cause.message : String(cause),
			});
		}
	});
}

function setVisibility(): void {
	popup(
		MkVisibilityPicker,
		{
			currentVisibility: visibility.value,
			currentLocalOnly: localOnly.value,
			src: visibilityButton.value,
		},
		{
			changeVisibility: (next: NoteVisibility) => {
				visibility.value = next;
				checkMissingMention();
				if (defaultStore.state.rememberNoteVisibility)
					defaultStore.set("visibility", next);
			},
			changeLocalOnly: (next: boolean) => {
				localOnly.value = next;
				if (defaultStore.state.rememberNoteVisibility)
					defaultStore.set("localOnly", next);
			},
		},
		"closed",
	);
}

function clear(): void {
	text.value = "";
	files.value = [];
	poll.value = null;
	quoteId.value = null;
	useCw.value = false;
	cw.value = "";
}

function saveDraft(): void {
	try {
		const drafts = JSON.parse(
			localStorage.getItem("drafts") ?? "{}",
		) as Record<string, unknown>;
		drafts[draftKey.value] = {
			updatedAt: new Date().toISOString(),
			data: {
				text: text.value,
				useCw: useCw.value,
				cw: cw.value,
				visibility: visibility.value,
				localOnly: localOnly.value,
				files: files.value,
				poll: poll.value,
			},
		};
		localStorage.setItem("drafts", JSON.stringify(drafts));
	} catch {
		// Quota or private mode: posting still works without drafts.
	}
}

function deleteDraft(): void {
	try {
		const drafts = JSON.parse(
			localStorage.getItem("drafts") ?? "{}",
		) as Record<string, unknown>;
		delete drafts[draftKey.value];
		localStorage.setItem("drafts", JSON.stringify(drafts));
	} catch {
		// ignored
	}
}

interface StoredDraft {
	data?: {
		text?: string;
		useCw?: boolean;
		cw?: string;
		visibility?: NoteVisibility;
		localOnly?: boolean;
		files?: DriveFile[];
		poll?: PollEditorValue | null;
	};
}

function restoreDraft(): void {
	try {
		const drafts = JSON.parse(
			localStorage.getItem("drafts") ?? "{}",
		) as Record<string, StoredDraft>;
		const draft = drafts[draftKey.value];
		if (draft?.data == null) return;
		text.value = draft.data.text ?? "";
		useCw.value = draft.data.useCw === true;
		cw.value = draft.data.cw ?? "";
		if (draft.data.visibility != null) visibility.value = draft.data.visibility;
		if (draft.data.localOnly != null) localOnly.value = draft.data.localOnly;
		files.value = (draft.data.files ?? []).filter(
			(file): file is DriveFile => file != null,
		);
		if (draft.data.poll != null) poll.value = draft.data.poll;
	} catch {
		// corrupt draft: start empty
	}
}

function watchForDraft(): void {
	watch([text, useCw, cw, poll, files, visibility, localOnly], () => {
		saveDraft();
	});
}

function onKeydown(ev: KeyboardEvent): void {
	if (
		ev.key === "Enter" &&
		(ev.ctrlKey || ev.metaKey) &&
		canPost.value
	) {
		ev.preventDefault();
		void post();
	}
	if (ev.key === "Escape") emit("cancel");
}

function onCompositionUpdate(ev: CompositionEvent): void {
	imeText.value = ev.data;
}

function onCompositionEnd(): void {
	imeText.value = "";
}

function onPaste(ev: ClipboardEvent): void {
	if (ev.clipboardData == null) return;

	const items = Array.from(ev.clipboardData.items ?? []);
	items.forEach((item, index) => {
		if (item.kind !== "file") return;
		const file = item.getAsFile();
		if (file == null) return;
		const extensionIndex = file.name.lastIndexOf(".");
		const extension = extensionIndex >= 0 ? file.name.slice(extensionIndex) : "";
		const formatted = `${formatTimeString(
			new Date(file.lastModified),
			defaultStore.state.pastedFileName,
		).replace(/{{number}}/g, String(index + 1))}${extension}`;
		upload(file, formatted);
	});

	const pasted = ev.clipboardData.getData("text");
	const origin = typeof location === "undefined" ? "" : location.origin;
	if (props.renote == null && quoteId.value == null && origin !== "" && pasted.startsWith(`${origin}/notes/`)) {
		ev.preventDefault();
		void yesno({ type: "info", text: t("quoteQuestion") }).then(
			({ canceled }) => {
				if (canceled) {
					insertAtCursor(textareaEl.value, pasted);
					return;
				}
				const match = pasted
					.slice(origin.length)
					.match(/^\/notes\/(.+?)\/?$/);
				if (match != null) quoteId.value = match[1];
			},
		);
	}
}

function onDragover(ev: DragEvent): void {
	if (ev.dataTransfer == null) return;
	const item = ev.dataTransfer.items[0];
	if (item == null || item.kind !== "file") return;
	ev.preventDefault();
	draghover.value = true;
	ev.dataTransfer.dropEffect = "copy";
}

function onDragenter(): void {
	draghover.value = true;
}

function onDragleave(): void {
	draghover.value = false;
}

function onDrop(ev: DragEvent): void {
	draghover.value = false;
	if (ev.dataTransfer == null) return;
	if (ev.dataTransfer.files.length === 0) return;
	ev.preventDefault();
	for (const file of Array.from(ev.dataTransfer.files)) upload(file);
}

async function applyInterruptors(params: NotesCreateParams): Promise<NotesCreateParams> {
	let result = params;
	for (const interruptor of notePostInterruptors) {
		result = await interruptor.handler(result);
	}
	return result;
}

async function post(): Promise<void> {
	if (!canPost.value) return;

	const processedText = preprocess(text.value).trim();

	let bodyText = processedText;
	if (withHashtags.value && hashtags.value.trim() !== "") {
		const list = hashtags.value
			.trim()
			.split(" ")
			.filter((tag) => tag !== "")
			.map((tag) => (tag.startsWith("#") ? tag : `#${tag}`))
			.join(" ");
		bodyText = bodyText === "" ? list : `${bodyText} ${list}`;
	}

	let params: NotesCreateParams = {
		text: bodyText === "" ? undefined : bodyText,
		fileIds: files.value.length > 0 ? files.value.map((file) => file.id) : undefined,
		replyId: props.reply?.id,
		renoteId: props.renote?.id ?? quoteId.value ?? undefined,
		poll: postPoll.value ?? undefined,
		cw: useCw.value ? (cw.value || "") : undefined,
		localOnly: localOnly.value,
		visibility: visibility.value,
		visibleUserIds:
			visibility.value === "specified"
				? visibleUsers.value.map((user) => user.id)
				: undefined,
	};

	params = await applyInterruptors(params);

	posting.value = true;
	try {
		await api("notes/create", params);
		clear();
		await nextTick();
		deleteDraft();
		if (bodyText !== "") {
			const tags = (
				mfm.extract(
					mfm.parse(bodyText),
					(node) => node.type === "hashtag",
				) as mfm.MfmHashtag[]
			).map((node) => node.props.hashtag);
			const history = unique([...tags, ...readHashtagHistory()]);
			localStorage.setItem("hashtags", JSON.stringify(history));
			recentHashtags.value = history;
		}
		emit("posted");
	} catch (cause) {
		const text_ =
			cause instanceof ApiError
				? `${cause.message}\n${cause.errorId ?? cause.code ?? ""}`.trim()
				: String(cause);
		void alert({ type: "error", text: text_ });
	} finally {
		posting.value = false;
	}
}

function cancel(): void {
	emit("cancel");
}

let autocompletes: Autocomplete[] = [];

onMounted(() => {
	void fetchMeta()
		.then((meta) => {
			if (meta.maxNoteTextLength > 0) maxTextLength.value = meta.maxNoteTextLength;
		})
		.catch(() => {
			// Keep the default limit; the server still enforces its own.
		});

	if (props.autofocus) {
		focus();
		void nextTick(() => focus());
	}

	if (textareaEl.value != null)
		autocompletes.push(new Autocomplete(textareaEl.value, text));
	if (cwInputEl.value != null)
		autocompletes.push(new Autocomplete(cwInputEl.value, cw));
	if (hashtagsInputEl.value != null)
		autocompletes.push(new Autocomplete(hashtagsInputEl.value, hashtags));

	if (props.reply != null) initFromReply(props.reply);
	if (props.initialNote != null) {
		// Editing has no endpoint in Cinderfish (contract §2): the note text is
		// pre-filled so the user can repost it, nothing is updated in place.
		text.value = props.initialNote.text ?? text.value;
	}
	if (defaultStore.state.keepCw && props.reply?.cw != null) {
		useCw.value = true;
		cw.value = props.reply.cw;
	}

	void nextTick(() => {
		restoreDraft();
		watchForDraft();
		checkMissingMention();
	});
});

onBeforeUnmount(() => {
	for (const autocomplete of autocompletes) autocomplete.detach();
	autocompletes = [];
});

watch(text, () => {
	checkMissingMention();
});
</script>

<style lang="scss" scoped>
.gafaadew {
	position: relative;

	&.draghover {
		outline: solid 2px var(--accent);
		outline-offset: -2px;
	}

	&.max-width_500px {
		> header {
			block-size: 50px;

			> .cancel {
				inline-size: 50px;
				line-height: 50px;
			}

			> .right > .submit {
				margin: 8px;
			}
		}

		> .form {
			> .to-specified {
				padding: 6px 16px;
			}

			> .cw,
			> .hashtags,
			> .text {
				padding: 0 16px;
			}

			> .text {
				min-block-size: 80px;
			}

			> footer {
				padding: 0 8px 8px;
			}
		}
	}

	&.max-width_310px > .form > footer > button {
		font-size: 14px;
		inline-size: 44px;
		block-size: 44px;
	}

	> header {
		display: flex;
		flex-wrap: wrap;
		z-index: 1000;
		block-size: 66px;
		align-items: center;

		> .cancel {
			padding: 0;
			font-size: 20px;
			inline-size: 64px;
			line-height: 66px;
			color: var(--fg);
		}

		> .account {
			block-size: 100%;
			aspect-ratio: 1 / 1;
			display: inline-flex;
			vertical-align: bottom;

			> .avatar {
				inline-size: 28px;
				block-size: 28px;
				margin: auto;
			}
		}

		> .right {
			margin-inline-start: auto;
			display: flex;
			align-items: center;
			padding-inline-end: 16px;

			> .text-count {
				opacity: 0.7;

				&.over {
					color: var(--error);
				}
			}

			> .visibility {
				block-size: 34px;
				inline-size: 34px;
				margin-inline-start: 8px;
				color: var(--fg);
			}

			> .local-only {
				margin-inline-start: 12px;
				opacity: 0.7;
			}

			> .preview {
				display: inline-block;
				padding: 0;
				margin-inline: 8px 0;
				font-size: 16px;
				inline-size: 34px;
				block-size: 34px;
				border-radius: 6px;
				color: var(--fg);

				&:hover {
					background: var(--X5);
				}

				&.active {
					color: var(--accent);
				}
			}

			> .submit {
				display: inline-flex;
				align-items: center;
				margin: 16px;
				padding: 0 12px;
				line-height: 34px;
				font-weight: bold;
				border-radius: var(--radius);
				font-size: 0.9em;

				&:disabled {
					opacity: 0.7;
				}

				> i {
					margin-inline-start: 6px;
				}
			}
		}
	}

	> .form {
		> .preview {
			padding: 16px;
		}

		> .with-quote {
			display: flex;
			align-items: center;
			gap: 0.4em;
			margin-inline: 24px;
			margin-block-end: 12px;
			color: var(--accent);

			> button {
				display: flex;
				padding: 0;
				color: var(--accent);
				opacity: 0.7;
			}
		}

		> .to-specified {
			padding: 6px 24px;
			margin-block-end: 8px;
			overflow: auto;
			line-height: 2rem;
			font-size: 0.9em;

			> .recipient-label {
				margin-inline-end: 8px;
			}

			> .visibleUsers {
				display: inline;
				font-size: 14px;

				> .chip {
					display: inline-flex;
					align-items: center;
					margin: 0.3rem;
					padding: 2px 4px 2px 8px;
					border-radius: 999px;
					background: var(--X3);
				}

				> .add-user {
					padding: 2px;
					border-radius: 8px;
					color: var(--fg);
				}
			}
		}

		> .hasNotSpecifiedMentions {
			margin: 0 20px 16px;
		}

		> .cw,
		> .hashtags,
		> .text {
			display: block;
			box-sizing: border-box;
			padding: 0 24px;
			margin: 0;
			inline-size: 100%;
			font-size: 1.05em;
			border: none;
			border-radius: 0;
			background: transparent;
			color: var(--fg);
			font-family: inherit;

			&:focus {
				outline: none;
			}

			&:disabled {
				opacity: 0.5;
			}
		}

		> .cw {
			z-index: 1;
			padding-block-end: 8px;
			border-block-end: solid 0.5px var(--divider);
		}

		> .hashtags {
			z-index: 1;
			padding-block: 8px;
			border-block-start: solid 0.5px var(--divider);
		}

		> .text {
			max-inline-size: 100%;
			min-inline-size: 100%;
			min-block-size: 90px;
			resize: vertical;

			&.withCw {
				padding-block-start: 8px;
			}
		}

		> .poll {
			position: relative;

			> .remove-poll {
				position: absolute;
				inset-block-start: 8px;
				inset-inline-end: 16px;
				padding: 4px;
				color: var(--fg);
			}
		}

		> footer {
			padding: 0 16px 16px;
			display: flex;
			flex-wrap: wrap;

			> button {
				display: inline-block;
				padding: 0;
				margin: 0;
				font-size: 16px;
				inline-size: 48px;
				block-size: 48px;
				border-radius: 6px;
				color: var(--fg);

				&:hover {
					background: var(--X5);
				}

				&.active {
					color: var(--accent);
				}

				&.right {
					margin-inline-start: auto;
				}

				&:disabled {
					opacity: 0.5;
				}
			}
		}
	}
}
</style>
