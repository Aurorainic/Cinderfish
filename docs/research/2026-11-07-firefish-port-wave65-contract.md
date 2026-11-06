# 第六十五波施工契约（远端 @提及：存储、渲染、投递与通知类型）

日期：2026-11-07
状态：已落地，证据见 `docs/HANDOVER.md` §12CD
上游参考：Misskey `4ff2edf`：

- `packages/backend/src/core/NoteCreateService.ts`（提及解析与投递目标）
- `packages/backend/src/core/activitypub/ApRendererService.ts`（`renderNote` 的 `tag: Mention`）
- `packages/backend/src/core/activitypub/ApInboxService.ts`（入站提及通知）

## 1. 发现的问题

写第十六波的远端提及用例时确认了三个缺陷：

1. **远端 @提及被丢弃**：`note.Service.Create` 的注释说远端提及「保留用于联邦寻址」，但代码
   直接 `continue`，`mentions[]` 只含本地用户 id——远端账号既不会收到笔记，也收不到通知。
2. **入站提及被记成 `reply`**：`Inbox.notifyForNote` 对所有目标（回复对象与 Mention 标签）
   一律写 `type = "reply"`，所以对端看到的提及通知类型错误。
3. 提及的 Actor URI 没有进入 `to`/`cc`，对端即使收到也无法判断这是发给谁的。

## 2. 实现

### 2.1 数据（迁移 v22）

```sql
ALTER TABLE notes ADD COLUMN IF NOT EXISTS remote_mentions JSONB NOT NULL DEFAULT '[]'::jsonb;
```

`mentions[]` 继续存本地用户 id（与 Misskey 列语义一致）；`remote_mentions` 存
`user@host`（小写、去重）的 acct 字符串，创建与编辑两条路径都写。

### 2.2 渲染

`renderNote` 读取 `RemoteMentions`，逐个解析成 Actor URI 后加入 `MentionRefs`，
于是 `to`/`cc` 与 `tag: Mention` 都带上目标；解析顺序：

1. 本地 `users` 行（`username_lower + host`，新增 `Store.UserByHandle`）——重复投递不再走 WebFinger
2. 否则 WebFinger + Actor 抓取（`Inbox.ResolveAcct`），结果落库

### 2.3 投递

`deliveryInboxes` 在关注者与本地提之外，对每个 `remote_mentions` 解析出 actor 的
shared inbox / inbox 并入队。解析失败只记日志，不影响其它目标。

### 2.4 入站通知类型

`Inbox.notifyForNote` 改为「每个目标一条、类型取最强理由」：Mention 优先于 reply，
与本地 `notifyNoteCreated` 的规则一致。

## 3. 写作用域

| 文件 | 动作 |
|------|------|
| `internal/db/db.go` | 迁移 v22 |
| `internal/note/service.go`、`edit.go` | `RemoteMentions` 字段、创建/编辑写入 |
| `internal/federation/store.go` | `UserByHandle` |
| `internal/httpapi/activitypub.go` | 渲染时解析远端提及 |
| `internal/httpapi/federation_wire.go` | 投递目标追加远端提及的 inbox |
| `internal/federation/inbox.go` | 入站通知类型修正 |
| `internal/httpapi/mention_federation_test.go` | 双实例端到端用例 |

## 4. 有意的限制

- 远端提及在**渲染/投递时**解析，不在发帖请求里做网络调用：发帖延迟不受对端可用性影响。
- 解析失败只记日志：笔记照常发布，目标这一次收不到。
- `remote_mentions` 不做 URI 缓存清理；actor 行会随 `users` 表正常留存。
- 入站提及仍要求对端把 Mention 标签写对；纯文本 `@user@host` 没有标签时不会通知
  （与 Misskey 一致）。

## 5. 验收口径

- `go vet ./...` 通过；`CINDERFISH_TEST_DATABASE_URL=… go test ./...` 全绿
- `TestRemoteMentionIsDeliveredAndNotifies`：beta 的 bob 发一条 @alpha 的 alice 的笔记，
  投递 worker 把 Create 送到 alpha，alpha 落库远端副本，alice 收到 `type = "mention"` 的通知，
  通知的发起者是 betabob
