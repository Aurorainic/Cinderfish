<!-- Ported from firefish@b425f61:packages/client/src/pages/admin/custom-css.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :display-back-button="true" />
		</template>
		<MkSpacer :content-max="900" :margin-min="16" :margin-max="32">
			<div v-if="!admin" class="cf-admin-custom-css">
				<CfInfo warn>
					需要管理员权限，无法编辑实例样式。
					<CfA to="/" class="_link">返回首页</CfA>
				</CfInfo>
			</div>

			<div v-else class="cf-admin-custom-css">
				<CfInfo warn class="_formBlock">
					这里的样式会作用于<strong>所有访客的每个页面</strong>，包括未登录的访客。
					改动前请先在浏览器开发者工具里试好。
				</CfInfo>

				<CfLoading v-if="loading" />
				<CfInfo v-else-if="loadError != null" warn>
					{{ loadError }}
					<CfButton inline @click="load">{{ t("retry") }}</CfButton>
				</CfInfo>

				<template v-else>
					<CfTextarea
						v-model="draft"
						class="_formBlock cssField"
						code
						tall
						:spellcheck="false"
					>
						<template #label>实例 CSS</template>
						<template #caption>
							{{ draft.length }} 字符（上限 100 KiB）· 留空并保存即清除
						</template>
					</CfTextarea>

					<div class="actions _formBlock">
						<CfButton primary :disabled="busy || !dirty" @click="save()">
							<i class="ph-floppy-disk ph-bold ph-lg"></i>
							{{ t("save") }}
						</CfButton>
						<CfButton :disabled="busy || !dirty" @click="revert()">
							还原
						</CfButton>
						<CfButton
							v-if="saved !== ''"
							danger
							:disabled="busy"
							@click="clear()"
						>
							{{ t("clear") }}
						</CfButton>
						<span v-if="notice != null" class="status">{{ notice }}</span>
					</div>
					<CfInfo v-if="errorText != null" warn class="_formBlock">
						{{ errorText }}
					</CfInfo>

					<section class="_formBlock">
						<h2 class="sectionTitle">生效方式</h2>
						<ul class="notes">
							<li>
								保存在 <code>instance_meta</code> 的 <code>customCss</code> 键，
								经 <code>admin/update-meta</code> 写入，随 <code>meta</code> 下发。
							</li>
							<li>
								前端在
								<code>&lt;style id="cinderfish-instance-custom-css"&gt;</code>
								里注入，排在账号级自定义样式之前，所以每个账号仍可以覆盖它。
							</li>
							<li>刷新页面后生效；语法错误会被浏览器静默忽略。</li>
						</ul>
					</section>
				</template>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script lang="ts" setup>
// Firefish's admin/custom-css edits instance-wide style. Cinderfish stores it
// in instance_meta and injects it from the meta payload, so the page is a
// plain editor over admin/update-meta. The account-level sheet lives at
// /settings/custom-css and is applied after this one.
import { computed, onMounted, ref } from "vue";
import { isAdmin } from "@/account";
import { ApiError, api } from "@/api/client";
import type { AdminInstanceMeta } from "@/api/types";
import CfA from "@/components/ui/CfA.vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import CfTextarea from "@/components/ui/CfTextarea.vue";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";
import { instance, updateInstance } from "@/instance";
import { t } from "@/i18n";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageAdminCustomCss" });

const admin = computed<boolean>(() => isAdmin());
const loading = ref(false);
const busy = ref(false);
const loadError = ref<string | null>(null);
const errorText = ref<string | null>(null);
const notice = ref<string | null>(null);
const saved = ref("");
const draft = ref("");

const dirty = computed(() => draft.value !== saved.value);

function errorMessage(cause: unknown): string {
	if (cause instanceof ApiError) return cause.message;
	return String(cause);
}

async function load(): Promise<void> {
	if (!admin.value) return;
	loading.value = true;
	loadError.value = null;
	try {
		const meta: AdminInstanceMeta = await api("admin/meta", {});
		saved.value = meta.customCss ?? "";
		draft.value = saved.value;
	} catch (cause) {
		loadError.value = errorMessage(cause);
	} finally {
		loading.value = false;
	}
}

async function save(): Promise<void> {
	if (busy.value || !dirty.value) return;
	busy.value = true;
	errorText.value = null;
	notice.value = null;
	try {
		const meta: AdminInstanceMeta = await api("admin/update-meta", {
			customCss: draft.value,
		});
		saved.value = meta.customCss ?? "";
		draft.value = saved.value;
		// The instance store carries the sheet to every page; refresh it so the
		// admin sees the result without a reload.
		instance.customCss = saved.value;
		void updateInstance(true);
		notice.value = t("saved");
	} catch (cause) {
		errorText.value = errorMessage(cause);
	} finally {
		busy.value = false;
	}
}

function revert(): void {
	draft.value = saved.value;
	errorText.value = null;
	notice.value = null;
}

async function clear(): Promise<void> {
	draft.value = "";
	await save();
}

onMounted(() => {
	void load();
});

definePageMetadata({ title: "实例 CSS", icon: "ph-code ph-bold ph-lg" });
</script>

<style lang="scss" scoped>
.cf-admin-custom-css {
	.cssField {
		font-family: monospace;
	}

	.actions {
		display: flex;
		align-items: center;
		gap: 8px;
		flex-wrap: wrap;
	}

	.status {
		font-size: 0.85em;
		opacity: 0.7;
	}

	.notes {
		margin: 0;
		padding-inline-start: 20px;
		font-size: 0.85em;
		line-height: 1.7;
		opacity: 0.8;

		code {
			font-family: monospace;
			background: var(--bg);
			padding: 1px 4px;
			border-radius: 4px;
		}
	}
}
</style>
