<!-- Ported from firefish@b425f61:packages/client/src/components/MkEmojiPicker.section.vue (AGPL-3.0-only) -->
<template>
	<section>
		<header class="_acrylic" @click="shown = !shown">
			<i
				class="toggle ph-fw ph-lg"
				:class="
					shown
						? 'ph-caret-down ph-bold ph-lg'
						: 'ph-caret-up ph-bold ph-lg'
				"
			></i>
			<slot></slot> ({{ localEmojis.length }})
			<span v-if="skinToneSelector && skinTones">
				<button
					v-for="skinTone in skinTones"
					:key="skinTone"
					class="_button tone"
					:aria-label="
						skinToneLabels
							? skinToneLabels[skinTones.indexOf(skinTone)]
							: ''
					"
					@click.stop="
						applyUnicodeSkinTone(skinTones.indexOf(skinTone) + 1)
					"
				>
					<i
						class="ph-circle ph-fill ph-fw ph-lg"
						:style="{ color: skinTone }"
					></i>
				</button>
			</span>
		</header>
		<div v-if="shown" class="body">
			<button
				v-for="emoji in localEmojis"
				:key="emoji"
				class="_button item"
				@click="emit('chosen', emoji, $event)"
			>
				<MkEmoji class="emoji" :emoji="emoji" :normal="true" />
			</button>
		</div>
	</section>
</template>

<script lang="ts" setup>
import { onMounted, ref, watch } from "vue";
import { addSkinTone } from "@/scripts/emojilist";

const props = withDefaults(
	defineProps<{
		emojis: string[];
		initialShown?: boolean;
		skinToneSelector?: boolean;
		skinTones?: string[];
		skinToneLabels?: string[];
	}>(),
	{
		initialShown: false,
		skinToneSelector: false,
		skinTones: undefined,
		skinToneLabels: undefined,
	},
);

const emit = defineEmits<{
	chosen: [emoji: string, event: MouseEvent];
}>();

const localEmojis = ref([...props.emojis]);
const shown = ref(props.initialShown);

function applyUnicodeSkinTone(tone?: number): void {
	localEmojis.value = localEmojis.value.map((emoji) =>
		addSkinTone(emoji, tone),
	);
}

onMounted(() => {
	if (props.skinToneSelector) applyUnicodeSkinTone();
});

watch(
	() => props.emojis,
	(next) => {
		localEmojis.value = [...next];
		if (props.skinToneSelector) applyUnicodeSkinTone();
	},
);
</script>

<style lang="scss" scoped>
section {
	> header {
		position: sticky;
		inset-block-start: 0;
		inset-inline-start: 0;
		block-size: 32px;
		line-height: 32px;
		z-index: 1;
		padding: 0 8px;
		font-size: 12px;
		cursor: pointer;
		display: flex;
		align-items: center;
		gap: 4px;

		&:hover {
			color: var(--accent);
		}

		> .tone {
			padding: 0;
			margin-inline-start: 2px;
			font-size: 12px;
		}
	}

	> .body {
		position: relative;
		padding: 8px;
		display: flex;
		flex-wrap: wrap;

		> .item {
			position: relative;
			padding: 0;
			inline-size: var(--eachSize, 40px);
			block-size: var(--eachSize, 40px);
			border-radius: 4px;
			font-size: 24px;

			&:focus-visible {
				outline: solid 2px var(--focus, var(--accent));
				z-index: 1;
			}

			&:hover {
				background: var(--listItemHoverBg);
			}

			&:active {
				background: var(--accent);
			}

			> .emoji {
				block-size: 1.25em;
				vertical-align: -0.25em;
				pointer-events: none;
			}
		}
	}
}
</style>
