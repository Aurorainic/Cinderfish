import { ref } from "vue";
import { getToken } from "../api/client";

// A thin client for the backend's Misskey-compatible Streaming API.
// Frames are {type, body}; channel events arrive as
// {type:"channel", body:{id, type, body}} and are routed back to the
// subscription that asked for them. One connection is shared by the whole
// app. A channel is connected to the server at most once per connection:
// shareable channels (main) reject a second connect, so subscribe() is
// refcounted per channel and every caller gets its own token pointing at
// the shared wire subscription. The `main` channel is held by this store
// itself because the server delivers per-user events only once per
// connection; dropping a caller's reference never kills the store's own.

export type StreamHandler = (channel: string, type: string, body: unknown) => void;

type ChannelSub = { id: string; channel: string; refs: number };

const RECONNECT_BASE_MS = 1000;
const RECONNECT_MAX_MS = 30000;
const RECONNECT_LIMIT = 6;

export const unreadNotifications = ref(0);
export const streamState = ref<"idle" | "connecting" | "open" | "closed">("idle");

export function setUnreadNotifications(count: number): void {
  unreadNotifications.value = Math.max(0, count);
}

class StreamingClient {
  private ws: WebSocket | null = null;
  private wanted = false;
  private token = "";
  private attempts = 0;
  private retryTimer: ReturnType<typeof setTimeout> | null = null;
  private counter = 0;
  private channelSubs = new Map<string, ChannelSub>();
  private subsByWire = new Map<string, ChannelSub>();
  private tokens = new Map<string, ChannelSub>();
  private handlers = new Set<StreamHandler>();
  private mainToken = "";

  start(): void {
    if (this.wanted) return;
    this.wanted = true;
    this.token = getToken();
    this.open();
  }

  // restart reconnects with the current token, after signin or signout.
  restart(): void {
    this.wanted = true;
    this.token = getToken();
    this.attempts = 0;
    this.clearRetry();
    this.dropMain();
    this.ws?.close();
    this.ws = null;
    this.open();
  }

  stop(): void {
    this.wanted = false;
    this.clearRetry();
    this.ws?.close();
    this.ws = null;
  }

  on(handler: StreamHandler): () => void {
    this.handlers.add(handler);
    return () => this.handlers.delete(handler);
  }

  // subscribe returns a caller token; the wire-level channel subscription
  // is shared and only disconnected once the last token is released.
  subscribe(channel: string): string {
    const token = `t${++this.counter}`;
    const existing = this.channelSubs.get(channel);
    if (existing) {
      existing.refs += 1;
      this.tokens.set(token, existing);
      return token;
    }
    const sub: ChannelSub = { id: `s${++this.counter}`, channel, refs: 1 };
    this.channelSubs.set(channel, sub);
    this.subsByWire.set(sub.id, sub);
    this.tokens.set(token, sub);
    if (this.ws?.readyState === WebSocket.OPEN) this.sendConnect(sub.id, channel);
    return token;
  }

  unsubscribe(token: string): void {
    const sub = this.tokens.get(token);
    if (!sub) return;
    this.tokens.delete(token);
    sub.refs -= 1;
    if (sub.refs > 0) return;
    this.channelSubs.delete(sub.channel);
    this.subsByWire.delete(sub.id);
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({ type: "disconnect", body: { id: sub.id } }));
    }
  }

  private open(): void {
    if (!this.wanted) return;
    if (typeof window === "undefined") return;

    const proto = window.location.protocol === "https:" ? "wss:" : "ws:";
    const query = this.token ? `?i=${encodeURIComponent(this.token)}` : "";
    streamState.value = "connecting";
    const ws = new WebSocket(`${proto}//${window.location.host}/streaming${query}`);
    this.ws = ws;

    ws.onopen = () => {
      this.attempts = 0;
      streamState.value = "open";
      for (const sub of this.channelSubs.values()) this.sendConnect(sub.id, sub.channel);
      // Re-assert the store's own main subscription; subscribe() joins the
      // existing wire subscription when one survived (e.g. another caller
      // still holds it), so no duplicate connect frame is sent.
      if (this.token && !this.mainToken) {
        this.mainToken = this.subscribe("main");
      }
      this.emit("", "__reconnected", null);
    };

    ws.onmessage = (event) => {
      let frame: { type?: string; body?: unknown };
      try {
        frame = JSON.parse(String(event.data)) as { type?: string; body?: unknown };
      } catch {
        return;
      }
      if (frame.type !== "channel" || !frame.body) return;
      const body = frame.body as { id?: string; type?: string; body?: unknown };
      if (!body.id || !body.type) return;
      const channel = this.subsByWire.get(body.id)?.channel ?? "";
      if (channel === "main" && body.type === "notification") {
        unreadNotifications.value += 1;
      }
      this.emit(channel, body.type, body.body);
    };

    ws.onclose = () => {
      if (this.ws !== ws) return;
      this.ws = null;
      streamState.value = "closed";
      this.dropMain();
      if (!this.wanted) return;
      if (this.attempts >= RECONNECT_LIMIT) return;
      const delay = Math.min(RECONNECT_BASE_MS * 2 ** this.attempts, RECONNECT_MAX_MS);
      this.attempts += 1;
      this.retryTimer = setTimeout(() => this.open(), delay);
    };
  }

  private sendConnect(id: string, channel: string): void {
    this.ws?.send(JSON.stringify({ type: "connect", body: { channel, id, pong: true } }));
  }

  private emit(channel: string, type: string, body: unknown): void {
    for (const handler of this.handlers) {
      try {
        handler(channel, type, body);
      } catch {
        // A broken listener must not stop the others.
      }
    }
  }

  private dropMain(): void {
    if (!this.mainToken) return;
    const token = this.mainToken;
    this.mainToken = "";
    this.unsubscribe(token);
  }

  private clearRetry(): void {
    if (this.retryTimer) {
      clearTimeout(this.retryTimer);
      this.retryTimer = null;
    }
  }
}

export const streaming = new StreamingClient();
