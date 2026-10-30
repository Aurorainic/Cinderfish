<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import AppIcon from "../components/AppIcon.vue";
import UserAvatar from "../components/UserAvatar.vue";
import { api, ApiError } from "../api/client";
import type { UserDetailed } from "../api/types";
import { instance, loadInstance } from "../stores/instance";
import { isAdmin, session } from "../stores/session";

type Mode = "public" | "allowlist" | "none";

const MODES: { id: Mode; label: string; desc: string }[] = [
  { id: "public", label: "公开联邦", desc: "与所有实例互相投递笔记与活动。" },
  { id: "allowlist", label: "仅白名单", desc: "只与白名单中的实例互通，其余实例的投递一律丢弃。" },
  { id: "none", label: "不开放联邦", desc: "完全关闭联邦，只保留本实例内部的笔记。" },
];

// The admin API speaks public/allowlist/none, while /api/meta exposes the
// Misskey vocabulary all/specified/none.
const META_TO_MODE: Record<string, Mode> = {
  all: "public",
  specified: "allowlist",
  none: "none",
};

const mode = ref<Mode | "">("");
const modeSaving = ref(false);
const modeNotice = ref<{ kind: "ok" | "bad"; text: string } | null>(null);

const hosts = ref<string[]>([]);
const hostsLoading = ref(true);
const hostsError = ref("");
const newHost = ref("");
const hostPending = ref(false);
const hostNotice = ref<{ kind: "ok" | "bad"; text: string } | null>(null);
const removing = ref<string | null>(null);
const reloading = ref(false);

const allowlistActive = computed(() => mode.value === "allowlist");

function labelOf(id: Mode): string {
  return MODES.find((m) => m.id === id)?.label ?? id;
}

function messageOf(err: unknown): string {
  if (err instanceof ApiError) return err.message || "请求失败";
  if (err instanceof Error) return err.message;
  return String(err);
}

watch(
  () => instance.meta,
  () => {
    if (mode.value) return;
    const federation = instance.meta?.federation;
    if (federation) mode.value = META_TO_MODE[federation] ?? "none";
  },
  { immediate: true },
);

async function reloadMeta(): Promise<void> {
  modeNotice.value = null;
  await loadInstance();
  if (!mode.value && !instance.meta) {
    modeNotice.value = { kind: "bad", text: "无法读取实例信息，请检查网络后重试。" };
  }
}

async function selectMode(next: Mode) {
  if (!mode.value || modeSaving.value || next === mode.value) return;
  modeSaving.value = true;
  modeNotice.value = null;
  try {
    const res = await api.federationMode(next);
    const saved = (res.mode || next) as Mode;
    mode.value = saved;
    modeNotice.value = { kind: "ok", text: `已切换为「${labelOf(saved)}」。` };
  } catch (err) {
    modeNotice.value = { kind: "bad", text: `切换失败：${messageOf(err)}` };
  } finally {
    modeSaving.value = false;
  }
}

async function loadHosts() {
  if (reloading.value) return;
  reloading.value = true;
  hostsLoading.value = true;
  hostsError.value = "";
  try {
    const res = await api.federationAllowlist();
    hosts.value = res.hosts ?? [];
  } catch (err) {
    hostsError.value = `白名单加载失败：${messageOf(err)}`;
    hosts.value = [];
  } finally {
    hostsLoading.value = false;
    reloading.value = false;
  }
}

const HOST_PATTERN = /^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?(?:\.[a-z0-9](?:[a-z0-9-]*[a-z0-9])?)*$/;

async function addHost() {
  if (!allowlistActive.value || hostPending.value) return;
  hostNotice.value = null;
  const host = newHost.value.trim().toLowerCase();
  if (!host) {
    hostNotice.value = { kind: "bad", text: "请输入要添加的实例域名。" };
    return;
  }
  if (!HOST_PATTERN.test(host)) {
    hostNotice.value = { kind: "bad", text: `「${host}」不是有效的实例域名。` };
    return;
  }
  if (hosts.value.includes(host)) {
    hostNotice.value = { kind: "bad", text: `「${host}」已经在白名单中。` };
    return;
  }
  hostPending.value = true;
  try {
    const res = await api.updateFederationAllowlist(host, false);
    hosts.value = res.hosts ?? [];
    newHost.value = "";
    hostNotice.value = { kind: "ok", text: `已添加 ${host}。` };
  } catch (err) {
    hostNotice.value = { kind: "bad", text: `添加失败：${messageOf(err)}` };
  } finally {
    hostPending.value = false;
  }
}

async function removeHost(host: string) {
  if (!allowlistActive.value || removing.value !== null) return;
  hostNotice.value = null;
  removing.value = host;
  try {
    const res = await api.updateFederationAllowlist(host, true);
    hosts.value = res.hosts ?? [];
    hostNotice.value = { kind: "ok", text: `已移除 ${host}。` };
  } catch (err) {
    hostNotice.value = { kind: "bad", text: `移除失败：${messageOf(err)}` };
  } finally {
    removing.value = null;
  }
}

// Account management. Rows are UserDetailed: isSuspended is present, but
// isAdmin is not packed into the list, so read it defensively.
const users = ref<UserDetailed[]>([]);
const usersLoading = ref(true);
const usersError = ref("");
const usersNotice = ref<{ kind: "ok" | "bad"; text: string } | null>(null);
const suspending = ref<string | null>(null);

const newUsername = ref("");
const newPassword = ref("");
const creating = ref(false);
const createNotice = ref<{ kind: "ok" | "bad"; text: string } | null>(null);

function rowIsAdmin(row: UserDetailed): boolean {
  return row.isAdmin === true;
}

function formatDate(iso: string): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return iso;
  return date.toLocaleDateString("zh-CN", { year: "numeric", month: "2-digit", day: "2-digit" });
}

async function loadUsers() {
  usersLoading.value = true;
  usersError.value = "";
  try {
    users.value = await api.adminShowUsers({ limit: 50 });
  } catch (err) {
    usersError.value = `用户列表加载失败：${messageOf(err)}`;
    users.value = [];
  } finally {
    usersLoading.value = false;
  }
}

async function toggleSuspend(row: UserDetailed) {
  if (suspending.value !== null || row.id === session.me?.id) return;
  const wasSuspended = row.isSuspended;
  if (!wasSuspended) {
    const ok = window.confirm(
      `确定要停用 @${row.username} 吗？停用后该账号将无法登录，也无法发布内容。`,
    );
    if (!ok) return;
  }
  suspending.value = row.id;
  usersNotice.value = null;
  try {
    if (wasSuspended) await api.adminUnsuspendUser(row.id);
    else await api.adminSuspendUser(row.id);
    row.isSuspended = !wasSuspended;
    usersNotice.value = {
      kind: "ok",
      text: wasSuspended ? `已恢复 @${row.username}。` : `已停用 @${row.username}。`,
    };
  } catch (err) {
    usersNotice.value = {
      kind: "bad",
      text: `${wasSuspended ? "恢复" : "停用"}失败：${messageOf(err)}`,
    };
  } finally {
    suspending.value = null;
  }
}

async function createAccount() {
  if (creating.value) return;
  createNotice.value = null;
  const username = newUsername.value.trim();
  const password = newPassword.value;
  if (!username || !password) {
    createNotice.value = { kind: "bad", text: "请填写用户名与密码。" };
    return;
  }
  creating.value = true;
  try {
    const created = await api.adminCreateAccount(username, password);
    users.value = [created, ...users.value];
    newUsername.value = "";
    newPassword.value = "";
    createNotice.value = { kind: "ok", text: `已创建账号 @${created.username}。` };
  } catch (err) {
    createNotice.value = { kind: "bad", text: `创建失败：${messageOf(err)}` };
  } finally {
    creating.value = false;
  }
}

let started = false;

watch(
  () => [session.ready, isAdmin.value] as const,
  ([ready, admin]) => {
    if (!ready || !admin || started) return;
    started = true;
    void loadHosts();
    void loadUsers();
  },
  { immediate: true },
);

onMounted(() => {
  void loadInstance();
});
</script>

<template>
  <header class="page-title">
    <AppIcon name="shield" />
    <h1>管理</h1>
  </header>

  <section v-if="!session.ready" class="card">
    <p class="muted small">正在确认登录状态…</p>
  </section>

  <section v-else-if="!isAdmin" class="card gate">
    <span class="gate-icon"><AppIcon name="shield" /></span>
    <h2>需要管理员权限</h2>
    <p class="muted gate-text">
      这个页面只对实例管理员开放。请使用管理员账号登录后再访问。
    </p>
    <RouterLink to="/login" class="button">登录</RouterLink>
  </section>

  <template v-else>
    <section class="card section">
      <div class="section-head">
        <h2 id="federation-mode-title">联邦模式</h2>
        <span v-if="modeSaving" class="hint">正在保存…</span>
      </div>
      <p class="muted small section-desc">
        决定本实例与其他实例互通的程度。切换后立即生效，无需重启。
      </p>

      <div class="modes" role="group" aria-labelledby="federation-mode-title">
        <button
          v-for="item in MODES"
          :key="item.id"
          type="button"
          class="mode"
          :class="{ active: mode === item.id }"
          :aria-pressed="mode === item.id"
          :disabled="!mode"
          @click="selectMode(item.id)"
        >
          <span class="mode-name">
            {{ item.label }}
            <span v-if="mode === item.id" class="mode-tag">
              <AppIcon name="check" />
              当前
            </span>
          </span>
          <span class="mode-desc">{{ item.desc }}</span>
        </button>
      </div>

      <p
        v-if="modeNotice"
        class="notice"
        :class="modeNotice.kind === 'ok' ? 'info' : 'error'"
        role="status"
        aria-live="polite"
      >
        {{ modeNotice.text }}
      </p>
      <p v-else-if="!mode" class="notice info">
        <span>正在读取当前联邦模式…</span>
        <button class="button ghost small" type="button" @click="reloadMeta">重新加载</button>
      </p>
    </section>

    <section class="card section" :class="{ dimmed: !allowlistActive }">
      <div class="section-head">
        <h2>联邦白名单</h2>
        <span v-if="hosts.length" class="count">{{ hosts.length }}</span>
      </div>
      <p class="muted small section-desc">
        按域名后缀匹配：加入 <code>example.com</code> 后，<code>example.com</code> 与子域
        <code>sub.example.com</code> 都会被允许，而 <code>notexample.com</code> 不会被允许。
      </p>

      <p v-if="!allowlistActive" class="notice info">
        当前联邦模式不是「仅白名单」，白名单不会生效。切换到「仅白名单」后即可编辑。
      </p>

      <form class="add-host" @submit.prevent="addHost">
        <label class="sr-only" for="allowlist-host">要添加的实例域名</label>
        <input
          id="allowlist-host"
          v-model="newHost"
          class="input"
          type="text"
          placeholder="example.com"
          autocomplete="off"
          spellcheck="false"
          :disabled="!allowlistActive || hostPending"
        />
        <button
          class="button"
          type="submit"
          :disabled="!allowlistActive || hostPending"
        >
          {{ hostPending ? "添加中…" : "添加" }}
        </button>
      </form>

      <p
        v-if="hostNotice"
        class="notice"
        :class="hostNotice.kind === 'ok' ? 'info' : 'error'"
        role="status"
        aria-live="polite"
      >
        {{ hostNotice.text }}
      </p>

      <div v-if="hostsError" class="notice error" role="alert">
        <span>{{ hostsError }}</span>
        <button class="button ghost small" type="button" @click="loadHosts">重试</button>
      </div>

      <p v-if="hostsLoading" class="hint">正在加载白名单…</p>
      <ul v-else-if="hosts.length" class="hosts">
        <li v-for="host in hosts" :key="host" class="host">
          <span class="host-name">{{ host }}</span>
          <button
            type="button"
            class="button ghost small remove"
            :disabled="!allowlistActive || removing !== null"
            :aria-label="`从白名单移除 ${host}`"
            @click="removeHost(host)"
          >
            {{ removing === host ? "移除中…" : "移除" }}
          </button>
        </li>
      </ul>
      <p v-else-if="!hostsError" class="hint">
        {{
          allowlistActive
            ? "白名单为空：仅白名单模式下，本实例不会与任何其他实例互通。"
            : "白名单为空。"
        }}
      </p>

      <p class="hint format-hint">只填域名，不需要写协议或路径。</p>
    </section>

    <section class="card section">
      <div class="section-head">
        <h2>账号管理</h2>
        <span v-if="users.length" class="count">{{ users.length }}</span>
      </div>
      <p class="muted small section-desc">
        这里是本实例的本地账号。被停用的账号无法登录，也无法发布内容。
      </p>

      <form class="create-account" @submit.prevent="createAccount">
        <label class="sr-only" for="new-username">新账号用户名</label>
        <input
          id="new-username"
          v-model="newUsername"
          class="input"
          type="text"
          placeholder="用户名"
          autocomplete="off"
          spellcheck="false"
          :disabled="creating"
        />
        <label class="sr-only" for="new-password">新账号初始密码</label>
        <input
          id="new-password"
          v-model="newPassword"
          class="input"
          type="password"
          placeholder="初始密码"
          autocomplete="new-password"
          :disabled="creating"
        />
        <button class="button" type="submit" :disabled="creating">
          {{ creating ? "创建中…" : "创建账号" }}
        </button>
      </form>
      <p class="muted small create-hint">
        注册模式为 <code>admin</code> 时自助注册已关闭，新账号只能由管理员在这里创建。
      </p>

      <p
        v-if="createNotice"
        class="notice"
        :class="createNotice.kind === 'ok' ? 'info' : 'error'"
        role="status"
        aria-live="polite"
      >
        {{ createNotice.text }}
      </p>

      <p
        v-if="usersNotice"
        class="notice"
        :class="usersNotice.kind === 'ok' ? 'info' : 'error'"
        role="status"
        aria-live="polite"
      >
        {{ usersNotice.text }}
      </p>

      <div v-if="usersError" class="notice error" role="alert">
        <span>{{ usersError }}</span>
        <button class="button ghost small" type="button" @click="loadUsers">重试</button>
      </div>

      <p v-if="usersLoading" class="hint">正在加载用户列表…</p>
      <ul v-else-if="users.length" class="users">
        <li v-for="row in users" :key="row.id" class="user">
          <UserAvatar :user="row" :size="40" />
          <div class="user-main">
            <div class="user-name">
              <span>{{ row.name || row.username }}</span>
              <span v-if="rowIsAdmin(row)" class="badge">管理员</span>
              <span v-if="row.isSuspended" class="badge suspended">已停用</span>
            </div>
            <div class="user-handle">@{{ row.username }}</div>
            <div class="user-joined">加入于 {{ formatDate(row.createdAt) }}</div>
          </div>
          <button
            type="button"
            class="button ghost small"
            :class="{ danger: !row.isSuspended }"
            :disabled="suspending !== null || row.id === session.me?.id"
            :title="row.id === session.me?.id ? '不能停用自己的账号' : undefined"
            :aria-label="
              row.id === session.me?.id
                ? `不能停用自己的账号`
                : row.isSuspended
                  ? `恢复 ${row.username} 的账号`
                  : `停用 ${row.username} 的账号`
            "
            @click="toggleSuspend(row)"
          >
            {{ suspending === row.id ? "处理中…" : row.isSuspended ? "恢复" : "停用" }}
          </button>
        </li>
      </ul>
      <p v-else-if="!usersError" class="hint">还没有本地账号。</p>
    </section>
  </template>
</template>

<style scoped>
.page-title .icon {
  width: 22px;
  height: 22px;
  color: var(--accent);
}

.gate {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  padding: 48px 24px;
  text-align: center;
}

.gate-icon {
  display: grid;
  place-items: center;
  width: 52px;
  height: 52px;
  border-radius: 50%;
  background: var(--accent-soft);
  color: var(--accent);
}

.gate-icon .icon {
  width: 24px;
  height: 24px;
}

.gate h2 {
  font-size: 1.15rem;
}

.gate-text {
  max-width: 40ch;
  margin-bottom: 4px;
}

.section + .section {
  margin-top: 16px;
}

.section-head {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 6px;
}

.section-head h2 {
  font-size: 1.05rem;
}

.section-desc {
  margin-bottom: 14px;
}

.count {
  display: inline-grid;
  place-items: center;
  min-width: 22px;
  height: 22px;
  padding: 0 7px;
  border-radius: 999px;
  background: var(--accent-soft);
  color: var(--accent);
  font-size: 0.78rem;
  font-weight: 700;
}

.modes {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
  margin-bottom: 14px;
}

.mode {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 14px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--bg-raised);
  color: var(--text);
  text-align: left;
  cursor: pointer;
  transition: border-color 0.15s ease, background 0.15s ease, opacity 0.15s ease;
}

.mode:hover:not(:disabled) {
  border-color: var(--accent);
}

.mode:focus-visible {
  outline: none;
  border-color: var(--accent);
  box-shadow: 0 0 0 3px var(--accent-soft);
}

.mode.active {
  border-color: var(--accent);
  background: var(--accent-soft);
}

.mode:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.mode-name {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 700;
  color: var(--text-strong);
}

.mode-tag {
  display: inline-flex;
  align-items: center;
  gap: 3px;
  padding: 1px 8px 1px 6px;
  border-radius: 999px;
  background: var(--accent);
  color: var(--on-accent);
  font-size: 0.7rem;
  font-weight: 700;
}

.mode-tag .icon {
  width: 0.9em;
  height: 0.9em;
}

.mode-desc {
  font-size: 0.82rem;
  color: var(--muted);
  line-height: 1.45;
}

.dimmed .add-host,
.dimmed .hosts {
  opacity: 0.55;
}

.add-host {
  display: flex;
  gap: 8px;
  margin-bottom: 14px;
}

.add-host .input {
  flex: 1;
  min-width: 0;
}

.hosts {
  list-style: none;
  margin: 0 0 4px;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.host {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 8px 8px 8px 14px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--bg-raised);
}

.host-name {
  font-weight: 600;
  overflow-wrap: anywhere;
}

.remove {
  color: var(--danger);
}

.remove:hover:not(:disabled) {
  border-color: var(--danger);
  color: var(--danger);
}

.create-account {
  display: flex;
  gap: 8px;
  margin-bottom: 8px;
}

.create-account .input {
  flex: 1;
  min-width: 0;
}

.create-hint {
  margin-bottom: 14px;
}

.users {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.user {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 8px 10px 14px;
  border: 1px solid var(--border);
  border-radius: var(--radius);
  background: var(--bg-raised);
}

.user-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.user-name {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
  font-weight: 600;
  color: var(--text-strong);
  overflow-wrap: anywhere;
}

.user-handle {
  font-size: 0.82rem;
  color: var(--muted);
  overflow-wrap: anywhere;
}

.user-joined {
  font-size: 0.78rem;
  color: var(--muted);
}

.badge {
  padding: 1px 8px;
  border-radius: 999px;
  background: var(--accent-soft);
  color: var(--accent);
  font-size: 0.7rem;
  font-weight: 700;
}

.badge.suspended {
  background: transparent;
  color: var(--danger);
  border: 1px solid var(--danger);
}

.notice {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.notice:last-child {
  margin-bottom: 0;
}

code {
  padding: 1px 5px;
  border-radius: 5px;
  background: var(--panel-2);
  font-size: 0.82em;
}

.format-hint {
  margin-top: 10px;
}

.sr-only {
  position: absolute;
  width: 1px;
  height: 1px;
  padding: 0;
  margin: -1px;
  overflow: hidden;
  clip: rect(0, 0, 0, 0);
  white-space: nowrap;
  border: 0;
}

@media (max-width: 620px) {
  .modes {
    grid-template-columns: 1fr;
  }

  .add-host {
    flex-direction: column;
  }

  .create-account {
    flex-direction: column;
  }

  .user {
    flex-wrap: wrap;
  }
}
</style>
