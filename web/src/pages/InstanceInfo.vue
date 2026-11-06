<!-- SPDX-License-Identifier: AGPL-3.0-only -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :display-back-button="true" />
		</template>
		<MkSpacer :content-max="600" :margin-min="16">
			<div class="cf-instance-info _formRoot">
				<form class="lookup" @submit.prevent="load()">
					<CfInput v-model="host" class="_formBlock">
						<template #label>实例主机</template>
						<template #caption>例如 <code>misskey.io</code>；可以省略 <code>https://</code>。</template>
					</CfInput>
					<CfButton type="submit" primary :disabled="loading || host.trim() === ''">
						查询
					</CfButton>
				</form>

				<CfLoading v-if="loading" />
				<CfInfo v-else-if="errorText != null" warn class="feedback">
					{{ errorText }}
				</CfInfo>
				<CfInfo v-else-if="instance == null" class="feedback">
					本实例没有收录 <code>{{ lastHost }}</code> 的任何用户或帖子。
					远端信息不会主动联网抓取，只来自已经进入本实例数据库的内容。
				</CfInfo>
				<template v-else>
					<section class="card">
						<h2 class="name">{{ instance.name ?? instance.host }}</h2>
						<p class="host _monospace">{{ instance.host }}</p>
						<p v-if="instance.description != null" class="description">
							{{ instance.description }}
						</p>
					</section>

					<section class="card">
						<h2 class="sectionTitle">软件</h2>
						<CfKeyValue class="kv">
							<template #key>名称</template>
							<template #value>{{ instance.softwareName ?? "未知" }}</template>
						</CfKeyValue>
						<CfKeyValue class="kv">
							<template #key>版本</template>
							<template #value>{{ instance.softwareVersion ?? "未知" }}</template>
						</CfKeyValue>
						<CfKeyValue class="kv">
							<template #key>开放注册</template>
							<template #value>
								{{ instance.openRegistrations == null ? "未知" : instance.openRegistrations ? "是" : "否" }}
							</template>
						</CfKeyValue>
					</section>

					<section class="card">
						<h2 class="sectionTitle">本实例收录</h2>
						<CfKeyValue class="kv">
							<template #key>用户</template>
							<template #value>{{ instance.usersCount }}</template>
						</CfKeyValue>
						<CfKeyValue class="kv">
							<template #key>帖子</template>
							<template #value>{{ instance.notesCount }}</template>
						</CfKeyValue>
					</section>

					<CfInfo class="note">
						远端实例的软件与注册状态没有在线校验；用户与帖子计数只统计本库已收录的行。
					</CfInfo>
				</template>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script lang="ts" setup>
// Firefish's /instance-info/:host page. Cinderfish's federation/show-instance
// answers the local host from runtime settings and any other host from rows it
// already stores, so this page never makes the server fetch a remote address.
import { onMounted, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import { api } from "@/api/client";
import type { FederationInstance } from "@/api/types";
import { definePageMetadata } from "@/scripts/page-metadata";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfInput from "@/components/ui/CfInput.vue";
import CfKeyValue from "@/components/ui/CfKeyValue.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";

defineOptions({ name: "CfPageInstanceInfo" });

const route = useRoute();
const router = useRouter();

const host = ref(String(route.params.host ?? ""));
const lastHost = ref("");
const loading = ref(false);
const errorText = ref<string | null>(null);
const instance = ref<FederationInstance | null>(null);

function normalize(value: string): string {
	return value
		.trim()
		.replace(/^https?:\/\//i, "")
		.replace(/\/+$/, "")
		.toLowerCase();
}

async function load(): Promise<void> {
	const target = normalize(host.value);
	if (target === "") return;
	lastHost.value = target;
	loading.value = true;
	errorText.value = null;
	instance.value = null;
	try {
		instance.value = await api("federation/show-instance", { host: target });
		if (target !== String(route.params.host ?? "")) {
			void router.replace(`/instance-info/${encodeURIComponent(target)}`);
		}
	} catch (error) {
		errorText.value =
			error instanceof Error && error.message !== ""
				? `查询失败：${error.message}`
				: "查询失败。";
	} finally {
		loading.value = false;
	}
}

onMounted(() => {
	if (host.value !== "") void load();
});

definePageMetadata({
	title: "实例信息",
	icon: "ph-planet ph-bold ph-lg",
});
</script>

<style lang="scss" scoped>
.cf-instance-info {
	.lookup {
		margin-block-end: 1.5em;
	}

	.feedback {
		margin-block-end: 1em;
	}

	.card {
		padding: 16px;
		margin-block-end: 12px;
		background: var(--panel);
		border: 1px solid var(--divider);
		border-radius: var(--radius);

		.name {
			margin: 0;
			font-size: 1.15em;
		}

		.host {
			margin: 4px 0 0;
			color: var(--fgTransparentWeak);
			font-size: 0.85em;
		}

		.description {
			margin: 8px 0 0;
			white-space: pre-wrap;
		}

		.sectionTitle {
			margin: 0 0 4px;
			color: var(--fg);
			font-size: 1em;
			opacity: 0.85;
		}

		.kv {
			margin-block-start: 8px;
		}
	}

	.note {
		font-size: 0.85em;
	}
}
</style>
