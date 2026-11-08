import { t } from "@/i18n";
import type { MenuItem } from "@/os";
import copyToClipboard from "@/scripts/copy-to-clipboard";

export interface CopyLinkMenuItemOptions {
	/** Icon classes; defaults to the note menu icon. */
	icon?: string;
	/** Runs after the URL has been copied successfully. */
	onCopied?: () => void;
}

export function makeCopyLinkMenuItem(
	url: string,
	options: CopyLinkMenuItemOptions = {},
): MenuItem {
	return {
		icon: options.icon ?? "ph-link-simple ph-bold ph-lg",
		text: t("copyLink"),
		action: () => {
			void copyToClipboard(url).then((ok) => {
				if (ok) options.onCopied?.();
			});
		},
	};
}
