<!-- Ported from firefish@b425f61:packages/client/src/components/MkSuperMenu.vue (AGPL-3.0-only) -->
<template>
	<nav class="rrevdjwu" :class="{ grid }">
		<section v-for="(group, gi) in def" :key="gi" class="group">
			<div v-if="group.title" class="title">{{ group.title }}</div>

			<div class="items" role="group" :aria-label="group.title">
				<template v-for="item in group.items" :key="item.to ?? item.href ?? item.text">
					<a
						v-if="item.type === 'a'"
						:href="item.href"
						:target="item.target"
						class="_button item"
						:class="{ danger: item.danger, active: item.active }"
						@click="item.action?.($event)"
					>
						<i v-if="item.icon" class="icon ph-fw ph-lg" :class="item.icon"></i>
						<span class="text">{{ item.text }}</span>
					</a>
					<button
						v-else-if="item.type === 'button'"
						class="_button item"
						:class="{ danger: item.danger, active: item.active }"
						:disabled="item.active"
						@click="item.action?.($event)"
					>
						<i v-if="item.icon" class="icon ph-fw ph-lg" :class="item.icon"></i>
						<span class="text">{{ item.text }}</span>
					</button>
					<MkA
						v-else
						:to="item.to ?? '#'"
						class="_button item"
						:class="{ danger: item.danger, active: item.active }"
					>
						<i v-if="item.icon" class="icon ph-fw ph-lg" :class="item.icon"></i>
						<span class="text">{{ item.text }}</span>
					</MkA>
				</template>
			</div>
		</section>
	</nav>
</template>

<script lang="ts">
// Exported from the plain <script> block so consumers can type their menu
// arrays; the component itself lives in <script setup> below.
export interface MkSuperMenuItem {
	type?: "a" | "button";
	icon?: string;
	text: string;
	/** In-app route, used by MkA when `type` is neither "a" nor "button". */
	to?: string;
	href?: string;
	target?: string;
	active?: boolean;
	danger?: boolean;
	action?: (event: MouseEvent) => void;
}

export interface MkSuperMenuGroup {
	title?: string;
	items: MkSuperMenuItem[];
}
</script>

<script setup lang="ts">
// Wave-3 contract §3.4: the wide container keeps Firefish's vertical list, the
// narrow one collapses to a single horizontally scrolling pill row instead of
// upstream's card grid (a 2-column grid clips labels in the ~350px a phone
// leaves for the settings column). No `os` dependency, as required.
import MkA from "@/components/global/MkA.vue";

withDefaults(
	defineProps<{
		def: MkSuperMenuGroup[];
		/** Narrow layout: one scrollable row of pills. */
		grid?: boolean;
	}>(),
	{
		grid: false,
	},
);
</script>

<style lang="scss" scoped>
.rrevdjwu {
	&:not(.grid) {
		position: sticky;
		inset-block-start: 0;
		padding-block-start: 8px;

		> .group {
			& + .group {
				margin-top: 16px;
				padding-top: 16px;
				border-top: solid 0.5px var(--divider);
			}

			> .title {
				color: var(--fgTransparentWeak);
				margin: 0 12px 8px;
				font-size: 0.9em;
			}
		}
	}

	> .group {
		& + .group {
			margin-top: 12px;
		}

		> .title {
			color: var(--fgTransparentWeak);
			margin: 0 4px 8px;
			font-size: 0.85em;
		}

		> .items {
			> .item {
				display: flex;
				align-items: center;
				width: 100%;
				box-sizing: border-box;
				padding: 10px 16px 10px 8px;
				border-radius: var(--radius);
				font-size: 0.9em;
				margin-bottom: 0.3rem;

				&:hover,
				&:focus-visible {
					text-decoration: none;
					background: var(--panelHighlight);
				}

				&.active {
					color: var(--accent);
					background: var(--accentedBg);
				}

				&.danger {
					color: var(--error);
				}

				> .icon {
					width: 32px;
					margin-inline-end: 2px;
					flex-shrink: 0;
					text-align: center;
					opacity: 0.8;
				}

				> .text {
					white-space: nowrap;
					text-overflow: ellipsis;
					overflow: hidden;
					padding-inline-end: 12px;
				}
			}
		}
	}

	&.grid {
		display: flex;
		align-items: center;
		gap: 8px;
		overflow-x: auto;
		overscroll-behavior-x: contain;
		scrollbar-width: none;
		padding-block: 4px;

		&::-webkit-scrollbar {
			display: none;
		}

		> .group {
			// `display: contents` dissolves the group wrappers so every pill
			// lands in the nav's single flex row; the DOM keeps the groups for
			// `role="group"` / aria-label.
			display: contents;

			& + .group {
				margin-top: 0;
			}

			> .title {
				display: none;
			}

			> .items {
				display: contents;

				> .item {
					flex: 0 0 auto;
					width: auto;
					margin-bottom: 0;
					padding: 8px 14px;
					border-radius: 999px;
					background: var(--buttonBg);
					white-space: nowrap;

					> .icon {
						width: auto;
						margin-inline-end: 6px;
					}

					> .text {
						overflow: visible;
						padding-inline-end: 0;
					}

					&.active {
						background: var(--accentedBg);
					}
				}
			}
		}
	}
}
</style>
