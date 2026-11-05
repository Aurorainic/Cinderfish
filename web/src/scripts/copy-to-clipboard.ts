// Ported from firefish@b425f61:packages/client/src/scripts/copy-to-clipboard.ts (AGPL-3.0-only)
//
// Firefish used a hidden <pre> + document.execCommand("copy") because the
// Clipboard API needs a secure context and a user gesture. Cinderfish prefers
// navigator.clipboard and keeps the legacy path as a fallback, so the call
// still works on plain-HTTP instances. The default export keeps Firefish's
// call shape: copyToClipboard("text").

function legacyCopy(value: string): boolean {
	const root = document.createElement("div");
	const pre = document.createElement("pre");
	// A parent with user-select: none would make the copy selection empty.
	pre.style.userSelect = "auto";
	pre.textContent = value;
	root.appendChild(pre);
	const style = root.style;
	style.position = "fixed";
	style.right = "200%";
	document.body.appendChild(root);
	let ok = false;
	try {
		const selection = document.getSelection();
		selection?.selectAllChildren(root);
		ok = document.execCommand("copy");
	} catch {
		ok = false;
	} finally {
		document.body.removeChild(root);
	}
	return ok;
}

export default function copyToClipboard(value: string): Promise<boolean> {
	if (
		typeof navigator !== "undefined" &&
		navigator.clipboard != null &&
		typeof navigator.clipboard.writeText === "function"
	) {
		return navigator.clipboard
			.writeText(value)
			.then(() => true)
			.catch(() => legacyCopy(value));
	}
	return Promise.resolve(legacyCopy(value));
}

export { copyToClipboard };
