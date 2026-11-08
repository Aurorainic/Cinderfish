<!-- Ported from firefish@b425f61:packages/client/src/pages/admin/settings.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :display-back-button="true" />
		</template>
		<MkSpacer :content-max="700" :margin-min="16" :margin-max="32">
			<div v-if="!admin" class="cf-admin-settings">
				<CfInfo warn>
					需要管理员权限，无法修改实例设置。
					<CfA to="/" class="_link">返回首页</CfA>
				</CfInfo>
			</div>
			<div v-else class="cf-admin-settings">
				<CfLoading v-if="loading" />
				<MkError v-else-if="loadError !== null" @retry="load" />
				<form v-else class="_formRoot" @submit.prevent="save">
					<CfInput v-model="name" class="_formBlock" :disabled="saving">
						<template #label>实例名称</template>
						<template #caption>显示在首页、页签与 ActivityPub 元数据里。</template>
					</CfInput>
					<CfTextarea
						v-model="description"
						class="_formBlock"
						tall
						:disabled="saving"
					>
						<template #label>简介</template>
						<template #caption>最多 500 个字符，显示在欢迎页。</template>
					</CfTextarea>
					<CfInput v-model="maintainerName" class="_formBlock" :disabled="saving">
						<template #label>维护者名称</template>
					</CfInput>
					<CfInput v-model="iconUrl" class="_formBlock" type="url" :disabled="saving">
						<template #label>实例图标 URL</template>
						<template #caption>方形图片，≤300 字符；留空使用默认图标。</template>
					</CfInput>
					<CfInput v-model="bannerUrl" class="_formBlock" type="url" :disabled="saving">
						<template #label>实例横幅 URL</template>
						<template #caption>显示在欢迎页等位置；留空表示没有横幅。</template>
					</CfInput>
					<CfInput v-model="themeColor" class="_formBlock" :disabled="saving">
						<template #label>主题色</template>
						<template #caption><code>#rrggbb</code>；留空使用默认的 <code>#86b300</code>。</template>
					</CfInput>
					<CfInput
						v-model="maintainerEmail"
						class="_formBlock"
						type="email"
						:disabled="saving"
					>
						<template #label>联系邮箱</template>
					</CfInput>
					<CfSelect
						v-model="registrationMode"
						class="_formBlock"
						:disabled="saving"
					>
						<template #label>注册模式</template>
						<option value="open">开放注册</option>
						<option value="invite">仅邀请码</option>
						<option value="admin">仅管理员创建</option>
						<template #caption>
							切换立即生效：开放注册对新用户开放，仅邀请码需要邀请码，仅管理员创建时只能由管理台添加账号。
						</template>
					</CfSelect>

					<CfTextarea
						v-model="preservedUsernamesText"
						class="_formBlock"
						tall
						:disabled="saving"
					>
						<template #label>保留用户名（每行一个）</template>
						<template #caption
							>注册和「用户名是否可用」检查会拒绝这些名字（不区分大小写，最多 200 个，每个 1–20 个字符）。管理员在管理台创建账号时不受限制。</template
						>
					</CfTextarea>
					<CfInput
						v-model="maxNoteTextLength"
						class="_formBlock"
						type="number"
						:disabled="saving"
					>
						<template #label>笔记字数上限</template>
						<template #caption
							>100–10000 个字符（按 Unicode 码点计，中文与 emoji 各算一个）。默认 3000，与 Misskey 一致；公开 meta 会同步这个值。</template
						>
					</CfInput>

					<CfInput v-model="tosUrl" class="_formBlock" type="url" :disabled="saving">
						<template #label>服务条款 URL</template>
						<template #caption>留空表示未提供；显示在关于页。</template>
					</CfInput>
					<CfInput v-model="privacyPolicyUrl" class="_formBlock" type="url" :disabled="saving">
						<template #label>隐私政策 URL</template>
					</CfInput>
					<CfInput v-model="repositoryUrl" class="_formBlock" type="url" :disabled="saving">
						<template #label>源码仓库 URL</template>
					</CfInput>
					<CfInput v-model="feedbackUrl" class="_formBlock" type="url" :disabled="saving">
						<template #label>反馈 URL</template>
					</CfInput>
					<CfInput v-model="inquiryUrl" class="_formBlock" type="url" :disabled="saving">
						<template #label>咨询 URL</template>
					</CfInput>
					<CfInput v-model="impressumUrl" class="_formBlock" type="url" :disabled="saving">
						<template #label>版本说明（Impressum）URL</template>
					</CfInput>
					<CfTextarea
						v-model="rulesText"
						class="_formBlock"
						tall
						:disabled="saving"
					>
						<template #label>实例规则（每行一条）</template>
						<template #caption
							>最多 20 条、每条 300 字；显示在关于页。空行会被忽略。</template
						>
					</CfTextarea>

					<CfInfo v-if="error !== null" warn class="_formBlock">
						{{ error }}
					</CfInfo>
					<CfInfo v-else-if="saved" class="_formBlock">
						已保存。公开页面需要刷新后才能看到新的名称与简介。
					</CfInfo>

					<CfButton type="submit" primary :disabled="saving">
						{{ saving ? "保存中…" : "保存" }}
					</CfButton>
				</form>

				<CfInfo class="notice">
					这里覆盖名称、简介、维护者、注册模式、保留用户名、笔记字数上限、合规链接与实例规则；数据库、媒体、联邦策略等其余设置仍来自
					<code>config.toml</code>，修改后需要重启。上游 <code>admin/update-meta</code> 还有大量字段，
					本实例忽略未列出的字段。
				</CfInfo>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script setup lang="ts">
// Firefish's admin settings page edits the whole meta object. Cinderfish only
// stores four text fields plus the registration mode in `instance_meta`;
// everything else stays in config.toml, and the page says so rather than
// showing controls that would silently do nothing.
import { computed, onMounted, ref } from "vue";
import { ApiError, api } from "@/api/client";
import type { AdminInstanceMeta } from "@/api/types";
import { updateInstance } from "@/instance";
import { isAdmin } from "@/account";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";
import MkError from "@/components/global/MkError.vue";
import CfA from "@/components/ui/CfA.vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfInput from "@/components/ui/CfInput.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import CfSelect from "@/components/ui/CfSelect.vue";
import CfTextarea from "@/components/ui/CfTextarea.vue";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageAdminSettings" });

const admin = computed<boolean>(() => isAdmin());

const name = ref<string | number | null>("");
const description = ref("");
const maintainerName = ref<string | number | null>("");
const maintainerEmail = ref<string | number | null>("");
const iconUrl = ref<string | number | null>("");
const bannerUrl = ref<string | number | null>("");
const themeColor = ref<string | number | null>("");
const registrationMode = ref("open");
const preservedUsernamesText = ref("");
const maxNoteTextLength = ref<string | number | null>(3000);
const tosUrl = ref<string | number | null>("");
const privacyPolicyUrl = ref<string | number | null>("");
const repositoryUrl = ref<string | number | null>("");
const feedbackUrl = ref<string | number | null>("");
const inquiryUrl = ref<string | number | null>("");
const impressumUrl = ref<string | number | null>("");
const rulesText = ref("");
const loading = ref(true);
const loadError = ref<string | null>(null);
const saving = ref(false);
const error = ref<string | null>(null);
const saved = ref(false);

function errorText(cause: unknown): string {
	return cause instanceof ApiError ? cause.message : String(cause);
}

function apply(meta: AdminInstanceMeta): void {
	name.value = meta.name;
	description.value = meta.description;
	maintainerName.value = meta.maintainerName;
	maintainerEmail.value = meta.maintainerEmail;
	iconUrl.value = meta.iconUrl ?? "";
	bannerUrl.value = meta.bannerUrl ?? "";
	themeColor.value = meta.themeColor ?? "";
	registrationMode.value = meta.registrationMode;
	preservedUsernamesText.value = (meta.preservedUsernames ?? []).join("\n");
	maxNoteTextLength.value = meta.maxNoteTextLength ?? 3000;
	tosUrl.value = meta.tosUrl ?? "";
	privacyPolicyUrl.value = meta.privacyPolicyUrl ?? "";
	repositoryUrl.value = meta.repositoryUrl ?? "";
	feedbackUrl.value = meta.feedbackUrl ?? "";
	inquiryUrl.value = meta.inquiryUrl ?? "";
	impressumUrl.value = meta.impressumUrl ?? "";
	rulesText.value = (meta.serverRules ?? []).join("\n");
}

async function load(): Promise<void> {
	if (!admin.value) return;
	loading.value = true;
	loadError.value = null;
	try {
		apply(await api("admin/meta", {}));
	} catch (cause) {
		loadError.value = errorText(cause);
	} finally {
		loading.value = false;
	}
}

async function save(): Promise<void> {
	if (saving.value) return;
	const limit = Number(maxNoteTextLength.value);
	if (!Number.isInteger(limit) || limit < 100 || limit > 10000) {
		error.value = "笔记字数上限必须是 100–10000 之间的整数。";
		return;
	}
	saving.value = true;
	error.value = null;
	saved.value = false;
	try {
		const updated = await api("admin/update-meta", {
			name: String(name.value ?? "").trim(),
			description: description.value.trim(),
			maintainerName: String(maintainerName.value ?? "").trim(),
			maintainerEmail: String(maintainerEmail.value ?? "").trim(),
			iconUrl: String(iconUrl.value ?? "").trim(),
			bannerUrl: String(bannerUrl.value ?? "").trim(),
			themeColor: String(themeColor.value ?? "").trim(),
			registrationMode: registrationMode.value as
				| "open"
				| "invite"
				| "admin",
			preservedUsernames: preservedUsernamesText.value
				.split("\n")
				.map((line) => line.trim())
				.filter((line) => line !== ""),
			maxNoteTextLength: limit,
			tosUrl: String(tosUrl.value ?? "").trim(),
			privacyPolicyUrl: String(privacyPolicyUrl.value ?? "").trim(),
			repositoryUrl: String(repositoryUrl.value ?? "").trim(),
			feedbackUrl: String(feedbackUrl.value ?? "").trim(),
			inquiryUrl: String(inquiryUrl.value ?? "").trim(),
			impressumUrl: String(impressumUrl.value ?? "").trim(),
			serverRules: rulesText.value
				.split("\n")
				.map((line) => line.trim())
				.filter((line) => line !== ""),
		});
		apply(updated);
		// The shared instance object feeds the header, the welcome page and the
		// about page; refresh it so the change is visible without a reload.
		void updateInstance(true);
		saved.value = true;
	} catch (cause) {
		error.value = errorText(cause);
	} finally {
		saving.value = false;
	}
}

onMounted(() => {
	void load();
});

definePageMetadata({
	title: "实例设置",
	icon: "ph-sliders-horizontal ph-bold ph-lg",
});
</script>

<style lang="scss" scoped>
.notice {
	margin-top: 16px;
}
</style>
