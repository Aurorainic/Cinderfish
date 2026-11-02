<script setup lang="ts">
import { computed, ref } from "vue";
import { api, ApiError } from "../../api/client";
import AppIcon from "../../components/AppIcon.vue";
import MkButton from "../../components/ui/MkButton.vue";
import MkInfo from "../../components/ui/MkInfo.vue";
import { session } from "../../stores/session";
import { setUnreadNotifications, streamState, unreadNotifications } from "../../stores/streaming";
import SettingsPage from "./SettingsPage.vue";
import SettingsSection from "./SettingsSection.vue";

const me = computed(() => session.me);
const marking = ref(false);
const error = ref("");
const done = ref(false);

const connection: Record<typeof streamState.value, string> = {
  idle: "未连接",
  connecting: "连接中…",
  open: "已连接",
  closed: "已断开",
};

const connectionLabel = computed(() => connection[streamState.value]);

function messageOf(err: unknown): string {
  if (err instanceof ApiError) return err.message || "标记已读失败";
  if (err instanceof Error) return err.message || "标记已读失败";
  return "标记已读失败";
}

async function markAllRead() {
  if (marking.value || unreadNotifications.value === 0) return;
  marking.value = true;
  error.value = "";
  done.value = false;
  try {
    await api.markAllNotificationsRead();
    setUnreadNotifications(0);
    const current = session.me;
    if (current) {
      current.unreadNotificationsCount = 0;
      current.hasUnreadNotification = false;
    }
    done.value = true;
  } catch (err) {
    error.value = messageOf(err);
  } finally {
    marking.value = false;
  }
}
</script>

<template>
  <SettingsPage>
    <div v-if="!me" class="_panel gate-panel">
      <h2 class="gate-title">需要登录</h2>
      <p class="settings-hint">登录之后才会显示通知相关的设置。</p>
      <div class="settings-actions">
        <MkButton link to="/login" primary inline>登录</MkButton>
        <MkButton link to="/signup" inline>注册</MkButton>
      </div>
    </div>

    <template v-else>
      <SettingsSection title="已读状态" persist-key="settings-notifications-read">
        <p class="settings-hint">
          通知的类型筛选在通知页顶部；服务器暂未提供保存通知偏好的接口，因此这里只保留已读状态的操作。
        </p>

        <MkInfo v-if="error" warn>{{ error }}</MkInfo>
        <MkInfo v-if="done">已全部标为已读</MkInfo>

        <div class="row">
          <div class="row-text">
            <span class="row-label">未读通知</span>
            <span class="row-desc">
              {{ unreadNotifications ? `${unreadNotifications} 条未读` : "没有未读通知" }}
            </span>
          </div>
          <MkButton
            class="action"
            inline
            :disabled="marking || unreadNotifications === 0"
            @click="markAllRead"
          >
            <AppIcon name="check" />
            {{ marking ? "处理中…" : "全部标为已读" }}
          </MkButton>
        </div>
      </SettingsSection>

      <SettingsSection title="实时连接" persist-key="settings-notifications-stream">
        <p class="settings-hint">
          新通知通过服务器推送到达，不轮询。连接状态由外壳自动维护，无需手动设置。
        </p>
        <div class="row">
          <div class="row-text">
            <span class="row-label">推送状态</span>
            <span class="row-desc">当前：{{ connectionLabel }}</span>
          </div>
          <span class="status" :class="{ live: streamState === 'open' }">{{ connectionLabel }}</span>
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

.row {
  display: flex;
  align-items: center;
  gap: var(--space-12);
  padding-top: var(--space-10);
  border-top: 1px solid var(--divider);
}

.row-text {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.row-label {
  color: var(--fgHighlighted);
  font-size: 0.9em;
  font-weight: 600;
}

.row-desc {
  color: var(--fgTransparentWeak);
  font-size: 0.8em;
}

.action {
  flex: none;
  margin-inline-start: auto;
}

.action .icon {
  width: 14px;
  height: 14px;
  margin-inline-end: 4px;
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

.status.live {
  border-color: color-mix(in srgb, var(--success) 40%, var(--divider));
  background: color-mix(in srgb, var(--success) 15%, transparent);
  color: var(--success);
}
</style>
