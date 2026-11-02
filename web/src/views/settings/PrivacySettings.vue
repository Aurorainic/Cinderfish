<script setup lang="ts">
import { computed, onMounted } from "vue";
import MkInfo from "../../components/ui/MkInfo.vue";
import { instance, loadInstance } from "../../stores/instance";
import { session } from "../../stores/session";
import SettingsPage from "./SettingsPage.vue";
import SettingsSection from "./SettingsSection.vue";

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
    <SettingsSection title="联邦" persist-key="settings-privacy-federation">
      <p class="settings-hint">
        联邦模式由实例管理员在服务端配置，会同时影响你能看到的远端内容和远端实例能否投递到这里。
      </p>
      <div class="row">
        <span class="row-label">当前模式</span>
        <span class="value">{{ federation }}</span>
      </div>
      <MkInfo v-if="instance.error" warn>读取实例信息失败：{{ instance.error }}</MkInfo>
    </SettingsSection>

    <SettingsSection v-if="me" title="可见性" persist-key="settings-privacy-visibility">
      <p class="settings-hint">
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
    </SettingsSection>

    <SettingsSection title="尚不可用" persist-key="settings-privacy-unavailable" :expanded="false">
      <p class="settings-hint">
        以下能力服务器还没有提供接口，这里不做任何前端模拟，等接口落地后再开放：
      </p>
      <ul class="todo">
        <li>屏蔽、静音与实例级屏蔽</li>
        <li>关注审批（锁定的账户）</li>
        <li>保存默认笔记可见性（目前每条笔记在编辑框里单独选择）</li>
        <li>私信与仅指定用户可见的偏好</li>
      </ul>
    </SettingsSection>
  </SettingsPage>
</template>

<style scoped>
.row {
  display: flex;
  align-items: center;
  gap: var(--space-12);
  padding-top: var(--space-10);
  border-top: 1px solid var(--divider);
}

.row-label {
  color: var(--fg);
  font-size: 0.9em;
}

.value {
  margin-left: auto;
  padding: 1px var(--space-10);
  border: 1px solid var(--divider);
  border-radius: var(--radius-full);
  background: var(--panelHighlight);
  color: var(--fgTransparentWeak);
  font-size: 0.72em;
  font-weight: 600;
  white-space: nowrap;
}

.todo {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin: 0;
  padding-left: var(--space-20);
  color: var(--fgTransparentWeak);
  font-size: 0.9em;
  line-height: 1.5;
}
</style>
