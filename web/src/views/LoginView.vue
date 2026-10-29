<script setup lang="ts">
import { ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import AuthLayout from "../layouts/AuthLayout.vue";
import { signin } from "../stores/session";

const router = useRouter();
const route = useRoute();

const username = ref("");
const password = ref("");
const error = ref("");
const busy = ref(false);

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
    <h2 style="font-size: 1.5rem; margin-bottom: 8px">登录</h2>
    <p class="muted small" style="margin-bottom: 24px">使用用户名和密码进入实例。</p>

    <form @submit.prevent="submit">
      <div class="field">
        <label for="username">用户名</label>
        <input id="username" v-model="username" class="input" type="text" autocomplete="username" placeholder="username" />
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

      <p v-if="error" class="notice error">{{ error }}</p>

      <button class="button" type="submit" :disabled="busy" style="width: 100%">
        {{ busy ? "登录中…" : "登录" }}
      </button>
    </form>

    <p class="muted small" style="margin-top: 16px; text-align: center">
      还没有账号？<RouterLink to="/signup" style="color: var(--accent)">注册</RouterLink>
    </p>
  </AuthLayout>
</template>
