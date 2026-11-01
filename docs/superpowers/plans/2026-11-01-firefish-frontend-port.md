# Firefish 前端整体移植实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 将 Firefish/Iceshrimp 风格前端整体改写为 Cinderfish 的 Vue 3 + TypeScript + Vite 前端，接入 Misskey 兼容 Go API，并为外站 Follow 联邦链路交付独立后端实现。

**Architecture:** 以现有 `web/src` 的 API client、Streaming、主题、Deck、MFM 和 SPA 静态托管为底座，按 Firefish 的壳、页面信息架构和组件密度重建前端。Firefish/Iceshrimp 的全局运行时用 Cinderfish shim 替换，页面只调用 Cinderfish API client；Misskey 仅作为后端端点契约。ActivityPub Follow 作为独立 Go 任务，前端只消费关系和通知结果。

**Tech Stack:** Go 1.26, PostgreSQL, Vue 3, TypeScript, Vite, vue-router, 原生 WebSocket, 现有 CSS token；不引入 UI 框架、Pinia、虚拟滚动库或新的服务端运行时。

**Spec:** `docs/superpowers/specs/2026-11-01-firefish-frontend-port-design.md`

## Global Constraints

- 生产后端是 Go 1.26 单二进制，前端生产产物由 Go 静态托管。
- 前端保持 Vue 3 + TypeScript + Vite；优先复用现有依赖，不复制 Firefish 已停止维护的构建链。
- Misskey 原生 RPC、Streaming、错误 envelope、游标和字段是后端契约；Misskey 前端不是 UI 参考。
- Firefish 是主要前端结构参考，Iceshrimp Vue fork 是较新的补充来源，Iceshrimp.NET 仅作布局证据。
- 保留现有主题 token、浅色/深色和 `prefers-color-scheme` 自动跟随机制。
- `tmp/refcheck/` 只存参考源码，复制或改写的来源、版本和许可证必须记录；不提交未审查的第三方源码。
- 不维护运行时双前端；回滚通过阶段提交和 `git revert`，不增加旧/新入口 feature flag。
- 不新增 Mastodon API；不因前端移植任意修改数据库 schema。
- 任何新 Misskey API 端点先核对 Misskey `paramDef`、`res` 和 `api.types.ts`。
- 文档中文，代码和代码注释英文；每个完成阶段独立提交。

## Review Focus

- Firefish 页面调用不存在的端点时，必须被能力矩阵拦截或明确降级，不能静默显示空数据；测试：端点能力表覆盖 277 个引用端点（对账口径），标出 26 个与现有 Go 路由同名的端点（`os.api` + `endpoint =` 口径为 25，差额是 XHR 直连的 `drive/files/create`）；首批 43 个白名单端点中 17 个有精确路由。
- 未登录、token 过期和 API 错误 envelope 必须回到登录或显示可理解错误；测试：`ApiError` 的 `code/id/kind/info` 保留并覆盖 401/业务错误。
- 380/600/840/1440px 下 Deck、侧栏和 Note 不得横向溢出；测试：浏览器四个 viewport 的 `scrollWidth === clientWidth`。
- `prefers-color-scheme` 未设置时主题首帧必须正确，手动主题必须优先；测试：浅/深系统偏好与 localStorage 覆盖组合。
- 入站 Follow 的重复、Undo、坏签名和不允许域必须幂等拒绝或更新关系；测试：Go federation handler 的 table tests 与真实 HTTP 签名路径。

---

### Task 1: 参考源码、许可证与能力矩阵

**Files:**
- Create: `docs/research/2026-11-01-firefish-port-source-and-license-register.md`
- Create: `docs/research/2026-11-01-firefish-port-capability-matrix.md`
- Create: `docs/research/2026-11-01-runtime-adaptation-contract.md`
- Modify: `AGENTS.md`
- Modify: `docs/HANDOVER.md`

**Interfaces:**
- Produces: 固定的参考版本、许可证边界、页面到端点的能力矩阵、shim 的 `shim/rewrite/drop` 决策。

- [ ] **Step 1: 记录参考版本与许可证**

固定 Firefish、Iceshrimp Vue fork、Misskey 的 commit、来源 URL、许可证和实际复制边界；Firefish/Iceshrimp 不可构建时记录为只读参考，不把它们当作本地生产依赖。

- [ ] **Step 2: 生成能力矩阵**

以 Firefish 页面与 `os.api` 引用为输入，列出端点、当前 Go 路由、所需字段、状态（已支持/需适配/需新增/隐藏）和降级行为；首批白名单只包含登录、用户、时间线、Note、发帖、反应、通知、媒体、Profile、Settings、主题和 Deck。

- [ ] **Step 3: 固定运行时 shim 契约**

明确 `api`、`apiWithDialog`、`alert`、`confirm`、`popup`、`popupMenu`、`contextMenu`、`inputText`、`post`、`claimZIndex`、单账户 `me`、zh-CN `i18n`、`instance`、vue-router 和 AppIcon 的归属；未支持的 Firefish service worker、多账户、插件和页面明确 drop。

- [ ] **Step 4: 更新项目指引与交接文档**

写明新目标取代旧的渐进式前端设计，记录 `/api/notifications` 为 canonical、`/api/i/notifications` 为兼容 alias 的策略、参考源码不可直接构建的事实，以及 schema 当前版本。

- [ ] **Step 5: 验证并提交**

Run: `git diff --check`
Expected: clean。

```bash
git add AGENTS.md docs/research docs/HANDOVER.md
git commit -m "docs: define Firefish port sources and capability boundaries"
```

---

### Task 2: Cinderfish 运行时适配层与主题 token

**Files:**
- Modify: `web/src/os.ts`
- Modify: `web/src/api/client.ts`
- Create: `web/src/me.ts`
- Create: `web/src/instance.ts`
- Create: `web/src/i18n/index.ts`
- Create: `web/src/i18n/zh-CN.ts`
- Create: `web/src/lib/z-index.ts`
- Create: `web/src/types/menu.ts`
- Create: `web/src/components/ui/DialogHost.vue`
- Create: `web/src/components/ui/PopupMenu.vue`
- Create: `web/src/styles/firefish-tokens.css`
- Modify: `web/src/components/AppIcon.vue`
- Modify: `web/src/stores/session.ts`
- Modify: `web/src/stores/instance.ts`
- Modify: `web/src/main.ts`
- Modify: `web/src/styles/main.css`
- Test: `web/src/lib/*.spec.ts` only if the existing test setup is extended; otherwise browser checks in Task 3.

**Interfaces:**
- Consumes: existing `api/client.ts` request primitives, `ApiError`, `session`, `theme`, `instance` stores.
- Produces: `os.api<T>(endpoint, params): Promise<T>` (delegates to `web/src/api/client.ts` `request()`), `apiWithDialog`, `t(key, params?)`, `me`, `instance`, `showDialog`, `showMenu`, `openComposer`, `claimZIndex`. Theme aliases live in `firefish-tokens.css` + `main.css`; the eleven shim files follow runtime-adaptation-contract §5.

- [ ] **Step 1: Add contract-level checks**

Before implementation, add the smallest available project-native checks for `t()` fallback, API error preservation, token alias presence and single-session `me`; if no frontend test runner exists, record browser scenarios and do not install a test framework solely for this task.

- [ ] **Step 2: Implement the runtime shim**

Map Firefish-shaped calls to the existing Cinderfish API client and stores. `web/src/os.ts` exports the Firefish-shaped `api`/`apiWithDialog` and delegates every request to `request()` in `web/src/api/client.ts`; only that file owns the token (`TOKEN_KEY = "cinderfish.token"`, `getToken`/`setToken`). `apiWithDialog` must preserve `ApiError` fields, `me` must be derived from the single session token, and unsupported operations must return a typed unavailable result or a visible error rather than issue Firefish-only requests.

- [ ] **Step 3: Implement zh-CN i18n and token aliases**

Keep `i18n.ts.*`-shaped access available through a Cinderfish adapter, initially covering migrated screens only. Add Firefish camelCase CSS aliases to existing kebab-case tokens, including accent, panel, divider, note, CW and focus tokens; do not replace the existing theme mechanism.

- [ ] **Step 4: Verify**

Run: `cd web && pnpm typecheck && pnpm build`
Expected: exit 0; Go static hosting remains unchanged.

```bash
git add web/src web/package.json
git commit -m "feat(web): add Firefish runtime and theme compatibility layers"
```

---

### Task 3: Firefish-style application shell, navigation and Deck

**Files:**
- Modify: `web/src/App.vue`
- Modify: `web/src/layouts/AppShell.vue`
- Modify: `web/src/layouts/DeckShell.vue`
- Modify: `web/src/components/layout/Sidebar.vue`
- Modify: `web/src/components/layout/MobileSidebar.vue`
- Modify: `web/src/components/ui/MobileNav.vue`
- Modify: `web/src/components/deck/DeckColumn.vue`
- Modify: `web/src/stores/deck.ts`
- Modify: `web/src/router/index.ts`
- Modify: `web/src/nav.ts`
- Modify: `web/src/os.ts` (implement runtime contract §3.5/§3.15 `openLayer`/`closeLayer`, `popup`, `pageWindow`, `modalPageWindow` and `openEmojiPicker`; verify overlay stacking, window navigation and emoji insertion)

**Interfaces:**
- Consumes: Task 2 runtime and theme contracts; existing `deck` store and vue-router.
- Produces: responsive universal shell, Deck column registry, mobile one-column navigation, stable route/deep-link behavior, and the remaining overlay/window entry points required by migrated Firefish components.

- [ ] **Step 1: Define shell acceptance checks**

Verify expected DOM regions for sidebar, main column, widget column, deck columns, mobile navigation and composer entry; include `aria-current`, focus-visible and keyboard navigation checks.

- [ ] **Step 2: Implement the wide shell**

Use Firefish/Iceshrimp information density: collapsible left system, constrained central content, optional right widgets, flat panel boundaries and no decorative Mastodon-style card stack. Reuse existing token/theme values.

- [ ] **Step 3: Implement adaptive Deck**

Keep `localStorage` persistence and the existing column registry. Use one column on narrow viewports, multiple columns when space allows, and avoid a second route/API system. Keep unsupported column types out of the registry.

- [ ] **Step 4: Implement mobile navigation**

Use a bottom/compact navigation model with reachable compose, home, notifications and profile actions; ensure no fixed element causes horizontal overflow.

- [ ] **Step 5: Verify and commit**

Run: `cd web && pnpm typecheck && pnpm build`
Expected: exit 0.

Browser acceptance: 380, 600, 840 and 1440px; `document.documentElement.scrollWidth === document.documentElement.clientWidth` and shell regions remain reachable.

```bash
git add web/src
git commit -m "feat(web): rebuild Firefish-style shell and adaptive deck"
```

---

### Task 4: Misskey API compatibility and core social closure

**Files:**
- Modify: `web/src/api/client.ts`
- Modify: `web/src/api/types.ts`
- Modify: `web/src/stores/session.ts`
- Modify: `web/src/stores/streaming.ts`
- Modify: `web/src/views/LoginView.vue`
- Modify: `web/src/views/SignupView.vue`
- Modify: `web/src/views/TimelineView.vue`
- Modify: `web/src/views/NotificationsView.vue`
- Modify: `web/src/views/ProfileView.vue`
- Modify: `web/src/components/TimelineFeed.vue`
- Modify: `web/src/components/note/*`
- Modify: `web/src/components/composer/PostForm.vue`
- Modify: `internal/httpapi/server.go` only for explicitly listed compatibility aliases.
- Test: existing Go HTTP/API tests and browser scenarios.

**Interfaces:**
- Consumes: Task 2 runtime and Task 3 shell.
- Produces: working login, current user, timeline, note detail, compose/reply/renote, reactions, notifications, profile, logout and streaming flows using only Cinderfish/Misskey contracts.

- [ ] **Step 1: Freeze endpoint tests**

For every first-release endpoint, add or update contract tests from Misskey `paramDef`/`res`. Keep `POST /api/notifications` canonical (the existing client and tests use it); register `POST /api/i/notifications` as a compatibility alias to the same handler. Never add a second notification implementation.

- [ ] **Step 2: Implement API/type mappings**

Remove Firefish-only endpoint names and map data to existing Cinderfish fields: visibility, localOnly, CW, files, user host/avatar/name, renote, replies, reactions and cursor pagination. Preserve errors and streaming reconnect behavior.

- [ ] **Step 3: Rebuild core pages against the runtime shim**

Migrate the first-release flows only. Hide Firefish pages requiring unavailable endpoints rather than rendering dead links. Keep the current route/deep-link contract for `/@handle` and `/notes/:id`.

- [ ] **Step 4: Verify**

Run: `go test ./internal/httpapi ./internal/federation ./internal/note ./internal/account && cd web && pnpm typecheck && pnpm build`
Expected: exit 0.

Browser acceptance: sign in, load home/local/social/global, create a note, reply, renote, react, open notification and profile, sign out.

```bash
git add internal web/src
git commit -m "feat(web): connect Firefish flows to Misskey-compatible APIs"
```

---

### Task 5: Note, Composer, Profile and Settings fidelity

**Files:**
- Modify: `web/src/components/note/*`
- Modify: `web/src/components/TimelineFeed.vue`
- Modify: `web/src/components/composer/PostForm.vue`
- Modify: `web/src/views/settings/*`
- Modify: `web/src/views/ProfileView.vue`
- Modify: `web/src/components/ProfileHeader.vue`
- Modify: `web/src/lib/mfm.ts`
- Modify: `web/src/styles/main.css`

**Interfaces:**
- Consumes: Task 4 API/types and Task 2 token/i18n contracts.
- Produces: Firefish-density flat notes with header/content/media/footer, CW, visibility, renote/reply bars, MFM, reactions, profile sections and existing settings groups.

- [ ] **Step 1: Define content acceptance cases**

Cover public/home/followers/specified visibility, localOnly, CW collapsed/expanded, reply/renote, media, unicode reaction and malformed/unknown MFM function fallback.

- [ ] **Step 2: Split and restyle Note rendering**

Use flat rows and separators, hover/focus action controls, compact metadata and Firefish/Iceshrimp density. Keep existing safe MFM pipeline and do not add a large renderer dependency without a separate decision.

- [ ] **Step 3: Rebuild Composer behavior**

Preserve draft, reply, renote, file, CW, visibility, localOnly and specified recipient semantics supported by the API. Use the global composer entry from the shell.

- [ ] **Step 4: Align Profile and Settings**

Keep the existing eight settings groups and hide unsupported Firefish settings. Preserve theme auto-follow and manual override.

- [ ] **Step 5: Verify and commit**

Run: `cd web && pnpm typecheck && pnpm build`
Expected: exit 0.

Browser acceptance: dense timeline scan, note actions, composer variants, settings theme switch, system theme change and mobile layout.

```bash
git add web/src
git commit -m "feat(web): match Firefish note composer profile and settings flows"
```

---

### Task 6: Federation Follow protocol for remote users

**Files:**
- Modify: `internal/federation/*`
- Modify: `internal/account/*`
- Modify: `internal/httpapi/*`
- Modify: `internal/db/*` only with a new immutable migration if required by the existing schema.
- Test: `internal/federation/*_test.go`, `internal/httpapi/*_test.go`, database integration tests when configured.

**Interfaces:**
- Consumes: existing ActivityPub actor, WebFinger, HTTP signature and deliver abstractions.
- Produces: verified inbound Follow/Undo, persisted remote relation, Accept/Reject response, policy enforcement, idempotent relation state and delivery queue records. Frontend consumes normal relation/notification data only.

- [ ] **Step 1: Read and pin protocol contracts**

Document the ActivityPub object shapes and signature requirements already used by Cinderfish; identify missing persistence fields without editing an applied migration.

- [ ] **Step 2: Write failing protocol tests**

Cover valid Follow, duplicate Follow, invalid signature, blocked federation domain, unknown actor, Undo(Follow), Accept response and delivery retry.

- [ ] **Step 3: Implement inbound Follow lifecycle**

Resolve and cache remote actor, enforce federation policy, verify HTTP Signature, persist relation idempotently, create local notification and enqueue Accept or Reject.

- [ ] **Step 4: Implement Undo and state reconciliation**

Remove or mark the relation according to current account semantics, keep repeated Undo harmless, and ensure follower/following collections reflect the relation.

- [ ] **Step 5: Verify and commit**

Run: `go test ./internal/federation ./internal/account ./internal/httpapi`
Expected: PASS; with database URL configured, run the full database test command from `AGENTS.md`.

```bash
git add internal
git commit -m "feat(federation): accept and reconcile remote follows"
```

---

### Task 7: Full browser acceptance, cleanup and handover

**Files:**
- Modify: `docs/HANDOVER.md`
- Modify: `docs/research/2026-11-01-firefish-port-capability-matrix.md`
- Modify: `web/src/*` and `internal/*` only for findings from acceptance.

**Interfaces:**
- Consumes: Tasks 1-6.
- Produces: verified release candidate, updated capability matrix and handover evidence.

- [ ] **Step 1: Run all standard gates**

Run `gofmt -l internal/ cmd/`, `go vet ./...`, `go test ./...`, configured database tests, `cd web && pnpm typecheck && pnpm build`, and `git diff --check`.

- [ ] **Step 2: Run real browser scenarios**

Use the built-in Kimi browser against a running Cinderfish binary. Test login, timeline, compose, notifications, profile, settings, light/dark/system theme, Deck and 380/600/840/1440px layouts. Record any unverified scenario instead of claiming it passed.

- [ ] **Step 3: Review capability boundaries**

Ensure every hidden or degraded Firefish feature is listed with its reason and endpoint requirement. Remove stale imports, dead routes and old comments describing the pre-port architecture.

- [ ] **Step 4: Commit final evidence**

```bash
git add docs web internal
git commit -m "docs: record Firefish port acceptance and remaining gaps"
```

- [ ] **Step 5: Final review**

Generate the whole-branch review package, obtain an independent review, fix Critical/Important findings with failing tests first, and retain Minor findings in the ledger for the user.

## Plan Self-Review

- Spec coverage: source/license register and capability matrix are Task 1; runtime shim and responsive themes Task 2; shell/Deck Task 3; Misskey API and core closure Task 4; content fidelity Task 5; ActivityPub Follow Task 6; browser and gates Task 7.
- Shared interfaces: Task 2 precedes Tasks 3-5; Task 3 shell precedes Task 4 pages; Task 4 API/types precedes Task 5 content; Task 6 is independent of frontend but shares account/notification contracts; Task 7 consumes all.
- Deliberate boundary: no Firefish backend contract is copied; only frontend structure and interaction are ported. Unsupported features are hidden via capability matrix.
- Test gap to resolve during execution: the repository currently has no frontend test script, so browser acceptance and existing Go contract tests are authoritative until a focused test runner is justified by a concrete pure-logic regression.
