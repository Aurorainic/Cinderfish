# 第五十七波施工契约（接线 notify / withReplies 的消费方）

日期：2026-11-06
状态：已落地，证据见 `docs/HANDOVER.md` §12BV
上游参考：Misskey `4ff2edf`：

- `packages/backend/src/core/NoteCreateService.ts`（`notify` 决定是否给关注者发 `type: note`）
- `packages/backend/src/core/CacheService.ts`（`userFollowingsCache` 带 `withReplies`）
- `packages/frontend/src/components/MkNotification.vue`（`note` / `follow` / `receiveFollowRequest` 的文案）

## 1. 这一波补什么

第五十六波把 `followings.notify` 与 `with_replies` 落了库但没有消费方，本波把它接上：

1. **`notify = normal`** → 关注者在该账号每次发帖时收到 `type: "note"` 通知
2. **`with_replies`** → Streaming 的 `homeTimeline` 频道按每条关注设置过滤回复
3. 前端补齐 `note` / `follow` / `receiveFollowRequest` 三种通知的图标、文案与过滤项
   （此前 `follow` 类通知只有头像和名字，没有任何说明文字）

## 2. 后端

### 2.1 通知

`notifyFollowersOfNote(ctx, note, authorID)`（`internal/httpapi/streaming_publish.go`）：

- 查 `followings` 中 `followee_id = author AND notify IS NOT NULL` 且 `follower` 是本地的行
- 逐条 `CreateForLocal(recipient, "note", &authorID, &noteID, nil)`
- 创建成功则发 Streaming `notification` 事件
- **直发笔记（`specified`）跳过**：受众是显式指定的，通知会泄露其存在

挂载点：

- 本地发帖：`handleNotesCreate` 里紧接 `publishNoteCreated`
- 远端导入：`federation_wire.go` 的 `OnNoteCreated` 回调（读取已存笔记，再调用同一函数）

### 2.2 withReplies

`Server.LoadFollows`（`internal/httpapi/streaming_adapters.go`）改为
`SELECT followee_id, with_replies`，写进 `streaming.Follow{WithReplies}`。
`homeTimeline` 的 `replyAllowed` 与 `followsWithReplies` 早已存在，缺的就是这份数据。

## 3. 前端

| 文件 | 内容 |
|------|------|
| `web/src/api/types.ts` | `NotificationType` 增加 `note`、`follow`、`receiveFollowRequest` |
| `web/src/components/MkNotification.vue` | 三种类型的子图标与文案；`note` 显示笔记摘要并可点进 |
| `web/src/components/MkNotificationSettingWindow.vue` | 三种类型加入类型过滤列表与默认开关 |
| `web/src/i18n/locales/{zh-CN,ja-JP}.json` | 新增 `_notification.newNote` |

## 4. 有意的限制

- `notify` 只产生**站内通知**，不发邮件：邮件触发只有 mention/reply/follow/receiveFollowRequest
  四类（第四十八波），扩展邮件类型需要先加开关，留待以后。
- `withReplies` 只影响 Streaming 的 `homeTimeline`；`notes/timeline`（HTTP 拉取）仍按
  Misskey 的 SQL 语义返回关注者的全部笔记（上游的 `messages` 也不在 SQL 里过滤）。
- 通知不聚合：同一作者的多条笔记各占一条，与 Misskey 的 `note` 类型一致。

## 5. 写作用域

| 文件 | 动作 |
|------|------|
| `internal/httpapi/streaming_publish.go` | `notifyFollowersOfNote` |
| `internal/httpapi/notes.go` | 本地发帖后触发 |
| `internal/httpapi/federation_wire.go` | 远端导入后触发 |
| `internal/httpapi/streaming_adapters.go` | `LoadFollows` 读 `with_replies` |
| `internal/httpapi/follow_notify_test.go` | 两条用例 |
| 前端四个文件 | 见上 |

## 6. 验收口径

- `go vet ./...` 通过；`CINDERFISH_TEST_DATABASE_URL=… go test ./...` 全绿
- `TestNotifyFollowNotifications`：`notify=normal` 的关注者收到 `type: note` 且 `noteId` 正确；
  普通关注者什么也收不到
- `TestLoadFollowsCarriesWithReplies`：默认 true，`following/update` 改 false 后读回 false
- `vue-tsc --noEmit`、`check:port`（hardTotal 0、missingKeys 0）、`pnpm build` 全绿
