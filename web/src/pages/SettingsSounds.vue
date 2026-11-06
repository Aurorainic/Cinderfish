<!-- Ported from firefish@b425f61:packages/client/src/pages/settings/sounds.vue (AGPL-3.0-only) -->
<template>
	<div class="_formRoot cf-settings-sounds">
		<section class="_formBlock">
			<h2 class="sectionTitle">{{ t("masterVolume") }}</h2>

			<div class="rangeField">
				<span class="label">
					<i :class="muted ? 'ph-speaker-none' : 'ph-speaker-high'" class="ph-bold ph-lg"></i>
					{{ percent(masterVolumeRef) }}
				</span>
				<div class="rangeRow">
					<input
						v-model.number="masterVolumeRef"
						type="range"
						min="0"
						max="1"
						step="0.05"
						:aria-label="t('masterVolume')"
					/>
					<CfButton inline @click="toggleMute()">
						{{ muted ? "取消静音" : "静音" }}
					</CfButton>
				</div>
			</div>

			<CfInfo class="note">
				音量设为 0 即静音，播放时会直接跳过（<code>scripts/sound.ts</code> 的
				<code>playFile</code>）。
			</CfInfo>
		</section>

		<section class="_formBlock">
			<h2 class="sectionTitle">{{ t("sounds") }}</h2>

			<ul class="sounds">
				<li v-for="key in soundKeys" :key="key.key" class="sound">
					<span class="soundName">{{ t(`_sfx.${key.sfx}`) }}</span>
					<select
						class="soundType"
						:value="key.value.type"
						@change="onTypeChange(key, $event)"
					>
						<option value="none">{{ t("none") }}</option>
						<option
							v-for="file in sounds"
							:key="file"
							:value="file"
						>{{ file }}</option>
					</select>
					<CfButton inline :disabled="key.value.type === 'none'" @click="preview(key)">
						<i class="ph-play ph-bold"></i>
					</CfButton>
					<label class="soundVolume">
						<input
							v-model.number="key.value.volume"
							type="range"
							min="0"
							max="1"
							step="0.05"
							@change="saveSound(key.storeKey, key.value)"
						/>
						<span class="value">{{ percent(key.value.volume) }}</span>
					</label>
				</li>
			</ul>

			<div class="row">
				<CfButton danger @click="reset()">
					<i class="ph-arrow-clockwise ph-bold ph-lg"></i>
					{{ t("default") }}
				</CfButton>
			</div>
		</section>

		<CfInfo class="note">
			音色来自 <code>get-sounds</code>（服务器扫描
			<code>static-assets/sounds/</code> 下的 mp3），试听按钮即时播放。自动播放受浏览器策略
			限制时首次点击后才会有声音。自定义音效（插件/上传）未实现。
		</CfInfo>
	</div>
</template>

<script lang="ts" setup>
// Firefish's sounds page edits a ColdDeviceStorage entry per sound type through
// `os.form` and lists the available files from `get-sounds`. The port uses a
// native select per row plus a preview button; the options come from the
// endpoint and the preview calls the same `playFile` the timeline uses.
import { computed, onMounted, reactive, ref, watch } from "vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import { t } from "@/i18n";
import { apiGet } from "@/api/client";
import { definePageMetadata } from "@/scripts/page-metadata";
import { playFile } from "@/scripts/sound";
import { ColdDeviceStorage, type SoundConfig } from "@/store";

defineOptions({ name: "CfPageSettingsSounds" });

type SoundKey =
	| "note"
	| "noteMy"
	| "notification"
	| "chat"
	| "chatBg"
	| "antenna"
	| "channel";

interface SoundRow {
	key: SoundKey;
	/** ColdDeviceStorage key (`sound_<key>`). */
	storeKey: "sound_note" | "sound_noteMy" | "sound_notification" | "sound_chat" | "sound_chatBg" | "sound_antenna" | "sound_channel";
	/** Flat i18n key suffix under `_sfx.` (Firefish names). */
	sfx: string;
	value: SoundConfig;
}

const SFX_KEYS: { key: SoundKey; storeKey: SoundRow["storeKey"]; sfx: string }[] = [
	{ key: "note", storeKey: "sound_note", sfx: "note" },
	{ key: "noteMy", storeKey: "sound_noteMy", sfx: "noteMy" },
	{ key: "notification", storeKey: "sound_notification", sfx: "notification" },
	{ key: "chat", storeKey: "sound_chat", sfx: "chat" },
	{ key: "chatBg", storeKey: "sound_chatBg", sfx: "chatBg" },
	{ key: "antenna", storeKey: "sound_antenna", sfx: "antenna" },
	{ key: "channel", storeKey: "sound_channel", sfx: "channel" },
];

const masterVolumeRef = ref<number>(ColdDeviceStorage.get("sound_masterVolume"));

const muted = computed(() => masterVolumeRef.value === 0);

watch(masterVolumeRef, (value) => {
	ColdDeviceStorage.set(
		"sound_masterVolume",
		Math.min(1, Math.max(0, Number(value) || 0)),
	);
});

// `reactive` (not `ref`) so the nested `volume` inputs mutate the same objects
// that are written back to the cold store on change.
const soundKeys = reactive<SoundRow[]>(
	SFX_KEYS.map((item) => ({
		...item,
		value: { ...ColdDeviceStorage.get(item.storeKey) },
	})),
);

function percent(value: number): string {
	return `${Math.floor(value * 100)}%`;
}

function toggleMute(): void {
	masterVolumeRef.value = muted.value ? 0.3 : 0;
}

const sounds = ref<string[]>([]);

onMounted(() => {
	apiGet<(string | null)[]>("get-sounds")
		.then((list) => {
			sounds.value = list.filter((item): item is string => typeof item === "string");
		})
		.catch(() => {
			// The current values stay selectable even if the list fails.
		});
});

function onTypeChange(row: SoundRow, event: Event): void {
	const type = (event.target as HTMLSelectElement).value;
	row.value.type = type;
	saveSound(row.storeKey, row.value);
	if (type !== "none") playFile(type, row.value.volume);
}

function preview(row: SoundRow): void {
	playFile(row.value.type, row.value.volume);
}

function saveSound(
	storeKey: SoundRow["storeKey"],
	value: SoundConfig,
): void {
	ColdDeviceStorage.set(storeKey, { ...value });
}

function reset(): void {
	masterVolumeRef.value = ColdDeviceStorage.defaults.sound_masterVolume;
	for (const row of soundKeys) {
		const fresh = { ...ColdDeviceStorage.defaults[row.storeKey] };
		row.value = fresh;
		ColdDeviceStorage.set(row.storeKey, fresh);
	}
}

definePageMetadata({
	title: t("sounds"),
	icon: "ph-speaker-high ph-bold ph-lg",
});
</script>

<style lang="scss" scoped>
.cf-settings-sounds {
	.sectionTitle {
		margin: 0 0 4px;
		color: var(--fg);
		font-size: 1em;
		opacity: 0.85;
	}

	.rangeField {
		display: block;
		margin-block-start: 12px;

		> .label {
			display: flex;
			align-items: center;
			gap: 8px;
			color: var(--fg);
			font-size: 0.9em;
		}
	}

	.rangeRow {
		display: flex;
		align-items: center;
		gap: 12px;
		margin-block-start: 8px;

		input[type="range"] {
			flex: 1;
			min-inline-size: 0;
			accent-color: var(--accent);
		}
	}

	.sounds {
		margin: 12px 0 0;
		padding: 0;
		list-style: none;
		border-radius: var(--radius);
		background: var(--panel);
		overflow: hidden;
	}

	.sound {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 8px 12px;
		padding: 10px 14px;

		&:not(:last-child) {
			border-block-end: solid 0.5px var(--divider);
		}

		> .soundName {
			flex: 1 1 8em;
			font-size: 0.9em;
		}

		> .soundType {
			color: var(--fgTransparentWeak);
			font-size: 0.8em;
		}

		> .soundVolume {
			display: flex;
			align-items: center;
			gap: 8px;
			flex: 1 1 12em;

			input[type="range"] {
				flex: 1;
				min-inline-size: 0;
				accent-color: var(--accent);
			}

			> .value {
				min-inline-size: 3em;
				color: var(--fgTransparentWeak);
				font-size: 0.8em;
				font-variant-numeric: tabular-nums;
				text-align: end;
			}
		}
	}

	.row {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
		margin-block-start: 16px;
	}

	.note {
		margin-block-start: 12px;
		font-size: 0.85em;

		code {
			word-break: break-all;
		}
	}
}
</style>
