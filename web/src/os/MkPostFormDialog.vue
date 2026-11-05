<!-- Ported from firefish@b425f61:packages/client/src/os/MkPostFormDialog.vue (AGPL-3.0-only) -->
<template>
	<CfModal :close-on-backdrop="false" @closed="emit('closed')">
		<div class="_popup _shadow dialog" @click.stop>
			<button
				class="_button close"
				:aria-label="t('close')"
				@click="close"
			>
				<i class="ph-x ph-bold ph-lg"></i>
			</button>
			<MkPostForm
				v-bind="formProps"
				fixed
				@posted="onPosted"
				@cancel="close"
			/>
		</div>
	</CfModal>
</template>

<script lang="ts" setup>
// Thin wrapper around MkPostForm: Firefish's MkModalWindow becomes CfModal here,
// and `posted` / `cancel` both end the popup (contract §3.1).
import { computed } from "vue";
import CfModal from "./CfModal.vue";
import MkPostForm from "@/components/MkPostForm.vue";
import { t } from "@/i18n";
import type { PostFormProps } from "./post";

const props = withDefaults(defineProps<PostFormProps>(), {
	reply: null,
	renote: null,
	initialText: "",
	initialVisibility: undefined,
	initialFiles: () => [],
	initialLocalOnly: undefined,
	initialNote: null,
	fixed: false,
	autofocus: true,
});

const emit = defineEmits<{
	posted: [];
	closed: [];
}>();

const formProps = computed(() => ({
	reply: props.reply,
	renote: props.renote,
	initialText: props.initialText,
	initialVisibility: props.initialVisibility,
	initialFiles: props.initialFiles,
	initialLocalOnly: props.initialLocalOnly,
	initialNote: props.initialNote,
	autofocus: props.autofocus,
}));

function onPosted(): void {
	emit("posted");
	emit("closed");
}

function close(): void {
	emit("closed");
}
</script>

<style lang="scss" scoped>
.dialog {
	position: relative;
	inline-size: min(520px, 100%);
	max-block-size: calc(100dvb - 64px);
	overflow-y: auto;
	background: var(--panel);
	border-radius: var(--radius);
	box-shadow: 0 4px 32px var(--shadow);

	> .close {
		position: absolute;
		inset-block-start: 8px;
		inset-inline-end: 8px;
		z-index: 2;
		padding: 8px;
		color: var(--fg);
	}

	// The form hides its own close button (`fixed`), so the header must leave
	// room for this one instead of letting the submit button sit under it.
	:deep(.gafaadew > header > .right) {
		padding-inline-end: 52px;
	}
}
</style>
