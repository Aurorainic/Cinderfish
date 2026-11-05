<!-- Ported from firefish@b425f61:packages/client/src/widgets/photos.vue (AGPL-3.0-only) -->
<template>
	<WidgetCard
		:title="t('_widgets.photos')"
		icon="ph-bold ph-camera"
		:show-header="widgetProps.showHeader === true"
		:naked="widgetProps.transparent === true"
	>
		<CfLoading v-if="fetching" />
		<CfInfo v-else-if="me == null">登录后可以查看网盘中的照片。</CfInfo>
		<CfInfo v-else-if="failed">无法读取网盘文件。</CfInfo>
		<CfInfo v-else-if="images.length === 0">网盘里还没有图片。</CfInfo>
		<div v-else class="stream">
			<a
				v-for="file in images"
				:key="file.id"
				class="img"
				:href="resolveMediaUrl(file.url)"
				target="_blank"
				rel="noopener noreferrer"
				:title="file.name"
				:style="{ backgroundImage: `url(${resolveMediaUrl(file.url)})` }"
			></a>
		</div>
	</WidgetCard>
</template>

<script setup lang="ts">
// Firefish streams `driveFileCreated` over the `main` channel and paginates
// `drive/stream`. Cinderfish has neither (api/endpoints.ts), so the widget takes
// the newest files from `drive/files` and keeps the image ones. Streaming
// insertion is not implemented; the list refreshes when the widget remounts.
import { onMounted, ref } from "vue";
import WidgetCard from "./WidgetCard.vue";
import { CfInfo, CfLoading } from "@/components/ui";
import { api, resolveMediaUrl } from "@/api/client";
import { ENDPOINTS } from "@/api/endpoints";
import type { DriveFile } from "@/api/types";
import { me } from "@/account";
import { t } from "@/i18n";
import { useWidgetPropsManager, type WidgetComponentExpose, type WidgetField } from "./index";
import type { Widget } from "@/store/widgets";

const name = "photos";

const props = defineProps<{ widget?: Widget }>();
const emit = defineEmits<{ updateProps: [data: Record<string, unknown>] }>();

const fields: WidgetField[] = [
	{ key: "showHeader", label: "显示标题栏", type: "boolean", default: true },
	{ key: "transparent", label: "透明背景", type: "boolean", default: false },
];

const { widgetProps, configure } = useWidgetPropsManager(
	name,
	fields,
	props.widget,
	(data) => emit("updateProps", data),
);

const images = ref<DriveFile[]>([]);
const fetching = ref(true);
const failed = ref(false);

onMounted(() => {
	if (me.value == null) {
		fetching.value = false;
		return;
	}
	api(ENDPOINTS.driveFiles, { limit: 24 })
		.then((files) => {
			images.value = files
				.filter((file) => /^image\//.test(file.type))
				.slice(0, 9);
		})
		.catch(() => {
			failed.value = true;
		})
		.finally(() => {
			fetching.value = false;
		});
});

defineExpose<WidgetComponentExpose>({
	name,
	configure,
	id: props.widget?.id ?? null,
});
</script>

<style lang="scss" scoped>
.stream {
	display: flex;
	justify-content: center;
	flex-wrap: wrap;
	padding: 8px;

	> .img {
		position: relative;
		flex: 1 1 33%;
		width: 33%;
		height: 80px;
		box-sizing: border-box;
		background-position: center center;
		background-size: cover;
		border: solid 2px transparent;
		border-radius: 4px;
	}
}
</style>
