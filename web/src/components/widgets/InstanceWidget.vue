<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { instance, loadInstance, reloadInstance } from "../../stores/instance";
import { session } from "../../stores/session";
import { NAV_ITEM_DEFS } from "../../nav";
import AppIcon from "../AppIcon.vue";
import MkButton from "../ui/MkButton.vue";
import MkInfo from "../ui/MkInfo.vue";
import MkLoading from "../ui/MkLoading.vue";
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

    <MkLoading v-if="loading" mini />

    <template v-else>
      <p v-if="description" class="desc">{{ description }}</p>
      <p v-else-if="instance.meta" class="desc state">这个实例还没有填写简介。</p>

      <div v-if="version || federation" class="facts">
        <span v-if="version" class="fact">{{ version }}</span>
        <span v-if="federation" class="fact">{{ federation }}</span>
      </div>
    </template>

    <MkInfo v-if="failure" class="failure" warn>
      <div class="failure-body">
        <span>{{ instance.meta ? "实例信息刷新失败" : "实例信息加载失败" }}</span>
        <MkButton mini :disabled="refreshing" @click="refresh">重试</MkButton>
        <span class="failure-detail" :title="failure">{{ failure }}</span>
      </div>
    </MkInfo>

    <div class="actions">
      <template v-if="session.me">
        <MkButton link mini :to="`/@${session.me.username}`">我的主页</MkButton>
        <MkButton link mini :to="NAV_ITEM_DEFS.settings.to">设置</MkButton>
      </template>
      <template v-else>
        <MkButton link mini to="/login">登录</MkButton>
        <MkButton link mini to="/signup">注册</MkButton>
      </template>
    </div>
  </WidgetCard>
</template>

<style scoped>
.refresh {
  display: grid;
  place-items: center;
  inline-size: 26px;
  block-size: 26px;
  padding: 0;
  border: solid 0.5px var(--divider);
  border-radius: 50%;
  background: none;
  color: var(--accent);
  cursor: pointer;
  transition: color 0.1s ease, border-color 0.1s ease, opacity 0.1s ease;
}

.refresh:hover:not(:disabled) {
  border-color: var(--accent);
}

.refresh:disabled {
  color: var(--fgTransparentWeak);
  opacity: 0.6;
  cursor: default;
}

.refresh .icon {
  inline-size: 13px;
  block-size: 13px;
}

.desc {
  margin: 0;
  color: var(--fgTransparentWeak);
  font-size: 0.9em;
  line-height: 1.6;
}

/* A loading or "no description" line is a state, not instance prose. */
.desc.state {
  color: var(--fgTransparent);
  font-style: italic;
}

.facts {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-block-start: 10px;
}

.fact {
  padding-block: 1px;
  padding-inline: 8px;
  border: solid 0.5px var(--divider);
  border-radius: 999px;
  color: var(--fgTransparentWeak);
  font-size: 0.8em;
  line-height: 1.6;
}

.failure {
  margin-block-start: 10px;
}

.failure-body {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  min-inline-size: 0;
}

.failure-detail {
  flex-basis: 100%;
  opacity: 0.75;
  font-size: 0.8em;
  overflow-wrap: anywhere;
}

.actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-block-start: 14px;
}
</style>
