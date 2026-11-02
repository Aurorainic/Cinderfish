<script setup lang="ts">
import { computed } from "vue";
import { useRoute } from "vue-router";
import type { UserDetailed } from "../api/types";
import UserAvatar from "./UserAvatar.vue";
import MkButton from "./ui/MkButton.vue";
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
const loginTarget = computed(() => `/login?redirect=${encodeURIComponent(route.fullPath)}`);

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
  <section class="_panel profile">
    <div class="banner">
      <img v-if="user.bannerUrl" :src="user.bannerUrl" alt="" />
    </div>

    <div class="body">
      <div class="head">
        <UserAvatar class="avatar" :user="user" :size="96" />

        <div class="actions">
          <MkButton v-if="isSelf" link :to="NAV_ITEM_DEFS.settings.to">编辑资料</MkButton>
          <MkButton
            v-else-if="loggedIn"
            :primary="!following && !pending"
            :disabled="followBusy || pending"
            @click="toggleFollow"
          >
            <template v-if="followBusy">处理中…</template>
            <template v-else-if="pending">请求已发送</template>
            <template v-else-if="following">取消关注</template>
            <template v-else>关注</template>
          </MkButton>
          <MkButton v-else link :to="loginTarget">登录后关注</MkButton>
        </div>
      </div>

      <div class="names">
        <h1 class="name">{{ displayName }}</h1>
        <p class="handle">@{{ user.username }}</p>
      </div>

      <p v-if="user.description" class="bio">{{ user.description }}</p>
      <p v-else class="bio empty">还没有简介</p>

      <p v-if="joinDate" class="joined _caption">{{ joinDate }}</p>

      <div class="counters">
        <div class="counter">
          <span class="value">{{ formatCount(user.notesCount) }}</span>
          <span class="label">笔记</span>
        </div>
        <div class="counter" title="关注列表接口尚未提供，暂时只能查看数量">
          <span class="value">{{ formatCount(user.followingCount) }}</span>
          <span class="label">关注</span>
        </div>
        <div class="counter" title="关注者列表接口尚未提供，暂时只能查看数量">
          <span class="value">{{ formatCount(user.followersCount) }}</span>
          <span class="label">关注者</span>
        </div>
      </div>

      <p class="lists-note">关注与关注者列表接口尚未提供，这里只显示数量。</p>
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
  background: var(--panelHighlight);
  border-block-end: solid 0.5px var(--divider);
}

.banner img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  display: block;
}

.body {
  padding-block: 0 20px;
  padding-inline: 20px;
}

.head {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
  margin-block-start: -40px;
  margin-block-end: 12px;
}

.avatar {
  border: 3px solid var(--panel);
  background: var(--panelHighlight);
  border-radius: 50%;
}

.actions {
  display: flex;
  gap: 8px;
  margin-inline-start: auto;
  padding-block-end: 4px;
}

.names {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.name {
  font-size: 1.45em;
  overflow-wrap: anywhere;
}

.handle {
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
}

.counters {
  display: flex;
  flex-wrap: wrap;
  gap: 22px;
  margin-block-start: 16px;
  padding-block-start: 14px;
  border-block-start: solid 0.5px var(--divider);
}

.counter {
  display: flex;
  align-items: baseline;
  gap: 6px;
}

.counter .value {
  font-weight: 700;
  color: var(--fgHighlighted);
}

.counter .label {
  font-size: 0.9em;
  color: var(--fgTransparentWeak);
}

.lists-note {
  margin-block-start: 8px;
  color: var(--fgTransparent);
  font-size: 0.8em;
  line-height: 1.5;
}

@media (max-width: 480px) {
  .body {
    padding-block: 0 16px;
    padding-inline: 14px;
  }
  .banner {
    height: 88px;
  }
  .name {
    font-size: 1.25em;
  }
  .counters {
    gap: 16px;
  }
}
</style>
