// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/pages/user/home.vue
// (profile block + contents column) and pages/user/index.vue (tab skeleton).
//
// Adaptations versus the original:
// - Remote users are not reachable (users/show has no host support), so the
//   MkRemoteCaution / MkMoved branch is replaced by the unavailable card.
// - Mfm renders through lib/mfm.ts; fields, birthday, location and the
//   city-timezone lookup are dropped (UserDetailed keeps those null/[]).
// - follow-list tabs stay out of the tab strip: users/following and
//   users/followers do not exist, so only the counts are shown.
// - The reactions tab is dropped: notes/reactions does not exist.
<script lang="ts" setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { api, ApiError } from "../../api/client";
import type { UserDetailed } from "../../api/types";
import AppIcon from "../../components/AppIcon.vue";
import MkA from "../../components/ui/MkA.vue";
import MkAvatar from "../../components/ui/MkAvatar.vue";
import MkInfo from "../../components/ui/MkInfo.vue";
import MkUserName from "../../components/ui/MkUserName.vue";
import PageHeader from "../../components/ui/PageHeader.vue";
import NoteItem from "../../components/note/Note.vue";
import { renderMFM } from "../../lib/mfm";
import { i18n } from "../../i18n";
import { isLoggedIn, session } from "../../stores/session";
import { NAV_ITEM_DEFS } from "../../nav";
import { usePageTitle } from "../../composables/use-page-title";
import { useFollowRelation } from "../../composables/use-follow-relation";
import FollowButton from "./FollowButton.vue";
import UserPhotos from "./UserPhotos.vue";
import UserTimeline from "./UserTimeline.vue";

type Tab = "home" | "media";

const TABS: { key: Tab; label: string; icon: "user" | "image" }[] = [
	{ key: "home", label: "概览", icon: "user" },
	{ key: "media", label: "媒体", icon: "image" },
];

const props = defineProps<{
	/** Local part of the handle, without the leading @. */
	username: string;
}>();

const user = ref<UserDetailed | null>(null);
const loading = ref(false);
const loadError = ref("");
const missing = ref(false);

const following = ref(false);
const followBusy = ref(false);
const followNotice = ref("");

const WIDE_THRESHOLD = 1000;
const rootEl = ref<HTMLElement | null>(null);
const wide = ref(true);

const isSelf = computed(() => {
	if (!user.value || !session.me) return false;
	return user.value.id === session.me.id || user.value.username === session.me.username;
});

// Editing the profile in settings replaces the session copy, so an own
// profile reflects a new name, avatar or description without a reload.
const viewUser = computed<UserDetailed | null>(() => {
	const current = user.value;
	if (!current || !isSelf.value || !session.me) return current;
	return {
		...current,
		name: session.me.name,
		avatarUrl: session.me.avatarUrl,
		description: session.me.description,
	};
});

const displayName = computed(() => viewUser.value?.name || viewUser.value?.username || "");
const headerTitle = computed(() => displayName.value || props.username);

usePageTitle(() => (displayName.value ? `${displayName.value} (@${props.username})` : ""));

const tab = ref<Tab>("home");

const renderedDescription = computed(() => renderMFM(viewUser.value?.description));

const acct = computed(() =>
	viewUser.value?.host
		? `${viewUser.value.username}@${viewUser.value.host}`
		: (viewUser.value?.username ?? props.username),
);

const pinnedNotes = computed(() => viewUser.value?.pinnedNotes ?? []);

const createdAtText = computed(() => {
	const date = viewUser.value ? new Date(viewUser.value.createdAt) : null;
	if (!date || Number.isNaN(date.getTime())) return "";
	return date.toLocaleString("zh-CN");
});

function formatCount(n: number): string {
	return n.toLocaleString("zh-CN");
}

function errorMessage(err: unknown, fallback: string): string {
	if (err instanceof ApiError) return err.message || fallback;
	if (err instanceof Error) return err.message || fallback;
	return fallback;
}

async function load() {
	user.value = null;
	loadError.value = "";
	missing.value = false;
	following.value = false;
	followNotice.value = "";

	if (!props.username) {
		missing.value = true;
		return;
	}

	loading.value = true;
	try {
		const fresh = await api.usersShow({ username: props.username });
		user.value = fresh;
		// The backend resolves the viewer's relationship, so the button starts
		// in the right state instead of assuming "not following".
		following.value = fresh.isFollowing ?? false;
	} catch (err) {
		if (err instanceof ApiError && err.status === 404) missing.value = true;
		else loadError.value = errorMessage(err, "加载用户失败");
	} finally {
		loading.value = false;
	}
}

// following/create answers with the followee's UserLite only (Misskey
// contract), so the relationship after a follow is read back from
// users/show; see useFollowRelation.
const pendingRef = ref(false);
const { refreshRelation } = useFollowRelation(user, following, pendingRef);

async function toggleFollow() {
	if (!user.value || followBusy.value) return;
	followBusy.value = true;
	followNotice.value = "";
	try {
		if (following.value) {
			await api.unfollow(user.value.id);
			following.value = false;
			user.value.followersCount = Math.max(0, user.value.followersCount - 1);
		} else {
			await api.follow(user.value.id);
			following.value = true;
			user.value.followersCount += 1;
		}
		await refreshRelation();
	} catch (err) {
		if (err instanceof ApiError && err.status === 401) {
			followNotice.value = "登录状态已失效，请重新登录后再操作。";
		} else {
			followNotice.value = errorMessage(err, "操作失败");
		}
	} finally {
		followBusy.value = false;
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

watch(
	() => props.username,
	() => {
		tab.value = "home";
		void load();
	},
	{ immediate: true },
);

function onDeleted(id: string) {
	if (!user.value) return;
	user.value.pinnedNotes = user.value.pinnedNotes.filter((note) => note.id !== id);
	user.value.pinnedNoteIds = user.value.pinnedNoteIds.filter((noteId) => noteId !== id);
}

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
});

onBeforeUnmount(() => {
	observer?.disconnect();
	observer = null;
});
</script>

<template>
	<div ref="rootEl" class="user-page">
		<PageHeader
			v-model:tab="(tab as string)"
			:title="headerTitle"
			:subtitle="`@${acct}`"
			:avatar="viewUser"
			:tabs="viewUser ? TABS : null"
			back
			back-to="/"
		>
			<template #actions>
				<template v-if="viewUser">
					<RouterLink
						v-if="isSelf"
						:to="NAV_ITEM_DEFS.settings.to"
						class="button ghost small"
					>
						{{ i18n.ts.editProfile }}
					</RouterLink>
					<FollowButton
						v-else
						:user="viewUser"
						full
						@refresh="toggleFollow"
					/>
				</template>
			</template>
		</PageHeader>

		<p v-if="followNotice" class="notice error page-notice">{{ followNotice }}</p>

		<div v-if="loading" class="skeleton"></div>

		<div v-else-if="missing" class="card placeholder">
			<AppIcon name="globe" />
			<h2>没有找到这个账户</h2>
			<p class="small">账户可能已经被删除或从未存在过；远程账户暂不支持浏览。</p>
			<RouterLink to="/explore" class="button ghost small">去探索页看看</RouterLink>
		</div>

		<MkInfo v-else-if="loadError" warn class="load-error">
			<span>{{ loadError }}</span>
			<button type="button" class="retry" @click="load">重试</button>
		</MkInfo>

		<div v-else-if="viewUser" class="body" :class="{ wide }">
			<div class="main">
				<section class="profile">
					<div class="banner-container">
						<div
							class="banner"
							:style="
								viewUser.bannerUrl
									? { backgroundImage: `url('${viewUser.bannerUrl}')` }
									: undefined
							"
						></div>
						<div class="fade"></div>
						<div class="title">
							<div class="nameColumn">
								<MkUserName class="name" :user="viewUser" :nowrap="true" />
								<span
									v-if="isLoggedIn && session.me && session.me.id !== viewUser.id && viewUser.isFollowed"
									class="followed"
								>
									{{ i18n.ts.followsYou }}
								</span>
							</div>
							<div class="bottom">
								<span class="username">@{{ acct }}</span>
								<span v-if="viewUser.isAdmin" class="badge" title="管理员">
									<AppIcon name="star" />
								</span>
								<span v-if="viewUser.isSuspended" class="badge danger" title="已冻结">
									<AppIcon name="warning" />
								</span>
							</div>
						</div>
					</div>
					<MkAvatar class="avatar" :user="viewUser" :disable-link="true" />
					<div class="title-below">
						<div class="nameColumn">
							<MkUserName class="name" :user="viewUser" :nowrap="true" />
							<span
								v-if="isLoggedIn && session.me && session.me.id !== viewUser.id && viewUser.isFollowed"
								class="followed"
							>
								{{ i18n.ts.followsYou }}
							</span>
						</div>
						<div class="bottom">
							<span class="username">@{{ acct }}</span>
							<span v-if="viewUser.isAdmin" class="badge" title="管理员">
								<AppIcon name="star" />
							</span>
							<span v-if="viewUser.isSuspended" class="badge danger" title="已冻结">
								<AppIcon name="warning" />
							</span>
						</div>
					</div>
					<div class="follow-container">
						<FollowButton
							v-if="!isSelf"
							:user="viewUser"
							full
							large
							@refresh="toggleFollow"
						/>
						<RouterLink v-else :to="NAV_ITEM_DEFS.settings.to" class="button ghost small">
							编辑资料
						</RouterLink>
					</div>
					<div class="description">
						<!-- eslint-disable-next-line vue/no-v-html -->
						<div v-if="viewUser.description" class="mfm" v-html="renderedDescription"></div>
						<p v-else class="empty">还没有简介</p>
					</div>
					<div class="fields system">
						<dl class="field">
							<dt class="name">
								<AppIcon name="clock" class="field-icon" />
								注册时间
							</dt>
							<dd class="value">{{ createdAtText }}</dd>
						</dl>
					</div>
					<div class="status">
						<MkA :to="`/@${viewUser.username}`" :class="{ active: tab === 'home' }">
							<b>{{ formatCount(viewUser.notesCount) }}</b>
							<span>{{ i18n.ts.notes }}</span>
						</MkA>
						<span class="static" :title="i18n.ts.following">
							<b>{{ formatCount(viewUser.followingCount) }}</b>
							<span>{{ i18n.ts.following }}</span>
						</span>
						<span class="static" :title="i18n.ts.followers">
							<b>{{ formatCount(viewUser.followersCount) }}</b>
							<span>{{ i18n.ts.followers }}</span>
						</span>
					</div>
					<p class="lists-note">关注与关注者列表接口尚未提供，这里只显示数量。</p>
				</section>

				<div class="contents">
					<div v-if="pinnedNotes.length > 0" class="pinned">
						<NoteItem
							v-for="note in pinnedNotes"
							:key="note.id"
							:note="note"
							:pinned="true"
							@deleted="onDeleted"
						/>
					</div>
					<MkInfo v-else-if="isSelf" class="pin-tip">
						置顶笔记会显示在个人页顶部，在笔记菜单中选择置顶。
					</MkInfo>

					<template v-if="tab === 'home'">
						<UserPhotos :key="viewUser.id" :user="viewUser" />
					</template>
					<UserTimeline v-else :key="viewUser.id" :user="viewUser" />
				</div>
			</div>
		</div>
	</div>
</template>

<style scoped>
.user-page {
	min-width: 0;
}

.page-notice {
	margin-top: var(--space-12);
}

.skeleton {
	block-size: 220px;
	margin-top: var(--space-16);
	border: 1px solid var(--border);
	border-radius: var(--radius-lg);
	background: var(--panel);
}

.load-error {
	display: flex;
	align-items: center;
	gap: var(--space-10);
	margin-top: var(--space-12);
}

.load-error .retry {
	padding: var(--space-4) var(--space-10);
	border: 1px solid currentcolor;
	border-radius: var(--radius-sm);
	background: transparent;
	color: inherit;
	font: inherit;
	font-size: var(--fs-xs);
	cursor: pointer;
}

.placeholder {
	display: flex;
	flex-direction: column;
	align-items: center;
	gap: var(--space-8);
	margin-top: var(--space-16);
	padding: var(--space-32) var(--space-20);
	text-align: center;
}

.body {
	display: grid;
	grid-template-columns: minmax(0, 1fr);
	margin-top: var(--space-16);
	min-width: 0;
}

.body.wide {
	grid-template-columns: minmax(0, 1fr) 320px;
	align-items: start;
}

.body.wide .main {
	grid-column: 1;
}

.main {
	display: flex;
	flex-direction: column;
	gap: var(--space-16);
	min-width: 0;
}

/* --- profile block ------------------------------------------------------ */
.profile {
	position: relative;
	overflow: hidden;
	border: 1px solid var(--border);
	border-radius: var(--radius-lg);
	background: var(--panel);
}

.banner-container {
	position: relative;
	block-size: 200px;
	border-block-end: 1px solid var(--divider);
	background: var(--surface-inset);
}

.banner {
	position: absolute;
	inset: 0;
	background-size: cover;
	background-position: center;
}

.fade {
	position: absolute;
	inset: 0;
	background: linear-gradient(transparent, rgb(0 0 0 / 0.35));
}

.banner-container .title {
	position: absolute;
	inset-block-end: 10px;
	inset-inline: 16px;
	display: none;
	color: #fff;
	text-shadow: 0 1px 3px rgb(0 0 0 / 0.5);
}

/* The overlaid title only shows on narrow viewports, mirroring the fork. */
.user-page:not(.wide) .banner-container .title {
	display: block;
}

.nameColumn {
	display: flex;
	align-items: center;
	gap: var(--space-8);
	min-width: 0;
}

.nameColumn .name {
	font-size: var(--fs-lg);
	font-weight: var(--weight-bold);
	overflow-wrap: anywhere;
}

.followed {
	padding: 1px var(--space-8);
	border-radius: var(--radius-full);
	background: var(--accent-soft);
	color: var(--accent);
	font-size: var(--fs-2xs);
	font-weight: var(--weight-bold);
}

.banner-container .followed {
	background: rgb(255 255 255 / 0.25);
	color: #fff;
}

.bottom {
	display: flex;
	align-items: center;
	gap: var(--space-6);
	margin-top: 2px;
}

.username {
	color: inherit;
	font-size: var(--fs-xs);
	opacity: 0.85;
	overflow-wrap: anywhere;
}

.banner-container .username {
	color: #fff;
}

.badge {
	display: inline-grid;
	place-items: center;
	color: var(--badge, var(--accent));
}

.badge :deep(.icon) {
	inline-size: 14px;
	block-size: 14px;
}

.badge.danger {
	color: var(--danger);
}

.avatar {
	display: block;
	inline-size: var(--profile-avatar, 84px);
	block-size: var(--profile-avatar, 84px);
	margin: -42px 0 0 var(--space-16);
	border: 3px solid var(--panel);
	border-radius: 50%;
	background: var(--panel);
}

.title-below {
	display: none;
	padding: var(--space-10) var(--space-16) 0;
	color: var(--text);
}

/* Below the banner, the name row replaces the overlay on wide viewports. */
.user-page.wide .title-below {
	display: block;
}

.title-below .nameColumn .name {
	color: var(--text-strong);
}

.follow-container {
	padding: var(--space-12) var(--space-16) 0;
}

.description {
	padding: var(--space-12) var(--space-16) 0;
	color: var(--text);
	font-size: var(--fs-sm);
	line-height: var(--leading-relaxed);
}

.description :deep(.mfm) {
	overflow-wrap: anywhere;
	white-space: pre-wrap;
}

.description .empty {
	margin: 0;
	color: var(--text-weak);
}

.fields.system {
	padding: var(--space-12) var(--space-16) 0;
}

.field {
	display: flex;
	gap: var(--space-8);
	margin: 0 0 var(--space-6);
}

.field .name {
	display: inline-flex;
	align-items: center;
	gap: var(--space-4);
	flex: none;
	color: var(--text-weak);
	font-size: var(--fs-xs);
	font-weight: var(--weight-bold);
}

.field .value {
	margin: 0;
	color: var(--text);
	font-size: var(--fs-xs);
	overflow-wrap: anywhere;
}

.field-icon {
	inline-size: 13px;
	block-size: 13px;
}

.status {
	display: flex;
	gap: var(--space-20);
	padding: var(--space-14) var(--space-16) 0;
}

.status a,
.status .static {
	display: inline-flex;
	align-items: baseline;
	gap: var(--space-4);
	color: var(--text-weak);
	font-size: var(--fs-xs);
}

.status b {
	color: var(--text-strong);
	font-variant-numeric: tabular-nums;
}

.status a.active,
.status a:hover {
	color: var(--accent);
}

.status a.active b,
.status a:hover b {
	color: var(--accent);
}

.lists-note {
	margin: var(--space-8) var(--space-16) var(--space-14);
	color: var(--text-faint);
	font-size: var(--fs-2xs);
	line-height: 1.5;
}

/* --- contents column ---------------------------------------------------- */
.contents {
	display: flex;
	flex-direction: column;
	gap: var(--space-16);
	min-width: 0;
}

.pinned {
	display: flex;
	flex-direction: column;
	gap: 0;
	border: 1px solid var(--border);
	border-radius: var(--radius-lg);
	background: var(--panel);
	overflow: hidden;
}

.pin-tip {
	margin: 0;
}
</style>
