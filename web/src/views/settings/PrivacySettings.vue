<script setup lang="ts">
import { computed, onMounted } from "vue";
import { instance, loadInstance } from "../../stores/instance";
import { session } from "../../stores/session";
import SettingsPage from "./SettingsPage.vue";

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

const visibilityLabels: Record<string, string> = {
  public: "公开",
  followers: "仅关注者",
};

const federation = computed(() => {
  const mode = instance.meta?.federation;
  if (!mode) return "未知";
  return federationLabels[mode] ?? mode;
});

const me = computed(() => session.me);

function visibility(value: unknown): string {
  if (typeof value !== "string" || !value) return "未知";
  return visibilityLabels[value] ?? value;
}

// The shell loads the instance too; the loader caches both the result and the
// in-flight request, so a direct visit costs one request.
onMounted(() => {
  void loadInstance();
});
</script>

<template>
  <SettingsPage>
    <section class="card">
      <h2 class="card-title">联邦</h2>
      <p class="muted small">
        联邦模式由实例管理员在服务端配置，会同时影响你能看到的远端内容和远端实例能否投递到这里。
      </p>
      <div class="row">
        <span class="row-label">当前模式</span>
        <span class="value">{{ federation }}</span>
      </div>
      <p v-if="instance.error" class="notice error">读取实例信息失败：{{ instance.error }}</p>
    </section>

    <section v-if="me" class="card">
      <h2 class="card-title">可见性</h2>
      <p class="muted small">
        当前取值来自服务器，网页端尚不能修改：关注者与关注中的列表权限需要在服务端或后续版本的设置接口里调整。
      </p>
      <div class="row">
        <span class="row-label">关注者列表</span>
        <span class="value">{{ visibility(me.followersVisibility) }}</span>
      </div>
      <div class="row">
        <span class="row-label">关注中列表</span>
        <span class="value">{{ visibility(me.followingVisibility) }}</span>
      </div>
    </section>

    <section class="card">
      <h2 class="card-title">尚不可用</h2>
      <p class="muted small">
        以下能力服务器还没有提供接口，这里不做任何前端模拟，等接口落地后再开放：
      </p>
      <ul class="todo">
        <li>屏蔽、静音与实例级屏蔽</li>
        <li>关注审批（锁定的账户）</li>
        <li>保存默认笔记可见性（目前每条笔记在编辑框里单独选择）</li>
        <li>私信与仅指定用户可见的偏好</li>
      </ul>
    </section>
  </SettingsPage>
</template>

<style scoped>
.row {
  display: flex;
  align-items: center;
  gap: var(--space-12);
  margin-top: var(--space-10);
  padding-top: var(--space-10);
  border-top: 1px solid var(--divider);
}

.row-label {
  color: var(--text);
  font-size: var(--fs-sm);
}

.value {
  margin-left: auto;
  padding: 1px var(--space-10);
  border: 1px solid var(--border);
  border-radius: var(--radius-full);
  background: var(--surface-inset);
  color: var(--text-weak);
  font-size: var(--fs-2xs);
  font-weight: var(--weight-semibold);
  white-space: nowrap;
}

.todo {
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
  margin: var(--space-10) 0 0;
  padding-left: var(--space-20);
  color: var(--text-weak);
  font-size: var(--fs-sm);
}
</style>
