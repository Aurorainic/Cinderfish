// os.ts is the imperative UI command bus: commands like openPostForm and
// confirmDialog are called from anywhere, and the singleton overlay container
// (OsOverlay, mounted once in App.vue) renders the resulting layers.
import { computed, defineComponent, h, onUnmounted, ref, type CSSProperties } from "vue";
import type { Note } from "./api/types";
import PageWindow from "./components/ui/PageWindow.vue";

let nextRequestId = 0;

type PostFormOptions = { reply?: Note; renote?: Note; quote?: boolean };
type PostFormRequest = PostFormOptions & { id: number };

const postFormRequest = ref<PostFormRequest | null>(null);

type ConfirmRequest = { id: number; message: string; resolve: (ok: boolean) => void };

const confirmQueue = ref<ConfirmRequest[]>([]);

const currentConfirm = computed(() => confirmQueue.value[0] ?? null);

/** Open the global post form. F9 swaps the stub panel for the real PostForm;
 * the signature is frozen here. */
export function openPostForm(opts: PostFormOptions = {}): void {
  postFormRequest.value = { ...opts, id: ++nextRequestId };
}

function closePostForm(): void {
  postFormRequest.value = null;
}

/** Ask for confirmation. Returns true when the user confirms, false when they
 * cancel or dismisses it with Escape. Concurrent dialogs queue FIFO. */
export function confirmDialog(message: string): Promise<boolean> {
  return new Promise((resolve) => {
    confirmQueue.value = [
      ...confirmQueue.value,
      { id: ++nextRequestId, message, resolve },
    ];
  });
}

function resolveConfirm(ok: boolean): void {
  const request = confirmQueue.value[0];
  if (!request) return;
  confirmQueue.value = confirmQueue.value.slice(1);
  request.resolve(ok);
}

function handleEscape(): void {
  // Dismiss the topmost layer only.
  if (currentConfirm.value) {
    resolveConfirm(false);
    return;
  }
  if (postFormRequest.value) closePostForm();
}

const overlayStyle: CSSProperties = {
  position: "fixed",
  inset: 0,
  zIndex: 1100,
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
};

const backdropStyle: CSSProperties = {
  position: "fixed",
  inset: 0,
  background: "rgba(0, 0, 0, 0.5)",
};

const confirmBoxStyle: CSSProperties = {
  position: "relative",
  width: "420px",
  maxWidth: "calc(100vw - 32px)",
  background: "var(--panel)",
  border: "1px solid var(--border)",
  borderRadius: "var(--radius-lg)",
  boxShadow: "var(--shadow-sm, 0 4px 16px rgba(0, 0, 0, 0.3))",
  padding: "20px",
};

const confirmMessageStyle: CSSProperties = {
  margin: "0 0 16px",
  color: "var(--text)",
  lineHeight: 1.6,
  whiteSpace: "pre-wrap",
};

const confirmActionsStyle: CSSProperties = {
  display: "flex",
  justifyContent: "flex-end",
  gap: "8px",
};

const buttonStyle: CSSProperties = {
  padding: "8px 16px",
  borderRadius: "var(--radius-sm)",
  border: "1px solid var(--border)",
  background: "var(--panel-2)",
  color: "var(--text)",
  cursor: "pointer",
  font: "inherit",
};

const primaryButtonStyle: CSSProperties = {
  ...buttonStyle,
  background: "var(--accent)",
  borderColor: "var(--accent)",
  color: "var(--on-accent)",
  fontWeight: 600,
};

const stubBodyStyle: CSSProperties = {
  padding: "16px",
  color: "var(--muted)",
  lineHeight: 1.7,
};

function contextLines(opts: PostFormOptions): string[] {
  const lines: string[] = [];
  if (opts.reply) lines.push(`回复 @${opts.reply.user.username}`);
  if (opts.renote && opts.quote) lines.push("引用转贴");
  return lines;
}

/** The singleton overlay container. Mount it once in App.vue; it renders
  * nothing until a command opens a layer. */
export const OsOverlay = defineComponent({
  name: "OsOverlay",
  setup() {
    const onKeydown = (event: KeyboardEvent) => {
      if (event.key === "Escape") handleEscape();
    };
    window.addEventListener("keydown", onKeydown);
    onUnmounted(() => window.removeEventListener("keydown", onKeydown));

    return () => {
      const confirm = currentConfirm.value;
      const form = postFormRequest.value;
      if (!confirm && !form) return null;
      return h("div", { style: overlayStyle }, [
        h("div", {
          key: "backdrop",
          style: backdropStyle,
          onClick: () => (confirm ? resolveConfirm(false) : closePostForm()),
        }),
        confirm
          ? h("div", { key: confirm.id, style: confirmBoxStyle, role: "alertdialog", "aria-modal": "true" }, [
              h("p", { style: confirmMessageStyle }, confirm.message),
              h("div", { style: confirmActionsStyle }, [
                h("button", { style: buttonStyle, type: "button", onClick: () => resolveConfirm(false) }, "取消"),
                h("button", { style: primaryButtonStyle, type: "button", onClick: () => resolveConfirm(true) }, "确定"),
              ]),
            ])
          : null,
        form
          ? h(
              PageWindow,
              {
                key: form.id,
                title: "发帖",
                onClose: () => closePostForm(),
              },
              {
                default: () => [
                  h("div", { style: stubBodyStyle }, [
                    ...contextLines(form).map((line) => h("p", { key: line, style: { margin: "0 0 8px" } }, line)),
                    h("p", { style: { margin: 0 } }, "发帖器尚未接入，此为占位面板（任务 F9 替换为真实组件）。"),
                  ]),
                ],
              },
            )
          : null,
      ]);
    };
  },
});
