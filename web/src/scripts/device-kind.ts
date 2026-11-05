// Ported from firefish@b425f61:packages/client/src/scripts/device-kind.ts (AGPL-3.0-only)
import { defaultStore } from "@/store";

export type DeviceKind = "smartphone" | "tablet" | "desktop";

const ua = navigator.userAgent.toLowerCase();
const isTablet =
	/ipad/.test(ua) ||
	(/mobile|iphone|android/.test(ua) && window.innerWidth > 700);
const isSmartphone = !isTablet && /mobile|iphone|android/.test(ua);

export const deviceKind: DeviceKind =
	defaultStore.state.overridedDeviceKind ??
	(isSmartphone ? "smartphone" : isTablet ? "tablet" : "desktop");
