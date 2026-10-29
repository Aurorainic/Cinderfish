import { ref } from "vue";
import { getToken } from "../api/client";

// A thin client for the backend's Misskey-compatible Streaming API.
// Frames are {type, body}; channel events arrive as
// {type:"channel", body:{id, type, body}} and are routed back to the
// subscription that asked for them. One connection is shared by the whole
// app, and the `main` channel is subscribed once by this store because the
// server delivers per-user events only once per connection.

export type StreamHandler = (channel: string, type: string, body: unknown) => void;

type Subscription = { id: string; channel: string };

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
  private subs = new Map<string, Subscription>();
  private handlers = new Set<StreamHandler>();
  private mainID: string | null = null;

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

  subscribe(channel: string): string {
    const id = `cf${++this.counter}`;
    this.subs.set(id, { id, channel });
    if (this.ws?.readyState === WebSocket.OPEN) this.sendConnect(id, channel);
    return id;
  }

  unsubscribe(id: string): void {
    if (!this.subs.delete(id)) return;
    if (this.ws?.readyState === WebSocket.OPEN) {
      this.ws.send(JSON.stringify({ type: "disconnect", body: { id } }));
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
      for (const sub of this.subs.values()) this.sendConnect(sub.id, sub.channel);
      if (this.token && !this.mainID) {
        this.mainID = this.subscribe("main");
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
      const channel = this.subs.get(body.id)?.channel ?? (body.id === this.mainID ? "main" : "");
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
    if (this.mainID) this.subs.delete(this.mainID);
    this.mainID = null;
  }

  private clearRetry(): void {
    if (this.retryTimer) {
      clearTimeout(this.retryTimer);
      this.retryTimer = null;
    }
  }
}

export const streaming = new StreamingClient();
