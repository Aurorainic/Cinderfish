<!-- Ported from firefish@b425f61:packages/client/src/components/MkUserCardMini.vue (AGPL-3.0-only) -->
<template>
	<MkA
		class="user-card-mini"
		:class="[{ yellow: silenced, red: suspended }]"
		:to="userPage(user)"
	>
		<MkAvatar
			class="avatar"
			:user="user"
			:disable-link="true"
			:show-indicator="true"
		/>
		<div class="body">
			<span class="name"><MkUserName class="name" :user="user" /></span>
			<span class="sub"
				><span class="acct _monospace">@{{ acct(user) }}</span></span
			>
		</div>
	</MkA>
</template>

<script lang="ts" setup>
// Degradations versus Firefish (contract §2):
//   * the MkMiniChart sparkline needs `charts/user/notes`, which the backend
//     does not serve, so no chart is drawn and calls have no chart numbers;
//   * `isSilenced` / `isSuspended` only exist on UserDetailed, so the hatch
//     patterns are derived defensively and stay off for a bare UserLite.
import { computed } from "vue";
import type { UserLite } from "@/api/types";
import { acct, userPage } from "@/filters/user";

const props = defineProps<{
	user: UserLite;
}>();

type MaybeModerated = { isSilenced?: boolean; isSuspended?: boolean };

const silenced = computed<boolean>(
	() => (props.user as MaybeModerated).isSilenced === true,
);
const suspended = computed<boolean>(
	() => (props.user as MaybeModerated).isSuspended === true,
);
</script>

<style lang="scss" scoped>
.user-card-mini {
	display: flex;
	align-items: center;
	padding: 16px;
	background: var(--panel);
	border-radius: 8px;
	transition: background 0.2s;

	> .avatar {
		display: block;
		width: 34px;
		height: 34px;
		margin-right: 12px;
	}

	> .body {
		flex: 1;
		overflow: hidden;
		font-size: 0.9em;
		color: var(--fg);
		padding-right: 8px;

		> .name {
			display: block;
			width: 100%;
			white-space: nowrap;
			overflow: hidden;
			text-overflow: ellipsis;
			line-height: 18px;
		}

		> .sub {
			display: block;
			width: 100%;
			font-size: 95%;
			opacity: 0.7;
			line-height: 16px;
			white-space: nowrap;
			overflow: hidden;
			text-overflow: ellipsis;
		}
	}

	&:hover,
	&:focus {
		background: var(--panelHighlight);
	}

	&.yellow {
		--c: rgb(255 196 0 / 15%);
		background-image: linear-gradient(
			45deg,
			var(--c) 16.67%,
			transparent 16.67%,
			transparent 50%,
			var(--c) 50%,
			var(--c) 66.67%,
			transparent 66.67%,
			transparent 100%
		);
		background-size: 16px 16px;
	}

	&.red {
		--c: rgb(255 0 0 / 15%);
		background-image: linear-gradient(
			45deg,
			var(--c) 16.67%,
			transparent 16.67%,
			transparent 50%,
			var(--c) 50%,
			var(--c) 66.67%,
			transparent 66.67%,
			transparent 100%
		);
		background-size: 16px 16px;
	}
}
</style>
