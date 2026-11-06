<!-- Ported from firefish@b425f61:packages/client/src/pages/settings/delete-account.vue (AGPL-3.0-only) -->
<template>
	<MkStickyContainer>
		<template #header>
			<MkPageHeader :display-back-button="true" />
		</template>
		<MkSpacer :content-max="700" :margin-min="16" :margin-max="32">
			<div class="_formRoot cf-settings-delete-account">
				<CfInfo warn class="_formBlock">
					注销会立即删除你的账号、帖子、关注关系、收藏与网盘文件，无法恢复。
					导出备份请先到 <CfA class="_link" to="/settings/import-export">导入/导出数据</CfA>。
				</CfInfo>

				<CfInput
					v-model="password"
					class="_formBlock"
					type="password"
					autocomplete="current-password"
					:disabled="deleting"
				>
					<template #label>当前密码</template>
					<template #caption
						>即使 token 泄露，没有密码也不能注销账号。</template
					>
				</CfInput>
				<CfInput
					v-model="confirmation"
					class="_formBlock"
					autocomplete="off"
					:spellcheck="false"
					:disabled="deleting"
				>
					<template #label>输入用户名确认</template>
					<template #prefix>@</template>
					<template #caption
						>请输入 <code>{{ username }}</code>，避免误触。</template
					>
				</CfInput>

				<CfInfo v-if="error !== null" warn class="_formBlock">
					{{ error }}
				</CfInfo>

				<CfButton
					danger
					:disabled="!canDelete || deleting"
					@click="remove"
				>
					{{ deleting ? "注销中…" : "永久注销账号" }}
				</CfButton>
			</div>
		</MkSpacer>
	</MkStickyContainer>
</template>

<script setup lang="ts">
// Firefish's delete-account page asks for the password and a typed
// confirmation. Cinderfish deletes synchronously and hard; the backend also
// refuses to remove the last administrator, whose error lands inline.
import { computed, ref } from "vue";
import { useRouter } from "vue-router";
import { ApiError, api } from "@/api/client";
import { me, signOut } from "@/account";
import MkPageHeader from "@/components/global/MkPageHeader.vue";
import MkSpacer from "@/components/global/MkSpacer.vue";
import MkStickyContainer from "@/components/global/MkStickyContainer.vue";
import CfA from "@/components/ui/CfA.vue";
import CfButton from "@/components/ui/CfButton.vue";
import CfInfo from "@/components/ui/CfInfo.vue";
import CfInput from "@/components/ui/CfInput.vue";

defineOptions({ name: "CfPageSettingsDeleteAccount" });

const router = useRouter();

const password = ref<string | number | null>("");
const confirmation = ref<string | number | null>("");
const deleting = ref(false);
const error = ref<string | null>(null);

const username = computed(() => me.value?.username ?? "");

const canDelete = computed(() => {
	const pass = String(password.value ?? "");
	const typed = String(confirmation.value ?? "").replace(/^@/, "");
	return pass !== "" && username.value !== "" && typed === username.value;
});

async function remove(): Promise<void> {
	if (!canDelete.value || deleting.value) return;
	deleting.value = true;
	error.value = null;
	try {
		await api("i/delete-account", {
			password: String(password.value ?? ""),
		});
		await signOut();
		await router.replace("/");
	} catch (cause) {
		if (cause instanceof ApiError) {
			error.value =
				cause.code === "LAST_ADMIN"
					? "这是最后一个管理员账号，实例会因此失去管理入口，不能注销。"
					: cause.code === "INVALID_PASSWORD"
						? "密码不正确。"
						: cause.message;
		} else {
			error.value = String(cause);
		}
	} finally {
		deleting.value = false;
	}
}
</script>
