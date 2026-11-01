import type { Note } from "../api/types";
import type { NoteUpdatedEvent } from "./streaming-frames";

export type NoteUpdate = NoteUpdatedEvent;

export function applyNoteUpdate(notes: Map<string, Note>, event: NoteUpdate, viewerId?: string): boolean {
  const note = notes.get(event.id);
  if (!note) return false;
  if (event.type === "deleted") {
    notes.delete(event.id);
    return true;
  }
  const body = event.body as { reaction?: unknown; userId?: unknown } | null;
  if (!body || typeof body.reaction !== "string" || typeof body.userId !== "string") return false;
  const current = note.reactions[body.reaction] ?? 0;
  const affectsViewer = viewerId !== undefined && body.userId === viewerId;
  if (event.type === "reacted") {
    note.reactions[body.reaction] = current + 1;
    note.reactionCount += 1;
    if (affectsViewer) note.myReaction = body.reaction;
    return true;
  }
  if (current > 1) note.reactions[body.reaction] = current - 1;
  else if (current === 1) delete note.reactions[body.reaction];
  if (current > 0) note.reactionCount = Math.max(0, note.reactionCount - 1);
  if (affectsViewer && note.myReaction === body.reaction) note.myReaction = undefined;
  return true;
}
