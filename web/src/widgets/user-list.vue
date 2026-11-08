<!-- Ported from firefish@b425f61:packages/client/src/widgets/user-list.vue (AGPL-3.0-only) -->
<template>
	<WidgetCard
		:title="t('_widgets.userList')"
		icon="ph-bold ph-user-list"
		:show-header="widgetProps.showHeader === true"
	>
		<CfInfo v-if="me == null">登录后可以查看用户列表。</CfInfo>
		<CfLoading v-else-if="fetching" />
		<CfInfo v-else-if="failed">无法读取用户列表。</CfInfo>
		<template v-else>
			<div v-if="lists.length > 0" class="picker">
				<CfSelect
					:model-value="selectedListId"
					@update:model-value="onSelect"
				>
					<option
						v-for="list in lists"
						:key="list.id"
						:value="list.id"
					>
						{{ list.name }}
					</option>
				</CfSelect>
			</div>
			<div v-if="users.length > 0" class="users">
				<MkAvatar
					v-for="user in users"
					:key="user.id"
					class="avatar"
					:user="user"
					:show-indicator="true"
				/>
			</div>
			<CfInfo v-else class="empty">
				{{ lists.length === 0 ? t("noLists") : t("nothing") }}
			</CfInfo>
			<p v-if="lists.length === 0" class="note">
				服务器上没有用户列表，这里显示你关注的人。
			</p>
		</template>
	</WidgetCard>
</template>

<script setup lang="ts">
// Firefish reads one named list (`users/lists/show`) and renders its members.
// Cinderfish's `users/lists/get-memberships` already carries the user object,
// so the widget needs no second hydration call. When the account has no lists
// at all the widget keeps the old `users/following` fallback and explains why.
import { onMounted, ref } from "vue";
import WidgetCard from "./WidgetCard.vue";
import MkAvatar from "@/components/global/MkAvatar.vue";
import { CfInfo, CfLoading, CfSelect } from "@/components/ui";
import { api } from "@/api/client";
import { ENDPOINTS } from "@/api/endpoints";
import type { UserLite, UserList } from "@/api/types";
import { me } from "@/account";
import { t } from "@/i18n";
import { useWidgetPropsManager, type WidgetComponentExpose, type WidgetField } from "./index";
import type { Widget } from "@/store/widgets";

const name = "userList";

const props = defineProps<{ widget?: Widget }>();
const emit = defineEmits<{ updateProps: [data: Record<string, unknown>] }>();

const fields: WidgetField[] = [
	{ key: "showHeader", label: "显示标题栏", type: "boolean", default: true },
	{ key: "listId", label: "列表 ID（留空则使用第一个列表）", type: "text", default: "" },
];

const { widgetProps, configure, save } = useWidgetPropsManager(
	name,
	fields,
	props.widget,
	(data) => emit("updateProps", data),
);

const lists = ref<UserList[]>([]);
const users = ref<UserLite[]>([]);
const selectedListId = ref("");
const fetching = ref(true);
const failed = ref(false);

async function loadMembers(listId: string): Promise<void> {
	const memberships = await api(ENDPOINTS.usersListsGetMemberships, {
		listId,
		limit: 24,
	});
	users.value = memberships
		.map((membership) => membership.user)
		.slice(0, 24);
}

async function onSelect(value: string | null): Promise<void> {
	if (value == null || value === selectedListId.value) return;
	selectedListId.value = value;
	widgetProps.listId = value;
	save();
	fetching.value = true;
	failed.value = false;
	try {
		await loadMembers(value);
	} catch {
		users.value = [];
		failed.value = true;
	} finally {
		fetching.value = false;
	}
}

onMounted(async () => {
	const account = me.value;
	if (account == null) {
		fetching.value = false;
		return;
	}
	try {
		lists.value = await api(ENDPOINTS.usersListsList, {});
		if (lists.value.length > 0) {
			const saved = String(widgetProps.listId ?? "");
			const chosen =
				lists.value.find((list) => list.id === saved) ?? lists.value[0];
			selectedListId.value = chosen.id;
			widgetProps.listId = chosen.id;
			await loadMembers(chosen.id);
			return;
		}
		// No lists: keep the old following-based content rather than an empty
		// card, and say so in the footer.
		const followings = await api(ENDPOINTS.usersFollowing, {
			userId: account.id,
			limit: 24,
		});
		users.value = followings
			.map((following) => following.followee)
			.filter((user): user is UserLite => user != null)
			.slice(0, 24);
	} catch {
		failed.value = true;
	} finally {
		fetching.value = false;
	}
});

defineExpose<WidgetComponentExpose>({
	name,
	configure,
	id: props.widget?.id ?? null,
});
</script>

<style lang="scss" scoped>
.picker {
	padding: 10px 12px 0;
}

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

.empty {
	margin: 12px;
}

.note {
	margin: 0;
	padding: 0 12px 10px;
	font-size: 75%;
	color: var(--fgTransparentWeak);
}
</style>
