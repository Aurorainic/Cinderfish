<!-- Ported from firefish@b425f61:packages/client/src/pages/settings/navbar.vue (AGPL-3.0-only) -->
<template>
	<div class="_formRoot cf-settings-navbar">
		<section class="_formBlock">
			<h2 class="_sectionTitle">{{ t("navbar") }}</h2>

			<div v-if="menu.length === 0" class="empty">
				导航栏里没有任何项目，从下面的「未启用」列表添加。
			</div>
			<ul v-else class="items">
				<li v-for="(entry, index) in menuEntries" :key="entry.key" class="item">
					<i class="icon ph-bold ph-lg" :class="entry.icon"></i>
					<span class="label">{{ entry.label }}</span>
					<span class="buttons">
						<button
							class="_button iconButton"
							:disabled="index === 0"
							:aria-label="'上移'"
							@click="move(index, -1)"
						>
							<i class="ph-arrow-up ph-bold"></i>
						</button>
						<button
							class="_button iconButton"
							:disabled="index === menu.length - 1"
							:aria-label="'下移'"
							@click="move(index, 1)"
						>
							<i class="ph-arrow-down ph-bold"></i>
						</button>
						<button
							class="_button iconButton"
							:aria-label="t('remove')"
							@click="disable(entry.key)"
						>
							<i class="ph-minus ph-bold"></i>
						</button>
					</span>
				</li>
			</ul>

			<template v-if="disabledEntries.length > 0">
				<h3 class="subTitle">{{ t("addItem") }}</h3>
				<ul class="items">
					<li v-for="entry in disabledEntries" :key="entry.key" class="item">
						<i class="icon ph-bold ph-lg" :class="entry.icon"></i>
						<span class="label">{{ entry.label }}</span>
						<span class="buttons">
							<button
								class="_button iconButton"
								:aria-label="t('add')"
								@click="enable(entry.key)"
							>
								<i class="ph-plus ph-bold"></i>
							</button>
						</span>
					</li>
				</ul>
			</template>

			<div class="_row">
				<CfButton danger @click="reset()">
					<i class="ph-arrow-clockwise ph-bold ph-lg"></i>
					{{ t("default") }}
				</CfButton>
				<CfButton @click="reload()">刷新页面</CfButton>
			</div>

			<CfInfo class="_note">
				顺序与显隐立即写入 <code>defaultStore.state.menu</code>；<code>MkNavbar</code>
				读取的是非响应式的 <code>store.state</code>，所以点「刷新页面」后才会反映到侧栏。
				Firefish 的「分割线」（<code>-</code>）在 Cinderfish 的菜单类型里<strong>未提供</strong>，
				因此不能插入；「顶部」显示模式同样未提供，只有横向与横向（图标）两种。
			</CfInfo>
		</section>

		<section class="_formBlock">
			<h2 class="_sectionTitle">{{ t("display") }}</h2>
			<CfRadios :model-value="menuDisplay" @update:model-value="onMenuDisplay">
				<CfRadio value="sideFull">{{ t("_menuDisplay.sideFull") }}</CfRadio>
				<CfRadio value="sideIcon">{{ t("_menuDisplay.sideIcon") }}</CfRadio>
			</CfRadios>
			<CfInfo class="_note">
				显示模式在侧栏重新挂载后才生效；仓库里没有 <code>unisonReload</code>，
				这里用页面刷新代替。
			</CfInfo>
		</section>
	</div>
</template>

<script lang="ts" setup>
// Firefish edits `defaultStore.state.menu` as a newline-separated textarea with a
// picker that appends entries, and reloads the client after every change. The
// textarea is replaced by an explicit ordered list: the menu is a closed union of
// seven entries in Cinderfish (`MenuOption` in store/base.ts), so a text field
// would accept values the navbar cannot render. Each change is persisted at once
// and applied on the next render; the reload is a button instead of a modal per
// keystroke.
import { computed } from "vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfRadio from "@/components/ui/CfRadio.vue";
import CfRadios from "@/components/ui/CfRadios.vue";
import type { RadioValue } from "@/components/ui/radios-key";
import { t } from "@/i18n";
import { definePageMetadata } from "@/scripts/page-metadata";
import { defaultStore, menuOptions, type MenuOption } from "@/store";

defineOptions({ name: "CfPageSettingsNavbar" });

interface EntryDef {
	key: MenuOption;
	labelKey: string;
	icon: string;
}

// Same table (and same icons) as `components/MkNavbar.vue` renders; the labels
// reuse the flat i18n keys that navbar already uses. `t()` is applied inside the
// computed lists so a locale switch re-renders the labels.
const ENTRY_DEFS: EntryDef[] = [
	{ key: "notifications", labelKey: "notifications", icon: "ph-bell ph-bold" },
	{ key: "followRequests", labelKey: "followRequests", icon: "ph-hand-waving ph-bold" },
	{ key: "messaging", labelKey: "messaging", icon: "ph-chats-teardrop ph-bold" },
	{ key: "explore", labelKey: "explore", icon: "ph-compass ph-bold" },
	{ key: "favorites", labelKey: "favorites", icon: "ph-bookmark-simple ph-bold" },
	{ key: "channels", labelKey: "channel", icon: "ph-television ph-bold" },
	{ key: "search", labelKey: "search", icon: "ph-magnifying-glass ph-bold" },
];

interface MenuEntryView {
	key: MenuOption;
	label: string;
	icon: string;
}

function view(def: EntryDef): MenuEntryView {
	return { key: def.key, label: t(def.labelKey), icon: def.icon };
}

const menu = computed<MenuOption[]>(
	() => defaultStore.reactiveState.menu.value,
);

const menuEntries = computed<MenuEntryView[]>(() =>
	menu.value
		.map((key) => ENTRY_DEFS.find((entry) => entry.key === key))
		.filter((entry): entry is EntryDef => entry != null)
		.map(view),
);

const disabledEntries = computed<MenuEntryView[]>(() =>
	ENTRY_DEFS.filter((entry) => !menu.value.includes(entry.key)).map(view),
);

const menuDisplay = computed<string>({
	get: () => defaultStore.reactiveState.menuDisplay.value,
	set: (value) => {
		if (value === "sideFull" || value === "sideIcon") {
			defaultStore.set("menuDisplay", value);
		}
	},
});

function save(next: MenuOption[]): void {
	defaultStore.set("menu", next);
}

function move(index: number, delta: -1 | 1): void {
	const next = [...menu.value];
	const target = index + delta;
	if (target < 0 || target >= next.length) return;
	[next[index], next[target]] = [next[target], next[index]];
	save(next);
}

function disable(key: MenuOption): void {
	save(menu.value.filter((item) => item !== key));
}

function enable(key: MenuOption): void {
	if (menu.value.includes(key)) return;
	save([...menu.value, key]);
}

function reset(): void {
	defaultStore.reset("menu");
}

function onMenuDisplay(value: RadioValue): void {
	if (typeof value === "string") menuDisplay.value = value;
}

function reload(): void {
	location.reload();
}

// `menuOptions` is the canonical list; the table above must not drift from it.
if (ENTRY_DEFS.length !== menuOptions.length) {
	console.warn("SettingsNavbar: entry table and menuOptions disagree");
}

definePageMetadata({ title: t("navbar"), icon: "ph-list-bullets ph-bold ph-lg" });
</script>

<style lang="scss" scoped>
.cf-settings-navbar {
	.subTitle {
		margin: 16px 0 4px;
		color: var(--fgTransparentWeak);
		font-size: 0.9em;
		font-weight: normal;
	}

	.empty {
		margin-block-start: 12px;
		color: var(--fgTransparentWeak);
		font-size: 0.9em;
	}

	.items {
		margin: 12px 0 0;
		padding: 0;
		list-style: none;
		border-radius: var(--radius);
		background: var(--panel);
		overflow: hidden;
	}

	.item {
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 8px 12px;

		&:not(:last-child) {
			border-block-end: solid 0.5px var(--divider);
		}

		> .label {
			flex: 1;
			min-inline-size: 0;
			font-size: 0.9em;
		}

		> .buttons {
			display: flex;
			gap: 2px;
		}

		> .iconButton {
			display: inline-flex;
			align-items: center;
			justify-content: center;
			inline-size: 2em;
			block-size: 2em;
			border-radius: calc(var(--radius) / 2);
			cursor: pointer;

			&:hover:not(:disabled) {
				background: var(--buttonBg);
			}

			&:disabled {
				opacity: 0.3;
				cursor: default;
			}
		}
	}

	._note code {
		word-break: break-all;
	}
}
</style>
