<!-- Ported from firefish@b425f61:packages/client/src/pages/welcome.vue (AGPL-3.0-only) -->
<template>
	<div class="welcome">
		<CfLoading v-if="loading" />

		<template v-else>
			<CfInfo v-if="!metaLoaded" class="_gap" warn>
				读取实例信息失败（网络或服务器错误），当前只显示本站地址
				<code>{{ host }}</code>。登录与注册仍然可用。
			</CfInfo>

			<section class="hero _panel">
				<div class="icon">
					<img v-if="iconUrl !== ''" :src="iconUrl" alt="" />
					<span v-else class="mark">C</span>
				</div>

				<h1 class="name">{{ displayName }}</h1>
				<p class="host">{{ host }}</p>

				<p v-if="description !== ''" class="description">
					<Mfm :text="description" />
				</p>
				<p v-else class="description muted">
					这是一个由 Cinderfish 驱动的小型联合社交实例：发帖、回应、关注，并与其它
					ActivityPub 实例互通。
				</p>

				<div v-if="me == null" class="actions">
					<CfButton primary gradate rounded @click="dialog = 'signup'">
						<i class="ph-user-plus ph-bold ph-lg"></i>
						<span>{{ registrationOpen ? t("signup") : "使用邀请码注册" }}</span>
					</CfButton>
					<CfButton rounded @click="dialog = 'signin'">
						<i class="ph-sign-in ph-bold ph-lg"></i>
						<span>{{ t("login") }}</span>
					</CfButton>
				</div>
				<div v-else class="actions">
					<CfButton primary rounded link to="/">
						<i class="ph-house ph-bold ph-lg"></i>
						<span>{{ t("home") }}</span>
					</CfButton>
				</div>

				<p class="links">
					<CfA to="/explore">{{ t("explore") }}</CfA>
					<span class="dot">·</span>
					<CfA to="/about">{{ t("instanceInfo") }}</CfA>
				</p>
			</section>

			<section class="facts _panel">
				<h2 class="title">本实例提供</h2>
				<ul class="list">
					<li v-for="fact in facts" :key="fact.title">
						<i :class="fact.icon"></i>
						<div class="body">
							<b>{{ fact.title }}</b>
							<span>{{ fact.detail }}</span>
						</div>
					</li>
				</ul>
			</section>

			<p class="powered">
				Powered by Cinderfish<span v-if="version !== ''"> {{ version }}</span>
			</p>
		</template>

		<transition name="welcome-dialog">
			<div
				v-if="dialog != null"
				class="dialog-back _modalBg"
				@click.self="dialog = null"
			>
				<div class="dialog _panel">
					<button
						class="close _buttonIcon"
						:aria-label="t('close')"
						@click="dialog = null"
					>
						<i class="ph-x ph-bold ph-lg"></i>
					</button>
					<CfSignin v-if="dialog === 'signin'" @login="onSignedIn()" />
					<CfSignup v-else @signup="onSignedIn()" />
				</div>
			</div>
		</transition>
	</div>
</template>

<script lang="ts" setup>
// The visitor landing page. Firefish splits this into welcome.vue (a meta
// fetch plus XSetup/XEntrance) and three interchangeable welcome.entrance.*
// variants; the setup wizard and the two other variants are out of scope
// (wave-4 contract §2), so this is one entrance rendered from the shared
// reactive `instance` object.
//
// The sign-in and sign-up forms are the account layer's own components
// (POST /api/signin, /api/signup), opened in the same modal VisitorLayout.vue
// uses, so credentials are handled in exactly one place.
//
// What the page claims about the instance comes from `meta`:
//   * registration  -> `disableRegistration` (open vs invite/admin only);
//   * timelines     -> `features.globalTimeline`;
//   * drive storage -> `features.objectStorage`;
//   * federation    -> `federation` (all | specified | none).
// Nothing here is invented: a value the backend does not report is described
// as unknown rather than guessed.
import { computed, onMounted, ref } from "vue";
import CfA from "@/components/ui/CfA.vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import { CfSignin, CfSignup } from "@/account/components";
import { me } from "@/account";
import { host } from "@/config";
import { instance, updateInstance } from "@/instance";
import { t } from "@/i18n";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageWelcome" });

type Dialog = "signin" | "signup" | null;

interface Fact {
	title: string;
	detail: string;
	icon: string;
}

const dialog = ref<Dialog>(null);
const settled = ref(false);

onMounted(() => {
	// The cached copy (or main.ts' request, already in flight) usually answers
	// immediately; only an empty instance needs its own fetch.
	if (metaLoaded.value) {
		settled.value = true;
		return;
	}
	// `updateInstance()` shares the in-flight request with main.ts and falls
	// back to the cached copy, so this is cheap for a visitor landing here
	// first.
	void updateInstance().finally(() => {
		settled.value = true;
	});
});

/** True once a real meta payload (not just the defaults) is in `instance`. */
const metaLoaded = computed(
	() =>
		instance.version !== "" ||
		instance.name !== "" ||
		instance.description !== "",
);

const loading = computed(() => !settled.value && !metaLoaded.value);

const displayName = computed(() =>
	instance.name !== "" ? instance.name : host,
);

const description = computed(() => instance.description ?? "");

const iconUrl = computed(() => instance.iconUrl ?? "");

const version = computed(() => instance.version ?? "");

const registrationOpen = computed(() => instance.disableRegistration !== true);

const federationDetail = computed(() => {
	switch (instance.federation) {
		case "all":
			return "公开联合：可以与任何支持 ActivityPub 的实例互通（对方实例的屏蔽规则照常生效）";
		case "specified":
			return "仅白名单实例：只与管理员允许的实例互通";
		case "none":
			return "未启用联合：只与本站用户互动，不接收也不投递远程内容";
		default:
			return "联合模式未知：服务器没有报告联邦状态";
	}
});

const facts = computed<Fact[]>(() => {
	const list: Fact[] = [
		{
			title: "笔记",
			detail: `每条最多 ${instance.maxNoteTextLength} 字，支持表情回应、转帖与引用`,
			icon: "ph-pencil-simple ph-bold",
		},
		{
			title: "时间线",
			detail:
				instance.features?.globalTimeline === true
					? "本地时间线与全局时间线"
					: "本地时间线（当前联邦模式下全局时间线不可用）",
			icon: "ph-clock ph-bold",
		},
		{
			title: "网盘",
			detail:
				instance.features?.objectStorage === true
					? "上传文件到网盘（服务器使用对象存储）"
					: "上传文件到网盘（服务器使用本地磁盘）",
			icon: "ph-cloud-arrow-up ph-bold",
		},
		{
			title: "联合",
			detail: federationDetail.value,
			icon: "ph-globe-hemisphere-west ph-bold",
		},
		{
			title: "注册",
			detail: registrationOpen.value
				? "开放注册：可以直接创建账号"
				: "自助注册已关闭：需要邀请码，或由管理员创建账号",
			icon: "ph-user-plus ph-bold",
		},
	];

	return list;
});

function onSignedIn(): void {
	// The account layer updated `me`; main.ts swaps the visitor shell for the
	// universal one and IndexRoute renders the timeline, so closing the modal
	// is all that is left to do.
	dialog.value = null;
}

definePageMetadata(
	computed(() => ({
		title: displayName.value,
		icon: null,
	})),
);
</script>

<style lang="scss" scoped>
.welcome {
	display: flex;
	flex-direction: column;
	gap: 16px;
	padding-block: 16px;
}

.hero {
	display: flex;
	flex-direction: column;
	align-items: center;
	padding: 40px 24px 24px;
	text-align: center;

	> .icon {
		display: flex;
		align-items: center;
		justify-content: center;
		inline-size: 76px;
		block-size: 76px;
		margin-block-end: 16px;
		border-radius: 999px;
		overflow: clip;
		background: var(--buttonBg);

		> img {
			inline-size: 100%;
			block-size: 100%;
			object-fit: cover;
		}

		> .mark {
			display: inline-flex;
			align-items: center;
			justify-content: center;
			inline-size: 100%;
			block-size: 100%;
			font-size: 34px;
			font-weight: 800;
			color: var(--fgOnAccent);
			background: linear-gradient(
				135deg,
				var(--buttonGradateA),
				var(--buttonGradateB)
			);
		}
	}

	> .name {
		margin: 0;
		font-size: 1.6em;
		line-height: 1.3;
	}

	> .host {
		margin: 4px 0 0 0;
		font-size: 0.85em;
		color: var(--fgTransparentWeak);
	}

	> .description {
		margin: 20px 0 0 0;
		max-inline-size: 34em;
		line-height: 1.7;
		text-align: start;
		white-space: pre-wrap;

		&.muted {
			color: var(--fgTransparentWeak);
			text-align: center;
		}
	}

	> .actions {
		display: flex;
		flex-wrap: wrap;
		justify-content: center;
		gap: 12px;
		margin-block-start: 28px;
	}

	> .links {
		margin: 20px 0 0 0;
		font-size: 0.9em;
		color: var(--fgTransparentWeak);

		> a {
			color: var(--accent);
		}

		> .dot {
			margin-inline: 8px;
		}
	}
}

.facts {
	padding: 20px 24px 24px;

	> .title {
		margin: 0 0 12px 0;
		font-size: 0.95em;
		color: var(--fgTransparentWeak);
	}

	> .list {
		display: flex;
		flex-direction: column;
		gap: 12px;
		margin: 0;
		padding: 0;
		list-style: none;

		> li {
			display: flex;
			align-items: flex-start;
			gap: 12px;

			> i {
				margin-block-start: 2px;
				font-size: 1.1em;
				color: var(--accent);
			}

			> .body {
				display: flex;
				flex-direction: column;
				gap: 2px;
				font-size: 0.92em;

				> b {
					font-weight: 700;
				}

				> span {
					color: var(--fgTransparentWeak);
					line-height: 1.5;
				}
			}
		}
	}
}

.powered {
	margin: 0;
	padding-block: 8px;
	text-align: center;
	font-size: 0.8em;
	color: var(--fgTransparentWeak);
}

.dialog-back {
	z-index: 1000;
	display: flex;
	align-items: center;
	justify-content: center;
	padding: var(--margin);
	box-sizing: border-box;
}

.dialog {
	position: relative;
	inline-size: 100%;
	max-inline-size: 380px;
	max-block-size: calc(100vh - var(--margin) * 2);
	overflow: auto;
	padding: 32px;
	box-sizing: border-box;

	> .close {
		position: absolute;
		inset-block-start: 8px;
		inset-inline-end: 8px;
	}
}

.welcome-dialog-enter-active,
.welcome-dialog-leave-active {
	opacity: 1;
	transition: opacity 200ms ease;
}

.welcome-dialog-enter-from,
.welcome-dialog-leave-to {
	opacity: 0;
}
</style>
