<!-- Ported from firefish@b425f61:packages/client/src/pages/settings/theme.vue (AGPL-3.0-only) -->
<template>
	<div class="_formRoot cf-settings-theme">
		<FormSection>
			<template #label>{{ t("theme") }}</template>
			<CfRadios
				class="_formBlock"
				:model-value="mode"
				@update:model-value="onMode"
			>
				<CfRadio value="light"
					><i class="ph-sun ph-bold ph-lg"></i> {{ t("light") }}</CfRadio
				>
				<CfRadio value="dark"
					><i class="ph-moon ph-bold ph-lg"></i> {{ t("dark") }}</CfRadio
				>
				<CfRadio value="auto">{{ t("auto") }}</CfRadio>
			</CfRadios>
			<p class="modeHint">{{ modeHint }}</p>
		</FormSection>

		<div class="selects _formBlock">
			<FormSelect v-model="lightThemeId" large class="select">
				<template #label>{{ t("themeForLightMode") }}</template>
				<template #prefix
					><i class="ph-sun ph-bold ph-lg"></i
				></template>
				<optgroup
					v-if="installedLightThemes.length > 0"
					:label="t('_theme.installedThemes')"
				>
					<option
						v-for="item in installedLightThemes"
						:key="item.id"
						:value="item.id"
					>
						{{ item.name }}
					</option>
				</optgroup>
				<optgroup :label="t('_theme.builtinThemes')">
					<option
						v-for="item in builtinLightThemes"
						:key="item.id"
						:value="item.id"
					>
						{{ item.name }}
					</option>
				</optgroup>
			</FormSelect>

			<FormSelect v-model="darkThemeId" large class="select">
				<template #label>{{ t("themeForDarkMode") }}</template>
				<template #prefix
					><i class="ph-moon ph-bold ph-lg"></i
				></template>
				<optgroup
					v-if="installedDarkThemes.length > 0"
					:label="t('_theme.installedThemes')"
				>
					<option
						v-for="item in installedDarkThemes"
						:key="item.id"
						:value="item.id"
					>
						{{ item.name }}
					</option>
				</optgroup>
				<optgroup :label="t('_theme.builtinThemes')">
					<option
						v-for="item in builtinDarkThemes"
						:key="item.id"
						:value="item.id"
					>
						{{ item.name }}
					</option>
				</optgroup>
			</FormSelect>
		</div>

		<div class="active _formBlock">
			<span class="activeSwatch" :style="activeSwatchStyle"></span>
			<div class="activeMeta">
				<div class="activeName">{{ theme.name }}</div>
				<div class="activeAuthor">
					{{ theme.author !== "" ? theme.author : "未署名" }}
				</div>
				<div class="activeCaption">
					当前生效：{{ themeMode(theme) === "dark" ? t("dark") : t("light") }}
					<span class="activeId">{{ theme.id }}</span>
				</div>
			</div>
		</div>

		<FormSection>
			<template #label>主题管理</template>
			<div class="_formLinks">
				<MkA class="linkRow" to="/settings/theme/manage">
					<i class="ph-folder-notch-open ph-bold ph-lg"></i>
					{{ t("_theme.manage") }}
					<span class="linkSuffix">{{ installedThemes.length }}</span>
				</MkA>
				<MkA class="linkRow" to="/settings/theme/install">
					<i class="ph-download-simple ph-bold ph-lg"></i>
					{{ t("_theme.install") }}
				</MkA>
				<MkA class="linkRow" to="/theme-editor">
					<i class="ph-paint-brush-broad ph-bold ph-lg"></i>
					{{ t("_theme.make") }}
				</MkA>
			</div>
		</FormSection>

		<FormSection>
			<template #label>未提供的主题能力</template>
			<CfInfo card>
				<div class="missing">
					主题只存在于当前浏览器：后端没有实例级默认主题、壁纸或自定义主题商店接口，所以下面这几项没有做成控件——
					<ul class="missingList">
						<li>
							实例默认主题（上游的 <code>defaultLightTheme</code> /
							<code>defaultDarkTheme</code>）
						</li>
						<li>
							壁纸（上游把地址写进 <code>localStorage.wallpaper</code>，本项目的布局不读这个键）
						</li>
						<li>「将深色模式与设备同步」（没有对应的设置项）</li>
						<li>
							主题商店链接（<code>assets.misskey.io</code> 是 Misskey 的列表，不是本实例的能力）
						</li>
					</ul>
				</div>
			</CfInfo>
			<CfInfo class="_formBlock">
				已安装的主题只保存在浏览器本地存储（<code>cinderfish.installedThemes</code>），不随账号同步。
			</CfInfo>
		</FormSection>
	</div>
</template>

<script lang="ts" setup>
// Firefish's theme settings page, reduced to the parts this instance can
// actually persist:
//   * light / dark / auto was a device-level `darkMode` flag plus a device
//     sync switch upstream; here it is the `useTheme()` mode, which also owns
//     the light/dark palette slots;
//   * the two palette pickers keep upstream's option groups (installed themes
//     of that mode first, then built-ins) and write through `setThemeForMode`;
//   * instance default themes, wallpaper, the misskey.io theme store and device
//     sync have no backend or consumer here, so they are documented in the
//     "unsupported" section instead of being rendered as dead controls.
import { computed, onActivated, onMounted, ref } from "vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfRadio from "@/components/ui/CfRadio.vue";
import CfRadios from "@/components/ui/CfRadios.vue";
import type { RadioValue } from "@/components/ui/radios-key";
import FormSection from "@/components/form/section.vue";
import FormSelect from "@/components/form/select.vue";
import { useTheme, type ThemeMode } from "@/composables/use-theme";
import { t } from "@/i18n";
import { definePageMetadata } from "@/scripts/page-metadata";
import {
	builtinThemes,
	compileTheme,
	defaultThemeIds,
	listInstalledThemes,
	themeMode,
	type Theme,
} from "@/themes";

defineOptions({ name: "CfPageSettingsTheme" });

const { mode, resolvedMode, selectedThemeIds, theme, setMode, setThemeForMode } =
	useTheme();

// `listInstalledThemes()` reads a module-level cache; refresh it when the page
// is re-entered so an install/uninstall in a sibling page shows up here.
const installedThemes = ref<Theme[]>(listInstalledThemes());

const installedLightThemes = computed(() =>
	installedThemes.value.filter((item) => themeMode(item) === "light"),
);
const installedDarkThemes = computed(() =>
	installedThemes.value.filter((item) => themeMode(item) === "dark"),
);
const builtinLightThemes = computed(() =>
	builtinThemes.filter((item) => themeMode(item) === "light"),
);
const builtinDarkThemes = computed(() =>
	builtinThemes.filter((item) => themeMode(item) === "dark"),
);

// A null slot means "that mode's default theme", which `activeTheme` already
// falls back to; the pickers show the resolved id instead of an empty value.
const lightThemeId = computed<string | null>({
	get: () => selectedThemeIds.light.value ?? defaultThemeIds.light,
	set: (id) => setThemeForMode("light", id),
});
const darkThemeId = computed<string | null>({
	get: () => selectedThemeIds.dark.value ?? defaultThemeIds.dark,
	set: (id) => setThemeForMode("dark", id),
});

const modeHint = computed(() => {
	if (mode.value === "auto") {
		return `跟随系统：当前解析为${resolvedMode.value === "dark" ? t("dark") : t("light")}模式。`;
	}
	return mode.value === "dark" ? "固定使用深色模式。" : "固定使用浅色模式。";
});

const activeSwatchStyle = computed<Record<string, string>>(() => {
	let accent = "var(--accent)";
	let bg = "var(--bg)";
	let divider = "var(--divider)";
	try {
		const compiled = compileTheme(theme.value);
		accent = compiled.accent ?? accent;
		bg = compiled.bg ?? bg;
		divider = compiled.divider ?? divider;
	} catch {
		// A palette with a broken reference keeps the live CSS variables.
	}
	return {
		background: `linear-gradient(135deg, ${accent} 0 50%, ${bg} 50% 100%)`,
		borderColor: divider,
	};
});

function onMode(value: RadioValue): void {
	if (value === "light" || value === "dark" || value === "auto") {
		setMode(value as ThemeMode);
	}
}

function refreshInstalled(): void {
	installedThemes.value = listInstalledThemes();
}

onMounted(refreshInstalled);
onActivated(refreshInstalled);

definePageMetadata({ title: t("theme"), icon: "ph-palette ph-bold ph-lg" });
</script>

<style lang="scss" scoped>
.cf-settings-theme {
	.modeHint {
		margin: 0;
		color: var(--fgTransparentWeak);
		font-size: 0.85em;
	}

	.selects {
		display: flex;
		flex-wrap: wrap;
		gap: 1.5em var(--margin);

		> .select {
			flex: 1;
			min-inline-size: 280px;
		}
	}

	.active {
		display: flex;
		align-items: center;
		gap: 14px;
		padding: 14px 16px;
		background: var(--panel);
		border: solid 1px var(--divider);
		border-radius: var(--radius);

		> .activeSwatch {
			flex-shrink: 0;
			inline-size: 42px;
			block-size: 42px;
			border: solid 1px var(--divider);
			border-radius: 8px;
		}

		> .activeMeta {
			min-inline-size: 0;

			> .activeName {
				font-weight: bold;
				color: var(--fg);
			}

			> .activeAuthor {
				color: var(--fg);
				font-size: 0.9em;
				opacity: 0.8;
			}

			> .activeCaption {
				margin-block-start: 2px;
				color: var(--fgTransparentWeak);
				font-size: 0.8em;
			}

			> .activeCaption > .activeId {
				margin-inline-start: 6px;
				font-family: var(--monospace, monospace);
			}
		}
	}

	.linkRow {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 10px 14px;
		border: solid 1px var(--divider);
		border-radius: var(--radius);
		color: var(--fg);
		text-decoration: none;

		&:hover {
			background: var(--accentedBg);
		}

		> .linkSuffix {
			margin-inline-start: auto;
			color: var(--fgTransparentWeak);
			font-variant-numeric: tabular-nums;
		}
	}

	.missingList {
		margin: 6px 0 0;
		padding-inline-start: 1.4em;

		> li {
			margin-block-start: 2px;
		}
	}

	.missing {
		min-inline-size: 0;
	}
}
</style>
