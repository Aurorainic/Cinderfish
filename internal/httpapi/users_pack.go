package httpapi

import (
	"time"

	"github.com/Aurorainic/cinderfish/internal/account"
)

// packUserLite renders a user in the UserLite schema, the shape used by
// most list endpoints.
func packUserLite(u *account.User, host string) map[string]any {
	return map[string]any{
		"id":                          u.ID,
		"name":                        nil,
		"username":                    u.Username,
		"host":                        nil,
		"avatarUrl":                   nil,
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

// packUserDetailed renders a user in the UserDetailed schema.
func packUserDetailed(u *account.User, createdAt time.Time, isMe bool) map[string]any {
	packed := packUserLite(u, "")
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
	packed["description"] = nil
	packed["location"] = nil
	packed["birthday"] = nil
	packed["lang"] = nil
	packed["fields"] = []any{}
	packed["verifiedLinks"] = []any{}
	packed["followersCount"] = 0
	packed["followingCount"] = 0
	packed["notesCount"] = 0
	packed["pinnedNoteIds"] = []any{}
	packed["pinnedNotes"] = []any{}
	packed["pinnedPageId"] = nil
	packed["pinnedPage"] = nil
	packed["publicReactions"] = false
	packed["followersVisibility"] = "public"
	packed["followingVisibility"] = "public"
	packed["chatScope"] = "followers"
	packed["canChat"] = false
	packed["roles"] = []any{}
	packed["memo"] = nil
	return packed
}

// packMeDetailed renders the authenticated user in the MeDetailed schema.
//
// includeSecrets controls whether the private fields (email and friends)
// are returned; Misskey only sets it for native-token requests.
func packMeDetailed(u *account.User, createdAt time.Time, includeSecrets bool) map[string]any {
	packed := packUserDetailed(u, createdAt, true)
	packed["avatarId"] = nil
	packed["bannerId"] = nil
	packed["followedMessage"] = nil
	packed["isModerator"] = u.IsAdmin
	packed["isAdmin"] = u.IsAdmin
	packed["injectFeaturedNote"] = false
	packed["receiveAnnouncementEmail"] = false
	packed["alwaysMarkNsfw"] = false
	packed["autoSensitive"] = false
	packed["carefulBot"] = false
	packed["autoAcceptFollowed"] = false
	packed["followApprovalLocalSeconds"] = 0
	packed["followApprovalRemoteSeconds"] = 0
	packed["noCrawle"] = false
	packed["preventAiLearning"] = false
	packed["isExplorable"] = true
	packed["isDeleted"] = false
	packed["twoFactorBackupCodesStock"] = "none"
	packed["hideOnlineStatus"] = false
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
	packed["mutedInstances"] = []any{}
	packed["mutingNotificationTypes"] = []any{}
	packed["notificationRecieveConfig"] = map[string]any{}
	packed["emailNotificationTypes"] = []any{}
	packed["achievements"] = []any{}
	packed["loggedInDays"] = 0
	packed["policies"] = map[string]any{}

	if includeSecrets {
		packed["email"] = nil
		packed["emailVerified"] = false
		packed["securityKeysList"] = []any{}
	}
	return packed
}
