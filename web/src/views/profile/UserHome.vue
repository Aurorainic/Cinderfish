// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/pages/user/home.vue
// (profile block + contents column) and pages/user/index.vue (tab skeleton).
//
// Adaptations versus the original:
// - Remote users are not reachable (users/show has no host support), so the
//   MkRemoteCaution / MkMoved branch is replaced by the unavailable card.
// - Mfm renders through lib/mfm.ts; fields, birthday, location and the
//   city-timezone lookup are dropped (UserDetailed keeps those null/[]).
// - The profile block keeps one text column at every width: the name sits
//   directly under the banner with the avatar overlapping its bottom edge,
//   and the 3:1 banner replaces the original's fixed 250px height. The
//   wide/narrow branch and its sub column are therefore dropped — the reading
//   column never exceeds the fork's 1000px narrow threshold.
// - follow-list tabs stay out of the tab strip: users/following and
//   users/followers do not exist, so only the counts are shown.
// - The reactions tab is dropped: notes/reactions does not exist.
<script lang="ts" setup>
import { computed, ref, watch } from "vue";
import { api, ApiError } from "../../api/client";
import type { UserDetailed } from "../../api/types";
import AppIcon from "../../components/AppIcon.vue";
import MkAvatar from "../../components/ui/MkAvatar.vue";
import MkButton from "../../components/ui/MkButton.vue";
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
// users/show; see useFollowRelation. The button already performs the
// follow/unfollow itself, so the refresh only re-reads the state.
const pendingRef = ref(false);
const { refreshRelation } = useFollowRelation(user, following, pendingRef);

async function onFollowRefresh() {
	await refreshRelation();
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
</script>

<template>
	<div class="user-page">
		<PageHeader
			v-model:tab="(tab as string)"
			:title="headerTitle"
			:subtitle="`@${acct}`"
			:avatar="viewUser"
			:tabs="viewUser ? TABS : null"
			back
			back-to="/"
		/>

		<div v-if="loading" class="skeleton"></div>

		<div v-else-if="missing" class="placeholder">
			<AppIcon name="globe" />
			<h2>没有找到这个账户</h2>
			<p>账户可能已经被删除或从未存在过；远程账户暂不支持浏览。</p>
			<RouterLink to="/explore" class="placeholder-link">去探索页看看</RouterLink>
		</div>

		<MkInfo v-else-if="loadError" warn class="load-error">
			<span>{{ loadError }}</span>
			<button type="button" class="retry" @click="load">重试</button>
		</MkInfo>

		<div v-else-if="viewUser" class="body">
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
					</div>

					<div class="head">
						<MkAvatar class="avatar" :user="viewUser" :disable-link="true" />
						<div class="identity">
							<div class="name-row">
								<MkUserName class="name" :user="viewUser" :nowrap="true" />
								<span
									v-if="isLoggedIn && session.me && session.me.id !== viewUser.id && viewUser.isFollowed"
									class="followed"
								>
									{{ i18n.ts.followsYou }}
								</span>
							</div>
							<div class="acct-row">
								<span class="acct">@{{ acct }}</span>
								<span v-if="viewUser.isAdmin" class="badge" title="管理员">
									<AppIcon name="star" />
								</span>
								<span v-if="viewUser.isSuspended" class="badge danger" title="已冻结">
									<AppIcon name="warning" />
								</span>
							</div>
						</div>
						<div class="actions">
							<MkButton
								v-if="isSelf"
								link
								rounded
								:to="NAV_ITEM_DEFS.settings.to"
							>
								{{ i18n.ts.editProfile }}
							</MkButton>
							<FollowButton v-else :user="viewUser" @refresh="onFollowRefresh" />
						</div>
					</div>

					<!-- Counts only: users/following and users/followers are not
					     implemented, so the follow lists have no page to link to. -->
					<div class="status">
						<div class="item">
							<b>{{ formatCount(viewUser.notesCount) }}</b>
							<span>{{ i18n.ts.notes }}</span>
						</div>
						<div class="item">
							<b>{{ formatCount(viewUser.followingCount) }}</b>
							<span>{{ i18n.ts.following }}</span>
						</div>
						<div class="item">
							<b>{{ formatCount(viewUser.followersCount) }}</b>
							<span>{{ i18n.ts.followers }}</span>
						</div>
					</div>

					<div class="description">
						<!-- eslint-disable-next-line vue/no-v-html -->
						<div v-if="viewUser.description" class="mfm" v-html="renderedDescription"></div>
						<p v-else class="empty">还没有简介</p>
					</div>

					<div class="fields">
						<dl class="field">
							<dt class="name">
								<AppIcon name="clock" class="field-icon" />
								注册时间
							</dt>
							<dd class="value">{{ createdAtText }}</dd>
						</dl>
					</div>
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
						<UserPhotos :key="`photos-${viewUser.id}`" :user="viewUser" />
						<UserTimeline :key="`timeline-${viewUser.id}`" :user="viewUser" />
					</template>
					<UserPhotos v-else :key="`photos-${viewUser.id}`" :user="viewUser" />
				</div>
			</div>
		</div>
	</div>
</template>

<style scoped>
.user-page {
	min-width: 0;
}

.skeleton {
	block-size: 220px;
	margin-block-start: var(--margin);
	border-radius: var(--radius);
	background: var(--panelHighlight);
}

.placeholder {
	display: flex;
	flex-direction: column;
	align-items: center;
	gap: 8px;
	margin-block-start: var(--margin);
	padding: 32px 20px;
	border: 1px solid var(--divider);
	border-radius: var(--radius);
	background: var(--panel);
	text-align: center;
}

.placeholder > .icon {
	inline-size: 40px;
	block-size: 40px;
	color: var(--fgTransparent);
}

.placeholder h2 {
	margin: 0;
	font-size: 1.1em;
	color: var(--fgHighlighted);
}

.placeholder p {
	margin: 0;
	color: var(--fgTransparentWeak);
	font-size: 0.9em;
}

.placeholder-link {
	color: var(--accent);
	font-size: 0.9em;
}

.load-error {
	display: flex;
	align-items: center;
	gap: 10px;
	margin-block-start: var(--margin);
}

.load-error .retry {
	padding: 4px 10px;
	border: 1px solid currentcolor;
	border-radius: 6px;
	background: transparent;
	color: inherit;
	font: inherit;
	font-size: 0.85em;
	cursor: pointer;
}

.body {
	margin-block-start: var(--margin);
	min-width: 0;
}

.main {
	display: flex;
	flex-direction: column;
	gap: var(--margin);
	min-width: 0;
}

/* --- profile block ------------------------------------------------------ */
.profile {
	position: relative;
	overflow: clip;
	border: 1px solid var(--divider);
	border-radius: var(--radius);
	background: var(--panel);
}

/* Firefish sizes the banner by the block height (250px); the port keeps the
   same proportion of roughly 3:1 so it scales with the reading column. */
.banner-container {
	position: relative;
	overflow: clip;
	aspect-ratio: 3 / 1;
	min-block-size: 120px;
	max-block-size: 260px;
	border-block-end: 1px solid var(--divider);
}

/* Accounts without a banner keep the panel colour with Firefish's inset
   shadow, so the strip still reads as a banner rather than a broken image. */
.banner {
	position: absolute;
	inset: 0;
	background-color: var(--panelHighlight);
	background-position: center;
	background-size: cover;
	box-shadow: 0 0 128px var(--shadow) inset;
}

.fade {
	position: absolute;
	inset-block-end: 0;
	inset-inline: 0;
	block-size: 78px;
	background: linear-gradient(transparent, rgb(0 0 0 / 0.45));
	pointer-events: none;
}

.head {
	display: flex;
	align-items: flex-start;
	gap: 16px;
	min-width: 0;
	padding-inline: 20px;
}

/* The avatar is centred on the banner's bottom edge, like the fork's
   `inset-block-start: 170px` against a 250px banner. */
.avatar {
	flex: none;
	--avatar-size: 96px;
	margin-block-start: -48px;
	background: var(--panel);
	box-shadow: 1px 1px 3px rgb(0 0 0 / 0.2);
}

.identity {
	flex: 1 1 auto;
	min-width: 0;
	padding-block-start: 10px;
}

.name-row {
	display: flex;
	align-items: center;
	gap: 8px;
	min-width: 0;
}

.name-row .name {
	min-inline-size: 0;
	overflow: hidden;
	color: var(--fgHighlighted);
	font-size: 1.4em;
	font-weight: bold;
	line-height: 1.25;
	text-overflow: ellipsis;
}

.followed {
	flex: none;
	padding: 2px 8px;
	border-radius: 24px;
	background: var(--accent);
	color: var(--fgOnAccent);
	font-size: 0.75em;
	line-height: 1.5;
	white-space: nowrap;
}

.acct-row {
	display: flex;
	align-items: center;
	gap: 6px;
	min-width: 0;
	margin-block-start: 2px;
	color: var(--fgTransparentWeak);
	font-size: 0.85em;
}

.acct {
	overflow-wrap: anywhere;
}

.badge {
	display: inline-grid;
	place-items: center;
	flex: none;
	color: var(--badge);
}

.badge :deep(.icon) {
	inline-size: 14px;
	block-size: 14px;
}

.badge.danger {
	color: var(--error);
}

.actions {
	display: flex;
	align-items: center;
	flex: none;
	padding-block-start: 8px;
}

.actions > * {
	flex: none;
}

.status {
	display: flex;
	margin-block-start: 12px;
	padding-block: 12px;
	padding-inline: 20px;
	border-block-start: 1px solid var(--divider);
}

.status > .item {
	display: flex;
	flex: 1 1 0;
	flex-direction: column;
	align-items: center;
	gap: 2px;
	min-inline-size: 0;
	text-align: center;
}

.status b {
	color: var(--fgHighlighted);
	font-size: 1.15em;
	font-weight: bold;
	font-variant-numeric: tabular-nums;
	line-height: 1.2;
}

.status .item > span {
	color: var(--fgTransparentWeak);
	font-size: 0.78em;
	line-height: 1.2;
	white-space: nowrap;
}

.description {
	padding: 12px 20px 0;
	color: var(--fg);
	font-size: 0.95em;
	line-height: 1.6;
	overflow-wrap: anywhere;
}

.description :deep(.mfm) {
	overflow-wrap: anywhere;
	white-space: pre-wrap;
}

.description .empty {
	margin: 0;
	color: var(--fgTransparent);
}

.fields {
	margin-block-start: 12px;
	padding: 12px 20px 16px;
	border-block-start: 1px solid var(--divider);
	font-size: 0.85em;
}

.field {
	display: flex;
	align-items: center;
	gap: 8px;
	margin: 0;
}

.field .name {
	display: inline-flex;
	align-items: center;
	flex: none;
	gap: 4px;
	color: var(--fgTransparentWeak);
	font-weight: bold;
}

.field .value {
	margin: 0;
	color: var(--fg);
	overflow-wrap: anywhere;
}

.field-icon {
	inline-size: 13px;
	block-size: 13px;
}

/* --- contents column ---------------------------------------------------- */
.contents {
	display: flex;
	flex-direction: column;
	gap: var(--margin);
	min-width: 0;
}

.pinned {
	display: flex;
	flex-direction: column;
	border: 1px solid var(--divider);
	border-radius: var(--radius);
	background: var(--panel);
	overflow: hidden;
}

.pin-tip {
	margin: 0;
}

@media (max-width: 500px) {
	.head {
		flex-wrap: wrap;
		gap: 10px 12px;
		padding-inline: 16px;
	}

	.actions {
		inline-size: 100%;
		justify-content: flex-end;
		padding-block-start: 0;
	}

	.avatar {
		--avatar-size: 76px;
		margin-block-start: -38px;
	}

	.name-row .name {
		font-size: 1.2em;
	}

	.status,
	.description,
	.fields {
		padding-inline: 16px;
	}
}
</style>
