# Project Guidance

> This is the single source of project rules. `AGENT.md` is a pointer to this file.

## Project Status

This repository is an exploratory project inspired by a private Go reimplementation of Misskey. `Cinderfish` is the project name (adopted 2026-10-28).

The intended product is a personal or very small-user federated social instance:

- Frontend experience should be close to Firefish where practical.
- Misskey compatibility and ActivityPub federation are first-class goals.
- Pleroma/Akkoma are product-positioning references only; do not use their source code or architecture as implementation references unless explicitly requested.
- The first release prioritizes personal-instance usability over full Misskey feature parity.
- Registration modes must include open registration, invite-only registration, and administrator-created accounts.
- Federation modes must include public federation, allowlist-only federation, and federation disabled, with runtime configuration switching.
- Media storage must support both local filesystem storage and S3-compatible object storage.
- The private Go reimplementation and its benchmark table are not public source material. Its measurements are feasibility references, not this project's acceptance criteria unless explicitly adopted later.

**Scope change (2026-11-04):** the frontend port has been handed to a separate workflow outside this
repository session. Work in this repository is now **backend API and federation only**: Misskey-compatible
endpoints under `internal/httpapi`, ActivityPub in `internal/federation`, their stores and tests. Progress is
tracked against `docs/research/2026-11-04-misskey-api-coverage.md` (implemented / missing endpoint register).
Do not resume frontend work here; the direction below is recorded for the receiving workflow.

**Frontend direction (settled 2026-11-01, frozen here):** the frontend is a whole-port of the Firefish client structure onto
Cinderfish, not an incremental restyling of the current pages. The new baseline supersedes the earlier
"progressive redesign" of the existing components (`docs/superpowers/plans/2026-10-29-frontend-redesign/`); work
finished there on the backend/API side stays, the frontend components built there are not extended further.
The port is defined by three documents, which are the authority for any frontend task:

- `docs/superpowers/specs/2026-11-01-firefish-frontend-port-design.md` — goals, non-goals and phases.
- `docs/research/2026-11-01-firefish-port-source-and-license-register.md` — pinned reference commits, licenses and the copy boundary.
- `docs/research/2026-11-01-firefish-port-capability-matrix.md` — page to endpoint matrix, status per capability and its fallback.
- `docs/research/2026-11-01-runtime-adaptation-contract.md` — which Firefish runtime member is `shim` / `rewrite` / `drop`.

Implementation progress is tracked in `docs/HANDOVER.md` and the milestone table in the architecture document.

## Decided Scope

These decisions are settled; do not reopen them without an explicit request.

| Area | Decision |
|------|----------|
| Language | Go 1.26 |
| Frontend | Vue 3 + TypeScript + Vite |
| Frontend baseline | **技术栈替换的整体移植（2026-11-04）：从零重建 `web/`，按 Firefish 客户端 `packages/client/src` 的结构逐文件翻译；旧前端全部废弃** |
| Frontend runtime | Firefish's `os.*` surface is provided as shims over Cinderfish code; see the runtime adaptation contract |
| Datastore | PostgreSQL is the only mandatory dependency; Redis is optional behind the queue abstraction and disabled by default |
| Mastodon API | Explicitly abandoned — no endpoints, no reserved routes |
| Notification endpoint path | `POST /api/notifications` is the canonical path (the existing client and tests use it). Upstream Misskey calls it `i/notifications`, so register `POST /api/i/notifications` as a compatibility alias to the same handler. Never add a second notification implementation |
| Federation vocabulary | Misskey's own `all` / `specified` / `none`, mapped from the configured `public` / `allowlist` / `none` |
| Data migration | No migration of the private project's data; instances start empty |
| Relay | Out of scope for the first release |

## Research Rules

- Keep research notes and source links in the repository so decisions remain auditable.
- **The Misskey source tree is the authoritative API contract.** Before implementing an endpoint, read its `paramDef` and `res` schema in `packages/backend/src/server/api/endpoints/`, then the type definitions in `packages/misskey-js/src/api.types.ts`. Official documentation is secondary and has been found to disagree with the source in several places.
- **Reference roles are fixed and must not be mixed up.** Misskey supplies the backend API contract only — never its UI, component structure, layout or visual language. Frontend experience is referenced from Firefish (primary) and Iceshrimp (secondary), considered together. Do not claim UI work is "Misskey-aligned"; the UI acceptance standard is Firefish/Iceshrimp information density and interaction.
- The reference clones live in `tmp/refcheck/`, which `.gitignore` covers, so they survive a reboot. Recreate them with:
  ```bash
  export https_proxy=http://127.0.0.1:19823 http_proxy=http://127.0.0.1:19823   # Misskey only
  # Misskey — API contract
  git clone --depth 1 --filter=blob:none --sparse https://github.com/misskey-dev/misskey.git tmp/refcheck/misskey
  cd tmp/refcheck/misskey && git sparse-checkout set packages/backend/src packages/misskey-js/src packages/frontend/src
  cd ../..   # back to the repository root before the next clone
  # Firefish — primary frontend reference (Codeberg, no proxy needed)
  git clone --depth 1 --filter=blob:none --sparse https://codeberg.org/firefish/firefish.git tmp/refcheck/firefish
  cd tmp/refcheck/firefish && git sparse-checkout set packages/client/src packages/sw/src
  cd ../..
  # Iceshrimp — supplemental frontend reference
  git clone --depth 1 --filter=blob:none --sparse https://codeberg.org/iceshrimp/Iceshrimp.NET.git tmp/refcheck/iceshrimp
  cd tmp/refcheck/iceshrimp && git sparse-checkout set Iceshrimp.Frontend
  cd ../..
  # Iceshrimp Vue fork — component mirror; the local snapshot has no .git and holds only packages/
  mkdir -p tmp/refcheck/iceshrimp-vue
  curl -L https://codeberg.org/banaanihillo/iceshrimp/archive/1fa829ed10f42f1511a033f5c22d01ec9efe5a41.tar.gz \
    | tar -xz -C tmp/refcheck/iceshrimp-vue --strip-components=1 --wildcards '*/packages/*'
  ```
  The Misskey backend source is about 11 MB; the Streaming implementation lives in `packages/backend/src/server/api/stream/`. The Vue-based Iceshrimp fork (`codeberg.org/banaanihillo/iceshrimp`) is the one whose components can be mirrored; `Iceshrimp.NET` is a Blazor rewrite, useful for layout and interaction decisions only. Codeberg intermittently returns 504 — retry, or use the pinned commit archive URL.
- **Pinned reference versions (2026-11-01).** Firefish `02c31e0` (2024-12-11), Iceshrimp Vue fork `1fa829ed10f4` (2025-07-25), Iceshrimp.NET `2c61359` (2026-10-27), Misskey `c05e844` (2026-10-29). Details, licences and the copy boundary are in `docs/research/2026-11-01-firefish-port-source-and-license-register.md`.
- **The reference frontends are read-only references and are not buildable in place.** The Firefish checkout is sparse (`packages/client/src`, `packages/sw/src`), so `packages/client/index.html`, the root `scripts/copy-index.mjs` and the `packages/firefish-js` workspace package are absent — `pnpm install` cannot even resolve `firefish-js: workspace:*`. Never add a reference to `package.json`, a workspace, or a build script; never build or ship reference sources.
- Misskey and both Firefish-descended frontends are AGPL-3.0; the Iceshrimp Vue fork's client is AGPL-3.0 while its `packages/iceshrimp-sdk` is MIT. Reusing data structures, field names, enum values and behaviour is fine. If a file is copied or rewritten, keep its copyright/SPDX header and record `<repo>@<commit>:<path>` in `docs/HANDOVER.md`. If the project is ever distributed, it must be distributed under AGPL-3.0.
- Do not claim compatibility without identifying the concrete API, protocol behaviour, or test evidence.
- GitHub network access must use the user's proxy at `127.0.0.1:19823`. Note that `dl.min.io` is blocked by that proxy.
- Avoid reading or transmitting secrets, credentials, private repositories, or unpublished code.

## Engineering Defaults

- The architecture baseline lives at `docs/superpowers/specs/2026-10-26-cinderfish-architecture-v0.md`. Read it before changing module boundaries or the data model.
- Prefer a small Go service with explicit interfaces for persistence, media storage, federation, and background work.
- Keep the frontend and backend compatibility boundaries explicit.
- Frontend work follows the runtime adaptation contract: ported pages and components never `fetch` directly, never read the token from `localStorage` themselves, and never concatenate RPC paths — every call goes through the `api` shim in `web/src/api/client.ts`. The shim's `shim` / `rewrite` / `drop` split is fixed in `docs/research/2026-11-01-runtime-adaptation-contract.md`.
- Use the repository's existing conventions once implementation begins.
- Add tests when the project has a test suite; do not create unrelated scaffolding.
- **Commit each finished piece of work to git as soon as it is done.** Do not leave a finished milestone or task sitting uncommitted, and do not batch several unrelated finished tasks into one late commit. Verify `go vet` and the tests pass, update `docs/HANDOVER.md` if the milestone state changed, then commit with a message that says what was done and what evidence backs it.
- Keep database migrations immutable: an applied version is never edited, new changes go into a new version number. The current schema version is 16.
- The S3 client is `github.com/minio/minio-go/v7`, not the AWS SDK. The AWS SDK v2 dependency tree is large and downloaded very slowly through the available proxy; minio-go works with AWS S3, Cloudflare R2, Backblaze B2 and MinIO alike.
- When building SQL dynamically, collect conditions and their arguments together. Building placeholder strings and passing them as values silently shifts the argument list and produces errors such as "could not determine data type of parameter".

## Environment

- Go 1.26.8 with `GOPROXY=https://mirrors.aliyun.com/goproxy/,direct`.
- PostgreSQL 18 with the roles and databases described in `docs/HANDOVER.md`.
- Access to GitHub requires the proxy `127.0.0.1:19823`.

```bash
go vet ./...
go test ./...
CINDERFISH_TEST_DATABASE_URL="postgres://cinderfish:cinderfish@localhost:5432/cinderfish_test" go test ./...
```

Database tests skip themselves when `CINDERFISH_TEST_DATABASE_URL` is unset.

## Work Records

Research notes belong under `docs/research/`. Decision records and project scope belong under `docs/` once agreed. Include dates and source URLs for externally verified claims. Handover material belongs in `docs/HANDOVER.md`.

Documentation in `docs/` is written in Chinese; code, identifiers and code comments are in English. Keep that split.
