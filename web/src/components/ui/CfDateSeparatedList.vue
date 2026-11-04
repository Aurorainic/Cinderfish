<template>
	<div
		class="sqadhkmv"
		:class="{ noGap }"
		:data-direction="direction"
		:data-reversed="reversed ? 'true' : 'false'"
	>
		<template v-for="(item, index) in items" :key="item.id">
			<slot :item="item"></slot>
			<div
				v-if="
					index !== items.length - 1 &&
					dayOf(item.createdAt) !== dayOf(items[index + 1].createdAt)
				"
				class="separator"
			>
				<p class="date">
					<span>
						<i class="icon ph-bold ph-caret-up"></i>
						{{ dateText(item.createdAt) }}
					</span>
					<span>
						{{ dateText(items[index + 1].createdAt) }}
						<i class="icon ph-bold ph-caret-down"></i>
					</span>
				</p>
			</div>
		</template>
	</div>
</template>

<script setup lang="ts" generic="T extends { id: string; createdAt: string }">
import { t } from "../../i18n";

withDefaults(
	defineProps<{
		items: T[];
		direction?: string;
		reversed?: boolean;
		noGap?: boolean;
		ad?: boolean;
	}>(),
	{
		direction: "down",
		reversed: false,
		noGap: false,
		ad: false,
	},
);

function dayOf(value: string): string {
	const date = new Date(value);
	return `${date.getFullYear()}-${date.getMonth()}-${date.getDate()}`;
}

function dateText(value: string): string {
	const date = new Date(value);
	return t("monthAndDay", {
		month: date.getMonth() + 1,
		day: date.getDate(),
	});
}

</script>

<style lang="scss" scoped>
.sqadhkmv {
	display: flex;
	flex-direction: column;

	&.noGap {
		gap: 0;
	}

	> .separator {
		position: relative;
		display: flex;
		align-items: center;
		justify-content: center;
		padding-block: 8px;

		&::before,
		&::after {
			content: "";
			flex: 1;
			block-size: 1px;
			background: var(--divider);
		}

		.date {
			display: flex;
			align-items: center;
			gap: 8px;
			margin: 0 16px;
			color: var(--dateLabelFg);
			font-size: 0.75em;
			opacity: 0.8;

			> span {
				display: inline-flex;
				align-items: center;
				gap: 4px;
			}

			.icon {
				font-size: 0.9em;
				opacity: 0.6;
			}
		}
	}
}
</style>
