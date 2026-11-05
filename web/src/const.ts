// Ported from firefish@b425f61:packages/client/src/const.ts (AGPL-3.0-only)

// File types the browser is allowed to render directly. Anything else is
// served as application/octet-stream. SVG is deliberately absent: it would
// give remote users a script execution vector.
export const FILE_TYPE_BROWSERSAFE: readonly string[] = [
	// Images
	"image/png",
	"image/gif",
	"image/jpeg",
	"image/webp",
	"image/apng",
	"image/bmp",
	"image/tiff",
	"image/x-icon",
	"image/avif",

	// OggS
	"audio/opus",
	"video/ogg",
	"audio/ogg",
	"application/ogg",

	// ISO/IEC base media file format
	"video/quicktime",
	"video/mp4",
	"audio/mp4",
	"video/x-m4v",
	"audio/x-m4a",
	"video/3gpp",
	"video/3gpp2",

	"video/mpeg",
	"audio/mpeg",

	"video/webm",
	"audio/webm",

	"audio/aac",
	"audio/x-flac",
	"audio/vnd.wave",
];

/** Images that can be shown inline (subset of the list above). */
export const FILE_TYPE_IMAGE: readonly string[] = [
	"image/png",
	"image/gif",
	"image/jpeg",
	"image/webp",
	"image/apng",
	"image/bmp",
	"image/tiff",
	"image/x-icon",
	"image/avif",
];

/** Video containers the timeline renders with a native video element. */
export const FILE_TYPE_VIDEO: readonly string[] = [
	"video/quicktime",
	"video/mp4",
	"video/x-m4v",
	"video/3gpp",
	"video/3gpp2",
	"video/mpeg",
	"video/webm",
	"video/ogg",
];
