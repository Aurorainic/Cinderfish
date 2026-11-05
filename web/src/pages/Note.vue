<!-- Ported from firefish@b425f61:packages/client/src/pages/note.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader
				:actions="headerActions"
				:tabs="headerTabs"
				:display-back-button="true"
				:to="'#' + noteId"
			/>
		</template>
		<MkSpacer :content-max="800" :margin-min="6">
			<div class="fcuexfpr">
				<transition :name="animation ? 'fade' : ''" mode="out-in">
					<div v-if="appearNote != null" :key="appearNote.id" class="note">
						<div v-if="showNext" class="_gap">
							<MkNotes
								v-if="nextPagination != null"
								class="_content"
								:pagination="nextPagination"
								:no-gap="true"
							/>
						</div>

						<div class="main _gap">
							<CfButton
								v-if="!showNext && hasNext"
								class="load next"
								@click="showNext = true"
							>
								<i class="ph-caret-up ph-bold ph-lg"></i>
								{{ `${t("loadMore")} (${t("newer")})` }}
							</CfButton>
							<div class="note _gap">
								<MkNoteDetailed :key="appearNote.id" :note="appearNote" class="note" />
							</div>
							<CfButton
								v-if="!showPrev && hasPrev"
								class="load prev"
								@click="showPrev = true"
							>
								<i class="ph-caret-down ph-bold ph-lg"></i>
								{{ `${t("loadMore")} (${t("older")})` }}
							</CfButton>
						</div>

						<div v-if="showPrev" class="_gap">
							<MkNotes
								v-if="prevPagination != null"
								class="_content"
								:pagination="prevPagination"
								:no-gap="true"
							/>
						</div>
					</div>
					<MkError v-else-if="error != null" @retry="fetchNote" />
					<MkLoading v-else />
				</transition>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script setup lang="ts">
// Note detail page.
//
// Firefish renders MkRemoteCaution (remote-note intercept warning) and passes
// `v-model:note` into MkNoteDetailed. Neither is available this wave: the
// caution component belongs to a later page pass, and MkNoteDetailed owns its
// own copy of the note (exactly as upstream, which is where useNoteCapture is
// called). The two "load more" blocks browse the author's neighbouring notes
// via users/notes, like Firefish.
import { computed, ref, watch } from "vue";
import { useRoute } from "vue-router";
// MkNoteDetailed lives in components/, not components/global/, so it is not
// auto-registered by `components/index.ts` (that glob only covers global/).
// Without this import Vue resolves it as an unknown custom element and the
// detail page renders only the two "load more" buttons.
import MkNoteDetailed from "@/components/MkNoteDetailed.vue";
import MkNotes from "@/components/MkNotes.vue";
import CfButton from "@/components/ui/CfButton.vue";
import type { Paging } from "@/components/ui/pagination";
import { api } from "@/api/client";
import type { NoteDetailed } from "@/api/types";
import { t } from "@/i18n";
import { defaultStore } from "@/store/base";
import { definePageMetadata } from "@/scripts/page-metadata";

const route = useRoute();

const noteId = computed(() => String(route.params.noteId ?? ""));

const appearNote = ref<NoteDetailed | null>(null);
const hasPrev = ref(false);
const hasNext = ref(false);
const showPrev = ref(false);
const showNext = ref(false);
const error = ref<unknown>(null);
const animation = computed(
	() => defaultStore.reactiveState.animation.value,
);

const prevPagination = computed<Paging | null>(() => {
	const current = appearNote.value;
	if (current == null) return null;
	return {
		endpoint: "users/notes",
		limit: 10,
		// Cinderfish's users/notes defaults withReplies to true (the opposite of
		// Misskey); Firefish's neighbouring-note lists are original notes only.
		params: { userId: current.userId, untilId: current.id, withReplies: false },
	};
});

const nextPagination = computed<Paging | null>(() => {
	const current = appearNote.value;
	if (current == null) return null;
	return {
		reversed: true,
		endpoint: "users/notes",
		limit: 10,
		params: { userId: current.userId, sinceId: current.id, withReplies: false },
	};
});

async function fetchNote(): Promise<void> {
	hasPrev.value = false;
	hasNext.value = false;
	showPrev.value = false;
	showNext.value = false;
	appearNote.value = null;
	error.value = null;

	try {
		const fetched = await api("notes/show", { noteId: noteId.value });

		// A pure renote (no text, files or poll of its own) is displayed as the
		// quoted note, exactly like Firefish. The nested object is typed as `Note`
		// but comes from the detail endpoint's projection, so it is rendered by
		// MkNoteDetailed unchanged (same cast upstream relies on implicitly).
		const renote = fetched.renote as NoteDetailed | null | undefined;
		const isRenote =
			renote != null &&
			fetched.text == null &&
			fetched.fileIds.length === 0 &&
			fetched.poll == null;
		appearNote.value = isRenote && renote != null ? renote : fetched;

		const [prev, next] = await Promise.all([
			api("users/notes", {
				userId: fetched.userId,
				untilId: fetched.id,
				limit: 1,
				withReplies: false,
			}),
			api("users/notes", {
				userId: fetched.userId,
				sinceId: fetched.id,
				limit: 1,
				withReplies: false,
			}),
		]);
		hasPrev.value = prev.length !== 0;
		hasNext.value = next.length !== 0;
	} catch (err) {
		error.value = err;
	}
}

watch(noteId, fetchNote, { immediate: true });

const headerActions = computed<
	{ text: string; icon: string; handler: () => void }[]
>(() => []);

const headerTabs = computed<{ key: string; title: string }[]>(() => []);

definePageMetadata(
	computed(() => {
		const current = appearNote.value;
		if (current == null) return null;
		const userName = current.user.name ?? current.user.username;
		return {
			title: t("noteOf", { user: userName }),
			subtitle: new Date(current.createdAt).toLocaleString(),
			avatar: current.user,
		};
	}),
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

.fcuexfpr {
	> .note {
		> .main {
			> .load {
				min-width: 0;
				margin: 0 auto;
				border-radius: 999px;

				&.next {
					margin-bottom: var(--margin);
				}

				&.prev {
					margin-top: var(--margin);
				}
			}

			> .note {
				> .note {
					border-radius: var(--radius);
					background: var(--panel);
				}
			}
		}
	}
}
</style>
