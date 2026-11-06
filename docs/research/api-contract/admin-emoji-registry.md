# API 对接契约：admin-emoji-registry

本文覆盖 Cinderfish 后端的四组端点：管理端账号管理（`admin/accounts/create`、`admin/show-users`、
`admin/suspend-user`、`admin/unsuspend-user`）、联邦运行期管理（`admin/federation/*`）、自定义表情
（`emojis`、`admin/emoji/*`）以及每用户客户端设置仓库（`i/registry/*`）。

所有结论来自本仓库实现代码，逐条标注了 `internal/...:行号`。Misskey 侧对照版本为
`tmp/refcheck/misskey`（`c05e844`），只在"已知缺口"里出现，不作为本实现的依据。

路由注册：`internal/httpapi/server.go:140`–`158`。

## 端点总览

| 路径 | 方法 | 认证 | 管理员 | 实现位置 | 一句话说明 |
|------|------|------|--------|----------|------------|
| `/api/admin/accounts/create` | POST | 必须 | 是 | `internal/httpapi/admin_accounts.go:13` | 管理员直接建号 |
| `/api/admin/show-users` | POST | 必须 | 是 | `internal/httpapi/admin_accounts.go:57` | 本地账号列表（limit/offset） |
| `/api/admin/suspend-user` | POST | 必须 | 是 | `internal/httpapi/admin_accounts.go:92` | 冻结账号 |
| `/api/admin/unsuspend-user` | POST | 必须 | 是 | `internal/httpapi/admin_accounts.go:96` | 解冻账号 |
| `/api/admin/federation/mode` | GET | 必须 | 是 | `internal/httpapi/federation_admin.go` | 读取运行期联邦模式 |
| `/api/admin/federation/mode` | POST | 必须 | 是 | `internal/httpapi/federation_admin.go` | 运行期切换联邦模式 |
| `/api/admin/federation/allowlist` | GET | 必须 | 是 | `internal/httpapi/federation_admin.go:79` | 读取联邦白名单 |
| `/api/admin/federation/allowlist` | POST | 必须 | 是 | `internal/httpapi/federation_admin.go:79` | 增删白名单主机 |
| `/api/emojis` | POST | 匿名 | 否 | `internal/httpapi/emojis.go:53` | 本地自定义表情清单（公开） |
| `/api/admin/emoji/list` | POST | 必须 | 是 | `internal/httpapi/emojis.go:75` | 管理端表情列表 |
| `/api/admin/emoji/add` | POST | 必须 | 是 | `internal/httpapi/emojis.go:113` | 以 drive 文件注册表情 |
| `/api/admin/emoji/delete` | POST | 必须 | 是 | `internal/httpapi/emojis.go:173` | 删除表情 |
| `/api/i/registry/set` | POST | 必须 | 否 | `internal/httpapi/registry.go:60` | 写入一条客户端设置 |
| `/api/i/registry/get` | POST | 必须 | 否 | `internal/httpapi/registry.go:81` | 读取原始值 |
| `/api/i/registry/get-detail` | POST | 必须 | 否 | `internal/httpapi/registry.go:105` | 读取值 + `updatedAt` |
| `/api/i/registry/keys` | POST | 必须 | 否 | `internal/httpapi/registry.go:130` | 列出 scope 下的 key |
| `/api/i/registry/scopes` | POST | 必须 | 否 | `internal/httpapi/registry.go` | 列出当前用户写过的 scope |
| `/api/i/registry/get-all` | POST | 必须 | 否 | `internal/httpapi/registry.go:148` | 取出整个 scope |
| `/api/i/registry/remove` | POST | 必须 | 否 | `internal/httpapi/registry.go:170` | 删除一个 key |
| `/api/i/registry/clear` | POST | 必须 | 否 | `internal/httpapi/registry.go:187` | 清空一个 scope |

不存在 GET 别名：`/api/emojis` 只有 POST（Misskey 允许 GET，见该节缺口）。`admin/federation/allowlist`
有 GET + POST 双方法（共用一个 handler）；`admin/federation/mode` 自 2026-11-06 起同样支持 GET（读运行期值）
与 POST（切换）。

## 通用约定

### 认证：token 从哪里读

`internal/httpapi/auth.go:30` 的 `authenticate` 依次尝试两处：

1. 请求体里的字符串字段 `i`（优先级最高）；
2. `Authorization: Bearer <token>` 头（大小写不敏感的前缀匹配）。

两处都没有 → 视为匿名，`authenticate` 返回 `(nil, nil)`。token 无效、过期或账号被冻结
（`users.is_suspended = TRUE`）→ 返回错误，所有需要认证的 handler 统一回 401 `AUTHENTICATION_FAILED`。

注意：`requireAdmin`（`internal/httpapi/auth.go:74`）的注释写的是"header first"，实际实现与
`authenticate` 一致，是 **body 的 `i` 优先**；以代码为准。管理端点也接受 header，两者等价。

前端建议：管理端与 registry 一律走 `Authorization: Bearer`，把 `i` 留给 Misskey 兼容客户端。

### 管理员判定

`requireAdmin` 只检查 `users.is_admin`，本项目没有 moderator 角色。三种结果：

| 情形 | HTTP | `error.code` | `error.kind` |
|------|------|--------------|--------------|
| 没有 token / token 无效 / 账号被冻结 | 401 | `AUTHENTICATION_FAILED` | `client` |
| token 有效但不是管理员 | 403 | `FORBIDDEN` | `permission` |
| 通过 | — | — | — |

401 响应额外带 `WWW-Authenticate` 头：
`Bearer realm="Misskey", error="invalid_token", error_description="Authentication failed. Please ensure your token is correct."`
（`writeAuthenticationError`，`internal/httpapi/server.go:304`）。

### 错误 JSON 形状

统一信封（`internal/httpapi/server.go:235`–`287`）：

```json
{"error":{"message":"...","code":"...","id":"...","kind":"client"}}
```

- `code` 是机器可读的短字符串，**前端应以 `code` 为准**，`message` 只用于日志。
- `id` 是 Misskey 风格的 UUID；多数与 Misskey 同名错误一致，少数是自造值（逐条在端点小节注明）。
- `kind` ∈ `client` | `server` | `permission`。
- 参数校验失败额外带 `info` 字段（`invalidParamError`，`server.go:290`）：`{"reason":"..."}` 或 `{"reason":"could not read request body"}`。

请求体处理（`internal/httpapi/ping.go:23`）：空 body 合法；body 上限 8 MiB；JSON 语法错误或字段类型
不符 → 400 `INVALID_PARAM`（`info.reason` = `body is not valid JSON`）；**未知字段被静默忽略**
（`json.Unmarshal` 语义），所以客户端多传字段不会报错，但也不会生效。

404 兜底：未注册的路径/方法由 `notFoundHandler`（`server.go:327`）回答
`404` + `NO_SUCH_ENDPOINT`，不会返回 Go 默认的纯文本。

### 时间与空值

- 所有时间字段格式为 `RFC3339Nano` 的 UTC 字符串，例如 `2026-11-04T12:00:00.123456Z`。
- 列表型字段一律是 `[]` 而不是 `null`（`packUserLite`、`allowlistJSON` 等专门做了兜底），客户端可以直接 `.length`。
- 可空字段明确输出 `null`（例如 `name`、`avatarUrl`、`category`、`host`），不会省略键。
- 返回空对象的端点（`suspend-user`、`registry/set` 等）body 是 `{}`，不是 `null`。

### 分页

本命名空间**没有** `untilId`/`sinceId` 的统一游标语义，只有两处各自不同的分页：

- `admin/show-users`：`limit` + `offset`（`internal/account/service.go:297`）。`limit <= 0 || limit > 100`
  会被改成 30，`offset < 0` 改成 0。没有 `sort`/`state`/`origin` 等筛选。
- `admin/emoji/list`：`limit` + `sinceId`/`untilId`（`internal/media/emoji.go:102`）。`limit <= 0` → 10，
  `limit > 100` → 100。**排序恒为 `id DESC`**；`sinceId` 只是加了 `id > sinceId` 过滤后仍然倒序返回，
  与 Misskey "sinceId 时改升序" 的行为不同（见该节缺口）。

`i/registry/*` 与 `emojis` 都是全量返回，不分页。

## POST /api/admin/accounts/create

管理员直接建号，用于 `registration_mode = "admin"`（自助注册关闭）的实例。与注册模式无关，
任何模式下都可以调用。

- 路由：`internal/httpapi/server.go:151`
- 实现：`internal/httpapi/admin_accounts.go:13`
- 校验逻辑在 `internal/account/service.go:113`（`Service.Create`）

### 请求

| 字段 | 类型 | 必填 | 默认 | 校验 |
|------|------|------|------|------|
| `username` | string | 是 | — | 非空、长度 ≤ 20、只允许 `[A-Za-z0-9_]`（`internal/auth/auth.go:150`）；大小写不敏感查重 |
| `password` | string | 是 | — | 只检查非空，**没有长度或强度下限**；服务端用 bcrypt 存哈希（`auth.HashPassword`，`internal/auth/auth.go:38`） |

`setupPassword`、`email` 等 Misskey 字段未实现，传了也被忽略。

### 响应

`200`，返回一个 **UserDetailed 对象**（`packUserDetailed`，见本文末"共享对象：UserDetailed"），
随后由 `applyUserCounts` 填上真实的 `followersCount` / `followingCount` / `notesCount`。

**不返回 `token`**，也不返回 MeDetailed 的私有字段。也就是说：本端点建出来的账号，管理员拿不到
它的 API token，用户只能自己走 `signin`（密码由管理员转告）。这是与 Misskey 最实质的差异（见缺口）。

### 错误

| HTTP | code | 触发 |
|------|------|------|
| 401 | `AUTHENTICATION_FAILED` | 无 token、token 无效、调用者被冻结 |
| 403 | `FORBIDDEN` | 调用者不是管理员 |
| 400 | `INVALID_PARAM` | body 非法 JSON / 字段类型错误 |
| 400 | `INVALID_USERNAME` | 用户名为空、超 20 字符或含非法字符（`RegistrationError`） |
| 400 | `INVALID_PASSWORD` | 密码为空 |
| 400 | `DUPLICATED_USERNAME` | 用户名已被占用（`errIDSignupFailed` = `4d371c9a-3a0f-4c8b-9b8e-2f9c0a1d3e45`） |
| 500 | `INTERNAL_ERROR` | 数据库故障，或 server 未接数据库（`s.accounts == nil`，此处用 500 而非 503） |

### 已知缺口

- 响应不含 `token`（Misskey 返回 `MeDetailed & {token}`），管理员无法代发凭据。
- 不支持 `setupPassword`、`emailAddress`、角色、`approvalRequired` 等 Misskey 参数。
- Misskey 的首次安装场景（无 root 用户时凭 `setupPassword` 免登录建首个管理员）未实现。
- 密码强度：Misskey 用 `passwordSchema` 约束，本实现的 `Service.Create` 只判空。

## POST /api/admin/show-users

本地账号列表，管理员在前、其后按创建时间升序。

- 路由：`internal/httpapi/server.go:152`
- 实现：`internal/httpapi/admin_accounts.go:57`
- 查询：`internal/account/service.go:297`（`ListLocal`）

### 请求

| 字段 | 类型 | 必填 | 默认 | 说明 |
|------|------|------|------|------|
| `limit` | number(int) | 否 | 30 | `<= 0` 或 `> 100` 一律按 30 处理（不是 10，也不是夹到 100） |
| `offset` | number(int) | 否 | 0 | 负数按 0 处理 |

其余 Misskey 参数（`sort`、`state`、`origin`、`username`、`hostname`）**全部被忽略**（结构体里没有
这些字段，`json.Unmarshal` 静默丢弃）。

### 响应

`200`，**裸数组**（不是 `{"users":[...]}`），元素为 UserDetailed 对象，每个都经过 `applyUserCounts`：

```
[
  {
    "id": "...", "username": "root", "name": null, "host": null,
    "avatarUrl": null, "description": null,
    "isAdmin": true, "isModerator": true, "isSuspended": false,
    "createdAt": "2026-11-04T...Z",
    "followersCount": 0, "followingCount": 0, "notesCount": 3,
    ... (完整字段见"共享对象：UserDetailed")
  }
]
```

字段全集与固定值见文末共享对象一节。要点：

- `isModerator` 与 `isAdmin` 取值相同（本项目无 moderator 概念）。
- 排序 SQL 为 `ORDER BY is_admin DESC, created_at ASC`，管理员恒在最前，同组内按创建时间升序。
- 只返回 `host IS NULL` 的行，远端账号不会被列出。
- 关系标志（`isFollowing` 等）恒为 `false`，不会针对调用者填充。

### 错误

| HTTP | code | 触发 |
|------|------|------|
| 401 | `AUTHENTICATION_FAILED` | 未认证或账号被冻结 |
| 403 | `FORBIDDEN` | 非管理员 |
| 400 | `INVALID_PARAM` | body 非法 |
| 500 | `INTERNAL_ERROR` | 数据库故障或服务未装配 |

### 已知缺口

- 缺 `sort`、`state`（`alive`/`suspended`/`admin`…）、`origin`（local/remote）、`username`、`hostname`
  筛选与排序，管理页无法做"只看被封账号"这类视图。
- 默认 `limit` 是 30，与 Misskey 的 10 不同；超出 100 时被压回 30 而不是夹到 100，分页步长会突变。
- `moderationNote` 未返回，管理员备注无法展示。
- 只支持本地账号，Misskey 的 `origin=remote` 无对应实现。

## POST /api/admin/suspend-user

冻结账号。冻结后该账号**无法 signin，也无法再用任何已签发的 token**（`authenticate` 里
`user.IsSuspended` 直接判失败）。

- 路由：`internal/httpapi/server.go:153`
- 实现：`internal/httpapi/admin_accounts.go:92` → 共用 `suspendUser`（`admin_accounts.go:100`）
- 写入：`internal/account/service.go:341`（`SetSuspended`，单条 `UPDATE users`）

### 请求

| 字段 | 类型 | 必填 | 默认 | 校验 |
|------|------|------|------|------|
| `userId` | string | 是 | — | 空字符串 → 400 `INVALID_PARAM`（不查库先判空） |

### 响应

`200`，body 为 `{}`。

### 错误

| HTTP | code | 触发 |
|------|------|------|
| 401 | `AUTHENTICATION_FAILED` | 未认证 / 被冻结 |
| 403 | `FORBIDDEN` | 非管理员 |
| 400 | `INVALID_PARAM` | `userId` 为空或 body 非法 |
| 400 | `CANNOT_SUSPEND_YOURSELF` | `userId` 等于调用者自己（`error.id` 用 `errIDForbidden`，`kind = client`） |
| 404 | `NO_SUCH_USER` | 目标不存在（`errIDNoSuchUser` = `6cc579cc-885d-43d8-95c2-b8c7fc963280`） |
| 500 | `INTERNAL_ERROR` | 数据库故障 |

### 已知缺口

- 只拦"冻结自己"，**允许冻结另一个管理员**（Misskey 拒绝冻结 moderator/admin）。一旦冻掉唯一
  的其他管理员，实例会失去管理能力。
- 没有 `admin/suspend-user` 的副作用（Misskey 会同时下发解冻/禁言相关通知），也不写审核日志。
- 冻结不撤销 token 行，只是每次鉴权时检查标志位；数据库里 `users.is_suspended` 是唯一来源。

## POST /api/admin/unsuspend-user

解冻账号，与 `suspend-user` 完全共用实现，只是 `suspend = false`。

- 路由：`internal/httpapi/server.go:154`
- 实现：`internal/httpapi/admin_accounts.go:96` → `suspendUser`

### 请求

| 字段 | 类型 | 必填 | 默认 |
|------|------|------|------|
| `userId` | string | 是 | — |

### 响应

`200`，body 为 `{}`。

### 错误

与 `suspend-user` 相同（401 / 403 / 400 `INVALID_PARAM` / 400 `CANNOT_SUSPEND_YOURSELF` /
404 `NO_SUCH_USER` / 500）。注意"不能解冻自己"这条借用的是同一个 `CANNOT_SUSPEND_YOURSELF` 码，
语义上略有错位但确实会返回。

### 已知缺口

- 目标本来就是未冻结状态时也返回 200（幂等），客户端无法区分"刚解冻"与"本来就没冻结"。
- 解冻后旧 token 立即恢复可用（因为 token 行从未删除）。

## GET / POST /api/admin/federation/mode

`POST` 运行期切换联邦模式，写入 `instance_meta` 表，**下一个请求立即生效**，不需要重启；重启后数据库值
仍然覆盖配置文件里的种子值（`ApplyFederationConfig`，`internal/httpapi/federation_admin.go:19`）。

`GET` 读取**运行期值**（同一张表的 `federationMode` 行；行缺失时回落到配置文件），供管理台显示当前模式。
`POST /api/meta` 的 `federation` 字段仍读启动配置，两者可能不同，前端应以本端点为准。

这是 Cinderfish 自有端点，Misskey 没有同名路径（对照 `admin/federation/` 下只有 `update-instance` 等）。

- 路由：`internal/httpapi/server.go`（GET 与 POST 两条）
- 实现：`internal/httpapi/federation_admin.go`
- 取值校验：`internal/federation/policy.go:32`（`ParseMode`）

### GET 请求

无 body。认证走 `Authorization: Bearer <token>`（与其他 GET 一致）。

### GET 响应

`200`：

```json
{"mode":"allowlist"}
```

`mode` 取值 `public` | `allowlist` | `none`，与 POST 入参同一词表，不是 Misskey 的
`all`/`specified`/`none`。

### 请求

| 字段 | 类型 | 必填 | 默认 | 取值 |
|------|------|------|------|------|
| `mode` | string | 是 | — | `public` \| `allowlist` \| `none`（仅小写，无其他拼写） |

### 响应

`200`：

```json
{"mode":"allowlist"}
```

`mode` 是**原样回显入参**，不做归一化，也不是 Misskey 的 `all`/`specified`/`none` 词表。
如果需要 Misskey 词表，请读 `POST /api/meta` 的 `policies`/`federation` 相关字段（属 meta-federation 范围）。

### 错误

| HTTP | code | 触发 |
|------|------|------|
| 401 | `AUTHENTICATION_FAILED` | 未认证 |
| 403 | `FORBIDDEN` | 非管理员 |
| 400 | `INVALID_PARAM` | `mode` 缺失或不在枚举内，`info.reason` = `mode must be public, allowlist or none` |
| 500 | `INTERNAL_ERROR` | 写 `instance_meta` 失败 |

### 已知缺口

- handler 直接使用 `s.pool`，**没有 nil 检查**：以 `NewServer`（无数据库）构建的服务调用本端点会
  空指针 panic（每个连接上被 net/http 兜住，客户端看到空响应）。生产部署总是带库，风险低但要知道。
- 语义影响的是**入站与出站的整体放行**（`Policy.IsAllowed`），本实例自身 host 永远放行。
- 没有任何审计记录，只在服务端日志里打一行 `httpapi: federation mode switched to ...`（`federation_admin.go:73`）。
- 切到 `none` 后，ActivityPub 的 actor 端点直接 403（见 `federation_admin_test.go:37` 的断言）。

## GET /api/admin/federation/allowlist

读取联邦白名单。白名单只在 `mode = allowlist` 时生效。

- 路由：`internal/httpapi/server.go:156`
- 实现：`internal/httpapi/federation_admin.go:79`（同一 handler 按 `r.Method` 分流）
- 数据源：`internal/federation/policy.go:150`（`Allowlist`，带 10 秒读缓存）

### 请求

无 body（GET）。带 `Authorization: Bearer` 或查询参数都不影响结果；token 只从 header 读（GET 没有 body）。

### 响应

`200`：

```json
{"hosts":["trusted.example","sub.example.org"]}
```

- 顺序：`ORDER BY host` 的字典序。
- 空白名单返回 `{"hosts":[]}`，不会是 `null`（`allowlistJSON`，`federation_admin.go:115`）。
- 主机名已归一化：小写、去首尾空白、去端口（`normalizeHost`，`policy.go:244`）。

### 错误

| HTTP | code | 触发 |
|------|------|------|
| 401 | `AUTHENTICATION_FAILED` | 无 token / token 无效 |
| 403 | `FORBIDDEN` | 非管理员 |

### 已知缺口

- handler 未判 `s.federationPolicy == nil`：未调用 `AttachFederation` 的服务（如纯 API 测试实例）会 panic。
- 白名单有 10 秒缓存（`allowlistTTL`），GET 读到的可能是最多 10 秒前的快照；写操作会主动失效缓存。
- 没有分页，宿主多时全量返回。

## POST /api/admin/federation/allowlist

新增或删除一条白名单主机，返回变更后的完整列表。

- 路由：`internal/httpapi/server.go:157`
- 实现：`internal/httpapi/federation_admin.go:79`（POST 分支）
- 写入：`internal/federation/policy.go:201`（`AddAllowlistHost`）/ `:218`（`RemoveAllowlistHost`）

### 请求

| 字段 | 类型 | 必填 | 默认 | 说明 |
|------|------|------|------|------|
| `host` | string | 是 | — | 主机名；小写归一化、去空白、去端口。空或纯空白 → 400 |
| `remove` | boolean | 否 | `false` | `true` 时删除该主机，否则添加 |

### 响应

`200`，变更后的完整列表，形状与 GET 相同：

```json
{"hosts":["trusted.example"]}
```

添加已存在的主机是成功（`ON CONFLICT (host) DO NOTHING`）；删除不存在的主机也是成功（`DELETE` 命中 0 行）。
两者都是幂等操作，客户端拿返回的 `hosts` 覆盖本地状态即可。

### 错误

| HTTP | code | 触发 |
|------|------|------|
| 401 | `AUTHENTICATION_FAILED` | 未认证 |
| 403 | `FORBIDDEN` | 非管理员 |
| 400 | `INVALID_PARAM` | `host` 为空/纯空白，或 body 非法 JSON |
| 500 | `INTERNAL_ERROR` | `federationPolicy` 未接数据库（`no database`）或 SQL 失败 |

### 已知缺口

- 只接受单个主机，没有批量增删；也没有导入/导出。
- 不做 DNS 解析或存在性校验，任意字符串（含非法域名）都能入库。
- 与 Misskey 的 `admin/federation/update-instance` 语义不同：这里是"放行名单"，Misskey 那边是
  "实例级联联合控制（followersOnly/blocking 等）"，没有对应关系。

## POST /api/emojis

公开的本地自定义表情清单。**无需认证**，与 Misskey 一样面向所有人（含未登录前端）。

- 路由：`internal/httpapi/server.go:147`
- 实现：`internal/httpapi/emojis.go:53`
- 查询：`internal/media/emoji.go:81`（`ListLocal`，`WHERE host IS NULL`）
- 打包：`packEmojiSimple`（`internal/httpapi/emojis.go:24`）

### 请求

空 body 或 `{}`。没有 `query`、`limit` 之类参数——传了也不会生效（handler 不解码 body）。

### 响应

`200`：

```json
{
  "emojis": [
    {
      "name": "blobcat",
      "category": "cats",
      "aliases": ["blob_cat"],
      "url": "/media/ab/cd/abcd...png",
      "localOnly": true,
      "isSensitive": false
    }
  ]
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `name` | string | 短代码，渲染 `:name:` 用 |
| `category` | string \| null | 分组名，可为 `null` |
| `aliases` | string[] | 别名短代码，恒为数组 |
| `url` | string | 取 `drive_files` 存时的 `storage.URL(storageKey)`；本地后端是 `/media/...`，S3 后端是对象 URL |
| `localOnly` | boolean | 当前恒为 `true`（`EmojiService.Add` 硬编码） |
| `isSensitive` | boolean | 当前恒为 `false`（见 `admin/emoji/add` 缺口） |

注意**不含 `id` 和 `host`**，客户端必须以 `name` 作为唯一键。排序：`category ASC NULLS FIRST, name ASC`。

### 错误

| HTTP | code | 触发 |
|------|------|------|
| 503 | `INTERNAL_ERROR` | 服务未接数据库（`s.emojis == nil`），`kind = server` |
| 500 | `INTERNAL_ERROR` | 查询失败 |

### 已知缺口

- **只注册了 POST**，`GET /api/emojis` 会落到 `NO_SUCH_ENDPOINT` 404。Misskey 该端点 `allowGet: true`，
  部分客户端（含 Firefish 系）习惯用 GET 拉表情表，需要改造或加别名。
- 没有 `Cache-Control`（Misskey `cacheSec: 3600`），每次请求都打库；前端应自行缓存。
- `url` 在注册表情时就被**固化字符串**存进 `emojis.url`（`admin/emoji/add` 取 `file.URL`）。若媒体后端是
  S3 且未配置 `PublicBaseURL`、启用了 `Presign`，存下来的就是**会过期的签名 URL**
  （`internal/media/s3.go:100`–`113`），过期后表情图 403。这种部署下需要前端/运维另行处理，API 不会刷新。
- Misskey 的 `EmojiSimple` 还有 `roleIdsThatCanBeUsedThisEmojiAsReaction`，本实现未返回。
- 不支持 `query`/`category` 过滤，也不支持分页；实例表情上千时整包下发。

## POST /api/admin/emoji/list

管理端表情列表，返回带 `id` 的完整表情对象。包含本地与远端（`host` 非空）行。

- 路由：`internal/httpapi/server.go:148`
- 实现：`internal/httpapi/emojis.go:75`
- 查询：`internal/media/emoji.go:102`（`ListAdmin`）
- 打包：`packAdminEmoji`（`internal/httpapi/emojis.go:37`）

### 请求

| 字段 | 类型 | 必填 | 默认 | 说明 |
|------|------|------|------|------|
| `query` | string \| null | 否 | null | 子串匹配 `name` **或** 任一 `aliases`（`ILIKE %q%`）。空串等同不传 |
| `limit` | number(int) | 否 | 10 | `<= 0` → 10，`> 100` → 100 |
| `sinceId` | string | 否 | — | 过滤 `id > sinceId`（注意仍是倒序返回，见缺口） |
| `untilId` | string | 否 | — | 过滤 `id < untilId`（标准游标：翻下一页用它） |

`sinceDate` / `untilDate` 不在解码结构体里，传了被忽略。

### 响应

`200`，**裸数组**：

```json
[
  {
    "id": "01J...",
    "name": "blobcat",
    "host": null,
    "category": "cats",
    "aliases": ["blob_cat"],
    "url": "/media/ab/cd/abcd...png",
    "isSensitive": false,
    "localOnly": true,
    "createdAt": "2026-11-04T12:00:00.123456Z"
  }
]
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | 18 字符 base36 id，删除时用 |
| `name` | string | 短代码 |
| `host` | string \| null | 本地行为 `null` |
| `category` | string \| null | 分组 |
| `aliases` | string[] | 别名 |
| `url` | string | 表情图片地址 |
| `isSensitive` | boolean | 敏感标记（当前写入端恒为 `false`） |
| `localOnly` | boolean | 本地行恒为 `true` |
| `createdAt` | string | `RFC3339Nano`，**Misskey 的 `EmojiDetailed` 没有这个字段** |

### 错误

| HTTP | code | 触发 |
|------|------|------|
| 401 | `AUTHENTICATION_FAILED` | 未认证 |
| 403 | `FORBIDDEN` | 非管理员 |
| 400 | `INVALID_PARAM` | body 非法 |
| 500 | `INTERNAL_ERROR` | 服务未装配或查询失败 |

### 已知缺口

- **`sinceId` 与 Misskey 语义不一致**：始终 `ORDER BY id DESC`。用 `sinceId` 轮询新表情时，返回的仍是
  倒序，且与上一页的衔接处需要客户端自己重排。翻页请只用 `untilId`。
- 不支持 `sinceDate`/`untilDate`。
- `query` 不匹配 `category`（Misskey 会匹配），也不解析 `:name:` 形式的表情码输入。
- `query` 未转义 SQL `LIKE` 通配符，输入 `%` 会变成通配符（仅影响搜索结果，不会注入）。
- 返回全部行（含 `host` 非空的远端行），Misskey 只返回 `host IS NULL`。当前没有远端表情导入，
  实践中等价，但将来会分叉。
- 缺 `license`、`roleIdsThatCanBeUsedThisEmojiAsReaction` 字段（Misskey `EmojiDetailed` 有）。
- Misskey 还有 `admin/emoji/update`、`copy`、`list-remote`、`delete-bulk`、`set-category-bulk`、
  `import-zip` 等；本实现全部缺失，管理界面只能"加/删/搜索"。

## POST /api/admin/emoji/add

把一张已上传的 drive 文件注册成自定义表情。文件名即短代码。

- 路由：`internal/httpapi/server.go:149`
- 实现：`internal/httpapi/emojis.go:113`
- 写入：`internal/media/emoji.go:48`（`EmojiService.Add`）
- 文件查询：`internal/media/drive.go:131`（`DriveService.Get`，**不校验归属**）

### 请求

| 字段 | 类型 | 必填 | 默认 | 校验 |
|------|------|------|------|------|
| `name` | string | 是 | — | 必须匹配 `^[a-zA-Z0-9_]+$`（`emojiNamePattern`，`emojis.go:20`）；`null`/空串都会失败 |
| `fileId` | string | 是 | — | `drive_files.id`；不存在 → `NO_SUCH_FILE` |
| `category` | string \| null | 否 | null | 原样存入，无长度限制 |
| `aliases` | string[] | 否 | `[]` | 每一项都必须匹配 `^[a-zA-Z0-9_]+$`，否则整体 400 |

Misskey 的 `license`、`isSensitive`、`localOnly`、`roleIdsThatCanBeUsedThisEmojiAsReaction` 均未支持：
`sensitive` 固定传 `false`，`localOnly` 由 `EmojiService.Add` 硬编码为 `true`。

### 响应

`200`，返回新建的管理端表情对象（与 `admin/emoji/list` 的元素同构，见上一节的字段表）：

```json
{"id":"01J...","name":"blobcat","host":null,"category":"cats","aliases":["blob_cat"],
 "url":"/media/ab/cd/abcd...png","isSensitive":false,"localOnly":true,
 "createdAt":"2026-11-04T12:00:00.123456Z"}
```

### 错误

| HTTP | code | error.id | 触发 |
|------|------|----------|------|
| 401 | `AUTHENTICATION_FAILED` | `b0a7f5f8-...` | 未认证 / 被冻结 |
| 403 | `FORBIDDEN` | `f4f0d9f1-...` | 非管理员 |
| 400 | `INVALID_PARAM` | `3d81ceae-...` | body 非法、`name` 或 `aliases` 不符合字符集 |
| 400 | `NO_SUCH_FILE` | `b6992544-63e7-67f0-fa7f-32444b1b5306` | `fileId` 不存在（**注意 Misskey 的同名错误 id 是 `fc46b5a4-6b92-4c33-ac66-b806659bb5cf`**） |
| 400 | `EMOJI_ALREADY_EXISTS` | `fdb8db8e-3a2a-4f4f-8c2b-2f2b1e0c9d11` | 同名本地表情已存在（**Misskey 用的 code 是 `DUPLICATE_NAME`**） |
| 500 | `INTERNAL_ERROR` | `5d37dbcb-...` | 未装配 drive/emoji 服务，或写库失败 |

前端按 `code` 分支时建议两个词都认：`EMOJI_ALREADY_EXISTS`（本实现）与 `DUPLICATE_NAME`（上游
Misskey）。对接 Cinderfish 时只会看到前者。

### 已知缺口

- 不校验文件类型：任意 drive 文件（文本、压缩包）都能注册成表情。Misskey 会拒绝非图片并报
  `UNSUPPORTED_FILE_TYPE`（错误 id `f7599d96-8750-af68-1633-9575d625c1a7`）。
- 不校验文件归属：管理员可以把**任何用户**的 drive 文件注册成表情。
- 未实现 `isSensitive` / `localOnly` / `license` / 角色限制参数，敏感表情无法通过 API 标记。
- 表情 URL 直接取 `storage.URL(storageKey)`，没有缩略图/webpublic 变体（Misskey 区分
  `originalUrl`/`publicUrl`）。
- 查重靠数据库唯一索引（`idx_emojis_name_local`），并发同名请求可能都进到插入阶段，后者拿到唯一冲突
  并被映射成 `EMOJI_ALREADY_EXISTS`，行为正确。

## POST /api/admin/emoji/delete

按 `id` 删除表情。

- 路由：`internal/httpapi/server.go:150`
- 实现：`internal/httpapi/emojis.go:173`
- 写入：`internal/media/emoji.go:135`（`DELETE FROM emojis WHERE id = $1`）

### 请求

| 字段 | 类型 | 必填 | 默认 | 校验 |
|------|------|------|------|------|
| `id` | string | 是 | — | 空串 → 400 `INVALID_PARAM` |

### 响应

`200`，body 为 `{}`（不是被删对象，也没有 Misskey 的 `EmojiDeleted` 流事件）。

### 错误

| HTTP | code | error.id | 触发 |
|------|------|----------|------|
| 401 | `AUTHENTICATION_FAILED` | — | 未认证 |
| 403 | `FORBIDDEN` | — | 非管理员 |
| 400 | `INVALID_PARAM` | `3d81ceae-...` | `id` 为空 / body 非法 |
| 404 | `NO_SUCH_EMOJI` | `6a5fbb34-9d25-4b4f-8d9f-3c3a5b1c8f2e` | id 不存在（**Misskey 的 id 是 `be83669b-773a-44b7-b1f8-e5e5170ac3a2`**） |
| 500 | `INTERNAL_ERROR` | — | 写库失败 |

### 已知缺口

- 不删除底层 drive 文件，只删 `emojis` 行；文件仍占用存储。
- 没有批量删除（Misskey `admin/emoji/delete-bulk`）。
- 不发送 Streaming 的 `emojiDeleted` 事件（本实现没有该事件类型，见 streaming 范围）。
- 错误 `id` 与 Misskey 不同，依靠 `id` 精确匹配的兼容层会漏判，请按 `code` 判断。

## POST /api/i/registry/set

写入当前用户的一条客户端设置。key + scope 唯一，重复写是 upsert。

- 路由：`internal/httpapi/server.go:140`
- 实现：`internal/httpapi/registry.go:60`
- 写入：`internal/account/registry.go:58`（`RegistryService.Set`）

### 请求

| 字段 | 类型 | 必填 | 默认 | 说明 |
|------|------|------|------|------|
| `key` | string | 是 | — | 空串 → 400 `INVALID_PARAM`（`info.reason` = `key is required`） |
| `value` | any JSON | 否 | `null` | 任意 JSON 值；**本实现允许省略**（Misskey 要求必填），省略时存 `null` |
| `scope` | string[] | 否 | `[]` | 分组维度；元素不做 `^[a-zA-Z0-9_]+$` 校验，空串元素会被丢弃 |
| `domain` | string \| null | 否 | — | **接受但完全忽略**（见缺口） |

scope 归一化：去空串、去重、字典序排序（`normalizeScope`，`registry.go:39`），所以
`["theme","client"]` 与 `["client","theme"]` 命中同一行。

### 响应

`200`，body 为 `{}`。

### 错误

| HTTP | code | 触发 |
|------|------|------|
| 401 | `AUTHENTICATION_FAILED` | 无 token / token 无效 / 账号被冻结 |
| 400 | `INVALID_PARAM` | body 非法，或 `key` 为空 |
| 503 | `INTERNAL_ERROR` | registry 服务未装配（`s.registry == nil`，`kind = server`，`registry.go:27`） |
| 500 | `INTERNAL_ERROR` | 写库失败 |

### 已知缺口

- `domain` 参数被丢弃：所有条目落在 `user_registry.domain = ''` 上。Misskey 用 `domain` 承载
  "按第三方应用隔离"的语义，本实现只有"每用户 + 每 scope"，两个应用用同一 scope 会互相覆盖。
- 未校验 `scope` 元素字符集（Misskey 限制 `^[a-zA-Z0-9_]+$`）。
- 没有值大小上限（除 8 MiB body 限制），也没有条目数配额。
- 没有跨实例同步（Misskey 的 registry 也是本地的，行为一致）。

## POST /api/i/registry/get

读取一条设置的**原始值**，不加信封。

- 路由：`internal/httpapi/server.go:141`
- 实现：`internal/httpapi/registry.go:81`
- 读取：`internal/account/registry.go:74`（`RegistryService.Get`）

### 请求

| 字段 | 类型 | 必填 | 默认 | 说明 |
|------|------|------|------|------|
| `key` | string | 否（实际必填） | — | 省略等于 `""`，直接落到 `NO_SUCH_KEY`，**不报 `INVALID_PARAM`** |
| `scope` | string[] | 否 | `[]` | 同 `set` 的归一化规则 |
| `domain` | string \| null | 否 | — | 忽略 |

### 响应

`200`，body 就是存进去的 JSON 值本身，例如存 `{"accent":"#86b300"}` 就原样返回该对象；
存字符串 `"mine"` 就返回 `"mine"`（带引号的 JSON 字符串）。`Content-Type: application/json; charset=utf-8`。

值的 `null` 情况：`set` 时省略 `value` 会存 `null`，此时 `get` 返回 `null`（HTTP 200，body 是 `null`）。
前端要区分"没这个 key"（404）与"值是 null"（200 `null`）。

### 错误

| HTTP | code | error.id | 触发 |
|------|------|----------|------|
| 401 | `AUTHENTICATION_FAILED` | — | 未认证 |
| 404 | `NO_SUCH_KEY` | `ac3ed68a-62f0-422b-a7bc-d5e09e8f6a6a` | 该 (user, scope, key) 不存在（与 Misskey `i/registry/get` 的 id 一致） |
| 503 | `INTERNAL_ERROR` | — | 服务未装配 |
| 500 | `INTERNAL_ERROR` | — | 读库失败 |

### 已知缺口

- 缺 `key` 时给的是 `NO_SUCH_KEY`（404）而不是 `INVALID_PARAM`（400），与 Misskey 的参数校验不同。
- 其他用户的 key 一律 404（不存在即不可见，`registry_test.go:96` 有断言），不泄露存在性。
- 与 Misskey 一样只能读本用户的；没有管理员代读端点。

## POST /api/i/registry/get-detail

读取一条设置的值与最后写入时间。

- 路由：`internal/httpapi/server.go:142`
- 实现：`internal/httpapi/registry.go:105`
- 读取：`internal/account/registry.go:89`（`GetDetail`）

### 请求

与 `get` 相同：`key`（string，实际必填）、`scope`（string[]，默认 `[]`）、`domain`（忽略）。

### 响应

`200`：

```json
{"updatedAt":"2026-11-04T12:34:56.789012Z","value":{"accent":"#86b300"}}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `updatedAt` | string | `RFC3339Nano` UTC，取自 `user_registry.updated_at`，upsert 时刷新为 `now()` |
| `value` | any JSON | 存进去的原始值，可能是 `null` |

### 错误

| HTTP | code | error.id | 触发 |
|------|------|----------|------|
| 401 | `AUTHENTICATION_FAILED` | — | 未认证 |
| 404 | `NO_SUCH_KEY` | `ac3ed68a-62f0-422b-a7bc-d5e09e8f6a6a` | key 不存在 |
| 503 / 500 | `INTERNAL_ERROR` | — | 服务未装配 / 读库失败 |

### 已知缺口

- Misskey 该端点用自己的 `NO_SUCH_KEY` id（`97a1e8e7-c0f7-47d2-957a-92e61256e01a`），本实现统一用
  `ac3ed68a-...`。按 `code` 判断即可，按 `id` 判断会不一致。
- 不返回 `key`/`scope` 回显，客户端需自己保存上下文。

## POST /api/i/registry/keys

列出某个 scope 下所有 key 的名字，按字典序升序。

- 路由：`internal/httpapi/server.go:143`
- 实现：`internal/httpapi/registry.go:130`
- 读取：`internal/account/registry.go:107`（`RegistryService.Keys`）

### 请求

| 字段 | 类型 | 必填 | 默认 | 说明 |
|------|------|------|------|------|
| `scope` | string[] | 否（实际必填） | `[]` | 漏传等于空 scope，只会命中空 scope 的条目 |
| `key` / `value` / `domain` | — | — | — | 本端点不使用（`registryRequest` 共用结构体，解码后忽略） |

### 响应

`200`，**裸字符串数组**，空时是 `[]`：

```json
["colors","fontSize"]
```

### 错误

| HTTP | code | 触发 |
|------|------|------|
| 401 | `AUTHENTICATION_FAILED` | 未认证 |
| 503 / 500 | `INTERNAL_ERROR` | 服务未装配 / 查询失败 |

### 已知缺口

- Misskey 的 `i/registry/keys` 要求 `scope` 必填且元素匹配 `^[a-zA-Z0-9_]+$`；本实现漏传即按空 scope
  处理，不会 400，容易把"没设置过"误判成"这个 scope 是空的"。
- 没有 `keys-with-type`（Misskey 有，返回 `{key, type}`），前端无法预知值的 JSON 类型，只能 `get-all` 后
  自己 `typeof`。
- 没有 `scopes-with-domain`（列出该用户所有 scope）——本实现也没有 `domain` 维度。

## POST /api/i/registry/get-all

一次取出某个 scope 的全部键值。

- 路由：`internal/httpapi/server.go:144`
- 实现：`internal/httpapi/registry.go:148`
- 读取：`internal/account/registry.go:129`（`RegistryService.All`）

### 请求

| 字段 | 类型 | 必填 | 默认 | 说明 |
|------|------|------|------|------|
| `scope` | string[] | 否（实际必填） | `[]` | 同 `keys` |

### 响应

`200`，**以 key 为属性名的对象**（不是数组）：

```json
{"colors":{"accent":"#86b300"},"fontSize":14}
```

空 scope 时返回 `{}`。

### 错误

| HTTP | code | 触发 |
|------|------|------|
| 401 | `AUTHENTICATION_FAILED` | 未认证 |
| 503 / 500 | `INTERNAL_ERROR` | 服务未装配 / 查询失败 |

### 已知缺口

- 不支持 `domain` 维度（参数被忽略），也没有分页。
- 值全部内联，scope 里存大对象时响应会很大（仅受 8 MiB 请求体限制约束，响应本身无上限）。

## POST /api/i/registry/remove

删除某个 scope 下的一个 key。

- 路由：`internal/httpapi/server.go:145`
- 实现：`internal/httpapi/registry.go:170`
- 写入：`internal/account/registry.go:151`（`Remove`）

### 请求

| 字段 | 类型 | 必填 | 默认 | 说明 |
|------|------|------|------|------|
| `key` | string | 否（实际必填） | — | 空串会尝试删 `key = ''`，通常无行可删 |
| `scope` | string[] | 否 | `[]` | 同前 |

### 响应

`200`，body 为 `{}`。**删除不存在的 key 也返回 200**（`DELETE` 命中 0 行不算错误）。

### 错误

| HTTP | code | 触发 |
|------|------|------|
| 401 | `AUTHENTICATION_FAILED` | 未认证 |
| 503 / 500 | `INTERNAL_ERROR` | 服务未装配 / 写库失败 |

### 已知缺口

- Misskey 的 `i/registry/remove` 在 key 不存在时抛 `NO_SUCH_KEY`（id `1fac4e8a-a6cd-4e39-a4a5-3a7e11f1b019`），
  本实现静默成功。前端若依赖"删不到就报错"来发现状态漂移，需要自己先 `keys` 校验。
- 不做存在性校验也不 400 `key` 缺失。

## POST /api/i/registry/clear

清空当前用户某个 scope 下的全部条目。

- 路由：`internal/httpapi/server.go:146`
- 实现：`internal/httpapi/registry.go:187`
- 写入：`internal/account/registry.go:161`（`Clear`）

### 请求

| 字段 | 类型 | 必填 | 默认 | 说明 |
|------|------|------|------|------|
| `scope` | string[] | 否 | `[]` | **漏传等于清空空 scope**，不会误删其他 scope；但也没有"清空全部"的开关 |
| `key` / `value` / `domain` | — | — | — | 忽略 |

### 响应

`200`，body 为 `{}`。

### 错误

| HTTP | code | 触发 |
|------|------|------|
| 401 | `AUTHENTICATION_FAILED` | 未认证 |
| 503 / 500 | `INTERNAL_ERROR` | 服务未装配 / 写库失败 |

### 已知缺口

- 上游 Misskey `c05e844` 的 `endpoints/i/registry/` 下**没有 `clear.ts`**（只有 get/get-all/get-detail/
  keys/keys-with-type/remove/scopes-with-domain/set）。本端点是 Cinderfish 保留的旧接口：
  调用它不会报错，但按 Misskey 契约实现的客户端不会调用它。
- 没有"清空该用户全部 scope"的管理端点，也没有管理员删除他人 registry 的能力（合规删除只能靠删号级联）。

## POST /api/i/registry/scopes

列出当前用户写过的全部 scope（Firefish 时代 `i/registry/scopes`；Misskey `c05e844` 的同位置端点是
`i/registry/scopes-with-domain`，多一个 `domain` 分组）。

- 路由：`internal/httpapi/server.go`（2026-11-06 新增）
- 实现：`internal/httpapi/registry.go`（`handleRegistryScopes`）
- 读取：`internal/account/registry.go`（`RegistryService.Scopes`，`SELECT DISTINCT scope`）

### 请求

| 字段 | 类型 | 必填 | 默认 | 说明 |
|------|------|------|------|------|
| `domain` | string \| null | 否 | — | 忽略（与其余 registry 端点一致） |

### 响应

`200`，`string[][]`，按 scope 排序（PostgreSQL 数组排序）：

```json
[["base","client"],["system"]]
```

没有条目时返回 `[]`，不是 `null`。空 scope `[]` 与 `domain = ''` 的条目也会出现在结果里。

### 错误

| HTTP | code | 触发 |
|------|------|------|
| 401 | `AUTHENTICATION_FAILED` | 未认证 |
| 503 / 500 | `INTERNAL_ERROR` | 服务未装配 / 读库失败 |

## 共享对象：UserDetailed

`admin/accounts/create`、`admin/show-users` 的响应元素都由 `packUserDetailed`
（`internal/httpapi/users_pack.go:46`，底层 `packUserLite` `:12`）生成，再经 `applyUserCounts`
（`internal/httpapi/users.go:287`）补计数。字段是**固定集合**，缺失的 Misskey 字段不会出现（不是 `undefined`，
而是根本没这个键）。

| 字段 | 类型 | 本实现取值 |
|------|------|------------|
| `id` | string | 18 字符 base36 id |
| `username` | string | 原始大小写 |
| `name` | string \| null | 取 `user_profiles.name`，空则 `null` |
| `host` | string \| null | 本地账号恒为 `null` |
| `avatarUrl` | string \| null | 取 `user_profiles.avatar_url`，空则 `null` |
| `avatarBlurhash` | null | 恒 `null` |
| `avatarDecorations` | array | 恒 `[]` |
| `isBot` | boolean | 恒 `false` |
| `isCat` | boolean | 恒 `false` |
| `requireSigninToViewContents` | boolean | 恒 `false` |
| `emojis` | array | 恒 `[]` |
| `onlineStatus` | string | 恒 `"unknown"` |
| `badgeRoles` | array | 恒 `[]` |
| `instance` | null | 恒 `null` |
| `url` | string | 恒 `""`（**空串，不是 null**） |
| `uri` | string | 恒 `""` |
| `movedTo` | null | 恒 `null` |
| `alsoKnownAs` | null | 恒 `null` |
| `createdAt` | string | `RFC3339Nano` UTC |
| `updatedAt` | null | 恒 `null` |
| `lastFetchedAt` | null | 恒 `null` |
| `bannerUrl` | null | 恒 `null` |
| `bannerBlurhash` | null | 恒 `null` |
| `isLocked` | boolean | 恒 `false`（无锁推功能） |
| `isModerator` | boolean | 等于 `users.is_admin` |
| `isAdmin` | boolean | 等于 `users.is_admin` |
| `isSilenced` | boolean | 恒 `false` |
| `isSuspended` | boolean | 等于 `users.is_suspended` |
| `description` | string \| null | 取 `user_profiles.description`，空则 `null` |
| `location` | null | 恒 `null` |
| `birthday` | null | 恒 `null` |
| `lang` | null | 恒 `null` |
| `fields` | array | 恒 `[]` |
| `verifiedLinks` | array | 恒 `[]` |
| `followersCount` | number | `users.followers_count`（读库失败时保留 0） |
| `followingCount` | number | `users.following_count` |
| `notesCount` | number | 现算：`deleted_at IS NULL AND local_only = FALSE AND visibility IN ('public','home')` |
| `pinnedNoteIds` | array | 恒 `[]` |
| `pinnedNotes` | array | 恒 `[]`（admin 端点不填） |
| `pinnedPageId` | null | 恒 `null` |
| `pinnedPage` | null | 恒 `null` |
| `publicReactions` | boolean | 恒 `false` |
| `followersVisibility` | string | 取 profile，空则 `"public"` |
| `followingVisibility` | string | 取 profile，空则 `"public"` |
| `chatScope` | string | 恒 `"followers"` |
| `canChat` | boolean | 恒 `false` |
| `roles` | array | 恒 `[]` |
| `memo` | null | 恒 `null`（管理员备注未实现） |
| `isFollowing` | boolean | 恒 `false`（admin 端点不针对调用者填充） |
| `isFollowed` | boolean | 恒 `false` |
| `hasPendingFollowRequestFromYou` | boolean | 恒 `false` |
| `hasPendingFollowRequestToYou` | boolean | 恒 `false` |

与 Misskey `UserDetailedNotMeOnly` 相比**未返回**的字段（前端不要读）：`moderationNote`、`isBlocking`、
`isBlocked`、`isMuted`、`isRenoteMuted`、`notify`、`withReplies`、`twoFactorEnabled`、
`usePasswordLessLogin`、`securityKeys`。

UserLite 部分也有简化：`name`/`avatarUrl` 只从 profile 行来，没有 blurhash、没有 `instance` 信息。

## 2026-11-06 增补：管理台用户筛选与文件盘点

### POST /api/admin/show-users（参数扩展）

原有 `limit` / `offset` 之外新增（全部可选，缺省时行为与原实现一致）：

| 参数 | 取值 | 说明 |
|------|------|------|
| `sort` | `+follower` `-follower` `+createdAt` `-createdAt` `+updatedAt` `-updatedAt` | `+` 为倒序（最新/最多在前） |
| `state` | `all` `alive` `available` `admin` `moderator` `adminOrModerator` `silenced` `suspended` | `alive` 退化为「未冻结」；`silenced` 恒为空集 |
| `origin` | `combined` `local` `remote` | 缺省 combined |
| `username` | string | 小写前缀匹配 |
| `hostname` | string | 精确匹配，缺省表示不限 |

返回 `UserDetailed[]`（无总数）。默认排序是按 `created_at ASC`。非法 `state`/`origin` 返回 400。

### POST /api/admin/drive/files（新）

管理员/版主专用，跨账号列出文件，按 id 倒序。

| 参数 | 说明 |
|------|------|
| `limit` | 默认 10，上限 100 |
| `sinceId` / `untilId` | 文件 id 游标 |
| `userId` | 只看某个账号；给出后忽略 origin/hostname |
| `type` | 内容类型前缀：`image`、`image/png`、`image/*`；其它字符一律 400 |
| `origin` | combined / local / remote（按上传者所在实例） |
| `hostname` | 上传者实例精确匹配 |

返回 `DriveFile[]`，每条额外带 `user`（UserLite，上传者）。非管理员返回 403。

## 2026-11-06 增补：实例公告

迁移 v10 新增 `announcements` 表；`announcements` 与 `admin/announcements/*` 四个端点共用它。

### POST /api/announcements（公开）

| 参数 | 说明 |
|------|------|
| `limit` | 默认 10，上限 100 |
| `untilId` | 公告 id 游标 |

返回 `Announcement[]`，只含 `is_active = TRUE` 且未过期的行，id 倒序。`isRead` 恒为 false
（没有已读跟踪）。

### POST /api/admin/announcements/create（管理员）

| 参数 | 校验 |
|------|------|
| `title` | 必填，≤256 字符（去首尾空白后非空） |
| `text` | 必填，≤10000 字符 |
| `imageUrl` | 可选，≤500 字符；仅 `http://`、`https://` 或 `/` 开头 |
| `expiresAt` | 可选，Unix 毫秒，必须晚于当前时间 |

返回创建的 `Announcement`。非管理员 403，未登录 401。

### POST /api/admin/announcements/list（管理员）

`limit` / `untilId`；返回全部公告（含已过期与停用），id 倒序。

### POST /api/admin/announcements/delete（管理员）

`{ id }`，返回 204；不存在的 id 也是 204（幂等）。

## 2026-11-06 增补：举报（report-abuse）

迁移 v11 新增 `abuse_reports`（举报者、被举报者、说明、resolved、时间）。

### POST /api/users/report-abuse（登录）

| 参数 | 校验 |
|------|------|
| `userId` | 必填；不存在 404；等于自己 400 |
| `comment` | 必填，去空白后 1–2048 字符 |

返回 204。管理员也可以调用，但用户菜单对管理员不显示举报项（管理员用冻结入口）。

### POST /api/admin/abuse-user-reports（管理员）

| 参数 | 说明 |
|------|------|
| `limit` | 默认 10，上限 100 |
| `sinceId` / `untilId` | 举报 id 游标 |
| `state` | `resolved` / `unresolved` / `null`（全部） |
| `reporterOrigin` / `targetUserOrigin` | combined / local / remote |

返回数组：`{ id, targetUser, reporter, comment, resolved, createdAt, forwarded: false }`，
用户为 `UserDetailed`。`state` 非法 400，非管理员 403。

### POST /api/admin/resolve-abuse-user-report（管理员）

`{ reportId }`，标记 `resolved = TRUE` 并返回 204；未知 id 也是 204（幂等）。
`forward` 未实现，请求里带 `forward` 也不会转发。
