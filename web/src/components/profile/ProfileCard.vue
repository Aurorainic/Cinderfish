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

// Only the note count has somewhere to go. The backend has no users/following
// or users/followers endpoint, so those two counters stay plain numbers and the
// note below them says why; linking them would promise a list that cannot be
// rendered.
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
        <span
          class="counter"
          :aria-label="`关注中 ${formatCount(user.followingCount)}，列表接口尚未提供`"
          title="关注列表接口尚未提供，暂时只能查看数量"
        >
          <b>{{ formatCount(user.followingCount) }}</b>
          <span>关注中</span>
        </span>
        <span
          class="counter"
          :aria-label="`关注者 ${formatCount(user.followersCount)}，列表接口尚未提供`"
          title="关注者列表接口尚未提供，暂时只能查看数量"
        >
          <b>{{ formatCount(user.followersCount) }}</b>
          <span>关注者</span>
        </span>
      </div>

      <p class="lists-note">关注与关注者列表接口尚未提供，这里只显示数量。</p>
    </div>
  </section>
</template>

<style scoped>
.profile-card {
  container-name: profile-card;
  container-type: inline-size;
  overflow: hidden;
  border: solid 0.5px var(--divider);
  border-radius: var(--radius);
  background: var(--panel);
  min-width: 0;
}

.banner {
  height: 132px;
  border-block-end: solid 0.5px var(--divider);
  background: var(--panelHighlight);
}

.banner img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.body {
  padding-block: 0 20px;
  padding-inline: 20px;
}

.avatar-frame {
  width: max-content;
  margin-block-start: -42px;
  padding: 3px;
  border-radius: 50%;
  background: var(--panel);
}

.name {
  margin: 10px 0 0;
  font-size: 1.4em;
  line-height: 1.2;
  color: var(--fgHighlighted);
  overflow-wrap: anywhere;
}

.handle {
  margin: 2px 0 0;
  color: var(--fgTransparentWeak);
  font-size: 0.9em;
  overflow-wrap: anywhere;
}

.bio {
  margin-block-start: 12px;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}

.bio.empty {
  color: var(--fgTransparentWeak);
}

.joined {
  margin-block-start: 8px;
  color: var(--fgTransparentWeak);
  font-size: 0.85em;
}

.counters {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 20px;
  margin-block-start: 16px;
  padding-block-start: 14px;
  border-block-start: solid 0.5px var(--divider);
}

.counter {
  display: inline-flex;
  align-items: baseline;
  gap: 6px;
  padding-block: 2px;
  color: var(--fgTransparentWeak);
  font-size: 0.9em;
  white-space: nowrap;
}

.counter b {
  color: var(--fgHighlighted);
  font-size: 1em;
  font-variant-numeric: tabular-nums;
}

.counter.link {
  border-radius: 4px;
  transition: color 0.1s ease;
}

.counter.link:hover b,
.counter.link:hover {
  color: var(--accent);
}

.counter.link:focus-visible {
  outline: solid 2px var(--accent);
  outline-offset: 2px;
}

.lists-note {
  margin-block-start: 8px;
  color: var(--fgTransparent);
  font-size: 0.8em;
  line-height: 1.5;
}

@container profile-card (max-width: 420px) {
  .banner {
    height: 96px;
  }

  .body {
    padding-block: 0 16px;
    padding-inline: 14px;
  }

  .avatar-frame {
    margin-block-start: -36px;
  }

  .name {
    font-size: 1.15em;
  }

  .counters {
    gap: 6px 14px;
  }
}
</style>
