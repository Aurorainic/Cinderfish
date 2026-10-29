<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { useRouter } from "vue-router";
import { api, ApiError } from "../api/client";
import AppIcon from "../components/AppIcon.vue";
import UserAvatar from "../components/UserAvatar.vue";
import { instance, loadInstance } from "../stores/instance";
import { isAdmin, session, signout } from "../stores/session";

type Theme = "light" | "dark";

const router = useRouter();

function readTheme(): Theme {
  const current = document.documentElement.dataset.theme;
  if (current === "light" || current === "dark") return current;
  return localStorage.getItem("cinderfish.theme") === "light" ? "light" : "dark";
}

const theme = ref<Theme>(readTheme());

function setTheme(next: Theme) {
  theme.value = next;
  document.documentElement.dataset.theme = next;
  localStorage.setItem("cinderfish.theme", next);
}

const themes: { id: Theme; label: string; icon: "sun" | "moon" }[] = [
  { id: "light", label: "浅色", icon: "sun" },
  { id: "dark", label: "深色", icon: "moon" },
];

// Misskey spells the federation modes all / specified / none. The public and
// allowlist spellings are accepted as aliases so a config change does not
// blank the label.
const federationLabels: Record<string, string> = {
  public: "公开联邦",
  all: "公开联邦",
  allowlist: "仅白名单",
  specified: "仅白名单",
  none: "不开放联邦",
};

const federationLabel = computed(() => {
  const mode = instance.meta?.federation;
  if (!mode) return "未知";
  return federationLabels[mode] ?? mode;
});

const registrationLabel = computed(() => {
  const meta = instance.meta;
  if (!meta) return "未知";
  const mode = meta.registrationMode;
  if (mode) {
    const labels: Record<string, string> = {
      open: "开放注册",
      invite: "仅限邀请",
      closed: "停止注册",
      admin: "仅管理员创建",
    };
    return labels[mode] ?? mode;
  }
  return meta.disableRegistration ? "停止注册" : "开放注册";
});

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

// The session may resolve after this view mounts, so the form follows it.
watch(
  () => session.me?.id,
  () => {
    syncProfileForm();
  },
  { immediate: true },
);

function handleSignout() {
  signout();
  router.push("/");
}

onMounted(() => {
  theme.value = readTheme();
  loadInstance();
});
</script>

<template>
  <header class="page-title">
    <AppIcon name="settings" />
    <h1>设置</h1>
  </header>

  <div v-if="!session.ready" class="card muted">正在读取会话…</div>

  <template v-else>
    <div v-if="!me" class="card signin-gate">
      <h2>需要登录</h2>
      <p class="muted small">登录之后才会显示账户相关的设置。</p>
      <div class="gate-actions">
        <RouterLink to="/login" class="button">登录</RouterLink>
        <RouterLink to="/signup" class="button ghost">注册</RouterLink>
      </div>
    </div>

    <section v-if="me" class="card">
      <h2 class="card-title">账户</h2>

      <div class="account-head">
        <UserAvatar :user="me" :size="64" />
        <div class="account-text">
          <div class="account-name">
            {{ me.name || me.username }}
            <span v-if="isAdmin" class="badge admin">管理员</span>
          </div>
          <div class="muted small">@{{ me.username }}</div>
          <div class="muted small mono">ID: {{ me.id }}</div>
        </div>
      </div>

      <div class="row-actions">
        <RouterLink :to="`/@${me.username}`" class="button ghost small">查看个人资料</RouterLink>
      </div>

      <p class="muted small account-hint">密码修改尚未在网页端开放，需要时请由管理员在服务器上处理。</p>
    </section>

    <section v-if="me" class="card">
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
        <button type="button" class="button" :disabled="!canSaveProfile" @click="saveProfile">
          {{ profileSaving ? "保存中…" : "保存" }}
        </button>
        <button
          type="button"
          class="button ghost"
          :disabled="!profileDirty || profileSaving"
          @click="resetProfile"
        >
          还原
        </button>
      </div>
    </section>

    <section class="card">
      <h2 class="card-title">外观</h2>
      <p class="muted small">选择界面主题，选择会保存在这台设备上。</p>
      <div class="theme-options">
        <button
          v-for="option in themes"
          :key="option.id"
          type="button"
          class="theme-option"
          :class="{ active: theme === option.id }"
          :aria-pressed="theme === option.id"
          @click="setTheme(option.id)"
        >
          <AppIcon :name="option.icon" />
          <span>{{ option.label }}</span>
          <AppIcon v-if="theme === option.id" name="check" class="tick" />
        </button>
      </div>
    </section>

    <section class="card">
      <h2 class="card-title">实例</h2>
      <dl v-if="instance.meta" class="meta-list">
        <div class="meta-row">
          <dt>名称</dt>
          <dd>{{ instance.meta.name }}</dd>
        </div>
        <div class="meta-row">
          <dt>地址</dt>
          <dd class="mono">{{ instance.meta.uri }}</dd>
        </div>
        <div class="meta-row">
          <dt>版本</dt>
          <dd>{{ instance.meta.version }}</dd>
        </div>
        <div class="meta-row">
          <dt>联邦模式</dt>
          <dd>{{ federationLabel }}</dd>
        </div>
        <div class="meta-row">
          <dt>注册方式</dt>
          <dd>{{ registrationLabel }}</dd>
        </div>
      </dl>
      <p v-else-if="instance.error" class="notice error">读取实例信息失败：{{ instance.error }}</p>
      <p v-else class="muted small">正在读取实例信息…</p>
    </section>

    <section v-if="me" class="card">
      <h2 class="card-title">会话</h2>
      <p class="muted small">退出后这台设备上的访问令牌会被清除。</p>
      <div class="row-actions">
        <button type="button" class="button danger" @click="handleSignout">
          <AppIcon name="logout" />
          退出登录
        </button>
      </div>
    </section>
  </template>
</template>

<style scoped>
.card + .card {
  margin-top: 16px;
}

.signin-gate h2 {
  font-size: 1.05rem;
  margin-bottom: 6px;
}

.gate-actions {
  display: flex;
  gap: 8px;
  margin-top: 16px;
}

.account-head {
  display: flex;
  align-items: center;
  gap: 14px;
}

.account-text {
  min-width: 0;
}

.account-name {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 700;
  color: var(--text-strong);
  font-size: 1.05rem;
  overflow-wrap: anywhere;
}

.badge {
  padding: 1px 8px;
  border-radius: 999px;
  font-size: 0.7rem;
  font-weight: 700;
  letter-spacing: 0.02em;
}

.badge.admin {
  background: var(--accent-soft);
  color: var(--accent);
  border: 1px solid color-mix(in srgb, var(--accent) 40%, transparent);
}

.row-actions {
  display: flex;
  gap: 8px;
  margin-top: 14px;
}

.account-hint {
  margin-top: 12px;
}

.avatar-row {
  display: flex;
  align-items: center;
  gap: 16px;
  margin: 14px 0 18px;
  flex-wrap: wrap;
}

.avatar-actions {
  display: flex;
  gap: 8px;
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
  gap: 8px;
}

.counter {
  font-weight: 500;
  font-size: 0.78rem;
  color: var(--muted);
  font-variant-numeric: tabular-nums;
}

.counter.over {
  color: var(--danger);
}

.mono {
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  overflow-wrap: anywhere;
}

.theme-options {
  display: flex;
  gap: 10px;
  margin-top: 12px;
  flex-wrap: wrap;
}

.theme-option {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 10px 18px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--bg-raised);
  color: var(--muted);
  font-weight: 600;
  cursor: pointer;
  transition: border-color 0.15s ease, color 0.15s ease, background 0.15s ease;
}

.theme-option:hover {
  color: var(--text);
  border-color: var(--accent);
}

.theme-option.active {
  color: var(--accent);
  border-color: var(--accent);
  background: var(--accent-soft);
}

.theme-option .tick {
  width: 16px;
  height: 16px;
}

.meta-list {
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.meta-row {
  display: flex;
  gap: 12px;
  font-size: 0.9rem;
}

.meta-row dt {
  flex: none;
  width: 88px;
  color: var(--muted);
}

.meta-row dd {
  margin: 0;
  min-width: 0;
  color: var(--text);
}
</style>
