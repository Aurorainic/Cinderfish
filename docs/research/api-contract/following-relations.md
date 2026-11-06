# API 对接契约：following-relations

本文档记录 Cinderfish 后端「关注关系」命名空间的真实 HTTP 行为，供新前端（Firefish 风格客户端）直接对接。所有结论来自通读 handler 与存储层代码，不引述 Misskey 文档；无法从代码确认的地方明确写「未返回」或「未实现」。

范围（路由注册见 `internal/httpapi/server.go:130`–`internal/httpapi/server.go:136`）：

- `POST /api/following/create`
- `POST /api/following/delete`
- `POST /api/following/requests/list`
- `POST /api/following/requests/sent`
- `POST /api/following/requests/accept`
- `POST /api/following/requests/reject`
- `POST /api/following/requests/cancel`

以上七个端点**没有别名**（未注册 `following/unfollow`、`users/follow` 之类路径）。

相邻但**不属于**本文档的部分：`users/followers` / `users/following`（关注列表读取，见 auth-users 契约）、`GET /streaming` 协议本身（见 streaming 契约）、ActivityPub 入站处理（见 meta-federation 契约）。本文只描述本命名空间如何触发与依赖它们。

本文档的验证方式：`CINDERFISH_TEST_DATABASE_URL=… go test ./internal/httpapi -run TestFollowing -count=1 -v`，12 个用例全部通过（2026-11-04 运行）。下文标注「实测」的结论有对应用例覆盖，标注「代码」的结论由阅读代码确定、无测试直接覆盖。

## 通用约定

### 传输与鉴权

- 全部七个端点都是 `POST`，请求体为 JSON，`Content-Type` **不做校验**（不发送也能工作）。
- 所有七个端点都**必须携带凭据**，匿名调用一律 401。仓储层错误（`internal/httpapi/following.go:34`、`internal/httpapi/following_requests.go:57`）统一走 `writeAuthenticationError`。
- Token 读取顺序（`internal/httpapi/auth.go:30`–`internal/httpapi/auth.go:46`）：
  1. 请求体的 `i` 字段（string）优先；
  2. 否则读 `Authorization: Bearer <token>` 头（大小写不敏感地匹配 `bearer ` 前缀）。
  - Misskey 客户端习惯把 token 放在 body 的 `i` 里，两种写法都支持；两者同时存在时 body 覆盖 header（body 非空则完全不看 header）。
- 凭据校验失败、token 不存在、账号被停用（`is_suspended`）三种情况返回**同一个** 401：`AUTHENTICATION_FAILED`，并带 `WWW-Authenticate: Bearer realm="Misskey"` 头（`internal/httpapi/server.go:304`）。
- 本命名空间**没有权限位检查**：不要求管理员，不检查 token 的 scope（token 表里也没有 scope 概念）。任何能解析成功的 token 都可以关注/取关/审批。

### 请求体解码

`decodeOptionalJSON`（`internal/httpapi/ping.go:23`）的行为对七个端点一致：

- 最多读取 8 MiB。
- 空 body 视为「所有字段缺省」，不报错。
- JSON 语法错误、或字段类型不匹配（例如 `"limit": "10"` 传给 `int`）→ 400 `INVALID_PARAM`，`info` 为 `{"reason":"body is not valid JSON"}`。
- 未知字段被静默忽略（不报 400）。
- 解码后 body 会被复原，handler 再调用 `bodyAsMap` 取回通用 map 以读取 `i`。

### 错误响应

失败响应统一为 Misskey 信封（`internal/httpapi/server.go:269`）：

```json
{
  "error": {
    "message": "You are already following that user.",
    "code": "ALREADY_FOLLOWING",
    "id": "35387507-38c7-4cb9-9197-300b93783fa0",
    "kind": "client"
  }
}
```

`kind` 恒存在，取值 `client` / `server` / `permission`。`info` 仅在部分错误上出现（本命名空间只有 `INVALID_PARAM` 带）。本命名空间会出现的错误码全集：

| code | HTTP | kind | 错误 id | 出现端点 | 说明 |
|------|------|------|---------|----------|------|
| `INVALID_PARAM` | 400 | client | `3d81ceae-475f-4600-b2a8-2bc116157532` | 全部 | body 不是合法 JSON，或 `userId` 缺失；`info` 为字符串 `"userId is required"` 或对象 `{"reason": …}` |
| `AUTHENTICATION_FAILED` | 401 | client | `b0a7f5f8-dc2f-4171-b91f-de88ad238e14` | 全部 | 无凭据 / 凭据无效 / 账号停用 |
| `NO_SUCH_USER` | 404 | client | 见下表（**每个端点不同**） | 全部 | `userId` 指向的账号不存在 |
| `FOLLOWEE_IS_YOURSELF` | 400 | client | `26fbe7bb-a331-4857-af17-205b426669a9` | create、delete | `userId` 等于调用者自己 |
| `ALREADY_FOLLOWING` | 400 | client | `35387507-38c7-4cb9-9197-300b93783fa0` | create | 已关注，**或**已有一条待审的发出请求 |
| `NOT_FOLLOWING` | 400 | client | `5dbf82f5-c92b-40b1-87d1-6c8c0741fd09` | delete | 不存在这条关注关系（注意：待审请求不算关注） |
| `NO_FOLLOW_REQUEST` | 400 | client | `bcde4f8b-0913-4614-8881-614e522fb041` | accept | 没有可批准的待审请求 |
| `FOLLOW_REQUEST_NOT_FOUND` | 400 | client | `089b125b-d338-482a-9a09-e2622ac9f8d4` | cancel | 没有可撤回的待审请求 |
| `INTERNAL_ERROR` | 500 | server | `5d37dbcb-891e-41ca-a3d6-e690c97775ac` | 全部 | 数据库写入失败等 |

`NO_SUCH_USER` 的错误 id 逐端点不同（代码里为每个端点保留了 Misskey 原始 id，见 `internal/httpapi/following.go:14`、`internal/httpapi/following_requests.go:17`）：

| 端点 | `NO_SUCH_USER` 的 id |
|------|----------------------|
| `following/create` | `6cc579cc-885d-43d8-95c2-b8c7fc963280` |
| `following/delete` | `5b12c78d-2b28-4dca-99d2-f56139b42ff8` |
| `following/requests/accept` | `66ce1645-d66c-46bb-8b79-96739af885bd` |
| `following/requests/reject` | `abc2ffa6-25b2-4380-ba99-321ff3a94555` |
| `following/requests/cancel` | `4e68c551-fc4c-4e46-bb41-7d4a37bf9dab` |

前端**不应当**用 `id` 做分支判断，用 `code` 即可。`id` 只用于与服务端日志、Misskey 原始实现对齐。

补充两个端点外行为：

- 未注册的路径（例如 `POST /api/following/list`）命中兜底 handler，返回 404 + `NO_SUCH_ENDPOINT`（`internal/httpapi/server.go:327`），message 形如 `No such endpoint: POST /api/following/list`。
- 已注册路径用了错误方法（例如 `GET /api/following/create`）时，因为 `server.go:223` 注册了无方法的兜底模式 `/`，请求同样落到 `notFoundHandler`，返回 **404 + `NO_SUCH_ENDPOINT`**（不是 405、也不是纯文本）。已用最小 `ServeMux` 复现验证：`GET /api/following/create` → 404 JSON，`NO_SUCH_ENDPOINT`。前端可以统一按 JSON 信封处理。

### 分页游标语义

适用于 `following/requests/list` 与 `following/requests/sent`（`internal/httpapi/following_requests.go:27`、`internal/federation/store.go:373`）。

| 参数 | 类型 | 默认 | 说明 |
|------|------|------|------|
| `limit` | int | 10 | `<= 0` 时按 10 处理；`> 100` 时截断为 100 |
| `sinceId` | string | 无 | 对行 `id` 做 `id > :sinceId`（**字符串比较**，不是数值比较） |
| `untilId` | string | 无 | 对行 `id` 做 `id < :untilId` |
| `sinceDate` | int64 | 无 | **毫秒**时间戳，对 `created_at` 做 `created_at > :sinceDate`；用 `time.UnixMilli` 转换（传秒会得到 1970 年附近的时间，等于不设边界） |
| `untilDate` | int64 | 无 | 同上，方向相反 `<` |

边界都是**排他**的（严格大于/小于），不会重复返回游标那一行。

排序规则照抄 Misskey `makePaginationQuery`：默认按 `id DESC`（最新在前）；只有在**仅给下界**时翻转为 `id ASC`（向上翻页）。判定条件（`internal/federation/store.go:411`）为：

- `sinceId` 有、`untilId` 无 → 升序；
- 两个 id 游标都无、`sinceDate` 有、`untilDate` 无 → 升序；
- 其余情况（包括只给 `untilId`、或日期上下界都给）→ 降序。

`id` 由 `internal/idgen` 生成：10 位 base36 毫秒时间戳 + 8 位随机（共 18 字符），字符串序即时间序，因此 `sinceId`/`untilId` 可以安全用于时间翻页。注意 `sinceDate`/`untilDate` 绑定的是 `created_at` 列，与 id 内嵌时间戳在极端情况下（同一毫秒内插入多行）可能不完全一致。

两个列表端点**直接返回 JSON 数组**，没有 `{items:[…]}` 之类的外层信封，也没有总数。空结果返回 `[]`（初始化时 `make([]any, 0, n)`），不会是 `null`。

### UserLite 对象

`packUserLite`（`internal/httpapi/users_pack.go:12`）是本命名空间里所有「用户」字段的形状。字段**全部恒存在**，缺值以 `null` / 空数组 / `false` 出现，不会省略。

| 字段 | 类型 | 取值与说明 |
|------|------|-----------|
| `id` | string | 用户 id（18 字符 idgen id） |
| `name` | string \| null | 仅当 `user_profiles.name` 非空才有值，否则 `null` |
| `username` | string | 不含 `@`，不含 host |
| `host` | string \| null | 本地用户为 `null`；远端用户为对端域名（取自 `users.host`） |
| `avatarUrl` | string \| null | 仅当 `user_profiles.avatar_url` 非空才有值，否则 `null` |
| `avatarBlurhash` | null | 恒为 `null`（未实现） |
| `avatarDecorations` | array | 恒为 `[]` |
| `isBot` | bool | 恒为 `false`（`users.is_bot` 列存在，但本打包器不读取） |
| `isCat` | bool | 恒为 `false` |
| `requireSigninToViewContents` | bool | 恒为 `false` |
| `emojis` | array | 恒为 `[]`（因此 `name` 里的 `:shortcode:` 需要前端自行渲染） |
| `onlineStatus` | string | 恒为 `"unknown"` |
| `badgeRoles` | array | 恒为 `[]` |
| `instance` | null | 恒为 `null` |

两个重要陷阱：

1. `packUserLite` 的第二个参数 `host` **在函数体里从未被使用**（`internal/httpapi/users_pack.go:12`–`internal/httpapi/users_pack.go:41`）。调用方传 `s.cfg.Domain` 或 `""` 都无所谓，`host` 永远来自 `users.host`。
2. 是否填充 `name`/`avatarUrl` 取决于调用方是否传入 profile：
   - `following/create`、`following/delete` 传 `nil` → **`name` 与 `avatarUrl` 恒为 `null`**，即使目标用户有昵称和头像（`internal/httpapi/following.go:80`、`internal/httpapi/following.go:136`）；
   - `following/requests/cancel`、以及请求列表里的 `follower`/`followee` 会加载 profile → 有值（`internal/httpapi/following_requests.go:118`、`internal/httpapi/following_requests.go:202`）。

因此前端在 `following/create`/`delete` 成功后**不要**用响应里的用户对象整体替换本地缓存，应保留已有的 name/avatar，或改用 `users/show` 重新拉取。

### FollowRequest 对象

`packFollowRequest`（`internal/httpapi/following_requests.go:107`）返回的待审关注请求形状：

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | `follow_requests.id`，18 字符 idgen id，用作分页游标 |
| `follower` | UserLite | 发起关注的一方（带 profile，`name`/`avatarUrl` 可用） |
| `followee` | UserLite | 被关注的一方（带 profile） |

**没有 `createdAt`**（代码注释明确对齐 Misskey 的 `FollowRequestEntityService`，`internal/httpapi/following_requests.go:105`）。这意味着前端无法在列表里显示「请求于何时发出」，只能用 `id` 或 `untilDate`/`sinceDate` 做翻页，不能显示时间文本。有测试固定此行为（`internal/httpapi/following_test.go:563`）。

如果列表里某一方的用户行在打包过程中读不出来（例如账号行已删除），该端点整体返回 500 `INTERNAL_ERROR`，而不是跳过该行（`internal/httpapi/following_requests.go:93`）。

### 实时事件（Streaming）

本命名空间会在共享 bus 上发事件（`internal/httpapi/streaming_publish.go:106`–`internal/httpapi/streaming_publish.go:120`），前端通过 `GET /streaming` 的 `main` 频道接收：

| 事件 | 发往 | payload | 触发点 |
|------|------|---------|--------|
| `follow` | 被关注者（followee）的 main | `{"id": <followerId>}` | `following/create`（本地目标）、`following/requests/accept` |
| `followed` | 关注者（follower）的 main | `{"id": <followeeId>}` | 同上 |
| `unfollow` | 被取关者（followee）的 main | `{"id": <followerId>}` | `following/delete` |

注意（代码确认，见下一节「已知缺口」）：**入站 ActivityPub 的 `Follow` / `Accept` / `Undo(Follow)` 不会发出这些事件**，因为 `federation.Inbox` 的 `OnFollow` / `OnUnfollow` 回调在服务端从未被赋值（`internal/federation/inbox.go:122`、`internal/federation/inbox.go:198`，全仓库仅 inbox 内部引用）。

## 端点总览

| 端点 | 用途 | 认证 | 请求参数（`*` 为必填） | 成功响应 | 主要错误码 |
|------|------|------|------------------------|----------|-----------|
| `POST /api/following/create` | 关注一个用户 | 必须 | `userId`* | UserLite（followee，`name`/`avatarUrl` 为 null） | `INVALID_PARAM`、`FOLLOWEE_IS_YOURSELF`、`NO_SUCH_USER`、`ALREADY_FOLLOWING` |
| `POST /api/following/delete` | 取关一个用户 | 必须 | `userId`* | UserLite（被取关者，`name`/`avatarUrl` 为 null） | `INVALID_PARAM`、`FOLLOWEE_IS_YOURSELF`、`NO_SUCH_USER`、`NOT_FOLLOWING` |
| `POST /api/following/requests/list` | 别人发给我、待我审批的请求 | 必须 | `limit`、`sinceId`、`untilId`、`sinceDate`、`untilDate` | FollowRequest 数组 | `INVALID_PARAM` |
| `POST /api/following/requests/sent` | 我发出的待审请求 | 必须 | 同上 | FollowRequest 数组 | `INVALID_PARAM` |
| `POST /api/following/requests/accept` | 批准收到的请求 | 必须 | `userId`*（请求方） | `{}` | `INVALID_PARAM`、`NO_SUCH_USER`、`NO_FOLLOW_REQUEST` |
| `POST /api/following/requests/reject` | 拒绝收到的请求 | 必须 | `userId`*（请求方） | `{}` | `INVALID_PARAM`、`NO_SUCH_USER` |
| `POST /api/following/requests/cancel` | 撤回我发出的请求 | 必须 | `userId`*（被关注方） | UserLite（followee，**带** `name`/`avatarUrl`） | `INVALID_PARAM`、`NO_SUCH_USER`、`FOLLOW_REQUEST_NOT_FOUND` |

所有端点的成功状态码都是 **200**（没有 201/204）。

## POST /api/following/create

关注一个用户。本地目标立即建立关系；远端目标发一条签名 `Follow` 并先记一条待审请求，等对端 `Accept` 到达才转为正式关注。

- 代码：`internal/httpapi/following.go:25`（handler）、`internal/httpapi/following.go:142`（远端投递）
- 路由：`internal/httpapi/server.go:130`

### 请求参数

| 字段 | 类型 | 必填 | 默认 | 校验与说明 |
|------|------|------|------|-----------|
| `userId` | string | 是 | — | 目标用户 id。空字符串或缺失 → 400 `INVALID_PARAM`，`info` 为 `"userId is required"` |
| `i` | string | 否 | — | 访问 token（也接受 `Authorization: Bearer`） |

不支持 `withReplies`、`notify`、`mute` 等 Misskey 参数——传了会被静默忽略。

校验顺序（决定了同时出错时返回哪个错误）：解码（400）→ 认证（401）→ `userId` 非空（400）→ 不能关注自己（400）→ 用户存在（404）→ 未关注且无待审请求（400）。因此**匿名且缺 `userId` 时返回 401，不是 400**。

### 成功响应

200，body 是目标用户的 UserLite（字段全集见「通用约定 / UserLite」）：

```json
{
  "id": "9k2m4p6q8s0u2w4y6a",
  "name": null,
  "username": "bob",
  "host": null,
  "avatarUrl": null,
  "avatarBlurhash": null,
  "avatarDecorations": [],
  "isBot": false,
  "isCat": false,
  "requireSigninToViewContents": false,
  "emojis": [],
  "onlineStatus": "unknown",
  "badgeRoles": [],
  "instance": null
}
```

`name` / `avatarUrl` 恒为 `null`（传了 `nil` profile，见上文陷阱 2）。远端目标时 `host` 是对端域名。

### 错误

| HTTP | code | 条件 |
|------|------|------|
| 400 | `INVALID_PARAM` | body 非法 JSON；`userId` 缺失/空 |
| 401 | `AUTHENTICATION_FAILED` | 无 token、token 无效、账号停用 |
| 400 | `FOLLOWEE_IS_YOURSELF` | `userId` 等于调用者 id |
| 404 | `NO_SUCH_USER` | `users` 表里没有该 id |
| 400 | `ALREADY_FOLLOWING` | `followings` 已有该行，**或** `follow_requests` 已有该行（`internal/httpapi/following.go:58`） |
| 500 | `INTERNAL_ERROR` | 本地关注写入失败 |

### 行为细节

- **本地目标**（`users.host IS NULL`）：直接插入 `followings`（`created_via = 'activitypub'`，`internal/federation/store.go:495`），随后重算双方的 `followers_count` / `following_count`，并发 `follow`（给目标）与 `followed`（给自己）两个 Streaming 事件（`internal/httpapi/following.go:70`）。
  - 注意：`followings` 插入带 `ON CONFLICT DO NOTHING`，而前面的 `HasFollowing` 检查把数据库错误吞成 `false`（`internal/federation/store.go:555`），所以极端情况下重复关注可能不报错地返回 200。
- **远端目标**：`sendRemoteFollow`（`internal/httpapi/following.go:142`）依次做：
  1. `federationPolicy.CheckAllowed(host)` 检查联邦模式；
  2. 从本地已存的远端 actor 行解析 actor（`UserByID`，不做远程抓取）；
  3. 用 `<本机>/users/<me>#follows/<目标本地行 id>` 作为 `Follow` 的 id，`object` 为对端 actor URI（`internal/httpapi/following.go:159`）；
  4. 投递到对端 `sharedInbox`，没有则 `inbox`；
  5. 成功后写一条 `follow_requests` 行（`internal/httpapi/following.go:179`），于是该请求立刻出现在 `following/requests/sent` 里，并且可以 `cancel`。
  - 对端 `Accept` 到达时，`inbox.accept` 会插入 `followings` 并删掉这条待审行（`internal/federation/inbox.go:152`）。
- **远端目标不会区分对端是否锁定账号**：一律先发 `Follow` 再记待审行；也**不检查自己是否被对方 block**（`blocks` 表在关注路径上完全未被读取）。
- **关注通知自第四十七波起落库**：本地关注与被接受的入站关注产生 `follow` 行，锁推账号产生
  `receiveFollowRequest` 行，并推 Streaming 通知事件。仍不产生 `followRequestAccepted`
  （Misskey 也没有这个类型）；远端请求方感知接受结果依旧靠对端 `Accept` 或 `followed` 事件。

### 已知限制

1. **远端投递失败仍返回 200**：策略拒绝（`internal/httpapi/following.go:146`）、actor 解析失败、无 inbox、入队失败都只写日志然后 `return`，handler 照样返回 200 和目标 UserLite（`internal/httpapi/following.go:77`–`internal/httpapi/following.go:80`）。此时既没有真正关注，也没有待审行；客户端看到的是「成功」。要判断真实结果只能再查 `users/show` 的 `isFollowing` / `hasPendingFollowRequestFromYou`，或查 `following/requests/sent`。
2. **联邦模式为 `none` / 主机不在 allowlist 时同样返回 200**（同上，`CheckAllowed` 的错误被吞）。前端应在 `meta` 里读 `federation` 模式并自行隐藏远端关注按钮。
3. **本地锁定账号不会产生待审请求**：`followings` 直接写入，没有 `is_locked` 判断（`internal/httpapi/following.go:70`）。同时 `is_locked` 也无法通过 API 设置（`i/update` 不接受该字段），只有直接改库才可能打开，`users/show` 的 `isLocked` 更是恒为 `false`（`internal/httpapi/users_pack.go:57`）。因此「关注需审批」在产品层面目前不可达。
4. **不检查目标是否停用**：`is_suspended` 的账号同样可以被关注。
5. **响应缺 UserDetailed 字段**：关系状态（`isFollowing`、`hasPendingFollowRequestFromYou` …）、`followersCount` 等都不在响应里，需要另调 `users/show`（带 token 时该端点会填充这四个关系标志，`internal/httpapi/account.go:276`–`internal/httpapi/account.go:281`）。
6. **`following/create` 没有实现 Misskey 的 `following/requests/*` 语义差别提醒**：对远端目标，本接口总是同时创建「待审行」，即使对端是公开账号并会立刻 `Accept` —— 短时间内 `following/requests/sent` 会短暂出现该行。
7. 覆盖台账 `docs/research/2026-11-04-misskey-api-coverage.md:84`–`:85` 写着「`following/create` 对远端关注者不会写入待审行」，该描述**已与代码不符**：`internal/httpapi/following.go:179` 会写，`internal/httpapi/following_test.go:902` 的 `TestFollowingCreateRemoteRecordsPendingRequest` 验证了这一点（实测通过）。对接应以代码为准。

## POST /api/following/delete

取消对某个用户的关注。与 Misskey 一致，只处理「已存在的关注关系」，**不能用它撤回待审请求**。

- 代码：`internal/httpapi/following.go:84`
- 路由：`internal/httpapi/server.go:131`

### 请求参数

| 字段 | 类型 | 必填 | 默认 | 校验与说明 |
|------|------|------|------|-----------|
| `userId` | string | 是 | — | 要取关的用户 id。空/缺失 → 400 `INVALID_PARAM` |
| `i` | string | 否 | — | token |

校验顺序与 create 相同：解码 → 认证 → `userId` 非空 → 不能是自己 → 用户存在 → 必须已关注。

### 成功响应

200，body 是被取关用户的 UserLite，`name` / `avatarUrl` **恒为 `null`**（`internal/httpapi/following.go:136`）。字段全集同上。

### 错误

| HTTP | code | 条件 |
|------|------|------|
| 400 | `INVALID_PARAM` | body 非法 JSON；`userId` 缺失 |
| 401 | `AUTHENTICATION_FAILED` | 无/错 token |
| 400 | `FOLLOWEE_IS_YOURSELF` | `userId` 是调用者自己 |
| 404 | `NO_SUCH_USER` | 用户不存在（错误 id 为 `5b12c78d-…`，与 create 不同） |
| 400 | `NOT_FOLLOWING` | `followings` 里没有这一行。**只有待审请求时也是这个错误** |
| 500 | `INTERNAL_ERROR` | 删除失败 |

### 行为细节

- 删除 `followings` 行，并**顺带**删除 `follow_requests` 中同方向的待审行，然后重算双方计数（`internal/federation/store.go:507`）。
- 本地目标的取关只影响本地，并发出 Streaming `unfollow` 事件（给被取关者的 main，payload `{"id": followerId}`）。
- 远端目标额外投递一条 `Undo`，内嵌的 `Follow` id 为 `<本机>/users/<me>#follows/<目标行 id>`（`internal/httpapi/following.go:194`）。投递失败只记日志：本地关系已删除，仍然返回 200（`internal/httpapi/following.go:130`）。

### 已知限制

1. **撤不回待审请求**：若只有待审请求（远端关注尚未被接受），本端点返回 400 `NOT_FOLLOWING`，且不会清理 `follow_requests`、也不会发 `Undo`。前端「取消关注」按钮必须按状态分流：已关注 → `following/delete`；待审 → `following/requests/cancel`。
2. **不检查对端 block / 不发 `Block` 联动**：忽略 `blocks` 表。
3. 响应缺关系状态与计数，理由同 create。

## POST /api/following/requests/list

列出**别人发给调用者、尚待调用者审批**的关注请求（`follow_requests.followee_id = 调用者`）。

- 代码：`internal/httpapi/following_requests.go:37`（薄封装）→ `internal/httpapi/following_requests.go:50`（共用实现）
- 路由：`internal/httpapi/server.go:132`

### 请求参数

全部可选；共同语义见「通用约定 / 分页游标语义」。

| 字段 | 类型 | 必填 | 默认 | 说明 |
|------|------|------|------|------|
| `limit` | int | 否 | 10 | `<=0` → 10；`>100` → 100 |
| `sinceId` | string | 否 | — | `id > sinceId`（排他），单独使用时结果升序 |
| `untilId` | string | 否 | — | `id < untilId`（排他），结果降序 |
| `sinceDate` | int64 (ms) | 否 | — | `created_at > 该时刻`；单独使用时升序 |
| `untilDate` | int64 (ms) | 否 | — | `created_at < 该时刻` |
| `i` | string | 否 | — | token（本端点必须提供，否则 401） |

只按 `followee_id` 过滤；**不接受 `userId`**——不能替别人查收件箱（传了会被忽略，不影响结果）。

### 成功响应

200，裸数组，元素是 FollowRequest（见「通用约定 / FollowRequest」）：

```json
[
  {
    "id": "9k2m4p6q8s0u2w4y6a",
    "follower": {
      "id": "9a1b2c3d4e5f6g7h8i",
      "name": "Bob",
      "username": "bob",
      "host": null,
      "avatarUrl": "https://fish.example/media/ab/cd",
      "avatarBlurhash": null,
      "avatarDecorations": [],
      "isBot": false,
      "isCat": false,
      "requireSigninToViewContents": false,
      "emojis": [],
      "onlineStatus": "unknown",
      "badgeRoles": [],
      "instance": null
    },
    "followee": { "…": "同上，值为调用者本人" }
  }
]
```

无结果时是 `[]`。

### 错误

| HTTP | code | 条件 |
|------|------|------|
| 401 | `AUTHENTICATION_FAILED` | 无/错 token |
| 400 | `INVALID_PARAM` | body 非法 JSON、`limit` 不是整数 |
| 500 | `INTERNAL_ERROR` | 查询失败；或列表中某用户行打包失败 |

### 已知限制

- **没有 `createdAt`**：客户端无法显示请求时间，也无法按时间排序（只能依赖 id 序）。
- 列表只包含 `follow_requests` 表的行。目前只有两种来源会写入该表：远端对锁定本地账号发起的 `Follow`（`internal/federation/inbox.go:113`）和 `following/create` 对远端目标的乐观待审行（方向相反，进 `sent`）。由于本地账号的 `is_locked` 无法通过 API 打开，**实际部署中这个列表通常是空的**。
- 没有「未读数」概念：`i`（`users/i`）响应里的 `hasPendingReceivedFollowRequest` 恒为 `false`（`internal/httpapi/users_pack.go:145`），前端不能依赖它显示红点。

## POST /api/following/requests/sent

列出**调用者发出、尚未被对方接受**的关注请求（`follow_requests.follower_id = 调用者`）。

- 代码：`internal/httpapi/following_requests.go:43` → `internal/httpapi/following_requests.go:50`
- 路由：`internal/httpapi/server.go:133`

### 请求参数与响应

与 `requests/list` **完全相同**（同一结构体、同一游标语义、同样的 FollowRequest 数组），唯一差别是过滤侧换成 `follower_id = 调用者`。字段表见上一节。

### 错误

与 `requests/list` 相同（401 / 400 `INVALID_PARAM` / 500）。

### 行为细节与限制

- 对远端用户执行 `following/create` 会立即在此列表出现一行（`internal/httpapi/following.go:179`），对端 `Accept` 到达后该行被删除（`internal/federation/inbox.go:157`），转为正式关注。
- 对端 `Reject` 到达时走 `inbox.reject` → `Unfollow`，同样会清掉待审行（`internal/federation/inbox.go:176`）。
- 本地目标不会产生待审行（本地关注是即时的），所以本地关注永远不出现在这里。
- 无法从这里看出「请求是否已被拒绝」——被拒后行直接消失，与「被接受」在列表上不可区分。前端只能结合 `users/show` 的 `isFollowing` 判断。

## POST /api/following/requests/accept

批准一条收到的关注请求：请求行删除、正式关注建立，远端请求方会收到 ActivityPub `Accept`。

- 代码：`internal/httpapi/following_requests.go:126`
- 路由：`internal/httpapi/server.go:134`

### 请求参数

| 字段 | 类型 | 必填 | 默认 | 说明 |
|------|------|------|------|------|
| `userId` | string | 是 | — | **发起关注的一方**（follower）的 id。空/缺失 → 400 `INVALID_PARAM` |
| `i` | string | 否 | — | token |

校验顺序（`followRequestTarget`，`internal/httpapi/following_requests.go:209`）：解码 → 认证 → 服务未就绪（500）→ `userId` 非空 → 用户存在。

没有 `userId` 之外的参数；不存在「批量批准」。

### 成功响应

200，空对象：

```json
{}
```

（`internal/httpapi/following_requests.go:147`；实测 `len(out) == 0`，见 `internal/httpapi/following_test.go:671`。）

### 错误

| HTTP | code | 条件 |
|------|------|------|
| 400 | `INVALID_PARAM` | body 非法 JSON；`userId` 缺失 |
| 401 | `AUTHENTICATION_FAILED` | 无/错 token |
| 404 | `NO_SUCH_USER` | `userId` 不存在（id `66ce1645-…`） |
| 400 | `NO_FOLLOW_REQUEST` | 没有 (follower=`userId`, followee=调用者) 的待审行（id `bcde4f8b-…`） |
| 500 | `INTERNAL_ERROR` | 事务失败 |

### 行为细节

- 在**单个事务**里：删除 `follow_requests` 行 → 插入 `followings`（`created_via = 'request'`，`uri` 沿用请求行里保存的原始 `Follow` id）→ 重算双方计数（`internal/federation/store.go:442`）。若 `followings` 已存在，插入被 `ON CONFLICT DO NOTHING` 吞掉，但请求行照样删除，仍返回 200。
- 发出 Streaming `follow`（给调用者）与 `followed`（给请求方）事件（`internal/httpapi/following_requests.go:145`）。
- 若请求方是远端 actor（`users.host` 非空），额外投递一条 `Accept`，内嵌原始 `Follow`（`internal/httpapi/following_requests.go:251`）：
  - 内嵌 `Follow` 的 id 优先用请求行里的 `follow_requests.uri`；缺失时回退为 `<请求方 actor URI>#follows/<调用者 id>`（`internal/httpapi/following_requests.go:278`）；
  - `Accept` 自身的 id 为 `<followID>#accept`（实测断言见 `internal/httpapi/following_test.go:739`）；
  - 投递失败只记日志，不影响 200（本地关系已建立）。
- 请求方是本地的（`users.host` 为空）时不投递任何 ActivityPub 活动（`internal/httpapi/following_requests.go:263`）。

### 已知限制

- 不产生 `followRequestAccepted` 通知（第四十七波只加了 `follow` 与 `receiveFollowRequest`），
  本地请求方的感知途径仍是 Streaming 的 `followed` 事件或下次 `users/show` 的 `isFollowing`。
- 批准后响应不返回新的关注对象或计数，前端需自行刷新 `users/followers`。

## POST /api/following/requests/reject

拒绝一条收到的关注请求。**对不存在的请求是幂等成功**（照 Misskey 语义）。

- 代码：`internal/httpapi/following_requests.go:153`
- 路由：`internal/httpapi/server.go:135`

### 请求参数

| 字段 | 类型 | 必填 | 默认 | 说明 |
|------|------|------|------|------|
| `userId` | string | 是 | — | **发起关注的一方**的 id |
| `i` | string | 否 | — | token |

### 成功响应

200，空对象 `{}`（`internal/httpapi/following_requests.go:171`）。即使请求行本来就不存在（重复点击、别人已撤回）也返回 200。

### 错误

| HTTP | code | 条件 |
|------|------|------|
| 400 | `INVALID_PARAM` | body 非法 JSON；`userId` 缺失 |
| 401 | `AUTHENTICATION_FAILED` | 无/错 token |
| 404 | `NO_SUCH_USER` | `userId` 不存在（id `abc2ffa6-…`） |
| 500 | `INTERNAL_ERROR` | 删除失败 |

注意与 `accept` / `cancel` 的不对称：`reject` **没有**「请求不存在」错误码。

### 行为细节

- 删除 `follow_requests` 行，不写 `followings`，不动计数（`internal/federation/store.go:481`）。
- 删除成功后若请求方是远端 actor，投递 `Reject`（内嵌原始 `Follow`，`internal/httpapi/following_requests.go:284`）；请求行不存在时不投递（`internal/httpapi/following_requests.go:168`）。
- **不发任何 Streaming 事件**：被拒绝的请求方不会收到 `unfollow` 或其它提示。

### 已知限制

- 幂等成功会让前端无从区分「真的拒了一条」与「本来就没有」。要做乐观 UI，请以本地列表为准移除该项。
- 无通知、无 streaming，远端请求方只能从对端 `Reject` 得到结果。

## POST /api/following/requests/cancel

撤回**调用者自己发出**、尚未被接受的关注请求；对远端目标会补发 `Undo(Follow)`。

- 代码：`internal/httpapi/following_requests.go:177`
- 路由：`internal/httpapi/server.go:136`

### 请求参数

| 字段 | 类型 | 必填 | 默认 | 说明 |
|------|------|------|------|------|
| `userId` | string | 是 | — | **被关注方**（followee）的 id，即当初 `following/create` 的目标 |
| `i` | string | 否 | — | token |

### 成功响应

200，body 是被关注方的 UserLite，**带** profile（`name`/`avatarUrl` 有值，因为传入了 `loadProfile`，`internal/httpapi/following_requests.go:202`）：

```json
{
  "id": "9k2m4p6q8s0u2w4y6a",
  "name": "Dave",
  "username": "dave",
  "host": "remote.example",
  "avatarUrl": null,
  "avatarBlurhash": null,
  "avatarDecorations": [],
  "isBot": false,
  "isCat": false,
  "requireSigninToViewContents": false,
  "emojis": [],
  "onlineStatus": "unknown",
  "badgeRoles": [],
  "instance": null
}
```

实测断言了 `id` 与 `username`（`internal/httpapi/following_test.go:815`）。

### 错误

| HTTP | code | 条件 |
|------|------|------|
| 400 | `INVALID_PARAM` | body 非法 JSON；`userId` 缺失 |
| 401 | `AUTHENTICATION_FAILED` | 无/错 token |
| 404 | `NO_SUCH_USER` | `userId` 不存在（id `4e68c551-…`） |
| 400 | `FOLLOW_REQUEST_NOT_FOUND` | 没有 (follower=调用者, followee=`userId`) 的待审行（id `089b125b-…`） |
| 500 | `INTERNAL_ERROR` | 删除失败 |

### 行为细节

- 删除 `follow_requests` 行（`internal/federation/store.go:481`），不碰 `followings`、不动计数。
- 若目标 `users.host` 非空，投递 `Undo(Follow)`，复用的是 `following/delete` 的同一条路径 `sendRemoteUndoFollow`（`internal/httpapi/following_requests.go:200`、`internal/httpapi/following.go:185`）。投递失败只记日志，仍返回 200。
- **不发 Streaming 事件**，本地与远端都没有 `unfollow` 提示。

### 已知限制

- **不删除可能已经建立的关注**：若在 `cancel` 之前对端已经 `Accept` 并写入 `followings`（竞态下可能），本端点只删待审行 → 返回 400 `FOLLOW_REQUEST_NOT_FOUND`，而关注关系仍然存在。前端在收到 `followed` 事件后应改调 `following/delete`。
- 与 `reject` 不同，这里对「没有请求」报错而不是幂等成功，前端需要处理 400。

## 已知缺口与前端降级建议

以下缺口都是代码层面确认的（不是 Misskey 文档推测），新前端需要为每一项准备降级策略。

### 完全未实现的相邻端点

`docs/research/2026-11-04-misskey-api-coverage.md:73` 记录 `following/*` 尚缺 4 个端点，均**未注册路由**，调用会得到 404 `NO_SUCH_ENDPOINT`：

| 端点 | 影响 | 建议降级 |
|------|------|----------|
| `following/list` | 无法按「关注关系」而非「用户」分页 | 用 `users/following`（返回 Following 对象，含 `id`/`createdAt`/`follower`/`followee`） |
| `following/update` | 无法设置「接收转推 / 仅显示」等关系级开关 | 隐藏该设置项 |
| `following/update-all` | 无法批量改关系设置 | 同上 |
| `following/invalidate` | 无法强制刷新与某远端用户的关系缓存 | 隐藏「重新同步」入口 |

另外相关但不在 `following/*` 命名空间、同样未实现的能力：

- `users/relation` / `users/relations`：查看关系状态只能靠 `users/show`（带 token 时填充 `isFollowing`、`isFollowed`、`hasPendingFollowRequestFromYou`、`hasPendingFollowRequestToYou`，`internal/httpapi/account.go:276`），且 `users/show` 一次只能查一个用户。
- 关注通知：`follow` 与 `receiveFollowRequest` 已实现（第四十七波）；`followRequestAccepted` 未实现。
- 「待审请求」红点：`i` 的 `hasPendingReceivedFollowRequest` 恒为 `false`。

### 语义缺口汇总

| 缺口 | 代码位置 | 对前端的影响 |
|------|----------|--------------|
| `following/create` 对远端投递失败仍返回 200 | `internal/httpapi/following.go:146`、`:154`、`:170` | 不能仅凭 200 认为已关注/已发出请求，需复查关系状态 |
| 联邦模式拒绝时不报错 | 同上（`CheckAllowed` 只记日志） | 需在 UI 层按 `meta.federation` 隐藏远端关注 |
| 锁定账号无审批流程，`isLocked` 恒 false | `internal/httpapi/following.go:70`、`internal/httpapi/users_pack.go:57` | 不要实现「关注需审批」的等待态 UI（除 `requests/sent` 这一远端场景） |
| 入站 `Follow`/`Accept`/`Undo` 不发 Streaming 事件 | `internal/federation/inbox.go:122`、`:198`（`OnFollow`/`OnUnfollow` 未接线） | 远端关注/被关注/被取关时在线客户端不会即时更新，需要轮询或手动刷新 |
| FollowRequest 无 `createdAt` | `internal/httpapi/following_requests.go:116` | 请求列表不能按时间显示/分组 |
| `following/create`/`delete` 的 UserLite 缺 `name`/`avatarUrl` | `internal/httpapi/following.go:80`、`:136` | 不要用响应覆盖本地用户缓存 |
| `users/show` 无 token 时关系标志全为 false | `internal/httpapi/users_pack.go:88`（默认值） | 未登录用户看不到「已关注」状态，需要登录后才可判断 |
| `blocks` 表不参与关注路径 | 全命名空间 | 不实现「已屏蔽则不能关注」的客户端校验，或仅在本地做提示 |
| 关注/取关对停用账号不做拒绝 | `internal/httpapi/following.go:50` | 需前端自行检查 `isSuspended` |
| `following/delete` 不能撤销待审请求 | `internal/httpapi/following.go:116` | 按状态分流按钮（见该端点限制 1） |

### 前端推荐调用流程

1. 用户点击「关注」：`followersVisibility` 与远端判断 → `following/create`；成功后**不要**直接用响应替换用户对象，改为把本地 `isFollowing`（本地目标）或 `hasPendingFollowRequestFromYou`（远端目标）置位，或调 `users/show` 复核。
2. 显示「请求待审」态：`hasPendingFollowRequestFromYou === true`（来自 `users/show`，本地目标不会置位）或该用户在 `following/requests/sent` 里。
3. 用户点击「取消关注」：若处于待审态 → `following/requests/cancel`；若已关注 → `following/delete`。
4. 通知/审批页：`following/requests/list`（`limit` 默认 10，用 `untilId` 翻页），`accept` / `reject` 以 `userId` = 请求方 id 调用；成功后本地移除该条（响应只是 `{}`）。
5. 实时更新：只依赖 `GET /streaming` 的 `main` 频道上的 `follow` / `followed` / `unfollow`，并明确这些事件**不覆盖**入站联邦关注。

## 附：本命名空间的代码索引

| 关注点 | 代码位置 |
|--------|----------|
| 路由注册 | `internal/httpapi/server.go:130`–`internal/httpapi/server.go:136` |
| 关注 / 取关 handler | `internal/httpapi/following.go:25`、`:84` |
| 远端 `Follow` / `Undo` 投递 | `internal/httpapi/following.go:142`、`:185` |
| 请求列表 / 审批 handler | `internal/httpapi/following_requests.go:50`、`:126`、`:153`、`:177` |
| 请求对象打包 | `internal/httpapi/following_requests.go:107` |
| 参数解码与鉴权辅助 | `internal/httpapi/following_requests.go:209`、`internal/httpapi/auth.go:30` |
| UserLite 打包 | `internal/httpapi/users_pack.go:12` |
| 存储：关注 / 取关 | `internal/federation/store.go:495`、`:507` |
| 存储：待审请求增删查 | `internal/federation/store.go:334`、`:373`、`:442`、`:481` |
| 存储：关系判定 | `internal/federation/store.go:555`、`:565` |
| Streaming 事件 | `internal/httpapi/streaming_publish.go:106`、`:115` |
| 入站 `Follow` / `Accept` / `Reject` / `Undo` | `internal/federation/inbox.go:101`、`:134`、`:167`、`:189` |
| 覆盖台账（注意其中的过时描述） | `docs/research/2026-11-04-misskey-api-coverage.md:73`、`:84` |
| 行为测试 | `internal/httpapi/following_test.go:88`–`:952` |

## 2026-11-06 增补：静音与屏蔽（mute / blocking）

这两组端点与 following 共用 `users` 表上的关系，故记在本文件。表结构 `mutes`、`blocks` 在 schema v1
就存在，没有新增迁移。

### POST /api/mute/create

请求 `{ userId }`，需登录。成功返回 **204 无正文**。

| 错误 | code | 条件 |
|------|------|------|
| 404 `NO_SUCH_USER` | `6fef56f3-e765-4957-88e5-c6f65329b8a5` | userId 不存在 |
| 400 `MUTEE_IS_YOURSELF` | `a4619cb2-5f23-484b-9301-94c903074e10` | 目标是自己 |
| 400 `ALREADY_MUTING` | `7e7359cb-160c-4956-b08f-4d1c653cd007` | 已在名单中 |

`expiresAt` 未实现（上游支持限时静音），当前静音永久有效。

### POST /api/mute/delete

请求 `{ userId }`。不存在的关系是 no-op，仍返回 204。

### POST /api/mute/list

请求 `{ limit?, sinceId?, untilId? }`（limit 默认 30、上限 100）。返回数组：

```json
[{ "id": "关系行 id", "createdAt": "…", "mutee": { /* UserDetailed */ } }]
```

按关系行 id 倒序，`sinceId`/`untilId` 是关系行 id，不是用户 id。

### POST /api/blocking/create

请求 `{ userId }`，需登录。成功返回 `UserDetailedNotMe` 对象。错误 code 与 mute 对称
（`NO_SUCH_USER`、`BLOCKEE_IS_YOURSELF`、`ALREADY_BLOCKING`）。

副作用：同一事务内删除双方的 `followings` 与 `follow_requests`，即屏蔽会自动解除互相关注，
并取消待审请求。

### POST /api/blocking/delete

请求 `{ userId }`，返回 204；已不存在的屏蔽是 no-op。解除屏蔽不会恢复关注关系。

### POST /api/blocking/list

参数与 `mute/list` 相同，返回 `[{ id, createdAt, blockee }]`。

### 对时间线与 `users/show` 的影响

- 所有时间线（home/local/social/global）在登录状态下都会排除：被你静音的作者、双向屏蔽关系中的作者、
  你静音实例上的作者。过滤在 SQL 里完成，因此分页仍是满页。
- `users/show` 在带凭据查看他人时返回 `isMuted`、`isBlocked`（是否由你发起）。匿名查看不带这两个字段。
- `UserDetailed` 只暴露 `isBlocked`（你屏蔽对方）；对方屏蔽你的方向不暴露，避免泄露。
