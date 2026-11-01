// SPDX-License-Identifier: AGPL-3.0-only
// Ported from firefish@02c31e0:packages/client/src/pizzax.ts
//
// Re-export shim: Firefish imports pizzax as `@/pizzax`, so the ported
// components can keep that path while the implementation lives in the store
// directory.
export { Storage } from "../store/pizzax";
export type { StateDef } from "../store/pizzax";
