/** Menu item shape shared by `popupMenu`, `CfA` and `CfSelect`. */
export interface MenuItem {
	type?: "label" | "button" | "a" | "link" | "parent" | null;
	text?: string;
	icon?: string;
	active?: boolean;
	action?: () => void;
	to?: string;
	href?: string;
	separator?: boolean;
	children?: MenuItem[];
}

/** Returned by `confirm`, `yesno` and dialogs with a cancel button. */
export interface DialogResult {
	canceled: boolean;
	result?: unknown;
}

export type DialogType =
	| "error"
	| "info"
	| "success"
	| "warning"
	| "waiting"
	| "question";

export interface DialogInput {
	type?: "text" | "email" | "password" | "url" | "search" | "number" | "paragraph" | "date";
	placeholder?: string | null;
	autocomplete?: string;
	default?: string | number | null;
	minLength?: number;
	maxLength?: number;
}

export interface DialogProps {
	type?: DialogType;
	title?: string | null;
	text?: string | null;
	isPlaintext?: boolean;
	showCancelButton?: boolean;
	isYesNo?: boolean;
	okText?: string;
	cancelText?: string;
	input?: DialogInput | null;
}
