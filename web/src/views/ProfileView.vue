<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { api, ApiError } from "../api/client";
import type { Note, UserDetailed } from "../api/types";
import AppIcon from "../components/AppIcon.vue";
import NoteItem from "../components/note/Note.vue";
import ProfileHeader from "../components/ProfileHeader.vue";
import { isLoggedIn, session } from "../stores/session";

const route = useRoute();
const router = useRouter();

const pageSize = 20;

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

function errorMessage(err: unknown, fallback: string): string {
  if (err instanceof ApiError) return err.message || fallback;
  if (err instanceof Error) return err.message || fallback;
  return fallback;
}

async function loadNotes(username: string) {
  notesLoading.value = true;
  notesError.value = "";
  notes.value = [];
  untilId.value = "";
  hasMore.value = false;
  try {
    const page = await api.usersNotes({ username, limit: pageSize });
    notes.value = page;
    untilId.value = page.length ? page[page.length - 1].id : "";
    hasMore.value = page.length >= pageSize;
  } catch (err) {
    notesError.value = errorMessage(err, "加载笔记失败");
  } finally {
    notesLoading.value = false;
  }
}

async function loadMore() {
  if (!untilId.value || notesLoadingMore.value) return;
  notesLoadingMore.value = true;
  notesError.value = "";
  try {
    const page = await api.usersNotes({
      username: handle.value.username,
      limit: pageSize,
      untilId: untilId.value,
    });
    const seen = new Set(notes.value.map((n) => n.id));
    notes.value = [...notes.value, ...page.filter((n) => !seen.has(n.id))];
    untilId.value = page.length ? page[page.length - 1].id : "";
    hasMore.value = page.length >= pageSize;
  } catch (err) {
    notesError.value = errorMessage(err, "加载更多失败");
  } finally {
    notesLoadingMore.value = false;
  }
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
    await loadNotes(username);
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

  <template v-else-if="user">
    <ProfileHeader
      :user="user"
      :is-self="isSelf"
      :logged-in="isLoggedIn"
      :following="following"
      :follow-busy="followBusy"
      @follow="follow"
      @unfollow="unfollow"
    />

    <p v-if="followNotice" class="notice error follow-notice">{{ followNotice }}</p>

    <div class="notes-head">
      <h2>笔记</h2>
      <span class="small muted">{{ notesLoading ? "加载中…" : `${notes.length} 条` }}</span>
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

      <p v-if="!notes.length && !notesError" class="placeholder">
        <AppIcon name="feather" />
        <span>还没有发布任何笔记。</span>
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

.notes-head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  padding: 0 4px;
  margin: 20px 0 12px;
}

.notes-head h2 {
  font-size: 1.05rem;
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
