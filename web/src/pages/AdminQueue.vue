<!-- Ported from firefish@b425f61:packages/client/src/pages/admin/queue.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :display-back-button="true" />
		</template>
		<MkSpacer :content-max="900" :margin-min="16" :margin-max="32">
			<div v-if="!admin" class="cf-admin-queue">
				<CfInfo warn>
					需要管理员权限，无法查看投递队列。
					<CfA to="/" class="_link">返回首页</CfA>
				</CfInfo>
			</div>
			<div v-else class="cf-admin-queue">
				<div class="toolbar">
					<div class="counters">
						<div class="counter">
							<span class="value">{{ info?.pending ?? "-" }}</span>
							<span class="caption">待投递</span>
						</div>
						<div class="counter">
							<span class="value">{{ info?.dueNow ?? "-" }}</span>
							<span class="caption">已到期</span>
						</div>
						<div class="counter" :class="{ bad: (info?.failed ?? 0) > 0 }">
							<span class="value">{{ info?.failed ?? "-" }}</span>
							<span class="caption">超过重试上限</span>
						</div>
					</div>
					<CfButton :disabled="loading" @click="load">
						<i class="ph-arrows-clockwise ph-bold"></i>
						刷新
					</CfButton>
				</div>

				<CfLoading v-if="loading && info == null" />
				<MkError v-else-if="error !== null && info == null" @retry="load" />
				<template v-else-if="info != null">
					<p v-if="info.oldestNextAt !== null" class="oldest">
						最早待投递时间：<MkTime :time="info.oldestNextAt" mode="absolute" />
					</p>

					<CfInfo v-if="info.recent.length === 0">
						队列为空。新的远程投递会先进入 PostgreSQL，再由后台 worker 发出。
					</CfInfo>
					<ul v-else class="entries">
						<li v-for="entry in info.recent" :key="entry.id" class="entry">
							<div class="row">
								<span class="host _monospace">{{ entry.host || "（无效 URL）" }}</span>
								<span class="attempts" :class="{ bad: entry.attempts >= 12 }"
									>尝试 {{ entry.attempts }} 次</span
								>
								<MkTime
									class="next"
									:time="entry.nextAttemptAt"
									mode="absolute"
								/>
							</div>
							<p v-if="entry.lastError !== null" class="error">
								{{ entry.lastError }}
							</p>
						</li>
					</ul>

					<CfInfo class="notice">
						队列是 <code>deliver_queue</code> 表：失败按退避重试，<code>attempts ≥ 12</code>
						后 worker 不再取用，计数显示在「超过重试上限」。本页只读，没有手动重试或清空。
					</CfInfo>
				</template>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script setup lang="ts">
// Firefish's queue page talks to a Bull dashboard. Cinderfish has no Redis by
// default and uses a PostgreSQL table for outbound delivery, so this page shows
// that table's health instead of pretending to speak Bull's API.
import { computed, onMounted, ref } from "vue";
import { ApiError, api } from "@/api/client";
import type { DeliverQueueInfo } from "@/api/types";
import { isAdmin } from "@/account";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";
import MkTime from "@/components/global/MkTime.vue";
import MkError from "@/components/global/MkError.vue";
import CfA from "@/components/ui/CfA.vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageAdminQueue" });

const admin = computed<boolean>(() => isAdmin());
const info = ref<DeliverQueueInfo | null>(null);
const loading = ref(false);
const error = ref<string | null>(null);

async function load(): Promise<void> {
	if (!admin.value) return;
	loading.value = true;
	error.value = null;
	try {
		info.value = await api("admin/deliver-queue", {});
	} catch (cause) {
		error.value = cause instanceof ApiError ? cause.message : String(cause);
	} finally {
		loading.value = false;
	}
}

onMounted(() => {
	void load();
});

definePageMetadata({
	title: "投递队列",
	icon: "ph-tray ph-bold ph-lg",
});
</script>

<style lang="scss" scoped>
.toolbar {
	display: flex;
	align-items: center;
	justify-content: space-between;
	gap: 12px;
	margin-bottom: 12px;
	padding: 14px 16px;
	background: var(--panel);
	border-radius: var(--radius);
}

.counters {
	display: flex;
	gap: 24px;
}

.counter {
	display: flex;
	flex-direction: column;

	.value {
		font-size: 1.3em;
		font-weight: 700;
	}

	.caption {
		font-size: 0.78em;
		opacity: 0.65;
	}

	&.bad .value {
		color: var(--error, #c33);
	}
}

.oldest {
	margin: 0 0 12px;
	font-size: 0.85em;
	opacity: 0.7;
}

.entries {
	list-style: none;
	margin: 0;
	padding: 0;
	display: flex;
	flex-direction: column;
	gap: 6px;
}

.entry {
	padding: 8px 12px;
	background: var(--panel);
	border-radius: var(--radius);

	.row {
		display: flex;
		gap: 12px;
		flex-wrap: wrap;
		align-items: baseline;
		font-size: 0.85em;
	}

	.host {
		flex: 1;
		min-width: 0;
		overflow-wrap: anywhere;
	}

	.attempts,
	.next {
		opacity: 0.65;
		white-space: nowrap;
	}

	.attempts.bad {
		color: var(--error, #c33);
		opacity: 1;
	}

	.error {
		margin: 4px 0 0;
		font-size: 0.78em;
		opacity: 0.7;
		overflow-wrap: anywhere;
	}
}

.notice {
	margin-top: 16px;
}
</style>
