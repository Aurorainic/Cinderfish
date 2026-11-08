<!-- Ported from firefish@b425f61:packages/client/src/pages/clip.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader
				:actions="headerActions"
				:tabs="headerTabs"
				:display-back-button="true"
			/>
		</template>
		<MkSpacer :content-max="800" :margin-min="16" :margin-max="32">
			<div v-if="clip" class="cf-clip">
				<div class="header _panel">
					<div v-if="clip.description" class="description">
						<Mfm :text="clip.description" :is-note="false" />
					</div>
					<div class="user">
						<MkAvatar
							:user="clip.user"
							class="avatar"
							:show-indicator="true"
						/>
						<MkA :to="userPage(clip.user)" class="userName">
							<MkUserName :user="clip.user" :nowrap="false" />
						</MkA>
						<span class="grow" />
						<span v-if="clip.isPublic" class="badge">{{
							t("public")
						}}</span>
						<CfButton
							v-if="showFavorite"
							class="favorite"
							:primary="clip.isFavorited === true"
							@click="toggleFavorite"
						>
							<i class="ph-star ph-bold ph-fill"></i>
							{{ clip.isFavorited ? t("unfavorite") : t("favorite") }}
							<span v-if="clip.favoritedCount > 0" class="count">{{
								clip.favoritedCount
							}}</span>
						</CfButton>
					</div>
				</div>

				<!-- Firefish edits through `os.form`; the port has no form helper,
				     so the same fields are an inline panel. -->
				<div v-if="editing" class="_formBlock editForm _panel">
					<CfInput v-model="editName" :placeholder="t('name')">
						<template #label>{{ t("name") }}</template>
					</CfInput>
					<CfTextarea
						v-model="editDescription"
						:placeholder="t('optional')"
					>
						<template #label>{{ t("description") }}</template>
					</CfTextarea>
					<CfSwitch v-model="editIsPublic">
						{{ t("public") }}
					</CfSwitch>
					<div class="actions">
						<CfButton
							primary
							:disabled="saving || editName.trim() === ''"
							@click="saveEdit"
						>
							{{ t("save") }}
						</CfButton>
						<CfButton :disabled="saving" @click="cancelEdit">
							{{ t("cancel") }}
						</CfButton>
					</div>
				</div>

				<MkNotes
					:key="`clip:${clipId}:${notesRefresh}`"
					:pagination="pagination"
				/>
			</div>
			<MkError v-else-if="error != null" @retry="fetchClip" />
			<MkLoading v-else />
		</MkSpacer>
	</MkStickyContainer>
</template>

<script setup lang="ts">
// Firefish renders `clips/notes` through MkNotes, and this port does the same:
// the endpoint returns base Notes, so the page must not force the detailed
// renderer. The clip channel carries `updated` / `deleted` events, which keep
// the header in sync and send a deleted clip's readers away.
import { computed, onUnmounted, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { api } from "@/api/client";
import { ENDPOINTS } from "@/api/endpoints";
import { useStream, type StreamChannel } from "@/api/stream";
import type { Clip } from "@/api/types";
import { me } from "@/account";
import { userPage } from "@/filters/user";
import type { Paging } from "@/components/ui";
import CfButton from "@/components/ui/CfButton.vue";
import CfInput from "@/components/ui/CfInput.vue";
import CfSwitch from "@/components/ui/CfSwitch.vue";
import CfTextarea from "@/components/ui/CfTextarea.vue";
import MkNotes from "@/components/MkNotes.vue";
import { t } from "@/i18n";
import * as os from "@/os";
import { router } from "@/router";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageClip" });

const route = useRoute();

const clipId = computed<string>(() => {
	const value = route.params.clipId;
	return typeof value === "string" ? value : "";
});

const clip = ref<Clip | null>(null);
const error = ref<unknown>(null);

const isSignedIn = computed(() => me.value != null);
const isOwner = computed(
	() => me.value != null && clip.value != null && me.value.id === clip.value.userId,
);
const showFavorite = computed(
	() => clip.value != null && clip.value.isPublic && isSignedIn.value,
);

const pagination = computed<Paging>(() => ({
	endpoint: ENDPOINTS.clipsNotes,
	limit: 10,
	params: { clipId: clipId.value },
}));

const headerTabs = computed(() => []);

const editing = ref(false);
const editName = ref("");
const editDescription = ref("");
const editIsPublic = ref(false);
const saving = ref(false);
const notesRefresh = ref(0);

const headerActions = computed<
	{ text: string; icon: string; handler: () => void }[]
>(() => {
	if (!isOwner.value || clip.value == null) return [];
	return [
		{
			icon: "ph-pencil-simple ph-bold ph-lg",
			text: t("edit"),
			handler: startEdit,
		},
		{
			icon: "ph-trash ph-bold ph-lg",
			text: t("delete"),
			handler: () => void deleteClip(),
		},
	];
});

function errorText(cause: unknown): string {
	return cause instanceof Error ? cause.message : String(cause);
}

async function fetchClip(): Promise<void> {
	if (clipId.value === "") return;
	error.value = null;
	try {
		clip.value = await api(ENDPOINTS.clipsShow, { clipId: clipId.value });
		editing.value = false;
	} catch (cause) {
		clip.value = null;
		error.value = cause;
	}
}

function startEdit(): void {
	if (clip.value == null) return;
	editName.value = clip.value.name;
	editDescription.value = clip.value.description ?? "";
	editIsPublic.value = clip.value.isPublic;
	editing.value = true;
}

function cancelEdit(): void {
	editing.value = false;
}

async function saveEdit(): Promise<void> {
	const current = clip.value;
	const name = editName.value.trim();
	if (current == null || saving.value || name === "") return;
	saving.value = true;
	try {
		const description = editDescription.value.trim();
		// clips/update treats an omitted description as "unchanged"; "" clears it.
		clip.value = await api(ENDPOINTS.clipsUpdate, {
			clipId: current.id,
			name,
			isPublic: editIsPublic.value,
			description: description === "" ? null : description,
		});
		editing.value = false;
		os.success();
	} catch (cause) {
		os.toast(errorText(cause), "error");
	} finally {
		saving.value = false;
	}
}

async function deleteClip(): Promise<void> {
	const current = clip.value;
	if (current == null) return;
	const { canceled } = await os.confirm({
		type: "warning",
		text: t("deleteAreYouSure", { x: current.name }),
	});
	if (canceled) return;
	try {
		await api(ENDPOINTS.clipsDelete, { clipId: current.id });
		await router.replace("/my/clips");
	} catch (cause) {
		os.toast(errorText(cause), "error");
	}
}

async function toggleFavorite(): Promise<void> {
	const current = clip.value;
	if (current == null) return;
	const favorited = current.isFavorited === true;
	try {
		await api(
			favorited ? ENDPOINTS.clipsUnfavorite : ENDPOINTS.clipsFavorite,
			{ clipId: current.id },
		);
		clip.value = {
			...current,
			isFavorited: !favorited,
			favoritedCount: Math.max(
				0,
				current.favoritedCount + (favorited ? -1 : 1),
			),
		};
	} catch (cause) {
		os.toast(errorText(cause), "error");
	}
}

let connection: StreamChannel | null = null;

function subscribe(id: string): void {
	connection?.dispose();
	connection = null;
	if (id === "") return;
	const channel = useStream().useChannel("clip", { clipId: id });
	channel.on("updated", () => {
		void fetchClip();
		notesRefresh.value += 1;
	});
	channel.on("deleted", () => {
		// The owner lands on their clip manager; everyone else goes home rather
		// than being sent to a login-only page.
		void router.replace(isOwner.value ? "/my/clips" : "/");
	});
	connection = channel;
}

watch(
	clipId,
	(id) => {
		clip.value = null;
		editing.value = false;
		notesRefresh.value += 1;
		subscribe(id);
		void fetchClip();
	},
	{ immediate: true },
);

onUnmounted(() => {
	connection?.dispose();
	connection = null;
});

definePageMetadata(
	computed(() =>
		clip.value == null
			? null
			: {
					title: clip.value.name,
					icon: "ph-paperclip ph-bold",
				},
	),
);
</script>

<style lang="scss" scoped>
.cf-clip {
	> .header {
		margin-block-end: var(--margin);

		> .description {
			padding: 16px;
			white-space: pre-wrap;
			overflow-wrap: anywhere;
		}

		> .description + .user {
			border-block-start: solid 0.5px var(--divider);
		}

		> .user {
			display: flex;
			align-items: center;
			gap: 8px;
			padding: 16px;

			> .avatar {
				inline-size: 32px;
				block-size: 32px;
				flex: 0 0 auto;
			}

			> .userName {
				min-width: 0;
				font-weight: bold;
			}

			> .grow {
				flex: 1;
			}

			> .badge {
				padding: 1px 8px;
				border: solid 1px var(--divider);
				border-radius: 999px;
				font-size: 0.82em;
				color: var(--fgTransparentWeak);
			}

			> .favorite .count {
				margin-inline-start: 4px;
				opacity: 0.8;
			}
		}
	}

	> .editForm {
		display: flex;
		flex-direction: column;
		gap: 10px;
		margin-block-end: var(--margin);

		.actions {
			display: flex;
			flex-wrap: wrap;
			gap: 8px;
		}
	}
}
</style>
