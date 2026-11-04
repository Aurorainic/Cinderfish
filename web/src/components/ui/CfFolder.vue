<template>
	<section class="ssazuxis">
		<header class="_button" @click="showBody = !showBody">
			<div class="title"><slot name="header"></slot></div>
			<div class="divider"></div>
			<button
				class="_button toggle"
				:aria-expanded="showBody"
				:aria-controls="bodyId"
			>
				<i
					:class="
						showBody ? 'ph-bold ph-caret-up' : 'ph-bold ph-caret-down'
					"
				></i>
			</button>
		</header>
		<div v-show="showBody" :id="bodyId">
			<slot></slot>
		</div>
	</section>
</template>

<script setup lang="ts">
import { ref, watch } from "vue";
import { getUniqueId } from "../../os/state";

const STORAGE_PREFIX = "ui:folder:";

const props = withDefaults(
	defineProps<{
		expanded?: boolean;
		persistKey?: string | null;
	}>(),
	{
		expanded: true,
		persistKey: null,
	},
);

const bodyId = ref(getUniqueId());

function initialExpanded(): boolean {
	if (props.persistKey == null) return props.expanded;
	const stored = localStorage.getItem(STORAGE_PREFIX + props.persistKey);
	return stored == null ? props.expanded : stored === "t";
}

const showBody = ref(initialExpanded());

watch(showBody, (value) => {
	if (props.persistKey != null) {
		localStorage.setItem(STORAGE_PREFIX + props.persistKey, value ? "t" : "f");
	}
});

function toggleContent(show: boolean): void {
	showBody.value = show;
}

defineExpose({ toggleContent });
</script>

<style lang="scss" scoped>
.ssazuxis {
	> header {
		display: flex;
		align-items: center;
		gap: 8px;
		inline-size: 100%;
		padding-block: 10px;
		color: var(--fg);
		text-align: start;

		.title {
			font-weight: bold;
			font-size: 0.95em;
		}

		.divider {
			flex: 1;
			block-size: 1px;
			background: var(--divider);
		}

		.toggle {
			display: inline-flex;
			align-items: center;
			justify-content: center;
			inline-size: 28px;
			block-size: 28px;
			border-radius: 6px;

			&:hover {
				background: var(--buttonBg);
			}
		}
	}
}
</style>
