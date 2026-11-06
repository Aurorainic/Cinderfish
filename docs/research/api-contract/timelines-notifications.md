# API 对接契约：timelines-notifications

本文覆盖 Cinderfish 后端的**时间线、通知与趋势**命名空间，供重新编写 Firefish 风格前端的开发者使用。
每一条结论都以本仓库（分支 `frontend-ui-redesign`）的**实际 handler 与 SQL** 为准，而不是 Misskey 文档；
与 Misskey 上游的差异单独列在每节末尾，上游结论来自 `tmp/refcheck/misskey` 的同名端点源码。

- 路由注册：`internal/httpapi/server.go:172`–`185`、`internal/httpapi/server.go:221`
- Handler：`internal/httpapi/timelines.go`、`notifications.go`、`hashtags.go`、`streaming.go`、`streaming_pack.go`、`streaming_publish.go`、`streaming_adapters.go`
- 服务层：`internal/note/timeline.go`、`internal/note/reaction.go`（通知服务在里面）、`internal/note/hashtag_trend.go`、`internal/streaming/`
- 上游覆盖台账：`docs/research/2026-11-04-misskey-api-coverage.md`

本命名空间的 8 个 HTTP 端点**全部是 `POST`**，路径前缀 `/api`；另有 1 个 WebSocket 端点 `GET /streaming`。
没有 GET 别名（`hashtags/trend` 在上游支持 GET，本实现不支持）。用错方法会落到兜底路由，返回 `404 NO_SUCH_ENDPOINT`（`internal/httpapi/server.go:223`、`:327`）——已用最小复现验证：在本项目的路由结构下，方法不匹配时请求由 `/` 兜底处理，`ServeMux` 不会返回 405。

---

## 端点总览

| 端点 | 方法 | 认证 | 核心参数 | 成功响应 | 实现位置 |
|------|------|------|----------|----------|----------|
| `/api/notes/timeline` | POST | **必需** | `limit`、`sinceId`、`untilId`、`sinceDate`、`untilDate`、`withFiles`、`withRenotes`、`withReplies`、`includeMyRenotes`、`includeRenotedMyNotes`、`includeLocalRenotes`、（`allowPartial` 被忽略） | `Note[]`（detail 形式） | `internal/httpapi/timelines.go:42` |
| `/api/notes/local-timeline` | POST | 匿名可 | 同上 | `Note[]` | `internal/httpapi/timelines.go:26` |
| `/api/notes/hybrid-timeline` | POST | **必需** | 同上 | `Note[]` | `internal/httpapi/timelines.go:31` |
| `/api/notes/global-timeline` | POST | 匿名可 | 同上 | `Note[]` | `internal/httpapi/timelines.go:36` |
| `/api/notifications` | POST | **必需** | `limit`、`sinceId`、`untilId`、`markAsRead`、`includeTypes`、`excludeTypes` | `Notification[]` | `internal/httpapi/notifications.go:41` |
| `/api/i/notifications` | POST | **必需** | 同 `notifications`（**同一 handler 的别名**） | `Notification[]` | `internal/httpapi/server.go:184` |
| `/api/notifications/mark-all-as-read` | POST | **必需** | 无 | `{}` | `internal/httpapi/notifications.go:106` |
| `/api/hashtags/trend` | POST | 匿名可 | 无 | `HashtagTrend[]` | `internal/httpapi/hashtags.go:13` |
| `/streaming` | GET（WebSocket Upgrade） | 可选（匿名可） | 无 body；令牌走 `?i=` 或 `Authorization` | WebSocket 帧 | `internal/httpapi/streaming.go:45` |

**没有任何端点要求管理员权限。** 本命名空间只有三种认证档位：需要登录、匿名可用、弱认证（时间线里"坏令牌当作匿名"）。

---

## 通用约定

### 1. 认证

`Server.authenticate`（`internal/httpapi/auth.go:30`）按顺序取令牌：

1. 请求体的 `i` 字段（必须是 JSON 字符串）；
2. `Authorization: Bearer <token>` 头（大小写不敏感的 `bearer` 前缀，值会 `TrimSpace`）。

**两者同时存在时以请求体的 `i` 为准。** 没有令牌 → 返回 `(nil, nil)`，即匿名。

- 令牌无法解析（`TokenService.Resolve` 查不到，见 `internal/account/tokens.go:88`）→ 端点调用 `writeAuthenticationError`：**HTTP 401**，`code = AUTHENTICATION_FAILED`，并带
  `WWW-Authenticate: Bearer realm="Misskey"`（`internal/httpapi/server.go:304`）。
  注意 `writeAuthenticationError` 先写的详细值（含 `error="invalid_token"`）随后被 `writeAPIError` 里的 `Set` 覆盖（`internal/httpapi/server.go:284`），所以**最终头是简版**；带 `error_description` 的详细头只出现在 Streaming 握手的 401 上。
- 账户被停用（`users.is_suspended`）即使令牌有效也按认证失败处理（`internal/httpapi/auth.go:52`）。
- **只校验令牌有效性，不校验 scope。** 本命名空间所有 HTTP 端点都无视 `access_tokens.scopes`（Misskey 的 `read:account` / `read:notifications` / `write:notifications` 在这里都不检查）。只有 Streaming 握手检查 `read:account`（见 `GET /streaming` 一节）。
- **弱认证**：四个时间线 handler 都写 `if info, err := s.authenticate(...); err == nil && info != nil`（`internal/httpapi/timelines.go:66`），即**忽略认证错误**。
  所以在 `local-timeline` / `global-timeline` 上传一个坏令牌不会得到 401，而是被当成匿名访问并返回 200。
  `notes/timeline` 与 `notes/hybrid-timeline` 因为要求登录，坏令牌仍然得到 401（`internal/httpapi/timelines.go:69`）。

### 2. 请求体解码

- `decodeOptionalJSON`（`internal/httpapi/ping.go:23`）：先读 ≤8 MiB 的 body 再放回，空 body 合法（等于全部缺省）。
- JSON 非法 / 字段类型不符（例如 `limit` 传字符串）→ **400 `INVALID_PARAM`**，`info` 为 `{"reason":"body is not valid JSON"}`。
- 未知字段被静默忽略（标准 `encoding/json` 行为）。所以 `sinceDate`、`untilDate` 之外的 Misskey 参数（如通知的 `sinceDate`）不会报错，只是**没有任何效果**。

### 3. 响应编码

`writeJSON`（`internal/httpapi/server.go:235`）：`Content-Type: application/json; charset=utf-8`，`SetEscapeHTML(false)`，末尾带一个 `\n`。

- 成功状态统一 **200**（`mark-all-as-read` 也是 200 + `{}`，没有 204）。
- 时间字段是 **RFC3339Nano 字符串**（`packNote`/`packNotification` 用 `time.RFC3339Nano`，UTC，例如 `2026-11-04T10:20:30.123456789Z`），只有 `DriveFile.createdAt` 例外，见共享对象一节。
- ID 是 18 字符 base36 时间序 id（10 位毫秒时间戳 + 8 位随机，`internal/idgen/idgen.go`），**字典序即时间序**，因此可以当游标用。

### 4. 错误信封

统一为（`writeAPIError`，`internal/httpapi/server.go:269`）：

```json
{"error":{"message":"...","code":"...","id":"<uuid>","kind":"client|server|permission"}}
```

- `kind` 恒有；`info` 只在部分 `INVALID_PARAM` 里出现。
- 401 额外带 `WWW-Authenticate` 头；其余状态没有额外头。
- 本命名空间实际会出现的错误：

| HTTP | code | 触发点 |
|------|------|--------|
| 400 | `INVALID_PARAM` | body 非法 JSON、字段类型不符 |
| 401 | `AUTHENTICATION_FAILED` | 需要登录但无令牌/坏令牌/账户被停用 |
| 404 | `NO_SUCH_ENDPOINT` | 路径不存在，或方法不是 POST |
| 500 | `INTERNAL_ERROR` | 时间线/通知查询失败 |
| 503 | `INTERNAL_ERROR` | 服务依赖未装配（无数据库启动的实例、或未 `AttachStreaming`） |

- 500 与 503 的 `code` 都是 `INTERNAL_ERROR`，只能靠 status 区分；503 的 message 是 `"Not available."`，500 的分别是 `"Could not load the timeline."` / `"Could not load the notifications."` / `"Could not mark the notifications as read."` / `"Internal error."`。

### 5. 分页游标

时间线与通知共用同一套 Misskey 语义，但**两者的排序实现不同**，前端必须分别处理。

**参数语义**

| 参数 | 类型 | 缺省 | 说明 |
|------|------|------|------|
| `limit` | int | 10 | `≤0` → 10；`>100` → **钳到 100**（不报错）。Misskey 是 `minimum:1, maximum:100` 校验、越界报 `INVALID_PARAM` |
| `sinceId` | string | — | 只要**严格更新**的记录（`id > sinceId`） |
| `untilId` | string | — | 只要**严格更旧**的记录（`id < untilId`），游标那条本身被排除 |
| `sinceDate` | int（毫秒时间戳） | — | **仅时间线支持**，且仅在 `sinceId` 为空时生效 |
| `untilDate` | int（毫秒时间戳） | — | **仅时间线支持**，且仅在 `untilId` 为空时生效 |

**排序方向**

- **时间线**：永远 `ORDER BY n.id DESC`（`internal/note/timeline.go:211`）。**`sinceId` 单独给出时也仍是倒序**——这与 Misskey 的 `makePaginationQuery`（since 单独给出时正序）不同，前端不要依赖"传 sinceId 就收到正序增量"。
- **通知**：`sinceId` 单独给出时 **正序 ASC**，其余情况倒序 DESC（`internal/note/reaction.go:296`–`300`），与上游一致。

**日期游标的实现细节（重要）**

`sinceDate` / `untilDate` 不是真正的按时间过滤，而是被翻译成 id 再过滤：
`TimelineService.SinceIDForDate`（`internal/note/timeline.go:232`）取的是 `created_at >= t` 的**最旧一条**笔记的 id。

- `sinceDate=t` → 等价于 `sinceId = (t 之后最早那条笔记的 id)`，所以**该条笔记本身被排除**。
- `untilDate=t` → 等价于 `untilId = 同一条 id`，同样排除它。
- **查不到符合条件的笔记时该 id 为空，日期过滤被静默忽略**（错误也被吞掉，`internal/httpapi/timelines.go:100`、`:103`），此时会返回最新一页。前端不要把"传了 sinceDate 拿到整页"当成服务端错误。

**通知不支持日期游标**：`/api/notifications` 的 body 结构里没有 `sinceDate`/`untilDate`，传了会被忽略。

**首次加载 / 翻页建议**

- 首屏：`{"limit":20}`。
- 加载更旧：`{"untilId": <当前最旧一条的 id>}`。
- 加载更新：时间线仍用 `{"sinceId": <当前最新一条 id>}`（注意返回仍是倒序）；通知用同样的 `sinceId`，返回正序，可直接 unshift。

### 6. 限流

**没有任何限流。** `internal/` 下没有速率限制中间件（`internal/httpapi/notes_favorites.go:14` 的注释也确认"Cinderfish has no rate limiter yet"）。
Misskey 的 `i/notifications`（30 次 / 30 秒）在本实现不存在，前端可以自由轮询，但也没有保护。

### 7. CORS

`/api/*` 与 `/streaming` **不返回任何 CORS 头**。只有 ActivityPub 相关响应会设置 `Access-Control-Allow-Origin: *`（`internal/httpapi/activitypub.go:49` 等）。
因此浏览器端只有在**同源**（或由反向代理补 CORS）时才能调用这些端点；详见 `docs/research/api-contract/meta-federation.md`。

---

## 共享对象

时间线、通知、Streaming 事件复用同一批 packer，字段完全一致。

### Note（`packNote`，`internal/httpapi/notes_pack.go:28`）

时间线的元素是**详细版**（`packNoteDetail`），除下表外还带 `### NoteDetailed 额外字段` 一节的字段。

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | 18 字符时间序 id |
| `createdAt` | string | RFC3339Nano（UTC） |
| `userId` | string | 作者 id |
| `user` | object | `UserLite`；作者查不到时退化为 `{"id": "<userId>"}` |
| `text` | string \| null | 正文（MFM） |
| `cw` | string \| null | 内容警告 |
| `visibility` | string | `public` / `home` / `followers` / `specified` |
| `localOnly` | bool | |
| `renoteCount` | int | 转发数（含引用） |
| `repliesCount` | int | |
| `reactionCount` | int | `reactions` 各值之和 |
| `reactions` | object\<string,int\> | emoji → 人数。没有回应时是 `{}`，**不会是 `null`** |
| `reactionEmojis` | array | **恒为空数组**（未实现自定义表情解析） |
| `emojis` | array | **恒为空数组** |
| `fileIds` | array\<string\> | 保留全部 id，包括已不存在的文件 |
| `files` | array\<DriveFile\> | 只含能查到的文件，顺序与 `fileIds` 一致 |
| `replyId` | string \| null | |
| `renoteId` | string \| null | 纯转发与引用都带 |
| `mentions` | array\<string\> | **本地被提及用户的 id**（不是用户名），见 `internal/note/service.go:252`；带 host 的远端提及不入库。恒为数组，不会是 `null` |
| `tags` | array\<string\> | 正文里抽取的 hashtag，保留原始大小写 |
| `uri` | string \| null | 远端对象的 AP id |
| `url` | string \| null | 远端原文链接 |
| `visibleUserIds` | array\<string\> | **仅 `visibility=specified` 时出现** |
| `reactionAcceptance` | string | **仅当该列非空时出现**，可能是 `likeOnly` 等 |

#### NoteDetailed 额外字段（`packNoteDetail`，`internal/httpapi/notes_pack.go:78`）

| 字段 | 类型 | 说明 |
|------|------|------|
| `clippedCount` | int | 恒 0（无剪贴功能） |
| `reply` | object \| null | **仅当 `replyId != null` 时出现**；父帖不可读或已删时为 `null`（`replyId` 保留）。内嵌对象是 `Note`（**非 detail**），且 `replyId`/`renoteId` 被置为 `null` |
| `renote` | object \| null | **仅当 `renoteId != null` 时出现**；不可读时为 `null`。内嵌对象是 `Note`（非 detail），保留自身 `replyId`/`renoteId` |
| `hasPoll` | bool | **仅当笔记带 poll 且读取成功时出现**（值为 `true`）。恒不存在表示"无投票" |
| `poll` | object | 与 `hasPoll` 同时出现：`{"multiple":bool,"expiresAt":int\|null,"choices":[{"text":string,"votes":int,"isVoted":bool}],"votesCount":int}`。`expiresAt` 是**毫秒时间戳** |
| `myReaction` | string | **仅登录且本人留过表情时出现** |

注意：`isSensitive` 虽然存在 `notes` 表并会被读出（`internal/note/service.go:430`），但 **packer 没有输出**，所以客户端拿不到笔记级的敏感标记。

### UserLite（`packUserLite`，`internal/httpapi/users_pack.go:12`）

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | |
| `name` | string \| null | 档案里的昵称；**通知里的 `user` 传的是 nil 档案，所以恒为 null** |
| `username` | string | |
| `host` | string \| null | 本地用户为 `null` |
| `avatarUrl` | string \| null | 同上，通知里恒为 null |
| `avatarBlurhash` | null | 恒 null |
| `avatarDecorations` | array | 恒 `[]` |
| `isBot` / `isCat` | bool | 恒 `false` |
| `requireSigninToViewContents` | bool | 恒 `false` |
| `emojis` | array | 恒 `[]` |
| `onlineStatus` | string | 恒 `"unknown"` |
| `badgeRoles` | array | 恒 `[]` |
| `instance` | null | 恒 null |

### DriveFile（`packDriveFile`，`internal/httpapi/drive.go:249`）

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | |
| `createdAt` | string | **毫秒精度**格式 `2006-01-02T15:04:05.000Z`（与其他对象的 RFC3339Nano 不同） |
| `name` | string | |
| `type` | string | MIME |
| `size` | int | 字节 |
| `isSensitive` | bool | |
| `url` | string | |
| `thumbnailUrl` / `webpublicUrl` | null | 恒 null（无缩略图/转码） |
| `comment` | null | 恒 null |
| `folderId` | string \| null | |
| `blurhash` | null | 恒 null |

### Notification（`packNotification`，`internal/httpapi/streaming_pack.go:58`）

HTTP 列表与 Streaming `notification` 事件共用此对象（注释明确要求两者不能漂移）。

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | |
| `createdAt` | string | RFC3339Nano |
| `type` | string | 实际只会出现 `mention` / `reply` / `renote` / `quote` / `reaction`（见下表） |
| `isRead` | bool | **Cinderfish 额外字段**，当前 Misskey 的 Notification schema 没有它 |
| `userId` | string \| null | 触发者 id（= `notifier_id`） |
| `user` | UserLite | 仅当 `userId` 非空且能查到用户时出现；此时 `name`/`avatarUrl` 恒为 null |
| `noteId` | string \| null | |
| `note` | Note（detail） | 仅当 `noteId` 非空且笔记行仍存在时出现（不校验可见性） |
| `reaction` | string \| null | 只有 `reaction` 类型有值 |

**不会出现的字段**：`body`（应用通知才有，本实现不产生）、`reaction` 之外的任何类型专属字段。
`userId` / `noteId` / `reaction` 三个键**始终存在**，缺值时为 `null`。

**通知类型来源**

| type | 产生位置 |
|------|----------|
| `mention` | 本地笔记正文 @到本地用户（`internal/httpapi/notes.go:147`、`:180`） |
| `reply` | 本地笔记的 `replyId` 目标（`internal/httpapi/notes.go:150`）；**远端入站笔记的回复和 @提及也都记成 `reply`**（`internal/federation/inbox.go:288`） |
| `renote` | 纯转发（`internal/httpapi/notes.go:153`） |
| `quote` | 引用转发（同上，`note.IsQuote()` 为真） |
| `reaction` | 本地用户给别人的笔记加表情（`internal/httpapi/notes.go:497`） |

同一个接收者只会收到一条：mention 优先级最低，reply/renote/quote 会覆盖同一条笔记上的 mention（`internal/httpapi/notes.go:131`–`145`）。
`specified` 笔记只通知被列入 `visibleUserIds` 的人（`:172`）。自己给自己的笔记做任何事都不产生通知。

**不会产生通知的行为**：关注/被关注、关注请求、投票结束、收藏、管理员公告、应用通知、`pollEnded`、`followRequestReceived`、`followRequestAccepted`。

---

## POST /api/notes/timeline

关注时间线（home）。`internal/httpapi/timelines.go:42`，内部走 `serveTimeline(..., note.TimelineHome, requireCredential=true)`。

### 请求

`Content-Type: application/json`，body 同通用分页参数：

| 字段 | 类型 | 必须 | 缺省 | 说明 |
|------|------|------|------|------|
| `i` | string | 是（或 Bearer） | — | 令牌 |
| `limit` | int | 否 | 10 | 见通用约定 |
| `sinceId` / `untilId` | string | 否 | — | id 游标 |
| `sinceDate` / `untilDate` | int（ms） | 否 | — | 见通用约定的近似语义 |
| `withFiles` | bool | 否 | false | 只要带文件的笔记 |
| `withRenotes` | bool | 否 | false | **在本实现中不影响纯转发过滤**，见"已知缺口" |
| `withReplies` | bool | 否 | false | Misskey 的 home timeline **没有**这个参数，本实现有 |
| `allowPartial` | bool | 否 | — | **解析但完全忽略** |
| `includeMyRenotes` | bool | 否 | **true** | false 时排除"我发的纯转发" |
| `includeRenotedMyNotes` | bool | 否 | **true** | false 时排除"转发了我的笔记的纯转发" |
| `includeLocalRenotes` | bool | 否 | **true** | false 时排除"来源是远端账号的纯转发"（带正文的引用不受影响） |

三个 `include*` 的语义对应 `internal/note/timeline.go:196`–`208`。注意 `includeLocalRenotes=false` 时判断的是 `n.renote_user_host IS NULL`——即**排除的是转发者来自远端**的转发，注释里的措辞（"renotes originating from a remote account"）是这个意思。

### 响应 `200`

```json
[
  {
    "id": "0000000000abcdefgh",
    "createdAt": "2026-11-04T10:20:30.123Z",
    "userId": "0000000000xxxxxxxx",
    "user": { "id": "...", "username": "alice", "host": null, "...": "..." },
    "text": "hello",
    "cw": null,
    "visibility": "public",
    "localOnly": false,
    "renoteCount": 0,
    "repliesCount": 0,
    "reactionCount": 1,
    "reactions": { "👍": 1 },
    "reactionEmojis": [],
    "emojis": [],
    "fileIds": [],
    "files": [],
    "replyId": null,
    "renoteId": null,
    "mentions": [],
    "tags": [],
    "uri": null,
    "url": null,
    "clippedCount": 0,
    "myReaction": "👍"
  }
]
```

**裸数组，没有外层包装**。空结果是 `[]`（不是 `null`——`out` 用 `make([]any, 0, len(notes))` 初始化）。每个元素都是 detail 形式（见共享对象）。

内容集合：`user_id = 我 OR user_id IN (我关注的人)`（`internal/note/timeline.go:153`–`156`），再叠加可见性过滤。

### 错误

| HTTP | code | 条件 |
|------|------|------|
| 400 | `INVALID_PARAM` | body 非法 |
| 401 | `AUTHENTICATION_FAILED` | 无令牌 / 坏令牌 / 账户停用 |
| 500 | `INTERNAL_ERROR` | 查询失败 |
| 503 | `INTERNAL_ERROR` | 实例无数据库（`s.timeline == nil`） |

### 已知缺口

- `withRenotes` **被解析但没有参与过滤**：`TimelineService.Timeline` 只在 `!WithRenotes` 时把三个 `include*` 重置为 true（`internal/note/timeline.go:137`–`141`），并**没有**排除纯转发。Misskey 的 home/local/hybrid 在 `withRenotes=false` 时会 `excludePureRenotes`。想"不要纯转发"的前端目前做不到。
- 没有屏蔽/静默数据（`internal/note/timeline.go:296` 注释明确说明），所以不会过滤被屏蔽用户。
- 没有频道（`channelId`）概念，因此也不会按"是否关注频道"过滤。
- `allowPartial` 无效；misskey 的 fanout（`enableFanoutTimeline`）也不存在，全部直查 `notes` 表。

---

## POST /api/notes/local-timeline

本地时间线。`internal/httpapi/timelines.go:26`。

### 请求

与 `/api/notes/timeline` 完全相同（共享 `timelineRequest`，`internal/httpapi/timelines.go:10`），且**可以匿名**。

### 响应 `200`

`Note[]`，detail 形式，倒序。空结果 `[]`。

内容集合：`visibility = 'public' AND user_host IS NULL`（`internal/note/timeline.go:147`–`148`）。
即使传了 `withReplies`、`includeMyRenotes` 等参数，可见性仍是"仅 public"，`home` 可见性的本地笔记**不会**出现在这里（已被 `TestLocalTimelineOnlyHasPublicLocalNotes` 钉住）。

匿名访问时访客看不到任何 `followers`/`specified` 笔记（`internal/note/timeline.go:84`–`88`），但这本来就被 `public` 条件挡住了。

### 错误

只有 400 / 500 / 503——**不会 401**。传坏令牌时 `authenticate` 的错误被忽略，按匿名处理并返回 200（`internal/httpapi/timelines.go:66`）。

### 已知缺口

- Misskey 在实例关闭 LTL 时返回 `LTL_DISABLED`；本实现没有开关，也不会返回该错误。
- 同 home：`withRenotes` 无效、无屏蔽过滤、无 `channelId` 概念（所以"频道投稿"概念上不存在，不需要排除）。
- `withFiles` 的实现是 `EXISTS (SELECT 1 FROM drive_files df WHERE df.id = ANY(n.file_ids))`（`internal/note/timeline.go:184`），与 Misskey 的 `fileIds != '{}'` 不同：**文件行被删掉的笔记会因此被漏掉**。

---

## POST /api/notes/hybrid-timeline

社交/混合时间线。`internal/httpapi/timelines.go:31`，`requireCredential=true`。

### 请求

同 `/api/notes/timeline`（`i` 必填）。

### 响应 `200`

`Note[]`（detail），倒序。内容集合：`我自己的 OR 我关注的 OR (public 且本地)`（`internal/note/timeline.go:158`–`161`）。

### 错误

400 / **401** / 500 / 503。`TestHybridTimelineRequiresCredential` 钉住了 401。

### 已知缺口

- Misskey 的 `STL_DISABLED` 不存在。
- 同 home：`withRenotes` 无效、无屏蔽、无频道。

---

## POST /api/notes/global-timeline

全局时间线。`internal/httpapi/timelines.go:36`，匿名可用。

### 请求

同 `/api/notes/timeline`，可匿名。`includeMyRenotes` 等开关在匿名时没有实际效果：`viewerArg` 会插入空字符串参数，而 `n.user_id <> ''` 恒真，条件恒成立（`internal/note/timeline.go:197`）。

### 响应 `200`

`Note[]`（detail），倒序。内容集合：`visibility = 'public'`（本地 + 远端，`internal/note/timeline.go:150`–`151`）。

### 错误

400 / 500 / 503；**不会 401**（坏令牌被忽略）。

### 已知缺口

- Misskey 的 `GTL_DISABLED` 不存在；也没有"联邦关闭时不显示远端笔记"以外的开关（联邦策略在入站侧决定是否落库，见 `meta-federation` 契约）。
- 无屏蔽过滤、无 `channelId` 过滤、`withRenotes` 无效。

---

## POST /api/notifications

当前用户的通知列表。`internal/httpapi/notifications.go:41`。别名 `POST /api/i/notifications` 注册了**同一个 handler**（`internal/httpapi/server.go:184`），两者响应逐字节相同（有测试 `TestINotificationsAliasMatchesCanonical` 钉住）。

### 请求

| 字段 | 类型 | 必须 | 缺省 | 说明 |
|------|------|------|------|------|
| `i` | string | 是（或 Bearer） | — | 令牌 |
| `limit` | int | 否 | 10 | `≤0`→10，`>100`→100（`internal/note/reaction.go:264`） |
| `sinceId` | string | 否 | — | 单独给出时结果**正序** |
| `untilId` | string | 否 | — | |
| `markAsRead` | bool | 否 | **true** | `true` 或省略 → 把本次返回的未读通知标为已读 |
| `includeTypes` | array\<string\> | 否 | — | 白名单。**`includeTypes: []`（显式空数组）返回空列表**；`null`/缺省表示不过滤 |
| `excludeTypes` | array\<string\> | 否 | — | 黑名单。**只有 `includeTypes` 为 null 时才生效**（include 优先，`internal/note/reaction.go:278`）。账号的 `mutingNotificationTypes` 会自动并入这里 |
| `sinceDate` / `untilDate` | — | — | — | **不支持，传了被忽略** |

`markAsRead` 的行为细节（`internal/httpapi/notifications.go:80`–`92`）：
只标记**这一页返回的、且原本未读**的行；标记成功后会把响应里的 `isRead` 一并置为 `true`。
所以：用 `{"includeTypes":["reaction"]}` 拉一页会把 reaction 通知标已读，而 mention 通知保持未读（`TestNotificationsMarkAsReadOnlyTouchesReturnedTypes`）。

### 响应 `200`

`Notification[]`（字段见共享对象）。顺序：默认最新在前；只给 `sinceId` 时最旧在前。

```json
[
  {
    "id": "0000000000abcdefgh",
    "createdAt": "2026-11-04T10:20:30.123Z",
    "type": "reaction",
    "isRead": true,
    "userId": "0000000000xxxxxxxx",
    "user": { "id": "...", "username": "bob", "name": null, "avatarUrl": null, "...": "..." },
    "noteId": "0000000000zzzzzzzz",
    "note": { "...": "Note detail" },
    "reaction": "🎉"
  }
]
```

### 错误

| HTTP | code | 条件 |
|------|------|------|
| 400 | `INVALID_PARAM` | body 非法 |
| 401 | `AUTHENTICATION_FAILED` | 无令牌 / 坏令牌 / 账户停用（`internal/httpapi/notifications.go:63`） |
| 500 | `INTERNAL_ERROR` | 查询失败 |
| 503 | `INTERNAL_ERROR` | 无通知服务 |

### 已知缺口

- 不支持 `sinceDate` / `untilDate`（上游支持）。
- 没有速率限制（上游 30/30s）。
- 通知种类只有 5 种（见共享对象表）；`follow`、`followRequestReceived`、`followRequestAccepted`、`pollEnded`、`app` 全部没有。
- 联邦入站产生的通知（目前只有 `reply`）**不会推送到 Streaming**：`federation.Store.Notification`（`internal/federation/store.go:620`）只写库，调用方没有 `publishNotificationCreated`。远端互动只能靠重新拉取这一页看到。
- `user` 不带昵称/头像（`packUserLite(author, "", nil)`，`internal/httpapi/streaming_pack.go:71`），前端要靠 `userId` 再查或直接用 `username`。
- `note` 字段只在笔记行仍存在时出现，**没有再校验可见性**（`internal/httpapi/streaming_pack.go:74`–`78` 只调 `notes.Get`）；笔记被软删则整个 `note` 键消失，`noteId` 仍保留。
- 本实现**总是**带 `isRead`，而当前 Misskey 的 `Notification` 联合类型里已经没有这个字段；前端如果直接复用上游类型定义，需要把它当成额外可选字段。

---

## POST /api/i/notifications

与 `POST /api/notifications` 完全等价（同一 handler，`internal/httpapi/server.go:184`）。请求、响应、错误、缺口见上一节。

这是 Firefish/Misskey 原生客户端的规范路径；保留 `/api/notifications` 是为了兼容早期本仓前端。

---

## POST /api/notifications/mark-all-as-read

把当前用户的所有通知标为已读。`internal/httpapi/notifications.go:106`。

### 请求

只有令牌。**body 里除 `i` 之外的字段全部忽略。**

### 响应 `200`

```json
{}
```

### 副作用

`markAllNotificationsRead`（`internal/httpapi/notifications.go:13`）在写库成功后，向该用户的 main 频道发一条 **`readAllNotifications`（body 为 `null`）** 事件，所以所有打开的客户端可以立即清零红点，无需轮询。

### 错误

401（未登录/坏令牌/停用）、500（更新失败）、503（无通知服务）。

### 已知缺口

- 上游返回空响应体（无内容），本实现返回 `{}` + 200。只判断状态码的客户端不受影响。
- 没有 `notifications/flush`（清空通知）端点。

---

## POST /api/hashtags/trend

趋势话题。`internal/httpapi/hashtags.go:13`。

### 请求

**无参数**（可以匿名，也可以带 `i`——带了也不影响结果）。limit 由 handler 固定为 10（`internal/httpapi/hashtags.go:22`）。

### 响应 `200`

```json
[
  { "tag": "golang", "chart": [0, 0, 0, 0, 0, 1, 2], "usersCount": 2 }
]
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `tag` | string | **小写**标签（SQL 里 `lower(tag)`） |
| `chart` | int[7] | 每天一个点，**最旧在前、今天在最后**，UTC 整天为桶；值是当天用过该标签的**去重本地用户数** |
| `usersCount` | int | `max(chart)`（`internal/note/hashtag_trend.go:107`） |

统计口径（`internal/note/hashtag_trend.go:54`–`70`）：

- 时间窗：包含今天在内的 7 个 UTC 日（`TrendBuckets = 7`）。
- 只统计 `user_host IS NULL`（本地作者）且 `deleted_at IS NULL` 的笔记。
- 按 `usersCount` 降序排序，同分按 `tag` 升序。
- 空闲实例返回 `[]`（`TestHashtagsTrendEmptyState`）。

### 错误

| HTTP | code | 条件 |
|------|------|------|
| 500 | `INTERNAL_ERROR` | 查询失败（`writeInternalServerError`） |
| 503 | `INTERNAL_ERROR` | 无 hashtag 服务 |

**不会 401 / 400**。

### 已知缺口

- 上游 `hashtags/trend` 声明了 `allowGet: true`（GET 也支持）与 `cacheSec: 60`；本实现**只注册了 POST**，GET 会得到 `404 NO_SUCH_ENDPOINT`，也没有缓存。
- 上游 `chart` 是 **20 个 10 分钟桶**（`HashtagService.getCharts(ranking, 20)`），本实现是 **7 个日桶**。字段形状一致（都是 `usersCount = max(chart)`），但横轴时间跨度完全不同；前端画折线图时要按 7 天处理。
- 上游把统计放在 Redis（`pfcount` 近似去重），本实现每次请求实时扫 `notes` 表：时间窗可走 `idx_notes_created`（`internal/db/db.go:93`），但 `jsonb_array_elements_text(n.tags)` 的展开与聚合**没有索引可用**（`tags` 列没有 GIN 索引），大实例上会随窗口内笔记数线性变慢。
- `hashtags/show`（单标签详情）**未实现**。

---

## GET /streaming

Misskey Streaming API 的 WebSocket 入口。`internal/httpapi/streaming.go:45` → `internal/streaming/Server.ServeHTTP`（`internal/streaming/server.go:92`）。

**本节只写时间线与通知相关的最小必需信息。** 完整协议（帧容错、连接级限制、跨进程中继、全部事件）见 `docs/research/api-contract/streaming.md`。

### 握手

- URL：`ws(s)://<host>/streaming`，可选 `?i=<token>`。
- 令牌优先级：`Authorization: Bearer <token>` → 否则 query 的 `i`（`internal/streaming/server.go:93`–`98`）。**没有** body 传参（WebSocket 握手没有 body）。
- **无令牌 = 合法匿名连接**。
- 令牌存在但无效 → **401**，带
  `WWW-Authenticate: Bearer realm="Misskey", error="invalid_token", error_description="Failed to authenticate"`，**响应体为空**（不是 JSON 错误信封）。
- 应用令牌（`access_tokens`）必须含 `read:account`，否则同样 401（`internal/httpapi/streaming_adapters.go:24`）；原生 16 位用户令牌不需要 scope。
- 已停用账户 → **403**，空响应体。
- 未 `AttachStreaming`（无数据库的实例）→ **503 + JSON 错误信封**（`internal/httpapi/streaming.go:46`）。
- 不是 WebSocket 请求（普通 GET）→ gorilla 的 `400 Bad Request`（纯文本）。

### 客户端 → 服务端帧

`{"type": "...", "body": ...}`，无法解析的帧被静默忽略（`internal/streaming/connection.go:123`）。

| `type` | 别名 | body | 行为 |
|--------|------|------|------|
| `connect` | — | `{"channel":string,"id":string,"params"?:object,"pong"?:bool}` | 订阅频道 |
| `disconnect` | — | `{"id":string}` | 退订 |
| `channel` | `ch` | `{"id","type","body"}` | 只校验形状，**无任何效果**（本实现的频道都不接收消息） |
| `subNote` | `s`、`sr` | `{"id":string}` | 订阅单条笔记的更新（引用计数） |
| `unsubNote` | `un` | `{"id":string}` | 退订 |
| `readNotification` | — | 忽略 | 等同于 `mark-all-as-read` |

`pong: true` 时服务端回一帧 `{"type":"connected","body":{"id":"<你给的 id>"}}`（`internal/streaming/connection.go:231`）。

### 频道

`internal/streaming/channel_main.go:20` 是唯一注册表，只有 5 个：

| channel | 需要登录 | params 与默认值 | 说明 |
|---------|----------|------------------|------|
| `main` | ✔ | 无 | 每连接最多一个（shareable）；个人事件 |
| `homeTimeline` | ✔ | `withRenotes`=**true**、`withFiles`=false；**没有 `withReplies` 参数** | `internal/streaming/channel_timelines.go:21` |
| `localTimeline` | ✘ | `withRenotes`=**false**、`withReplies`=false、`withFiles`=false | `:81` |
| `hybridTimeline` | ✔ | `withRenotes`=**false**、`withReplies`=false、`withFiles`=false | `:182` |
| `globalTimeline` | ✘ | `withRenotes`=**false**、`withFiles`=false | `:143` |

**注意 `withRenotes` 的默认值与上游相反**：Misskey 四个频道都是 `withRenotes ?? true`（`tmp/refcheck/misskey/packages/backend/src/server/api/stream/channels/*-timeline.ts`），本实现除 `homeTimeline` 外都是 `false`——即**默认丢弃纯转发**。客户端要收到转发必须在 `params` 里显式传 `"withRenotes": true`。

未实现的频道（`userList`、`clip`、`hashtag`、`antenna`、`channel`、`roleTimeline`、`drive`、`serverStats`、`admin`、`chat*`、`reversi*`）一律**静默丢弃** `connect`：不回帧、不断连。

### 服务端 → 客户端帧

| `type` | body | 触发 |
|--------|------|------|
| `connected` | `{"id"}` | `connect` 且 `pong:true` |
| `channel` | `{"id","type","body"}` | 频道事件，`id` 是客户端给的那个 id |
| `noteUpdated` | `{"id","type","body"}` | `subNote` 的笔记更新 |

### 时间线相关事件

- 各 timeline 频道的 `channel` 帧里 `type = "note"`，`body` 是 **Note（detail 形式，按匿名访客打包）**：
  `packNoteFor(n, "")`（`internal/httpapi/streaming_publish.go:32`），因此**没有 `myReaction`**，且 `reply`/`renote` 内嵌对象也按匿名可见性判断，可能是 `null`。
- `subNote` 订阅收到的 `noteUpdated.type` 只有三种：
  - `reacted` → body `{"reaction":"👍","userId":"...","emoji":null}`（`emoji` 恒为 null，`internal/httpapi/streaming_publish.go:76`–`83`）
  - `unreacted` → body `{"reaction":"👍","userId":"..."}`（**没有 `emoji` 键**）
  - `deleted` → body `{"deletedAt":"<RFC3339Nano>"}`
- **没有 `pollVoted`**：`notes/polls/vote` 不发布任何 noteStream 事件（`internal/httpapi/notes_polls.go:67`–`93`）。

### 通知相关事件（`main` 频道）

| `type` | body | 说明 |
|--------|------|------|
| `notification` | `Notification` | 与 HTTP 列表**同一 packer**，可用于增量插入 |
| `mention` | Note（detail，按被提及者打包） | 只有本地新建笔记 @到你时才发（`internal/httpapi/streaming_publish.go:50`） |
| `reply` | Note（detail，按匿名打包） | 别人回复你的笔记（仅 `replyUserID` 指向你时） |
| `renote` | Note（detail，按匿名打包） | 别人转发你的笔记 |
| `follow` | `{"id":"<关注者的 userId>"}` | 有人关注你 |
| `followed` | `{"id":"<被关注者的 userId>"}` | 你关注了别人（你自己的操作回显） |
| `unfollow` | `{"id":"<取关者的 userId>"}` | 有人取关你 |
| `readAllNotifications` | `null` | 任一处调用了 `mark-all-as-read` |

**不存在的事件**：`unreadNotification`、`notificationFlushed`、`meUpdated`、`driveFileCreated`、`signup`、`announcementCreated`。

### 连接级行为（前端必须知道的）

| 项 | 值 | 位置 |
|----|----|------|
| 每连接频道数上限 | 32（超限的 `connect` 静默丢弃） | `internal/streaming/conn.go:13` |
| `subNote` 上限 | 1536，超出时淘汰最久未用的一条 | `:15` |
| 发送队列 | 256 帧，写不进去直接**断开连接** | `:32` |
| 心跳 | 服务端每 60s ping 一次 | `:19` |
| 超时断开 | 超过 120s **没有收到任何客户端数据帧**即断开 | `:22`、`internal/streaming/connection.go:98` |
| follow 图刷新 | 每 10s 重载一次 | `internal/streaming/connection.go:27`、`:108` |
| 同 id 重连频道 | 先销毁旧频道，再建新的 | `internal/streaming/connection.go:175` |

**心跳的坑**：`lastSeen` 只在读到**数据帧**时刷新（`internal/streaming/connection.go:94`–`98`），
标准 WebSocket 的 pong 控制帧由 gorilla 内部处理、不会让 `ReadMessage` 返回，因此**不会刷新活跃时间**。
一个只回 pong、从不发数据帧的客户端会在 120s 后被断开。前端应定期（建议 60s 内）发一帧 `{"type":"ping"}` 之类会被服务端忽略的数据帧来续命——
本实现会忽略未知 `type`（`internal/streaming/connection.go:132`–`149` 没有 default 分支的动作），但 `ReadMessage` 成功仍会刷新时间，正好够用。

### 已知缺口

- 远端笔记的 Streaming 事件是**残缺的**：联邦入站只发 `PublishTyped("notesStream", "note", {"id","userId"})`（`internal/httpapi/federation_wire.go:50`）。
  `localTimeline` / `globalTimeline` 会因为 `visibility` 缺失（不是 `public`）而丢掉它；但 `homeTimeline` / `hybridTimeline` 在"已关注该远端作者"时会把这个**只有 id 和 userId 的对象**当成一条笔记推给客户端。
  前端收到缺少 `text`/`user`/`createdAt` 的 note 时应回落到 REST 拉取（`notes/show`）或直接忽略。
- 没有 `unreadNotification` 延迟事件，红点只能靠 `notification` 事件自增或轮询 `/api/i` 的 `unreadNotificationsCount`。`unreadNotificationsCount` 与列表一样排除 `mutingNotificationTypes` 里的类型。
- `follow`/`followed`/`unfollow` 的 body 是 `{"id":...}`，上游是 Packed User；只读 `body.id` 的客户端没问题。
- 无 mute/block，`isNoteMutedOrBlocked` 恒 false（`internal/streaming/channel_timeline.go:99`）；无频道概念，`hasChannelID` 恒 false（`:90`）。
- `withReplies` 的回复过滤用 `following.withReplies`，而 `followings` 表**没有 `withReplies` 列**，`LoadFollows` 恒返回 `false`（`internal/httpapi/streaming_adapters.go:59`）——即"关注时选择显示回复"这个设置目前不生效。

---

## 本命名空间与 Misskey 的差异汇总

| 主题 | Misskey | Cinderfish | 影响 |
|------|---------|-----------|------|
| `withRenotes=false` | home/local/hybrid 排除纯转发 | 只重置 `include*`，**不排除** | 前端无法隐藏纯转发 |
| 时间线 `sinceId` 单独给出 | 正序 | **倒序** | 增量加载要自己反序 |
| `withReplies`（`notes/timeline`） | 无此参数 | 有，默认 false | 兼容，多一个可用开关 |
| `limit` 越界 | `INVALID_PARAM` | `≤0`→10、`>100`→100 | 静默降级 |
| `allowPartial` | 生效（fanout） | 忽略 | 实效上等价于 false |
| LTL/GTL/STL 关闭 | `*_DISABLED` | 不支持 | 无对应错误码 |
| `hashtags/trend` GET | 支持 | 不支持（404） | 只能用 POST |
| `hashtags/trend` chart | 20×10min | 7×1day | 图表横轴不同 |
| `i/notifications` 日期游标 | 支持 | 忽略 | 只能用 id |
| `i/notifications` 限流 | 30/30s | 无 | — |
| `notifications/mark-all-as-read` 响应 | 空体 | `{}` + 200 | — |
| 通知 `isRead` 字段 | 当前 schema 无 | 恒返回 | — |
| Streaming `withRenotes` 默认 | 四频道皆 true | 仅 home 为 true | 其它频道默认丢转发 |
| Streaming 心跳刷新 | pong 刷新 | 仅数据帧刷新 | 空闲客户端 120s 被断 |
| 通知类型 | 十余种 | 5 种 | 关注/投票结束等无通知 |

---

## 对接检查清单

1. 首屏：`POST /api/notes/hybrid-timeline`（登录）或 `/api/notes/local-timeline`（匿名）带 `{"limit":20}`。
2. 翻页：`untilId` = 当前最旧 id；"加载更新"用 `sinceId`，**记得结果可能仍是倒序**。
3. 通知：`POST /api/i/notifications` 带 `{"limit":20,"markAsRead":false}` 进入通知页，离开时再调 `mark-all-as-read`（避免进页面就清红点）。
4. 红点：读 `/api/i` 的 `unreadNotificationsCount` / `hasUnreadNotification`，或订阅 `main` 频道的 `notification` 事件自增。
5. 实时：连 `/streaming?i=<token>`，`connect` `main`（pong:true）+ `hybridTimeline`（显式传 `withRenotes:true`）。
6. 定时发心跳数据帧（<120s），否则连接会被服务端断开。
7. 趋势：`POST /api/hashtags/trend`，按 7 天日桶绘制。

## 2026-11-06 增补：通知未读过滤与提及

### POST /api/i/notifications 的 `unreadOnly`

新增可选 `unreadOnly: boolean`。为 true 时只返回 `is_read = FALSE` 的通知，与
`includeTypes`/`excludeTypes` 可组合。

### POST /api/notes/mentions

需登录。参数：`following`（只保留关注者与自己）、`limit`、`sinceId`/`untilId`、`visibility`。
返回 `Note[]`，匹配 `notes.mentions`（本地用户 id 数组）并套用可见性、静音、屏蔽与实例静音过滤。
远端账号的提及不会进入该数组，因此不出现在此列表。
