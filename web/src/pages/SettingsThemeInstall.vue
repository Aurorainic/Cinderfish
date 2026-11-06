<!-- Ported from firefish@b425f61:packages/client/src/pages/settings/theme.install.vue (AGPL-3.0-only) -->
<template>
	<div class="_formRoot cf-settings-theme-install">
		<FormTextarea
			v-model="code"
			class="_formBlock"
			code
			tall
			:placeholder="placeholder"
		>
			<template #label>{{ t("_theme.code") }}</template>
			<template #caption>
				支持 JSON5（注释、未加引号的键、单引号、尾逗号），与 Firefish 主题文件一致。解析在本机完成，不会上传。
			</template>
		</FormTextarea>

		<div class="actions _formBlock">
			<CfButton primary :disabled="code.trim() === ''" @click="install">
				<i class="ph-check ph-bold ph-lg"></i>
				{{ t("install") }}
			</CfButton>
			<CfButton :disabled="code === ''" @click="reset">
				清空
			</CfButton>
		</div>

		<CfInfo v-if="errorMessage !== ''" class="_formBlock" warn>
			{{ errorMessage }}
		</CfInfo>

		<CfInfo v-if="installedTheme != null" class="_formBlock">
			{{ t("_theme.installed", { name: installedTheme.name }) }}（id：<code>{{
				installedTheme.id
			}}</code>）。到
			<MkA to="/settings/theme/manage">主题管理</MkA>
			可以在两种模式之间切换、导出或卸载它。
		</CfInfo>

		<FormSection>
			<template #label>安装规则</template>
			<CfInfo card>
				<div class="rulesWrap">
					<ul class="rules">
						<li><code>id</code> 必填，且不能与内置主题或已安装主题重复。</li>
						<li><code>name</code> 必填，<code>author</code> 缺省时按空字符串保存。</li>
						<li>
							需要有 <code>base: "light" | "dark"</code>（或上游写法
							<code>kind</code>）；未覆盖的颜色会从该基础主题继承。
						</li>
						<li>
							已安装 {{ installedCount }} 个主题，数据只存在本浏览器的
							localStorage。
						</li>
					</ul>
				</div>
			</CfInfo>
			<CfInfo class="_formBlock" icon="info">
				上游的「预览」按钮会把主题立刻套到整个界面上；这里没有页面预览窗口，直接应用会改动正在浏览的界面，所以只提供安装，装好后在主题页切换。
			</CfInfo>
		</FormSection>
	</div>
</template>

<script lang="ts" setup>
// Firefish's theme installer: a textarea plus preview/install. The parser is
// `parseJson5` from the theme engine (no `json5` package), and the install goes
// straight to localStorage through `installTheme` — there is no network call.
//
// Differences from upstream:
//   * `preview()` is gone: it called `applyTheme()` on the live document with
//     an uninstalled theme and there is no preview window to scope it to;
//   * the theme is normalised before validation (missing `author` → "") because
//     Firefish's own `d-u0` / `l-u0` palettes ship without one.
import { computed, ref } from "vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import FormSection from "@/components/form/section.vue";
import FormTextarea from "@/components/form/textarea.vue";
import { t } from "@/i18n";
import { definePageMetadata } from "@/scripts/page-metadata";
import {
	findInstalledTheme,
	findTheme,
	installTheme,
	listInstalledThemes,
	parseJson5,
	validateTheme,
	type Theme,
} from "@/themes";

defineOptions({ name: "CfPageSettingsThemeInstall" });

const placeholder = `{
  id: "my-theme",
  name: "My Theme",
  author: "@you",
  base: "dark",
  props: {
    accent: "#86b300",
    bg: "#191919",
  },
}`;

const code = ref("");
const errorMessage = ref("");
const installedTheme = ref<Theme | null>(null);
const installedCount = ref(listInstalledThemes().length);

const trimmedCode = computed(() => code.value.trim());

function errorText(error: unknown): string {
	return error instanceof Error ? error.message : String(error);
}

/**
 * Parses, normalises and validates the pasted document. Returns the theme, or
 * null after writing a user-facing message into `errorMessage`.
 */
function readTheme(source: string): Theme | null {
	let parsed: unknown;
	try {
		parsed = parseJson5(source);
	} catch (error) {
		errorMessage.value = `主题代码解析失败：${errorText(error)}`;
		return null;
	}

	if (parsed == null || typeof parsed !== "object" || Array.isArray(parsed)) {
		errorMessage.value = "主题代码的最外层必须是一个对象。";
		return null;
	}

	// Copy before normalising so the parsed value is not mutated in place.
	const candidate: Record<string, unknown> = { ...parsed };

	if (typeof candidate.id !== "string" || candidate.id.trim() === "") {
		errorMessage.value = "主题缺少 id 字段（字符串，且不能为空）。";
		return null;
	}
	candidate.id = candidate.id.trim();

	if (typeof candidate.name !== "string" || candidate.name.trim() === "") {
		errorMessage.value = "主题缺少 name 字段（字符串，且不能为空）。";
		return null;
	}

	if (candidate.author === undefined || candidate.author === null) {
		candidate.author = "";
	} else if (typeof candidate.author !== "string") {
		errorMessage.value = "author 必须是字符串。";
		return null;
	}

	if (candidate.base === undefined && candidate.kind === undefined) {
		errorMessage.value =
			'主题缺少 base 字段：需要 base: "light" 或 base: "dark"（旧文件用 kind）。';
		return null;
	}

	if (!validateTheme(candidate)) {
		errorMessage.value =
			"主题结构不合法：需要 id、name、author、base（或 kind）以及字符串值组成的 props。";
		return null;
	}

	const theme = candidate as unknown as Theme;
	if (findTheme(theme.id) != null) {
		errorMessage.value = findInstalledTheme(theme.id)
			? `已经安装过 id 为「${theme.id}」的主题，先到主题管理卸载后再装。`
			: `id「${theme.id}」与内置主题冲突，换一个 id 再试。`;
		return null;
	}

	return theme;
}

function install(): void {
	errorMessage.value = "";
	installedTheme.value = null;
	if (trimmedCode.value === "") {
		errorMessage.value = "先粘贴主题代码。";
		return;
	}

	const theme = readTheme(trimmedCode.value);
	if (theme == null) return;

	try {
		installTheme(theme);
	} catch (error) {
		errorMessage.value = `安装失败：${errorText(error)}`;
		return;
	}

	installedTheme.value = theme;
	installedCount.value = listInstalledThemes().length;
	code.value = "";
}

function reset(): void {
	code.value = "";
	errorMessage.value = "";
	installedTheme.value = null;
}

definePageMetadata({
	title: t("_theme.install"),
	icon: "ph-download-simple ph-bold ph-lg",
});
</script>

<style lang="scss" scoped>
.cf-settings-theme-install {
	.actions {
		display: flex;
		flex-wrap: wrap;
		gap: var(--margin);
		align-items: center;
	}

	.rules {
		margin: 6px 0 0;
		padding-inline-start: 1.4em;

		> li {
			margin-block-start: 2px;
		}
	}
}
</style>
