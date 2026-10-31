<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { api } from "../../api/client";
import type { Note, UserDetailed } from "../../api/types";
import { isLoggedIn, session } from "../../stores/session";
import AppIcon from "../AppIcon.vue";
import NoteDetail from "../note/NoteDetail.vue";
import NoteItem from "../note/Note.vue";
import ProfileHeader from "../ProfileHeader.vue";
import PageWindow from "../ui/PageWindow.vue";
import PagedFeed from "../ui/PagedFeed.vue";
import { closeDeckPage, deckPage } from "./deck-page";
import { interceptDeckClick } from "./deck-links";

const PROFILE_PAGE_SIZE = 20;

// The overlay is a single layer above the deck: a column raises a page here and
// this component fetches it. `/@user@host` handles are accepted in the shape
// the profile route uses, but only local accounts can be served today.
const noteId = computed(() => (deckPage.value?.kind === "note" ? deckPage.value.id : ""));
const handle = computed(() => (deckPage.value?.kind === "profile" ? deckPage.value.handle : ""));
const username = computed(() => handle.value.split("@")[0] ?? "");
const host = computed(() => (handle.value.includes("@") ? handle.value.split("@").slice(1).join("@") : ""));
const title = computed(() => (noteId.value ? "笔记" : handle.value ? `@${handle.value}` : ""));

const note = ref<Note | null>(null);
const noteLoading = ref(false);
const noteError = ref("");

const user = ref<UserDetailed | null>(null);
const userLoading = ref(false);
const userError = ref("");
const following = ref(false);
const followPending = ref(false);
const followBusy = ref(false);
const followNotice = ref("");

const isSelf = computed(() => {
  if (!user.value || !session.me) return false;
  return user.value.id === session.me.id || user.value.username === session.me.username;
});

function errorMessage(err: unknown, fallback: string): string {
  return err instanceof Error && err.message ? err.message : fallback;
}

async function loadNote(id: string): Promise<void> {
  note.value = null;
  noteError.value = "";
  if (!id) return;
  noteLoading.value = true;
  try {
    note.value = await api.showNote(id);
  } catch (err) {
    noteError.value = errorMessage(err, "加载笔记失败");
  } finally {
    noteLoading.value = false;
  }
}

async function loadUser(value: string): Promise<void> {
  user.value = null;
  userError.value = "";
  followNotice.value = "";
  following.value = false;
  followPending.value = false;
  if (!value) return;
  if (host.value) {
    userError.value = "远程账户暂不可用：Cinderfish 目前只支持浏览本地账户的资料。";
    return;
  }
  userLoading.value = true;
  try {
    const fresh = await api.usersShow({ username: username.value });
    user.value = fresh;
    // The backend resolves the viewer's relationship, so the button starts in
    // the right state instead of assuming "not following".
    following.value = fresh.isFollowing ?? false;
    followPending.value = fresh.hasPendingFollowRequestFromYou ?? false;
  } catch (err) {
    userError.value = errorMessage(err, "加载用户失败");
  } finally {
    userLoading.value = false;
  }
}

watch(noteId, (id) => void loadNote(id), { immediate: true });
watch(handle, (value) => void loadUser(value), { immediate: true });

function profileNotesFetcher(p: { untilId?: string }): Promise<Note[]> {
  return api.usersNotes({ username: username.value, limit: PROFILE_PAGE_SIZE, untilId: p.untilId });
}

// following/create answers with the followee's UserLite only (Misskey
// contract), so a pending request on a locked or remote account is read
// back from users/show instead of being returned there.
async function refreshRelation(): Promise<void> {
  if (!user.value) return;
  try {
    const fresh = await api.usersShow({ username: user.value.username });
    if (!user.value) return;
    following.value = fresh.isFollowing ?? false;
    followPending.value = fresh.hasPendingFollowRequestFromYou ?? false;
    if (typeof fresh.followersCount === "number") user.value.followersCount = fresh.followersCount;
  } catch {
    // The optimistic state from the follow call stands on a failed read.
  }
}

async function follow(): Promise<void> {
  if (!user.value || followBusy.value || followPending.value) return;
  followBusy.value = true;
  followNotice.value = "";
  try {
    await api.follow(user.value.id);
    following.value = true;
    user.value.followersCount += 1;
    await refreshRelation();
  } catch (err) {
    followNotice.value = errorMessage(err, "关注失败");
  } finally {
    followBusy.value = false;
  }
}

async function unfollow(): Promise<void> {
  if (!user.value || followBusy.value) return;
  followBusy.value = true;
  followNotice.value = "";
  try {
    await api.unfollow(user.value.id);
    following.value = false;
    user.value.followersCount = Math.max(0, user.value.followersCount - 1);
  } catch (err) {
    followNotice.value = errorMessage(err, "取消关注失败");
  } finally {
    followBusy.value = false;
  }
}

function onNoteDeleted(id: string): void {
  if (note.value?.id === id) closeDeckPage();
}
</script>

<template>
  <PageWindow v-if="title" :title="title" @close="closeDeckPage">
    <div class="window-pad" @click.capture="interceptDeckClick">
      <template v-if="noteId">
        <div v-if="noteLoading" class="skeleton"></div>
        <p v-else-if="noteError" class="notice error">{{ noteError }}</p>
        <NoteDetail v-else-if="note" :note="note" @deleted="onNoteDeleted" />
      </template>

      <template v-else>
        <div v-if="userLoading" class="skeleton"></div>
        <p v-else-if="userError" class="notice error">{{ userError }}</p>
        <template v-else-if="user">
          <ProfileHeader
            :user="user"
            :is-self="isSelf"
            :logged-in="isLoggedIn"
            :following="following"
            :pending="followPending"
            :follow-busy="followBusy"
            @follow="follow"
            @unfollow="unfollow"
          />
          <p v-if="followNotice" class="notice error">{{ followNotice }}</p>
          <PagedFeed :key="username" class="profile-feed" :fetcher="profileNotesFetcher">
            <template #item="{ item }">
              <NoteItem :note="(item as Note)" @deleted="onNoteDeleted" />
            </template>
            <template #empty>
              <div class="placeholder">
                <AppIcon name="feather" />
                <span>还没有笔记。</span>
              </div>
            </template>
          </PagedFeed>
        </template>
      </template>
    </div>
  </PageWindow>
</template>

<style scoped>
.window-pad {
  padding: 14px 18px 24px;
}

.profile-feed {
  margin-top: 16px;
}

.profile-feed :deep(.paged-feed) {
  gap: 0;
}

.skeleton {
  height: 160px;
  border-radius: var(--radius);
  background: var(--panel-2);
}
</style>
