// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/os.ts (adapted) and
// firefish@02c31e0:packages/client/src/account.ts (accessors only).
//
// Firefish-shaped runtime surface over Cinderfish code:
//  - api / apiWithDialog delegate to web/src/api/client.ts (never fetch here);
//  - dialog/toast/popup use a simple Vue-reactive queue rendered by the ported
//    UI host components instead of Firefish's MkDialog component tree;
//  - me / isSignedIn / isAdmin delegate to web/src/me.ts.
// Dropped per the runtime adaptation contract: selectUser/selectInstance/
// selectDrive*/form/openEmojiPicker (capabilities Cinderfish lacks).
import { markRaw, ref } from "vue";
import { request, ApiError, pendingApiRequestsCount } from "../api/client";
import {
  isSignedIn,
  isAdmin as sessionIsAdmin,
  isModerator as sessionIsModerator,
  me,
} from "../me";
import { claimZIndex as zClaim } from "./z-index";

export { pendingApiRequestsCount };

// ---------------------------------------------------------------------------
// API
// ---------------------------------------------------------------------------

export const api = (
  endpoint: string,
  data: Record<string, unknown> = {},
  token?: string | null | undefined,
): Promise<unknown> => {
  return request(endpoint, token != null ? { ...data, i: token } : data);
};

export const apiGet = api as never; // rewritten to api per the adaptation contract

// Firefish's apiWithDialog: run the call and surface failures as an alert.
export async function apiWithDialog(
  endpoint: string,
  data: Record<string, unknown> = {},
  token?: string | null | undefined,
): Promise<unknown> {
  try {
    return await api(endpoint, data, token);
  } catch (err) {
    const message =
      err instanceof ApiError ? err.message : "请求失败，请稍后重试";
    alert({
      type: "error",
      text: message,
    });
    throw err;
  }
}

// ---------------------------------------------------------------------------
// Popup machinery (subset of os.popup)
// ---------------------------------------------------------------------------

export interface PopupType {
  component: unknown;
  props: Record<string, unknown>;
  events: Record<string, unknown>;
  id: number;
}

export const popups = ref<PopupType[]>([]);

let popupIdCount = 0;

export async function popup<Props>(
  component: unknown,
  props: Props,
  events: Record<string, unknown> = {},
  disposeEvent?: string,
): Promise<{ dispose: () => void }> {
  markRaw(component as object);
  const id = ++popupIdCount;
  const dispose = () => {
    window.setTimeout(() => {
      popups.value = popups.value.filter((p) => p.id !== id);
    }, 0);
  };
  popups.value.push({
    component,
    props: props as Record<string, unknown>,
    events: disposeEvent ? { ...events, [disposeEvent]: dispose } : events,
    id,
  });
  return { dispose };
}

export function claimZIndex(priority: "low" | "middle" | "high" = "low"): number {
  // Delegates to the shared layer helper; kept here so ported components can
  // call os.claimZIndex unchanged.
  return zClaim(priority);
}

let uniqueId = 0;
export function getUniqueId(): string {
  return String(uniqueId++);
}

// ---------------------------------------------------------------------------
// Dialogs
// ---------------------------------------------------------------------------

export type DialogType = "error" | "info" | "success" | "warning" | "waiting" | "question";

export interface DialogRequest {
  id: number;
  type: DialogType;
  title?: string | null;
  text?: string | null;
  showCancelButton: boolean;
  okText?: string;
  cancelText?: string;
  input?: {
    placeholder?: string | null;
    default?: string | null;
    type?: "text" | "password" | "email" | "url";
  } | null;
  select?: { value: string; text: string }[] | null;
  resolve: (value: { canceled: boolean; result?: string | null } | null) => void;
}

export const dialogQueue = ref<DialogRequest[]>([]);

function enqueueDialog(
  request: Omit<DialogRequest, "id" | "resolve">,
): Promise<{ canceled: boolean; result?: string | null } | null> {
  return new Promise((resolve) => {
    dialogQueue.value.push({ ...request, id: ++popupIdCount, resolve });
  });
}

/** Answers the frontmost dialog; called by the dialog host component. */
export function resolveDialog(
  id: number,
  value: { canceled: boolean; result?: string | null } | null,
): void {
  const index = dialogQueue.value.findIndex((d) => d.id === id);
  if (index === -1) return;
  const [request] = dialogQueue.value.splice(index, 1);
  request.resolve(value);
}

export function alert(props: {
  type?: DialogType;
  title?: string | null;
  text?: string | null;
}): Promise<void> {
  return enqueueDialog({
    type: props.type ?? "info",
    title: props.title,
    text: props.text ?? (props.type === "error" ? "发生了一些问题" : null),
    showCancelButton: false,
  }).then(() => undefined);
}

export function confirm(props: {
  type: DialogType;
  title?: string | null;
  text?: string | null;
  okText?: string;
  cancelText?: string;
}): Promise<{ canceled: boolean }> {
  return enqueueDialog({
    type: props.type,
    title: props.title,
    text: props.text,
    showCancelButton: true,
    okText: props.okText,
    cancelText: props.cancelText,
  }).then((result) => ({ canceled: result?.canceled ?? true }));
}

export function yesno(props: {
  type: DialogType;
  title?: string | null;
  text?: string | null;
}): Promise<{ canceled: boolean }> {
  return confirm({ ...props, okText: "是", cancelText: "否" });
}

export function inputText(props: {
  type?: DialogType;
  title?: string | null;
  text?: string | null;
  placeholder?: string | null;
  default?: string | null;
}): Promise<{ canceled: true; result: null } | { canceled: false; result: string }> {
  return enqueueDialog({
    type: props.type ?? "question",
    title: props.title,
    text: props.text,
    showCancelButton: true,
    input: {
      placeholder: props.placeholder,
      default: props.default,
      type: "text",
    },
  }).then((result) =>
    result == null || result.canceled || result.result == null
      ? { canceled: true, result: null }
      : { canceled: false, result: result.result },
  );
}

export function select<C extends string>(props: {
  type?: DialogType;
  title?: string | null;
  text?: string | null;
  items: { value: C; text: string }[];
  default?: C | null;
}): Promise<{ canceled: true; result: null } | { canceled: false; result: C }> {
  return enqueueDialog({
    type: props.type ?? "question",
    title: props.title,
    text: props.text,
    showCancelButton: true,
    select: props.items,
    input: props.default != null ? { default: props.default } : null,
  }).then((result) =>
    result == null || result.canceled || result.result == null
      ? { canceled: true, result: null }
      : { canceled: false, result: result.result as C },
  );
}

// ---------------------------------------------------------------------------
// Toast
// ---------------------------------------------------------------------------

export interface ToastRequest {
  id: number;
  message: string;
}

export const toasts = ref<ToastRequest[]>([]);

export function toast(message: string): void {
  const id = ++popupIdCount;
  toasts.value.push({ id, message });
  window.setTimeout(() => {
    toasts.value = toasts.value.filter((t) => t.id !== id);
  }, 4000);
}

/** Removes a toast early; called by the toast host component. */
export function dismissToast(id: number): void {
  toasts.value = toasts.value.filter((t) => t.id !== id);
}

// ---------------------------------------------------------------------------
// Account accessors (firefish@02c31e0:packages/client/src/account.ts, subset)
// ---------------------------------------------------------------------------

export { me, isSignedIn, sessionIsAdmin as isAdmin, sessionIsModerator as isModerator };

// Firefish's account module also signs out and updates the account remotely;
// those live in web/src/stores/session.ts, so the ported UI calls them there
// and os keeps read accessors only.
