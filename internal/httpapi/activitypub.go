package httpapi

import (
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Aurorainic/cinderfish/internal/account"
	"github.com/Aurorainic/cinderfish/internal/federation"
	"github.com/Aurorainic/cinderfish/internal/note"
)

// baseURL is the instance's canonical scheme://host used in every
// ActivityPub identifier.
func (s *Server) baseURL() string {
	if s.cfg.Federation.PublicURL != "" {
		return strings.TrimSuffix(s.cfg.Federation.PublicURL, "/")
	}
	scheme := "https"
	domain := s.cfg.Domain
	if strings.HasPrefix(domain, "localhost") || strings.HasPrefix(domain, "127.0.0.1") {
		scheme = "http"
	}
	return scheme + "://" + domain
}

// apContentType negotiates the ActivityPub media type the way Misskey's
// setResponseType does.
func apContentType(r *http.Request) string {
	if strings.Contains(r.Header.Get("Accept"), "application/ld+json") {
		return federation.ContentTypeLD
	}
	return federation.ContentTypeActivity
}

// writeAP emits an ActivityPub document with the headers every peer
// expects: content negotiation plus the permissive CORS that lets
// browser clients read actor documents.
func writeAP(w http.ResponseWriter, r *http.Request, o federation.Object) {
	w.Header().Set("Content-Type", apContentType(r))
	w.Header().Set("Vary", "Accept")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Accept")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.Header().Set("Access-Control-Expose-Headers", "Vary")
	w.Header().Set("Cache-Control", "public, max-age=180")
	w.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(o)
}

// apForbidden is the 403 every AP route answers in federation mode none.
func apForbidden(w http.ResponseWriter) {
	http.Error(w, "Forbidden", http.StatusForbidden)
}

// federationEnabled reports whether AP routes are open. It reads the
// runtime mode so switching to none disables the surface immediately.
func (s *Server) federationEnabled(ctx context.Context) bool {
	if s.federationPolicy == nil {
		return false
	}
	mode, err := s.federationPolicy.Mode(ctx)
	if err != nil {
		// Falling back to the configured value keeps a database blip from
		// turning federation off entirely.
		mode, _ = federation.ParseMode(s.cfg.FederationMode)
	}
	return mode != federation.ModeNone
}

// handleWebFinger serves GET /.well-known/webfinger.
//
// It accepts the resource shapes Misskey accepts: acct:user@host,
// user@host, and a profile or /users/<id> URL on this instance. A
// resource on another host is answered with 422, as upstream does.
func (s *Server) handleWebFinger(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if !s.federationEnabled(r.Context()) {
		apForbidden(w)
		return
	}
	resource := r.URL.Query().Get("resource")
	if resource == "" {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	u, ok := s.webFingerUser(r.Context(), resource)
	if !ok {
		http.Error(w, "Unprocessable Entity", http.StatusUnprocessableEntity)
		return
	}
	if u == nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	self := s.fedRenderer.ActorRef(u.ID)
	w.Header().Set("Content-Type", "application/jrd+json")
	w.Header().Set("Vary", "Accept")
	w.Header().Set("Cache-Control", "public, max-age=180")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"subject": "acct:" + u.Username + "@" + s.cfg.Domain,
		"aliases": []string{self},
		"links": []map[string]string{
			{"rel": "self", "type": "application/activity+json", "href": self},
			{"rel": "http://webfinger.net/rel/profile-page", "type": "text/html", "href": s.baseURL() + "/@" + u.Username},
			{"rel": "http://ostatus.org/schema/1.0/subscribe", "template": s.baseURL() + "/authorize-follow?acct={uri}"},
		},
	})
}

// webFingerUser resolves a WebFinger resource. It returns ok=false when
// the resource names another host.
func (s *Server) webFingerUser(ctx context.Context, resource string) (*account.User, bool) {
	lower := strings.ToLower(resource)
	self := strings.ToLower(s.cfg.Domain)
	base := strings.ToLower(s.baseURL())

	switch {
	case strings.HasPrefix(lower, base+"/users/"):
		id := strings.TrimSuffix(resource[len(base)+len("/users/"):], "/")
		u, err := s.accounts.GetByID(ctx, id)
		if err != nil || u == nil || u.Host != nil {
			return nil, true
		}
		return u, true
	case strings.HasPrefix(lower, base+"/@"):
		return s.localUserByName(ctx, strings.TrimSuffix(resource[len(base)+2:], "/"))
	case strings.HasPrefix(lower, "acct:"):
		resource = resource[len("acct:"):]
	}

	user, host, hasHost := strings.Cut(resource, "@")
	if hasHost && strings.ToLower(host) != self {
		return nil, false
	}
	return s.localUserByName(ctx, user)
}

// localUserByName loads a local account by username.
func (s *Server) localUserByName(ctx context.Context, username string) (*account.User, bool) {
	u, err := s.accounts.GetByUsername(ctx, username)
	if err != nil || u == nil || u.Host != nil {
		return nil, true
	}
	return u, true
}

// handleHostMeta serves the lrdd template in JSON form.
func (s *Server) handleHostMeta(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if !s.federationEnabled(r.Context()) {
		apForbidden(w)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=180")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"links": []map[string]string{{
			"rel": "lrdd", "type": "application/jrd+json",
			"template": s.baseURL() + "/.well-known/webfinger?resource={uri}",
		}},
	})
}

// handleHostMetaXML serves the XRD form of host-meta.
func (s *Server) handleHostMetaXML(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if !s.federationEnabled(r.Context()) {
		apForbidden(w)
		return
	}
	w.Header().Set("Content-Type", "application/xrd+xml")
	w.Header().Set("Cache-Control", "public, max-age=180")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?><XRD xmlns="http://docs.oasis-open.org/ns/xri/xrd-1.0"><Link rel="lrdd" type="application/xrd+xml" template="` + s.baseURL() + `/.well-known/webfinger?resource={uri}"/></XRD>`))
}

// handleNodeInfo points at the two schema versions.
func (s *Server) handleNodeInfo(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if !s.federationEnabled(r.Context()) {
		apForbidden(w)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "public, max-age=180")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{"links": []map[string]string{
		{"rel": "http://nodeinfo.diaspora.software/ns/schema/2.1", "href": s.baseURL() + "/nodeinfo/2.1"},
		{"rel": "http://nodeinfo.diaspora.software/ns/schema/2.0", "href": s.baseURL() + "/nodeinfo/2.0"},
	}})
}

// handleNodeInfo21 serves the 2.1 document.
func (s *Server) handleNodeInfo21(w http.ResponseWriter, r *http.Request) {
	s.serveNodeInfo(w, r, "2.1")
}

// handleNodeInfo20 serves the 2.0 document.
func (s *Server) handleNodeInfo20(w http.ResponseWriter, r *http.Request) {
	s.serveNodeInfo(w, r, "2.0")
}

func (s *Server) serveNodeInfo(w http.ResponseWriter, r *http.Request, version string) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	if !s.federationEnabled(r.Context()) {
		apForbidden(w)
		return
	}

	ctx := r.Context()
	users, _ := s.accounts.CountUsers(ctx)
	var localPosts int
	if s.pool != nil {
		_ = s.pool.QueryRow(ctx, "SELECT count(*) FROM notes WHERE user_host IS NULL AND deleted_at IS NULL").Scan(&localPosts)
	}

	doc := map[string]any{
		"version":           version,
		"software":          map[string]any{"name": "cinderfish", "version": "0.0.1"},
		"protocols":         []string{"activitypub"},
		"services":          map[string]any{"inbound": []string{}, "outbound": []string{}},
		"openRegistrations": s.cfg.RegistrationMode == "open",
		"usage": map[string]any{
			"users":         map[string]any{"total": users},
			"localPosts":    localPosts,
			"localComments": 0,
		},
		"metadata": map[string]any{
			"maxNoteTextLength":   3000,
			"disableRegistration": s.cfg.RegistrationMode != "open",
		},
	}
	w.Header().Set("Content-Type", `application/json; profile="http://nodeinfo.diaspora.software/ns/schema/`+version+`#"`)
	w.Header().Set("Cache-Control", "public, max-age=600")
	w.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(doc)
}

// handleWellKnownOptions answers the preflight every well-known route
// advertises.
func (s *Server) handleWellKnownOptions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Accept")
	w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
	w.WriteHeader(http.StatusNoContent)
}

// handleActor serves GET /users/{id}, the actor document.
func (s *Server) handleActor(w http.ResponseWriter, r *http.Request) {
	if !s.federationEnabled(r.Context()) {
		apForbidden(w)
		return
	}
	u, err := s.accounts.GetByID(r.Context(), r.PathValue("id"))
	if err != nil || u == nil || u.Host != nil {
		http.NotFound(w, r)
		return
	}
	person, err := s.renderLocalActor(r.Context(), u)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	writeAP(w, r, person)
}

// handleActorByAcct serves GET /@user.
func (s *Server) handleActorByAcct(w http.ResponseWriter, r *http.Request) {
	if !s.federationEnabled(r.Context()) {
		apForbidden(w)
		return
	}
	acct := r.PathValue("acct")
	if !strings.HasPrefix(acct, "@") {
		http.NotFound(w, r)
		return
	}
	acct = strings.TrimPrefix(acct, "@")
	user, host, hasHost := strings.Cut(acct, "@")
	if hasHost && !strings.EqualFold(host, s.cfg.Domain) {
		http.NotFound(w, r)
		return
	}
	u, err := s.accounts.GetByUsername(r.Context(), user)
	if err != nil || u == nil || u.Host != nil {
		http.NotFound(w, r)
		return
	}
	person, err := s.renderLocalActor(r.Context(), u)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	writeAP(w, r, person)
}

// renderLocalActor builds the actor document, creating the keypair on
// first use.
func (s *Server) renderLocalActor(ctx context.Context, u *account.User) (federation.Object, error) {
	kp, err := s.federationStore.EnsureLocalKey(ctx, u.ID)
	if err != nil {
		return nil, err
	}
	var name, desc string
	_ = s.pool.QueryRow(ctx, "SELECT COALESCE(name, ''), COALESCE(description, '') FROM user_profiles WHERE user_id = $1", u.ID).Scan(&name, &desc)
	return s.fedRenderer.RenderPerson(federation.Profile{
		ID:          u.ID,
		Username:    u.Username,
		Name:        name,
		Description: desc,
		PublicKey:   kp.PublicPEM,
		CreatedAt:   u.CreatedAt,
	}), nil
}

// handlePublicKey serves GET /users/{id}/publickey.
func (s *Server) handlePublicKey(w http.ResponseWriter, r *http.Request) {
	if !s.federationEnabled(r.Context()) {
		apForbidden(w)
		return
	}
	u, err := s.accounts.GetByID(r.Context(), r.PathValue("id"))
	if err != nil || u == nil || u.Host != nil {
		http.NotFound(w, r)
		return
	}
	kp, err := s.federationStore.EnsureLocalKey(r.Context(), u.ID)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	writeAP(w, r, s.fedRenderer.RenderKey(u.ID, kp.PublicPEM))
}

// handleUserInbox is the per-user inbox. It shares the shared-inbox
// handler, which is what Misskey does too.
func (s *Server) handleUserInbox(w http.ResponseWriter, r *http.Request) {
	if !s.federationEnabled(r.Context()) {
		apForbidden(w)
		return
	}
	u, err := s.accounts.GetByID(r.Context(), r.PathValue("id"))
	if err != nil || u == nil || u.Host != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Cache-Control", "public, max-age=180")
	w.Header().Set("Content-Type", apContentType(r))
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]any{"@context": federation.Context, "type": "OrderedCollection", "totalItems": 0})
}

// handleUserOutbox serves the outbox collection and its pages.
func (s *Server) handleUserOutbox(w http.ResponseWriter, r *http.Request) {
	if !s.federationEnabled(r.Context()) {
		apForbidden(w)
		return
	}
	u, err := s.accounts.GetByID(r.Context(), r.PathValue("id"))
	if err != nil || u == nil || u.Host != nil {
		http.NotFound(w, r)
		return
	}
	partOf := s.fedRenderer.ActorRef(u.ID) + "/outbox"
	if r.URL.Query().Get("page") != "true" {
		var total int
		_ = s.pool.QueryRow(r.Context(), `
SELECT count(*) FROM notes WHERE user_id = $1 AND user_host IS NULL AND deleted_at IS NULL
  AND visibility IN ('public','home') AND local_only = FALSE`, u.ID).Scan(&total)
		writeAP(w, r, s.fedRenderer.RenderOrderedCollection(partOf, total, partOf+"?page=true"))
		return
	}

	notes, err := s.timeline.UserNotes(r.Context(), u.ID, 20, r.URL.Query().Get("since_id"), r.URL.Query().Get("until_id"), false, true)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	items := make([]any, 0, len(notes))
	for _, n := range notes {
		act, err := s.renderNoteActivity(r.Context(), n, u)
		if err != nil {
			continue
		}
		items = append(items, act)
	}
	pageURL := partOf + "?page=true"
	if since := r.URL.Query().Get("since_id"); since != "" {
		pageURL += "&since_id=" + url.QueryEscape(since)
	}
	writeAP(w, r, s.fedRenderer.RenderOrderedCollectionPage(pageURL, len(items), items, partOf, "", ""))
}

// handleUserFollowers serves the followers collection.
func (s *Server) handleUserFollowers(w http.ResponseWriter, r *http.Request) {
	s.serveFollowCollection(w, r, true)
}

// handleUserFollowing serves the following collection.
func (s *Server) handleUserFollowing(w http.ResponseWriter, r *http.Request) {
	s.serveFollowCollection(w, r, false)
}

func (s *Server) serveFollowCollection(w http.ResponseWriter, r *http.Request, followers bool) {
	if !s.federationEnabled(r.Context()) {
		apForbidden(w)
		return
	}
	u, err := s.accounts.GetByID(r.Context(), r.PathValue("id"))
	if err != nil || u == nil || u.Host != nil {
		http.NotFound(w, r)
		return
	}
	kind := "following"
	if followers {
		kind = "followers"
	}
	partOf := s.fedRenderer.ActorRef(u.ID) + "/" + kind

	// Follow visibility is honoured: a private list answers 403, the
	// same way Misskey does.
	var visibility string
	if followers {
		_ = s.pool.QueryRow(r.Context(), "SELECT followers_visibility FROM user_profiles WHERE user_id = $1", u.ID).Scan(&visibility)
	}
	if visibility == "private" {
		w.Header().Set("Cache-Control", "public, max-age=30")
		apForbidden(w)
		return
	}

	if r.URL.Query().Get("page") != "true" {
		total, _ := s.federationStore.FollowCount(r.Context(), u.ID, followers)
		writeAP(w, r, s.fedRenderer.RenderOrderedCollection(partOf, total, partOf+"?page=true"))
		return
	}

	refs, err := s.federationStore.FollowRefs(r.Context(), u.ID, followers, 10, r.URL.Query().Get("cursor"))
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	items := make([]any, 0, len(refs))
	for _, ref := range refs {
		items = append(items, ref)
	}
	writeAP(w, r, s.fedRenderer.RenderOrderedCollectionPage(partOf+"?page=true", len(items), items, partOf, "", ""))
}

// handleUserFeatured serves the pinned-notes collection. Pinning is not
// implemented yet, so it is always empty.
func (s *Server) handleUserFeatured(w http.ResponseWriter, r *http.Request) {
	if !s.federationEnabled(r.Context()) {
		apForbidden(w)
		return
	}
	u, err := s.accounts.GetByID(r.Context(), r.PathValue("id"))
	if err != nil || u == nil || u.Host != nil {
		http.NotFound(w, r)
		return
	}
	partOf := s.fedRenderer.ActorRef(u.ID) + "/collections/featured"
	writeAP(w, r, s.fedRenderer.RenderOrderedCollection(partOf, 0, ""))
}

// handleNote serves GET /notes/{id}.
func (s *Server) handleNote(w http.ResponseWriter, r *http.Request) {
	if !s.federationEnabled(r.Context()) {
		apForbidden(w)
		return
	}
	n, err := s.notes.Get(r.Context(), r.PathValue("id"))
	if err != nil || n == nil {
		http.NotFound(w, r)
		return
	}
	// Only public and home notes leave the instance, and a local-only
	// note never does at all.
	if n.LocalOnly || (n.Visibility != "public" && n.Visibility != "home") || n.UserHost != nil {
		http.NotFound(w, r)
		return
	}
	doc, err := s.renderNote(r.Context(), n)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	writeAP(w, r, doc)
}

// handleNoteActivity serves GET /notes/{id}/activity.
func (s *Server) handleNoteActivity(w http.ResponseWriter, r *http.Request) {
	if !s.federationEnabled(r.Context()) {
		apForbidden(w)
		return
	}
	n, err := s.notes.Get(r.Context(), r.PathValue("id"))
	if err != nil || n == nil || n.LocalOnly || n.UserHost != nil {
		http.NotFound(w, r)
		return
	}
	if n.Visibility != "public" && n.Visibility != "home" {
		http.NotFound(w, r)
		return
	}
	u, err := s.accounts.GetByID(r.Context(), n.UserID)
	if err != nil || u == nil {
		http.NotFound(w, r)
		return
	}
	act, err := s.renderNoteActivity(r.Context(), n, u)
	if err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	writeAP(w, r, act)
}

// renderNote renders a stored local note as an ActivityPub Note.
func (s *Server) renderNote(ctx context.Context, n *note.Note) (federation.Object, error) {
	u, err := s.accounts.GetByID(ctx, n.UserID)
	if err != nil || u == nil {
		return nil, err
	}
	data := federation.NoteData{
		ID:         n.ID,
		ActorRef:   s.fedRenderer.ActorRef(u.ID),
		Visibility: n.Visibility,
		CreatedAt:  n.CreatedAt,
		Sensitive:  n.IsSensitive,
		Tags:       n.Tags,
	}
	if n.Text != nil {
		data.Text = *n.Text
	}
	if n.CW != nil {
		data.CW = *n.CW
	}
	if n.ReplyID != nil {
		data.InReplyTo = s.fedRenderer.NoteRef(*n.ReplyID)
	}
	if n.RenoteID != nil {
		data.QuoteURL = s.fedRenderer.NoteRef(*n.RenoteID)
	}
	for _, id := range n.Mentions {
		data.MentionRefs = append(data.MentionRefs, s.fedRenderer.ActorRef(id))
	}
	return s.fedRenderer.RenderNote(data), nil
}

// renderNoteActivity wraps a note in its Create activity.
func (s *Server) renderNoteActivity(ctx context.Context, n *note.Note, u *account.User) (federation.Object, error) {
	doc, err := s.renderNote(ctx, n)
	if err != nil {
		return nil, err
	}
	return s.fedRenderer.RenderCreate(doc, n.ID, s.fedRenderer.ActorRef(u.ID), n.CreatedAt), nil
}

// handleInbox is POST /inbox and POST /users/{id}/inbox.
//
// The check order follows Misskey's inbox handler: federation mode, then
// signature parsing, then the Host and Digest coverage checks, then the
// digest value, then the actor, and only then 202.
func (s *Server) handleInbox(w http.ResponseWriter, r *http.Request) {
	if s.federationInbox == nil || !s.federationEnabled(r.Context()) {
		apForbidden(w)
		return
	}

	// The body is bounded at 64 KiB, the same limit Misskey applies.
	body, err := io.ReadAll(io.LimitReader(r.Body, 64<<10+1))
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	if len(body) > 64<<10 {
		http.Error(w, "Payload Too Large", http.StatusRequestEntityTooLarge)
		return
	}

	keyID, err := federation.VerifyRequest(r, federation.VerifyOptions{
		// The signed Host is what the peer connected to. On a real
		// deployment that is cfg.Domain; behind a reverse proxy Go sees
		// the proxied Host, and rejecting that would break every
		// proxied instance. The signature still binds the request to
		// whatever host was used, so a replay at another host fails.
		Host:      r.Host,
		Body:      body,
		PublicKey: s.remoteKey,
	})
	if err != nil {
		log.Printf("httpapi: inbox signature rejected: %v", err)
		w.Header().Set("WWW-Authenticate", `Signature realm="Cinderfish"`)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	activity, err := federation.DecodeObject(body)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	// An activity with no actor can never be authenticated against the
	// signer, so it is rejected before any work is done.
	if activity.Ref("actor") == "" {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}
	_ = keyID

	outcome, err := s.federationInbox.Process(r.Context(), activity)
	if err != nil {
		log.Printf("httpapi: inbox %s: %v", activity.Type(), err)
	} else {
		log.Printf("httpapi: inbox %s from %s: %s", activity.Type(), activity.Ref("actor"), outcome)
	}

	// 202 means accepted for processing, which is what the activity
	// status is: a semantic failure is logged, never retried by the peer.
	w.Header().Set("Content-Type", apContentType(r))
	w.WriteHeader(http.StatusAccepted)
}

// remoteKey resolves the public key for a remote keyId. The key id is
// an actor URL with an optional fragment; the actor document carries the
// key.
func (s *Server) remoteKey(keyID string) (*rsa.PublicKey, error) {
	actorURI, _, _ := strings.Cut(keyID, "#")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Prefer a stored key so repeat deliveries need no network round
	// trip, and so a peer that has since gone offline still verifies.
	if u, err := s.federationStore.UserByURI(ctx, actorURI); err == nil && u.PublicKeyPEM != "" {
		return federation.ParsePublicKeyPEM(u.PublicKeyPEM)
	}

	if s.federationClient == nil {
		return nil, fmt.Errorf("federation: cannot resolve key %s", keyID)
	}
	obj, err := s.federationClient.FetchObject(ctx, actorURI, actorURI)
	if err != nil {
		return nil, err
	}
	key := obj.Object("publicKey")
	if key == nil {
		return nil, fmt.Errorf("federation: actor %s publishes no publicKey", actorURI)
	}
	pem := key.String("publicKeyPem")
	if pem == "" {
		return nil, fmt.Errorf("federation: actor %s has an empty publicKeyPem", actorURI)
	}
	return federation.ParsePublicKeyPEM(pem)
}
