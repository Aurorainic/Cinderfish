<!-- Ported from firefish@b425f61:packages/client/src/components/global/MkUrl.vue (AGPL-3.0-only) -->
<template>
	<MkA
		v-if="self"
		class="ieqqeuvs url _link"
		:to="path"
		:title="url"
		:rel="rel"
		@contextmenu.stop="() => {}"
		@click.stop
	>
		<template v-if="pathname === '/'">
			<span class="self">{{ hostname }}</span>
		</template>
		<span v-if="pathname != ''" class="pathname">{{
			pathname.substring(1)
		}}</span>
		<span class="query">{{ query }}</span>
		<span class="hash">{{ hash }}</span>
	</MkA>
	<span v-else-if="invalid" class="ieqqeuvs url invalid">{{ url }}</span>
	<a
		v-else
		class="ieqqeuvs url _link"
		:href="url"
		:title="url"
		:rel="rel"
		target="_blank"
		@contextmenu.stop="() => {}"
		@click.stop
	>
		<span class="schema">{{ schema }}//</span>
		<span class="hostname">{{ hostname }}</span>
		<span v-if="port != ''" class="port">:{{ port }}</span>
		<span v-if="pathname != ''" class="pathname">{{ pathname }}</span>
		<span class="query">{{ query }}</span>
		<span class="hash">{{ hash }}</span>
		<i class="ph-arrow-square-out ph-bold ph-lg icon"></i>
	</a>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { url as local } from "../../config";
import MkA from "./MkA.vue";

// Degraded versus Firefish: the URL preview popup (MkUrlPreviewPopup) and the
// punycode hostname decoding are not ported - punycode is not in the dependency
// set and there is no url-preview endpoint. The link itself, including the
// schema/host/path/query/hash breakdown, is preserved.
const props = defineProps<{
	url: string;
	rel?: string;
}>();

const parsed = computed<URL | null>(() => {
	try {
		const url = new URL(props.url);
		if (!["http:", "https:"].includes(url.protocol)) return null;
		return url;
	} catch {
		return null;
	}
});

const invalid = computed<boolean>(() => parsed.value == null);
const self = computed<boolean>(
	() => parsed.value != null && props.url.startsWith(local),
);

/** Router path for a link that points at this instance. */
const path = computed<string>(() => {
	const suffix = props.url.substring(local.length);
	if (suffix === "") return "/";
	return suffix.startsWith("/") ? suffix : "/" + suffix;
});

function decode(value: string): string {
	try {
		return decodeURIComponent(value);
	} catch {
		return value;
	}
}

const schema = computed<string>(() => parsed.value?.protocol ?? "");
const hostname = computed<string>(() => parsed.value?.hostname ?? "");
const port = computed<string>(() => parsed.value?.port ?? "");
const pathname = computed<string>(() => decode(parsed.value?.pathname ?? ""));
const query = computed<string>(() => decode(parsed.value?.search ?? ""));
const hash = computed<string>(() => decode(parsed.value?.hash ?? ""));
</script>

<style lang="scss" scoped>
.url {
	text-decoration: none;

	> span {
		text-decoration: underline var(--fgTransparent);
		text-decoration-thickness: 1px;
		transition: text-decoration-color 0.2s;
	}

	> .icon {
		padding-left: 2px;
		font-size: 0.9em;
	}

	> .self {
		font-weight: bold;
	}

	> .schema {
		color: var(--fgTransparentWeak);
	}

	> .hostname {
		font-weight: bold;
	}

	> .pathname {
		color: var(--fgTransparentWeak);
	}

	> .query {
		color: var(--fgTransparentWeak);
	}

	> .hash {
		font-style: italic;
	}

	&:hover span {
		text-decoration-color: var(--link);
	}
}

.invalid {
	overflow-wrap: anywhere;
}
</style>
