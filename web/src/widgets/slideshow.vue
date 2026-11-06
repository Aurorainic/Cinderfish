<!-- Ported from firefish@b425f61:packages/client/src/widgets/slideshow.vue (AGPL-3.0-only) -->
<template>
	<WidgetCard
		:show-header="false"
		:naked="widgetProps.transparent === true"
		:bare="true"
	>
		<div class="slideshow" :style="{ height: `${height}px` }">
			<CfLoading v-if="fetching" />
			<CfInfo v-else-if="me == null">登录后可以放映网盘里的图片。</CfInfo>
			<CfInfo v-else-if="failed">无法读取网盘文件。</CfInfo>
			<CfInfo v-else-if="images.length === 0">网盘里还没有图片。</CfInfo>
			<template v-else>
				<div
					v-for="(image, index) in images"
					:key="image.id"
					class="slide"
					:class="{ active: index === current }"
					:style="{ backgroundImage: `url(${resolveMediaUrl(image.url)})` }"
				></div>
				<a
					class="caption"
					:href="resolveMediaUrl(images[current].url)"
					target="_blank"
					rel="noopener noreferrer"
					:title="images[current].name"
					>{{ images[current].name }}</a
				>
			</template>
		</div>
	</WidgetCard>
</template>

<script setup lang="ts">
// Upstream picks files from a chosen drive folder and crossfades two layers.
// Cinderfish's folder picker does not exist, so this port uses the newest 24
// images from the user's drive and cycles through all of them with a CSS
// opacity transition. The `folderId` prop is therefore not offered.
import { computed, onMounted, ref } from "vue";
import WidgetCard from "./WidgetCard.vue";
import { CfInfo, CfLoading } from "@/components/ui";
import { api, resolveMediaUrl } from "@/api/client";
import { ENDPOINTS } from "@/api/endpoints";
import type { DriveFile } from "@/api/types";
import { me } from "@/account";
import { useInterval } from "@/composables/use-interval";
import { useWidgetPropsManager, type WidgetComponentExpose, type WidgetField } from "./index";
import type { Widget } from "@/store/widgets";

const name = "slideshow";

const props = defineProps<{ widget?: Widget }>();
const emit = defineEmits<{ updateProps: [data: Record<string, unknown>] }>();

const fields: WidgetField[] = [
	{ key: "height", label: "高度（px）", type: "number", default: 300, min: 120, max: 800, step: 20 },
	{ key: "intervalSec", label: "切换间隔（秒）", type: "number", default: 7, min: 2, max: 120, step: 1 },
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
const current = ref(0);

const height = computed(() => Number(widgetProps.height ?? 300));
const interval = computed(() => Math.max(2, Number(widgetProps.intervalSec ?? 7)) * 1000);

onMounted(() => {
	if (me.value == null) {
		fetching.value = false;
		return;
	}
	api(ENDPOINTS.driveFiles, { limit: 24 })
		.then((files) => {
			images.value = files.filter((file) => /^image\//.test(file.type));
		})
		.catch(() => {
			failed.value = true;
		})
		.finally(() => {
			fetching.value = false;
		});
});

useInterval(
	() => {
		if (images.value.length < 2) return;
		current.value = (current.value + 1) % images.value.length;
	},
	interval.value,
);

defineExpose<WidgetComponentExpose>({
	name,
	configure,
	id: props.widget?.id ?? null,
});
</script>

<style lang="scss" scoped>
.slideshow {
	position: relative;
	overflow: hidden;
	border-radius: var(--radius);
	background: var(--bg);

	.slide {
		position: absolute;
		inset: 0;
		background-position: center;
		background-size: cover;
		opacity: 0;
		transition: opacity 1s ease;
	}

	.slide.active {
		opacity: 1;
	}

	.caption {
		position: absolute;
		left: 0;
		right: 0;
		bottom: 0;
		padding: 6px 10px;
		font-size: 0.78em;
		color: #fff;
		background: linear-gradient(transparent, rgba(0, 0, 0, 0.55));
		text-decoration: none;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}
}
</style>
