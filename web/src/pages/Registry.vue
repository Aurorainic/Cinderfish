<!-- Ported from firefish@b425f61:packages/client/src/pages/registry.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :actions="headerActions" />
		</template>
		<MkSpacer :content-max="600" :margin-min="16">
			<div class="_formRoot">
				<CfInfo v-if="me == null" warn>{{ t("signinRequired") }}</CfInfo>
				<template v-else>
					<MkLoading v-if="loading" />
					<MkError v-else-if="loadFailed" @retry="fetchScopes" />
					<template v-else>
						<FormSection>
							<template #label>{{ t("system") }}</template>
							<div v-if="rows.length === 0" class="empty">
								<CfInfo>
									注册表里还没有任何 scope。用下面的表单创建第一个键，它会出现在这里。
								</CfInfo>
							</div>
							<div v-else class="_formLinks">
								<template v-for="row in rows" :key="row.label">
									<CfA
										v-if="row.href !== null"
										class="linkRow _monospace"
										:to="row.href"
										>{{ row.label }}</CfA
									>
									<div v-else class="linkRow unsupported _monospace">
										<span class="label">{{ row.label }}</span>
										<span class="hint"
											>非 system 域，本页只提供 system 域的键列表</span
										>
									</div>
								</template>
							</div>
						</FormSection>

						<FormSection>
							<template #label>创建键</template>
							<FormInput v-model="createScope" :disabled="creating">
								<template #label>作用域</template>
								<template #caption
									>用 / 分隔的层级。留空等于
									<code>system</code>。</template
								>
							</FormInput>
							<FormInput v-model="createKey" :disabled="creating">
								<template #label>键名</template>
							</FormInput>
							<FormTextarea
								v-model="createValue"
								code
								tall
								:disabled="creating"
								placeholder="{ }"
							>
								<template #label>{{ t("value") }}（JSON5）</template>
								<template #caption
									>可以是对象、数组、字符串、数字、布尔或
									null；支持注释与尾逗号。</template
								>
							</FormTextarea>

							<CfInfo v-if="createError !== null" warn>
								{{ createError }}
							</CfInfo>

							<div class="actions">
								<CfButton
									primary
									:disabled="creating"
									@click="create"
								>
									{{ t("create") }}
								</CfButton>
							</div>
						</FormSection>
					</template>
				</template>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script setup lang="ts">
// Route: /registry.
//
// Firefish's registry overview lists the caller's scopes as links and opens a
// modal form (`os.form`) to create a key. Cinderfish has no `os.form`, so the
// create form is inline below the list; it stays visible in the empty state so
// the first key can actually be created.
//
// The keys routes are fixed to the `system` domain
// (`/registry/keys/system/:path(.*)*`, scope = ["system", ...path]), so a scope
// is linked through its segments *after* the leading "system" — joining the
// whole scope would duplicate the domain segment and land on a different scope.
// Scopes in another domain have no page to open and are rendered as text.
//
// `i/registry/scopes` returns string[][] (one entry per distinct scope); there
// is no key-count endpoint, so rows show the scope only.
import { computed, onMounted, ref } from "vue";
import { me } from "@/account";
import { api } from "@/api/client";
import { ENDPOINTS } from "@/api/endpoints";
import { success } from "@/os";
import { parseJson5 } from "@/themes/json5";
import { t } from "@/i18n";
import { definePageMetadata } from "@/scripts/page-metadata";
import CfA from "@/components/ui/CfA.vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import FormInput from "@/components/form/input.vue";
import FormSection from "@/components/form/section.vue";
import FormTextarea from "@/components/form/textarea.vue";
import MkError from "@/components/global/MkError.vue";
import MkLoading from "@/components/global/MkLoading.vue";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";

defineOptions({ name: "CfPageRegistry" });

const scopes = ref<string[][]>([]);
const loading = ref(true);
const loadFailed = ref(false);

const createScope = ref("system");
const createKey = ref("");
const createValue = ref("");
const creating = ref(false);
const createError = ref<string | null>(null);

function describe(error: unknown): string {
	return error instanceof Error ? error.message : String(error);
}

/** `/registry/keys/system/<segments>` for a scope in the system domain. */
function keysHref(scope: string[]): string | null {
	if (scope[0] !== "system") return null;
	const path = scope.slice(1).map(encodeURIComponent).join("/");
	return path === "" ? "/registry/keys/system" : `/registry/keys/system/${path}`;
}

const rows = computed(() =>
	scopes.value.map((scope) => ({
		label: scope.join(" / "),
		href: keysHref(scope),
	})),
);

/** Text field → scope array; an empty field means the root system scope. */
function parseScope(raw: string): string[] {
	const segments = raw
		.split("/")
		.map((segment) => segment.trim())
		.filter((segment) => segment !== "");
	return segments.length === 0 ? ["system"] : segments;
}

async function fetchScopes(): Promise<void> {
	loading.value = true;
	loadFailed.value = false;
	try {
		const result = await api(ENDPOINTS.registryScopes, {});
		scopes.value = result
			.slice()
			.sort((a, b) => a.join("/").localeCompare(b.join("/")));
	} catch {
		loadFailed.value = true;
	} finally {
		loading.value = false;
	}
}

async function create(): Promise<void> {
	if (creating.value) return;
	createError.value = null;

	const key = createKey.value.trim();
	if (key === "") {
		createError.value = "请填写键名。";
		return;
	}

	let value: unknown;
	try {
		value = parseJson5(createValue.value);
	} catch (error) {
		createError.value = `值不是合法的 JSON5：${describe(error)}`;
		return;
	}

	creating.value = true;
	try {
		await api(ENDPOINTS.registrySet, {
			scope: parseScope(createScope.value),
			key,
			value,
		});
		createKey.value = "";
		createValue.value = "";
		success();
		await fetchScopes();
	} catch (error) {
		createError.value = describe(error);
	} finally {
		creating.value = false;
	}
}

const headerActions = computed(() => []);

onMounted(() => {
	if (me.value != null) void fetchScopes();
	else loading.value = false;
});

definePageMetadata({
	title: t("registry"),
	icon: "ph-gear-six ph-bold ph-lg",
});
</script>

<style lang="scss" scoped>
.empty {
	margin-block-end: 12px;
}

.linkRow {
	display: flex;
	align-items: center;
	padding: 12px 16px;
	border-radius: var(--radius);
	background: var(--panel);
	color: var(--fg);
	text-decoration: none;
	overflow-wrap: anywhere;

	&:hover {
		background: var(--panelHighlight);
	}

	&.unsupported {
		flex-direction: column;
		align-items: flex-start;
		gap: 4px;
		opacity: 0.7;

		&:hover {
			background: var(--panel);
		}

		> .hint {
			font-size: 0.8em;
			color: var(--fgTransparentWeak);
		}
	}
}

.actions {
	display: flex;
	justify-content: flex-end;
	margin-block-start: 12px;
}
</style>
