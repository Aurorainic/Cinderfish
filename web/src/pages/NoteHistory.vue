<!-- SPDX-License-Identifier: AGPL-3.0-only -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :actions="headerActions" />
		</template>
		<MkSpacer :content-max="600">
			<div class="note-history">
				<CfInfo warn icon="clock-counter-clockwise">
					<div class="paragraphs">
						<p class="paragraph">笔记历史版本接口尚未提供。</p>
						<p class="paragraph">
							后端没有注册 <code>notes/history</code> 端点（见
							<code>docs/research/2026-11-04-misskey-api-coverage.md</code>），
							服务器也没有保存编辑前的版本，因此这里无法列出任何历史记录，也不会用假数据填充。
						</p>
					</div>
				</CfInfo>

				<div class="meta _panel">
					<div class="row">
						<span class="label">笔记 ID</span>
						<code class="value">{{ noteId !== "" ? noteId : "（缺少 noteId）" }}</code>
					</div>
				</div>

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
// /notes/:noteId/history. Firefish opens a diff view fed by `notes/history`
// (plus `notes/revisions`); neither endpoint exists on Cinderfish and the note
// store keeps no prior revisions, so the page states that plainly and links
// back to the note instead of inventing a history list (wave-4 contract §2).
import { computed } from "vue";
import { useRoute } from "vue-router";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import { t } from "@/i18n";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageNoteHistory" });

const route = useRoute();

const noteId = computed(() => {
	const raw = route.params.noteId;
	return typeof raw === "string" ? raw : "";
});

const notePath = computed(() =>
	noteId.value === "" ? "/" : `/notes/${encodeURIComponent(noteId.value)}`,
);

const headerActions = computed(() => []);

definePageMetadata({
	title: "笔记历史",
	icon: "ph-clock-counter-clockwise ph-bold ph-lg",
});
</script>

<style lang="scss" scoped>
.note-history {
	display: flex;
	flex-direction: column;
	gap: 16px;
	padding-block: 8px;
}

.paragraphs {
	display: flex;
	flex-direction: column;
	gap: 6px;
	text-align: start;
}

.paragraph {
	margin: 0;
	line-height: 1.7;
}

.meta {
	padding: 16px 20px;

	> .row {
		display: flex;
		align-items: baseline;
		gap: 12px;

		> .label {
			font-size: 0.85em;
			color: var(--fgTransparentWeak);
		}

		> .value {
			font-family: var(--monospace, monospace);
			font-size: 0.9em;
			overflow-wrap: anywhere;
		}
	}
}

.actions {
	display: flex;
	flex-wrap: wrap;
	gap: 8px;
}
</style>
