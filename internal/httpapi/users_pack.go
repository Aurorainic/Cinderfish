package httpapi

import (
	"time"

	"github.com/Aurorainic/cinderfish/internal/account"
)

// packUserLite renders a user in the UserLite schema, the shape used by
// most list endpoints. prof may be nil, which leaves name and avatarUrl
// unset as before the profile packing existed.
func packUserLite(u *account.User, host string, prof *account.Profile) map[string]any {
	var name, avatarURL, hostField any
	if prof != nil {
		if prof.Name != "" {
			name = prof.Name
		}
		if prof.AvatarURL != "" {
			avatarURL = prof.AvatarURL
		}
	}
	if u.Host != nil && *u.Host != "" {
		hostField = *u.Host
	}
	return map[string]any{
		"id":                          u.ID,
		"name":                        name,
		"username":                    u.Username,
		"host":                        hostField,
		"avatarUrl":                   avatarURL,
		"avatarBlurhash":              nil,
		"avatarDecorations":           []any{},
		"isBot":                       false,
		"isCat":                       false,
		"requireSigninToViewContents": false,
		"emojis":                      []any{},
		"onlineStatus":                "unknown",
		"badgeRoles":                  []any{},
		"instance":                    nil,
	}
}

// packUserDetailed renders a user in the UserDetailed schema. prof
// supplies the description; the caller loads it so the packer stays free
// of database access.
func packUserDetailed(u *account.User, createdAt time.Time, isMe bool, prof *account.Profile) map[string]any {
	packed := packUserLite(u, "", prof)
	packed["url"] = ""
	packed["uri"] = ""
	packed["movedTo"] = nil
	packed["alsoKnownAs"] = nil
	packed["createdAt"] = createdAt.UTC().Format(time.RFC3339Nano)
	packed["updatedAt"] = nil
	packed["lastFetchedAt"] = nil
	packed["bannerUrl"] = nil
	packed["bannerBlurhash"] = nil
	packed["isLocked"] = false
	packed["isModerator"] = u.IsAdmin
	packed["isAdmin"] = u.IsAdmin
	packed["isSilenced"] = false
	packed["isSuspended"] = u.IsSuspended
	// The signin dialog reads twoFactorEnabled from users/show before it
	// decides to ask for a TOTP code, so the detailed pack carries it for
	// every user, as Firefish does.
	packed["twoFactorEnabled"] = prof != nil && prof.TwoFactorEnabled
	packed["usePasswordLessLogin"] = prof != nil && prof.UsePasswordLessLogin
	packed["securityKeys"] = false
	var description any
	if prof != nil && prof.Description != "" {
		description = prof.Description
	}
	packed["description"] = description
	packed["location"] = nil
	packed["birthday"] = nil
	packed["lang"] = nil
	packed["fields"] = []any{}
	packed["publicReactions"] = false
	if prof != nil {
		packed["location"] = nilIfEmpty(prof.Location)
		packed["birthday"] = nilIfEmpty(prof.Birthday)
		packed["lang"] = nilIfEmpty(prof.Lang)
		fields := make([]any, 0, len(prof.Fields))
		for _, f := range prof.Fields {
			fields = append(fields, map[string]any{"name": f.Name, "value": f.Value})
		}
		packed["fields"] = fields
		packed["publicReactions"] = prof.PublicReactions
	}
	packed["verifiedLinks"] = []any{}
	packed["followersCount"] = 0
	packed["followingCount"] = 0
	packed["notesCount"] = 0
	packed["pinnedNoteIds"] = []any{}
	packed["pinnedNotes"] = []any{}
	packed["pinnedPageId"] = nil
	packed["pinnedPage"] = nil
	packed["followersVisibility"] = visibilityOrDefault(prof, true)
	packed["followingVisibility"] = visibilityOrDefault(prof, false)
	packed["chatScope"] = "followers"
	packed["canChat"] = false
	packed["roles"] = []any{}
	packed["memo"] = nil
	// The four relationship flags of UserDetailed. They default to false
	// and are filled in by callers that know the viewer.
	packed["isFollowing"] = false
	packed["isFollowed"] = false
	packed["hasPendingFollowRequestFromYou"] = false
	packed["hasPendingFollowRequestToYou"] = false
	return packed
}

// visibilityOrDefault reads one follow-list switch off the profile. The
// column defaults to "public", and a user without a profile row reads as the
// zero value, so an empty string means "public".
func visibilityOrDefault(prof *account.Profile, followers bool) string {
	if prof == nil {
		return "public"
	}
	v := prof.FollowingVisibility
	if followers {
		v = prof.FollowersVisibility
	}
	if v == "" {
		return "public"
	}
	return v
}

// packMeDetailed renders the authenticated user in the MeDetailed schema.
//
// includeSecrets controls whether the private fields (email and friends)
// are returned; Misskey only sets it for native-token requests.
func packMeDetailed(u *account.User, createdAt time.Time, includeSecrets bool, prof *account.Profile) map[string]any {
	packed := packUserDetailed(u, createdAt, true, prof)
	packed["avatarId"] = nil
	packed["bannerId"] = nil
	packed["followedMessage"] = nil
	packed["isModerator"] = u.IsAdmin
	packed["isAdmin"] = u.IsAdmin
	packed["injectFeaturedNote"] = prof != nil && prof.InjectFeaturedNote
	packed["receiveAnnouncementEmail"] = false
	packed["alwaysMarkNsfw"] = prof != nil && prof.AlwaysMarkNsfw
	packed["autoSensitive"] = prof != nil && prof.AutoSensitive
	packed["carefulBot"] = prof != nil && prof.CarefulBot
	packed["autoAcceptFollowed"] = prof != nil && prof.AutoAcceptFollowed
	packed["followApprovalLocalSeconds"] = 0
	packed["followApprovalRemoteSeconds"] = 0
	packed["noCrawle"] = prof != nil && prof.NoCrawle
	packed["preventAiLearning"] = prof != nil && prof.PreventAiLearning
	packed["isExplorable"] = prof == nil || prof.IsExplorable
	packed["isDeleted"] = false
	packed["twoFactorBackupCodesStock"] = "none"
	packed["hideOnlineStatus"] = prof != nil && prof.HideOnlineStatus
	packed["hasUnreadSpecifiedNotes"] = false
	packed["hasUnreadMentions"] = false
	packed["hasUnreadChatMessages"] = false
	packed["hasUnreadAnnouncement"] = false
	packed["unreadAnnouncements"] = []any{}
	packed["hasUnreadAntenna"] = false
	packed["hasUnreadChannel"] = false
	packed["hasUnreadNotification"] = false
	packed["hasPendingReceivedFollowRequest"] = false
	packed["unreadNotificationsCount"] = 0
	packed["mutedWords"] = []any{}
	packed["hardMutedWords"] = []any{}
	if prof != nil && prof.MutedInstances != nil {
		packed["mutedInstances"] = prof.MutedInstances
	} else {
		packed["mutedInstances"] = []any{}
	}
	if prof != nil && prof.MutedNotificationTypes != nil {
		packed["mutingNotificationTypes"] = prof.MutedNotificationTypes
	} else {
		packed["mutingNotificationTypes"] = []any{}
	}
	packed["notificationRecieveConfig"] = map[string]any{}
	packed["emailNotificationTypes"] = []any{}
	packed["achievements"] = []any{}
	packed["loggedInDays"] = 0
	packed["policies"] = map[string]any{}

	if includeSecrets {
		if prof != nil && prof.Email != "" {
			packed["email"] = prof.Email
		} else {
			packed["email"] = nil
		}
		packed["emailVerified"] = prof != nil && prof.EmailVerified
		packed["securityKeysList"] = []any{}
	}
	return packed
}
