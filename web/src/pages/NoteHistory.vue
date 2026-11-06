<!-- SPDX-License-Identifier: AGPL-3.0-only -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :actions="headerActions" />
		</template>
		<MkSpacer :content-max="600">
			<div class="note-history">
				<MkLoading v-if="loading" />
				<CfInfo v-else-if="errorText != null" warn icon="clock-counter-clockwise">
					{{ errorText }}
				</CfInfo>
				<CfInfo v-else-if="edits.length === 0" warn>
					这条笔记还没有保存过编辑记录。
				</CfInfo>
				<ul v-else class="edits">
					<li v-for="edit in edits" :key="edit.id" class="edit">
						<div class="meta">
							<CfTime :time="edit.createdAt" mode="absolute" class="time" />
							<span class="id _monospace">{{ edit.id }}</span>
						</div>
						<div v-if="edit.cw" class="cw">内容警告：{{ edit.cw }}</div>
						<div v-if="edit.text" class="text">{{ edit.text }}</div>
						<div v-else class="text empty">（无正文）</div>
					</li>
				</ul>

				<CfInfo class="note">
					每条记录是保存前的文本；当前内容请返回笔记页查看。历史只对作者可见。
				</CfInfo>

				<div class="actions">
					<CfButton primary rounded link :to="notePath">
						<i class="ph-note ph-bold ph-lg"></i>
						<span>返回笔记</span>
					</CfButton>
					<CfButton rounded link to="/">
						<i class="ph-house ph-bold ph-lg"></i>
						<span>{{ t("home") }}</span>
					</CfButton>
				</div>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script lang="ts" setup>
// Cinderfish's note history page. `notes/history` is this project's own
// endpoint (Misskey at the pinned commit has none), restricted to the author.
import { computed, onMounted, ref } from "vue";
import { useRoute } from "vue-router";
import { api } from "@/api/client";
import type { NoteEditEntry } from "@/api/types";
import { t } from "@/i18n";
import { definePageMetadata } from "@/scripts/page-metadata";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfTime from "@/components/ui/CfTime.vue";
import MkLoading from "@/components/global/MkLoading.vue";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";

defineOptions({ name: "CfPageNoteHistory" });

const route = useRoute();
const noteId = String(route.params.noteId ?? "");

const loading = ref(true);
const errorText = ref<string | null>(null);
const edits = ref<NoteEditEntry[]>([]);

const notePath = computed(() => `/notes/${noteId}`);
const headerActions = computed(() => []);

async function load(): Promise<void> {
	loading.value = true;
	errorText.value = null;
	try {
		edits.value = await api("notes/history", { noteId, limit: 50 });
	} catch (error) {
		errorText.value =
			error instanceof Error && error.message !== ""
				? `无法读取编辑历史：${error.message}`
				: "无法读取编辑历史。";
	} finally {
		loading.value = false;
	}
}

onMounted(() => {
	void load();
});

definePageMetadata({
	title: "编辑历史",
	icon: "ph-clock-counter-clockwise ph-bold ph-lg",
});
</script>

<style lang="scss" scoped>
.note-history {
	.edits {
		margin: 0;
		padding: 0;
		list-style: none;
	}

	.edit {
		padding: 12px 0;

		&:not(:last-child) {
			border-bottom: 1px solid var(--divider);
		}
	}

	.meta {
		display: flex;
		align-items: baseline;
		gap: 8px;
		margin-block-end: 6px;
		color: var(--fgTransparentWeak);
		font-size: 0.85em;
	}

	.id {
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.cw {
		margin-block-end: 4px;
		font-weight: bold;
	}

	.text {
		white-space: pre-wrap;
		word-break: break-word;

		&.empty {
			color: var(--fgTransparentWeak);
		}
	}

	.note {
		margin-block-start: 1.5em;
		font-size: 0.85em;
	}

	.actions {
		display: flex;
		gap: 8px;
		margin-block-start: 1.5em;
	}
}
</style>
