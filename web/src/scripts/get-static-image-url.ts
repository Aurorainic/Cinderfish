// Ported from firefish@b425f61:packages/client/src/scripts/get-static-image-url.ts (AGPL-3.0-only)

/**
 * Firefish rewrites an image URL to the media proxy (`/proxy/<host><path>?static=1`)
 * so that animated GIF/WebP files are served as a still frame.
 *
 * Cinderfish has no media proxy (`webpublicUrl` stays null) and generates its
 * own thumbnails server-side (`thumbnailUrl`, wave 46), so there is no still
 * frame to request here: the function is kept for import parity and returns
 * the URL it was given. `null` is accepted because the thumbnail fields are
 * nullable.
 */
export function getStaticImageUrl(
	baseUrl: string | null | undefined,
): string {
	if (baseUrl == null) return "";
	return baseUrl;
}
