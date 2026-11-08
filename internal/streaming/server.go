package streaming

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
)

// Authenticator resolves a streaming token into a viewer.
//
// Misskey accepts the token from the Authorization header or the "i"
// query parameter, and requires the read:account permission on app
// tokens. Implementations live in httpapi so this package stays free of
// database details.
type Authenticator interface {
	// AuthenticateStream resolves token. A token that resolves to no
	// user is an anonymous connection. It returns ErrInvalidToken when
	// the token is present but unusable, and ErrSuspended when the
	// account is suspended.
	AuthenticateStream(ctx context.Context, token string) (*StreamUser, error)
}

// StreamUser is the authenticated identity behind a connection.
type StreamUser struct {
	ID          string
	IsSuspended bool
}

// Errors returned by Authenticator.
var (
	// ErrInvalidToken means the token is present but unusable.
	ErrInvalidToken = errors.New("streaming: invalid token")
	// ErrSuspended means the account is suspended.
	ErrSuspended = errors.New("streaming: account suspended")
)

// ViewerLoader loads the follow graph used by channel filters.
type ViewerLoader interface {
	// LoadFollows returns the follow edges of userID.
	LoadFollows(ctx context.Context, userID string) (map[string]Follow, error)
}

// ListMembership records one member's list settings.
type ListMembership struct {
	// WithReplies is the member's "show replies in the list timeline"
	// setting.
	WithReplies bool
}

// ListLoader loads the membership map of one user list, the state the
// userList channel filters notes with.
type ListLoader interface {
	// LoadListMembers returns the members of listID. ok is false when the
	// list does not exist or is not owned by userID, which rejects the
	// subscription the way upstream's init throw does.
	LoadListMembers(ctx context.Context, listID, userID string) (map[string]ListMembership, bool, error)
}

// Options configures a Server.
type Options struct {
	Bus       *Bus
	Auth      Authenticator
	Followers ViewerLoader
	Lists     ListLoader
	Clips     ClipLoader
	Antennas  AntennaLoader
	Logger    func(string, ...any)
	// MarkRead is called for the "readNotification" client frame: an empty
	// id list means "mark everything read", which is what a frame without any
	// id asks for.
	MarkRead func(ctx context.Context, userID string, notificationIDs []string) error
	// TokenHasReadAccount reports whether an app token carries the
	// read:account permission. Misskey rejects the connection without it.
	TokenHasReadAccount func(ctx context.Context, token string) (bool, error)
}

// Server serves the /streaming WebSocket endpoint.
type Server struct {
	bus         *Bus
	auth        Authenticator
	followers   ViewerLoader
	lists       ListLoader
	clipsLoader ClipLoader
	antennas    AntennaLoader
	logf        func(string, ...any)
	markRead    func(ctx context.Context, userID string, notificationIDs []string) error

	upgrader websocket.Upgrader
}

// New returns a streaming Server.
func New(opts Options) *Server {
	return &Server{
		bus:         opts.Bus,
		auth:        opts.Auth,
		followers:   opts.Followers,
		lists:       opts.Lists,
		clipsLoader: opts.Clips,
		antennas:    opts.Antennas,
		logf:        logfOr(opts.Logger),
		markRead:    opts.MarkRead,
		// Misskey performs no origin check on the streaming endpoint.
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1 << 10,
			WriteBufferSize: 1 << 10,
			CheckOrigin:     func(*http.Request) bool { return true },
		},
	}
}

// ServeHTTP upgrades the request and runs the connection until it ends.
//
// The handshake follows StreamingApiServerService.attach: a suspended
// account gets 403 and an unusable token 401 with a WWW-Authenticate
// header. A request with no token is accepted anonymously.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	token := ""
	if h := r.Header.Get("Authorization"); strings.HasPrefix(h, "Bearer ") {
		token = strings.TrimPrefix(h, "Bearer ")
	} else {
		token = r.URL.Query().Get("i")
	}

	var user *StreamUser
	if token != "" {
		u, err := s.auth.AuthenticateStream(r.Context(), token)
		if err != nil {
			if errors.Is(err, ErrSuspended) {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			w.Header().Set("WWW-Authenticate",
				`Bearer realm="Misskey", error="invalid_token", error_description="Failed to authenticate"`)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		user = u
	}
	if user != nil && user.IsSuspended {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	ws, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		// Upgrade already wrote an error response.
		return
	}

	viewer := Viewer{}
	if user != nil {
		viewer.UserID = user.ID
		if s.followers != nil {
			if f, err := s.followers.LoadFollows(r.Context(), user.ID); err == nil {
				viewer.Following = f
			}
		}
	}

	conn := newConnection(r.Context(), newWSConn(ws), s.bus, viewer)
	if user != nil && s.markRead != nil {
		uid := user.ID
		conn.onReadNotification = func(_ string, notificationIDs []string) {
			// Detached from the request context: the client may ask to
			// mark notifications read while the connection is closing.
			go func() { _ = s.markRead(context.Background(), uid, notificationIDs) }()
		}
	}
	if user != nil && s.followers != nil {
		uid := user.ID
		conn.refreshViewer = func(c *Connection) {
			f, err := s.followers.LoadFollows(context.Background(), uid)
			if err != nil {
				return
			}
			c.setFollowing(f)
		}
	}
	conn.lists = s.lists
	conn.clipLoader = s.clipsLoader
	conn.antennaLoader = s.antennas

	conn.serve()
}
