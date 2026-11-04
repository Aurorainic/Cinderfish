<template>
	<div v-if="visible" class="info" :class="{ warn, card }">
		<i :class="warn ? 'ph-bold ph-warning' : iconClass"></i>
		<slot></slot>
		<button
			v-if="closeable"
			class="_buttonIcon close"
			:aria-label="t('close')"
			@click.stop="close"
		>
			<i class="ph-bold ph-x"></i>
		</button>
	</div>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { t } from "../../i18n";

const props = withDefaults(
	defineProps<{
		icon?: string;
		warn?: boolean;
		card?: boolean;
		closeable?: boolean;
	}>(),
	{
		icon: undefined,
		warn: false,
		card: false,
		closeable: false,
	},
);

const emit = defineEmits<{
	close: [];
}>();

const visible = ref(true);

const iconClass = computed(() => `ph-bold ph-${props.icon ?? "info"}`);

function close(): void {
	visible.value = false;
	emit("close");
}
</script>

<style lang="scss" scoped>
.info {
	display: flex;
	align-items: center;
	gap: 8px;
	padding: 10px 14px;
	border-radius: var(--radius);
	background: var(--infoBg);
	color: var(--infoFg);
	font-size: 0.9em;

	&.warn {
		background: var(--infoWarnBg);
		color: var(--infoWarnFg);
	}

	&.card {
		border: solid 1px var(--divider);
	}

	> .close {
		margin-inline-start: auto;
	}
}
</style>
