<!-- Ported from firefish@b425f61:packages/client/src/pages/announcements.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :display-back-button="true" />
		</template>
		<MkSpacer :content-max="800" :margin-min="16" :margin-max="32">
			<div class="announcements _formRoot">
				<CfLoading v-if="loading && list.length === 0" />
				<MkError v-else-if="error !== null && list.length === 0" @retry="load" />
				<div v-else-if="list.length === 0" class="empty _formBlock">
					<i class="ph-megaphone-simple ph-bold ph-lg icon"></i>
					<p class="title">{{ t("announcements") }}</p>
					<p class="caption">当前没有可显示的公告。</p>
				</div>
				<template v-else>
					<article
					v-for="item in list"
					:key="item.id"
					class="_formBlock notice"
					:class="{ unread: !item.isRead }"
				>
						<header class="head">
							<h2 class="title">{{ item.title }}</h2>
							<MkTime class="date" :time="item.createdAt" mode="absolute" />
						</header>
						<img
							v-if="item.imageUrl !== null"
							class="image"
							:src="resolveMediaUrl(item.imageUrl)"
							alt=""
						/>
						<p class="text">{{ item.text }}</p>
						<p v-if="item.expiresAt !== null" class="expiry">
							有效至 <MkTime :time="item.expiresAt" mode="absolute" />
						</p>
					</article>
					<div v-if="hasMore" class="more">
						<CfButton v-if="!loadingMore" @click="loadMore">
							{{ t("loadMore") }}
						</CfButton>
						<CfLoading v-else mini />
					</div>
					<CfInfo class="readNote">
						打开本页会把已显示的公告标记为已读（`i/read-announcement`），
						未读的公告带左侧强调线。
					</CfInfo>
				</template>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script lang="ts" setup>
// Firefish pages `announcements` and marks each read through
// `i/read-announcement`. The port marks every notice it displays; the service
// records the mark, so the list and `hasUnreadAnnouncement` stay in sync.
import { onMounted, ref } from "vue";
import { ApiError, api, resolveMediaUrl } from "@/api/client";
import { signedIn, updateMe } from "@/account";
import type { Announcement } from "@/api/types";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";
import MkTime from "@/components/global/MkTime.vue";
import MkError from "@/components/global/MkError.vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import { t } from "@/i18n";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageAnnouncements" });

const LIMIT = 10;

const list = ref<Announcement[]>([]);
const loading = ref(false);
const loadingMore = ref(false);
const hasMore = ref(false);
const error = ref<string | null>(null);

function errorText(cause: unknown): string {
	return cause instanceof ApiError ? cause.message : String(cause);
}

async function load(): Promise<void> {
	loading.value = true;
	error.value = null;
	try {
		const page = await api("announcements", { limit: LIMIT + 1 });
		hasMore.value = page.length > LIMIT;
		list.value = page.slice(0, LIMIT);
		void markRead(list.value);
	} catch (cause) {
		error.value = errorText(cause);
		list.value = [];
	} finally {
		loading.value = false;
	}
}

async function loadMore(): Promise<void> {
	const last = list.value[list.value.length - 1];
	if (last == null || loadingMore.value) return;
	loadingMore.value = true;
	try {
		const page = await api("announcements", {
			limit: LIMIT + 1,
			untilId: last.id,
		});
		hasMore.value = page.length > LIMIT;
		const added = page.slice(0, LIMIT);
		list.value = [...list.value, ...added];
		void markRead(added);
	} catch (cause) {
		error.value = errorText(cause);
	} finally {
		loadingMore.value = false;
	}
}

async function markRead(items: Announcement[]): Promise<void> {
	// i/read-announcement needs a credential; a visitor sees the list but
	// nothing is marked, and the unread badge is not theirs to clear anyway.
	if (!signedIn.value) return;
	for (const item of items) {
		if (item.isRead) continue;
		try {
			await api("i/read-announcement", { announcementId: item.id });
			item.isRead = true;
		} catch {
			// Leave it unread; the next visit tries again.
		}
	}
	if (!hasMore.value && list.value.every((item) => item.isRead)) {
		updateMe({ hasUnreadAnnouncement: false });
	}
}

onMounted(() => {
	void load();
});

definePageMetadata({
	title: t("announcements"),
	icon: "ph-megaphone-simple ph-bold ph-lg",
});
</script>

<style lang="scss" scoped>
.announcements {
	padding: 16px;
}

.empty {
	display: flex;
	flex-direction: column;
	align-items: center;
	gap: 8px;
	padding: 48px 16px;
	border-radius: var(--radius);
	background: var(--panel);
	text-align: center;

	> .icon {
		font-size: 32px;
		opacity: 0.5;
	}

	> .title {
		margin: 0;
		font-weight: bold;
	}

	> .caption {
		margin: 0;
		font-size: 0.9em;
		color: var(--fgTransparentWeak);
	}
}

.notice {
	padding: 16px;
	background: var(--panel);
	border-radius: var(--radius);
	border-inline-start: 3px solid transparent;

	&.unread {
		border-inline-start-color: var(--accent);
	}

	.head {
		display: flex;
		align-items: baseline;
		justify-content: space-between;
		gap: 12px;
	}

	.title {
		margin: 0;
		font-size: 1.05em;
	}

	.date {
		flex: none;
		font-size: 0.8em;
		opacity: 0.6;
	}

	.image {
		display: block;
		max-width: 100%;
		margin-top: 10px;
		border-radius: var(--radius);
	}

	.text {
		margin: 10px 0 0;
		white-space: pre-wrap;
		overflow-wrap: anywhere;
	}

	.expiry {
		margin: 8px 0 0;
		font-size: 0.8em;
		opacity: 0.6;
	}
}

.more {
	display: flex;
	justify-content: center;
	padding: 12px 0;
}

.readNote {
	margin-top: 12px;
	font-size: 0.85em;
}
</style>
