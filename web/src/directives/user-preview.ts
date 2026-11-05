// Ported from firefish@b425f61:packages/client/src/directives/user-preview.ts (AGPL-3.0-only)
//
// TODO(port): Firefish opens MkUserPreview on hover after 500ms and needs
// users/show. Cinderfish has neither the component nor the endpoint wiring yet,
// so this directive is a no-op placeholder that keeps the ported templates
// compiling and lets the feature be added without touching the components.
import type { Directive } from "vue";

type UserPreviewValue = string | null | undefined;

export default {
	mounted(_el: HTMLElement, _binding: { value: UserPreviewValue }) {
		// no-op (see TODO above)
	},

	unmounted(_el: HTMLElement, _binding: { value: UserPreviewValue }) {
		// no-op (see TODO above)
	},
} as Directive<HTMLElement, UserPreviewValue>;
