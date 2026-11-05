<!-- Ported from firefish@b425f61:packages/client/src/pages/settings/privacy.vue (AGPL-3.0-only) -->
<template>
	<div class="_formRoot cf-settings-privacy">
		<CfInfo v-if="me == null" warn>{{ t("signinRequired") }}</CfInfo>

		<template v-else>
			<section class="_formBlock">
				<h2 class="sectionTitle">{{ t("ffVisibility") }}</h2>
				<CfSelect
					:model-value="followersVisibility"
					:disabled="saving"
					@update:model-value="onFollowersVisibility"
				>
					<template #label>{{ t("followers") }}</template>
					<option value="public">{{ t("_ffVisibility.public") }}</option>
					<option value="followers">{{ t("_ffVisibility.followers") }}</option>
					<option value="private">{{ t("_ffVisibility.private") }}</option>
					<template #caption>{{ t("ffVisibilityDescription") }}</template>
				</CfSelect>

				<CfSelect
					class="field"
					:model-value="followingVisibility"
					:disabled="saving"
					@update:model-value="onFollowingVisibility"
				>
					<template #label>{{ t("following") }}</template>
					<option value="public">{{ t("_ffVisibility.public") }}</option>
					<option value="followers">{{ t("_ffVisibility.followers") }}</option>
					<option value="private">{{ t("_ffVisibility.private") }}</option>
					<template #caption>{{ t("ffVisibilityDescription") }}</template>
				</CfSelect>

				<p class="status">
					<CfLoading v-if="saving" em />
					<span v-else-if="saved">{{ t("saved") }}</span>
				</p>

				<CfInfo v-if="saveError != null" warn class="error">
					<span class="errorText">{{ saveError }}</span>
					<CfButton inline @click="save">{{ t("save") }}</CfButton>
				</CfInfo>
			</section>

			<section class="_formBlock">
				<h2 class="sectionTitle">{{ t("note") }}</h2>
				<CfSwitch v-model="rememberNoteVisibility" class="_formBlock">
					{{ t("rememberNoteVisibility") }}
				</CfSwitch>

				<template v-if="!rememberNoteVisibility">
					<CfSelect
						:model-value="defaultNoteVisibility"
						@update:model-value="onDefaultNoteVisibility"
					>
						<template #label>{{ t("defaultNoteVisibility") }}</template>
						<option value="public">{{ t("_visibility.public") }}</option>
						<option value="home">{{ t("_visibility.home") }}</option>
						<option value="followers">
							{{ t("_visibility.followers") }}
						</option>
						<option value="specified">
							{{ t("_visibility.specified") }}
						</option>
					</CfSelect>
					<CfSwitch v-model="defaultNoteLocalOnly" class="_formBlock">
						{{ t("_visibility.localOnly") }}
					</CfSwitch>
				</template>

				<CfSwitch v-model="keepCw" class="_formBlock">
					{{ t("keepCw") }}
				</CfSwitch>
			</section>

			<section class="_formBlock">
				<h2 class="sectionTitle">{{ t("other") }}</h2>
				<ul class="unavailableList">
					<li v-for="item in unavailableItems" :key="item.title" class="item">
						<div class="row">
							<span class="name">{{ item.title }}</span>
							<span class="badge">{{ t("unavailable") }}</span>
						</div>
						<p v-if="item.description != null" class="description">
							{{ item.description }}
						</p>
					</li>
				</ul>
			</section>
		</template>
	</div>
</template>

<script lang="ts" setup>
// Firefish's privacy page writes eight profile switches through `i/update`.
// Cinderfish only stores two of them (followersVisibility / followingVisibility,
// docs/research/api-contract/auth-users.md §POST /api/i/update); the rest are
// silently ignored by the backend and read back as constants, so they are listed
// as unavailable instead of being rendered as dead switches.
import { computed, ref, watch } from "vue";
import type { FollowVisibility, NoteVisibility } from "@/api/types";
import { me, updateProfile } from "@/account";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import CfSelect from "@/components/ui/CfSelect.vue";
import CfSwitch from "@/components/ui/CfSwitch.vue";
import { t } from "@/i18n";
import { definePageMetadata } from "@/scripts/page-metadata";
import { defaultStore } from "@/store";

defineOptions({ name: "CfPageSettingsPrivacy" });

type StoreKey = keyof typeof defaultStore.state & string;
type StoreValue<K extends StoreKey> = (typeof defaultStore.state)[K];

function storeRef<K extends StoreKey>(key: K) {
	return computed<StoreValue<K>>({
		get: () => defaultStore.reactiveState[key].value as StoreValue<K>,
		set: (value) => defaultStore.set(key, value),
	});
}

// The note-visibility block is local (defaultStore) in Firefish and stays local
// here; only the two ffVisibility fields go to the server.
const rememberNoteVisibility = storeRef("rememberNoteVisibility");
const defaultNoteVisibility = storeRef("defaultNoteVisibility");
const defaultNoteLocalOnly = storeRef("defaultNoteLocalOnly");
const keepCw = storeRef("keepCw");

const noteVisibilityOptions: NoteVisibility[] = [
	"public",
	"home",
	"followers",
	"specified",
];

function onDefaultNoteVisibility(value: string | null): void {
	if (value == null) return;
	if (!noteVisibilityOptions.includes(value as NoteVisibility)) return;
	defaultNoteVisibility.value = value as NoteVisibility;
}

const visibilityOptions: FollowVisibility[] = ["public", "followers", "private"];

function currentVisibility(
	value: FollowVisibility | undefined,
): FollowVisibility {
	return value != null && visibilityOptions.includes(value) ? value : "public";
}

const followersVisibility = ref<FollowVisibility>(
	currentVisibility(me.value?.followersVisibility),
);
const followingVisibility = ref<FollowVisibility>(
	currentVisibility(me.value?.followingVisibility),
);

const saving = ref(false);
const saved = ref(false);
const saveError = ref<string | null>(null);

watch(me, (account) => {
	followersVisibility.value = currentVisibility(account?.followersVisibility);
	followingVisibility.value = currentVisibility(account?.followingVisibility);
});

interface UnavailableItem {
	title: string;
	description?: string;
}

const unavailableItems = computed<UnavailableItem[]>(() => [
	{
		title: t("makeFollowManuallyApprove"),
		description: t("lockedAccountInfo"),
	},
	{ title: t("autoAcceptFollowed") },
	{
		title: t("makeReactionsPublic"),
		description: t("makeReactionsPublicDescription"),
	},
	{
		title: t("hideOnlineStatus"),
		description: t("hideOnlineStatusDescription"),
	},
	{ title: t("noCrawle"), description: t("noCrawleDescription") },
	{
		title: t("preventAiLearning"),
		description: t("preventAiLearningDescription"),
	},
	{
		title: t("makeExplorable"),
		description: t("makeExplorableDescription"),
	},
]);

async function save(): Promise<void> {
	if (saving.value || me.value == null) return;
	saving.value = true;
	saved.value = false;
	saveError.value = null;
	try {
		await updateProfile({
			followersVisibility: followersVisibility.value,
			followingVisibility: followingVisibility.value,
		});
		saved.value = true;
	} catch (error) {
		saveError.value = error instanceof Error ? error.message : String(error);
	} finally {
		saving.value = false;
	}
}

function onFollowersVisibility(value: string | null): void {
	if (value == null || !visibilityOptions.includes(value as FollowVisibility)) {
		return;
	}
	followersVisibility.value = value as FollowVisibility;
	void save();
}

function onFollowingVisibility(value: string | null): void {
	if (value == null || !visibilityOptions.includes(value as FollowVisibility)) {
		return;
	}
	followingVisibility.value = value as FollowVisibility;
	void save();
}

definePageMetadata({ title: t("privacy"), icon: "ph-keyhole ph-bold ph-lg" });
</script>

<style lang="scss" scoped>
.cf-settings-privacy {
	.sectionTitle {
		margin: 0 0 12px;
		color: var(--fg);
		font-size: 1em;
		opacity: 0.85;
	}

	.field {
		margin-block-start: 12px;
	}

	.status {
		min-block-size: 20px;
		margin: 12px 0 0;
		color: var(--fgTransparentWeak);
		font-size: 0.85em;
	}

	.error {
		margin-block-start: 8px;

		.errorText {
			flex: 1;
			min-inline-size: 0;
		}
	}

	.unavailableList {
		margin: 0;
		padding: 0;
		list-style: none;
		border: solid 1px var(--divider);
		border-radius: var(--radius);

		> .item {
			padding: 10px 14px;

			&:not(:first-child) {
				border-block-start: solid 1px var(--divider);
			}
		}

		.row {
			display: flex;
			align-items: center;
			gap: 12px;
		}

		.name {
			flex: 1;
			min-inline-size: 0;
			color: var(--fg);
		}

		.badge {
			flex-shrink: 0;
			padding: 2px 8px;
			border-radius: 999px;
			background: var(--accentedBg);
			color: var(--fgTransparentWeak);
			font-size: 0.75em;
		}

		.description {
			margin: 6px 0 0;
			color: var(--fgTransparentWeak);
			font-size: 0.85em;
		}
	}
}
</style>
