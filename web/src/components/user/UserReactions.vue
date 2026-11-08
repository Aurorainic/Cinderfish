<!-- Ported from firefish@b425f61:packages/client/src/pages/user/reactions.vue (AGPL-3.0-only) -->
<template>
	<div class="reactions _panel">
		<header class="header">
			<i class="ph-smiley ph-bold ph-lg"></i>
			<span>{{ t("reaction") }}</span>
		</header>
		<div class="body">
			<CfLoading v-if="fetching && items.length === 0" />
			<CfInfo v-else-if="notice !== ''" warn class="notice">{{ notice }}</CfInfo>
			<CfInfo v-else-if="items.length === 0" class="notice">
				这个账号还没有公开的回应记录。
			</CfInfo>
			<template v-else>
				<div v-for="item in items" :key="item.id" class="item">
					<div class="item-header">
						<MkAvatar class="avatar" :user="user" />
						<MkReactionIcon
							class="reaction"
							:reaction="item.type"
							:custom-emojis="item.note.emojis"
							no-style
						/>
						<MkTime :time="item.createdAt" class="created-at" />
					</div>
					<MkNote :key="item.id" :note="item.note" />
				</div>
				<div v-if="hasMore" class="more">
					<CfButton :disabled="fetching" @click="loadMore">
						{{ fetching ? "加载中…" : "加载更多" }}
					</CfButton>
				</div>
			</template>
		</div>
	</div>
</template>

<script lang="ts" setup>
// Firefish pages `users/reactions` with MkPagination. The port fetches the same
// endpoint and appends pages with the `untilId` cursor.
import { onMounted, ref, watch } from "vue";
import type { NoteReactionWithNote, UserDetailed } from "@/api/types";
import { api } from "@/api/client";
import { CfButton, CfInfo, CfLoading } from "@/components/ui";
import MkNote from "@/components/MkNote.vue";
import MkReactionIcon from "@/components/MkReactionIcon.vue";
import MkTime from "@/components/global/MkTime.vue";
import { t } from "@/i18n";

const props = defineProps<{
	user: UserDetailed;
}>();

const PAGE_SIZE = 20;

const items = ref<NoteReactionWithNote[]>([]);
const fetching = ref(false);
const hasMore = ref(false);
const notice = ref("");

function load(reset: boolean): void {
	fetching.value = true;
	notice.value = "";
	const untilId = reset || items.value.length === 0
		? undefined
		: items.value[items.value.length - 1].id;
	api<NoteReactionWithNote[]>("users/reactions", {
		userId: props.user.id,
		limit: PAGE_SIZE,
		...(untilId == null ? {} : { untilId }),
	})
		.then((page) => {
			items.value = reset ? page : [...items.value, ...page];
			hasMore.value = page.length === PAGE_SIZE;
		})
		.catch((error: unknown) => {
			const code =
				error != null && typeof error === "object" && "code" in error
					? (error as { code?: unknown }).code
					: null;
			if (code === "REACTIONS_NOT_PUBLIC") {
				notice.value = "该用户没有公开回应记录（可在隐私设置中打开「公开回应」）。";
			} else if (code === "IS_REMOTE_USER") {
				notice.value = "远端账号的回应记录不在本实例保存，无法列出。";
			} else {
				notice.value = "回应记录加载失败，请稍后重试。";
			}
			items.value = [];
			hasMore.value = false;
		})
		.finally(() => {
			fetching.value = false;
		});
}

function loadMore(): void {
	load(false);
}

onMounted(() => load(true));
watch(() => props.user.id, () => load(true));
</script>

<style lang="scss" scoped>
.reactions {
	overflow: hidden;

	> .header {
		display: flex;
		align-items: center;
		gap: 0.5em;
		padding: 12px 16px;
		border-bottom: solid 0.5px var(--divider);
		font-weight: bold;
	}

	> .body {
		padding: 16px;
	}

	.notice {
		font-size: 0.85em;
	}

	.item {
		margin-bottom: 16px;

		> .item-header {
			display: flex;
			align-items: center;
			padding: 8px 16px;
			margin-bottom: 8px;
			border-bottom: solid 2px var(--divider);

			> .avatar {
				width: 24px;
				height: 24px;
				margin-right: 8px;
			}

			> .reaction {
				margin-right: 8px;
			}

			> .created-at {
				font-size: 0.8em;
				color: var(--fgTransparentWeak);
			}
		}
	}

	.more {
		display: flex;
		justify-content: center;
	}
}
</style>
