// Ported from firefish@b425f61:packages/client/src/scripts/page-metadata.ts (AGPL-3.0-only)
//
// Firefish hands the current page's metadata to the UI shell, which composes
// `document.title` from it. Cinderfish's wave-1 shell has no metadata receiver
// wired yet, so this shim also writes `document.title` directly (contract
// §3: "definePageMetadata：设置 document.title"). The Firefish injection
// contract is kept intact, so a shell that later calls
// `provideMetadataReceiver` silently takes over again for its own display.
import {
	inject,
	isRef,
	onActivated,
	onMounted,
	provide,
	ref,
	watch,
	type ComputedRef,
	type Ref,
} from "vue";
import { instance } from "@/instance";
import type { UserLite } from "@/api/types";

export const setPageMetadata = Symbol("setPageMetadata");
export const pageMetadataProvider = Symbol("pageMetadataProvider");

export type PageMetadata = {
	title: string;
	subtitle?: string;
	icon?: string | null;
	avatar?: UserLite | null;
	userName?: UserLite | null;
	bg?: string;
};

type MetadataRef = Ref<PageMetadata | null> | ComputedRef<PageMetadata | null>;
type MetadataReceiver = (info: MetadataRef) => void;

/**
 * Register the current page's metadata.
 *
 * Must be called from a component `setup()` (it uses `provide`/`inject`).
 * Accepts a plain object, a ref or a computed ref, exactly like Firefish.
 */
export function definePageMetadata(
	metadata: PageMetadata | null | MetadataRef,
): void {
	// A computed ref is kept as-is so providers stay lazy; a plain value is
	// wrapped so downstream receivers can watch it.
	const _metadata = (isRef(metadata) ? metadata : ref(metadata)) as MetadataRef;

	provide(pageMetadataProvider, _metadata);

	const set = inject<MetadataReceiver | null>(setPageMetadata, null);
	if (set != null) {
		set(_metadata);

		onMounted(() => {
			set(_metadata);
		});

		onActivated(() => {
			set(_metadata);
		});
	}

	// Composing the tab title is the client's own concern (Firefish does it in
	// ui/universal.vue); doing it here keeps every ported page consistent even
	// without a receiver.
	watch(
		_metadata,
		(value) => {
			applyDocumentTitle(value);
		},
		{ immediate: true, deep: true },
	);
}

function applyDocumentTitle(metadata: PageMetadata | null): void {
	if (typeof document === "undefined") return;
	if (metadata == null) return;
	const instanceName = (instance as { name?: string }).name;
	// Pages whose title already is the instance name (the welcome page, for
	// one) would otherwise render as "Name | Name".
	const suffix =
		instanceName != null &&
		instanceName !== "" &&
		metadata.title !== instanceName
			? ` | ${instanceName}`
			: "";
	document.title = metadata.title + suffix;
}

export function provideMetadataReceiver(
	callback: (info: MetadataRef) => void,
): void {
	provide(setPageMetadata, callback);
}

/**
 * Firefish returns the injected metadata (a ref in practice — templates
 * auto-unwrap top-level refs). The declared type mirrors upstream so ported
 * headers can be copied verbatim.
 */
export function injectPageMetadata(): PageMetadata | undefined {
	return inject(pageMetadataProvider) as PageMetadata | undefined;
}
