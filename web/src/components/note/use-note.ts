import { computed, ref, watch, type Ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import { api } from "../../api/client";
import type { Note } from "../../api/types";
import { isLoggedIn, session } from "../../stores/session";

// Interaction state for a single note: optimistic reactions, renote and
// delete. The DOM layer (Note.vue and its parts) only renders; every
// mutation and its optimistic bookkeeping lives here.
export function useNote(note: Ref<Note>) {
  const router = useRouter();
  const route = useRoute();

  const reactions = ref<Record<string, number>>({});
  const myReaction = ref<string | undefined>();
  const reactionCount = ref(0);
  const busy = ref(false);
  const notice = ref("");

  watch(
    note,
    (n) => {
      reactions.value = { ...n.reactions };
      myReaction.value = n.myReaction;
      reactionCount.value = n.reactionCount;
    },
    { immediate: true },
  );

  const isMine = computed(() => !!session.me && session.me.id === note.value.userId);

  const sortedReactions = computed(() =>
    Object.entries(reactions.value)
      .filter(([, count]) => count > 0)
      .sort((a, b) => b[1] - a[1]),
  );

  let noticeTimer: ReturnType<typeof setTimeout> | undefined;

  function flash(text: string, ttl = 2000) {
    notice.value = text;
    clearTimeout(noticeTimer);
    noticeTimer = setTimeout(() => {
      notice.value = "";
    }, ttl);
  }

  function errorText(err: unknown, fallback: string): string {
    return err instanceof Error && err.message ? err.message : fallback;
  }

  function requireSignin(): boolean {
    if (isLoggedIn.value) return true;
    router.push({ path: "/login", query: { redirect: route.fullPath } });
    return false;
  }

  async function toggleReaction(emoji: string): Promise<void> {
    if (!requireSignin() || busy.value) return;
    busy.value = true;
    try {
      if (myReaction.value === emoji) {
        await api.unreact(note.value.id);
        reactions.value[emoji] = (reactions.value[emoji] ?? 1) - 1;
        if (reactions.value[emoji] <= 0) delete reactions.value[emoji];
        myReaction.value = undefined;
        reactionCount.value = Math.max(0, reactionCount.value - 1);
      } else {
        await api.react(note.value.id, emoji);
        if (myReaction.value) {
          const old = myReaction.value;
          reactions.value[old] = (reactions.value[old] ?? 1) - 1;
          if (reactions.value[old] <= 0) delete reactions.value[old];
          reactionCount.value = Math.max(0, reactionCount.value - 1);
        }
        reactions.value[emoji] = (reactions.value[emoji] ?? 0) + 1;
        myReaction.value = emoji;
        reactionCount.value += 1;
      }
    } catch (err) {
      flash(errorText(err, "反应失败"), 4000);
    } finally {
      busy.value = false;
    }
  }

  // Returns the freshly created renote so callers can prepend it to a list.
  async function renote(): Promise<Note | null> {
    if (!requireSignin() || busy.value) return null;
    busy.value = true;
    try {
      const created = await api.createNote({ renoteId: note.value.id });
      flash("已转贴");
      return created;
    } catch (err) {
      flash(errorText(err, "转贴失败"), 4000);
      return null;
    } finally {
      busy.value = false;
    }
  }

  async function remove(): Promise<boolean> {
    if (!isMine.value || busy.value) return false;
    if (!window.confirm("删除这条笔记？")) return false;
    busy.value = true;
    try {
      await api.deleteNote(note.value.id);
      return true;
    } catch (err) {
      flash(errorText(err, "删除失败"), 4000);
      return false;
    } finally {
      busy.value = false;
    }
  }

  return {
    reactions,
    myReaction,
    reactionCount,
    sortedReactions,
    isMine,
    busy,
    notice,
    flash,
    toggleReaction,
    renote,
    remove,
  };
}
