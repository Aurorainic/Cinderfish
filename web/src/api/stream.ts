import { getAuthToken } from "./client";
import type { DriveFile, MeDetailed, NoteDetailed, Notification } from "./types";

/**
 * Cinderfish Streaming API client.
 *
 * Protocol (docs/research/api-contract/streaming.md):
 *   - `GET /streaming` WebSocket upgrade, token via `?i=` (headers are not
 *     available to browser WebSockets);
 *   - frames are `{ type, body }`;
 *   - `connect` with `pong: true` answers `connected`;
 *   - idle connections are killed after ~120-180s because server pongs do not
 *     refresh liveness, so an application-level heartbeat is sent every 30s;
 *   - channel names are limited to `main`, `homeTimeline`, `localTimeline`,
 *     `hybridTimeline`, `globalTimeline` by the backend. Unknown channels are
 *     ignored silently, so subscriptions time out locally.
 */

export type StreamChannelName =
	| "main"
	| "homeTimeline"
	| "localTimeline"
	| "hybridTimeline"
	| "globalTimeline";

export interface ChannelNoteEvent {
	note: NoteDetailed;
}

export type FollowEvent = { id: string };

export interface ChannelEventMap {
	note: NoteDetailed;
	notification: Notification;
	mention: NoteDetailed;
	reply: NoteDetailed;
	renote: NoteDetailed;
	follow: FollowEvent;
	followed: FollowEvent;
	unfollow: FollowEvent;
	readAllNotifications: null;
	meUpdated: MeDetailed;
	driveFileCreated: DriveFile;
	connected: { id: string };
	unavailable: { id: string };
}

export interface NoteUpdatedEvent {
	id: string;
	type: "reacted" | "unreacted" | "deleted";
	body: unknown;
}

export interface StreamEventMap {
	connected: { id: string };
	disconnected: { code: number; reason: string };
	reconnecting: { attempt: number; delay: number };
	error: Error;
	noteUpdated: NoteUpdatedEvent;
}

type Handler<T> = (payload: T) => void;

class Emitter<Events extends object> {
	private readonly handlers = new Map<keyof Events, Set<Handler<unknown>>>();

	on<K extends keyof Events>(event: K, handler: Handler<Events[K]>): () => void {
		let set = this.handlers.get(event);
		if (set == null) {
			set = new Set();
			this.handlers.set(event, set);
		}
		const untyped = handler as Handler<unknown>;
		set.add(untyped);
		return () => set?.delete(untyped);
	}

	off<K extends keyof Events>(event: K, handler: Handler<Events[K]>): void {
		this.handlers.get(event)?.delete(handler as Handler<unknown>);
	}

	emit<K extends keyof Events>(event: K, payload: Events[K]): void {
		const set = this.handlers.get(event);
		if (set == null) return;
		for (const handler of [...set]) {
			try {
				handler(payload);
			} catch (error) {
				console.error(`[stream] listener for '${String(event)}' failed`, error);
			}
		}
	}

	clear(): void {
		this.handlers.clear();
	}
}

interface Subscription {
	id: string;
	name: StreamChannelName;
	params: Record<string, unknown>;
	handlers: Map<string, Set<Handler<unknown>>>;
	connected: boolean;
	timer: number | null;
	disposed: boolean;
}

export type StreamStatus =
	| "idle"
	| "connecting"
	| "connected"
	| "disconnected"
	| "reconnecting"
	| "closed";

const HEARTBEAT_INTERVAL = 30_000;
const MAX_RECONNECT_DELAY = 30_000;
const CONNECT_TIMEOUT = 8_000;

export class StreamChannel {
	constructor(
		private readonly client: StreamClient,
		private readonly subscription: Subscription,
	) {}

	get id(): string {
		return this.subscription.id;
	}

	get name(): StreamChannelName {
		return this.subscription.name;
	}

	get connected(): boolean {
		return this.subscription.connected;
	}

	on<K extends keyof ChannelEventMap>(
		event: K,
		handler: Handler<ChannelEventMap[K]>,
	): () => void {
		let set = this.subscription.handlers.get(event as string);
		if (set == null) {
			set = new Set();
			this.subscription.handlers.set(event as string, set);
		}
		const untyped = handler as Handler<unknown>;
		set.add(untyped);
		return () => set?.delete(untyped);
	}

	off<K extends keyof ChannelEventMap>(
		event: K,
		handler: Handler<ChannelEventMap[K]>,
	): void {
		this.subscription.handlers.get(event as string)?.delete(handler as Handler<unknown>);
	}

	/** Send a channel-scoped message. No backend channel consumes them today. */
	send(type: string, body: unknown): void {
		this.client.send("channel", {
			id: this.subscription.id,
			type,
			body,
		});
	}

	dispose(): void {
		if (this.subscription.disposed) return;
		this.subscription.disposed = true;
		if (this.subscription.timer != null) {
			window.clearTimeout(this.subscription.timer);
			this.subscription.timer = null;
		}
		this.client.disconnectChannel(this.subscription.id);
	}
}

export class StreamClient {
	private socket: WebSocket | null = null;
	private readonly subscriptions = new Map<string, Subscription>();
	private readonly events = new Emitter<StreamEventMap>();
	private readonly outbox: string[] = [];
	private heartbeatTimer: number | null = null;
	private reconnectTimer: number | null = null;
	private attempt = 0;
	private shouldRun = false;
	private currentStatus: StreamStatus = "idle";
	private nextId = 0;
	private socketGeneration = 0;

	constructor(private readonly baseUrl?: string) {}

	get status(): StreamStatus {
		return this.currentStatus;
	}

	on<K extends keyof StreamEventMap>(
		event: K,
		handler: Handler<StreamEventMap[K]>,
	): () => void {
		return this.events.on(event, handler);
	}

	off<K extends keyof StreamEventMap>(
		event: K,
		handler: Handler<StreamEventMap[K]>,
	): void {
		this.events.off(event, handler);
	}

	private buildUrl(): string {
		if (this.baseUrl != null && this.baseUrl !== "") {
			const url = new URL(this.baseUrl, location.href);
			url.protocol = url.protocol === "https:" ? "wss:" : "ws:";
			return url.toString();
		}
		const protocol = location.protocol === "https:" ? "wss:" : "ws:";
		return `${protocol}//${location.host}/streaming`;
	}

	private authenticatedUrl(): string {
		const url = new URL(this.buildUrl());
		const token = getAuthToken();
		if (token != null && token !== "") url.searchParams.set("i", token);
		return url.toString();
	}

	connect(): void {
		if (typeof WebSocket === "undefined") return;
		if (this.socket != null) return;
		this.shouldRun = true;
		this.openSocket();
	}

	close(): void {
		this.shouldRun = false;
		this.stopHeartbeat();
		this.clearReconnect();
		this.currentStatus = "closed";
		this.socketGeneration++;
		const socket = this.socket;
		this.socket = null;
		if (socket != null && socket.readyState <= WebSocket.OPEN) {
			socket.close(1000, "client closed");
		}
	}

	/** Drop the current connection and resubscribe immediately. */
	reconnect(): void {
		this.shouldRun = true;
		this.attempt = 0;
		this.clearReconnect();
		this.socketGeneration++;
		const socket = this.socket;
		this.socket = null;
		if (socket != null && socket.readyState <= WebSocket.OPEN) {
			socket.close(1000, "reconnect");
		}
		this.openSocket();
	}

	private clearReconnect(): void {
		if (this.reconnectTimer != null) {
			window.clearTimeout(this.reconnectTimer);
			this.reconnectTimer = null;
		}
	}

	private openSocket(): void {
		if (!this.shouldRun || typeof WebSocket === "undefined") return;
		this.clearReconnect();
		this.currentStatus = this.attempt === 0 ? "connecting" : "reconnecting";

		let socket: WebSocket;
		try {
			socket = new WebSocket(this.authenticatedUrl());
		} catch (error) {
			this.events.emit("error", error instanceof Error ? error : new Error(String(error)));
			this.scheduleReconnect();
			return;
		}
		this.socket = socket;
		const generation = ++this.socketGeneration;

		socket.onopen = () => {
			if (generation !== this.socketGeneration) return;
			this.attempt = 0;
			this.currentStatus = "connected";
			this.flushOutbox();
			for (const subscription of this.subscriptions.values()) {
				if (!subscription.disposed) this.sendConnect(subscription);
			}
			this.startHeartbeat();
		};

		socket.onmessage = (event: MessageEvent<string>) => {
			if (generation !== this.socketGeneration) return;
			this.handleMessage(event.data);
		};

		socket.onerror = () => {
			if (generation !== this.socketGeneration) return;
			this.events.emit("error", new Error("Streaming socket error"));
		};

		socket.onclose = (event: CloseEvent) => {
			if (generation !== this.socketGeneration) return;
			if (this.socket === socket) this.socket = null;
			this.stopHeartbeat();
			this.currentStatus = "disconnected";
			this.events.emit("disconnected", {
				code: event.code,
				reason: event.reason,
			});
			for (const subscription of this.subscriptions.values()) {
				subscription.connected = false;
			}
			if (this.shouldRun) this.scheduleReconnect();
		};
	}

	private scheduleReconnect(): void {
		if (!this.shouldRun || this.reconnectTimer != null) return;
		this.attempt++;
		const exponential = Math.min(
			MAX_RECONNECT_DELAY,
			1000 * 2 ** Math.min(this.attempt, 5),
		);
		const delay = Math.round(exponential * (0.75 + Math.random() * 0.5));
		this.currentStatus = "reconnecting";
		this.events.emit("reconnecting", { attempt: this.attempt, delay });
		this.reconnectTimer = window.setTimeout(() => {
			this.reconnectTimer = null;
			this.openSocket();
		}, delay);
	}

	private startHeartbeat(): void {
		this.stopHeartbeat();
		this.heartbeatTimer = window.setInterval(() => {
			if (this.socket?.readyState !== WebSocket.OPEN) return;
			// Application-level frame: `lastSeen` only advances on data frames,
			// so this keeps the connection alive through the ~120s idle kill.
			this.send("channel", {
				id: "__heartbeat__",
				type: "__ping__",
				body: {},
			});
		}, HEARTBEAT_INTERVAL);
	}

	private stopHeartbeat(): void {
		if (this.heartbeatTimer != null) {
			window.clearInterval(this.heartbeatTimer);
			this.heartbeatTimer = null;
		}
	}

	private flushOutbox(): void {
		const socket = this.socket;
		if (socket == null || socket.readyState !== WebSocket.OPEN) return;
		while (this.outbox.length > 0) {
			const frame = this.outbox.shift();
			if (frame != null) socket.send(frame);
		}
	}

	send(type: string, body: unknown): void {
		const frame = JSON.stringify({ type, body });
		const socket = this.socket;
		if (socket == null || socket.readyState !== WebSocket.OPEN) {
			if (this.outbox.length < 128) this.outbox.push(frame);
			return;
		}
		socket.send(frame);
	}

	private handleMessage(raw: string): void {
		let frame: unknown;
		try {
			frame = JSON.parse(raw) as unknown;
		} catch {
			return;
		}
		if (frame == null || typeof frame !== "object") return;
		const { type, body } = frame as { type?: unknown; body?: unknown };
		if (typeof type !== "string") return;

		switch (type) {
			case "connected": {
				const id = (body as { id?: unknown } | null)?.id;
				if (typeof id === "string") this.markConnected(id);
				break;
			}
			case "channel": {
				const payload = body as
					| { id?: unknown; type?: unknown; body?: unknown }
					| null;
				if (payload == null) return;
				if (typeof payload.id !== "string" || typeof payload.type !== "string") {
					return;
				}
				this.dispatchChannel(payload.id, payload.type, payload.body);
				break;
			}
			case "noteUpdated": {
				const payload = body as
					| { id?: unknown; type?: unknown; body?: unknown }
					| null;
				if (payload == null || typeof payload.id !== "string") return;
				this.events.emit("noteUpdated", {
					id: payload.id,
					type: payload.type as NoteUpdatedEvent["type"],
					body: payload.body,
				});
				break;
			}
			default:
				// Unknown frame types are ignored by the server and by us.
				break;
		}
	}

	private markConnected(id: string): void {
		const subscription = this.subscriptions.get(id);
		if (subscription == null) return;
		subscription.connected = true;
		if (subscription.timer != null) {
			window.clearTimeout(subscription.timer);
			subscription.timer = null;
		}
		this.dispatchChannel(id, "connected", { id });
		this.events.emit("connected", { id });
	}

	private dispatchChannel(id: string, event: string, payload: unknown): void {
		const subscription = this.subscriptions.get(id);
		if (subscription == null) return;
		const listeners = subscription.handlers.get(event);
		if (listeners == null) return;
		for (const listener of [...listeners]) {
			try {
				listener(payload);
			} catch (error) {
				console.error(`[stream] channel '${subscription.name}' handler failed`, error);
			}
		}
	}

	private sendConnect(subscription: Subscription): void {
		this.send("connect", {
			channel: subscription.name,
			id: subscription.id,
			params: subscription.params,
			pong: true,
		});
		if (subscription.timer != null) window.clearTimeout(subscription.timer);
		subscription.timer = window.setTimeout(() => {
			if (!subscription.connected && !subscription.disposed) {
				this.dispatchChannel(subscription.id, "unavailable", {
					id: subscription.id,
				});
			}
		}, CONNECT_TIMEOUT);
	}

	/**
	 * Subscribe to a channel. `main` is a per-connection singleton (the server
	 * silently ignores a second `main` connect), so repeated calls return the
	 * same channel object.
	 */
	useChannel(
		name: StreamChannelName,
		params: Record<string, unknown> = {},
		id?: string,
	): StreamChannel {
		if (name === "main") {
			for (const existing of this.subscriptions.values()) {
				if (existing.name === "main" && !existing.disposed) {
					return new StreamChannel(this, existing);
				}
			}
		}

		const channelId = id ?? (name === "main" ? "main" : `ch-${++this.nextId}`);
		const subscription: Subscription = {
			id: channelId,
			name,
			params,
			handlers: new Map(),
			connected: false,
			timer: null,
			disposed: false,
		};
		this.subscriptions.set(channelId, subscription);

		if (this.socket?.readyState === WebSocket.OPEN) {
			this.sendConnect(subscription);
		} else {
			this.connect();
		}

		return new StreamChannel(this, subscription);
	}

	disconnectChannel(id: string): void {
		const subscription = this.subscriptions.get(id);
		if (subscription == null) return;
		this.subscriptions.delete(id);
		if (subscription.timer != null) window.clearTimeout(subscription.timer);
		if (this.socket?.readyState === WebSocket.OPEN) {
			this.send("disconnect", { id });
		}
	}

	/** Subscribe to updates of a single note (`reacted`/`unreacted`/`deleted`). */
	subscribeNote(noteId: string, handler: Handler<NoteUpdatedEvent>): () => void {
		this.send("subNote", { id: noteId });
		const off = this.events.on("noteUpdated", (event) => {
			if (event.id === noteId) handler(event);
		});
		return () => {
			off();
			this.send("unsubNote", { id: noteId });
		};
	}

	/** Ask the server to mark all notifications as read (no REST round-trip). */
	readNotification(): void {
		this.send("readNotification", {});
	}
}

let singleton: StreamClient | null = null;

/** Lazily created shared client, mirroring Firefish's `useStream()`. */
export function useStream(): StreamClient {
	if (singleton == null) singleton = new StreamClient();
	return singleton;
}

export function getStream(): StreamClient {
	return useStream();
}

/** Close and recreate the connection (used after sign-in / sign-out). */
export function reloadStream(): StreamClient {
	const client = useStream();
	client.reconnect();
	return client;
}
