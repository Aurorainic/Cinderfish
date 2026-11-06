<!-- Ported from firefish@b425f61:packages/client/src/pages/settings/webhook.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :display-back-button="true" />
		</template>
		<MkSpacer :content-max="800" :margin-min="16" :margin-max="32">
			<div class="_formRoot cf-settings-webhook">
				<CfInfo class="_formBlock">
					Webhook 会在你发帖、回复或转帖时向指定地址 POST JSON。投递是异步的，
					失败不会影响发帖本身；最近一次投递的状态显示在下面的卡片里。
				</CfInfo>

				<div class="toolbar">
					<CfButton primary to="/settings/webhook/new">
						<i class="ph-plus ph-bold"></i>
						新建 Webhook
					</CfButton>
				</div>

				<CfLoading v-if="loading" />
				<MkError v-else-if="loadError !== null" @retry="load" />
				<CfInfo v-else-if="hooks.length === 0">
					还没有 Webhook。新建后，你勾选的事件会推送到该地址。
				</CfInfo>
				<ul v-else class="hooks">
					<li v-for="hook in hooks" :key="hook.id" class="hook">
						<div class="body">
							<div class="head">
								<span class="name">{{ hook.name }}</span>
								<span class="badge" :class="{ off: !hook.active }">{{
									hook.active ? "启用" : "停用"
								}}</span>
							</div>
							<div class="url _monospace">{{ hook.url }}</div>
							<div class="meta">
								<span>事件：{{ hook.on.join("、") }}</span>
								<span v-if="hook.latestSentAt !== null">
									最近投递：
									<MkTime :time="hook.latestSentAt" mode="absolute" />
									· HTTP {{ hook.latestStatus ?? "?" }}
								</span>
								<span v-else>尚未投递</span>
							</div>
						</div>
						<div class="buttons">
							<CfA class="edit" :to="`/settings/webhook/edit/${hook.id}`"
								>编辑</CfA
							>
							<CfButton
								mini
								:disabled="deletingId === hook.id"
								@click="remove(hook)"
							>
								{{ deletingId === hook.id ? "删除中…" : "删除" }}
							</CfButton>
						</div>
					</li>
				</ul>

				<CfInfo v-if="actionError !== null" warn class="actionError">
					{{ actionError }}
				</CfInfo>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue";
import { ApiError, api } from "@/api/client";
import type { Webhook } from "@/api/types";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";
import MkError from "@/components/global/MkError.vue";
import MkTime from "@/components/global/MkTime.vue";
import CfA from "@/components/ui/CfA.vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import { confirm } from "@/os";

defineOptions({ name: "CfPageSettingsWebhook" });

const hooks = ref<Webhook[]>([]);
const loading = ref(false);
const loadError = ref<string | null>(null);
const actionError = ref<string | null>(null);
const deletingId = ref<string | null>(null);

function errorText(cause: unknown): string {
	return cause instanceof ApiError ? cause.message : String(cause);
}

async function load(): Promise<void> {
	loading.value = true;
	loadError.value = null;
	try {
		hooks.value = await api("i/webhooks/list", {});
	} catch (cause) {
		hooks.value = [];
		loadError.value = errorText(cause);
	} finally {
		loading.value = false;
	}
}

async function remove(hook: Webhook): Promise<void> {
	if (deletingId.value !== null) return;
	const { canceled } = await confirm({
		type: "warning",
		text: `删除 Webhook「${hook.name}」后，事件将不再投递到该地址。确定删除吗？`,
	});
	if (canceled) return;
	deletingId.value = hook.id;
	actionError.value = null;
	try {
		await api("i/webhooks/delete", { webhookId: hook.id });
		hooks.value = hooks.value.filter((row) => row.id !== hook.id);
	} catch (cause) {
		actionError.value = errorText(cause);
	} finally {
		deletingId.value = null;
	}
}

onMounted(() => {
	void load();
});
</script>

<style lang="scss" scoped>
.toolbar {
	display: flex;
	justify-content: flex-end;
	margin-bottom: 12px;
}

.hooks {
	list-style: none;
	margin: 0;
	padding: 0;
	display: flex;
	flex-direction: column;
	gap: 8px;
}

.hook {
	display: flex;
	align-items: flex-start;
	gap: 12px;
	padding: 14px 16px;
	background: var(--panel);
	border-radius: var(--radius);

	.body {
		flex: 1;
		min-width: 0;
	}

	.head {
		display: flex;
		align-items: center;
		gap: 8px;
	}

	.name {
		font-weight: 600;
	}

	.badge {
		padding: 1px 6px;
		border-radius: 999px;
		background: var(--accented-bg, rgba(0, 0, 0, 0.08));
		font-size: 0.72em;

		&.off {
			opacity: 0.6;
		}
	}

	.url {
		margin-top: 4px;
		font-size: 0.82em;
		opacity: 0.75;
		overflow-wrap: anywhere;
	}

	.meta {
		display: flex;
		flex-wrap: wrap;
		gap: 4px 14px;
		margin-top: 6px;
		font-size: 0.78em;
		opacity: 0.65;
	}

	.buttons {
		display: flex;
		align-items: center;
		gap: 10px;
		flex: none;
	}

	.edit {
		font-size: 0.85em;
	}
}

.actionError {
	margin-top: 12px;
}
</style>
