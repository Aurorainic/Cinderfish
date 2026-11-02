<script setup lang="ts">
import { computed, onMounted } from "vue";
import MkInfo from "../../components/ui/MkInfo.vue";
import { instance, loadInstance } from "../../stores/instance";
import SettingsPage from "./SettingsPage.vue";
import SettingsSection from "./SettingsSection.vue";

// Misskey spells the federation modes all / specified / none; the public and
// allowlist spellings are accepted as aliases.
const federationLabels: Record<string, string> = {
  public: "公开联邦",
  all: "公开联邦",
  allowlist: "仅白名单",
  specified: "仅白名单",
  none: "不开放联邦",
};

const federationLabel = computed(() => {
  const mode = instance.meta?.federation;
  if (!mode) return "未知";
  return federationLabels[mode] ?? mode;
});

const registrationLabel = computed(() => {
  const meta = instance.meta;
  if (!meta) return "未知";
  const mode = meta.registrationMode;
  if (mode) {
    const labels: Record<string, string> = {
      open: "开放注册",
      invite: "仅限邀请",
      closed: "停止注册",
      admin: "仅管理员创建",
    };
    return labels[mode] ?? mode;
  }
  return meta.disableRegistration ? "停止注册" : "开放注册";
});

// The two discovery documents the Go server serves under /.well-known.
const links = [
  { label: "NodeInfo", href: "/.well-known/nodeinfo" },
  { label: "HostMeta", href: "/.well-known/host-meta" },
];

onMounted(() => {
  void loadInstance();
});
</script>

<template>
  <SettingsPage>
    <SettingsSection title="实例" persist-key="settings-about-instance">
      <dl v-if="instance.meta" class="meta-list">
        <div class="meta-row">
          <dt>名称</dt>
          <dd>{{ instance.meta.name }}</dd>
        </div>
        <div class="meta-row">
          <dt>地址</dt>
          <dd class="mono">{{ instance.meta.uri }}</dd>
        </div>
        <div class="meta-row">
          <dt>版本</dt>
          <dd>{{ instance.meta.version }}</dd>
        </div>
        <div class="meta-row">
          <dt>联邦模式</dt>
          <dd>{{ federationLabel }}</dd>
        </div>
        <div class="meta-row">
          <dt>注册方式</dt>
          <dd>{{ registrationLabel }}</dd>
        </div>
        <div class="meta-row">
          <dt>正文上限</dt>
          <dd>{{ instance.meta.maxNoteTextLength }} 字符</dd>
        </div>
      </dl>
      <MkInfo v-else-if="instance.error" warn>读取实例信息失败：{{ instance.error }}</MkInfo>
      <p v-else class="settings-hint">正在读取实例信息…</p>
    </SettingsSection>

    <SettingsSection title="协议入口" persist-key="settings-about-protocol">
      <ul class="links">
        <li v-for="link in links" :key="link.href">
          <a :href="link.href" target="_blank" rel="noopener">{{ link.label }}</a>
          <span class="mono settings-hint">{{ link.href }}</span>
        </li>
      </ul>
    </SettingsSection>

    <SettingsSection title="客户端" persist-key="settings-about-client">
      <p class="settings-hint">
        Cinderfish 前端使用 Vue 3 与 TypeScript 构建，界面遵循 Firefish / Iceshrimp 的信息密度；
        后端遵循 Misskey API 契约，通信层使用 ActivityPub。
      </p>
      <p class="license">Cinderfish · AGPL-3.0</p>
    </SettingsSection>
  </SettingsPage>
</template>

<style scoped>
.meta-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-8);
  margin: 0;
}

.meta-row {
  display: flex;
  gap: var(--space-12);
  font-size: 0.9em;
}

.meta-row dt {
  flex: none;
  width: 80px;
  color: var(--fgTransparentWeak);
}

.meta-row dd {
  margin: 0;
  min-width: 0;
  color: var(--fg);
}

.mono {
  font-family: var(--font-mono);
  overflow-wrap: anywhere;
}

.links {
  display: flex;
  flex-direction: column;
  gap: var(--space-8);
  margin: 0;
  padding: 0;
  list-style: none;
  font-size: 0.9em;
}

.links li {
  display: flex;
  align-items: baseline;
  gap: var(--space-10);
  min-width: 0;
}

.links a {
  flex: none;
  color: var(--link);
  font-weight: 600;
}

.links a:hover {
  color: var(--accent-hover);
}

.links .mono {
  color: var(--fgTransparentWeak);
  font-size: 0.8em;
}

.license {
  margin: 0;
  padding-top: var(--space-10);
  border-top: 1px solid var(--divider);
  color: var(--fgTransparentWeak);
  font-size: 0.8em;
  text-align: center;
}
</style>
