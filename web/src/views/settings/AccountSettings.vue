// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/pages/settings/account-info.vue
// and pages/settings/security.vue
//
// Adaptations: the users/stats endpoint does not exist, so the statistics
// section of account-info is dropped (the capability matrix marks the usage
// block as hidden). Password change, 2FA and signin-history have no backend
// endpoints and are rendered as explicit unavailable rows, matching the
// project's "no fake buttons" rule.
<script setup lang="ts">
import { computed } from "vue";
import { useRouter } from "vue-router";
import AppIcon from "../../components/AppIcon.vue";
import MkKeyValue from "../../components/ui/MkKeyValue.vue";
import MkTime from "../../components/ui/MkTime.vue";
import UserAvatar from "../../components/UserAvatar.vue";
import { i18n } from "../../i18n";
import { isAdmin, session, signout } from "../../stores/session";
import SettingsPage from "./SettingsPage.vue";

const router = useRouter();

const me = computed(() => session.me);
const email = computed(() => {
  const value = session.me?.email;
  return typeof value === "string" && value ? value : "";
});

const registeredDate = computed(() => {
  const date = session.me ? new Date(session.me.createdAt) : null;
  return date && !Number.isNaN(date.getTime()) ? date.toISOString() : null;
});

function handleSignout() {
  signout();
  router.push("/");
}
</script>

<template>
  <SettingsPage>
    <div v-if="!me" class="card">
      <h2 class="card-title">需要登录</h2>
      <p class="muted small">登录之后才会显示账户与会话相关的设置。</p>
      <div class="gate-actions">
        <RouterLink to="/login" class="button small">登录</RouterLink>
        <RouterLink to="/signup" class="button ghost small">注册</RouterLink>
      </div>
    </div>

    <template v-else>
      <section class="card">
        <h2 class="card-title">账户</h2>

        <div class="account-head">
          <UserAvatar :user="me" :size="64" />
          <div class="account-text">
            <div class="account-name">
              {{ me.name || me.username }}
              <span v-if="isAdmin" class="badge admin">管理员</span>
            </div>
            <div class="muted small">@{{ me.username }}</div>
          </div>
        </div>

        <div class="key-values">
          <MkKeyValue :copy="me.id" oneline>
            <template #key>ID</template>
            <template #value><span class="mono">{{ me.id }}</span></template>
          </MkKeyValue>
          <MkKeyValue v-if="registeredDate" oneline>
            <template #key>注册时间</template>
            <template #value>
              <MkTime :time="registeredDate" mode="detail" />
            </template>
          </MkKeyValue>
          <MkKeyValue v-if="email" oneline>
            <template #key>{{ i18n.ts.email ?? "邮箱" }}</template>
            <template #value><span class="mono">{{ email }}</span></template>
          </MkKeyValue>
          <MkKeyValue oneline>
            <template #key>帖子数量</template>
            <template #value>{{ me.notesCount.toLocaleString("zh-CN") }}</template>
          </MkKeyValue>
        </div>

        <div class="row-actions">
          <RouterLink :to="`/@${me.username}`" class="button ghost small">
            <AppIcon name="user" />
            {{ i18n.ts.profile }}
          </RouterLink>
          <RouterLink to="/settings/profile" class="button ghost small">
            {{ i18n.ts.editProfile }}
          </RouterLink>
        </div>
      </section>

      <section class="card">
        <h2 class="card-title">安全</h2>

        <div class="setting-row">
          <div class="setting-text">
            <div class="setting-label">密码</div>
            <p class="muted small">网页端尚未开放自助修改密码，需要时请由管理员在服务器上处理。</p>
          </div>
          <span class="status">未开放</span>
        </div>

        <div class="setting-row">
          <div class="setting-text">
            <div class="setting-label">两步验证</div>
            <p class="muted small">服务器尚未提供两步验证接口。</p>
          </div>
          <span class="status">未开放</span>
        </div>

        <div class="setting-row">
          <div class="setting-text">
            <div class="setting-label">登录历史</div>
            <p class="muted small">服务器尚未提供 i/signin-history 接口。</p>
          </div>
          <span class="status">未开放</span>
        </div>
      </section>

      <section class="card">
        <h2 class="card-title">会话</h2>
        <p class="muted small">退出后这台设备上的访问令牌会被清除。</p>
        <div class="row-actions">
          <button type="button" class="button danger" @click="handleSignout">
            <AppIcon name="logout" />
            退出登录
          </button>
        </div>
      </section>

      <section class="card danger-card">
        <h2 class="card-title">危险操作</h2>
        <div class="setting-row">
          <div class="setting-text">
            <div class="setting-label">注销账户</div>
            <p class="muted small">
              服务器尚未提供账户注销接口，笔记与媒体的删除也无法在网页端完成；需要时请联系实例管理员。
            </p>
          </div>
          <span class="status danger">不可用</span>
        </div>
      </section>
    </template>
  </SettingsPage>
</template>

<style scoped>
.gate-actions {
  display: flex;
  gap: var(--space-8);
  margin-top: var(--space-16);
}

.account-head {
  display: flex;
  align-items: center;
  gap: var(--space-14);
}

.account-text {
  min-width: 0;
}

.account-name {
  display: flex;
  align-items: center;
  gap: var(--space-8);
  color: var(--text-strong);
  font-size: var(--fs-md);
  font-weight: var(--weight-bold);
  overflow-wrap: anywhere;
}

.badge {
  padding: 1px var(--space-8);
  border: 1px solid var(--accent-border);
  border-radius: var(--radius-full);
  background: var(--accent-soft);
  color: var(--accent);
  font-size: var(--fs-2xs);
  font-weight: var(--weight-bold);
  letter-spacing: 0.02em;
}

.key-values {
  display: flex;
  flex-direction: column;
  gap: var(--space-10);
  margin-top: var(--space-14);
}

.row-actions {
  display: flex;
  align-items: center;
  gap: var(--space-8);
  flex-wrap: wrap;
  margin-top: var(--space-14);
}

.row-actions .icon {
  width: 15px;
  height: 15px;
}

.setting-row {
  display: flex;
  align-items: flex-start;
  gap: var(--space-12);
  padding: var(--space-12) 0;
  border-top: 1px solid var(--divider);
}

.setting-row:first-of-type {
  border-top: 0;
  padding-top: var(--space-4);
}

.setting-text {
  min-width: 0;
}

.setting-label {
  color: var(--text-strong);
  font-size: var(--fs-sm);
  font-weight: var(--weight-semibold);
  margin-bottom: var(--space-2);
}

.status {
  flex: none;
  margin-left: auto;
  padding: 1px var(--space-8);
  border: 1px solid var(--border);
  border-radius: var(--radius-full);
  color: var(--text-weak);
  font-size: var(--fs-2xs);
  font-weight: var(--weight-semibold);
  white-space: nowrap;
}

.status.danger {
  border-color: var(--danger-border);
  background: var(--danger-soft);
  color: var(--danger);
}

.danger-card {
  border-color: var(--danger-border);
}

.mono {
  font-family: var(--font-mono);
  overflow-wrap: anywhere;
}
</style>
