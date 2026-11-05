<!-- Ported from firefish@b425f61:packages/client/src/ui/visitor/b.vue (AGPL-3.0-only) -->
<template>
	<div class="mk-app">
		<header class="header">
			<CfA to="/" class="brand">
				<span class="mark">C</span>
				<span class="wordmark">Cinderfish</span>
			</CfA>
			<nav class="links">
				<CfA to="/explore" active-class="active">
					<i class="ph-compass ph-bold ph-lg"></i>
					<span>{{ t("explore") }}</span>
				</CfA>
				<CfA to="/about" active-class="active">
					<i class="ph-info ph-bold ph-lg"></i>
					<span>{{ t("about") }}</span>
				</CfA>
				<button class="login _button" @click="signInShowing = true">
					<i class="ph-sign-in ph-bold ph-lg"></i>
					<span>{{ t("login") }}</span>
				</button>
			</nav>
		</header>

		<main class="contents">
			<RouterView />
		</main>

		<footer class="powered-by">
			<b><CfA to="/">{{ host }}</CfA></b>
			<small>Powered by Cinderfish</small>
		</footer>

		<transition name="signin">
			<div
				v-if="signInShowing"
				class="signin-back _modalBg"
				@click.self="signInShowing = false"
			>
				<div class="signin _panel">
					<button
						class="close _buttonIcon"
						:aria-label="t('close')"
						@click="signInShowing = false"
					>
						<i class="ph-x ph-bold ph-lg"></i>
					</button>
					<CfSignin @login="onSignedIn" />
				</div>
			</div>
		</transition>
	</div>
</template>

<script setup lang="ts">
// Firefish's visitor layout (design B) is a side kanban of recent notes, a
// header with the instance name and a footer, with the sign-in form behind a
// header button. Cinderfish has no kanban component and no public timeline
// widget, so the kanban is dropped and the page body is the routed page.
//
// The sign-in form is the account layer's real `CfSignin` (POST /api/signin);
// a successful login swaps the shell to universal because `main.ts` reads
// `me` reactively.
import { ref } from "vue";
import { useRouter } from "vue-router";
import { useLocale } from "@/i18n";
import CfA from "@/components/ui/CfA.vue";
import { CfSignin } from "@/account/components";

const router = useRouter();
const { t } = useLocale();

const signInShowing = ref(false);
const host =
	typeof window === "undefined" ? "localhost" : window.location.host;

function onSignedIn(): void {
	signInShowing.value = false;
	void router.push("/");
}
</script>

<style lang="scss" scoped>
.mk-app {
	display: flex;
	flex-direction: column;
	min-block-size: 100vh;
	box-sizing: border-box;
	background: var(--bg);
	color: var(--fg);

	> .header {
		position: sticky;
		inset-block-start: 0;
		z-index: 10;
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 16px;
		inline-size: 100%;
		max-inline-size: 900px;
		margin-inline: auto;
		padding: 16px var(--margin);
		box-sizing: border-box;
		background: var(--bg);

		> .brand {
			display: flex;
			align-items: center;
			gap: 10px;
			font-weight: 800;
			font-size: 1.1em;
			color: var(--fg);

			> .mark {
				display: inline-flex;
				align-items: center;
				justify-content: center;
				inline-size: 30px;
				block-size: 30px;
				border-radius: 10px;
				color: var(--fgOnAccent);
				background: linear-gradient(
					135deg,
					var(--buttonGradateA),
					var(--buttonGradateB)
				);
			}
		}

		> .links {
			display: flex;
			align-items: center;
			gap: 8px;
			font-size: 0.9em;

			> a,
			> .login {
				display: inline-flex;
				align-items: center;
				gap: 6px;
				padding: 8px 12px;
				color: var(--navFg);
				border-radius: 999px;

				&:hover {
					color: var(--navHoverFg);
					background: var(--buttonBg);
				}

				&.active {
					color: var(--accent);
					background: var(--accentedBg);
				}
			}

			> .login {
				color: var(--fgOnAccent);
				background: var(--accent);

				&:hover {
					color: var(--fgOnAccent);
					background: var(--accentLighten);
				}
			}
		}
	}

	> .contents {
		flex: 1;
		inline-size: 100%;
		max-inline-size: 900px;
		margin-inline: auto;
		padding: 0 var(--margin) var(--margin);
		box-sizing: border-box;
	}

	> .powered-by {
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 4px;
		padding: 24px;
		color: var(--fgTransparentWeak);
		font-size: 0.8em;

		> b > a {
			color: var(--fg);
		}
	}

	.signin-back {
		position: fixed;
		z-index: 1000;
		inset: 0;
		display: flex;
		align-items: center;
		justify-content: center;
		padding: var(--margin);
	}

	.signin {
		position: relative;
		inline-size: 100%;
		max-inline-size: 380px;
		padding: 32px;
		box-sizing: border-box;
		border-radius: var(--radius);
		background: var(--panel);

		> .close {
			position: absolute;
			inset-block-start: 8px;
			inset-inline-end: 8px;
		}
	}
}

.signin-enter-active,
.signin-leave-active {
	opacity: 1;
	transition: opacity 200ms ease;
}

.signin-enter-from,
.signin-leave-to {
	opacity: 0;
}
</style>
