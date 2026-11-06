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

	// Sign each outbound activity with the key of the account it speaks for:
	// a peer checks that the signature's key belongs to the actor the
	// activity names, so one fixed instance key would get a delivery from a
	// second user rejected. Requests with no user context (plain object
	// fetches) fall back to the instance key, which is the first
	// administrator's.
	s.federationClient = federation.NewClient(s.federationPolicy, s.fedRenderer, func(ctx context.Context, userID string) (string, *rsa.PrivateKey, error) {
		id := userID
		if id == "" {
			resolved, err := s.signingUserID(ctx)
			if err != nil {
				return "", nil, err
			}
			id = resolved
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
	s.federationInbox.OnFollow = func(ctx context.Context, followerID, followeeID string) {
		s.publishFollowEvent(followerID, followeeID)
	}
	s.federationInbox.OnNotification = func(_ context.Context, notificationID, recipientID string) {
		s.publishNotificationCreated(notificationID, recipientID)
	}
	s.federationInbox.OnNoteCreated = func(ctx context.Context, noteID, userID string) {
		// Remote notes land in the global timeline; publishing the
		// streaming event keeps open clients in step with it.
		if s.streamBus != nil {
			s.streamBus.PublishTyped("notesStream", "note", map[string]any{"id": noteID, "userId": userID})
		}
		// A local follower that asked for this account's notes gets a
		// notification, the same as for a local author's note.
		if s.notes != nil {
			if stored, err := s.notes.Get(ctx, noteID); err == nil {
				s.notifyFollowersOfNote(ctx, stored, userID)
			}
		}
	}
	s.federationInbox.OnMoved = func(ctx context.Context, oldUserID, newUserID string) {
		s.migrateFollowersOf(ctx, oldUserID, newUserID)
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

// deliverable reports whether a local note should leave the instance at all.
// Remote-authored notes and local-only ones never do.
func (s *Server) deliverable(n *note.Note) bool {
	return s.federationDeliver != nil && s.fedRenderer != nil &&
		!n.LocalOnly && n.UserHost == nil
}

// deliveryInboxes resolves the remote inboxes a note addresses: the author's
// remote followers, every mentioned account, and the audience of a direct
// note. A direct note must not reach followers, or its recipients would learn
// it exists.
func (s *Server) deliveryInboxes(ctx context.Context, n *note.Note) []string {
	var targets []string
	if n.Visibility != "specified" {
		if inboxes, err := s.federationStore.InboxesForUser(ctx, n.UserID, false); err == nil {
			targets = append(targets, inboxes...)
		} else {
			log.Printf("httpapi: follower inboxes: %v", err)
		}
	}
	if len(n.Mentions) > 0 {
		if inboxes, err := s.federationStore.MentionedInboxes(ctx, n.Mentions); err == nil {
			targets = append(targets, inboxes...)
		}
	}
	// Remote mentions are stored as acct strings; each one needs its actor
	// resolved to an inbox, which also gives the peer the Mention tag it uses
	// to notify the account.
	for _, acct := range n.RemoteMentions {
		user, err := s.resolveRemoteMention(ctx, acct)
		if err != nil || user == nil {
			log.Printf("httpapi: resolve mention %s for delivery: %v", acct, err)
			continue
		}
		inbox := user.SharedInbox
		if inbox == "" {
			inbox = user.Inbox
		}
		if inbox != "" {
			targets = append(targets, inbox)
		}
	}
	if n.Visibility == "specified" {
		audience := s.notes.VisibleUserIDs(ctx, n.ID)
		if inboxes, err := s.federationStore.MentionedInboxes(ctx, audience); err == nil {
			targets = append(targets, inboxes...)
		}
	}
	return targets
}

// objectRef is the ActivityPub id a note is addressed by: the local URL for a
// local note, its stored URI for a remote one.
func (s *Server) objectRef(n *note.Note) string {
	if n.UserHost == nil {
		return s.fedRenderer.NoteRef(n.ID)
	}
	if n.URI != nil && *n.URI != "" {
		return *n.URI
	}
	return ""
}

// DeliverNote fans a newly created local note out to its remote audience. A
// pure renote goes out as an Announce of the original note, everything else
// as a Create of the note itself.
func (s *Server) DeliverNote(ctx context.Context, n *note.Note) {
	if !s.deliverable(n) {
		return
	}
	if n.Visibility != "public" && n.Visibility != "home" && n.Visibility != "followers" && n.Visibility != "specified" {
		return
	}

	u, err := s.accounts.GetByID(ctx, n.UserID)
	if err != nil || u == nil {
		return
	}
	actor := s.fedRenderer.ActorRef(u.ID)

	var activity federation.Object
	if n.IsPureRenote() {
		target, err := s.notes.Get(ctx, *n.RenoteID)
		if err != nil || target == nil {
			log.Printf("httpapi: render announce for %s: %v", n.ID, err)
			return
		}
		object := s.objectRef(target)
		if object == "" {
			return
		}
		activity = s.fedRenderer.RenderAnnounce(n.ID, actor, object, n.CreatedAt)
	} else {
		doc, err := s.renderNote(ctx, n)
		if err != nil {
			log.Printf("httpapi: render note for delivery: %v", err)
			return
		}
		activity = s.fedRenderer.RenderCreate(doc, n.ID, actor, n.CreatedAt)
	}

	seen := map[string]bool{}
	for _, inbox := range s.deliveryInboxes(ctx, n) {
		if seen[inbox] {
			continue
		}
		seen[inbox] = true
		if err := s.federationDeliver.Enqueue(ctx, inbox, s.fedRenderer.ActivityRef(n.ID), s.fedRenderer.KeyID(u.ID), u.ID, activity); err != nil {
			log.Printf("httpapi: enqueue delivery to %s: %v", inbox, err)
		}
	}
}

// DeliverUpdate tells a note's remote audience that its content changed.
// Renotes have nothing of their own to update, so they are skipped.
func (s *Server) DeliverUpdate(ctx context.Context, n *note.Note) {
	if !s.deliverable(n) || n.IsPureRenote() {
		return
	}
	u, err := s.accounts.GetByID(ctx, n.UserID)
	if err != nil || u == nil {
		return
	}
	doc, err := s.renderNote(ctx, n)
	if err != nil {
		log.Printf("httpapi: render note for update: %v", err)
		return
	}
	published := time.Now()
	if n.UpdatedAt != nil {
		published = *n.UpdatedAt
	}
	activity := s.fedRenderer.RenderUpdate(doc, n.ID, s.fedRenderer.ActorRef(u.ID), published)

	seen := map[string]bool{}
	for _, inbox := range s.deliveryInboxes(ctx, n) {
		if seen[inbox] {
			continue
		}
		seen[inbox] = true
		if err := s.federationDeliver.Enqueue(ctx, inbox, "", s.fedRenderer.KeyID(u.ID), u.ID, activity); err != nil {
			log.Printf("httpapi: enqueue update to %s: %v", inbox, err)
		}
	}
}

// DeliverDelete withdraws a note from its remote audience: a Delete for a
// note, an Undo of the Announce for a pure renote.
func (s *Server) DeliverDelete(ctx context.Context, n *note.Note) {
	if !s.deliverable(n) {
		return
	}
	actor := s.fedRenderer.ActorRef(n.UserID)

	var activity federation.Object
	if n.IsPureRenote() {
		target, err := s.notes.Get(ctx, *n.RenoteID)
		object := ""
		if err == nil && target != nil {
			object = s.objectRef(target)
		}
		if object == "" {
			// Without the original note's id an Undo would be empty; a Delete
			// of the renote itself is the weaker but valid signal.
			activity = s.fedRenderer.RenderDelete(s.fedRenderer.NoteRef(n.ID), actor)
		} else {
			announce := s.fedRenderer.RenderAnnounce(n.ID, actor, object, n.CreatedAt)
			activity = s.fedRenderer.RenderUndo(announce, actor)
		}
	} else {
		activity = s.fedRenderer.RenderDelete(s.fedRenderer.NoteRef(n.ID), actor)
	}

	seen := map[string]bool{}
	for _, inbox := range s.deliveryInboxes(ctx, n) {
		if seen[inbox] {
			continue
		}
		seen[inbox] = true
		if err := s.federationDeliver.Enqueue(ctx, inbox, "", s.fedRenderer.KeyID(n.UserID), n.UserID, activity); err != nil {
			log.Printf("httpapi: enqueue delete to %s: %v", inbox, err)
		}
	}
}

var _ = time.Now
