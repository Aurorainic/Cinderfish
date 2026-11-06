# API 对接契约：notes-core

本文覆盖 Cinderfish 后端的**笔记核心**命名空间：笔记的创建/读取/删除、回复与转发列表、会话与子帖、回应（reaction）、收藏、投票。目标读者是重新编写 Firefish 风格前端的开发者，因此每条结论都以**本仓库实际 handler 代码**为准，而不是 Misskey 文档。

- 路由注册：`internal/httpapi/server.go:163`–`177`
- Handler：`internal/httpapi/notes.go`、`notes_pack.go`、`notes_thread.go`、`notes_favorites.go`、`notes_reactions_list.go`、`notes_polls.go`
- 服务层：`internal/note/service.go`、`timeline.go`、`conversation.go`、`reaction.go`、`favorite.go`、`renote.go`、`mention.go`、`hashtag.go`
- 与 Misskey 的差距参照：`docs/research/2026-11-04-misskey-api-coverage.md`

本命名空间内**全部端点都是 `POST`**，且都以 JSON 请求体传参（包括 `i` 令牌）。除特别说明外，路径前缀是 `/api`。

---

## 端点总览

| 端点 | 认证 | 核心参数 | 成功响应 | 实现位置 |
|------|------|----------|----------|----------|
| `POST /api/notes/create` | 必需 | `visibility`、`text`、`cw`、`fileIds`/`mediaIds`、`replyId`、`renoteId`、`poll`、`visibleUserIds`、`localOnly` | `{"createdNote": NoteDetailed}` | `internal/httpapi/notes.go:32` |
| `POST /api/notes/show` | 可选（弱认证） | `noteId` | `NoteDetailed` | `internal/httpapi/notes.go:255` |
| `POST /api/notes/delete` | 必需（仅作者） | `noteId` | `{}` | `internal/httpapi/notes.go:313` |
| `POST /api/notes/edit` | 必需（仅作者） | `editId`、`text`、`cw`、`visibility`、`localOnly` | `{"createdNote": Note}` | `internal/httpapi/note_edit.go:26` |
| `POST /api/notes/history` | 必需（仅作者） | `noteId`、`limit` | `NoteEdit[]`（自有端点） | `internal/httpapi/note_edit.go:96` |
| `POST /api/notes/replies` | 可选（弱认证） | `noteId`、`limit`、`sinceId`、`untilId` | `NoteDetailed[]` | `internal/httpapi/notes.go:370` |
| `POST /api/notes/renotes` | 可选（弱认证） | 同上 | `NoteDetailed[]` | `internal/httpapi/notes.go:376` |
| `POST /api/notes/conversation` | 可选（弱认证） | `noteId`、`limit`、`offset` | `Note[]`（非 detail） | `internal/httpapi/notes_thread.go:13` |
| `POST /api/notes/children` | 可选（弱认证） | `noteId`、`limit`、`sinceId`、`untilId`、`sinceDate`、`untilDate` | `Note[]`（非 detail） | `internal/httpapi/notes_thread.go:59` |
| `POST /api/notes/reactions` | 可选（弱认证） | `noteId`、`type`、`limit`、`sinceId`、`untilId`、`sinceDate`、`untilDate` | `NoteReaction[]` | `internal/httpapi/notes_reactions_list.go:20` |
| `POST /api/notes/reactions/create` | 必需 | `noteId`、`reaction` | `{}` | `internal/httpapi/notes.go:438` |
| `POST /api/notes/reactions/delete` | 必需 | `noteId`、`reaction`（可省略） | `{}` | `internal/httpapi/notes.go:508` |
| `POST /api/notes/favorites/create` | 必需 | `noteId` | `NoteFavorite` | `internal/httpapi/notes_favorites.go:16` |
| `POST /api/notes/favorites/delete` | 必需 | `noteId` | `{}` | `internal/httpapi/notes_favorites.go:71` |
| `POST /api/i/favorites` | 必需 | `limit`、`sinceId`、`untilId`、`sinceDate`、`untilDate` | `NoteFavorite[]` | `internal/httpapi/notes_favorites.go:106` |
| `POST /api/notes/polls/vote` | 必需 | `noteId`、`choice` | `{}` | `internal/httpapi/notes_polls.go:23` |

本命名空间**没有** GET 别名，也没有短路径别名（`notes/create`、`notes/show` 等即为 Misskey 原生路径）。

---

## 通用约定

### 1. 认证

`Server.authenticate`（`internal/httpapi/auth.go:30`）按以下顺序取令牌：

1. 请求体的 `i` 字段（字符串）；
2. `Authorization: Bearer <token>` 头。

**两者都给时以请求体 `i` 为准**。没有令牌时 `authenticate` 返回 `(nil, nil)`，即匿名访问，由各 handler 自行决定是否要求登录。

- 令牌无效或已吊销 → handler 调用 `writeAuthenticationError`，返回 **HTTP 401**，body 为 `AUTHENTICATION_FAILED`，并带 `WWW-Authenticate` 头（`internal/httpapi/server.go:304`）。
- 账户被停用（`users.is_suspended`）即使令牌有效也按认证失败处理（`internal/httpapi/auth.go:52`）。
- **弱认证端点**：`notes/show`、`notes/replies`、`notes/renotes`、`notes/conversation`、`notes/children`、`notes/reactions` 会**忽略认证错误**（代码写成 `info, _ := s.authenticate(...)`），所以传一个坏令牌不会得到 401，而是被当成匿名访问。前端不要依赖这些端点报错来发现令牌失效。
- 没有 admin 要求：本命名空间所有端点都只区分"登录/未登录/是否本人"。

### 2. 请求体解码

`decodeOptionalJSON`（`internal/httpapi/ping.go:23`）：

- 请求体上限 **8 MiB**（`io.LimitReader(r.Body, 8<<20)`）：超出部分被直接截断，JSON 随之解析失败，得到 `400 INVALID_PARAM`。
- 空 body 合法，视为"全部参数缺省"。
- JSON 非法 → `400 INVALID_PARAM`，`info` 为 `{"reason":"body is not valid JSON"}`。
- **未知字段被静默忽略**（标准 `encoding/json` 行为），所以拼错参数名不会报错，只会表现为"按默认值处理"。

### 3. 响应编码

- 成功统一走 `writeJSON`（`internal/httpapi/server.go:235`）：`Content-Type: application/json; charset=utf-8`，`SetEscapeHTML(false)`（文本里的 `<`、`>`、`&` 不会被转成 `\u003c` 之类）。
- 空结果对象写作 `{}`（例如删除类端点），空列表写作 `[]`，**不会是 `null`**。
- 字段名一律 camelCase，与 Misskey 一致。

### 4. 错误信封

`APIError` + `writeAPIError`（`internal/httpapi/server.go:246`、`269`）：

```json
{ "error": { "message": "...", "code": "NO_SUCH_NOTE", "id": "uuid", "kind": "client", "info": ... } }
```

- `kind` 恒存在，取值 `client` / `server` / `permission`。
- `info` 只在部分校验错误里出现，**类型不稳定**：有时是字符串（`"noteId is required"`），有时是对象（`{"reason": "..."}`）。前端应把它当不透明值（用于调试展示），不要按结构解析。
- HTTP 状态码由 handler 指定，默认 `400`。
- 未匹配任何路由（包括路由存在但 HTTP 方法不对）→ `404 NO_SUCH_ENDPOINT`，`id` 为 `fc9c4a0b-f3a1-4b0e-9a8c-6f7f3d0f6a12`（`internal/httpapi/server.go:327`）。**注意**：因为注册了兜底 `/` 路由，用 `GET` 访问 `POST /api/notes/show` 返回的是这个 404，而不是 405。

常用错误 id（前端若要按 id 而不是 code 分支，用这些值）：

| code | id | 说明 |
|------|----|------|
| `INVALID_PARAM` | `3d81ceae-475f-4600-b2a8-2bc116157532` | 参数校验失败 |
| `AUTHENTICATION_FAILED` | `b0a7f5f8-dc2f-4171-b91f-de88ad238e14` | 未登录/令牌无效 |
| `INTERNAL_ERROR` | `5d37dbcb-891e-41ca-a3d6-e690c97775ac` | 服务端错误 |
| `NO_SUCH_ENDPOINT` | `fc9c4a0b-f3a1-4b0e-9a8c-6f7f3d0f6a12` | 路由不存在 |
| `NO_SUCH_NOTE` | 因端点而异，**共 5 个不同 id** | 见下表 |

`NO_SUCH_NOTE` 的 id **不统一**：多数照抄 Misskey 各端点自己的定义，而 `notes/delete`、`notes/conversation`、`notes/favorites/create` 直接复用了 `notes/show` 的 id（Misskey 这三个端点各有独立 id）。前端不要跨端点硬编码：

| 使用处 | id |
|--------|-----|
| `notes/show`、`notes/delete`、`notes/conversation`、`notes/favorites/create` | `24fcbfc6-2e37-42b6-8388-c29b3861a08d` |
| `notes/renotes` | `12908022-2e21-46cd-ba6a-3edaf6093f46` |
| `notes/reactions/create` | `033d0620-5bfe-4027-965d-980b0c85a3ea` |
| `notes/reactions` | `263fff3d-d0e1-4af4-bea7-8408059b451a` |
| `notes/polls/vote` | `ecafbd2e-c283-4d6d-aecb-1a0a33b75396` |

`notes/replies` 与 `notes/children` 对不存在的笔记**不报错**，返回空数组。

### 5. 隐私语义（可见性）

可见性判定有两处实现，规则大体一致但不完全相同：单条笔记用 `internal/note/service.go:485` `VisibleTo`（内部走 `noteVisibleToTx`，`:373`），列表用 `internal/note/timeline.go:84` `addNoteVisibility`（一条 SQL 条件）。

- `public` / `home`：任何人可读，匿名可读。
- `followers`：作者、作者粉丝、笔记回复的对象、被提及者可读。另外 `VisibleTo` 多一条放宽：当笔记作者与访问者**都是远端用户**时，二者的关注关系在本机不可知，按"假设已关注"处理（`internal/note/service.go:408`–`420`）。这条放宽**没有**出现在列表过滤里，所以同一条远端 followers 笔记可能"`notes/show` 能看、timeline 不出现"。
- `specified`：作者与 `visibleUserIds` 中的用户。

对**不可读**的笔记，`notes/show`、`notes/reactions`、`notes/polls/vote` 一律按"不存在"回答（`404 NO_SUCH_NOTE`），避免泄露存在性；`notes/conversation` 的父链在某层不可读时就地截断；`notes/replies`/`notes/renotes`/`notes/children` 只是把它从结果里过滤掉。

### 6. 分页与游标

所有列表端点共用同一套 Misskey 语义：

- `limit`：`<= 0` 取默认 **10**，`> 100` 被压到 **100**。各端点默认值相同。
- `sinceId` / `untilId`：id 字符串，按字符串比较（`n.id > $x` / `n.id < $x`）。id 是时间有序的，所以字典序即时间序。
- **方向规则**（`internal/note/timeline.go:116`、`internal/note/reaction.go:107`、`internal/note/favorite.go:126`）：只给 `sinceId`（且未给 `untilId`／`untilDate`）时按 **升序** 返回（"加载更新的"方向）；其他任何组合都是 **降序**（最新在前）。
- `sinceDate` / `untilDate`：**毫秒时间戳**（`int64`，`time.UnixMilli`）。只在 `notes/children`、`notes/reactions`、`i/favorites` 上支持；`notes/replies`、`notes/renotes`、`notes/conversation` 不支持。
- `notes/reactions` 与 `i/favorites` 的升降序判断还额外看 `untilDate` 是否为空。
- 游标是以**记录自身的 id** 为基准的：`i/favorites` 的游标是 favorite 行的 id，不是 note 的 id。

### 7. 时间与 ID 格式

- 笔记、收藏、回应的 `createdAt`：Go `time.RFC3339Nano`，UTC，例如 `2026-11-04T12:34:56.789012345Z`（小数位数不固定）。
- **例外**：`DriveFile.createdAt` 用固定毫秒格式 `2006-01-02T15:04:05.000Z`（`internal/httpapi/drive.go:249`）。前端解析时间需容忍两种。
- 所有 id 是 **18 字符 base36** 字符串（`internal/idgen/idgen.go`，Misskey `aid` 风格）：前 10 位是毫秒时间戳、后 8 位随机。**不是 UUID，也不是 ULID**——没有连字符，长度固定 18。
- 投票的 `expiresAt` 是**毫秒时间戳**（不是 ISO 字符串），见 `Note.poll`。

### 8. 限流

**本命名空间没有任何限流**。Misskey 的 `notes/favorites/create`（每小时 20 次）上限在代码里只留了 TODO（`internal/httpapi/notes_favorites.go:13`–`15`）。前端不应假设会收到 `RATE_LIMIT_EXCEEDED`。

---

## 共享对象

### Note（`packNote`，`internal/httpapi/notes_pack.go:28`）

这是 Misskey `Note` schema 的实现子集。`notes/conversation`、`notes/children`、`NoteFavorite.note` 直接返回它；`NoteDetailed` 在此基础上加字段。

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | 18 字符 base36，时间有序 |
| `createdAt` | string | RFC3339Nano |
| `userId` | string | 作者 id |
| `user` | `UserLite` | 作者；作者行缺失时退化为 `{"id": "<userId>"}` |
| `text` | string \| null | 正文（含 MFM），纯转发时为 `null` |
| `cw` | string \| null | 内容警告 |
| `visibility` | string | `public` / `home` / `followers` / `specified`（已按继承规则改写后的最终值） |
| `localOnly` | bool | 是否不联邦 |
| `renoteCount` | int | 被转发数（来自 `notes.renote_count`，**删除转发不回退**） |
| `repliesCount` | int | 回复数（来自 `notes.replies_count`，**删除回复不回退**） |
| `reactionCount` | int | 回应总数（实时按 `note_reactions` 聚合，非落库列） |
| `reactions` | object | `{ "👍": 3, ":custom:": 1 }`，无回应时为 `{}`（不是 `null`） |
| `reactionEmojis` | array | **恒为 `[]`**。Misskey 这里是 `{shortcode: url}` 映射，Cinderfish 未实现 |
| `emojis` | array | **恒为 `[]`**。Misskey 这里是 `{shortcode: url}` **对象**，Cinderfish 既未收集表情，类型也对不上 |
| `fileIds` | string[] | 附件 id，恒为数组 |
| `files` | `DriveFile[]` | 按 `fileIds` 顺序内嵌；找不到的文件被跳过 |
| `replyId` | string \| null | |
| `renoteId` | string \| null | 纯转发与引用都会填 |
| `mentions` | string[] | **本地用户 id 列表**（不是用户名）。仅解析出本地用户，远端提及不入此列 |
| `tags` | string[] | 从正文提取的 hashtag（去重不区分大小写，保留首次出现时的原始大小写） |
| `uri` | string \| null | 远端笔记的 ActivityPub id；本地笔记为 `null` |
| `url` | string \| null | 远端笔记的原站链接；本地为 `null` |
| `visibleUserIds` | string[] | **仅当 `visibility === "specified"` 时出现** |
| `reactionAcceptance` | string | **仅当数据库列非空时出现**，取值未在本仓库约束 |

**不返回**的 Misskey 字段（前端不要引用）：`isSensitive`（库里有列但不输出）、`clippedCount`（只在 detail 里）、`channelId`、`isHidden`、`poll`（只在 detail 里）、`reply`/`renote`（只在 detail 里）、`myReaction`（只在 detail 里）。

### NoteDetailed（`packNoteDetail`，`internal/httpapi/notes_pack.go:78`）

在 `Note` 全部字段之上追加：

| 字段 | 类型 | 说明 |
|------|------|------|
| `clippedCount` | int | 恒为 0（剪辑功能未实现） |
| `reply` | `Note` \| null | **仅当 `replyId != null` 时存在**。用 `packNote`（非 detail）打包，且其自身的 `replyId`、`renoteId` 被置为 `null`（"浅拷贝"）。父帖不可读/已删除时为 `null`，`replyId` 仍保留 |
| `renote` | `Note` \| null | **仅当 `renoteId != null` 时存在**。用 `packNote` 打包；转发目标不可读/已删除时为 `null` |
| `hasPoll` | bool | **仅当笔记带投票时出现**，值为 `true` |
| `poll` | object | **仅当笔记带投票时出现**，见下 |
| `myReaction` | string | **仅当登录者自己留过回应时出现**；没有该键（不是 `null`） |

`poll` 结构：

```json
{
  "multiple": false,
  "expiresAt": 1730000000000,
  "choices": [ { "text": "A", "votes": 3, "isVoted": false } ],
  "votesCount": 7
}
```

- `expiresAt`：**毫秒数字**，无期限时为 `null`。**注意与 Misskey 不同**：Misskey 的 `Note.poll.expiresAt` 是 ISO 8601 字符串（`format: 'date-time'`），前端拿到数字后要自己格式化。
- `choices[].votes`：该选项票数。
- `choices[].isVoted`：当前登录者是否选了该项（多选投票会多个为 `true`；匿名恒 `false`）。
- `votesCount`：总票数，Cinderfish 的额外字段（Misskey 的 `Note.poll` 没有它，客户端自己加总 `choices[].votes`）。

注意 detail 字段**不参与嵌套**：`Note.reply` / `Note.renote` 里的子对象没有 `poll`、`myReaction`、`clippedCount`，但**有** `files` 与 `visibleUserIds`。

### UserLite（`packUserLite`，`internal/httpapi/users_pack.go:12`）

笔记里 `user` 字段用的就是这个。字段固定为：

| 字段 | 值 |
|------|-----|
| `id` | string |
| `name` | string \| null（profile 里没设名字就是 `null`） |
| `username` | string（不含 `@`，不含 host） |
| `host` | string \| null（本地用户为 `null`） |
| `avatarUrl` | string \| null（本地相对路径或绝对 URL；未上传则为 `null`） |
| `avatarBlurhash` | 恒 `null` |
| `avatarDecorations` | 恒 `[]` |
| `isBot` | 恒 `false` |
| `isCat` | 恒 `false` |
| `requireSigninToViewContents` | 恒 `false` |
| `emojis` | 恒 `[]` |
| `onlineStatus` | 恒 `"unknown"` |
| `badgeRoles` | 恒 `[]` |
| `instance` | 恒 `null` |

前端如需显示 `avatarUrl`，注意它可能是 **root-relative**（例如 `/media/objects/...`），需要拼 origin；S3 后端配置了公网基址或被预签名时才是绝对 URL。

### DriveFile（`packDriveFile`，`internal/httpapi/drive.go:249`）

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | |
| `createdAt` | string | 固定毫秒格式 |
| `name` | string | 文件名 |
| `type` | string | MIME |
| `size` | int | 字节 |
| `isSensitive` | bool | |
| `url` | string | 可能是相对路径（本地存储）或绝对 URL（S3） |
| `thumbnailUrl` | 恒 `null` | 未生成缩略图 |
| `webpublicUrl` | 恒 `null` | |
| `comment` | 恒 `null` | |
| `folderId` | string \| null | |
| `blurhash` | 恒 `null` | |

**未返回**：`userId`、`isLink`、`md5`、`requestHeaders`。

### NoteFavorite

`notes/favorites/create`、`i/favorites` 的元素：

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | favorite 行 id（**不是** note id） |
| `createdAt` | string | RFC3339Nano |
| `noteId` | string | |
| `note` | `Note` | **非 detail**，因此没有 `poll`/`myReaction` |

### NoteReaction（`notes/reactions` 的元素）

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | 回应行 id |
| `createdAt` | string | RFC3339Nano |
| `type` | string | 表情（unicode 或 `:shortcode:`） |
| `user` | `UserLite` | 回应者 |

**未返回**：`noteId`、`reaction`（Misskey 用 `type` 表示表情，这里照做）。

---

## POST /api/notes/create

创建笔记、回复、转发或引用。实现：`internal/httpapi/notes.go:32`；校验与落库在 `internal/note/service.go:137`。

### 请求字段

| 字段 | 类型 | 必填 | 默认 | 校验 / 行为 |
|------|------|------|------|-------------|
| `visibility` | string | 否 | `"public"` | 只接受 `public` / `home` / `followers` / `specified`，否则 `INVALID_VISIBILITY`。**不接受 `direct` / `private`** |
| `visibleUserIds` | string[] | 视情况 | `[]` | `visibility === "specified"` 时**必须非空**，否则 `INVALID_VISIBILITY` |
| `text` | string \| null | 否 | `null` | 长度上限 **3000 字节**（`len()`，不是字符数），超出 `NOTE_TOO_LONG`。中文按 UTF-8 三字节计，约 1000 字 |
| `cw` | string \| null | 否 | `null` | 不限长度 |
| `localOnly` | bool | 否 | `false` | 回复/转发时会被目标加强为 `true` |
| `fileIds` | string[] | 否 | `[]` | 附件。**不校验文件是否存在、是否属于本人**（`internal/note/service.go:320`–`326` 的检查实际是空操作） |
| `mediaIds` | string[] | 否 | — | `fileIds` 为空时才使用（`internal/httpapi/notes.go:69`–`72`）。两者都给时只认 `fileIds` |
| `replyId` | string \| null | 否 | `null` | 见下方回复规则 |
| `renoteId` | string \| null | 否 | `null` | 见下方转发规则 |
| `poll` | object | 否 | `null` | 见下 |

`poll` 子字段：

| 字段 | 类型 | 说明 |
|------|------|------|
| `choices` | string[] | 选项文本。**无数量上限、无重复校验、无空串校验** |
| `multiple` | bool | 是否多选 |
| `expiredAfter` | int64 | **相对毫秒**（`now + n ms`）。优先级高于 `expiresAt` |
| `expiresAt` | int64 | **绝对毫秒时间戳** |

### 回复规则（`internal/note/service.go:165`–`207`）

按 Misskey 的顺序校验：存在性 → 是否纯转发 → 可见性 → specified 受众规则。之后还会**收窄可见性**与 `localOnly`：

- 目标不存在 → `NO_SUCH_REPLY_TARGET`
- 目标是纯转发 → `CANNOT_REPLY_TO_A_PURE_RENOTE`
- 目标对回复者不可读 → `CANNOT_REPLY_TO_AN_INVISIBLE_NOTE`
- 目标是 `specified` 而本条不是 `specified` → `CANNOT_REPLY_TO_SPECIFIED_VISIBILITY_NOTE_WITH_EXTENDED_VISIBILITY`
- 目标是 `home` 而本条是 `public` → 本条降为 `home`
- 目标是 `followers` 而本条是 `public`/`home` → 本条降为 `followers`
- 目标 `localOnly` → 本条也 `localOnly`

### 转发规则（`internal/note/service.go:211`–`247`）

- 目标不存在 → `NO_SUCH_RENOTE_TARGET`
- 目标是纯转发（且本条无自有内容）→ `CANNOT_RENOTE_TO_A_PURE_RENOTE`
- 目标 `followers` 且不是自己发的 → `CANNOT_RENOTE_DUE_TO_VISIBILITY`
- 目标 `specified` → `CANNOT_RENOTE_DUE_TO_VISIBILITY`
- 可见性与 `localOnly` 同样只会被收窄/加强，不会放宽

"纯转发"的判定见 `internal/note/renote.go:13`：有 `renoteId`，且 `text`、`cw`、`poll`、`replyId`、`fileIds` **全为空**。带任一自有内容即为引用（quote），可以再回复/再转发。

### 空笔记判定

`text == nil && fileIds 为空 && poll == nil && renoteId == nil` → `NOTE_EMPTY`。注意 `"text": ""`（空串而非缺省）**不算空**，会成功创建一条正文为空的笔记。

### 响应

```json
{ "createdNote": { /* NoteDetailed */ } }
```

### 副作用

- 通知（`internal/httpapi/notes.go:126`）：提及 → 回复目标 → 转发/引用目标，同一个人只保留一条、后到的非 `mention` 原因覆盖 `mention`；`specified` 笔记只通知受众；作者不通知自己；远端收件人不落通知行（交给 ActivityPub）。通知类型：`mention` / `reply` / `renote` / `quote`。
- Streaming：笔记进 `notesStream`，并向回复目标发 `reply`、向转发目标发 `renote`、向被提及者发 `mention`（`internal/httpapi/streaming_publish.go:27`、`50`）。协议细节见 streaming 契约文档。

### 错误

| HTTP | code | 触发 |
|------|------|------|
| 401 | `AUTHENTICATION_FAILED` | 未登录/令牌无效 |
| 400 | `INVALID_PARAM` | body 不是合法 JSON |
| 400 | `INVALID_VISIBILITY` | 可见性取值非法，或 `specified` 缺 `visibleUserIds` |
| 400 | `NOTE_EMPTY` | 无正文、无附件、无投票、无转发目标 |
| 400 | `NOTE_TOO_LONG` | `text` 超 3000 字节 |
| 400 | `NO_SUCH_REPLY_TARGET` | `replyId` 不存在 |
| 400 | `NO_SUCH_RENOTE_TARGET` | `renoteId` 不存在 |
| 400 | `CANNOT_REPLY_TO_A_PURE_RENOTE` | |
| 400 | `CANNOT_REPLY_TO_AN_INVISIBLE_NOTE` | |
| 400 | `CANNOT_REPLY_TO_SPECIFIED_VISIBILITY_NOTE_WITH_EXTENDED_VISIBILITY` | |
| 400 | `CANNOT_RENOTE_TO_A_PURE_RENOTE` | |
| 400 | `CANNOT_RENOTE_DUE_TO_VISIBILITY` | |
| 400 | `CANNOT_CREATE_ALREADY_EXPIRED_POLL` | **实际不可达**：创建路径没有过期校验（`ErrPollExpired` 只在投票时抛出，`internal/note/service.go:635`）。传一个过去的 `expiresAt` 会创建成功 |
| 500 | `INTERNAL_ERROR` | 落库失败 |

### 已知缺口

- **出站联邦投递未接线**：`DeliverNote`（`internal/httpapi/federation_wire.go:148`）在 `internal/`、`cmd/` 中没有任何调用点，`handleNotesCreate` 只做通知与 Streaming。也就是说本地新笔记目前不会推送给远端关注者。
- 附件不校验归属/存在，可以引用别人的 `fileId`。
- 投票：没有选项数上限（Misskey 为 10）、没有重复选项校验、不校验过期时间。
- `text` 长度按**字节**而非字符，与 Misskey 的字符计数不同。
- 缺少 Misskey `paramDef` 里的这些约束，前端表单需自行兜底：
  - `text` 的 `minLength: 1`（Misskey 拒绝空串，这里接受 `""`）；
  - `cw` 的 `maxLength: 100`（这里不限）；
  - `fileIds` 的 `maxItems: 16`（这里不限）；
  - `poll.choices` 的 `minItems: 2`、`maxItems: 10`、每项长度 1–50（这里全不限）；
  - `expiredAfter` 的 `minimum: 1`。
- **不支持的入参**（传了会被静默忽略）：`reactionAcceptance`、`noExtractMentions`、`noExtractHashtags`、`noExtractEmojis`、`channelId`、`files`。其中 `reactionAcceptance` 只能来自数据库既有值，前端无法在创建时设置。
- 自定义表情：`emojis`/`reactionEmojis` 恒为空数组，正文里的 `:shortcode:` 无法据此渲染成图片（需走 `emojis` 元数据端点自行匹配）。
- 不返回 `isSensitive`，也没有 `sensitive` 入参。
- 无频次限制；Misskey 的 `notes/create` 为 300 次/小时、最小间隔 1 秒。

---

## POST /api/notes/show

按 id 读取一条笔记。实现：`internal/httpapi/notes.go:255`。

### 请求字段

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `noteId` | string | 名义必填，实际未校验 | 缺失或空串会直接走到"查不到"分支，返回 `404 NO_SUCH_NOTE`（不是 `INVALID_PARAM`） |

### 响应

直接返回 `NoteDetailed` 对象（**不包裹**在 `createdNote` 之类字段里）。

### 错误

| HTTP | code | 触发 |
|------|------|------|
| 404 | `NO_SUCH_NOTE`（id `24fcbfc6-...`） | 不存在、已软删除，或**对当前访问者不可见**（不区分，避免泄露存在性） |
| 500 | `INTERNAL_ERROR` | 可见性判定或打包失败 |

匿名可读 `public` / `home`；`followers` / `specified` 需带有效令牌。

### 已知缺口

- 与 Misskey 不同，这里**做了可见性检查**（Misskey 的 `notes/show` 没有），前端不能靠它探测私有笔记。
- 缺 `isSensitive`；`clippedCount` 恒 0。
- 坏令牌被当成匿名（见"通用约定 §1"）。

---

## POST /api/notes/delete

软删除自己的笔记。实现：`internal/httpapi/notes.go:313`；服务层 `internal/note/service.go:508`。

### 请求字段

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `noteId` | string | 是 | 缺失按"不存在"处理 |

### 响应

`{}`（HTTP 200）。

### 错误

| HTTP | code | 触发 |
|------|------|------|
| 401 | `AUTHENTICATION_FAILED` | 未登录 |
| 404 | `NO_SUCH_NOTE` | 笔记不存在，**或不是本人所发**（他人笔记同样回答 404） |

删除**不幂等**：第二次删除同一条笔记返回 404（已软删除，`Get` 查不到）。

### 副作用

- 向笔记的 Streaming 通道发 `deleted` 事件，带 `{"deletedAt": "<RFC3339Nano>"}`（`internal/httpapi/streaming_publish.go:89`）。
- 因为事件需要可见性与受众，handler 会**先加载笔记再删**。

### 已知缺口

- Misskey 对"他人笔记"抛 `ACCESS_DENIED`（`fe8d7103-0ea8-4ec3-814d-f8b401dc69e9`），这里统一按 `NO_SUCH_NOTE` 回答；`notes/delete` 也没有 Misskey 的 300 次/小时限流。
- **出站 `Delete` 未接线**：`DeliverDelete`（`internal/httpapi/federation_wire.go:194`）无调用点，远端实例不会收到删除。
- 软删除**不回退计数**：父笔记的 `repliesCount`、被转发笔记的 `renoteCount` 不会减。
- 不清理该笔记上的 reaction / favorite 行（它们仍在库中，只是笔记不可读）。

---

## POST /api/notes/edit

编辑自己的笔记（Firefish 的端点；Misskey 固定版本没有）。实现：`internal/httpapi/note_edit.go:26`；
服务层 `internal/note/edit.go`。

### 请求字段

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `editId` | string | 是 | 缺失返回 400 `NEEDS_EDIT_ID`（id `d697edc8-…`） |
| `text` | string \| null | 否 | 不出现 = 不改；`null`/`""` = 清空；≤ 3000 字符 |
| `cw` | string \| null | 否 | 同上；`null`/`""` = 清空 |
| `visibility` | string | 否 | `public` / `home` / `followers` / `specified`；缺省保持原值 |
| `localOnly` | boolean | 否 | 缺省保持原值 |

`fileIds`、`poll`、`replyId`、`renoteId`、`visibleUserIds` 会被忽略：附件与投票不随编辑变化。

### 响应 200

```json
{ "createdNote": { "id": "…", "text": "…", "cw": null, "visibility": "public", … } }
```

### 错误

| HTTP | code | 触发 |
|------|------|------|
| 401 | `AUTHENTICATION_FAILED` | 未登录 |
| 400 | `NEEDS_EDIT_ID` | 缺 `editId` |
| 404 | `NO_SUCH_NOTE` | 笔记不存在或已删除 |
| 403 | `YOU_ARE_NOT_THE_AUTHOR` | 不是作者 |
| 400 | `INVALID_PARAM` | 文本超长或 visibility 非法 |

### 副作用

- `notes.updated_at = now()`；`tags` 与本地 `mentions` 按新正文重算，`emojis` 清空。
- 写入一条 `note_edits` 记录（保存前的文本/CW/附件 id）。
- **不发 Streaming `updated` 事件，也不投递 ActivityPub `Update`**：关注者看到的仍是旧文本。

### 已知缺口

- 不能编辑附件、投票、回复目标、转发目标与指定可见用户。
- 远端副本不会更新。
- 没有 Misskey 的 300 次/小时限流。

---

## POST /api/notes/history

列出自己笔记的编辑记录。**这是 Cinderfish 自有的端点**：Misskey 固定版本（`c05e844`）的
`endpoints/` 下没有 `notes/history`，所以不计入「Misskey 同名端点」。

### 请求字段

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `noteId` | string | 是 | 缺失返回 400 `INVALID_PARAM` |
| `limit` | integer | 否 | 默认 20，上限 100 |

### 响应 200

`NoteEdit[]`，按 `id` 降序（最新在前）：

```json
[{ "id": "…", "noteId": "…", "text": "保存前的正文", "cw": null,
   "fileIds": [], "createdAt": "2026-11-06T…" }]
```

只有作者能读：他人笔记返回 403 `YOU_ARE_NOT_THE_AUTHOR`，不存在的笔记返回 404 `NO_SUCH_NOTE`。

---

## POST /api/notes/replies

列出回复了某条笔记的笔记。实现：`internal/httpapi/notes.go:370` → `serveNoteList`（`:383`）；查询 `internal/note/timeline.go:300`。

### 请求字段

| 字段 | 类型 | 必填 | 默认 | 说明 |
|------|------|------|------|------|
| `noteId` | string | **是** | — | 缺失 → `400 INVALID_PARAM`（`info` 为字符串 `"noteId is required"`） |
| `limit` | int | 否 | 10 | 上限 100 |
| `sinceId` | string | 否 | `""` | |
| `untilId` | string | 否 | `""` | |

不支持 `sinceDate` / `untilDate`（传了会被静默忽略）；Misskey 的 `notes/replies` 是支持这两个日期游标的。

### 响应

`NoteDetailed[]`，按 id 排序；只给 `sinceId`（无 `untilId`）时升序，否则降序。

**注意**：Misskey 的 `notes/replies` 返回的是 `Note[]`，这里返回的是 **`NoteDetailed[]`**（多出 `reply`/`renote`/`poll`/`myReaction`/`clippedCount`）。多出来的字段可以直接用，但别指望别的列表端点也这样。

### 错误

| HTTP | code | 触发 |
|------|------|------|
| 400 | `INVALID_PARAM` | `noteId` 缺失 |
| 500 | `INTERNAL_ERROR` | 查询失败 |

目标笔记不存在**不报错**，返回 `[]`（与 Misskey 一致，有测试 `TestNotesRepliesUnknownNote` 固定）。

### 已知缺口

- 目标笔记本身不可读时也不报错，只按可见性过滤结果——即私有笔记的公开回复仍会被列出来。
- 没有 `sinceDate`/`untilDate`（Misskey 支持），只能用 id 游标。
- 未过滤被屏蔽/静音用户的回复（这两项功能本身未实现）。

---

## POST /api/notes/renotes

列出转发了某条笔记的笔记（纯转发与引用都包含）。实现：`internal/httpapi/notes.go:376`；查询 `internal/note/timeline.go:307`。

### 请求字段

与 `notes/replies` 完全相同：`noteId`（必填）、`limit`（默认 10，上限 100）、`sinceId`、`untilId`。

### 响应

`NoteDetailed[]`，排序规则同上。

### 错误

| HTTP | code | 触发 |
|------|------|------|
| 400 | `INVALID_PARAM` | `noteId` 缺失 |
| 404 | `NO_SUCH_NOTE`（id `12908022-2e21-46cd-ba6a-3edaf6093f46`） | **目标笔记不存在**。这是与 `notes/replies` 的重要差异 |
| 500 | `INTERNAL_ERROR` | |

### 已知缺口

- 目标存在但不可读时**不报错**，仅按可见性过滤（与 `notes/replies` 相同的宽松行为）。
- 不支持日期游标（Misskey 的 `notes/renotes` 支持 `sinceDate`/`untilDate`）。

---

## POST /api/notes/conversation

返回一条笔记的祖先链（回复关系向上），最近的祖先在前。实现：`internal/httpapi/notes_thread.go:13`；遍历逻辑 `internal/note/conversation.go:21`。

### 请求字段

| 字段 | 类型 | 必填 | 默认 | 说明 |
|------|------|------|------|------|
| `noteId` | string | **是** | — | 缺失 → `400 INVALID_PARAM` |
| `limit` | int | 否 | 10 | 上限 100，计的是**返回条数**（不是遍历深度） |
| `offset` | int | 否 | 0 | 从最近祖先起跳过多少条；负数按 0 处理 |

### 响应

`Note[]`（**非 detail**）：没有 `reply`、`renote`、`poll`、`hasPoll`、`myReaction`、`clippedCount`。字段列表见"共享对象 / Note"。

顺序：`[最近的父帖, 祖父帖, …]`。

### 错误

| HTTP | code | 触发 |
|------|------|------|
| 400 | `INVALID_PARAM` | `noteId` 缺失 |
| 404 | `NO_SUCH_NOTE`（id `24fcbfc6-...`） | 起点笔记不存在/已删除 |
| 500 | `INTERNAL_ERROR` | |

### 行为要点

- 向上走链条时，遇到**不可读或已删除**的祖先就**停止**，不报错（有测试 `TestNotesConversationStopsAtPrivateParent`）。这比 Misskey 更严格（Misskey 不做可见性检查）。
- 有环保护：链条回环时终止。
- `offset` 在 `limit` 之前生效。

### 已知缺口

- 只给祖先，**不给子孙**（子孙走 `notes/children`）。
- 返回的是精简 `Note`，前端若要在会话串里显示投票或"我的回应"，需要再逐条 `notes/show`，或改用 `notes/replies` 的思路自行拼装。
- 不支持 `sinceId`/`untilId`（Misskey 也不支持，这里一致）。

---

## POST /api/notes/children

列出直接回复某条笔记的笔记（只有一层）。实现：`internal/httpapi/notes_thread.go:59`；查询 `internal/note/conversation.go:92`。

### 请求字段

| 字段 | 类型 | 必填 | 默认 | 说明 |
|------|------|------|------|------|
| `noteId` | string | **是** | — | 缺失 → `400 INVALID_PARAM` |
| `limit` | int | 否 | 10 | 上限 100 |
| `sinceId` | string | 否 | `""` | |
| `untilId` | string | 否 | `""` | |
| `sinceDate` | int64 | 否 | `null` | **毫秒时间戳**，比较 `created_at >` |
| `untilDate` | int64 | 否 | `null` | **毫秒时间戳**，比较 `created_at <` |

### 响应

`Note[]`（**非 detail**）。降序（最新在前）；只给 `sinceId` 且无 `untilId` 时升序。

### 错误

| HTTP | code | 触发 |
|------|------|------|
| 400 | `INVALID_PARAM` | `noteId` 缺失 |
| 500 | `INTERNAL_ERROR` | |

**父笔记不存在不报错**，返回 `[]`（handler 不校验父笔记存在性）。

### 已知缺口

- 返回非 detail 对象，同 `notes/conversation`。
- 父笔记不可读时也不报错。
- 只返回**直接**子帖，不做递归。

---

## POST /api/notes/reactions

"谁回应了这条笔记"。实现：`internal/httpapi/notes_reactions_list.go:20`；查询 `internal/note/reaction.go:74`。

### 请求字段

| 字段 | 类型 | 必填 | 默认 | 说明 |
|------|------|------|------|------|
| `noteId` | string | **是** | — | 缺失 → `400 INVALID_PARAM` |
| `type` | string | 否 | `null` | 精确匹配一个表情（如 `"👍"`、`":smile:"`）。给了就只返回该表情的回应 |
| `limit` | int | 否 | 10 | 上限 100 |
| `sinceId` | string | 否 | `""` | 回应行 id 游标 |
| `untilId` | string | 否 | `""` | 回应行 id 游标 |
| `sinceDate` | int64 | 否 | `null` | 毫秒 |
| `untilDate` | int64 | 否 | `null` | 毫秒 |

### 响应

`NoteReaction[]`：`{id, createdAt, type, user}`，见"共享对象"。

排序：降序；只在给了 `sinceId`、且 `untilId` 与 `untilDate` 都为空时升序。

### 错误

| HTTP | code | 触发 |
|------|------|------|
| 400 | `INVALID_PARAM` | `noteId` 缺失 |
| 404 | `NO_SUCH_NOTE`（id `263fff3d-d0e1-4af4-bea7-8408059b451a`） | 笔记不存在**或不可读** |
| 500 | `INTERNAL_ERROR` | |

匿名可读（对公开笔记）。

### 已知缺口

- 回应者的用户行已被删除时，该条**被跳过**而不是返回 `user: null`——因此一页可能不足 `limit` 条，前端"加载更多"不要用"返回条数 == limit"判断是否还有下一页。
- 不返回 `noteId`；不返回总数。
- `type` 不做校验/归一化，传入不存在的表情只会得到空数组。

---

## POST /api/notes/reactions/create

给笔记留一个回应。实现：`internal/httpapi/notes.go:438`；写入 `internal/note/reaction.go:46`。

### 请求字段

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `noteId` | string | 是 | 缺失/不存在都落到 `404 NO_SUCH_NOTE` |
| `reaction` | string | **是** | 空串 → `400 INVALID_PARAM`，`info` 为 `{"reason":"reaction is required"}`。不做格式校验，也不校验是否已注册的自定义表情 |

### 响应

`{}`（HTTP 200）。重复提交同一个表情是幂等的（`ON CONFLICT DO NOTHING`），同样返回 `{}`。

### 错误

| HTTP | code | 触发 |
|------|------|------|
| 401 | `AUTHENTICATION_FAILED` | 未登录 |
| 400 | `INVALID_PARAM` | `reaction` 为空 |
| 404 | `NO_SUCH_NOTE`（id `033d0620-5bfe-4027-965d-980b0c85a3ea`） | 笔记不存在 |
| 400 | `CANNOT_REACT_TO_RENOTE` | 目标是**纯转发**（回应应留给被转发的原帖） |
| 500 | `INTERNAL_ERROR` | |

### 副作用

- Streaming：向该笔记通道发 `reacted`，`{"reaction": "<emoji>", "userId": "<id>", "emoji": null}`（`internal/httpapi/streaming_publish.go:72`）。
- 通知：作者不是自己时创建 `reaction` 类型通知（带 `reaction` 表情字段）。

### 已知缺口

- **幂等而非报错**：Misskey 对"已经留过同样回应"抛 `ALREADY_REACTED`（`71efcf98-86d6-4e2b-b2ad-9d032369366b`），还有 `YOU_HAVE_BEEN_BLOCKED` 与被屏蔽检查；这里重复请求直接返回 `{}`，也没有屏蔽检查。
- **不做可见性检查**：只要知道 noteId，就能对不可读的笔记回应（`notes/reactions` 与 `notes/show` 会拒绝对外展示）。
- 不校验表情是否已注册（`emojis` 表），可以回应任意字符串。
- 通知用 `CreateReturningID` 而非 `CreateForLocal`（对比 `notes/create` 的 `reply`/`renote` 通知），因此给远端作者也会落一行通知。
- 不触发 ActivityPub `Like`/`EmojiReact` 投递。

---

## POST /api/notes/reactions/delete

撤销自己留的回应。实现：`internal/httpapi/notes.go:508`；删除逻辑 `internal/note/reaction.go:62`。

### 请求字段

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `noteId` | string | 是（但未校验） | 空串时等于空操作，仍返回 200 |
| `reaction` | string | 否 | **省略时**先查出自己在该笔记上的回应再删（`MyReaction`），即"撤销我留的那个" |

### 响应

`{}`（HTTP 200）。不存在、重复删除、自己本来没回应——全都返回 200（幂等）。

### 错误

| HTTP | code | 触发 |
|------|------|------|
| 401 | `AUTHENTICATION_FAILED` | 未登录 |
| 500 | `INTERNAL_ERROR` | 数据库失败 |

**没有 `NO_SUCH_NOTE`**：即使笔记不存在也返回 200。

### 副作用

只有真的删掉了一个具名表情时才发 Streaming `unreacted` 事件；`emoji` 为空（本来就没回应）时不发，避免幻影事件。

### 已知缺口

- 不校验笔记可见性。
- 签名与 Misskey 不同：Misskey 的参数表**只有 `noteId`**（一次只能撤"自己那条"回应），在自己没有回应时抛 `NOT_REACTED`（`92f4426d-4196-4125-aa5b-02943e2ec8fc`）；这里额外接受可选的 `reaction`（便于同一笔记存在多条不同表情时精确撤销），并且任何情况都返回 200。
- 无 ActivityPub `Undo(Like)` 投递。

---

## POST /api/notes/favorites/create

收藏一条笔记。实现：`internal/httpapi/notes_favorites.go:16`；`internal/note/favorite.go:41`。

### 请求字段

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `noteId` | string | **是** | 缺失/空串 → `400 INVALID_PARAM`；不存在或已删除 → `404 NO_SUCH_NOTE` |

### 响应

`NoteFavorite`：

```json
{ "id": "<favorite-id>", "createdAt": "2026-11-04T...", "noteId": "<note-id>", "note": { /* Note，非 detail */ } }
```

重复收藏（同一 user + note）返回**同一行**的 `id`，不报错。

### 错误

| HTTP | code | 触发 |
|------|------|------|
| 401 | `AUTHENTICATION_FAILED` | 未登录 |
| 400 | `INVALID_PARAM` | `noteId` 缺失 |
| 404 | `NO_SUCH_NOTE`（id `24fcbfc6-...`） | 笔记不存在或已软删除 |
| 500 | `INTERNAL_ERROR` | |

### 已知缺口

- **无限流**：Misskey 对 `notes/favorites/create` 限 20 次/小时，这里只留了 TODO（`internal/httpapi/notes_favorites.go:13`）。Misskey 还要求 `write:favorites` scope，并在重复收藏时抛 `ALREADY_FAVORITED`（`a402c12b-34dd-41d2-97d8-4d2ffd96a1a6`）；这里重复收藏直接返回已有行。
- **不校验可见性**：知道 noteId 就能收藏不可读的笔记。
- `note` 是精简 `Note`，没有 `poll`/`myReaction`。
- 不触发任何 Streaming 事件或通知。

---

## POST /api/notes/favorites/delete

取消收藏。实现：`internal/httpapi/notes_favorites.go:71`；`internal/note/favorite.go:74`。

### 请求字段

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `noteId` | string | **是** | 缺失/空串 → `400 INVALID_PARAM` |

### 响应

`{}`（HTTP 200）。没收藏过也返回 200（幂等，有测试固定）。

### 错误

| HTTP | code | 触发 |
|------|------|------|
| 401 | `AUTHENTICATION_FAILED` | 未登录 |
| 400 | `INVALID_PARAM` | `noteId` 缺失 |
| 500 | `INTERNAL_ERROR` | |

### 已知缺口

- **比 Misskey 宽松**：Misskey 在笔记不存在时抛 `NO_SUCH_NOTE`（`80848a2c-398f-4343-baa9-df1d57696c56`），在自己没收藏时抛 `NOT_FAVORITED`（`b625fc69-635e-45e9-86f4-dbefbef35af5`）；这里两种都静默成功。前端不能靠错误码判断"是否真的取消掉了"。

---

## POST /api/i/favorites

当前登录者的收藏列表。实现：`internal/httpapi/notes_favorites.go:106`；`internal/note/favorite.go:95`。

### 请求字段

| 字段 | 类型 | 必填 | 默认 | 说明 |
|------|------|------|------|------|
| `limit` | int | 否 | 10 | 上限 100 |
| `sinceId` | string | 否 | `""` | **favorite 行 id** 游标 |
| `untilId` | string | 否 | `""` | favorite 行 id 游标 |
| `sinceDate` | int64 | 否 | `null` | 毫秒，比较 favorite 的 `created_at` |
| `untilDate` | int64 | 否 | `null` | 毫秒 |

### 响应

`NoteFavorite[]`（`{id, createdAt, noteId, note}`），`note` 为精简 `Note`。

排序：降序；只在给了 `sinceId`、且 `untilId` 与 `untilDate` 都为空时升序。

### 错误

| HTTP | code | 触发 |
|------|------|------|
| 401 | `AUTHENTICATION_FAILED` | 未登录 |
| 400 | `INVALID_PARAM` | body 非法 JSON（参数本身都可不传） |
| 500 | `INTERNAL_ERROR` | |

### 已知缺口

- 已被删除的笔记对应的收藏会被**跳过**（不是返回 `note: null`），因此一页可能少于 `limit` 条；`untilId` 分页仍然安全。
- `note` 非 detail：收藏页若要显示投票，需要额外请求。
- 不返回总数。

---

## POST /api/notes/polls/vote

给笔记的投票投一票。实现：`internal/httpapi/notes_polls.go:23`；`internal/note/service.go:626`。

### 请求字段

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `noteId` | string | **是** | 空/缺失 → `400 INVALID_PARAM`（`info` 为 `"noteId and choice are required"`） |
| `choice` | int | **是** | **0 下标**；`0` 是合法值（用指针区分"未传"与"0"）。越界 → `400 INVALID_PARAM`（`info` 为 `"choice is out of range"`） |

### 响应

`{}`（HTTP 200）。**不返回**新的票数——前端需重新 `notes/show`（与 Misskey 一致）。

### 错误

| HTTP | code | id | 触发 |
|------|------|-----|------|
| 401 | `AUTHENTICATION_FAILED` | `b0a7f5f8-...` | 未登录 |
| 400 | `INVALID_PARAM` | `3d81ceae-...` | `noteId`/`choice` 缺失，或 `choice` 越界 |
| 404 | `NO_SUCH_NOTE` | `ecafbd2e-c283-4d6d-aecb-1a0a33b75396` | 笔记不存在**或对当前用户不可读** |
| 400 | `NO_POLL` | `5f979967-52d9-4314-a911-1c673727f92f` | 笔记没有投票 |
| 400 | `ALREADY_VOTED` | `0963fc77-efac-419b-9424-b391608dc6d8` | 单选重复投票，或多选重复选同一项 |
| 400 | `ALREADY_EXPIRED` | `1022a357-b085-4054-9083-8f8de358337e` | 投票已过期 |
| 500 | `INTERNAL_ERROR` | | |

### 投票语义

- **单选**：第一次投票成功；**第二次（换不换选项都）返回 `ALREADY_VOTED`**——不支持改票。落库时会先删旧行再插，但前置检查已经挡住了第二次。
- **多选**：每个不同选项各算一票；同一选项重复投 → `ALREADY_VOTED`。
- 过期判定用服务器当前时间与 `polls.expires_at` 比较。

### 已知缺口

- 投票**不联邦**：`Question`/`Answer` 活动未实现（见 `docs/research/2026-11-04-misskey-api-coverage.md` §6），远端笔记的投票只在本地记票。
- `choice` 越界报 `INVALID_PARAM`（`info` 为字符串 `"choice is out of range"`）；Misskey 有专用错误 `INVALID_CHOICE`（`e0cc9a04-f2e8-41e4-a5f1-4127293260cc`），也有 `YOU_HAVE_BEEN_BLOCKED` 检查与 `write:votes` scope，这里都没有。
- 不能改票，也不能撤票（无 `notes/polls/vote` 的撤销入口）。
- 投票创建侧没有"选项数 ≥ 2"、"选项数 ≤ 10"、"选项非空"、"过期时间在未来"的校验（见 `notes/create`），前端表单需自行约束。
- 创建者能否给自己的投票投票：代码**没有禁止**（`Vote` 只查重复与过期）。

---

## 与 Misskey 的差异汇总（本命名空间）

| 项目 | Misskey | Cinderfish 现状 |
|------|---------|-----------------|
| `notes/show` 可见性 | 不检查 | 检查，不可读按 404 处理 |
| `notes/replies`、`notes/renotes` 返回类型 | `Note[]` | `NoteDetailed[]`（字段超集） |
| `notes/conversation`、`notes/children` 返回类型 | `Note[]` | `Note[]`（一致） |
| `notes/replies`、`notes/renotes` 日期游标 | 支持 `sinceDate`/`untilDate` | 静默忽略 |
| `notes/conversation` 可见性 | 不检查 | 逐层检查，遇到不可读即截断 |
| `notes/children`、`notes/replies` 目标不存在 | 空数组 | 一致 |
| `notes/renotes` 目标不存在 | `NO_SUCH_NOTE` | 一致（同 id） |
| `notes/delete` 删他人笔记 | `ACCESS_DENIED` | `NO_SUCH_NOTE` |
| `notes/edit` 可编辑字段 | 正文、CW、可见性、附件、投票、回复/转发目标 | 只有正文、CW、可见性、localOnly；附件与投票不变 |
| `notes/edit` 副作用 | Streaming `updated` + ActivityPub `Update` | 只改本地行并记录历史 |
| `notes/history` | Misskey 固定版本无此端点 | Cinderfish 自有，仅作者可读 |
| `notes/reactions/create` 重复回应 | `ALREADY_REACTED` | 幂等，返回 `{}` |
| `notes/reactions/delete` 参数与失败 | 仅 `noteId`；无回应抛 `NOT_REACTED`；204 | 可选 `reaction`；恒 200 `{}` |
| `notes/favorites/create` 重复收藏 | `ALREADY_FAVORITED` | 幂等，返回已有行 |
| `notes/favorites/create` 限流 | 20 次/小时 | 无限流 |
| `notes/favorites/delete` 失败语义 | 未收藏/笔记不存在分别抛 `NOT_FAVORITED` / `NO_SUCH_NOTE` | 一律 200 `{}` |
| `notes/polls/vote` 越界选项 | `INVALID_CHOICE` | `INVALID_PARAM` |
| `notes/create` 空串正文 | `minLength: 1`，拒绝 | 接受，创建空正文笔记 |
| `notes/create` 附件上限 | 16 | 无 |
| `notes/create` 投票选项约束 | 2–10 项，每项 1–50 字 | 无 |
| `notes/create` 可写入参 | 含 `reactionAcceptance`、`noExtractMentions/Hashtags/Emojis`、`channelId` | 静默忽略 |
| `Note.isSensitive` | 有 | 未返回 |
| `Note.emojis` | `{shortcode: url}` 对象 | 恒为 `[]`（数组） |
| `poll.expiresAt`（响应） | ISO 8601 字符串 | **毫秒数字**（创建入参两边都是毫秒） |
| `poll.votesCount` | 无此字段（客户端自算） | 额外返回总票数 |
| 笔记长度上限 | 3000 字符 | 3000 **字节** |
| 出站 `Create`/`Delete` 投递 | 有 | 未接线（`DeliverNote`/`DeliverDelete` 无调用点） |
| 投票联邦（`Question`） | 有 | 未实现 |
| 令牌 scope | `write:notes` / `write:reactions` / `write:favorites` / `write:votes` | 无 scope 概念，登录即全部权限 |
| 用非 POST 方法访问 | 未核实 | 404 `NO_SUCH_ENDPOINT`（因兜底 `/` 路由），不是 405 |

## 前端实现建议（基于以上事实）

1. 列表端点优先用 `notes/replies`（有 detail 字段），会话串用 `notes/children` + `notes/conversation`，并接受它们缺 `poll`/`myReaction` 的事实。
2. 判断"是否还有下一页"不要用 `len(page) === limit`：`notes/reactions`、`i/favorites` 会因丢弃行而变短。改用返回的最后一条 id 作为下一次的 `untilId`，为空数组时才停止。
3. 用 `untilId` 分页（降序）而不是 `sinceId`，除非确实在"向上加载新内容"（此时结果会是升序，需要自行 reverse）。
4. 上传附件后把 `fileIds` 直接塞进 `notes/create`；`notes/create` 不会替你校验归属，前端要在 `drive/files/create` 成功后自己保存 id。
5. 收藏/回应按钮的乐观更新可以直接做：两者都是幂等写入，重复请求不会报错。
6. 投票提交后必须重新拉取笔记才能看到新票数（响应是空对象）。
7. 时间解析对两种格式都要兼容：笔记/收藏/回应是 RFC3339Nano，DriveFile 是固定毫秒。

## 2026-11-06 增补：搜索（notes/search、notes/search-by-tag、users/search）

### POST /api/notes/search

匿名可用；带凭据时额外套用静音、屏蔽与实例静音过滤。

| 参数 | 类型 | 说明 |
|------|------|------|
| `query` | string，必填 | 正文包含匹配（`ILIKE`），大小写不敏感；`%`、`_`、`\` 会被转义 |
| `limit` | int，默认 10，上限 100 | |
| `offset` | int，默认 0 | 只有不带游标时生效 |
| `sinceId` / `untilId` | id | 笔记 id 游标；与时间线同一套升/降序规则 |
| `host` | string \| null | 缺省=全部；`null`=仅本地；字符串=指定实例 |
| `userId` | id \| null | 只看某个作者的帖子 |
| `withFiles` | bool | 只返回带附件的帖子 |

返回 `Note[]`（与时间线同形的 NoteDetailed）。错误：`query` 为空 → 400。

### POST /api/notes/search-by-tag

| 参数 | 类型 | 说明 |
|------|------|------|
| `tag` | string，必填 | 标签名（不带 `#`）；大小写不敏感 |
| `limit` / `sinceId` / `untilId` / `withFiles` | | 同上 |

只实现上游的 `tag` 单标签形式；`query: string[][]` 布尔组合未实现。
返回 `Note[]`，可见性与静音/屏蔽规则同时间线。

### POST /api/users/search

| 参数 | 类型 | 说明 |
|------|------|------|
| `query` | string，必填 | 匹配 `username`、昵称、简介（`ILIKE` 包含） |
| `origin` | `combined` \| `local` \| `remote` | 缺省视为 combined |
| `host` | string \| null | 缺省=全部；`null`=本地 |
| `limit` / `offset` | | 默认 10 / 0 |
| `sinceId` / `untilId` | id | 本实现额外支持，供前端游标分页 |

返回 `UserDetailed[]`，排除冻结账号；带凭据时排除双向屏蔽关系；每条结果带
`notesCount` / `followersCount` / `followingCount`。`origin` 非法 → 400。
