<script setup lang="ts">
import { computed } from "vue";
import type { Notification } from "../api/types";
import { EMOJI_CATEGORIES, emojiName } from "../lib/emoji-catalog";
import { absoluteTime, relativeTime } from "../lib/time";
import AppIcon from "./AppIcon.vue";
import UserAvatar from "./UserAvatar.vue";

type IconName = "bell" | "heart" | "reply" | "renote" | "user" | "check" | "feather";

const props = defineProps<{ notification: Notification }>();

// Chinese wording per Misskey notification type, plus the tone that colours
// the type badge. Unknown types fall back to a neutral line so a new
// server-side type still renders something sensible.
const copy: Record<string, { icon: IconName; text: string; tone: string }> = {
  reaction: { icon: "heart", text: "反应了你的笔记", tone: "reaction" },
  reply: { icon: "reply", text: "回复了你", tone: "reply" },
  mention: { icon: "bell", text: "提到了你", tone: "mention" },
  follow: { icon: "user", text: "关注了你", tone: "follow" },
  followRequestAccepted: { icon: "user", text: "接受了你的关注请求", tone: "follow" },
  receiveFollowRequest: { icon: "user", text: "请求关注你", tone: "follow" },
  renote: { icon: "renote", text: "转贴了你的笔记", tone: "renote" },
  quote: { icon: "renote", text: "引用了你的笔记", tone: "renote" },
  pollEnded: { icon: "check", text: "你的投票已结束", tone: "poll" },
  note: { icon: "feather", text: "发布了新笔记", tone: "note" },
};

const meta = computed(
  () => copy[props.notification.type] ?? { icon: "bell" as IconName, text: "有一条新通知", tone: "note" },
);

const actorName = computed(
  () => props.notification.user?.name || props.notification.user?.username || "某位用户",
);

const to = computed(() => {
  const n = props.notification;
  if (n.noteId) return `/notes/${n.noteId}`;
  if (n.user?.username) return `/@${n.user.username}`;
  return "/notifications";
});

// The reaction picker renders from the same curated catalog. Reactions come
// from the API as a unicode character or as a custom `:shortcode:`; the
// shortcode is resolved to its catalog character and otherwise shown as-is,
// so an instance emoji that is not in the catalog still reads as a name.
const EMOJI_BY_NAME = new Map<string, string>();
for (const category of EMOJI_CATEGORIES) {
  for (const entry of category.emojis) EMOJI_BY_NAME.set(entry.name, entry.char);
}

const reaction = computed(() => {
  const raw = props.notification.reaction?.trim();
  if (!raw) return "";
  const shortcode = /^:([A-Za-z0-9_+-]+):$/.exec(raw);
  if (!shortcode) return raw;
  return EMOJI_BY_NAME.get(shortcode[1].toLowerCase()) ?? raw;
});

const reactionTitle = computed(() => emojiName(reaction.value));

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
const timeTitle = computed(() => absoluteTime(props.notification.createdAt));
</script>

<template>
  <RouterLink
    :to="to"
    class="item"
    :class="[`tone-${meta.tone}`, { unread: !notification.isRead }]"
  >
    <span class="badge">
      <AppIcon :name="meta.icon" />
    </span>

    <span class="avatar-slot">
      <UserAvatar v-if="notification.user" :user="notification.user" :size="36" />
      <span v-else class="avatar-empty" aria-hidden="true">?</span>
    </span>

    <span class="content">
      <span class="line">
        <strong class="actor">{{ actorName }}</strong>
        <span class="action">{{ meta.text }}</span>
        <span
          v-if="reaction"
          class="reaction"
          :title="`反应 ${reactionTitle}`"
          :aria-label="`反应 ${reactionTitle}`"
          >{{ reaction }}</span
        >
      </span>
      <span v-if="excerpt" class="excerpt">{{ excerpt }}</span>
    </span>

    <span class="tail">
      <time class="time" :title="timeTitle">{{ time }}</time>
      <span v-if="!notification.isRead" class="dot" title="未读" aria-label="未读"></span>
    </span>
  </RouterLink>
</template>

<style scoped>
/* Borderless notification row: the hairline belongs to the row, so rows butt
   against each other inside PagedFeed. The row is its own size container so
   it holds up in a ~350px deck column and a ~680px main column alike. */
.item {
  --tone: var(--muted);
  --tone-bg: var(--panel-2);
  container-name: notification;
  container-type: inline-size;
  display: grid;
  grid-template-columns: auto auto minmax(0, 1fr) auto;
  align-items: start;
  gap: 10px;
  padding: 12px 14px;
  border-bottom: 1px solid var(--divider);
  color: var(--text);
  transition: background 0.15s ease;
}

.item:hover {
  background: var(--panel-2);
  color: var(--text);
}

.item.unread {
  background: color-mix(in srgb, var(--accent) 6%, transparent);
  box-shadow: inset 2px 0 0 var(--accent);
}

.item.unread:hover {
  background: color-mix(in srgb, var(--accent) 11%, transparent);
}

.badge {
  flex: none;
  display: grid;
  place-items: center;
  width: 24px;
  height: 24px;
  margin-top: 6px;
  border-radius: 50%;
  background: var(--tone-bg);
  color: var(--tone);
}

.badge .icon {
  width: 14px;
  height: 14px;
}

.tone-reaction {
  --tone: #ff5c8a;
  --tone-bg: color-mix(in srgb, #ff5c8a 16%, transparent);
}

.tone-reply,
.tone-mention {
  --tone: var(--mention);
  --tone-bg: color-mix(in srgb, var(--mention) 16%, transparent);
}

.tone-renote {
  --tone: var(--renote);
  --tone-bg: color-mix(in srgb, var(--renote) 16%, transparent);
}

.tone-follow {
  --tone: var(--success);
  --tone-bg: color-mix(in srgb, var(--success) 16%, transparent);
}

.tone-poll {
  --tone: var(--hashtag);
  --tone-bg: color-mix(in srgb, var(--hashtag) 16%, transparent);
}

.tone-note {
  --tone: var(--accent);
  --tone-bg: var(--accent-soft);
}

.avatar-slot {
  flex: none;
  display: block;
  line-height: 0;
  margin-top: 1px;
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
  display: block;
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
  align-self: center;
  padding: 0 8px;
  border: 1px solid var(--border);
  border-radius: 999px;
  background: var(--panel);
  font-size: 0.95rem;
  line-height: 1.6;
  color: var(--text);
}

.excerpt {
  display: -webkit-box;
  margin-top: 3px;
  color: var(--muted);
  font-size: 0.84rem;
  line-height: 1.45;
  overflow: hidden;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  -webkit-box-orient: vertical;
  word-break: break-word;
}

.tail {
  display: flex;
  align-items: center;
  gap: 8px;
  padding-top: 2px;
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
}

/* Narrow deck columns: pull the row in and let the time drop under the text
   instead of squeezing the actor line. */
@container notification (max-width: 430px) {
  .item {
    grid-template-columns: auto auto minmax(0, 1fr);
    gap: 8px;
    padding: 11px 10px;
  }

  .badge {
    width: 22px;
    height: 22px;
    margin-top: 5px;
  }

  .tail {
    grid-column: 3;
    grid-row: 2;
    justify-content: flex-start;
    padding-top: 2px;
  }

  .time {
    font-size: 0.75rem;
  }
}
</style>
