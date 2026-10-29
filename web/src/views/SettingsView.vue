<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
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

      <p class="notice info">
        修改昵称、简介、头像与密码暂时不能在网页端完成：后端尚未提供对应的编辑接口。
      </p>
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
