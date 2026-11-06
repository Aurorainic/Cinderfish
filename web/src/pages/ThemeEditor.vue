<!-- Ported from firefish@b425f61:packages/client/src/pages/theme-editor.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :actions="headerActions" />
		</template>
		<MkSpacer :content-max="800" :margin-min="16" :margin-max="32">
			<div class="_formRoot cf-theme-editor">
				<CfInfo v-if="me == null" warn>
					主题编辑器需要登录后才能保存主题（主题会记在账号名下的本地存储里）。
				</CfInfo>

				<template v-else>
					<FormSection>
						<template #label>主题信息</template>

						<FormInput v-model="name" class="_formBlock">
							<template #label>{{ t("name") }}</template>
							<template #caption
								>保存时会据此生成 id（小写 slug），再追加时间戳。</template
							>
						</FormInput>

						<FormInput v-model="author" class="_formBlock">
							<template #label>{{ t("author") }}</template>
							<template #caption
								>留空则按空字符串保存，主题管理页会显示「未署名」。</template
							>
						</FormInput>

						<FormSelect
							:model-value="base"
							class="_formBlock"
							@update:model-value="onBase"
						>
							<template #label>{{ t("_theme.base") }}</template>
							<option value="light">{{ t("light") }}</option>
							<option value="dark">{{ t("dark") }}</option>
							<template #caption>
								基础主题决定没被覆盖的全部颜色（面板、按钮、输入框等）。
							</template>
						</FormSelect>
					</FormSection>

					<FormSection>
						<template #label>颜色</template>
						<p class="hint">
							只写想改的颜色；留空表示继承基础主题。左侧色块直接选颜色，右侧输入框接受主题语法（<code>#86b300</code>、<code>@accent</code>、<code>:darken&lt;10&lt;@accent</code>）。
						</p>

						<FormInput
							v-for="token in TOKENS"
							:key="token"
							class="_formBlock"
							:model-value="overrides[token] ?? ''"
							:placeholder="resolvedColors[token] ?? ''"
							@update:model-value="
								(value) => setToken(token, value)
							"
						>
							<template #label>
								{{ tokenLabel(token) }}
								<code class="tokenName">{{ token }}</code>
							</template>
							<template #prefix>
								<input
									class="swatchInput"
									type="color"
									:value="hexOf(resolvedColors[token])"
									:aria-label="tokenLabel(token)"
									@input="pickColor(token, $event)"
								/>
							</template>
							<template #caption>
								{{
									overrides[token] === undefined
										? `继承：${resolvedColors[token] ?? "—"}`
										: `当前：${overrides[token]}`
								}}
							</template>
						</FormInput>

						<CfButton
							:disabled="overrideCount === 0"
							@click="resetColors"
						>
							清除全部自定义颜色
						</CfButton>
					</FormSection>

					<FormSection>
						<template #label>预览</template>
						<CfInfo v-if="compileError !== null" class="_formBlock" warn>
							{{ compileError }}
						</CfInfo>
						<div class="preview" :style="previewVars">
							<div class="previewCard">
								<div class="previewHead">
									<span class="previewAvatar"></span>
									<div>
										<div class="previewName">
											{{ name.trim() === "" ? "未命名主题" : name }}
										</div>
										<div class="previewAuthor">
											{{ author.trim() === "" ? "未署名" : author }}
										</div>
									</div>
								</div>
								<p class="previewText">
									正文使用 <code>fg</code>，卡片背景使用
									<code>panel</code>，描边使用 <code>divider</code>。
								</p>
								<div class="previewDivider"></div>
								<div class="previewRow">
									<span class="previewAccent">强调色按钮</span>
									<span class="previewLink">链接文本</span>
									<span class="previewWarn">警告</span>
									<span class="previewError">错误</span>
								</div>
							</div>
						</div>
					</FormSection>

					<CfInfo v-if="errorMessage !== ''" class="_formBlock" warn>
						{{ errorMessage }}
					</CfInfo>
					<CfInfo v-if="savedTheme !== null" class="_formBlock">
						已保存「{{ savedTheme.name }}」（id：<code>{{ savedTheme.id }}</code>）并切换为当前主题。到
						<MkA to="/settings/theme/manage">主题管理</MkA>
						可以导出或卸载。
					</CfInfo>

					<div class="actions _formBlock">
						<CfButton
							primary
							:disabled="name.trim() === '' || compileError !== null"
							@click="save"
						>
							<i class="ph-check ph-bold ph-lg"></i>
							保存并应用
						</CfButton>
					</div>
				</template>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script lang="ts" setup>
// Firefish's theme editor, rebuilt around the eight palette tokens this port
// keeps in the UI (contract §5) instead of upstream's colour grids:
//   * no `tinycolor2`, `uuid` or `punycode` — the id is a slug of the name plus
//     a base-36 timestamp, and the author field is plain text;
//   * no `applyTheme()`-per-keystroke and no `/preview` window: upstream wrote
//     the draft straight into the live CSS variables, which would wipe the
//     user's current theme if the page were abandoned. Here the draft is
//     compiled into a local preview panel, and only "save" installs + applies;
//   * no code pane / description folder: `FormTextarea` JSON editing lives on
//     the install page, and `desc` stays optional;
//   * every unset token is omitted from `props`, so `compileTheme()` layers the
//     selected base theme underneath it.
import { computed, ref, watch } from "vue";
import { me } from "@/account";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import FormInput from "@/components/form/input.vue";
import FormSection from "@/components/form/section.vue";
import FormSelect from "@/components/form/select.vue";
import { useTheme } from "@/composables/use-theme";
import { t } from "@/i18n";
import { definePageMetadata } from "@/scripts/page-metadata";
import { success } from "@/os";
import {
	compileTheme,
	installTheme,
	type CompiledTheme,
	type Theme,
	type ThemeMode,
} from "@/themes";

defineOptions({ name: "CfPageThemeEditor" });

const TOKENS = [
	"accent",
	"bg",
	"fg",
	"panel",
	"divider",
	"link",
	"warn",
	"error",
] as const;
type Token = (typeof TOKENS)[number];

const TOKEN_LABELS: Record<Token, () => string> = {
	accent: () => t("_theme.keys.accent"),
	bg: () => t("_theme.keys.bg"),
	fg: () => t("_theme.keys.fg"),
	panel: () => t("_theme.keys.panel"),
	divider: () => t("_theme.keys.divider"),
	link: () => t("_theme.keys.link"),
	// No locale key for these two in zh-CN.json; literal per contract §1.6.
	warn: () => "警告色",
	error: () => "错误色",
};

const { setTheme } = useTheme();

const name = ref("");
const author = ref(me.value != null ? `@${me.value.username}` : "");
const base = ref<ThemeMode>("light");
const overrides = ref<Record<string, string>>({});
const errorMessage = ref("");
const savedTheme = ref<Theme | null>(null);

watch(me, (user) => {
	if (user != null && author.value === "") author.value = `@${user.username}`;
});

const draftTheme = computed<Theme>(() => ({
	id: "preview",
	name: name.value.trim() === "" ? "未命名主题" : name.value.trim(),
	author: author.value.trim(),
	base: base.value,
	props: { ...overrides.value },
}));

/**
 * The draft compiled on top of its base theme. A bad reference (`@nope`) makes
 * `compileTheme` throw; the last usable base palette is kept for the preview so
 * the panel does not go blank while the user is mid-edit.
 */
const compiled = computed<CompiledTheme | null>(() => {
	try {
		return compileTheme(draftTheme.value);
	} catch {
		return null;
	}
});

const baseCompiled = computed<CompiledTheme>(() =>
	compileTheme({
		id: "base",
		name: "base",
		author: "",
		base: base.value,
		props: {},
	}),
);

const resolvedColors = computed<CompiledTheme>(
	() => compiled.value ?? baseCompiled.value,
);

const compileError = computed<string | null>(() =>
	compiled.value === null
		? "有颜色无法解析（例如引用了不存在的属性），修正前不能保存。"
		: null,
);

const overrideCount = computed(() => Object.keys(overrides.value).length);

const previewVars = computed<Record<string, string>>(() => {
	const colors = resolvedColors.value;
	return {
		"--pv-bg": colors.bg ?? "",
		"--pv-fg": colors.fg ?? "",
		"--pv-panel": colors.panel ?? "",
		"--pv-accent": colors.accent ?? "",
		"--pv-divider": colors.divider ?? "",
		"--pv-link": colors.link ?? "",
		"--pv-warn": colors.warn ?? "",
		"--pv-error": colors.error ?? "",
	};
});

function tokenLabel(token: Token): string {
	return TOKEN_LABELS[token]();
}

/** `compileTheme` emits `rgb(r, g, b)` / `rgba(...)`; `<input type=color>` needs hex. */
function hexOf(color: string | undefined): string {
	if (color == null) return "#000000";
	const match = /^rgba?\(\s*(\d+)[,\s]+(\d+)[,\s]+(\d+)/.exec(color);
	if (match == null) {
		return /^#[0-9a-f]{6}$/i.test(color) ? color : "#000000";
	}
	return `#${[match[1], match[2], match[3]]
		.map((part) => Number(part).toString(16).padStart(2, "0"))
		.join("")}`;
}

function setToken(token: Token, value: string | number | null): void {
	const raw = value == null ? "" : String(value).trim();
	if (raw === "") delete overrides.value[token];
	else overrides.value[token] = raw;
	savedTheme.value = null;
	errorMessage.value = "";
}

function pickColor(token: Token, event: Event): void {
	const input = event.target as HTMLInputElement | null;
	if (input == null) return;
	setToken(token, input.value);
}

function resetColors(): void {
	overrides.value = {};
	savedTheme.value = null;
	errorMessage.value = "";
}

function onBase(value: string | null): void {
	if (value === "light" || value === "dark") {
		base.value = value;
		savedTheme.value = null;
	}
}

function slugify(value: string): string {
	const slug = value
		.toLowerCase()
		.replace(/[^a-z0-9]+/g, "-")
		.replace(/^-+|-+$/g, "")
		.slice(0, 40)
		.replace(/-+$/g, "");
	return slug === "" ? "theme" : slug;
}

function save(): void {
	errorMessage.value = "";
	savedTheme.value = null;

	if (me.value == null) {
		errorMessage.value = "登录后才能保存主题。";
		return;
	}

	const trimmedName = name.value.trim();
	if (trimmedName === "") {
		errorMessage.value = "请填写主题名称。";
		return;
	}
	if (compiled.value === null) {
		errorMessage.value = "主题里有无法解析的颜色，先修正后再保存。";
		return;
	}

	const theme: Theme = {
		id: `${slugify(trimmedName)}-${Date.now().toString(36)}`,
		name: trimmedName,
		author: author.value.trim(),
		base: base.value,
		props: { ...overrides.value },
	};

	try {
		installTheme(theme);
	} catch (error) {
		errorMessage.value = `保存失败：${error instanceof Error ? error.message : String(error)}`;
		return;
	}

	// `setTheme(id)` writes the id into the slot of the theme's own mode and
	// re-applies the CSS variables.
	setTheme(theme.id);
	savedTheme.value = theme;
	success();
}

const headerActions = computed(() =>
	me.value == null
		? []
		: [
				{
					icon: "ph-check ph-bold ph-lg",
					text: "保存并应用",
					handler: () => save(),
				},
			],
);

definePageMetadata({ title: t("themeEditor"), icon: "ph-palette ph-bold ph-lg" });
</script>

<style lang="scss" scoped>
.cf-theme-editor {
	.hint {
		margin: 0 0 12px;
		color: var(--fgTransparentWeak);
		font-size: 0.85em;

		> code {
			font-size: 0.95em;
		}
	}

	.tokenName {
		margin-inline-start: 4px;
		color: var(--fgTransparentWeak);
		font-size: 0.85em;
	}

	.swatchInput {
		inline-size: 22px;
		block-size: 22px;
		padding: 0;
		border: solid 1px var(--divider);
		border-radius: 4px;
		background: none;
		cursor: pointer;

		&::-webkit-color-swatch-wrapper {
			padding: 1px;
		}

		&::-webkit-color-swatch {
			border: none;
			border-radius: 2px;
		}
	}

	.preview {
		padding: 16px;
		background: var(--pv-bg, var(--bg));
		border: solid 1px var(--pv-divider, var(--divider));
		border-radius: var(--radius);

		> .previewCard {
			padding: 14px;
			background: var(--pv-panel, var(--panel));
			border: solid 1px var(--pv-divider, var(--divider));
			border-radius: 10px;
			color: var(--pv-fg, var(--fg));

			> .previewHead {
				display: flex;
				align-items: center;
				gap: 10px;

				> .previewAvatar {
					inline-size: 32px;
					block-size: 32px;
					flex-shrink: 0;
					background: var(--pv-accent, var(--accent));
					border-radius: 50%;
				}

				> div > .previewName {
					font-weight: bold;
				}

				> div > .previewAuthor {
					font-size: 0.85em;
					opacity: 0.75;
				}
			}

			> .previewText {
				margin: 12px 0 0;
				line-height: 1.5;
			}

			> .previewDivider {
				margin: 12px 0;
				block-size: 0;
				border-block-start: solid 1px var(--pv-divider, var(--divider));
			}

			> .previewRow {
				display: flex;
				flex-wrap: wrap;
				align-items: center;
				gap: 10px;

				> .previewAccent {
					padding: 5px 12px;
					background: var(--pv-accent, var(--accent));
					border-radius: 999px;
					color: var(--pv-bg, var(--bg));
					font-size: 0.85em;
					font-weight: bold;
				}

				> .previewLink {
					color: var(--pv-link, var(--link));
					font-size: 0.85em;
					text-decoration: underline;
				}

				> .previewWarn {
					padding: 2px 8px;
					border: solid 1px var(--pv-warn, var(--warn));
					border-radius: 999px;
					color: var(--pv-warn, var(--warn));
					font-size: 0.8em;
				}

				> .previewError {
					padding: 2px 8px;
					border: solid 1px var(--pv-error, var(--error));
					border-radius: 999px;
					color: var(--pv-error, var(--error));
					font-size: 0.8em;
				}
			}
		}
	}

	.actions {
		display: flex;
		justify-content: flex-end;
	}
}
</style>
