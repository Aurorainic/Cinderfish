// Ported from firefish@b425f61:packages/client/src/scripts/select-file.ts (AGPL-3.0-only)
//
// Wave-2 differences from upstream (contract §5):
//   * "from URL" calls `drive/files/upload-from-url` and uses its response
//     directly: upstream waits for the `urlUploadFinished` stream event, which
//     this instance does not publish because the endpoint already answers with
//     the created file;
//   * recompression happens inside `uploadFile` (scripts/image-compress.ts), so
//     the per-upload switch upstream shows is redundant here;
//   * the drive picker is our own MkDriveSelectDialog instead of
//     `os.selectDriveFile`; selecting *folders* is not needed by this wave and
//     `drive/folders` has no "choose" surface, so only files are selectable;
//   * a dismissed menu resolves with an empty array instead of never settling.
//
// Signature fixed by the wave-2 contract for the PostForm port:
//   selectFiles(opts?: { multiple?: boolean; label?: string }): Promise<DriveFile[]>
import { alert, inputText, popup, popupMenu, success, type MenuItem } from "@/os";
import { api, ApiError } from "@/api/client";
import type { DriveFile } from "@/api/types";
import { defaultStore } from "@/store/base";
import { t } from "@/i18n";
import { uploadFile } from "@/scripts/upload";
import MkDriveSelectDialog from "@/components/MkDriveSelectDialog.vue";

export interface SelectFilesOptions {
	multiple?: boolean;
	/** Optional heading shown above the menu items. */
	label?: string;
	/**
	 * Where the menu opens. PostForm passes the button event; plain calls get
	 * a viewport-relative fallback.
	 */
	src?: MouseEvent | { x: number; y: number };
}

function fallbackPoint(): { x: number; y: number } {
	if (typeof window === "undefined") return { x: 0, y: 0 };
	return { x: Math.round(window.innerWidth / 2), y: Math.round(window.innerHeight / 3) };
}

/** Opens the drive picker and resolves with the chosen files (possibly none). */
function chooseFromDrive(multiple: boolean): Promise<DriveFile[]> {
	return new Promise((resolve) => {
		popup(
			MkDriveSelectDialog,
			{ multiple, type: "file" },
			{
				done: (files?: DriveFile[]) => {
					resolve(files == null ? [] : files);
				},
			},
			"closed",
		);
	});
}

/** Opens the OS file chooser and uploads everything that was picked. */
function chooseFromPc(multiple: boolean): Promise<DriveFile[]> {
	return new Promise((resolve) => {
		const input = document.createElement("input");
		input.type = "file";
		input.multiple = multiple;
		input.onchange = () => {
			const picked = input.files == null ? [] : Array.from(input.files);
			if (picked.length === 0) {
				resolve([]);
				return;
			}
			const folderId = defaultStore.state.uploadFolder;
			Promise.all(picked.map((file) => uploadFile(file, { folderId })))
				.then((driveFiles) => resolve(multiple ? driveFiles : driveFiles.slice(0, 1)))
				// uploadFile already showed its own dialog; never double-alert.
				.catch(() => resolve([]));
		};
		// iOS Safari only opens the picker when the input is still referenced.
		(window as unknown as { __cinderfish_input_ref__?: HTMLInputElement })
			.__cinderfish_input_ref__ = input;
		input.click();
	});
}

/**
 * Asks for a URL and lets the server fetch it. The endpoint answers with the
 * created file, so there is nothing to wait for beyond the request itself.
 */
async function chooseFromUrl(): Promise<DriveFile[]> {
	const { canceled, result } = await inputText({
		title: t("uploadFromUrl"),
		text: t("uploadFromUrlDescription"),
		placeholder: "https://",
	});
	const url = canceled || result == null ? "" : result.trim();
	if (url === "") return [];

	try {
		const file = await api("drive/files/upload-from-url", {
			url,
			folderId: defaultStore.state.uploadFolder,
		});
		void success();
		return [file];
	} catch (cause) {
		const text =
			cause instanceof ApiError && cause.message !== ""
				? cause.message
				: t("failedToUpload");
		void alert({ type: "error", title: t("failedToUpload"), text });
		return [];
	}
}

export function selectFiles(options: SelectFilesOptions = {}): Promise<DriveFile[]> {
	const multiple = options.multiple === true;

	return new Promise((resolve) => {
		let settled = false;
		let actionTaken = false;

		const finish = (files: DriveFile[]): void => {
			if (settled) return;
			settled = true;
			resolve(files);
		};

		const items: MenuItem[] = [];
		if (options.label != null) {
			items.push({ type: "label", text: options.label });
		}
		items.push({
			text: t("upload"),
			icon: "ph-upload-simple ph-bold ph-lg",
			action: () => {
				actionTaken = true;
				void chooseFromPc(multiple).then(finish);
			},
		});
		items.push({
			text: t("fromDrive"),
			icon: "ph-cloud ph-bold ph-lg",
			action: () => {
				actionTaken = true;
				void chooseFromDrive(multiple).then(finish);
			},
		});
		items.push({
			text: t("uploadFromUrl"),
			icon: "ph-link ph-bold ph-lg",
			action: () => {
				actionTaken = true;
				void chooseFromUrl().then(finish);
			},
		});

		// CfPopupMenu runs the item action right after closing the menu, so the
		// menu-close callback sees actionTaken already set.
		void popupMenu(items, options.src ?? fallbackPoint()).then(() => {
			if (!actionTaken) finish([]);
		});
	});
}

/** Single-file convenience wrapper. */
export function selectFile(
	options: Omit<SelectFilesOptions, "multiple"> = {},
): Promise<DriveFile | null> {
	return selectFiles(options).then((files) => files[0] ?? null);
}
