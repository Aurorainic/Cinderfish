<!-- Ported from firefish@b425f61:packages/client/src/pages/settings/mute-block.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :display-back-button="true" />
		</template>
		<MkSpacer :content-max="700" :margin-min="16" :margin-max="32">
			<div class="_formRoot cf-mute-block">
				<CfInfo class="_formBlock">
					静音只是让对方的帖子从你的时间线消失，对方不会收到通知；屏蔽则同时解除双方的关注、
					取消待审的跟随请求，并且双方都看不到对方。两者都只影响你自己的账号。
				</CfInfo>

				<CfRadios v-model="tab" class="_formBlock">
					<CfRadio value="mute">静音名单（{{ mutes.length }}）</CfRadio>
					<CfRadio value="block">屏蔽名单（{{ blocks.length }}）</CfRadio>
				</CfRadios>

				<section class="_formBlock">
					<h2 class="_sectionTitle">添加</h2>
					<div class="addRow">
						<CfInput
							v-model="username"
							class="addInput"
							placeholder="用户名（本实例账号）"
							autocomplete="off"
							:spellcheck="false"
							@enter="add"
						>
							<template #prefix>@</template>
						</CfInput>
						<CfButton primary :disabled="adding" @click="add">
							{{ adding ? "处理中…" : tab === "mute" ? "静音" : "屏蔽" }}
						</CfButton>
					</div>
					<p class="hint">
						远端账号请先在帖子菜单里操作，或在个人资料页使用同一入口。
					</p>
					<CfInfo v-if="addError !== null" warn class="_formBlock">
						{{ addError }}
					</CfInfo>
				</section>

				<CfLoading v-if="loading" />
				<MkError v-else-if="loadError !== null" @retry="load" />
				<CfInfo v-else-if="current.length === 0">
					{{ tab === "mute" ? "还没有静音任何账号。" : "还没有屏蔽任何账号。" }}
				</CfInfo>
				<ul v-else class="entries">
					<li v-for="entry in current" :key="entry.id" class="entry">
						<MkAvatar class="avatar" :user="entry.user" />
						<div class="body">
							<CfA class="name" :to="userPage(entry.user)">
								<MkUserName :user="entry.user" :nowrap="true" />
							</CfA>
							<span class="acct _monospace">@{{ acct(entry.user) }}</span>
							<span class="date">
								<MkTime :time="entry.createdAt" mode="absolute" />
							</span>
						</div>
						<CfButton
							mini
							:disabled="busyUserId === entry.user.id"
							@click="remove(entry)"
						>
							{{ tab === "mute" ? "解除静音" : "解除屏蔽" }}
						</CfButton>
					</li>
				</ul>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script setup lang="ts">
// Firefish's mute-block page is a tabbed `MkPagination` over
// `mute/list` and `blocking/list`. Cinderfish gained those endpoints together
// with this page, so the list, the removal action and the add form all talk to
// the server; nothing is kept client side.
//
// Deliberate differences:
//   * the add form resolves `@username` through `users/show` because there is
//     no user picker dialog here;
//   * rows are plain list items, not `MkUserCardMini`, so the removal button
//     has a stable place on narrow screens.
import { computed, onMounted, ref, watch } from "vue";
import { ApiError, api } from "@/api/client";
import type { BlockingRecord, MuteRecord, UserDetailed } from "@/api/types";
import MkAvatar from "@/components/global/MkAvatar.vue";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";
import MkUserName from "@/components/global/MkUserName.vue";
import MkTime from "@/components/global/MkTime.vue";
import CfA from "@/components/ui/CfA.vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfInput from "@/components/ui/CfInput.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import MkError from "@/components/global/MkError.vue";
import CfRadios from "@/components/ui/CfRadios.vue";
import CfRadio from "@/components/ui/CfRadio.vue";
import { acct, userPage } from "@/filters/user";

type Tab = "mute" | "block";

interface Entry {
	id: string;
	createdAt: string;
	user: UserDetailed;
}

const LIMIT = 50;

const tab = ref<Tab>("mute");
const mutes = ref<Entry[]>([]);
const blocks = ref<Entry[]>([]);
const loading = ref(false);
const loadError = ref<string | null>(null);
const busyUserId = ref<string | null>(null);
const username = ref("");
const adding = ref(false);
const addError = ref<string | null>(null);

const current = computed(() => (tab.value === "mute" ? mutes.value : blocks.value));

function errorText(error: unknown): string {
	if (error instanceof ApiError) {
		if (error.code === "NO_SUCH_USER") return "找不到这个本地账号。";
		if (error.code === "MUTEE_IS_YOURSELF" || error.code === "BLOCKEE_IS_YOURSELF")
			return "不能对自己执行这个操作。";
		if (error.code === "ALREADY_MUTING") return "这个账号已经在静音名单里了。";
		if (error.code === "ALREADY_BLOCKING") return "这个账号已经在屏蔽名单里了。";
		return error.message;
	}
	return String(error);
}

async function load(): Promise<void> {
	loading.value = true;
	loadError.value = null;
	try {
		if (tab.value === "mute") {
			const rows = await api("mute/list", { limit: LIMIT });
			mutes.value = rows.map((row: MuteRecord) => ({
				id: row.id,
				createdAt: row.createdAt,
				user: row.mutee,
			}));
		} else {
			const rows = await api("blocking/list", { limit: LIMIT });
			blocks.value = rows.map((row: BlockingRecord) => ({
				id: row.id,
				createdAt: row.createdAt,
				user: row.blockee,
			}));
		}
	} catch (error) {
		loadError.value = errorText(error);
	} finally {
		loading.value = false;
	}
}

async function add(): Promise<void> {
	const name = username.value.trim().replace(/^@/, "");
	if (name === "" || adding.value) return;
	adding.value = true;
	addError.value = null;
	try {
		const user = await api("users/show", { username: name });
		if (tab.value === "mute") {
			await api("mute/create", { userId: user.id });
		} else {
			await api("blocking/create", { userId: user.id });
		}
		username.value = "";
		await load();
	} catch (error) {
		addError.value = errorText(error);
	} finally {
		adding.value = false;
	}
}

async function remove(entry: Entry): Promise<void> {
	if (busyUserId.value !== null) return;
	busyUserId.value = entry.user.id;
	addError.value = null;
	try {
		if (tab.value === "mute") {
			await api("mute/delete", { userId: entry.user.id });
			mutes.value = mutes.value.filter((row) => row.id !== entry.id);
		} else {
			await api("blocking/delete", { userId: entry.user.id });
			blocks.value = blocks.value.filter((row) => row.id !== entry.id);
		}
	} catch (error) {
		addError.value = errorText(error);
	} finally {
		busyUserId.value = null;
	}
}

watch(tab, () => {
	addError.value = null;
	void load();
});

onMounted(() => {
	void load();
});
</script>

<style lang="scss" scoped>
.addRow {
	display: flex;
	gap: 8px;
	align-items: flex-end;

	.addInput {
		flex: 1;
		min-width: 0;
	}
}

.hint {
	margin: 6px 0 0;
	font-size: 0.85em;
	color: var(--fgTransparentWeak);
}

.entries {
	list-style: none;
	margin: 0;
	padding: 0;
	display: flex;
	flex-direction: column;
	gap: 8px;
}

.entry {
	display: flex;
	align-items: center;
	gap: 12px;
	padding: 10px 12px;
	background: var(--panel);
	border-radius: var(--radius);

	.avatar {
		width: 42px;
		height: 42px;
		flex: none;
	}

	.body {
		display: flex;
		flex-direction: column;
		gap: 2px;
		min-width: 0;
		flex: 1;
	}

	.name {
		font-weight: 600;
		overflow: hidden;
		text-overflow: ellipsis;
		white-space: nowrap;
	}

	.acct {
		font-size: 0.82em;
		color: var(--fgTransparentWeak);
	}

	.date {
		font-size: 0.78em;
		color: var(--fgTransparentWeak);
	}
}
</style>
