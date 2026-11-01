// Types mirror the JSON the Go handlers emit. Field names follow the
// Misskey schema, so a misskey-js client and this app see the same shape.

export interface UserLite {
  id: string;
  name: string | null;
  username: string;
  host: string | null;
  avatarUrl: string | null;
  avatarBlurhash: string | null;
  avatarDecorations: unknown[];
  isBot: boolean;
  isCat: boolean;
  requireSigninToViewContents: boolean;
  emojis: unknown[];
  onlineStatus: string;
  badgeRoles: unknown[];
  instance: { name?: string; faviconUrl?: string } | null;
}

export interface UserDetailed extends UserLite {
  url: string;
  uri: string;
  movedTo: string | null;
  alsoKnownAs: string | null;
  createdAt: string;
  updatedAt: string | null;
  bannerUrl: string | null;
  isLocked: boolean;
  isSilenced: boolean;
  isSuspended: boolean;
  // Relationship fields are relative to the authenticated viewer. The
  // backend always emits them (false for an anonymous or invalid-token
  // viewer), but they stay optional so payloads produced before the field
  // existed still typecheck; callers fall back to false.
  isFollowing?: boolean;
  isFollowed?: boolean;
  hasPendingFollowRequestFromYou?: boolean;
  hasPendingFollowRequestToYou?: boolean;
  isModerator?: boolean;
  isAdmin?: boolean;
  description: string | null;
  location: string | null;
  birthday: string | null;
  lang: string | null;
  fields: unknown[];
  verifiedLinks: unknown[];
  followersCount: number;
  followingCount: number;
  notesCount: number;
  pinnedNoteIds: string[];
  pinnedNotes: Note[];
  publicReactions: boolean;
  followersVisibility: string;
  followingVisibility: string;
  memo: string | null;
}

export interface MeDetailed extends UserDetailed {
  avatarId: string | null;
  bannerId: string | null;
  isModerator: boolean;
  isAdmin: boolean;
  email?: string | null;
  emailVerified?: boolean;
  hasUnreadNotification: boolean;
  unreadNotificationsCount: number;
  [key: string]: unknown;
}

export type NoteVisibility = "public" | "home" | "followers" | "specified";

export interface Note {
  id: string;
  createdAt: string;
  userId: string;
  user: UserLite;
  text: string | null;
  cw: string | null;
  visibility: NoteVisibility;
  localOnly: boolean;
  renoteCount: number;
  repliesCount: number;
  reactionCount: number;
  reactions: Record<string, number>;
  reactionEmojis: unknown[];
  emojis: unknown[];
  fileIds: string[];
  files: DriveFile[];
  replyId: string | null;
  renoteId: string | null;
  mentions: string[];
  tags: string[];
  uri: string | null;
  url: string | null;
  visibleUserIds?: string[];
  reply?: Note | null;
  renote?: Note | null;
  myReaction?: string;
  // Omitted by the packer when the note left it unset; Misskey clients treat
  // an absent value like "all".
  reactionAcceptance?: string | null;
  hasPoll?: boolean;
  poll?: Poll;
  clippedCount?: number;
}

export interface Poll {
  multiple: boolean;
  expiresAt: number | null;
  choices: { text: string; votes: number; isVoted: boolean }[];
  votesCount: number;
}

export interface DriveFile {
  id: string;
  createdAt: string;
  name: string;
  type: string;
  size: number;
  isSensitive: boolean;
  url: string;
  thumbnailUrl: string | null;
  webpublicUrl: string | null;
  comment: string | null;
  folderId: string | null;
  blurhash: string | null;
}

export interface Notification {
  id: string;
  createdAt: string;
  type: string;
  isRead: boolean;
  userId: string | null;
  noteId: string | null;
  reaction: string | null;
  body?: string;
  user?: UserLite;
  note?: Note;
}

export interface InstanceMeta {
  name: string;
  shortName: string;
  uri: string;
  version: string;
  description: string;
  disabledRegistration?: boolean;
  disableRegistration: boolean;
  federation: "all" | "specified" | "none";
  registrationMode?: string;
  maxNoteTextLength: number;
  [key: string]: unknown;
}

export interface SigninResponse {
  finished: boolean;
  id: string;
  i: string;
}

export interface TimelineParams {
  limit?: number;
  sinceId?: string;
  untilId?: string;
  withFiles?: boolean;
  withRenotes?: boolean;
  withReplies?: boolean;
  includeMyRenotes?: boolean;
  includeRenotedMyNotes?: boolean;
  includeLocalRenotes?: boolean;
}
