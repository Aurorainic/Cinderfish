<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { api, isContentWarningValid, normalizeContentWarning } from "../../api/client";
import type { DriveFile, Note, NoteVisibility } from "../../api/types";
import { relativeTime } from "../../lib/time";
import { instance, loadInstance } from "../../stores/instance";
import { isLoggedIn, session } from "../../stores/session";
import AppIcon from "../AppIcon.vue";
import UserAvatar from "../UserAvatar.vue";
import MkButton from "../ui/MkButton.vue";
import MkInfo from "../ui/MkInfo.vue";
import MkInput from "../ui/MkInput.vue";

// The global post form. `os.openPostForm` renders it inside a PageWindow when
// the post entry lives in the shell (bottom bar, deck), and the timeline pins
// the same component above its feed with `fixed`.
const props = withDefaults(
  defineProps<{ replyTo?: Note | null; renoteTo?: Note | null; fixed?: boolean }>(),
  { replyTo: null, renoteTo: null, fixed: false },
);
const emit = defineEmits<{ posted: [note: Note]; cancel: [] }>();

const DRAFT_KEY = "cinderfish.draft";
const DRAFT_DEBOUNCE_MS = 800;

type Draft = {
  text: string;
  cw: string;
  visibility: NoteVisibility;
  localOnly: boolean;
  fileIds: string[];
  // Preview snapshot so restored attachments still render; only fileIds are
  // authoritative for the request.
  files: DriveFile[];
};

type Recipient = { raw: string; username: string; host: string | null };

const text = ref("");
const cw = ref("");
const showCw = ref(false);
const visibility = ref<NoteVisibility>("public");
const localOnly = ref(false);
const files = ref<DriveFile[]>([]);
const uploading = ref(false);
const posting = ref(false);
const error = ref("");
const textarea = ref<HTMLTextAreaElement | null>(null);

const recipientsInput = ref("");
const recipientErrors = ref<Record<string, string>>({});

const draftPrompt = ref<Draft | null>(null);
let draftTimer: ReturnType<typeof setTimeout> | undefined;

const maxLength = computed(() => instance.meta?.maxNoteTextLength ?? 3000);
const remaining = computed(() => maxLength.value - text.value.length);
const contentWarningRemaining = computed(() => 100 - cw.value.length);

// Only the standalone composer owns the draft: reply and quote targets would
// otherwise be lost when the dialog closes.
const standalone = computed(() => !props.replyTo && !props.renoteTo);

const recipientList = computed(() => parseHandles(recipientsInput.value));

const canPost = computed(() => {
  if (posting.value || uploading.value || remaining.value < 0) return false;
  if (showCw.value && !isContentWarningValid(cw.value)) return false;
  if (visibility.value === "specified" && recipientList.value.length === 0) return false;
  return text.value.trim().length > 0 || files.value.length > 0 || !!props.renoteTo;
});

// Misskey's composer prompt reads as a question ("今どうしてる？"), Chinese
// equivalent here; the header title names the action instead of repeating it.
const title = computed(() => (props.replyTo ? "回复" : props.renoteTo ? "引用" : "发帖"));

const quoteExcerpt = computed(() => {
  const target = props.renoteTo;
  if (!target) return "";
  const body = (target.text ?? "").replace(/\s+/g, " ").trim();
  if (!body) return target.files.length ? "（仅附件）" : "（无正文）";
  return body.length > 120 ? `${body.slice(0, 120)}…` : body;
});

// Hand-written icons: the segmented control needs a lock the shared AppIcon
// set does not carry, so all four live here for a consistent stroke weight.
const visibilityOptions: { value: NoteVisibility; label: string; path: string }[] = [
  { value: "public", label: "公开", path: "M12 21a9 9 0 1 0 0-18 9 9 0 0 0 0 18ZM3 12h18M12 3c2.5 2.5 3.8 5.5 3.8 9s-1.3 6.5-3.8 9c-2.5-2.5-3.8-5.5-3.8-9S9.5 5.5 12 3Z" },
  { value: "home", label: "首页", path: "M3 10.5 12 3l9 7.5M5 9.5V21h5v-6h4v6h5V9.5" },
  { value: "followers", label: "关注者", path: "M12 12a4 4 0 1 0 0-8 4 4 0 0 0 0 8Zm-8 9a8 8 0 0 1 16 0" },
  { value: "specified", label: "指定", path: "M7 10V7a5 5 0 0 1 10 0v3M5 10h14v11H5z" },
];

function parseHandles(input: string): Recipient[] {
  const seen = new Set<string>();
  const out: Recipient[] = [];
  for (const token of input.split(/[,，、\s]+/)) {
    const raw = token.trim().replace(/^@+/, "");
    if (!raw) continue;
    const [username, host] = raw.split("@");
    if (!username) continue;
    const key = raw.toLowerCase();
    if (seen.has(key)) continue;
    seen.add(key);
    out.push({ raw, username, host: host || null });
  }
  return out;
}

function displayHandle(recipient: Recipient): string {
  return recipient.host ? `@${recipient.username}@${recipient.host}` : `@${recipient.username}`;
}

// The dialog mounts with replyTo already set, so the mention prefix has to be
// seeded on the immediate run rather than on a later change.
watch(
  () => props.replyTo,
  async (reply) => {
    if (!reply) return;
    showCw.value = true;
    await nextTick();
    textarea.value?.focus();
    const handle = reply.user.host ? `@${reply.user.username}@${reply.user.host}` : `@${reply.user.username}`;
    if (!text.value.includes(handle)) text.value = `${handle} ${text.value}`;
  },
  { immediate: true },
);

function draftSnapshot(): Draft {
  return {
    text: text.value,
    cw: cw.value,
    visibility: visibility.value,
    localOnly: localOnly.value,
    fileIds: files.value.map((f) => f.id),
    files: files.value,
  };
}

function isEmptyDraft(draft: Draft): boolean {
  return !draft.text.trim() && !draft.cw.trim() && draft.fileIds.length === 0;
}

function readDraft(): Draft | null {
  try {
    const raw = localStorage.getItem(DRAFT_KEY);
    if (!raw) return null;
    const parsed = JSON.parse(raw) as Partial<Draft>;
    const draft: Draft = {
      text: parsed.text ?? "",
      cw: parsed.cw ?? "",
      visibility: (parsed.visibility as NoteVisibility) ?? "public",
      localOnly: parsed.localOnly === true,
      fileIds: Array.isArray(parsed.fileIds) ? parsed.fileIds : [],
      files: Array.isArray(parsed.files) ? (parsed.files as DriveFile[]) : [],
    };
    return isEmptyDraft(draft) ? null : draft;
  } catch {
    return null;
  }
}

function saveDraft() {
  if (!standalone.value) return;
  const draft = draftSnapshot();
  try {
    if (isEmptyDraft(draft)) localStorage.removeItem(DRAFT_KEY);
    else localStorage.setItem(DRAFT_KEY, JSON.stringify(draft));
  } catch {
    // Storage may be full or disabled; losing a draft is not worth an error.
  }
}

function clearDraft() {
  clearTimeout(draftTimer);
  draftTimer = undefined;
  try {
    localStorage.removeItem(DRAFT_KEY);
  } catch {
    // ignore
  }
  draftPrompt.value = null;
}

function scheduleDraftSave() {
  if (!standalone.value) return;
  clearTimeout(draftTimer);
  draftTimer = setTimeout(saveDraft, DRAFT_DEBOUNCE_MS);
}

watch([text, cw, visibility, localOnly, files], scheduleDraftSave, { deep: true });

function restoreDraft() {
  const draft = draftPrompt.value;
  if (!draft) return;
  text.value = draft.text;
  cw.value = draft.cw;
  showCw.value = !!draft.cw;
  visibility.value = draft.visibility;
  localOnly.value = draft.localOnly;
  // A draft written by an older version may carry ids without a preview
  // snapshot; those attachments cannot be shown, so they are dropped.
  files.value = draft.files;
  draftPrompt.value = null;
  nextTick(() => textarea.value?.focus());
}

function discardDraft() {
  clearDraft();
}

async function pickFiles(event: Event) {
  const input = event.target as HTMLInputElement;
  if (!input.files?.length) return;
  uploading.value = true;
  error.value = "";
  try {
    for (const file of Array.from(input.files)) {
      files.value.push(await api.uploadFile(file));
    }
  } catch (err) {
    error.value = err instanceof Error ? err.message : "上传失败";
  } finally {
    uploading.value = false;
    input.value = "";
  }
}

function removeFile(id: string) {
  files.value = files.value.filter((f) => f.id !== id);
}

async function resolveRecipients(list: Recipient[]): Promise<{ userIds: string[]; ok: boolean }> {
  const errors: Record<string, string> = {};
  const userIds: string[] = [];
  await Promise.all(
    list.map(async (recipient) => {
      const handle = displayHandle(recipient);
      try {
        const user = await api.usersShow({
          username: recipient.username,
          host: recipient.host ?? undefined,
        });
        if (!userIds.includes(user.id)) userIds.push(user.id);
        delete errors[handle];
      } catch (err) {
        errors[handle] = err instanceof Error ? err.message : "无法解析该用户";
      }
    }),
  );
  recipientErrors.value = errors;
  return { userIds, ok: Object.keys(errors).length === 0 && userIds.length > 0 };
}

async function submit() {
  if (!canPost.value) return;
  posting.value = true;
  error.value = "";
  try {
    let visibleUserIds: string[] | undefined;
    if (visibility.value === "specified") {
      const resolved = await resolveRecipients(recipientList.value);
      if (!resolved.ok) {
        error.value = "部分收件人无法解析，请检查后重试。";
        return;
      }
      visibleUserIds = resolved.userIds;
    }

    const created = await api.createNote({
      text: text.value.trim() || undefined,
      cw: normalizeContentWarning(showCw.value, cw.value),
      visibility: visibility.value,
      localOnly: localOnly.value,
      fileIds: files.value.map((f) => f.id),
      replyId: props.replyTo?.id,
      renoteId: props.renoteTo?.id,
      visibleUserIds,
    });
    text.value = "";
    cw.value = "";
    showCw.value = false;
    files.value = [];
    recipientsInput.value = "";
    recipientErrors.value = {};
    clearDraft();
    emit("posted", created);
  } catch (err) {
    error.value = err instanceof Error ? err.message : "发布失败";
  } finally {
    posting.value = false;
  }
}

function cancel() {
  clearTimeout(draftTimer);
  if (standalone.value) saveDraft();
  emit("cancel");
}

onMounted(() => {
  loadInstance();
  const draft = readDraft();
  if (standalone.value && draft) draftPrompt.value = draft;
});

onBeforeUnmount(() => {
  clearTimeout(draftTimer);
  if (standalone.value) saveDraft();
});
</script>

<template>
  <section v-if="isLoggedIn" class="composer" :class="{ 'composer-fixed': fixed }">
    <div class="composer-head">
      <UserAvatar v-if="session.me" :user="session.me" :size="fixed ? 34 : 40" />
      <div class="context">
        <span v-if="replyTo" class="_caption">
          回复 <RouterLink :to="`/notes/${replyTo.id}`">@{{ replyTo.user.username }}</RouterLink>
          · {{ relativeTime(replyTo.createdAt) }}
        </span>
        <span v-if="renoteTo" class="context-quote">引用 @{{ renoteTo.user.username }} 的笔记</span>
        <span class="composer-title">{{ title }}</span>
      </div>
      <button
        v-if="!fixed && (replyTo || renoteTo)"
        class="icon-button"
        type="button"
        title="取消"
        aria-label="取消"
        @click="cancel"
      >
        <AppIcon name="close" />
      </button>
    </div>

    <div v-if="draftPrompt" class="draft-banner">
      <span>有未发送的草稿</span>
      <button type="button" class="draft-action" @click="restoreDraft">恢复</button>
      <button type="button" class="draft-action quiet" @click="discardDraft">丢弃</button>
    </div>

    <div v-if="showCw" class="field cw-field">
      <input
        v-model="cw"
        class="cw"
        type="text"
        maxlength="100"
        placeholder="内容警告（可选）"
        :aria-invalid="contentWarningRemaining < 0"
      />
      <span class="_caption" aria-live="polite">剩余 {{ contentWarningRemaining }} 个字符</span>
    </div>

    <textarea
      ref="textarea"
      v-model="text"
      class="composer-text"
      :placeholder="replyTo ? '写下回复…' : renoteTo ? '引用并评论…' : '写点什么吧？'"
      :rows="fixed ? 3 : 4"
      @keydown.ctrl.enter="submit"
      @keydown.meta.enter="submit"
    ></textarea>

    <div v-if="renoteTo" class="quote-preview">
      <UserAvatar :user="renoteTo.user" :size="26" />
      <div class="quote-body">
        <span class="quote-name">{{ renoteTo.user.name || renoteTo.user.username }}</span>
        <span class="quote-text">{{ quoteExcerpt }}</span>
      </div>
    </div>

    <div v-if="files.length" class="previews">
      <div v-for="file in files" :key="file.id" class="preview">
        <img v-if="file.type.startsWith('image/')" :src="file.url" :alt="file.name" />
        <span v-else class="_caption preview-name">{{ file.name }}</span>
        <button type="button" class="remove" title="移除" aria-label="移除" @click="removeFile(file.id)">
          <AppIcon name="close" />
        </button>
      </div>
    </div>

    <div v-if="visibility === 'specified'" class="recipients">
      <MkInput
        v-model="recipientsInput"
        type="text"
        placeholder="收件人，逗号分隔：@alice, @bob@remote.example"
      >
        <template #label>收件人</template>
      </MkInput>
      <p v-for="recipient in recipientList" :key="recipient.raw" class="recipient-row">
        <span class="recipient-handle" :class="{ bad: recipientErrors[displayHandle(recipient)] }">
          {{ displayHandle(recipient) }}
        </span>
        <span v-if="recipientErrors[displayHandle(recipient)]" class="recipient-error">
          {{ recipientErrors[displayHandle(recipient)] }}
        </span>
        <span v-else class="_caption">将仅对这些用户可见</span>
      </p>
    </div>

    <MkInfo v-if="error" class="form-error" warn>{{ error }}</MkInfo>

    <div class="composer-foot">
      <label class="tool-button" title="添加附件">
        <AppIcon name="image" />
        <input type="file" multiple hidden :disabled="uploading" @change="pickFiles" />
      </label>
      <button
        type="button"
        class="tool-button"
        :class="{ active: showCw }"
        title="内容警告"
        aria-label="内容警告"
        :aria-pressed="showCw"
        @click="showCw = !showCw"
      >
        <AppIcon name="eyeSlash" />
      </button>

      <div class="vis-segmented" role="radiogroup" aria-label="可见性">
        <button
          v-for="option in visibilityOptions"
          :key="option.value"
          type="button"
          class="vis-option"
          :class="{ active: visibility === option.value }"
          role="radio"
          :aria-checked="visibility === option.value"
          :title="`可见性：${option.label}`"
          @click="visibility = option.value"
        >
          <svg
            class="vis-icon"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="1.8"
            stroke-linecap="round"
            stroke-linejoin="round"
            aria-hidden="true"
          >
            <path :d="option.path" />
          </svg>
          <span class="vis-label">{{ option.label }}</span>
        </button>
      </div>

      <label class="toggle">
        <input v-model="localOnly" type="checkbox" />
        仅本地
      </label>

      <span class="spacer"></span>
      <span class="counter" :class="{ over: remaining < 0 }">{{ remaining }}</span>
      <MkButton gradate :disabled="!canPost" @click="submit">
        {{ posting ? "发布中…" : uploading ? "上传中…" : "发布" }}
      </MkButton>
    </div>
  </section>
</template>

<style scoped>
.composer {
  container-type: inline-size;
  padding-block: 16px;
  padding-inline: 18px;
  margin-block-end: 16px;
  background: var(--panel);
  border: solid 0.5px var(--divider);
  border-radius: var(--radius);
}

/* Pinned above a timeline: the panel merges into the feed's column. */
.composer-fixed {
  margin-block-end: 0;
  border: 0;
  border-block-end: solid 0.5px var(--divider);
  border-radius: 0;
  padding-block: 12px 10px;
  padding-inline: 14px;
}

.composer-head {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-block-end: 10px;
}

.context {
  display: flex;
  flex-direction: column;
  line-height: 1.3;
  min-width: 0;
  flex: 1;
}

.context-quote {
  font-size: 0.85em;
  color: var(--renote);
}

.composer-title {
  font-weight: 700;
  color: var(--fgHighlighted);
}

.composer-fixed .composer-title {
  display: none;
}

.composer-fixed .composer-head {
  margin-block-end: 4px;
}

.icon-button {
  display: grid;
  place-items: center;
  flex: none;
  inline-size: 32px;
  block-size: 32px;
  padding: 0;
  border: 0;
  border-radius: 50%;
  background: none;
  color: var(--fgTransparentWeak);
  cursor: pointer;
}

.icon-button:hover {
  background: var(--buttonBg);
  color: var(--fg);
}

.icon-button:focus-visible {
  outline: solid 2px var(--accent);
  outline-offset: 1px;
}

.icon-button .icon {
  inline-size: 18px;
  block-size: 18px;
}

.draft-banner {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-block-end: 8px;
  padding-block: 6px;
  padding-inline: 10px;
  font-size: 0.85em;
  color: var(--fg);
  background: var(--accentedBg);
  border-radius: var(--radius);
}

.draft-action {
  padding-block: 2px;
  padding-inline: 8px;
  font: inherit;
  font-size: 0.9em;
  color: var(--accent);
  background: none;
  border: solid 1px transparent;
  border-radius: var(--radius);
  cursor: pointer;
}

.draft-action.quiet {
  color: var(--fgTransparentWeak);
}

.draft-action:hover {
  border-color: var(--accent);
}

.draft-action:focus-visible {
  outline: solid 2px var(--accent);
  outline-offset: 1px;
}

.cw-field {
  margin-block-end: 8px;
}

/* Firefish renders the content warning and the body as bare fields: no border,
   no background, the composer panel is the surface. */
.cw,
.composer-text {
  display: block;
  box-sizing: border-box;
  inline-size: 100%;
  margin: 0;
  padding-block: 0;
  padding-inline: 0;
  border: none;
  border-radius: 0;
  background: transparent;
  color: var(--fg);
  font: inherit;
}

.cw {
  margin-block-end: 6px;
  font-size: 1em;
}

.cw:focus,
.composer-text:focus {
  outline: none;
}

.composer-text {
  min-height: 72px;
  font-size: 1.05em;
  resize: none;
}

.quote-preview {
  display: flex;
  gap: 8px;
  align-items: flex-start;
  margin-block: 8px 4px;
  padding-block: 8px;
  padding-inline: 10px;
  border-inline-start: 2px solid var(--renote);
  border-radius: var(--radius);
  background: var(--panelHighlight);
}

.quote-body {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.quote-name {
  font-size: 0.85em;
  font-weight: 700;
  color: var(--fgHighlighted);
}

.quote-text {
  font-size: 0.9em;
  color: var(--fgTransparentWeak);
  overflow: hidden;
  text-overflow: ellipsis;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}

.previews {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-block: 10px;
}

.preview {
  position: relative;
  inline-size: 88px;
  block-size: 88px;
  border-radius: var(--radius);
  overflow: hidden;
  background: var(--bg);
  display: grid;
  place-items: center;
  border: solid 0.5px var(--divider);
}

.preview img {
  inline-size: 100%;
  block-size: 100%;
  object-fit: cover;
}

.preview-name {
  padding: 4px;
  text-align: center;
  word-break: break-all;
}

.remove {
  position: absolute;
  inset-block-start: 4px;
  inset-inline-end: 4px;
  display: grid;
  place-items: center;
  inline-size: 22px;
  block-size: 22px;
  padding: 0;
  border: 0;
  border-radius: 50%;
  background: rgba(0, 0, 0, 0.6);
  color: #fff;
  cursor: pointer;
}

.remove:focus-visible {
  outline: solid 2px var(--accent);
  outline-offset: 1px;
}

.recipients {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-block: 4px 8px;
}

.recipient-row {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: 6px;
  margin: 0;
  font-size: 0.85em;
}

.recipient-handle {
  color: var(--mention);
  font-weight: 600;
}

.recipient-handle.bad {
  color: var(--error);
}

.recipient-error {
  color: var(--error);
}

.form-error {
  margin-block: 0 8px;
}

.composer-foot {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  border-block-start: solid 0.5px var(--divider);
  padding-block-start: 10px;
  margin-block-start: 4px;
}

.tool-button {
  display: grid;
  place-items: center;
  inline-size: 34px;
  block-size: 34px;
  border-radius: 50%;
  border: solid 0.5px var(--divider);
  background: none;
  color: var(--fgTransparentWeak);
  cursor: pointer;
  font-size: 0.78em;
  font-weight: 700;
}

.tool-button:hover,
.tool-button.active {
  color: var(--accent);
  border-color: var(--accent);
  background: var(--accentedBg);
}

.tool-button:focus-within,
.tool-button:focus-visible {
  outline: solid 2px var(--accent);
  outline-offset: 1px;
}

.vis-segmented {
  display: inline-flex;
  align-items: center;
  padding: 2px;
  gap: 2px;
  border: solid 0.5px var(--divider);
  border-radius: var(--radius);
  background: var(--buttonBg);
}

.vis-option {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding-block: 4px;
  padding-inline: 9px;
  border: 0;
  border-radius: 6px;
  background: none;
  color: var(--fgTransparentWeak);
  font: inherit;
  font-size: 0.82em;
  cursor: pointer;
}

.vis-option:hover {
  color: var(--fg);
}

.vis-option.active {
  color: var(--accent);
  background: var(--accentedBg);
}

.vis-option:focus-visible {
  outline: solid 2px var(--accent);
  outline-offset: 1px;
}

.vis-icon {
  inline-size: 14px;
  block-size: 14px;
  display: block;
}

.toggle {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  color: var(--fgTransparentWeak);
}

.spacer {
  flex: 1;
}

.counter {
  font-size: 0.85em;
  color: var(--fgTransparentWeak);
  font-variant-numeric: tabular-nums;
}

.counter.over {
  color: var(--error);
  font-weight: 700;
}

/* Narrow columns (~350px in deck) drop the segmented labels to icons only and
   let the footer wrap. */
@container (max-width: 420px) {
  .vis-label {
    display: none;
  }

  .vis-option {
    padding: 5px 7px;
  }

  .preview {
    width: 72px;
    height: 72px;
  }
}
</style>
