<!-- Ported from firefish@b425f61:packages/client/src/pages/registry.keys.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :actions="headerActions" display-back-button />
		</template>
		<MkSpacer :content-max="600" :margin-min="16">
			<div class="_formRoot">
				<CfInfo v-if="me == null" warn>{{ t("signinRequired") }}</CfInfo>
				<template v-else>
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
					</FormSplit>

					<FormSection>
						<template #label>键</template>
						<MkLoading v-if="loading" />
						<MkError v-else-if="loadFailed" @retry="fetchKeys" />
						<template v-else>
							<CfInfo v-if="keys.length === 0">
								这个作用域下还没有键。用下面的表单添加一个。
							</CfInfo>
							<div v-else class="_formLinks">
								<div
									v-for="key in keys"
									:key="key"
									class="keyRow"
								>
									<CfA
										class="keyLink _monospace"
										:to="valueHref(key)"
										>{{ key }}</CfA
									>
									<CfButton
										mini
										danger
										:title="removeTitle(key)"
										:disabled="removingKey === key"
										@click="remove(key)"
									>
										<i class="ph-trash ph-bold"></i>
									</CfButton>
								</div>
							</div>
							<CfInfo v-if="removeError !== null" warn>
								{{ removeError }}
							</CfInfo>
						</template>
					</FormSection>

					<FormSection>
						<template #label>添加键</template>
						<FormInput v-model="newKey" :disabled="saving">
							<template #label>键名</template>
						</FormInput>
						<FormTextarea
							v-model="newValue"
							code
							tall
							:disabled="saving"
							placeholder="{ }"
						>
							<template #label>{{ t("value") }}（JSON5）</template>
							<template #caption
								>写入到这个作用域（<span class="_monospace"
									>{{ scope.join("/") }}</span
								>）。键已存在时会被覆盖。</template
							>
						</FormTextarea>

						<CfInfo v-if="saveError !== null" warn>
							{{ saveError }}
						</CfInfo>

						<div class="actions">
							<CfButton primary :disabled="saving" @click="add">
								{{ t("add") }}
							</CfButton>
						</div>
					</FormSection>
				</template>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script setup lang="ts">
// Route: /registry/keys/system/:path(.*)*
//
// The scope is fixed to the `system` domain: the route carries only the
// segments after it, so scope = ["system", ...segments]. The same segments are
// re-appended for the value route, which is why they are kept as one list.
//
// Degraded against Firefish: upstream calls `i/registry/keys-with-type` and
// prints each value's type as a suffix. The backend exposes `i/registry/keys`
// (string[]) only, so rows show the key alone.
//
// The add form replaces upstream's `os.form` modal; the scope is taken from the
// route instead of being editable, because writing into another scope from this
// page would leave the user on a list that does not contain the new key.
import { computed, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { me } from "@/account";
import { api } from "@/api/client";
import { ENDPOINTS } from "@/api/endpoints";
import { confirm, success } from "@/os";
import { parseJson5 } from "@/themes/json5";
import { t } from "@/i18n";
import { definePageMetadata } from "@/scripts/page-metadata";
import CfA from "@/components/ui/CfA.vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfKeyValue from "@/components/ui/CfKeyValue.vue";
import FormInput from "@/components/form/input.vue";
import FormSection from "@/components/form/section.vue";
import FormSplit from "@/components/form/split.vue";
import FormTextarea from "@/components/form/textarea.vue";
import MkError from "@/components/global/MkError.vue";
import MkLoading from "@/components/global/MkLoading.vue";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";

defineOptions({ name: "CfPageRegistryKeys" });

const route = useRoute();

/** `:path(.*)*` arrives as a string or an array depending on how many segments. */
const pathSegments = computed<string[]>(() => {
	const raw = route.params.path;
	const parts = Array.isArray(raw) ? raw : raw == null ? [] : [raw];
	return parts
		.flatMap((part) => String(part).split("/"))
		.filter((part) => part !== "");
});

const scope = computed<string[]>(() => ["system", ...pathSegments.value]);

const keys = ref<string[]>([]);
const loading = ref(true);
const loadFailed = ref(false);

const newKey = ref("");
const newValue = ref("");
const saving = ref(false);
const saveError = ref<string | null>(null);
const removeError = ref<string | null>(null);
const removingKey = ref<string | null>(null);

function describe(error: unknown): string {
	return error instanceof Error ? error.message : String(error);
}

function removeTitle(key: string): string {
	return `${t("delete")}: ${key}`;
}

function valueHref(key: string): string {
	const path = [...pathSegments.value, key]
		.map((segment) => encodeURIComponent(segment))
		.join("/");
	return `/registry/value/system/${path}`;
}

async function fetchKeys(): Promise<void> {
	if (me.value == null) {
		loading.value = false;
		return;
	}
	loading.value = true;
	loadFailed.value = false;
	try {
		const result = await api(ENDPOINTS.registryKeys, {
			scope: scope.value,
		});
		keys.value = result
			.slice()
			.sort((a, b) => a.localeCompare(b));
	} catch {
		loadFailed.value = true;
	} finally {
		loading.value = false;
	}
}

async function add(): Promise<void> {
	if (saving.value) return;
	saveError.value = null;

	const key = newKey.value.trim();
	if (key === "") {
		saveError.value = "请填写键名。";
		return;
	}

	let value: unknown;
	try {
		value = parseJson5(newValue.value);
	} catch (error) {
		saveError.value = `值不是合法的 JSON5：${describe(error)}`;
		return;
	}

	saving.value = true;
	try {
		await api(ENDPOINTS.registrySet, { scope: scope.value, key, value });
		newKey.value = "";
		newValue.value = "";
		success();
		await fetchKeys();
	} catch (error) {
		saveError.value = describe(error);
	} finally {
		saving.value = false;
	}
}

async function remove(key: string): Promise<void> {
	removeError.value = null;
	const { canceled } = await confirm({
		type: "warning",
		text: `删除键 ${key}？这会同时删除它保存的值。`,
	});
	if (canceled) return;

	removingKey.value = key;
	try {
		await api(ENDPOINTS.registryRemove, { scope: scope.value, key });
		success();
		await fetchKeys();
	} catch (error) {
		removeError.value = describe(error);
	} finally {
		removingKey.value = null;
	}
}

const headerActions = computed(() => []);

watch(pathSegments, () => void fetchKeys(), { immediate: true });

definePageMetadata({
	title: t("registry"),
	icon: "ph-gear-six ph-bold ph-lg",
});
</script>

<style lang="scss" scoped>
.keyRow {
	display: flex;
	align-items: center;
	gap: 8px;

	> .keyLink {
		flex: 1;
		min-inline-size: 0;
		padding: 10px 14px;
		border-radius: var(--radius);
		background: var(--panel);
		color: var(--fg);
		text-decoration: none;
		overflow-wrap: anywhere;

		&:hover {
			background: var(--panelHighlight);
		}
	}
}

.actions {
	display: flex;
	justify-content: flex-end;
	margin-block-start: 12px;
}
</style>
