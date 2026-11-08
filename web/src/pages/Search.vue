<!-- Ported from firefish@b425f61:packages/client/src/pages/search.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :display-back-button="true" />
		</template>
		<MkSpacer :content-max="800" :margin-min="16" :margin-max="32">
			<form class="searchForm" @submit.prevent="submit">
				<CfInput
					v-model="draft"
					class="input"
					autocomplete="off"
					:spellcheck="false"
					placeholder="搜索帖子正文或用户"
				>
					<template #prefix>
						<i class="ph-magnifying-glass ph-bold"></i>
					</template>
				</CfInput>
				<CfButton type="submit" primary :disabled="draft.trim() === ''">
					搜索
				</CfButton>
			</form>

			<CfInfo v-if="query === ''" class="notice">
				输入关键词后回车。帖子只在正文里做不区分大小写的包含匹配；用户按用户名、昵称与简介匹配。
				可见范围遵循你的权限：私密帖子、静音与屏蔽的账号不会出现在结果里。
			</CfInfo>

			<template v-else>
				<div class="headline">
					<span class="queryText"
						>“<strong>{{ query }}</strong>” 的结果</span
					>
					<CfRadios v-model="tab" class="tabs">
						<CfRadio value="notes">帖子</CfRadio>
						<CfRadio value="users">用户</CfRadio>
					</CfRadios>
				</div>

				<MkNotes
					v-if="tab === 'notes'"
					:key="`notes:${query}`"
					:pagination="notesPagination"
				/>
				<MkUserList
					v-else
					:key="`users:${query}`"
					:pagination="usersPagination"
				/>
			</template>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script lang="ts" setup>
// Firefish's search page is a swiper of `notes/search` and `users/search`.
// Cinderfish now implements both (ILIKE fallback, no search engine), so the
// page is a real tabbed result view. The swiper is dropped: tabs render one
// pane, which also keeps the page usable with a keyboard.
//
// Each pane is keyed by the query so a new search remounts the paginator
// instead of mixing results from two queries in one list.
import { computed, ref, watch } from "vue";
import { useRoute, useRouter } from "vue-router";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfInput from "@/components/ui/CfInput.vue";
import CfRadio from "@/components/ui/CfRadio.vue";
import CfRadios from "@/components/ui/CfRadios.vue";
import type { Paging } from "@/components/ui";
import MkNotes from "@/components/MkNotes.vue";
import MkUserList from "@/components/MkUserList.vue";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";
import { t } from "@/i18n";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageSearch" });

const route = useRoute();
const router = useRouter();

const query = computed(() => {
	const raw = route.query.q;
	if (typeof raw === "string") return raw.trim();
	if (Array.isArray(raw)) return raw.join(" ").trim();
	return "";
});

const draft = ref(query.value);
const tab = ref<"notes" | "users">("notes");

const notesPagination = computed<Paging>(() => ({
	endpoint: "notes/search",
	limit: 10,
	params: { query: query.value },
}));

const usersPagination = computed<Paging>(() => ({
	endpoint: "users/search",
	limit: 20,
	params: { query: query.value, origin: "combined" },
}));

function submit(): void {
	const q = draft.value.trim();
	if (q === "") return;
	void router.replace({ path: "/search", query: { q } });
}

watch(query, (value) => {
	draft.value = value;
});

definePageMetadata(
	computed(() => ({
		title:
			query.value.length > 0
				? t("searchWith", { q: query.value })
				: t("search"),
		icon: "ph-magnifying-glass ph-bold ph-lg",
	})),
);
</script>

<style lang="scss" scoped>
.searchForm {
	display: flex;
	gap: 8px;
	align-items: flex-end;
	margin-bottom: var(--margin);

	.input {
		flex: 1;
		min-width: 0;
	}
}

.notice {
	margin-bottom: var(--margin);
}

.headline {
	display: flex;
	align-items: center;
	justify-content: space-between;
	flex-wrap: wrap;
	gap: 10px;
	margin-bottom: var(--margin);

	.queryText {
		font-size: 0.92em;
		color: var(--fgTransparentWeak);
		overflow-wrap: anywhere;
	}

	.tabs {
		margin: 0;
	}
}
</style>
