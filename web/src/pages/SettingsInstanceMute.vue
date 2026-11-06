<!-- Ported from firefish@b425f61:packages/client/src/pages/settings/instance-mute.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :display-back-button="true" />
		</template>
		<MkSpacer :content-max="700" :margin-min="16" :margin-max="32">
			<div class="_formRoot cf-instance-mute">
				<CfInfo class="_formBlock">
					被静音的实例，其账号的帖子不会再出现在你的任何时间线里；你自己的帖子、
					已经打开的帖子详情和通知不受影响。名单保存在账号设置里，换设备后仍然生效。
				</CfInfo>

				<CfTextarea
					v-model="text"
					class="_formBlock"
					tall
					code
					:disabled="saving"
					placeholder="mastodon.example&#10;misskey.example"
				>
					<template #label>实例主机（每行一个）</template>
					<template #caption>
						只写主机名，不要加 <code>https://</code> 或路径。保存时自动转小写、去重。
					</template>
				</CfTextarea>

				<CfInfo v-if="error !== null" warn class="_formBlock">
					{{ error }}
				</CfInfo>
				<CfInfo v-else-if="saved" class="_formBlock">
					已保存，共 {{ hosts.length }} 个实例。
				</CfInfo>

				<CfButton primary :disabled="!changed || saving" @click="save">
					<i class="ph-floppy-disk ph-bold"></i>
					{{ saving ? "保存中…" : "保存" }}
				</CfButton>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script setup lang="ts">
// Upstream keeps the list in `$i.mutedInstances` and saves it through
// `i/update`. Cinderfish implements the same contract: the server stores the
// hosts in the profile settings and filters the timelines, so this page is a
// plain textarea over the same field.
//
// The textarea is normalised on save (lowercase, trim, drop duplicates) so the
// stored list and the text the user sees afterwards agree.
import { computed, ref, watch } from "vue";
import { ApiError, api } from "@/api/client";
import { me, updateMe } from "@/account";
import type { MeDetailed } from "@/api/types";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfTextarea from "@/components/ui/CfTextarea.vue";

const existing = (me.value as MeDetailed | null)?.mutedInstances ?? [];
const initial = existing.join("\n");
const text = ref(initial);
const saving = ref(false);
const saved = ref(false);
const error = ref<string | null>(null);

const hosts = computed(() => {
	const seen = new Set<string>();
	const out: string[] = [];
	for (const line of text.value.split("\n")) {
		const host = line.trim().toLowerCase();
		if (host === "" || seen.has(host)) continue;
		seen.add(host);
		out.push(host);
	}
	return out;
});

const changed = computed(() => hosts.value.join("\n") !== initial);

async function save(): Promise<void> {
	if (saving.value || !changed.value) return;
	saving.value = true;
	error.value = null;
	saved.value = false;
	try {
		const updated = await api("i/update", { mutedInstances: hosts.value });
		updateMe(updated);
		text.value = hosts.value.join("\n");
		saved.value = true;
	} catch (err) {
		error.value =
			err instanceof ApiError
				? err.message
				: String(err);
	} finally {
		saving.value = false;
	}
}

watch(text, () => {
	saved.value = false;
});
</script>
