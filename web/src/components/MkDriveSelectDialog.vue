<!-- Ported from firefish@b425f61:packages/client/src/components/MkDriveSelectDialog.vue (AGPL-3.0-only) -->
<template>
	<CfModal :close-on-backdrop="true" @closed="onWindowClosed">
		<CfWindow
			:initial-width="800"
			:initial-height="500"
			:can-resize="true"
			@closed="onWindowClosed"
		>
			<template #header>
				{{ title }}
				<span v-if="selected.length > 0" class="count"
					>({{ selected.length }})</span
				>
			</template>
			<div class="body">
				<MkDrive
					class="drive"
					:multiple="multiple"
					:select="type"
					@change-selection="onChangeSelection"
					@selected="ok"
				/>
				<div class="footer">
					<CfButton @click="cancel">{{ t("cancel") }}</CfButton>
					<CfButton primary :disabled="okDisabled" @click="ok">{{
						t("ok")
					}}</CfButton>
				</div>
			</div>
		</CfWindow>
	</CfModal>
</template>

<script lang="ts" setup>
// Wave-2 difference from upstream: Firefish builds this on MkModalWindow
// (modal + window + ok/cancel bar). Cinderfish has no MkModalWindow yet, so the
// dialog is assembled from CfModal + CfWindow + CfButton; the Drive browser and
// the emitted events are unchanged. `done` is always emitted exactly once.
import { computed, ref } from "vue";
import type { DriveFile, DriveFolder } from "@/api/types";
import CfButton from "@/components/ui/CfButton.vue";
import CfModal from "@/os/CfModal.vue";
import CfWindow from "@/components/ui/CfWindow.vue";
import MkDrive from "@/components/MkDrive.vue";
import { t } from "@/i18n";

const props = withDefaults(
	defineProps<{
		type?: "file" | "folder";
		multiple: boolean;
	}>(),
	{
		type: "file",
	},
);

const emit = defineEmits<{
	(ev: "done", r?: DriveFile[]): void;
	(ev: "closed"): void;
}>();

const selected = ref<DriveFile[]>([]);
let finished = false;

const title = computed(() => {
	if (props.multiple) {
		return props.type === "file" ? t("selectFiles") : t("selectFolders");
	}
	return props.type === "file" ? t("selectFile") : t("selectFolder");
});

const okDisabled = computed(
	() => props.type === "file" && selected.value.length === 0,
);

function finish(result?: DriveFile[]): void {
	if (finished) return;
	finished = true;
	emit("done", result);
	emit("closed");
}

function ok(): void {
	if (okDisabled.value) return;
	finish(selected.value);
}

function cancel(): void {
	finish(undefined);
}

// Fired by the window's close button, Escape or the modal backdrop.
function onWindowClosed(): void {
	finish(undefined);
}

function onChangeSelection(files: DriveFile[] | DriveFolder[]): void {
	selected.value = files.filter(
		(file): file is DriveFile => "type" in file && "url" in file,
	);
}
</script>

<style lang="scss" scoped>
.count {
	margin-left: 8px;
	opacity: 0.5;
}

.body {
	display: flex;
	flex-direction: column;
	block-size: 100%;
	min-block-size: 0;
}

.drive {
	flex: 1;
	min-block-size: 0;
	overflow: auto;
}

.footer {
	display: flex;
	justify-content: flex-end;
	gap: 8px;
	padding: 12px;
	border-block-start: solid 1px var(--divider);
}
</style>
