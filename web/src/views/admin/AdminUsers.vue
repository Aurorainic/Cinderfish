<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { api } from "../../api/client";
import type { UserDetailed } from "../../api/types";
import UserAvatar from "../../components/UserAvatar.vue";
import MkButton from "../../components/ui/MkButton.vue";
import MkInfo from "../../components/ui/MkInfo.vue";
import MkInput from "../../components/ui/MkInput.vue";
import MkLoading from "../../components/ui/MkLoading.vue";
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
  <section class="_card">
    <div class="_title section-head">
      <h2>创建账号</h2>
      <MkLoading v-if="creating" em :size-em="0.8" />
    </div>
    <div class="_content">
      <p class="_caption section-desc">
        注册模式为 <code>admin</code> 时自助注册已关闭，新账号只能由管理员在这里创建。
      </p>

      <form class="row-form" @submit.prevent="createAccount">
        <MkInput
          v-model="newUsername"
          class="row-input"
          type="text"
          placeholder="用户名"
          autocomplete="off"
          :spellcheck="false"
          :disabled="creating"
        >
          <template #label>用户名</template>
        </MkInput>
        <MkInput
          v-model="newPassword"
          class="row-input"
          type="password"
          placeholder="初始密码"
          autocomplete="new-password"
          :disabled="creating"
        >
          <template #label>初始密码</template>
        </MkInput>
        <MkButton type="submit" :disabled="creating">
          {{ creating ? "创建中…" : "创建账号" }}
        </MkButton>
      </form>

      <MkInfo
        v-if="createNotice"
        class="notice-row"
        :warn="createNotice.kind === 'bad'"
        role="status"
        aria-live="polite"
      >
        {{ createNotice.text }}
      </MkInfo>
    </div>
  </section>

  <section class="_card">
    <div class="_title section-head">
      <h2>本地账号</h2>
      <span class="count">{{ activeUsers.length }}</span>
    </div>
    <div class="_content">
      <p class="_caption section-desc">
        共 {{ users.length }} 个账号，其中 {{ suspendedUsers.length }} 个已停用。停用的账号会移到下方的危险操作区。
      </p>

      <MkInfo
        v-if="listNotice"
        class="notice-row"
        :warn="listNotice.kind === 'bad'"
        role="status"
        aria-live="polite"
      >
        {{ listNotice.text }}
      </MkInfo>

      <MkInfo v-if="error" class="notice-row" warn role="alert">
        <span>{{ error }}</span>
        <MkButton @click="loadUsers">重试</MkButton>
      </MkInfo>

      <MkLoading v-if="loading" />
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
          <MkButton
            mini
            danger
            :disabled="busyId !== null || row.id === session.me?.id"
            :title="row.id === session.me?.id ? '不能停用自己的账号' : undefined"
            :aria-label="`停用 ${row.username} 的账号`"
            @click="toggleSuspend(row)"
          >
            {{ busyId === row.id ? "处理中…" : "停用" }}
          </MkButton>
        </li>
      </ul>
      <p v-else-if="users.length" class="_caption">
        所有账号都已停用，可在下方的危险操作区恢复。
      </p>
      <p v-else-if="!error" class="_caption">还没有本地账号。</p>
    </div>
  </section>

  <AdminCallout v-if="!loading && !error" title="危险操作" class="danger-zone">
    <p>停用账号会立即阻止该账号登录，也无法再发布内容。恢复需要在这里手动操作。</p>

    <MkInfo
      v-if="dangerNotice"
      class="notice-row"
      :warn="dangerNotice.kind === 'bad'"
      role="status"
      aria-live="polite"
    >
      {{ dangerNotice.text }}
    </MkInfo>

    <ul v-if="suspendedUsers.length" class="suspended-list">
      <li v-for="row in suspendedUsers" :key="row.id" class="suspended-row">
        <UserAvatar :user="row" :size="32" />
        <div class="suspended-main">
          <span class="suspended-name">{{ row.name || row.username }}</span>
          <span class="suspended-handle">@{{ row.username }}</span>
        </div>
        <MkButton
          mini
          :disabled="busyId !== null"
          :aria-label="`恢复 ${row.username} 的账号`"
          @click="toggleSuspend(row)"
        >
          {{ busyId === row.id ? "处理中…" : "恢复" }}
        </MkButton>
      </li>
    </ul>
    <p v-else class="_caption">当前没有被停用的账号。</p>
  </AdminCallout>
</template>

<style scoped>
.section-head {
  display: flex;
  align-items: center;
  gap: 10px;
}

.section-head h2 {
  font-size: 1em;
}

.section-desc {
  margin-block-end: var(--margin);
}

.count {
  display: inline-grid;
  place-items: center;
  min-inline-size: 22px;
  block-size: 22px;
  padding-inline: 6px;
  border-radius: 999px;
  background: var(--accentedBg);
  color: var(--accent);
  font-size: 0.78em;
  font-weight: bold;
}

.notice-row {
  margin-block-end: var(--margin);
}

.row-form {
  display: flex;
  align-items: flex-end;
  flex-wrap: wrap;
  gap: 8px;
  margin-block-end: var(--margin);
}

.row-input {
  flex: 1;
  min-inline-size: 160px;
}

.user-list {
  list-style: none;
  margin: 0;
  padding: 0;
}

.user-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding-block: 10px;
  border-block-start: solid 0.5px var(--hairline);
}

.user-row:first-child {
  border-block-start: none;
}

.user-main {
  flex: 1;
  min-inline-size: 0;
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
  color: var(--fgHighlighted);
  overflow-wrap: anywhere;
}

.user-handle,
.user-meta {
  font-size: 0.85em;
  color: var(--fgTransparentWeak);
  overflow-wrap: anywhere;
}

.badge {
  padding-block: 1px;
  padding-inline: 8px;
  border-radius: 999px;
  background: var(--accentedBg);
  color: var(--accent);
  font-size: 0.75em;
  font-weight: bold;
}

.danger-zone :deep(.callout-body) {
  display: flex;
  flex-direction: column;
  gap: 12px;
  inline-size: 100%;
}

.suspended-list {
  list-style: none;
  margin: 0;
  padding: 0;
}

.suspended-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding-block: 6px;
  border-block-start: solid 0.5px var(--hairline);
}

.suspended-row:first-child {
  border-block-start: none;
}

.suspended-main {
  flex: 1;
  min-inline-size: 0;
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: 6px;
}

.suspended-name {
  font-weight: 600;
  color: var(--fg);
  overflow-wrap: anywhere;
}

.suspended-handle {
  font-size: 0.85em;
  color: var(--fgTransparentWeak);
  overflow-wrap: anywhere;
}

code {
  padding-block: 1px;
  padding-inline: 4px;
  border-radius: 4px;
  background: var(--panelHighlight);
  font-size: 0.88em;
}

@media (max-width: 560px) {
  .row-form {
    flex-direction: column;
    align-items: stretch;
  }

  .user-row {
    flex-wrap: wrap;
  }
}
</style>
