<script setup lang="ts">
import { ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { api } from "../api/client";
import type { Note } from "../api/types";
import AppIcon from "../components/AppIcon.vue";
import NoteItem from "../components/note/Note.vue";
import { openPostForm } from "../os";

const route = useRoute();
const router = useRouter();

const note = ref<Note | null>(null);
const loading = ref(false);
const error = ref("");

async function load() {
  const id = String(route.params.id ?? "");
  if (!id) return;
  loading.value = true;
  error.value = "";
  try {
    note.value = await api.showNote(id);
  } catch (err) {
    note.value = null;
    error.value = err instanceof Error ? err.message : "加载失败";
  } finally {
    loading.value = false;
  }
}

function onDeleted(id: string) {
  if (note.value?.id === id) router.push("/");
}

watch(() => route.params.id, load, { immediate: true });
</script>

<template>
  <header class="page-title">
    <button class="icon-button" type="button" title="返回" @click="router.back()">
      <AppIcon name="arrowLeft" />
    </button>
    <h1>笔记</h1>
  </header>

  <div v-if="loading" class="skeleton"></div>
  <p v-else-if="error" class="notice error">{{ error }}</p>

  <template v-else-if="note">
    <div class="note-detail">
      <NoteItem
        :note="note"
        detail
        @reply="openPostForm({ reply: $event })"
        @deleted="onDeleted"
      />
    </div>
  </template>
</template>

<style scoped>
.note-detail {
  padding: 4px 0 8px;
  border-bottom: 1px solid var(--divider);
}

.skeleton {
  height: 160px;
  border-radius: 0;
  background: var(--panel-2);
}
</style>
