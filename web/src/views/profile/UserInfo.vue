// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/components/MkUserInfo.vue
//
// Adaptations: Mfm renders through lib/mfm.ts (plain HTML), fields are hidden
// (the backend keeps UserDetailed.fields empty), and the follow button is the
// ported profile/FollowButton instead of the full MkFollowButton menu runtime.
<script lang="ts" setup>
import { computed, ref } from "vue";
import type { UserDetailed } from "../../api/types";
import MkA from "../../components/ui/MkA.vue";
import MkAvatar from "../../components/ui/MkAvatar.vue";
import MkUserName from "../../components/ui/MkUserName.vue";
import { renderMFM } from "../../lib/mfm";
import { i18n } from "../../i18n";
import { isSignedIn, me } from "../../me";
import FollowButton from "./FollowButton.vue";

const props = defineProps<{
	user: UserDetailed;
	detailed?: boolean;
}>();

const SHOW_MORE_THRESHOLD_LINES = 9;
const SHOW_MORE_THRESHOLD_CHARS = 400;

const isLong = ref(
	!!props.detailed &&
		!!props.user.description &&
		(props.user.description.split("\n").length > SHOW_MORE_THRESHOLD_LINES ||
			props.user.description.length > SHOW_MORE_THRESHOLD_CHARS),
);
const collapsed = ref(isLong.value);

const renderedDescription = computed(() => renderMFM(props.user.description));

const acct = computed(() =>
	props.user.host ? `${props.user.username}@${props.user.host}` : props.user.username,
);

const createdAt = computed(() => {
	const date = new Date(props.user.createdAt);
	if (Number.isNaN(date.getTime())) return "";
	return `${date.getFullYear()} 年 ${date.getMonth() + 1} 月加入`;
});

function number(value: number): string {
	return value.toLocaleString("zh-CN");
}
</script>

<template>
	<article class="user-card">
		<div
			class="banner"
			:class="{ detailed }"
			:style="
				user.bannerUrl ? { backgroundImage: `url('${user.bannerUrl}')` } : undefined
			"
		></div>
		<span
			v-if="isSignedIn(me) && me.id !== user.id && user.isFollowed"
			class="followed"
		>
			{{ i18n.ts.followsYou }}
		</span>
		<h3 class="title">
			<MkAvatar class="avatar" :user="user" />
			<MkA class="name" :to="`/@${user.username}`">
				<MkUserName :user="user" :nowrap="true" />
			</MkA>
			<p class="username">@{{ acct }}</p>
		</h3>
		<div class="description" :class="{ collapsed: isLong && collapsed }">
			<!-- eslint-disable-next-line vue/no-v-html -->
			<div v-if="user.description" class="mfm" v-html="renderedDescription"></div>
			<span v-else class="empty">还没有简介</span>
		</div>
		<button v-if="isLong" type="button" class="show-more" @click="collapsed = !collapsed">
			{{ collapsed ? i18n.ts.showMore : i18n.ts.showLess }}
		</button>
		<div class="status">
			<p>
				<b>{{ number(user.notesCount) }}</b>
				{{ i18n.ts.notes }}
			</p>
			<p>
				<b>{{ number(user.followingCount) }}</b>
				{{ i18n.ts.following }}
			</p>
			<p>
				<b>{{ number(user.followersCount) }}</b>
				{{ i18n.ts.followers }}
			</p>
		</div>
		<div class="buttons">
			<slot>
				<FollowButton v-if="isSignedIn(me) && user.id !== me.id" :user="user" />
			</slot>
		</div>
		<p v-if="createdAt" class="joined">{{ createdAt }}</p>
	</article>
</template>

<style scoped>
.user-card {
	position: relative;
	display: flex;
	flex-direction: column;
	gap: 12px;
	padding-bottom: 16px;
	border: 1px solid var(--border);
	border-radius: var(--radius-lg);
	background: var(--panel);
	overflow: hidden;
}

.banner {
	block-size: 84px;
	background-color: rgb(0 0 0 / 0.1);
	background-position: center;
	background-size: cover;
}

.banner.detailed {
	block-size: 120px;
}

/* MkUserInfo's "follows you" chip sits over the banner in the accent colour. */
.followed {
	position: absolute;
	inset-block-start: 8px;
	inset-inline-start: 8px;
	padding: 2px 8px;
	border-radius: 6px;
	background: var(--accent);
	color: var(--fgOnAccent);
	font-size: 0.75em;
	font-weight: bold;
}

.title {
	display: flex;
	flex-direction: column;
	align-items: flex-start;
	gap: 2px;
	margin: -28px 16px 0;
}

.title :deep(.cinderfish-avatar) {
	--avatar-size: 56px;
	border: 3px solid var(--panel);
}

.name {
	color: var(--fgHighlighted);
	font-size: 1em;
	font-weight: bold;
}

.username {
	margin: 0;
	color: var(--text-weak);
	font-size: 0.8em;
	overflow-wrap: anywhere;
}

.description {
	padding: 0 16px;
	color: var(--text);
	font-size: 0.8em;
	line-height: 1.5;
}

.description :deep(.mfm) {
	overflow-wrap: anywhere;
	white-space: pre-wrap;
}

.description.collapsed {
	display: -webkit-box;
	-webkit-box-orient: vertical;
	-webkit-line-clamp: 3;
	overflow: hidden;
}

.description .empty {
	opacity: 0.7;
}

.show-more {
	margin: 4px 16px 0;
	padding: 4px 8px;
	border: 0;
	border-radius: 6px;
	background: transparent;
	color: var(--accent);
	font: inherit;
	font-size: 0.85em;
	text-align: start;
	cursor: pointer;
}

.show-more:hover {
	background: var(--surface-hover);
}

.status {
	display: flex;
	gap: 16px;
	padding: 0 16px;
	color: var(--text-weak);
	font-size: 0.8em;
}

.status p {
	display: inline-flex;
	align-items: baseline;
	gap: 4px;
	margin: 0;
}

/* MkUserInfo paints the count itself; the label keeps the muted colour. */
.status b {
	color: var(--accent);
	font-weight: bold;
	font-variant-numeric: tabular-nums;
}

.buttons {
	display: flex;
	justify-content: flex-end;
	padding: 0 16px;
}

.joined {
	margin: 0 16px;
	color: var(--text-faint);
	font-size: 0.75em;
}
</style>
