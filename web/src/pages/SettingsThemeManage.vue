<!-- Ported from firefish@b425f61:packages/client/src/pages/settings/theme.manage.vue (AGPL-3.0-only) -->
<template>
	<div class="_formRoot cf-settings-theme-manage">
		<CfInfo v-if="themes.length === 0" card>
			还没有安装任何主题。到
			<MkA to="/settings/theme/install">安装主题</MkA>
			粘贴主题代码，或在
			<MkA to="/theme-editor">主题编辑器</MkA>
			里做一个。
		</CfInfo>

		<div v-else class="themes">
			<article
				v-for="item in themes"
				:key="item.id"
				class="theme"
				:style="swatchStyle(item)"
			>
				<div class="swatch" aria-hidden="true">
					<span class="swatchPanel">
						<span class="swatchAccent"></span>
						<span class="swatchLine"></span>
						<span class="swatchLine short"></span>
					</span>
				</div>

				<div class="meta">
					<div class="name">{{ item.name }}</div>
					<div class="author">
						{{ item.author !== "" ? item.author : "未署名" }}
					</div>
					<div class="id">{{ item.id }}</div>
					<div class="tags">
						<span class="tag">{{ themeMode(item) === "dark" ? t("dark") : t("light") }}</span>
						<span v-if="activeId === item.id" class="tag accent">使用中</span>
						<span v-if="item.desc" class="desc">{{ item.desc }}</span>
					</div>
				</div>

				<div class="actions">
					<CfButton inline @click="download(item)">
						<i class="ph-download-simple ph-bold ph-lg"></i>
						导出 JSON
					</CfButton>
					<CfButton inline danger @click="remove(item)">
						<i class="ph-trash ph-bold ph-lg"></i>
						{{ t("uninstall") }}
					</CfButton>
				</div>
			</article>
		</div>

		<FormSection v-if="themes.length > 0">
			<template #label>说明</template>
			<CfInfo card>
				导出用
				<code>JSON.stringify</code>
				写出完整主题对象（含 <code>base</code> 与全部 <code>props</code>），可以直接当作主题文件再安装。上游支持在这里重命名主题；本项目的已安装主题以 <code>id</code> 为键，重命名需要重新安装，所以不提供。
			</CfInfo>
			<CfInfo class="_formBlock">
				卸载会同时清掉该主题在两种模式下被选中的记录，界面立刻回落到默认主题。
			</CfInfo>
		</FormSection>
	</div>
</template>

<script lang="ts" setup>
// Firefish's theme manager, rewritten around a list instead of a single
// `<FormSelect>` + detail pane, because the palette swatch is what makes a list
// of installed themes usable. Behaviour kept:
//   * only installed themes are listed (built-ins are not removable);
//   * the theme JSON can be exported — upstream copied it to the clipboard,
//     here it downloads a file through a Blob URL, which needs no clipboard
//     permission;
//   * removal asks for confirmation and is refused for anything that is not an
//     installed theme.
// Differences: no rename (the id is the storage key) and no select-then-act
// pane, so the page has an empty state instead.
import { computed, onActivated, onMounted, ref } from "vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import FormSection from "@/components/form/section.vue";
import { useTheme } from "@/composables/use-theme";
import { t } from "@/i18n";
import { alert, confirm, success } from "@/os";
import { definePageMetadata } from "@/scripts/page-metadata";
import {
	compileTheme,
	fetchInstalledThemes,
	listInstalledThemes,
	themeMode,
	uninstallTheme,
	type Theme,
} from "@/themes";

defineOptions({ name: "CfPageSettingsThemeManage" });

const { theme, selectedThemeIds, setThemeForMode } = useTheme();

const themes = ref<Theme[]>(listInstalledThemes());
const activeId = computed(() => theme.value.id);

async function refresh(): Promise<void> {
	try {
		await fetchInstalledThemes();
	} catch {
		// Offline: the localStorage cache is the best list available.
	}
	themes.value = listInstalledThemes();
}

/**
 * Compiles the theme once per card so the preview uses the real palette
 * (including inherited base tokens) instead of raw prop strings like
 * `:darken<3<@bg`, which CSS cannot read.
 */
function swatchStyle(item: Theme): Record<string, string> {
	try {
		const compiled = compileTheme(item);
		return {
			"--sw-bg": compiled.bg ?? "",
			"--sw-panel": compiled.panel ?? "",
			"--sw-accent": compiled.accent ?? "",
			"--sw-fg": compiled.fg ?? "",
			"--sw-divider": compiled.divider ?? "",
		};
	} catch {
		// Invalid references: the CSS fallbacks in `.swatch` take over.
		return {};
	}
}

function download(item: Theme): void {
	const blob = new Blob([JSON.stringify(item, null, "\t")], {
		type: "application/json",
	});
	const url = URL.createObjectURL(blob);
	const anchor = document.createElement("a");
	anchor.href = url;
	anchor.download = `${item.id}.json`;
	anchor.style.display = "none";
	document.body.append(anchor);
	anchor.click();
	anchor.remove();
	// Revoking synchronously can cancel the download in some browsers.
	setTimeout(() => URL.revokeObjectURL(url), 0);
}

async function remove(item: Theme): Promise<void> {
	const { canceled } = await confirm({
		type: "warning",
		text: `确定要卸载主题「${item.name}」吗？卸载后不能撤销。`,
	});
	if (canceled) return;

	try {
		await uninstallTheme(item.id);
	} catch (error) {
		const text = error instanceof Error ? error.message : String(error);
		alert({ type: "error", text: `卸载失败：${text}` });
		return;
	}
	// The slot would otherwise keep pointing at a theme that no longer exists,
	// leaving the previous CSS variables in place until the next reload.
	const mode = themeMode(item);
	if (selectedThemeIds[mode].value === item.id) setThemeForMode(mode, null);

	refresh();
	success();
}

onMounted(refresh);
onActivated(refresh);

definePageMetadata({
	title: t("_theme.manage"),
	icon: "ph-folder-notch-open ph-bold ph-lg",
});
</script>

<style lang="scss" scoped>
.cf-settings-theme-manage {
	.themes {
		display: flex;
		flex-direction: column;
		gap: 12px;
	}

	.theme {
		display: flex;
		align-items: center;
		gap: 14px;
		padding: 12px 14px;
		background: var(--panel);
		border: solid 1px var(--divider);
		border-radius: var(--radius);
		flex-wrap: wrap;

		> .swatch {
			flex-shrink: 0;
			inline-size: 64px;
			block-size: 52px;
			padding: 6px;
			box-sizing: border-box;
			background: var(--sw-bg, var(--bg));
			border: solid 1px var(--sw-divider, var(--divider));
			border-radius: 8px;
			overflow: hidden;

			> .swatchPanel {
				display: flex;
				flex-direction: column;
				gap: 3px;
				block-size: 100%;
				padding: 5px;
				box-sizing: border-box;
				background: var(--sw-panel, var(--panel));
				border-radius: 4px;
				border: solid 1px var(--sw-divider, var(--divider));

				> .swatchAccent {
					display: block;
					inline-size: 14px;
					block-size: 5px;
					background: var(--sw-accent, var(--accent));
					border-radius: 999px;
				}

				> .swatchLine {
					display: block;
					block-size: 2px;
					background: var(--sw-fg, var(--fg));
					opacity: 0.5;
					border-radius: 999px;

					&.short {
						inline-size: 60%;
					}
				}
			}
		}

		> .meta {
			flex: 1;
			min-inline-size: 180px;

			> .name {
				font-weight: bold;
				color: var(--fg);
				overflow-wrap: anywhere;
			}

			> .author {
				color: var(--fgTransparentWeak);
				font-size: 0.9em;
				overflow-wrap: anywhere;
			}

			> .id {
				color: var(--fgTransparentWeak);
				font-size: 0.8em;
				font-family: var(--monospace, monospace);
				overflow-wrap: anywhere;
			}

			> .tags {
				display: flex;
				flex-wrap: wrap;
				align-items: center;
				gap: 6px;
				margin-block-start: 4px;

				> .tag {
					padding: 1px 6px;
					border: solid 1px var(--divider);
					border-radius: 999px;
					color: var(--fgTransparentWeak);
					font-size: 0.75em;
				}

				> .tag.accent {
					border-color: var(--accent);
					color: var(--accent);
				}

				> .desc {
					color: var(--fgTransparentWeak);
					font-size: 0.8em;
					overflow-wrap: anywhere;
				}
			}
		}

		> .actions {
			display: flex;
			flex-wrap: wrap;
			gap: 8px;
		}
	}
}
</style>
