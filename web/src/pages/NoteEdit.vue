<!-- SPDX-License-Identifier: AGPL-3.0-only -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :display-back-button="true" />
		</template>
		<MkSpacer :content-max="600" :margin-min="16">
			<div class="_formRoot cf-note-edit">
				<MkLoading v-if="loading" />
				<CfInfo v-else-if="loadError != null" warn>
					{{ loadError }}
				</CfInfo>
				<template v-else>
					<FormTextarea v-model="text" class="_formBlock" tall :max="TEXT_MAX">
						<template #label>{{ t("_postForm._placeholders.a") }}</template>
					</FormTextarea>

					<FormInput v-model="cw" class="_formBlock" :max="CW_MAX">
						<template #label>内容警告</template>
						<template #caption>留空表示不添加内容警告。</template>
					</FormInput>

					<CfSelect v-model="visibility" class="_formBlock">
						<template #label>{{ t("visibility") }}</template>
						<option value="public">{{ t("_visibility.public") }}</option>
						<option value="home">{{ t("_visibility.home") }}</option>
						<option value="followers">{{ t("_visibility.followers") }}</option>
					</CfSelect>

					<CfInfo v-if="saveError != null" warn class="_formBlock">
						{{ saveError }}
					</CfInfo>

					<div class="actions">
						<CfButton primary :disabled="saving" @click="save()">
							{{ t("save") }}
						</CfButton>
						<CfButton :disabled="saving" @click="cancel()">
							{{ t("cancel") }}
						</CfButton>
					</div>

					<CfInfo class="note">
						附件与投票不随编辑变化；远端实例不会收到 Update 活动，
						因此关注者的副本仍是旧文本。历史版本页面可以查看每次保存前的文本。
					</CfInfo>
				</template>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script lang="ts" setup>
// Cinderfish's note editor. Firefish opens an edit dialog from the note menu;
// this port uses a page route so the form does not need a modal component.
// `notes/edit` accepts text, CW, visibility and localOnly only.
import { onMounted, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import { api } from "@/api/client";
import { t } from "@/i18n";
import { definePageMetadata } from "@/scripts/page-metadata";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfSelect from "@/components/ui/CfSelect.vue";
import FormInput from "@/components/form/input.vue";
import FormTextarea from "@/components/form/textarea.vue";
import MkLoading from "@/components/global/MkLoading.vue";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";

defineOptions({ name: "CfPageNoteEdit" });

const TEXT_MAX = 3000;
const CW_MAX = 250;

const route = useRoute();
const router = useRouter();
const noteId = String(route.params.noteId ?? "");

const loading = ref(true);
const loadError = ref<string | null>(null);
const saving = ref(false);
const saveError = ref<string | null>(null);

const text = ref("");
const cw = ref("");
const visibility = ref("public");

function describe(error: unknown, fallback: string): string {
	return error instanceof Error && error.message !== ""
		? error.message
		: fallback;
}

async function load(): Promise<void> {
	loading.value = true;
	loadError.value = null;
	try {
		const note = await api("notes/show", { noteId });
		text.value = note.text ?? "";
		cw.value = note.cw ?? "";
		visibility.value = note.visibility;
	} catch (error) {
		loadError.value = describe(error, "无法读取这条笔记。");
	} finally {
		loading.value = false;
	}
}

async function save(): Promise<void> {
	if (saving.value) return;
	saving.value = true;
	saveError.value = null;
	try {
		await api("notes/edit", {
			editId: noteId,
			text: text.value === "" ? null : text.value,
			cw: cw.value.trim() === "" ? null : cw.value,
			visibility: visibility.value as "public" | "home" | "followers" | "specified",
		});
		await router.push(`/notes/${noteId}`);
	} catch (error) {
		saveError.value = describe(error, "保存失败，请稍后再试。");
	} finally {
		saving.value = false;
	}
}

function cancel(): void {
	void router.push(`/notes/${noteId}`);
}

onMounted(() => {
	void load();
});

definePageMetadata({ title: t("edit"), icon: "ph-pencil-simple ph-bold ph-lg" });
</script>

<style lang="scss" scoped>
.cf-note-edit {
	.actions {
		display: flex;
		gap: 8px;
		margin-block-start: 1.5em;
	}

	.note {
		margin-block-start: 1.5em;
		font-size: 0.85em;
	}
}
</style>
