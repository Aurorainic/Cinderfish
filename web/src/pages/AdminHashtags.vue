<!-- Ported from firefish@b425f61:packages/client/src/pages/admin/hashtags.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :display-back-button="true" />
		</template>
		<MkSpacer :content-max="700" :margin-min="16" :margin-max="32">
			<div v-if="!admin" class="cf-admin-hashtags">
				<CfInfo warn>
					需要管理员权限，无法修改隐藏标签。
					<CfA to="/" class="_link">返回首页</CfA>
				</CfInfo>
			</div>
			<div v-else class="cf-admin-hashtags">
				<CfLoading v-if="loading" />
				<CfInfo v-else-if="loadError !== null" warn>
					{{ loadError }}
					<CfButton inline @click="load()">{{ t("retry") }}</CfButton>
				</CfInfo>
				<form v-else class="_formRoot" @submit.prevent="save">
					<CfTextarea
						v-model="hiddenTags"
						class="_formBlock"
						tall
						:disabled="saving"
					>
						<template #label>{{ t("hiddenTags") }}</template>
						<template #caption>
							{{ t("hiddenTagsDescription") }}每行一个；保存时会去掉开头的
							<code>#</code>、转为小写并去重。
						</template>
					</CfTextarea>
					<CfInfo v-if="saveError !== null" warn class="_formBlock">
						{{ saveError }}
					</CfInfo>
					<CfButton type="submit" primary :disabled="saving">
						{{ saving ? "保存中…" : t("save") }}
					</CfButton>
				</form>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script lang="ts" setup>
// Firefish's admin hashtags page: `admin/meta` returns hiddenTags and
// `admin/update-meta` replaces them. Cinderfish applies the list to
// hashtags/trend only (see docs/research/api-contract/timelines-notifications.md).
import { onMounted, ref } from "vue";
import { isAdmin } from "@/account";
import { api } from "@/api/client";
import type { AdminInstanceMeta } from "@/api/types";
import CfA from "@/components/ui/CfA.vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import CfTextarea from "@/components/ui/CfTextarea.vue";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";
import { t } from "@/i18n";
import { success } from "@/os";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageAdminHashtags" });

const admin = isAdmin();
const loading = ref(true);
const loadError = ref<string | null>(null);
const saving = ref(false);
const saveError = ref<string | null>(null);
const hiddenTags = ref("");

function describe(error: unknown, fallback: string): string {
	return error instanceof Error && error.message !== ""
		? error.message
		: fallback;
}

async function load(): Promise<void> {
	loading.value = true;
	loadError.value = null;
	try {
		const meta = await api("admin/meta");
		hiddenTags.value = ((meta as AdminInstanceMeta).hiddenTags ?? []).join("\n");
	} catch (error) {
		loadError.value = describe(error, "无法读取隐藏标签。");
	} finally {
		loading.value = false;
	}
}

async function save(): Promise<void> {
	if (saving.value) return;
	saving.value = true;
	saveError.value = null;
	try {
		const tags = hiddenTags.value
			.split("\n")
			.map((line) => line.trim())
			.filter((line) => line !== "");
		await api("admin/update-meta", { hiddenTags: tags });
		success();
	} catch (error) {
		saveError.value = describe(error, "保存失败，请稍后再试。");
	} finally {
		saving.value = false;
	}
}

onMounted(() => {
	if (admin) void load();
	else loading.value = false;
});

definePageMetadata({
	title: t("hiddenTags"),
	icon: "ph-hash ph-bold ph-lg",
});
</script>

<style lang="scss" scoped>
.cf-admin-hashtags {
	code {
		font-family: var(--monospace, monospace);
	}
}
</style>
