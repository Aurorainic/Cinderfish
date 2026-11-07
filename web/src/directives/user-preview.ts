// Ported from firefish@b425f61:packages/client/src/directives/user-preview.ts (AGPL-3.0-only)
//
// The directive opens MkUserPreview after a 500ms hover, keeps it while the
// pointer is over the avatar or the card, and closes it 500ms after leaving
// either. The component fetches `users/show` itself.
import { defineAsyncComponent, ref, type Directive, type Ref } from "vue";
import { popup } from "@/os";

type UserPreviewValue = string | null | undefined;

class UserPreview {
	private el: HTMLElement;
	private userId: string;
	private showTimer: number | null = null;
	private hideTimer: number | null = null;
	private checkTimer: number | null = null;
	private showing: Ref<boolean> | null = null;

	constructor(el: HTMLElement, userId: string) {
		this.el = el;
		this.userId = userId;
		this.el.addEventListener("mouseover", this.onMouseover);
		this.el.addEventListener("mouseleave", this.onMouseleave);
		this.el.addEventListener("click", this.onClick);
	}

	private show = (): void => {
		if (!document.body.contains(this.el)) return;
		if (this.showing != null) return;

		const showing = ref(true);
		this.showing = showing;
		void popup(
			defineAsyncComponent(() => import("@/components/MkUserPreview.vue")),
			{
				showing,
				q: this.userId,
				source: this.el,
			},
			{
				mouseover: () => {
					if (this.hideTimer != null) window.clearTimeout(this.hideTimer);
				},
				mouseleave: () => {
					if (this.showTimer != null) window.clearTimeout(this.showTimer);
					this.hideTimer = window.setTimeout(this.close, 500);
				},
			},
			"closed",
		);

		this.checkTimer = window.setInterval(() => {
			if (!document.body.contains(this.el)) {
				this.close();
			}
		}, 1000);
	};

	private close = (): void => {
		if (this.showing == null) return;
		if (this.checkTimer != null) window.clearInterval(this.checkTimer);
		this.showing.value = false;
		this.showing = null;
	};

	private onMouseover = (): void => {
		if (this.showTimer != null) window.clearTimeout(this.showTimer);
		if (this.hideTimer != null) window.clearTimeout(this.hideTimer);
		this.showTimer = window.setTimeout(this.show, 500);
	};

	private onMouseleave = (): void => {
		if (this.showTimer != null) window.clearTimeout(this.showTimer);
		if (this.hideTimer != null) window.clearTimeout(this.hideTimer);
		this.hideTimer = window.setTimeout(this.close, 500);
	};

	private onClick = (): void => {
		if (this.showTimer != null) window.clearTimeout(this.showTimer);
		this.close();
	};

	detach(): void {
		this.el.removeEventListener("mouseover", this.onMouseover);
		this.el.removeEventListener("mouseleave", this.onMouseleave);
		this.el.removeEventListener("click", this.onClick);
		if (this.showTimer != null) window.clearTimeout(this.showTimer);
		if (this.hideTimer != null) window.clearTimeout(this.hideTimer);
		if (this.checkTimer != null) window.clearInterval(this.checkTimer);
		this.close();
	}
}

const instances = new WeakMap<HTMLElement, UserPreview>();

export default {
	mounted(el: HTMLElement, binding: { value: UserPreviewValue }) {
		if (binding.value == null || binding.value === "") return;
		instances.set(el, new UserPreview(el, binding.value));
	},

	updated(el: HTMLElement, binding: { value: UserPreviewValue }) {
		const previous = instances.get(el);
		if (previous != null) {
			previous.detach();
			instances.delete(el);
		}
		if (binding.value != null && binding.value !== "") {
			instances.set(el, new UserPreview(el, binding.value));
		}
	},

	unmounted(el: HTMLElement) {
		const previous = instances.get(el);
		if (previous != null) {
			previous.detach();
			instances.delete(el);
		}
	},
} as Directive<HTMLElement, UserPreviewValue>;
