<!-- Ported from firefish@b425f61:packages/client/src/pages/admin/instance-block.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :display-back-button="true" />
		</template>
		<MkSpacer :content-max="700" :margin-min="16" :margin-max="32">
			<div v-if="!admin" class="cf-admin-instance-block">
				<CfInfo warn>
					需要管理员权限，无法管理实例屏蔽。
					<CfA to="/" class="_link">返回首页</CfA>
				</CfInfo>
			</div>
			<div v-else class="cf-admin-instance-block">
				<CfLoading v-if="loading" />
				<MkError v-else-if="loadError !== null" @retry="load" />
				<form v-else class="_formRoot" @submit.prevent="save">
					<CfTextarea
						v-model="hostsText"
						class="_formBlock"
						tall
						code
						:disabled="saving"
						placeholder="spam.example&#10;bad.example"
					>
						<template #label>被屏蔽的实例主机（每行一个）</template>
						<template #caption>
							保存后立即生效：来自这些主机的 ActivityPub 投递会被拒绝，发往它们的投递也会停止。
							匹配是精确的（<code>bad.example</code> 不会自动匹配 <code>sub.bad.example</code>），
							需要时请把子域也列一行。
						</template>
					</CfTextarea>

					<CfInfo v-if="error !== null" warn class="_formBlock">
						{{ error }}
					</CfInfo>
					<CfInfo v-else-if="saved" class="_formBlock">
						已保存，共 {{ hostList.length }} 个被屏蔽实例。
					</CfInfo>

					<CfButton type="submit" primary :disabled="saving || !changed">
						{{ saving ? "保存中…" : "保存" }}
					</CfButton>
				</form>

				<CfInfo class="notice">
					屏蔽不会删除已经收到的帖子与文件，也不会取消已有的关注关系；它只阻止之后的收发。
					只与白名单模式同时使用时，请确认目标不在白名单里。
				</CfInfo>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script setup lang="ts">
// Firefish's instance-block page edits `blockedHosts` through admin/update-meta.
// The backend enforces the list in the ActivityPub policy: outbound requests
// are refused and inbound activities are rejected after signature verification.
import { computed, onMounted, ref } from "vue";
import { ApiError, api } from "@/api/client";
import type { AdminInstanceMeta } from "@/api/types";
import { isAdmin } from "@/account";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";
import MkError from "@/components/global/MkError.vue";
import CfA from "@/components/ui/CfA.vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import CfTextarea from "@/components/ui/CfTextarea.vue";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageAdminInstanceBlock" });

const admin = computed<boolean>(() => isAdmin());

const initial = ref("");
const hostsText = ref("");
const loading = ref(true);
const loadError = ref<string | null>(null);
const saving = ref(false);
const error = ref<string | null>(null);
const saved = ref(false);

const hostList = computed(() => {
	const seen = new Set<string>();
	const out: string[] = [];
	for (const line of hostsText.value.split("\n")) {
		const host = line.trim().toLowerCase();
		if (host === "" || seen.has(host)) continue;
		seen.add(host);
		out.push(host);
	}
	return out;
});

const changed = computed(() => hostList.value.join("\n") !== initial.value);

function errorText(cause: unknown): string {
	return cause instanceof ApiError ? cause.message : String(cause);
}

async function load(): Promise<void> {
	if (!admin.value) return;
	loading.value = true;
	loadError.value = null;
	try {
		const meta: AdminInstanceMeta = await api("admin/meta", {});
		const list = [...(meta.blockedHosts ?? [])].sort();
		initial.value = list.join("\n");
		hostsText.value = list.join("\n");
	} catch (cause) {
		loadError.value = errorText(cause);
	} finally {
		loading.value = false;
	}
}

async function save(): Promise<void> {
	if (saving.value || !changed.value) return;
	saving.value = true;
	error.value = null;
	saved.value = false;
	try {
		const meta: AdminInstanceMeta = await api("admin/update-meta", {
			blockedHosts: hostList.value,
		});
		const list = [...(meta.blockedHosts ?? [])].sort();
		initial.value = list.join("\n");
		hostsText.value = list.join("\n");
		saved.value = true;
	} catch (cause) {
		error.value = errorText(cause);
	} finally {
		saving.value = false;
	}
}

onMounted(() => {
	void load();
});

definePageMetadata({
	title: "实例屏蔽",
	icon: "ph-prohibit ph-bold ph-lg",
});
</script>

<style lang="scss" scoped>
.notice {
	margin-top: 16px;
}
</style>
