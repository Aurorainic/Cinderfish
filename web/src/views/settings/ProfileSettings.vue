// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/pages/settings/profile.vue
//
// Adaptations: `i/update` only accepts name / description / avatarId, so the
// location, birthday, fields, isBot / isCat / speakAsCat controls of the
// original are dropped rather than faked. The banner is not editable because
// bannerId is not in the whitelist either. Saving goes through the session
// store's updateAccount so the whole UI sees the new values.
<script lang="ts" setup>
import { computed, ref, watch } from "vue";
import { api, ApiError } from "../../api/client";
import MkButton from "../../components/ui/MkButton.vue";
import MkInfo from "../../components/ui/MkInfo.vue";
import UserAvatar from "../../components/UserAvatar.vue";
import { i18n } from "../../i18n";
import { session, updateAccount } from "../../stores/session";
import SettingsPage from "./SettingsPage.vue";
import SettingsSection from "./SettingsSection.vue";

const me = computed(() => session.me);

// i/update caps the profile fields at 50 and 500 characters; the counters
// below count runes, the same way the server counts them.
const MAX_NAME = 50;
const MAX_DESCRIPTION = 500;

const profileName = ref("");
const profileDescription = ref("");
const profileSaving = ref(false);
const avatarBusy = ref(false);
const profileError = ref("");
const profileSaved = ref(false);
let savedTimer: number | undefined;

const nameLength = computed(() => [...profileName.value].length);
const descriptionLength = computed(() => [...profileDescription.value].length);
const overLimit = computed(
	() => nameLength.value > MAX_NAME || descriptionLength.value > MAX_DESCRIPTION,
);

const profileDirty = computed(() => {
	const current = session.me;
	if (!current) return false;
	return (
		profileName.value !== (current.name ?? "") ||
		profileDescription.value !== (current.description ?? "")
	);
});

const canSaveProfile = computed(
	() =>
		profileDirty.value &&
		!overLimit.value &&
		!profileSaving.value &&
		!avatarBusy.value &&
		!!session.me,
);

function syncProfileForm() {
	profileName.value = session.me?.name ?? "";
	profileDescription.value = session.me?.description ?? "";
}

function errorMessage(err: unknown, fallback: string): string {
	if (err instanceof ApiError) return err.message || fallback;
	if (err instanceof Error) return err.message || fallback;
	return fallback;
}

function flashSaved() {
	profileSaved.value = true;
	window.clearTimeout(savedTimer);
	savedTimer = window.setTimeout(() => (profileSaved.value = false), 2500);
}

async function saveProfile() {
	if (!canSaveProfile.value) return;
	profileSaving.value = true;
	profileError.value = "";
	try {
		// An empty string clears the field; i/update treats "" and null alike.
		// Both keys are always sent, so the form state is exactly what is saved.
		await updateAccount({
			name: profileName.value.trim(),
			description: profileDescription.value,
		});
		syncProfileForm();
		flashSaved();
	} catch (err) {
		profileError.value = errorMessage(err, "保存失败");
	} finally {
		profileSaving.value = false;
	}
}

function resetProfile() {
	syncProfileForm();
	profileError.value = "";
}

async function onAvatarPicked(event: Event) {
	const input = event.target as HTMLInputElement;
	const file = input.files?.[0];
	input.value = "";
	if (!file) return;
	if (!file.type.startsWith("image/")) {
		profileError.value = "请选择图片文件。";
		return;
	}
	avatarBusy.value = true;
	profileError.value = "";
	try {
		const uploaded = await api.uploadFile(file);
		await updateAccount({ avatarId: uploaded.id });
		flashSaved();
	} catch (err) {
		profileError.value = errorMessage(err, "头像上传失败");
	} finally {
		avatarBusy.value = false;
	}
}

async function removeAvatar() {
	if (avatarBusy.value || !session.me) return;
	avatarBusy.value = true;
	profileError.value = "";
	try {
		await updateAccount({ avatarId: "" });
		flashSaved();
	} catch (err) {
		profileError.value = errorMessage(err, "移除头像失败");
	} finally {
		avatarBusy.value = false;
	}
}

// The session may resolve after this page mounts, so the form follows it.
watch(
	() => session.me?.id,
	() => {
		syncProfileForm();
	},
	{ immediate: true },
);
</script>

<template>
	<SettingsPage>
		<div v-if="!me" class="_panel gate-panel">
			<h2 class="gate-title">需要登录</h2>
			<p class="settings-hint">登录之后才会显示账户相关的设置。</p>
			<div class="settings-actions">
				<MkButton link to="/login" primary inline>登录</MkButton>
				<MkButton link to="/signup" inline>注册</MkButton>
			</div>
		</div>

		<template v-else>
			<MkInfo v-if="profileSaved">{{ i18n.ts.saved }}</MkInfo>

			<SettingsSection title="头像" persist-key="settings-profile-avatar">
				<p class="settings-hint">
					头像会显示在个人页与每条笔记的头部。横幅图片暂不可编辑：服务器还没有开放对应字段。
				</p>
				<div class="avatar-row">
					<UserAvatar :user="me" :size="72" />
					<div class="avatar-actions">
						<label class="upload-button">
							<input type="file" accept="image/*" class="file-input" @change="onAvatarPicked" />
							{{ avatarBusy ? i18n.ts.uploading : me.avatarUrl ? "更换头像" : i18n.ts._profile.changeAvatar }}
						</label>
						<MkButton v-if="me.avatarUrl" inline :disabled="avatarBusy" @click="removeAvatar">
							移除头像
						</MkButton>
					</div>
				</div>
			</SettingsSection>

			<SettingsSection title="昵称" persist-key="settings-profile-name">
				<div class="field">
					<label for="profile-name" class="field-head">
						<span class="sr-only">{{ i18n.ts._profile.name }}</span>
						<span class="counter" :class="{ over: nameLength > MAX_NAME }">
							{{ nameLength }}/{{ MAX_NAME }}
						</span>
					</label>
					<input
						id="profile-name"
						v-model="profileName"
						class="text-input"
						type="text"
						:maxlength="MAX_NAME"
						autocomplete="nickname"
						:placeholder="i18n.ts._profile.name"
					/>
				</div>
			</SettingsSection>

			<SettingsSection title="简介" persist-key="settings-profile-description">
				<p class="settings-hint">{{ i18n.ts._profile.youCanIncludeHashtags }}</p>
				<div class="field">
					<label for="profile-description" class="field-head">
						<span class="sr-only">{{ i18n.ts._profile.description }}</span>
						<span class="counter" :class="{ over: descriptionLength > MAX_DESCRIPTION }">
							{{ descriptionLength }}/{{ MAX_DESCRIPTION }}
						</span>
					</label>
					<textarea
						id="profile-description"
						v-model="profileDescription"
						class="text-input"
						rows="5"
						:maxlength="MAX_DESCRIPTION"
						:placeholder="i18n.ts._profile.description"
					></textarea>
				</div>

				<MkInfo v-if="profileError" warn>{{ profileError }}</MkInfo>

				<div class="settings-actions">
					<MkButton primary inline :disabled="!canSaveProfile" @click="saveProfile">
						{{ profileSaving ? i18n.ts.processing : i18n.ts.save }}
					</MkButton>
					<MkButton inline :disabled="!profileDirty || profileSaving" @click="resetProfile">
						还原
					</MkButton>
					<MkButton link inline :to="`/@${me.username}`" class="view-link">
						{{ i18n.ts.profile }}
					</MkButton>
				</div>
			</SettingsSection>

			<SettingsSection
				title="不可编辑"
				persist-key="settings-profile-readonly"
				:expanded="false"
			>
				<p class="settings-hint">
					服务器侧的个人资料写接口只接受昵称、简介和头像（i/update
					字段白名单），位置、生日、附加信息与 isBot / isCat
					标记暂时没有写路径，这里不做前端模拟。
				</p>
			</SettingsSection>
		</template>
	</SettingsPage>
</template>

<style scoped>
.gate-panel {
	display: flex;
	flex-direction: column;
	gap: var(--space-8);
	padding: var(--margin);
}

.gate-title {
	margin: 0;
	font-size: 1.1em;
}

.avatar-row {
	display: flex;
	align-items: center;
	gap: var(--space-16);
	flex-wrap: wrap;
}

.avatar-actions {
	display: flex;
	align-items: center;
	gap: var(--space-8);
	flex-wrap: wrap;
}

/* The label is the button; the real input stays off-screen but focusable
   through the label, so keyboard users still reach it. */
.upload-button {
	position: relative;
	display: inline-flex;
	align-items: center;
	overflow: hidden;
	padding: 6px 14px;
	border: 1px solid var(--divider);
	border-radius: var(--radius);
	background: transparent;
	color: inherit;
	font-size: 0.9em;
	font-weight: 600;
	cursor: pointer;
	transition: border-color 0.15s ease, color 0.15s ease;
}

.upload-button:hover,
.upload-button:focus-within {
	border-color: var(--accent);
	color: var(--accent);
}

.file-input {
	position: absolute;
	inset: 0;
	opacity: 0;
	cursor: pointer;
}

.field {
	display: flex;
	flex-direction: column;
	gap: 6px;
}

.field-head {
	display: flex;
	align-items: baseline;
	justify-content: space-between;
	gap: var(--space-8);
	min-block-size: 1.4em;
}

.sr-only {
	position: absolute;
	inline-size: 1px;
	block-size: 1px;
	overflow: hidden;
	clip: rect(0 0 0 0);
}

.counter {
	color: var(--text-weak);
	font-size: 0.8em;
	font-variant-numeric: tabular-nums;
}

.counter.over {
	color: var(--error);
}

.text-input {
	box-sizing: border-box;
	inline-size: 100%;
	padding: 8px 10px;
	border: 1px solid var(--divider);
	border-radius: var(--radius);
	background: var(--panelHighlight);
	color: var(--fg);
	font: inherit;
	line-height: 1.5;
	resize: vertical;
}

.text-input:focus-visible {
	outline: 2px solid var(--focus);
	outline-offset: 1px;
}

.view-link {
	margin-inline-start: auto;
}
</style>
