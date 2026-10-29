<script setup lang="ts">
import AuthLayout from "../layouts/AuthLayout.vue";
</script>

<template>
  <AuthLayout>
    <h2 style="font-size: 1.5rem; margin-bottom: 8px">创建账号</h2>
    <p class="muted small" style="margin-bottom: 24px">实例的注册模式决定谁能加入。</p>
    <div class="field">
      <label>用户名</label>
      <input class="input" type="text" placeholder="username" disabled />
    </div>
    <div class="field">
      <label>密码</label>
      <input class="input" type="password" placeholder="••••••••" disabled />
    </div>
    <button class="button" type="button" disabled style="width: 100%">注册</button>
    <p class="muted small" style="margin-top: 16px; text-align: center">
      已有账号？<RouterLink to="/login" class="muted" style="color: var(--accent)">登录</RouterLink>
    </p>
  </AuthLayout>
</template>
