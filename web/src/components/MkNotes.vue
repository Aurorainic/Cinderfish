<!-- Ported from firefish@b425f61:packages/client/src/components/MkNotes.vue (AGPL-3.0-only) -->
<template>
	<CfPagination ref="pagingComponent" :pagination="pagination">
		<template #empty>
			<div class="_fullinfo">
				<div>{{ t("noNotes") }}</div>
			</div>
		</template>

		<template #default="{ items: notes }">
			<div class="giivymft" :class="{ noGap }" ref="tlEl">
				<CfDateSeparatedList
					v-slot="{ item: note }"
					:items="asNotes(notes)"
					:direction="pagination.reversed ? 'up' : 'down'"
					:reversed="pagination.reversed"
					:no-gap="noGap"
					:ad="true"
					class="notes"
				>
					<MkNote :key="note.id" class="qtqtichx" :note="note" />
				</CfDateSeparatedList>
			</div>
		</template>
	</CfPagination>
</template>

<script lang="ts" setup>
// Degradations vs Firefish:
//   * the empty state drops the /static-assets/badges/info.svg illustration
//     (that asset is not part of this port);
//   * scrollTop() uses element scrolling instead of scripts/scroll.ts.
// pagingComponent is exposed for MkTimeline, which prepends streamed notes: it
// is the CfPagination instance (items / prepend / append / reload / refresh).
import { ref } from "vue";
import type { NoteDetailed } from "@/api/types";
import type { Paging } from "@/components/ui";
import CfPagination from "@/components/ui/CfPagination.vue";
import CfDateSeparatedList from "@/components/ui/CfDateSeparatedList.vue";
import MkNote from "@/components/MkNote.vue";
import { t } from "@/i18n";

interface NotesPagingComponent {
	items: NoteDetailed[];
	prepend: (...notes: NoteDetailed[]) => void;
	append: (...notes: NoteDetailed[]) => void;
	reload: () => Promise<NoteDetailed[]>;
	refresh: () => Promise<void>;
}

const tlEl = ref<HTMLElement | null>(null);

defineProps<{
	pagination: Paging;
	noGap?: boolean;
}>();

const pagingComponent = ref<NotesPagingComponent | null>(null);

/**
 * CfPagination is generic and its slot items fall back to `{ id: string }`;
 * the list endpoint is always a notes endpoint, so this narrows back for the
 * note components.
 */
function asNotes(items: { id: string }[]): NoteDetailed[] {
	return items as NoteDetailed[];
}

function scrollTop(): void {
	tlEl.value?.scrollTo({ top: 0, behavior: "smooth" });
}

defineExpose({
	pagingComponent,
	scrollTop,
});
</script>

<style lang="scss" scoped>
.giivymft {
	&.noGap {
		> .notes {
			background: var(--panel) !important;
			border-radius: var(--radius);
		}
	}
	&:not(.noGap) {
		> .notes {
			.qtqtichx {
				background: var(--panel);
				border-radius: var(--radius);
			}
		}
	}
}
</style>
