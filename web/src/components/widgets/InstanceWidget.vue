<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { instance, loadInstance, reloadInstance } from "../../stores/instance";
import { session } from "../../stores/session";
import { NAV_ITEM_DEFS } from "../../nav";
import AppIcon from "../AppIcon.vue";
import WidgetCard from "./WidgetCard.vue";

defineProps<{ widgetProps?: Record<string, unknown> }>();

const federationLabels: Record<string, string> = {
  all: "开放联邦",
  specified: "仅白名单",
  none: "关闭联邦",
};

const name = computed(() => instance.meta?.name ?? "Cinderfish");
// The card only ever prints what the instance reported. A missing description
// reads as missing; there is no stand-in blurb for an instance nobody wrote one
// for, and nothing is shown while the first read is still running.
const description = computed(() => instance.meta?.description ?? "");
const version = computed(() => (instance.meta?.version ? `v${instance.meta.version}` : ""));
const federation = computed(() => {
  const mode = instance.meta?.federation;
  return mode ? (federationLabels[mode] ?? mode) : "";
});

const refreshing = ref(false);
const refreshError = ref("");
// A failed refresh keeps the metadata that is already loaded, so the only
// failure worth reporting here is one the widget has no data behind: either the
// refresh it just ran or the shared first load.
const failure = computed(() => refreshError.value || instance.error);
const loading = computed(() => instance.meta === null && !failure.value);

async function refresh(): Promise<void> {
  if (refreshing.value) return;
  refreshing.value = true;
  refreshError.value = "";
  try {
    await reloadInstance();
  } catch (err) {
    refreshError.value = err instanceof Error && err.message ? err.message : String(err);
  } finally {
    refreshing.value = false;
  }
}

// The shell loads the instance once; a widget rendered on its own (drawer)
// still gets metadata because the loader caches both the result and the
// in-flight request.
onMounted(() => {
  void loadInstance();
});
</script>

<template>
  <WidgetCard :title="name" icon="flame">
    <template #actions>
      <button
        class="refresh"
        type="button"
        title="刷新实例信息"
        aria-label="刷新实例信息"
        :disabled="refreshing || loading"
        @click="refresh"
      >
        <AppIcon name="renote" />
      </button>
    </template>

    <p v-if="loading" class="desc state" aria-live="polite">正在读取实例信息…</p>

    <template v-else>
      <p v-if="description" class="desc">{{ description }}</p>
      <p v-else-if="instance.meta" class="desc state">这个实例还没有填写简介。</p>

      <div v-if="version || federation" class="facts">
        <span v-if="version" class="fact">{{ version }}</span>
        <span v-if="federation" class="fact">{{ federation }}</span>
      </div>
    </template>

    <p v-if="failure" class="notice error failure" :title="failure">
      <span>{{ instance.meta ? "实例信息刷新失败" : "实例信息加载失败" }}</span>
      <button type="button" class="button ghost small" :disabled="refreshing" @click="refresh">重试</button>
      <span class="failure-detail">{{ failure }}</span>
    </p>

    <div class="actions">
      <template v-if="session.me">
        <RouterLink :to="`/@${session.me.username}`" class="button ghost small">我的主页</RouterLink>
        <RouterLink :to="NAV_ITEM_DEFS.settings.to" class="button small">设置</RouterLink>
      </template>
      <template v-else>
        <RouterLink to="/login" class="button ghost small">登录</RouterLink>
        <RouterLink to="/signup" class="button small">注册</RouterLink>
      </template>
    </div>
  </WidgetCard>
</template>

<style scoped>
.refresh {
  display: grid;
  place-items: center;
  width: 26px;
  height: 26px;
  padding: 0;
  border: 1px solid var(--border);
  border-radius: 50%;
  background: transparent;
  color: var(--accent);
  cursor: pointer;
  transition: color 0.15s ease, border-color 0.15s ease, opacity 0.15s ease;
}

.refresh:hover:not(:disabled) {
  border-color: var(--accent);
}

.refresh:disabled {
  color: var(--muted);
  opacity: 0.6;
  cursor: default;
}

.refresh .icon {
  width: 13px;
  height: 13px;
}

.desc {
  margin: 0;
  color: var(--muted);
  font-size: 0.85rem;
  line-height: 1.6;
}

/* A loading or "no description" line is a state, not instance prose. */
.desc.state {
  color: var(--text-faint);
  font-style: italic;
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

.failure {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 10px;
}

.failure-detail {
  flex-basis: 100%;
  color: var(--text-faint);
  font-size: 0.72rem;
  overflow-wrap: anywhere;
}

.actions {
  display: flex;
  gap: 8px;
  margin-top: 14px;
}
</style>
