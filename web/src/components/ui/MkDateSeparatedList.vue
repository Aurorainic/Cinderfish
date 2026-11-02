// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/components/MkDateSeparatedList.vue
//
// The ad slot (MkAd) of the original is dropped: TODO Cinderfish has no ads
// backend. The animation toggle always reflects reduced-motion preference.
<template>
	<TransitionGroup
		tag="div"
		name="list"
		class="cinderfish-date-separated-list"
		:class="{ noGap }"
		:data-direction="direction"
		:data-reversed="reversed ? 'true' : 'false'"
	>
		<template v-for="(item, index) in items" :key="item.id">
			<slot :item="item"></slot>
			<div
				v-if="
					index !== items.length - 1 &&
					new Date(item.createdAt).getDate() !==
						new Date(items[index + 1].createdAt).getDate()
				"
				class="separator"
			>
				<p class="date">
					<span>
						<i class="caret">▲</i>
						{{ getDateText(item.createdAt) }}
					</span>
					<span>
						{{ getDateText(items[index + 1].createdAt) }}
						<i class="caret">▼</i>
					</span>
				</p>
			</div>
		</template>
	</TransitionGroup>
</template>

<script lang="ts" setup generic="T extends Item">
import { TransitionGroup } from "vue";

export interface Item {
	id: string;
	createdAt: string;
}

// Local UI strings (hardcoded Chinese until a shared i18n layer exists).
const strings = {
	yearMonthDay: (year: number, month: number, day: number) =>
		`${year}年${month}月${day}日`,
};

withDefaults(
	defineProps<{
		items: T[];
		direction?: "up" | "down";
		reversed?: boolean;
		noGap?: boolean;
	}>(),
	{
		direction: "down",
		reversed: false,
		noGap: false,
	},
);

defineSlots<{
	default(props: { item: T }): unknown;
}>();

function getDateText(time: string): string {
	const date = new Date(time);
	if (Number.isNaN(date.getTime())) return "";
	return strings.yearMonthDay(
		date.getFullYear(),
		date.getMonth() + 1,
		date.getDate(),
	);
}
</script>

<style>
/* Unscoped on purpose: the TransitionGroup's generated classes land on the
   slotted children, mirroring the original's non-scoped block. */
.cinderfish-date-separated-list > *:empty {
	display: none;
}

.cinderfish-date-separated-list:not(.noGap) > *:not(:last-child) {
	margin-block-end: var(--margin, 16px);
}

/* Firefish's noGap variant stacks rows flush and draws a 0.5px hairline
   between them, which is what makes a timeline read as one continuous panel. */
.cinderfish-date-separated-list.noGap > * {
	margin: 0 !important;
	border: none;
	border-radius: 0;
	box-shadow: none;
}

.cinderfish-date-separated-list.noGap > *:first-child {
	border-radius: var(--radius) var(--radius) 0 0;
}

.cinderfish-date-separated-list.noGap > *:last-child {
	border-radius: 0 0 var(--radius) var(--radius);
}

.cinderfish-date-separated-list.noGap > *:not(:last-child) {
	border-block-end: solid 0.5px var(--divider);
}

.cinderfish-date-separated-list > .list-move {
	transition: transform 0.7s cubic-bezier(0.23, 1, 0.32, 1);
}

.cinderfish-date-separated-list > .list-enter-active {
	transition:
		transform 0.7s cubic-bezier(0.23, 1, 0.32, 1),
		opacity 0.7s cubic-bezier(0.23, 1, 0.32, 1);
}

.cinderfish-date-separated-list[data-direction="up"] > .list-enter-from {
	opacity: 0;
	transform: translateY(64px);
}

.cinderfish-date-separated-list[data-direction="down"] > .list-enter-from {
	opacity: 0;
	transform: translateY(-64px);
}

@media (prefers-reduced-motion: reduce) {
	.cinderfish-date-separated-list > .list-move,
	.cinderfish-date-separated-list > .list-enter-active {
		transition: none;
	}
}

.cinderfish-date-separated-list > .separator {
	text-align: center;
}

/* Firefish draws the date label as flat 12px text, not a panel chip. */
.cinderfish-date-separated-list > .separator > .date {
	display: inline-block;
	position: relative;
	margin: 0;
	padding-block: 0;
	padding-inline: 16px;
	line-height: 32px;
	text-align: center;
	font-size: 12px;
	color: var(--dateLabelFg, var(--fg));
}

.cinderfish-date-separated-list > .separator > .date > span:first-child {
	margin-inline-end: 8px;
}

.cinderfish-date-separated-list > .separator > .date > span:last-child {
	margin-inline-start: 8px;
}

.cinderfish-date-separated-list > .separator > .date > span > .caret {
	font-style: normal;
	font-size: 0.9em;
	margin-inline: 8px;
	opacity: 0.6;
}
</style>
