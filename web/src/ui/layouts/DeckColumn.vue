<template>
	<section class="deck-column">
		<header class="header">
			<span class="title">{{ column?.name ?? column?.type ?? "?" }}</span>
			<RouterLink to="/settings/deck" class="settings">⚙</RouterLink>
		</header>
		<div class="body">
			<router-view v-if="column?.type === 'main'" />
			<div v-else class="placeholder">
				TODO: port deck column <code>{{ column?.type ?? "unknown" }}</code>
			</div>
		</div>
	</section>
</template>

<script setup lang="ts">
import type { Column } from "@/store/deck";

defineProps<{ column?: Column }>();
</script>

<style scoped>
.deck-column {
	display: flex;
	flex-direction: column;
	block-size: 100%;
	background: var(--panel);
	border-radius: var(--radius, 12px);
	overflow: clip;

	> .header {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 8px 12px;
		background: var(--panelHeaderBg);
		border-block-end: solid 0.5px var(--divider);

		> .title {
			flex: 1;
			font-weight: bold;
			font-size: 0.9em;
			color: var(--panelHeaderFg);
			overflow: hidden;
			text-overflow: ellipsis;
			white-space: nowrap;
		}

		> .settings {
			color: var(--fgTransparentWeak);
		}
	}

	> .body {
		flex: 1;
		min-block-size: 0;
		overflow: auto;
	}

	.placeholder {
		padding: 32px 16px;
		text-align: center;
		color: var(--fgTransparentWeak);
	}
}
</style>
