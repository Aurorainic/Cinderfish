<!-- Ported from firefish@b425f61:packages/client/src/pages/admin/object-storage.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :display-back-button="true" />
		</template>
		<MkSpacer :content-max="700" :margin-min="16" :margin-max="32">
			<div v-if="!admin" class="cf-admin-object-storage">
				<CfInfo warn>
					需要管理员权限，无法修改对象存储配置。
					<CfA to="/" class="_link">返回首页</CfA>
				</CfInfo>
			</div>
			<div v-else class="cf-admin-object-storage">
				<CfLoading v-if="loading" />
				<MkError v-else-if="loadError !== null" @retry="load" />
				<form v-else class="_formRoot" @submit.prevent="save">
					<CfSelect v-model="backend" class="_formBlock" :disabled="saving">
						<template #label>存储后端</template>
						<option value="local">本地文件系统</option>
						<option value="s3">S3 兼容对象存储</option>
						<template #caption>
							切换立即生效；已有文件不会自动迁移，需要时用
							<code>cinderfish-media-migrate</code> 搬运。
						</template>
					</CfSelect>

					<template v-if="backend === 'local'">
						<CfInput v-model="localPath" class="_formBlock" :disabled="saving">
							<template #label>本地路径</template>
							<template #caption>相对于进程工作目录或绝对路径；目录不存在时会自动创建。</template>
						</CfInput>
					</template>

					<template v-else>
						<CfInput v-model="endpoint" class="_formBlock" :disabled="saving">
							<template #label>Endpoint</template>
							<template #caption
								>S3 服务的主机名（如 <code>s3.example.com</code> 或
								<code>127.0.0.1:9000</code>）。</template
							>
						</CfInput>
						<CfInput v-model="bucket" class="_formBlock" :disabled="saving">
							<template #label>Bucket</template>
						</CfInput>
						<CfInput v-model="region" class="_formBlock" :disabled="saving">
							<template #label>Region（可选）</template>
						</CfInput>
						<CfInput v-model="accessKey" class="_formBlock" :disabled="saving">
							<template #label>Access Key</template>
							<template #caption>
								{{ info != null && info.s3 != null && info.s3.accessKeySet
									? "已配置；留空保留现有值。"
									: "尚未配置。" }}
							</template>
						</CfInput>
						<CfInput
							v-model="secretKey"
							class="_formBlock"
							type="password"
							:disabled="saving"
						>
							<template #label>Secret Key</template>
							<template #caption>
								{{ info != null && info.s3 != null && info.s3.secretKeySet
									? "已配置；留空保留现有值。密钥不会回传。"
									: "尚未配置。" }}
							</template>
						</CfInput>
						<label class="checkRow">
							<input v-model="useSSL" type="checkbox" :disabled="saving" />
							<span>使用 HTTPS 连接 Endpoint</span>
						</label>
						<CfInput v-model="publicBaseUrl" class="_formBlock" :disabled="saving">
							<template #label>公开基础 URL（可选）</template>
							<template #caption
								>桶公开时可填，用于生成直链；留空则使用预签名 URL。</template
							>
						</CfInput>
						<CfInput v-model="caCert" class="_formBlock" :disabled="saving">
							<template #label>CA 证书路径（可选）</template>
							<template #caption>自建 MinIO / 私有 S3 网关需要时填写 PEM 文件路径。</template>
						</CfInput>
					</template>

					<CfInfo v-if="formError !== null" warn class="_formBlock">
						{{ formError }}
					</CfInfo>
					<CfInfo v-else-if="saved" class="_formBlock">
						已保存并立即生效。当前来源：{{ stored ? "管理台设置" : "config.toml" }}。
					</CfInfo>

					<CfButton type="submit" primary :disabled="saving">
						{{ saving ? "保存中…" : "保存" }}
					</CfButton>
				</form>

				<CfInfo class="notice">
					这里保存的设置覆盖 <code>config.toml</code> 的 <code>[media]</code>，无需重启；访问密钥与私钥保存在实例数据库中，接口只返回「已配置」状态。
					切换后端不会移动已有文件，历史文件仍留在原来的存储里。
				</CfInfo>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script setup lang="ts">
// Runtime object storage configuration. The server wraps the backend in a
// media.SwitchableStorage, so saving this form swaps the live storage
// without restarting; the persisted override lives in instance_meta and
// wins over config.toml from the next start as well.
import { computed, onMounted, ref } from "vue";
import { ApiError, api } from "@/api/client";
import type { ObjectStorageInfo } from "@/api/types";
import { isAdmin } from "@/account";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";
import MkError from "@/components/global/MkError.vue";
import CfA from "@/components/ui/CfA.vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfInput from "@/components/ui/CfInput.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import CfSelect from "@/components/ui/CfSelect.vue";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageAdminObjectStorage" });

const admin = computed<boolean>(() => isAdmin());
const info = ref<ObjectStorageInfo | null>(null);
const stored = ref(false);
const loading = ref(true);
const loadError = ref<string | null>(null);
const saving = ref(false);
const saved = ref(false);
const formError = ref<string | null>(null);

const backend = ref<"local" | "s3">("local");
const localPath = ref<string | number | null>("");
const endpoint = ref<string | number | null>("");
const bucket = ref<string | number | null>("");
const region = ref<string | number | null>("");
const accessKey = ref<string | number | null>("");
const secretKey = ref("");
const useSSL = ref(false);
const publicBaseUrl = ref<string | number | null>("");
const caCert = ref<string | number | null>("");

function errorText(cause: unknown): string {
	return cause instanceof ApiError ? cause.message : String(cause);
}

function apply(next: ObjectStorageInfo): void {
	info.value = next;
	stored.value = next.stored;
	backend.value = next.backend === "s3" ? "s3" : "local";
	localPath.value = next.localPath ?? "";
	const s3 = next.s3;
	endpoint.value = s3?.endpoint ?? "";
	bucket.value = s3?.bucket ?? "";
	region.value = s3?.region ?? "";
	accessKey.value = "";
	secretKey.value = "";
	useSSL.value = s3?.useSSL ?? false;
	publicBaseUrl.value = s3?.publicBaseUrl ?? "";
	caCert.value = s3?.caCert ?? "";
}

async function load(): Promise<void> {
	if (!admin.value) return;
	loading.value = true;
	loadError.value = null;
	try {
		apply(await api("admin/object-storage", {}));
	} catch (cause) {
		loadError.value = errorText(cause);
	} finally {
		loading.value = false;
	}
}

async function save(): Promise<void> {
	if (saving.value) return;
	formError.value = null;
	saved.value = false;
	const body =
		backend.value === "local"
			? {
					backend: "local" as const,
					localPath: String(localPath.value ?? "").trim(),
				}
			: {
					backend: "s3" as const,
					s3: {
						endpoint: String(endpoint.value ?? "").trim(),
						bucket: String(bucket.value ?? "").trim(),
						region: String(region.value ?? "").trim(),
						accessKey: String(accessKey.value ?? "").trim(),
						secretKey: secretKey.value.trim(),
						useSSL: useSSL.value,
						publicBaseUrl: String(publicBaseUrl.value ?? "").trim(),
						caCert: String(caCert.value ?? "").trim(),
					},
				};
	if (body.backend === "local" && body.localPath === "") {
		formError.value = "本地路径不能为空。";
		return;
	}
	if (
		body.backend === "s3" &&
		(body.s3.endpoint === "" || body.s3.bucket === "")
	) {
		formError.value = "Endpoint 与 Bucket 都不能为空。";
		return;
	}
	saving.value = true;
	try {
		apply(await api("admin/object-storage/update", body));
		saved.value = true;
	} catch (cause) {
		formError.value = errorText(cause);
	} finally {
		saving.value = false;
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
.checkRow {
	display: flex;
	align-items: center;
	gap: 8px;
	margin-bottom: 12px;
	font-size: 0.9em;

	input {
		width: 16px;
		height: 16px;
	}
}

.notice {
	margin-top: 16px;
}
</style>
