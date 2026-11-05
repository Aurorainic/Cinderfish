<!-- Ported from firefish@b425f61:packages/client/src/components/MkEmojiPickerDialog.vue (AGPL-3.0-only) -->
<template>
	<CfModal
		ref="modal"
		:close-on-backdrop="true"
		@click="checkForShift"
		@closed="emit('closed')"
	>
		<div class="_popup _shadow panel" :class="{ drawer: asDrawer }" @click.stop>
			<MkEmojiPicker
				ref="picker"
				:show-pinned="showPinned"
				:as-reaction-picker="asReactionPicker"
				:as-drawer="asDrawer"
				:max-height="maxHeight"
				@chosen="chosen"
			/>
		</div>
	</CfModal>
</template>

<script lang="ts" setup>
// Wave-3 degradation (contract §2): CfModal always centres its content, so the
// `src` anchor is accepted for signature compatibility but not used for pixel
// positioning. The mobile drawer variant is kept as a full-width layout.
import { computed, onBeforeUnmount, onMounted, ref } from "vue";
import CfModal from "@/os/CfModal.vue";
import MkEmojiPicker from "@/components/MkEmojiPicker.vue";
import { deviceKind } from "@/scripts/device-kind";

const props = withDefaults(
	defineProps<{
		manualShowing?: boolean | null;
		src?: HTMLElement | null;
		showPinned?: boolean;
		asReactionPicker?: boolean;
	}>(),
	{
		manualShowing: null,
		src: null,
		showPinned: true,
		asReactionPicker: false,
	},
);

const emit = defineEmits<{
	done: [emoji: string];
	close: [];
	closed: [];
}>();

const modal = ref<InstanceType<typeof CfModal> | null>(null);
const picker = ref<InstanceType<typeof MkEmojiPicker> | null>(null);

const asDrawer = computed(
	() => props.asReactionPicker && deviceKind === "smartphone",
);

const maxHeight = computed(() => Math.min(window.innerHeight * 0.7, 480));

function checkForShift(ev?: MouseEvent): void {
	if (ev?.shiftKey === true) return;
	modal.value?.close();
}

function chosen(emoji: string, ev: MouseEvent | null): void {
	emit("done", emoji);
	if (ev?.shiftKey !== true) modal.value?.close();
}

function onKeydown(event: KeyboardEvent): void {
	if (event.key === "Escape") {
		event.preventDefault();
		modal.value?.close();
	}
}

onMounted(() => {
	picker.value?.reset();
	picker.value?.focus();
	window.addEventListener("keydown", onKeydown);
});

onBeforeUnmount(() => {
	window.removeEventListener("keydown", onKeydown);
});
</script>

<style lang="scss" scoped>
.panel {
	padding: 4px;
	background: var(--popup);
	border-radius: var(--radius);
	box-shadow: 0 4px 32px var(--shadow);
	overflow: hidden;

	&.drawer {
		inline-size: 100%;
		border-start-start-radius: 24px;
		border-start-end-radius: 24px;
		border-end-start-radius: 0;
		border-end-end-radius: 0;
	}
}
</style>
