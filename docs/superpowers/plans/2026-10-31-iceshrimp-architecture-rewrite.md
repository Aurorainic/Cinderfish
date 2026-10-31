# Iceshrimp Frontend Architecture Rewrite Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rewrite Cinderfish’s Vue frontend around Iceshrimp’s page/layout/component architecture while keeping the Go backend’s Misskey-native API contract as the data and behavior source.

**Architecture:** Translate Iceshrimp’s Sidebar/Main/WidgetPanel, top-bar/section actions, page composition and note/feed boundaries into Vue 3 components and vue-router routes; do not port Blazor code or adopt Misskey frontend internals. Keep existing API/session/streaming/theme stores where sound, add or correct only Misskey-contracted Go endpoints needed by user-visible flows, and migrate screens in small runnable waves.

**Tech Stack:** Go 1.26, PostgreSQL, Vue 3, TypeScript, Vite, vue-router, CSS custom properties, pnpm. No new frontend dependencies.

**Spec:** `/home/lainsaka/.kimi-code/sessions/wd_milnfish_b7e724046d22/session_32633931-e964-4b04-90de-ec435cab5b97/agents/main/plans/tempest-sam-alexander-nebula.md` (approved design and scope)

## Global Constraints

- Misskey backend source `paramDef`, `res`, and `packages/misskey-js/src/api.types.ts` are authoritative before implementing an API endpoint.
- Iceshrimp Vue fork is the architecture/component reference; Iceshrimp.NET is limited to layout/interaction evidence. Do not port C# or Blazor.
- Preserve Go + Vue 3 + TypeScript + Vite + vue-router; do not introduce an UI framework, new state framework, drag/drop package, or virtual scroller.
- No Mastodon API compatibility endpoints or Misskey frontend `Mk*`/nirax/`os` architecture.
- Frontend must not fabricate unavailable remote relationships, thread data, reaction lists, or instance data.
- Preserve existing `auto/light/dark`, ember accent, Inter font, hand-written SVG icon set, Chinese UI copy, and API error envelope.
- Keep applied migrations immutable. Add a new migration version only when the data model requires it.
- Update `docs/HANDOVER.md` when a milestone changes and commit each independently verified task.

## Review Focus

1. Anonymous and invalid-token requests must still render public pages without viewer-only state; pin this in the `users/show` and route-gating tasks.
2. An account with a pending follow request must not be offered a duplicate follow action; pin this in the profile/API task.
3. Empty, filtered, and fully paged notification feeds must not skip records or mark them read unexpectedly; pin this in the notifications task.
4. Narrow screens and resized Deck columns must not overflow horizontally or hide reachable navigation/actions; pin this in shell and browser verification tasks.
5. Nested/quoted/pure-renote notes must preserve the distinction between displayed content and the acted-on note; pin this in the Note task.

---

## File and Interface Map

| Area | Files and responsibility |
|---|---|
| API and session boundary | `web/src/api/client.ts`, `web/src/api/types.ts`, `web/src/stores/session.ts`; API calls, token handling, typed Misskey payloads |
| Application shell | `web/src/App.vue`, `web/src/layouts/AppShell.vue`, `web/src/layouts/DeckShell.vue`, new `web/src/components/layout/Sidebar.vue`, `WidgetPanel.vue`, `MobileSidebar.vue` |
| Shared section header | `web/src/components/ui/PageHeader.vue`; title, back, tabs, actions via slots/props |
| Navigation | `web/src/nav.ts`, `web/src/stores/nav.ts`; one route/item registry consumed by desktop, mobile and drawer navigation |
| Page composition | `web/src/views/*`, `web/src/views/settings/*`, `web/src/views/admin/*`; Iceshrimp-style page sections and route/query state |
| Timeline and notes | `web/src/components/TimelineFeed.vue`, `PagedFeed.vue`, `components/note/*`; date grouping, stream updates, note display and interactions |
| Widgets | `web/src/widgets/registry.ts`, `web/src/stores/widgets.ts`, `web/src/components/widgets/*`; panel contents and optional user ordering |
| Go API | `internal/httpapi/*.go`, `internal/note/*`, `internal/account/*`, `internal/federation/*`; Misskey-contract endpoints and server-side filtering |
| Verification and records | Existing Go tests adjacent to packages, `docs/HANDOVER.md`, `docs/research/` for upstream contract evidence |

## Tasks

### Task 1: Revalidate Iceshrimp source map and freeze the Vue translation contract

**Files:**
- Create: `docs/research/2026-10-31-iceshrimp-architecture-translation.md`
- Read: `tmp/refcheck/iceshrimp-vue/packages/client/src/` when present
- Read: `tmp/refcheck/iceshrimp/Iceshrimp.Frontend/`
- Read: `web/src/App.vue`, `web/src/layouts/AppShell.vue`, `web/src/layouts/DeckShell.vue`, `web/src/nav.ts`, `web/src/components/ui/PageHeader.vue`

**Interfaces:**
- Produces the source-path-to-Vue-component mapping used by Tasks 2–8.
- Record which evidence comes from Vue Iceshrimp and which only from Iceshrimp.NET; do not treat a missing clone as verified source.

- [ ] **Step 1: Inspect the available reference tree and document exact source paths** for MainLayout, Sidebar, WidgetPanel, section header, note, timeline, notifications, profile and settings. Clearly mark unavailable Vue-fork paths as unavailable rather than inferring them.
- [ ] **Step 2: Compare current Vue ownership boundaries** against that map and list only files that need migration; avoid copying Iceshrimp business services or backend calls.
- [ ] **Step 3: Record the translation contract** in the research note, including Vue component names, responsive behavior evidenced by source, and explicit Cinderfish/API differences.
- [ ] **Step 4: Verify references and working tree** with `git status --short` and review every cited path before committing.
- [ ] **Step 5: Commit** `docs(research): record the Iceshrimp Vue architecture mapping`.

### Task 2: Build the Iceshrimp-style shell and shared section header

**Files:**
- Modify: `web/src/App.vue`
- Modify: `web/src/layouts/AppShell.vue`
- Modify: `web/src/layouts/DeckShell.vue`
- Modify: `web/src/components/ui/PageHeader.vue`
- Create: `web/src/components/layout/Sidebar.vue`
- Create: `web/src/components/layout/WidgetPanel.vue`
- Create: `web/src/components/layout/MobileSidebar.vue`
- Test: browser geometry checks; no frontend test runner is currently installed

**Interfaces:**
- `Sidebar.vue` consumes the navigation definition exported from `web/src/nav.ts` and emits `navigate`/`close` only for shell actions.
- `WidgetPanel.vue` renders existing widget components and does not fetch its own unrelated data.
- `PageHeader.vue` accepts `title: string`, optional `back: boolean`, optional `tabs`, and exposes `#actions` slot; pages own tab destinations.
- `AppShell.vue` composes `Sidebar | main slot | WidgetPanel`; mobile layout uses the same nav definition through `MobileSidebar.vue`.

- [ ] **Step 1: Add a minimal shell-level contract check** using existing project conventions (if no browser-test harness exists, use a manual browser checklist recorded in the task report; do not add a new test framework).
- [ ] **Step 2: Implement the three-region shell** in `AppShell.vue` and route the existing page view through its main slot; keep authentication layout bare.
- [ ] **Step 3: Implement shared `PageHeader.vue`** with sticky positioning, title/back/tabs/actions and narrow-width action-label collapse.
- [ ] **Step 4: Implement desktop Sidebar and WidgetPanel adapters** that reuse existing `nav.ts` and widget components without duplicate route or data definitions.
- [ ] **Step 5: Implement mobile Sidebar presentation** as a drawer plus bottom navigation, retaining keyboard focus, Escape/overlay close and route-change close behavior.
- [ ] **Step 6: Run `cd web && pnpm typecheck && pnpm build`**; verify no route loses its shell and login/signup remain bare.
- [ ] **Step 7: Browser-check 1440, 1100, 840, 600 and 380 px** for three-region layout, widget collapse, reachable navigation, no horizontal overflow and safe-area bottom spacing; record measured `scrollWidth/clientWidth`.
- [ ] **Step 8: Commit** `feat(web): establish the Iceshrimp-style application shell` with verification evidence.

### Task 3: Consolidate navigation, route state and account gates

**Files:**
- Modify: `web/src/nav.ts`
- Modify: `web/src/stores/nav.ts`
- Modify: `web/src/router/index.ts`
- Modify: `web/src/components/layout/Sidebar.vue`
- Modify: `web/src/components/layout/MobileSidebar.vue`
- Modify: `web/src/views/settings/SettingsMenu.vue`
- Test: add route/store tests only if an existing frontend test harness is present; otherwise verify via browser deep links

**Interfaces:**
- Navigation item definition includes `id`, `to`, `label`, icon key, `requiresAuth`, `mobile`, and optional unread indicator key.
- Route metadata uses `requiresAuth` for account-only settings/admin actions; client appearance preferences remain available without sign-in.
- `nav.ts` is the only source consumed by desktop nav, mobile nav, drawer and settings shortcut links.

- [ ] **Step 1: Enumerate current navigation consumers and route requirements**; retain existing paths and query semantics.
- [ ] **Step 2: Define one typed navigation registry** and migrate Sidebar, mobile drawer, bottom bar and settings shortcuts to it.
- [ ] **Step 3: Add `requiresAuth` route metadata and a router guard** that checks the already-loaded session state, sends unauthenticated users to `/login?redirect=<fullPath>`, and does not block public profile/note routes.
- [ ] **Step 4: Verify direct deep-link, refresh, back navigation, anonymous appearance settings, and unauthenticated account settings** in browser.
- [ ] **Step 5: Run frontend typecheck/build and commit** `refactor(web): centralize navigation and account route gates`.

### Task 4: Align Misskey user relationships and follow state end to end

**Files:**
- Verify/update: `web/src/api/types.ts`, `web/src/api/client.ts`
- Verify/update: `internal/httpapi/account.go`, `internal/httpapi/users_pack.go`, `internal/federation/store.go`
- Modify: `web/src/views/ProfileView.vue`, `web/src/components/deck/DeckPageWindow.vue`, `web/src/components/ProfileHeader.vue`
- Test: `internal/httpapi/users_relations_test.go`, `internal/httpapi/following_test.go`

**Interfaces:**
- `UserDetailed` viewer relation fields: `isFollowing`, `isFollowed`, `hasPendingFollowRequestFromYou`, `hasPendingFollowRequestToYou`.
- Profile control derives following/pending state from `api.usersShow()`; pending state disables duplicate follow and displays a pending label.
- `users/show` remains anonymous-accessible; invalid credentials degrade to anonymous, not 401.

- [ ] **Step 1: Read the current Misskey `users/show` paramDef/res and UserDetailed type**, confirming all relationship flags and their direction.
- [ ] **Step 2: Add/adjust Go tests** for anonymous, invalid-token, following, followed and pending request directions; assert fields always exist as booleans.
- [ ] **Step 3: Implement only the contract-backed relation packing** and database query helpers needed for the tests.
- [ ] **Step 4: Update ProfileHeader API** with an explicit pending state and make ProfileView and DeckPageWindow initialize from server-returned relationship fields.
- [ ] **Step 5: Verify test failure before implementation and pass after** using `CINDERFISH_TEST_DATABASE_URL=... go test ./internal/httpapi -run 'TestUsersShowRelation|TestFollowing' -count=1`.
- [ ] **Step 6: Verify frontend typecheck/build and commit** `fix(web): render server-resolved follow state` (split backend/API into its own commit if independently complete).

### Task 5: Rebuild Timeline and notifications around Iceshrimp feed behavior

**Files:**
- Modify: `web/src/views/TimelineView.vue`, `web/src/views/NotificationsView.vue`, `web/src/components/TimelineFeed.vue`, `web/src/components/ui/PagedFeed.vue`, `web/src/components/NotificationItem.vue`, `web/src/components/widgets/NotificationsWidget.vue`
- Modify: `internal/httpapi/notifications.go`, `internal/note/reaction.go` (only where existing notification persistence/query owns filtering)
- Test: `internal/httpapi/notifications_test.go` or existing notification test file; add `internal/note` tests adjacent to list/query code

**Interfaces:**
- `api.notifications({ limit, untilId, includeTypes, excludeTypes, markAsRead })` passes Misskey parameter names unchanged.
- Server-side filtering occurs before pagination; `markAsRead` preserves the upstream `paramDef` default and explicit request semantics.
- Notification item presentation receives typed notification data and renders action, actor, related note and read state without fetching per row.

- [ ] **Step 1: Read Misskey `i/notifications` paramDef/res and `misskey-js` type**, including `markAsRead` default, `includeTypes`/`excludeTypes`, and cursor behavior.
- [ ] **Step 2: Add failing Go tests** for include-only, exclude-only, combined filters, empty results, cursor pagination, and mark-as-read default/explicit false.
- [ ] **Step 3: Implement DB-level notification filtering before limit/cursor selection** and preserve exact upstream default for `markAsRead`.
- [ ] **Step 4: Add/adjust typed API client params** and move UI filtering out of the rendered-page layer.
- [ ] **Step 5: Update NotificationItem and the shared widget** for Iceshrimp-style action labels, read opacity, and related-note presentation.
- [ ] **Step 6: Run focused Go tests, full Go checks and frontend typecheck/build**; verify filtered pages neither skip nor duplicate notifications.
- [ ] **Step 7: Commit** `fix(api): apply Misskey notification filters before paging` and `feat(web): present notifications with Iceshrimp page composition` as separate commits if review boundaries are clear.

### Task 6: Align Note, TimelineFeed and Composer component boundaries

**Files:**
- Modify: `web/src/components/note/Note.vue`, `NoteHeader.vue`, `NoteContent.vue`, `NoteFooter.vue`, `NoteMedia.vue`, `NoteParent.vue`, `RenoteBar.vue`, `NoteReactions.vue`, `use-note.ts`
- Modify: `web/src/components/TimelineFeed.vue`, `web/src/components/ui/PagedFeed.vue`, `web/src/components/composer/PostForm.vue`
- Modify API only as required: `web/src/api/client.ts`, `web/src/api/types.ts`, corresponding `internal/httpapi` handlers
- Test: backend integration tests for every new or changed endpoint; browser flows for frontend interactions

**Interfaces:**
- Displayed note, outer timeline entity, and mutation target remain separately typed/identified for pure renote and quote cases.
- `PagedFeed` owns cursor progression, date separators, initial/loading/empty/error states and stream insert coordination; page views provide fetcher and row renderer.
- Composer remains a Vue component with explicit props/events and calls only typed `api.*` functions.

- [ ] **Step 1: Read Misskey note endpoint paramDef/res for the flows this task touches** (timeline, show, create, delete, reactions); do not add a note endpoint without its upstream contract.
- [ ] **Step 2: Add/adjust tests for pure-renote action target, quote, reply context and pagination boundaries** using existing test harnesses; for frontend-only behavior use a written scenario matrix until a test harness exists.
- [ ] **Step 3: Reshape Note subcomponents to the Iceshrimp Vue component boundaries** found in Task 1 while keeping mutation targets explicit and current behavior intact.
- [ ] **Step 4: Implement date-boundary grouping in the user’s local timezone** with full year/month/day comparison, not day-of-month only; keep rows stable during pagination.
- [ ] **Step 5: Ensure composer owns draft, visibility, reply/renote target and uploaded-file state** while API calls remain in `api/client.ts`.
- [ ] **Step 6: Verify public/home/followers/specified visibility, pure renote delete/pin, quote/reply, long-content/CW and attachment rendering** in browser plus focused Go tests.
- [ ] **Step 7: Run all applicable Go/frontend gates and commit** `refactor(web): align notes and feeds with Iceshrimp component boundaries`.

### Task 7: Migrate Profile, Settings, Admin and widget panel

**Files:**
- Modify: `web/src/views/ProfileView.vue`, `web/src/components/ProfileHeader.vue`, `web/src/components/profile/*`
- Modify: `web/src/views/SettingsView.vue`, `web/src/views/settings/*`, `web/src/views/AdminView.vue`, `web/src/views/admin/*`
- Modify: `web/src/widgets/registry.ts`, `web/src/stores/widgets.ts`, `web/src/components/widgets/*`, `web/src/components/layout/WidgetPanel.vue`
- Modify API only for endpoints explicitly verified against Misskey source

**Interfaces:**
- Profile tabs use `route.query.tab`; unsupported followers/following lists show a truthful unavailable state until their endpoint exists.
- Settings layout owns its sidebar and renders one nested child route; admin remains protected by backend authorization and frontend route gating.
- Widget registry identifies stable widget IDs and component loaders; each widget uses existing API/store data and has a defined loading/empty/error state.

- [ ] **Step 1: Compare current profile/settings/admin/widget pages with the Task 1 Iceshrimp map** and list reused versus replaced components.
- [ ] **Step 2: Add tests for route/query persistence and existing admin server authorization**; do not rely on hidden navigation as an authorization control.
- [ ] **Step 3: Refactor Profile tabs and sections** to Iceshrimp composition while retaining accurate relationship state, pinned notes and current API limitations.
- [ ] **Step 4: Refactor Settings and Admin** to consistent sidebar/content layout using route children and server-enforced permissions.
- [ ] **Step 5: Refactor WidgetPanel/widgets** to explicit title/action/content slots and existing local ordering; hide/degrade them at documented breakpoints.
- [ ] **Step 6: Verify anonymous/self/other profile, admin/non-admin, settings deep links, empty widget and widget failure states** in browser and Go integration tests.
- [ ] **Step 7: Update `docs/HANDOVER.md`, run all gates and commit** `refactor(web): migrate profile settings admin and widgets`.

### Task 8: API gap closure and final architecture cleanup

**Files:**
- Modify only endpoint-specific Go files selected after upstream source verification.
- Modify only corresponding `web/src/api/*` and page components.
- Modify: `web/src/layouts/*`, `web/src/components/layout/*`, `web/src/nav.ts`, `docs/HANDOVER.md`
- Test: new Go contract tests adjacent to each implemented endpoint; browser smoke checklist

**Interfaces:**
- Each added endpoint has a written Misskey source location, exact request/response type, error behavior, persistence mapping and frontend consumer.
- Unsupported features remain explicit and cannot silently return fabricated success data.

- [ ] **Step 1: Audit remaining Iceshrimp page actions against existing API routes** and group gaps into (a) upstream-contracted and backed by existing data, (b) upstream-contracted but requiring schema/work, (c) absent from Misskey API.
- [ ] **Step 2: Implement only group (a) in this plan**; group (b) becomes a separately scoped migration/data-model plan, group (c) remains out of scope.
- [ ] **Step 3: Remove obsolete Cinderfish shell/navigation/page wrappers** only after `Grep` confirms no route/import/store references.
- [ ] **Step 4: Run final backend and frontend gates** exactly:

```bash
gofmt -l internal/ cmd/
go vet ./...
CINDERFISH_TEST_DATABASE_URL="postgres://cinderfish:cinderfish@localhost:5432/cinderfish_test" CINDERFISH_TEST_REDIS_URL="redis://127.0.0.1:6379/0" go test -count=1 ./...
cd web && pnpm typecheck && pnpm build
```

- [ ] **Step 5: Run real browser/API smoke** at 1440, 1280, 1100, 840, 720, 600 and 380 px; exercise anonymous/authenticated page loads, timeline sources, notifications, post/reply/react/renote, follow state, settings and Deck; record overflow measurements and any endpoint downgrade.
- [ ] **Step 6: Update `docs/HANDOVER.md`** with new frontend architecture, commit SHAs, exact verification output, and remaining API gaps.
- [ ] **Step 7: Request a whole-branch review** before suggesting merge; do not merge or push without a separate user instruction.

## Execution Notes

- Execute in dependency order; Tasks 4 and 5 may run in parallel only after Task 1 and with disjoint Go/frontend file ownership. Tasks 6–8 depend on shell/header contracts from Tasks 1–3.
- Every task should follow test-first for behavior changes, run its smallest relevant verification before broader gates, and commit immediately when its own evidence is green.
- The currently supported named subagent model list may not contain `LithosAI-glm-5.3-flash`; if unavailable, report that constraint and use the available model chosen by the user/runtime rather than claiming otherwise.
- Do not alter unrelated branch history, merge, or push.
