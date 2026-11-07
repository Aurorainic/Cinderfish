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
				<ul class="types">
					<li v-for="type in knownTypes" :key="type" class="type">
						<CfSwitch
							:model-value="!mutedTypes.includes(type)"
							:disabled="savingType"
							@update:model-value="(value: boolean) => setTypeEnabled(type, value)"
						>
							{{ typeLabel(type) }}
						</CfSwitch>
					</li>
				</ul>
				<CfInfo v-if="typeError != null" class="note" warn>
					{{ typeError }}
				</CfInfo>
				<CfInfo v-else class="note">
					关闭某一类通知后，服务端不再在 <code>notifications</code> 列表中返回它，
					未读角标也不再计入。设置写入 <code>i/update { mutingNotificationTypes }</code>。
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
					同一帖子上的同类通知（如同一个表情的多条回应、同一帖被多次转贴）合并成一条
					并显示条数；回复、提及这类每帖只会发生一次的通知不合并。偏好保存在本地
					store 的 <code>foldNotification</code>（Firefish 同名键）。合并后的行进入
					视口时会把整组标记已读。
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
//   * `i/update` stores `mutingNotificationTypes`; the list below toggles it
//     and the notification endpoint filters on the stored value.
// The push-notification block is dropped entirely: `sw/*` is not implemented.
import { computed, onMounted, ref } from "vue";
import { me, updateMe, updateProfile } from "@/account";
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
const savingType = ref(false);
const typeError = ref<string | null>(null);

const mutedTypes = computed<string[]>(() => me.value?.mutingNotificationTypes ?? []);

async function setTypeEnabled(
	type: NotificationType,
	enabled: boolean,
): Promise<void> {
	if (savingType.value) return;
	savingType.value = true;
	typeError.value = null;
	const muted = new Set(mutedTypes.value);
	if (enabled) muted.delete(type);
	else muted.add(type);
	const next = knownTypes.filter((candidate) => muted.has(candidate));
	try {
		await updateProfile({ mutingNotificationTypes: next });
	} catch (error) {
		typeError.value =
			error instanceof Error && error.message !== ""
				? error.message
				: t("somethingHappened");
	} finally {
		savingType.value = false;
	}
}

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
