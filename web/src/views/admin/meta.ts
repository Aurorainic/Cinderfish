import type { InstanceMeta } from "../../api/types";

export type FederationMode = "public" | "allowlist" | "none";

export interface FederationModeDef {
  id: FederationMode;
  label: string;
  desc: string;
}

export const FEDERATION_MODES: readonly FederationModeDef[] = [
  { id: "public", label: "公开联邦", desc: "与所有实例互相投递笔记与活动。" },
  { id: "allowlist", label: "仅白名单", desc: "只与白名单中的实例互通，其余实例的投递一律丢弃。" },
  { id: "none", label: "不开放联邦", desc: "完全关闭联邦，只保留本实例内部的笔记。" },
];

// The admin API speaks public/allowlist/none, while /api/meta exposes the
// Misskey vocabulary all/specified/none. Accept both spellings.
export function federationModeOf(value: string | null | undefined): FederationMode {
  if (value === "public" || value === "all") return "public";
  if (value === "allowlist" || value === "specified") return "allowlist";
  return "none";
}

export function federationLabel(meta: InstanceMeta | null): string {
  if (!meta?.federation) return "未知";
  const id = federationModeOf(meta.federation);
  return FEDERATION_MODES.find((item) => item.id === id)?.label ?? "未知";
}

const REGISTRATION_LABELS: Record<string, string> = {
  open: "开放注册",
  invite: "仅限邀请",
  admin: "仅管理员创建",
  closed: "停止注册",
};

export function registrationLabel(meta: InstanceMeta | null): string {
  if (!meta) return "未知";
  const mode = meta.registrationMode;
  if (mode) return REGISTRATION_LABELS[mode] ?? mode;
  return meta.disableRegistration ? "未开放自助注册" : "开放自助注册";
}

export function storageLabel(meta: InstanceMeta | null): string {
  const features = meta?.features;
  if (!features || typeof features !== "object") return "未知";
  const objectStorage = (features as Record<string, unknown>).objectStorage;
  return objectStorage === true ? "S3 兼容对象存储" : "本地文件系统";
}
