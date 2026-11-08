# Mobile Navigation and Overlay UX Repair Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Remove the mobile UX defects reported in settings navigation, popup menus, bottom navigation, narrow composer layout and tooltips, while restoring frontend work as an explicit repository scope.

**Architecture:** Fix shared components and state at the source while preserving desktop presentation and existing route/API semantics. Use responsive CSS and measured popup positioning; keep the settings menu available beside the child view on narrow screens.

**Tech Stack:** Vue 3, TypeScript, Vite, SCSS, Vue Router.

**Spec:** `docs/superpowers/specs/2026-11-08-mobile-ux-repair-design.md`

## Global Constraints

- Keep the API and backend/federation behavior unchanged.
- Do not deploy to or interact with the public production instance.
- Do not add dependencies; use the existing Vue/TypeScript/Vite/SCSS/Phosphor stack.
- Preserve AGPL-3.0 source headers and document rewritten upstream files in `docs/HANDOVER.md`.
- Keep database migrations immutable; this work does not need migrations.

## Review Focus

- Settings route starts narrow but crosses the 600px container threshold after mounting; redirect once based on current measured width, not stale width.
- A long settings menu must remain horizontally scrollable without shrinking/hiding the current child page.
- Popup content at each viewport edge must remain inside the visual viewport, including taller menus and mobile browser chrome.
- The composer may contain long usernames, counts, poll controls, attachments and optional hashtags simultaneously; no control may overlap or force page-level horizontal scrolling.
- Tooltips must remain above a modal even when opened after that modal and must not be clipped at any viewport edge.

---

### Task 1: Restore frontend repository scope

**Files:**
- Modify: `AGENTS.md:20-37`

**Interfaces:**
- Produces: one authoritative project scope that permits ongoing frontend and backend development while retaining the Firefish port and source/license policies.

- [ ] **Step 1: Replace the obsolete scope lock**

Remove the 2026-11-04 statement that work is backend-only and frontend must not resume. State that both frontend and backend are active project areas; keep the existing frontend baseline and its references as guidance rather than a receiving-workflow handoff.

- [ ] **Step 2: Check for contradictory scope statements**

Run: `rg -n "backend API and federation only|Do not resume frontend|separate workflow|handed to a separate" AGENTS.md`
Expected: no scope restriction remains; historical facts elsewhere are not rewritten.

- [ ] **Step 3: Commit**

```bash
git add AGENTS.md
git commit -m "docs: restore frontend as active project scope"
```

### Task 2: Keep mobile settings navigation available

**Files:**
- Modify: `web/src/pages/SettingsIndex.vue`

**Interfaces:**
- Consumes: existing `narrow`, `hasChild`, `menuDef`, router and `MkSuperMenu` interfaces.
- Produces: narrow route behavior where `/settings` resolves to `/settings/profile` and settings navigation remains rendered on child routes.

- [ ] **Step 1: Update narrow template and redirect behavior**

Render the settings menu on narrow screens when a child is open, while retaining the existing child page. Change the empty-route redirect so `/settings` redirects to `/settings/profile` regardless of width; preserve resize/activation behavior and desktop layout.

- [ ] **Step 2: Validate TypeScript and production build**

Run from `web/`: `pnpm typecheck && pnpm build`
Expected: both pass.

- [ ] **Step 3: Browser-check settings navigation**

At 390×844, open `/settings`, verify it ends at `/settings/profile`, then tap Privacy and Notifications from the still-visible horizontal menu without a back-navigation step. At 1280×900 verify the vertical menu + child layout remains side-by-side.

- [ ] **Step 4: Commit**

```bash
git add web/src/pages/SettingsIndex.vue
git commit -m "fix(web): keep mobile settings navigation available"
```

### Task 3: Improve popup menu touch readability

**Files:**
- Modify: `web/src/os/CfPopupMenu.vue`

**Interfaces:**
- Consumes: existing `MenuItem` and popup positioning contracts.
- Produces: readable, touch-sized menu entries on narrow screens without reducing desktop menu usability.

- [ ] **Step 1: Adjust menu entry icon and text sizing**

Raise the menu row label to a readable 1rem-equivalent and define a stable 20–22px icon box; provide at least 44px row height through padding/min-block-size. Keep caption visually secondary and preserve scrolling/clamping.

- [ ] **Step 2: Validate at mobile and desktop widths**

Run: `pnpm typecheck && pnpm build` from `web/`; inspect a multi-item menu at 390×844 and 1280×900.
Expected: text and icons are legible, every row has a comfortable hit target, and menu remains within viewport.

- [ ] **Step 3: Commit**

```bash
git add web/src/os/CfPopupMenu.vue
git commit -m "fix(web): improve popup menu touch targets"
```

### Task 4: Remove decorative underline from bottom navigation

**Files:**
- Modify: `web/src/styles/_reset.scss`
- Modify if needed: `web/src/ui/layouts/UniversalLayout.vue`

**Interfaces:**
- Produces: links styled as navigation/buttons suppress hover underline while content links retain underline affordance and keyboard focus stays visible.

- [ ] **Step 1: Scope link decoration to content links**

Use the existing `_button`/navigation class convention to prevent hover underlines on bottom navigation links, without globally disabling link affordance for article/content links. Preserve `:focus-visible` outline.

- [ ] **Step 2: Verify interaction and builds**

Run: `pnpm typecheck && pnpm build` from `web/`; at 390×844 hover/focus each bottom navigation item and confirm no underline; verify an ordinary content link still shows its hover affordance.

- [ ] **Step 3: Commit**

```bash
git add web/src/styles/_reset.scss web/src/ui/layouts/UniversalLayout.vue
git commit -m "fix(web): scope link hover styling away from navigation"
```

### Task 5: Prevent narrow composer control overlap

**Files:**
- Modify: `web/src/components/MkPostForm.vue`
- Modify only if confirmed necessary: `web/src/os/MkPostFormDialog.vue`

**Interfaces:**
- Consumes: existing `v-size` container classes and composer controls.
- Produces: header controls and textarea/footer that fit 320–412px CSS widths without overlap or horizontal overflow.

- [ ] **Step 1: Reproduce and isolate the layout boundary**

Use browser responsive widths 320, 360, 390 and 412 CSS px with the mobile composer open. Record which header controls overlap and whether the footer wraps; include the 3000 character counter and all toolbar controls.

- [ ] **Step 2: Apply minimal container-responsive sizing**

Use the existing size directive classes to shrink/grow the account/header action regions appropriately, permit safe wrapping when necessary, and keep textarea width constrained to its parent. Keep target sizes >=44px where the layout allows; do not hide functionality.

- [ ] **Step 3: Verify composer behavior**

Run: `pnpm typecheck && pnpm build`; repeat the four widths and check no control overlaps, toolbar buttons remain reachable, and page `scrollWidth` does not exceed `clientWidth` due to composer.

- [ ] **Step 4: Commit**

```bash
git add web/src/components/MkPostForm.vue web/src/os/MkPostFormDialog.vue
git commit -m "fix(web): prevent composer controls overlapping on narrow screens"
```

### Task 6: Keep tooltip above its source popup and within viewport

**Files:**
- Modify: `web/src/components/CfTooltip.vue`
- Modify: `web/src/os/state.ts` only if investigation confirms z-index counters cause the ordering issue.

**Interfaces:**
- Consumes: `source`, `direction`, `showing`, and the existing tooltip popup contract.
- Produces: tooltip above the modal/form containing its trigger, clamped using its actual rendered dimensions.

- [ ] **Step 1: Reproduce z-order and edge positioning**

Open the post composer modal, focus/hover each footer toolbar button, and reproduce a tooltip above the composer. Test triggers near each viewport edge.

- [ ] **Step 2: Fix tooltip layer and measured clamping**

Set tooltip layer above the source dialog layer using existing z-index bands; account for popup creation order. Measure the rendered tooltip after mount and clamp both axes using its actual bounding box with a small viewport margin, respecting top/bottom direction. Retain existing delays, transitions and pointer behavior.

- [ ] **Step 3: Verify tooltip behavior**

Run: `pnpm typecheck && pnpm build`; test hover and keyboard focus inside the modal at 320, 390 and 412px. Confirm full text is visible above the dialog and at all edges; verify leave/click dismissal is unchanged.

- [ ] **Step 4: Commit**

```bash
git add web/src/components/CfTooltip.vue web/src/os/state.ts
git commit -m "fix(web): keep tooltips visible above popup forms"
```

### Task 7: Whole-batch verification and handover

**Files:**
- Modify: `docs/HANDOVER.md` (append a concise dated record with verified behaviors and command/browser evidence).

- [ ] **Step 1: Run the web project gates**

Run from `web/`: `pnpm typecheck && pnpm run check:port && pnpm build`
Expected: all pass.

- [ ] **Step 2: Re-run responsive scenarios and inspect workspace**

Verify 320/360/390/412 widths and 1280px desktop settings/nav behavior. Run `git status --short` and ensure no build artifacts or screenshots are added unintentionally.

- [ ] **Step 3: Record evidence and commit**

Update `docs/HANDOVER.md` with actual results only, then commit:

```bash
git add docs/HANDOVER.md
git commit -m "docs: record mobile UX repair verification"
```
