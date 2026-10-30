<script setup lang="ts">
import { ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import AuthLayout from "../layouts/AuthLayout.vue";
import { usePageTitle } from "../composables/use-page-title";
import { signin } from "../stores/session";

const router = useRouter();
const route = useRoute();

const username = ref("");
const password = ref("");
const error = ref("");
const busy = ref(false);

usePageTitle("登录");

async function submit() {
  if (busy.value) return;
  error.value = "";
  if (!username.value.trim() || !password.value) {
    error.value = "请输入用户名和密码。";
    return;
  }
  busy.value = true;
  try {
    await signin(username.value.trim(), password.value);
    const redirect = typeof route.query.redirect === "string" ? route.query.redirect : "/";
    await router.push(redirect);
  } catch (err) {
    error.value = err instanceof Error ? err.message : "登录失败。";
  } finally {
    busy.value = false;
  }
}
</script>

<template>
  <AuthLayout>
    <h2 class="auth-title">登录</h2>
    <p class="muted small auth-lead">使用用户名和密码进入实例。</p>

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
        />
      </div>
      <div class="field">
        <label for="password">密码</label>
        <input
          id="password"
          v-model="password"
          class="input"
          type="password"
          autocomplete="current-password"
          placeholder="••••••••"
        />
      </div>

      <p v-if="error" class="notice error" role="alert">{{ error }}</p>

      <button class="button auth-submit" type="submit" :disabled="busy">
        {{ busy ? "登录中…" : "登录" }}
      </button>
    </form>

    <p class="muted small auth-switch">
      还没有账号？<RouterLink to="/signup">注册</RouterLink>
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
