<script setup lang="ts">
import { computed, onMounted } from "vue";
import AppIcon from "../../components/AppIcon.vue";
import MkButton from "../../components/ui/MkButton.vue";
import MkInfo from "../../components/ui/MkInfo.vue";
import MkKeyValue from "../../components/ui/MkKeyValue.vue";
import MkLoading from "../../components/ui/MkLoading.vue";
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
  <section class="_card">
    <div class="_title section-head">
      <h2>实例信息</h2>
      <MkButton v-if="!meta && instance.error" @click="reloadMeta">重新加载</MkButton>
    </div>

    <div class="_content">
      <MkInfo v-if="instance.error" warn role="alert">{{ instance.error }}</MkInfo>
      <MkLoading v-else-if="!meta" />

      <template v-else>
        <p v-if="meta.description" class="description">{{ meta.description }}</p>
        <div class="facts">
          <MkKeyValue v-for="fact in facts" :key="fact.label" class="fact">
            <template #key>{{ fact.label }}</template>
            <template #value>
              <span class="fact-value">{{ fact.value }}</span>
            </template>
          </MkKeyValue>
        </div>
      </template>
    </div>
  </section>

  <section class="_card">
    <div class="_title">快捷入口</div>
    <div class="_content">
      <p class="_caption section-desc">日常管理常用的几个分区。</p>
      <div class="links">
        <MkButton link to="/admin?tab=users">
          <AppIcon name="user" />
          用户管理
        </MkButton>
        <MkButton link to="/admin?tab=settings">
          <AppIcon name="settings" />
          联邦与注册
        </MkButton>
        <MkButton link to="/admin?tab=files">
          <AppIcon name="image" />
          文件
        </MkButton>
      </div>
    </div>
  </section>
</template>

<style scoped>
.section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.section-head h2 {
  font-size: 1em;
}

.section-desc {
  margin-block-end: var(--margin);
}

.description {
  margin-block-end: var(--margin);
  font-size: 0.9em;
  color: var(--fgTransparentWeak);
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

.facts {
  display: flex;
  flex-direction: column;
  margin: 0;
}

.fact + .fact {
  margin-block-start: 0.6em;
  padding-block-start: 0.6em;
  border-block-start: solid 0.5px var(--hairline);
}

.fact-value {
  font-weight: 600;
  overflow-wrap: anywhere;
}

.links {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.links .icon {
  inline-size: 16px;
  block-size: 16px;
}
</style>
