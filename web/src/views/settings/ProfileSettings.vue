<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { api, ApiError } from "../../api/client";
import AppIcon from "../../components/AppIcon.vue";
import UserAvatar from "../../components/UserAvatar.vue";
import { session } from "../../stores/session";
import SettingsPage from "./SettingsPage.vue";

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
    session.me = await api.iUpdate({
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
    session.me = await api.iUpdate({ avatarId: uploaded.id });
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
    session.me = await api.iUpdate({ avatarId: "" });
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
    <div v-if="!me" class="card signin-gate">
      <h2 class="card-title">需要登录</h2>
      <p class="muted small">登录之后才会显示账户相关的设置。</p>
      <div class="gate-actions">
        <RouterLink to="/login" class="button small">登录</RouterLink>
        <RouterLink to="/signup" class="button ghost small">注册</RouterLink>
      </div>
    </div>

    <section v-else class="card">
      <h2 class="card-title">个人资料</h2>
      <p class="muted small">昵称、简介和头像会显示在个人页与每条笔记的头部，保存后立即生效。</p>

      <div class="avatar-row">
        <UserAvatar :user="me" :size="72" />
        <div class="avatar-actions">
          <label class="button ghost small upload-button">
            <input type="file" accept="image/*" class="file-input" @change="onAvatarPicked" />
            {{ avatarBusy ? "上传中…" : me.avatarUrl ? "更换头像" : "上传头像" }}
          </label>
          <button
            v-if="me.avatarUrl"
            type="button"
            class="button ghost small"
            :disabled="avatarBusy"
            @click="removeAvatar"
          >
            移除头像
          </button>
        </div>
      </div>

      <p v-if="profileError" class="notice error">{{ profileError }}</p>

      <div class="field">
        <label for="profile-name">
          昵称
          <span class="counter" :class="{ over: nameLength > MAX_NAME }">{{ nameLength }}/{{ MAX_NAME }}</span>
        </label>
        <input
          id="profile-name"
          v-model="profileName"
          class="input"
          type="text"
          :maxlength="MAX_NAME"
          autocomplete="nickname"
          placeholder="留空则显示用户名"
        />
      </div>

      <div class="field">
        <label for="profile-description">
          简介
          <span class="counter" :class="{ over: descriptionLength > MAX_DESCRIPTION }">
            {{ descriptionLength }}/{{ MAX_DESCRIPTION }}
          </span>
        </label>
        <textarea
          id="profile-description"
          v-model="profileDescription"
          class="textarea"
          rows="4"
          :maxlength="MAX_DESCRIPTION"
          placeholder="介绍一下自己…"
        ></textarea>
      </div>

      <p v-if="profileSaved" class="notice info">已保存</p>

      <div class="row-actions">
        <button type="button" class="button small" :disabled="!canSaveProfile" @click="saveProfile">
          {{ profileSaving ? "保存中…" : "保存" }}
        </button>
        <button
          type="button"
          class="button ghost small"
          :disabled="!profileDirty || profileSaving"
          @click="resetProfile"
        >
          还原
        </button>
        <RouterLink :to="`/@${me.username}`" class="button ghost small view-link">
          <AppIcon name="user" />
          查看个人资料
        </RouterLink>
      </div>
    </section>
  </SettingsPage>
</template>

<style scoped>
.signin-gate h2 {
  margin-bottom: var(--space-6);
}

.gate-actions {
  display: flex;
  gap: var(--space-8);
  margin-top: var(--space-16);
}

.avatar-row {
  display: flex;
  align-items: center;
  gap: var(--space-16);
  margin: var(--space-14) 0 var(--space-16);
  flex-wrap: wrap;
}

.avatar-actions {
  display: flex;
  gap: var(--space-8);
  flex-wrap: wrap;
}

/* The label is the button; the real input stays off-screen but focusable
   through the label, so keyboard users still reach it. */
.upload-button {
  position: relative;
  overflow: hidden;
  cursor: pointer;
}

.file-input {
  position: absolute;
  inset: 0;
  opacity: 0;
  cursor: pointer;
}

.upload-button:focus-within {
  border-color: var(--accent);
  color: var(--accent);
}

.field label {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: var(--space-8);
}

.counter {
  color: var(--text-weak);
  font-size: var(--fs-xs);
  font-variant-numeric: tabular-nums;
}

.counter.over {
  color: var(--danger);
}

.row-actions {
  display: flex;
  align-items: center;
  gap: var(--space-8);
  flex-wrap: wrap;
  margin-top: var(--space-14);
}

.view-link {
  margin-left: auto;
}

.view-link .icon {
  width: 15px;
  height: 15px;
}
</style>
