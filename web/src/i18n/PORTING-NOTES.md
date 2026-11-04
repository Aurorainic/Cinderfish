# Cinderfish frontend — i18n port notes

Date: 2026-11-04. This file documents the `web/src/i18n` deliverable, plus the
adjacent modules produced under the same brief while the frontend was being
rebuilt in parallel.

Reference snapshot: `tmp/refcheck/firefish-jclaudan-mirror` (client source at
`packages/client/src`, locales at `locales/`). Contract docs:
`docs/research/api-contract/*.md`.

---

## 1. i18n (exclusive deliverable)

### Files

| File | Purpose |
|------|---------|
| `web/src/i18n/index.ts` | Locale runtime: `translate`/`t`, `useLocale()`, `i18n` singleton, `locale` ref, `setLocale`, `getLocale`. |
| `web/src/i18n/locales/zh-CN.json` | Flat dot-key messages, generated from Firefish `locales/zh-CN.yml` (1838 keys). |
| `web/src/i18n/locales/ja-JP.json` | Same for `ja-JP.yml` (1833 keys). |
| `web/scripts/gen-locale.mjs` | Build-time generator, YAML → flat JSON. |

### What was ported

- `packages/client/src/i18n.ts` — the `I18n` class (`t(key, args)` with dot-path
  lookup, `{name}` interpolation, fail-soft on missing keys) and the exported
  singleton. Firefish stores messages as a nested object; Cinderfish stores a
  **flat** `Record<string, string>` with dot keys and resolves paths against it.
- `packages/client/src/config.ts` — locale detection from `localStorage.lang`
  (same key Firefish uses), then `navigator.languages`, then `zh-CN`.
- Firefish locale YAML — all keys, including nested groups (`_ago`, `_time`,
  `_theme.keys.*`, `_permissions.*`, `_deck.*`) flattened with `.` separators.
  `_lang_` is preserved as a key (`locale["_lang_"]`).

### Runtime API

```ts
import { t, translate, useLocale, i18n, setLocale } from "@/i18n";

t("_ago.justNow");
t("monthAndDay", { month: 11, day: 4 });
const { t, locale, setLocale, availableLocales } = useLocale();
i18n.ts.someFlatKey; // flat map (see caveat below)
```

- `translate` is the canonical function; `t` is a plain alias (both exist
  because different ported modules chose one or the other).
- Missing keys fail soft: the key string is returned and a warning is logged in
  dev only (`import.meta.env.DEV`). No throw, no empty string.
- `useLocale()` returns `{ locale, t, ts, setLocale, availableLocales }`;
  `locale` is a module-level `Ref<LocaleCode>`, so every component re-renders
  when the language changes.
- `setLocale()` persists to `localStorage.lang` and broadcasts on
  `BroadcastChannel("cinderfish:locale")`, so other tabs switch too.

### Generator

`web/scripts/gen-locale.mjs` flattens nested YAML maps into dot keys and writes
`web/src/i18n/locales/<code>.json`. It deliberately uses the `yaml` package's
YAML 1.2 parser:

- **Do not use PyYAML's default loader.** YAML 1.1 turns the keys `yes` / `no`
  into booleans, silently producing `True` / `False` keys. The locale files use
  `yes: "是"` / `no: "否"`.
- A naive line parser also drops keys containing a colon
  (`_permissions.read:account`) and folded multi-line values
  (`_experiments.postImportsCaption`). The previous generated output was
  missing 34 (zh) / 36 (ja) keys because of exactly these two bugs; it has
  been regenerated with the full set.

**Dependency note:** `yaml` is *not* added to `web/package.json`. Generation is
a one-off per locale update and the JSON is committed, so builds and everyday
development need no YAML parser. Adding it later is a one-line
`devDependencies` change if the locale files start changing frequently.

### Renames / deviations

| Firefish | Cinderfish | Why |
|----------|------------|-----|
| `i18n.ts` class instance | `i18n` plain object + `translate` | Tree-shakeable, no class. |
| `i18n.ts._ago.justNow` (nested) | `t("_ago.justNow")` | Flat dictionary; `i18n.ts` is kept for flat keys only. |
| `i18n.ts.somethingHappened` | `t("somethingHappened")` | Same key. |
| Webpack YAML loader | committed JSON + generator | Vite, no YAML runtime. |

`i18n.ts` is a getter returning the **flat** message map. Nested access
(`i18n.ts._ago.justNow`) therefore returns `undefined`; all ported components
use `t()` / `useLocale().t` instead. This is the one compatibility gap for
future page ports that copy Firefish template code verbatim.

### Stubs / known gaps

- No pluralisation rules (Firefish has none either; keys carry `{n}`).
- No per-locale date formatting; `CfTime` uses the browser `Intl` locale.
- `_ago.invalid` does not exist in the Firefish snapshot; `CfTime` falls back
  to `invalidValue` instead, so the UI never shows a raw key.
- No language picker page yet. `useLocale().setLocale()` is ready for
  `/settings/general`.
- Account `lang` from `MeDetailed` is not applied on sign-in (the API always
  returns `lang: null`), so only the local preference is used.

---

## 2. Adjacent modules produced under the same brief

The brief for this task listed eight areas. While they were being built, other
agents were writing the same areas into sibling directories
(`api-client/`, `store-composables/`, `os-dialog-ui/`, `account-session/`,
`scaffold-theme/`, `router-entry/`, `ui/`). The tree therefore contains two
implementations for several areas. This section records what this agent
produced and what it changed, so the parent can pick a canonical set.

### 2.1 Themes (`web/src/themes`)

- `light.ts` / `dark.ts` — verbatim port of `themes/_light.json5` and
  `themes/_dark.json5` into typed `Theme` modules (all props, including the
  `swutchOff*` spelling and `X*` slots).
- `color.ts` — tinycolor subset used by themes: parse hex/rgb/hsl, darken,
  lighten, saturate, hue spin, alpha; tinycolor-compatible `rgb()/rgba()`.
- `compile.ts` — Firefish value language: `@prop`, `$const`,
  `:darken|lighten|alpha|hue|saturate<amount<value>`, quoted literals; cycle
  detection.
- `apply.ts` — writes every prop as a `--<prop>` CSS variable on `<html>`,
  sets `data-theme`, updates `<meta name="theme-color">`, optional persistence.
- `builtin.ts` — base theme table; the other agent's `builtin/` palettes layer
  on top of these tables.

Renames: none in prop names (CSS variable names must match Firefish exactly,
including `--swutchOffBg`). See §5 for the `useTheme()` hook.

### 2.2 Styles (`web/src/styles`)

`style.scss` (root tokens, reset, typography, scrollbars, selection,
`_button`/`_panel`/`_acrylic`/`_fullinfo` utilities) and `global.scss` (shell
and page helpers). The other agent split the same content into
`styles/index.scss` + partials; both are present, and `main.ts` loads
`styles/index.scss`.

### 2.3 API (`web/src/api`)

- `client.ts` — `api(endpoint, params, token?)`: POST JSON to `/api/<endpoint>`,
  body `{ i, ...params }` (`i` is first-class; omitted token = current session,
  `null` = anonymous), `apiGet` for the two admin GETs, `uploadDriveFile`
  (multipart, token must be `Authorization: Bearer`), `postOAuthToken`,
  `resolveMediaUrl` (root-relative `/media/...` → origin), pending-request ref.
- `errors.ts` — `ApiError` for the Misskey envelope, the signin id-only
  envelope, and the OAuth `{error, error_description}` shape; signin error-id
  classification.
- `endpoints.ts` — every route registered in `internal/httpapi/server.go` with
  request/response types (`EndpointMap`), plus pagination param types.
- `pagination.ts` — `untilId`/`sinceId` cursors, `limit + 1` trick for
  `hasMore`, ascending-cursor rule (`sinceId` alone flips to ASC), merge/dedupe.
- `stream.ts` — WebSocket client: `?i=` token, `connect` with `pong: true`,
  channel dispatch, 30s application-level heartbeat (`channel` frame with
  `type:"__ping__"`; server pongs do not refresh `lastSeen`), exponential
  reconnect with jitter, resubscribe, `subNote`/`unsubNote` helpers, typed
  emitter. Supported channels are only the five the backend implements:
  `main`, `homeTimeline`, `localTimeline`, `hybridTimeline`, `globalTimeline`.

The other agent's `api-client/` carries a near-identical layer; `api/index.ts`
is the barrel currently loaded by `store/base.ts`.

### 2.4 Store / composables (`web/src/store`, `web/src/composables`)

- `store/pizzax.ts` — `createPizzax(bucket, defs, options?)` with the three
  Firefish scopes (`device`, `deviceAccount`, `account`), `pizzax::<bucket>`
  localStorage keys, registry sync/cache, `BroadcastChannel` cross-tab sync,
  `makeGetterSetter`, `push`, `reset`. `PizzaxOptions`/`PizzaxRemote` were added
  to stay compatible with the other agent's `store/base.ts`.
- `store/store.ts` — full Firefish `defaultStore` key list (bucket `base`).
- `store/deck.ts` — deck columns/layout/profiles, registry scope
  `["client","deck","profiles"]`, all swap/stack helpers.
- `store/cold-store.ts` — `miux:` device preferences (sounds, volumes).
- `store/layout.ts` — layout preference (`localStorage.ui`) and injection key.
- `composables/use-pagination.ts` — reactive cursor pagination.
- `composables/use-note-capture.ts` — `subNote` capture for
  `reacted`/`unreacted`/`deleted` (the only events the backend sends).
- `composables/use-interval.ts` — Firefish's interval helper.
- `composables/use-theme.ts` — superseded by the other agent's version; the
  active contract is `useTheme() → { themeName, setTheme, resolvedMode }` plus
  `mode/setMode/toggleMode/themes/theme` and `initTheme()`.

`store/base.ts`, `store/sound.ts`, `store/ui.ts`, `store/widgets.ts` and
`store-composables/*` come from the other agent; two compatibility fixes were
applied there (`setRemote`/`PizzaxOptions` in `pizzax.ts`, `Widget` alias in
`store/index.ts`, `overridedDeviceKind` read from the base store in `ui.ts`).

### 2.5 OS (`web/src/os`)

- `state.ts` — reactive popup stack, `claimZIndex` bands
  (1M/2M/3M, +100 per claim), `getUniqueId`, `popup(component, props, events,
  disposeEvent)`.
- `dialogs.ts` — `alert`, `confirm`, `yesno`, `inputText`/`prompt`,
  `inputParagraph`, `inputNumber`, `inputDate`, `waiting`, `toast`, `success`,
  `error`, `promiseDialog`/`awaitDialog`, `popupMenu`/`contextMenu`,
  `apiWithDialog`.
- Components: `CfPopupHost` (root host), `CfModal` (backdrop + layer),
  `CfDialog` (rounded panel, backdrop blur, icon/title/text/input/buttons),
  `CfWaitingDialog`, `CfToast`, `CfPopupMenu`.
- Visuals use `--panel`, `--modalBg`, `--radius`, `--shadow`, `--fg`,
  `--fgTransparentWeak`, `--accent`.

Not ported: page windows / modal page windows, nested context menus, sounds.

### 2.6 UI atoms (`web/src/components/ui`)

Ported with Firefish file names and SCSS class names (`Mk*` → `Cf*`):
`CfButton`, `CfInput`, `CfTextarea`, `CfSwitch`, `CfSelect`, `CfRadios`,
`CfRadio` (helper), `CfA`, `CfAvatar`, `CfUserName`, `CfTime`, `CfLoading`,
`CfEllipsis`, `CfPagination`, `CfDateSeparatedList`, `CfFolder`, `CfInfo`,
`CfKeyValue`, `CfTab`, `CfWindow`.

Stubs / intentional differences:

| Component | Difference |
|-----------|------------|
| all | Phosphor icon font not shipped: `icon()` emits the same `ph-*` classes, styled as 1em boxes until the icon asset lands. |
| `CfUserName` | No MFM renderer; renders plain text (shortcodes not expanded). |
| `CfAvatar` | No PhotoSwipe lightbox, no blurhash average colour, no online indicator polling. |
| `CfSelect` | Native `<select>` (same markup/classes/props) instead of Firefish's VNode-scraping popup menu. |
| `CfRadios` | Group model passed by `provide`/`inject` instead of cloning slot VNodes; usage shape is unchanged. |
| `CfPagination` | Cursor pagination + IntersectionObserver infinite scroll; no ad injection or folded items. |
| `CfDateSeparatedList` | No ad insertion (`_shouldInsertAd_`); date separators match Firefish. |
| `CfWindow` | Drag/maximise/8-handle resize implemented; no pinch/touch multi-pointer and no window snapping menu. |

### 2.7 Account (`web/src/account`)

`account-session` agent's modules (`me.ts`, `session.ts`, `storage.ts`,
`guard.ts`, `miauth.ts`, …) are the active session layer and are re-exported by
`account/index.ts`; the router imports `authGuard` from there. This agent's
`CfAccountButton.vue` was kept and adapted to `me`/`useLocale`.

### 2.8 Router / entry / pages

`router-entry` agent's `router/routes.ts` (all Firefish paths, `requiresLogin`
meta), `router/index.ts`, `main.ts`, `App.vue` and `ui/*Shell.vue` are active.
This agent generated `web/src/pages/<Name>.vue` placeholders for every route
component (each renders `TODO: port <firefish path>`); route names resolve to
those files. The `/` route uses the router agent's `IndexRoute` page, which
chooses between the timeline and welcome layouts.

---

## 3. Contract notes that shaped the client

- **Token**: all RPC endpoints read `i` from the JSON body first, then
  `Authorization: Bearer`. Streaming is the opposite: it is a GET, so it only
  reads `?i=` or the header (browsers cannot set WebSocket headers) → the
  stream client always puts the token in the query string.
- **Errors**: `{error:{message, code, id, kind, info?}}`; `signin` failures are
  `{error:{id}}` with no code; OAuth uses RFC 6749. Branch on `code`, never on
  `id`.
- **Pagination**: `untilId` walks down (default DESC); `sinceId` alone walks up
  (ASC) for followers/following. `users/notes` never flips.
- **Streaming liveness**: server pongs do not refresh `lastSeen`; a data frame
  every <120s is mandatory. 30s heartbeat implemented.
- **Streaming gaps**: no `registryUpdated`, `driveFileCreated`, `pollVoted`,
  `meUpdated`; only the five channels above. UI must poll or refetch.

## 4. Verification performed

- Regenerated locales and verified parity against the source YAML: **1838**
  zh-CN keys, **1833** ja-JP keys, including `yes`/`no`,
  `_permissions.read:account` and `_experiments.postImportsCaption`.
- Ran a static relative-import resolver over all 359 `.ts`/`.vue` files in
  `web/src`: **0 unresolved imports**.
- No `pnpm install` / build was run, per the task instructions. `vue-tsc` is
  expected to be run centrally after the parallel agents' trees are deduplicated.

## 5. Renames summary

| Firefish | Cinderfish |
|----------|------------|
| `os.ts` | `os/index.ts` + `os/state.ts` + `os/dialogs.ts` |
| `os.popup` / `os.alert` / `os.confirm` | same names, typed |
| `MkButton` … `MkWindow` | `CfButton` … `CfWindow` (same files/classes) |
| `pizzax.ts` `Storage` | `pizzax.ts` `Storage` + `createPizzax()` |
| `defaultStore` | `defaultStore` (bucket `base`) |
| `MkPagination` paging object | `Paging` + `usePagination()` |
| `useStream().useChannel` | `useStream().useChannel` (same names) |
| `i18n.ts` singleton | `i18n` + `translate`/`t` + `useLocale` |

## 6. Open questions

1. **Duplicate implementations.** `api/` vs `api-client/`, `store/` vs
   `store-composables/`, `os/` vs `os-dialog-ui/`, `account/` vs
   `account-session/`, `themes/` vs `scaffold-theme/`, `components/ui` vs the
   `ui-primitives` agent's output. The parent must choose one canonical set and
   delete the other before running `vue-tsc`; both are present today.
2. **`i18n.ts` shape.** Should future page ports get a nested `ts` object
   (Firefish-compatible `i18n.ts._ago.justNow`) or stay flat? Flat is what the
   runtime uses; nested would need a Proxy or a second lookup table.
3. **YAML dependency.** Keep the generator `yaml`-based (documented above) or
   add `yaml` to `devDependencies` and wire `gen:locale` into `package.json`.
4. **Theme parity.** Only the two base tables and two Rosé Pine palettes are
   ported; Firefish ships ~28. The JSON5 parser and theme editor are present in
   `themes/json5.ts` from the theme agent.
5. **Locale switching UI.** No settings page consumes `setLocale` yet; needs a
   `SettingsGeneral` port.
6. **Account `lang`.** `MeDetailed.lang` is always `null` in Cinderfish, so
   per-account language cannot be synced; only the device preference applies.
