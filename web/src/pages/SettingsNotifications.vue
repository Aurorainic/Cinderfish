<!-- Ported from firefish@b425f61:packages/client/src/pages/settings/notifications.vue (AGPL-3.0-only) -->
<template>
	<div class="_formRoot cf-settings-notifications">
		<section class="_formBlock">
			<h2 class="sectionTitle">{{ t("notificationSetting") }}</h2>

			<div v-if="loading" class="state"><CfLoading /></div>
			<div v-else-if="me == null" class="state">
				<CfInfo>{{ t("signinRequired") }}</CfInfo>
			</div>
			<div v-else-if="loadErrorText != null" class="state">
				<CfInfo warn>{{ loadErrorText }}</CfInfo>
				<CfButton class="retry" inline @click="load()">
					{{ t("retry") }}
				</CfButton>
			</div>
			<template v-else>
				<p class="desc">{{ t("notificationSettingDesc") }}</p>
				<div v-if="mutedTypes.length === 0" class="empty">
					当前没有屏蔽任何通知类型，全部通知都会显示。
				</div>
				<ul v-else class="types">
					<li v-for="type in mutedTypes" :key="type" class="type">
						{{ typeLabel(type) }}
					</li>
				</ul>
				<CfInfo class="note">
					通知类型由服务端的 <code>mutingNotificationTypes</code> 决定，通知列表会据此过滤。
					修改它需要 <code>i/update</code> 接受该字段，后端目前只接受
					name / description / avatarId / followersVisibility / followingVisibility，
					所以这里只读显示，修改功能<strong>未提供</strong>。
				</CfInfo>
			</template>
		</section>

		<section class="_formBlock">
			<h2 class="sectionTitle">{{ t("markAllAsRead") }}</h2>
			<div class="actions">
				<CfButton
					primary
					:wait="marking"
					:disabled="marking || me == null"
					@click="markAllAsRead()"
				>
					<i class="ph-check ph-bold ph-lg"></i>
					{{ t("markAsReadAllNotifications") }}
				</CfButton>
				<CfButton disabled>
					{{ t("markAsReadAllUnreadNotes") }}
				</CfButton>
				<CfButton disabled>
					{{ t("markAsReadAllTalkMessages") }}
				</CfButton>
			</div>
			<CfInfo v-if="actionErrorText != null" class="note" :warn="true">
				{{ actionErrorText }}
			</CfInfo>
			<CfInfo v-else class="note">
				「{{ t("markAsReadAllUnreadNotes") }}」与「{{ t("markAsReadAllTalkMessages") }}」
				所需的 <code>i/read-all-unread-notes</code> / <code>i/read-all-messaging-messages</code>
				端点<strong>未提供</strong>，因此保持禁用。
			</CfInfo>
		</section>

		<section class="_formBlock">
			<h2 class="sectionTitle">{{ t("clientSettings") }}</h2>
			<CfSwitch v-model="foldNotification" class="_formBlock">
				合并重复通知
				<template #caption>
					同一帖子收到的多条相同通知合并成一条显示。偏好写入本地 store 的
					<code>foldNotification</code>（Firefish 同名键）；通知列表尚未读取它，
					所以暂时只保存偏好、不改变渲染。
				</template>
			</CfSwitch>
			<CfInfo class="note">
				推送通知（<code>sw/update-registration</code>、<code>sw/*</code>）在本实例
				<strong>未提供</strong>，所以没有推送开关；通知设置窗口
				（<code>MkNotificationSettingWindow</code>）保存的也是同一个服务端字段，同样不可写。
			</CfInfo>
		</section>
	</div>
</template>

<script lang="ts" setup>
// Firefish's notification settings page writes `i/update { mutingNotificationTypes }`
// after the notification-type window closes, and offers three "mark all read"
// buttons. Only one of those has a Cinderfish endpoint:
//   * `notifications/mark-all-as-read` exists and is called for real;
//   * `i/read-all-unread-notes` / `i/read-all-messaging-messages` do not exist,
//     so those buttons stay disabled with an explicit note;
//   * `i/update` ignores `mutingNotificationTypes` (internal/httpapi/i_update.go),
//     so the type list is shown read-only instead of being written silently.
// The push-notification block is dropped entirely: `sw/*` is not implemented.
import { computed, onMounted, ref } from "vue";
import { me, updateMe } from "@/account";
import type { MeDetailed, NotificationType } from "@/api/types";
import { api } from "@/api/client";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import CfSwitch from "@/components/ui/CfSwitch.vue";
import { t } from "@/i18n";
import { success } from "@/os";
import { definePageMetadata } from "@/scripts/page-metadata";
import { defaultStore } from "@/store";

defineOptions({ name: "CfPageSettingsNotifications" });

// The backend produces exactly these five kinds (docs/research/api-contract).
const knownTypes: NotificationType[] = [
	"mention",
	"reply",
	"renote",
	"quote",
	"reaction",
];

const loading = ref(true);
const loadErrorText = ref<string | null>(null);
const actionErrorText = ref<string | null>(null);
const marking = ref(false);

const mutedTypes = computed<string[]>(() => me.value?.mutingNotificationTypes ?? []);

const foldNotification = computed<boolean>({
	get: () => defaultStore.reactiveState.foldNotification.value,
	set: (value) => defaultStore.set("foldNotification", value),
});

function typeLabel(type: string): string {
	return (knownTypes as string[]).includes(type)
		? t(`_notification._types.${type}`)
		: type;
}

async function load(): Promise<void> {
	loading.value = true;
	loadErrorText.value = null;
	try {
		const fresh = await api("i");
		updateMe(fresh as Partial<MeDetailed>);
	} catch (error) {
		loadErrorText.value =
			error instanceof Error && error.message !== ""
				? error.message
				: t("somethingHappened");
	} finally {
		loading.value = false;
	}
}

async function markAllAsRead(): Promise<void> {
	if (marking.value) return;
	marking.value = true;
	actionErrorText.value = null;
	try {
		await api("notifications/mark-all-as-read");
		updateMe({ hasUnreadNotification: false, unreadNotificationsCount: 0 });
		success();
	} catch (error) {
		actionErrorText.value =
			error instanceof Error && error.message !== ""
				? error.message
				: t("somethingHappened");
	} finally {
		marking.value = false;
	}
}

onMounted(() => {
	void load();
});

definePageMetadata({
	title: t("notifications"),
	icon: "ph-bell ph-bold ph-lg",
});
</script>

<style lang="scss" scoped>
.cf-settings-notifications {
	.sectionTitle {
		margin: 0 0 4px;
		color: var(--fg);
		font-size: 1em;
		opacity: 0.85;
	}

	.desc {
		margin: 8px 0;
		color: var(--fgTransparentWeak);
		font-size: 0.9em;
	}

	.state {
		display: flex;
		flex-direction: column;
		align-items: flex-start;
		gap: 12px;
		margin-block-start: 12px;
	}

	.empty {
		margin-block-start: 8px;
		color: var(--fgTransparentWeak);
		font-size: 0.9em;
	}

	.types {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
		margin: 12px 0 0;
		padding: 0;
		list-style: none;

		> .type {
			padding: 4px 10px;
			border-radius: var(--radius);
			background: var(--accentedBg);
			color: var(--accent);
			font-size: 0.9em;
		}
	}

	.actions {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
		margin-block-start: 12px;
	}

	.note {
		margin-block-start: 12px;
		font-size: 0.85em;

		code {
			word-break: break-all;
		}
	}
}
</style>
