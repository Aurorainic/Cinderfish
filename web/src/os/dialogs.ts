import { ref } from "vue";
import { t } from "../i18n";
import CfDialog from "./CfDialog.vue";
import CfPopupMenu from "./CfPopupMenu.vue";
import CfToast from "./CfToast.vue";
import CfWaitingDialog from "./CfWaitingDialog.vue";
import { popup } from "./state";
import type { DialogInput, DialogProps, DialogResult, DialogType, MenuItem } from "./types";

export interface AlertProps {
	type?: DialogType;
	title?: string | null;
	text?: string | null;
	isPlaintext?: boolean;
}

export interface ConfirmProps extends AlertProps {
	okText?: string;
	cancelText?: string;
}

export interface InputTextProps {
	type?: "text" | "email" | "password" | "url" | "search";
	title?: string | null;
	text?: string | null;
	placeholder?: string | null;
	autocomplete?: string;
	default?: string | null;
	minLength?: number;
	maxLength?: number;
	isPlaintext?: boolean;
}

export type InputResult =
	| { canceled: true; result?: undefined }
	| { canceled: false; result: string };

export type NumberInputResult =
	| { canceled: true; result?: undefined }
	| { canceled: false; result: number };

export type DateInputResult =
	| { canceled: true; result?: undefined }
	| { canceled: false; result: Date };

function dialogEvents(resolve: (result: DialogResult) => void) {
	return {
		done: (result: DialogResult) => resolve(result),
	};
}

export function alert(props: AlertProps): Promise<void> {
	return new Promise((resolve) => {
		const text =
			props.text == null && props.type === "error"
				? t("somethingHappened")
				: props.text;
		popup(
			CfDialog,
			{ ...props, text },
			{
				done: () => resolve(),
			},
			"closed",
		);
	});
}

export function confirm(props: ConfirmProps): Promise<DialogResult> {
	return new Promise((resolve) => {
		popup(
			CfDialog,
			{ ...props, showCancelButton: true },
			dialogEvents(resolve),
			"closed",
		);
	});
}

export function yesno(props: AlertProps): Promise<DialogResult> {
	return new Promise((resolve) => {
		popup(
			CfDialog,
			{ ...props, showCancelButton: true, isYesNo: true },
			dialogEvents(resolve),
			"closed",
		);
	});
}

export function inputText(props: InputTextProps): Promise<InputResult> {
	return new Promise((resolve) => {
		const input: DialogInput = {
			type: props.type ?? "text",
			placeholder: props.placeholder,
			autocomplete: props.autocomplete,
			default: props.default,
			minLength: props.minLength,
			maxLength: props.maxLength,
		};
		popup(
			CfDialog,
			{
				title: props.title,
				text: props.text,
				input,
				isPlaintext: props.isPlaintext,
			},
			{
				done: (result: DialogResult) => {
					if (result.canceled) {
						resolve({ canceled: true });
					} else {
						resolve({ canceled: false, result: String(result.result ?? "") });
					}
				},
			},
			"closed",
		);
	});
}

export function prompt(props: InputTextProps): Promise<InputResult> {
	return inputText(props);
}

export function inputParagraph(props: {
	title?: string | null;
	text?: string | null;
	placeholder?: string | null;
	default?: string | null;
}): Promise<InputResult> {
	return new Promise((resolve) => {
		popup(
			CfDialog,
			{
				title: props.title,
				text: props.text,
				input: {
					type: "paragraph",
					placeholder: props.placeholder,
					default: props.default,
				},
			},
			{
				done: (result: DialogResult) => {
					if (result.canceled) {
						resolve({ canceled: true });
					} else {
						resolve({ canceled: false, result: String(result.result ?? "") });
					}
				},
			},
			"closed",
		);
	});
}

export function inputNumber(props: {
	title?: string | null;
	text?: string | null;
	placeholder?: string | null;
	default?: number | null;
	autocomplete?: string;
}): Promise<NumberInputResult> {
	return new Promise((resolve) => {
		popup(
			CfDialog,
			{
				title: props.title,
				text: props.text,
				input: {
					type: "number",
					placeholder: props.placeholder,
					autocomplete: props.autocomplete,
					default: props.default,
				},
			},
			{
				done: (result: DialogResult) => {
					if (result.canceled) {
						resolve({ canceled: true });
					} else {
						resolve({
							canceled: false,
							result: Number(result.result ?? 0),
						});
					}
				},
			},
			"closed",
		);
	});
}

export function inputDate(props: {
	title?: string | null;
	text?: string | null;
	placeholder?: string | null;
	default?: Date | string | null;
}): Promise<DateInputResult> {
	const fallback = props.default ?? new Date();
	return new Promise((resolve) => {
		popup(
			CfDialog,
			{
				title: props.title,
				text: props.text,
				input: {
					type: "date",
					placeholder: props.placeholder,
					default:
						fallback instanceof Date
							? fallback.toISOString().slice(0, 10)
							: fallback,
				},
			},
			{
				done: (result: DialogResult) => {
					if (result.canceled || result.result == null) {
						resolve({ canceled: true });
					} else {
						resolve({
							canceled: false,
							result: new Date(String(result.result)),
						});
					}
				},
			},
			"closed",
		);
	});
}

export interface WaitingHandle {
	close: () => void;
	success: () => void;
}

/** Non-blocking waiting indicator; call `close()` or `success()`. */
export function waiting(text?: string): WaitingHandle {
	const showing = ref(true);
	const success = ref(false);

	popup(
		CfWaitingDialog,
		{ success, showing, text: text ?? null },
		{},
		"closed",
	);

	return {
		close: () => {
			showing.value = false;
		},
		success: () => {
			success.value = true;
			window.setTimeout(() => {
				showing.value = false;
			}, 1000);
		},
	};
}

export function toast(message: string, type: "info" | "success" | "error" = "info"): void {
	popup(CfToast, { message, type }, {}, "closed");
}

export function success(message: string = t("done")): void {
	toast(message, "success");
}

export function error(message: string = t("somethingHappened")): void {
	toast(message, "error");
}

/**
 * Show a waiting dialog while `promise` settles. Firefish uses this for every
 * mutation; the returned promise is the original one so callers can await it.
 */
export function promiseDialog<T>(
	promise: Promise<T>,
	onSuccess?: ((result: T) => void) | null,
	onFailure?: ((error: unknown) => void) | null,
	text?: string,
): Promise<T> {
	const handle = waiting(text);

	promise.then(
		(result) => {
			if (onSuccess != null) {
				handle.close();
				onSuccess(result);
			} else {
				handle.success();
			}
		},
		(cause: unknown) => {
			handle.close();
			if (onFailure != null) {
				onFailure(cause);
			} else {
				const message =
					cause != null && typeof cause === "object" && "message" in cause
						? String((cause as { message?: unknown }).message)
						: String(cause);
				void alert({ type: "error", text: message });
			}
		},
	);

	return promise;
}

export const awaitDialog = promiseDialog;

/** Open a context menu at the pointer position. */
export function popupMenu(
	items: MenuItem[],
	point: MouseEvent | { x: number; y: number },
	_options: { width?: number; onClosing?: () => void; align?: string } = {},
): Promise<void> {
	const x = point instanceof MouseEvent ? point.clientX : point.x;
	const y = point instanceof MouseEvent ? point.clientY : point.y;
	return new Promise((resolve) => {
		popup(
			CfPopupMenu,
			{ items, x, y },
			{
				closed: () => resolve(),
			},
			"closed",
		);
	});
}

export function contextMenu(items: MenuItem[], event: MouseEvent): Promise<void> {
	return popupMenu(items, event);
}

export type { DialogProps, DialogResult, DialogType, MenuItem };
