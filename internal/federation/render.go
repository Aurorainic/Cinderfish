package federation

import (
	"fmt"
	"html"
	"strings"
	"time"
)

// Renderer turns Cinderfish's stored rows into ActivityPub documents. It
// holds only the instance's own identity; anything that needs the
// database is passed in by the caller so rendering stays pure and
// testable.
type Renderer struct {
	// BaseURL is the instance's scheme://host, no trailing slash.
	BaseURL string
}

// NewRenderer returns a renderer for the given base URL.
func NewRenderer(baseURL string) *Renderer {
	return &Renderer{BaseURL: strings.TrimSuffix(baseURL, "/")}
}

// ActorRef returns the ActivityPub actor id for a local user id.
func (r *Renderer) ActorRef(userID string) string {
	return r.BaseURL + "/users/" + userID
}

// KeyID returns the actor's signing key id.
func (r *Renderer) KeyID(userID string) string {
	return r.ActorRef(userID) + "#main-key"
}

// NoteRef returns the ActivityPub id of a local note.
func (r *Renderer) NoteRef(noteID string) string {
	return r.BaseURL + "/notes/" + noteID
}

// ActivityRef returns the id of the Create activity for a local note.
func (r *Renderer) ActivityRef(noteID string) string {
	return r.NoteRef(noteID) + "/activity"
}

// Profile is the data RenderPerson needs. It mirrors the columns rather
// than the account/note types so the federation package does not depend
// on them.
type Profile struct {
	ID          string
	Username    string
	Name        string
	Description string // MFM
	AvatarURL   string
	BannerURL   string
	Fields      []ProfileField
	IsBot       bool
	IsLocked    bool
	CreatedAt   time.Time
	PublicKey   string

	// MovedToURI and AlsoKnownAs carry the account migration state. Both are
	// published so peers can validate a Move: the target actor must list the
	// mover's URI in alsoKnownAs before anyone follows the move.
	MovedToURI  string
	AlsoKnownAs []string
}

// ProfileField is one PropertyValue on the actor document.
type ProfileField struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// RenderPerson builds the actor document for a local user.
//
// The field set follows ApRendererService.renderPerson: the five
// collection URLs are all under /users/<id>, the shared inbox points at
// the instance-level /inbox, and the key is embedded so peers can verify
// signatures without a second fetch.
func (r *Renderer) RenderPerson(p Profile) Object {
	id := r.ActorRef(p.ID)
	isSystem := strings.Contains(p.Username, ".")

	typ := "Person"
	switch {
	case isSystem:
		typ = "Application"
	case p.IsBot:
		typ = "Service"
	}

	person := Object{
		"id":                        id,
		"type":                      typ,
		"inbox":                     id + "/inbox",
		"outbox":                    id + "/outbox",
		"followers":                 id + "/followers",
		"following":                 id + "/following",
		"featured":                  id + "/collections/featured",
		"sharedInbox":               r.BaseURL + "/inbox",
		"endpoints":                 Object{"sharedInbox": r.BaseURL + "/inbox"},
		"url":                       r.BaseURL + "/@" + p.Username,
		"preferredUsername":         p.Username,
		"name":                      p.Name,
		"summary":                   MFMToHTML(p.Description),
		"_misskey_summary":          p.Description,
		"manuallyApprovesFollowers": p.IsLocked,
		"discoverable":              true,
		"publicKey": Object{
			"id":           r.KeyID(p.ID),
			"type":         "Key",
			"owner":        id,
			"publicKeyPem": p.PublicKey,
		},
	}

	if !p.CreatedAt.IsZero() {
		person["published"] = p.CreatedAt.UTC().Format(time.RFC3339)
	}
	if p.AvatarURL != "" {
		person["icon"] = Object{"type": "Image", "url": p.AvatarURL}
	} else {
		person["icon"] = nil
	}
	if p.BannerURL != "" {
		person["image"] = Object{"type": "Image", "url": p.BannerURL}
	} else {
		person["image"] = nil
	}

	if len(p.Fields) > 0 {
		attachments := make([]any, 0, len(p.Fields))
		for _, f := range p.Fields {
			attachments = append(attachments, Object{
				"type":  "PropertyValue",
				"name":  f.Name,
				"value": linkifyField(f.Value),
			})
		}
		person["attachment"] = attachments
	}

	// Account migration, as ApRendererService.renderPerson does: both fields
	// are omitted when empty rather than sent as null.
	if p.MovedToURI != "" {
		person["movedTo"] = p.MovedToURI
	}
	if len(p.AlsoKnownAs) > 0 {
		person["alsoKnownAs"] = p.AlsoKnownAs
	}

	return withContext(person)
}

// linkifyField turns a bare URL in a profile field into an anchor, the
// way Misskey's tryRewriteUrl does, so the value renders as a link in
// peers' web clients.
func linkifyField(v string) string {
	if !strings.HasPrefix(v, "http://") && !strings.HasPrefix(v, "https://") {
		return v
	}
	esc := html.EscapeString(v)
	return `<a href="` + esc + `" rel="me nofollow noopener" target="_blank">` + esc + `</a>`
}

// NoteData is the data RenderNote needs.
type NoteData struct {
	// ID is the local note id; used to build the object URL.
	ID string
	// APID overrides the object id for notes that came from a remote
	// instance (it is their ActivityPub uri).
	APID string
	// ActorRef is the author's ActivityPub actor id.
	ActorRef string
	// ActorHost is empty for local authors.
	ActorHost  string
	Visibility string
	Text       string
	CW         string
	InReplyTo  string
	QuoteURL   string
	Tags       []string
	// Emojis are the custom emoji the note text uses, so a peer can render
	// them instead of showing the shortcode.
	Emojis []EmojiRef
	// MentionRefs are the actor ids of the mentioned users.
	MentionRefs []string
	Attachments []Object
	Sensitive   bool
	CreatedAt   time.Time
	// Poll makes the note a Question. Empty choices keep it a Note.
	PollChoices   []string
	PollMultiple  bool
	PollExpiresAt *time.Time
}

// RenderNote builds the ActivityPub representation of a note.
//
// The visibility-to-to/cc mapping and the _misskey_content/_misskey_quote
// extensions follow ApRendererService.renderNote. content is generated
// from the note's MFM text (see MFMToHTML) since Cinderfish has no stored
// HTML yet.
func (r *Renderer) RenderNote(n NoteData) Object {
	id := n.APID
	if id == "" {
		id = r.NoteRef(n.ID)
	}
	followers := n.ActorRef + "/followers"

	tos := []string{}
	ccs := []string{}
	switch n.Visibility {
	case "public":
		tos = []string{Public}
		ccs = append([]string{followers}, n.MentionRefs...)
	case "home":
		tos = []string{followers}
		ccs = append([]string{Public}, n.MentionRefs...)
	case "followers":
		tos = []string{followers}
		ccs = append(ccs, n.MentionRefs...)
	default: // specified
		tos = append(tos, n.MentionRefs...)
	}

	note := Object{
		"id":               id,
		"type":             "Note",
		"attributedTo":     n.ActorRef,
		"content":          MFMToHTML(n.Text),
		"to":               tos,
		"cc":               ccs,
		"sensitive":        n.Sensitive,
		"published":        n.CreatedAt.UTC().Format(time.RFC3339),
		"_misskey_content": n.Text,
		"source": Object{
			"content":   n.Text,
			"mediaType": "text/x.misskeymarkdown",
		},
	}

	if n.CW != "" {
		note["summary"] = n.CW
	}
	if n.InReplyTo != "" {
		note["inReplyTo"] = n.InReplyTo
	}
	if n.QuoteURL != "" {
		note["_misskey_quote"] = n.QuoteURL
		note["quoteUrl"] = n.QuoteURL
	}

	tag := make([]any, 0, len(n.Tags)+len(n.MentionRefs))
	for _, t := range n.Tags {
		tag = append(tag, Object{
			"type": "Hashtag",
			"href": r.BaseURL + "/tags/" + t,
			"name": "#" + t,
		})
	}
	for _, m := range n.MentionRefs {
		tag = append(tag, Object{"type": "Mention", "href": m})
	}
	for _, e := range n.Emojis {
		tag = append(tag, Object{
			"type": "Emoji",
			"name": ":" + e.Name + ":",
			"icon": Object{"type": "Image", "url": e.URL},
		})
	}
	if len(tag) > 0 {
		note["tag"] = tag
	}

	if len(n.Attachments) > 0 {
		atts := make([]any, 0, len(n.Attachments))
		for _, a := range n.Attachments {
			atts = append(atts, a)
		}
		note["attachment"] = atts
		// A note carrying a sensitive attachment is sensitive as a
		// whole, which is what peers use to gate the media.
		if n.Sensitive {
			note["sensitive"] = true
		}
	}

	// A poll federates as a Question. Misskey wraps each choice in a Note
	// with an empty replies collection; Mastodon reads the same shape.
	if len(n.PollChoices) >= 2 {
		note["type"] = "Question"
		choices := make([]any, 0, len(n.PollChoices))
		for _, choice := range n.PollChoices {
			choices = append(choices, Object{
				"type":    "Note",
				"name":    choice,
				"replies": Object{"type": "Collection", "totalItems": 0},
			})
		}
		if n.PollMultiple {
			note["anyOf"] = choices
		} else {
			note["oneOf"] = choices
		}
		if n.PollExpiresAt != nil {
			note["endTime"] = n.PollExpiresAt.UTC().Format(time.RFC3339)
		}
	}

	return withContext(note)
}

// RenderCreate wraps a rendered note in a Create activity. The activity
// inherits the note's to/cc so that relays and peers that only read the
// activity audience still address the right servers.
func (r *Renderer) RenderCreate(note Object, noteID string, actorRef string, published time.Time) Object {
	activity := Object{
		"id":        r.ActivityRef(noteID),
		"type":      "Create",
		"actor":     actorRef,
		"published": published.UTC().Format(time.RFC3339),
		"object":    note,
	}
	if to, ok := note["to"]; ok {
		activity["to"] = to
	}
	if cc, ok := note["cc"]; ok {
		activity["cc"] = cc
	}
	return withContext(activity)
}

// RenderUpdate wraps a rendered note in an Update activity.
//
// The id carries the edit's timestamp so a peer that de-duplicates by activity
// id does not drop the second edit of the same note. Peers that ignore status
// updates (Mastodon) keep the old copy; Misskey, Firefish and Iceshrimp apply
// it.
func (r *Renderer) RenderUpdate(note Object, noteID string, actorRef string, published time.Time) Object {
	activity := Object{
		"id":        fmt.Sprintf("%s#updates/%d", r.ActivityRef(noteID), published.UnixMilli()),
		"type":      "Update",
		"actor":     actorRef,
		"published": published.UTC().Format(time.RFC3339),
		"object":    note,
	}
	if to, ok := note["to"]; ok {
		activity["to"] = to
	}
	if cc, ok := note["cc"]; ok {
		activity["cc"] = cc
	}
	return withContext(activity)
}

// RenderDelete builds a Delete activity. object may be an id string or the
// embedded object; the string form is what every peer accepts.
func (r *Renderer) RenderDelete(objectID, actorRef string) Object {
	return withContext(Object{
		"id":        objectID + "#delete",
		"type":      "Delete",
		"actor":     actorRef,
		"object":    objectID,
		"published": time.Now().UTC().Format(time.RFC3339),
	})
}

// RenderFollow builds a Follow activity from a local user to a remote
// actor. id is the Follow activity id, which the peer echoes back in its
// Accept.
func (r *Renderer) RenderFollow(id, actorRef, objectRef string) Object {
	return withContext(Object{
		"id":     id,
		"type":   "Follow",
		"actor":  actorRef,
		"object": objectRef,
	})
}

// RenderFlag builds a Flag activity: a moderation complaint forwarded to the
// instance hosting the reported account. Misskey's renderFlag carries actor,
// content and object (the reported actor's URI) and no id, so this matches.
func (r *Renderer) RenderFlag(actorRef, objectRef, content string) Object {
	return withContext(Object{
		"type":    "Flag",
		"actor":   actorRef,
		"content": content,
		"object":  objectRef,
	})
}

// RenderAccept builds an Accept activity for an inbound Follow. The
// embedded object is the original Follow, which is what lets the peer
// match the acceptance to its request.
func (r *Renderer) RenderAccept(actorRef string, follow Object) Object {
	return withContext(Object{
		"id":     follow.Ref("id") + "#accept",
		"type":   "Accept",
		"actor":  actorRef,
		"object": follow,
	})
}

// RenderReject builds a Reject activity for an inbound Follow.
func (r *Renderer) RenderReject(actorRef string, follow Object) Object {
	return withContext(Object{
		"id":     follow.Ref("id") + "#reject",
		"type":   "Reject",
		"actor":  actorRef,
		"object": follow,
	})
}

// RenderUndo builds an Undo activity for a previously sent activity.
func (r *Renderer) RenderUndo(activity Object, actorRef string) Object {
	undo := Object{
		"id":        activity.Ref("id") + "/undo",
		"type":      "Undo",
		"actor":     actorRef,
		"object":    activity,
		"published": time.Now().UTC().Format(time.RFC3339),
	}
	if undo["id"] == "/undo" {
		delete(undo, "id")
	}
	return withContext(undo)
}

// RenderMove announces that a local account migrated to the account at
// targetRef. Both object and target are actor URIs; Firefish and Misskey
// send the mover as object and the destination as target.
func (r *Renderer) RenderMove(id, actorRef, objectRef, targetRef string) Object {
	return withContext(Object{
		"id":        id,
		"type":      "Move",
		"actor":     actorRef,
		"object":    objectRef,
		"target":    targetRef,
		"published": time.Now().UTC().Format(time.RFC3339),
	})
}

// RenderLike builds a Like activity carrying a Misskey reaction.
func (r *Renderer) RenderLike(id, actorRef, objectRef, reaction string) Object {
	like := Object{
		"id":      id,
		"type":    "Like",
		"actor":   actorRef,
		"object":  objectRef,
		"content": reaction,
	}
	// Misskey reads _misskey_reaction first and falls back to content
	// and name; emitting both keeps Misskey and Mastodon happy.
	like["_misskey_reaction"] = reaction
	return withContext(like)
}

// RenderAnnounce builds an Announce (renote) activity.
func (r *Renderer) RenderAnnounce(noteID, actorRef, objectRef string, published time.Time) Object {
	return withContext(Object{
		"id":        r.ActivityRef(noteID),
		"type":      "Announce",
		"actor":     actorRef,
		"object":    objectRef,
		"published": published.UTC().Format(time.RFC3339),
		"to":        []string{Public},
	})
}

// RenderOrderedCollection builds the first page link of a collection.
func (r *Renderer) RenderOrderedCollection(id string, totalItems int, firstURL string) Object {
	c := Object{
		"id":         id,
		"type":       "OrderedCollection",
		"totalItems": totalItems,
	}
	if firstURL != "" {
		c["first"] = firstURL
	}
	return withContext(c)
}

// RenderOrderedCollectionPage builds one page of an ordered collection.
func (r *Renderer) RenderOrderedCollectionPage(id string, totalItems int, items []any, partOf string, prev, next string) Object {
	page := Object{
		"id":           id,
		"type":         "OrderedCollectionPage",
		"partOf":       partOf,
		"totalItems":   totalItems,
		"orderedItems": items,
	}
	if prev != "" {
		page["prev"] = prev
	}
	if next != "" {
		page["next"] = next
	}
	return withContext(page)
}

// RenderKey builds the standalone public key document served at
// /users/<id>/publickey.
func (r *Renderer) RenderKey(userID, publicKeyPEM string) Object {
	return withContext(Object{
		"id":           r.KeyID(userID),
		"type":         "Key",
		"owner":        r.ActorRef(userID),
		"publicKeyPem": publicKeyPEM,
	})
}

// RenderEmptyCollectionPage is the empty page answer for a user with no
// entries, so peers stop paging immediately.
func (r *Renderer) RenderEmptyCollectionPage(id, partOf string) Object {
	return r.RenderOrderedCollectionPage(id, 0, []any{}, partOf, "", "")
}

// String renders an Object for logs and error messages.
func (o Object) StringDump() string {
	return fmt.Sprintf("%v", map[string]any(o))
}
