<template>
	<div class="alqyeyti" :class="{ oneline }">
		<div class="key"><slot name="key"></slot></div>
		<div class="value">
			<slot name="value"></slot>
			<button
				v-if="copy != null"
				class="_textButton copy"
				:title="t('copy')"
				@click.stop="copyValue"
			>
				<i class="ph-bold ph-clipboard-text"></i>
			</button>
		</div>
	</div>
</template>

<script setup lang="ts">
import { success } from "../../os";
import { t } from "../../i18n";

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

async function copyValue(): Promise<void> {
	if (props.copy == null) return;
	try {
		await navigator.clipboard.writeText(props.copy);
		success();
	} catch {
		console.info(props.copy);
	}
}
</script>

<style lang="scss" scoped>
.alqyeyti {
	display: flex;
	align-items: baseline;
	gap: 8px;
	padding-block: 6px;

	&.oneline {
		align-items: center;
	}

	> .key {
		flex: 0 0 35%;
		color: var(--fgTransparentWeak);
		font-size: 0.85em;
	}

	> .value {
		flex: 1;
		min-inline-size: 0;
		display: flex;
		align-items: center;
		gap: 4px;
		overflow-wrap: anywhere;
	}

	.copy {
		font-size: 0.9em;
	}
}
</style>
