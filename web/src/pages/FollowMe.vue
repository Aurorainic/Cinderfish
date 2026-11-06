<!-- SPDX-License-Identifier: AGPL-3.0-only -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :actions="headerActions" />
		</template>
		<MkSpacer :content-max="600">
			<div class="follow-me">
				<CfInfo v-if="me == null" warn icon="sign-in">
					本页需要登录后才能生成关注链接。<CfA to="/">返回首页登录</CfA>
				</CfInfo>

				<template v-else>
					<div class="intro _panel">
						<h2 class="title">让别人关注你</h2>
						<p class="description">
							把下面的链接发给别人：在本实例打开它，会直接跳到关注确认。其它实例的用户无法打开这个链接，
							请在对方实例搜索你的完整账号 <code>{{ myAcct }}</code> 并关注。
						</p>
					</div>

					<div class="field _panel">
						<div class="label">我的账号</div>
						<div class="row">
							<code class="value">{{ myAcct }}</code>
							<CfButton mini @click="copy(myAcct)">
								<i class="ph-clipboard-text ph-bold"></i>
								<span>{{ t("copyContent") }}</span>
							</CfButton>
						</div>
					</div>

					<div class="field _panel">
						<div class="label">关注链接</div>
						<div class="row">
							<code class="value link">{{ followLink }}</code>
							<CfButton mini primary @click="copy(followLink)">
								<i class="ph-link ph-bold"></i>
								<span>{{ t("copyLink") }}</span>
							</CfButton>
						</div>
					</div>

					<CfInfo v-if="status === 'copied'" icon="check">
						已复制到剪贴板。
					</CfInfo>
					<CfInfo v-else-if="status === 'failed'" warn icon="warning">
						复制失败，请手动选择文本复制。
					</CfInfo>

					<p class="hint">
						这个链接依赖本实例的 <code>/authorize-follow</code> 页面，只对本站账号生效。
					</p>
				</template>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script lang="ts" setup>
// /follow-me. Firefish has no such page in the pinned snapshot, so this is
// written from the components around it: the acct is the account layer's `me`,
// copying uses the shared clipboard helper, and the link is the same
// /authorize-follow URL that Follow.vue parses. The page never reads a token and
// never builds the acct by hand from storage.
import { computed, onUnmounted, ref } from "vue";
import CfA from "@/components/ui/CfA.vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import { me } from "@/account";
import { host } from "@/config";
import { t } from "@/i18n";
import copyToClipboard from "@/scripts/copy-to-clipboard";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageFollowMe" });

const headerActions = computed(() => []);

const status = ref<"idle" | "copied" | "failed">("idle");
let statusTimer: number | null = null;

const myAcct = computed(() =>
	me.value == null ? "" : `@${me.value.username}@${host}`,
);

const origin = computed(() =>
	typeof location === "undefined" ? "" : location.origin,
);

const followLink = computed(
	() => `${origin.value}/authorize-follow?acct=${myAcct.value}`,
);

function setStatus(next: "copied" | "failed"): void {
	status.value = next;
	if (statusTimer != null) window.clearTimeout(statusTimer);
	statusTimer = window.setTimeout(() => {
		status.value = "idle";
	}, 3000);
}

async function copy(value: string): Promise<void> {
	if (value === "") return;
	const ok = await copyToClipboard(value);
	setStatus(ok ? "copied" : "failed");
}

onUnmounted(() => {
	if (statusTimer != null) window.clearTimeout(statusTimer);
});

definePageMetadata({
	title: "关注我",
	icon: "ph-user-plus ph-bold ph-lg",
});
</script>

<style lang="scss" scoped>
.follow-me {
	display: flex;
	flex-direction: column;
	gap: 16px;
	padding-block: 8px;
}

.intro {
	padding: 20px 24px;

	> .title {
		margin: 0 0 8px 0;
		font-size: 1.1em;
	}

	> .description {
		margin: 0;
		line-height: 1.7;
		font-size: 0.92em;
		color: var(--fgTransparentWeak);
	}
}

.field {
	padding: 16px 20px;

	> .label {
		margin-block-end: 8px;
		font-size: 0.85em;
		color: var(--fgTransparentWeak);
	}

	> .row {
		display: flex;
		align-items: center;
		gap: 12px;

		> .value {
			flex: 1;
			min-inline-size: 0;
			overflow-wrap: anywhere;
			font-family: var(--monospace, monospace);
			font-size: 0.9em;
		}

		> .value.link {
			color: var(--accent);
		}
	}
}

.hint {
	margin: 0;
	font-size: 0.85em;
	color: var(--fgTransparentWeak);
}
</style>
