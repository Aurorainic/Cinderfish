<!-- Ported from firefish@b425f61:packages/client/src/components/MkRenoteButton.vue (AGPL-3.0-only) -->
<template>
	<button
		v-if="canRenote"
		ref="buttonRef"
		v-tooltip.noDelay.bottom="t('renote')"
		class="button _button canRenote"
		:class="{ renoted: hasRenotedBefore }"
		@click.stop="renote($event)"
	>
		<i class="ph-repeat ph-bold ph-lg"></i>
		<p v-if="count > 0 && !detailedView" class="count">{{ count }}</p>
	</button>
	<button
		v-else
		v-tooltip.noDelay.bottom="t('disabled')"
		class="eddddedb _button"
		disabled
	>
		<i class="ph-repeat ph-bold ph-lg"></i>
	</button>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { useRouter } from "vue-router";
import type { Note } from "@/api/types";
import { ENDPOINTS } from "@/api/endpoints";
import { apiWithDialog, popupMenu } from "@/os";
import type { MenuItem } from "@/os";
import { me } from "@/account";
import { pleaseLogin } from "@/scripts/please-login";
import { defaultStore } from "@/store";
import { t } from "@/i18n";

/**
 * Ported from Firefish, with three contract-forced changes:
 *   - `notes/renotes` cannot be filtered by author (no `userId` parameter), so
 *     "already renoted" can only be tracked locally after this component's own
 *     action; it is never queried per note.
 *   - `notes/unrenote` does not exist in Cinderfish (docs/research/api-contract/
 *     notes-core.md); the undo entry is not rendered.
 *   - Quoting opens the share composer route with `?renoteId=`, the Firefish
 *     share-form query (there is no `os.post` dialog in this wave).
 */
const props = defineProps<{
	note: Note;
	count: number;
	detailedView?: boolean;
}>();

const router = useRouter();
const buttonRef = ref<HTMLElement | null>(null);
const hasRenotedBefore = ref(false);

const canRenote = computed(
	() =>
		["public", "home"].includes(props.note.visibility) ||
		props.note.userId === me.value?.id,
);

function quote(): void {
	void router.push({ path: "/share", query: { renoteId: props.note.id } });
}

function renote(ev?: MouseEvent): void {
	pleaseLogin();

	const buttonActions: MenuItem[] = [];

	if (props.note.visibility === "public") {
		buttonActions.push({
			text: t("renote"),
			icon: "ph-repeat ph-bold ph-lg",
			action: () => {
				void apiWithDialog(ENDPOINTS.notesCreate, {
					renoteId: props.note.id,
					visibility: "public",
				});
				hasRenotedBefore.value = true;
			},
		});
	}

	if (["public", "home"].includes(props.note.visibility)) {
		buttonActions.push({
			text: `${t("renote")} (${t("_visibility.home")})`,
			icon: "ph-house ph-bold ph-lg",
			action: () => {
				void apiWithDialog(ENDPOINTS.notesCreate, {
					renoteId: props.note.id,
					visibility: "home",
				});
				hasRenotedBefore.value = true;
			},
		});
	}

	if (props.note.visibility === "specified") {
		buttonActions.push({
			text: `${t("renote")} (${t("recipient")})`,
			icon: "ph-envelope-simple-open ph-bold ph-lg",
			action: () => {
				void apiWithDialog(ENDPOINTS.notesCreate, {
					renoteId: props.note.id,
					visibility: "specified",
					visibleUserIds: props.note.visibleUserIds,
				});
				hasRenotedBefore.value = true;
			},
		});
	} else {
		buttonActions.push({
			text: `${t("renote")} (${t("_visibility.followers")})`,
			icon: "ph-lock ph-bold ph-lg",
			action: () => {
				void apiWithDialog(ENDPOINTS.notesCreate, {
					renoteId: props.note.id,
					visibility: "followers",
				});
				hasRenotedBefore.value = true;
			},
		});
	}

	if (canRenote.value) {
		buttonActions.push({
			text: `${t("renote")} (${t("local")})`,
			icon: "ph-hand-fist ph-bold ph-lg",
			action: () => {
				void apiWithDialog(
					ENDPOINTS.notesCreate,
					props.note.visibility === "specified"
						? {
								renoteId: props.note.id,
								visibility: props.note.visibility,
								visibleUserIds: props.note.visibleUserIds,
								localOnly: true,
							}
						: {
								renoteId: props.note.id,
								visibility: props.note.visibility,
								localOnly: true,
							},
				);
				hasRenotedBefore.value = true;
			},
		});
	}

	if (!defaultStore.state.seperateRenoteQuote) {
		buttonActions.push({
			text: t("quote"),
			icon: "ph-quotes ph-bold ph-lg",
			action: quote,
		});
	}

	void popupMenu(buttonActions, ev ?? anchorPoint());
}

function anchorPoint(): { x: number; y: number } {
	const rect = buttonRef.value?.getBoundingClientRect();
	if (rect == null) return { x: 0, y: 0 };
	return { x: rect.left, y: rect.bottom };
}
</script>

<style lang="scss" scoped>
.button {
	&:not(.canRenote) {
		cursor: default;
	}
	&.renoted {
		color: var(--accent) !important;
		opacity: 1 !important;
		font-weight: 700;
	}
}
</style>
