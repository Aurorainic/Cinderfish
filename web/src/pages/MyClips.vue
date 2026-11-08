<!-- Ported from firefish@b425f61:packages/client/src/pages/my-clips/index.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader
				:actions="headerActions"
				:tabs="headerTabs"
				:display-back-button="true"
			/>
		</template>
		<MkSpacer :content-max="700" :margin-min="16" :margin-max="32">
			<div class="cf-my-clips">
				<div class="_formBlock">
					<CfButton primary @click="startCreate">
						<i class="ph-plus ph-bold ph-lg"></i>
						{{ t("createNewClip") }}
					</CfButton>
				</div>

				<div v-if="creating" class="_formBlock createForm _panel">
					<CfInput
						v-model="newName"
						:placeholder="t('name')"
						@enter="create"
					>
						<template #label>{{ t("name") }}</template>
					</CfInput>
					<CfTextarea
						v-model="newDescription"
						:placeholder="t('optional')"
					>
						<template #label>{{ t("description") }}</template>
					</CfTextarea>
					<CfSwitch v-model="newIsPublic" class="_formBlock">
						{{ t("public") }}
					</CfSwitch>
					<div class="actions">
						<CfButton
							primary
							:disabled="busy || newName.trim() === ''"
							@click="create"
						>
							{{ t("createNewClip") }}
						</CfButton>
						<CfButton :disabled="busy" @click="cancelCreate">
							{{ t("cancel") }}
						</CfButton>
					</div>
				</div>

				<CfPagination
					ref="pagingComponent"
					:pagination="pagination"
					class="list"
				>
					<template #empty>
						<CfInfo class="_formBlock">{{ t("clipsDesc") }}</CfInfo>
					</template>
					<template #default="{ items }">
						<CfA
							v-for="item in asClips(items)"
							:key="item.id"
							:to="`/clips/${item.id}`"
							class="item _panel"
						>
							<b>{{ item.name }}</b>
							<div v-if="item.description" class="description">
								{{ item.description }}
							</div>
							<div class="meta">
								<span v-if="item.isPublic" class="badge">{{
									t("public")
								}}</span>
								<span v-if="item.notesCount != null" class="count">
									{{ item.notesCount }} {{ t("notes") }}
								</span>
							</div>
						</CfA>
					</template>
				</CfPagination>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script setup lang="ts">
// Firefish pages the caller's own clips with MkPagination. The endpoint is a
// plain cursor list, so CfPagination maps directly. Creation uses an inline
// form instead of Firefish's `os.form` (the port has no form helper); the
// description is omitted when blank, which leaves the stored value unset.
import { computed, ref } from "vue";
import { api } from "@/api/client";
import { ENDPOINTS } from "@/api/endpoints";
import type { Clip } from "@/api/types";
import type { Paging } from "@/components/ui";
import CfA from "@/components/ui/CfA.vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfInput from "@/components/ui/CfInput.vue";
import CfPagination from "@/components/ui/CfPagination.vue";
import CfSwitch from "@/components/ui/CfSwitch.vue";
import CfTextarea from "@/components/ui/CfTextarea.vue";
import { t } from "@/i18n";
import * as os from "@/os";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageMyClips" });

const pagination: Paging = {
	endpoint: ENDPOINTS.clipsList,
	limit: 10,
};

const pagingComponent = ref<{ reload: () => Promise<unknown> } | null>(null);

const creating = ref(false);
const newName = ref("");
const newDescription = ref("");
const newIsPublic = ref(false);
const busy = ref(false);

const headerTabs = computed(() => []);

const headerActions = computed(() => [
	{
		icon: "ph-plus ph-bold ph-lg",
		text: t("createNewClip"),
		handler: () => startCreate(),
	},
]);

/** CfPagination's slot is generic over `{ id: string }`; this endpoint is always clips. */
function asClips(items: { id: string }[]): Clip[] {
	return items as Clip[];
}

function errorText(cause: unknown): string {
	return cause instanceof Error ? cause.message : String(cause);
}

function startCreate(): void {
	creating.value = true;
	newName.value = "";
	newDescription.value = "";
	newIsPublic.value = false;
}

function cancelCreate(): void {
	creating.value = false;
	newName.value = "";
	newDescription.value = "";
	newIsPublic.value = false;
}

async function create(): Promise<void> {
	const name = newName.value.trim();
	if (busy.value || name === "") return;
	busy.value = true;
	try {
		await api(ENDPOINTS.clipsCreate, {
			name,
			isPublic: newIsPublic.value,
			...(newDescription.value.trim() === ""
				? {}
				: { description: newDescription.value.trim() }),
		});
		cancelCreate();
		os.success();
		await pagingComponent.value?.reload();
	} catch (cause) {
		os.toast(errorText(cause), "error");
	} finally {
		busy.value = false;
	}
}

definePageMetadata(
	computed(() => ({
		title: t("clip"),
		icon: "ph-paperclip ph-bold",
	})),
);
</script>

<style lang="scss" scoped>
.cf-my-clips {
	.createForm {
		display: flex;
		flex-direction: column;
		gap: 10px;
	}

	.actions {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
	}

	> .list {
		> .item {
			display: block;
			padding: 16px;
			margin-block-end: var(--margin);

			&:hover {
				border-color: var(--accent);
				text-decoration: none;
			}

			> .description {
				margin-block-start: 8px;
				padding-block-start: 8px;
				border-block-start: solid 0.5px var(--divider);
				white-space: pre-wrap;
				overflow-wrap: anywhere;
			}

			> .meta {
				display: flex;
				align-items: center;
				gap: 8px;
				margin-block-start: 8px;
				font-size: 0.82em;
				color: var(--fgTransparentWeak);
			}

			.badge {
				padding: 1px 6px;
				border: solid 1px var(--divider);
				border-radius: 999px;
			}
		}
	}
}
</style>
