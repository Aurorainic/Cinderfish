<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { api } from "../../api/client";
import type { DriveFile } from "../../api/types";
import AppIcon from "../../components/AppIcon.vue";
import MkButton from "../../components/ui/MkButton.vue";
import MkInfo from "../../components/ui/MkInfo.vue";
import MkLoading from "../../components/ui/MkLoading.vue";
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
  <section class="_card">
    <div class="_title section-head">
      <h2>文件</h2>
      <span v-if="files.length" class="count">{{ files.length }}</span>
    </div>
    <div class="_content">
      <p class="_caption section-desc">
        媒体存储后端：<strong>{{ storage }}</strong>。这里显示你最近上传的 30 个文件。
      </p>

      <AdminCallout class="callout-row" title="实例级文件管理尚未提供" tone="warn">
        <p>
          后端目前没有列出或清理全实例文件的接口，本页只能列出你自己上传的文件，无法查看其他账号的文件或实例总占用。
        </p>
      </AdminCallout>

      <MkInfo v-if="error" class="notice-row" warn role="alert">
        <span>{{ error }}</span>
        <MkButton @click="loadFiles">重试</MkButton>
      </MkInfo>

      <MkLoading v-if="loading" />
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
          <a
            class="_textButton open"
            :href="file.url"
            target="_blank"
            rel="noopener noreferrer"
          >
            打开
          </a>
        </li>
      </ul>
      <p v-else-if="!error" class="_caption">还没有上传过文件。</p>
    </div>
  </section>
</template>

<style scoped>
.section-head {
  display: flex;
  align-items: center;
  gap: 10px;
}

.section-head h2 {
  font-size: 1em;
}

.section-desc {
  margin-block-end: var(--margin);
}

.count {
  display: inline-grid;
  place-items: center;
  min-inline-size: 22px;
  block-size: 22px;
  padding-inline: 6px;
  border-radius: 999px;
  background: var(--accentedBg);
  color: var(--accent);
  font-size: 0.78em;
  font-weight: bold;
}

.callout-row,
.notice-row {
  margin-block-end: var(--margin);
}

.file-list {
  list-style: none;
  margin: 0;
  padding: 0;
}

.file-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding-block: 8px;
  border-block-start: solid 0.5px var(--hairline);
}

.file-row:first-child {
  border-block-start: none;
}

.thumb {
  display: grid;
  place-items: center;
  flex: none;
  inline-size: 40px;
  block-size: 40px;
  overflow: hidden;
  border-radius: 4px;
  background: var(--panelHighlight);
  color: var(--fgTransparentWeak);
}

.thumb img {
  inline-size: 100%;
  block-size: 100%;
  object-fit: cover;
}

.thumb .icon {
  inline-size: 18px;
  block-size: 18px;
}

.file-main {
  flex: 1;
  min-inline-size: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.file-name {
  font-size: 0.9em;
  font-weight: 600;
  color: var(--fg);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-meta {
  font-size: 0.85em;
  color: var(--fgTransparentWeak);
  overflow-wrap: anywhere;
}

.open {
  flex: none;
  font-size: 0.9em;
}

@media (max-width: 520px) {
  .file-row {
    flex-wrap: wrap;
  }
}
</style>
