<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { api, ApiError } from "../api/client";
import type { Note, UserDetailed } from "../api/types";
import AppIcon from "../components/AppIcon.vue";
import NoteItem from "../components/note/Note.vue";
import ProfileHeader from "../components/ProfileHeader.vue";
import { isLoggedIn, session } from "../stores/session";

type Tab = "notes" | "media" | "replies";

const route = useRoute();
const router = useRouter();

const pageSize = 20;
// A tab whose filter drops most of a page (replies) asks for more than one
// page per click so the list does not look empty while older items exist.
const maxPagesPerFetch = 3;

const tabs: { id: Tab; label: string; empty: string }[] = [
  { id: "notes", label: "笔记", empty: "还没有发布任何笔记。" },
  { id: "media", label: "媒体", empty: "还没有带媒体的笔记。" },
  { id: "replies", label: "回复", empty: "还没有回复过别人。" },
];

const tab = ref<Tab>("notes");

const user = ref<UserDetailed | null>(null);
const loading = ref(false);
const loadError = ref("");
const missing = ref(false);
const missingRemote = ref(false);

const notes = ref<Note[]>([]);
const notesLoading = ref(false);
const notesLoadingMore = ref(false);
const notesError = ref("");
const untilId = ref("");
const hasMore = ref(false);

const following = ref(false);
const followBusy = ref(false);
const followNotice = ref("");

// The route is `/@:handle`, so the param holds `username` for a local
// account and `username@host` for a remote one. Only the local instance
// can serve a profile today.
const handle = computed(() => {
  const raw = route.params.handle;
  const value = Array.isArray(raw) ? (raw[0] ?? "") : String(raw ?? "");
  const at = value.indexOf("@");
  if (at < 0) return { username: value, host: "" };
  return { username: value.slice(0, at), host: value.slice(at + 1) };
});

const isRemote = computed(() => {
  const host = handle.value.host.toLowerCase();
  if (!host) return false;
  return host !== window.location.hostname.toLowerCase();
});

const isSelf = computed(() => {
  if (!user.value || !session.me) return false;
  return user.value.id === session.me.id || user.value.username === session.me.username;
});

// Editing the profile in settings replaces the session copy, so an own
// profile reflects a new name, avatar or description without a reload.
const viewUser = computed<UserDetailed | null>(() => {
  const current = user.value;
  if (!current) return null;
  if (!isSelf.value || !session.me) return current;
  return {
    ...current,
    name: session.me.name,
    avatarUrl: session.me.avatarUrl,
    description: session.me.description,
  };
});

const pinnedNotes = computed(() => user.value?.pinnedNotes ?? []);

const currentTab = computed(() => tabs.find((t) => t.id === tab.value) ?? tabs[0]);

function errorMessage(err: unknown, fallback: string): string {
  if (err instanceof ApiError) return err.message || fallback;
  if (err instanceof Error) return err.message || fallback;
  return fallback;
}

// The three tabs differ only in their users/notes filter: the notes tab
// drops replies to other users (Misskey's profile default), the media tab
// keeps notes with files, and the replies tab asks for everything and
// keeps only the replies — the backend has no replies-only switch.
interface NotesParams {
  username?: string;
  limit?: number;
  untilId?: string;
  withFiles?: boolean;
  withReplies?: boolean;
}

function pageParams(cursor: string): NotesParams {
  const params: NotesParams = { username: handle.value.username, limit: pageSize };
  if (cursor) params.untilId = cursor;
  if (tab.value === "notes") params.withReplies = false;
  else if (tab.value === "media") params.withFiles = true;
  return params;
}

function keepForTab(page: Note[]): Note[] {
  if (tab.value === "replies") return page.filter((n) => !!n.replyId);
  return page;
}

// Switching tabs while a page is in flight must not append the old tab's
// notes to the new list, so every fetch carries the generation it started in.
let generation = 0;

async function fetchNotes(reset: boolean, mine: number): Promise<void> {
  const seen = new Set(reset ? [] : notes.value.map((n) => n.id));
  const collected: Note[] = [];
  let cursor = reset ? "" : untilId.value;
  let more = true;

  for (let request = 0; request < maxPagesPerFetch && more; request += 1) {
    const page = await api.usersNotes(pageParams(cursor));
    if (mine !== generation) return;
    if (page.length) cursor = page[page.length - 1].id;
    more = page.length >= pageSize;
    for (const note of keepForTab(page)) {
      if (!seen.has(note.id)) {
        seen.add(note.id);
        collected.push(note);
      }
    }
    if (collected.length >= pageSize || !more) break;
  }

  untilId.value = cursor;
  hasMore.value = more;
  notes.value = reset ? collected : [...notes.value, ...collected];
}

async function loadNotes(): Promise<void> {
  const mine = ++generation;
  notesLoading.value = true;
  notesError.value = "";
  notes.value = [];
  untilId.value = "";
  hasMore.value = false;
  try {
    await fetchNotes(true, mine);
  } catch (err) {
    if (mine === generation) notesError.value = errorMessage(err, "加载笔记失败");
  } finally {
    if (mine === generation) notesLoading.value = false;
  }
}

async function loadMore(): Promise<void> {
  if (!untilId.value || notesLoadingMore.value) return;
  const mine = generation;
  notesLoadingMore.value = true;
  notesError.value = "";
  try {
    await fetchNotes(false, mine);
  } catch (err) {
    if (mine === generation) notesError.value = errorMessage(err, "加载更多失败");
  } finally {
    if (mine === generation) notesLoadingMore.value = false;
  }
}

function selectTab(next: Tab) {
  if (tab.value === next) return;
  tab.value = next;
  void loadNotes();
}

async function load() {
  const username = handle.value.username;
  user.value = null;
  loadError.value = "";
  missing.value = false;
  missingRemote.value = false;
  following.value = false;
  followNotice.value = "";
  notes.value = [];
  notesError.value = "";
  untilId.value = "";
  hasMore.value = false;
  tab.value = "notes";

  if (!username) {
    missing.value = true;
    return;
  }
  if (isRemote.value) {
    missing.value = true;
    missingRemote.value = true;
    return;
  }

  loading.value = true;
  try {
    user.value = await api.usersShow({ username });
    await loadNotes();
  } catch (err) {
    if (err instanceof ApiError && err.status === 404) {
      missing.value = true;
    } else {
      loadError.value = errorMessage(err, "加载用户失败");
    }
  } finally {
    loading.value = false;
  }
}

// Pinning happens in the note footer, which only updates the session copy
// of the user. Re-reading the profile keeps the pinned strip in step.
async function refreshPinned() {
  if (!user.value) return;
  try {
    const fresh = await api.usersShow({ username: user.value.username });
    if (user.value) {
      user.value.pinnedNoteIds = fresh.pinnedNoteIds;
      user.value.pinnedNotes = fresh.pinnedNotes;
    }
  } catch {
    // A stale strip is better than an error notice here.
  }
}

watch(
  () => session.me?.pinnedNoteIds?.join(",") ?? "",
  (key) => {
    if (!isSelf.value || !user.value) return;
    if (key === (user.value.pinnedNoteIds ?? []).join(",")) return;
    void refreshPinned();
  },
);

async function follow() {
  if (!user.value || followBusy.value) return;
  followBusy.value = true;
  followNotice.value = "";
  try {
    await api.follow(user.value.id);
    following.value = true;
    user.value.followersCount += 1;
  } catch (err) {
    if (err instanceof ApiError) {
      if (err.code === "ALREADY_FOLLOWING") {
        following.value = true;
      } else if (err.status === 401) {
        followNotice.value = "登录状态已失效，请重新登录后再关注。";
      } else {
        followNotice.value = errorMessage(err, "关注失败");
      }
    } else {
      followNotice.value = errorMessage(err, "关注失败");
    }
  } finally {
    followBusy.value = false;
  }
}

async function unfollow() {
  if (!user.value || followBusy.value) return;
  followBusy.value = true;
  followNotice.value = "";
  try {
    await api.unfollow(user.value.id);
    following.value = false;
    user.value.followersCount = Math.max(0, user.value.followersCount - 1);
  } catch (err) {
    if (err instanceof ApiError) {
      if (err.code === "NOT_FOLLOWING") {
        following.value = false;
      } else if (err.status === 401) {
        followNotice.value = "登录状态已失效，请重新登录后再操作。";
      } else {
        followNotice.value = errorMessage(err, "取消关注失败");
      }
    } else {
      followNotice.value = errorMessage(err, "取消关注失败");
    }
  } finally {
    followBusy.value = false;
  }
}

function onReply(note: Note) {
  router.push(`/notes/${note.id}`);
}

function onDeleted(id: string) {
  notes.value = notes.value.filter((n) => n.id !== id);
  if (user.value) {
    user.value.pinnedNotes = user.value.pinnedNotes.filter((n) => n.id !== id);
    user.value.pinnedNoteIds = user.value.pinnedNoteIds.filter((noteId) => noteId !== id);
  }
}

function onRenoted(note: Note) {
  if (!notes.value.some((n) => n.id === note.id)) notes.value = [note, ...notes.value];
}

watch(
  () => handle.value.username,
  () => {
    void load();
  },
  { immediate: true },
);
</script>

<template>
  <p v-if="loading" class="skeleton"></p>

  <div v-else-if="missing" class="card placeholder">
    <AppIcon name="globe" />
    <h2>远程账户暂不可用</h2>
    <p class="small">
      <template v-if="missingRemote">
        @{{ handle.username }}@{{ handle.host }} 位于其他实例，Cinderfish
        目前只支持浏览本地账户的资料。
      </template>
      <template v-else>没有找到这个账户，可能已经被删除或从未存在过。</template>
    </p>
    <RouterLink to="/explore" class="button ghost small">去发现页看看</RouterLink>
  </div>

  <p v-else-if="loadError" class="notice error">{{ loadError }}</p>

  <template v-else-if="viewUser">
    <ProfileHeader
      :user="viewUser"
      :is-self="isSelf"
      :logged-in="isLoggedIn"
      :following="following"
      :follow-busy="followBusy"
      @follow="follow"
      @unfollow="unfollow"
    />

    <p v-if="followNotice" class="notice error follow-notice">{{ followNotice }}</p>

    <section v-if="pinnedNotes.length" class="pinned" aria-labelledby="pinned-head">
      <h2 id="pinned-head" class="pinned-head">
        <AppIcon name="check" />
        置顶笔记
      </h2>
      <div class="notes">
        <NoteItem
          v-for="note in pinnedNotes"
          :key="`pinned-${note.id}`"
          :note="note"
          @reply="onReply"
          @deleted="onDeleted"
          @renoted="onRenoted"
        />
      </div>
    </section>

    <div class="tab-row" role="group" aria-label="个人资料分页">
      <button
        v-for="option in tabs"
        :key="option.id"
        type="button"
        class="tab"
        :class="{ active: tab === option.id }"
        :aria-pressed="tab === option.id"
        @click="selectTab(option.id)"
      >
        {{ option.label }}
      </button>
    </div>

    <p v-if="notesError" class="notice error">{{ notesError }}</p>

    <div v-if="notesLoading" class="skeleton-list">
      <div v-for="i in 3" :key="i" class="skeleton-item"></div>
    </div>

    <template v-else>
      <div class="notes">
        <NoteItem
          v-for="note in notes"
          :key="note.id"
          :note="note"
          @reply="onReply"
          @deleted="onDeleted"
          @renoted="onRenoted"
        />
      </div>

      <p v-if="!notes.length && !hasMore && !notesError" class="placeholder">
        <AppIcon name="feather" />
        <span>{{ currentTab.empty }}</span>
      </p>

      <div v-if="hasMore" class="more">
        <button class="button ghost small" type="button" :disabled="notesLoadingMore" @click="loadMore">
          {{ notesLoadingMore ? "加载中…" : "加载更多" }}
        </button>
      </div>
    </template>
  </template>
</template>

<style scoped>
.follow-notice {
  margin-top: 12px;
}

.pinned {
  margin-top: 16px;
  padding: 10px 12px 0;
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  background: var(--panel);
}

.pinned-head {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 0.85rem;
  font-weight: 700;
  color: var(--accent);
  padding: 0 2px 4px;
}

.pinned-head :deep(.icon) {
  width: 1em;
  height: 1em;
}

.pinned .notes > :deep(.note:last-child) {
  border-bottom: 0;
}

.pinned .notes {
  margin: 0 -12px;
}

.pinned .notes > :deep(.note) {
  padding-left: 12px;
  padding-right: 12px;
}

.tab-row {
  display: flex;
  gap: 4px;
  margin: 18px 0 8px;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--divider);
  overflow-x: auto;
}

.tab {
  padding: 7px 16px;
  border: 1px solid transparent;
  border-radius: 999px;
  background: transparent;
  color: var(--muted);
  font-weight: 600;
  font-size: 0.9rem;
  cursor: pointer;
  transition: color 0.15s ease, background 0.15s ease, border-color 0.15s ease;
  white-space: nowrap;
}

.tab:hover,
.tab:focus-visible {
  color: var(--text);
  background: var(--panel-2);
}

.tab.active {
  color: var(--accent);
  background: var(--accent-soft);
  border-color: color-mix(in srgb, var(--accent) 40%, transparent);
}

/* Borderless rows: dividers come from the note itself, so no gap. */
.notes {
  display: flex;
  flex-direction: column;
}

.notes > :deep(.note:last-child) {
  border-bottom: 0;
}

.more {
  display: flex;
  justify-content: center;
  padding: 16px 0 0;
}

.skeleton {
  height: 220px;
  border-radius: var(--radius-lg);
  background: var(--panel);
  border: 1px solid var(--border);
}

.skeleton-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.skeleton-item {
  height: 120px;
  border-radius: var(--radius-lg);
  background: linear-gradient(90deg, var(--panel) 25%, var(--panel-2) 50%, var(--panel) 75%);
  background-size: 200% 100%;
  animation: shimmer 1.4s infinite linear;
}

@keyframes shimmer {
  from {
    background-position: 200% 0;
  }
  to {
    background-position: -200% 0;
  }
}
</style>
