package httpapi

import (
	"context"
	"fmt"
	"log"
	"slices"
	"time"

	"github.com/Aurorainic/cinderfish/internal/mail"
)

// notificationMailTimeout bounds one outbound message. Mail is sent off the
// request path, so a slow SMTP server delays nothing the user waits for.
const notificationMailTimeout = 30 * time.Second

// maybeSendNotificationMail mails a notification to the recipient when their
// profile asks for that reason and the address is verified. It never blocks
// the request and never fails it: a broken mailbox is the recipient's problem,
// not the actor's.
//
// Only the reasons profile.EmailNotificationTypes accepts are mailed at all;
// the in-app list is wider (reactions, renotes and quotes stay in-app).
func (s *Server) maybeSendNotificationMail(ctx context.Context, recipientID, kind, actorID, noteID string) {
	if s.mail == nil || s.accounts == nil {
		return
	}
	profile, err := s.accounts.GetProfile(ctx, recipientID)
	if err != nil {
		return
	}
	if profile.Email == "" || !profile.EmailVerified {
		return
	}
	if !slices.Contains(profile.EmailNotificationTypes, kind) {
		return
	}

	actor := ""
	if actorID != "" {
		if u, err := s.accounts.GetByID(ctx, actorID); err == nil && u != nil {
			actor = "@" + u.Username
			if u.Host != nil && *u.Host != "" {
				actor += "@" + *u.Host
			}
		}
	}

	subject, body := notificationMail(s.baseURL(), kind, actor, noteID)
	to := profile.Email
	go func() {
		sendCtx, cancel := context.WithTimeout(context.Background(), notificationMailTimeout)
		defer cancel()
		if err := s.mail.Send(sendCtx, mail.Message{To: to, Subject: subject, Body: body}); err != nil {
			log.Printf("mail: notification %s to %s: %v", kind, to, err)
		}
	}()
}

// notificationMail renders the subject and body for one reason.
func notificationMail(base, kind, actor, noteID string) (string, string) {
	if actor == "" {
		actor = "有人"
	}
	noteLine := ""
	if noteID != "" {
		noteLine = fmt.Sprintf("\n\n查看：%s/notes/%s", base, noteID)
	}
	switch kind {
	case "reply":
		return actor + " 回复了你的帖子",
			fmt.Sprintf("%s 回复了你的帖子。%s\n\n你可以在账号设置里关闭这类邮件通知。", actor, noteLine)
	case "mention":
		return actor + " 在帖子中提到了你",
			fmt.Sprintf("%s 在帖子中提到了你。%s\n\n你可以在账号设置里关闭这类邮件通知。", actor, noteLine)
	case "follow":
		return actor + " 关注了你",
			fmt.Sprintf("%s 关注了你。\n\n你可以在账号设置里关闭这类邮件通知。", actor)
	case "receiveFollowRequest":
		return actor + " 请求关注你",
			fmt.Sprintf("%s 请求关注你，需要你手动批准。\n\n%s/admin 或通知页可以处理。\n\n你可以在账号设置里关闭这类邮件通知。", actor, base)
	default:
		return "Cinderfish 通知", fmt.Sprintf("你有一条新的通知（%s）。", kind)
	}
}
