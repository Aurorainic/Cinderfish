import type {
  DriveFile,
  InstanceMeta,
  MeDetailed,
  Note,
  Notification,
  SigninResponse,
  TimelineParams,
  UserDetailed,
  UserLite,
} from "./types";

const TOKEN_KEY = "cinderfish.token";

export class ApiError extends Error {
  status: number;
  code: string;
  id: string;
  kind: string;
  info: unknown;

  constructor(
    message: string,
    status: number,
    code = "",
    id = "",
    kind = "",
    info: unknown = null,
  ) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
    this.id = id;
    this.kind = kind;
    this.info = info;
  }
}

export function getToken(): string {
  return localStorage.getItem(TOKEN_KEY) ?? "";
}

export function setToken(token: string): void {
  if (token) localStorage.setItem(TOKEN_KEY, token);
  else localStorage.removeItem(TOKEN_KEY);
}

async function parseError(res: Response): Promise<ApiError> {
  let body: Record<string, unknown> = {};
  try {
    body = (await res.json()) as Record<string, unknown>;
  } catch {
    // keep the empty body
  }
  const raw = body.error;
  if (raw && typeof raw === "object") {
    const e = raw as Record<string, unknown>;
    return new ApiError(
      typeof e.message === "string" ? e.message : res.statusText || "请求失败",
      res.status,
      typeof e.code === "string" ? e.code : "",
      typeof e.id === "string" ? e.id : "",
      typeof e.kind === "string" ? e.kind : "",
      e.info ?? null,
    );
  }
  return new ApiError(res.statusText || `HTTP ${res.status}`, res.status);
}

// Every note carries a copy of its author. After the signed-in user edits
// their own profile, the copies already in memory are stale, so the author
// fields a note header shows (name and avatar) are restamped on the way out
// whenever the author is the viewer. Notes fetched before the session
// resolved simply keep the server's copy until the next fetch.
interface ViewerStamp {
  id: string;
  name: string | null;
  avatarUrl: string | null;
}

const VIEWER_PATHS = new Set([
  "/api/i",
  "/api/signup",
  "/api/i/update",
  "/api/i/pin",
  "/api/i/unpin",
]);

let viewer: ViewerStamp | null = null;

function rememberViewer(value: unknown): void {
  if (value === null || typeof value !== "object") return;
  const me = value as Record<string, unknown>;
  if (typeof me.id !== "string" || typeof me.username !== "string") return;
  viewer = {
    id: me.id,
    name: typeof me.name === "string" ? me.name : null,
    avatarUrl: typeof me.avatarUrl === "string" ? me.avatarUrl : null,
  };
}

function stampViewer(value: unknown, depth = 0): void {
  if (!viewer || depth > 4 || value === null || typeof value !== "object") return;
  if (Array.isArray(value)) {
    for (const item of value) stampViewer(item, depth + 1);
    return;
  }
  const record = value as Record<string, unknown>;
  const author = record.user;
  if (record.userId === viewer.id && author && typeof author === "object") {
    record.user = {
      ...(author as Record<string, unknown>),
      name: viewer.name,
      avatarUrl: viewer.avatarUrl,
    };
  }
  for (const key of ["reply", "renote", "note"]) {
    const nested = record[key];
    if (nested && typeof nested === "object") stampViewer(nested, depth + 1);
  }
}

// sendToken defaults to true: every endpoint that can serve viewer-specific
// fields (myReaction on notes, unread counts, the pinned list) reads the
// token from the request, so a read that forgets it silently loses those
// fields. Pass false only where a token would be meaningless or harmful:
// signin, signup, availability checks and instance meta.
async function post<T>(path: string, body: Record<string, unknown> = {}, sendToken = true): Promise<T> {
  const payload: Record<string, unknown> = { ...body };
  const token = getToken();
  if (!token) viewer = null;
  if (sendToken && token) payload.i = token;

  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    Accept: "application/json",
  };
  // The admin handlers authenticate from the header only, so send both:
  // `i` for Misskey compatibility and Bearer for every backend route.
  if (sendToken && token) headers.Authorization = `Bearer ${token}`;
  const res = await fetch(path, {
    method: "POST",
    headers,
    body: JSON.stringify(payload),
  });
  if (!res.ok) throw await parseError(res);
  const data = (await res.json()) as T;
  if (VIEWER_PATHS.has(path)) rememberViewer(data);
  else stampViewer(data);
  return data;
}

async function upload(file: File): Promise<DriveFile> {
  const form = new FormData();
  form.append("file", file);
  form.append("name", file.name);
  const token = getToken();
  const res = await fetch("/api/drive/files/create", {
    method: "POST",
    headers: token ? { Authorization: `Bearer ${token}` } : {},
    body: form,
  });
  if (!res.ok) throw await parseError(res);
  return (await res.json()) as DriveFile;
}

export const api = {
  meta: (detail = true) => post<InstanceMeta>("/api/meta", { detail }, false),

  signin: (username: string, password: string) =>
    post<SigninResponse>("/api/signin", { username, password }, false),

  signup: (username: string, password: string, invitationCode: string) =>
    post<MeDetailed & { token: string }>(
      "/api/signup",
      { username, password, invitationCode },
      false,
    ),

  usernameAvailable: (username: string) =>
    post<{ available: boolean }>("/api/username/available", { username }, false),

  i: () => post<MeDetailed>("/api/i"),

  // iUpdate writes the editable profile fields. An absent key leaves the
  // stored value alone; null or an empty string clears the field, which is
  // how Misskey's own client clears a description. name is capped at 50
  // and description at 500 characters by the server.
  iUpdate: (params: { name?: string | null; description?: string | null; avatarId?: string | null }) =>
    post<MeDetailed>("/api/i/update", params as Record<string, unknown>),

  // pin/unpin answer with the freshly packed MeDetailed user, so callers can
  // replace the session copy and get the new pinnedNoteIds for free.
  pin: (noteId: string) => post<MeDetailed>("/api/i/pin", { noteId }),

  unpin: (noteId: string) => post<MeDetailed>("/api/i/unpin", { noteId }),

  usersShow: (params: { userId?: string; username?: string; host?: string }) =>
    post<UserDetailed>("/api/users/show", params),

  usersNotes: (params: {
    userId?: string;
    username?: string;
    limit?: number;
    sinceId?: string;
    untilId?: string;
    withFiles?: boolean;
    withReplies?: boolean;
  }) => post<Note[]>("/api/users/notes", params as Record<string, unknown>),

  createNote: (params: {
    text?: string;
    cw?: string | null;
    visibility?: string;
    localOnly?: boolean;
    fileIds?: string[];
    replyId?: string;
    renoteId?: string;
    visibleUserIds?: string[];
  }) => post<Note>("/api/notes/create", params),

  showNote: (noteId: string) => post<Note>("/api/notes/show", { noteId }),

  deleteNote: (noteId: string) => post<Record<string, never>>("/api/notes/delete", { noteId }),

  react: (noteId: string, reaction: string) =>
    post<Record<string, never>>("/api/notes/reactions/create", { noteId, reaction }),

  unreact: (noteId: string) => post<Record<string, never>>("/api/notes/reactions/delete", { noteId }),

  localTimeline: (params: TimelineParams = {}) =>
    post<Note[]>("/api/notes/local-timeline", params as Record<string, unknown>),

  globalTimeline: (params: TimelineParams = {}) =>
    post<Note[]>("/api/notes/global-timeline", params as Record<string, unknown>),

  hybridTimeline: (params: TimelineParams = {}) =>
    post<Note[]>("/api/notes/hybrid-timeline", params as Record<string, unknown>),

  homeTimeline: (params: TimelineParams = {}) =>
    post<Note[]>("/api/notes/timeline", params as Record<string, unknown>),

  notifications: (
    params: {
      limit?: number;
      sinceId?: string;
      untilId?: string;
      markAsRead?: boolean;
      includeTypes?: string[];
      excludeTypes?: string[];
    } = {},
  ) => post<Notification[]>("/api/notifications", params),

  markAllNotificationsRead: () => post<Record<string, never>>("/api/notifications/mark-all-as-read"),

  follow: (userId: string) => post<UserLite>("/api/following/create", { userId }),
  unfollow: (userId: string) => post<UserLite>("/api/following/delete", { userId }),

  driveFiles: (params: { limit?: number; untilId?: string } = {}) =>
    post<DriveFile[]>("/api/drive/files", params),

  uploadFile: upload,

  deleteDriveFile: (fileId: string) => post<Record<string, never>>("/api/drive/files/delete", { fileId }),

  federationMode: (mode: string) => post<{ mode: string }>("/api/admin/federation/mode", { mode }),

  federationAllowlist: async () => {
    const token = getToken();
    const res = await fetch("/api/admin/federation/allowlist", {
      headers: token ? { Authorization: `Bearer ${token}` } : {},
    });
    if (!res.ok) throw await parseError(res);
    return (await res.json()) as { hosts: string[] };
  },

  updateFederationAllowlist: (host: string, remove = false) =>
    post<{ hosts: string[] }>("/api/admin/federation/allowlist", { host, remove }),

  adminCreateAccount: (username: string, password: string) =>
    post<UserDetailed>("/api/admin/accounts/create", { username, password }),

  adminShowUsers: (params: { limit?: number; offset?: number } = {}) =>
    post<UserDetailed[]>("/api/admin/show-users", params as Record<string, unknown>),

  adminSuspendUser: (userId: string) =>
    post<Record<string, never>>("/api/admin/suspend-user", { userId }),

  adminUnsuspendUser: (userId: string) =>
    post<Record<string, never>>("/api/admin/unsuspend-user", { userId }),
};
