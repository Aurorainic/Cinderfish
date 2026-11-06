package federation

import (
	"context"
	"errors"
	"fmt"
	"html"
	"strings"
	"time"
)

// Inbox handles inbound ActivityPub activities.
//
// Every handler returns a human readable outcome the way Misskey's
// ApInboxService does ("ok" or "skip: <reason>"), which lands in the
// log and in ap_activities. The HTTP status is always 202 once the
// signature verified: a peer cannot fix a semantic problem by retrying,
// and it will retry forever if we answer with an error.
type Inbox struct {
	store    *Store
	renderer *Renderer
	client   *Client
	policy   *Policy
	// LocalUserByUsername resolves a username to a local user id.
	LocalUserByUsername func(ctx context.Context, username string) (id string, err error)
	// Hooks let the HTTP layer publish streaming events and create
	// notifications; they are nil when there is nothing to notify.
	OnNoteCreated func(ctx context.Context, noteID, userID string)
	OnFollow      func(ctx context.Context, followerID, followeeID string)
	OnUnfollow    func(ctx context.Context, followerID, followeeID string)
	OnReaction    func(ctx context.Context, noteID, userID, reaction string)
}

// NewInbox returns an inbox over the given dependencies.
func NewInbox(store *Store, renderer *Renderer, client *Client, policy *Policy) *Inbox {
	return &Inbox{store: store, renderer: renderer, client: client, policy: policy}
}

// Process verifies nothing and assumes the caller has authenticated the
// request; it resolves the actor and dispatches the activity.
func (i *Inbox) Process(ctx context.Context, activity Object) (string, error) {
	if activity == nil {
		return "", errors.New("federation: empty activity")
	}
	actorURI := activity.Ref("actor")
	if actorURI == "" {
		return "", errors.New("federation: activity has no actor")
	}
	actor, err := i.resolveActor(ctx, actorURI)
	if err != nil {
		return "", err
	}
	return i.perform(ctx, actor, activity)
}

// perform dispatches one activity, recursing into collections.
func (i *Inbox) perform(ctx context.Context, actor *RemoteUser, activity Object) (string, error) {
	if activity.IsType("Collection") || activity.IsType("OrderedCollection") {
		outcomes := make([]string, 0, 4)
		items := activity.Objects("items")
		if len(items) == 0 {
			items = activity.Objects("orderedItems")
		}
		for _, item := range items {
			out, err := i.perform(ctx, actor, item)
			if err != nil {
				out = "error: " + err.Error()
			}
			outcomes = append(outcomes, out)
		}
		return strings.Join(outcomes, "; "), nil
	}

	i.recordActivity(ctx, actor, activity)

	switch {
	case activity.IsType("Follow"):
		return i.follow(ctx, actor, activity)
	case activity.IsType("Accept"):
		return i.accept(ctx, actor, activity)
	case activity.IsType("Reject"):
		return i.reject(ctx, actor, activity)
	case activity.IsType("Undo"):
		return i.undo(ctx, actor, activity)
	case activity.IsType("Create"):
		return i.create(ctx, actor, activity)
	case activity.IsType("Delete"):
		return i.deleteActivity(ctx, actor, activity)
	case activity.IsType("Like"):
		return i.like(ctx, actor, activity)
	case activity.IsType("Announce"):
		return i.announce(ctx, actor, activity)
	case activity.IsType("Update"):
		return i.update(ctx, actor, activity)
	case activity.IsType("Block"):
		return i.block(ctx, actor, activity)
	}
	return fmt.Sprintf("unrecognized activity type: %s", activity.Type()), nil
}

// follow handles an inbound Follow targeting a local user.
func (i *Inbox) follow(ctx context.Context, actor *RemoteUser, activity Object) (string, error) {
	object := activity.Ref("object")
	if object == "" {
		return "skip: follow has no object", nil
	}
	target, err := i.localUserByURI(ctx, object)
	if err != nil {
		return "skip: followee not found", nil
	}

	// A locked account queues the request; an open one is accepted at
	// once and answered with an Accept activity.
	if target.locked {
		if err := i.store.FollowRequest(ctx, activity.Ref("id"), actor.ID, target.id); err != nil {
			return "", err
		}
		return "ok: follow request queued", nil
	}
	if err := i.store.Follow(ctx, activity.Ref("id"), actor.ID, target.id); err != nil {
		return "", err
	}
	if i.OnFollow != nil {
		i.OnFollow(ctx, actor.ID, target.id)
	}

	if err := i.client.PostActivity(ctx, actor.Inbox, i.renderer.RenderAccept(i.renderer.ActorRef(target.id), activity), target.id); err != nil {
		return "ok: followed but Accept delivery failed: " + err.Error(), nil
	}
	return "ok", nil
}

// accept handles an Accept of a Follow this instance sent. The embedded
// object is the original Follow, whose actor is our local user.
func (i *Inbox) accept(ctx context.Context, actor *RemoteUser, activity Object) (string, error) {
	obj := activity.Object("object")
	if obj == nil {
		return "skip: accept has no embedded object", nil
	}
	if !obj.IsType("Follow") {
		return "skip: accept is not for a Follow", nil
	}
	// The follower is us: the local user id is the last path segment
	// of the actor URI we sent.
	followerURI := obj.Ref("actor")
	followerID := lastPathSegment(followerURI)
	if followerID == "" {
		return "skip: accept has no follower", nil
	}
	if _, _, err := i.store.LocalKey(ctx, followerID); err != nil {
		return "skip: follower is not a local user", nil
	}
	if err := i.store.Follow(ctx, obj.Ref("id"), followerID, actor.ID); err != nil {
		return "", err
	}
	// The pending outgoing request is settled by this Accept; a missing row
	// means it was cancelled meanwhile and is not an error.
	if _, err := i.store.RemoveFollowRequest(ctx, followerID, actor.ID); err != nil && !errors.Is(err, ErrRemoteNotFound) {
		return "", err
	}
	if i.OnFollow != nil {
		i.OnFollow(ctx, followerID, actor.ID)
	}
	return "ok", nil
}

// reject handles a Reject of a Follow this instance sent.
func (i *Inbox) reject(ctx context.Context, actor *RemoteUser, activity Object) (string, error) {
	obj := activity.Object("object")
	if obj == nil {
		return "skip: reject has no embedded object", nil
	}
	followerID := lastPathSegment(obj.Ref("actor"))
	if followerID == "" {
		return "skip: reject has no follower", nil
	}
	if err := i.store.Unfollow(ctx, followerID, actor.ID); err != nil {
		return "", err
	}
	return "ok: follow rejected", nil
}

// undo handles Undo of Follow, Like, Announce, Block or Accept.
func (i *Inbox) undo(ctx context.Context, actor *RemoteUser, activity Object) (string, error) {
	obj := activity.Object("object")
	if obj == nil {
		return "skip: undo has no embedded object", nil
	}
	switch {
	case obj.IsType("Follow"):
		// The remote actor was following one of our users.
		target, err := i.localUserByURI(ctx, obj.Ref("object"))
		if err != nil {
			return "skip: followee not found", nil
		}
		if err := i.store.Unfollow(ctx, actor.ID, target.id); err != nil {
			return "", err
		}
		if i.OnUnfollow != nil {
			i.OnUnfollow(ctx, actor.ID, target.id)
		}
		return "ok: unfollowed", nil
	case obj.IsType("Like"):
		if err := i.store.Unreact(ctx, obj.Ref("object"), actor.ID); err != nil {
			return "", err
		}
		return "ok: reaction removed", nil
	case obj.IsType("Announce"):
		id := obj.Ref("id")
		if id == "" {
			return "skip: undo announce has no id", nil
		}
		if err := i.store.DeleteNoteByAPID(ctx, id, actor.ID); err != nil {
			return "skip: no such Announce", nil
		}
		return "ok: announce undone", nil
	case obj.IsType("Accept"):
		return "ok: undo accept ignored", nil
	}
	return fmt.Sprintf("skip: cannot undo %s", obj.Type()), nil
}

// create handles Create<Note>. The note is stored, the author notified
// if it replies or mentions a local user, and local followers see it.
func (i *Inbox) create(ctx context.Context, actor *RemoteUser, activity Object) (string, error) {
	note := activity.Object("object")
	if note == nil {
		return "skip: create has no object", nil
	}
	if !note.IsPost() {
		return fmt.Sprintf("skip: unknown object type %s", note.Type()), nil
	}
	noteID := note.Ref("id")
	if noteID == "" {
		return "skip: note has no id", nil
	}

	// The note must be authored by the actor that delivered it.
	if attr := note.Ref("attributedTo"); attr != "" && attr != actor.URI {
		return "skip: attributedTo does not match actor", nil
	}
	// And it must live on the actor's host, otherwise a peer could
	// inject notes attributed to someone else.
	host, err := HostOf(noteID)
	if err != nil || host != actor.Host {
		return "skip: note id host does not match actor", nil
	}

	exists, err := i.store.NoteExists(ctx, noteID)
	if err != nil {
		return "", err
	}
	if exists {
		return "skip: note exists", nil
	}

	data := NoteDataFromAP(note, actor, i.renderer)
	data.ActivityURI = activity.Ref("id")
	storedID, err := i.store.InsertRemoteNote(ctx, data)
	if err != nil {
		return "", err
	}
	if i.OnNoteCreated != nil {
		i.OnNoteCreated(ctx, storedID, actor.ID)
	}

	if err := i.notifyForNote(ctx, actor, note, storedID); err != nil {
		return "ok: note stored, notification failed: " + err.Error(), nil
	}
	return "ok", nil
}

// notifyForNote notifies local users the note replies to or mentions.
func (i *Inbox) notifyForNote(ctx context.Context, actor *RemoteUser, note Object, storedID string) error {
	var targets []string
	if replyTo := note.Ref("inReplyTo"); replyTo != "" {
		if u, err := i.localUserByAPID(ctx, replyTo); err == nil {
			targets = append(targets, u)
		}
	}
	for _, tag := range note.Objects("tag") {
		if tag.IsType("Mention") {
			if u, err := i.localUserByAPID(ctx, tag.Ref("href")); err == nil {
				targets = append(targets, u)
			}
		}
	}
	for _, userID := range targets {
		if _, err := i.store.Notification(ctx, userID, "reply", actor.ID, storedID); err != nil {
			return err
		}
	}
	return nil
}

// deleteActivity handles Delete of a note or of the actor itself.
func (i *Inbox) deleteActivity(ctx context.Context, actor *RemoteUser, activity Object) (string, error) {
	obj := activity.Object("object")
	objectID := ""
	formerType := ""
	if obj != nil {
		objectID = obj.Ref("id")
		formerType = obj.Type()
	}
	if objectID == "" {
		objectID = activity.Ref("object")
	}
	if objectID == "" {
		return "skip: delete has no object", nil
	}
	if obj != nil && obj.IsType("Tombstone") {
		formerType = obj.String("formerType")
	}
	if formerType == "" {
		if objectID == actor.URI {
			formerType = "Person"
		} else {
			formerType = "Note"
		}
	}

	switch formerType {
	case "Person", "Service", "Application":
		if objectID != actor.URI {
			return "skip: delete actor does not match signer", nil
		}
		if err := i.store.MarkUserDeleted(ctx, actor.ID); err != nil {
			return "", err
		}
		return "ok: actor deleted", nil
	}

	if err := i.store.DeleteNoteByAPID(ctx, objectID, actor.ID); err != nil {
		return "skip: note not found or not owned by actor", nil
	}
	return "ok: note deleted", nil
}

// like handles Like, which carries a Misskey reaction when present.
func (i *Inbox) like(ctx context.Context, actor *RemoteUser, activity Object) (string, error) {
	objectID := activity.Ref("object")
	if objectID == "" {
		return "skip: like has no object", nil
	}
	reaction := activity.String("_misskey_reaction")
	if reaction == "" {
		reaction = activity.String("content")
	}
	if reaction == "" {
		reaction = activity.String("name")
	}
	if reaction == "" {
		reaction = "⭐"
	}
	if err := i.store.React(ctx, objectID, actor.ID, reaction); err != nil {
		if errors.Is(err, ErrRemoteNotFound) {
			return "skip: target note not found", nil
		}
		return "", err
	}
	if i.OnReaction != nil {
		if owner, host, err := i.store.NoteOwnerByAPID(ctx, objectID); err == nil && host == nil {
			i.OnReaction(ctx, owner, actor.ID, reaction)
		}
	}
	return "ok", nil
}

// announce handles Announce (a renote).
func (i *Inbox) announce(ctx context.Context, actor *RemoteUser, activity Object) (string, error) {
	id := activity.Ref("id")
	if id == "" {
		return "skip: announce has no id", nil
	}
	targetID := activity.Ref("object")
	if targetID == "" {
		return "skip: announce has no object", nil
	}
	exists, err := i.store.NoteExists(ctx, id)
	if err != nil {
		return "", err
	}
	if exists {
		return "skip: announce exists", nil
	}
	data := RemoteNote{
		APID:        id,
		ActivityURI: id,
		UserID:      actor.ID,
		UserHost:    actor.Host,
		Visibility:  "public",
		QuoteAPID:   targetID,
		CreatedAt:   publishedOrNow(activity),
	}
	storedID, err := i.store.InsertRemoteNote(ctx, data)
	if err != nil {
		return "", err
	}
	if i.OnNoteCreated != nil {
		i.OnNoteCreated(ctx, storedID, actor.ID)
	}
	return "ok", nil
}

// update handles Update of an actor document.
func (i *Inbox) update(ctx context.Context, actor *RemoteUser, activity Object) (string, error) {
	obj := activity.Object("object")
	if obj == nil {
		return "skip: update has no embedded object", nil
	}
	if obj.IsPost() {
		return i.updateNote(ctx, actor, obj)
	}
	if !obj.IsActor() {
		return fmt.Sprintf("skip: cannot update %s", obj.Type()), nil
	}
	if _, err := i.upsertActor(ctx, actor.URI, obj); err != nil {
		return "", err
	}
	return "ok: actor updated", nil
}

// updateNote applies an edit a peer made to one of its own notes. The object
// must belong to the delivering actor and live on its host, the same checks
// create() applies, otherwise any peer could rewrite someone else's note.
func (i *Inbox) updateNote(ctx context.Context, actor *RemoteUser, note Object) (string, error) {
	apID := note.Ref("id")
	if apID == "" {
		return "skip: updated note has no id", nil
	}
	if attr := note.Ref("attributedTo"); attr != "" && attr != actor.URI {
		return "skip: attributedTo does not match actor", nil
	}
	host, err := HostOf(apID)
	if err != nil || host != actor.Host {
		return "skip: note id host does not match actor", nil
	}

	updatedAt := time.Now().UTC()
	if raw := note.String("updated"); raw != "" {
		if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
			updatedAt = parsed
		}
	}
	var text *string
	if note.Has("content") {
		value := html.UnescapeString(strings.TrimSpace(note.String("content")))
		text = &value
	}
	var cw *string
	if note.Has("summary") && note.String("summary") != "" {
		value := html.UnescapeString(strings.TrimSpace(note.String("summary")))
		cw = &value
	}

	if _, err := i.store.UpdateRemoteNote(ctx, apID, text, cw, note.Bool("sensitive"), updatedAt); err != nil {
		if errors.Is(err, ErrRemoteNotFound) {
			// The peer edited a note this instance never stored (or already
			// dropped); nothing to apply.
			return "skip: note unknown", nil
		}
		return "", err
	}
	return "ok: note updated", nil
}

// block handles an inbound Block. The remote peer blocked our user, so
// the relationship is removed in both directions.
func (i *Inbox) block(ctx context.Context, actor *RemoteUser, activity Object) (string, error) {
	target, err := i.localUserByURI(ctx, activity.Ref("object"))
	if err != nil {
		return "skip: blockee not found", nil
	}
	if err := i.store.RecordBlock(ctx, actor.ID, target.id); err != nil {
		return "", err
	}
	return "ok", nil
}

// resolveActor loads the remote actor, fetching it when unknown.
func (i *Inbox) resolveActor(ctx context.Context, uri string) (*RemoteUser, error) {
	if u, err := i.store.UserByURI(ctx, uri); err == nil {
		return u, nil
	}
	obj, err := i.client.FetchObject(ctx, uri, uri)
	if err != nil {
		return nil, fmt.Errorf("federation: resolve actor: %w", err)
	}
	return i.upsertActor(ctx, uri, obj)
}

// upsertActor stores an actor document.
func (i *Inbox) upsertActor(ctx context.Context, uri string, obj Object) (*RemoteUser, error) {
	host, err := HostOf(uri)
	if err != nil {
		return nil, err
	}
	username := obj.String("preferredUsername")
	if username == "" {
		return nil, fmt.Errorf("federation: actor %s has no preferredUsername", uri)
	}

	inbox := obj.Ref("inbox")
	shared := obj.Ref("endpoints")
	_ = shared
	if ep := obj.Object("endpoints"); ep != nil {
		shared = ep.Ref("sharedInbox")
	}
	if shared == "" {
		shared = obj.Ref("sharedInbox")
	}

	rem := RemoteUser{
		Username:     username,
		Host:         host,
		URI:          uri,
		Inbox:        inbox,
		SharedInbox:  shared,
		FollowersURI: obj.Ref("followers"),
		FeaturedURI:  obj.Ref("featured"),
		IsLocked:     obj.Bool("manuallyApprovesFollowers"),
		IsBot:        obj.IsType("Service") || obj.IsType("Application"),
		Name:         obj.String("name"),
		Description:  obj.String("_misskey_summary"),
	}
	if rem.Description == "" {
		rem.Description = obj.String("summary")
	}
	if icon := obj.Object("icon"); icon != nil {
		rem.AvatarURL = icon.Ref("url")
	}
	if image := obj.Object("image"); image != nil {
		rem.BannerURL = image.Ref("url")
	}
	if key := obj.Object("publicKey"); key != nil {
		rem.KeyID = key.Ref("id")
		rem.PublicKeyPEM = key.String("publicKeyPem")
	}
	return i.store.UpsertRemoteUser(ctx, rem)
}

// localActor is a resolved local user address.
type localActor struct {
	id     string
	locked bool
}

// localUserByURI resolves an actor URI to a local user.
func (i *Inbox) localUserByURI(ctx context.Context, uri string) (*localActor, error) {
	if uri == "" {
		return nil, ErrRemoteNotFound
	}
	if base, ok := strings.CutPrefix(uri, i.renderer.BaseURL+"/users/"); ok {
		id := lastPathSegment(base)
		if id != "" {
			return i.store.LocalUserByID(ctx, id)
		}
	}
	if rest, ok := strings.CutPrefix(uri, i.renderer.BaseURL+"/@"); ok {
		username := lastPathSegment(rest)
		if i.LocalUserByUsername != nil && username != "" {
			id, err := i.LocalUserByUsername(ctx, username)
			if err != nil {
				return nil, err
			}
			return i.store.LocalUserByID(ctx, id)
		}
	}
	return nil, ErrRemoteNotFound
}

// localUserByAPID resolves a note id or actor id to the local user that
// owns it.
func (i *Inbox) localUserByAPID(ctx context.Context, apID string) (string, error) {
	return i.store.LocalUserOfAPID(ctx, apID, i.renderer.BaseURL)
}

// recordActivity stores the inbound activity for audit.
func (i *Inbox) recordActivity(ctx context.Context, actor *RemoteUser, activity Object) {
	if err := i.store.RecordActivity(ctx, "inbound", actor.URI, activity); err != nil {
		logf("federation: record inbound activity: %v", err)
	}
}

// lastPathSegment returns the final path element of a URL.
func lastPathSegment(u string) string {
	u = strings.TrimSuffix(u, "/")
	if idx := strings.LastIndex(u, "/"); idx >= 0 {
		return u[idx+1:]
	}
	return u
}

// publishedOrNow reads an activity's published timestamp.
func publishedOrNow(o Object) time.Time {
	if s := o.String("published"); s != "" {
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			return t
		}
	}
	return time.Now()
}
