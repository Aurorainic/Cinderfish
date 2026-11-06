<!-- Ported from firefish@b425f61:packages/client/src/pages/settings/webhook.edit.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :display-back-button="true" />
		</template>
		<MkSpacer :content-max="700" :margin-min="16" :margin-max="32">
			<div class="_formRoot cf-settings-webhook-edit">
				<CfLoading v-if="loading" />
				<MkError v-else-if="loadError !== null" @retry="load" />
				<template v-else-if="hook != null">
					<WebhookEditor
						:initial="hook"
						:saving="saving"
						:error="error"
						submit-label="保存"
						@submit="save"
					>
						<template #actions>
							<CfButton :disabled="deleting" @click="remove">
								{{ deleting ? "删除中…" : "删除" }}
							</CfButton>
						</template>
					</WebhookEditor>
				</template>
				<CfInfo v-else warn>这个 Webhook 不存在或不属于当前账号。</CfInfo>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import { ApiError, api } from "@/api/client";
import type { Webhook } from "@/api/types";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";
import MkError from "@/components/global/MkError.vue";
import WebhookEditor from "@/components/WebhookEditor.vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import { confirm } from "@/os";

defineOptions({ name: "CfPageSettingsWebhookEdit" });

const route = useRoute();
const router = useRouter();

const id = String(route.params.webhookId ?? "");
const hook = ref<Webhook | null>(null);
const loading = ref(true);
const loadError = ref<string | null>(null);
const saving = ref(false);
const deleting = ref(false);
const error = ref<string | null>(null);

function errorText(cause: unknown): string {
	return cause instanceof ApiError ? cause.message : String(cause);
}

async function load(): Promise<void> {
	loading.value = true;
	loadError.value = null;
	try {
		hook.value = await api("i/webhooks/show", { webhookId: id });
	} catch (cause) {
		hook.value = null;
		loadError.value = errorText(cause);
	} finally {
		loading.value = false;
	}
}

async function save(payload: {
	name: string;
	url: string;
	secret: string;
	on: string[];
	active: boolean;
}): Promise<void> {
	if (saving.value || hook.value == null) return;
	saving.value = true;
	error.value = null;
	try {
		hook.value = await api("i/webhooks/update", {
			webhookId: hook.value.id,
			...payload,
		});
	} catch (cause) {
		error.value = errorText(cause);
	} finally {
		saving.value = false;
	}
}

async function remove(): Promise<void> {
	if (deleting.value || hook.value == null) return;
	const { canceled } = await confirm({
		type: "warning",
		text: `删除 Webhook「${hook.value.name}」后，事件将不再投递到该地址。确定删除吗？`,
	});
	if (canceled) return;
	deleting.value = true;
	error.value = null;
	try {
		await api("i/webhooks/delete", { webhookId: hook.value.id });
		await router.push("/settings/webhook");
	} catch (cause) {
		error.value = errorText(cause);
	} finally {
		deleting.value = false;
	}
}

onMounted(() => {
	void load();
});
</script>
