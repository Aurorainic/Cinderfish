export interface StreamingFrame {
  type: string;
  body: unknown;
}

export type StreamHandler = (channel: string, type: string, body: unknown) => void;

export function dispatchStreamingFrame(
  handlers: Iterable<StreamHandler>,
  frame: StreamingFrame,
  channelForWireId: (id: string) => string | undefined = () => undefined,
): void {
  if (frame.type === "noteUpdated") {
    for (const handler of handlers) handler("", "noteUpdated", frame.body);
    return;
  }
  if (frame.type !== "channel" || !frame.body || typeof frame.body !== "object") return;
  const body = frame.body as { id?: string; type?: string; body?: unknown };
  if (!body.id || !body.type) return;
  const channel = channelForWireId(body.id) ?? "";
  for (const handler of handlers) handler(channel, body.type, body.body);
}
