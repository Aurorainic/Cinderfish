<!-- Ported from firefish@b425f61:packages/client/src/components/MkDriveWindow.vue (AGPL-3.0-only) -->
<template>
	<CfWindow
		:initial-width="800"
		:initial-height="500"
		:can-resize="true"
		@closed="emit('closed')"
	>
		<template #header>
			{{ t("drive") }}
		</template>
		<MkDrive :initial-folder="initialFolder" />
	</CfWindow>
</template>

<script lang="ts" setup>
// Upstream wraps MkDrive in its MkWindow; Cinderfish's shell has no window
// manager, so CfWindow (components/ui) is used directly. The window is opened
// through os.popup(), which is why it must emit `closed` itself.
import type { DriveFolder } from "@/api/types";
import CfWindow from "@/components/ui/CfWindow.vue";
import MkDrive from "@/components/MkDrive.vue";
import { t } from "@/i18n";

defineProps<{
	initialFolder?: DriveFolder | null;
}>();

const emit = defineEmits<{
	(ev: "closed"): void;
}>();
</script>

<style lang="scss" scoped>
:deep(.window-body) {
	display: flex;
	flex-direction: column;
}
</style>
