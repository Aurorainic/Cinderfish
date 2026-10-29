<script setup lang="ts">
import { computed } from "vue";
import type { Notification } from "../api/types";
import { relativeTime } from "../lib/time";
import AppIcon from "./AppIcon.vue";
import UserAvatar from "./UserAvatar.vue";

type IconName = "bell" | "heart" | "reply" | "renote" | "user" | "check" | "feather";

const props = defineProps<{ notification: Notification }>();

// Chinese wording per Misskey notification type. Unknown types fall back to a
// neutral line so a new server-side type still renders something sensible.
const copy: Record<string, { icon: IconName; text: string }> = {
  reaction: { icon: "heart", text: "赞了你的笔记" },
  reply: { icon: "reply", text: "回复了你" },
  mention: { icon: "bell", text: "提到了你" },
  follow: { icon: "user", text: "关注了你" },
  followRequestAccepted: { icon: "user", text: "接受了你的关注请求" },
  receiveFollowRequest: { icon: "user", text: "请求关注你" },
  renote: { icon: "renote", text: "转贴了你的笔记" },
  quote: { icon: "renote", text: "引用了你的笔记" },
  pollEnded: { icon: "check", text: "你的投票已结束" },
  note: { icon: "feather", text: "发布了新笔记" },
};

const meta = computed(() => copy[props.notification.type] ?? { icon: "bell" as IconName, text: "有一条新通知" });

const actorName = computed(
  () => props.notification.user?.name || props.notification.user?.username || "某位用户",
);

const to = computed(() => {
  const n = props.notification;
  if (n.noteId) return `/notes/${n.noteId}`;
  if (n.user?.username) return `/@${n.user.username}`;
  return "/notifications";
});

const reactionLabel = computed(() => {
  const reaction = props.notification.reaction;
  return props.notification.type === "reaction" && reaction ? reaction : "";
});

// The related note, flattened to plain text: a boost shows the boosted body,
// and media-only notes get a short hint instead of an empty line.
const excerpt = computed(() => {
  const note = props.notification.note;
  if (!note) return "";
  let text = (note.text ?? "").trim();
  if (!text && note.renote) text = (note.renote.text ?? "").trim();
  if (!text) {
    if (note.renote) return "转贴了一条笔记";
    if (note.files.length) return "[附件]";
    return "";
  }
  text = text.replace(/\s+/g, " ").trim();
  return text.length > 140 ? `${text.slice(0, 140)}…` : text;
});

const time = computed(() => relativeTime(props.notification.createdAt));
</script>

<template>
  <RouterLink :to="to" class="item" :class="{ unread: !notification.isRead }">
    <div class="badge" :class="`badge-${notification.type}`">
      <AppIcon :name="meta.icon" />
    </div>

    <div class="avatar-slot">
      <UserAvatar v-if="notification.user" :user="notification.user" :size="36" />
      <span v-else class="avatar-empty" aria-hidden="true">?</span>
    </div>

    <div class="content">
      <p class="line">
        <strong class="actor">{{ actorName }}</strong>
        <span class="action">{{ meta.text }}</span>
        <span v-if="reactionLabel" class="reaction">{{ reactionLabel }}</span>
      </p>
      <p v-if="excerpt" class="excerpt">{{ excerpt }}</p>
    </div>

    <div class="tail">
      <time class="time">{{ time }}</time>
      <span v-if="!notification.isRead" class="dot" title="未读" aria-label="未读"></span>
    </div>
  </RouterLink>
</template>

<style scoped>
.item {
  position: relative;
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 12px 16px;
  color: var(--text);
  transition: background 0.15s ease;
}

.item::after {
  content: "";
  position: absolute;
  inset: 0;
  pointer-events: none;
  border-radius: inherit;
}

.item:hover {
  background: var(--panel-2);
  color: var(--text);
}

.item.unread {
  background: var(--accent-soft);
}

.item.unread:hover {
  background: color-mix(in srgb, var(--accent) 22%, var(--panel));
}

.badge {
  flex: none;
  display: grid;
  place-items: center;
  width: 26px;
  height: 26px;
  margin-top: 5px;
  border-radius: 50%;
  background: var(--panel-2);
  color: var(--muted);
}

.badge .icon {
  width: 15px;
  height: 15px;
}

.unread .badge {
  color: var(--accent);
  background: color-mix(in srgb, var(--accent) 18%, var(--panel));
}

.badge-heart,
.badge-reaction {
  color: #ff5c8a;
}

.avatar-slot {
  flex: none;
}

.avatar-empty {
  display: grid;
  place-items: center;
  width: 36px;
  height: 36px;
  border-radius: 50%;
  background: var(--panel-2);
  border: 1px solid var(--border);
  color: var(--muted);
  font-weight: 700;
}

.content {
  flex: 1;
  min-width: 0;
}

.line {
  display: flex;
  align-items: baseline;
  flex-wrap: wrap;
  gap: 5px;
  font-size: 0.92rem;
  line-height: 1.45;
}

.actor {
  color: var(--text-strong);
  font-weight: 700;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 14em;
}

.action {
  color: var(--muted);
}

.reaction {
  padding: 0 7px;
  border-radius: 999px;
  border: 1px solid var(--border);
  background: var(--panel);
  font-size: 0.85rem;
  color: var(--text);
}

.excerpt {
  margin-top: 3px;
  color: var(--muted);
  font-size: 0.84rem;
  line-height: 1.45;
  overflow: hidden;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  word-break: break-word;
}

.tail {
  flex: none;
  display: flex;
  align-items: center;
  gap: 8px;
  padding-top: 3px;
}

.time {
  color: var(--muted);
  font-size: 0.78rem;
  white-space: nowrap;
}

.dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--accent);
  box-shadow: 0 0 0 3px var(--accent-soft);
}

@media (max-width: 720px) {
  .badge {
    display: none;
  }
}
</style>
