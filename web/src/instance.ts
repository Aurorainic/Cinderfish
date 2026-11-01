// Firefish-shaped view of the instance metadata.
//
// Firefish's `instance.ts` keeps a `DetailedInstanceMetadata` copy seeded from
// IndexedDB and refreshed with `api("meta", { detail: true })`. Cinderfish has
// one source for that data — `stores/instance.ts` over `POST /api/meta` — so
// this module only reshapes it: cache and refresh stay in the store.
//
// Missing fields get an explicit empty value instead of a plausible-looking
// one. In particular `POST /api/meta` carries no emoji catalogue, so `emojis`
// is always empty here; that is a projection gap (the emoji picker therefore
// offers Unicode only), not a claim that the instance has none. The same goes
// for `features.serviceWorker` and `features.searchFilters`, which the backend
// does not provide.
import { computed } from "vue";
import type { InstanceMeta } from "./api/types";
import { instance } from "./stores/instance";

export interface EmojiSimple {
  name: string;
  url: string;
  category: string | null;
  aliases: string[];
}

export interface InstanceFeatures {
  registration: boolean;
  localTimeLine: boolean;
  recommendedTimeLine: boolean;
  globalTimeLine: boolean;
  searchFilters: boolean;
  hcaptcha: boolean;
  recaptcha: boolean;
  objectStorage: boolean;
  serviceWorker: boolean;
}

/** The slice of Firefish's DetailedInstanceMetadata the ported screens read. */
export interface InstanceInfo {
  name: string | null;
  shortName: string | null;
  uri: string;
  version: string;
  description: string | null;
  maintainerName: string;
  maintainerEmail: string;
  tosUrl: string | null;
  disableRegistration: boolean;
  emailRequiredForSignup: boolean;
  maxNoteTextLength: number;
  iconUrl: string | null;
  bannerUrl: string | null;
  backgroundImageUrl: string | null;
  mascotImageUrl: string;
  emojis: EmojiSimple[];
  features: InstanceFeatures;
}

const DEFAULT_FEATURES: InstanceFeatures = {
  registration: false,
  localTimeLine: false,
  recommendedTimeLine: false,
  globalTimeLine: false,
  searchFilters: false,
  hcaptcha: false,
  recaptcha: false,
  objectStorage: false,
  serviceWorker: false,
};

const FALLBACK: InstanceInfo = {
  name: null,
  shortName: null,
  uri: "",
  version: "",
  description: null,
  maintainerName: "",
  maintainerEmail: "",
  tosUrl: null,
  disableRegistration: true,
  emailRequiredForSignup: false,
  maxNoteTextLength: 3000,
  iconUrl: null,
  bannerUrl: null,
  backgroundImageUrl: null,
  mascotImageUrl: "",
  emojis: [],
  features: DEFAULT_FEATURES,
};

function readString(meta: InstanceMeta, key: string, fallback: string): string {
  const value = meta[key];
  return typeof value === "string" ? value : fallback;
}

function readNullableString(meta: InstanceMeta, key: string): string | null {
  const value = meta[key];
  return typeof value === "string" && value !== "" ? value : null;
}

function readBoolean(meta: InstanceMeta, key: string, fallback: boolean): boolean {
  const value = meta[key];
  return typeof value === "boolean" ? value : fallback;
}

function readNumber(meta: InstanceMeta, key: string, fallback: number): number {
  const value = meta[key];
  return typeof value === "number" ? value : fallback;
}

function flag(features: Record<string, unknown>, key: string, fallback = false): boolean {
  const value = features[key];
  return typeof value === "boolean" ? value : fallback;
}

function projectFeatures(meta: InstanceMeta | null): InstanceFeatures {
  if (!meta) return DEFAULT_FEATURES;
  const raw = meta.features;
  const features = raw !== null && typeof raw === "object" ? (raw as Record<string, unknown>) : {};
  return {
    registration: flag(features, "registration", !meta.disableRegistration),
    // The backend spells these `localTimeline`/`globalTimeline`; the Firefish
    // spelling is what ported components read.
    localTimeLine: flag(features, "localTimeline", true),
    recommendedTimeLine: false,
    globalTimeLine: flag(features, "globalTimeline", false),
    searchFilters: false,
    hcaptcha: flag(features, "hcaptcha"),
    recaptcha: flag(features, "recaptcha"),
    objectStorage: flag(features, "objectStorage"),
    serviceWorker: false,
  };
}

/** A snapshot built from the store; callers never mutate the stored meta. */
export function getInstanceInfo(): InstanceInfo {
  const meta = instance.meta;
  if (!meta) return { ...FALLBACK, features: { ...DEFAULT_FEATURES } };
  return {
    name: readNullableString(meta, "name"),
    shortName: readNullableString(meta, "shortName"),
    uri: readString(meta, "uri", ""),
    version: readString(meta, "version", ""),
    description: readNullableString(meta, "description"),
    maintainerName: readString(meta, "maintainerName", ""),
    maintainerEmail: readString(meta, "maintainerEmail", ""),
    tosUrl: readNullableString(meta, "tosUrl"),
    disableRegistration: readBoolean(meta, "disableRegistration", true),
    emailRequiredForSignup: readBoolean(meta, "emailRequiredForSignup", false),
    maxNoteTextLength: readNumber(meta, "maxNoteTextLength", 3000),
    iconUrl: readNullableString(meta, "iconUrl"),
    bannerUrl: readNullableString(meta, "bannerUrl"),
    backgroundImageUrl: readNullableString(meta, "backgroundImageUrl"),
    mascotImageUrl: readString(meta, "mascotImageUrl", ""),
    emojis: [],
    features: projectFeatures(meta),
  };
}

// Kept for the ported call sites that iterate categories/tags; both are empty
// for the same reason `emojis` is.
export const emojiCategories = computed<string[]>(() => []);
export const emojiTags = computed<string[]>(() => []);
