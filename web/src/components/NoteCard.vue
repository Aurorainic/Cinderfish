<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { api } from "../api/client";
import type { Note } from "../api/types";
import { isImageType, renderMFM } from "../lib/mfm";
import { relativeTime, absoluteTime } from "../lib/time";
import { isLoggedIn, session } from "../stores/session";
import AppIcon from "./AppIcon.vue";
import QuotedNote from "./QuotedNote.vue";
import UserAvatar from "./UserAvatar.vue";

const props = withDefaults(defineProps<{ note: Note; nested?: boolean }>(), { nested: false });
const emit = defineEmits<{
  reply: [note: Note];
  deleted: [id: string];
  renoted: [note: Note];
  updated: [note: Note];
}>();

const router = useRouter();
const route = useRoute();

const reactions = ref<Record<string, number>>({});
const myReaction = ref<string | undefined>();
const reactionCount = ref(0);
const revealCw = ref(false);
const paletteOpen = ref(false);
const busy = ref(false);
const notice = ref("");

const quickReactions = ["👍", "❤️", "😂", "🎉", "👀", "🤔", "🔥", "✨"];

watch(
  () => props.note,
  (n) => {
    reactions.value = { ...n.reactions };
    myReaction.value = n.myReaction;
    reactionCount.value = n.reactionCount;
    revealCw.value = false;
  },
  { immediate: true },
);

const isMine = computed(() => session.me?.id === props.note.userId);
const bodyHtml = computed(() => renderMFM(props.note.text));
const sortedReactions = computed(() =>
  Object.entries(reactions.value)
    .filter(([, count]) => count > 0)
    .sort((a, b) => b[1] - a[1]),
);
const imageFiles = computed(() => props.note.files.filter((f) => isImageType(f.type)));
const otherFiles = computed(() => props.note.files.filter((f) => !isImageType(f.type)));

const visibilityLabel = computed(
  () =>
    ({ public: "公开", home: "首页", followers: "关注者", specified: "指定" })[props.note.visibility] ??
    props.note.visibility,
);

function requireSignin(): boolean {
  if (isLoggedIn.value) return true;
  router.push({ path: "/login", query: { redirect: route.fullPath } });
  return false;
}

async function toggleReaction(emoji: string) {
  if (!requireSignin() || busy.value) return;
  paletteOpen.value = false;
  busy.value = true;
  try {
    if (myReaction.value === emoji) {
      await api.unreact(props.note.id);
      reactions.value[emoji] = (reactions.value[emoji] ?? 1) - 1;
      if (reactions.value[emoji] <= 0) delete reactions.value[emoji];
      myReaction.value = undefined;
      reactionCount.value = Math.max(0, reactionCount.value - 1);
    } else {
      await api.react(props.note.id, emoji);
      if (myReaction.value) {
        const old = myReaction.value;
        reactions.value[old] = (reactions.value[old] ?? 1) - 1;
        if (reactions.value[old] <= 0) delete reactions.value[old];
        reactionCount.value = Math.max(0, reactionCount.value - 1);
      }
      reactions.value[emoji] = (reactions.value[emoji] ?? 0) + 1;
      myReaction.value = emoji;
      reactionCount.value += 1;
    }
    emit("updated", props.note);
  } catch (err) {
    notice.value = err instanceof Error ? err.message : "反应失败";
  } finally {
    busy.value = false;
  }
}

async function renote() {
  if (!requireSignin() || busy.value) return;
  busy.value = true;
  try {
    const created = await api.createNote({ renoteId: props.note.id });
    emit("renoted", created);
    notice.value = "已转贴";
    setTimeout(() => (notice.value = ""), 2000);
  } catch (err) {
    notice.value = err instanceof Error ? err.message : "转贴失败";
  } finally {
    busy.value = false;
  }
}

async function remove() {
  if (!isMine.value || busy.value) return;
  if (!window.confirm("删除这条笔记？")) return;
  busy.value = true;
  try {
    await api.deleteNote(props.note.id);
    emit("deleted", props.note.id);
  } catch (err) {
    notice.value = err instanceof Error ? err.message : "删除失败";
  } finally {
    busy.value = false;
  }
}
</script>

<template>
  <article class="note" :class="{ nested }">
    <header class="note-head">
      <RouterLink :to="`/@${note.user.username}`" class="avatar-link">
        <UserAvatar :user="note.user" :size="nested ? 30 : 42" />
      </RouterLink>
      <div class="who">
        <RouterLink :to="`/@${note.user.username}`" class="name">
          {{ note.user.name || note.user.username }}
        </RouterLink>
        <span class="handle">
          @{{ note.user.username }}<template v-if="note.user.host">@{{ note.user.host }}</template>
        </span>
      </div>
      <RouterLink :to="`/notes/${note.id}`" class="time" :title="absoluteTime(note.createdAt)">
        {{ relativeTime(note.createdAt) }}
      </RouterLink>
    </header>

    <p v-if="note.reply" class="reply-context muted small">
      回复 <RouterLink :to="`/@${note.reply.user.username}`">@{{ note.reply.user.username }}</RouterLink>
    </p>

    <div v-if="note.cw" class="cw">
      <button v-if="!revealCw" class="cw-toggle" type="button" @click="revealCw = true">
        <span class="cw-label">内容警告：{{ note.cw }}</span>
        <span class="muted small">点击显示</span>
      </button>
      <template v-else>
        <p class="cw-label">内容警告：{{ note.cw }}</p>
        <div class="body" v-html="bodyHtml"></div>
      </template>
    </div>
    <div v-else class="body" v-html="bodyHtml"></div>

    <div v-if="imageFiles.length" class="media" :class="`media-${Math.min(imageFiles.length, 4)}`">
      <a v-for="file in imageFiles" :key="file.id" :href="file.url" target="_blank" rel="noopener" class="media-item">
        <img :src="file.url" :alt="file.name" loading="lazy" />
      </a>
    </div>
    <ul v-if="otherFiles.length" class="files">
      <li v-for="file in otherFiles" :key="file.id">
        <a :href="file.url" target="_blank" rel="noopener">
          <AppIcon name="image" />
          <span>{{ file.name }}</span>
        </a>
      </li>
    </ul>

    <div v-if="note.hasPoll && note.poll" class="poll">
      <div v-for="(choice, i) in note.poll.choices" :key="i" class="poll-choice">
        <span>{{ choice.text }}</span>
        <span class="muted small">{{ choice.votes }}</span>
      </div>
      <p class="muted small">{{ note.poll.votesCount }} 票</p>
    </div>

    <div v-if="note.renote" class="quote">
      <QuotedNote :note="note.renote" />
    </div>

    <div v-if="sortedReactions.length" class="reactions">
      <button
        v-for="[emoji, count] in sortedReactions"
        :key="emoji"
        type="button"
        class="reaction"
        :class="{ mine: myReaction === emoji }"
        @click="toggleReaction(emoji)"
      >
        <span>{{ emoji }}</span>
        <span class="count">{{ count }}</span>
      </button>
    </div>

    <footer v-if="!nested" class="note-foot">
      <button type="button" class="action" title="回复" @click="emit('reply', note)">
        <AppIcon name="reply" />
        <span v-if="note.repliesCount">{{ note.repliesCount }}</span>
      </button>
      <button type="button" class="action" title="转贴" @click="renote">
        <AppIcon name="renote" />
        <span v-if="note.renoteCount">{{ note.renoteCount }}</span>
      </button>
      <div class="react-wrap">
        <button type="button" class="action" title="反应" @click="paletteOpen = !paletteOpen">
          <AppIcon name="heart" />
          <span v-if="reactionCount">{{ reactionCount }}</span>
        </button>
        <div v-if="paletteOpen" class="palette">
          <button
            v-for="emoji in quickReactions"
            :key="emoji"
            type="button"
            :class="{ mine: myReaction === emoji }"
            @click="toggleReaction(emoji)"
          >
            {{ emoji }}
          </button>
        </div>
      </div>
      <span class="spacer"></span>
      <span class="muted small">{{ visibilityLabel }}</span>
      <button v-if="isMine" type="button" class="action danger" title="删除" @click="remove">
        <AppIcon name="close" />
      </button>
    </footer>

    <p v-if="notice" class="notice small" style="margin: 10px 0 0">{{ notice }}</p>
  </article>
</template>

<style scoped>
.note {
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: 18px 20px;
  transition: border-color 0.15s ease;
}

.note:hover {
  border-color: color-mix(in srgb, var(--accent) 30%, var(--border));
}

.note.nested {
  padding: 12px 14px;
  border-radius: var(--radius);
  background: var(--bg-raised);
}

.note-head {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 10px;
}

.avatar-link {
  flex: none;
}

.who {
  display: flex;
  align-items: baseline;
  gap: 6px;
  min-width: 0;
  flex-wrap: wrap;
}

.name {
  font-weight: 700;
  color: var(--text-strong);
}

.name:hover {
  color: var(--accent);
}

.handle,
.time {
  color: var(--muted);
  font-size: 0.82rem;
}

.handle {
  overflow: hidden;
  text-overflow: ellipsis;
}

.time {
  margin-left: auto;
  flex: none;
}

.reply-context {
  margin-bottom: 6px;
}

.cw-toggle {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 2px;
  width: 100%;
  padding: 10px 12px;
  margin-bottom: 8px;
  border: 1px dashed var(--border);
  border-radius: var(--radius);
  background: var(--panel-2);
  cursor: pointer;
  text-align: left;
}

.cw-label {
  font-weight: 600;
  color: var(--text);
  margin-bottom: 6px;
}

.body {
  word-break: break-word;
  white-space: normal;
}

.body :deep(.mfm-link),
.body :deep(.mfm-mention),
.body :deep(.mfm-hashtag) {
  color: var(--link);
}

.body :deep(.mfm-code) {
  background: var(--bg);
  padding: 10px 12px;
  border-radius: var(--radius-sm);
  overflow-x: auto;
  font-size: 0.85rem;
}

.body :deep(.mfm-inline-code) {
  background: var(--bg);
  padding: 1px 5px;
  border-radius: 5px;
  font-size: 0.88em;
}

.media {
  display: grid;
  gap: 6px;
  margin-top: 12px;
  border-radius: var(--radius);
  overflow: hidden;
}

.media-1 {
  grid-template-columns: 1fr;
}

.media-2,
.media-3,
.media-4 {
  grid-template-columns: 1fr 1fr;
}

.media-item img {
  width: 100%;
  max-height: 360px;
  object-fit: cover;
  display: block;
  background: var(--bg);
}

.files {
  list-style: none;
  padding: 0;
  margin: 12px 0 0;
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.files a {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 10px;
  border: 1px solid var(--border);
  border-radius: 999px;
  font-size: 0.82rem;
  color: var(--muted);
}

.poll {
  margin-top: 12px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.poll-choice {
  display: flex;
  justify-content: space-between;
  padding: 8px 12px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--bg-raised);
}

.quote {
  margin-top: 12px;
}

.reactions {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 12px;
}

.reaction {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 3px 10px;
  border: 1px solid var(--border);
  border-radius: 999px;
  background: var(--panel-2);
  cursor: pointer;
  font-size: 0.9rem;
}

.reaction.mine,
.reaction:hover {
  border-color: var(--accent);
  background: var(--accent-soft);
}

.count {
  font-size: 0.78rem;
  color: var(--muted);
}

.note-foot {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 12px;
}

.action {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 5px 9px;
  border: 0;
  border-radius: 999px;
  background: transparent;
  color: var(--muted);
  cursor: pointer;
  font-size: 0.82rem;
}

.action:hover {
  color: var(--accent);
  background: var(--accent-soft);
}

.action.danger:hover {
  color: var(--danger);
  background: color-mix(in srgb, var(--danger) 12%, transparent);
}

.spacer {
  flex: 1;
}

.react-wrap {
  position: relative;
}

.palette {
  position: absolute;
  bottom: calc(100% + 6px);
  left: 0;
  z-index: 10;
  display: flex;
  gap: 2px;
  padding: 6px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--panel);
  box-shadow: var(--shadow);
}

.palette button {
  border: 0;
  background: transparent;
  font-size: 1.15rem;
  padding: 4px 6px;
  border-radius: 8px;
  cursor: pointer;
}

.palette button:hover,
.palette button.mine {
  background: var(--accent-soft);
}
</style>
