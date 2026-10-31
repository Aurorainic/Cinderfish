<script setup lang="ts">
import { computed } from "vue";
import { useRoute } from "vue-router";
import type { UserDetailed } from "../api/types";
import UserAvatar from "./UserAvatar.vue";
import { NAV_ITEM_DEFS } from "../nav";

const props = defineProps<{
  user: UserDetailed;
  isSelf: boolean;
  loggedIn: boolean;
  following: boolean;
  // A pending follow request (locked or remote followee) keeps the button
  // disabled until the other side answers.
  pending?: boolean;
  followBusy: boolean;
}>();

const emit = defineEmits<{ follow: []; unfollow: [] }>();

const route = useRoute();

const displayName = computed(() => props.user.name || props.user.username);
const joinDate = computed(() => {
  const d = new Date(props.user.createdAt);
  if (Number.isNaN(d.getTime())) return "";
  return `${d.getFullYear()} 年 ${d.getMonth() + 1} 月加入`;
});
const loginTarget = computed(() => ({ path: "/login", query: { redirect: route.fullPath } }));

function formatCount(n: number): string {
  return n.toLocaleString("zh-CN");
}

function toggleFollow() {
  if (props.followBusy) return;
  if (props.following) emit("unfollow");
  else emit("follow");
}
</script>

<template>
  <section class="card profile">
    <div class="banner">
      <img v-if="user.bannerUrl" :src="user.bannerUrl" alt="" />
    </div>

    <div class="body">
      <div class="head">
        <UserAvatar class="avatar" :user="user" :size="96" />

        <div class="actions">
          <RouterLink v-if="isSelf" :to="NAV_ITEM_DEFS.settings.to" class="button ghost small">编辑资料</RouterLink>
          <button
            v-else-if="loggedIn"
            type="button"
            class="button small"
            :class="{ ghost: following || pending }"
            :disabled="followBusy || pending"
            @click="toggleFollow"
          >
            <template v-if="followBusy">处理中…</template>
            <template v-else-if="pending">请求已发送</template>
            <template v-else-if="following">取消关注</template>
            <template v-else>关注</template>
          </button>
          <RouterLink v-else :to="loginTarget" class="button small">登录后关注</RouterLink>
        </div>
      </div>

      <div class="names">
        <h1 class="name">{{ displayName }}</h1>
        <p class="handle">@{{ user.username }}</p>
      </div>

      <p v-if="user.description" class="bio">{{ user.description }}</p>
      <p v-else class="bio muted">还没有简介</p>

      <p v-if="joinDate" class="joined small muted">{{ joinDate }}</p>

      <div class="counters">
        <div class="counter">
          <span class="value">{{ formatCount(user.notesCount) }}</span>
          <span class="label">笔记</span>
        </div>
        <div class="counter">
          <span class="value">{{ formatCount(user.followingCount) }}</span>
          <span class="label">关注</span>
        </div>
        <div class="counter">
          <span class="value">{{ formatCount(user.followersCount) }}</span>
          <span class="label">关注者</span>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.profile {
  padding: 0;
  overflow: hidden;
}

.banner {
  height: 108px;
  background: var(--panel-2);
  border-bottom: 1px solid var(--divider);
}

.banner img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.body {
  padding: 0 20px 20px;
}

.head {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
  margin-top: -40px;
  margin-bottom: 12px;
}

.avatar {
  border: 3px solid var(--panel);
  background: var(--panel-2);
  border-radius: 50%;
}

.actions {
  display: flex;
  gap: 8px;
  margin-left: auto;
  padding-bottom: 4px;
}

.names {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.name {
  font-size: 1.45rem;
  overflow-wrap: anywhere;
}

.handle {
  color: var(--muted);
  font-size: 0.9rem;
  overflow-wrap: anywhere;
}

.bio {
  margin-top: 12px;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

.joined {
  margin-top: 8px;
}

.counters {
  display: flex;
  flex-wrap: wrap;
  gap: 22px;
  margin-top: 16px;
  padding-top: 14px;
  border-top: 1px solid var(--border);
}

.counter {
  display: flex;
  align-items: baseline;
  gap: 6px;
}

.counter .value {
  font-weight: 700;
  color: var(--text-strong);
}

.counter .label {
  font-size: 0.85rem;
  color: var(--muted);
}

@media (max-width: 480px) {
  .body {
    padding: 0 14px 16px;
  }
  .banner {
    height: 88px;
  }
  .name {
    font-size: 1.25rem;
  }
  .counters {
    gap: 16px;
  }
}
</style>
