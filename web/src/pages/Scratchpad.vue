<!-- Ported from firefish@b425f61:packages/client/src/pages/scratchpad.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :actions="headerActions" />
		</template>
		<MkSpacer :content-max="800">
			<div class="scratchpad">
				<CfInfo class="notice" icon="info">
					AiScript 解释器（<code>@syuilo/aiscript</code>）不在 Cinderfish
					的依赖集合里（第三波施工契约 §2），所以本页不执行脚本，只作为本地草稿本：内容保存在本机
					localStorage 的 <code>scratchpad</code> 键下，不会上传到服务器。
				</CfInfo>

				<div class="editor _panel">
					<CfTextarea
						v-model="draft"
						code
						tall
						placeholder="在这里记录草稿…"
					>
						<template #label>草稿</template>
					</CfTextarea>
					<div class="bar">
						<span class="count">{{ charCount }} 字符</span>
						<span class="saved">{{ savedLabel }}</span>
						<div class="actions">
							<CfButton mini @click="copy()">
								<i class="ph-clipboard-text ph-bold ph-lg"></i>
								{{ t("copyContent") }}
							</CfButton>
							<CfButton mini danger @click="clear()">
								<i class="ph-trash ph-bold ph-lg"></i>
								{{ t("clear") }}
							</CfButton>
						</div>
					</div>
				</div>

				<p v-if="draft === ''" class="empty">
					草稿为空。输入的任何内容都会自动保存在本机，刷新页面后仍然存在。
				</p>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script lang="ts" setup>
// Firefish's scratchpad is an AiScript console: a Prism editor plus an AiScript
// interpreter that can call the API with the signed-in token. Neither
// `@syuilo/aiscript` nor the Prism editor is in the dependency set (wave-3
// contract §2, "不引入任何新依赖"), so the page is degraded to a local draft
// pad (wave-3 task definition). It keeps Firefish's storage key so an existing
// draft from the upstream client is still shown, and never reads the token.
import { computed, onMounted, ref, watch } from "vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfTextarea from "@/components/ui/CfTextarea.vue";
import copyToClipboard from "@/scripts/copy-to-clipboard";
import { t } from "@/i18n";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageScratchpad" });

const STORAGE_KEY = "scratchpad";

const draft = ref("");
const savedAt = ref<number | null>(null);

const charCount = computed(() => Array.from(draft.value).length);

const savedLabel = computed(() => {
	if (savedAt.value == null) return "尚未保存";
	const date = new Date(savedAt.value);
	const time = `${String(date.getHours()).padStart(2, "0")}:${String(
		date.getMinutes(),
	).padStart(2, "0")}`;
	return `已保存到本机 ${time}`;
});

function readDraft(): void {
	try {
		const saved = localStorage.getItem(STORAGE_KEY);
		if (saved != null) draft.value = saved;
	} catch {
		// Private mode: the draft simply starts empty.
	}
}

function writeDraft(): void {
	try {
		if (draft.value === "") localStorage.removeItem(STORAGE_KEY);
		else localStorage.setItem(STORAGE_KEY, draft.value);
		savedAt.value = Date.now();
	} catch {
		// Quota / private mode: keep the in-memory draft only.
	}
}

let timer: number | null = null;

watch(draft, () => {
	if (timer != null) window.clearTimeout(timer);
	timer = window.setTimeout(() => {
		writeDraft();
	}, 500);
});

async function copy(): Promise<void> {
	if (draft.value === "") return;
	await copyToClipboard(draft.value);
}

function clear(): void {
	draft.value = "";
	writeDraft();
}

const headerActions = computed(() => []);

onMounted(() => {
	readDraft();
});

definePageMetadata({
	title: t("scratchpad"),
	icon: "ph-terminal-window ph-bold ph-lg",
});
</script>

<style lang="scss" scoped>
.scratchpad {
	padding: 16px;
	display: flex;
	flex-direction: column;
	gap: 16px;

	> .notice {
		display: block;
	}
}

.editor {
	padding: 16px;

	> .bar {
		display: flex;
		align-items: center;
		flex-wrap: wrap;
		gap: 12px;
		padding-block-start: 12px;

		> .count,
		> .saved {
			font-size: 0.85em;
			color: var(--fgTransparentWeak);
		}

		> .actions {
			margin-inline-start: auto;
			display: flex;
			gap: 8px;
		}
	}
}

.empty {
	margin: 0;
	font-size: 0.9em;
	color: var(--fgTransparentWeak);
}
</style>
