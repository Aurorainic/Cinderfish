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

// An explicit re-read for the instance widget's refresh action and its error
// state. The cached metadata is dropped first so the next load cannot answer
// from it; an in-flight request is still shared rather than raced.
export async function reloadInstance(): Promise<void> {
  instance.meta = null;
  instance.error = "";
  await loadInstance();
}
