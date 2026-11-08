<!-- Ported from firefish@b425f61:packages/client/src/pages/admin/file.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :display-back-button="true" />
		</template>
		<MkSpacer :content-max="900" :margin-min="16" :margin-max="32">
			<div v-if="!admin" class="cf-admin-file">
				<CfInfo warn>
					需要管理员权限，无法查看文件详情。
					<CfA to="/" class="_link">返回首页</CfA>
				</CfInfo>
			</div>

			<div v-else class="cf-admin-file">
				<CfLoading v-if="loading" />
				<CfInfo v-else-if="loadError != null" warn>
					{{ loadError }}
					<CfButton inline @click="load">{{ t("retry") }}</CfButton>
				</CfInfo>

				<template v-else-if="file != null">
					<div class="preview _panel">
						<img
							v-if="kind === 'image'"
							:src="resolveMediaUrl(file.url)"
							:alt="file.name"
							:class="{ sensitive: file.isSensitive }"
						/>
						<video
							v-else-if="kind === 'video'"
							:src="resolveMediaUrl(file.url)"
							controls
							preload="metadata"
						/>
						<audio
							v-else-if="kind === 'audio'"
							:src="resolveMediaUrl(file.url)"
							controls
							preload="metadata"
						/>
						<div v-else class="noPreview">
							<i class="ph-file ph-bold ph-4x"></i>
							<span>{{ file.type || "未知类型" }} 没有内联预览</span>
						</div>
					</div>

					<section class="_formBlock">
						<h2 class="_sectionTitle">文件</h2>
						<CfKeyValue :copy="file.name">
							<template #key>{{ t("name") }}</template>
							<template #value>{{ file.name }}</template>
						</CfKeyValue>
						<CfKeyValue :copy="file.id" oneline>
							<template #key>ID</template>
							<template #value
								><code class="_monospace">{{ file.id }}</code></template
							>
						</CfKeyValue>
						<CfKeyValue oneline>
							<template #key>{{ t("type") }}</template>
							<template #value
								><code class="_monospace">{{ file.type || "—" }}</code></template
							>
						</CfKeyValue>
						<CfKeyValue oneline>
							<template #key>{{ t("size") }}</template>
							<template #value>{{ formatSize(file.size) }}</template>
						</CfKeyValue>
						<CfKeyValue oneline>
							<template #key>上传时间</template>
							<template #value
								><MkTime :time="file.createdAt" mode="detail"
							/></template>
						</CfKeyValue>
						<CfKeyValue oneline>
							<template #key>{{ t("sensitive") }}</template>
							<template #value>{{
								file.isSensitive ? t("yes") : t("no")
							}}</template>
						</CfKeyValue>
						<CfKeyValue v-if="file.comment != null" oneline>
							<template #key>{{ t("caption") }}</template>
							<template #value>{{ file.comment }}</template>
						</CfKeyValue>
						<CfKeyValue oneline>
							<template #key>链接</template>
							<template #value>
								<a
									:href="resolveMediaUrl(file.url)"
									target="_blank"
									rel="noopener noreferrer"
									>{{ file.url }}</a
								>
							</template>
						</CfKeyValue>
					</section>

					<section class="_formBlock">
						<h2 class="_sectionTitle">上传者</h2>
						<template v-if="file.user != null">
							<CfKeyValue oneline>
								<template #key>{{ t("username") }}</template>
								<template #value>
									<MkA :to="userPage(file.user)"
										>@{{ acct(file.user) }}</MkA
									>
								</template>
							</CfKeyValue>
							<CfKeyValue oneline>
								<template #key>ID</template>
								<template #value
									><code class="_monospace">{{ file.user.id }}</code></template
								>
							</CfKeyValue>
						</template>
						<CfInfo v-else class="note">
							上传者账号已不存在。账号删除会连同云盘文件一起删除，这里的数据属于异常遗留。
						</CfInfo>
					</section>

					<CfInfo class="note">
						本页只读：`drive/files/delete` 只允许删除自己的文件，管理员没有删除他人文件的端点。
						需要清理时请让上传者本人删除，或走数据库运维流程。
					</CfInfo>
				</template>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script setup lang="ts">
// Admin file detail. Firefish's admin/file.vue also shows the upload IP and
// the original request; Cinderfish stores neither, so the page shows what
// admin/drive/show-file returns and says why there is no delete button.
import { computed, onMounted, ref } from "vue";
import { useRoute } from "vue-router";
import { isAdmin } from "@/account";
import { ApiError, api, resolveMediaUrl } from "@/api/client";
import type { AdminDriveFile } from "@/api/types";
import CfA from "@/components/ui/CfA.vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfKeyValue from "@/components/ui/CfKeyValue.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import MkA from "@/components/global/MkA.vue";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";
import MkTime from "@/components/global/MkTime.vue";
import { acct, userPage } from "@/filters/user";
import { t } from "@/i18n";
import { definePageMetadata } from "@/scripts/page-metadata";

function formatSize(bytes: number): string {
	if (bytes < 1024) return `${bytes} B`;
	if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
	if (bytes < 1024 * 1024 * 1024)
		return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
	return `${(bytes / 1024 / 1024 / 1024).toFixed(2)} GB`;
}

defineOptions({ name: "CfPageAdminFile" });

const route = useRoute();
const admin = computed<boolean>(() => isAdmin());
const loading = ref(false);
const loadError = ref<string | null>(null);
const file = ref<AdminDriveFile | null>(null);

const fileId = computed(() => String(route.params.fileId ?? ""));

const kind = computed<"image" | "video" | "audio" | "other">(() => {
	const type = file.value?.type ?? "";
	if (type.startsWith("image/")) return "image";
	if (type.startsWith("video/")) return "video";
	if (type.startsWith("audio/")) return "audio";
	return "other";
});

function errorMessage(cause: unknown): string {
	if (cause instanceof ApiError) return cause.message;
	return String(cause);
}

async function load(): Promise<void> {
	if (!admin.value) return;
	if (fileId.value === "") {
		loadError.value = "缺少文件 id。";
		return;
	}
	loading.value = true;
	loadError.value = null;
	try {
		file.value = await api("admin/drive/show-file", { fileId: fileId.value });
	} catch (cause) {
		loadError.value = errorMessage(cause);
		file.value = null;
	} finally {
		loading.value = false;
	}
}

onMounted(() => {
	void load();
});

definePageMetadata({ title: t("file"), icon: "ph-file ph-bold ph-lg" });
</script>

<style lang="scss" scoped>
.cf-admin-file {
	.preview {
		display: flex;
		align-items: center;
		justify-content: center;
		min-height: 200px;
		margin-bottom: 20px;
		padding: 16px;
		border-radius: var(--radius);
		background: var(--panel);
		overflow: hidden;

		img,
		video {
			max-width: 100%;
			max-height: 420px;
			border-radius: 6px;
		}

		img.sensitive {
			filter: blur(24px);
		}

		audio {
			width: 100%;
		}
	}

	.noPreview {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 8px;
		opacity: 0.6;
		font-size: 0.9em;
	}

	.note {
		margin-top: 16px;
	}
}
</style>
