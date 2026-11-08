<template>
	<div class="cf-captcha">
		<div ref="mount" class="mount"></div>
		<p v-if="error !== null" class="error">{{ error }}</p>
	</div>
</template>

<script lang="ts" setup>
// Renders the instance's captcha widget (hCaptcha, reCAPTCHA or Turnstile).
// The scripts come from the providers' fixed CDN hosts; the verification
// happens server-side. The token is exposed through v-model.
import { onBeforeUnmount, onMounted, ref, watch } from "vue";

type CaptchaProvider = "hcaptcha" | "recaptcha" | "turnstile";

const props = defineProps<{
	provider: CaptchaProvider;
	siteKey: string;
}>();

const emit = defineEmits<{
	"update:modelValue": [value: string];
}>();

const scriptUrls: Record<CaptchaProvider, string> = {
	hcaptcha: "https://js.hcaptcha.com/1/api.js?render=explicit",
	recaptcha: "https://www.recaptcha.net/recaptcha/api.js?render=explicit",
	turnstile: "https://challenges.cloudflare.com/turnstile/v0/api.js?render=explicit",
};

const globalNames: Record<CaptchaProvider, string> = {
	hcaptcha: "hcaptcha",
	recaptcha: "grecaptcha",
	turnstile: "turnstile",
};

// One script load per provider, shared by every widget instance.
const scriptPromises = new Map<CaptchaProvider, Promise<void>>();

function loadScript(provider: CaptchaProvider): Promise<void> {
	const existing = scriptPromises.get(provider);
	if (existing != null) return existing;
	const promise = new Promise<void>((resolve, reject) => {
		const script = document.createElement("script");
		script.src = scriptUrls[provider];
		script.async = true;
		script.defer = true;
		script.addEventListener("load", () => resolve());
		script.addEventListener("error", () => reject(new Error("script load failed")));
		document.head.appendChild(script);
	});
	scriptPromises.set(provider, promise);
	return promise;
}

const mount = ref<HTMLElement | null>(null);
const error = ref<string | null>(null);
let api: any = null;
let widgetId: string | number | null = null;

function captchaApi(provider: CaptchaProvider): any {
	return (window as unknown as Record<string, any>)[globalNames[provider]];
}

function render(): void {
	const element = mount.value;
	api = captchaApi(props.provider);
	if (element == null || api == null) return;
	widgetId = api.render(element, {
		sitekey: props.siteKey,
		callback: (token: string) => emit("update:modelValue", token),
		"expired-callback": () => emit("update:modelValue", ""),
		"error-callback": () => emit("update:modelValue", ""),
	});
}

onMounted(async () => {
	try {
		await loadScript(props.provider);
		render();
	} catch {
		error.value = "验证组件加载失败，请刷新页面重试。";
	}
});

watch(
	() => [props.provider, props.siteKey],
	() => {
		if (api != null && widgetId != null && typeof api.remove === "function") {
			try {
				api.remove(widgetId);
			} catch {
				// Ignore: the widget may already be gone.
			}
		}
		widgetId = null;
		emit("update:modelValue", "");
		render();
	},
);

function reset(): void {
	emit("update:modelValue", "");
	if (api != null && widgetId != null && typeof api.reset === "function") {
		api.reset(widgetId);
	}
}

onBeforeUnmount(() => {
	if (api != null && widgetId != null && typeof api.remove === "function") {
		try {
			api.remove(widgetId);
		} catch {
			// Ignore teardown races.
		}
	}
});

defineExpose({ reset });
</script>

<style lang="scss" scoped>
.cf-captcha {
	.mount {
		min-height: 4em;
	}

	.error {
		margin: 4px 0 0;
		font-size: 0.8em;
		color: var(--error, #d33);
	}
}
</style>
