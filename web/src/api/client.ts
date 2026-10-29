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

async function post<T>(path: string, body: Record<string, unknown> = {}, authed = true): Promise<T> {
  const payload: Record<string, unknown> = { ...body };
  const token = getToken();
  if (authed && token) payload.i = token;

  const res = await fetch(path, {
    method: "POST",
    headers: { "Content-Type": "application/json", Accept: "application/json" },
    body: JSON.stringify(payload),
  });
  if (!res.ok) throw await parseError(res);
  return (await res.json()) as T;
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

  usersShow: (params: { userId?: string; username?: string; host?: string }) =>
    post<UserDetailed>("/api/users/show", params, false),

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

  showNote: (noteId: string) => post<Note>("/api/notes/show", { noteId }, false),

  deleteNote: (noteId: string) => post<Record<string, never>>("/api/notes/delete", { noteId }),

  react: (noteId: string, reaction: string) =>
    post<Record<string, never>>("/api/notes/reactions/create", { noteId, reaction }),

  unreact: (noteId: string) => post<Record<string, never>>("/api/notes/reactions/delete", { noteId }),

  localTimeline: (params: TimelineParams = {}) =>
    post<Note[]>("/api/notes/local-timeline", params as Record<string, unknown>, false),

  globalTimeline: (params: TimelineParams = {}) =>
    post<Note[]>("/api/notes/global-timeline", params as Record<string, unknown>, false),

  hybridTimeline: (params: TimelineParams = {}) =>
    post<Note[]>("/api/notes/hybrid-timeline", params as Record<string, unknown>),

  homeTimeline: (params: TimelineParams = {}) =>
    post<Note[]>("/api/notes/timeline", params as Record<string, unknown>),

  notifications: (params: { limit?: number; sinceId?: string; untilId?: string; markAsRead?: boolean } = {}) =>
    post<Notification[]>("/api/notifications", params),

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
};
