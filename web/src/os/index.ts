import { api } from "../api/client";
import { ApiError } from "../api/errors";
import { alert } from "./dialogs";

// Firefish's `os.api` surface: ported call sites write `os.api("endpoint", params)`.
// Keep the alias here so the 300+ upstream call sites need no rewrite.
export {
  api,
  apiForm,
  apiGet,
  pendingApiRequestsCount,
  resolveMediaUrl,
} from "../api/client";

export { CfDialog, CfModal, CfPopupHost, CfPopupMenu, CfToast, CfWaitingDialog } from "./components";
export {
	alert,
	awaitDialog,
	confirm,
	contextMenu,
	error,
	inputDate,
	inputNumber,
	inputParagraph,
	inputText,
	prompt,
	promiseDialog,
	popupMenu,
	success,
	toast,
	waiting,
	yesno,
	type AlertProps,
	type ConfirmProps,
	type DateInputResult,
	type InputResult,
	type InputTextProps,
	type NumberInputResult,
	type WaitingHandle,
} from "./dialogs";
export {
	claimZIndex,
	getUniqueId,
	popup,
	popups,
	type PopupEntry,
	type PopupHandle,
	type PopupPriority,
} from "./state";
export type { DialogInput, DialogProps, DialogResult, DialogType, MenuItem } from "./types";

/**
 * Firefish's `apiWithDialog`: run a mutation and surface failures as an error
 * dialog instead of rejecting into an unhandled promise.
 */
export function apiWithDialog<T>(
	endpoint: string,
	params: Record<string, unknown> = {},
	token?: string | null,
): Promise<T> {
	const promise = api<T>(endpoint, params, token);
	void promise.catch((cause: unknown) => {
		const text =
			cause instanceof ApiError
				? `${cause.message}\n${cause.errorId ?? cause.code ?? ""}`.trim()
				: String(cause);
		void alert({ type: "error", text });
	});
	return promise;
}