<!-- Ported from firefish@b425f61:packages/client/src/pages/admin/object-storage.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :display-back-button="true" />
		</template>
		<MkSpacer :content-max="700" :margin-min="16" :margin-max="32">
			<div v-if="!admin" class="cf-admin-object-storage">
				<CfInfo warn>
					需要管理员权限，无法查看对象存储配置。
					<CfA to="/" class="_link">返回首页</CfA>
				</CfInfo>
			</div>
			<div v-else class="cf-admin-object-storage">
				<CfLoading v-if="loading" />
				<MkError v-else-if="error !== null" @retry="load" />
				<template v-else-if="info != null">
					<section class="card">
						<div class="head">
							<h2 class="title">媒体存储</h2>
							<span class="badge">{{ info.backend }}</span>
						</div>
						<dl v-if="info.backend !== 's3'" class="facts">
							<div>
								<dt>本地路径</dt>
								<dd class="_monospace">{{ info.localPath || "（未配置）" }}</dd>
							</div>
						</dl>
						<dl v-else-if="info.s3 != null" class="facts">
							<div>
								<dt>Endpoint</dt>
								<dd class="_monospace">{{ info.s3.endpointHost || "（未配置）" }}</dd>
							</div>
							<div>
								<dt>Bucket</dt>
								<dd class="_monospace">{{ info.s3.bucket || "（未配置）" }}</dd>
							</div>
							<div>
								<dt>Region</dt>
								<dd class="_monospace">{{ info.s3.region || "（默认）" }}</dd>
							</div>
							<div>
								<dt>HTTPS</dt>
								<dd>{{ info.s3.useSSL ? "是" : "否" }}</dd>
							</div>
							<div>
								<dt>公开基础 URL</dt>
								<dd class="_monospace">
									{{ info.s3.publicBaseUrl || "（使用预签名 URL）" }}
								</dd>
							</div>
							<div>
								<dt>凭据</dt>
								<dd>{{ info.s3.hasCredentials ? "已配置" : "未配置" }}</dd>
							</div>
						</dl>
					</section>

					<CfInfo class="notice">
						这里只读：存储后端、路径与桶来自 <code>config.toml</code> 的 <code>[media]</code>，
						修改后需要重启。访问密钥与私钥不会被接口返回。
					</CfInfo>
				</template>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script setup lang="ts">
// Firefish's object-storage page edits S3 settings through admin endpoints this
// port does not have. The page is therefore read-only: it shows the effective
// configuration (secrets excluded) and points at config.toml.
import { computed, onMounted, ref } from "vue";
import { ApiError, api } from "@/api/client";
import type { ObjectStorageInfo } from "@/api/types";
import { isAdmin } from "@/account";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";
import MkError from "@/components/global/MkError.vue";
import CfA from "@/components/ui/CfA.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageAdminObjectStorage" });

const admin = computed<boolean>(() => isAdmin());
const info = ref<ObjectStorageInfo | null>(null);
const loading = ref(true);
const error = ref<string | null>(null);

async function load(): Promise<void> {
	if (!admin.value) return;
	loading.value = true;
	error.value = null;
	try {
		info.value = await api("admin/object-storage", {});
	} catch (cause) {
		error.value = cause instanceof ApiError ? cause.message : String(cause);
	} finally {
		loading.value = false;
	}
}

onMounted(() => {
	void load();
});

definePageMetadata({
	title: "对象存储",
	icon: "ph-cloud-arrow-up ph-bold ph-lg",
});
</script>

<style lang="scss" scoped>
.card {
	padding: 16px;
	background: var(--panel);
	border-radius: var(--radius);

	.head {
		display: flex;
		align-items: center;
		gap: 10px;
		margin-bottom: 12px;
	}

	.title {
		margin: 0;
		font-size: 1em;
	}

	.badge {
		padding: 1px 8px;
		border-radius: 999px;
		background: var(--bg);
		font-size: 0.75em;
	}
}

.facts {
	display: flex;
	flex-direction: column;
	gap: 8px;
	margin: 0;

	> div {
		display: flex;
		gap: 12px;
		flex-wrap: wrap;
	}

	dt {
		min-width: 110px;
		font-size: 0.85em;
		opacity: 0.65;
	}

	dd {
		margin: 0;
		overflow-wrap: anywhere;
	}
}

.notice {
	margin-top: 16px;
}
</style>
