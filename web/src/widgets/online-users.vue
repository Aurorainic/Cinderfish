<!-- Ported from firefish@b425f61:packages/client/src/widgets/online-users.vue (AGPL-3.0-only) -->
<template>
	<WidgetCard
		:show-header="false"
		:naked="widgetProps.transparent === true"
		:bare="true"
	>
		<div class="mkw-online-users">
			<span class="text">
				{{ count === null ? "—" : t("onlineUsersCount", { n: count }) }}
			</span>
		</div>
	</WidgetCard>
</template>

<script setup lang="ts">
// Firefish polls get-online-users-count every 15 seconds and prints the count
// through the `onlineUsersCount` message. Cinderfish counts accounts whose last
// authenticated request is younger than Misskey's ten-minute window.
import { ref } from "vue";
import WidgetCard from "./WidgetCard.vue";
import { api } from "@/api/client";
import { useInterval } from "@/composables/use-interval";
import { t } from "@/i18n";
import { useWidgetPropsManager, type WidgetComponentExpose, type WidgetField } from "./index";
import type { Widget } from "@/store/widgets";

const name = "onlineUsers";

const props = defineProps<{ widget?: Widget }>();
const emit = defineEmits<{ updateProps: [data: Record<string, unknown>] }>();

const fields: WidgetField[] = [
	{ key: "transparent", label: "透明背景", type: "boolean", default: true },
];

const { widgetProps, configure } = useWidgetPropsManager(
	name,
	fields,
	props.widget,
	(data) => emit("updateProps", data),
);

const count = ref<number | null>(null);

function tick(): void {
	api<{ count: number }>("get-online-users-count")
		.then((result) => {
			count.value = result.count;
		})
		.catch(() => {
			// Keep the last value; the next tick retries.
		});
}

useInterval(tick, 1000 * 15, { immediate: true, afterMounted: true });

defineExpose<WidgetComponentExpose>({
	name,
	configure,
	id: props.widget?.id ?? null,
});
</script>

<style lang="scss" scoped>
.mkw-online-users {
	padding: 16px 0;
	text-align: center;

	> .text {
		opacity: 0.85;
	}
}
</style>
