<!-- Ported from firefish@b425f61:packages/client/src/components/MkUserInfo.vue (AGPL-3.0-only) -->
<template>
	<article class="_panel user-card">
		<h3 class="title">
			<span
				v-if="followsYou"
				class="followed"
				>{{ t("followsYou") }}</span
			>
			<MkAvatar
				class="avatar"
				:user="user"
				:disable-preview="true"
				:show-indicator="true"
			/>
			<MkA class="name" :to="userPage(user)"
				><MkUserName :user="user" :nowrap="true"
			/></MkA>
			<p class="username"><MkAcct :user="user" /></p>
			<div class="badges">
				<span v-if="detail?.isAdmin" :title="t('isAdmin')" class="badge"
					><i class="ph-bookmark-simple ph-fill ph-lg"></i
				></span>
				<span
					v-else-if="detail?.isModerator"
					:title="t('isModerator')"
					class="badge"
					><i class="ph-bookmark-simple ph-bold ph-lg"></i
				></span>
				<span v-if="detail?.isLocked" :title="t('isLocked')"
					><i class="ph-lock ph-bold ph-lg"></i
				></span>
				<span v-if="user.isBot" :title="t('isBot')"
					><i class="ph-robot ph-bold ph-lg"></i
				></span>
			</div>
		</h3>
		<div class="description" :class="{ collapsed, truncate: !detailed }">
			<Mfm
				v-if="detail?.description"
				class="mfm"
				:text="detail.description"
				:author="user"
				:i="me"
				:custom-emojis="emojis"
			/>
			<span v-else class="empty">{{ t("noAccountDescription") }}</span>
		</div>
		<MkShowMoreButton
			v-if="isLong"
			v-model="collapsed"
			class="showMore"
		></MkShowMoreButton>
		<div v-if="fields.length > 0 && detailed" class="fields">
			<dl v-for="(field, i) in fields" :key="i" class="field">
				<dt class="name">
					<Mfm
						:text="field.name"
						:plain="true"
						:custom-emojis="emojis"
					/>
				</dt>
				<dd class="value">
					<Mfm :text="field.value" :author="user" :i="me" :custom-emojis="emojis" />
				</dd>
			</dl>
		</div>
		<div class="status">
			<p>
				<MkNumber :value="detail?.notesCount ?? 0" />
				{{ t("notes") }}
			</p>
			<p>
				<MkNumber :value="detail?.followingCount ?? 0" />
				{{ t("following") }}
			</p>
			<p>
				<MkNumber :value="detail?.followersCount ?? 0" />
				{{ t("followers") }}
			</p>
		</div>
		<div class="buttons">
			<slot>
				<template v-if="detail != null && me != null && me.id !== user.id">
					<MkFollowButton :user="detail" />
				</template>
			</slot>
		</div>
	</article>
</template>

<script lang="ts" setup>
// Degradations versus Firefish:
//   * no banner: `bannerUrl` is always null and the blurred backdrop needs the
//     banner image (docs/research/api-contract/auth-users.md);
//   * no field "verified" seal: `fields` is always [] on Cinderfish;
//   * the card receives a UserLite per the cross-scope interface (contract
//     §3.5); counts, description and fields are read only when the caller
//     passed a UserDetailed, so a lite user renders without faking zeros.
import { computed, ref } from "vue";
import type { EmojiSimple, UserDetailed, UserLite } from "@/api/types";
import { me } from "@/account";
import MkFollowButton from "@/components/MkFollowButton.vue";
import MkNumber from "@/components/MkNumber.vue";
import MkShowMoreButton from "@/components/MkShowMoreButton.vue";
import { userPage } from "@/filters/user";
import { t } from "@/i18n";

const props = withDefaults(
	defineProps<{
		user: UserLite;
		detailed?: boolean;
	}>(),
	{
		detailed: false,
	},
);

interface ProfileField {
	name: string;
	value: string;
	verified?: boolean;
}

function isDetailed(user: UserLite): user is UserDetailed {
	return typeof (user as Partial<UserDetailed>).notesCount === "number";
}

const detail = computed<UserDetailed | null>(() =>
	isDetailed(props.user) ? props.user : null,
);

const emojis = computed<EmojiSimple[]>(
	() => (props.user.emojis ?? []) as unknown as EmojiSimple[],
);

const fields = computed<ProfileField[]>(
	() => (detail.value?.fields ?? []) as ProfileField[],
);

const followsYou = computed<boolean>(
	() =>
		me.value != null &&
		me.value.id !== props.user.id &&
		detail.value?.isFollowed === true,
);

const isLong = computed<boolean>(
	() =>
		props.detailed &&
		detail.value?.description != null &&
		(detail.value.description.split("\n").length > 9 ||
			detail.value.description.length > 400),
);

const collapsed = ref(isLong.value);
</script>

<style lang="scss" scoped>
.user-card {
	position: relative;
	display: flex;
	flex-direction: column;

	.title {
		position: relative;
		display: block;
		padding: 10px;
		padding-inline-start: 84px;
		margin: 0;
		font-size: 1em;
		border-radius: var(--radius);
		background: var(--panel);
		line-height: 1;
		z-index: 4;

		> .followed {
			display: inline-block;
			margin-bottom: 6px;
			padding: 4px 8px;
			background: var(--buttonBg);
			font-size: 0.7em;
			border-radius: 6px;
		}

		.avatar {
			display: block;
			position: absolute;
			bottom: 6px;
			inset-inline-start: 10px;
			z-index: 2;
			width: 58px;
			height: 58px;
			background: var(--panel);
			border: solid 4px var(--panel);
		}

		.name {
			display: inline-block;
			margin: 0;
			font-weight: bold;
			line-height: 16px;
			overflow: hidden;
			text-overflow: ellipsis;
			height: 1.5em;
			margin-block: -0.5em;
			padding-block: 0.5em 0.25em;
			max-width: 100%;
		}

		.username {
			display: block;
			margin: 4px 0 0;
			font-size: 0.8em;
			color: var(--fgTransparentWeak);
			font-weight: 500;
			line-height: 1;
		}

		> .badges {
			display: inline-flex;
			align-items: center;
			gap: 4px;
			margin-inline-start: 6px;
			vertical-align: middle;
			opacity: 0.7;
		}
	}

	.description {
		overflow: hidden;
		padding-inline: 16px;
		margin-bottom: 10px;
		font-size: 0.8em;

		&.truncate {
			display: -webkit-box;
			-webkit-line-clamp: 6;
			-webkit-box-orient: vertical;
		}

		&.collapsed {
			position: relative;
			max-height: calc(9em + 50px);
			mask: linear-gradient(black calc(100% - 64px), transparent);
			-webkit-mask: linear-gradient(black calc(100% - 64px), transparent);
		}

		> .empty {
			color: var(--fgTransparentWeak);
		}
	}

	:deep(.fade) {
		position: relative;
		display: block;
		width: 100%;
		margin-top: -4.5em;
		z-index: 2;
	}

	:deep(.showLess) {
		width: 100%;
		position: sticky;
		bottom: var(--stickyBottom);
	}

	> .fields {
		padding-inline: 16px;
		font-size: 0.8em;
		padding-block: 1em;
		border-top: 1px solid var(--divider);

		> .field {
			display: flex;
			padding: 0;
			margin: 0;
			align-items: center;

			&:not(:last-child) {
				margin-bottom: 8px;
			}

			> .name {
				width: 30%;
				overflow: hidden;
				white-space: nowrap;
				text-overflow: ellipsis;
				font-weight: bold;
				text-align: center;
				padding-inline-end: 10px;
			}

			> .value {
				width: 70%;
				overflow: hidden;
				white-space: nowrap;
				text-overflow: ellipsis;
				margin: 0;
			}
		}
	}

	.status {
		display: flex;
		gap: 1em;
		padding-inline: 16px;
		font-size: 0.8em;
		margin-top: auto;
		border-top: 1px solid var(--divider);

		> p > :deep(span) {
			font-weight: 700;
			color: var(--accent);
		}
	}

	.buttons {
		position: absolute;
		top: 8px;
		inset-inline-end: 8px;
		margin-bottom: 1rem;
		// Above the title panel (z-index 4): the button sits in the title's
		// corner, and with the lower z-index the title's opaque background
		// painted over most of the round follow button, leaving only a
		// crescent of its border visible.
		z-index: 5;
	}
}
</style>
