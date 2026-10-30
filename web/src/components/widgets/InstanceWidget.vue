<script setup lang="ts">
import { computed, onMounted } from "vue";
import { instance, loadInstance } from "../../stores/instance";
import { session } from "../../stores/session";
import WidgetCard from "./WidgetCard.vue";

defineProps<{ widgetProps?: Record<string, unknown> }>();

const fallbackDescription =
  "一个个人或极小规模的联邦社交实例。后端遵循 Misskey API，通信层使用 ActivityPub。";

const federationLabels: Record<string, string> = {
  all: "开放联邦",
  specified: "仅白名单",
  none: "关闭联邦",
};

const name = computed(() => instance.meta?.name ?? "Cinderfish");
const description = computed(() => instance.meta?.description || fallbackDescription);
const version = computed(() => (instance.meta?.version ? `v${instance.meta.version}` : ""));
const federation = computed(() => {
  const mode = instance.meta?.federation;
  return mode ? (federationLabels[mode] ?? mode) : "";
});

// The shell loads the instance once; a widget rendered on its own (drawer)
// still gets metadata because the loader caches both the result and the
// in-flight request.
onMounted(() => {
  void loadInstance();
});
</script>

<template>
  <WidgetCard :title="name" icon="flame" hint="个人联邦实例">
    <p class="desc">{{ description }}</p>

    <div v-if="version || federation" class="facts">
      <span v-if="version" class="fact">{{ version }}</span>
      <span v-if="federation" class="fact">{{ federation }}</span>
    </div>

    <p v-if="instance.error" class="notice error">{{ instance.error }}</p>

    <div class="actions">
      <template v-if="session.me">
        <RouterLink :to="`/@${session.me.username}`" class="button ghost small">我的主页</RouterLink>
        <RouterLink to="/settings" class="button small">设置</RouterLink>
      </template>
      <template v-else>
        <RouterLink to="/login" class="button ghost small">登录</RouterLink>
        <RouterLink to="/signup" class="button small">注册</RouterLink>
      </template>
    </div>
  </WidgetCard>
</template>

<style scoped>
.desc {
  margin: 0;
  color: var(--muted);
  font-size: 0.85rem;
  line-height: 1.6;
}

.facts {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 10px;
}

.fact {
  padding: 1px 8px;
  border: 1px solid var(--border);
  border-radius: 999px;
  color: var(--muted);
  font-size: 0.72rem;
  line-height: 1.6;
}

.notice {
  margin-top: 10px;
}

.actions {
  display: flex;
  gap: 8px;
  margin-top: 14px;
}
</style>
