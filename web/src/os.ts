// os.ts is the imperative UI command bus: commands like openPostForm and
// confirmDialog are called from anywhere, and the singleton overlay container
// (OsOverlay, mounted once in App.vue) renders the resulting layers.
//
// Escape is owned by one stack (lib/layer-stack.ts) and one document-capture
// listener: whatever opened last closes first, and the event stops there, so
// components with their own Escape listeners (nav drawer, widgets drawer, page
// window, note menus) can no longer close a second layer in the same key press.
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
  isRef,
  markRaw,
  onUnmounted,
  shallowRef,
  toValue,
  watch,
  type CSSProperties,
  type Component,
  type MaybeRefOrGetter,
  type Ref,
} from "vue";
import { ApiError, request } from "./api/client";
import type { DriveFile, Note } from "./api/types";
import PostForm from "./components/composer/PostForm.vue";
import { closeDeckPage, deckPage, deckPageFor, openDeckPage } from "./components/deck/deck-page";
import DialogHost from "./components/ui/DialogHost.vue";
import EmojiPicker from "./components/ui/EmojiPicker.vue";
import PageWindow from "./components/ui/PageWindow.vue";
import PopupMenu from "./components/ui/PopupMenu.vue";
import { i18n, t } from "./i18n";
import { claimZIndex } from "./lib/z-index";
import { createLayerStack, type LayerEntry } from "./lib/layer-stack";
import { router } from "./router";
import { session } from "./stores/session";
import type { MenuItem } from "./types/menu";

export { pendingApiRequestsCount } from "./api/client";
export { claimZIndex } from "./lib/z-index";
export type { ZIndexPriority } from "./lib/z-index";

let nextRequestId = 0;

// ---------------------------------------------------------------------------
// Layer stack
// ---------------------------------------------------------------------------

// Everything Escape can close is registered here, in opening order:
// dialogs, menus, the composer, the picker, generic popups and the deck page
// window. OsOverlay installs one document-capture listener that closes exactly
// the top entry and stops the event, so a component's own Escape handler
// (nav drawer, widgets drawer, page window, note menus) can no longer close a
// second layer at the same time. That ownership is the point: registration
// order between independent components was not a contract.
const layerStack = createLayerStack();

function pushLayer(close: () => void, zIndex: number): LayerEntry {
  return layerStack.push(close, zIndex);
}

export interface LayerOptions {
  /** Z band, as in Firefish's claimZIndex. Ported overlays are "middle". */
  priority?: "low" | "middle" | "high";
  /** Dim the page behind the layer and close it on a backdrop click. */
  modal?: boolean;
  /** Fixed position of the layer (popovers). Defaults to a full-viewport,
   * click-through container so the component can place itself. */
  position?: { left: number; top: number } | null;
  /** Emit handlers, keyed by the event name Firefish would use. */
  events?: Record<string, (...args: never[]) => void>;
  /** Runs once, however the layer closed. */
  onClose?: () => void;
}

export interface LayerHandle {
  id: number;
  zIndex: number;
  /** Closes the layer. Idempotent and safe to call twice. */
  dispose: () => void;
}

interface GenericLayer extends LayerHandle {
  component: Component;
  props: MaybeRefOrGetter<Record<string, unknown>>;
  eventProps: Record<string, (...args: never[]) => void>;
  entry: LayerEntry;
  modal: boolean;
  position: { left: number; top: number } | null;
  onClose?: () => void;
  closing: boolean;
}

const genericLayers = shallowRef<GenericLayer[]>([]);

/**
 * Mount a component as a global layer. This is the primitive behind `popup`
 * and the emoji picker: the component is rendered by OsOverlay at a claimed
 * z-index, and Escape closes the topmost layer through the same stack as the
 * dialogs.
 */
export function openLayer<P extends Record<string, unknown>>(
  component: Component,
  props: MaybeRefOrGetter<P>,
  options: LayerOptions = {},
): LayerHandle {
  const id = ++nextRequestId;
  const renderZIndex = claimZIndex("high");
  const eventProps = eventPropsFor(options.events ?? {});
  let layer!: GenericLayer;
  const entry = pushLayer(() => closeLayer(id), renderZIndex);
  layer = {
    id,
    zIndex: renderZIndex,
    component: markRaw(component),
    props,
    eventProps,
    entry,
    modal: options.modal === true,
    position: options.position ?? null,
    onClose: options.onClose,
    closing: false,
    dispose: () => closeLayer(id),
  };
  genericLayers.value = [...genericLayers.value, layer];
  return layer;
}

/** Close one layer, or the topmost one when no target is given. */
export function closeLayer(target?: LayerHandle | number | null): boolean {
  if (target === undefined || target === null) return layerStack.closeTop();
  const id = typeof target === "number" ? target : target.id;
  const layer = genericLayers.value.find((candidate) => candidate.id === id);
  if (!layer || layer.closing) return false;
  layer.closing = true;
  layerStack.remove(layer.entry);
  // Firefish defers the removal by a tick because unmounting inside the emit
  // that asked for the close breaks Vue's event/autocomplete cleanup.
  window.setTimeout(() => {
    genericLayers.value = genericLayers.value.filter((candidate) => candidate.id !== id);
    layer.onClose?.();
  }, 0);
  return true;
}

function eventPropsFor(events: Record<string, (...args: never[]) => void>): Record<string, (...args: never[]) => void> {
  const props: Record<string, (...args: never[]) => void> = {};
  for (const [name, handler] of Object.entries(events)) {
    props[`on${name.charAt(0).toUpperCase()}${name.slice(1)}`] = handler;
  }
  return props;
}

// Top-level refs in `props` are read inside the render function below, which
// keeps them live (the Firefish `CanUseRef` shape) without handing the mounted
// component a Ref it would have to unwrap itself.
function resolveLayerProps(layer: GenericLayer): Record<string, unknown> {
  const raw = toValue(layer.props) ?? {};
  const resolved: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(raw)) {
    resolved[key] = isRef(value) ? value.value : value;
  }
  return resolved;
}

/**
 * Firefish's `os.popup(component, props, events?, disposeEvent?)`: mount a
 * component as a layer and resolve once it can be disposed. The returned
 * `dispose` is idempotent; `disposeEvent` names the component event that
 * closes the layer (Firefish's `"closed"` convention).
 */
export function popup<P extends Record<string, unknown>>(
  component: Component,
  props: MaybeRefOrGetter<P>,
  events: Record<string, (...args: never[]) => void> = {},
  disposeEvent?: string,
): Promise<{ dispose: () => void }> {
  const handlers = { ...events };
  let handle!: LayerHandle;
  if (disposeEvent) {
    handlers[disposeEvent] = () => handle.dispose();
  }
  handle = openLayer(component, props, { events: handlers });
  return Promise.resolve({ dispose: handle.dispose });
}

// The deck page window can be raised by a deck column click as well as by
// pageWindow, so the stack follows the state instead of the command: whichever
// path opened it, Escape closes it like any other layer.
let deckPageLayer: LayerEntry | null = null;

watch(deckPage, (page) => {
  if (page && !deckPageLayer) {
    deckPageLayer = pushLayer(() => closeDeckPage(), claimZIndex("middle"));
    return;
  }
  if (!page && deckPageLayer) {
    layerStack.remove(deckPageLayer);
    deckPageLayer = null;
  }
});

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
let postFormLayer: LayerEntry | null = null;

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
  closePostForm();
  postFormRequest.value = { ...opts, id: ++nextRequestId };
  postFormLayer = pushLayer(closePostForm, claimZIndex("high"));
}

/** The shell's compose entry. Same command as openPostForm, named the way the
 * ported application shell asks for it. */
export function openComposer(opts: PostFormOptions = {}): void {
  openPostForm(opts);
}

function closePostForm(): void {
  const active = postFormRequest.value;
  postFormRequest.value = null;
  if (postFormLayer) {
    layerStack.remove(postFormLayer);
    postFormLayer = null;
  }
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
    closePostForm();
    postFormRequest.value = {
      id: ++nextRequestId,
      reply: props.reply,
      renote: props.renote,
      resolve,
      onClosed,
    };
    postFormLayer = pushLayer(closePostForm, claimZIndex("high"));
  });
}

// ---------------------------------------------------------------------------
// Emoji picker and reaction picker
// ---------------------------------------------------------------------------

// Mirrors the sizes EmojiPicker's stylesheet caps itself at, so the popover can
// be positioned without mounting it first.
const PICKER_WIDTH = 336;
const PICKER_HEIGHT = 420;
const PICKER_GAP = 8;

/** Where the picker sits: above the anchor when there is room, below it
 * otherwise, always inside the viewport. Shared by the reaction and annotation
 * entries so both popovers land the same way. */
function pickerPosition(anchor?: Element | null): { left: number; top: number } {
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
  return { left, top };
}

export interface EmojiPickerOptions {
  /** Reaction mode: the picker is titled and called the same way, but the
   * caller consumes the emoji instead of having it inserted into a field. */
  asReaction?: boolean;
}

/** Insert `text` at the caret of an input or textarea, keeping the field the
 * focused element and firing `input` so a v-model updates. */
function insertTextAtCursor(field: HTMLTextAreaElement | HTMLInputElement, text: string): void {
  field.focus();
  const start = field.selectionStart ?? field.value.length;
  const end = field.selectionEnd ?? start;
  if (typeof field.setRangeText === "function") {
    field.setRangeText(text, start, end, "end");
    // A textarea's value is plain state; re-dispatch `input` so the component
    // that owns it sees the change exactly like a keystroke.
    field.dispatchEvent(new Event("input", { bubbles: true }));
    return;
  }
  // setRangeText is missing on a few input types; fall back to the append form.
  field.value = `${field.value.slice(0, start)}${text}${field.value.slice(end)}`;
  field.dispatchEvent(new Event("input", { bubbles: true }));
}

let pickerLayer: LayerHandle | null = null;

function closePicker(): void {
  if (!pickerLayer) return;
  const handle = pickerLayer;
  pickerLayer = null;
  handle.dispose();
}

function openPickerLayer(
  anchor: Element | null | undefined,
  onSelect: (emoji: string) => void,
): void {
  closePicker();
  let handle!: LayerHandle;
  handle = openLayer(
    EmojiPicker,
    {},
    {
      priority: "middle",
      position: pickerPosition(anchor),
      events: {
        select: (emoji: string) => {
          handle.dispose();
          onSelect(emoji);
        },
        close: () => handle.dispose(),
      },
      onClose: () => {
        if (pickerLayer === handle) pickerLayer = null;
      },
    },
  );
  pickerLayer = handle;
}

/** Open the single reaction picker. The chosen emoji goes to `onSelect`;
 * `anchor` places the popover above (or below) the button that opened it.
 * Opening it again replaces the previous request. */
export function openReactionPicker(onSelect: (emoji: string) => void, anchor?: Element | null): void {
  openPickerLayer(anchor, onSelect);
}

/**
 * Firefish's `os.openEmojiPicker(src, opts, initialTextarea)`: a Unicode emoji
 * picker anchored to `src` that inserts the chosen emoji at the caret of
 * `initialTextarea`.
 *
 * Cinderfish has no custom emoji catalogue (`POST /api/meta` carries no
 * `emojis` field), so the picker is the Unicode set the reaction picker
 * already uses. Firefish's global "remember the last focused textarea" focus
 * listener and MutationObserver are deliberately not ported: the call site owns
 * its field, and a hidden document-wide listener is a leak, not a feature.
 */
export function openEmojiPicker(
  src?: HTMLElement | null,
  opts: EmojiPickerOptions = {},
  initialTextarea?: HTMLTextAreaElement | HTMLInputElement | null,
): Promise<void> {
  const target = initialTextarea ?? null;
  if (!target) {
    console.warn("os.openEmojiPicker: no target field was given, the picked emoji is discarded");
  }
  openPickerLayer(src ?? null, (emoji) => {
    if (!target) return;
    insertTextAtCursor(target, emoji);
  });
  // `asReaction` is accepted for signature compatibility with the ported call
  // sites; the picker is the same Unicode set either way, so the only thing it
  // can change is how the caller reads the result.
  void opts;
  return Promise.resolve();
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
    // Only the head is on screen; the rest queue behind it. Registering the
    // head here and again when the queue advances keeps Escape's target and the
    // visible dialog the same layer.
    if (dialogQueue.value.length === 1) registerDialogLayer(request.id);
  });
}

const dialogLayers = new Map<number, LayerEntry>();

function registerDialogLayer(id: number): void {
  if (dialogLayers.has(id)) return;
    dialogLayers.set(
    id,
    pushLayer(
      () => resolveDialog(id, { canceled: true }),
      claimZIndex("high"),
    ),
  );
}

function resolveDialog(id: number, outcome: DialogOutcome): void {
  const [head, ...rest] = dialogQueue.value;
  if (!head || head.id !== id) return;
  dialogQueue.value = rest;
  const entry = dialogLayers.get(id);
  if (entry) {
    dialogLayers.delete(id);
    layerStack.remove(entry);
  }
  head.resolve(outcome);
  if (rest[0]) registerDialogLayer(rest[0].id);
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
let menuLayer: LayerEntry | null = null;

function enqueueMenu(
  items: MenuItem[] | Ref<MenuItem[]>,
  src: HTMLElement | null,
  placement: MenuPlacement,
): Promise<void> {
  return new Promise((resolve) => {
    // One popover at a time: opening a second menu closes and resolves the first.
    closeMenu();
    const zIndex = claimZIndex("high");
    const entry = pushLayer(() => {
      const active = menuRequest.value;
      if (!active) return;
      menuRequest.value = null;
      menuLayer = null;
      active.resolve();
    }, zIndex);
    menuRequest.value = { id: ++nextRequestId, items, placement, src, resolve };
    menuLayer = entry;
  });
}

function closeMenu(): void {
  const active = menuRequest.value;
  if (menuLayer) {
    layerStack.remove(menuLayer);
    menuLayer = null;
  }
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
// Page windows
// ---------------------------------------------------------------------------

/** Open `path` in a deck page window. Returns false when there is no window
 * for it: the main router's catch-all resolves every string as `not-found`, so
 * an unknown path must be refused explicitly instead of raising a 404 window. */
function openPageWindow(path: string): boolean {
  const resolved = router.resolve(path);
  const name = typeof resolved.name === "string" ? resolved.name : null;
  if (!name || name === "not-found") {
    console.warn(`os.pageWindow: ${path} is not a Cinderfish route`);
    return false;
  }
  const page = deckPageFor(name, resolved.params);
  if (page) {
    openDeckPage(page);
    return true;
  }
  // Settings, admin and the timelines have no standalone window view: their
  // components read `useRoute()`, and a second router is out of scope (the
  // contract keeps one router tree). Navigating the main surface is the
  // documented degradation; the deck preference keeps the URL, so returning to
  // a deck route restores the columns.
  void router.push(resolved.fullPath);
  return true;
}

/** Firefish's `os.pageWindow(path)`: a note or a profile opens in the deck's
 * page window; any other route is navigated. */
export function pageWindow(path: string): void {
  openPageWindow(path);
}

/** Firefish's `os.modalPageWindow(path)`. Cinderfish's PageWindow is always
 * modal (backdrop + aria-modal), so this is the same command; the difference
 * Firefish draws between a draggable window and a modal one is not offered. */
export function modalPageWindow(path: string): void {
  openPageWindow(path);
}

// ---------------------------------------------------------------------------
// Overlay
// ---------------------------------------------------------------------------

// One document-capture listener owns Escape for every layer. Capture runs
// before component handlers, and stopImmediatePropagation keeps the listeners
// other components registered on document (nav drawer, widgets drawer, note
// menus) from closing a second layer in the same key press.
function handleEscape(event: KeyboardEvent): void {
  if (event.key !== "Escape") return;
  if (!layerStack.closeTop()) return;
  event.preventDefault();
  event.stopImmediatePropagation();
}

const backdropStyle: CSSProperties = {
  position: "fixed",
  inset: 0,
  zIndex: 1000,
  background: "rgba(0, 0, 0, 0.5)",
};

function postFormTitle(opts: PostFormOptions): string {
  if (opts.reply) return t("reply");
  if (opts.renote) return t("quote");
  return t("toPost");
}

function renderGenericLayer(layer: GenericLayer) {
  const props = { ...resolveLayerProps(layer), ...layer.eventProps };
  const style: CSSProperties = layer.position
    ? {
        position: "fixed",
        left: `${layer.position.left}px`,
        top: `${layer.position.top}px`,
        zIndex: layer.zIndex,
      }
    : { position: "fixed", inset: 0, zIndex: layer.zIndex, pointerEvents: "none" };
  return h(
    "div",
    {
      key: `layer-${layer.id}`,
      class: layer.modal ? "os-layer os-layer--modal" : "os-layer",
      style,
      // Only a click on the backdrop itself closes: clicks inside the component
      // bubble to this wrapper, and closing on those would make every modal
      // layer unusable.
      onClick: layer.modal
        ? (event: MouseEvent) => {
            if (event.target === event.currentTarget) layer.dispose();
          }
        : undefined,
    },
    [h(layer.component, props)],
  );
}

/** The singleton overlay container. Mount it once in App.vue; it renders
 * nothing until a command opens a layer. Dialogs and menus own their own
 * backdrop and z-index (claimZIndex), the generic layers get theirs from
 * openLayer, and the composer keeps the backdrop below. */
export const OsOverlay = defineComponent({
  name: "OsOverlay",
  setup() {
    document.addEventListener("keydown", handleEscape, true);
    onUnmounted(() => document.removeEventListener("keydown", handleEscape, true));

    return () => {
      const dialog = currentDialog.value;
      const menu = menuRequest.value;
      const form = postFormRequest.value;
      const layers = genericLayers.value;
      if (!dialog && !menu && !form && !layers.length) return null;
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
        ...layers.map(renderGenericLayer),
        form
          ? h("div", {
              key: "backdrop",
              style: backdropStyle,
              onClick: () => closePostForm(),
            })
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
