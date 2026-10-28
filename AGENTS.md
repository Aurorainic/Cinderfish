# Project Guidance

> This is the single source of project rules. `AGENT.md` is a pointer to this file.

## Project Status

This repository is an exploratory project inspired by a private Go reimplementation of Misskey. `MilnFish` is a temporary working name and must not be treated as the final product name.

The intended product is a personal or very small-user federated social instance:

- Frontend experience should be close to Firefish where practical.
- Misskey compatibility and ActivityPub federation are first-class goals.
- Pleroma/Akkoma are product-positioning references only; do not use their source code or architecture as implementation references unless explicitly requested.
- The first release prioritizes personal-instance usability over full Misskey feature parity.
- Registration modes must include open registration, invite-only registration, and administrator-created accounts.
- Federation modes must include public federation, allowlist-only federation, and federation disabled, with runtime configuration switching.
- Media storage must support both local filesystem storage and S3-compatible object storage.
- The private Go reimplementation and its benchmark table are not public source material. Its measurements are feasibility references, not this project's acceptance criteria unless explicitly adopted later.

Implementation progress is tracked in `docs/HANDOVER.md` and the milestone table in the architecture document.

## Decided Scope

These decisions are settled; do not reopen them without an explicit request.

| Area | Decision |
|------|----------|
| Language | Go 1.26 |
| Frontend | Vue 3 + TypeScript + Vite |
| Datastore | PostgreSQL is the only mandatory dependency; Redis is optional behind the queue abstraction and disabled by default |
| Mastodon API | Explicitly abandoned — no endpoints, no reserved routes |
| Federation vocabulary | Misskey's own `all` / `specified` / `none`, mapped from the configured `public` / `allowlist` / `none` |
| Data migration | No migration of the private project's data; instances start empty |
| Relay | Out of scope for the first release |

## Research Rules

- Keep research notes and source links in the repository so decisions remain auditable.
- **The Misskey source tree is the authoritative API contract.** Before implementing an endpoint, read its `paramDef` and `res` schema in `packages/backend/src/server/api/endpoints/`, then the type definitions in `packages/misskey-js/src/api.types.ts`. Official documentation is secondary and has been found to disagree with the source in several places.
- The reference clone lives outside the repository and does not survive a reboot. Recreate it with:
  ```bash
  export https_proxy=http://127.0.0.1:19823 http_proxy=http://127.0.0.1:19823
  git clone --depth 1 --filter=blob:none --sparse https://github.com/misskey-dev/misskey.git /tmp/refcheck/misskey
  cd /tmp/refcheck/misskey && git sparse-checkout set packages/backend/src packages/misskey-js/src
  ```
  The full backend source is about 11 MB; the Streaming implementation lives in `packages/backend/src/server/api/stream/`.
- Misskey is AGPL-3.0-only and the user has accepted AGPL obligations for this project. Reusing its data structures, field names, enum values and behaviour is fine. If the project is ever distributed, it must be distributed under AGPL-3.0.
- Do not claim compatibility without identifying the concrete API, protocol behaviour, or test evidence.
- GitHub network access must use the user's proxy at `127.0.0.1:19823`. Note that `dl.min.io` is blocked by that proxy.
- Avoid reading or transmitting secrets, credentials, private repositories, or unpublished code.

## Engineering Defaults

- The architecture baseline lives at `docs/superpowers/specs/2026-10-26-milnfish-architecture-v0.md`. Read it before changing module boundaries or the data model.
- Prefer a small Go service with explicit interfaces for persistence, media storage, federation, and background work.
- Keep the frontend and backend compatibility boundaries explicit.
- Use the repository's existing conventions once implementation begins.
- Add tests when the project has a test suite; do not create unrelated scaffolding.
- **Commit each finished piece of work to git as soon as it is done.** Do not leave a finished milestone or task sitting uncommitted, and do not batch several unrelated finished tasks into one late commit. Verify `go vet` and the tests pass, update `docs/HANDOVER.md` if the milestone state changed, then commit with a message that says what was done and what evidence backs it.
- Keep database migrations immutable: an applied version is never edited, new changes go into a new version number. The current schema version is 3.
- The S3 client is `github.com/minio/minio-go/v7`, not the AWS SDK. The AWS SDK v2 dependency tree is large and downloaded very slowly through the available proxy; minio-go works with AWS S3, Cloudflare R2, Backblaze B2 and MinIO alike.
- When building SQL dynamically, collect conditions and their arguments together. Building placeholder strings and passing them as values silently shifts the argument list and produces errors such as "could not determine data type of parameter".

## Environment

- Go 1.26.8 with `GOPROXY=https://mirrors.aliyun.com/goproxy/,direct`.
- PostgreSQL 18 with the roles and databases described in `docs/HANDOVER.md`.
- Access to GitHub requires the proxy `127.0.0.1:19823`.

```bash
go vet ./...
go test ./...
MILNFISH_TEST_DATABASE_URL="postgres://milnfish:milnfish@localhost:5432/milnfish_test" go test ./...
```

Database tests skip themselves when `MILNFISH_TEST_DATABASE_URL` is unset.

## Work Records

Research notes belong under `docs/research/`. Decision records and project scope belong under `docs/` once agreed. Include dates and source URLs for externally verified claims. Handover material belongs in `docs/HANDOVER.md`.

Documentation in `docs/` is written in Chinese; code, identifiers and code comments are in English. Keep that split.
