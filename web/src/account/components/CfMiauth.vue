<template>
	<div class="cf-miauth">
		<div v-if="!signedIn" class="signin">
			<CfSignin />
		</div>

		<div v-else-if="state === 'waiting'" class="section waiting">
			<span class="spinner" aria-hidden="true"></span>
		</div>

		<div v-else-if="state === 'denied'" class="section">
			<p>{{ t("_auth.denied") }}</p>
		</div>

		<div v-else-if="state === 'accepted'" class="section">
			<p v-if="callback != null">{{ t("_auth.callback") }}…</p>
			<p v-else>{{ t("_auth.pleaseGoBack") }}</p>
		</div>

		<div v-else class="section">
			<h2 class="title">
				<template v-if="name != null">{{ t("_auth.shareAccess", { name: name ?? "" }) }}</template>
				<template v-else>{{ t("_auth.shareAccessAsk") }}</template>
			</h2>

			<p class="description">{{ t("_auth.permissionAsk") }}</p>

			<div class="permissions">
				<div v-if="permissions.length === 32" class="permission all">
					<i class="ph-shield-warning ph-bold ph-xl"></i>
					{{ t("_auth.allPermissions") }}
				</div>
				<div v-for="permission in permissions" :key="permission" class="permission">
					<i :class="`ph-${iconFor(permission)}`" class="ph-bold ph-xl"></i>
					{{ t(`_permissions.${permission}`) }}
				</div>
			</div>

			<div v-if="errorText" class="info error" role="alert">{{ errorText }}</div>

			<div class="footer">
				<button class="_button border" type="button" @click="deny">
					{{ t("cancel") }}
				</button>
				<button class="_button primary" type="button" @click="accept">
					{{ t("accept") }}
				</button>
			</div>
		</div>
	</div>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { useLocale } from "../../i18n";
import CfSignin from "./CfSignin.vue";
import { acceptMiAuth, parsePermission, permissionIcon } from "../miauth";
import { signedIn } from "../me";
import { AccountApiError } from "../transport";

const props = withDefaults(
	defineProps<{
		session: string;
		callback?: string;
		name?: string;
		icon?: string;
		/** Comma-separated permission list, as in Firefish's query params. */
		permission?: string;
	}>(),
	{
		callback: undefined,
		name: undefined,
		icon: undefined,
		permission: "",
	},
);

const emit = defineEmits<{
	accepted: [token: string];
	denied: [];
}>();

const { t } = useLocale();

type State = "idle" | "waiting" | "accepted" | "denied";

const state = ref<State>("idle");
const errorText = ref<string | null>(null);

const permissions = computed(() => parsePermission(props.permission));

function iconFor(permission: string): string {
	return permissionIcon(permission);
}

function deny(): void {
	state.value = "denied";
	emit("denied");
}

async function accept(): Promise<void> {
	if (state.value === "waiting") return;
	state.value = "waiting";
	errorText.value = null;
	try {
		const result = await acceptMiAuth({
			session: props.session,
			name: props.name,
			icon: props.icon,
			permission: props.permission,
			callback: props.callback,
		});
		state.value = "accepted";
		emit("accepted", result.token);
		if (result.callbackUrl != null) {
			// Firefish navigates immediately; the "accepted" state is only seen
			// when there is no callback.
			window.location.assign(result.callbackUrl);
		}
	} catch (error) {
		state.value = "idle";
		errorText.value =
			error instanceof AccountApiError ? error.message : t("somethingHappened");
	}
}
</script>

<style lang="scss" scoped>
.cf-miauth {
	max-width: 800px;
	margin: 0 auto;

	.section {
		padding: 32px;
		border-radius: var(--radius);
		background: var(--panel);

		@media (max-width: 500px) {
			padding: 16px;
		}
	}

	.title {
		margin: 0 0 1em;
		font-size: 1.1em;
	}

	.description {
		margin: 0 0 1.5em;
		color: var(--fgTransparentWeak);
	}

	.permissions {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
		gap: 8px;
		margin-bottom: 1.5em;
	}

	.permission {
		display: flex;
		align-items: center;
		padding: 10px 12px;
		border-radius: 8px;
		background: var(--buttonBg);
		font-size: 0.95em;

		> i {
			margin-right: 0.5rem;
		}

		&.all {
			background: var(--error);
			color: var(--fgOnAccent);
		}
	}

	.footer {
		display: flex;
		justify-content: flex-end;
		gap: 8px;

		.border {
			padding: 8px 16px;
			border: solid 1px var(--divider);
			border-radius: 6px;
		}

		.primary {
			padding: 8px 16px;
			border-radius: 6px;
			background: var(--accent);
			color: var(--fgOnAccent);
			font-weight: bold;
		}

		button:disabled {
			opacity: 0.6;
			cursor: not-allowed;
		}
	}

	.info.error {
		margin-bottom: 1em;
		padding: 12px 16px;
		border-radius: var(--radius);
		background: var(--infoWarnBg);
		color: var(--infoWarnFg);
	}

	.waiting {
		display: grid;
		place-items: center;
		min-height: 160px;
	}

	.spinner {
		width: 32px;
		height: 32px;
		border: solid 3px var(--divider);
		border-top-color: var(--accent);
		border-radius: 100%;
		animation: cf-miauth-spin 0.9s linear infinite;
	}

	@keyframes cf-miauth-spin {
		to {
			transform: rotate(360deg);
		}
	}
}
</style>
