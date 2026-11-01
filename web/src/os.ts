// os.ts is the imperative UI command bus: commands like openPostForm and
// confirmDialog are called from anywhere, and the singleton overlay container
// (OsOverlay, mounted once in App.vue) renders the resulting layers.
//
// It also carries the Firefish-shaped runtime shim. Two rules keep the port
// honest (see docs/research/2026-11-01-runtime-adaptation-contract.md):
//
//   * every request goes through `request()` in web/src/api/client.ts, which
//     owns the token, the error envelope and viewer stamping — this file only
//     reshapes the call;
//   * a member Firefish has but Cinderfish cannot support is either absent or
//     returns an explicit result, never a silent fake.
import {
  computed,
  defineComponent,
  h,
  onUnmounted,
  shallowRef,
  type CSSProperties,
  type Ref,
} from "vue";
import { ApiError, request } from "./api/client";
import type { DriveFile, Note } from "./api/types";
import PostForm from "./components/composer/PostForm.vue";
import DialogHost from "./components/ui/DialogHost.vue";
import EmojiPicker from "./components/ui/EmojiPicker.vue";
import PageWindow from "./components/ui/PageWindow.vue";
import PopupMenu from "./components/ui/PopupMenu.vue";
import { i18n, t } from "./i18n";
import { router } from "./router";
import { session } from "./stores/session";
import type { MenuItem } from "./types/menu";

export { pendingApiRequestsCount } from "./api/client";
export { claimZIndex } from "./lib/z-index";
export type { ZIndexPriority } from "./lib/z-index";

let nextRequestId = 0;

// ---------------------------------------------------------------------------
// Firefish-shaped API
// ---------------------------------------------------------------------------

/**
 * Firefish's `os.api(endpoint, params, token?, useToken?)`. `endpoint` is a
 * bare RPC name ("i", "notes/show"), a path already under /api/, or an absolute
 * URL; web/src/api/client.ts resolves it and applies the token rules.
 */
export function api<T = unknown>(
  endpoint: string,
  params: Record<string, unknown> = {},
  token?: string | null,
  useToken = true,
): Promise<T> {
  return request<T>(endpoint, params, { token, useToken });
}

/**
 * `api` plus a visible failure. The original error is rethrown untouched so
 * callers can still read `status`/`code`/`id`; the dialog mirrors Firefish's
 * `message + "\n" + id` so the request id stays copyable for a bug report.
 */
export async function apiWithDialog<T = unknown>(
  endpoint: string,
  params: Record<string, unknown> = {},
  token?: string | null,
  useToken = true,
): Promise<T> {
  try {
    return await api<T>(endpoint, params, token, useToken);
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err);
    const id = err instanceof ApiError ? err.id : "";
    void alert({ type: "error", text: `${message}\n${id}` });
    throw err;
  }
}

// ---------------------------------------------------------------------------
// Post form
// ---------------------------------------------------------------------------

export interface PostFormOptions {
  reply?: Note;
  renote?: Note;
  /** Called once after the form closes (openPostForm's only extension). */
  onClosed?: () => void;
}

interface PostFormRequest extends PostFormOptions {
  id: number;
  resolve?: () => void;
}

const postFormRequest = shallowRef<PostFormRequest | null>(null);

/** Open the global post form. A `renote` target is rendered as a quote preview
 * and submitted as renoteId plus text (the backend accepts both on one note).
 *
 * The form itself is authenticated-only, so a guest goes to the login page
 * instead of getting an empty dialog. */
export function openPostForm(opts: PostFormOptions = {}): void {
  if (!session.me) {
    void router.push("/login");
    opts.onClosed?.();
    return;
  }
  postFormRequest.value = { ...opts, id: ++nextRequestId };
}

/** The shell's compose entry. Same command as openPostForm, named the way the
 * ported application shell asks for it. */
export function openComposer(opts: PostFormOptions = {}): void {
  openPostForm(opts);
}

function closePostForm(): void {
  const active = postFormRequest.value;
  postFormRequest.value = null;
  active?.resolve?.();
  active?.onClosed?.();
}

/** The props a ported Firefish call site passes to `os.post`. */
export interface PostFormProps {
  reply?: Note;
  renote?: Note;
  initialText?: string;
  initialCw?: string | null;
  initialVisibility?: string;
  initialFiles?: DriveFile[];
  instant?: boolean;
}

/** Firefish's `os.post(props, onClosed)`: opens the composer and resolves once
 * it closes. The Cinderfish composer currently accepts only reply/renote
 * targets; initial* props are accepted so ported call sites typecheck, and are
 * reported instead of quietly discarded until the composer rebuild can honour
 * them. */
export function post(props: PostFormProps = {}, onClosed?: () => void): Promise<void> {
  if (
    props.initialText ||
    props.initialCw ||
    props.initialVisibility ||
    props.initialFiles?.length ||
    props.instant
  ) {
    console.warn("os.post: initial composer content is not wired up yet");
  }
  return new Promise((resolve) => {
    if (!session.me) {
      void router.push("/login");
      resolve();
      onClosed?.();
      return;
    }
    postFormRequest.value = {
      id: ++nextRequestId,
      reply: props.reply,
      renote: props.renote,
      resolve,
      onClosed,
    };
  });
}

// ---------------------------------------------------------------------------
// Reaction picker (existing command, unchanged)
// ---------------------------------------------------------------------------

type ReactionPickerRequest = {
  id: number;
  left: number;
  top: number;
  onSelect: (emoji: string) => void;
};

const reactionPickerRequest = shallowRef<ReactionPickerRequest | null>(null);

// Mirrors the sizes EmojiPicker's stylesheet caps itself at, so the popover can
// be positioned without mounting it first.
const PICKER_WIDTH = 336;
const PICKER_HEIGHT = 420;
const PICKER_GAP = 8;

/** Open the single reaction picker. The chosen emoji goes to `onSelect`;
 * `anchor` places the popover above (or below) the button that opened it.
 * Opening it again replaces the previous request. */
export function openReactionPicker(onSelect: (emoji: string) => void, anchor?: Element | null): void {
  const viewportWidth = window.innerWidth;
  const viewportHeight = window.innerHeight;
  let left = (viewportWidth - PICKER_WIDTH) / 2;
  let top = (viewportHeight - PICKER_HEIGHT) / 2;

  if (anchor) {
    const rect = anchor.getBoundingClientRect();
    left = rect.left;
    top = rect.top - PICKER_HEIGHT - PICKER_GAP;
    if (top < PICKER_GAP) top = rect.bottom + PICKER_GAP;
  }

  left = Math.max(PICKER_GAP, Math.min(left, viewportWidth - PICKER_WIDTH - PICKER_GAP));
  top = Math.max(PICKER_GAP, Math.min(top, viewportHeight - PICKER_HEIGHT - PICKER_GAP));

  reactionPickerRequest.value = { id: ++nextRequestId, left, top, onSelect };
}

function closeReactionPicker(): void {
  reactionPickerRequest.value = null;
}

// ---------------------------------------------------------------------------
// Dialogs (DialogHost.vue renders the head of the queue)
// ---------------------------------------------------------------------------

export type DialogType = "error" | "info" | "success" | "warning" | "waiting" | "question";

/** `result` carries the entered text (input) or chosen value (select); the
 * alert/confirm/yesno modes resolve `{ canceled: false, result: "ok" }`. */
export type DialogOutcome = { canceled: true; result?: undefined } | { canceled: false; result: string };

export interface DialogItem {
  value: string;
  text: string;
}

export interface DialogGroup {
  label: string;
  items: DialogItem[];
}

interface DialogBase {
  id: number;
  type: DialogType;
  title: string | null;
  text: string | null;
  isPlaintext: boolean;
  resolve: (outcome: DialogOutcome) => void;
}

export type DialogRequest = DialogBase &
  (
    | { kind: "alert" }
    | { kind: "confirm"; okText: string | null; cancelText: string | null; showCancelButton: boolean }
    | { kind: "yesno" }
    | {
        kind: "input";
        inputType: InputType;
        placeholder: string | null;
        autocomplete: string | null;
        defaultValue: string;
        minLength: number | null;
        maxLength: number | null;
      }
    | {
        kind: "select";
        items: DialogItem[] | null;
        groupedItems: DialogGroup[] | null;
        defaultValue: string | null;
      }
  );

export type InputType = "text" | "email" | "password" | "url" | "search" | "paragraph";

/** What a caller hands to showDialog; ids and resolvers are added here. */
export type DialogSpec =
  | { kind: "alert"; type?: DialogType; title?: string | null; text?: string | null; isPlaintext?: boolean }
  | {
      kind: "confirm";
      type?: DialogType;
      title?: string | null;
      text?: string | null;
      okText?: string | null;
      cancelText?: string | null;
      showCancelButton?: boolean;
      isPlaintext?: boolean;
    }
  | { kind: "yesno"; type?: DialogType; title?: string | null; text?: string | null; isPlaintext?: boolean }
  | {
      kind: "input";
      type?: DialogType;
      inputType?: InputType;
      title?: string | null;
      text?: string | null;
      placeholder?: string | null;
      autocomplete?: string | null;
      default?: string | null;
      minLength?: number;
      maxLength?: number;
      isPlaintext?: boolean;
    }
  | {
      kind: "select";
      type?: DialogType;
      title?: string | null;
      text?: string | null;
      default?: string | null;
      items?: DialogItem[];
      groupedItems?: DialogGroup[];
      isPlaintext?: boolean;
    };

const dialogQueue = shallowRef<DialogRequest[]>([]);

const currentDialog = computed(() => dialogQueue.value[0] ?? null);

/** Queue a dialog. The primitive behind alert/confirm/yesno/inputText/select. */
export function showDialog(spec: DialogSpec): Promise<DialogOutcome> {
  return new Promise((resolve) => {
    const base: DialogBase = {
      id: ++nextRequestId,
      type: spec.type ?? (spec.kind === "select" ? "question" : "info"),
      title: spec.title ?? null,
      text: spec.text ?? null,
      // Cinderfish has no Markdown renderer for dialog bodies; the flag is kept
      // so ported call sites keep their shape, and text is always plain.
      isPlaintext: spec.isPlaintext !== false,
      resolve,
    };
    let request: DialogRequest;
    switch (spec.kind) {
      case "alert":
        request = { ...base, kind: "alert" };
        break;
      case "confirm":
        request = {
          ...base,
          kind: "confirm",
          okText: spec.okText ?? null,
          cancelText: spec.cancelText ?? null,
          showCancelButton: spec.showCancelButton !== false,
        };
        break;
      case "yesno":
        request = { ...base, kind: "yesno" };
        break;
      case "input":
        request = {
          ...base,
          kind: "input",
          inputType: spec.inputType ?? "text",
          placeholder: spec.placeholder ?? null,
          autocomplete: spec.autocomplete ?? null,
          defaultValue: spec.default ?? "",
          minLength: spec.minLength ?? null,
          maxLength: spec.maxLength ?? null,
        };
        break;
      case "select":
        request = {
          ...base,
          kind: "select",
          items: spec.items ?? null,
          groupedItems: spec.groupedItems ?? null,
          defaultValue: spec.default ?? null,
        };
        break;
    }
    dialogQueue.value = [...dialogQueue.value, request];
  });
}

function resolveDialog(id: number, outcome: DialogOutcome): void {
  const [head, ...rest] = dialogQueue.value;
  if (!head || head.id !== id) return;
  dialogQueue.value = rest;
  head.resolve(outcome);
}

export interface AlertProps {
  type?: DialogType;
  title?: string | null;
  text?: string | null;
  isPlaintext?: boolean;
}

export interface ConfirmProps extends AlertProps {
  okText?: string;
  cancelText?: string;
  showCancelButton?: boolean;
}

export type YesNoProps = AlertProps;

/** Resolves after the user dismisses it, so an `await alert(...)` really waits. */
export function alert(props: AlertProps): Promise<void> {
  const text = props.text ?? (props.type === "error" ? i18n.ts.somethingHappened : null);
  return showDialog({ ...props, kind: "alert", text }).then(() => undefined);
}

export interface InputTextProps {
  type?: InputType;
  title?: string | null;
  text?: string | null;
  placeholder?: string | null;
  autocomplete?: string;
  default?: string | null;
  minLength?: number;
  maxLength?: number;
  isPlaintext?: boolean;
}

export interface SelectItem<C extends string> {
  value: C;
  text: string;
}

export interface SelectGroup<C extends string> {
  label: string;
  items: SelectItem<C>[];
}

export type SelectProps<C extends string> = {
  title?: string | null;
  text?: string | null;
  default?: string | null;
} & (
  | { items: SelectItem<C>[]; groupedItems?: undefined }
  | { items?: undefined; groupedItems: SelectGroup<C>[] }
);

/** Ask for confirmation. Returns `{ canceled }` as Firefish does. */
export function confirm(props: ConfirmProps): Promise<{ canceled: boolean }> {
  return showDialog({
    ...props,
    kind: "confirm",
    showCancelButton: props.showCancelButton !== false,
  }).then((outcome) => ({ canceled: outcome.canceled }));
}

/** Yes/No variant of `confirm`; the button labels are `yes`/`no`. */
export function yesno(props: YesNoProps): Promise<{ canceled: boolean }> {
  return showDialog({ ...props, kind: "yesno" }).then((outcome) => ({ canceled: outcome.canceled }));
}

/** Ask for one line of text. The discriminated union is preserved so callers
 * that narrow on `canceled` keep their `result` narrowing. */
export function inputText(
  props: InputTextProps,
): Promise<{ canceled: true; result?: undefined } | { canceled: false; result: string }> {
  return showDialog({
    kind: "input",
    inputType: props.type ?? "text",
    title: props.title,
    text: props.text,
    placeholder: props.placeholder,
    autocomplete: props.autocomplete,
    default: props.default,
    minLength: props.minLength,
    maxLength: props.maxLength,
    isPlaintext: props.isPlaintext,
  }).then((outcome) => (outcome.canceled ? { canceled: true } : { canceled: false, result: outcome.result }));
}

/** Single-choice dialog, flat or grouped; Escape and the backdrop cancel. */
export async function select<C extends string>(
  props: SelectProps<C>,
): Promise<{ canceled: true; result?: undefined } | { canceled: false; result: C }> {
  const outcome = await showDialog({
    kind: "select",
    title: props.title,
    text: props.text,
    default: props.default,
    items: props.items,
    groupedItems: props.groupedItems,
  });
  return outcome.canceled ? { canceled: true } : { canceled: false, result: outcome.result as C };
}

/** The confirm the existing views already call; kept as its own signature and
 * now backed by the dialog queue. */
export function confirmDialog(message: string): Promise<boolean> {
  return confirm({ type: "question", text: message }).then(({ canceled }) => !canceled);
}

// ---------------------------------------------------------------------------
// Menus (PopupMenu.vue renders the head of the queue)
// ---------------------------------------------------------------------------

export interface MenuOptions {
  align?: "left" | "center" | "right";
  width?: number;
  viaKeyboard?: boolean;
  noReturnFocus?: boolean;
  anchor?: { x: "left" | "center" | "right"; y: "top" | "center" | "bottom" };
}

export interface MenuPlacement {
  /** Anchor point in viewport coordinates. */
  x: number;
  y: number;
  /** Which edge of the menu sits on the anchor point. */
  attachX: "left" | "center" | "right";
  attachY: "top" | "bottom";
  width: number | null;
  viaKeyboard: boolean;
  noReturnFocus: boolean;
}

export interface MenuRequest {
  id: number;
  items: MenuItem[] | Ref<MenuItem[]>;
  placement: MenuPlacement;
  src: HTMLElement | null;
  resolve: () => void;
}

const menuRequest = shallowRef<MenuRequest | null>(null);

function enqueueMenu(
  items: MenuItem[] | Ref<MenuItem[]>,
  src: HTMLElement | null,
  placement: MenuPlacement,
): Promise<void> {
  return new Promise((resolve) => {
    // One popover at a time: opening a second menu resolves the first.
    closeMenu();
    menuRequest.value = { id: ++nextRequestId, items, placement, src, resolve };
  });
}

function closeMenu(): void {
  const active = menuRequest.value;
  if (!active) return;
  menuRequest.value = null;
  active.resolve();
}

function closeMenuById(id: number): void {
  if (menuRequest.value?.id !== id) return;
  closeMenu();
}

/** Firefish's `os.popupMenu(items, src?, options?)`: resolves when the menu
 * closes. `items` may be a ref; the menu follows it without reopening. */
export function popupMenu(
  items: MenuItem[] | Ref<MenuItem[]>,
  src?: HTMLElement | null,
  options: MenuOptions = {},
): Promise<void> {
  const attachX = options.anchor?.x ?? options.align ?? "left";
  const anchorY = options.anchor?.y ?? "bottom";
  const rect = src?.getBoundingClientRect();
  const x = rect
    ? attachX === "center"
      ? rect.left + rect.width / 2
      : attachX === "right"
        ? rect.right
        : rect.left
    : window.innerWidth / 2;
  const y = rect
    ? anchorY === "center"
      ? rect.top + rect.height / 2
      : anchorY === "top"
        ? rect.top
        : rect.bottom
    : window.innerHeight / 2;
  return enqueueMenu(items, src ?? null, {
    x,
    y,
    attachX,
    attachY: anchorY === "top" ? "bottom" : "top",
    width: options.width ?? null,
    viaKeyboard: options.viaKeyboard === true,
    noReturnFocus: options.noReturnFocus === true,
  });
}

/** Firefish's `os.showMenu`: the same command under the shell's name. */
export const showMenu = popupMenu;

/** A menu at the pointer, with the browser menu suppressed. */
export function contextMenu(items: MenuItem[] | Ref<MenuItem[]>, ev: MouseEvent): Promise<void> {
  ev.preventDefault();
  return enqueueMenu(items, null, {
    x: ev.clientX,
    y: ev.clientY,
    attachX: "left",
    attachY: "top",
    width: null,
    viaKeyboard: false,
    noReturnFocus: true,
  });
}

// ---------------------------------------------------------------------------
// Overlay
// ---------------------------------------------------------------------------

function handleEscape(): void {
  // DialogHost owns Escape while a dialog is open: it is modal, keeps focus
  // inside and stops the event, so this handler must not also close the layer
  // underneath.
  if (currentDialog.value) return;
  if (menuRequest.value) {
    closeMenu();
    return;
  }
  if (reactionPickerRequest.value) {
    closeReactionPicker();
    return;
  }
  if (postFormRequest.value) closePostForm();
}

const backdropStyle: CSSProperties = {
  position: "fixed",
  inset: 0,
  zIndex: 1000,
  background: "rgba(0, 0, 0, 0.5)",
};

// The picker is a popover: it dims nothing, the backdrop only catches clicks.
const pickerBackdropStyle: CSSProperties = {
  position: "fixed",
  inset: 0,
  zIndex: 1100,
  background: "transparent",
};

const pickerLayerStyle = (request: ReactionPickerRequest): CSSProperties => ({
  position: "fixed",
  left: `${request.left}px`,
  top: `${request.top}px`,
  zIndex: 1101,
});

function postFormTitle(opts: PostFormOptions): string {
  if (opts.reply) return t("reply");
  if (opts.renote) return t("quote");
  return t("toPost");
}

/** The singleton overlay container. Mount it once in App.vue; it renders
  * nothing until a command opens a layer. Dialogs and menus own their own
  * backdrop and z-index (claimZIndex), so only the picker and the post form
  * use the backdrop below. */
export const OsOverlay = defineComponent({
  name: "OsOverlay",
  setup() {
    const onKeydown = (event: KeyboardEvent) => {
      if (event.key === "Escape") handleEscape();
    };
    window.addEventListener("keydown", onKeydown);
    onUnmounted(() => window.removeEventListener("keydown", onKeydown));

    function pickEmoji(request: ReactionPickerRequest, emoji: string): void {
      closeReactionPicker();
      request.onSelect(emoji);
    }

    return () => {
      const dialog = currentDialog.value;
      const menu = menuRequest.value;
      const form = postFormRequest.value;
      const picker = reactionPickerRequest.value;
      if (!dialog && !menu && !form && !picker) return null;
      return [
        dialog
          ? h(DialogHost, {
              key: `dialog-${dialog.id}`,
              request: dialog,
              onResolve: (outcome: DialogOutcome) => resolveDialog(dialog.id, outcome),
            })
          : null,
        menu
          ? h(PopupMenu, {
              key: `menu-${menu.id}`,
              request: menu,
              onClose: () => closeMenuById(menu.id),
            })
          : null,
        form || picker
          ? h("div", {
              key: "backdrop",
              style: form ? backdropStyle : pickerBackdropStyle,
              onClick: () => (form ? closePostForm() : closeReactionPicker()),
            })
          : null,
        picker
          ? h("div", { key: `picker-${picker.id}`, style: pickerLayerStyle(picker) }, [
              h(EmojiPicker, {
                onSelect: (emoji: string) => pickEmoji(picker, emoji),
                onClose: () => closeReactionPicker(),
              }),
            ])
          : null,
        form
          ? h(
              PageWindow,
              {
                key: form.id,
                title: postFormTitle(form),
                onClose: () => closePostForm(),
              },
              {
                default: () =>
                  h(PostForm, {
                    replyTo: form.reply ?? null,
                    renoteTo: form.renote ?? null,
                    onPosted: () => closePostForm(),
                    onCancel: () => closePostForm(),
                  }),
              },
            )
          : null,
      ];
    };
  },
});
