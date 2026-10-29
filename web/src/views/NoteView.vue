<script setup lang="ts">
import { ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { api } from "../api/client";
import type { Note } from "../api/types";
import AppIcon from "../components/AppIcon.vue";
import NoteCard from "../components/NoteCard.vue";
import NoteComposer from "../components/NoteComposer.vue";
import { isLoggedIn } from "../stores/session";

const route = useRoute();
const router = useRouter();

const note = ref<Note | null>(null);
const loading = ref(false);
const error = ref("");
const replyTo = ref<Note | null>(null);

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
    <NoteCard :note="note" @reply="replyTo = $event" @deleted="onDeleted" @updated="note = $event" />
    <NoteComposer
      v-if="isLoggedIn"
      class="reply-composer"
      :reply-to="replyTo"
      @posted="replyTo = null"
      @cancel="replyTo = null"
    />
  </template>
</template>

<style scoped>
.reply-composer {
  margin-top: 16px;
}

.skeleton {
  height: 180px;
  border-radius: var(--radius-lg);
  background: var(--panel);
  border: 1px solid var(--border);
}
</style>
