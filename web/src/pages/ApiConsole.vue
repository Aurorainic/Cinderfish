<!-- Ported from firefish@b425f61:packages/client/src/pages/api-console.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :actions="headerActions" />
		</template>
		<MkSpacer :content-max="700">
			<div class="_formRoot">
				<div class="_formBlock">
					<CfInput
						v-model="endpoint"
						:datalist="endpointNames"
						@enter="send()"
					>
						<template #label>Endpoint</template>
					</CfInput>
					<CfTextarea v-model="body" class="_formBlock" code tall>
						<template #label>Params (JSON)</template>
					</CfTextarea>
					<CfButton
						class="_formBlock"
						primary
						:disabled="sending"
						@click="send()"
					>
						<CfEllipsis v-if="sending" />
						<template v-else>
							<i class="ph-paper-plane-tilt ph-bold ph-lg"></i>
							{{ t("send") }}
						</template>
					</CfButton>
				</div>

				<CfInfo v-if="errorMessage !== null" class="_formBlock" warn>
					{{ errorMessage }}
				</CfInfo>

				<CfLoading v-if="sending" />
				<div v-else-if="responseText !== ''" class="_formBlock">
					<CfTextarea
						:model-value="responseText"
						code
						tall
						readonly
					>
						<template #label>Response</template>
					</CfTextarea>
				</div>
				<p v-else-if="errorMessage === null" class="hint">
					填写端点名称与 JSON 参数后发送；响应（或错误）会显示在这里。请求以当前登录身份发出，令牌由
					<code>@/api/client</code> 自动附加。
				</p>

				<CfInfo class="_formBlock" icon="info">
					相比 Firefish 的降级：后端没有 <code>endpoints</code> / <code>endpoint</code>
					端点，无法取得端点参数模板，因此发送前不会自动填充参数字段；<code>json5</code>
					不在依赖集合里，参数与响应都按标准 JSON 解析与展开。
				</CfInfo>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script lang="ts" setup>
// Route: /api-console (login required, see router/routes.ts).
//
// Firefish's console asks the backend for the endpoint list (`endpoints`) and
// for each endpoint's parameter schema (`endpoint`) so it can prefill the body.
// Neither endpoint exists on Cinderfish, so the datalist is built from the
// client's own typed endpoint table (@/api/endpoints) and the body is typed by
// hand. Every call still goes through api(), which appends the session token —
// Firefish's "With credential" switch is dropped because api() falls back to
// the shared token even when null is passed (api/client.ts).
import { computed, ref } from "vue";
import type { EndpointName } from "@/api/endpoints";
import { ENDPOINTS } from "@/api/endpoints";
import { api, ApiError } from "@/api/client";
import CfButton from "@/components/ui/CfButton.vue";
import CfEllipsis from "@/components/ui/CfEllipsis.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfInput from "@/components/ui/CfInput.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import CfTextarea from "@/components/ui/CfTextarea.vue";
import { t } from "@/i18n";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageApiConsole" });

const endpoint = ref<string | number | null>("");
const body = ref("{}");
const sending = ref(false);
const response = ref<string | null>(null);
const errorMessage = ref<string | null>(null);

const endpointNames = computed(() =>
	[...new Set(Object.values(ENDPOINTS) as EndpointName[])].sort(),
);

const responseText = computed(() => response.value ?? "");

function describe(err: unknown): string {
	if (err instanceof ApiError) {
		const code = err.code ?? err.errorId ?? "";
		return code === "" ? err.message : `${err.message} (${code})`;
	}
	return err instanceof Error ? err.message : String(err);
}

function parseParams(raw: string): Record<string, unknown> {
	const text = raw.trim();
	const parsed: unknown = text === "" ? {} : JSON.parse(text);
	if (parsed == null || typeof parsed !== "object" || Array.isArray(parsed)) {
		throw new Error("参数必须是一个 JSON 对象");
	}
	return parsed as Record<string, unknown>;
}

async function send(): Promise<void> {
	const name = String(endpoint.value ?? "").trim();
	response.value = null;
	errorMessage.value = null;

	if (name === "") {
		errorMessage.value = "请先填写端点名称。";
		return;
	}

	let params: Record<string, unknown>;
	try {
		params = parseParams(body.value);
	} catch (err) {
		errorMessage.value = `参数不是合法的 JSON 对象：${describe(err)}`;
		return;
	}

	sending.value = true;
	try {
		const result = await api<unknown>(name, params);
		response.value = JSON.stringify(result, null, 2) ?? "null";
	} catch (err) {
		errorMessage.value = describe(err);
	} finally {
		sending.value = false;
	}
}

const headerActions = computed(() => []);

definePageMetadata({
	title: "API console",
	icon: "ph-terminal-window ph-bold ph-lg",
});
</script>

<style lang="scss" scoped>
.hint {
	margin: 0;
	padding: 16px;
	font-size: 0.9em;
	color: var(--fgTransparentWeak);
}
</style>
