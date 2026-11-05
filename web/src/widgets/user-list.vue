<!-- Ported from firefish@b425f61:packages/client/src/widgets/user-list.vue (AGPL-3.0-only) -->
<template>
	<WidgetCard
		:title="t('_widgets.userList')"
		icon="ph-bold ph-user-list"
		:show-header="widgetProps.showHeader === true"
	>
		<CfInfo v-if="me == null">登录后可以查看关注的用户。</CfInfo>
		<CfLoading v-else-if="fetching" />
		<CfInfo v-else-if="failed">无法读取关注列表，可能被设为不可见。</CfInfo>
		<CfInfo v-else-if="users.length === 0">还没有关注任何人。</CfInfo>
		<template v-else>
			<div class="users">
				<MkAvatar
					v-for="user in users"
					:key="user.id"
					class="avatar"
					:user="user"
					:show-indicator="true"
				/>
			</div>
			<p class="note">服务器未提供用户列表接口，这里显示你关注的人。</p>
		</template>
	</WidgetCard>
</template>

<script setup lang="ts">
// Firefish reads a named list (`users/lists/show`) and renders its members.
// Neither `users/lists/list` nor `users/lists/show` exists on Cinderfish, so the
// widget falls back to the real `users/following` endpoint and says so in the
// card footer instead of offering a picker that cannot work (contract §2).
import { onMounted, ref } from "vue";
import WidgetCard from "./WidgetCard.vue";
import { CfInfo, CfLoading } from "@/components/ui";
import { api } from "@/api/client";
import { ENDPOINTS } from "@/api/endpoints";
import type { UserLite } from "@/api/types";
import { me } from "@/account";
import { t } from "@/i18n";
import { useWidgetPropsManager, type WidgetComponentExpose, type WidgetField } from "./index";
import type { Widget } from "@/store/widgets";

const name = "userList";

const props = defineProps<{ widget?: Widget }>();
const emit = defineEmits<{ updateProps: [data: Record<string, unknown>] }>();

const fields: WidgetField[] = [
	{ key: "showHeader", label: "显示标题栏", type: "boolean", default: true },
];

const { widgetProps, configure } = useWidgetPropsManager(
	name,
	fields,
	props.widget,
	(data) => emit("updateProps", data),
);

const users = ref<UserLite[]>([]);
const fetching = ref(true);
const failed = ref(false);

onMounted(() => {
	const account = me.value;
	if (account == null) {
		fetching.value = false;
		return;
	}
	api(ENDPOINTS.usersFollowing, { userId: account.id, limit: 24 })
		.then((followings) => {
			users.value = followings
				.map((following) => following.followee)
				.filter((user): user is UserLite => user != null)
				.slice(0, 24);
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
.users {
	display: grid;
	grid-template-columns: repeat(auto-fill, minmax(32px, 1fr));
	gap: 8px;
	padding: 12px;

	> .avatar {
		width: 100%;
		aspect-ratio: 1;
	}
}

.note {
	margin: 0;
	padding: 0 12px 10px;
	font-size: 75%;
	color: var(--fgTransparentWeak);
}
</style>
