import type { Ref } from "vue";
import { api } from "../api/client";
import type { UserDetailed } from "../api/types";

// following/create answers with the followee's UserLite only (Misskey
// contract), so the relationship after a follow is read back from users/show
// instead of being returned there. The relation flags and the authoritative
// followersCount are adopted wholesale, which also corrects the optimistic
// count bump when the follow turned out to be only a pending request.
export function useFollowRelation(
  user: Ref<UserDetailed | null>,
  following: Ref<boolean>,
  followPending: Ref<boolean>,
): { refreshRelation: () => Promise<void> } {
  async function refreshRelation(): Promise<void> {
    if (!user.value) return;
    const id = user.value.id;
    try {
      const fresh = await api.usersShow({ username: user.value.username });
      // A slow read must not apply one profile's relationship to another
      // that was opened while it was in flight.
      if (!user.value || user.value.id !== id) return;
      following.value = fresh.isFollowing ?? false;
      followPending.value = fresh.hasPendingFollowRequestFromYou ?? false;
      if (typeof fresh.followersCount === "number") user.value.followersCount = fresh.followersCount;
    } catch {
      // The optimistic state from the follow call stands on a failed read.
    }
  }
  return { refreshRelation };
}
