<!-- Ported from firefish@b425f61:packages/client/src/pages/authorize_interaction.vue (AGPL-3.0-only) -->
<template>
	<MkSpacer :content-max="800" :margin-min="16" :margin-max="32">
		<div class="cf-interaction">
			<div v-if="uri === ''" class="section">
				<CfInfo warn>
					链接缺少 <code>uri</code> 参数。请从原实例的「在其它实例中打开」入口重新进入。
				</CfInfo>
			</div>

			<div v-else-if="errorText != null" class="section">
				<CfInfo warn>{{ errorText }}</CfInfo>
				<p class="hint">
					可以
					<a :href="uri" rel="nofollow noopener" target="_blank">在原实例查看</a>
					，或用地址栏里的链接直接访问。
				</p>
			</div>

			<div v-else-if="loading" class="section loading">
				<CfLoading />
			</div>

			<div v-else-if="kind === 'User' && user != null" class="section">
				<h2 class="title">与这个账号互动</h2>
				<div class="user">
					<CfAvatar :user="user" class="avatar" />
					<div class="body">
						<div class="name">{{ user.name ?? user.username }}</div>
						<div class="acct">
							<MkAcct :user="user" :detail="true" />
						</div>
					</div>
					<MkFollowButton :user="user" />
				</div>
				<p class="hint">
					<a :href="uri" rel="nofollow noopener" target="_blank">在原实例查看</a>
				</p>
			</div>

			<div v-else-if="kind === 'Note' && note != null" class="section">
				<h2 class="title">与这条笔记互动</h2>
				<MkNoteDetailed :key="note.id" :note="note" />
				<p class="hint">
					<a :href="uri" rel="nofollow noopener" target="_blank">在原实例查看</a>
				</p>
			</div>
		</div>
	</MkSpacer>
</template>

<script lang="ts" setup>
// Misskey's /authorize_interaction?uri=… resolves a remote URL and offers the
// actions the viewer can take here. Cinderfish has ap/show, which imports the
// object through the same code path inbound federation uses, so the page can
// render the note (with its built-in actions) or the account plus a follow
// button. Mastodon's /authorize_interaction flow is not implemented.
import { onMounted, ref } from "vue";
import { useRoute } from "vue-router";
import { ApiError, api } from "@/api/client";
import type { NoteDetailed, UserDetailed } from "@/api/types";
import CfAvatar from "@/components/ui/CfAvatar.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import MkAcct from "@/components/global/MkAcct.vue";
import MkFollowButton from "@/components/MkFollowButton.vue";
import MkNoteDetailed from "@/components/MkNoteDetailed.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageAuthorizeInteraction" });

const route = useRoute();

function first(value: unknown): string {
	if (typeof value === "string") return value;
	if (Array.isArray(value) && typeof value[0] === "string") return value[0];
	return "";
}

const uri = first(route.query.uri);
const loading = ref(uri !== "");
const errorText = ref<string | null>(null);
const kind = ref<"Note" | "User" | null>(null);
const note = ref<NoteDetailed | null>(null);
const user = ref<UserDetailed | null>(null);

onMounted(async () => {
	if (uri === "") return;
	try {
		const result = await api("ap/show", { uri });
		if (result.type === "Note") {
			kind.value = "Note";
			note.value = result.object as NoteDetailed;
		} else {
			kind.value = "User";
			user.value = result.object as UserDetailed;
		}
	} catch (cause) {
		errorText.value =
			cause instanceof ApiError
				? `无法解析这个链接：${cause.message}`
				: "无法解析这个链接。";
	} finally {
		loading.value = false;
	}
});

definePageMetadata({
	title: "与他处互动",
	icon: "ph-arrows-left-right ph-bold ph-lg",
});
</script>

<style lang="scss" scoped>
.cf-interaction {
	max-width: 800px;
	margin: 0 auto;

	.section {
		padding: 24px;
		border-radius: var(--radius);
		background: var(--panel);

		@media (max-width: 500px) {
			padding: 16px;
		}
	}

	.title {
		margin: 0 0 16px;
		font-size: 1.15em;
	}

	.user {
		display: flex;
		align-items: center;
		gap: 12px;

		.avatar {
			flex: none;
			width: 48px;
			height: 48px;
		}

		.body {
			flex: 1;
			min-width: 0;
			display: flex;
			flex-direction: column;
			gap: 2px;
		}

		.name {
			font-weight: 700;
		}

		.acct {
			color: var(--fgTransparentWeak);
			font-size: 0.92em;
		}
	}

	.hint {
		margin: 12px 0 0;
		font-size: 0.92em;
		color: var(--fgTransparentWeak);
	}

	.loading {
		display: flex;
		justify-content: center;
	}
}
</style>
