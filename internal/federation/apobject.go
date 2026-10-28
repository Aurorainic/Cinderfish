package federation

import (
	"encoding/json"
	"fmt"
)

// Public is the special ActivityStreams collection id that marks a post
// as public. Misskey emits exactly this URI in to/cc.
const Public = "https://www.w3.org/ns/activitystreams#Public"

// Content types peers use for ActivityPub payloads. The inbox accepts
// both; the server renders activity+json by default and ld+json when the
// client asked for it.
const (
	ContentTypeActivity = "application/activity+json"
	ContentTypeLD       = `application/ld+json; profile="https://www.w3.org/ns/activitystreams"`
)

// Object is a decoded ActivityPub object. ActivityPub objects are open
// maps, so rather than modelling every type we keep the raw JSON and
// provide typed accessors for the fields the implementation needs.
type Object map[string]any

// Type returns the object's @type as a single string. ActivityPub allows
// a list of types; the first one is what every peer keys on.
func (o Object) Type() string {
	return o.String("type")
}

// IsType reports whether the object carries t among its types.
func (o Object) IsType(t string) bool {
	switch v := o["type"].(type) {
	case string:
		return v == t
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok && s == t {
				return true
			}
		}
	}
	return false
}

// String returns a string field, or "" when absent or not a string.
func (o Object) String(key string) string {
	if s, ok := o[key].(string); ok {
		return s
	}
	return ""
}

// Bool returns a boolean field, or false when absent or not a boolean.
func (o Object) Bool(key string) bool {
	if b, ok := o[key].(bool); ok {
		return b
	}
	return false
}

// Has reports whether the key is present with a non-nil value.
func (o Object) Has(key string) bool {
	v, ok := o[key]
	return ok && v != nil
}

// Object returns a nested object field.
func (o Object) Object(key string) Object {
	if m, ok := o[key].(map[string]any); ok {
		return Object(m)
	}
	return nil
}

// Ref returns the id of a reference field. ActivityPub lets a field be
// either a bare id string or an embedded object; both forms appear.
func (o Object) Ref(key string) string {
	switch v := o[key].(type) {
	case string:
		return v
	case map[string]any:
		if s, ok := v["id"].(string); ok {
			return s
		}
	}
	return ""
}

// Refs returns a reference field as a list, accepting a string, a single
// object or an array of either.
func (o Object) Refs(key string) []string {
	v, ok := o[key]
	if !ok || v == nil {
		return nil
	}
	switch t := v.(type) {
	case string:
		if t == "" {
			return nil
		}
		return []string{t}
	case []string:
		if len(t) == 0 {
			return nil
		}
		out := make([]string, len(t))
		copy(out, t)
		return out
	case map[string]any:
		if s, ok := t["id"].(string); ok && s != "" {
			return []string{s}
		}
		return nil
	case []any:
		out := make([]string, 0, len(t))
		for _, item := range t {
			switch it := item.(type) {
			case string:
				if it != "" {
					out = append(out, it)
				}
			case map[string]any:
				if s, ok := it["id"].(string); ok && s != "" {
					out = append(out, s)
				}
			}
		}
		return out
	}
	return nil
}

// Objects returns a field as a list of embedded objects, skipping bare
// id strings, which carry no fields to read.
func (o Object) Objects(key string) []Object {
	v, ok := o[key]
	if !ok || v == nil {
		return nil
	}
	switch t := v.(type) {
	case map[string]any:
		return []Object{t}
	case []any:
		out := make([]Object, 0, len(t))
		for _, item := range t {
			if m, ok := item.(map[string]any); ok {
				out = append(out, m)
			}
		}
		return out
	}
	return nil
}

// DecodeObject parses raw JSON into an Object.
func DecodeObject(b []byte) (Object, error) {
	var o Object
	if err := json.Unmarshal(b, &o); err != nil {
		return nil, fmt.Errorf("federation: decode activity: %w", err)
	}
	if o == nil {
		return nil, fmt.Errorf("federation: activity is not a JSON object")
	}
	return o, nil
}

// activityTypes lists the activity and object types the implementation
// understands, used to tell a stray object from an activity.
var activityTypes = map[string]bool{
	"Create": true, "Delete": true, "Update": true, "Follow": true,
	"Accept": true, "Reject": true, "Add": true, "Remove": true,
	"Announce": true, "Like": true, "Undo": true, "Block": true,
	"Flag": true, "Move": true,
}

// IsActivity reports whether o is one of the activity types MilnFish
// dispatches on.
func (o Object) IsActivity() bool {
	for _, t := range []string{"Collection", "OrderedCollection"} {
		if o.IsType(t) {
			return true
		}
	}
	for t := range activityTypes {
		if o.IsType(t) {
			return true
		}
	}
	return false
}

// postTypes are the object types MilnFish stores as notes.
var postTypes = []string{"Note", "Question", "Article", "Page"}

// IsPost reports whether o is a note-like object.
func (o Object) IsPost() bool {
	for _, t := range postTypes {
		if o.IsType(t) {
			return true
		}
	}
	return false
}

// actorTypes are the object types MilnFish stores as remote users.
var actorTypes = []string{"Person", "Service", "Application", "Group", "Organization"}

// IsActor reports whether o is an actor document.
func (o Object) IsActor() bool {
	for _, t := range actorTypes {
		if o.IsType(t) {
			return true
		}
	}
	return false
}

// Context is the @context every rendered document carries. It is the
// Misskey context: plain ActivityStreams and security/v1 IRIs plus one
// inline extension object defining the Misskey, Mastodon and schema.org
// terms this project emits.
var Context = []any{
	"https://www.w3.org/ns/activitystreams",
	"https://w3id.org/security/v1",
	map[string]any{
		"Key":                                   "sec:Key",
		"manuallyApprovesFollowers":             "as:manuallyApprovesFollowers",
		"sensitive":                             "as:sensitive",
		"Hashtag":                               "as:Hashtag",
		"quoteUrl":                              "as:quoteUrl",
		"toot":                                  "http://joinmastodon.org/ns#",
		"Emoji":                                 "toot:Emoji",
		"featured":                              "toot:featured",
		"discoverable":                          "toot:discoverable",
		"schema":                                "http://schema.org#",
		"PropertyValue":                         "schema:PropertyValue",
		"value":                                 "schema:value",
		"misskey":                               "https://misskey-hub.net/ns#",
		"_misskey_content":                      "misskey:_misskey_content",
		"_misskey_quote":                        "misskey:_misskey_quote",
		"_misskey_reaction":                     "misskey:_misskey_reaction",
		"_misskey_votes":                        "misskey:_misskey_votes",
		"_misskey_summary":                      "misskey:_misskey_summary",
		"_misskey_followedMessage":              "misskey:_misskey_followedMessage",
		"_misskey_requireSigninToViewContents":  "misskey:_misskey_requireSigninToViewContents",
		"_misskey_makeNotesFollowersOnlyBefore": "misskey:_misskey_makeNotesFollowersOnlyBefore",
		"_misskey_makeNotesHiddenBefore":        "misskey:_misskey_makeNotesHiddenBefore",
		"_misskey_license":                      "misskey:_misskey_license",
		"freeText": map[string]any{
			"@id":   "misskey:freeText",
			"@type": "schema:text",
		},
		"isCat": "misskey:isCat",
		"vcard": "http://www.w3.org/2006/vcard/ns#",
	},
}

// withContext wraps a parsed body as an Object that carries @context.
func withContext(o Object) Object {
	out := Object{"@context": Context}
	for k, v := range o {
		out[k] = v
	}
	return out
}
