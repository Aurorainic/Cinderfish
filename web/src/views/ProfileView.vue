<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { api, ApiError } from "../api/client";
import type { Note, UserDetailed } from "../api/types";
import AppIcon from "../components/AppIcon.vue";
import NoteItem from "../components/note/Note.vue";
import ProfileCard from "../components/profile/ProfileCard.vue";
import ProfileSection from "../components/profile/ProfileSection.vue";
import ProfileSidebar from "../components/profile/ProfileSidebar.vue";
import PageHeader from "../components/ui/PageHeader.vue";
import PagedFeed from "../components/ui/PagedFeed.vue";
import type { PagedFeedParams, PagedFeedResult } from "../components/ui/PagedFeed.vue";
import { usePageTitle } from "../composables/use-page-title";
import { useFollowRelation } from "../composables/use-follow-relation";
import { NAV_ITEM_DEFS } from "../nav";
import { isLoggedIn, session } from "../stores/session";

type Tab = "notes" | "media" | "replies" | "following" | "followers";
// The two follow lists have no endpoint yet; their tabs exist so the
// unavailable state is reachable and shareable instead of silently falling
// back to the note list.
type ListTab = "following" | "followers";

type SectionIcon = InstanceType<typeof AppIcon>["$props"]["name"];

const TABS = [
  { key: "notes", label: "笔记", icon: "feather" },
  { key: "media", label: "媒体", icon: "image" },
  { key: "replies", label: "回复", icon: "reply" },
  { key: "following", label: "关注中", icon: "user" },
  { key: "followers", label: "关注者", icon: "users" },
] as const satisfies readonly { key: Tab; label: string; icon: SectionIcon }[];

const TAB_KEYS: readonly string[] = TABS.map((item) => item.key);

const EMPTY_TEXT: Record<Tab, string> = {
  notes: "还没有发布任何笔记。",
  media: "还没有带媒体的笔记。",
  replies: "还没有回复过别人。",
  following: "",
  followers: "",
};

// Each tab names its section, so the list below the tab strip always says what
// it is instead of relying on the tab alone.
const SECTIONS: Record<Tab, { title: string; icon: SectionIcon; hint: string }> = {
  notes: { title: "最近的笔记", icon: "feather", hint: "按发布时间从新到旧" },
  media: { title: "最近的媒体", icon: "image", hint: "带附件的笔记" },
  replies: { title: "最近的回复", icon: "reply", hint: "回复别人的笔记" },
  following: { title: "关注中", icon: "user", hint: "这个账户关注的账户" },
  followers: { title: "关注者", icon: "users", hint: "关注这个账户的账户" },
};

const PAGE_SIZE = 20;
const WIDE_THRESHOLD = 1000;

const route = useRoute();
const router = useRouter();

const rootEl = ref<HTMLElement | null>(null);
const wide = ref(false);

const user = ref<UserDetailed | null>(null);
const loading = ref(false);
const loadError = ref("");
const missing = ref(false);
const missingRemote = ref(false);

const following = ref(false);
const followPending = ref(false);
const followBusy = ref(false);
const followNotice = ref("");

const moreOpen = ref(false);
const moreNotice = ref("");
const moreWrap = ref<HTMLElement | null>(null);

const feed = ref<InstanceType<typeof PagedFeed> | null>(null);

const handle = computed(() => {
  const raw = route.params.handle;
  const value = Array.isArray(raw) ? (raw[0] ?? "") : String(raw ?? "");
  const at = value.indexOf("@");
  if (at < 0) return { username: value, host: "" };
  return { username: value.slice(0, at), host: value.slice(at + 1) };
});

const acct = computed(() => (handle.value.host ? `${handle.value.username}@${handle.value.host}` : handle.value.username));

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

const displayName = computed(() => viewUser.value?.name || viewUser.value?.username || "");
const headerTitle = computed(() => displayName.value || acct.value);
const headerSubtitle = computed(() => (acct.value ? `@${acct.value}` : ""));
const pinnedNotes = computed(() => user.value?.pinnedNotes ?? []);
const loginTarget = computed(() => ({ path: "/login", query: { redirect: route.fullPath } }));
// The note list lives at the bare profile URL, so a counter or an unavailable
// panel can point back at it without repeating the tab bookkeeping.
const notesTabLink = computed(() => ({ path: route.path, query: { ...route.query, tab: undefined } }));

usePageTitle(() => (displayName.value ? `${displayName.value} (@${handle.value.username})` : ""));

// The selected tab lives in `?tab=`, so a reload, a shared link and the back
// button all land on the same list.
const tab = computed<string>({
  get() {
    const raw = typeof route.query.tab === "string" ? route.query.tab : "";
    return TAB_KEYS.includes(raw) ? raw : "notes";
  },
  set(value) {
    if (tab.value === value) return;
    const query = { ...route.query };
    if (value === "notes") delete query.tab;
    else query.tab = value;
    void router.push({ query });
  },
});

const activeTab = computed<Tab>(() => (TAB_KEYS.includes(tab.value) ? (tab.value as Tab) : "notes"));
const section = computed(() => SECTIONS[activeTab.value]);
const emptyText = computed(() => EMPTY_TEXT[activeTab.value]);
const feedKey = computed(() => `${handle.value.username}:${activeTab.value}`);

const listTab = computed<ListTab | null>(() => {
  if (activeTab.value === "following" || activeTab.value === "followers") return activeTab.value;
  return null;
});

// The follow lists have no endpoint (users/following, users/followers are not
// implemented), so the tab reports exactly that and never invents rows. The
// body names the missing endpoint; the count beside it comes from the user
// record and is accurate.
const unavailable = computed(() => {
  if (listTab.value === "followers") {
    return {
      title: "关注者列表暂不可用",
      body: `后端还没有提供 users/followers 接口，这一页无法列出关注 @${acct.value} 的账户。`,
      count: viewUser.value?.followersCount ?? 0,
      countLabel: "关注者",
    };
  }
  return {
    title: "关注列表暂不可用",
    body: `后端还没有提供 users/following 接口，这一页无法列出 @${acct.value} 关注的账户。`,
    count: viewUser.value?.followingCount ?? 0,
    countLabel: "关注中",
  };
});

function formatCount(n: number): string {
  return n.toLocaleString("zh-CN");
}

interface NotesParams {
  username?: string;
  limit?: number;
  untilId?: string;
  withFiles?: boolean;
  withReplies?: boolean;
}

// Switching tabs remounts the feed, which resets this cursor too: it only
// exists so the replies tab can paginate on the raw page while showing a
// filtered subset of it.
let rawCursor = "";

async function fetcher(p: PagedFeedParams): Promise<PagedFeedResult> {
  const params: NotesParams = { username: handle.value.username, limit: PAGE_SIZE };
  const filtering = activeTab.value === "replies";

  if (filtering) {
    if (p.untilId === undefined) rawCursor = "";
    if (rawCursor) params.untilId = rawCursor;
    // Keep withReplies on (the backend default) so the replies filter has
    // rows to select from; the unfiltered notes tab excludes them instead.
  } else if (p.untilId) {
    params.untilId = p.untilId;
  }

  if (activeTab.value === "notes") {
    // The replies tab filters client-side instead of passing
    // withReplies: false here, so this branch only shapes the notes tab;
    // users/notes defaults to including replies (backend divergence from
    // Misskey's false default, kept deliberate).
    params.withReplies = false;
  } else if (activeTab.value === "media") {
    params.withFiles = true;
  }

  const page = await api.usersNotes(params);
  if (page.length) rawCursor = page[page.length - 1].id;
  if (!filtering) return page;
  return { items: page.filter((note) => !!note.replyId), hasMore: page.length >= PAGE_SIZE };
}

function errorMessage(err: unknown, fallback: string): string {
  if (err instanceof ApiError) return err.message || fallback;
  if (err instanceof Error) return err.message || fallback;
  return fallback;
}

async function load() {
  const username = handle.value.username;
  user.value = null;
  loadError.value = "";
  missing.value = false;
  missingRemote.value = false;
  following.value = false;
  followPending.value = false;
  followNotice.value = "";

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
    const fresh = await api.usersShow({ username });
    user.value = fresh;
    // The backend resolves the viewer's relationship, so the button starts in
    // the right state instead of assuming "not following".
    following.value = fresh.isFollowing ?? false;
    followPending.value = fresh.hasPendingFollowRequestFromYou ?? false;
  } catch (err) {
    if (err instanceof ApiError && err.status === 404) missing.value = true;
    else loadError.value = errorMessage(err, "加载用户失败");
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

// following/create answers with the followee's UserLite only (Misskey
// contract), so the relationship after a follow is read back from
// users/show instead of being returned there; see useFollowRelation.
const { refreshRelation } = useFollowRelation(user, following, followPending);

async function follow() {
  if (!user.value || followBusy.value || followPending.value) return;
  followBusy.value = true;
  followNotice.value = "";
  try {
    await api.follow(user.value.id);
    following.value = true;
    user.value.followersCount += 1;
    await refreshRelation();
  } catch (err) {
    if (err instanceof ApiError) {
      if (err.code === "ALREADY_FOLLOWING") {
        following.value = true;
        await refreshRelation();
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

function toggleFollow() {
  if (following.value) void unfollow();
  else void follow();
}

function onReply(note: Note) {
  router.push(`/notes/${note.id}`);
}

function onDeleted(id: string) {
  feed.value?.remove(id);
  if (!user.value) return;
  user.value.pinnedNotes = user.value.pinnedNotes.filter((note) => note.id !== id);
  user.value.pinnedNoteIds = user.value.pinnedNoteIds.filter((noteId) => noteId !== id);
}

let moreTimer: ReturnType<typeof setTimeout> | undefined;

function flashMore(text: string) {
  moreNotice.value = text;
  clearTimeout(moreTimer);
  moreTimer = setTimeout(() => {
    moreNotice.value = "";
  }, 2500);
}

async function copyLink() {
  moreOpen.value = false;
  try {
    await navigator.clipboard.writeText(window.location.href);
    flashMore("已复制个人页链接");
  } catch {
    flashMore("复制失败，请手动复制地址栏链接");
  }
}

function reload() {
  moreOpen.value = false;
  void load();
}

function onDocumentClick(event: MouseEvent) {
  if (!moreOpen.value) return;
  const target = event.target;
  if (target instanceof Node && moreWrap.value?.contains(target)) return;
  moreOpen.value = false;
}

function onKeydown(event: KeyboardEvent) {
  if (event.key === "Escape") moreOpen.value = false;
}

// The side column joins the main column below this width. A container
// measurement is used instead of a media query because the page also renders
// inside shells whose main area is narrower than the viewport.
let observer: ResizeObserver | null = null;

function measure() {
  wide.value = (rootEl.value?.clientWidth ?? 0) >= WIDE_THRESHOLD;
}

onMounted(() => {
  measure();
  if (rootEl.value && typeof ResizeObserver !== "undefined") {
    observer = new ResizeObserver(measure);
    observer.observe(rootEl.value);
  }
  document.addEventListener("click", onDocumentClick);
  document.addEventListener("keydown", onKeydown);
});

onBeforeUnmount(() => {
  observer?.disconnect();
  observer = null;
  clearTimeout(moreTimer);
  document.removeEventListener("click", onDocumentClick);
  document.removeEventListener("keydown", onKeydown);
});

watch(
  () => handle.value.username,
  () => {
    void load();
  },
  { immediate: true },
);
</script>

<template>
  <div ref="rootEl" class="profile-page">
    <PageHeader
      v-model:tab="tab"
      :title="headerTitle"
      :subtitle="headerSubtitle"
      :avatar="viewUser"
      :tabs="viewUser ? TABS : null"
    >
      <template #actions>
        <template v-if="viewUser">
          <RouterLink v-if="isSelf" :to="NAV_ITEM_DEFS.settings.to" class="button ghost small">编辑资料</RouterLink>
          <button
            v-else-if="isLoggedIn"
            type="button"
            class="button small"
            :class="{ ghost: following || followPending }"
            :disabled="followBusy || followPending"
            @click="toggleFollow"
          >
            <template v-if="followBusy">处理中…</template>
            <template v-else-if="followPending">请求已发送</template>
            <template v-else-if="following">取消关注</template>
            <template v-else>关注</template>
          </button>
          <RouterLink v-else :to="loginTarget" class="button small">登录后关注</RouterLink>

          <div ref="moreWrap" class="more">
            <button
              class="icon-button"
              type="button"
              title="更多"
              aria-label="更多"
              aria-haspopup="menu"
              :aria-expanded="moreOpen"
              @click="moreOpen = !moreOpen"
            >
              <AppIcon name="more" />
            </button>
            <div v-if="moreOpen" class="more-menu" role="menu">
              <button type="button" class="more-item" role="menuitem" @click="copyLink">复制链接</button>
              <button type="button" class="more-item" role="menuitem" @click="reload">刷新资料</button>
            </div>
          </div>
        </template>
      </template>
    </PageHeader>

    <p v-if="moreNotice" class="notice info page-notice">{{ moreNotice }}</p>
    <p v-if="followNotice" class="notice error page-notice">{{ followNotice }}</p>

    <div v-if="loading" class="skeleton"></div>

    <div v-else-if="missing" class="card placeholder">
      <AppIcon name="globe" />
      <h2>远程账户暂不可用</h2>
      <p class="small">
        <template v-if="missingRemote">
          @{{ handle.username }}@{{ handle.host }} 位于其他实例，Cinderfish 目前只支持浏览本地账户的资料。
        </template>
        <template v-else>没有找到这个账户，可能已经被删除或从未存在过。</template>
      </p>
      <RouterLink to="/explore" class="button ghost small">去探索页看看</RouterLink>
    </div>

    <p v-else-if="loadError" class="notice error">
      <span>{{ loadError }}</span>
      <button type="button" class="button ghost small retry" @click="load">重试</button>
    </p>

    <div v-else-if="viewUser" class="profile-body" :class="{ wide }">
      <div class="area-stack">
        <ProfileCard :user="viewUser" />

        <ProfileSection
          v-if="pinnedNotes.length"
          title="置顶笔记"
          icon="check"
          :count="pinnedNotes.length"
          hint="作者固定在个人页顶部的笔记"
        >
          <div class="pinned-notes">
            <NoteItem
              v-for="note in pinnedNotes"
              :key="`pinned-${note.id}`"
              :note="note"
              @reply="onReply"
              @deleted="onDeleted"
            />
          </div>
        </ProfileSection>

        <p v-else-if="isSelf" class="notice info pin-tip">
          置顶笔记会显示在个人页顶部，在笔记菜单中选择置顶。
        </p>
      </div>

      <ProfileSidebar class="area-side" :user="viewUser" />

      <div class="area-feed">
        <ProfileSection v-if="!listTab" :title="section.title" :icon="section.icon" :hint="section.hint">
          <PagedFeed ref="feed" :key="feedKey" :fetcher="fetcher" :page-size="PAGE_SIZE">
            <template #item="{ item }">
              <NoteItem :note="(item as Note)" @reply="onReply" @deleted="onDeleted" />
            </template>
            <template #empty>
              <div class="placeholder">
                <AppIcon name="feather" />
                <span>{{ emptyText }}</span>
              </div>
            </template>
          </PagedFeed>
        </ProfileSection>

        <ProfileSection v-else :title="section.title" :icon="section.icon" :hint="section.hint">
          <div class="unavailable">
            <span class="unavailable-icon"><AppIcon :name="section.icon" /></span>
            <h3 class="unavailable-title">{{ unavailable.title }}</h3>
            <p class="small muted unavailable-body">{{ unavailable.body }}</p>
            <p class="unavailable-count">
              <b>{{ formatCount(unavailable.count) }}</b>
              <span>{{ unavailable.countLabel }}</span>
            </p>
            <p class="unavailable-note">
              数量来自用户资料，是准确的；接口就绪后列表会出现在这里。
            </p>
            <RouterLink :to="notesTabLink" class="button ghost small">查看笔记</RouterLink>
          </div>
        </ProfileSection>
      </div>
    </div>
  </div>
</template>

<style scoped>
.profile-page {
  min-width: 0;
}

.page-notice {
  margin-top: var(--space-12);
}

.retry {
  margin-inline-start: var(--space-10);
}

.more {
  position: relative;
  display: flex;
}

.more-menu {
  position: absolute;
  top: calc(100% + var(--space-6));
  right: 0;
  z-index: 2;
  display: flex;
  flex-direction: column;
  min-width: 150px;
  padding: var(--space-4);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--popup-bg);
  box-shadow: var(--shadow-popover);
}

.more-item {
  padding: var(--space-8) var(--space-10);
  border: 0;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--text);
  font: inherit;
  font-size: var(--fs-sm);
  text-align: start;
  cursor: pointer;
}

.more-item:hover {
  background: var(--surface-hover);
}

.more-item:focus-visible {
  outline: var(--focus-outline);
  outline-offset: -2px;
}

/* The narrow grid keeps the side column's place right after the profile card. */
.profile-body {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  grid-template-areas: "stack" "side" "feed";
  gap: var(--space-16);
  margin-top: var(--space-16);
  min-width: 0;
}

.profile-body.wide {
  grid-template-columns: minmax(0, 1fr) var(--side-width);
  grid-template-areas: "stack side" "feed side";
  align-items: start;
}

.area-stack {
  grid-area: stack;
  display: flex;
  flex-direction: column;
  gap: var(--space-16);
  min-width: 0;
}

.area-side {
  grid-area: side;
  min-width: 0;
}

.profile-body.wide .area-side {
  position: sticky;
  top: var(--space-64);
}

.area-feed {
  grid-area: feed;
  min-width: 0;
}

.pinned-notes :deep(.note:last-child) {
  border-bottom: 0;
}

.pin-tip {
  margin: 0;
}

.unavailable {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-8);
  padding: var(--space-32) var(--space-20);
  text-align: center;
}

.unavailable-icon {
  display: grid;
  place-items: center;
  width: 46px;
  height: 46px;
  border-radius: 50%;
  background: var(--surface-inset);
  color: var(--text-faint);
}

.unavailable-icon :deep(.icon) {
  width: 22px;
  height: 22px;
}

.unavailable-title {
  font-size: var(--fs-md);
}

.unavailable-body {
  max-width: 40ch;
  margin: 0;
}

.unavailable-count {
  display: inline-flex;
  align-items: baseline;
  gap: var(--space-6);
  margin-top: var(--space-4);
  color: var(--text-weak);
  font-size: var(--fs-sm);
}

.unavailable-count b {
  color: var(--text-strong);
  font-size: var(--fs-lg);
  font-variant-numeric: tabular-nums;
}

.unavailable-note {
  max-width: 42ch;
  margin: 0 0 var(--space-6);
  color: var(--text-faint);
  font-size: var(--fs-2xs);
  line-height: 1.5;
}

.skeleton {
  height: 220px;
  margin-top: var(--space-16);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  background: var(--panel);
}
</style>
