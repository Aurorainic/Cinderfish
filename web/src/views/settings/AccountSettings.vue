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
import MkButton from "../../components/ui/MkButton.vue";
import MkKeyValue from "../../components/ui/MkKeyValue.vue";
import MkTime from "../../components/ui/MkTime.vue";
import UserAvatar from "../../components/UserAvatar.vue";
import { i18n } from "../../i18n";
import { isAdmin, session, signout } from "../../stores/session";
import SettingsPage from "./SettingsPage.vue";
import SettingsSection from "./SettingsSection.vue";

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
    <div v-if="!me" class="_panel gate-panel">
      <h2 class="gate-title">需要登录</h2>
      <p class="settings-hint">登录之后才会显示账户与会话相关的设置。</p>
      <div class="settings-actions">
        <MkButton link to="/login" primary inline>登录</MkButton>
        <MkButton link to="/signup" inline>注册</MkButton>
      </div>
    </div>

    <template v-else>
      <SettingsSection title="账户信息" persist-key="settings-account-info">
        <div class="account-head">
          <UserAvatar :user="me" :size="64" />
          <div class="account-text">
            <div class="account-name">
              {{ me.name || me.username }}
              <span v-if="isAdmin" class="badge admin">管理员</span>
            </div>
            <div class="settings-hint">@{{ me.username }}</div>
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

        <div class="settings-actions">
          <MkButton link inline :to="`/@${me.username}`">
            <AppIcon name="user" />
            {{ i18n.ts.profile }}
          </MkButton>
          <MkButton link inline to="/settings/profile">
            {{ i18n.ts.editProfile }}
          </MkButton>
        </div>
      </SettingsSection>

      <SettingsSection title="安全" persist-key="settings-account-security">
        <div class="setting-row">
          <div class="setting-text">
            <div class="setting-label">密码</div>
            <p class="settings-hint">网页端尚未开放自助修改密码，需要时请由管理员在服务器上处理。</p>
          </div>
          <span class="status">未开放</span>
        </div>

        <div class="setting-row">
          <div class="setting-text">
            <div class="setting-label">两步验证</div>
            <p class="settings-hint">服务器尚未提供两步验证接口。</p>
          </div>
          <span class="status">未开放</span>
        </div>

        <div class="setting-row">
          <div class="setting-text">
            <div class="setting-label">登录历史</div>
            <p class="settings-hint">服务器尚未提供 i/signin-history 接口。</p>
          </div>
          <span class="status">未开放</span>
        </div>
      </SettingsSection>

      <SettingsSection title="会话" persist-key="settings-account-session">
        <p class="settings-hint">退出后这台设备上的访问令牌会被清除。</p>
        <div class="settings-actions">
          <MkButton danger inline @click="handleSignout">
            <AppIcon name="logout" />
            退出登录
          </MkButton>
        </div>
      </SettingsSection>

      <SettingsSection title="危险操作" persist-key="settings-account-danger">
        <div class="setting-row">
          <div class="setting-text">
            <div class="setting-label">注销账户</div>
            <p class="settings-hint">
              服务器尚未提供账户注销接口，笔记与媒体的删除也无法在网页端完成；需要时请联系实例管理员。
            </p>
          </div>
          <span class="status danger">不可用</span>
        </div>
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
  gap: var(--space-8);
  color: var(--fgHighlighted);
  font-size: 1em;
  font-weight: 700;
  overflow-wrap: anywhere;
}

.badge {
  padding: 1px var(--space-8);
  border: 1px solid var(--accent);
  border-radius: var(--radius-full);
  background: var(--accent-soft);
  color: var(--accent);
  font-size: 0.72em;
  font-weight: 700;
  letter-spacing: 0.02em;
}

.key-values {
  display: flex;
  flex-direction: column;
  gap: var(--space-10);
}

.settings-actions .icon {
  width: 15px;
  height: 15px;
  margin-inline-end: 4px;
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
  padding-top: 0;
}

.setting-text {
  min-width: 0;
}

.setting-label {
  margin-bottom: 2px;
  color: var(--fgHighlighted);
  font-size: 0.9em;
  font-weight: 600;
}

.status {
  flex: none;
  margin-left: auto;
  padding: 1px var(--space-8);
  border: 1px solid var(--divider);
  border-radius: var(--radius-full);
  color: var(--fgTransparentWeak);
  font-size: 0.72em;
  font-weight: 600;
  white-space: nowrap;
}

.status.danger {
  border-color: var(--error);
  background: color-mix(in srgb, var(--error) 15%, transparent);
  color: var(--error);
}

.mono {
  font-family: var(--font-mono);
  overflow-wrap: anywhere;
}
</style>
