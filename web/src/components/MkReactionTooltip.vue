<!-- Ported from firefish@b425f61:packages/client/src/components/MkReactionTooltip.vue (AGPL-3.0-only) -->
<template>
	<div v-if="isShowing" class="beeadbfb _popup _shadow" :style="style">
		<MkReactionIcon
			:reaction="reaction"
			:custom-emojis="emojis"
			class="icon"
			:no-style="true"
		/>
		<div class="name">{{ reaction.replace("@.", "") }}</div>
	</div>
</template>

<script setup lang="ts">
import { computed, ref, watch, type CSSProperties, type Ref } from "vue";
import MkReactionIcon from "@/components/MkReactionIcon.vue";

/**
 * Ported from Firefish. Used by `MkNotification` (notifications task) through
 * `os.popup`. Firefish wraps this in `MkTooltip`; that component is not part of
 * this wave, so the popup positions itself under `targetElement`.
 */
const props = defineProps<{
	reaction: string;
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
	const size = 120;
	const left = Math.min(
		Math.max(8, rect.left + rect.width / 2 - size / 2),
		Math.max(8, window.innerWidth - size - 8),
	);
	const top = Math.min(rect.bottom + 8, Math.max(8, window.innerHeight - size));
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
.beeadbfb {
	text-align: center;
	z-index: 2000000;
	padding: 8px 12px;

	> .icon {
		display: block;
		width: 60px;
		font-size: 60px; // unicodeな絵文字についてはwidthが効かないため
		margin: 0 auto;
	}

	> .name {
		font-size: 0.9em;
	}
}
</style>
