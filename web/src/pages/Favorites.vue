<!-- Ported from firefish@b425f61:packages/client/src/pages/favorites.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header><MkPageHeader /></template>
		<MkSpacer :content-max="800">
			<CfPagination ref="paginationComponent" :pagination="pagination">
				<template #empty>
					<div class="_fullinfo">
						<i class="ph-bookmark-simple ph-bold emptyIcon"></i>
						<div>{{ t("noNotes") }}</div>
					</div>
				</template>

				<template #default="{ items }">
					<CfDateSeparatedList
						v-slot="{ item: favorite }"
						:items="favorites(items)"
						direction="down"
						class="list"
					>
						<div class="favorite">
							<div class="bar">
								<span class="added">
									<i class="ph-bookmark-simple ph-bold"></i>
									<MkTime :time="favorite.createdAt" mode="detail" />
								</span>
								<button
									class="remove _button"
									:title="t('unfavorite')"
									:aria-label="t('unfavorite')"
									@click="unfavorite(favorite)"
								>
									<i class="ph-x ph-bold"></i>
									<span>{{ t("unfavorite") }}</span>
								</button>
							</div>
							<MkNote
								:key="favorite.note.id"
								:note="favorite.note"
								class="note"
							/>
						</div>
					</CfDateSeparatedList>
				</template>
			</CfPagination>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script lang="ts" setup>
// Firefish renders MkPagination + XNote only. Cinderfish adds one affordance the
// upstream page lacks: a per-row "取消收藏" button, because `notes/favorites/delete`
// is the only way to leave this list (MkNote's bookmark toggle lives in the note
// menu, which the wave-2 note tree does not render on this page).
import { computed, ref } from "vue";
import type { NoteFavorite } from "@/api/types";
import { CfDateSeparatedList, CfPagination } from "@/components/ui";
import type { Paging } from "@/components/ui";
import MkNote from "@/components/MkNote.vue";
import { t } from "@/i18n";
import { apiWithDialog } from "@/os";
import { definePageMetadata } from "@/scripts/page-metadata";

interface FavoritesPaging {
	removeItem: (finder: (item: { id: string }) => boolean) => boolean;
}

const pagination: Paging = {
	endpoint: "i/favorites",
	limit: 10,
};

const paginationComponent = ref<FavoritesPaging | null>(null);

// CfPagination is generic over `{ id: string }` when instantiated from a
// template, so the slot payload is re-typed here.
function favorites(items: readonly { id: string }[]): NoteFavorite[] {
	return items as NoteFavorite[];
}

async function unfavorite(favorite: NoteFavorite): Promise<void> {
	try {
		await apiWithDialog("notes/favorites/delete", {
			noteId: favorite.noteId,
		});
	} catch {
		// apiWithDialog already showed the error dialog; the row stays.
		return;
	}
	// The cursor of i/favorites is the favorite row id, so removal by row id
	// keeps the remaining pagination state valid.
	paginationComponent.value?.removeItem((item) => item.id === favorite.id);
}

definePageMetadata(
	computed(() => ({
		title: t("favorites"),
		icon: "ph-bookmark-simple ph-bold ph-lg",
	})),
);
</script>

<style lang="scss" scoped>
.emptyIcon {
	display: block;
	margin: 0 auto 8px auto;
	font-size: 32px;
	opacity: 0.5;
}

.list {
	> .favorite {
		margin-block: var(--margin);
		background: var(--panel);
		border-radius: var(--radius);

		> .bar {
			display: flex;
			align-items: center;
			justify-content: space-between;
			padding: 8px 16px;
			font-size: 0.85em;
			color: var(--fgTransparentWeak);
			border-bottom: solid 1px var(--divider);

			> .added {
				display: inline-flex;
				align-items: center;
				gap: 6px;
			}

			> .remove {
				display: inline-flex;
				align-items: center;
				gap: 4px;
				padding: 4px 8px;
				border-radius: var(--radius);
				color: inherit;

				&:hover {
					color: var(--accent);
					background: var(--buttonBg);
				}
			}
		}

		> .note {
			background: none;
		}
	}
}
</style>
