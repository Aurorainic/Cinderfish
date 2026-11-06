<!-- Ported from firefish@b425f61:packages/client/src/pages/settings/word-mute.vue (AGPL-3.0-only) -->
<template>
	<div class="_formRoot cf-settings-word-mute">
		<CfInfo class="_formBlock">
			匹配的帖子会在客户端被<strong>软过滤</strong>：时间线里直接隐藏，服务器不会删除或改动它们
			（<code>scripts/check-word-mute.ts</code> 的 <code>getWordSoftMute</code>，
			自己发的帖子不受影响）。
		</CfInfo>

		<CfInfo warn class="_formBlock">
			降级说明：上游的「硬过滤」连接 <code>i/update</code> 把词表写进账号设置，
			Cinderfish 没有这个端点，也没有服务器侧词表；这里只维护本机/本账号的软过滤列表。
		</CfInfo>

		<section class="_formBlock">
			<h2 class="sectionTitle">
				过滤词
				<span class="count">共 {{ entries.length }} 条</span>
			</h2>

			<ul v-if="entries.length > 0" class="entries">
				<li v-for="(entry, index) in entries" :key="index" class="entry">
					<div class="entryBody">
						<span v-if="isRegex(entry)" class="entryKind">正则</span>
						<span v-else class="entryKind">关键词组 · AND</span>

						<template v-if="isRegex(entry)">
							<code class="entryValue">{{ entry }}</code>
							<span v-if="!isValidRegex(entry)" class="invalid">
								格式无效，匹配时会被跳过
							</span>
						</template>
						<template v-else>
							<code
								v-for="(keyword, i) in entry"
								:key="i"
								class="keyword"
								>{{ keyword }}</code
							>
						</template>
					</div>
					<button
						class="_buttonIcon remove"
						:aria-label="t('remove')"
						:title="t('remove')"
						@click="removeAt(index)"
					>
						<i class="ph-x ph-bold"></i>
					</button>
				</li>
			</ul>
			<CfInfo v-else>
				还没有过滤词。下面添加的条目会立即保存并生效。
			</CfInfo>
		</section>

		<section class="_formBlock addForm">
			<h2 class="sectionTitle">添加条目</h2>

			<CfRadios
				class="_formBlock"
				:model-value="mode"
				@update:model-value="onMode"
			>
				<CfRadio value="regex">正则表达式</CfRadio>
				<CfRadio value="keywords">关键词组（全部包含才命中）</CfRadio>
			</CfRadios>

			<template v-if="mode === 'regex'">
				<div class="regexRow">
					<CfInput
						v-model="regexPattern"
						class="patternInput"
						placeholder="例如 note|ノート"
						@enter="addRegex()"
					>
						<template #label>模式（不含斜线）</template>
					</CfInput>
					<CfInput
						v-model="regexFlags"
						class="flagsInput"
						placeholder="im"
					>
						<template #label>修饰符</template>
					</CfInput>
				</div>
				<p class="hint">
					保存为 <code>/模式/修饰符</code>，例如
					<code>/note|ノート/i</code>。修饰符可留空，常用
					<code>i</code>（忽略大小写）、<code>m</code>（多行）、<code>s</code>（点号匹配换行）。
					也可以把完整写法 <code>/note|ノート/i</code> 直接粘进「模式」框，
					修饰符留空时会自动拆开。
				</p>
			</template>

			<template v-else>
				<CfInput
					v-model="keywordInput"
					placeholder="例如 广告 推广"
					@enter="addKeywords()"
				>
					<template #label>关键词（空格分隔）</template>
					<template #caption>
						空格分隔的多个词按 AND 处理：所有词都出现在正文里才隐藏该帖。
					</template>
				</CfInput>
			</template>

			<CfInfo v-if="errorMessage !== null" warn class="_formBlock">
				{{ errorMessage }}
			</CfInfo>

			<div class="actions">
				<CfButton primary @click="mode === 'regex' ? addRegex() : addKeywords()">
					<i class="ph-plus ph-bold ph-lg"></i>
					{{ t("add") }}
				</CfButton>
				<CfButton
					v-if="entries.length > 0"
					inline
					danger
					@click="clearAll()"
				>
					清空全部
				</CfButton>
				<span v-if="justSaved" class="status">{{ t("saved") }}</span>
			</div>
		</section>

		<section class="_formBlock">
			<h2 class="sectionTitle">两种条目格式</h2>
			<ul class="notes">
				<li>
					<strong>正则</strong>：直接是字符串，必须写成
					<code>/模式/修饰符</code>。匹配逻辑用
					<code>new RegExp(模式, 修饰符)</code> 对「CW + 正文 + 附件说明」做
					<code>test</code>，所以不带斜线的字符串永远不会命中。
				</li>
				<li>
					<strong>关键词组</strong>：字符串数组，例如
					<code>["广告", "优惠"]</code>，渲染为
					<span class="inlineExample"><code>广告</code> + <code>优惠</code></span
					>，表示两个词都要出现。
				</li>
				<li>
					匹配范围是帖子自身的 CW、正文与附件说明；转帖/引用/回复里被过滤时，
					整条也会被隐藏。转发和回复的判定见
					<code>check-word-mute.ts</code>。
				</li>
			</ul>
		</section>
	</div>
</template>

<script setup lang="ts">
// Firefish's page had a soft (client) and a hard (server) tab and edited a
// single textarea that it parsed on save. Cinderfish has no server-side word
// mute, so this page keeps only the soft list, but makes the two real entry
// shapes (regex string / keyword array) explicit instead of hiding them behind
// a textarea grammar, and validates regexes before writing.
import { computed, ref } from "vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfInput from "@/components/ui/CfInput.vue";
import CfRadio from "@/components/ui/CfRadio.vue";
import CfRadios from "@/components/ui/CfRadios.vue";
import type { RadioValue } from "@/components/ui/radios-key";
import { t } from "@/i18n";
import { confirm } from "@/os";
import { definePageMetadata } from "@/scripts/page-metadata";
import { defaultStore } from "@/store";

defineOptions({ name: "CfPageSettingsWordMute" });

type MutedWord = string | string[];
type Mode = "regex" | "keywords";

const mode = ref<Mode>("regex");
const regexPattern = ref("");
const regexFlags = ref("");
const keywordInput = ref("");
const errorMessage = ref<string | null>(null);
const justSaved = ref(false);

let savedTimer: number | null = null;

const entries = computed<MutedWord[]>(
	() => defaultStore.reactiveState.mutedWords.value,
);

function isRegex(entry: MutedWord): entry is string {
	return typeof entry === "string";
}

function onMode(value: RadioValue): void {
	if (value === "regex" || value === "keywords") mode.value = value;
	errorMessage.value = null;
}

function splitRegex(entry: string): { pattern: string; flags: string } | null {
	const matched = entry.match(/^\/(.+)\/(.*)$/);
	if (matched == null) return null;
	return { pattern: matched[1], flags: matched[2] };
}

function isValidRegex(entry: string): boolean {
	const parsed = splitRegex(entry);
	if (parsed == null) return false;
	try {
		new RegExp(parsed.pattern, parsed.flags);
		return true;
	} catch {
		return false;
	}
}

function persist(next: MutedWord[]): void {
	defaultStore.set("mutedWords", next);
	if (savedTimer != null) window.clearTimeout(savedTimer);
	justSaved.value = true;
	savedTimer = window.setTimeout(() => {
		justSaved.value = false;
	}, 2000);
}

function addRegex(): void {
	let pattern = regexPattern.value.trim();
	let flags = regexFlags.value.trim();

	// Accept a pasted literal (`/foo/i`) as well as a bare pattern.
	if (flags === "") {
		const literal = pattern.match(/^\/(.+)\/([dgimsuvy]*)$/);
		if (literal != null) {
			pattern = literal[1];
			flags = literal[2];
		}
	}

	if (pattern === "") {
		errorMessage.value = "正则表达式内容不能为空（斜线外的部分）。";
		return;
	}
	if (/[\r\n]/.test(pattern) || /[\r\n]/.test(flags)) {
		errorMessage.value =
			"这里只能写单行正则；check-word-mute 只认 /模式/修饰符 这一种单行格式。";
		return;
	}
	try {
		new RegExp(pattern, flags);
	} catch (err) {
		errorMessage.value = `正则表达式无效：${String(err)}`;
		return;
	}
	const entry = `/${pattern}/${flags}`;
	if (entries.value.includes(entry)) {
		errorMessage.value = "这条正则已经在列表里。";
		return;
	}
	persist([...entries.value, entry]);
	regexPattern.value = "";
	regexFlags.value = "";
	errorMessage.value = null;
}

function addKeywords(): void {
	const keywords = keywordInput.value
		.split(/\s+/)
		.map((keyword) => keyword.trim())
		.filter((keyword) => keyword !== "");
	if (keywords.length === 0) {
		errorMessage.value = "请输入至少一个关键词。";
		return;
	}
	if (entries.value.some((entry) => Array.isArray(entry) && entry.join(" ") === keywords.join(" "))) {
		errorMessage.value = "这组关键词已经在列表里。";
		return;
	}
	persist([...entries.value, keywords]);
	keywordInput.value = "";
	errorMessage.value = null;
}

function removeAt(index: number): void {
	persist(entries.value.filter((_, i) => i !== index));
}

async function clearAll(): Promise<void> {
	const { canceled } = await confirm({
		type: "warning",
		text: `确定清空全部 ${entries.value.length} 条过滤词吗？`,
	});
	if (canceled) return;
	persist([]);
}

definePageMetadata({
	title: t("wordMute"),
	icon: "ph-speaker-x ph-bold ph-lg",
});
</script>

<style lang="scss" scoped>
.cf-settings-word-mute {
	.sectionTitle {
		display: flex;
		align-items: baseline;
		gap: 8px;
		margin: 0 0 8px;
		color: var(--fg);
		font-size: 1em;
		opacity: 0.85;
	}

	.count {
		font-size: 0.8em;
		color: var(--fgTransparentWeak);
		opacity: 1;
	}

	.entries {
		display: flex;
		flex-direction: column;
		gap: 8px;
		margin: 0;
		padding: 0;
		list-style: none;
	}

	.entry {
		display: flex;
		align-items: center;
		gap: 8px;
		padding: 8px 12px;
		border: solid 1px var(--divider);
		border-radius: var(--radius);
		background: var(--panel);
	}

	.entryBody {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 6px;
		min-inline-size: 0;
		flex: 1;
	}

	.entryKind {
		font-size: 0.75em;
		padding: 1px 6px;
		border-radius: 999px;
		background: var(--accentedBg);
		color: var(--fg);
		white-space: nowrap;
	}

	.entryValue,
	.keyword {
		padding: 1px 6px;
		border-radius: 4px;
		background: var(--X4);
		overflow-wrap: anywhere;
	}

	.invalid {
		color: var(--error);
		font-size: 0.85em;
	}

	.remove {
		flex: none;
		color: var(--fgTransparentWeak);

		&:hover {
			color: var(--fg);
		}
	}

	.addForm {
		border-top: solid 0.5px var(--divider);
		padding-block-start: 16px;
	}

	.regexRow {
		display: flex;
		flex-wrap: wrap;
		gap: 12px;

		> .patternInput {
			flex: 1 1 240px;
			min-inline-size: 0;
		}

		> .flagsInput {
			flex: 0 1 120px;
		}
	}

	.hint {
		margin: 8px 0 0;
		color: var(--fgTransparentWeak);
		font-size: 0.85em;
	}

	.actions {
		display: flex;
		flex-wrap: wrap;
		align-items: center;
		gap: 8px;
		margin-block-start: 12px;
	}

	.status {
		color: var(--fgTransparentWeak);
		font-size: 0.85em;
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

	.inlineExample {
		white-space: nowrap;
	}
}
</style>
