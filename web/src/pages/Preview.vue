<!-- Ported from firefish@b425f61:packages/client/src/pages/preview.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :actions="headerActions" />
		</template>
		<MkSpacer :content-max="700">
			<div class="preview _formRoot">
				<div class="_formBlock">
					<CfInput
						v-model="input"
						placeholder="笔记 ID / 用户 ID / @用户名"
						@enter="previewNote()"
					>
						<template #label>帖子或用户</template>
					</CfInput>
					<div class="kinds">
						<CfButton
							primary
							:disabled="query === '' || loading"
							@click="previewNote()"
						>
							<i class="ph-note ph-bold ph-lg"></i>
							预览帖子
						</CfButton>
						<CfButton
							:disabled="query === '' || loading"
							@click="previewUser()"
						>
							<i class="ph-user ph-bold ph-lg"></i>
							预览用户
						</CfButton>
					</div>
				</div>

				<CfLoading v-if="loading" />
				<CfInfo v-else-if="errorMessage !== null" class="_formBlock" warn>
					{{ errorMessage }}
				</CfInfo>

				<div v-else-if="note !== null" class="_formBlock">
					<MkNoteDetailed :note="note" />
				</div>

				<div v-else-if="user !== null" class="_formBlock userCard _panel">
					<div class="head">
						<MkAvatar :user="user" class="avatar" />
						<div class="names">
							<MkUserName :user="user" class="name" />
							<div class="acct">{{ acct(user) }}</div>
						</div>
					</div>
					<Mfm
						v-if="user.description != null && user.description !== ''"
						class="description"
						:text="user.description"
						:author="user"
					/>
					<CfKeyValue>
						<template #key>{{ t("notesCount") }}</template>
						<template #value>{{ user.notesCount }}</template>
					</CfKeyValue>
					<CfKeyValue>
						<template #key>{{ t("followersCount") }}</template>
						<template #value>{{ user.followersCount }}</template>
					</CfKeyValue>
					<CfKeyValue>
						<template #key>{{ t("followingCount") }}</template>
						<template #value>{{ user.followingCount }}</template>
					</CfKeyValue>
					<CfKeyValue>
						<template #key>{{ t("createdAt") }}</template>
						<template #value><MkTime :time="user.createdAt" /></template>
					</CfKeyValue>
					<CfA class="link" :to="userPage(user)">打开个人页</CfA>
				</div>

				<p v-else class="hint">
					输入笔记 ID 或用户 ID（也可以粘贴笔记链接、个人页链接或
					@用户名），再选择「预览帖子」或「预览用户」。
				</p>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script lang="ts" setup>
// Firefish's preview.vue renders MkSample — a development page showing sample
// UI components with hard-coded data. Cinderfish has no MkSample and no sample
// fixtures, so the page is repurposed (wave-3 task definition) into a preview
// of a real entity: paste a note id / user id and the entity is fetched with
// `notes/show` / `users/show` and rendered with the production components.
//
// `users/show` only accepts local usernames or user ids; remote users return an
// error (api-contract §5), which the error state surfaces verbatim.
import { computed, onMounted, ref } from "vue";
import { useRoute } from "vue-router";
import type { NoteDetailed, UserDetailed } from "@/api/types";
import { api, ApiError } from "@/api/client";
import { acct } from "@/components/ui/icon";
import CfA from "@/components/ui/CfA.vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfInput from "@/components/ui/CfInput.vue";
import CfKeyValue from "@/components/ui/CfKeyValue.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import MkNoteDetailed from "@/components/MkNoteDetailed.vue";
import { userPage } from "@/filters/user";
import { t } from "@/i18n";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPagePreview" });

const route = useRoute();

const input = ref<string | number | null>("");
const note = ref<NoteDetailed | null>(null);
const user = ref<UserDetailed | null>(null);
const loading = ref(false);
const errorMessage = ref<string | null>(null);

const query = computed(() => String(input.value ?? "").trim());

/**
 * Accepts a bare id, an `@user`, a profile URL or a note URL. A note URL is
 * `/notes/<id>`; a profile URL is `/<@user>/<id>` (ActivityPub actor URLs are
 * `/users/<id>`, which users/show also accepts as a userId).
 */
function extract(raw: string): string {
	const value = raw.trim();
	if (value === "") return "";
	if (/^https?:\/\//.test(value)) {
		try {
			const path = new URL(value).pathname.replace(/\/+$/, "");
			const noteMatch = path.match(/\/notes\/([^/]+)$/);
			if (noteMatch != null) return decodeURIComponent(noteMatch[1]);
			const userMatch = path.match(/\/users\/([^/]+)$/);
			if (userMatch != null) return decodeURIComponent(userMatch[1]);
			const profileMatch = path.match(/^\/(@[^/]+)/);
			if (profileMatch != null) return decodeURIComponent(profileMatch[1]);
			return path.split("/").pop() ?? "";
		} catch {
			return value;
		}
	}
	return value;
}

function reset(): void {
	note.value = null;
	user.value = null;
	errorMessage.value = null;
}

function describe(err: unknown): string {
	if (err instanceof ApiError) return err.message;
	return err instanceof Error ? err.message : String(err);
}

async function previewNote(): Promise<void> {
	const id = extract(query.value);
	if (id === "") return;
	reset();
	loading.value = true;
	try {
		note.value = await api("notes/show", { noteId: id });
	} catch (err) {
		errorMessage.value = describe(err);
	} finally {
		loading.value = false;
	}
}

async function previewUser(): Promise<void> {
	const value = extract(query.value);
	if (value === "") return;
	reset();
	loading.value = true;
	try {
		// A bare id is passed as userId, an @handle as username. `users/show`
		// rejects remote hosts, so a full acct cannot be previewed.
		const handle = value.replace(/^@/, "").replace(/@[^@]+$/, "");
		const isId = /^[0-9a-z]{18}$/i.test(handle);
		user.value = await api(
			"users/show",
			isId ? { userId: handle } : { username: handle },
		);
	} catch (err) {
		errorMessage.value = describe(err);
	} finally {
		loading.value = false;
	}
}

const headerActions = computed(() => []);

onMounted(() => {
	const noteId = route.query.noteId;
	const userId = route.query.userId;
	const username = route.query.username;
	if (typeof noteId === "string" && noteId !== "") {
		input.value = noteId;
		void previewNote();
	} else if (typeof userId === "string" && userId !== "") {
		input.value = userId;
		void previewUser();
	} else if (typeof username === "string" && username !== "") {
		input.value = username;
		void previewUser();
	}
});

definePageMetadata({
	title: t("preview"),
	icon: "ph-eye ph-bold ph-lg",
});
</script>

<style lang="scss" scoped>
.preview {
	padding: 16px;
}

.kinds {
	display: flex;
	flex-wrap: wrap;
	gap: 8px;
}

.hint {
	margin: 0;
	padding: 16px;
	color: var(--fgTransparentWeak);
	font-size: 0.9em;
}

.userCard {
	padding: 16px;

	> .head {
		display: flex;
		align-items: center;
		gap: 12px;

		> .avatar {
			flex: 0 0 auto;
			width: 48px;
			height: 48px;
		}

		> .names {
			min-inline-size: 0;

			> .name {
				font-weight: bold;
			}

			> .acct {
				font-size: 0.9em;
				color: var(--fgTransparentWeak);
			}
		}
	}

	> .description {
		display: block;
		padding-block: 12px;
	}

	> .link {
		display: inline-block;
		padding-block-start: 8px;
		color: var(--accent);
		text-decoration: none;

		&:hover {
			text-decoration: underline;
		}
	}
}
</style>
