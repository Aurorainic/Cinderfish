// Ported from firefish@b425f61:packages/client/src/scripts/check-word-mute.ts (AGPL-3.0-only)
//
// The muted-word list itself is a client setting (defaultStore.state.mutedWords),
// so this stays faithful: Cinderfish has no server-side word mute, but the local
// list is honoured exactly like in Firefish.

export type Muted = {
	muted: boolean;
	matched: string[];
	what?: "note" | "reply" | "renote" | "quote";
};

interface MuteUser {
	id: string;
}

interface MuteFile {
	comment?: string | null;
}

interface MuteNote {
	cw?: string | null;
	text?: string | null;
	files?: MuteFile[] | null;
	userId?: string;
	renote?: MuteNote | null;
	reply?: MuteNote | null;
}

const NotMuted: Muted = { muted: false, matched: [] };

function checkWordMute(
	note: MuteNote,
	mutedWords: (string | string[])[],
): Muted {
	let text = `${note.cw ?? ""} ${note.text ?? ""}`;
	if (note.files != null) {
		text += ` ${note.files.map((f) => f.comment ?? "").join(" ")}`;
	}
	text = text.trim();

	if (text === "") return NotMuted;

	const result: Muted = { muted: false, matched: [] };

	for (const mutePattern of mutedWords) {
		if (Array.isArray(mutePattern)) {
			// Clean up
			const keywords = mutePattern.filter((keyword) => keyword !== "");

			if (
				keywords.length > 0 &&
				keywords.every((keyword) => text.includes(keyword))
			) {
				result.muted = true;
				result.matched.push(...keywords);
			}
		} else {
			// represents RegExp
			const regexp = mutePattern.match(/^\/(.+)\/(.*)$/);

			// This should never happen due to input sanitisation.
			if (!regexp) {
				console.warn(`Found invalid regex in word mutes: ${mutePattern}`);
				continue;
			}

			try {
				if (new RegExp(regexp[1], regexp[2]).test(text)) {
					result.muted = true;
					result.matched.push(mutePattern);
				}
			} catch {
				// This should never happen due to input sanitisation.
			}
		}
	}

	result.matched = [...new Set(result.matched)];
	return result;
}

export function getWordSoftMute(
	note: MuteNote,
	me: MuteUser | null | undefined,
	mutedWords: (string | string[])[],
): Muted {
	// Yourself
	if (me && note.userId === me.id) {
		return NotMuted;
	}

	if (mutedWords.length > 0) {
		const noteMuted = checkWordMute(note, mutedWords);
		if (noteMuted.muted) {
			noteMuted.what = "note";
			return noteMuted;
		}

		if (note.renote) {
			const renoteMuted = checkWordMute(note.renote, mutedWords);
			if (renoteMuted.muted) {
				renoteMuted.what = note.text == null ? "renote" : "quote";
				return renoteMuted;
			}
		}

		if (note.reply) {
			const replyMuted = checkWordMute(note.reply, mutedWords);
			if (replyMuted.muted) {
				replyMuted.what = "reply";
				return replyMuted;
			}
		}
	}

	return NotMuted;
}
