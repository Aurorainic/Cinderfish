<!-- Ported from firefish@b425f61:packages/client/src/components/MkVisibilityPicker.vue (AGPL-3.0-only) -->
<template>
	<CfModal :close-on-backdrop="true" @closed="emit('closed')">
		<div class="_popup _shadow panel" @click.stop>
			<button
				v-for="item in items"
				:key="item.value"
				class="_button item"
				:class="{ active: v === item.value, localOnly: item.value === 'localOnly' && localOnly }"
				:disabled="item.value === 'specified' && localOnly"
				@click="choose(item.value)"
			>
				<div class="icon">
					<i :class="item.icon"></i>
				</div>
				<div class="body">
					<span class="itemTitle">{{ t(item.title) }}</span>
					<span class="itemDescription">{{ t(item.description) }}</span>
				</div>
				<div v-if="item.value === 'localOnly'" class="toggle">
					<i
						:class="
							localOnly
								? 'ph-toggle-right ph-bold ph-lg'
								: 'ph-toggle-left ph-bold ph-lg'
						"
					></i>
				</div>
			</button>
		</div>
	</CfModal>
</template>

<script lang="ts" setup>
// Degradation: Firefish anchors the picker to the visibility button through
// MkModal's `src`; CfModal centres it, so `src` is accepted but unused.
import { nextTick, ref, watch } from "vue";
import type { NoteVisibility } from "@/api/types";
import CfModal from "@/os/CfModal.vue";
import { t } from "@/i18n";

type ItemValue = NoteVisibility | "localOnly";

interface VisibilityItem {
	value: ItemValue;
	icon: string;
	title: string;
	description: string;
}

const props = withDefaults(
	defineProps<{
		currentVisibility: NoteVisibility;
		currentLocalOnly: boolean;
		src?: HTMLElement | null;
	}>(),
	{
		src: null,
	},
);

const emit = defineEmits<{
	changeVisibility: [visibility: NoteVisibility];
	changeLocalOnly: [localOnly: boolean];
	closed: [];
}>();

const items: VisibilityItem[] = [
	{
		value: "public",
		icon: "ph-planet ph-bold ph-lg",
		title: "_visibility.public",
		description: "_visibility.publicDescription",
	},
	{
		value: "home",
		icon: "ph-house ph-bold ph-lg",
		title: "_visibility.home",
		description: "_visibility.homeDescription",
	},
	{
		value: "followers",
		icon: "ph-lock ph-bold ph-lg",
		title: "_visibility.followers",
		description: "_visibility.followersDescription",
	},
	{
		value: "specified",
		icon: "ph-envelope-simple-open ph-bold ph-lg",
		title: "_visibility.specified",
		description: "_visibility.specifiedDescription",
	},
	{
		value: "localOnly",
		icon: "ph-hand-fist ph-bold ph-lg",
		title: "_visibility.localOnly",
		description: "_visibility.localOnlyDescription",
	},
];

const v = ref<NoteVisibility>(props.currentVisibility);
const localOnly = ref<boolean>(props.currentLocalOnly);

watch(localOnly, (next) => {
	emit("changeLocalOnly", next);
});

function choose(value: ItemValue): void {
	if (value === "localOnly") {
		localOnly.value = !localOnly.value;
		return;
	}
	v.value = value;
	emit("changeVisibility", value);
	nextTick(() => emit("closed"));
}
</script>

<style lang="scss" scoped>
.panel {
	inline-size: 240px;
	padding: 8px 0;
	background: var(--popup);
	border-radius: var(--radius);
	box-shadow: 0 4px 32px var(--shadow);

	> .item {
		display: flex;
		padding: 8px 14px;
		font-size: 12px;
		text-align: start;
		inline-size: 100%;
		box-sizing: border-box;
		color: var(--fg);

		&:hover {
			background: var(--listItemHoverBg);
		}

		&:active {
			background: var(--buttonHoverBg);
		}

		&.active {
			color: var(--fgOnAccent);
			background: var(--accent);
		}

		&.localOnly.active {
			color: var(--accent);
			background: inherit;
		}

		&:disabled {
			opacity: 0.5;
		}

		> .icon {
			display: flex;
			justify-content: center;
			align-items: center;
			margin-inline-end: 10px;
			inline-size: 16px;
			margin-block: auto;
		}

		> .body {
			flex: 1 1 auto;
			overflow: hidden;
			text-overflow: ellipsis;

			> .itemTitle {
				display: block;
				font-weight: bold;
			}

			> .itemDescription {
				opacity: 0.6;
				display: block;
				line-height: 1.5;
			}
		}

		> .toggle {
			display: flex;
			justify-content: center;
			align-items: center;
			margin-inline-start: 10px;
			inline-size: 16px;
			margin-block: auto;
		}
	}
}
</style>
