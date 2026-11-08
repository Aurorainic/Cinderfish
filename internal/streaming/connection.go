package streaming

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"
)

// Connection is one WebSocket session: it owns the channel table, the
// per-note subscriptions and the cached follow graph used by channel
// filters.
type Connection struct {
	ws       *wsConn
	bus      *Bus
	ctx      context.Context
	cancel   context.CancelFunc
	lastSeen *atomicTime

	// viewerMu guards viewer, which is reloaded periodically.
	viewerMu sync.RWMutex
	viewer   Viewer

	// mu guards the channel table and note subscriptions.
	mu        sync.Mutex
	channels  map[string]Channel
	noteSubs  map[string]int
	noteOrder []string
	// noteUnsubs maps a subscribed note id to its bus unsubscribe.
	noteUnsubs map[string]func()

	// Hooks installed by the Server. onReadNotification receives every id the
	// client asked to mark read; an empty slice means "all notifications".
	onReadNotification func(userID string, notificationIDs []string)
	refreshViewer      func(*Connection)
	// lists loads user-list membership for the userList channel.
	lists ListLoader
	// clipLoader checks clip visibility for the clip channel.
	clipLoader ClipLoader
}

// newConnection assembles a Connection for an upgraded WebSocket.
func newConnection(ctx context.Context, ws *wsConn, bus *Bus, v Viewer) *Connection {
	cctx, cancel := context.WithCancel(ctx)
	ls := &atomicTime{}
	ls.store(time.Now())
	return &Connection{
		ws:         ws,
		bus:        bus,
		ctx:        cctx,
		cancel:     cancel,
		lastSeen:   ls,
		viewer:     v,
		channels:   make(map[string]Channel),
		noteSubs:   make(map[string]int),
		noteUnsubs: make(map[string]func()),
	}
}

func (c *Connection) viewerSnapshot() Viewer {
	c.viewerMu.RLock()
	defer c.viewerMu.RUnlock()
	return c.viewer
}

func (c *Connection) setViewer(v Viewer) {
	c.viewerMu.Lock()
	c.viewer = v
	c.viewerMu.Unlock()
}

func (c *Connection) setFollowing(f map[string]Follow) {
	c.viewerMu.Lock()
	c.viewer.Following = f
	c.viewerMu.Unlock()
}

// maxFrameSize bounds one client frame. Misskey's frames are tiny JSON
// envelopes; without a read limit a single oversized frame is parsed into
// memory wholesale, so an anonymous client can exhaust the process.
const maxFrameSize = 64 << 10

// serve runs the connection until the peer closes it.
func (c *Connection) serve() {
	defer c.dispose()

	// notesStream feeds every timeline channel. Misskey subscribes each
	// channel to the bus independently; one connection-level listener
	// dispatching to the channel table is equivalent and avoids N fold
	// delivery for N timeline channels.
	unsubNotes := c.bus.Subscribe(func(subject string, msg Message) {
		if subject == SubjectNotes {
			c.dispatchNote(msg)
		}
	})
	defer unsubNotes()

	go c.ws.writeLoop()
	go c.ws.heartbeat(c.lastSeen)
	go c.refreshLoop()

	c.ws.ws.SetReadLimit(maxFrameSize)
	for {
		_, data, err := c.ws.ws.ReadMessage()
		if err != nil {
			return
		}
		c.lastSeen.store(time.Now())
		c.onMessage(data)
	}
}

// refreshLoop reloads the follow graph every contextRefreshInterval,
// matching Misskey's per-connection cache refresh.
func (c *Connection) refreshLoop() {
	if c.refreshViewer == nil {
		return
	}
	ticker := time.NewTicker(contextRefreshInterval)
	defer ticker.Stop()
	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.refreshViewer(c)
		}
	}
}

// onMessage handles one client frame. Unparseable frames are ignored,
// as upstream.
func (c *Connection) onMessage(data []byte) {
	var env struct {
		Type string          `json:"type"`
		Body json.RawMessage `json:"body"`
	}
	if err := json.Unmarshal(data, &env); err != nil {
		return
	}

	switch env.Type {
	case "connect":
		c.onConnect(env.Body)
	case "disconnect":
		c.onDisconnect(env.Body)
	case "channel", "ch":
		c.onChannelMessage(env.Body)
	case "subNote", "s", "sr":
		c.onSubscribeNote(env.Body)
	case "unsubNote", "un":
		c.onUnsubscribeNote(env.Body)
	case "readNotification":
		if c.onReadNotification != nil {
			if uid := c.viewerSnapshot().UserID; uid != "" {
				// Firefish sends `{ id }`; its own broadcast payload (and
				// Misskey's client) uses `notificationIds`. Accept both so a
				// folded row can mark its whole group in one frame.
				var b struct {
					ID              *string  `json:"id"`
					NotificationIDs []string `json:"notificationIds"`
				}
				_ = json.Unmarshal(env.Body, &b)
				ids := make([]string, 0, len(b.NotificationIDs)+1)
				if b.ID != nil && *b.ID != "" {
					ids = append(ids, *b.ID)
				}
				for _, id := range b.NotificationIDs {
					if id != "" {
						ids = append(ids, id)
					}
				}
				c.onReadNotification(uid, ids)
			}
		}
	}
}

// onConnect handles a channel subscription request.
func (c *Connection) onConnect(body json.RawMessage) {
	var b struct {
		Channel *string                    `json:"channel"`
		ID      *string                    `json:"id"`
		Params  map[string]json.RawMessage `json:"params"`
		Pong    *bool                      `json:"pong"`
	}
	if len(body) == 0 || json.Unmarshal(body, &b) != nil {
		return
	}
	if b.ID == nil || b.Channel == nil {
		return
	}
	c.connectChannel(*b.ID, *b.Channel, b.Params, b.Pong != nil && *b.Pong)
}

// connectChannel instantiates and initialises a channel.
//
// Every rejection path is silent: upstream neither answers nor closes,
// so a client subscribing to an unknown channel simply receives nothing.
func (c *Connection) connectChannel(id, name string, params map[string]json.RawMessage, pong bool) {
	c.mu.Lock()
	if old, exists := c.channels[id]; exists {
		delete(c.channels, id)
		c.mu.Unlock()
		old.Dispose()
		c.mu.Lock()
	}
	if len(c.channels) >= maxChannelsPerConnection {
		c.mu.Unlock()
		return
	}
	c.mu.Unlock()

	spec, ok := channelSpecs[name]
	if !ok {
		// Upstream throws here; the throw is swallowed by its message
		// queue, so the client sees nothing either way.
		return
	}
	if spec.RequireCredential && c.viewerSnapshot().UserID == "" {
		return
	}

	// A shareable channel (main) delivers per-user events; a second
	// subscription on the same connection would duplicate them.
	if spec.Shareable {
		c.mu.Lock()
		for _, existing := range c.channels {
			if existing.Name() == name {
				c.mu.Unlock()
				return
			}
		}
		c.mu.Unlock()
	}

	ch := spec.New(c, id)

	// Register before Init so events racing with Init are not lost, then
	// remove again if Init rejects the subscription.
	c.mu.Lock()
	c.channels[id] = ch
	c.mu.Unlock()

	if params == nil {
		params = map[string]json.RawMessage{}
	}
	if ok := ch.Init(params); !ok {
		c.mu.Lock()
		if cur, exists := c.channels[id]; exists && cur == ch {
			delete(c.channels, id)
		}
		c.mu.Unlock()
		ch.Dispose()
		return
	}

	if pong {
		c.ws.send("connected", map[string]any{"id": id})
	}
}

// onDisconnect handles a "disconnect" frame.
func (c *Connection) onDisconnect(body json.RawMessage) {
	var b struct {
		ID *string `json:"id"`
	}
	if len(body) == 0 || json.Unmarshal(body, &b) != nil || b.ID == nil {
		return
	}
	c.mu.Lock()
	ch := c.channels[*b.ID]
	delete(c.channels, *b.ID)
	c.mu.Unlock()
	if ch != nil {
		ch.Dispose()
	}
}

// onChannelMessage validates a client-to-channel message. None of the
// M4 channels receive messages, so this only enforces the frame shape.
func (c *Connection) onChannelMessage(body json.RawMessage) {
	var b struct {
		ID   *string          `json:"id"`
		Type *string          `json:"type"`
		Body *json.RawMessage `json:"body"`
	}
	if len(body) == 0 || json.Unmarshal(body, &b) != nil {
		return
	}
	if b.ID == nil || b.Type == nil || b.Body == nil {
		return
	}
}

// dispatchNote fans a notesStream event out to every timeline channel.
func (c *Connection) dispatchNote(msg Message) {
	var note PackedNote
	if err := json.Unmarshal(msg.Body, &note); err != nil || note == nil {
		return
	}

	c.mu.Lock()
	chans := make([]Channel, 0, len(c.channels))
	for _, ch := range c.channels {
		chans = append(chans, ch)
	}
	c.mu.Unlock()

	for _, ch := range chans {
		ch.HandleNote(note)
	}
}

// onSubscribeNote handles subNote. Subscriptions are reference-counted
// so an alias or duplicate frame does not double-deliver.
func (c *Connection) onSubscribeNote(body json.RawMessage) {
	var b struct {
		ID *string `json:"id"`
	}
	if len(body) == 0 || json.Unmarshal(body, &b) != nil || b.ID == nil || *b.ID == "" {
		return
	}
	id := *b.ID

	c.mu.Lock()
	defer c.mu.Unlock()

	current := c.noteSubs[id]

	// Always refresh recency: a note counts as recently used whether this
	// is a new subscription or a repeated one.
	c.noteOrder = removeString(c.noteOrder, id)

	if current == 0 && len(c.noteSubs) >= maxSubscribingNotesPerConnection {
		// Evict the least recently used subscription so a client cannot
		// pin unbounded memory. noteOrder can contain duplicates from
		// repeated frames, so drop entries that are no longer live.
		for len(c.noteOrder) > 0 {
			oldest := c.noteOrder[0]
			c.noteOrder = c.noteOrder[1:]
			if _, live := c.noteSubs[oldest]; live && oldest != id {
				delete(c.noteSubs, oldest)
				c.removeNoteListenerLocked(oldest)
				break
			}
		}
	}
	c.noteOrder = append(c.noteOrder, id)
	c.noteSubs[id] = current + 1

	if current+1 == 1 {
		c.addNoteListenerLocked(id)
	}
}

// onUnsubscribeNote handles unsubNote.
func (c *Connection) onUnsubscribeNote(body json.RawMessage) {
	var b struct {
		ID *string `json:"id"`
	}
	if len(body) == 0 || json.Unmarshal(body, &b) != nil || b.ID == nil || *b.ID == "" {
		return
	}
	id := *b.ID

	c.mu.Lock()
	defer c.mu.Unlock()

	current, ok := c.noteSubs[id]
	if !ok {
		return
	}
	if current-1 <= 0 {
		delete(c.noteSubs, id)
		c.noteOrder = removeString(c.noteOrder, id)
		c.removeNoteListenerLocked(id)
		return
	}
	c.noteSubs[id] = current - 1
}

// addNoteListenerLocked registers the bus listener for one note. The
// caller holds c.mu.
func (c *Connection) addNoteListenerLocked(noteID string) {
	subject := NoteSubject(noteID)
	unsub := c.bus.Subscribe(func(s string, msg Message) {
		if s == subject {
			c.onNoteStreamMessage(noteID, msg)
		}
	})
	c.noteUnsubs[noteID] = unsub
}

// removeNoteListenerLocked drops the bus listener for one note. The
// caller holds c.mu.
func (c *Connection) removeNoteListenerLocked(noteID string) {
	if unsub, ok := c.noteUnsubs[noteID]; ok {
		unsub()
		delete(c.noteUnsubs, noteID)
	}
}

// onNoteStreamMessage forwards a note update when the viewer may see the
// note, wrapped in Misskey's noteUpdated envelope.
func (c *Connection) onNoteStreamMessage(noteID string, msg Message) {
	var meta NoteMeta
	if err := json.Unmarshal(msg.Body, &meta); err != nil {
		return
	}
	if meta.ID == "" {
		meta.ID = noteID
	}

	v := c.viewerSnapshot()
	if meta.UserID != v.UserID {
		switch meta.Visibility {
		case "specified":
			if v.UserID == "" || !containsString(meta.VisibleUserIDs, v.UserID) {
				return
			}
		case "followers":
			if !v.follows(meta.UserID) {
				return
			}
		}
	}

	typ := ""
	if msg.Type != nil {
		typ = *msg.Type
	}

	var env struct {
		Body any `json:"body"`
	}
	if err := json.Unmarshal(msg.Body, &env); err != nil {
		return
	}

	c.ws.send("noteUpdated", map[string]any{
		"id":   meta.ID,
		"type": typ,
		"body": env.Body,
	})
}

// sendChannel emits a channel event to the client.
func (c *Connection) sendChannel(id, typ string, body any) {
	c.ws.send("channel", map[string]any{"id": id, "type": typ, "body": body})
}

// dispose tears the connection down exactly once.
func (c *Connection) dispose() {
	c.cancel()

	c.mu.Lock()
	chans := c.channels
	c.channels = make(map[string]Channel)
	for id, unsub := range c.noteUnsubs {
		unsub()
		delete(c.noteUnsubs, id)
	}
	c.noteSubs = make(map[string]int)
	c.noteOrder = nil
	c.mu.Unlock()

	for _, ch := range chans {
		ch.Dispose()
	}

	c.ws.terminate()
}

func removeString(list []string, want string) []string {
	out := list[:0]
	for _, s := range list {
		if s != want {
			out = append(out, s)
		}
	}
	return out
}

// logfOr returns a logger, defaulting to the standard one.
func logfOr(f func(string, ...any)) func(string, ...any) {
	if f == nil {
		return log.Printf
	}
	return f
}
