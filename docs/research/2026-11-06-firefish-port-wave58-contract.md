# 第五十八波施工契约（入站反应：EmojiReact 与本地笔记解析）

日期：2026-11-06
状态：已落地，证据见 `docs/HANDOVER.md` §12BW
上游参考：Misskey `4ff2edf`：

- `packages/backend/src/core/activitypub/type.ts`（`isLike` 同时接受 `Like` / `EmojiReaction` / `EmojiReact`）
- `packages/backend/src/core/activitypub/ApInboxService.ts`（`like()` 读
  `_misskey_reaction ?? content ?? name`）

## 1. 发现的问题

写本波用例时发现两个既有缺陷：

1. **远端用户对本地笔记的反应从未落库**：`federation.Store.React` 只按
   `notes.ap_id = $1 OR notes.uri = $1` 找目标笔记，而本地笔记创建时不写这两列
   （`note.Service.Create` 的 INSERT 里没有 ap_id/uri）。入站 `Like` 因此总是
   `skip: target note not found`。
2. **只认 `Like`**：Firefish / Misskey 对自定义表情发送 `EmojiReact`，对其它反应发送带
   `content` 的 `Like`。收到的 `EmojiReact` 走到默认分支，被当成未知活动丢弃。

## 2. 修法

### 2.1 笔记解析

`Store.NoteIDByAPIDOrURL(ctx, apID, baseURL)`：

1. 若 `apID` 以 `<baseURL>/notes/` 开头，取最后一段作为本地 id，直接查 `notes.id`
2. 否则回退到既有的 `NoteIDByAPID`（远端笔记的 `ap_id` / `uri`）

### 2.2 反应写入

- 新增 `ReactNoteID(noteID, userID, reaction)` 与 `UnreactNoteID(noteID, userID)`，
  按本地笔记 id 操作；`Store.React` / `Store.Unreact` 已无调用方，删除
- `Inbox.like` 改为「先解析 note id，再写反应」，反应来源仍是
  `_misskey_reaction` → `content` → `name` → `⭐`
- `OnReaction` 的所有者查询改用 `Store.NoteOwner(noteID)`（按 id），不再依赖 ap_id

### 2.3 活动类型

`perform` 与 `undo` 的分支从 `Like` 扩展为
`Like | EmojiReact | EmojiReaction`。

## 3. 写作用域

| 文件 | 动作 |
|------|------|
| `internal/federation/store.go` | 新增 `NoteIDByAPIDOrURL`、`ReactNoteID`、`UnreactNoteID`；删除 `React`、`Unreact` |
| `internal/federation/inbox.go` | `like` / `undo` 走新路径，接收三种反应活动 |
| `internal/httpapi/reaction_federation_test.go` | 新用例 |
| 台账 §6 | 入站活动清单更新 |

## 4. 有意的限制

- 出站反应仍是 `Like` + `content`（Misskey 的 `renderLike` 也是这么发的），没有改发
  `EmojiReact`；两种形式对端都接受。
- 不处理 `EmojiReact` 的 `tag` 里的自定义表情定义（`extractEmojis`）：Cinderfish 还没有
  远端 emoji 表，反应文本原样存库。
- `Flag`、`Question`（投票）仍未处理。

## 5. 验收口径

- `go vet ./...` 通过；`CINDERFISH_TEST_DATABASE_URL=… go test ./...` 全绿
- `TestInboundEmojiReact`：双实例、真实签名。beta 的账号给 alpha 的本地笔记发
  `EmojiReact`，`notes/reactions` 能看到 🍮；再发 `Undo(EmojiReact)` 后列表为空
- 该用例同时覆盖「本地笔记 URL 解析」这条修复前必然失败的路径
