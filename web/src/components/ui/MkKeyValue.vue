// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/components/MkKeyValue.vue
<template>
	<div class="cinderfish-key-value" :class="{ oneline }">
		<div class="key">
			<slot name="key"></slot>
		</div>
		<div class="value">
			<slot name="value"></slot>
			<button
				v-if="copy"
				class="copy"
				:title="strings.copy"
				:aria-label="strings.copy"
				style="margin-inline-start: 0.5em"
				@click="copy_"
			>
				⧉
			</button>
		</div>
	</div>
</template>

<script lang="ts" setup>
// Local UI strings (hardcoded Chinese until a shared i18n layer exists).
const strings = {
	copy: "复制",
};

const props = withDefaults(
	defineProps<{
		copy?: string | null;
		oneline?: boolean;
	}>(),
	{
		copy: null,
		oneline: false,
	},
);

const copy_ = () => {
	if (props.copy) void navigator.clipboard.writeText(props.copy);
	// TODO: Firefish shows an os.success() toast here; Cinderfish lacks the
	// os popup runtime, so no feedback is shown yet.
};
</script>

<style scoped>
.cinderfish-key-value > .key {
	font-size: 0.85em;
	padding: 0 0 0.25em;
	opacity: 0.75;
}

.cinderfish-key-value.oneline {
	display: flex;
}

.cinderfish-key-value.oneline > .key {
	inline-size: 30%;
	font-size: 1em;
	padding: 0 8px 0 0;
}

.cinderfish-key-value.oneline > .value {
	inline-size: 70%;
	white-space: nowrap;
	overflow: hidden;
	text-overflow: ellipsis;
}

.cinderfish-key-value > .value > .copy {
	background: none;
	border: none;
	cursor: pointer;
	color: inherit;
	padding: 0;
}
</style>
