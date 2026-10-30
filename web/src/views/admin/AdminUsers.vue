<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { api } from "../../api/client";
import type { UserDetailed } from "../../api/types";
import UserAvatar from "../../components/UserAvatar.vue";
import { session } from "../../stores/session";
import AdminCallout from "./AdminCallout.vue";

const users = ref<UserDetailed[]>([]);
const loading = ref(true);
const error = ref("");
const listNotice = ref<{ kind: "ok" | "bad"; text: string } | null>(null);
const dangerNotice = ref<{ kind: "ok" | "bad"; text: string } | null>(null);
const busyId = ref<string | null>(null);

const newUsername = ref("");
const newPassword = ref("");
const creating = ref(false);
const createNotice = ref<{ kind: "ok" | "bad"; text: string } | null>(null);

const activeUsers = computed(() => users.value.filter((row) => !row.isSuspended));
const suspendedUsers = computed(() => users.value.filter((row) => row.isSuspended));

function rowIsAdmin(row: UserDetailed): boolean {
  return row.isAdmin === true;
}

function messageOf(err: unknown): string {
  return err instanceof Error ? err.message : String(err);
}

function formatDate(iso: string): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return iso;
  return date.toLocaleDateString("zh-CN", { year: "numeric", month: "2-digit", day: "2-digit" });
}

async function loadUsers() {
  loading.value = true;
  error.value = "";
  try {
    users.value = await api.adminShowUsers({ limit: 50 });
  } catch (err) {
    error.value = `用户列表加载失败：${messageOf(err)}`;
    users.value = [];
  } finally {
    loading.value = false;
  }
}

async function toggleSuspend(row: UserDetailed) {
  if (busyId.value !== null || row.id === session.me?.id) return;
  const wasSuspended = row.isSuspended;
  if (!wasSuspended) {
    const ok = window.confirm(
      `确定要停用 @${row.username} 吗？停用后该账号将无法登录，也无法发布内容。`,
    );
    if (!ok) return;
  }
  busyId.value = row.id;
  listNotice.value = null;
  dangerNotice.value = null;
  try {
    if (wasSuspended) await api.adminUnsuspendUser(row.id);
    else await api.adminSuspendUser(row.id);
    row.isSuspended = !wasSuspended;
    const text = wasSuspended ? `已恢复 @${row.username}。` : `已停用 @${row.username}。`;
    if (wasSuspended) dangerNotice.value = { kind: "ok", text };
    else listNotice.value = { kind: "ok", text };
  } catch (err) {
    const text = `${wasSuspended ? "恢复" : "停用"}失败：${messageOf(err)}`;
    if (wasSuspended) dangerNotice.value = { kind: "bad", text };
    else listNotice.value = { kind: "bad", text };
  } finally {
    busyId.value = null;
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

onMounted(loadUsers);
</script>

<template>
  <section class="card">
    <div class="section-head">
      <h2>创建账号</h2>
      <span v-if="creating" class="hint">正在创建…</span>
    </div>
    <p class="muted small section-desc">
      注册模式为 <code>admin</code> 时自助注册已关闭，新账号只能由管理员在这里创建。
    </p>

    <form class="row-form" @submit.prevent="createAccount">
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

    <p
      v-if="createNotice"
      class="notice"
      :class="createNotice.kind === 'ok' ? 'info' : 'error'"
      role="status"
      aria-live="polite"
    >
      {{ createNotice.text }}
    </p>
  </section>

  <section class="card">
    <div class="section-head">
      <h2>本地账号</h2>
      <span class="count">{{ activeUsers.length }}</span>
    </div>
    <p class="muted small section-desc">
      共 {{ users.length }} 个账号，其中 {{ suspendedUsers.length }} 个已停用。停用的账号会移到下方的危险操作区。
    </p>

    <p
      v-if="listNotice"
      class="notice"
      :class="listNotice.kind === 'ok' ? 'info' : 'error'"
      role="status"
      aria-live="polite"
    >
      {{ listNotice.text }}
    </p>

    <div v-if="error" class="notice error" role="alert">
      <span>{{ error }}</span>
      <button class="button ghost small" type="button" @click="loadUsers">重试</button>
    </div>

    <p v-if="loading" class="hint">正在加载用户列表…</p>
    <ul v-else-if="activeUsers.length" class="user-list">
      <li v-for="row in activeUsers" :key="row.id" class="user-row">
        <UserAvatar :user="row" :size="40" />
        <div class="user-main">
          <div class="user-name">
            <span>{{ row.name || row.username }}</span>
            <span v-if="rowIsAdmin(row)" class="badge">管理员</span>
          </div>
          <div class="user-handle">@{{ row.username }}</div>
          <div class="user-meta">加入于 {{ formatDate(row.createdAt) }}</div>
        </div>
        <button
          type="button"
          class="button ghost small suspend"
          :disabled="busyId !== null || row.id === session.me?.id"
          :title="row.id === session.me?.id ? '不能停用自己的账号' : undefined"
          :aria-label="`停用 ${row.username} 的账号`"
          @click="toggleSuspend(row)"
        >
          {{ busyId === row.id ? "处理中…" : "停用" }}
        </button>
      </li>
    </ul>
    <p v-else-if="users.length" class="hint">
      所有账号都已停用，可在下方的危险操作区恢复。
    </p>
    <p v-else-if="!error" class="hint">还没有本地账号。</p>
  </section>

  <AdminCallout v-if="!loading && !error" title="危险操作" class="danger-zone">
    <p>停用账号会立即阻止该账号登录，也无法再发布内容。恢复需要在这里手动操作。</p>

    <p
      v-if="dangerNotice"
      class="notice"
      :class="dangerNotice.kind === 'ok' ? 'info' : 'error'"
      role="status"
      aria-live="polite"
    >
      {{ dangerNotice.text }}
    </p>

    <ul v-if="suspendedUsers.length" class="suspended-list">
      <li v-for="row in suspendedUsers" :key="row.id" class="suspended-row">
        <UserAvatar :user="row" :size="32" />
        <div class="suspended-main">
          <span class="suspended-name">{{ row.name || row.username }}</span>
          <span class="suspended-handle">@{{ row.username }}</span>
        </div>
        <button
          type="button"
          class="button ghost small"
          :disabled="busyId !== null"
          :aria-label="`恢复 ${row.username} 的账号`"
          @click="toggleSuspend(row)"
        >
          {{ busyId === row.id ? "处理中…" : "恢复" }}
        </button>
      </li>
    </ul>
    <p v-else class="muted small">当前没有被停用的账号。</p>
  </AdminCallout>
</template>

<style scoped>
.section-head {
  display: flex;
  align-items: center;
  gap: var(--space-10);
  margin-bottom: var(--space-6);
}

.section-head h2 {
  font-size: var(--fs-md);
}

.section-desc {
  margin-bottom: var(--space-14);
}

.count {
  display: inline-grid;
  place-items: center;
  min-width: 22px;
  height: 22px;
  padding: 0 var(--space-6);
  border-radius: var(--radius-full);
  background: var(--accent-soft);
  color: var(--accent);
  font-size: var(--fs-xs);
  font-weight: var(--weight-bold);
}

.row-form {
  display: flex;
  gap: var(--space-8);
  margin-bottom: var(--space-12);
}

.row-form .input {
  flex: 1;
  min-width: 0;
}

.user-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
}

.user-row {
  display: flex;
  align-items: center;
  gap: var(--space-12);
  padding: var(--space-10) var(--space-8) var(--space-10) var(--space-14);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--surface-inset);
}

.user-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.user-name {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: var(--space-6);
  font-weight: var(--weight-semibold);
  color: var(--text-strong);
  overflow-wrap: anywhere;
}

.user-handle,
.user-meta {
  font-size: var(--fs-xs);
  color: var(--text-weak);
  overflow-wrap: anywhere;
}

.badge {
  padding: 1px var(--space-8);
  border-radius: var(--radius-full);
  background: var(--accent-soft);
  color: var(--accent);
  font-size: var(--fs-2xs);
  font-weight: var(--weight-bold);
}

.suspend {
  color: var(--danger);
}

.suspend:hover:not(:disabled) {
  border-color: var(--danger);
  color: var(--danger);
}

.danger-zone {
  display: flex;
  flex-direction: column;
  gap: var(--space-12);
}

.danger-zone :deep(.callout-body) {
  display: flex;
  flex-direction: column;
  gap: var(--space-12);
}

.suspended-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
}

.suspended-row {
  display: flex;
  align-items: center;
  gap: var(--space-10);
  padding: var(--space-6) var(--space-8);
  border-radius: var(--radius-sm);
  background: color-mix(in srgb, var(--panel) 55%, transparent);
}

.suspended-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: var(--space-6);
}

.suspended-name {
  font-weight: var(--weight-semibold);
  color: var(--text);
  overflow-wrap: anywhere;
}

.suspended-handle {
  font-size: var(--fs-xs);
  color: var(--text-weak);
  overflow-wrap: anywhere;
}

code {
  padding: 1px var(--space-4);
  border-radius: var(--radius-xs);
  background: var(--panel-2);
  font-size: 0.88em;
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

@media (max-width: 560px) {
  .row-form {
    flex-direction: column;
  }

  .user-row {
    flex-wrap: wrap;
  }
}
</style>
