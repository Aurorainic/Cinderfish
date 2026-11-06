<!-- Ported from firefish@b425f61:packages/client/src/pages/registry.value.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :actions="headerActions" display-back-button />
		</template>
		<MkSpacer :content-max="600" :margin-min="16">
			<div class="_formRoot">
				<CfInfo v-if="me == null" warn>{{ t("signinRequired") }}</CfInfo>
				<template v-else-if="key === null">
					<CfInfo warn>这条路径里没有键名。</CfInfo>
					<div class="actions">
						<CfA :to="keysHref">返回键列表</CfA>
					</div>
				</template>
				<template v-else>
					<MkLoading v-if="loading" />
					<MkError v-else-if="loadFailed" @retry="fetchValue" />
					<template v-else-if="notFound">
						<CfInfo warn>
							键 <span class="_monospace">{{ key }}</span>
							不存在，可能已被删除。
						</CfInfo>
						<div class="actions">
							<CfA :to="keysHref">返回键列表</CfA>
						</div>
					</template>
					<template v-else>
						<CfInfo warn>
							这些值是拥有它的客户端功能自己读写的。随意改动可能让对应功能出错或丢失设置。
						</CfInfo>

						<FormSplit>
							<CfKeyValue>
								<template #key>域</template>
								<template #value>{{ t("system") }}</template>
							</CfKeyValue>
							<CfKeyValue>
								<template #key>作用域</template>
								<template #value
									><span class="_monospace">{{
										scope.join(" / ")
									}}</span></template
								>
							</CfKeyValue>
							<CfKeyValue>
								<template #key>键</template>
								<template #value
									><span class="_monospace">{{
										key
									}}</span></template
								>
							</CfKeyValue>
							<CfKeyValue>
								<template #key>{{ t("updatedAt") }}</template>
								<template #value>
									<MkTime
										v-if="detail !== null"
										:time="detail.updatedAt"
										mode="detail"
									/>
									<span v-else>—</span>
								</template>
							</CfKeyValue>
						</FormSplit>

						<FormTextarea
							v-model="editor"
							code
							tall
							:disabled="saving || removing"
						>
							<template #label>{{ t("value") }}（JSON）</template>
							<template #caption
								>提交前按 JSON5 解析（允许注释与尾逗号），失败不会写入。</template
							>
						</FormTextarea>

						<CfInfo v-if="saveError !== null" warn>
							{{ saveError }}
						</CfInfo>

						<div class="actions">
							<CfButton
								primary
								:disabled="saving || removing"
								@click="save"
							>
								<i class="ph-floppy-disk-back ph-bold ph-lg"></i>
								{{ t("save") }}
							</CfButton>
							<CfButton
								danger
								:disabled="saving || removing"
								@click="remove"
							>
								<i class="ph-trash ph-bold ph-lg"></i>
								{{ t("delete") }}
							</CfButton>
						</div>
					</template>
				</template>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script setup lang="ts">
// Route: /registry/value/system/:path(.*)* — the last segment is the key, the
// ones before it are the scope under the fixed `system` domain.
//
// Firefish serialises the value with `JSON5.stringify` and re-parses with the
// `json5` package. Cinderfish has neither (no `json5` dependency), so the editor
// shows `JSON.stringify(value, null, 2)` and parses with `parseJson5`
// (@/themes/json5), which accepts comments, unquoted keys and trailing commas —
// the syntax the registry values are written in.
//
// Saving no longer asks for confirmation (upstream `os.confirm({type:"warning"})`)
// because the page already carries the warning banner and every save is
// reversible by editing again; deleting still confirms, as it is not.
import { computed, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import { me } from "@/account";
import { api } from "@/api/client";
import { ApiError, hasErrorCode } from "@/api/errors";
import { ENDPOINTS } from "@/api/endpoints";
import type { RegistryDetail } from "@/api/types";
import { confirm, success } from "@/os";
import { parseJson5 } from "@/themes/json5";
import { t } from "@/i18n";
import { definePageMetadata } from "@/scripts/page-metadata";
import CfA from "@/components/ui/CfA.vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfKeyValue from "@/components/ui/CfKeyValue.vue";
import FormSplit from "@/components/form/split.vue";
import FormTextarea from "@/components/form/textarea.vue";
import MkError from "@/components/global/MkError.vue";
import MkLoading from "@/components/global/MkLoading.vue";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";
import MkTime from "@/components/global/MkTime.vue";

defineOptions({ name: "CfPageRegistryValue" });

const route = useRoute();
const router = useRouter();

/** `:path(.*)*` arrives as a string or an array depending on how many segments. */
const pathSegments = computed<string[]>(() => {
	const raw = route.params.path;
	const parts = Array.isArray(raw) ? raw : raw == null ? [] : [raw];
	return parts
		.flatMap((part) => String(part).split("/"))
		.filter((part) => part !== "");
});

const key = computed<string | null>(() => {
	const segments = pathSegments.value;
	if (segments.length === 0) return null;
	return segments[segments.length - 1] ?? null;
});

const scope = computed<string[]>(() => [
	"system",
	...pathSegments.value.slice(0, -1),
]);

/** Back to the key list of this scope (`/registry/keys/system[/…]`). */
const keysHref = computed(() => {
	const path = pathSegments.value
		.slice(0, -1)
		.map((segment) => encodeURIComponent(segment))
		.join("/");
	return path === "" ? "/registry/keys/system" : `/registry/keys/system/${path}`;
});

const detail = ref<RegistryDetail | null>(null);
const editor = ref("");
const loading = ref(true);
const loadFailed = ref(false);
const notFound = ref(false);
const saving = ref(false);
const removing = ref(false);
const saveError = ref<string | null>(null);

function describe(error: unknown): string {
	if (error instanceof ApiError) {
		const code = error.code ?? error.errorId ?? "";
		return code === "" ? error.message : `${error.message}（${code}）`;
	}
	return error instanceof Error ? error.message : String(error);
}

function toEditorText(value: unknown): string {
	const text = JSON.stringify(value, null, 2);
	return text ?? "null";
}

async function fetchValue(): Promise<void> {
	const currentKey = key.value;
	if (me.value == null || currentKey === null) {
		loading.value = false;
		return;
	}
	loading.value = true;
	loadFailed.value = false;
	notFound.value = false;
	saveError.value = null;
	try {
		const result = await api(ENDPOINTS.registryGetDetail, {
			key: currentKey,
			scope: scope.value,
		});
		detail.value = result;
		editor.value = toEditorText(result.value);
	} catch (error) {
		if (hasErrorCode(error, "NO_SUCH_KEY")) {
			notFound.value = true;
			detail.value = null;
		} else {
			loadFailed.value = true;
		}
	} finally {
		loading.value = false;
	}
}

async function save(): Promise<void> {
	const currentKey = key.value;
	if (saving.value || currentKey === null) return;
	saveError.value = null;

	let value: unknown;
	try {
		value = parseJson5(editor.value);
	} catch (error) {
		saveError.value = `${t("invalidValue")} ${describe(error)}`;
		return;
	}

	saving.value = true;
	try {
		await api(ENDPOINTS.registrySet, {
			scope: scope.value,
			key: currentKey,
			value,
		});
		success();
		await fetchValue();
	} catch (error) {
		saveError.value = describe(error);
	} finally {
		saving.value = false;
	}
}

async function remove(): Promise<void> {
	const currentKey = key.value;
	if (removing.value || currentKey === null) return;
	const { canceled } = await confirm({
		type: "warning",
		text: t("deleteConfirm"),
	});
	if (canceled) return;

	removing.value = true;
	saveError.value = null;
	try {
		await api(ENDPOINTS.registryRemove, {
			scope: scope.value,
			key: currentKey,
		});
		success();
		await router.push(keysHref.value);
	} catch (error) {
		saveError.value = describe(error);
	} finally {
		removing.value = false;
	}
}

const headerActions = computed(() => []);

watch(pathSegments, () => void fetchValue(), { immediate: true });

definePageMetadata({
	title: t("registry"),
	icon: "ph-gear-six ph-bold ph-lg",
});
</script>

<style lang="scss" scoped>
.actions {
	display: flex;
	justify-content: flex-end;
	gap: 8px;
	margin-block-start: 12px;
}
</style>
