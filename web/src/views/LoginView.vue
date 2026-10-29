<script setup lang="ts">
import AuthLayout from "../layouts/AuthLayout.vue";
</script>

<template>
  <AuthLayout>
    <h2 style="font-size: 1.5rem; margin-bottom: 8px">登录</h2>
    <p class="muted small" style="margin-bottom: 24px">使用用户名和密码进入实例。</p>
    <div class="field">
      <label>用户名</label>
      <input class="input" type="text" placeholder="username" disabled />
    </div>
    <div class="field">
      <label>密码</label>
      <input class="input" type="password" placeholder="••••••••" disabled />
    </div>
    <button class="button" type="button" disabled style="width: 100%">登录</button>
    <p class="muted small" style="margin-top: 16px; text-align: center">
      还没有账号？<RouterLink to="/signup" class="muted" style="color: var(--accent)">注册</RouterLink>
    </p>
  </AuthLayout>
</template>
