<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { api } from "../../api/client";
import type { DriveFile } from "../../api/types";
import AppIcon from "../../components/AppIcon.vue";
import { instance } from "../../stores/instance";
import AdminCallout from "./AdminCallout.vue";
import { storageLabel } from "./meta";

const files = ref<DriveFile[]>([]);
const loading = ref(true);
const error = ref("");

const storage = computed(() => storageLabel(instance.meta));

function messageOf(err: unknown): string {
  return err instanceof Error ? err.message : String(err);
}

function isImage(file: DriveFile): boolean {
  return file.type.startsWith("image/");
}

function formatSize(bytes: number): string {
  if (!Number.isFinite(bytes) || bytes < 0) return "—";
  if (bytes < 1024) return `${bytes} B`;
  const units = ["KB", "MB", "GB", "TB"];
  let value = bytes / 1024;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024;
    unit += 1;
  }
  return `${value.toFixed(value >= 10 ? 0 : 1)} ${units[unit]}`;
}

function formatDate(iso: string): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return iso;
  return date.toLocaleDateString("zh-CN", { year: "numeric", month: "2-digit", day: "2-digit" });
}

async function loadFiles() {
  loading.value = true;
  error.value = "";
  try {
    files.value = await api.driveFiles({ limit: 30 });
  } catch (err) {
    error.value = `文件列表加载失败：${messageOf(err)}`;
    files.value = [];
  } finally {
    loading.value = false;
  }
}

onMounted(loadFiles);
</script>

<template>
  <section class="card">
    <div class="section-head">
      <h2>文件</h2>
      <span v-if="files.length" class="count">{{ files.length }}</span>
    </div>
    <p class="muted small section-desc">
      媒体存储后端：<strong>{{ storage }}</strong>。这里显示你最近上传的 30 个文件。
    </p>

    <AdminCallout
      class="files-alert"
      title="实例级文件管理尚未提供"
      tone="warn"
    >
      <p>
        后端目前没有列出或清理全实例文件的接口，本页只能列出你自己上传的文件，无法查看其他账号的文件或实例总占用。
      </p>
    </AdminCallout>

    <div v-if="error" class="notice error" role="alert">
      <span>{{ error }}</span>
      <button class="button ghost small" type="button" @click="loadFiles">重试</button>
    </div>

    <p v-if="loading" class="hint">正在加载文件…</p>
    <ul v-else-if="files.length" class="file-list">
      <li v-for="file in files" :key="file.id" class="file-row">
        <span class="thumb">
          <img
            v-if="isImage(file) && (file.thumbnailUrl || file.url)"
            :src="file.thumbnailUrl || file.url"
            alt=""
            loading="lazy"
          />
          <AppIcon v-else name="image" />
        </span>
        <div class="file-main">
          <span class="file-name">{{ file.name }}</span>
          <span class="file-meta">
            {{ file.type || "未知类型" }} · {{ formatSize(file.size) }} ·
            {{ formatDate(file.createdAt) }}
          </span>
        </div>
        <a class="button ghost small" :href="file.url" target="_blank" rel="noopener noreferrer">
          打开
        </a>
      </li>
    </ul>
    <p v-else-if="!error" class="hint">还没有上传过文件。</p>
  </section>
</template>

<style scoped>
.section-head {
  display: flex;
  align-items: center;
  gap: var(--space-10);
  margin-bottom: var(--space-6);
}

.section-head h2 {
  font-size: var(--fs-md);
}

.section-desc {
  margin-bottom: var(--space-14);
}

.count {
  display: inline-grid;
  place-items: center;
  min-width: 22px;
  height: 22px;
  padding: 0 var(--space-6);
  border-radius: var(--radius-full);
  background: var(--accent-soft);
  color: var(--accent);
  font-size: var(--fs-xs);
  font-weight: var(--weight-bold);
}

.files-alert {
  margin-bottom: var(--space-14);
}

.file-list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
}

.file-row {
  display: flex;
  align-items: center;
  gap: var(--space-12);
  padding: var(--space-8) var(--space-10);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--surface-inset);
}

.thumb {
  display: grid;
  place-items: center;
  flex: none;
  width: 40px;
  height: 40px;
  overflow: hidden;
  border-radius: var(--radius-sm);
  background: var(--panel-2);
  color: var(--text-weak);
}

.thumb img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.thumb .icon {
  width: 18px;
  height: 18px;
}

.file-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.file-name {
  font-size: var(--fs-sm);
  font-weight: var(--weight-semibold);
  color: var(--text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-meta {
  font-size: var(--fs-xs);
  color: var(--text-weak);
  overflow-wrap: anywhere;
}

@media (max-width: 520px) {
  .file-row {
    flex-wrap: wrap;
  }
}
</style>
