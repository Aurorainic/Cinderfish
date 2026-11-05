<!-- Ported from firefish@b425f61:packages/client/src/components/MkReactionsViewer.details.vue (AGPL-3.0-only) -->
<template>
	<div v-if="isShowing" class="bqxuuuey _popup _shadow" :style="style">
		<div class="reaction">
			<MkReactionIcon
				:reaction="reaction"
				:custom-emojis="emojis"
				class="icon"
				:no-style="true"
			/>
			<div class="name">{{ reaction.replace("@.", "") }}</div>
		</div>
		<div class="users">
			<div v-for="u in users" :key="u.id" class="user">
				<MkAvatar class="avatar" :user="u" />
				<MkUserName class="name" :user="u" :nowrap="true" />
			</div>
			<div v-if="users.length > 10" class="omitted">
				+{{ count - 10 }}
			</div>
		</div>
	</div>
</template>

<script setup lang="ts">
import { computed, ref, watch, type CSSProperties, type Ref } from "vue";
import type { UserLite } from "@/api/types";
import MkReactionIcon from "@/components/MkReactionIcon.vue";

/**
 * Ported from Firefish. Firefish renders this inside `MkTooltip`, which owns
 * the anchored placement; `MkTooltip` is not part of this wave's component
 * set, so the popup is placed by itself: fixed, under the hovered reaction
 * button, clamped to the viewport. The markup, classes and users layout are
 * unchanged.
 */
const props = defineProps<{
	reaction: string;
	users: UserLite[];
	count: number;
	emojis?: unknown[];
	targetElement?: HTMLElement | null;
	/** `useTooltip` hands the popup a reactive flag; a plain boolean also works. */
	showing?: Ref<boolean> | boolean;
}>();

const emit = defineEmits<{
	closed: [];
}>();

const isShowing = computed<boolean>(() => {
	const value = props.showing;
	if (value == null) return true;
	if (typeof value === "object" && "value" in value) {
		return (value as Ref<boolean>).value;
	}
	return Boolean(value);
});

const style = ref<CSSProperties>({});

function place(): void {
	const target = props.targetElement;
	if (target == null) return;
	const rect = target.getBoundingClientRect();
	const width = 340;
	const left = Math.min(
		Math.max(8, rect.left),
		Math.max(8, window.innerWidth - width - 8),
	);
	const top = Math.min(rect.bottom + 8, Math.max(8, window.innerHeight - 200));
	style.value = { position: "fixed", left: `${left}px`, top: `${top}px` };
}

watch(
	isShowing,
	(value) => {
		if (value) {
			place();
		} else {
			emit("closed");
		}
	},
	{ immediate: true },
);
</script>

<style lang="scss" scoped>
.bqxuuuey {
	display: flex;
	z-index: 2000000;
	max-width: 340px;

	> .reaction {
		max-width: 100px;
		text-align: center;

		> .icon {
			display: block;
			width: 60px;
			font-size: 60px; // unicodeな絵文字についてはwidthが効かないため
			margin: 0 auto;
		}

		> .name {
			font-size: 1em;
		}
	}

	> .users {
		flex: 1;
		min-width: 0;
		font-size: 0.95em;
		border-left: solid 0.5px var(--divider);
		padding-left: 10px;
		margin-left: 10px;
		margin-right: 14px;
		text-align: left;

		> .user {
			line-height: 24px;
			white-space: nowrap;
			overflow: hidden;
			text-overflow: ellipsis;

			&:not(:last-child) {
				margin-bottom: 3px;
			}

			> .avatar {
				width: 24px;
				height: 24px;
				margin-right: 3px;
			}
		}
	}
}
</style>
