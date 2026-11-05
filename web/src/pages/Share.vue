<!-- Ported from firefish@b425f61:packages/client/src/pages/share.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :actions="headerActions" />
		</template>
		<MkSpacer :content-max="800">
			<div class="share _formRoot">
				<CfInfo v-if="!hasContent" class="_formBlock" icon="share-network">
					没有可分享的内容：本页需要 <code>title</code>、<code>text</code> 或 <code>url</code> 查询参数（例如
					<code>/share?title=标题&amp;url=https://example.com</code>）。
				</CfInfo>

				<template v-else>
					<div class="_formBlock">
						<CfKeyValue v-if="title !== ''">
							<template #key>{{ t("title") }}</template>
							<template #value>{{ title }}</template>
						</CfKeyValue>
						<CfKeyValue v-if="shareUrl !== ''">
							<template #key>链接</template>
							<template #value>{{ shareUrl }}</template>
						</CfKeyValue>
					</div>

					<div class="_formBlock">
						<CfTextarea
							:model-value="shareText"
							readonly
							tall
							code
						>
							<template #label>{{ t("text") }}</template>
						</CfTextarea>
					</div>

					<div class="_formBlock buttons">
						<CfButton
							v-if="canWebShare"
							primary
							:disabled="shareText === ''"
							@click="share()"
						>
							<i class="ph-share-network ph-bold ph-lg"></i>
							{{ t("share") }}
						</CfButton>
						<CfButton
							v-if="shareUrl !== ''"
							@click="copy(shareUrl)"
						>
							<i class="ph-link ph-bold ph-lg"></i>
							{{ t("copyLink") }}
						</CfButton>
						<CfButton
							:disabled="shareText === ''"
							@click="copy(shareText)"
						>
							<i class="ph-clipboard-text ph-bold ph-lg"></i>
							{{ t("copyContent") }}
						</CfButton>
					</div>

					<CfInfo v-if="!canWebShare" class="_formBlock" icon="info">
						当前浏览器不支持 Web Share API（或页面不在安全上下文中），已降级为复制链接与复制全文。
					</CfInfo>

					<CfInfo
						v-if="status === 'copied'"
						class="_formBlock"
						icon="check"
					>
						已复制到剪贴板。
					</CfInfo>
					<CfInfo
						v-else-if="status === 'failed'"
						class="_formBlock"
						warn
					>
						复制失败，请手动选择文本复制。
					</CfInfo>
				</template>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script lang="ts" setup>
// SPECIFICATION: https://misskey-hub.net/docs/features/share-form.html
//
// Firefish's share.vue is a share-form target: it parses the query string and
// immediately opens MkPostForm with the composed text. Cinderfish has no
// composer in this page's scope (the composer is a separate wave-3 workstream
// reachable through os.post, contract §3.1), so the page is ported as a share
// sheet instead: Web Share API first, copy-link / copy-text as the fallback
// (contract §2 degradation for the unavailable composer wiring).
//
// Query parameters understood: title, text, url. Firefish's visibility,
// localOnly, replyId/renoteId, fileIds and visibleUserIds only matter when a
// post form is opened, so they are ignored here and documented in the report.
import { computed, ref } from "vue";
import { useRoute } from "vue-router";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfKeyValue from "@/components/ui/CfKeyValue.vue";
import CfTextarea from "@/components/ui/CfTextarea.vue";
import copyToClipboard from "@/scripts/copy-to-clipboard";
import { t } from "@/i18n";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageShare" });

const route = useRoute();

function queryValue(key: string): string {
	const raw = route.query[key];
	const value = Array.isArray(raw) ? raw[0] : raw;
	return typeof value === "string" ? value : "";
}

const title = computed(() => queryValue("title"));
const body = computed(() => queryValue("text"));
const shareUrl = computed(() => queryValue("url"));

const shareText = computed(() => {
	let text = "";
	if (title.value !== "") text += `[ ${title.value} ]\n`;
	// Google News appends "<title>.\n" to the body; strip the duplicate.
	const prefix = `${title.value}.\n`;
	if (body.value !== "" && title.value !== "" && body.value.startsWith(prefix)) {
		text += body.value.slice(prefix.length);
	} else if (body.value !== "" && body.value !== title.value) {
		text += `${body.value}\n`;
	}
	if (shareUrl.value !== "") text += shareUrl.value;
	return text.trim();
});

const hasContent = computed(
	() => title.value !== "" || body.value !== "" || shareUrl.value !== "",
);

const canWebShare = computed(
	() =>
		typeof navigator !== "undefined" &&
		typeof navigator.share === "function",
);

const status = ref<"idle" | "copied" | "failed">("idle");
let statusTimer: number | null = null;

function setStatus(next: "copied" | "failed"): void {
	status.value = next;
	if (statusTimer != null) window.clearTimeout(statusTimer);
	statusTimer = window.setTimeout(() => {
		status.value = "idle";
	}, 3000);
}

async function share(): Promise<void> {
	try {
		await navigator.share({
			title: title.value !== "" ? title.value : undefined,
			text: shareText.value,
		});
	} catch (err) {
		// The user dismissing the sheet is not an error.
		if (err instanceof Error && err.name === "AbortError") return;
		setStatus("failed");
	}
}

async function copy(value: string): Promise<void> {
	if (value === "") return;
	const ok = await copyToClipboard(value);
	setStatus(ok ? "copied" : "failed");
}

const headerActions = computed(() => []);

definePageMetadata({
	title: t("share"),
	icon: "ph-share-network ph-bold ph-lg",
});
</script>

<style lang="scss" scoped>
.share {
	padding: 16px;
}

.buttons {
	display: flex;
	flex-wrap: wrap;
	gap: 8px;
}
</style>
