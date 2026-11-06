<!-- Ported from firefish@b425f61:packages/client/src/pages/settings/migration.vue (AGPL-3.0-only) -->
<template>
	<div class="_formRoot cf-settings-migration">
		<CfInfo v-if="me == null">{{ t("signinRequired") }}</CfInfo>
		<template v-else>
			<section class="_formBlock">
				<h2 class="sectionTitle">{{ t("moveTo") }}</h2>
				<CfInfo warn class="note">{{ t("moveAccountDescription") }}</CfInfo>
				<CfInfo v-if="me.movedToUri" warn class="note">
					此账号已经迁移到
					<MkMention
						:username="me.movedToUri.username"
						:host="me.movedToUri.host ?? ''"
					/>
					，不能再迁移到其它账号。
				</CfInfo>
				<template v-else>
					<CfInput v-model="moveToAccount" class="field">
						<template #label>{{ t("moveToLabel") }}</template>
					</CfInput>
					<CfButton
						primary
						danger
						:disabled="busy || moveToAccount.trim() === ''"
						@click="move()"
					>
						<i class="ph-airplane-takeoff ph-bold ph-lg"></i>
						{{ t("moveAccount") }}
					</CfButton>
				</template>
				<CfInfo v-if="moveError != null" warn class="note">{{
					moveError
				}}</CfInfo>
				<CfInfo v-if="moveNotice != null" class="note">{{
					moveNotice
				}}</CfInfo>
			</section>

			<section class="_formBlock">
				<h2 class="sectionTitle">{{ t("moveFrom") }}</h2>
				<CfInfo warn class="note">{{ t("moveFromDescription") }}</CfInfo>
				<CfInput
					v-for="(_, i) in aliases"
					:key="i"
					v-model="aliases[i]"
					class="field"
				>
					<template #label
						>#{{ i + 1 }} {{ t("moveFromLabel") }}</template
					>
				</CfInput>
				<div class="row">
					<CfButton
						inline
						:disabled="savingAliases || aliases.length >= 10"
						@click="add()"
					>
						<i class="ph-plus ph-bold ph-lg"></i>
						{{ t("add") }}
					</CfButton>
					<CfButton
						inline
						primary
						:disabled="savingAliases"
						@click="saveAliases()"
					>
						<i class="ph-floppy-disk ph-bold ph-lg"></i>
						{{ t("save") }}
					</CfButton>
				</div>
				<CfInfo v-if="aliasError != null" warn class="note">{{
					aliasError
				}}</CfInfo>
				<CfInfo v-if="aliasNotice != null" class="note">{{
					aliasNotice
				}}</CfInfo>
			</section>
		</template>
	</div>
</template>

<script lang="ts" setup>
// Firefish's settings/migration.vue uses i/known-as (aliases of this account)
// and i/move (migrate to another instance). The port keeps both forms; the
// alias list is resolved back to handles through users/show, which accepts the
// stored actor URIs. A move cannot be undone, so the confirmation is a warning
// dialog like Firefish's.
import { ref } from "vue";
import { me, updateMe } from "@/account";
import { ApiError, api } from "@/api/client";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfInput from "@/components/ui/CfInput.vue";
import MkMention from "@/components/MkMention.vue";
import { t } from "@/i18n";
import { confirm } from "@/os";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageSettingsMigration" });

const moveToAccount = ref("");
const aliases = ref<string[]>([""]);
const busy = ref(false);
const savingAliases = ref(false);
const moveError = ref<string | null>(null);
const moveNotice = ref<string | null>(null);
const aliasError = ref<string | null>(null);
const aliasNotice = ref<string | null>(null);

function errorMessage(cause: unknown): string {
	if (cause instanceof ApiError) return cause.message;
	return String(cause);
}

async function loadAliases(): Promise<void> {
	const current = me.value?.alsoKnownAs ?? [];
	if (current.length === 0) {
		aliases.value = [""];
		return;
	}
	const handles = await Promise.all(
		current.map(async (uri) => {
			try {
				const user = await api("users/show", { userId: uri });
				return user.host == null
					? `@${user.username}`
					: `@${user.username}@${user.host}`;
			} catch {
				return "";
			}
		}),
	);
	aliases.value = handles.length > 0 ? handles : [""];
}

void loadAliases();

function add(): void {
	if (aliases.value.length >= 10) return;
	aliases.value.push("");
}

async function saveAliases(): Promise<void> {
	if (savingAliases.value) return;
	savingAliases.value = true;
	aliasError.value = null;
	aliasNotice.value = null;
	try {
		const updated = await api("i/known-as", {
			alsoKnownAs: aliases.value.map((a) => a.trim()).filter((a) => a !== ""),
		});
		updateMe(updated);
		await loadAliases();
		aliasNotice.value = t("saved");
	} catch (cause) {
		aliasError.value = errorMessage(cause);
	} finally {
		savingAliases.value = false;
	}
}

async function move(): Promise<void> {
	if (busy.value) return;
	const account = moveToAccount.value.trim();
	if (account === "") return;
	const confirmed = await confirm({
		type: "warning",
		text: t("migrationConfirm", { account }),
	});
	if (confirmed.canceled) return;

	busy.value = true;
	moveError.value = null;
	moveNotice.value = null;
	try {
		const updated = await api("i/move", { moveToAccount: account });
		updateMe(updated);
		moveNotice.value = t("accountMoved");
	} catch (cause) {
		moveError.value = errorMessage(cause);
	} finally {
		busy.value = false;
	}
}

definePageMetadata({
	title: t("security"),
	icon: "ph-lock ph-bold ph-lg",
});
</script>

<style lang="scss" scoped>
.cf-settings-migration {
	.sectionTitle {
		font-size: 1.05em;
		font-weight: 700;
		margin: 0 0 12px;
	}

	.note {
		margin-bottom: 12px;
	}

	.field {
		margin-bottom: 12px;
	}

	.row {
		display: flex;
		gap: 8px;
		flex-wrap: wrap;
	}
}
</style>
