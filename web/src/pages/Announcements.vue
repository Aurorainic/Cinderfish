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
					<article v-for="item in list" :key="item.id" class="_formBlock notice">
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
						已读状态还没有服务端记录（`i/read-announcement` 未实现），所以这里每次都会显示全部有效公告。
					</CfInfo>
				</template>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script lang="ts" setup>
// Firefish pages `announcements` and marks each read through
// `i/read-announcement`. The list endpoint now exists; read tracking does not,
// so `isRead` is always false and the page says so instead of pretending to
// remember which notices were dismissed.
import { onMounted, ref } from "vue";
import { ApiError, api, resolveMediaUrl } from "@/api/client";
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
		list.value = [...list.value, ...page.slice(0, LIMIT)];
	} catch (cause) {
		error.value = errorText(cause);
	} finally {
		loadingMore.value = false;
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
