<!-- SPDX-License-Identifier: AGPL-3.0-only -->
<template>
	<div class="admin-user-row" :class="{ suspended: user.isSuspended }">
		<MkAvatar class="avatar" :user="user" />
		<div class="body">
			<div class="head">
				<CfA class="name" :to="userPage(user)">
					<MkUserName :user="user" :nowrap="true" />
				</CfA>
				<span v-if="user.isAdmin" class="badge admin">
					{{ t("administrator") }}
				</span>
				<span v-if="user.isSuspended" class="badge suspended">已冻结</span>
			</div>
			<div class="sub">
				<span class="acct _monospace">@{{ acct(user) }}</span>
				<span class="sep" aria-hidden="true">·</span>
				<span class="created">
					{{ t("registeredDate") }}
					<MkTime :time="user.createdAt" mode="absolute" />
				</span>
			</div>
		</div>
		<div class="actions">
			<CfA class="profile" :to="userPage(user)">资料</CfA>
			<span v-if="!canModerate" class="self-hint">当前账号</span>
			<CfButton
				mini
				:disabled="busy || !canModerate"
				@click="emit('toggle', user)"
			>
				{{ busy ? "处理中…" : user.isSuspended ? t("unsuspend") : t("suspend") }}
			</CfButton>
		</div>
	</div>
</template>

<script setup lang="ts">
// New component for the admin user list: Firefish renders each account with
// `MkUserCardMini`, which has no suspend action and no registration date. The
// row keeps that card's shape (avatar, display name, @acct) and adds the two
// facts `admin/show-users` returns for moderators (`isAdmin`, `isSuspended`)
// plus the inline suspend toggle.
//
// Firefish shows `lastFetchedAt` in a tooltip; Cinderfish always reports null,
// so the row shows `createdAt` instead.
import type { UserDetailed } from "@/api/types";
import MkUserName from "@/components/global/MkUserName.vue";
import CfA from "@/components/ui/CfA.vue";
import CfButton from "@/components/ui/CfButton.vue";
import { acct, userPage } from "@/filters/user";
import { t } from "@/i18n";

withDefaults(
	defineProps<{
		user: UserDetailed;
		/** The toggle request for this row is in flight. */
		busy?: boolean;
		/** False for the signed-in administrator: the backend refuses self-suspend. */
		canModerate?: boolean;
	}>(),
	{
		busy: false,
		canModerate: true,
	},
);

const emit = defineEmits<{
	toggle: [user: UserDetailed];
}>();
</script>

<style lang="scss" scoped>
.admin-user-row {
	display: flex;
	align-items: center;
	gap: 12px;
	padding: 12px 16px;
	background: var(--panel);
	border-radius: var(--radius);

	&.suspended {
		border: solid 1px var(--error);
	}

	> .avatar {
		inline-size: 42px;
		block-size: 42px;
		flex-shrink: 0;
	}

	> .body {
		flex: 1;
		min-width: 0;

		> .head {
			display: flex;
			align-items: center;
			gap: 8px;
			flex-wrap: wrap;

			> .name {
				font-weight: bold;
				overflow: hidden;
				text-overflow: ellipsis;
				white-space: nowrap;
				max-width: 100%;
			}

			> .badge {
				font-size: 0.75em;
				padding: 1px 8px;
				border-radius: 999px;
				line-height: 1.5;
				white-space: nowrap;

				&.admin {
					background: var(--accentedBg);
					color: var(--accent);
				}

				&.suspended {
					color: var(--error);
					border: solid 1px currentColor;
				}
			}
		}

		> .sub {
			display: flex;
			flex-wrap: wrap;
			gap: 6px;
			font-size: 0.85em;
			color: var(--fgTransparentWeak);

			> .sep {
				opacity: 0.6;
			}
		}
	}

	> .actions {
		display: flex;
		align-items: center;
		gap: 8px;
		flex-shrink: 0;

		> .profile {
			font-size: 0.85em;
			color: var(--fgTransparentWeak);
		}

		> .self-hint {
			font-size: 0.8em;
			color: var(--fgTransparentWeak);
		}
	}

	@media (max-width: 480px) {
		flex-wrap: wrap;

		> .body {
			flex-basis: calc(100% - 54px);
		}

		> .actions {
			inline-size: 100%;
			justify-content: flex-end;
		}
	}
}
</style>
