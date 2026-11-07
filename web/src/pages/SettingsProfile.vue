<!-- Ported from firefish@b425f61:packages/client/src/pages/settings/profile.vue (AGPL-3.0-only) -->
<template>
	<div class="_formRoot">
		<MkLoading v-if="loading" />
		<MkError v-else-if="loadFailed" @retry="load" />
		<CfInfo v-else-if="user == null" warn>
			未登录，无法读取个人资料。
		</CfInfo>
		<template v-else>
			<div class="header">
				<MkAvatar
					class="avatar"
					:user="user"
					:disable-link="true"
					:disable-preview="true"
					@click="changeAvatar"
				/>
				<CfButton
					primary
					rounded
					:disabled="avatarSaving"
					@click="changeAvatar"
				>
					<i class="ph-camera-rotate ph-bold ph-lg"></i>
					{{ t("_profile.changeAvatar") }}
				</CfButton>
			</div>

			<div class="bannerBlock">
				<div class="bannerPreview" :style="bannerPreviewStyle" />
				<CfButton rounded :disabled="bannerSaving" @click="changeBanner">
					<i class="ph-image ph-bold ph-lg"></i>
					{{ t("_profile.changeBanner") }}
				</CfButton>
			</div>

			<FormInput
				v-model="name"
				class="_formBlock"
				:max="NAME_MAX"
				:disabled="saving"
			>
				<template #label>{{ t("_profile.name") }}</template>
				<template #caption
					>最多 {{ NAME_MAX }} 个字符（服务端上限）。</template
				>
			</FormInput>

			<FormTextarea
				v-model="description"
				class="_formBlock"
				tall
				:max="DESCRIPTION_MAX"
				:disabled="saving"
			>
				<template #label>{{ t("_profile.description") }}</template>
				<template #caption>{{
					t("_profile.youCanIncludeHashtags")
				}}</template>
			</FormTextarea>

			<FormInput
				v-model="location"
				class="_formBlock"
				:max="LOCATION_MAX"
				:disabled="saving"
			>
				<template #label>{{ t("location") }}</template>
				<template #caption>所在地，最多 {{ LOCATION_MAX }} 个字符。</template>
			</FormInput>

			<FormInput
				v-model="birthday"
				class="_formBlock"
				type="date"
				:disabled="saving"
			>
				<template #label>{{ t("birthday") }}</template>
			</FormInput>

			<FormInput
				v-model="lang"
				class="_formBlock"
				:max="LANG_MAX"
				:disabled="saving"
			>
				<template #label>{{ t("language") }}</template>
				<template #caption>BCP-47 语言标签，例如 <code>zh-CN</code> 或 <code>ja</code>。</template>
			</FormInput>

			<FormSection>
				<template #label>{{ t("_profile.metadata") }}</template>
				<div v-for="(field, index) in fields" :key="index" class="fieldRow">
					<FormInput v-model="field.name" :max="FIELD_NAME_MAX" :disabled="saving">
						<template #label>{{ t("_profile.metadataLabel") }}</template>
					</FormInput>
					<FormInput v-model="field.value" :max="FIELD_VALUE_MAX" :disabled="saving">
						<template #label>{{ t("_profile.metadataContent") }}</template>
					</FormInput>
					<CfButton danger :disabled="saving" @click="removeField(index)">
						{{ t("remove") }}
					</CfButton>
				</div>
				<CfButton :disabled="saving || fields.length >= FIELDS_MAX" @click="addField">
					添加一项
				</CfButton>
			</FormSection>

			<CfInfo v-if="saveFailed" class="_formBlock" warn>
				{{ saveErrorText }}
			</CfInfo>

			<div class="actions">
				<CfButton
					primary
					:disabled="saving || avatarSaving"
					@click="save"
				>
					{{ t("save") }}
				</CfButton>
			</div>

			<FormSection>
				<template #label>仍不支持的字段</template>
				<CfInfo>
					横幅、<code>isCat</code>/<code>isBot</code> 标记与发言猫语仍没有写入路径，
					因此这里不提供输入框。附加信息最多 16 项，标签 255 字符、内容 1024 字符。
				</CfInfo>
			</FormSection>
		</template>
	</div>
</template>

<script setup lang="ts">
// Firefish's profile settings page. `POST /api/i/update` writes name,
// description, avatar, location, birthday, lang and the metadata fields; the
// image cropper (`os.cropImage`) does not exist here, so the chosen file is
// used as-is. Banner, isCat/isBot and speakAsCat still have no write path and
// are not rendered as dead controls.
import { computed, onMounted, ref } from "vue";
import { me, refreshAccount, updateProfile } from "@/account";
import { success } from "@/os";
import { selectFile } from "@/scripts/select-file";
import { t } from "@/i18n";
import { definePageMetadata } from "@/scripts/page-metadata";
import MkAvatar from "@/components/global/MkAvatar.vue";
import MkError from "@/components/global/MkError.vue";
import MkLoading from "@/components/global/MkLoading.vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import FormInput from "@/components/form/input.vue";
import FormSection from "@/components/form/section.vue";
import FormTextarea from "@/components/form/textarea.vue";

/** internal/httpapi/i_update.go: maxProfileNameLength / maxProfileDescriptionLength */
const NAME_MAX = 50;
const DESCRIPTION_MAX = 500;
const LOCATION_MAX = 50;
const LANG_MAX = 16;
const FIELDS_MAX = 16;
const FIELD_NAME_MAX = 255;
const FIELD_VALUE_MAX = 1024;

const loading = ref(true);
const loadFailed = ref(false);
const saving = ref(false);
const avatarSaving = ref(false);
const bannerSaving = ref(false);

const bannerPreviewStyle = computed(() =>
	user.value?.bannerUrl == null || user.value.bannerUrl === ""
		? undefined
		: {
				backgroundImage: `url(${user.value.bannerUrl})`,
				backgroundSize: "cover",
				backgroundPosition: "center",
			},
);
const saveFailed = ref(false);
const saveErrorText = ref("");

const name = ref("");
const description = ref("");
const location = ref("");
const birthday = ref("");
const lang = ref("");
const fields = ref<{ name: string; value: string }[]>([]);

const user = computed(() => me.value);

function errorText(error: unknown): string {
	return error instanceof Error ? error.message : String(error);
}

function syncFromUser(): void {
	name.value = me.value?.name ?? "";
	description.value = me.value?.description ?? "";
	location.value = me.value?.location ?? "";
	birthday.value = me.value?.birthday ?? "";
	lang.value = me.value?.lang ?? "";
	fields.value = (me.value?.fields ?? []).map((f) => ({
		name: f.name,
		value: f.value,
	}));
}

function addField(): void {
	if (fields.value.length >= FIELDS_MAX) return;
	fields.value = [...fields.value, { name: "", value: "" }];
}

function removeField(index: number): void {
	fields.value = fields.value.filter((_, i) => i !== index);
}

async function load(): Promise<void> {
	loading.value = true;
	loadFailed.value = false;
	try {
		// refreshAccount() re-reads POST /api/i and keeps the token; it returns
		// null instead of throwing when the session is gone.
		await refreshAccount();
	} finally {
		loading.value = false;
	}

	if (me.value == null) {
		loadFailed.value = true;
		return;
	}
	syncFromUser();
}

async function save(): Promise<void> {
	if (saving.value) return;
	saving.value = true;
	saveFailed.value = false;
	try {
		const trimmedName = name.value.trim();
		await updateProfile({
			// null clears the field, matching what Misskey clients send.
			name: trimmedName === "" ? null : trimmedName,
			description:
				description.value.trim() === "" ? null : description.value,
			location: location.value.trim() === "" ? null : location.value.trim(),
			birthday: birthday.value === "" ? null : birthday.value,
			lang: lang.value.trim() === "" ? null : lang.value.trim(),
			fields: fields.value
				.map((f) => ({ name: f.name.trim(), value: f.value.trim() }))
				.filter((f) => f.name !== ""),
		});
		syncFromUser();
		success();
	} catch (error) {
		saveFailed.value = true;
		saveErrorText.value = errorText(error);
	} finally {
		saving.value = false;
	}
}

async function changeBanner(event: MouseEvent): Promise<void> {
	if (bannerSaving.value) return;
	const file = await selectFile({ label: t("banner"), src: event });
	if (file == null) return;

	bannerSaving.value = true;
	saveFailed.value = false;
	try {
		await updateProfile({ bannerId: file.id });
		success();
	} catch (error) {
		saveFailed.value = true;
		saveErrorText.value = errorText(error);
	} finally {
		bannerSaving.value = false;
	}
}

async function changeAvatar(event: MouseEvent): Promise<void> {
	if (avatarSaving.value) return;
	const file = await selectFile({ label: t("avatar"), src: event });
	if (file == null) return;

	avatarSaving.value = true;
	saveFailed.value = false;
	try {
		await updateProfile({ avatarId: file.id });
		success();
	} catch (error) {
		saveFailed.value = true;
		saveErrorText.value = errorText(error);
	} finally {
		avatarSaving.value = false;
	}
}

onMounted(() => {
	void load();
});

definePageMetadata({
	title: t("profile"),
	icon: "ph-user ph-bold ph-lg",
});
</script>

<style lang="scss" scoped>
.header {
	display: flex;
	align-items: center;
	gap: 16px;
	padding: 16px;
	margin-bottom: 1.5em;
	background: var(--panel);
	border: solid 1px var(--divider);
	border-radius: var(--radius);

	.bannerBlock {
		display: flex;
		align-items: center;
		gap: 12px;
		margin-block-start: 12px;

		> .bannerPreview {
			flex: 1;
			block-size: 80px;
			border-radius: var(--radius);
			background: linear-gradient(120deg, var(--accentedBg), var(--buttonBg));
			background-size: cover;
			background-position: center;
			box-shadow: inset 0 0 0 0.5px var(--divider);
		}
	}

	> .avatar {
		inline-size: 72px;
		block-size: 72px;
		flex-shrink: 0;
		cursor: pointer;
	}
}

.actions {
	display: flex;
	justify-content: flex-end;
	margin-top: 1.5em;
}

.fieldRow {
	display: flex;
	flex-wrap: wrap;
	align-items: flex-end;
	gap: 8px;
	margin-block-end: 8px;
}
</style>
