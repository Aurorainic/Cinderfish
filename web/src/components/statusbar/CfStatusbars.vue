<!-- Ported from firefish@b425f61:packages/client/src/ui/_common_/statusbars.vue (AGPL-3.0-only) -->
<template>
	<div v-if="items.length > 0" class="cf-statusbars">
		<div
			v-for="item in items"
			:key="item.id"
			class="item"
			:class="[
				{ black: item.black },
				item.size,
			]"
			:title="item.name"
		>
			<span v-if="item.name !== ''" class="name">{{ item.name }}</span>
			<CfStatusbarFederation
				v-if="item.type === 'federation'"
				class="body"
				:colored="item.props.colored === true"
				:refresh-interval-sec="numberProp(item, 'refreshIntervalSec', 300)"
				:marquee-duration="numberProp(item, 'marqueeDuration', 40)"
				:marquee-reverse="item.props.marqueeReverse === true"
			/>
			<CfStatusbarRss
				v-else-if="item.type === 'rss'"
				class="body"
				:url="stringProp(item, 'url', '')"
				:shuffle="item.props.shuffle === true"
				:refresh-interval-sec="numberProp(item, 'refreshIntervalSec', 300)"
				:marquee-duration="numberProp(item, 'marqueeDuration', 40)"
				:marquee-reverse="item.props.marqueeReverse === true"
			/>
			<span v-else class="body unknown">
				未知的条目类型：{{ item.type ?? "（未设置）" }}
			</span>
		</div>
	</div>
</template>

<script lang="ts" setup>
// Renders the status bar items configured in the settings page. Firefish also
// supports `userList` (lists are out of the first-release scope); an unknown
// type is reported inline instead of rendering an empty box.
import { computed } from "vue";
import type { StatusbarItem } from "@/store";
import { defaultStore } from "@/store";
import CfStatusbarFederation from "./CfStatusbarFederation.vue";
import CfStatusbarRss from "./CfStatusbarRss.vue";

defineOptions({ name: "CfStatusbars" });

const items = computed<StatusbarItem[]>(
	() => defaultStore.reactiveState.statusbars.value,
);

function numberProp(
	item: StatusbarItem,
	key: string,
	fallback: number,
): number {
	const value = item.props[key];
	return typeof value === "number" && Number.isFinite(value) ? value : fallback;
}

function stringProp(
	item: StatusbarItem,
	key: string,
	fallback: string,
): string {
	const value = item.props[key];
	return typeof value === "string" ? value : fallback;
}
</script>

<style lang="scss" scoped>
.cf-statusbars {
	background: var(--panel);
	border-top: solid 0.5px var(--divider);

	> .item {
		--height: 26px;
		display: flex;
		align-items: center;
		height: var(--height);
		overflow: hidden;
		font-size: 0.85em;

		&.verySmall {
			--height: 18px;
			font-size: 0.72em;
		}

		&.small {
			--height: 22px;
			font-size: 0.78em;
		}

		&.large {
			--height: 32px;
			font-size: 0.92em;
		}

		&.veryLarge {
			--height: 40px;
			font-size: 1em;
		}

		&.black {
			background: #000;
			color: #fff;

			.name,
			.unknown {
				color: #fff;
			}
		}

		.name {
			flex-shrink: 0;
			padding: 0 10px;
			font-weight: 700;
			color: var(--fgTransparentWeak);
		}

		.body {
			flex: 1;
			min-width: 0;
			height: 100%;
		}

		.unknown {
			font-size: 0.85em;
			color: var(--fgTransparentWeak);
			padding-inline-start: 10px;
		}
	}
}
</style>
