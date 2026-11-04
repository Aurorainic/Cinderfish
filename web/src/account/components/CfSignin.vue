<template>
	<form class="cf-signin _monolithic_" autocomplete="on" @submit.prevent="onSubmit">
		<div class="auth _section _formRoot">
			<div
				v-if="withAvatar"
				class="avatar"
				:style="avatarStyle"
				:class="{ 'is-empty': avatarUrl == null }"
			>
				<i v-if="avatarUrl == null" class="ph-user ph-bold ph-2x"></i>
			</div>

			<div v-if="message" class="info">{{ message }}</div>
			<div v-if="errorText" class="info error" role="alert">{{ errorText }}</div>

			<div class="field">
				<label class="label" for="cf-signin-username">{{ t("username") }}</label>
				<div class="input">
					<span class="prefix">@</span>
					<input
						id="cf-signin-username"
						v-model="username"
						type="text"
						name="username"
						pattern="^[a-zA-Z0-9_]+$"
						autocomplete="username"
						:spellcheck="false"
						:disabled="signing"
						autofocus
						required
					/>
					<span class="suffix">@{{ host }}</span>
				</div>
			</div>

			<div class="field">
				<label class="label" for="cf-signin-password">{{ t("password") }}</label>
				<div class="input">
					<span class="prefix"><i class="ph-lock ph-bold ph-lg"></i></span>
					<input
						id="cf-signin-password"
						v-model="password"
						type="password"
						name="password"
						autocomplete="current-password"
						:disabled="signing"
						required
					/>
				</div>
			</div>

			<button
				class="_button primary submit"
				type="submit"
				:disabled="signing || username === '' || password === ''"
			>
				{{ signing ? t("loggingIn") : t("login") }}
			</button>
		</div>
	</form>
</template>

<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from "vue";
import { useLocale } from "../../i18n";
import { accountApi, AccountApiError, classifySigninFailure } from "../transport";
import { signIn } from "../session";
import type { MeDetailed, UserDetailed } from "../types";

const { t } = useLocale();

withDefaults(
	defineProps<{
		/** Firefish's avatar bubble above the form. */
		withAvatar?: boolean;
		/** Firefish compatibility; the port always logs in on success. */
		autoSet?: boolean;
		/** Optional callout rendered above the fields. */
		message?: string;
	}>(),
	{
		withAvatar: true,
		autoSet: true,
		message: "",
	},
);

const emit = defineEmits<{
	login: [account: MeDetailed];
}>();

const username = ref("");
const password = ref("");
const signing = ref(false);
const errorText = ref<string | null>(null);
const user = ref<UserDetailed | null>(null);

const host = typeof window === "undefined" ? "localhost" : window.location.host;
const avatarUrl = computed(() => user.value?.avatarUrl ?? null);
const avatarStyle = computed(() =>
	avatarUrl.value != null
		? { backgroundImage: `url("${avatarUrl.value}")` }
		: undefined,
);

let lookupTimer: number | undefined;

// Firefish looks the username up on every change to show the avatar and to
// decide whether to ask for a TOTP code. Cinderfish has no TOTP fields
// (`UserDetailed` never returns `twoFactorEnabled`), so only the avatar is
// looked up, debounced.
watch(username, (value) => {
	if (typeof window !== "undefined" && lookupTimer !== undefined) {
		window.clearTimeout(lookupTimer);
		lookupTimer = undefined;
	}
	if (value === "") {
		user.value = null;
		return;
	}
	if (typeof window === "undefined") return;
	lookupTimer = window.setTimeout(() => {
		accountApi<UserDetailed>("users/show", { username: value })
			.then((result) => {
				user.value = result;
			})
			.catch(() => {
				user.value = null;
			});
	}, 300);
});

onUnmounted(() => {
	if (typeof window !== "undefined" && lookupTimer !== undefined) {
		window.clearTimeout(lookupTimer);
	}
});

function describeFailure(error: unknown): string {
	switch (classifySigninFailure(error)) {
		case "no-such-user":
			return t("noSuchUser");
		case "incorrect-password":
			return t("incorrectPassword");
		case "suspended":
			return `${t("yourAccountSuspendedTitle")} — ${t("yourAccountSuspendedDescription")}`;
		case "rate-limited":
			return t("rateLimitExceeded");
		default:
			return error instanceof AccountApiError
				? `${t("loginFailed")}: ${error.message}`
				: t("somethingHappened");
	}
}

async function onSubmit(): Promise<void> {
	if (signing.value) return;
	signing.value = true;
	errorText.value = null;
	try {
		const account = await signIn(username.value, password.value);
		emit("login", account.account);
	} catch (error) {
		errorText.value = describeFailure(error);
	} finally {
		signing.value = false;
	}
}
</script>

<style lang="scss" scoped>
.cf-signin {
	.auth {
		display: flex;
		flex-direction: column;
	}

	.avatar {
		align-self: center;
		width: 74px;
		height: 74px;
		margin-bottom: 1.5em;
		border-radius: 100%;
		background-position: center;
		background-size: cover;
		box-shadow: 0 4px 16px var(--shadow);

		&.is-empty {
			display: grid;
			place-items: center;
			background: var(--panelHighlight);
			color: var(--fgTransparentWeak);
		}
	}

	.info {
		margin-bottom: 1em;
		padding: 12px 16px;
		border-radius: var(--radius);
		background: var(--infoBg);
		color: var(--infoFg);
		font-size: 0.95em;

		&.error {
			background: var(--infoWarnBg);
			color: var(--infoWarnFg);
		}
	}

	.field {
		margin: 1.5em 0;

		> .label {
			display: block;
			padding: 0 0 8px;
			font-size: 0.85em;
			user-select: none;
		}

		> .input {
			position: relative;
			display: flex;
			align-items: center;
			height: 39px;
			background: var(--panel);
			border: solid 1px var(--panel);
			border-radius: 6px;
			transition: border-color 0.1s ease-out;

			&:hover {
				border-color: var(--inputBorderHover);
			}

			&:focus-within {
				border-color: var(--accent);
			}

			> input {
				flex: 1;
				min-width: 0;
				height: 100%;
				padding: 0 12px;
				border: none;
				outline: none;
				background: none;
				color: var(--fg);
				font: inherit;
			}

			> .prefix,
			> .suffix {
				padding: 0 8px;
				color: var(--fgTransparentWeak);
				white-space: nowrap;
				user-select: none;
			}

			> .prefix {
				padding-left: 12px;
			}

			> .suffix {
				padding-right: 12px;
			}
		}
	}

	.submit {
		width: 100%;
		padding: 10px 16px;
		border-radius: 6px;
		background: var(--accent);
		color: var(--fgOnAccent);
		font-weight: bold;
		transition: background 0.1s ease-out;

		&:not(:disabled):hover {
			background: var(--accentDarken);
		}

		&:disabled {
			opacity: 0.6;
			cursor: not-allowed;
		}
	}
}
</style>
