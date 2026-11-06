<!-- Ported from firefish@b425f61:packages/client/src/pages/settings/webhook.new.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :display-back-button="true" />
		</template>
		<MkSpacer :content-max="700" :margin-min="16" :margin-max="32">
			<div class="_formRoot cf-settings-webhook-new">
				<CfInfo class="_formBlock">
					实例会在事件发生时向你填写的地址 POST JSON：
					<code>{ hookId, userId, eventId, createdAt, type, body }</code>，
					并用 <code>X-Firefish-Hook-Secret</code> 头带上密钥。
				</CfInfo>
				<WebhookEditor
					:saving="saving"
					:error="error"
					submit-label="创建"
					@submit="create"
				/>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { useRouter } from "vue-router";
import { ApiError, api } from "@/api/client";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";
import WebhookEditor from "@/components/WebhookEditor.vue";
import CfInfo from "@/components/ui/CfInfo.vue";

defineOptions({ name: "CfPageSettingsWebhookNew" });

const router = useRouter();
const saving = ref(false);
const error = ref<string | null>(null);

async function create(payload: {
	name: string;
	url: string;
	secret: string;
	on: string[];
}): Promise<void> {
	if (saving.value) return;
	saving.value = true;
	error.value = null;
	try {
		await api("i/webhooks/create", payload);
		await router.push("/settings/webhook");
	} catch (cause) {
		error.value = cause instanceof ApiError ? cause.message : String(cause);
	} finally {
		saving.value = false;
	}
}
</script>
