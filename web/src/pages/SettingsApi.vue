<!-- Ported from firefish@b425f61:packages/client/src/pages/settings/api.vue (AGPL-3.0-only) -->
<template>
	<div class="_formRoot cf-settings-api">
		<CfInfo v-if="me == null" warn class="_formBlock">
			{{ t("signinRequired") }}
		</CfInfo>

		<template v-else>
			<section class="_formBlock">
				<h2 class="_sectionTitle">{{ t("token") }}</h2>

				<p class="account">
					当前账号：
					<strong>{{ acct }}</strong>
				</p>

				<template v-if="token !== null">
					<div class="tokenBox">
						<code class="token">{{ token }}</code>
						<CfButton inline @click="copy()">
							<i class="ph-copy ph-bold ph-lg"></i>
							{{ copied ? "已复制" : t("copy") }}
						</CfButton>
					</div>

					<CfInfo v-if="copyError !== null" warn class="_formBlock">
						{{ copyError }}
					</CfInfo>

					<CfInfo warn class="_formBlock">
						这个令牌等同于<strong>完整账号权限</strong>：拿到它的人可以以你的身份发帖、改资料、读私信。
						不要粘贴到聊天、截图、日志或任何第三方站点；一旦泄露，只能通过重新登录换取新令牌
						（本波没有 <code>i/apps</code> 端点，无法在页面上吊销或重新生成）。
					</CfInfo>

					<p class="hint">
						令牌保存在账号层的内存与 <code>account</code> 存储键里，由
						<code>@/api/client</code> 在为每个请求附加 <code>Authorization</code> 时读取；
						本页只展示，不写入任何地方。
					</p>
				</template>

				<CfInfo v-else class="_formBlock" icon="info">
					这个会话没有暴露令牌：当前登录状态由其它凭据恢复（例如 MiAuth 会话），
					账号对象里没有 <code>token</code> 字段。此时仍可正常使用 API 控制台 —
					请求会由客户端自动附加可用的凭据。
				</CfInfo>
			</section>

			<section class="_formBlock">
				<h2 class="_sectionTitle">API 控制台</h2>
				<p class="hint">
					在控制台里手写端点名与 JSON 参数，直接以当前账号身份调用
					<code>/api</code>；返回的 JSON 会原样显示。
				</p>
				<CfButton link to="/api-console">
					<i class="ph-terminal-window ph-bold ph-lg"></i>
					打开 API 控制台
				</CfButton>
			</section>

			<section class="_formBlock">
				<h2 class="_sectionTitle">与上游的差异</h2>
				<ul class="notes">
					<li>
						上游这一页是「生成访问令牌」按钮与「管理访问令牌」链接，分别依赖
						<code>miauth/gen-token</code> 与 <code>i/apps</code>；
						<code>i/apps</code> 在本项目里不存在，所以这里改为展示当前会话的原生令牌。
					</li>
					<li>
						上游只能在设置页看到令牌；Cinderfish 保留同样的限制：不从
						<code>localStorage</code> 直接读取，只读取账号层的 <code>me</code>。
					</li>
				</ul>
			</section>
		</template>
	</div>
</template>

<script setup lang="ts">
// Firefish's page generated a MiAuth token and linked to its token manager.
// `i/apps` does not exist here, so the page shows the native token the account
// layer already holds (read through `me`, never through localStorage).
import { computed, ref } from "vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import { accountAcct, me } from "@/account";
import { t } from "@/i18n";
import copyToClipboard from "@/scripts/copy-to-clipboard";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageSettingsApi" });

const copied = ref(false);
const copyError = ref<string | null>(null);

let copiedTimer: number | null = null;

const token = computed<string | null>(() => {
	const value = me.value?.token;
	return value != null && value !== "" ? value : null;
});

const acct = computed<string>(() =>
	me.value == null ? "" : accountAcct(me.value),
);

async function copy(): Promise<void> {
	const value = token.value;
	if (value == null) return;
	copyError.value = null;
	const ok = await copyToClipboard(value);
	if (!ok) {
		copyError.value = "复制失败：浏览器拒绝了剪贴板访问，请手动选中令牌复制。";
		return;
	}
	copied.value = true;
	if (copiedTimer != null) window.clearTimeout(copiedTimer);
	copiedTimer = window.setTimeout(() => {
		copied.value = false;
	}, 2000);
}

definePageMetadata({
	title: "API",
	icon: "ph-key ph-bold ph-lg",
});
</script>

<style lang="scss" scoped>
.cf-settings-api {
	.account {
		margin: 0 0 12px;
		color: var(--fgTransparentWeak);
		font-size: 0.9em;

		strong {
			color: var(--fg);
		}
	}

	.tokenBox {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 8px;
		padding: 10px 12px;
		border: solid 1px var(--divider);
		border-radius: var(--radius);
		background: var(--panel);
		margin-block-end: 12px;
	}

	.token {
		flex: 1 1 220px;
		min-inline-size: 0;
		font-family: "Fira code", "Fira Mono", Consolas, Menlo, Courier, monospace;
		overflow-wrap: anywhere;
		user-select: all;
	}

	.hint {
		margin: 8px 0 12px;
		color: var(--fgTransparentWeak);
		font-size: 0.9em;
	}

	.notes {
		margin: 0;
		padding-inline-start: 1.2em;
		color: var(--fgTransparentWeak);
		font-size: 0.9em;

		> li + li {
			margin-block-start: 6px;
		}

		code {
			color: var(--fg);
		}
	}
}
</style>
