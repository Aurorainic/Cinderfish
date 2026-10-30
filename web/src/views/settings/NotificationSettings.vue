<script setup lang="ts">
import { computed, ref } from "vue";
import { api, ApiError } from "../../api/client";
import AppIcon from "../../components/AppIcon.vue";
import { session } from "../../stores/session";
import { setUnreadNotifications, streamState, unreadNotifications } from "../../stores/streaming";
import SettingsPage from "./SettingsPage.vue";

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
    <div v-if="!me" class="card">
      <h2 class="card-title">需要登录</h2>
      <p class="muted small">登录之后才会显示通知相关的设置。</p>
      <div class="gate-actions">
        <RouterLink to="/login" class="button small">登录</RouterLink>
        <RouterLink to="/signup" class="button ghost small">注册</RouterLink>
      </div>
    </div>

    <template v-else>
      <section class="card">
        <h2 class="card-title">通知</h2>
        <p class="muted small">
          通知的类型筛选在通知页顶部；服务器暂未提供保存通知偏好的接口，因此这里只保留已读状态的操作。
        </p>

        <p v-if="error" class="notice error">{{ error }}</p>
        <p v-if="done" class="notice info">已全部标为已读</p>

        <div class="row">
          <div class="row-text">
            <span class="row-label">未读通知</span>
            <span class="row-desc">
              {{ unreadNotifications ? `${unreadNotifications} 条未读` : "没有未读通知" }}
            </span>
          </div>
          <button
            type="button"
            class="button ghost small"
            :disabled="marking || unreadNotifications === 0"
            @click="markAllRead"
          >
            <AppIcon name="check" />
            {{ marking ? "处理中…" : "全部标为已读" }}
          </button>
        </div>
      </section>

      <section class="card">
        <h2 class="card-title">实时连接</h2>
        <p class="muted small">
          新通知通过服务器推送到达，不轮询。连接状态由外壳自动维护，无需手动设置。
        </p>
        <div class="row">
          <div class="row-text">
            <span class="row-label">推送状态</span>
            <span class="row-desc">当前：{{ connectionLabel }}</span>
          </div>
          <span class="status" :class="{ live: streamState === 'open' }">{{ connectionLabel }}</span>
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

.row {
  display: flex;
  align-items: center;
  gap: var(--space-12);
  margin-top: var(--space-12);
  padding-top: var(--space-12);
  border-top: 1px solid var(--divider);
}

.row-text {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.row-label {
  color: var(--text-strong);
  font-size: var(--fs-sm);
  font-weight: var(--weight-semibold);
}

.row-desc {
  color: var(--text-weak);
  font-size: var(--fs-xs);
}

.row .button {
  margin-left: auto;
  flex: none;
}

.row .button .icon {
  width: 14px;
  height: 14px;
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

.status.live {
  border-color: color-mix(in srgb, var(--success) 40%, var(--border));
  background: var(--success-soft);
  color: var(--success);
}
</style>
