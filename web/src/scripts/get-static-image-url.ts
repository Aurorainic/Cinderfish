// Ported from firefish@b425f61:packages/client/src/scripts/get-static-image-url.ts (AGPL-3.0-only)

/**
 * Firefish rewrites an image URL to the media proxy (`/proxy/<host><path>?static=1`)
 * so that animated GIF/WebP files are served as a still frame.
 *
 * Cinderfish has no media proxy and no thumbnail pipeline
 * (docs/research/api-contract/drive-media.md: `thumbnailUrl` and `webpublicUrl`
 * are always null, files are served from their own `url`), so the function is
 * kept for import parity but returns the original URL. `null` is accepted
 * because every thumbnail URL field in the API contract is nullable.
 */
export function getStaticImageUrl(
	baseUrl: string | null | undefined,
): string {
	if (baseUrl == null) return "";
	return baseUrl;
}
