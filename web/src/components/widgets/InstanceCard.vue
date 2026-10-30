<script setup lang="ts">
import AppIcon from "../AppIcon.vue";
import { instance } from "../../stores/instance";
import { session } from "../../stores/session";

// The widget column's default stack: instance identity, the action pair
// (login/signup or profile/settings) and the protocol links. It was the
// hard-coded right column of the old shell; keeping it in one component lets
// AppShell expose the column as a `#widgets` slot.
const fallbackDescription =
  "一个个人或极小规模的联邦社交实例。后端遵循 Misskey API，通信层使用 ActivityPub。";
</script>

<template>
  <section class="card instance-card">
    <div class="instance-head">
      <span class="instance-logo"><AppIcon name="flame" /></span>
      <div class="instance-title">
        <h2>{{ instance.meta?.name ?? "Cinderfish" }}</h2>
        <p class="muted small">个人联邦实例</p>
      </div>
    </div>
    <p class="instance-desc muted small">
      {{ instance.meta?.description || fallbackDescription }}
    </p>
    <div class="instance-actions">
      <template v-if="session.me">
        <RouterLink :to="`/@${session.me.username}`" class="button ghost small">我的主页</RouterLink>
        <RouterLink to="/settings" class="button small">设置</RouterLink>
      </template>
      <template v-else>
        <RouterLink to="/login" class="button ghost small">登录</RouterLink>
        <RouterLink to="/signup" class="button small">注册</RouterLink>
      </template>
    </div>
  </section>

  <section class="card">
    <h3 class="card-title">关于</h3>
    <ul class="link-list">
      <li><RouterLink to="/explore">探索时间线</RouterLink></li>
      <li><a :href="`/.well-known/nodeinfo`" target="_blank" rel="noopener">NodeInfo</a></li>
      <li><a :href="`/.well-known/host-meta`" target="_blank" rel="noopener">HostMeta</a></li>
    </ul>
  </section>

  <p class="side-foot muted small">Cinderfish · AGPL-3.0</p>
</template>

<style scoped>
.instance-head {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 12px;
}

.instance-logo {
  display: grid;
  place-items: center;
  flex: none;
  width: 38px;
  height: 38px;
  border-radius: var(--radius-lg);
  color: var(--on-accent);
  background: var(--accent);
}

.instance-logo .icon {
  width: 20px;
  height: 20px;
}

.instance-title {
  min-width: 0;
}

.instance-title h2 {
  font-size: 1.05rem;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.instance-desc {
  margin: 0;
  line-height: 1.6;
}

.instance-actions {
  display: flex;
  gap: 8px;
  margin-top: 16px;
}

.link-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
  color: var(--muted);
}

.link-list a:hover {
  color: var(--accent);
}

.side-foot {
  text-align: center;
}
</style>
