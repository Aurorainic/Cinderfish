import { reactive } from "vue";
import { api } from "../api/client";
import type { InstanceMeta } from "../api/types";

// Instance metadata is fetched once and shared; it drives the registration
// hints and the federation badge.
export const instance = reactive({
  meta: null as InstanceMeta | null,
  error: "",
});

let inflight: Promise<void> | null = null;

export async function loadInstance(): Promise<void> {
  if (instance.meta) return;
  if (inflight) return inflight;
  inflight = (async () => {
    try {
      instance.meta = await api.meta();
    } catch (err) {
      instance.error = err instanceof Error ? err.message : String(err);
    } finally {
      inflight = null;
    }
  })();
  return inflight;
}

// An explicit re-read, used by the instance widget's refresh action. Unlike the
// first load it never clears what is already in the store: a failed refresh
// leaves the sidebar brand, the composer's length limit and the about copy
// usable, and the error is thrown to the caller instead of being published
// app-wide, so only the widget that asked for the refresh reports it.
export async function reloadInstance(): Promise<void> {
  // Join a first load that is already running rather than racing it; that load
  // reports its own failure through `instance.error`.
  if (inflight) await inflight;
  instance.meta = await api.meta();
  instance.error = "";
}
