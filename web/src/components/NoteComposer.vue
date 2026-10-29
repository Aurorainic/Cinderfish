<script setup lang="ts">
import { computed, nextTick, ref, watch } from "vue";
import { api } from "../api/client";
import type { DriveFile, Note } from "../api/types";
import { relativeTime } from "../lib/time";
import { instance, loadInstance } from "../stores/instance";
import AppIcon from "./AppIcon.vue";
import UserAvatar from "./UserAvatar.vue";
import { session, isLoggedIn } from "../stores/session";

const props = defineProps<{ replyTo?: Note | null; renoteOf?: Note | null }>();
const emit = defineEmits<{ posted: [note: Note]; cancel: [] }>();

const text = ref("");
const cw = ref("");
const showCw = ref(false);
const visibility = ref<"public" | "home" | "followers">("public");
const localOnly = ref(false);
const files = ref<DriveFile[]>([]);
const uploading = ref(false);
const posting = ref(false);
const error = ref("");
const textarea = ref<HTMLTextAreaElement | null>(null);

const maxLength = computed(() => instance.meta?.maxNoteTextLength ?? 3000);
const remaining = computed(() => maxLength.value - text.value.length);
const canPost = computed(
  () =>
    !posting.value &&
    !uploading.value &&
    remaining.value >= 0 &&
    (text.value.trim().length > 0 || files.value.length > 0 || !!props.renoteOf),
);

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
);

const visibilityLabel = computed(
  () => ({ public: "公开", home: "首页", followers: "关注者" })[visibility.value],
);

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

async function submit() {
  if (!canPost.value) return;
  posting.value = true;
  error.value = "";
  try {
    const created = await api.createNote({
      text: text.value.trim() || undefined,
      cw: showCw.value && cw.value ? cw.value : null,
      visibility: visibility.value,
      localOnly: localOnly.value,
      fileIds: files.value.map((f) => f.id),
      replyId: props.replyTo?.id,
      renoteId: props.renoteOf?.id,
    });
    emit("posted", created);
    text.value = "";
    cw.value = "";
    showCw.value = false;
    files.value = [];
    emit("cancel");
  } catch (err) {
    error.value = err instanceof Error ? err.message : "发布失败";
  } finally {
    posting.value = false;
  }
}

loadInstance();
</script>

<template>
  <section v-if="isLoggedIn" class="composer">
    <div class="composer-head">
      <UserAvatar v-if="session.me" :user="session.me" :size="40" />
      <div class="context">
        <span v-if="replyTo" class="muted small">
          回复 <RouterLink :to="`/notes/${replyTo.id}`">@{{ replyTo.user.username }}</RouterLink>
          · {{ relativeTime(replyTo.createdAt) }}
        </span>
        <span v-if="renoteOf" class="muted small">转贴 @{{ renoteOf.user.username }} 的笔记</span>
        <span class="composer-title">{{ replyTo ? "回复" : renoteOf ? "转贴" : "说点什么" }}</span>
      </div>
      <button v-if="replyTo || renoteOf" class="icon-button" type="button" title="取消" @click="emit('cancel')">
        <AppIcon name="close" />
      </button>
    </div>

    <div v-if="showCw" class="field cw-field">
      <input v-model="cw" class="input" type="text" placeholder="内容警告（可选）" />
    </div>

    <textarea
      ref="textarea"
      v-model="text"
      class="textarea composer-text"
      :placeholder="replyTo ? '写下回复…' : '现在在想什么？'"
      rows="4"
      @keydown.ctrl.enter="submit"
      @keydown.meta.enter="submit"
    ></textarea>

    <div v-if="files.length" class="previews">
      <div v-for="file in files" :key="file.id" class="preview">
        <img v-if="file.type.startsWith('image/')" :src="file.url" :alt="file.name" />
        <span v-else class="muted small">{{ file.name }}</span>
        <button type="button" class="remove" title="移除" @click="removeFile(file.id)">
          <AppIcon name="close" />
        </button>
      </div>
    </div>

    <p v-if="error" class="notice error small">{{ error }}</p>

    <div class="composer-foot">
      <label class="tool-button" title="添加附件">
        <AppIcon name="image" />
        <input type="file" multiple hidden :disabled="uploading" @change="pickFiles" />
      </label>
      <button type="button" class="tool-button" :class="{ active: showCw }" title="内容警告" @click="showCw = !showCw">
        <span>CW</span>
      </button>

      <select v-model="visibility" class="select" :title="visibilityLabel">
        <option value="public">公开</option>
        <option value="home">首页</option>
        <option value="followers">关注者</option>
      </select>

      <label class="toggle small">
        <input v-model="localOnly" type="checkbox" />
        仅本地
      </label>

      <span class="spacer"></span>
      <span class="counter" :class="{ over: remaining < 0 }">{{ remaining }}</span>
      <button class="button small" type="button" :disabled="!canPost" @click="submit">
        {{ posting ? "发布中…" : uploading ? "上传中…" : "发布" }}
      </button>
    </div>
  </section>
</template>

<style scoped>
.composer {
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: 16px 18px;
  margin-bottom: 16px;
}

.composer-head {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 10px;
}

.context {
  display: flex;
  flex-direction: column;
  line-height: 1.3;
}

.composer-title {
  font-weight: 700;
  color: var(--text-strong);
}

.cw-field {
  margin-bottom: 8px;
}

.composer-text {
  min-height: 88px;
  border: 0;
  background: transparent;
  padding: 4px 0;
}

.composer-text:focus {
  box-shadow: none;
}

.previews {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin: 10px 0;
}

.preview {
  position: relative;
  width: 96px;
  height: 96px;
  border-radius: var(--radius);
  overflow: hidden;
  background: var(--bg);
  display: grid;
  place-items: center;
  border: 1px solid var(--border);
}

.preview img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.remove {
  position: absolute;
  top: 4px;
  right: 4px;
  display: grid;
  place-items: center;
  width: 22px;
  height: 22px;
  border: 0;
  border-radius: 50%;
  background: rgba(0, 0, 0, 0.6);
  color: #fff;
  cursor: pointer;
}

.composer-foot {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  border-top: 1px solid var(--border);
  padding-top: 12px;
  margin-top: 4px;
}

.tool-button {
  display: grid;
  place-items: center;
  width: 36px;
  height: 36px;
  border-radius: 50%;
  border: 1px solid var(--border);
  background: transparent;
  color: var(--muted);
  cursor: pointer;
  font-size: 0.72rem;
  font-weight: 700;
}

.tool-button:hover,
.tool-button.active {
  color: var(--accent);
  border-color: var(--accent);
  background: var(--accent-soft);
}

.select {
  padding: 7px 10px;
  border: 1px solid var(--border);
  border-radius: 999px;
  background: var(--bg-raised);
  color: var(--text);
  font-size: 0.82rem;
}

.toggle {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  color: var(--muted);
}

.spacer {
  flex: 1;
}

.counter {
  font-size: 0.82rem;
  color: var(--muted);
  font-variant-numeric: tabular-nums;
}

.counter.over {
  color: var(--danger);
  font-weight: 700;
}
</style>
