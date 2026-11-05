// Ported from firefish@b425f61:packages/client/src/stream.ts (AGPL-3.0-only)
//
// Firefish constructs a firefish-js Stream at module load and wraps it in
// markRaw. Cinderfish's client lives in "@/api/stream" (useStream()); this shim
// only exposes the shared instance under the Firefish name, so a ported
// component keeps saying "import { stream } from '@/stream'".
//
// The heartbeat Firefish schedules here is already handled inside
// StreamClient (30s application-level frames), so it is not duplicated.
import { markRaw } from "vue";
import { getStream, useStream, type StreamClient } from "./api/stream";

export const stream: StreamClient = markRaw(useStream());

export { getStream, useStream };
export type {
	ChannelEventMap,
	StreamChannel,
	StreamChannelName,
	StreamClient,
	StreamEventMap,
	StreamStatus,
} from "./api/stream";
