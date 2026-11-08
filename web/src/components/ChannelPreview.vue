<!-- Ported from firefish@b425f61:packages/client/src/components/MkChannelPreview.vue (AGPL-3.0-only) -->
<template>
	<MkA :to="`/channels/${channel.id}`" class="cf-channel-preview _panel">
		<div class="banner" :style="bannerStyle">
			<div class="name">
				<i class="ph-television ph-bold ph-lg"></i>
				<span>{{ channel.name }}</span>
			</div>
			<div class="status">
				<span>
					<i class="ph-users ph-bold ph-lg"></i>
					{{ t("_channel.usersCount", { n: channel.usersCount }) }}
				</span>
				<span>
					<i class="ph-pencil ph-bold ph-lg"></i>
					{{ t("_channel.notesCount", { n: channel.notesCount }) }}
				</span>
			</div>
		</div>
		<div v-if="excerpt !== ''" class="description" :title="channel.description ?? ''">
			{{ excerpt }}
		</div>
		<footer>
			<span v-if="channel.isArchived" class="flag">
				<i class="ph-archive ph-bold ph-lg"></i>
				Archived
			</span>
			<span v-if="channel.isFollowing" class="flag">
				<i class="ph-check ph-bold ph-lg"></i>
				{{ t("following") }}
			</span>
			<span v-if="channel.lastNotedAt != null" class="updated">
				{{ t("updatedAt") }}:
				<CfTime :time="channel.lastNotedAt" />
			</span>
		</footer>
	</MkA>
</template>

<script setup lang="ts">
// Firefish's channel card. The 200px banner is shortened so the list stays
// scannable, and the "updated at" line uses the packed `lastNotedAt`.
import { computed } from "vue";
import type { Channel } from "@/api/types";
import CfTime from "@/components/ui/CfTime.vue";
import { t } from "@/i18n";

const props = defineProps<{
	channel: Channel;
}>();

const bannerStyle = computed(() =>
	props.channel.bannerUrl != null && props.channel.bannerUrl !== ""
		? { backgroundImage: `url(${props.channel.bannerUrl})` }
		: {
				backgroundColor:
					props.channel.color !== "" ? props.channel.color : undefined,
			},
);

const excerpt = computed<string>(() => {
	const description = props.channel.description ?? "";
	if (description === "") return "";
	return description.length > 85
		? `${description.slice(0, 85)}…`
		: description;
});
</script>

<style lang="scss" scoped>
.cf-channel-preview {
	display: block;
	overflow: hidden;
	inline-size: 100%;
	margin-block-end: var(--margin);

	&:hover {
		text-decoration: none;
	}

	> .banner {
		position: relative;
		inline-size: 100%;
		block-size: 96px;
		background-color: var(--accent);
		background-position: center;
		background-size: cover;

		> .name {
			position: absolute;
			inset-block-start: 12px;
			inset-inline-start: 12px;
			display: flex;
			align-items: center;
			gap: 6px;
			max-inline-size: calc(100% - 24px);
			padding-block: 6px;
			padding-inline: 12px;
			font-size: 1.05em;
			font-weight: bold;
			color: var(--fg);
			background: var(--panel);
			border-radius: 999px;

			> span {
				overflow: hidden;
				white-space: nowrap;
				text-overflow: ellipsis;
			}
		}

		> .status {
			position: absolute;
			inset-block-end: 8px;
			inset-inline-end: 8px;
			display: flex;
			align-items: center;
			gap: 10px;
			padding-block: 4px;
			padding-inline: 8px;
			font-size: 0.78em;
			color: var(--fg);
			background: var(--panel);
			border-radius: 6px;

			> span {
				display: inline-flex;
				align-items: center;
				gap: 4px;
			}
		}
	}

	> .description {
		padding: 12px 16px;
		font-size: 0.9em;
		overflow-wrap: anywhere;
	}

	> footer {
		display: flex;
		align-items: center;
		gap: 12px;
		padding: 8px 16px;
		border-block-start: solid 0.5px var(--divider);
		font-size: 0.8em;
		color: var(--fgTransparentWeak);

		.flag {
			display: inline-flex;
			align-items: center;
			gap: 4px;
		}

		.updated {
			margin-inline-start: auto;
		}
	}
}
</style>
