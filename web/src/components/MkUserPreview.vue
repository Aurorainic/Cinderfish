<!-- Ported from firefish@b425f61:packages/client/src/components/MkUserPreview.vue (AGPL-3.0-only) -->
<template>
	<transition name="popup" appear @after-leave="emit('closed')">
		<div
			v-if="showing"
			class="cf-user-preview _popup _shadow"
			:style="{ zIndex, top: `${top}px`, left: `${left}px` }"
			@mouseover="emit('mouseover')"
			@mouseleave="emit('mouseleave')"
		>
			<MkUserInfo v-if="user != null" :user="user" detailed />
			<MkLoading v-else />
		</div>
	</transition>
</template>

<script setup lang="ts">
// Firefish positions the card centred under the avatar (300px wide) and keeps
// it open while the pointer is over either element; the directive owns the
// timers, this component only fetches and draws.
import { onMounted, ref } from "vue";
import MkLoading from "@/components/global/MkLoading.vue";
import MkUserInfo from "@/components/MkUserInfo.vue";
import { api } from "@/api/client";
import type { UserDetailed } from "@/api/types";
import { claimZIndex } from "@/os/state";

const props = defineProps<{
	showing: boolean;
	q: string;
	source: HTMLElement;
}>();

const emit = defineEmits<{
	closed: [];
	mouseover: [];
	mouseleave: [];
}>();

const zIndex = claimZIndex("middle");
const user = ref<UserDetailed | null>(null);
const top = ref(0);
const left = ref(0);

onMounted(() => {
	api<UserDetailed>("users/show", { userId: props.q })
		.then((result) => {
			if (!props.showing) return;
			user.value = result;
		})
		.catch(() => {
			// The popup closes through the directive's listener on a fetch error;
			// there is nothing useful to draw for a missing user.
		});

	const rect = props.source.getBoundingClientRect();
	top.value = rect.top + props.source.offsetHeight + window.scrollY;
	left.value = rect.left + props.source.offsetWidth / 2 - 150 + window.scrollX;
});
</script>

<style lang="scss" scoped>
.cf-user-preview {
	position: absolute;
	width: 300px;
	max-width: calc(100vw - 16px);
}

.popup-enter-active,
.popup-leave-active {
	transition:
		opacity 0.2s,
		transform 0.2s;
}

.popup-enter-from,
.popup-leave-to {
	opacity: 0;
	transform: translateY(-8px);
}
</style>
