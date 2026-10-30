<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from "vue";
import { useRouter } from "vue-router";
import type { UserLite } from "../../api/types";
import AppIcon from "../AppIcon.vue";
import UserAvatar from "../UserAvatar.vue";

type IconName = InstanceType<typeof AppIcon>["$props"]["name"];
type AvatarUser = Pick<UserLite, "username" | "name" | "avatarUrl">;

interface PageHeaderTab {
  key: string;
  label: string;
  icon?: IconName;
  locked?: boolean;
}

const props = withDefaults(
  defineProps<{
    title?: string;
    subtitle?: string;
    icon?: IconName;
    back?: boolean;
    backTo?: string;
    tabs?: readonly (string | PageHeaderTab)[] | null;
    avatar?: AvatarUser | null;
    accountAvatar?: AvatarUser | null;
  }>(),
  {
    title: "",
    subtitle: "",
    back: false,
    backTo: "/",
    tabs: null,
    avatar: null,
    accountAvatar: null,
  },
);

const tab = defineModel<string>("tab");
const emit = defineEmits<{ account: [event: MouseEvent] }>();

const router = useRouter();

const rootEl = ref<HTMLElement | null>(null);
const navEl = ref<HTMLElement | null>(null);
const narrow = ref(false);
const tabEls = new Map<string, HTMLElement>();
const marker = ref({ x: 0, width: 0, visible: false });

const tabsList = computed<PageHeaderTab[]>(() =>
  (props.tabs ?? []).map((item) => (typeof item === "string" ? { key: item, label: item } : item)),
);

const markerStyle = computed(() => ({
  width: `${marker.value.width}px`,
  transform: `translateX(${marker.value.x}px)`,
  opacity: marker.value.visible ? "1" : "0",
}));

let observer: ResizeObserver | null = null;

function measure() {
  const host = rootEl.value?.parentElement ?? rootEl.value;
  if (host) narrow.value = host.offsetWidth < 500;
}

function setTabEl(key: string, el: unknown) {
  if (el instanceof HTMLElement) tabEls.set(key, el);
  else tabEls.delete(key);
}

function syncMarker() {
  const nav = navEl.value;
  const active = tab.value ? tabEls.get(tab.value) : undefined;
  if (!nav || !active) {
    marker.value = { x: 0, width: 0, visible: false };
    return;
  }
  const navRect = nav.getBoundingClientRect();
  const tabRect = active.getBoundingClientRect();
  marker.value = {
    x: tabRect.left - navRect.left + nav.scrollLeft,
    width: tabRect.width,
    visible: true,
  };
  if (nav.scrollWidth > nav.clientWidth + 1) {
    const target = active.offsetLeft + active.offsetWidth / 2 - nav.clientWidth / 2;
    nav.scrollTo({ left: Math.max(0, target), behavior: "smooth" });
  }
}

function select(key: string) {
  tab.value = key;
}

function goBack() {
  if (window.history.state?.back) router.back();
  else router.push(props.backTo);
}

watch([tab, tabsList], () => nextTick(syncMarker));
watch(narrow, () => nextTick(syncMarker));

onMounted(() => {
  measure();
  const host = rootEl.value?.parentElement ?? rootEl.value;
  if (host) {
    observer = new ResizeObserver(() => {
      measure();
      syncMarker();
    });
    observer.observe(host);
  }
  nextTick(syncMarker);
  document.fonts?.ready.then(syncMarker);
});

onBeforeUnmount(() => {
  observer?.disconnect();
  observer = null;
});
</script>

<template>
  <header ref="rootEl" class="page-header" :class="{ 'is-narrow': narrow, 'has-tabs': tabsList.length > 0 }">
    <div class="heading">
      <button v-if="back" type="button" class="round-button back" title="返回" aria-label="返回" @click="goBack">
        <AppIcon name="arrowLeft" />
      </button>
      <button
        v-if="narrow && accountAvatar"
        type="button"
        class="round-button account"
        title="账户"
        aria-label="账户"
        @click="emit('account', $event)"
      >
        <UserAvatar :user="accountAvatar" :size="28" />
      </button>
      <UserAvatar v-if="avatar" class="heading-avatar" :user="avatar" :size="28" />
      <slot name="icon">
        <AppIcon v-if="icon" class="heading-icon" :name="icon" />
      </slot>
      <div v-if="title || $slots.title" class="heading-text">
        <h1 class="title"><slot name="title">{{ title }}</slot></h1>
        <p v-if="subtitle && !narrow" class="subtitle">{{ subtitle }}</p>
      </div>
    </div>

    <nav
      v-if="tabsList.length"
      ref="navEl"
      class="tabs"
      role="tablist"
      aria-label="页面分页"
    >
      <button
        v-for="item in tabsList"
        :key="item.key"
        :ref="(el) => setTabEl(item.key, el)"
        type="button"
        role="tab"
        class="tab"
        :class="{ active: item.key === tab }"
        :aria-selected="item.key === tab"
        @click="select(item.key)"
      >
        <AppIcon v-if="item.icon" class="tab-icon" :name="item.icon" />
        <span class="tab-label">{{ item.label }}</span>
        <span v-if="item.locked" class="tab-lock" title="需要登录" aria-hidden="true">·</span>
      </button>
      <span class="highlight" :style="markerStyle" aria-hidden="true"></span>
    </nav>

    <div v-if="$slots.actions" class="actions">
      <slot name="actions" />
    </div>
  </header>
</template>

<style scoped>
.page-header {
  position: sticky;
  top: 0;
  z-index: 20;
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 52px;
  border-bottom: 1px solid var(--divider);
  background: color-mix(in srgb, var(--panel) 86%, transparent);
  -webkit-backdrop-filter: blur(14px) saturate(1.2);
  backdrop-filter: blur(14px) saturate(1.2);
}

@supports not ((backdrop-filter: blur(1px)) or (-webkit-backdrop-filter: blur(1px))) {
  .page-header {
    background: var(--panel);
  }
}

.heading {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 0 1 auto;
  min-width: 0;
  max-width: 45%;
}

.page-header:not(.has-tabs) .heading {
  max-width: none;
}

.heading-text {
  min-width: 0;
}

.title {
  margin: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 1.05rem;
  font-weight: 700;
  line-height: 1.25;
  color: var(--text-strong);
}

.subtitle {
  margin: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 0.76rem;
  font-weight: 500;
  line-height: 1.3;
  color: var(--muted);
}

.heading-icon {
  flex: none;
  width: 20px;
  height: 20px;
  color: var(--accent);
}

.heading-avatar {
  flex: none;
}

.round-button {
  display: inline-grid;
  place-items: center;
  flex: none;
  width: 34px;
  height: 34px;
  padding: 0;
  border: 0;
  border-radius: 50%;
  background: transparent;
  color: var(--muted);
  cursor: pointer;
  transition: color 0.15s ease, background 0.15s ease;
}

.round-button:hover {
  color: var(--text);
  background: var(--panel-2);
}

.round-button:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 2px;
}

.back .icon {
  width: 20px;
  height: 20px;
}

.tabs {
  position: relative;
  display: flex;
  align-items: stretch;
  align-self: stretch;
  flex: 0 1 auto;
  min-width: 0;
  overflow-x: auto;
  overflow-y: hidden;
  scrollbar-width: none;
  overscroll-behavior-x: contain;
}

.tabs::-webkit-scrollbar {
  display: none;
}

.tab {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  flex: none;
  padding: 0 12px;
  border: 0;
  background: transparent;
  color: var(--muted);
  font-size: 0.92rem;
  font-weight: 600;
  white-space: nowrap;
  cursor: pointer;
  transition: color 0.15s ease;
}

.tab:hover {
  color: var(--text);
}

.tab:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: -4px;
  border-radius: var(--radius-sm);
}

.tab.active {
  color: var(--accent);
}

.tab-icon {
  width: 17px;
  height: 17px;
}

.tab-lock {
  margin-inline-start: -2px;
  line-height: 1;
}

.highlight {
  position: absolute;
  bottom: 0;
  left: 0;
  height: 3px;
  border-radius: 999px;
  background: var(--accent);
  pointer-events: none;
  transition:
    transform 0.22s cubic-bezier(0, 0, 0.2, 1.2),
    width 0.22s ease,
    opacity 0.15s ease;
}

.actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 4px;
  flex: none;
  margin-inline-start: auto;
  min-width: 0;
}

.actions :deep(.icon-button) {
  width: 34px;
  height: 34px;
  border-color: transparent;
  background: transparent;
}

.page-header.is-narrow {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  grid-template-rows: auto auto;
  column-gap: 6px;
  row-gap: 0;
}

.is-narrow .heading {
  grid-column: 1;
  grid-row: 1;
  min-height: 48px;
  max-width: none;
}

.is-narrow .actions {
  grid-column: 2;
  grid-row: 1;
}

.is-narrow .tabs {
  grid-column: 1 / -1;
  grid-row: 2;
  height: 38px;
}

.is-narrow .tab {
  padding: 0 10px;
  font-size: 0.88rem;
}

.is-narrow .title {
  font-size: 1rem;
}
</style>
