<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { api } from "../api/client";
import AuthLayout from "../layouts/AuthLayout.vue";
import { usePageTitle } from "../composables/use-page-title";
import { instance, loadInstance } from "../stores/instance";
import { signup } from "../stores/session";

const router = useRouter();

const username = ref("");
const password = ref("");
const invitationCode = ref("");
const error = ref("");
const busy = ref(false);
const availability = ref<"unknown" | "checking" | "free" | "taken">("unknown");

const closed = computed(
  () => instance.meta !== null && instance.meta.disableRegistration && !invitationCode.value,
);

usePageTitle("创建账号");

let checkTimer: ReturnType<typeof setTimeout> | undefined;

function scheduleAvailabilityCheck() {
  availability.value = "unknown";
  clearTimeout(checkTimer);
  const name = username.value.trim();
  if (name.length < 2) return;
  checkTimer = setTimeout(async () => {
    availability.value = "checking";
    try {
      const res = await api.usernameAvailable(name);
      availability.value = res.available ? "free" : "taken";
    } catch {
      availability.value = "unknown";
    }
  }, 400);
}

async function submit() {
  if (busy.value) return;
  error.value = "";
  const name = username.value.trim();
  if (!name || !password.value) {
    error.value = "请输入用户名和密码。";
    return;
  }
  if (availability.value === "taken") {
    error.value = "这个用户名已被占用。";
    return;
  }
  busy.value = true;
  try {
    await signup(name, password.value, invitationCode.value.trim());
    await router.push("/");
  } catch (err) {
    error.value = err instanceof Error ? err.message : "注册失败。";
  } finally {
    busy.value = false;
  }
}

onMounted(loadInstance);
</script>

<template>
  <AuthLayout>
    <h2 class="auth-title">创建账号</h2>
    <p class="muted small auth-lead">实例的注册模式决定谁能加入。</p>

    <p v-if="instance.meta?.disableRegistration" class="notice info">
      本实例未开放自助注册。如果管理员已发放邀请码，请在下方填写。
    </p>

    <form class="auth-form" @submit.prevent="submit">
      <div class="field">
        <label for="username">用户名</label>
        <input
          id="username"
          v-model="username"
          class="input"
          type="text"
          autocomplete="username"
          placeholder="username"
          @input="scheduleAvailabilityCheck"
        />
        <span v-if="availability === 'free'" class="hint ok">用户名可用</span>
        <span v-else-if="availability === 'taken'" class="hint bad">用户名已被占用</span>
        <span v-else-if="availability === 'checking'" class="hint">检查中…</span>
      </div>
      <div class="field">
        <label for="password">密码</label>
        <input
          id="password"
          v-model="password"
          class="input"
          type="password"
          autocomplete="new-password"
          placeholder="至少 8 位"
        />
      </div>
      <div class="field">
        <label for="invite">邀请码（可选）</label>
        <input id="invite" v-model="invitationCode" class="input" type="text" placeholder="邀请码" />
      </div>

      <p v-if="error" class="notice error" role="alert">{{ error }}</p>

      <button class="button auth-submit" type="submit" :disabled="busy || closed">
        {{ busy ? "创建中…" : "注册" }}
      </button>
    </form>

    <p class="muted small auth-switch">
      已有账号？<RouterLink to="/login">登录</RouterLink>
    </p>
  </AuthLayout>
</template>

<style scoped>
.auth-title {
  margin-bottom: var(--space-8);
  font-size: var(--fs-xl);
}

.auth-lead {
  margin-bottom: var(--space-24);
}

.auth-form {
  min-width: 0;
}

.auth-submit {
  width: 100%;
}

.auth-switch {
  margin-top: var(--space-16);
  text-align: center;
}

.auth-switch a {
  color: var(--accent);
  font-weight: var(--weight-semibold);
}

.auth-switch a:hover {
  color: var(--accent-hover);
}

@media (max-width: 380px) {
  .auth-title {
    font-size: var(--fs-lg);
  }
}
</style>
