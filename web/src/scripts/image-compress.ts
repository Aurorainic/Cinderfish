// Port of firefish@b425f61:packages/client/src/scripts/upload.ts's compression
// step (AGPL-3.0-only). Upstream calls `browser-image-resizer`; Cinderfish has
// no such dependency, so the same pipeline (2048 px box, quality 0.85) runs on
// a canvas.
//
// Differences from upstream:
//   * only JPEG and WebP are re-encoded; upstream also converts SVG to PNG,
//     which `createImageBitmap` does not guarantee to decode;
//   * the extension follows the produced MIME type, so a WebP re-encoded as
//     WebP keeps its name and a fallback to JPEG gains `.jpg`.

const MAX_EDGE = 2048;
const QUALITY = 0.85;

/** Types the canvas pipeline re-encodes, with the type it produces. */
const COMPRESSIBLE: Record<string, "image/jpeg" | "image/webp"> = {
	"image/jpeg": "image/jpeg",
	"image/webp": "image/webp",
};

const EXTENSION: Record<string, string> = {
	"image/jpeg": "jpg",
	"image/webp": "webp",
};

function renamed(name: string, type: string): string {
	const base = name.replace(/\.[^./\\]+$/, "");
	return `${base}.${EXTENSION[type] ?? "jpg"}`;
}

/**
 * Re-encode an image for upload. Returns null when the type is not handled, so
 * the caller keeps the original file.
 */
export async function compressImage(file: File): Promise<File | null> {
	const outputType = COMPRESSIBLE[file.type];
	if (outputType == null) return null;

	const bitmap = await createImageBitmap(file);
	try {
		const scale = Math.min(1, MAX_EDGE / Math.max(bitmap.width, bitmap.height));
		const width = Math.max(1, Math.round(bitmap.width * scale));
		const height = Math.max(1, Math.round(bitmap.height * scale));

		const canvas = document.createElement("canvas");
		canvas.width = width;
		canvas.height = height;
		const context = canvas.getContext("2d");
		if (context == null) return null;
		context.drawImage(bitmap, 0, 0, width, height);

		const blob = await new Promise<Blob | null>((resolve) => {
			canvas.toBlob(resolve, outputType, QUALITY);
		});
		if (blob == null) return null;

		return new File([blob], renamed(file.name, outputType), {
			type: blob.type || outputType,
			lastModified: file.lastModified,
		});
	} finally {
		bitmap.close();
	}
}
