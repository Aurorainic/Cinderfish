# 第四十八波施工契约：邮件通知（2026-11-06）

第三十八波把 SMTP 与验证码做完了，但通知邮件一直是空的：`i/update` 不存
`emailNotificationTypes` / `receiveAnnouncementEmail`，设置页因此写着「没有服务端存储」。
本波补齐存储、发送与开关。

## 1. 存储

`user_profiles.settings` 新增两个键，走既有的 `UpdateProfileDetails` 合并路径（无需迁移）：

- `emailNotificationTypes`：字符串数组，取值 `mention` / `reply` / `follow` / `receiveFollowRequest`；
  其它值 400。空数组等于全部关闭。
- `receiveAnnouncementEmail`：布尔，订阅实例公告邮件。

`MeDetailed` 的这两个字段不再是常量（`false` / `[]`），改读 profile。

## 2. 发送

`internal/httpapi/notification_mail.go`：

- `maybeSendNotificationMail(ctx, recipientID, kind, actorID, noteID)`：读收件人 profile，要求
  **邮箱非空且已验证**、`kind` 在其 `emailNotificationTypes` 里；然后组装正文并在
  **后台 goroutine**（30 秒超时）投递，失败只记日志。
- 调用点：`notifyNoteCreated`（mention / reply / renote / quote 里只有前两个会真的发，因为类型
  白名单不含 renote/quote —— 由 profile 过滤兜住）与 `Server.createNotification`（follow /
  receiveFollowRequest）。
- `mailAnnouncement(title, text)`：`admin/announcements/create` 成功后触发，收件人由
  `account.AnnouncementMailRecipients` 从 settings 筛出（已验证 + 已订阅），2 分钟总超时。
  逐封发送，单封失败不影响后续。

邮件在请求路径之外发出，所以 SMTP 慢不会拖慢发帖或关注；反过来，邮件发不出去也不会让请求失败。

## 3. 前端

`pages/SettingsEmail.vue` 的「邮件通知」块从说明文字换成真实开关：

- 「接收实例公告邮件」；
- 四个类型开关（被提及、被回复、被关注、收到关注请求）。

每个开关切换即保存（`i/update`），失败在块内提示。邮箱未验证时块首显示警告——开关会保存，
但不会发出任何邮件。`ProfileUpdateRequest` 增加两个字段。

## 4. 有意的限制

1. 邮件正文是固定中文纯文本，没有 HTML 模板与多语言。
2. 不发「转发/引用/反应」邮件（与上游一致）。
3. 没有退订链接与 `List-Unsubscribe` 头，退订要在设置页操作。
4. 公告群发是逐封串行发送，实例用户多时较慢；个人实例可接受。
5. 没有发送失败重试队列，失败即丢（记日志）。
6. 未新增测试。

## 5. 验收

```
gofmt -l internal/ cmd/                 # 空
go vet ./...                            # exit 0
cd web && pnpm exec vue-tsc --noEmit    # exit 0
cd web && pnpm run check:port           # hardTotal 0
cd web && pnpm build                    # exit 0
```

台账不变（137/130/310）。未执行 `go test`、数据库测试与浏览器走查。
