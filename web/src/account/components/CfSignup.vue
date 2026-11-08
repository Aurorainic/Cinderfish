<template>
	<form class="cf-signup _formRoot" autocomplete="new-password" @submit.prevent="onSubmit">
		<div v-if="meta?.disableRegistration" class="notice">
			{{ t("signupsDisabled") }}
		</div>

		<div v-if="meta?.disableRegistration" class="field">
			<label class="label" for="cf-signup-invite">{{ t("invitationCode") }}</label>
			<div class="input">
				<span class="prefix"><i class="ph-key ph-bold ph-lg"></i></span>
				<input
					id="cf-signup-invite"
					v-model="invitationCode"
					type="text"
					:spellcheck="false"
					autocomplete="off"
				/>
			</div>
		</div>

		<div class="field">
			<label class="label" for="cf-signup-username">
				{{ t("username") }}
				<span class="caption">{{ t("usernameInfo") }}</span>
			</label>
			<div class="input">
				<span class="prefix">@</span>
				<input
					id="cf-signup-username"
					v-model="username"
					type="text"
					pattern="^[a-zA-Z0-9_]{1,20}$"
					:spellcheck="false"
					autocomplete="username"
					maxlength="20"
					required
				/>
				<span class="suffix">@{{ host }}</span>
			</div>
			<p v-if="usernameState != null" class="caption" :class="usernameState">
				<template v-if="usernameState === 'wait'">{{ t("checking") }}</template>
				<template v-else-if="usernameState === 'ok'">{{ t("available") }}</template>
				<template v-else-if="usernameState === 'unavailable'">{{
					t("unavailable")
				}}</template>
				<template v-else-if="usernameState === 'invalid-format'">{{
					t("usernameInvalidFormat")
				}}</template>
				<template v-else-if="usernameState === 'min-range'">{{
					t("tooShort")
				}}</template>
				<template v-else-if="usernameState === 'max-range'">{{
					t("tooLong")
				}}</template>
				<template v-else>{{ t("error") }}</template>
			</p>
		</div>

		<div class="field">
			<label class="label" for="cf-signup-password">{{ t("password") }}</label>
			<div class="input">
				<span class="prefix"><i class="ph-lock ph-bold ph-lg"></i></span>
				<input
					id="cf-signup-password"
					v-model="password"
					type="password"
					autocomplete="new-password"
					required
				/>
			</div>
			<p v-if="passwordStrength !== ''" class="caption" :class="`strength-${passwordStrength}`">
				{{ t(`${passwordStrength}Password`) }}
			</p>
		</div>

		<div class="field">
			<label class="label" for="cf-signup-retype">{{ t("password") }}</label>
			<div class="input">
				<span class="prefix"><i class="ph-lock ph-bold ph-lg"></i></span>
				<input
					id="cf-signup-retype"
					v-model="retypedPassword"
					type="password"
					autocomplete="new-password"
					required
				/>
			</div>
			<p v-if="passwordRetypeState != null" class="caption" :class="passwordRetypeState">
				{{ passwordRetypeState === "match" ? t("passwordMatched") : t("passwordNotMatched") }}
			</p>
		</div>

		<div v-if="captchaProvider !== null" class="field captchaField">
			<label class="label">人机验证</label>
			<CfCaptcha
				ref="captchaRef"
				v-model="captchaToken"
				:provider="captchaProvider"
				:site-key="captchaSiteKey"
			/>
		</div>

		<div v-if="errorText" class="info error" role="alert">{{ errorText }}</div>

		<button class="_button primary submit" type="submit" :disabled="shouldDisableSubmitting">
			{{ submitting ? `${t("signup")}…` : t("signup") }}
		</button>

		<!-- The instance's terms and rules are part of signup, as in Firefish:
		     the links only appear when the administrator configured them. -->
		<p v-if="hasTerms" class="terms">
			注册即表示你同意
			<a
				v-if="meta?.tosUrl"
				:href="meta.tosUrl"
				target="_blank"
				rel="nofollow noopener"
				>本实例的服务条款</a
			>
			<template v-if="meta?.tosUrl && (meta?.serverRules ?? []).length > 0"
				>与</template
			>
			<RouterLink
				v-if="(meta?.serverRules ?? []).length > 0"
				to="/about"
				>实例规则</RouterLink
			>。
		</p>
	</form>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from "vue";
import CfCaptcha from "../../components/CfCaptcha.vue";
import { useLocale } from "../../i18n";
import { fetchMeta } from "../meta";
import { accountApi, AccountApiError } from "../transport";
import { signUp } from "../session";
import type { MeDetailed, MetaLite, UsernameAvailableResponse } from "../types";

const { t } = useLocale();

const emit = defineEmits<{
	signup: [account: MeDetailed];
}>();

const host = typeof window === "undefined" ? "localhost" : window.location.host;

const meta = ref<MetaLite | null>(null);
const hasTerms = computed(
	() => (meta.value?.tosUrl ?? "") !== "" || (meta.value?.serverRules ?? []).length > 0,
);
const username = ref("");
const password = ref("");
const retypedPassword = ref("");
const invitationCode = ref("");
const submitting = ref(false);
const errorText = ref<string | null>(null);

// Captcha: the instance enables at most one provider (server-side order is
// turnstile, recaptcha, hcaptcha; the public meta sets exactly one flag).
type CaptchaProvider = "hcaptcha" | "recaptcha" | "turnstile";
const captchaProvider = computed<CaptchaProvider | null>(() => {
	const current = meta.value;
	if (current == null) return null;
	if (current.enableTurnstile) return "turnstile";
	if (current.enableRecaptcha) return "recaptcha";
	if (current.enableHcaptcha) return "hcaptcha";
	return null;
});
const captchaSiteKey = computed<string>(() => {
	const current = meta.value;
	if (current == null) return "";
	if (current.enableTurnstile) return current.turnstileSiteKey;
	if (current.enableRecaptcha) return current.recaptchaSiteKey;
	if (current.enableHcaptcha) return current.hcaptchaSiteKey;
	return "";
});
const captchaToken = ref("");
const captchaRef = ref<InstanceType<typeof CfCaptcha> | null>(null);

type UsernameState =
	| null
	| "wait"
	| "ok"
	| "unavailable"
	| "error"
	| "invalid-format"
	| "min-range"
	| "max-range";

const usernameState = ref<UsernameState>(null);
const passwordStrength = ref<"" | "low" | "medium" | "high">("");
const passwordRetypeState = ref<null | "match" | "not-match">(null);

const shouldDisableSubmitting = computed(
	() =>
		submitting.value ||
		passwordRetypeState.value === "not-match" ||
		password.value === "" ||
		retypedPassword.value === "" ||
		usernameState.value === "unavailable" ||
		usernameState.value === "invalid-format" ||
		usernameState.value === "min-range" ||
		usernameState.value === "max-range" ||
		(captchaProvider.value !== null && captchaToken.value === ""),
);

onMounted(() => {
	fetchMeta()
		.then((result) => {
			meta.value = result;
		})
		.catch(() => {
			// The form still works against an open instance; signup reports the
			// real error if registration turns out to be disabled.
		});
});

let usernameTimer: number | undefined;

function clearUsernameTimer(): void {
	if (typeof window !== "undefined" && usernameTimer !== undefined) {
		window.clearTimeout(usernameTimer);
		usernameTimer = undefined;
	}
}

watch(username, (value) => {
	clearUsernameTimer();
	if (value === "") {
		usernameState.value = null;
		return;
	}

	if (!/^[a-zA-Z0-9_]+$/.test(value)) {
		usernameState.value = "invalid-format";
		return;
	}
	if (value.length < 1) {
		usernameState.value = "min-range";
		return;
	}
	if (value.length > 20) {
		usernameState.value = "max-range";
		return;
	}

	usernameState.value = "wait";
	if (typeof window === "undefined") return;
	usernameTimer = window.setTimeout(() => {
		accountApi<UsernameAvailableResponse>("username/available", {
			username: value,
		})
			.then((result) => {
				usernameState.value = result.available ? "ok" : "unavailable";
			})
			.catch(() => {
				usernameState.value = "error";
			});
	}, 300);
});

onUnmounted(clearUsernameTimer);

function scorePassword(value: string): "" | "low" | "medium" | "high" {
	if (value === "") return "";
	let score = 0;
	if (value.length >= 8) score += 1;
	if (value.length >= 12) score += 1;
	if (/[a-z]/.test(value) && /[A-Z]/.test(value)) score += 1;
	if (/\d/.test(value)) score += 1;
	if (/[^a-zA-Z0-9]/.test(value)) score += 1;
	return score >= 4 ? "high" : score >= 2 ? "medium" : "low";
}

watch(password, (value) => {
	passwordStrength.value = scorePassword(value);
	passwordRetypeState.value =
		retypedPassword.value === ""
			? null
			: value === retypedPassword.value
				? "match"
				: "not-match";
});

watch(retypedPassword, (value) => {
	passwordRetypeState.value =
		value === "" ? null : password.value === value ? "match" : "not-match";
});

function describeFailure(error: unknown): string {
	if (!(error instanceof AccountApiError)) return t("somethingHappened");
	switch (error.code) {
		case "REGISTRATION_DISABLED":
			return t("signupsDisabled");
		case "DUPLICATED_USERNAME":
			return `${t("username")}: ${t("unavailable")} (${error.message})`;
		case "INVALID_USERNAME":
			return t("usernameInvalidFormat");
		case "INVALID_INVITE":
			return `${t("invitationCode")}: ${error.message}`;
		case "INVALID_PASSWORD":
			return error.message;
		default:
			return `${t("error")}: ${error.message}`;
	}
}

async function onSubmit(): Promise<void> {
	// The native form validation runs first; this guard covers programmatic
	// submits and the async username check.
	if (submitting.value || shouldDisableSubmitting.value) return;
	submitting.value = true;
	errorText.value = null;
	const captcha =
		captchaProvider.value === null
			? undefined
			: captchaProvider.value === "hcaptcha"
				? { hcaptcha: captchaToken.value }
				: captchaProvider.value === "recaptcha"
					? { recaptcha: captchaToken.value }
					: { turnstile: captchaToken.value };
	try {
		const account = await signUp(username.value, password.value, {
			...(invitationCode.value !== "" ? { invitationCode: invitationCode.value } : {}),
			...(captcha !== undefined ? { captcha } : {}),
		});
		emit("signup", account.account);
	} catch (error) {
		errorText.value = describeFailure(error);
	} finally {
		submitting.value = false;
		// A captcha token is single use; ask for a fresh one on the next
		// attempt whether this one succeeded or not.
		captchaRef.value?.reset();
		captchaToken.value = "";
	}
}
</script>

<style lang="scss" scoped>
.cf-signup {
	.notice {
		margin-bottom: 1.5em;
		padding: 12px 16px;
		border-radius: var(--radius);
		background: var(--infoBg);
		color: var(--infoFg);
		font-size: 0.95em;
	}

	.field {
		margin: 1.5em 0;

		> .label {
			display: flex;
			flex-direction: column;
			gap: 4px;
			padding: 0 0 8px;
			font-size: 0.85em;
			user-select: none;
		}

		> .label .caption {
			color: var(--fgTransparentWeak);
			font-size: 0.9em;
			font-weight: normal;
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

		> .caption {
			margin: 6px 0 0;
			font-size: 0.85em;

			&.ok,
			&.match {
				color: var(--success);
			}

			&.unavailable,
			&.invalid-format,
			&.min-range,
			&.max-range,
			&.error,
			&.not-match,
			&.strength-low {
				color: var(--error);
			}

			&.wait {
				color: var(--fgTransparentWeak);
			}

			&.strength-medium {
				color: var(--warn);
			}

			&.strength-high {
				color: var(--success);
			}
		}
	}

	.info {
		margin-bottom: 1em;
		padding: 12px 16px;
		border-radius: var(--radius);

		&.error {
			background: var(--infoWarnBg);
			color: var(--infoWarnFg);
		}
	}

	.terms {
		margin: 12px 0 0;
		font-size: 0.86em;
		line-height: 1.5;
		opacity: 0.8;
		text-align: center;

		> a {
			color: var(--accent);
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
