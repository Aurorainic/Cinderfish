<!-- Ported from firefish@b425f61:packages/client/src/pages/settings/preferences-backups.vue (AGPL-3.0-only) -->
<template>
	<div class="_formRoot cf-settings-preferences-backups">
		<CfInfo class="_formBlock">
			这里备份的是<strong>客户端偏好</strong>：<code>pizzax::base</code> 里的
			{{ knownKeys.length }} 个键（界面、时间线、通知、过滤词、自定义 CSS 等），
			不是服务器上的账号数据（资料、帖子、关注关系都不在备份里）。
			作用域为 <code>account</code> 的键本身会经 <code>i/registry/set</code>
			同步到本账号，<code>device</code> / <code>deviceAccount</code> 的键只存在这台设备的浏览器里。
		</CfInfo>

		<section class="_formBlock">
			<h2 class="sectionTitle">导出</h2>
			<p class="hint">
				下载一个 JSON 文件，内含
				<code>&#123; type: "cinderfish.preferences", version: 1, base: { … } &#125;</code>
				信封。文件里没有登录令牌，也没有服务器密钥。
			</p>
			<CfButton primary @click="exportFile()">
				<i class="ph-download-simple ph-bold ph-lg"></i>
				{{ t("export") }}
			</CfButton>
		</section>

		<section class="_formBlock">
			<h2 class="sectionTitle">导入</h2>
			<p class="hint">
				选择一个之前导出的 JSON 文件。只写入本页面认识的键，未知键会被忽略并在下方列出；
				写入前会再确认一次。导入不会删除备份里没有的键。
			</p>

			<input
				ref="fileInput"
				class="fileInput"
				type="file"
				accept="application/json,.json"
				@change="onFileChange"
			/>
			<CfButton @click="pickFile()">
				<i class="ph-upload-simple ph-bold ph-lg"></i>
				{{ t("import") }}
			</CfButton>

			<CfInfo v-if="importError !== null" warn class="_formBlock">
				{{ importError }}
			</CfInfo>

			<CfInfo v-if="importSummary !== null" class="_formBlock" icon="check-circle">
				<div>
					<div>{{ importSummary }}</div>
					<ul v-if="ignoredKeys.length > 0" class="ignored">
						<li v-for="key in ignoredKeys" :key="key">
							<code>{{ key }}</code>
						</li>
					</ul>
				</div>
			</CfInfo>
		</section>

		<section class="_formBlock">
			<h2 class="sectionTitle">重置</h2>
			<p class="hint">
				把上述每个键写回定义里的默认值（<code>defaultStore.reset(key)</code>）。
				相当于在新设备上第一次登录的状态，不会影响服务器上的账号数据。
			</p>
			<CfButton danger @click="resetAll()">
				<i class="ph-arrow-counter-clockwise ph-bold ph-lg"></i>
				重置为默认值
			</CfButton>
		</section>

		<section class="_formBlock">
			<h2 class="sectionTitle">与上游的差异</h2>
			<ul class="notes">
				<li>
					上游维护「命名配置档列表」，把多份配置存在服务器注册表
					<code>clientPreferencesProfiles</code> 作用域里，并支持重命名、覆盖、删除。
					本波按契约只做单份 JSON 文件的导出/导入：想保留多份配置时，把导出的文件另存即可。
				</li>
				<li>
					上游还会备份 <code>fontSize</code>、<code>useSystemFont</code>、
					<code>wallpaper</code> 三个裸 <code>localStorage</code> 键；Cinderfish 没有对应设置，
					因此既不导出也不导入。
				</li>
				<li>
					导入不会触发整页刷新：store 的 <code>reactiveState</code> 会立即更新，
					依赖它的界面（主题、导航、时间线偏好等）在下次渲染时取到新值。
				</li>
			</ul>
		</section>
	</div>
</template>

<script setup lang="ts">
// Firefish's page kept named profiles in the server registry. The wave-4
// contract scopes preference backup to browser-local files (`i/export` /
// `i/import` do not exist), so this page exports one JSON envelope and imports
// it back. Every write goes through `defaultStore.set`, which is also what
// keeps account-scoped keys in sync with `i/registry/set`.
import { ref } from "vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import { t } from "@/i18n";
import { confirm } from "@/os";
import { definePageMetadata } from "@/scripts/page-metadata";
import { defaultStore } from "@/store";

defineOptions({ name: "CfPageSettingsPreferencesBackups" });

const BACKUP_TYPE = "cinderfish.preferences";
const BACKUP_VERSION = 1;

type StoreKey = keyof typeof defaultStore.state & string;

const knownKeys = Object.keys(defaultStore.def) as StoreKey[];
const knownKeySet = new Set<string>(knownKeys);

const fileInput = ref<HTMLInputElement | null>(null);
const importError = ref<string | null>(null);
const importSummary = ref<string | null>(null);
const ignoredKeys = ref<string[]>([]);

function buildSnapshot(): Record<string, unknown> {
	const base: Record<string, unknown> = {};
	for (const key of knownKeys) {
		base[key] = defaultStore.state[key];
	}
	return base;
}

function exportFile(): void {
	const payload = {
		type: BACKUP_TYPE,
		version: BACKUP_VERSION,
		createdAt: new Date().toISOString(),
		base: buildSnapshot(),
	};
	const blob = new Blob([JSON.stringify(payload, null, 2)], {
		type: "application/json",
	});
	const url = URL.createObjectURL(blob);
	const anchor = document.createElement("a");
	anchor.href = url;
	anchor.download = `cinderfish-preferences-${fileStamp()}.json`;
	anchor.style.display = "none";
	document.body.append(anchor);
	anchor.click();
	anchor.remove();
	// Revoking synchronously can cancel the download in some browsers.
	window.setTimeout(() => URL.revokeObjectURL(url), 0);
}

function fileStamp(): string {
	const now = new Date();
	const pad = (value: number): string => String(value).padStart(2, "0");
	return [
		now.getFullYear(),
		pad(now.getMonth() + 1),
		pad(now.getDate()),
		"-",
		pad(now.getHours()),
		pad(now.getMinutes()),
		pad(now.getSeconds()),
	].join("");
}

function pickFile(): void {
	importError.value = null;
	importSummary.value = null;
	ignoredKeys.value = [];
	fileInput.value?.click();
}

function isPlainObject(value: unknown): value is Record<string, unknown> {
	return value != null && typeof value === "object" && !Array.isArray(value);
}

/** Returns the backup's `base` object, or a human-readable reason it is invalid. */
function parseEnvelope(text: string): Record<string, unknown> | string {
	let data: unknown;
	try {
		data = JSON.parse(text);
	} catch (err) {
		return `文件不是合法 JSON：${String(err)}`;
	}
	if (!isPlainObject(data)) return "文件顶层不是对象，不是 Cinderfish 的偏好备份。";
	if (data.type !== BACKUP_TYPE) {
		return `type 字段是 ${JSON.stringify(data.type)}，不是 "${BACKUP_TYPE}"。`;
	}
	if (data.version !== BACKUP_VERSION) {
		return `备份格式版本 ${JSON.stringify(data.version)} 不受支持（仅支持 version ${BACKUP_VERSION}）。`;
	}
	if (!isPlainObject(data.base)) return "备份缺少 base 对象。";
	return data.base;
}

async function onFileChange(): Promise<void> {
	const input = fileInput.value;
	if (input == null) return;
	const file = input.files?.[0];
	// Allow re-selecting the same file later.
	input.value = "";
	if (file == null) return;

	importError.value = null;
	importSummary.value = null;
	ignoredKeys.value = [];

	if (file.size > 2 * 1024 * 1024) {
		importError.value = "文件超过 2 MiB，不像是偏好备份。";
		return;
	}

	const parsed = parseEnvelope(await file.text());
	if (typeof parsed === "string") {
		importError.value = parsed;
		return;
	}

	const keys = Object.keys(parsed);
	const applyKeys: StoreKey[] = [];
	const ignored: string[] = [];
	for (const key of keys) {
		if (knownKeySet.has(key)) applyKeys.push(key as StoreKey);
		else ignored.push(key);
	}

	if (applyKeys.length === 0) {
		importError.value = "备份里没有任何本页面认识设置键，未做任何修改。";
		ignoredKeys.value = ignored;
		return;
	}

	const { canceled } = await confirm({
		type: "warning",
		text: `将用备份覆盖当前的 ${applyKeys.length} 项偏好设置${
			ignored.length > 0 ? `，另有 ${ignored.length} 个未知键会被忽略` : ""
		}。继续吗？`,
	});
	if (canceled) return;

	for (const key of applyKeys) {
		defaultStore.set(key, parsed[key] as never);
	}

	ignoredKeys.value = ignored;
	importSummary.value = `已写入 ${applyKeys.length} 项偏好设置。`;
}

async function resetAll(): Promise<void> {
	const { canceled } = await confirm({
		type: "warning",
		text: `确定把全部 ${knownKeys.length} 项偏好设置重置为默认值吗？此操作无法撤销。`,
	});
	if (canceled) return;

	for (const key of knownKeys) defaultStore.reset(key);

	importError.value = null;
	ignoredKeys.value = [];
	importSummary.value = `已把 ${knownKeys.length} 项偏好设置重置为默认值。`;
}

definePageMetadata({
	title: t("preferencesBackups"),
	icon: "ph-floppy-disk ph-bold ph-lg",
});
</script>

<style lang="scss" scoped>
.cf-settings-preferences-backups {
	.sectionTitle {
		margin: 0 0 4px;
		color: var(--fg);
		font-size: 1em;
		opacity: 0.85;
	}

	.hint {
		margin: 0 0 12px;
		color: var(--fgTransparentWeak);
		font-size: 0.9em;
	}

	.fileInput {
		display: none;
	}

	.ignored {
		margin: 6px 0 0;
		padding-inline-start: 1.2em;
		font-size: 0.9em;
	}

	.notes {
		margin: 0;
		padding-inline-start: 1.2em;
		color: var(--fgTransparentWeak);
		font-size: 0.9em;

		> li + li {
			margin-block-start: 6px;
		}

		code {
			color: var(--fg);
		}
	}
}
</style>
