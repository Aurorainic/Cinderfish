<!-- Ported from firefish@b425f61:packages/client/src/components/global/MkPageHeader.vue (AGPL-3.0-only) -->
<template>
	<header
		v-if="show"
		ref="el"
		class="fdidabkb"
		:class="{ thin: thin_, tabs: hasTabs }"
		:style="{ background: bg }"
		@click="onClick"
	>
		<div class="left">
			<div class="buttons">
				<button
					v-if="displayBackButton"
					:title="t('goBack')"
					class="_buttonIcon button icon backButton"
					@click.stop="goBack()"
					@touchstart="preventDrag"
				>
					<i class="ph-caret-left ph-bold ph-lg"></i>
				</button>
				<MkAvatar
					v-if="narrow && displayMyAvatar && myUser"
					class="avatar button"
					:user="myUser"
					:disable-preview="true"
					disable-link
					@click.stop="openAccountMenu"
				/>
			</div>
			<div
				v-if="!hideTitle && header"
				class="titleContainer"
				@click="showTabsPopup"
			>
				<MkAvatar
					v-if="headerAvatar"
					class="avatar"
					:user="headerAvatar"
					:show-indicator="true"
				/>
				<i
					v-else-if="header.icon && !narrow"
					class="icon"
					:class="header.icon"
				></i>

				<div class="title">
					<MkUserName
						v-if="headerUserName"
						:user="headerUserName"
						:nowrap="true"
						class="title"
					/>
					<!-- Cinderfish: the page title is only rendered when it says
					     more than the site name already shown in the shell sidebar
					     (`CfA.logo > .wordmark`). Without this the timeline reads
					     "C Cinderfish … Cinderfish" on the first screen. -->
					<div
						v-else-if="showHeaderTitle"
						class="title"
					>
						{{ header.title }}
					</div>
					<div v-if="!narrow && header.subtitle" class="subtitle">
						{{ header.subtitle }}
					</div>
				</div>
			</div>
		</div>
		<nav
			v-if="hasTabs"
			ref="tabsEl"
			class="tabs"
			:class="{ collapse: tabs.length > 3 }"
		>
			<button
				v-for="tab in tabs"
				:key="tab.key ?? tab.title"
				:ref="(tabEl) => setTabRef(tab, tabEl)"
				:title="tab.title"
				class="tab _button"
				:class="{ active: tab.key != null && tab.key === props.tab }"
				@mousedown="(ev) => onTabMousedown(tab, ev)"
				@click="(ev) => onTabClick(tab, ev)"
			>
				<i v-if="tab.icon" class="icon" :class="tab.icon"></i>
				<span class="title">{{ tab.title }}</span>
			</button>
			<div ref="tabHighlightEl" class="highlight"></div>
		</nav>
		<div class="buttons right">
			<MkFollowButton
				v-if="followUser"
				:user="followUser"
				:full="!narrow"
				class="fullButton"
			/>
			<button
				v-for="(action, i) in actions"
				:key="i"
				:title="action.text"
				class="_buttonIcon button"
				:class="{ highlighted: action.highlighted }"
				@click.stop="action.handler"
				@touchstart="preventDrag"
			>
				<i :class="action.icon"></i>
			</button>
		</div>
	</header>
</template>

<script setup lang="ts">
import { computed, inject, isRef, nextTick, onMounted, onUnmounted, ref, watch } from "vue";
import { useRouter } from "vue-router";
import type { UserDetailed, UserLite } from "../../api/types";
import { me, signOut } from "../../account";
import { t } from "../../i18n";
import { instance } from "../../instance";
import { popupMenu } from "../../os";
import {
	injectPageMetadata,
	type PageMetadata,
} from "../../scripts/page-metadata";
import { scrollToTop } from "../../scripts/scroll";
import { userPage } from "../ui/user";
import MkAvatar from "./MkAvatar.vue";
import MkFollowButton from "../MkFollowButton.vue";
import MkUserName from "./MkUserName.vue";

/** Compare titles ignoring surrounding space and letter case. */
function normalizeTitle(value: string): string {
	return value.trim().toLocaleLowerCase();
}

interface Tab {
	key?: string | null;
	title: string;
	icon?: string;
	iconOnly?: boolean;
	onClick?: (ev: MouseEvent) => void;
}

const props = withDefaults(
	defineProps<{
		tabs?: Tab[];
		tab?: string;
		actions?: {
			text: string;
			icon: string;
			highlighted?: boolean;
			handler: (ev: MouseEvent) => void;
		}[];
		thin?: boolean;
		displayMyAvatar?: boolean;
		displayBackButton?: boolean;
		/** Alias kept for call sites that name the prop "backButton". */
		backButton?: boolean;
		/** Fallback title/icon for pages that do not call definePageMetadata. */
		title?: string;
		icon?: string | null;
		to?: string;
	}>(),
	{
		tabs: () => [],
		tab: undefined,
		actions: () => [],
		thin: false,
		displayMyAvatar: false,
		displayBackButton: false,
		backButton: false,
		title: undefined,
		icon: null,
		to: undefined,
	},
);

const emit = defineEmits<{
	"update:tab": [key: string];
}>();

const router = useRouter();

// Firefish disables the back button when the caller did not ask for it or when
// there is nothing to go back to; the surrounding modal shell can veto it.
const shouldBackButton = inject<boolean>("shouldBackButton", true);
const displayBackButton = computed<boolean>(
	() =>
		(props.displayBackButton || props.backButton) &&
		window.history.length > 1 &&
		shouldBackButton,
);

// The shell may hide the title (e.g. zen layout) and may force a thin header.
const hideTitle = inject<boolean>("shouldOmitHeaderTitle", false);
const shouldHeaderThin = inject<boolean>("shouldHeaderThin", false);
const thin_ = computed<boolean>(() => props.thin || shouldHeaderThin);

// Firefish injects the ref provided by definePageMetadata; the shim declares a
// plain object, so unwrap defensively and keep a reactive view of it.
const injectedMetadata = injectPageMetadata() as unknown;
const pageMetadata = computed<PageMetadata | null>(() => {
	if (injectedMetadata == null) return null;
	const value = isRef(injectedMetadata)
		? (injectedMetadata.value as PageMetadata | null)
		: (injectedMetadata as PageMetadata | null);
	return value ?? null;
});

const header = computed<PageMetadata | null>(() => {
	const metadata = pageMetadata.value;
	const title = metadata?.title ?? props.title;
	const icon = metadata?.icon ?? props.icon ?? null;
	if (metadata == null && (title == null || title === "")) return null;
	return {
		title: title ?? "",
		subtitle: metadata?.subtitle,
		icon,
		avatar: metadata?.avatar ?? null,
		userName: metadata?.userName ?? null,
		bg: metadata?.bg,
	};
});

// Narrowed views keep the template free of casts: each of these is checked with
// v-if on the element that consumes it.
const myUser = computed<UserLite | null>(() => me.value ?? null);
const myProfilePath = computed<string>(() =>
	myUser.value == null ? "/" : userPage(myUser.value),
);
const headerAvatar = computed<UserLite | null>(
	() => header.value?.avatar ?? null,
);
const headerUserName = computed<UserLite | null>(
	() => header.value?.userName ?? null,
);

// The universal shell already prints the site name in its sidebar logo, so a
// page header whose title is exactly that name would repeat it on screen (the
// placeholder title is filled from `meta.name`, falling back to "Cinderfish").
const repeatsShellTitle = computed<boolean>(() => {
	const title = normalizeTitle(header.value?.title ?? "");
	if (title === "") return false;
	const site = normalizeTitle(instance.name ?? "");
	return title === site || title === "cinderfish";
});
const showHeaderTitle = computed<boolean>(() => {
	const value = header.value;
	if (value == null || value.title === "") return false;
	if (repeatsShellTitle.value) return false;
	// Tabs already name the page on narrow screens.
	return !(hasTabs.value && narrow.value);
});

const followUser = computed<UserDetailed | null>(() => {
	const avatar = header.value?.avatar;
	if (avatar == null) return null;
	return avatar as unknown as UserDetailed;
});

const hasTabs = computed<boolean>(() => props.tabs.length > 0);
const hasActions = computed<boolean>(() => props.actions.length > 0);
const show = computed<boolean>(() => !hideTitle || hasTabs.value || hasActions.value);

const el = ref<HTMLElement | null>(null);
const tabsEl = ref<HTMLElement | null>(null);
const tabHighlightEl = ref<HTMLElement | null>(null);
const bg = ref<string | undefined>(undefined);
const narrow = ref(false);

const tabRefs: Record<string, HTMLElement | null> = {};

function setTabRef(tab: Tab, tabEl: unknown): void {
	const key = tab.key ?? tab.title;
	tabRefs[key] = tabEl instanceof HTMLElement ? tabEl : null;
}

function updateHighlight(): void {
	void nextTick(() => {
		if (props.tab == null) return;
		const tabEl = tabRefs[props.tab];
		if (tabEl == null || tabHighlightEl.value == null) return;
		// getBoundingClientRect would round less, but offsetWidth/offsetLeft match
		// upstream's behaviour closely enough for the sliding indicator.
		const tabSizeX = tabEl.scrollWidth + 20;
		if (props.tabs.length > 3) {
			tabEl.style.setProperty("--width", tabSizeX + "px");
		}
		window.setTimeout(() => {
			if (tabHighlightEl.value == null) return;
			tabHighlightEl.value.style.width = tabSizeX + "px";
			tabHighlightEl.value.style.transform =
				"translateX(" + tabEl.offsetLeft + "px)";
			window.requestAnimationFrame(() => {
				tabsEl.value?.scrollTo({
					left: tabEl.offsetLeft - 60,
					behavior: "smooth",
				});
			});
		}, 200);
	});
}

watch(() => [props.tab, props.tabs], updateHighlight, { immediate: true });

// Firefish's openAccountMenu lives in account.ts and offers the whole account
// switcher. Cinderfish has no account switcher yet, so the avatar opens a small
// menu with the entries the shell can actually serve.
function openAccountMenu(ev: MouseEvent): void {
	void popupMenu(
		[
			{
				text: t("profile"),
				icon: "ph-bold ph-user",
				action: () => {
					void router.push(myProfilePath.value);
				},
			},
			{
				text: t("settings"),
				icon: "ph-bold ph-gear",
				action: () => {
					void router.push("/settings");
				},
			},
			{ separator: true },
			{
				text: t("logout"),
				icon: "ph-bold ph-sign-out",
				action: () => {
					void signOut();
				},
			},
		],
		ev,
	);
}

function preventDrag(ev: TouchEvent): void {
	ev.stopPropagation();
}

function onClick(): void {
	if (props.to != null) {
		// Firefish assigns location.href here (a page reload); the port keeps the
		// router in charge so the SPA never reloads.
		void router.push(props.to);
	} else {
		scrollToTop(el.value, { behavior: "smooth" });
	}
}

function showTabsPopup(ev: MouseEvent): void {
	if (!hasTabs.value) return;
	if (!narrow.value) return;
	ev.preventDefault();
	ev.stopPropagation();
	const menu = props.tabs.map((tab) => ({
		text: tab.title,
		icon: tab.icon,
		active: tab.key != null && tab.key === props.tab,
		action: () => {
			onTabClick(tab, ev);
		},
	}));
	popupMenu(menu, ev);
}

function onTabMousedown(tab: Tab, _ev: MouseEvent): void {
	// ユーザビリティの観点からmousedown時にはonClickは呼ばない
	if (tab.key) emit("update:tab", tab.key);
}

function onTabClick(tab: Tab, ev: MouseEvent): void {
	if (tab.onClick) {
		ev.preventDefault();
		ev.stopPropagation();
		tab.onClick(ev);
	}
	if (tab.key) emit("update:tab", tab.key);
}

function goBack(): void {
	window.history.back();
}

let ro: ResizeObserver | null = null;

onMounted(() => {
	const parent = el.value?.parentElement;
	if (parent != null) {
		narrow.value = parent.offsetWidth < 500;
		ro = new ResizeObserver(() => {
			if (el.value == null) return;
			const current = el.value.parentElement;
			if (current != null && document.body.contains(el.value)) {
				narrow.value = current.offsetWidth < 500;
			}
		});
		ro.observe(parent);
	}
});

onUnmounted(() => {
	if (ro != null) ro.disconnect();
});
</script>

<style lang="scss" scoped>
.fdidabkb {
	--height: 55px;
	display: flex;
	justify-content: space-between;
	width: 100%;
	height: var(--height);
	padding-inline: 24px;
	box-sizing: border-box;
	overflow: hidden;
	-webkit-backdrop-filter: var(--blur, blur(15px));
	backdrop-filter: var(--blur, blur(15px));
	@media (max-width: 500px) {
		padding-inline: 16px;
		&.tabs > .buttons > :deep(.follow-button > span) {
			display: none;
		}
	}
	@media (max-width: 700px) {
		> .left {
			min-width: unset;
			max-width: 40%;
		}
		> .left,
		> .right {
			flex: unset;
		}
		&:not(.tabs) {
			> .left {
				width: 0;
				flex-grow: 1;
				max-width: unset;
			}
		}
		&.tabs {
			> .left {
				flex-shrink: 0;
			}

			.buttons ~ .titleContainer > .title {
				display: none;
			}
		}
	}

	&::before {
		content: "";
		position: absolute;
		inset: 0;
		border-bottom: solid 0.5px var(--divider);
		z-index: -1;
	}
	&::after {
		content: "";
		position: absolute;
		inset: 0;
		background: var(--bg);
		opacity: 0.85;
		z-index: -2;
	}

	&.thin {
		--height: 45px;

		.buttons {
			> .button {
				font-size: 0.9em;
			}
		}
	}

	> .left {
		display: flex;
		> .buttons {
			&:not(:empty) {
				margin-right: 8px;
				margin-left: calc(0px - var(--margin));
			}
			> .avatar {
				width: 32px;
				height: 32px;
				margin-left: var(--margin);
			}
		}
	}

	.buttons {
		--margin: 8px;
		display: flex;
		align-items: center;
		height: var(--height);
		&.right {
			justify-content: flex-end;
			z-index: 2;
			> .button:last-child {
				margin-right: calc(0px - var(--margin));
			}
		}

		> .fullButton {
			& + .fullButton {
				margin-left: 12px;
			}
		}
	}

	> .left {
		> .backButton {
			display: flex;
			align-items: center;
			justify-content: center;
		}
		> .titleContainer {
			display: flex;
			align-items: center;
			max-width: 400px;
			overflow: auto;
			white-space: nowrap;
			text-align: left;
			font-weight: bold;
			flex-shrink: 0;
			> .avatar {
				$size: 32px;
				display: inline-block;
				width: $size;
				height: $size;
				vertical-align: bottom;
				margin-right: 8px;
			}

			> .icon {
				margin-right: 8px;
				min-width: 16px;
				width: 1em;
				text-align: center;
			}

			> .title {
				min-width: 0;
				overflow: hidden;
				text-overflow: ellipsis;
				white-space: nowrap;
				line-height: 1.1;

				> .subtitle {
					opacity: 0.6;
					font-size: 0.8em;
					font-weight: normal;
					white-space: nowrap;
					overflow: hidden;
					text-overflow: ellipsis;

					&.activeTab {
						text-align: center;

						> .chevron {
							display: inline-block;
							margin-left: 6px;
						}
					}
				}
			}
		}
	}

	> .left,
	> .right {
		flex-basis: 100%;
		flex-shrink: 9999;
		overflow: hidden;
	}
	> .left {
		min-width: 20%;
		margin-left: -10px;
		padding-left: 10px;
	}
	> .right {
		min-width: max-content;
		margin-right: -10px;
		padding-right: 10px;
	}

	> .tabs {
		position: relative;
		font-size: 1em;
		overflow-x: auto;
		white-space: nowrap;
		contain: content;
		display: flex;
		padding-inline: 20px;
		margin-inline: -20px;
		mask: linear-gradient(
			to right,
			transparent,
			black 20px calc(100% - 20px),
			transparent
		);
		-webkit-mask: linear-gradient(
			to right,
			transparent,
			black 20px calc(100% - 20px),
			transparent
		);
		scrollbar-width: none;
		&::-webkit-scrollbar {
			display: none;
		}

		&.collapse {
			--width: 2.7em;
			> .tab {
				width: 2.7em;
				min-width: 2.7em;
				&:not(.active) > .title {
					opacity: 0;
				}
			}
		}
		&:not(.collapse) > .tab {
			--width: max-content;
		}

		> .tab {
			display: inline-flex;
			align-items: center;
			position: relative;
			border-inline: 10px solid transparent;
			height: 100%;
			min-width: max-content;
			font-weight: normal;
			opacity: 0.7;
			overflow: hidden;
			transition:
				color 0.2s,
				opacity 0.2s,
				width 0.2s,
				min-width 0.2s;
			--width: 38px;

			&:hover {
				opacity: 1;
				text-decoration: none;
			}

			&.active {
				opacity: 1;
				color: var(--accent);
				font-weight: 600;
				width: var(--width);
				min-width: var(--width);
			}

			> .icon + .title {
				margin-left: 8px;
			}
			> .title {
				transition: opacity 0.2s;
			}
		}
		> .highlight {
			position: absolute;
			bottom: 0;
			left: 0;
			height: 3px;
			background: var(--accent);
			border-radius: 999px;
			transition:
				width 0.2s,
				transform 0.2s;
			transition-timing-function: cubic-bezier(0, 0, 0, 1.2);
			pointer-events: none;
		}
	}
}
</style>
