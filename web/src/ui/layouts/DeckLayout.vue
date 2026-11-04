<template>
	<div class="mk-deck" :class="{ isMobile }">
		<XSidebar v-if="!isMobile" />

		<div class="main">
			<div class="columns" :class="align">
				<template v-for="(ids, index) in layout" :key="index">
					<section v-if="ids.length > 1" class="folder column">
						<article
							v-for="id in ids"
							:key="id"
							class="column-stack"
						>
							<DeckColumn :column="findColumn(id)" />
						</article>
					</section>
					<article v-else class="column" :key="ids[0]">
						<DeckColumn :column="findColumn(ids[0])" />
					</article>
				</template>
				<div v-if="layout.length === 0" class="intro _panel">
					<p>TODO: port ui/deck.vue column introduction</p>
					<RouterLink to="/settings/deck" class="link">
						{{ t("_deck.addColumn") }}
					</RouterLink>
				</div>
			</div>
		</div>
	</div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { t } from "@/i18n";
import { deckStore, type Column } from "@/store/deck";
import { useLayout } from "@/router/layout";
import XSidebar from "./XSidebar.vue";
import DeckColumn from "./DeckColumn.vue";

const { isMobile } = useLayout();
const layout = computed(() => deckStore.state.layout);
const align = computed(() => deckStore.state.columnAlign);

function findColumn(id: string): Column | undefined {
	return deckStore.state.columns.find((column) => column.id === id);
}
</script>

<style scoped>
.mk-deck {
	display: flex;
	min-block-size: 100dvh;
	background: var(--bg);

	> .main {
		flex: 1;
		min-inline-size: 0;
		display: flex;
		flex-direction: column;
	}

	> .main > .columns {
		display: flex;
		gap: 8px;
		padding: 8px;
		overflow-x: auto;
		align-items: flex-start;

		&.left {
			justify-content: flex-start;
		}

		&.center {
			justify-content: center;
		}

		&.right {
			justify-content: flex-end;
		}
	}

	.column {
		flex: 0 0 auto;
		inline-size: 350px;
		max-block-size: calc(100dvh - 32px);
		overflow: auto;
		background: var(--panel);
		border-radius: var(--radius, 12px);
	}

	.column-stack {
		inline-size: 350px;
		border-radius: var(--radius, 12px);
		overflow: clip;
	}

	.folder {
		inline-size: 350px;
		display: flex;
		flex-direction: column;
		gap: 4px;
	}

	.intro {
		padding: 32px;
		text-align: center;
		color: var(--fgTransparentWeak);
	}
}

.link {
	color: var(--accent);
}
</style>
