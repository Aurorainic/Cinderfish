<!-- Ported from firefish@b425f61:packages/client/src/pages/settings/reaction.vue (AGPL-3.0-only) -->
<template>
	<div class="_formRoot cf-settings-reaction">
		<CfSwitch v-model="enableEmojiReactions" class="_formBlock">
			{{ t("enableEmojiReactions") }}
		</CfSwitch>

		<template v-if="enableEmojiReactions">
			<section class="_formBlock">
				<h2 class="sectionTitle">{{ t("reactionSetting") }}</h2>

				<div class="picker">
					<div v-if="reactions.length === 0" class="empty">
						回应选择器里还没有快捷回应，点右侧的 + 添加。
					</div>
					<template v-else>
						<button
							v-for="(item, index) in reactions"
							:key="item"
							class="_button chip"
							:title="item"
							@click="openItemMenu(item, index, $event)"
						>
							<MkEmoji :emoji="item" class="emoji" />
						</button>
					</template>
					<button class="_button add" @click="chooseEmoji($event)">
						<i class="ph-plus ph-bold ph-lg"></i>
					</button>
				</div>

				<p class="caption">
					{{ t("reactionSettingDescription2") }}
				</p>
				<div class="row">
					<CfButton inline @click="preview($event)">
						<i class="ph-eye ph-bold ph-lg"></i>
						{{ t("preview") }}
					</CfButton>
					<CfButton inline danger @click="resetReactions()">
						<i class="ph-arrow-counter-clockwise ph-bold ph-lg"></i>
						{{ t("default") }}
					</CfButton>
				</div>
				<CfInfo class="note">
					拖拽排序需要 <code>vue-draggable-plus</code>，本波不引入：点击图标后在菜单中选择
					「上移 / 下移 / 删除」完成同样的调整。
				</CfInfo>
			</section>

			<section class="_formBlock">
				<h2 class="sectionTitle">最近使用</h2>
				<div v-if="recentlyUsed.length === 0" class="empty">
					最近还没有用过回应；发出一次回应后会出现在这里。
				</div>
				<div v-else class="picker">
					<button
						v-for="item in recentlyUsed"
						:key="item"
						class="_button chip"
						:title="item"
						@click="openRecentMenu(item, $event)"
					>
						<MkEmoji :emoji="item" class="emoji" />
					</button>
				</div>
				<div v-if="recentlyUsed.length > 0" class="row">
					<CfButton inline @click="clearRecentlyUsed()">
						{{ t("clear") }}
					</CfButton>
				</div>
			</section>
		</template>

		<template v-else>
			<CfSwitch v-model="showEmojisInReactionNotifications" class="_formBlock">
				{{ t("showEmojisInReactionNotifications") }}
			</CfSwitch>
		</template>

		<div class="pickerPrefs _formBlock">
			<label class="pref">
				<span>表情尺寸</span>
				<select :value="reactionPickerSize" @change="setPicker('reactionPickerSize', $event)">
					<option :value="1">小</option>
					<option :value="2">中</option>
					<option :value="3">大</option>
				</select>
			</label>
			<label class="pref">
				<span>每行列数</span>
				<select :value="reactionPickerWidth" @change="setPicker('reactionPickerWidth', $event)">
					<option v-for="n in 5" :key="n" :value="n">{{ n + 4 }}</option>
				</select>
			</label>
			<label class="pref">
				<span>可见行数</span>
				<select :value="reactionPickerHeight" @change="setPicker('reactionPickerHeight', $event)">
					<option :value="1">4</option>
					<option :value="2">6</option>
					<option :value="3">8</option>
					<option :value="4">10</option>
				</select>
			</label>
		</div>

		<CfInfo class="note">
			尺寸/列数/高度会实时作用到回应选择器。默认肤色仍由选择器内的色块手动选择；
			自定义 MFM 主题与声音同样未实现。
		</CfInfo>
	</div>
</template>

<script lang="ts" setup>
// Firefish edits `defaultStore.state.reactions` in place (drag to sort, click to
// remove) and stores the picker's tone/size/width/height preferences.
// Degradations for this wave (contract §2):
//   * no `vue-draggable-plus`: ordering moves into the per-item context menu;
//   * the emoji picker is composer-core's `MkEmojiPickerDialog` opened through
//     `os.popup`, which is what the contract fixes for reaction pickers;
//   * picker tone/size/width/height have no consumer yet, so they are not
//     exposed as controls (a switch that changes nothing would be a fake).
// `reactions` and `recentlyUsedEmojis` are both real consumers: MkEmojiPicker
// pins the former and lists the latter.
import { computed } from "vue";
import MkEmojiPickerDialog from "@/components/MkEmojiPickerDialog.vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfSwitch from "@/components/ui/CfSwitch.vue";
import { t } from "@/i18n";
import { confirm, popup, popupMenu, type MenuItem } from "@/os";
import { definePageMetadata } from "@/scripts/page-metadata";
import { defaultReactions, defaultStore } from "@/store";

const reactionPickerSize = defaultStore.reactiveState.reactionPickerSize;
const reactionPickerWidth = defaultStore.reactiveState.reactionPickerWidth;
const reactionPickerHeight = defaultStore.reactiveState.reactionPickerHeight;

defineOptions({ name: "CfPageSettingsReaction" });

const reactions = computed<string[]>(
	() => defaultStore.reactiveState.reactions.value,
);
const recentlyUsed = computed<string[]>(
	() => defaultStore.reactiveState.recentlyUsedEmojis.value,
);

const enableEmojiReactions = computed<boolean>({
	get: () => defaultStore.reactiveState.enableEmojiReactions.value,
	set: (value) => {
		defaultStore.set("enableEmojiReactions", value);
		void reloadAsk();
	},
});

const showEmojisInReactionNotifications = computed<boolean>({
	get: () => defaultStore.reactiveState.showEmojisInReactionNotifications.value,
	set: (value) => defaultStore.set("showEmojisInReactionNotifications", value),
});

async function reloadAsk(): Promise<void> {
	const { canceled } = await confirm({
		type: "info",
		text: t("reloadToApplySetting"),
	});
	if (canceled) return;
	location.reload();
}

function save(next: string[]): void {
	defaultStore.set("reactions", next);
}

function openItemMenu(item: string, index: number, ev: MouseEvent): void {
	const items: MenuItem[] = [];

	if (index > 0) {
		items.push({
			icon: "ph-arrow-up ph-bold ph-lg",
			text: "上移",
			action: () => {
				const next = [...reactions.value];
				[next[index - 1], next[index]] = [next[index], next[index - 1]];
				save(next);
			},
		});
	}
	if (index < reactions.value.length - 1) {
		items.push({
			icon: "ph-arrow-down ph-bold ph-lg",
			text: "下移",
			action: () => {
				const next = [...reactions.value];
				[next[index + 1], next[index]] = [next[index], next[index + 1]];
				save(next);
			},
		});
	}
	if (items.length > 0) items.push({ separator: true });
	items.push({
		icon: "ph-trash ph-bold ph-lg",
		text: t("remove"),
		danger: true,
		action: () => {
			save(reactions.value.filter((x) => x !== item));
		},
	});

	void popupMenu(items, ev);
}

function openRecentMenu(item: string, ev: MouseEvent): void {
	void popupMenu(
		[
			{
				icon: "ph-plus ph-bold ph-lg",
				text: "加入回应选择器",
				action: () => {
					if (!reactions.value.includes(item)) {
						save([...reactions.value, item]);
					}
				},
			},
			{
				icon: "ph-trash ph-bold ph-lg",
				text: t("remove"),
				danger: true,
				action: () => {
					defaultStore.set(
						"recentlyUsedEmojis",
						recentlyUsed.value.filter((x) => x !== item),
					);
				},
			},
		],
		ev,
	);
}

function chooseEmoji(ev: MouseEvent): void {
	popup(
		MkEmojiPickerDialog,
		{
			asReactionPicker: true,
			showPinned: false,
			src: ev.currentTarget ?? ev.target,
		},
		{
			done: (emoji: string) => {
				if (reactions.value.includes(emoji)) return;
				save([...reactions.value, emoji]);
			},
		},
		"closed",
	);
}

function preview(ev: MouseEvent): void {
	popup(
		MkEmojiPickerDialog,
		{
			asReactionPicker: true,
			showPinned: true,
			src: ev.currentTarget ?? ev.target,
		},
		{},
		"closed",
	);
}

async function resetReactions(): Promise<void> {
	const { canceled } = await confirm({
		type: "warning",
		text: t("resetAreYouSure"),
	});
	if (canceled) return;
	save([...defaultReactions]);
}

function clearRecentlyUsed(): void {
	defaultStore.set("recentlyUsedEmojis", []);
}

type PickerKey = "reactionPickerSize" | "reactionPickerWidth" | "reactionPickerHeight";

function setPicker(key: PickerKey, event: Event): void {
	const value = Number((event.target as HTMLSelectElement).value);
	defaultStore.set(key, value);
}

definePageMetadata({ title: t("reaction"), icon: "ph-smiley ph-bold ph-lg" });
</script>

<style lang="scss" scoped>
.cf-settings-reaction {
	.sectionTitle {
		margin: 0 0 4px;
		color: var(--fg);
		font-size: 1em;
		opacity: 0.85;
	}

	.picker {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 4px;
		margin-block-start: 12px;
		padding: 10px;
		border-radius: var(--radius);
		background: var(--panel);
		border: solid 0.5px var(--divider);
	}

	.chip,
	.add {
		display: inline-flex;
		align-items: center;
		justify-content: center;
		min-inline-size: 2.4em;
		block-size: 2.4em;
		border-radius: calc(var(--radius) / 2);
		cursor: pointer;
		transition: background 0.1s ease;

		&:hover {
			background: var(--buttonBg);
		}

		&:active {
			background: var(--accentedBg);
		}
	}

	.emoji {
		block-size: 1.7em;
	}

	.caption {
		margin: 8px 0 0;
		color: var(--fgTransparentWeak);
		font-size: 0.85em;
	}

	.empty {
		color: var(--fgTransparentWeak);
		font-size: 0.9em;
	}

	.row {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
		margin-block-start: 12px;
	}

	.pickerPrefs {
		display: flex;
		flex-wrap: wrap;
		gap: 12px;
		margin-block-start: 12px;

		> .pref {
			display: flex;
			flex-direction: column;
			gap: 4px;
			font-size: 0.85em;
			opacity: 0.85;

			> select {
				padding: 4px 8px;
				border-radius: 6px;
				border: solid 0.5px var(--divider);
				background: var(--panel);
				color: var(--fg);
			}
		}
	}

	.note {
		margin-block-start: 16px;
		font-size: 0.85em;

		code {
			word-break: break-all;
		}
	}
}
</style>
