<script setup lang="ts">
// Renders the head of os.ts's menu queue. Firefish's MkMenu is the reference:
// a popover anchored to an element (popupMenu) or to the pointer (contextMenu),
// one level of `parent` children expanded inline, and `switch` items writing
// straight back to the ref they were given.
import { nextTick, onBeforeUnmount, onMounted, ref, shallowRef, unref } from "vue";
import { RouterLink } from "vue-router";
import { claimZIndex } from "../../lib/z-index";
import type { MenuRequest } from "../../os";
import type {
  InnerMenuItem,
  MenuButton,
  MenuParent,
  MenuSwitch,
  MenuUser,
} from "../../types/menu";
import AppIcon, { iconPaths, type AppIconName } from "../AppIcon.vue";
import UserAvatar from "../UserAvatar.vue";

const props = defineProps<{ request: MenuRequest }>();
const emit = defineEmits<{ close: [] }>();

const zIndex = claimZIndex("high");
const box = ref<HTMLElement | null>(null);
// shallowRef: a deep ref would unwrap the `ref` of a switch item into a plain
// boolean, and the menu must write back through the Ref the caller passed.
const items = shallowRef<InnerMenuItem[]>([]);
const loading = ref(true);
const expanded = shallowRef<MenuParent | null>(null);

const style = ref<{
  left: string;
  top: string;
  width?: string;
  maxHeight: string;
  zIndex: number;
}>({ left: "0px", top: "0px", maxHeight: "0px", zIndex });

// Ported markup still names Phosphor icons; map the ones whose name already
// matches a path here and drop the rest until the icon mapping table lands.
function iconName(icon?: string): AppIconName | null {
  if (!icon) return null;
  const name = icon.replace(/^ph-/, "");
  return name in iconPaths ? (name as AppIconName) : null;
}

function close(): void {
  emit("close");
}

function run(action: (ev: MouseEvent) => void, ev: MouseEvent): void {
  close();
  action(ev);
}

function activateButton(item: MenuButton, ev: MouseEvent): void {
  if (item.hidden) return;
  run(item.action, ev);
}

function activateUser(item: MenuUser, ev: MouseEvent): void {
  if (item.hidden) return;
  run(item.action, ev);
}

// Firefish's FormSwitch writes the model immediately and leaves the menu open,
// which is what the settings menus rely on.
function toggleSwitch(item: MenuSwitch): void {
  if (item.disabled) return;
  item.ref.value = !item.ref.value;
}

function isExpanded(item: MenuParent): boolean {
  return expanded.value === item;
}

function toggleParent(item: MenuParent): void {
  expanded.value = expanded.value === item ? null : item;
}

async function loadItems(): Promise<void> {
  const raw = unref(props.request.items);
  try {
    items.value = (await Promise.all(
      raw.map((item) => (item instanceof Promise ? item : Promise.resolve(item))),
    )) as InnerMenuItem[];
  } catch (err) {
    // A rejected async item must not leave the menu stuck in its loading state.
    console.warn("popupMenu: an async item failed to resolve", err);
    items.value = [];
  } finally {
    loading.value = false;
  }
}

function place(): void {
  const element = box.value;
  if (!element) return;
  const margin = 8;
  const width = props.request.placement.width ?? element.offsetWidth;
  const height = element.offsetHeight;
  const { x, y, attachX, attachY } = props.request.placement;
  let left = attachX === "left" ? x : attachX === "center" ? x - width / 2 : x - width;
  left = Math.max(margin, Math.min(left, window.innerWidth - width - margin));
  let top = attachY === "top" ? y : y - height;
  // Flip above the anchor when the menu would run off the bottom edge.
  if (top + height > window.innerHeight - margin) top = y - height;
  top = Math.max(margin, Math.min(top, window.innerHeight - height - margin));
  const maxHeight = Math.max(120, window.innerHeight - top - margin);
  style.value = {
    left: `${left}px`,
    top: `${top}px`,
    width: `${width}px`,
    maxHeight: `${maxHeight}px`,
    zIndex,
  };
}

function focusFirst(): void {
  box.value?.querySelector<HTMLElement>(".item")?.focus();
}

function onKeydown(event: KeyboardEvent): void {
  if (
    event.key !== "ArrowDown" &&
    event.key !== "ArrowUp" &&
    event.key !== "Home" &&
    event.key !== "End"
  ) {
    return;
  }
  const element = box.value;
  const focusable = element
    ? Array.from(element.querySelectorAll<HTMLElement>(".item:not([disabled])"))
    : [];
  if (focusable.length === 0) return;
  event.preventDefault();
  const active = document.activeElement as HTMLElement | null;
  const index = active ? focusable.indexOf(active) : -1;
  let next = 0;
  if (event.key === "ArrowDown") next = index < 0 ? 0 : (index + 1) % focusable.length;
  else if (event.key === "ArrowUp") {
    next = index < 0 ? focusable.length - 1 : (index - 1 + focusable.length) % focusable.length;
  } else if (event.key === "End") next = focusable.length - 1;
  focusable[next].focus();
}

function onResize(): void {
  close();
}

onMounted(async () => {
  await loadItems();
  await nextTick();
  place();
  focusFirst();
  window.addEventListener("resize", onResize);
});

onBeforeUnmount(() => {
  window.removeEventListener("resize", onResize);
  const src = props.request.src;
  if (!props.request.placement.noReturnFocus && src) src.focus();
});
</script>

<template>
  <div class="menu-layer" :style="{ zIndex }" @keydown="onKeydown">
    <div class="menu-backdrop" @click="close" @contextmenu.prevent="close"></div>
    <div ref="box" class="menu" :style="style" role="menu" tabindex="-1">
      <p v-if="loading" class="menu-pending" aria-hidden="true">…</p>
      <template v-for="(item, index) in items" :key="index">
        <hr v-if="item === null" class="menu-divider" />
        <p v-else-if="item === undefined"></p>
        <p v-else-if="item.type === 'pending'" class="menu-pending" aria-hidden="true">…</p>
        <p v-else-if="item.type === 'label'" class="menu-label" :style="item.textStyle">
          {{ item.text }}
        </p>
        <RouterLink
          v-else-if="item.type === 'link'"
          :to="item.to"
          class="item"
          role="menuitem"
          :style="item.textStyle"
          @click="close"
        >
          <AppIcon v-if="iconName(item.icon)" :name="iconName(item.icon)!" />
          <span class="item-text">{{ item.text }}</span>
          <span v-if="item.indicate" class="indicate"></span>
        </RouterLink>
        <a
          v-else-if="item.type === 'a'"
          class="item"
          :href="item.href"
          :target="item.target"
          :download="item.download"
          role="menuitem"
          :style="item.textStyle"
          @click="close"
        >
          <AppIcon v-if="iconName(item.icon)" :name="iconName(item.icon)!" />
          <span class="item-text">{{ item.text }}</span>
          <span v-if="item.indicate" class="indicate"></span>
        </a>
        <button
          v-else-if="item.type === 'user'"
          type="button"
          class="item"
          role="menuitem"
          :class="{ active: item.active }"
          @click="activateUser(item, $event)"
        >
          <UserAvatar :user="item.user" :size="20" />
          <span class="item-text">{{ item.user.name || item.user.username }}</span>
          <span v-if="item.indicate" class="indicate"></span>
        </button>
        <label
          v-else-if="item.type === 'switch'"
          class="item switch"
          :class="{ disabled: item.disabled }"
          :style="item.textStyle"
        >
          <input
            type="checkbox"
            :checked="item.ref.value"
            :disabled="item.disabled"
            @change="toggleSwitch(item)"
          />
          <span class="item-text">{{ item.text }}</span>
        </label>
        <button
          v-else-if="item.type === 'parent'"
          type="button"
          class="item"
          :class="{ open: isExpanded(item) }"
          :style="item.textStyle"
          @click.stop="toggleParent(item)"
        >
          <AppIcon v-if="iconName(item.icon)" :name="iconName(item.icon)!" />
          <span class="item-text">{{ item.text }}</span>
          <AppIcon :name="isExpanded(item) ? 'caretUp' : 'caretRight'" class="item-caret" />
        </button>
        <button
          v-else
          type="button"
          class="item"
          role="menuitem"
          :class="{ danger: item.danger, accent: item.accent, active: item.active, hidden: item.hidden }"
          :style="item.textStyle"
          @click="activateButton(item, $event)"
        >
          <AppIcon v-if="iconName(item.icon)" :name="iconName(item.icon)!" />
          <span class="item-text">{{ item.text }}</span>
          <span v-if="item.indicate" class="indicate"></span>
        </button>

        <!-- One level of children, expanded inline; deeper nesting is not used
             by the first-release menus and is intentionally not rendered. -->
        <div v-if="item && item.type === 'parent' && isExpanded(item)" class="menu-children">
          <template v-for="(child, childIndex) in item.children" :key="childIndex">
            <hr v-if="child === null" class="menu-divider" />
            <p v-else-if="child === undefined"></p>
            <p v-else-if="child.type === 'label'" class="menu-label">{{ child.text }}</p>
            <RouterLink
              v-else-if="child.type === 'link'"
              :to="child.to"
              class="item"
              @click="close"
            >
              <span class="item-text">{{ child.text }}</span>
            </RouterLink>
            <a
              v-else-if="child.type === 'a'"
              class="item"
              :href="child.href"
              :target="child.target"
              :download="child.download"
              @click="close"
            >
              <span class="item-text">{{ child.text }}</span>
            </a>
            <label v-else-if="child.type === 'switch'" class="item switch">
              <input
                type="checkbox"
                :checked="child.ref.value"
                :disabled="child.disabled"
                @change="toggleSwitch(child)"
              />
              <span class="item-text">{{ child.text }}</span>
            </label>
            <button
              v-else-if="child.type === 'user'"
              type="button"
              class="item"
              @click="activateUser(child, $event)"
            >
              <UserAvatar :user="child.user" :size="20" />
              <span class="item-text">{{ child.user.name || child.user.username }}</span>
            </button>
            <button
              v-else-if="child.type === 'button' || child.type === undefined"
              type="button"
              class="item"
              :class="{ danger: child.danger, accent: child.accent }"
              @click="activateButton(child, $event)"
            >
              <span class="item-text">{{ child.text }}</span>
            </button>
          </template>
        </div>
      </template>
    </div>
  </div>
</template>

<style scoped>
.menu-layer {
  position: fixed;
  inset: 0;
}

.menu-backdrop {
  position: absolute;
  inset: 0;
}

.menu {
  position: absolute;
  min-width: 180px;
  max-width: min(320px, calc(100vw - 16px));
  overflow: auto;
  padding: 6px;
  background: var(--popup-bg);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-popover);
  color: var(--text);
}

.menu:focus {
  outline: none;
}

.menu-pending {
  margin: 0;
  padding: 8px 10px;
  color: var(--text-weak);
}

.item {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 7px 10px;
  font: inherit;
  font-size: 0.88rem;
  color: var(--text);
  text-align: start;
  text-decoration: none;
  background: transparent;
  border: 0;
  border-radius: var(--radius-sm);
  cursor: pointer;
}

.item:hover,
.item:focus-visible {
  background: var(--surface-hover);
}

.item:focus-visible {
  outline: var(--focus-outline);
  outline-offset: -2px;
}

.item.active {
  color: var(--accent);
}

.item.accent {
  color: var(--accent);
}

.item.danger {
  color: var(--danger);
}

.item.hidden {
  display: none;
}

.item.switch {
  cursor: default;
}

.item.switch input {
  flex: none;
}

.item-text {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.item-caret {
  flex: none;
  color: var(--text-weak);
}

.indicate {
  flex: none;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--accent);
}

.menu-divider {
  margin: 4px 6px;
  border: 0;
  border-top: 1px solid var(--divider);
}

.menu-label {
  margin: 4px 0 2px;
  padding: 4px 10px;
  font-size: 0.72rem;
  font-weight: 700;
  color: var(--text-weak);
}

.menu-children {
  margin: 2px 0 4px;
  padding-inline-start: 12px;
  border-inline-start: 1px solid var(--divider);
}
</style>
