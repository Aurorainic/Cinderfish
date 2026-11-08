<!-- Ported from misskey@c05e844:packages/frontend/src/pages/settings/accessibility.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :display-back-button="true" />
		</template>
		<MkSpacer :content-max="700" :margin-min="16" :margin-max="32">
			<div class="_formRoot cf-settings-accessibility">
				<CfInfo class="_formBlock">
					这些选项只影响本设备上的显示方式，不会改变帖子内容，也不需要保存到账号。
				</CfInfo>

				<section class="_formBlock">
					<h2 class="_sectionTitle">动画</h2>
					<CfSwitch v-model="animation" class="_formBlock">
						启用界面动画
						<template #caption>
							关闭后侧栏、弹窗和下拉菜单的过渡动画会被跳过，适合晕动症或低性能设备。
						</template>
					</CfSwitch>
					<CfSwitch v-model="animatedMfm" class="_formBlock">
						播放 MFM 动画
						<template #caption>
							关闭后 <code>$[spin]</code>、<code>$[jelly]</code> 等效果保持静止。
						</template>
					</CfSwitch>
					<CfSwitch v-model="advancedMfm" class="_formBlock">
						启用扩展 MFM
						<template #caption>
							关闭后只渲染基础语法，动画与复杂效果按纯文本显示。
						</template>
					</CfSwitch>
				</section>

				<section class="_formBlock">
					<h2 class="_sectionTitle">排版</h2>
					<CfSwitch v-model="showGapBetweenNotesInTimeline" class="_formBlock">
						时间线中显示帖子间距
						<template #caption>
							关闭后相邻帖子贴在一起，信息密度更高。
						</template>
					</CfSwitch>
					<CfSwitch v-model="squareAvatars" class="_formBlock">
						使用方形头像
					</CfSwitch>
					<CfSwitch v-model="squareCatAvatars" class="_formBlock">
						使用方形猫耳头像
						<template #caption>
							只影响显示：头像裁剪成方形，猫耳装饰仍按原样式绘制。
						</template>
					</CfSwitch>
				</section>

				<CfInfo class="_formBlock">
					模糊效果、原始图片加载、图标集等选项在上游存在，但 Cinderfish 目前没有对应的样式或组件读取它们，
					因此这里不提供会毫无作用的开关。
				</CfInfo>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script setup lang="ts">
// Firefish inherits this page from Misskey. Every switch here is backed by a
// `defaultStore` key that something actually reads:
//   animation / animatedMfm / advancedMfm  -> MFM renderer and page transitions
//   showGapBetweenNotesInTimeline          -> note list spacing
//   squareAvatars / squareCatAvatars        -> avatar renderer
// Keys without a consumer (blur effects, image loading, icon set) are listed in
// the closing CfInfo instead of being drawn as dead controls.
import { computed } from "vue";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfSwitch from "@/components/ui/CfSwitch.vue";
import { defaultStore } from "@/store";

defineOptions({ name: "CfPageSettingsAccessibility" });

type StoreKey = keyof typeof defaultStore.state & string;
type StoreValue<K extends StoreKey> = (typeof defaultStore.state)[K];

function storeRef<K extends StoreKey>(key: K) {
	return computed<StoreValue<K>>({
		get: () => defaultStore.reactiveState[key].value as StoreValue<K>,
		set: (value) => defaultStore.set(key, value),
	});
}

const animation = storeRef("animation");
const animatedMfm = storeRef("animatedMfm");
const advancedMfm = storeRef("advancedMfm");
const showGapBetweenNotesInTimeline = storeRef("showGapBetweenNotesInTimeline");
const squareAvatars = storeRef("squareAvatars");
const squareCatAvatars = storeRef("squareCatAvatars");
</script>
