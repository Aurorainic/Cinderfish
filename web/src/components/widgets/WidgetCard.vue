<script setup lang="ts">
import { useId } from "vue";
import AppIcon from "../AppIcon.vue";
import type { WidgetIcon } from "../../widgets/registry";

// The card every widget sits in: one header row (icon, title, optional hint,
// header actions) above a body the widget fills. `flush` removes the body
// padding for widgets that paint their own edge-to-edge rows. The title is the
// card's accessible name, so a stack of cards reads as a list of labelled
// regions instead of anonymous sections.
withDefaults(
  defineProps<{ title: string; icon: WidgetIcon; hint?: string; flush?: boolean }>(),
  { hint: "", flush: false },
);

const titleId = useId();
</script>

<template>
  <section class="widget" :aria-labelledby="titleId">
    <header class="widget-head">
      <span class="widget-icon"><AppIcon :name="icon" /></span>
      <div class="widget-heading">
        <h2 :id="titleId" class="widget-title">{{ title }}</h2>
        <p v-if="hint" class="widget-hint">{{ hint }}</p>
      </div>
      <div class="widget-actions"><slot name="actions" /></div>
    </header>
    <div class="widget-body" :class="{ flush }">
      <slot />
    </div>
  </section>
</template>

<style scoped>
.widget {
  background: var(--panel);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  overflow: hidden;
}

.widget-head {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 14px;
  border-bottom: 1px solid var(--divider);
}

.widget-icon {
  display: grid;
  place-items: center;
  flex: none;
  color: var(--accent);
}

.widget-icon .icon {
  width: 16px;
  height: 16px;
}

.widget-heading {
  min-width: 0;
}

.widget-title {
  font-size: 0.92rem;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.widget-hint {
  margin: 0;
  color: var(--muted);
  font-size: 0.74rem;
  line-height: 1.3;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.widget-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-left: auto;
}

.widget-body {
  padding: 14px;
}

.widget-body.flush {
  padding: 0;
}
</style>
