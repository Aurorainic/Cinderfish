<script setup lang="ts">
import { computed } from "vue";
import { useRoute } from "vue-router";
import type { UserDetailed } from "../../api/types";
import UserAvatar from "../UserAvatar.vue";

const props = defineProps<{ user: UserDetailed }>();

const route = useRoute();

const displayName = computed(() => props.user.name || props.user.username);
const acct = computed(() =>
  props.user.host ? `${props.user.username}@${props.user.host}` : props.user.username,
);
const joinDate = computed(() => {
  const d = new Date(props.user.createdAt);
  if (Number.isNaN(d.getTime())) return "";
  return `${d.getFullYear()} 年 ${d.getMonth() + 1} 月加入`;
});

function formatCount(n: number): string {
  return n.toLocaleString("zh-CN");
}

// Only the note count has somewhere to go: the backend has no users/following
// or users/followers endpoint, so those two counters stay inert and say why.
const notesLink = computed(() => {
  const query = { ...route.query };
  delete query.tab;
  return { path: route.path, query };
});
</script>

<template>
  <section class="profile-card">
    <div class="banner">
      <img v-if="user.bannerUrl" :src="user.bannerUrl" alt="" />
    </div>

    <div class="body">
      <div class="avatar-frame">
        <UserAvatar :user="user" :size="84" />
      </div>

      <h2 class="name">{{ displayName }}</h2>
      <p class="handle">@{{ acct }}</p>

      <p v-if="user.description" class="bio">{{ user.description }}</p>
      <p v-else class="bio empty">还没有简介</p>

      <p v-if="joinDate" class="joined">{{ joinDate }}</p>

      <div class="counters">
        <RouterLink class="counter link" :to="notesLink" :title="`查看 ${formatCount(user.notesCount)} 条帖子`">
          <b>{{ formatCount(user.notesCount) }}</b>
          <span>帖子</span>
        </RouterLink>
        <span class="counter" aria-disabled="true" title="暂不支持查看关注列表">
          <b>{{ formatCount(user.followingCount) }}</b>
          <span>关注中</span>
        </span>
        <span class="counter" aria-disabled="true" title="暂不支持查看关注者列表">
          <b>{{ formatCount(user.followersCount) }}</b>
          <span>关注者</span>
        </span>
      </div>
    </div>
  </section>
</template>

<style scoped>
.profile-card {
  container-name: profile-card;
  container-type: inline-size;
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  background: var(--panel);
  min-width: 0;
}

.banner {
  height: 132px;
  border-bottom: 1px solid var(--divider);
  background: var(--surface-inset);
}

.banner img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.body {
  padding: 0 var(--space-20) var(--space-20);
}

.avatar-frame {
  width: max-content;
  margin-top: -42px;
  padding: 3px;
  border-radius: 50%;
  background: var(--panel);
}

.name {
  margin: var(--space-10) 0 0;
  font-size: var(--fs-xl);
  line-height: 1.2;
  color: var(--text-strong);
  overflow-wrap: anywhere;
}

.handle {
  margin: var(--space-2) 0 0;
  color: var(--text-weak);
  font-size: var(--fs-sm);
  overflow-wrap: anywhere;
}

.bio {
  margin-top: var(--space-12);
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

.bio.empty {
  color: var(--text-weak);
}

.joined {
  margin-top: var(--space-8);
  color: var(--text-weak);
  font-size: var(--fs-xs);
}

.counters {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-8) var(--space-20);
  margin-top: var(--space-16);
  padding-top: var(--space-14);
  border-top: 1px solid var(--border);
}

.counter {
  display: inline-flex;
  align-items: baseline;
  gap: var(--space-6);
  padding: var(--space-2) 0;
  color: var(--text-weak);
  font-size: var(--fs-sm);
  white-space: nowrap;
}

.counter b {
  color: var(--text-strong);
  font-size: var(--fs-md);
  font-variant-numeric: tabular-nums;
}

.counter.link {
  border-radius: var(--radius-xs);
  transition: color var(--dur-fast) ease;
}

.counter.link:hover b,
.counter.link:hover {
  color: var(--accent);
}

.counter.link:focus-visible {
  outline: var(--focus-outline);
  outline-offset: 2px;
}

@container profile-card (max-width: 420px) {
  .banner {
    height: 96px;
  }

  .body {
    padding: 0 var(--space-14) var(--space-16);
  }

  .avatar-frame {
    margin-top: -36px;
  }

  .name {
    font-size: var(--fs-lg);
  }

  .counters {
    gap: var(--space-6) var(--space-14);
  }
}
</style>
