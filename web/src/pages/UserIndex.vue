<!-- Ported from firefish@b425f61:packages/client/src/pages/user/index.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader
				:tab="tab"
				:actions="headerActions"
				:tabs="headerTabs"
				:display-back-button="true"
				@update:tab="onTabUpdate"
			/>
		</template>
		<div>
			<transition name="fade" mode="out-in">
				<div v-if="user" :key="tab">
					<UserHome
						v-if="tab === 'notes'"
						:user="user"
						@refresh="fetchUser"
					/>
					<MkSpacer v-else-if="tab === 'replies'" :content-max="800">
						<UserNotes :user="user" :with-replies="true" />
					</MkSpacer>
					<MkSpacer v-else-if="tab === 'media'" :content-max="800">
						<UserPhotos :user="user" paginated />
					</MkSpacer>
					<MkSpacer v-else-if="tab === 'activity'" :content-max="800">
						<UserActivity :user="user" />
					</MkSpacer>
					<MkSpacer v-else-if="tab === 'reactions'" :content-max="800">
						<UserReactions :user="user" />
					</MkSpacer>
				</div>
				<MkError v-else-if="error != null" @retry="fetchUser" />
				<MkLoading v-else />
			</transition>
		</div>
	</MkStickyContainer>
</template>

<script lang="ts" setup>
// Firefish's tab set is home / reactions / clips / pages / gallery, gated on
// `publicReactions` and on the user being local. Cinderfish has no clips, pages
// or gallery endpoints, so the five tabs here are the ones the API can feed:
//   notes      users/notes { withReplies: false }  + pinned notes block
//   replies    users/notes { withReplies: true }   (backend has no reply-only filter)
//   media      users/notes { withFiles: true }
//   activity   no charts/user/notes endpoint -> stated as unavailable
//   reactions  no users/reactions endpoint     -> stated as unavailable
// `users/show` only serves local accounts (contract §5), so a remote `acct` is
// rejected by the route wrapper (UserProfile.vue) before this page mounts.
import { computed, ref, watch } from "vue";
import type { UserDetailed } from "@/api/types";
import { api } from "@/api/client";
import UserActivity from "@/components/user/UserActivity.vue";
import UserHome from "@/components/user/UserHome.vue";
import UserNotes from "@/components/user/UserNotes.vue";
import UserPhotos from "@/components/user/UserPhotos.vue";
import UserReactions from "@/components/user/UserReactions.vue";
import { acct as formatAcct } from "@/filters/user";
import { t } from "@/i18n";
import { definePageMetadata } from "@/scripts/page-metadata";
import { getUserMenu } from "@/scripts/get-user-menu";
import { useRouter } from "vue-router";

type TabKey = "notes" | "replies" | "media" | "activity" | "reactions";

interface TabDefinition {
	key: TabKey;
	title: string;
	icon: string;
}

const TAB_KEYS: readonly TabKey[] = [
	"notes",
	"replies",
	"media",
	"activity",
	"reactions",
];

const props = withDefaults(
	defineProps<{
		acct: string;
		page?: string;
	}>(),
	{
		page: "notes",
	},
);

const router = useRouter();

function toTabKey(page: string | undefined): TabKey {
	if (page != null && (TAB_KEYS as readonly string[]).includes(page)) {
		return page as TabKey;
	}
	// Firefish's default sub-page is "home"; the router may also spell it
	// "index". Both mean the notes tab.
	return "notes";
}

const tab = ref<TabKey>(toTabKey(props.page));
const user = ref<UserDetailed | null>(null);
const error = ref<unknown>(null);

const headerTabs = computed<TabDefinition[]>(() => [
	{ key: "notes", title: t("notes"), icon: "ph-file-text ph-bold ph-lg" },
	{ key: "replies", title: t("reply"), icon: "ph-chat-circle ph-bold ph-lg" },
	{ key: "media", title: t("files"), icon: "ph-image-square ph-bold ph-lg" },
	{ key: "activity", title: t("activity"), icon: "ph-chart-bar ph-bold ph-lg" },
	{ key: "reactions", title: t("reaction"), icon: "ph-smiley ph-bold ph-lg" },
]);

const headerActions = computed(() => [
	{
		text: t("menu"),
		icon: "ph-dots-three ph-bold ph-lg",
		handler: (ev: MouseEvent) => {
			if (user.value != null) getUserMenu(user.value, ev);
		},
	},
]);

function tabPath(key: TabKey): string {
	return key === "notes" ? `/@${props.acct}` : `/@${props.acct}/${key}`;
}

function onTabUpdate(key: string): void {
	const next = toTabKey(key);
	if (next === tab.value) return;
	tab.value = next;
	void router.replace(tabPath(next));
}

async function fetchUser(): Promise<void> {
	if (props.acct === "") return;
	error.value = null;
	user.value = null;
	try {
		const username = props.acct.includes("@")
			? props.acct.slice(0, props.acct.indexOf("@"))
			: props.acct;
		user.value = await api("users/show", { username });
	} catch (cause) {
		error.value = cause;
	}
}

watch(
	() => props.acct,
	() => {
		tab.value = toTabKey(props.page);
		void fetchUser();
	},
	{ immediate: true },
);

watch(
	() => props.page,
	(page) => {
		tab.value = toTabKey(page);
	},
);

definePageMetadata(
	computed(() =>
		user.value == null
			? {
					title: `@${props.acct}`,
					icon: "ph-user ph-bold ph-lg",
				}
			: {
					icon: "ph-user ph-bold ph-lg",
					title: user.value.name
						? `${user.value.name} (@${user.value.username})`
						: `@${user.value.username}`,
					subtitle: `@${formatAcct(user.value)}`,
					userName: user.value,
					avatar: user.value,
				},
	),
);
</script>

<style lang="scss" scoped>
.fade-enter-active,
.fade-leave-active {
	transition: opacity 0.125s ease;
}

.fade-enter-from,
.fade-leave-to {
	opacity: 0;
}
</style>
