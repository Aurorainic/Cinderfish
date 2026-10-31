<script setup lang="ts">
import AppIcon from "../AppIcon.vue";

type IconName = InstanceType<typeof AppIcon>["$props"]["name"];

// A labelled block of the profile page. The pinned strip, the note list and the
// follow-list placeholders share this header shape, so every block says what it
// is before its content — the same reading order the fork gets from its section
// markup plus the tab strip. `plain` drops the card chrome inside the deck
// window, where the section already sits on the page window's panel.
withDefaults(
  defineProps<{
    title: string;
    icon?: IconName;
    hint?: string;
    count?: number;
    plain?: boolean;
  }>(),
  { icon: undefined, hint: "", count: undefined, plain: false },
);
</script>

<template>
  <section class="profile-section" :class="{ plain }">
    <header class="section-head">
      <span class="section-label">
        <AppIcon v-if="icon" class="section-icon" :name="icon" />
        <h2 class="section-title">{{ title }}</h2>
        <span v-if="count !== undefined" class="section-count">{{ count }}</span>
      </span>
      <span v-if="hint" class="section-hint">{{ hint }}</span>
      <div v-if="$slots.actions" class="section-actions">
        <slot name="actions" />
      </div>
    </header>
    <div class="section-body">
      <slot />
    </div>
  </section>
</template>

<style scoped>
.profile-section {
  min-width: 0;
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  background: var(--panel);
}

.profile-section.plain {
  border: 0;
  border-radius: 0;
  background: transparent;
}

.section-head {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: var(--space-6) var(--space-10);
  padding: var(--space-10) var(--space-14);
  border-bottom: 1px solid var(--divider);
}

.plain .section-head {
  padding-inline: 0;
}

.section-label {
  display: inline-flex;
  align-items: center;
  gap: var(--space-6);
  min-width: 0;
}

.section-icon {
  flex: none;
  width: 15px;
  height: 15px;
  color: var(--accent);
}

.section-title {
  font-size: var(--fs-sm);
  font-weight: var(--weight-bold);
  color: var(--text-strong);
  white-space: nowrap;
}

.section-count {
  padding: 0 var(--space-6);
  border-radius: var(--radius-full);
  background: var(--surface-inset);
  color: var(--text-weak);
  font-size: var(--fs-2xs);
  font-weight: var(--weight-semibold);
  font-variant-numeric: tabular-nums;
  line-height: 1.7;
}

.section-hint {
  min-width: 0;
  color: var(--text-faint);
  font-size: var(--fs-2xs);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.section-actions {
  display: flex;
  align-items: center;
  gap: var(--space-4);
  margin-inline-start: auto;
}

.section-body {
  min-width: 0;
}

/* Rows carry their own hairline, so a note list needs no extra padding. */
.section-body :deep(.paged-feed) {
  gap: 0;
}

@media (max-width: 500px) {
  .section-head {
    padding-inline: var(--space-12);
  }

  .section-hint {
    display: none;
  }
}
</style>
