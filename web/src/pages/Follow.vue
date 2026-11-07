<!-- Ported from firefish@b425f61:packages/client/src/pages/follow.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :actions="headerActions" />
		</template>
		<MkSpacer :content-max="600">
			<div class="follow-page">
				<CfInfo v-if="me == null" warn icon="sign-in">
					关注操作需要登录。<CfA to="/">返回首页登录</CfA>
				</CfInfo>

				<CfLoading v-else-if="status === 'loading'" />

				<MkError v-else-if="status === 'error'" @retry="load()" />

				<CfInfo v-else-if="status === 'missing'" warn icon="link">
					URL 缺少 <code>acct</code> 参数。示例：
					<code>/authorize-follow?acct=@alice@{{ host }}</code>
				</CfInfo>

				<CfInfo v-else-if="status === 'invalid'" warn icon="warning">
					无法解析 <code>{{ rawAcct }}</code>。支持 <code>@user@host</code>、
					<code>user@host</code> 与 <code>user</code>
					三种写法，用户名只能包含字母、数字与下划线。
				</CfInfo>

				<CfInfo v-else-if="status === 'remote'" warn icon="globe">
					<div class="paragraphs">
						<p class="paragraph">
							无法从本站解析远程账号：<code>{{ rawAcct }}</code>
						</p>
						<p class="paragraph">
							{{ remoteError }}
						</p>
					</div>
				</CfInfo>

				<CfInfo v-else-if="status === 'notfound'" warn icon="user">
					用户不存在：<code>@{{ targetLabel }}</code>
				</CfInfo>

				<CfInfo v-else-if="status === 'self'" warn icon="user">
					不能关注自己。
					<CfA :to="`/@${myUsername}`">打开我的主页</CfA>
				</CfInfo>

				<template v-else-if="status === 'found' && user != null">
					<div class="_gaps">
						<MkUserInfo :user="user" detailed />

						<CfInfo v-if="actionError !== ''" warn icon="warning">
							{{ actionError }}
						</CfInfo>

						<CfInfo v-if="user.isFollowing" icon="check">
							你已经关注了 <b>@{{ user.username }}</b>。
							<CfA :to="`/@${user.username}`">打开主页</CfA>
						</CfInfo>

						<div v-else class="confirm">
							<CfButton
								primary
								gradate
								rounded
								full
								:wait="following"
								@click="follow()"
							>
								<i class="ph-plus ph-bold ph-lg"></i>
								<span>{{ t("follow") }}</span>
							</CfButton>
						</div>
					</div>
				</template>

				<template v-else-if="status === 'done' && user != null">
					<div class="_gaps">
						<div class="success _panel">
							<i class="ph-check-circle ph-bold icon"></i>
							<p class="headline">
								{{ alreadyFollowing ? "你已经在关注" : "已关注" }}
								<b>@{{ user.username }}</b>
							</p>
							<div class="actions">
								<CfButton primary rounded link :to="`/@${user.username}`">
									<i class="ph-user ph-bold ph-lg"></i>
									<span>打开主页</span>
								</CfButton>
								<CfButton rounded link to="/">
									<i class="ph-house ph-bold ph-lg"></i>
									<span>{{ t("home") }}</span>
								</CfButton>
							</div>
						</div>

						<MkUserInfo :user="user" detailed />
					</div>
				</template>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script lang="ts" setup>
// /authorize-follow?acct=@user@host — the target of a "follow me" link that a
// user shares (see FollowMe.vue).
//
// Remote handles are resolved through WebFinger and users/show; local handles
// keep the direct users/show path.
// The acct parser is written here because the firefish-js Acct module is not
// part of the dependency set.
import { computed, onMounted, ref } from "vue";
import { api } from "@/api/client";
import { ApiError } from "@/api/errors";
import type { UserDetailed } from "@/api/types";
import MkUserInfo from "@/components/MkUserInfo.vue";
import CfA from "@/components/ui/CfA.vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import { me } from "@/account";
import { host } from "@/config";
import { t } from "@/i18n";
import * as os from "@/os";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageFollow" });

type Status =
	| "loading"
	| "found"
	| "notfound"
	| "error"
	| "missing"
	| "invalid"
	| "remote"
	| "self"
	| "done";

interface ParsedAcct {
	username: string;
	host: string | null;
}

const status = ref<Status>("loading");
const rawAcct = ref("");
const user = ref<UserDetailed | null>(null);
const following = ref(false);
const alreadyFollowing = ref(false);
const actionError = ref("");
const remoteError = ref("请检查账号写法和本站联邦配置后重试。");

const myUsername = computed(() => me.value?.username ?? "");

const headerActions = computed(() => []);

/** The acct as the user wrote it, without the leading "@". */
const targetLabel = computed(() => rawAcct.value.replace(/^@/, ""));

function readAcct(): string {
	// The acct arrives as a query parameter of a link the user may have opened
	// from another client; a plain `location.search` read keeps it independent
	// of how the router decoded the URL.
	try {
		return new URL(location.href).searchParams.get("acct") ?? "";
	} catch {
		return "";
	}
}

/** `@user@host` / `user@host` / `user` -> parts, or null when malformed. */
function parseAcct(value: string): ParsedAcct | null {
	let rest = value.trim();
	if (rest === "") return null;
	if (rest.startsWith("@")) rest = rest.slice(1);

	let username = rest;
	let remoteHost: string | null = null;
	const at = rest.lastIndexOf("@");
	if (at !== -1) {
		username = rest.slice(0, at);
		const tail = rest.slice(at + 1).trim();
		remoteHost = tail === "" ? null : tail.toLowerCase().replace(/\.$/, "");
	}

	if (!/^[A-Za-z0-9_]+$/.test(username)) return null;
	return { username, host: remoteHost };
}

function isLocal(acct: ParsedAcct): boolean {
	if (acct.host == null) return true;
	return acct.host === host.toLowerCase().replace(/\.$/, "");
}

function describeApiError(error: unknown): string {
	if (error instanceof ApiError) {
		return error.message !== "" ? error.message : t("somethingHappened");
	}
	return t("somethingHappened");
}

async function load(): Promise<void> {
	if (me.value == null) return;

	rawAcct.value = readAcct();
	actionError.value = "";
	alreadyFollowing.value = false;

	if (rawAcct.value === "") {
		status.value = "missing";
		return;
	}

	const parsed = parseAcct(rawAcct.value);
	if (parsed == null) {
		status.value = "invalid";
		return;
	}

	status.value = "loading";

	try {
		if (isLocal(parsed)) {
			user.value = await api<UserDetailed>("users/show", { username: parsed.username });
		} else {
			user.value = await api<UserDetailed>("users/show", {
				username: parsed.username,
				host: parsed.host,
			});
		}
		status.value = "found";
	} catch (error) {
		if (error instanceof ApiError && error.code === "NO_SUCH_USER") {
			status.value = "notfound";
			return;
		}
		remoteError.value = describeApiError(error);
		status.value = isLocal(parsed) ? "error" : "remote";
	}
}

async function follow(): Promise<void> {
	const target = user.value;
	if (target == null || following.value) return;

	const { canceled } = await os.confirm({
		type: "question",
		text: t("followConfirm", {
			name:
				target.name != null && target.name !== ""
					? target.name
					: target.username,
		}),
	});
	if (canceled) return;

	following.value = true;
	actionError.value = "";

	try {
		await api("following/create", { userId: target.id });
		status.value = "done";
	} catch (error) {
		if (error instanceof ApiError) {
			switch (error.code) {
				case "ALREADY_FOLLOWING":
					alreadyFollowing.value = true;
					status.value = "done";
					break;
				case "FOLLOWEE_IS_YOURSELF":
					status.value = "self";
					break;
				default:
					actionError.value = describeApiError(error);
			}
		} else {
			actionError.value = describeApiError(error);
		}
	} finally {
		following.value = false;
	}
}

onMounted(() => {
	void load();
});

definePageMetadata({
	title: t("follow"),
	icon: "ph-user-plus ph-bold ph-lg",
});
</script>

<style lang="scss" scoped>
.follow-page {
	padding-block: 8px;
}

.paragraphs {
	display: flex;
	flex-direction: column;
	gap: 6px;
	text-align: start;
}

.confirm {
	display: flex;
}

.success {
	display: flex;
	flex-direction: column;
	align-items: center;
	padding: 32px 24px;
	text-align: center;

	> .icon {
		font-size: 56px;
		color: var(--accent);
	}

	> .headline {
		margin: 12px 0 0 0;
		font-size: 1.05em;
	}

	> .actions {
		display: flex;
		flex-wrap: wrap;
		justify-content: center;
		gap: 8px;
		margin-block-start: 20px;
	}
}

.paragraph {
	margin: 0;
	line-height: 1.6;
}
</style>
