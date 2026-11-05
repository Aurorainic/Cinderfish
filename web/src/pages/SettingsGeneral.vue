<!-- Ported from firefish@b425f61:packages/client/src/pages/settings/general.vue (AGPL-3.0-only) -->
<template>
	<div class="_formRoot cf-settings-general">
		<CfSelect class="_formBlock" :model-value="locale" @update:model-value="onLocale">
			<template #label>{{ t("uiLanguage") }}</template>
			<option v-for="code in availableLocales" :key="code" :value="code">
				{{ localeLabel(code) }}
			</option>
			<template #caption>{{ t("i18nInfo") }}</template>
		</CfSelect>

		<CfRadios
			class="_formBlock"
			:model-value="mode"
			@update:model-value="onThemeMode"
		>
			<template #label>{{ t("theme") }}</template>
			<CfRadio value="light"
				><i class="ph-sun ph-bold ph-lg"></i> {{ t("light") }}</CfRadio
			>
			<CfRadio value="dark"
				><i class="ph-moon ph-bold ph-lg"></i> {{ t("dark") }}</CfRadio
			>
			<CfRadio value="auto">{{ t("auto") }}</CfRadio>
		</CfRadios>

		<CfRadios
			class="_formBlock"
			:model-value="overridedDeviceKind"
			@update:model-value="onDeviceKind"
		>
			<template #label>{{ t("overridedDeviceKind") }}</template>
			<CfRadio :value="null">{{ t("auto") }}</CfRadio>
			<CfRadio value="smartphone"
				><i class="ph-device-mobile ph-bold ph-lg"></i>
				{{ t("smartphone") }}</CfRadio
			>
			<CfRadio value="tablet"
				><i class="ph-device-tablet ph-bold ph-lg"></i>
				{{ t("tablet") }}</CfRadio
			>
			<CfRadio value="desktop"
				><i class="ph-desktop ph-bold ph-lg"></i>
				{{ t("desktop") }}</CfRadio
			>
		</CfRadios>

		<section class="_formBlock">
			<h2 class="sectionTitle">{{ t("behavior") }}</h2>
			<CfSwitch v-model="imageNewTab" class="_formBlock">
				{{ t("openImageInNewTab") }}
			</CfSwitch>
			<CfSwitch v-model="enableInfiniteScroll" class="_formBlock">
				{{ t("enableInfiniteScroll") }}
			</CfSwitch>
			<CfSwitch v-model="useReactionPickerForContextMenu" class="_formBlock">
				{{ t("useReactionPickerForContextMenu") }}
			</CfSwitch>
			<CfSwitch
				v-if="deviceKind !== 'desktop'"
				v-model="swipeOnMobile"
				class="_formBlock"
			>
				{{ t("swipeOnMobile") }}
			</CfSwitch>
			<CfSwitch
				v-if="deviceKind === 'desktop'"
				v-model="swipeOnDesktop"
				class="_formBlock"
			>
				{{ t("swipeOnDesktop") }}
			</CfSwitch>
			<CfSwitch v-model="enterSendsMessage" class="_formBlock">
				{{ t("enterSendsMessage") }}
			</CfSwitch>
			<CfSwitch v-model="disablePagesScript" class="_formBlock">
				{{ t("disablePagesScript") }}
			</CfSwitch>
			<CfSwitch v-model="showTimelineReplies" class="_formBlock">
				{{ t("flagShowTimelineReplies") }}
				<template #caption>
					{{ t("flagShowTimelineRepliesDescription") }}
					{{ t("reflectMayTakeTime") }}
				</template>
			</CfSwitch>

			<CfSelect
				:model-value="serverDisconnectedBehavior"
				@update:model-value="onServerDisconnectedBehavior"
			>
				<template #label>{{ t("whenServerDisconnected") }}</template>
				<option value="reload">
					{{ t("_serverDisconnectedBehavior.reload") }}
				</option>
				<option value="dialog">
					{{ t("_serverDisconnectedBehavior.dialog") }}
				</option>
				<option value="quiet">
					{{ t("_serverDisconnectedBehavior.quiet") }}
				</option>
				<option value="nothing">
					{{ t("_serverDisconnectedBehavior.nothing") }}
				</option>
			</CfSelect>
		</section>

		<section class="_formBlock">
			<h2 class="sectionTitle">{{ t("accessibility") }}</h2>
			<CfSwitch v-model="expandOnNoteClick" class="_formBlock">
				{{ t("expandOnNoteClick") }}
				<template #caption>{{ t("expandOnNoteClickDesc") }}</template>
			</CfSwitch>
			<CfSwitch v-model="advancedMfm" class="_formBlock">
				{{ t("_mfm.advanced") }}
				<template #caption>{{ t("_mfm.advancedDescription") }}</template>
			</CfSwitch>
			<CfSwitch v-model="autoplayMfm" class="_formBlock">
				{{ t("_mfm.alwaysPlay") }}
				<template #caption>{{ t("_mfm.warn") }}</template>
			</CfSwitch>
			<CfSwitch v-model="reduceAnimation" class="_formBlock">
				{{ t("reduceUiAnimation") }}
			</CfSwitch>
			<CfSwitch v-model="disableShowingAnimatedImages" class="_formBlock">
				{{ t("disableShowingAnimatedImages") }}
			</CfSwitch>
		</section>

		<section class="_formBlock">
			<h2 class="sectionTitle">{{ t("appearance") }}</h2>
			<CfSwitch v-model="showAds" class="_formBlock">
				{{ t("showAds") }}
			</CfSwitch>
			<CfSwitch v-model="useBlurEffect" class="_formBlock">
				{{ t("useBlurEffect") }}
			</CfSwitch>
			<CfSwitch v-model="useBlurEffectForModal" class="_formBlock">
				{{ t("useBlurEffectForModal") }}
			</CfSwitch>
			<CfSwitch v-model="showGapBetweenNotesInTimeline" class="_formBlock">
				{{ t("showGapBetweenNotesInTimeline") }}
			</CfSwitch>
			<CfSwitch v-model="loadRawImages" class="_formBlock">
				{{ t("loadRawImages") }}
			</CfSwitch>
			<CfSwitch v-model="squareAvatars" class="_formBlock">
				{{ t("squareAvatars") }}
			</CfSwitch>
			<CfSwitch v-model="seperateRenoteQuote" class="_formBlock">
				{{ t("seperateRenoteQuote") }}
			</CfSwitch>
			<CfSwitch v-model="useOsNativeEmojis" class="_formBlock">
				{{ t("useOsNativeEmojis") }}
			</CfSwitch>
			<CfSwitch v-model="disableDrawer" class="_formBlock">
				{{ t("disableDrawer") }}
			</CfSwitch>
			<CfSwitch v-model="showUpdates" class="_formBlock">
				{{ t("showUpdates") }}
			</CfSwitch>
			<CfSwitch v-model="showFixedPostForm" class="_formBlock">
				{{ t("showFixedPostForm") }}
			</CfSwitch>
			<CfSwitch v-if="me?.isAdmin === true" v-model="showAdminUpdates" class="_formBlock">
				{{ t("showAdminUpdates") }}
			</CfSwitch>

			<CfSelect
				:model-value="instanceTicker"
				@update:model-value="onInstanceTicker"
			>
				<template #label>{{ t("instanceTicker") }}</template>
				<option value="none">{{ t("_instanceTicker.none") }}</option>
				<option value="remote">{{ t("_instanceTicker.remote") }}</option>
				<option value="always">{{ t("_instanceTicker.always") }}</option>
			</CfSelect>

			<CfSelect class="field" :model-value="nsfw" @update:model-value="onNsfw">
				<template #label>{{ t("nsfw") }}</template>
				<option value="respect">{{ t("_nsfw.respect") }}</option>
				<option value="ignore">{{ t("_nsfw.ignore") }}</option>
				<option value="force">{{ t("_nsfw.force") }}</option>
			</CfSelect>

			<CfKeyValue class="field" oneline>
				<template #key>{{ t("fontSize") }}</template>
				<template #value>{{ t("unavailable") }}</template>
			</CfKeyValue>
		</section>

		<section class="_formBlock">
			<h2 class="sectionTitle">{{ t("other") }}</h2>
			<label class="rangeField">
				<span class="label">{{ t("numberOfPageCache") }}</span>
				<span class="rangeRow">
					<input
						v-model.number="numberOfPageCache"
						type="range"
						min="1"
						max="10"
						step="1"
					/>
					<span class="rangeValue">{{ numberOfPageCache }}</span>
				</span>
				<span class="caption">{{ t("numberOfPageCacheDescription") }}</span>
			</label>

			<div class="_formLinks">
				<CfA class="linkRow" to="/settings/deck">
					<i class="ph-columns ph-bold ph-lg"></i>
					{{ t("deck") }}
				</CfA>
				<CfA class="linkRow" to="/settings/custom-katex-macro">
					<i class="ph-radical ph-bold ph-lg"></i>
					{{ t("customKaTeXMacro") }}
				</CfA>
			</div>
		</section>
	</div>
</template>

<script lang="ts" setup>
// Firefish keeps this page's preferences in `defaultStore` (device/account
// buckets) and in three bare localStorage keys (`lang`, `fontSize`,
// `useSystemFont`). Cinderfish has the store keys, so every switch below is a
// live two-way binding; language comes from the i18n runtime and the light/dark
// choice from useTheme() instead of `localStorage.lang`. `fontSize` and
// `useSystemFont` have neither a store key nor a consumer here, so font size is
// shown as unavailable rather than as a control that would do nothing.
import { computed, watch } from "vue";
import { me } from "@/account";
import CfA from "@/components/ui/CfA.vue";
import CfKeyValue from "@/components/ui/CfKeyValue.vue";
import CfRadio from "@/components/ui/CfRadio.vue";
import CfRadios from "@/components/ui/CfRadios.vue";
import CfSelect from "@/components/ui/CfSelect.vue";
import CfSwitch from "@/components/ui/CfSwitch.vue";
import type { RadioValue } from "@/components/ui/radios-key";
import { useTheme } from "@/composables/use-theme";
import { t, useLocale, type LocaleCode } from "@/i18n";
import { confirm } from "@/os";
import { deviceKind } from "@/scripts/device-kind";
import { definePageMetadata } from "@/scripts/page-metadata";
import { defaultStore } from "@/store";

defineOptions({ name: "CfPageSettingsGeneral" });

type StoreKey = keyof typeof defaultStore.state & string;
type StoreValue<K extends StoreKey> = (typeof defaultStore.state)[K];

function storeRef<K extends StoreKey>(key: K) {
	return computed<StoreValue<K>>({
		get: () => defaultStore.reactiveState[key].value as StoreValue<K>,
		set: (value) => defaultStore.set(key, value),
	});
}

const { locale, setLocale, availableLocales } = useLocale();
const { mode, setMode } = useTheme();

const localeLabels: Record<LocaleCode, string> = {
	"zh-CN": "简体中文",
	"ja-JP": "日本語",
};

function localeLabel(code: LocaleCode): string {
	return localeLabels[code] ?? code;
}

function onLocale(value: string | null): void {
	if (value == null) return;
	if (!(availableLocales as readonly string[]).includes(value)) return;
	setLocale(value as LocaleCode);
}

function onThemeMode(value: RadioValue): void {
	if (value === "light" || value === "dark" || value === "auto") {
		setMode(value);
	}
}

const overridedDeviceKind = storeRef("overridedDeviceKind");

async function reloadAsk(): Promise<void> {
	const { canceled } = await confirm({
		type: "info",
		text: t("reloadToApplySetting"),
	});
	if (canceled) return;
	location.reload();
}

function onDeviceKind(value: RadioValue): void {
	const next =
		value === "smartphone" || value === "tablet" || value === "desktop"
			? value
			: null;
	if (overridedDeviceKind.value === next) return;
	overridedDeviceKind.value = next;
	void reloadAsk();
}

const imageNewTab = storeRef("imageNewTab");
const enableInfiniteScroll = storeRef("enableInfiniteScroll");
const useReactionPickerForContextMenu = storeRef(
	"useReactionPickerForContextMenu",
);
const swipeOnMobile = storeRef("swipeOnMobile");
const swipeOnDesktop = storeRef("swipeOnDesktop");
const enterSendsMessage = storeRef("enterSendsMessage");
const disablePagesScript = storeRef("disablePagesScript");
const showTimelineReplies = storeRef("showTimelineReplies");
const serverDisconnectedBehavior = storeRef("serverDisconnectedBehavior");
const expandOnNoteClick = storeRef("expandOnNoteClick");
const advancedMfm = storeRef("advancedMfm");
const disableShowingAnimatedImages = storeRef("disableShowingAnimatedImages");
const showAds = storeRef("showAds");
const useBlurEffect = storeRef("useBlurEffect");
const useBlurEffectForModal = storeRef("useBlurEffectForModal");
const showGapBetweenNotesInTimeline = storeRef("showGapBetweenNotesInTimeline");
const loadRawImages = storeRef("loadRawImages");
const squareAvatars = storeRef("squareAvatars");
const seperateRenoteQuote = storeRef("seperateRenoteQuote");
const useOsNativeEmojis = storeRef("useOsNativeEmojis");
const disableDrawer = storeRef("disableDrawer");
const showUpdates = storeRef("showUpdates");
const showFixedPostForm = storeRef("showFixedPostForm");
const showAdminUpdates = storeRef("showAdminUpdates");
const instanceTicker = storeRef("instanceTicker");
const nsfw = storeRef("nsfw");
const numberOfPageCache = storeRef("numberOfPageCache");

// Firefish inverts these two store keys for display.
const reduceAnimation = computed<boolean>({
	get: () => !defaultStore.reactiveState.animation.value,
	set: (value) => defaultStore.set("animation", !value),
});
const autoplayMfm = computed<boolean>({
	get: () => !defaultStore.reactiveState.animatedMfm.value,
	set: (value) => defaultStore.set("animatedMfm", !value),
});

function onServerDisconnectedBehavior(value: string | null): void {
	if (
		value === "reload" ||
		value === "dialog" ||
		value === "quiet" ||
		value === "nothing"
	) {
		serverDisconnectedBehavior.value = value;
	}
}

function onInstanceTicker(value: string | null): void {
	if (value === "none" || value === "remote" || value === "always") {
		instanceTicker.value = value;
	}
}

function onNsfw(value: string | null): void {
	if (value === "respect" || value === "ignore" || value === "force") {
		nsfw.value = value;
	}
}

// Upstream: turning on the desktop swipe gesture also enables it on mobile.
watch(swipeOnDesktop, () => {
	if (swipeOnDesktop.value) swipeOnMobile.value = true;
});

definePageMetadata({ title: t("general"), icon: "ph-gear-six ph-bold ph-lg" });
</script>

<style lang="scss" scoped>
.cf-settings-general {
	.sectionTitle {
		margin: 0 0 4px;
		color: var(--fg);
		font-size: 1em;
		opacity: 0.85;
	}

	.field {
		margin-block-start: 12px;
	}

	.rangeField {
		display: block;
		margin-block-start: 12px;

		> .label {
			display: block;
			color: var(--fg);
			font-size: 0.9em;
		}

		.rangeRow {
			display: flex;
			align-items: center;
			gap: 12px;
			margin-block-start: 8px;
		}

		input[type="range"] {
			flex: 1;
			min-inline-size: 0;
			accent-color: var(--accent);
		}

		.rangeValue {
			min-inline-size: 2em;
			color: var(--fg);
			font-variant-numeric: tabular-nums;
			text-align: end;
		}

		.caption {
			display: block;
			margin-block-start: 6px;
			color: var(--fgTransparentWeak);
			font-size: 0.85em;
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
	}
}
</style>
