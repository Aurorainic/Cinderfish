<template>
	<div class="cf-visitor">
		<header class="hero">
			<h1 class="title">Cinderfish</h1>
			<p class="lead">TODO: port ui/visitor/a.vue · {{ t("introMisskey") }}</p>
			<div class="actions">
				<RouterLink to="/timeline" class="link primary">
					{{ t("timeline") }}
				</RouterLink>
				<RouterLink to="/explore" class="link">
					{{ t("explore") }}
				</RouterLink>
				<RouterLink to="/about" class="link">
					{{ t("about") }}
				</RouterLink>
			</div>
		</header>
		<main class="content">
			<router-view />
		</main>
	</div>
</template>

<script setup lang="ts">
import { translate as t } from "@/i18n";
</script>

<style scoped>
.cf-visitor {
	min-block-size: 100dvh;
	background: var(--bg);
	color: var(--fg);

	> .hero {
		max-inline-size: 720px;
		margin-inline: auto;
		padding: 64px 24px 32px;
		text-align: center;

		> .title {
			margin: 0;
			font-size: 2.2em;
			letter-spacing: 0.02em;
		}

		> .lead {
			color: var(--fgTransparentWeak);
		}

		> .actions {
			display: flex;
			justify-content: center;
			gap: 8px;
			flex-wrap: wrap;
			margin-block-start: 16px;
		}
	}

	> .content {
		max-inline-size: 720px;
		margin-inline: auto;
		padding-block-end: 64px;
	}
}

.link {
	display: inline-block;
	padding: 8px 16px;
	color: var(--fg);
	background: var(--buttonBg);
	border-radius: 5px;
	text-decoration: none;

	&:hover {
		background: var(--buttonHoverBg);
	}

	&.primary {
		color: var(--fgOnAccent);
		background: var(--accent);
	}
}
</style>
