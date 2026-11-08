<!-- Ported from firefish@b425f61:packages/client/src/pages/my-antennas/editor.vue (AGPL-3.0-only) -->
<!--
	Shape port of Firefish's antenna editor. Firefish also offers the `group`
	source and an `instances` textarea and reads `antenna.instances`; neither
	field exists in the Misskey `antennas/create` contract this backend
	implements (docs/research/api-contract/), so they are dropped. The source
	list is exactly the four values the backend accepts: all, users,
	users_blacklist, list.
-->
<template>
	<div class="cf-antenna-editor">
		<CfInput v-model="name" class="_formBlock">
			<template #label>{{ t("name") }}</template>
		</CfInput>

		<CfSelect
			class="_formBlock"
			:model-value="src"
			:placeholder="srcPlaceholder"
			@update:model-value="onSrcUpdate"
		>
			<template #label>{{ t("antennaSource") }}</template>
			<option value="all">{{ t("_antennaSources.all") }}</option>
			<option value="users">{{ t("_antennaSources.users") }}</option>
			<option value="users_blacklist">
				{{ t("_antennaSources.users") }} (excluded)
			</option>
			<option value="list">{{ t("_antennaSources.userList") }}</option>
		</CfSelect>

		<CfSelect v-if="src === 'list'" v-model="userListId" class="_formBlock">
			<template #label>{{ t("userList") }}</template>
			<option v-for="list in userLists" :key="list.id" :value="list.id">
				{{ list.name }}
			</option>
		</CfSelect>

		<CfTextarea
			v-if="src === 'users' || src === 'users_blacklist'"
			v-model="usersText"
			class="_formBlock"
		>
			<template #label>{{ t("users") }}</template>
			<template #caption>{{ t("antennaUsersDescription") }}</template>
		</CfTextarea>

		<CfSwitch v-model="withReplies" class="_formBlock">
			{{ t("withReplies") }}
		</CfSwitch>

		<CfTextarea v-model="keywordsText" class="_formBlock">
			<template #label>{{ t("antennaKeywords") }}</template>
			<template #caption>{{
				t("antennaKeywordsDescription")
			}}</template>
		</CfTextarea>

		<CfTextarea v-model="excludeKeywordsText" class="_formBlock">
			<template #label>{{ t("antennaExcludeKeywords") }}</template>
			<template #caption>{{
				t("antennaKeywordsDescription")
			}}</template>
		</CfTextarea>

		<CfSwitch v-model="caseSensitive" class="_formBlock">
			{{ t("caseSensitive") }}
		</CfSwitch>

		<CfSwitch v-model="withFile" class="_formBlock">
			{{ t("withFileAntenna") }}
		</CfSwitch>

		<CfSwitch v-model="excludeBots" class="_formBlock">
			<!-- No locale key for this Misskey field; plain literal like the
			     other missing-key fallbacks in ported pages. -->
			Exclude bots
		</CfSwitch>

		<CfSwitch v-model="localOnly" class="_formBlock">
			{{ t("localOnly") }}
		</CfSwitch>

		<CfInfo v-if="keywordsEmpty" warn class="_formBlock">
			{{ t("antennaKeywordsDescription") }}
		</CfInfo>

		<div class="actions _formBlock">
			<CfButton primary :disabled="!canSubmit" @click="submit">
				<i class="ph-floppy-disk-back ph-bold ph-lg"></i>
				{{ t("save") }}
			</CfButton>
		</div>
	</div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { api } from "@/api/client";
import { ENDPOINTS, type AntennaInput } from "@/api/endpoints";
import type { Antenna, AntennaSrc, UserList } from "@/api/types";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfInput from "@/components/ui/CfInput.vue";
import CfSelect from "@/components/ui/CfSelect.vue";
import CfSwitch from "@/components/ui/CfSwitch.vue";
import CfTextarea from "@/components/ui/CfTextarea.vue";
import { t } from "@/i18n";

const props = withDefaults(
	defineProps<{
		/** The antenna to edit; `null` starts a blank draft. */
		initial?: Antenna | null;
		saving?: boolean;
	}>(),
	{
		initial: null,
		saving: false,
	},
);

const emit = defineEmits<{
	submit: [payload: AntennaInput];
}>();

const name = ref(props.initial?.name ?? "");
const src = ref<AntennaSrc>(props.initial?.src ?? "all");
const userListId = ref<string | null>(props.initial?.userListId ?? null);
const usersText = ref((props.initial?.users ?? []).join("\n"));
const keywordsText = ref(serializeGroups(props.initial?.keywords ?? []));
const excludeKeywordsText = ref(
	serializeGroups(props.initial?.excludeKeywords ?? []),
);
const caseSensitive = ref(props.initial?.caseSensitive ?? false);
const withReplies = ref(props.initial?.withReplies ?? true);
const withFile = ref(props.initial?.withFile ?? false);
const excludeBots = ref(props.initial?.excludeBots ?? false);
const localOnly = ref(props.initial?.localOnly ?? false);

const userLists = ref<UserList[] | null>(null);

// `home` is a valid packed src (the backend accepts it) but the editor does not
// offer it as a choice. When editing such an antenna, show its label instead of
// an empty trigger.
const srcPlaceholder = computed(() =>
	src.value === "home" ? t("_antennaSources.homeTimeline") : undefined,
);

// Each line is one OR-group; words inside a line are AND-ed. Blank lines are
// dropped so an empty editor serialises as `[]`, which the backend accepts as
// "no keywords" (only when the other keyword array is non-empty).
function parseGroups(text: string): string[][] {
	return text
		.split("\n")
		.map((line) => line.trim())
		.filter((line) => line !== "")
		.map((line) => line.split(/\s+/).filter((word) => word !== ""))
		.filter((group) => group.length > 0);
}

function serializeGroups(groups: string[][]): string {
	return groups.map((group) => group.join(" ")).join("\n");
}

const keywords = computed(() => parseGroups(keywordsText.value));
const excludeKeywords = computed(() =>
	parseGroups(excludeKeywordsText.value),
);

const keywordsEmpty = computed(
	() => keywords.value.length === 0 && excludeKeywords.value.length === 0,
);

const canSubmit = computed(
	() =>
		!props.saving &&
		name.value.trim() !== "" &&
		!keywordsEmpty.value &&
		(src.value !== "list" || userListId.value != null),
);

function onSrcUpdate(value: string | null): void {
	if (value == null) return;
	src.value = value as AntennaSrc;
}

async function loadUserLists(): Promise<void> {
	if (userLists.value != null) return;
	try {
		userLists.value = await api(ENDPOINTS.usersListsList, {});
	} catch {
		// The select simply stays empty; the save button stays disabled for
		// src="list" until a list is chosen.
		userLists.value = [];
	}
}

watch(
	src,
	(value) => {
		if (value === "list") void loadUserLists();
	},
	{ immediate: true },
);

function submit(): void {
	if (!canSubmit.value) return;
	emit("submit", {
		name: name.value.trim(),
		src: src.value,
		userListId: src.value === "list" ? userListId.value : null,
		users:
			src.value === "users" || src.value === "users_blacklist"
				? usersText.value
						.split("\n")
						.map((line) => line.trim())
						.filter((line) => line !== "")
				: [],
		keywords: keywords.value,
		excludeKeywords: excludeKeywords.value,
		caseSensitive: caseSensitive.value,
		withReplies: withReplies.value,
		withFile: withFile.value,
		excludeBots: excludeBots.value,
		localOnly: localOnly.value,
	});
}
</script>

<style lang="scss" scoped>
.cf-antenna-editor {
	.actions {
		display: flex;
		justify-content: flex-end;
	}
}
</style>
