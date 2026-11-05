<!-- Ported from firefish@b425f61:packages/client/src/pages/settings/other.vue (AGPL-3.0-only) -->
<template>
	<div class="_formRoot cf-settings-other">
		<section class="_formBlock">
			<h2 class="sectionTitle">{{ t("otherSettings") }}</h2>

			<div class="kvRow">
				<span class="kvKey">{{ t("showFeaturedNotesInTimeline") }}</span>
				<span class="kvValue">
					{{ me?.injectFeaturedNote ? t("yes") : t("no") }}
				</span>
			</div>
			<CfInfo class="note">
				Firefish 用 <code>i/update { injectFeaturedNote }</code> 切换这一项，
				而 Cinderfish 的 <code>i/update</code> 只接受 name / description / avatarId /
				followersVisibility / followingVisibility，所以这里只读显示，修改<strong>未提供</strong>。
			</CfInfo>
		</section>

		<section class="_formBlock">
			<h2 class="sectionTitle">{{ t("clientSettings") }}</h2>

			<CfSwitch v-model="showFullAcct" class="_formBlock">
				显示完整的 @user@host
				<template #caption>
					关闭时只显示 @user；远程用户的服务器名会被省略。
				</template>
			</CfSwitch>

			<CfSwitch v-model="postFormWithHashtags" class="_formBlock">
				发帖时自动附带标签
				<template #caption>在发帖框里预填下面这些标签。</template>
			</CfSwitch>
			<CfInput
				v-if="postFormWithHashtags"
				v-model="postFormHashtags"
				class="_formBlock"
				placeholder="#cinderfish #fediverse"
			>
				<template #label>自动标签</template>
				<template #caption>以空格分隔；发帖时会追加到正文末尾。</template>
			</CfInput>

			<CfInput v-model="pastedFileName" class="_formBlock">
				<template #label>粘贴文件名模板</template>
				<template #caption>
					粘贴图片上传时的文件名，使用 <code>yyyy</code> <code>MM</code>
					<code>dd</code> <code>HH</code> <code>mm</code> <code>ss</code> 与
					<code>[&#123;&#123; number &#125;&#125;]</code> 占位符。
				</template>
			</CfInput>

			<CfSwitch v-model="woozyMode" class="_formBlock">
				摇晃模式
				<template #caption>发送 ⭐ 回应时让整个页面摇晃一下（彩蛋）。</template>
			</CfSwitch>
		</section>

		<section class="_formBlock">
			<h2 class="sectionTitle">{{ t("accountInfo") }}</h2>
			<div class="_formLinks">
				<CfA class="linkRow" to="/settings/account-info">
					<i class="ph-identification-card ph-bold ph-lg"></i>
					{{ t("accountInfo") }}
				</CfA>
				<CfA class="linkRow" to="/registry">
					<i class="ph-gear-six ph-bold ph-lg"></i>
					{{ t("registry") }}
				</CfA>
				<CfA class="linkRow danger" to="/settings/delete-account">
					<i class="ph-warning ph-bold ph-lg"></i>
					{{ t("closeAccount") }}
				</CfA>
			</div>
			<CfInfo class="note">
				<code>registry</code> 与注销账号页仍由各自的移植任务负责；这里只做入口聚合。
			</CfInfo>
		</section>
	</div>
</template>

<script lang="ts" setup>
// Firefish's "other" page is a single server-side switch plus three links. The
// switch (`injectFeaturedNote`) is not writable here, so it is read-only. The
// client-side preferences that no other settings page owns are aggregated in the
// second section: they are all real `defaultStore` keys with live consumers
// (MkAcct, MkPostForm, MkStarButton), unlike the upstream page which keeps them
// scattered or drop them entirely.
import { computed } from "vue";
import { me } from "@/account";
import CfA from "@/components/ui/CfA.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfInput from "@/components/ui/CfInput.vue";
import CfSwitch from "@/components/ui/CfSwitch.vue";
import { t } from "@/i18n";
import { definePageMetadata } from "@/scripts/page-metadata";
import { defaultStore } from "@/store";

defineOptions({ name: "CfPageSettingsOther" });

function storeRef<K extends keyof typeof defaultStore.state & string>(key: K) {
	return computed<(typeof defaultStore.state)[K]>({
		get: () => defaultStore.reactiveState[key].value as (typeof defaultStore.state)[K],
		set: (value) => defaultStore.set(key, value),
	});
}

const showFullAcct = storeRef("showFullAcct");
const postFormWithHashtags = storeRef("postFormWithHashtags");
const postFormHashtags = storeRef("postFormHashtags");
const pastedFileName = storeRef("pastedFileName");
const woozyMode = storeRef("woozyMode");

definePageMetadata({
	title: t("other"),
	icon: "ph-dots-three-outline ph-bold ph-lg",
});
</script>

<style lang="scss" scoped>
.cf-settings-other {
	.sectionTitle {
		margin: 0 0 4px;
		color: var(--fg);
		font-size: 1em;
		opacity: 0.85;
	}

	.kvRow {
		display: flex;
		align-items: baseline;
		gap: 12px;
		margin-block-start: 12px;

		> .kvKey {
			flex: 0 0 35%;
			color: var(--fgTransparentWeak);
			font-size: 0.85em;
		}

		> .kvValue {
			font-size: 0.9em;
		}
	}

	.note {
		margin-block-start: 12px;
		font-size: 0.85em;

		code {
			word-break: break-all;
		}
	}

	._formLinks {
		margin-block-start: 12px;
	}

	.linkRow {
		display: flex;
		align-items: center;
		gap: 10px;
		padding: 12px 16px;
		border-radius: var(--radius);
		background: var(--panel);
		color: var(--fg);
		text-decoration: none;

		&:hover {
			background: var(--panelHighlight);
		}

		&.danger {
			color: var(--error);
		}
	}
}
</style>
