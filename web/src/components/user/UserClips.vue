<!-- Ported from firefish@b425f61:packages/client/src/pages/user/clips.vue (AGPL-3.0-only) -->
<template>
	<div class="cf-user-clips">
		<CfPagination :pagination="pagination" class="list">
			<template #empty>
				<CfInfo>{{ t("clipsDesc") }}</CfInfo>
			</template>
			<template #default="{ items }">
				<CfA
					v-for="item in asClips(items)"
					:key="item.id"
					:to="`/clips/${item.id}`"
					class="item _panel"
				>
					<div class="body">
						<div class="name">{{ item.name }}</div>
						<div v-if="item.description" class="description">
							{{ item.description }}
						</div>
						<div class="meta">
							<span v-if="item.notesCount != null">
								{{ item.notesCount }} {{ t("notes") }}
							</span>
							<span>
								{{ item.favoritedCount }} {{ t("favorites") }}
							</span>
						</div>
					</div>
					<i class="ph-caret-right ph-bold"></i>
				</CfA>
			</template>
		</CfPagination>
	</div>
</template>

<script setup lang="ts">
// `users/clips` returns public clips only and pages with the standard cursor,
// so the Firefish MkPagination list maps straight onto CfPagination. Unlike
// the owner's view, `notesCount` is normally absent here; the favourited
// count is always packed.
import { computed } from "vue";
import { ENDPOINTS } from "@/api/endpoints";
import type { Clip, UserDetailed } from "@/api/types";
import type { Paging } from "@/components/ui";
import CfA from "@/components/ui/CfA.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfPagination from "@/components/ui/CfPagination.vue";
import { t } from "@/i18n";

const props = defineProps<{
	user: UserDetailed;
}>();

const pagination = computed<Paging>(() => ({
	endpoint: ENDPOINTS.usersClips,
	limit: 20,
	params: { userId: props.user.id },
}));

/** CfPagination's slot is generic over `{ id: string }`; this endpoint is always clips. */
function asClips(items: { id: string }[]): Clip[] {
	return items as Clip[];
}
</script>

<style lang="scss" scoped>
.cf-user-clips {
	> .list {
		> .item {
			display: flex;
			align-items: center;
			gap: 12px;
			padding: 16px;
			margin-block-end: var(--margin);

			&:hover {
				border-color: var(--accent);
				text-decoration: none;
			}

			> .body {
				flex: 1;
				min-width: 0;
			}

			.name {
				font-weight: bold;
			}

			.description {
				margin-block-start: 8px;
				padding-block-start: 8px;
				border-block-start: solid 0.5px var(--divider);
				white-space: pre-wrap;
				overflow-wrap: anywhere;
			}

			.meta {
				display: flex;
				align-items: center;
				gap: 8px;
				margin-block-start: 8px;
				font-size: 0.82em;
				color: var(--fgTransparentWeak);
			}

			> i {
				flex: 0 0 auto;
				color: var(--fgTransparentWeak);
			}
		}
	}
}
</style>
