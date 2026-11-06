<!-- Ported from firefish@b425f61:packages/client/src/pages/admin/files.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :display-back-button="true" />
		</template>
		<MkSpacer :content-max="1000" :margin-min="16" :margin-max="32">
			<div v-if="!admin" class="cf-admin-files">
				<CfInfo warn>
					需要管理员权限，无法查看实例文件。
					<CfA to="/" class="_link">返回首页</CfA>
				</CfInfo>
			</div>
			<div v-else class="cf-admin-files">
				<form class="filters" @submit.prevent="apply">
					<CfSelect v-model="origin" class="field" @update:model-value="apply">
						<template #label>来源</template>
						<option value="local">本地</option>
						<option value="remote">远程</option>
						<option value="combined">全部</option>
					</CfSelect>
					<CfSelect v-model="typeFilter" class="field" @update:model-value="apply">
						<template #label>类型</template>
						<option value="">全部</option>
						<option value="image">图片</option>
						<option value="video">视频</option>
						<option value="audio">音频</option>
						<option value="text">文本</option>
						<option value="application">其它</option>
					</CfSelect>
					<CfInput v-model="username" class="field">
						<template #label>上传者</template>
						<template #prefix>@</template>
					</CfInput>
					<CfInput v-model="hostname" class="field">
						<template #label>实例主机</template>
					</CfInput>
					<div class="filterActions">
						<CfButton type="submit" primary>应用</CfButton>
						<CfButton type="button" @click="reset">重置</CfButton>
					</div>
				</form>

				<CfInfo v-if="applyError !== null" warn class="applyError">
					{{ applyError }}
				</CfInfo>

				<CfLoading v-if="fetching && items.length === 0" />
				<MkError v-else-if="error != null && items.length === 0" @retry="reload" />
				<CfInfo v-else-if="items.length === 0">
					没有符合筛选条件的文件。
				</CfInfo>
				<template v-else>
					<ul class="files">
						<li v-for="file in items" :key="file.id" class="file">
							<a
								class="thumb"
								:href="resolveMediaUrl(file.url)"
								target="_blank"
								rel="noopener noreferrer"
								:style="thumbStyle(file)"
							>
								<i v-if="!isImage(file)" class="ph-file ph-bold"></i>
							</a>
							<div class="body">
								<a
									class="name"
									:href="resolveMediaUrl(file.url)"
									target="_blank"
									rel="noopener noreferrer"
									:title="file.name"
									>{{ file.name }}</a
								>
								<div class="meta">
									<span class="_monospace">{{ file.type }}</span>
									<span>{{ formatSize(file.size) }}</span>
									<span v-if="file.user != null" class="uploader">
										<MkA :to="userPage(file.user)"
											>@{{ acct(file.user) }}</MkA
										>
									</span>
									<span class="date"
										><MkTime :time="file.createdAt" mode="absolute"
									/></span>
									<MkA class="detail" :to="`/admin/file/${file.id}`"
										>详情</MkA
									>
								</div>
							</div>
						</li>
					</ul>

					<div v-if="hasMore" class="more">
						<CfButton v-if="!moreFetching" @click="loadMore">
							加载更多
						</CfButton>
						<CfLoading v-else mini />
					</div>
				</template>

				<CfInfo class="notice">
					这里只做盘点：文件删除与批量清理还没有端点，避免管理员误删无法恢复。
				</CfInfo>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script setup lang="ts">
// Firefish's admin file browser pages `admin/drive/files` with origin / type /
// username / hostname filters. Cinderfish now serves that endpoint with the
// same filters; the delete action is deliberately absent because upstream
// deletes through `admin/drive/cleanup` and per-file removal, neither of which
// is implemented here.
//
// Username filtering is resolved through `users/show` first, because the
// endpoint takes `userId`; an unknown username is reported inline instead of
// silently returning the whole instance.
import { computed, onMounted, ref } from "vue";
import { ApiError, api, resolveMediaUrl } from "@/api/client";
import type { AdminDriveFile, UserDetailed } from "@/api/types";
import { isAdmin } from "@/account";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";
import MkError from "@/components/global/MkError.vue";
import MkTime from "@/components/global/MkTime.vue";
import CfA from "@/components/ui/CfA.vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfInput from "@/components/ui/CfInput.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import CfSelect from "@/components/ui/CfSelect.vue";
import { usePagination } from "@/composables/use-pagination";
import { acct, userPage } from "@/filters/user";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageAdminFiles" });

const admin = computed<boolean>(() => isAdmin());

const origin = ref("local");
const typeFilter = ref("");
const username = ref<string | number | null>("");
const hostname = ref<string | number | null>("");
const userId = ref<string | null>(null);
const applyError = ref<string | null>(null);

const { items, fetching, moreFetching, hasMore, error, reload, loadMore } =
	usePagination<AdminDriveFile>({
		endpoint: "admin/drive/files",
		limit: 30,
		immediate: false,
		params: () => ({
			origin: origin.value,
			...(typeFilter.value !== "" ? { type: typeFilter.value } : {}),
			...(userId.value != null ? { userId: userId.value } : {}),
			...(() => {
				const host = String(hostname.value ?? "").trim();
				return host !== "" ? { hostname: host } : {};
			})(),
		}),
	});

async function apply(): Promise<void> {
	if (!admin.value) return;
	applyError.value = null;
	const name = String(username.value ?? "").trim().replace(/^@/, "");
	if (name !== "") {
		try {
			const user = await api("users/show", { username: name });
			userId.value = (user as UserDetailed).id;
		} catch (cause) {
			userId.value = null;
			applyError.value =
				cause instanceof ApiError && cause.code === "NO_SUCH_USER"
					? "找不到这个本地账号，请检查用户名。"
					: "无法解析上传者，请稍后再试。";
			return;
		}
	} else {
		userId.value = null;
	}
	await reload();
}

function reset(): void {
	origin.value = "local";
	typeFilter.value = "";
	username.value = "";
	hostname.value = "";
	userId.value = null;
	void apply();
}

function isImage(file: AdminDriveFile): boolean {
	return file.type.startsWith("image/");
}

function thumbStyle(file: AdminDriveFile): Record<string, string> {
	return isImage(file)
		? { backgroundImage: `url(${resolveMediaUrl(file.url)})` }
		: {};
}

function formatSize(bytes: number): string {
	if (bytes < 1024) return `${bytes} B`;
	if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
	if (bytes < 1024 * 1024 * 1024)
		return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
	return `${(bytes / 1024 / 1024 / 1024).toFixed(2)} GB`;
}

onMounted(() => {
	if (admin.value) void apply();
});

definePageMetadata({
	title: "文件",
	icon: "ph-folder-open ph-bold ph-lg",
});
</script>

<style lang="scss" scoped>
.filters {
	display: grid;
	grid-template-columns: repeat(auto-fit, minmax(170px, 1fr));
	gap: 10px;
	align-items: end;
	margin-bottom: 16px;
	padding: 14px 16px;
	background: var(--panel);
	border-radius: var(--radius);

	.field {
		margin: 0;
	}

	.filterActions {
		display: flex;
		gap: 8px;
	}
}

.applyError {
	margin-bottom: 12px;
}

.files {
	list-style: none;
	margin: 0;
	padding: 0;
	display: flex;
	flex-direction: column;
	gap: 8px;
}

.file {
	display: flex;
	align-items: center;
	gap: 12px;
	padding: 10px 12px;
	background: var(--panel);
	border-radius: var(--radius);

	.thumb {
		width: 56px;
		height: 56px;
		flex: none;
		border-radius: 6px;
		background: var(--bg) center / cover no-repeat;
		display: grid;
		place-items: center;
		opacity: 0.9;
	}

	.body {
		flex: 1;
		min-width: 0;
	}

	.name {
		display: block;
		font-weight: 600;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.meta {
		display: flex;
		flex-wrap: wrap;
		gap: 4px 12px;
		margin-top: 2px;
		font-size: 0.8em;
		opacity: 0.7;
	}
}

.more {
	display: flex;
	justify-content: center;
	padding: 16px 0;
}

.notice {
	margin-top: 16px;
}

@media (max-width: 520px) {
	.filterActions {
		grid-column: 1 / -1;
	}
}
</style>
