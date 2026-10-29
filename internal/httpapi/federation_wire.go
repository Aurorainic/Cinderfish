package httpapi

import (
	"context"
	"crypto/rsa"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Aurorainic/cinderfish/internal/federation"
	"github.com/Aurorainic/cinderfish/internal/note"
)

// AttachFederation wires the ActivityPub surface: policy, renderer,
// store, client, inbox and the deliver worker.
//
// It is separate from NewServerWithDB so tests can build an instance
// without a database and still exercise the AP routes' mode checks.
func (s *Server) AttachFederation() {
	s.federationPolicy = federation.NewPolicy(s.pool, s.cfg.Domain)
	s.fedRenderer = federation.NewRenderer(s.baseURL())
	s.federationStore = federation.NewStore(s.pool)

	// Sign outbound requests with the instance's first administrator
	// key. A personal instance has one actor that speaks for it, which
	// is what Misskey does with its instance actor too.
	s.federationClient = federation.NewClient(s.federationPolicy, s.fedRenderer, func(ctx context.Context) (string, *rsa.PrivateKey, error) {
		id, err := s.signingUserID(ctx)
		if err != nil {
			return "", nil, err
		}
		kp, err := s.federationStore.EnsureLocalKey(ctx, id)
		if err != nil {
			return "", nil, err
		}
		priv, err := federation.ParsePrivateKeyPEM(kp.PrivatePEM)
		if err != nil {
			return "", nil, err
		}
		return s.fedRenderer.KeyID(id), priv, nil
	})

	s.federationInbox = federation.NewInbox(s.federationStore, s.fedRenderer, s.federationClient, s.federationPolicy)
	s.federationInbox.OnNoteCreated = func(ctx context.Context, noteID, userID string) {
		// Remote notes land in the global timeline; publishing the
		// streaming event keeps open clients in step with it.
		if s.streamBus != nil {
			s.streamBus.PublishTyped("notesStream", "note", map[string]any{"id": noteID, "userId": userID})
		}
	}

	if s.pool != nil {
		s.federationDeliver = federation.NewDeliverer(s.pool, s.federationStore, s.federationClient, s.federationPolicy)
	}
}

// StartFederationWorkers runs the delivery worker until ctx is done.
func (s *Server) StartFederationWorkers(ctx context.Context) {
	if s.federationDeliver == nil {
		return
	}
	log.Println("httpapi: federation delivery worker started")
	go s.federationDeliver.Run(ctx)
}

// FederationLocalKey returns a local user's keypair, creating it if
// needed. Tests use it to sign activities as a real local actor would.
func (s *Server) FederationLocalKey(ctx context.Context, userID string) (*federation.Keypair, error) {
	return s.federationStore.EnsureLocalKey(ctx, userID)
}

// Federation exposes the federation dependencies to tests.
type FederationDeps struct {
	Policy *federation.Policy
	Store  *federation.Store
	Inbox  *federation.Inbox
}

// RemoteKeyForTest resolves an inbound signature key through the same
// path the inbox uses, so a test proves the verification chain works.
func (s *Server) RemoteKeyForTest(keyID string) (*rsa.PublicKey, error) {
	return s.remoteKey(keyID)
}

// HasFollowingByIDs reports whether a remote actor (by URI) follows a
// local user (by id).
func (s *Server) HasFollowingByActorURI(ctx context.Context, followerURI, followeeID string) bool {
	u, err := s.federationStore.UserByURI(ctx, followerURI)
	if err != nil {
		return false
	}
	return s.federationStore.HasFollowing(ctx, u.ID, followeeID)
}

// HasFollowingByActorURIIDs reports whether a local user (by id)
// follows a remote actor (by URI).
func (s *Server) HasFollowingByActorURIIDs(ctx context.Context, followerID, followeeURI string) bool {
	u, err := s.federationStore.UserByURI(ctx, followeeURI)
	if err != nil {
		return false
	}
	return s.federationStore.HasFollowing(ctx, followerID, u.ID)
}

// SetFederationHTTPClient points the federation client at a custom
// transport, which tests use to route one instance's domain to another's
// httptest server.
func (s *Server) SetFederationHTTPClient(c *http.Client) {
	if s.federationClient != nil {
		s.federationClient.SetHTTPClient(c)
	}
}

// RenderActorTest fetches a stored user's actor document.
func (s *Server) RenderActorTest(ctx context.Context, userID string) (federation.Object, error) {
	u, err := s.accounts.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.renderLocalActor(ctx, u)
}

// TestPool exposes the database pool so package-external tests can
// arrange rows the API cannot create yet.
func (s *Server) TestPool() *pgxpool.Pool { return s.pool }

// Deliverer returns the outbound delivery worker, or nil when federation
// is not attached.
func (s *Server) Deliverer() *federation.Deliverer { return s.federationDeliver }

// Federation exposes the federation dependencies, mainly for tests.
func (s *Server) Federation() (*federation.Policy, *federation.Store, *federation.Inbox, *federation.Deliverer) {
	return s.federationPolicy, s.federationStore, s.federationInbox, s.federationDeliver
}

// signingUserID returns the local user whose key signs outbound
// requests: the first administrator, else the first local user.
func (s *Server) signingUserID(ctx context.Context) (string, error) {
	var id string
	err := s.pool.QueryRow(ctx, `
SELECT id FROM users WHERE host IS NULL ORDER BY is_admin DESC, created_at ASC LIMIT 1
	`).Scan(&id)
	return id, err
}

// DeliverNote fans a newly created local note out to every remote
// follower and mentioned user.
func (s *Server) DeliverNote(ctx context.Context, n *note.Note) {
	if s.federationDeliver == nil || s.fedRenderer == nil {
		return
	}
	if n.LocalOnly || n.UserHost != nil {
		return
	}
	if n.Visibility != "public" && n.Visibility != "home" && n.Visibility != "followers" && n.Visibility != "specified" {
		return
	}

	u, err := s.accounts.GetByID(ctx, n.UserID)
	if err != nil || u == nil {
		return
	}
	doc, err := s.renderNote(ctx, n)
	if err != nil {
		log.Printf("httpapi: render note for delivery: %v", err)
		return
	}
	activity := s.fedRenderer.RenderCreate(doc, n.ID, s.fedRenderer.ActorRef(u.ID), n.CreatedAt)

	inboxes, err := s.federationStore.InboxesForUser(ctx, u.ID, false)
	if err != nil {
		log.Printf("httpapi: follower inboxes: %v", err)
	}
	mentionInboxes, err := s.federationStore.MentionedInboxes(ctx, n.Mentions)
	if err == nil {
		inboxes = append(inboxes, mentionInboxes...)
	}

	seen := map[string]bool{}
	for _, inbox := range inboxes {
		if seen[inbox] {
			continue
		}
		seen[inbox] = true
		if err := s.federationDeliver.Enqueue(ctx, inbox, s.fedRenderer.ActivityRef(n.ID), s.fedRenderer.KeyID(u.ID), u.ID, activity); err != nil {
			log.Printf("httpapi: enqueue delivery to %s: %v", inbox, err)
		}
	}
}

// DeliverDelete fans a note deletion out to the same inboxes.
func (s *Server) DeliverDelete(ctx context.Context, n *note.Note) {
	if s.federationDeliver == nil || s.fedRenderer == nil {
		return
	}
	if n.LocalOnly || n.UserHost != nil {
		return
	}
	inboxes, err := s.federationStore.InboxesForUser(ctx, n.UserID, false)
	if err != nil {
		return
	}
	activity := s.fedRenderer.RenderDelete(s.fedRenderer.NoteRef(n.ID), s.fedRenderer.ActorRef(n.UserID))
	for _, inbox := range inboxes {
		if err := s.federationDeliver.Enqueue(ctx, inbox, "", s.fedRenderer.KeyID(n.UserID), n.UserID, activity); err != nil {
			log.Printf("httpapi: enqueue delete to %s: %v", inbox, err)
		}
	}
}

var _ = time.Now
