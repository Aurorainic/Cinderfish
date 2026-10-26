# Project Guidance

## Project Status

This repository is an exploratory project inspired by a private Go reimplementation of Misskey. `MilinFish` is a temporary working name and must not be treated as the final product name.

The intended product is a personal or very small-user federated social instance:

- Frontend experience should be close to Firefish where practical.
- Misskey compatibility and ActivityPub federation are first-class goals.
- Pleroma/Akkoma are product-positioning references only; do not use their source code or architecture as implementation references unless explicitly requested.
- The first release prioritizes personal-instance usability over full Misskey feature parity.
- Registration modes must include open registration, invite-only registration, and administrator-created accounts.
- Federation modes must include public federation, allowlist-only federation, and federation disabled, with runtime configuration switching.
- Media storage must support both local filesystem storage and S3-compatible object storage.
- The private Go reimplementation and its benchmark table are not public source material. Its measurements are feasibility references, not this project's acceptance criteria unless explicitly adopted later.

## Research Rules

- Keep research notes and source links in the repository so decisions remain auditable.
- Prefer primary sources: official Misskey, Firefish, ActivityPub, and relevant standards documentation.
- **The Misskey source tree is the authoritative API contract.** Before implementing an endpoint, read its `paramDef` and `res` schema in `packages/backend/src/server/api/endpoints/`, and the type definitions in `packages/misskey-js/src/api.types.ts`. Documents are secondary; source wins. A local shallow sparse clone is kept outside the repository for this purpose.
- Read source to determine behavior and field names. Do not translate implementation code line by line: Misskey is AGPL-3.0-only, and a line-by-line port would make this project a derivative work. Data structures, enum values and field names are factual contracts and may be reproduced; expressions are to be rewritten in idiomatic Go.
- Do not claim compatibility without identifying the concrete API, protocol behavior, or test evidence.
- GitHub network access must use the user's proxy at `127.0.0.1:19823`.
- Avoid reading or transmitting secrets, credentials, private repositories, or unpublished code.

## Engineering Defaults

- Do not add product code before the written scope and architecture are reviewed in the conversation. The architecture baseline lives at `docs/superpowers/specs/2026-10-26-milnfish-architecture-v0.md`.
- Implementation language is Go (decided 2026-10-26). The frontend is Vue 3 + TypeScript. PostgreSQL is the only mandatory datastore; Redis is optional behind the queue abstraction. Mastodon API compatibility is explicitly out of scope, including reserved routes.
- Prefer a small Go service with explicit interfaces for persistence, media storage, federation, and background work.
- Keep the frontend and backend compatibility boundaries explicit.
- Use the repository's existing conventions once implementation begins.
- Add tests when the project has a test suite; do not create unrelated scaffolding.

## Work Records

Research notes belong under `docs/research/`. Decision records and project scope belong under `docs/` once agreed. Include dates and source URLs for externally verified claims.
