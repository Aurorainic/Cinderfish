<!-- Ported from firefish@b425f61:packages/client/src/pages/admin/database.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :display-back-button="true" />
		</template>
		<MkSpacer :content-max="900" :margin-min="16" :margin-max="32">
			<div v-if="!admin" class="cf-admin-database">
				<CfInfo warn>
					需要管理员权限，无法查看数据库统计。
					<CfA to="/" class="_link">返回首页</CfA>
				</CfInfo>
			</div>
			<div v-else class="cf-admin-database">
				<div class="toolbar">
					<div class="summary">
						<span class="value">{{ formatSize(totalSize) }}</span>
						<span class="caption"
							>{{ tables.length }} 张表 · {{ indexes.length }} 个索引</span
						>
					</div>
					<CfButton :disabled="loading" @click="load">
						<i class="ph-arrows-clockwise ph-bold"></i>
						刷新
					</CfButton>
				</div>

				<CfLoading v-if="loading && tables.length === 0" />
				<MkError v-else-if="error !== null && tables.length === 0" @retry="load" />
				<template v-else>
					<section class="_formBlock">
						<h2 class="sectionTitle">表</h2>
						<ul class="tables">
							<li v-for="table in sortedTables" :key="table.name" class="table">
								<div class="row">
									<span class="name _monospace">{{ table.name }}</span>
									<span class="count">{{ formatNumber(table.count) }} 行</span>
									<span class="size">{{ formatSize(table.size) }}</span>
								</div>
								<div class="meter">
									<div
										class="val"
										:style="{ width: `${barWidth(table.size)}%` }"
									></div>
								</div>
							</li>
						</ul>
					</section>

					<section class="_formBlock">
						<h2 class="sectionTitle">索引（{{ indexes.length }}）</h2>
						<ul class="indexes">
							<li
								v-for="index in indexes"
								:key="`${index.schemaname}.${index.indexname}`"
								class="index"
							>
								<span class="indexName _monospace">{{ index.indexname }}</span>
								<span class="indexTable">on {{ index.tablename }}</span>
								<details>
									<summary>定义</summary>
									<code class="def">{{ index.indexdef }}</code>
								</details>
							</li>
						</ul>
					</section>
				</template>

				<CfInfo class="notice">
					行数是 PostgreSQL 规划器的估算（<code>pg_class.reltuples</code>），不是
					<code>COUNT(*)</code>；从未分析过的表显示 0。大小含索引与 TOAST。
				</CfInfo>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script setup lang="ts">
// Firefish's database page reads `admin/get-table-stats` and
// `admin/get-index-stats`. Both are implemented here with the same SQL shape;
// the page adds a size bar and a definition disclosure because the raw
// `pg_indexes` rows are long.
import { computed, onMounted, ref } from "vue";
import { ApiError, api } from "@/api/client";
import type { IndexStat, TableStat } from "@/api/types";
import { isAdmin } from "@/account";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";
import MkError from "@/components/global/MkError.vue";
import CfA from "@/components/ui/CfA.vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfLoading from "@/components/ui/CfLoading.vue";
import { t } from "@/i18n";
import { definePageMetadata } from "@/scripts/page-metadata";

defineOptions({ name: "CfPageAdminDatabase" });

const admin = computed<boolean>(() => isAdmin());

const stats = ref<Record<string, TableStat>>({});
const indexes = ref<IndexStat[]>([]);
const loading = ref(false);
const error = ref<string | null>(null);

const tables = computed(() =>
	Object.entries(stats.value).map(([name, stat]) => ({
		name,
		count: stat.count,
		size: stat.size,
	})),
);

const sortedTables = computed(() =>
	[...tables.value].sort((a, b) => b.size - a.size),
);

const totalSize = computed(() =>
	tables.value.reduce((sum, table) => sum + table.size, 0),
);

const maxSize = computed(() =>
	tables.value.reduce((max, table) => Math.max(max, table.size), 0),
);

function barWidth(size: number): number {
	return maxSize.value === 0 ? 0 : Math.max(1, (size / maxSize.value) * 100);
}

function formatSize(bytes: number): string {
	if (bytes < 1024) return `${bytes} B`;
	if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`;
	if (bytes < 1024 * 1024 * 1024) return `${(bytes / 1024 / 1024).toFixed(1)} MB`;
	return `${(bytes / 1024 / 1024 / 1024).toFixed(2)} GB`;
}

function formatNumber(value: number): string {
	return new Intl.NumberFormat("zh-CN").format(Math.round(value));
}

async function load(): Promise<void> {
	if (!admin.value) return;
	loading.value = true;
	error.value = null;
	try {
		const [tableStats, indexStats] = await Promise.all([
			api("admin/get-table-stats", {}),
			api("admin/get-index-stats", {}),
		]);
		stats.value = tableStats;
		indexes.value = indexStats;
	} catch (cause) {
		error.value = cause instanceof ApiError ? cause.message : String(cause);
		stats.value = {};
		indexes.value = [];
	} finally {
		loading.value = false;
	}
}

onMounted(() => {
	void load();
});

definePageMetadata({
	title: t("database"),
	icon: "ph-database ph-bold ph-lg",
});
</script>

<style lang="scss" scoped>
.toolbar {
	display: flex;
	align-items: center;
	justify-content: space-between;
	gap: 12px;
	margin-bottom: 16px;
	padding: 14px 16px;
	background: var(--panel);
	border-radius: var(--radius);

	.summary {
		display: flex;
		flex-direction: column;

		.value {
			font-size: 1.3em;
			font-weight: 700;
		}

		.caption {
			font-size: 0.8em;
			opacity: 0.65;
		}
	}
}

.sectionTitle {
	margin: 0 0 8px;
	font-size: 0.95em;
	opacity: 0.8;
}

.tables,
.indexes {
	list-style: none;
	margin: 0;
	padding: 0;
	display: flex;
	flex-direction: column;
	gap: 6px;
}

.table {
	padding: 8px 12px;
	background: var(--panel);
	border-radius: var(--radius);

	.row {
		display: flex;
		gap: 12px;
		align-items: baseline;

		.name {
			flex: 1;
			min-width: 0;
			overflow: hidden;
			text-overflow: ellipsis;
		}

		.count,
		.size {
			font-size: 0.82em;
			opacity: 0.7;
			white-space: nowrap;
		}
	}

	.meter {
		height: 4px;
		margin-top: 6px;
		border-radius: 999px;
		background: var(--bg);
		overflow: hidden;
	}

	.val {
		height: 100%;
		background: var(--accent, #86b300);
	}
}

.index {
	padding: 8px 12px;
	background: var(--panel);
	border-radius: var(--radius);
	font-size: 0.85em;

	.indexName {
		margin-right: 8px;
	}

	.indexTable {
		opacity: 0.6;
	}

	details {
		margin-top: 4px;

		summary {
			cursor: pointer;
			opacity: 0.65;
		}
	}

	.def {
		display: block;
		margin-top: 4px;
		padding: 8px 10px;
		overflow-x: auto;
		background: var(--bg);
		border-radius: 4px;
		white-space: pre-wrap;
		overflow-wrap: anywhere;
	}
}

.notice {
	margin-top: 16px;
}
</style>
