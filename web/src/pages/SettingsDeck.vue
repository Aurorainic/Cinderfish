<!-- Ported from firefish@b425f61:packages/client/src/pages/settings/deck.vue (AGPL-3.0-only) -->
<template>
	<div class="_formRoot cf-settings-deck">
		<section class="_formBlock">
			<h2 class="_sectionTitle">{{ t("defaultNavigationBehaviour") }}</h2>
			<CfSwitch v-model="navWindow" class="_formBlock">
				{{ t("openInWindow") }}
			</CfSwitch>
			<CfSwitch v-model="alwaysShowMainColumn" class="_formBlock">
				{{ t("_deck.alwaysShowMainColumn") }}
			</CfSwitch>
			<CfRadios :model-value="columnAlign" @update:model-value="onColumnAlign">
				<template #label>{{ t("_deck.columnAlign") }}</template>
				<CfRadio value="left">{{ t("left") }}</CfRadio>
				<CfRadio value="center">{{ t("center") }}</CfRadio>
			</CfRadios>
		</section>

		<section class="_formBlock">
			<h2 class="_sectionTitle">{{ t("_deck.profile") }}</h2>

			<div v-if="phase === 'loading'" class="state"><CfLoading /></div>
			<div v-else-if="phase === 'error'" class="state">
				<CfInfo :warn="true">{{ errorText }}</CfInfo>
				<CfButton class="retry" inline @click="load()">{{ t("retry") }}</CfButton>
			</div>
			<template v-else>
				<div class="profileRow">
					<span class="profileName">{{ deckStore.state.profile }}</span>
					<CfButton inline @click="load()">
						<i class="ph-arrows-clockwise ph-bold ph-lg"></i>
						重新读取
					</CfButton>
				</div>

				<div v-if="columns.length === 0" class="empty">
					{{ t("_deck.introduction") }}
					{{ t("_deck.introduction2") }}
				</div>
				<ul v-else class="columns">
					<li v-for="column in columns" :key="column.id" class="column">
						<span class="columnName">{{ columnLabel(column) }}</span>
						<label class="columnWidth">
							<span class="label">{{ t("width") }}</span>
							<input
								type="number"
								:value="column.width"
								min="250"
								max="800"
								step="10"
								@change="setWidth(column.id, $event)"
							/>
							<span class="unit">px</span>
						</label>
						<span class="buttons">
							<button
								class="_button iconButton"
								:disabled="!inLayout(column.id)"
								:aria-label="t('_deck.swapLeft')"
								@click="swapLeftColumn(column.id)"
							>
								<i class="ph-arrow-left ph-bold"></i>
							</button>
							<button
								class="_button iconButton"
								:disabled="!inLayout(column.id)"
								:aria-label="t('_deck.swapRight')"
								@click="swapRightColumn(column.id)"
							>
								<i class="ph-arrow-right ph-bold"></i>
							</button>
							<button
								class="_button iconButton danger"
								:aria-label="t('remove')"
								@click="remove(column)"
							>
								<i class="ph-trash ph-bold"></i>
							</button>
						</span>
					</li>
				</ul>

				<CfInfo class="_note">
					Deck 的列布局保存在账号 registry 的 <code>["client", "deck", "profiles"]</code>
					下，键是当前工作区名。这里可以调整列宽、左右顺序并删除列；
					添加列与列内小部件需要 Deck 界面的列菜单，本页<strong>未提供</strong>。
					不在布局里的列（<code>inLayout</code> 为否）无法移动，先在 Deck 界面打开它。
				</CfInfo>
			</template>
		</section>
	</div>
</template>

<script lang="ts" setup>
// Firefish's deck settings page holds the three navigation switches. This port
// adds the column list the task asks for: it reads the same `deckStore` (and the
// same registry profile) as `ui/layouts/DeckLayout.vue`, so width and order
// changes made here are what the deck shell renders. Adding a column stays in
// the deck UI, whose column menu owns the type-specific arguments.
import { computed, onMounted, ref } from "vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import CfRadio from "@/components/ui/CfRadio.vue";
import CfRadios from "@/components/ui/CfRadios.vue";
import type { RadioValue } from "@/components/ui/radios-key";
import { t } from "@/i18n";
import { confirm } from "@/os";
import { definePageMetadata } from "@/scripts/page-metadata";
import {
	deckStore,
	loadDeck,
	removeColumn,
	swapLeftColumn,
	swapRightColumn,
	updateColumn,
	type Column,
} from "@/store";

defineOptions({ name: "CfPageSettingsDeck" });

const phase = ref<"loading" | "ready" | "error">("loading");
const errorText = ref<string | null>(null);

const columns = computed<Column[]>(
	() => deckStore.reactiveState.columns.value,
);
const layoutIds = computed<string[]>(
	() => deckStore.reactiveState.layout.value.flat(),
);

const navWindow = computed<boolean>({
	get: () => deckStore.reactiveState.navWindow.value,
	set: (value) => deckStore.set("navWindow", value),
});
const alwaysShowMainColumn = computed<boolean>({
	get: () => deckStore.reactiveState.alwaysShowMainColumn.value,
	set: (value) => deckStore.set("alwaysShowMainColumn", value),
});
const columnAlign = computed<string>({
	get: () => deckStore.reactiveState.columnAlign.value,
	set: (value) => {
		if (value === "left" || value === "center" || value === "right") {
			deckStore.set("columnAlign", value);
		}
	},
});

/** Static key per kind, mirroring the deck column header (no dynamic keys). */
function columnLabel(column: Column): string {
	if (column.name != null && column.name !== "") return column.name;
	switch (column.type) {
		case "main":
			return t("_deck._columns.main");
		case "notifications":
			return t("_deck._columns.notifications");
		case "tl":
			return t("_deck._columns.tl");
		case "widgets":
			return t("_deck._columns.widgets");
		case "mentions":
			return t("_deck._columns.mentions");
		case "direct":
			return t("_deck._columns.direct");
		case "antenna":
			return t("_deck._columns.antenna");
		case "list":
			return t("_deck._columns.list");
		case "channel":
			return t("_deck._columns.channel");
		case "favorites":
			return t("favorites");
	}
}

function inLayout(id: string): boolean {
	return layoutIds.value.includes(id);
}

async function load(): Promise<void> {
	phase.value = "loading";
	errorText.value = null;
	try {
		await loadDeck();
		phase.value = "ready";
	} catch (error) {
		errorText.value =
			error instanceof Error && error.message !== ""
				? error.message
				: t("somethingHappened");
		phase.value = "error";
	}
}

function setWidth(id: string, event: Event): void {
	const input = event.target as HTMLInputElement | null;
	if (input == null) return;
	const parsed = Number(input.value);
	if (!Number.isFinite(parsed)) return;
	const width = Math.min(800, Math.max(250, Math.round(parsed)));
	input.value = String(width);
	updateColumn(id, { width });
}

async function remove(column: Column): Promise<void> {
	const { canceled } = await confirm({
		type: "warning",
		text: t("removeAreYouSure", { x: columnLabel(column) }),
	});
	if (canceled) return;
	removeColumn(column.id);
}

function onColumnAlign(value: RadioValue): void {
	if (typeof value === "string") columnAlign.value = value;
}

onMounted(() => {
	void load();
});

definePageMetadata({ title: t("deck"), icon: "ph-layout ph-bold ph-lg" });
</script>

<style lang="scss" scoped>
.cf-settings-deck {
	.state {
		display: flex;
		flex-direction: column;
		align-items: flex-start;
		gap: 12px;
		margin-block-start: 12px;
	}

	.profileRow {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 12px;
		margin-block-start: 12px;

		> .profileName {
			font-weight: bold;
			overflow: hidden;
			text-overflow: ellipsis;
			white-space: nowrap;
		}
	}

	.empty {
		margin-block-start: 12px;
		color: var(--fgTransparentWeak);
		font-size: 0.9em;
	}

	.columns {
		margin: 12px 0 0;
		padding: 0;
		list-style: none;
		border-radius: var(--radius);
		background: var(--panel);
		overflow: hidden;
	}

	.column {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 8px 12px;
		padding: 10px 14px;

		&:not(:last-child) {
			border-block-end: solid 0.5px var(--divider);
		}

		> .columnName {
			flex: 1 1 8em;
			font-size: 0.9em;
			overflow: hidden;
			text-overflow: ellipsis;
			white-space: nowrap;
		}

		> .columnWidth {
			display: flex;
			align-items: center;
			gap: 6px;
			font-size: 0.85em;
			color: var(--fgTransparentWeak);

			input[type="number"] {
				inline-size: 5em;
				padding: 4px 6px;
				border: solid 1px var(--divider);
				border-radius: calc(var(--radius) / 2);
				background: var(--bg);
				color: var(--fg);
				font-size: 0.9em;
			}
		}

		> .buttons {
			display: flex;
			gap: 2px;

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

				&.danger {
					color: var(--error);
				}
			}
		}
	}

	._note code {
		word-break: break-all;
	}
}
</style>
