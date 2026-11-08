<!-- Ported from firefish@b425f61:packages/client/src/pages/_error_.vue (AGPL-3.0-only) -->
<template>
	<div class="error-page">
		<CfLoading v-if="!loaded" />

		<div v-else class="_fullinfo">
			<i class="ph-warning-octagon ph-bold icon"></i>

			<p class="headline">
				<b><i class="ph-warning ph-bold ph-lg"></i> {{ t("pageLoadError") }}</b>
			</p>

			<p v-if="serverIsDead" class="description">
				{{ t("serverIsDead") }}
			</p>
			<p v-else-if="versionStale" class="description">
				{{ t("newVersionOfClientAvailable") }}
				{{ t("youShouldUpgradeClient") }}
			</p>
			<p v-else class="description">
				{{ t("pageLoadErrorDescription") }}
			</p>

			<div class="actions">
				<CfButton primary rounded @click="reload()">
					<i class="ph-arrows-clockwise ph-bold ph-lg"></i>
					<span>{{ t("reload") }}</span>
				</CfButton>
				<CfButton rounded link to="/">
					<i class="ph-house ph-bold ph-lg"></i>
					<span>{{ t("home") }}</span>
				</CfButton>
			</div>

			<p v-if="errorText !== ''" class="detail">ERROR: {{ errorText }}</p>
		</div>
	</div>
</template>

<script lang="ts" setup>
// Firefish's _error_ page probes `meta` to tell three situations apart: the
// server is unreachable, the running client is older than the server, or the
// page simply failed to load. The same probe is kept here so the message is
// true; the "unison reload" helper and the troubleshooting link are dropped
// (the latter points at Misskey docs that Cinderfish does not serve).
//
// The page is currently unrouted (routes.ts wires `errorComponent: CfInfo`),
// but it is a real page so a route-level error handler can use it as-is.
import { computed, onMounted, ref } from "vue";
import { api } from "@/api/client";
import CfButton from "@/components/ui/CfButton.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import { version } from "@/config";
import { t } from "@/i18n";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageError" });

const props = defineProps<{
	error?: Error;
}>();

const loaded = ref(false);
const serverIsDead = ref(false);
const serverVersion = ref<string | null>(null);

const versionStale = computed(
	() =>
		!serverIsDead.value &&
		serverVersion.value != null &&
		serverVersion.value !== version,
);

const errorText = computed(() => props.error?.message ?? "");

onMounted(() => {
	api<{ version: string }>("meta", { detail: false })
		.then((meta) => {
			serverVersion.value = typeof meta.version === "string" ? meta.version : null;
			serverIsDead.value = false;
		})
		.catch(() => {
			serverIsDead.value = true;
		})
		.finally(() => {
			loaded.value = true;
		});
});

function reload(): void {
	location.reload();
}

definePageMetadata({
	title: t("error"),
	icon: "ph-warning ph-bold ph-lg",
});
</script>

<style lang="scss" scoped>
.error-page {
	inline-size: 100%;
	max-inline-size: 800px;
	margin-inline: auto;
	padding: 16px;
	box-sizing: border-box;

	._fullinfo {
		display: flex;
		flex-direction: column;
		align-items: center;
		padding: 48px 24px;
	}

	> ._fullinfo > .icon {
		font-size: 64px;
		opacity: 0.4;
		margin-block-end: 12px;
	}

	> ._fullinfo > .headline {
		margin: 0;
		font-size: 1.1em;
	}

	> ._fullinfo > .description {
		margin: 8px 0 0 0;
		max-inline-size: 32em;
		line-height: 1.7;
		color: var(--fgTransparentWeak);
		text-align: center;
	}

	> ._fullinfo > .actions {
		display: flex;
		flex-wrap: wrap;
		justify-content: center;
		gap: 8px;
		margin-block-start: 24px;
	}

	> ._fullinfo > .detail {
		margin-block-start: 16px;
		font-family: var(--monospace, monospace);
		font-size: 0.8em;
		color: var(--fgTransparentWeak);
		word-break: break-word;
	}
}
</style>
