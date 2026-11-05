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
				<template #label>未提供的字段</template>
				<CfInfo>
					后端 <code>i/update</code> 只接受 name、description、avatarId、
					followersVisibility、followingVisibility。所在地、生日、语言、附加信息
					以及「猫／机器人」标记都没有写入接口，因此这里不提供输入框；两项关注可见性在「隐私」设置页。
				</CfInfo>
			</FormSection>
		</template>
	</div>
</template>

<script setup lang="ts">
// Firefish's profile settings page, reduced to what `POST /api/i/update`
// actually writes (docs/research/api-contract/auth-users.md):
//   * name / description — saved with the button, not on every keystroke like
//     upstream, so a failed request cannot silently drop the old value;
//   * avatar — `selectFile` + `avatarId`, applied immediately as upstream does;
//   * the image cropper (`os.cropImage`) does not exist here, so the chosen file
//     is used as-is;
//   * banner, location, birthday, lang, fields (metadata), isCat/isBot/speakAsCat
//     have no write path and are not rendered as dead controls — the section at
//     the bottom says so instead;
//   * `langmap` is not consumed here for the same reason.
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

const loading = ref(true);
const loadFailed = ref(false);
const saving = ref(false);
const avatarSaving = ref(false);
const saveFailed = ref(false);
const saveErrorText = ref("");

const name = ref("");
const description = ref("");

const user = computed(() => me.value);

function errorText(error: unknown): string {
	return error instanceof Error ? error.message : String(error);
}

function syncFromUser(): void {
	name.value = me.value?.name ?? "";
	description.value = me.value?.description ?? "";
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
</style>
