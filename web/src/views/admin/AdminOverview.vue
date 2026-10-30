<script setup lang="ts">
import { computed, onMounted } from "vue";
import AppIcon from "../../components/AppIcon.vue";
import { instance, loadInstance } from "../../stores/instance";
import { federationLabel, registrationLabel, storageLabel } from "./meta";

const meta = computed(() => instance.meta);

const facts = computed(() => {
  const current = meta.value;
  if (!current) return [];
  return [
    { label: "实例名称", value: current.name || "—" },
    { label: "域名", value: current.uri || "—" },
    { label: "软件版本", value: current.version || "—" },
    { label: "笔记长度上限", value: `${current.maxNoteTextLength} 字符` },
    { label: "联邦模式", value: federationLabel(current) },
    { label: "自助注册", value: registrationLabel(current) },
    { label: "媒体存储", value: storageLabel(current) },
  ];
});

function reloadMeta() {
  instance.error = "";
  void loadInstance();
}

onMounted(loadInstance);
</script>

<template>
  <section class="card">
    <div class="section-head">
      <h2>实例信息</h2>
      <button
        v-if="!meta && instance.error"
        class="button ghost small"
        type="button"
        @click="reloadMeta"
      >
        重新加载
      </button>
    </div>

    <p v-if="instance.error" class="notice error" role="alert">{{ instance.error }}</p>
    <p v-else-if="!meta" class="hint">正在读取实例信息…</p>

    <template v-else>
      <p v-if="meta.description" class="description">{{ meta.description }}</p>
      <dl class="facts">
        <div v-for="fact in facts" :key="fact.label" class="fact">
          <dt>{{ fact.label }}</dt>
          <dd>{{ fact.value }}</dd>
        </div>
      </dl>
    </template>
  </section>

  <section class="card">
    <h2 class="section-title">快捷入口</h2>
    <p class="muted small section-desc">日常管理常用的几个分区。</p>
    <div class="links">
      <RouterLink :to="{ path: '/admin', query: { tab: 'users' } }" class="button ghost small">
        <AppIcon name="user" />
        用户管理
      </RouterLink>
      <RouterLink :to="{ path: '/admin', query: { tab: 'settings' } }" class="button ghost small">
        <AppIcon name="settings" />
        联邦与注册
      </RouterLink>
      <RouterLink :to="{ path: '/admin', query: { tab: 'files' } }" class="button ghost small">
        <AppIcon name="image" />
        文件
      </RouterLink>
    </div>
  </section>
</template>

<style scoped>
.section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-10);
  margin-bottom: var(--space-10);
}

.section-head h2,
.section-title {
  font-size: var(--fs-md);
}

.section-desc {
  margin: var(--space-6) 0 var(--space-14);
}

.description {
  margin-bottom: var(--space-14);
  font-size: var(--fs-sm);
  color: var(--text-weak);
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

.facts {
  display: flex;
  flex-direction: column;
  margin: 0;
}

.fact {
  display: grid;
  grid-template-columns: 112px minmax(0, 1fr);
  gap: var(--space-12);
  padding: var(--space-8) 0;
  border-top: 1px solid var(--divider);
  font-size: var(--fs-sm);
}

.fact:first-child {
  border-top: 0;
}

dt {
  color: var(--text-weak);
}

dd {
  margin: 0;
  font-weight: var(--weight-semibold);
  color: var(--text);
  overflow-wrap: anywhere;
}

.links {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-8);
}

.links .icon {
  width: 16px;
  height: 16px;
}
</style>
