<!-- Shared body for admin pages whose upstream feature has no backend here. -->
<template>
	<div class="cf-admin-unavailable">
		<CfInfo warn class="_formBlock">
			<strong>{{ title }}</strong>
			<p class="summary">{{ summary }}</p>
		</CfInfo>

		<section v-if="available.length > 0" class="_formBlock">
			<h2 class="sectionTitle">这里已经有</h2>
			<ul class="list">
				<li v-for="item in available" :key="item.label" class="item ok">
					<i class="ph-check-circle ph-bold"></i>
					<div class="body">
						<span class="label">{{ item.label }}</span>
						<span v-if="item.note != null" class="note">{{ item.note }}</span>
					</div>
					<RouterLink v-if="item.to != null" class="link" :to="item.to">
						打开
					</RouterLink>
				</li>
			</ul>
		</section>

		<section v-if="missing.length > 0" class="_formBlock">
			<h2 class="sectionTitle">上游有、这里没有</h2>
			<ul class="list">
				<li v-for="item in missing" :key="item" class="item">
					<i class="ph-x-circle ph-bold"></i>
					<div class="body">
						<span class="label">{{ item }}</span>
					</div>
				</li>
			</ul>
			<CfInfo class="note">
				这些功能需要对应的服务端支持（端点、表结构或外部服务），当前没有实现。
				与其放置点了没反应的表单，这个页面直接说明现状。
			</CfInfo>
		</section>
	</div>
</template>

<script lang="ts" setup>
// Several administration pages in Firefish (experiments, integrations,
// promotions, proxy accounts, security tuning, other settings) describe
// features this backend does not have. Rendering their forms would create
// controls that silently do nothing, so those pages share this body instead:
// it states what exists, what does not, and links to the pages that do work.
export interface AvailableItem {
	label: string;
	note?: string;
	to?: string;
}

withDefaults(
	defineProps<{
		title: string;
		summary: string;
		available?: AvailableItem[];
		missing?: string[];
	}>(),
	{ available: () => [], missing: () => [] },
);

defineOptions({ name: "CfAdminUnavailable" });
</script>

<style lang="scss" scoped>
.cf-admin-unavailable {
	.summary {
		margin: 8px 0 0;
		font-size: 0.92em;
		line-height: 1.7;
	}

	.list {
		margin: 0;
		padding: 0;
		list-style: none;
		display: flex;
		flex-direction: column;
		gap: 8px;
	}

	.item {
		display: flex;
		align-items: flex-start;
		gap: 8px;
		padding: 10px 12px;
		border-radius: var(--radius);
		background: var(--panel);

		> i {
			margin-top: 2px;
			opacity: 0.6;
		}

		&.ok > i {
			color: var(--success, #3ba55d);
			opacity: 1;
		}

		.body {
			flex: 1;
			min-width: 0;
			display: flex;
			flex-direction: column;
			gap: 2px;

			.label {
				font-size: 0.95em;
			}

			.note {
				font-size: 0.82em;
				color: var(--fgTransparentWeak);
			}
		}

		.link {
			flex-shrink: 0;
			font-size: 0.85em;
		}
	}

	.note {
		margin-top: 12px;
	}
}
</style>
