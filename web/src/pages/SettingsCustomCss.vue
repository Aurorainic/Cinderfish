<!-- Ported from firefish@b425f61:packages/client/src/pages/settings/custom-css.vue (AGPL-3.0-only) -->
<template>
	<div class="_formRoot cf-settings-custom-css">
		<CfInfo warn class="_formBlock">{{ t("customCssWarn") }}</CfInfo>

		<CfTextarea
			v-model="draft"
			class="_formBlock cssField"
			code
			tall
			:spellcheck="false"
		>
			<template #label>CSS</template>
			<template #caption>
				{{ draft.length }} 字符 · 留空并保存即清除自定义样式
			</template>
		</CfTextarea>

		<div class="actions _formBlock">
			<CfButton primary :disabled="!dirty" @click="save()">
				<i class="ph-floppy-disk ph-bold ph-lg"></i>
				{{ t("save") }}
			</CfButton>
			<CfButton inline :disabled="!dirty" @click="revert()">
				还原
			</CfButton>
			<CfButton inline danger :disabled="saved === ''" @click="clear()">
				{{ t("clear") }}
			</CfButton>
			<span v-if="justSaved" class="status">{{ t("saved") }}</span>
		</div>

		<section class="_formBlock">
			<h2 class="sectionTitle">生效方式</h2>
			<ul class="notes">
				<li>
					保存后样式<strong>立即生效</strong>，不需要刷新页面：值写入
					<code>defaultStore.state.customCss</code>，
					<code>composables/use-custom-css.ts</code> 维护一个
					<code>&lt;style id="cinderfish-custom-css"&gt;</code> 并跟随该值重写内容。
				</li>
				<li>
					样式按账号保存（存储作用域为 <code>account</code>）：登录状态下经
					<code>i/registry/set</code> 同步到本账号，同一浏览器的其他账号互不影响；
					未登录时只保留本机缓存。
				</li>
				<li>
					浏览器会静默忽略语法错误的规则，所以这里不做校验。样式写坏导致界面错乱时，
					回到本页清空并保存即可恢复。
				</li>
				<li>
					上游 Firefish 把 CSS 存在裸 <code>localStorage</code> 键里并要求整页刷新；
					这里改为账号级存储 + 实时应用，能力不减少。
				</li>
			</ul>
		</section>
	</div>
</template>

<script setup lang="ts">
// Firefish's version wrote `localStorage.customCss` and reloaded the page. The
// CSS itself is still applied by the browser only, so nothing here talks to the
// API directly; persistence goes through the account-scoped store entry.
import { computed, ref, watch } from "vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfTextarea from "@/components/ui/CfTextarea.vue";
import { t } from "@/i18n";
import { definePageMetadata } from "@/scripts/page-metadata";
import { defaultStore } from "@/store";

defineOptions({ name: "CfPageSettingsCustomCss" });

const saved = computed(() => defaultStore.reactiveState.customCss.value);
const draft = ref(saved.value);
const justSaved = ref(false);
const touched = ref(false);

let savedTimer: number | null = null;

const dirty = computed(() => draft.value !== saved.value);

watch(draft, (value) => {
	if (value !== saved.value) touched.value = true;
});

// The registry answers after the first paint; adopt its value unless the user
// is already editing.
watch(saved, (value) => {
	if (!touched.value) draft.value = value;
});

function save(): void {
	defaultStore.set("customCss", draft.value);
	touched.value = false;
	justSaved.value = true;
	if (savedTimer != null) window.clearTimeout(savedTimer);
	savedTimer = window.setTimeout(() => {
		justSaved.value = false;
	}, 2000);
}

function revert(): void {
	draft.value = saved.value;
	touched.value = false;
}

function clear(): void {
	draft.value = "";
	save();
}

definePageMetadata({
	title: t("customCss"),
	icon: "ph-code ph-bold ph-lg",
});
</script>

<style lang="scss" scoped>
.cf-settings-custom-css {
	.sectionTitle {
		margin: 0 0 4px;
		color: var(--fg);
		font-size: 1em;
		opacity: 0.85;
	}

	.cssField :deep(textarea) {
		min-block-size: 360px;
		tab-size: 2;
	}

	.actions {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 8px;
	}

	.status {
		color: var(--fgTransparentWeak);
		font-size: 0.85em;
	}

	.notes {
		margin: 0;
		padding-inline-start: 1.2em;
		color: var(--fgTransparentWeak);
		font-size: 0.9em;

		> li + li {
			margin-block-start: 6px;
		}

		code {
			color: var(--fg);
		}
	}
}
</style>
