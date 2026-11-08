<!-- Ported from firefish@b425f61:packages/client/src/pages/settings/custom-katex-macro.vue (AGPL-3.0-only) -->
<template>
	<div class="_formRoot cf-settings-katex-macro">
		<CfInfo class="note">{{ t("customKaTeXMacroDescription") }}</CfInfo>

		<CfTextarea
			v-model="macroText"
			code
			tall
			class="field"
			:placeholder="'\\newcommand{\\R}{\\mathbb{R}}'"
		>
			<template #label>{{ t("customKaTeXMacro") }}</template>
		</CfTextarea>

		<CfSwitch :model-value="enabled" @update:model-value="onToggle">
			{{ t("enableCustomKaTeXMacro") }}
		</CfSwitch>

		<CfInfo warn class="note">
			宏只做纯文本展开：Cinderfish 没有引入 katex，公式在本站以代码显示，
			但展开后的内容会随笔记发出去，Misskey / Firefish 客户端能看到排版后的公式。
		</CfInfo>

		<div class="_row">
			<CfButton :disabled="!dirty" primary @click="apply()">
				<i class="ph-floppy-disk ph-bold ph-lg"></i>
				{{ t("save") }}
			</CfButton>
			<CfButton @click="reloadAsk()">
				<i class="ph-arrow-clockwise ph-bold ph-lg"></i>
				重新载入
			</CfButton>
		</div>
		<CfInfo v-if="notice != null" class="note">{{ notice }}</CfInfo>
	</div>
</template>

<script lang="ts" setup>
// Firefish stores the macro source plus its parsed form in localStorage and
// reloads through unisonReload after every edit. The port keeps the storage
// keys (preprocess.ts reads them) and replaces unisonReload with a plain
// reload the user confirms once.
import { computed, ref } from "vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfSwitch from "@/components/ui/CfSwitch.vue";
import CfTextarea from "@/components/ui/CfTextarea.vue";
import { t } from "@/i18n";
import { confirm } from "@/os";
import { definePageMetadata } from "@/scripts/page-metadata";
import { parseKaTeXMacros } from "@/scripts/katex-macro";
import { defaultStore } from "@/store";

defineOptions({ name: "CfPageSettingsCustomKatexMacro" });

const stored = localStorage.getItem("customKaTeXMacro") ?? "";
const macroText = ref(stored);
const saved = ref(stored);
const notice = ref<string | null>(null);

const dirty = computed(() => macroText.value !== saved.value);

const enabled = computed<boolean>(() => defaultStore.state.enableCustomKaTeXMacro);

function onToggle(value: boolean | null): void {
	if (value == null) return;
	defaultStore.set("enableCustomKaTeXMacro", value);
}

function apply(): void {
	localStorage.setItem("customKaTeXMacro", macroText.value);
	try {
		localStorage.setItem(
			"customKaTeXMacroParsed",
			parseKaTeXMacros(macroText.value),
		);
	} catch (cause) {
		notice.value = `保存了原文，但解析失败：${String(cause)}`;
		return;
	}
	saved.value = macroText.value;
	notice.value = t("saved");
}

async function reloadAsk(): Promise<void> {
	const { canceled } = await confirm({
		type: "info",
		text: t("reloadToApplySetting"),
	});
	if (canceled) return;
	location.reload();
}

definePageMetadata({
	title: t("customKaTeXMacro"),
	icon: "ph-radical ph-bold ph-lg",
});
</script>

<style lang="scss" scoped>
.cf-settings-katex-macro {
	.note {
		margin-bottom: 12px;
	}

	.field {
		margin-bottom: 12px;
	}
}
</style>
