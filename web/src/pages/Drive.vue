<!-- Ported from firefish@b425f61:packages/client/src/pages/drive.vue (AGPL-3.0-only) -->
<template>
	<div class="cf-page-drive">
		<MkDrive :initial-folder-id="folderId" @cd="onCd" />
	</div>
</template>

<script lang="ts" setup>
// The page is a thin wrapper around MkDrive, exactly like upstream. Upstream
// keeps the current folder in a local ref for the (hidden) page header; the
// route /my/drive/folder/:folder additionally deep-links into a folder, which
// is passed down as `initialFolderId` because the route only carries the id.
import { computed, ref } from "vue";
import { useRoute } from "vue-router";
import type { DriveFolder } from "@/api/types";
import MkDrive from "@/components/MkDrive.vue";
import { t } from "@/i18n";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageDrive" });

const route = useRoute();
const folder = ref<DriveFolder | null>(null);

const folderId = computed(() => {
	const raw = route.params.folder;
	if (typeof raw === "string") return raw === "" ? null : raw;
	if (Array.isArray(raw) && typeof raw[0] === "string") return raw[0];
	return null;
});

function onCd(next: DriveFolder | null): void {
	folder.value = next;
}

definePageMetadata(
	computed(() => ({
		title: folder.value != null ? folder.value.name : t("drive"),
		icon: "ph-cloud ph-bold ph-lg",
		hideHeader: true,
	})),
);
</script>

<style lang="scss" scoped>
.cf-page-drive {
	display: flex;
	flex-direction: column;
	block-size: 100%;
	min-block-size: 60vh;
}
</style>
