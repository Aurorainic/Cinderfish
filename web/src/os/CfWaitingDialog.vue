<template>
	<CfModal ref="modal" :close-on-backdrop="Boolean(success)" @closed="emit('closed')">
		<div class="body" :class="{ iconOnly: text == null || success }">
			<i v-if="success" class="icon success ph-bold ph-check"></i>
			<CfLoading v-else class="icon waiting" :em="true" :size-em="2" />
			<div v-if="text && !success" class="text">
				{{ text }}<span class="ellipsis">...</span>
			</div>
		</div>
	</CfModal>
</template>

<script lang="ts" setup>
import { ref, watch } from "vue";
import CfLoading from "../components/ui/CfLoading.vue";
import CfModal from "./CfModal.vue";

const props = withDefaults(
	defineProps<{
		success: boolean;
		showing: boolean;
		text?: string | null;
	}>(),
	{
		text: null,
	},
);

const emit = defineEmits<{
	done: [];
	closed: [];
}>();

const modal = ref<InstanceType<typeof CfModal> | null>(null);

watch(
	() => props.showing,
	(value) => {
		if (!value) {
			emit("done");
			modal.value?.close();
		}
	},
);
</script>

<style lang="scss" scoped>
.body {
	display: flex;
	align-items: center;
	gap: 16px;
	padding: 32px 40px;
	background: var(--panel);
	border-radius: var(--radius);
	box-shadow: 0 4px 32px var(--shadow);
	color: var(--fg);

	&.iconOnly {
		justify-content: center;
		padding: 32px;
	}
}

.icon {
	font-size: 1.6em;
	line-height: 1;

	&.success {
		color: var(--success);
	}
}

.text {
	font-size: 0.95em;
	color: var(--fgTransparentWeak);
}

.ellipsis {
	display: inline-block;
	margin-inline-start: 2px;
}
</style>
