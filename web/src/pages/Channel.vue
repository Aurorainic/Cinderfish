<!-- Ported from firefish@b425f61:packages/client/src/pages/channel.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader
				:actions="headerActions"
				:tabs="headerTabs"
				:display-back-button="true"
			/>
		</template>
		<MkSpacer :content-max="700" :margin-min="16" :margin-max="32">
			<div v-if="channel" class="cf-channel">
				<div
					class="banner _panel"
					:style="bannerStyle"
				>
					<div class="fade"></div>
					<div class="status">
						<span>
							<i class="ph-users ph-bold ph-lg"></i>
							{{
								t("_channel.usersCount", {
									n: channel.usersCount,
								})
							}}
						</span>
						<span>
							<i class="ph-pencil ph-bold ph-lg"></i>
							{{
								t("_channel.notesCount", {
									n: channel.notesCount,
								})
							}}
						</span>
					</div>
				</div>

				<section class="_panel headerPanel _formBlock">
					<div class="titleRow">
						<div class="title">
							<span class="accent" :style="accentStyle"></span>
							<h1>{{ channel.name }}</h1>
							<span v-if="channel.isArchived" class="archived">
								<i class="ph-archive ph-bold ph-lg"></i>
								Archived
							</span>
						</div>
						<div v-if="signedIn" class="actions">
							<CfButton
								:primary="!isFollowing"
								:disabled="relationBusy"
								@click="toggleFollow"
							>
								<i
									:class="
										isFollowing
											? 'ph-minus ph-bold ph-lg'
											: 'ph-plus ph-bold ph-lg'
									"
								></i>
								{{
									isFollowing ? t("unfollow") : t("follow")
								}}
							</CfButton>
							<CfButton
								:primary="isFavorited"
								:disabled="relationBusy"
								@click="toggleFavorite"
							>
								<i
									class="ph-star ph-bold ph-lg"
									:class="{ 'ph-fill': isFavorited }"
								></i>
								{{
									isFavorited
										? t("unfavorite")
										: t("favorite")
								}}
							</CfButton>
						</div>
					</div>
					<div v-if="channel.description" class="description">
						<Mfm :text="channel.description" :is-note="false" />
					</div>
				</section>

				<CfInfo v-if="channel.isArchived" warn class="_formBlock">
					This channel is archived. New notes cannot be posted to
					it, but its timeline stays readable.
				</CfInfo>

				<MkPostForm
					v-if="signedIn && !channel.isArchived"
					class="_panel _formBlock"
					:channel="channel"
					fixed
					@posted="onPosted"
				/>

				<MkTimeline
					:key="channelId"
					class="_gap"
					src="channel"
					:channel-id="channelId"
				/>
			</div>
			<CfLoading v-else-if="loadError === null" />
			<CfInfo v-else warn class="_formBlock">{{ loadError }}</CfInfo>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script setup lang="ts">
// Firefish's channel page: banner + counters, follow and favourite buttons,
// an inline composer and the channel timeline. Mute is left out (the backend
// has `channels/mute/*`, but no UI key exists for it yet). A channel note is
// forced public + localOnly by the backend, and archived channels reject new
// notes, so the composer is hidden once `isArchived` is set.
import { computed, ref, watch } from "vue";
import { useRoute } from "vue-router";
import { api } from "@/api/client";
import { ENDPOINTS } from "@/api/endpoints";
import type { ChannelShow } from "@/api/types";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";
import MkPostForm from "@/components/MkPostForm.vue";
import MkTimeline from "@/components/MkTimeline.vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import { t } from "@/i18n";
import { me } from "@/account";
import { router } from "@/router";
import * as os from "@/os";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageChannel" });

const route = useRoute();

const channelId = computed<string>(() => {
	const value = route.params.channelId;
	return typeof value === "string" ? value : "";
});

const channel = ref<ChannelShow | null>(null);
const loadError = ref<string | null>(null);
const relationBusy = ref(false);

const signedIn = computed(() => me.value != null);

const isFollowing = computed(() => channel.value?.isFollowing === true);
const isFavorited = computed(() => channel.value?.isFavorited === true);

const bannerStyle = computed(() => {
	const current = channel.value;
	if (current == null) return {};
	if (current.bannerUrl != null && current.bannerUrl !== "") {
		return { backgroundImage: `url(${current.bannerUrl})` };
	}
	return { backgroundColor: current.color !== "" ? current.color : undefined };
});

const accentStyle = computed(() =>
	channel.value != null && channel.value.color !== ""
		? { backgroundColor: channel.value.color }
		: {},
);

const headerTabs = computed(() => []);

const headerActions = computed(() => {
	const current = channel.value;
	if (current == null || me.value == null) return [];
	if (current.userId !== me.value.id) return [];
	return [
		{
			icon: "ph-pencil-simple ph-bold ph-lg",
			text: t("edit"),
			handler: () => void router.push(`/channels/${current.id}/edit`),
		},
	];
});

function errorText(cause: unknown): string {
	return cause instanceof Error ? cause.message : String(cause);
}

async function load(): Promise<void> {
	if (channelId.value === "") return;
	loadError.value = null;
	try {
		channel.value = await api(ENDPOINTS.channelsShow, {
			channelId: channelId.value,
		});
	} catch (cause) {
		channel.value = null;
		loadError.value = errorText(cause);
	}
}

async function toggleFollow(): Promise<void> {
	const current = channel.value;
	if (current == null || relationBusy.value) return;
	const following = current.isFollowing === true;
	relationBusy.value = true;
	try {
		await api(
			following
				? ENDPOINTS.channelsUnfollow
				: ENDPOINTS.channelsFollow,
			{ channelId: current.id },
		);
		channel.value = {
			...current,
			isFollowing: !following,
			usersCount: Math.max(
				0,
				current.usersCount + (following ? -1 : 1),
			),
		};
	} catch (cause) {
		os.toast(errorText(cause), "error");
	} finally {
		relationBusy.value = false;
	}
}

async function toggleFavorite(): Promise<void> {
	const current = channel.value;
	if (current == null || relationBusy.value) return;
	const favorited = current.isFavorited === true;
	relationBusy.value = true;
	try {
		await api(
			favorited
				? ENDPOINTS.channelsUnfavorite
				: ENDPOINTS.channelsFavorite,
			{ channelId: current.id },
		);
		channel.value = { ...current, isFavorited: !favorited };
	} catch (cause) {
		os.toast(errorText(cause), "error");
	} finally {
		relationBusy.value = false;
	}
}

function onPosted(): void {
	os.success();
}

// Reload when the route param changes, e.g. following a link from one channel
// into another without unmounting the page.
watch(
	channelId,
	() => {
		channel.value = null;
		void load();
	},
	{ immediate: true },
);

definePageMetadata(
	computed(() =>
		channel.value == null
			? null
			: {
					title: channel.value.name,
					icon: "ph-television ph-bold",
				},
	),
);
</script>

<style lang="scss" scoped>
.cf-channel {
	> .banner {
		position: relative;
		block-size: 160px;
		background-color: var(--panel);
		background-position: center;
		background-size: cover;
		overflow: hidden;

		> .fade {
			position: absolute;
			inset-block-end: 0;
			inline-size: 100%;
			block-size: 64px;
			background: linear-gradient(transparent, var(--panel));
		}

		> .status {
			position: absolute;
			inset-block-end: 12px;
			inset-inline-end: 12px;
			z-index: 1;
			display: flex;
			align-items: center;
			gap: 12px;
			padding-block: 6px;
			padding-inline: 10px;
			font-size: 0.8em;
			color: var(--fg);
			background: var(--panel);
			border-radius: 6px;
			box-shadow: 0 2px 8px var(--shadow);

			> span {
				display: inline-flex;
				align-items: center;
				gap: 4px;
			}
		}
	}

	> .headerPanel {
		display: flex;
		flex-direction: column;
		gap: 12px;
		padding: 16px;

		> .titleRow {
			display: flex;
			align-items: center;
			flex-wrap: wrap;
			gap: 12px;

			> .title {
				display: flex;
				align-items: center;
				gap: 8px;
				flex: 1;
				min-inline-size: 0;

				> .accent {
					inline-size: 6px;
					block-size: 24px;
					border-radius: 3px;
					background: var(--accent);
				}

				> h1 {
					margin: 0;
					font-size: 1.3em;
					overflow: hidden;
					white-space: nowrap;
					text-overflow: ellipsis;
				}

				> .archived {
					display: inline-flex;
					align-items: center;
					gap: 4px;
					font-size: 0.75em;
					color: var(--fgTransparentWeak);
					border: solid 1px var(--divider);
					border-radius: 999px;
					padding: 2px 8px;
				}
			}

			> .actions {
				display: flex;
				flex-wrap: wrap;
				gap: 8px;
			}
		}

		> .description {
			font-size: 0.95em;
			overflow-wrap: anywhere;
		}
	}
}
</style>
