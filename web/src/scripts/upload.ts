// Ported from firefish@b425f61:packages/client/src/scripts/upload.ts (AGPL-3.0-only)
//
// Wave-2 differences from upstream (contract §5, docs/research/api-contract/drive-media.md):
//   * the upload goes through `apiForm` instead of a bare XMLHttpRequest, so the
//     token travels in the Authorization header — `drive/files/create` cannot
//     read the multipart `i` field (the handler authenticates before parsing);
//   * no progress events: the Fetch API reports no upload progress, so
//     `progressMax` / `progressValue` stay undefined until a streaming upload
//     transport exists;
//   * no client-side recompression: `browser-image-resizer` is not a dependency,
//     so the original bytes are always uploaded (the `keepOriginalUploading`
//     setting therefore has no effect here);
//   * `force`, `name`, `comment` and `isSensitive` are not sent: the backend
//     ignores them, and the returned file always carries `isSensitive: false`;
//   * the only size limit is `meta.maxFileSize` (20 MiB, not configurable), the
//     error code is `FILE_TOO_LARGE` instead of upstream's 413.
import { ref, type Ref } from "vue";
import { apiForm, ApiError } from "@/api/client";
import type { DriveFile } from "@/api/types";
import { fetchMeta } from "@/account";
import { alert } from "@/os";
import { t } from "@/i18n";

export interface Uploading {
	id: string;
	name: string;
	progressMax: number | undefined;
	progressValue: number | undefined;
	img: string;
}

/** In-flight uploads, for progress UIs (PostForm in the next wave). */
export const uploads: Ref<Uploading[]> = ref<Uploading[]>([]);

export interface UploadOptions {
	/** Target folder; `null`/`undefined` means the drive root. */
	folderId?: string | null;
	/** Overrides the name shown in the progress list (not the stored name). */
	name?: string;
}

/** Instance upload limit in bytes, or null when meta is unreachable. */
async function maxFileSize(): Promise<number | null> {
	try {
		const meta = await fetchMeta();
		return meta.maxFileSize > 0 ? meta.maxFileSize : null;
	} catch {
		return null;
	}
}

function reportFailure(cause: unknown): void {
	if (cause instanceof ApiError && cause.code === "FILE_TOO_LARGE") {
		void alert({
			type: "error",
			title: t("failedToUpload"),
			text: t("cannotUploadBecauseExceedsFileSizeLimit"),
		});
		return;
	}

	const detail =
		cause instanceof ApiError
			? [cause.message, cause.code, cause.errorId].filter((x) => x != null).join("\n")
			: String(cause);

	void alert({
		type: "error",
		title: t("failedToUpload"),
		text: detail,
	});
}

/**
 * Upload one file and resolve with the created drive file.
 *
 * The promise rejects on every failure path after showing an error dialog;
 * callers that do not want a second alert can ignore the rejection.
 */
export async function uploadFile(
	file: File,
	options: UploadOptions = {},
): Promise<DriveFile> {
	const limit = await maxFileSize();
	if (limit != null && file.size > limit) {
		void alert({
			type: "error",
			title: t("failedToUpload"),
			text: t("cannotUploadBecauseExceedsFileSizeLimit"),
		});
		throw new Error("FILE_TOO_LARGE");
	}

	const id = Math.random().toString();
	const ctx: Uploading = {
		id,
		name: options.name ?? (file.name === "" ? "untitled" : file.name),
		progressMax: undefined,
		progressValue: undefined,
		img: URL.createObjectURL(file),
	};

	uploads.value.push(ctx);

	const formData = new FormData();
	formData.append("file", file);
	if (options.folderId != null && options.folderId !== "") {
		formData.append("folderId", options.folderId);
	}

	try {
		return await apiForm<DriveFile>("drive/files/create", formData);
	} catch (cause) {
		reportFailure(cause);
		throw cause;
	} finally {
		uploads.value = uploads.value.filter((x) => x.id !== id);
		URL.revokeObjectURL(ctx.img);
	}
}
