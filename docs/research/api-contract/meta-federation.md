# API 对接契约：meta-federation

本文覆盖 Cinderfish 后端的**实例元信息与联邦（ActivityPub）**命名空间：`meta`、`ping`、WebFinger、host-meta、NodeInfo、ActivityPub 的 actor / inbox / outbox / followers / following / featured / note 端点，以及全站共享的错误信封、404 行为、CORS 与令牌读取语义。目标读者是重新编写 Firefish 风格前端的开发者，因此每条结论都以**本仓库实际 handler 代码**为准，而不是 Misskey 文档。

- 路由注册：`internal/httpapi/server.go:107`–`224`
- Handler：`internal/httpapi/meta.go`、`ping.go`、`activitypub.go`、`auth.go`、`web.go`
- 服务层：`internal/federation/render.go`（AP 文档渲染）、`policy.go`（三模式与白名单）、`store.go`（关注计数 / 关注 URI 列表 / 密钥）、`inbox.go`（入站活动）、`mfm.go`（MFM 转 HTML）
- 与 Misskey 的差距参照：`docs/research/2026-11-04-misskey-api-coverage.md`、`docs/HANDOVER.md` §12

**本命名空间的形态与 `/api/*` 其余命名空间不同**：只有 `meta` 与 `ping` 是 Misskey 风格的 JSON RPC；其余全部是联邦协议端点（GET 为主，返回 ActivityPub / WebFinger / NodeInfo 文档，**错误不是 JSON 信封**）。前端对接时只有两类东西会用到：`meta` / `ping`（启动时读取实例状态），以及 `/@user`、`/notes/{id}`（作为链接目标，不需要前端自己去 fetch）。

---

## 端点总览

| 端点 | 方法 | 认证 | 联邦开关 | 成功响应 | 实现位置 |
|------|------|------|----------|----------|----------|
| `/api/meta` | POST | 不需要 | **不受影响**（永远 200） | `MetaLite` / `MetaDetailed` 对象 | `internal/httpapi/meta.go:11` |
| `/api/ping` | POST / GET | 不需要 | 不受影响 | `{"pong": <毫秒时间戳>}` | `internal/httpapi/ping.go:13` |
| `/.well-known/webfinger` | GET | 不需要 | `none` → 403 | JRD（`application/jrd+json`） | `internal/httpapi/activitypub.go:85` |
| `/.well-known/host-meta` | GET | 不需要 | `none` → 403 | XRD XML | `internal/httpapi/activitypub.go:179` |
| `/.well-known/host-meta.json` | GET | 不需要 | `none` → 403 | `{"links":[...]}` | `internal/httpapi/activitypub.go:161` |
| `/.well-known/nodeinfo` | GET | 不需要 | `none` → 403 | `{"links":[2.1, 2.0]}` | `internal/httpapi/activitypub.go:192` |
| `/nodeinfo/2.1` | GET | 不需要 | `none` → 403 | NodeInfo 2.1 文档 | `internal/httpapi/activitypub.go:208` |
| `/nodeinfo/2.0` | GET | 不需要 | `none` → 403 | NodeInfo 2.0 文档（字段同 2.1） | `internal/httpapi/activitypub.go:213` |
| `/.well-known/*` | OPTIONS | 不需要 | **不受影响** | `204` 空体 | `internal/httpapi/activitypub.go:257` |
| `/users/{id}` | GET | 不需要 | `none` → 403 | Actor（`Person`） | `internal/httpapi/activitypub.go:265` |
| `/@{acct}` | GET | 不需要 | `none` → 403 | Actor（`Person`） | `internal/httpapi/activitypub.go:284` |
| `/users/{id}/publickey` | GET | 不需要 | `none` → 403 | `Key` 对象 | `internal/httpapi/activitypub.go:333` |
| `/users/{id}/inbox` | GET | 不需要 | `none` → 403 | 空 `OrderedCollection` 桩 | `internal/httpapi/activitypub.go:353` |
| `/inbox` | POST | HTTP Signature | `none` → 403 | `202` 空体 | `internal/httpapi/activitypub.go:581` |
| `/users/{id}/inbox` | POST | HTTP Signature | `none` → 403 | `202` 空体（与 `/inbox` 同一 handler） | `internal/httpapi/activitypub.go:581` |
| `/users/{id}/outbox` | GET | 不需要 | `none` → 403 | `OrderedCollection` / `OrderedCollectionPage` | `internal/httpapi/activitypub.go:370` |
| `/users/{id}/followers` | GET | 不需要 | `none` → 403 | `OrderedCollection` / Page | `internal/httpapi/activitypub.go:411` |
| `/users/{id}/following` | GET | 不需要 | `none` → 403 | `OrderedCollection` / Page | `internal/httpapi/activitypub.go:416` |
| `/users/{id}/collections/featured` | GET | 不需要 | `none` → 403 | 空 `OrderedCollection` | `internal/httpapi/activitypub.go:468` |
| `/notes/{id}` | GET | 不需要 | `none` → 403 | AP `Note` | `internal/httpapi/activitypub.go:483` |
| `/notes/{id}/activity` | GET | 不需要 | `none` → 403 | AP `Create` 活动 | `internal/httpapi/activitypub.go:508` |
| 其它任意路径 | * | — | — | `404 NO_SUCH_ENDPOINT`（JSON） | `internal/httpapi/server.go:327` |

**“联邦开关”列的含义**：这些路由先查 `federationEnabled`（`internal/httpapi/activitypub.go:65`），它读的是数据库 `instance_meta.federationMode`（运行时值），不是配置文件。模式为 `none` 时统一回 `403` 纯文本 `Forbidden`。

`/.well-known/oauth-authorization-server`、`/oauth/*` 属于 auth-users 命名空间，本文不涉及。

---

## 通用约定

### 1. 认证：令牌怎么读

`Server.authenticate`（`internal/httpapi/auth.go:30`）按以下顺序取令牌：

1. 请求体的 `i` 字段（必须是字符串，`internal/httpapi/auth.go:32`–`36`）；
2. `Authorization: Bearer <token>` 头（大小写不敏感的前缀匹配，`internal/httpapi/auth.go:37`–`43`）。

**两者都给时以请求体 `i` 为准**——这一点与 Misskey **相反**：Misskey 先看 `Authorization` 头，头不存在时才退回 body 的 `i`（`packages/backend/src/server/api/ApiCallService.ts:204`–`206`）。前端只要保证两处不同时发令牌即可，但排障时要知道本实现是 body 优先。

- 没有令牌 → `authenticate` 返回 `(nil, nil)`，即匿名；由各 handler 自己决定是否必须登录。
- 令牌无效 / 已吊销 → `writeAuthenticationError`，HTTP 401，`code: AUTHENTICATION_FAILED`。
- **账户被停用**（`users.is_suspended = true`）时，即使令牌有效也当作认证失败返回 401（`internal/httpapi/auth.go:52`–`55`）。
- 原生用户令牌（16 字符）查 `users.token`，其余令牌查 `access_tokens.token`（`internal/account/tokens.go:88`–`126`）。两种令牌在 API 上等价。
- **`i` 不是字符串**（例如数字）时，本实现静默忽略它（`body["i"].(string)` 断言失败，`internal/httpapi/auth.go:33`），随后退回 `Authorization` 头；两者都没有才算匿名。Misskey 遇到非字符串 `i` 会直接返回 400（`ApiCallService.ts:206`–`209`）。
- **不支持 GET 查询串里的 `i`**。Misskey 在 GET 请求上从 query 取令牌（`ApiCallService.ts:198`–`206`），本实现的 `authenticate` 只看 body 与 Header。本命名空间的 GET 端点全部匿名，因此不受影响。
- **没有全局鉴权中间件**：`ServeHTTP`（`internal/httpapi/server.go:226`）只做“SPA 静态文件 → mux”的分发，鉴权是每个 handler 自己调用的。所以“某个端点需不需要登录”只能逐端点看代码，没有统一开关。

### 2. 令牌作用域（scope）

`access_tokens.scopes` 会写入数据库（OAuth 流程发放的令牌带 `read`/`write`/`follow`/`upload`），但 **`/api/*` 的 handler 从不检查 scope**。唯一的 scope 校验在 Streaming 的 `read:account` 检查（`internal/httpapi/streaming_adapters.go:24`–`40`）。因此：只读令牌同样能调用写端点。这是已知缺口，不是前端可依赖的行为。

### 3. 请求体解码

`decodeOptionalJSON`（`internal/httpapi/ping.go:23`）：

- 请求体上限 **8 MiB**（`io.LimitReader(r.Body, 8<<20)`）。
- 空 body 合法，视为“全部参数缺省”。
- JSON 非法 → `400 INVALID_PARAM`，`info` 为 `{"reason":"body is not valid JSON"}`。
- 未知字段被静默忽略。
- 注意：`POST /api/meta` 是本命名空间里**唯一忽略解码错误**的端点（`internal/httpapi/meta.go:15` 没有检查返回值），非法 JSON 仍返回 200 的默认 meta。

### 4. 响应编码

- `writeJSON`（`internal/httpapi/server.go:235`）：`Content-Type: application/json; charset=utf-8`，`SetEscapeHTML(false)`。
- 对象键名 camelCase；空数组写作 `[]`，空值写作 `null`，不会出现字段缺失与 `null` 混用的情况（`meta` 的字段是**全量固定输出**的，见下）。

### 5. 错误信封（全站共享）

`APIError` + `writeAPIError`（`internal/httpapi/server.go:246`、`269`）：

```json
{ "error": { "message": "...", "code": "NO_SUCH_ENDPOINT", "id": "uuid", "kind": "client", "info": ... } }
```

- `kind` 恒存在，取值 `client` / `server` / `permission`。
- `info` 只在部分校验错误里出现，**类型不稳定**：有时是字符串（`"mode must be public, allowlist or none"`），有时是对象（`{"reason": "..."}`）。前端应把它当不透明值。
- HTTP 状态码由 handler 指定，默认 `400`。
- 401 时 `writeAPIError` 会补 `WWW-Authenticate: Bearer realm="Misskey"`（`internal/httpapi/server.go:283`–`285`）。
- **已知缺陷**：`writeAuthenticationError` 先写入带 `error="invalid_token"`、`error_description="..."` 的完整 `WWW-Authenticate`（`internal/httpapi/server.go:305`），随后 `writeAPIError` 用 `Header().Set` 覆盖成了简版 `Bearer realm="Misskey"`。线上实际收到的头**只有简版**，客户端读不到 `error_description`。

常用错误 id：

| code | id | kind | 说明 |
|------|----|------|------|
| `INVALID_PARAM` | `3d81ceae-475f-4600-b2a8-2bc116157532` | client | 参数校验失败 |
| `AUTHENTICATION_FAILED` | `b0a7f5f8-dc2f-4171-b91f-de88ad238e14` | client | 未登录 / 令牌无效 / 账户停用 |
| `INTERNAL_ERROR` | `5d37dbcb-891e-41ca-a3d6-e690c97775ac` | server | 服务端错误 |
| `NO_SUCH_ENDPOINT` | `fc9c4a0b-f3a1-4b0e-9a8c-6f7f3d0f6a12` | client | 路由不存在（**本实现自定义**，Misskey 没有这个 id） |
| `FORBIDDEN` | `f4f0d9f1-3a2e-4a8b-9c7d-2e1b0f6a8c53` | permission | 非管理员调用管理端点（**本实现自定义**） |

### 6. 404 行为：JSON 与纯文本并存

这是对接时最容易踩的一点。全站有**三种** 404 / 403 形态：

| 场景 | 状态 | Content-Type | Body |
|------|------|--------------|------|
| 未匹配任何路由（含方法不匹配） | 404 | `application/json` | `{"error":{"message":"No such endpoint: <METHOD> <PATH>","code":"NO_SUCH_ENDPOINT",...}}`（`internal/httpapi/server.go:327`） |
| AP 路由找不到对象（`/users/{id}`、`/notes/{id}` 等） | 404 | `text/plain` | `404 page not found`（`http.NotFound`） |
| AP 路由被联邦开关挡住 | 403 | `text/plain` | `Forbidden`（`internal/httpapi/activitypub.go:61`） |

- 因为注册了兜底 `/` 路由，**方法不匹配也不会得到 405**，而是上面第一条的 JSON 404。例如 `GET /api/meta` → `404 NO_SUCH_ENDPOINT`。
- `GET /{acct}`（`internal/httpapi/server.go:217`）是单段路径的兜底：**单段** GET 路径（`/nodeinfo`、`/.well-known`；或未挂 SPA 时的 `/favicon.ico` 等）会进入 `handleActorByAcct`，其中不以 `@` 开头的路径直接 `http.NotFound`，即**纯文本 404**。多段路径（`/api/nope`）才落到 JSON 404。挂了 SPA 时，不属于协议前缀的单段 GET（如 `/timeline`）先被前端接走，不会走到这里。

### 7. CORS

- **`/api/*` 完全没有 CORS 头**。全仓库只有 ActivityPub / well-known 相关函数设置 `Access-Control-Allow-Origin`（`internal/httpapi/activitypub.go:49`、`86`、`162`、`180`、`193`、`218`、`258`）。因此前端必须与后端**同源**（由同一个二进制托管 `web_dir`），开发期用 Vite proxy 转发。跨域直连 `/api` 会被浏览器拦掉，包括预检。
- AP 成功响应带全套：`Access-Control-Allow-Origin: *`、`Access-Control-Allow-Headers: Accept`、`Access-Control-Allow-Methods: GET, OPTIONS`、`Access-Control-Expose-Headers: Vary`（`internal/httpapi/activitypub.go:46`–`58`）。
- **AP 的错误响应（403 / 404）不带 ACAO**：这些 header 只在 `writeAP` 里设置，而 403/404 走的是 `apForbidden` / `http.NotFound`。浏览器侧读不到错误体，只能看到网络错误。
- `OPTIONS /.well-known/*` 统一回 `204`，带 `Allow-Headers: Accept`、`Allow-Methods: GET, OPTIONS`（`internal/httpapi/activitypub.go:257`–`262`），且**不受联邦模式影响**。

### 8. 内容协商与缓存（AP 端点）

- `Accept` 含 `application/ld+json` → `Content-Type: application/ld+json; profile="https://www.w3.org/ns/activitystreams"`；否则 `application/activity+json`（`internal/httpapi/activitypub.go:36`–`41`）。
- 所有 `writeAP` 响应带 `Vary: Accept` 与 `Cache-Control: public, max-age=180`。
- NodeInfo 文档的 `Content-Type` 附带 profile：`application/json; profile="http://nodeinfo.diaspora.software/ns/schema/2.1#"`（`internal/httpapi/activitypub.go:247`），缓存 600 秒。
- WebFinger / host-meta / nodeinfo-links 缓存 180 秒。

### 9. 分页与游标

本命名空间唯一的分页在三个 collection 上，**语义与 `/api` 的 `limit`/`sinceId`/`untilId` 完全不同**：

| 端点 | 参数 | 默认 | 上限 | 游标基准 |
|------|------|------|------|----------|
| `/users/{id}/outbox?page=true` | `since_id`、`until_id` | 20（写死在调用处） | 无外部可调 | 笔记 ULID（`notes.id`），降序 |
| `/users/{id}/followers?page=true` | `cursor` | 10（写死） | 无外部可调 | `followings.id`（**关注关系行 id，不是用户 id**），降序 |
| `/users/{id}/following?page=true` | `cursor` | 10（写死） | 无外部可调 | 同上 |

- 参数名是 **snake_case**（`since_id` / `until_id` / `page` / `cursor`），不是 `/api` 的 camelCase。
- 不传 `page=true` 时返回 collection（含 `totalItems` 与 `first` 链接），传了才返回 page。
- 两个分页端点都**不返回 `next`/`prev`**，客户端得自己拼 URL（把最后一项的 `followings.id` 当 cursor 是不行的——page 里并不返回该 id，见各端点小节的限制）。

### 10. 时间与 ID 格式

- AP 文档里的 `published`：Go `time.RFC3339`（秒级，无小数），UTC，例如 `2026-11-04T12:34:56Z`（`internal/federation/render.go:111`、`211`）。
- `/api/meta` 与 `/api/ping` 不含时间字段；`ping.pong` 是**毫秒时间戳**（int64）。
- 用户 id 与笔记 id 都是 18 字符 base36 字符串（`internal/idgen`，Misskey `aid` 风格：前 10 位毫秒 + 后 8 位随机），不是带连字符的 ULID。actor URI 形如 `https://<domain>/users/<id>`，handle URI 形如 `https://<domain>/@<username>`，笔记 URI 形如 `https://<domain>/notes/<id>`（`internal/federation/render.go:25`–`42`）。
- base URL 的来源（`internal/httpapi/activitypub.go:22`–`32`）：配置了 `federation.public_url` 就用它；否则 `https://<domain>`，**当 domain 以 `localhost` 或 `127.0.0.1` 开头时改用 `http://`**。

### 11. SPA 与 Go mux 的分工

前端托管在同一个进程里时，`ServeHTTP` 先问 SPA 要不要处理（`internal/httpapi/web.go:40`–`57`）：

- `/api`、`/media`、`/.well-known`、`/oauth`、`/users`、`/nodeinfo`、`/inbox`、`/streaming` 前缀**永远**交给 Go mux。
- `/@user` 与 `/notes/{id}`：`Accept` 含 `text/html` 时给前端（浏览器直接打开 → 前端路由渲染），否则给 ActivityPub（联邦抓取）。
- 其余 GET/HEAD：先找真实文件，找不到回退 `index.html`（`internal/httpapi/web.go:59`–`81`），`/assets/*` 带一年 immutable 缓存，`index.html` 每次读盘且 `Cache-Control: no-cache`。

因此前端可以放心使用 `/@user`、`/notes/{id}` 作为浏览器路由；但**不要**在 `fetch` 里带 `Accept: text/html` 去请求它们，否则拿到的是 HTML 外壳而不是 AP 文档。

---

## 共享对象

### Actor（`Person`）

`Renderer.RenderPerson`（`internal/federation/render.go:73`–`137`），`GET /users/{id}` 与 `GET /@{acct}` 返回它。所有 AP 文档都带 `@context`（见下）。

| 字段 | 类型 | 说明 |
|------|------|------|
| `@context` | array | Misskey 风格上下文（`internal/federation/apobject.go:223`–`257`） |
| `id` | string | `https://<base>/users/<id>` |
| `type` | string | 恒为 `Person`（见限制；`Application`/`Service` 分支实际不可达） |
| `inbox` | string | `<id>/inbox` |
| `outbox` | string | `<id>/outbox` |
| `followers` | string | `<id>/followers` |
| `following` | string | `<id>/following` |
| `featured` | string | `<id>/collections/featured` |
| `sharedInbox` | string | `https://<base>/inbox`（顶层，非标准位置） |
| `endpoints` | object | `{"sharedInbox": "https://<base>/inbox"}`（标准位置） |
| `url` | string | `https://<base>/@<username>`（HTML profile 页） |
| `preferredUsername` | string | 用户名（不含 domain） |
| `name` | string | 显示名，来自 `user_profiles.name`，可为 `""` |
| `summary` | string | 简介经 MFM→HTML 转换（`internal/federation/mfm.go:55`） |
| `_misskey_summary` | string | 简介原文（MFM） |
| `manuallyApprovesFollowers` | bool | 实际恒为 `false`（见限制） |
| `discoverable` | bool | 恒为 `true` |
| `publicKey` | object | `{id: "<id>#main-key", type: "Key", owner: "<id>", publicKeyPem: "..."}` |
| `published` | string | 注册时间，RFC3339；`created_at` 为零值时**该字段缺失** |
| `icon` | object \| null | 头像，实际恒为 `null`（见限制） |
| `image` | object \| null | 横幅，实际恒为 `null`（见限制） |
| `attachment` | array | 资料字段 `PropertyValue[]`；**实际恒不出现**（`Fields` 从未被填充） |

`renderLocalActor`（`internal/httpapi/activitypub.go:315`–`330`）只查询 `user_profiles` 的 `name`、`description` 两列，并把 `Profile.IsBot` / `IsLocked` / `AvatarURL` / `BannerURL` / `Fields` 留空，所以上表中标记“实际恒为”的字段是当前的真实行为，不是文档推测。

**副作用**：首次访问某个本地用户的 actor（或 publickey、或首次向外投递）会自动生成并入库 RSA-2048 密钥对（`internal/federation/store.go:245`–`268`）。这是按需生成，不是注册时生成。

### Note（AP）

`Renderer.RenderNote`（`internal/federation/render.go:180`–`259`），`GET /notes/{id}`、`/notes/{id}/activity` 与 outbox 条目内嵌使用。

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | `https://<base>/notes/<ulid>` |
| `type` | string | 恒为 `Note` |
| `attributedTo` | string | 作者 actor URI |
| `content` | string | 正文 HTML，由 MFM 转换：提及、话题标签、自定义 emoji、裸链接、`**粗体**`、`_斜体_`、行内代码、围栏代码块；其余一律 HTML 转义 |
| `to` / `cc` | string[] | 见下方可见性映射 |
| `sensitive` | bool | 来自 `notes.is_sensitive` |
| `published` | string | RFC3339 |
| `_misskey_content` | string | 正文原文（MFM） |
| `source` | object | `{"content": "<MFM>", "mediaType": "text/x.misskeymarkdown"}` |
| `summary` | string | 仅在 `cw` 非空时出现（内容警告） |
| `inReplyTo` | string | 仅在是回复时出现 |
| `_misskey_quote` / `quoteUrl` | string | 仅在转贴/引用时出现，值为被引用笔记 URI |
| `tag` | array | `Hashtag`（`{type, href: "<base>/tags/<tag>", name: "#<tag>"}`）与 `Mention`（`{type, href: "<actor uri>"}`）；无标签时该字段不出现 |
| `attachment` | array | **本地笔记恒不出现**（见限制） |

可见性映射（`internal/federation/render.go:189`–`201`）：

| `notes.visibility` | `to` | `cc` |
|--------------------|------|------|
| `public` | `[Public]` | `[followers] + mentions` |
| `home` | `[followers]` | `[Public] + mentions` |
| `followers` | `[followers]` | `mentions` |
| 其它（`specified`） | `mentions` | `[]` |

`Public = "https://www.w3.org/ns/activitystreams#Public"`。

`Create` 活动（`internal/federation/render.go:264`–`279`）：`{@context, id: "<note uri>/activity", type: "Create", actor, published, object: <Note>, to, cc}`，`to`/`cc` 从内嵌 note 复制。

### Key

`GET /users/{id}/publickey` 返回（`internal/federation/render.go:402`–`409`）：`{@context, id: "<actor>#main-key", type: "Key", owner: "<actor>", publicKeyPem: "-----BEGIN PUBLIC KEY-----..."}`。

### OrderedCollection / OrderedCollectionPage

- Collection（`internal/federation/render.go:370`–`380`）：`{@context, id, type: "OrderedCollection", totalItems, first?}`。`first` 只在非空时出现——`featured` 传了空串，所以**没有 `first`**。
- Page（`internal/federation/render.go:383`–`398`）：`{@context, id, type: "OrderedCollectionPage", partOf, totalItems, orderedItems, prev?, next?}`。**本实现从不填 `prev`/`next`**。
- Page 的 `totalItems` 是**本页条数**，不是整个集合的总数（调用处传的是 `len(items)`）。

### `@context`

固定的一组 IRI + 内联扩展对象（`internal/federation/apobject.go:223`–`257`）：`https://www.w3.org/ns/activitystreams`、`https://w3id.org/security/v1`，以及定义 `_misskey_content`、`_misskey_quote`、`_misskey_reaction`、`_misskey_summary`、`manuallyApprovesFollowers`、`sensitive`、`Hashtag`、`quoteUrl`、`toot:`、`schema:`、`misskey:` 等前缀的内联映射。所有 AP 文档都由 `withContext` 包装（`internal/federation/apobject.go:260`–`266`），`@context` 总是第一个键。

---

## POST /api/meta

**别名**：无（只有 `POST`，`GET /api/meta` 会得到 `404 NO_SUCH_ENDPOINT`）。
**认证**：不需要（匿名可调用，带不带令牌结果相同）。
**联邦开关**：不受影响，`none` 模式下也正常返回。
**源码**：`internal/httpapi/meta.go:11`；路由 `internal/httpapi/server.go:108`；测试 `internal/httpapi/server_test.go:32`、`76`。

### 请求

| 字段 | 类型 | 必填 | 默认 | 说明 |
|------|------|------|------|------|
| `detail` | bool | 否 | `true` | `true` → `MetaDetailed`；`false` → `MetaLite`（少 5 个字段） |

空 body 合法，等价于 `{"detail": true}`。非法 JSON 被忽略，仍返回详细版（`internal/httpapi/meta.go:15` 未检查解码错误）。

### 响应

两个变体共用同一套 55 个字段，`detail` 版额外多 5 个。**没有任何字段是可选的**——所有键每次都出现，空值用 `null` 或 `""`/`[]`/`{}` 表示。

| 字段 | 类型 | 实际值 / 说明 |
|------|------|----------------|
| `maintainerName` | string | 恒 `""` |
| `maintainerEmail` | string | 恒 `""` |
| `version` | string | 恒 `"0.1.0"`（与 NodeInfo 的 `software.version` 不一致，见缺口） |
| `providesTarball` | bool | 恒 `false` |
| `name` | string | 恒 `"Cinderfish"` |
| `shortName` | string | 恒 `"cinderfish"` |
| `uri` | string | `"https://" + cfg.Domain`（**恒为 https**，即使 domain 是 localhost） |
| `description` | string | 恒 `""` |
| `langs` | string[] | 恒 `["en"]` |
| `tosUrl` | null | 恒 `null` |
| `repositoryUrl` | string | 恒 `""` |
| `feedbackUrl` | string | 恒 `""` |
| `impressumUrl` | string | 恒 `""` |
| `privacyPolicyUrl` | string | 恒 `""` |
| `inquiryUrl` | string | 恒 `""` |
| `disableRegistration` | bool | `registration_mode != "open"`（即 `invite`、`admin` 均为 `true`） |
| `emailRequiredForSignup` | bool | 恒 `false` |
| `enableHcaptcha` | bool | 恒 `false` |
| `hcaptchaSiteKey` | string | 恒 `""` |
| `enableMcaptcha` | bool | 恒 `false` |
| `mcaptchaSiteKey` | string | 恒 `""` |
| `mcaptchaInstanceUrl` | string | 恒 `""` |
| `enableRecaptcha` | bool | 恒 `false` |
| `recaptchaSiteKey` | string | 恒 `""` |
| `enableTurnstile` | bool | 恒 `false` |
| `turnstileSiteKey` | string | 恒 `""` |
| `enableTestcaptcha` | bool | 恒 `false` |
| `googleAnalyticsMeasurementId` | string | 恒 `""` |
| `swPublickey` | string | 恒 `""` |
| `themeColor` | string | 运行时 `themeColor`，未设置为 `"#86b300"` |
| `mascotImageUrl` | string | 恒 `"/assets/ai.png"` |
| `bannerUrl` | string | 运行时 `bannerUrl`，未设置为 `""` |
| `infoImageUrl` | string | 恒 `""` |
| `serverErrorImageUrl` | string | 恒 `""` |
| `notFoundImageUrl` | string | 恒 `""` |
| `iconUrl` | string | 运行时 `iconUrl`，未设置为 `""` |
| `backgroundImageUrl` | string | 恒 `""` |
| `logoImageUrl` | string | 恒 `""` |
| `maxNoteTextLength` | number | 恒 `3000` |
| `defaultLightTheme` | null | 恒 `null` |
| `defaultDarkTheme` | null | 恒 `null` |
| `clientOptions` | object | 恒 `{}`（Misskey 默认为 `{}`，一致） |
| `ads` | array | 恒 `[]` |
| `notesPerOneAd` | number | 恒 `0` |
| `enableEmail` | bool | 恒 `false` |
| `enableServiceWorker` | bool | 恒 `false` |
| `translatorAvailable` | bool | 恒 `false` |
| `serverRules` | array | 运行时 `serverRules`（迁移 v13 后的实例设置），默认 `[]` |
| `policies` | object | 只有 `canUseFederation: true`、`canCreateInstantNote: true` 两个键（`internal/httpapi/meta.go:121`–`126`） |
| `sentryForFrontend` | null | 恒 `null` |
| `mediaProxy` | string | 恒 `""` |
| `enableUrlPreview` | bool | 恒 `true` |
| `noteSearchableScope` | string | 恒 `"global"` |
| `maxFileSize` | number | 恒 `20971520`（20 MiB） |
| `federation` | string | `public` → `"all"`；`allowlist` → `"specified"`；其它 → `"none"`（`internal/httpapi/meta.go:105`–`117`） |

`detail: true` 额外字段：

| 字段 | 类型 | 实际值 / 说明 |
|------|------|----------------|
| `cacheRemoteFiles` | bool | 恒 `true`（与实现不符：远端媒体并不缓存，见缺口） |
| `cacheRemoteSensitiveFiles` | bool | 恒 `false` |
| `requireSetup` | bool | 恒 `false` |
| `proxyAccountName` | string | 恒 `""`（没有 proxy 账号这个概念） |
| `features` | object | 见下表，10 个布尔键 |

`features` 对象：

| 键 | 计算方式 |
|----|----------|
| `localTimeline` | 恒 `true` |
| `globalTimeline` | `federation_mode == "public"`（**配置文件的值**） |
| `registration` | `registration_mode == "open"` |
| `emailRequiredForSignup` | 恒 `false` |
| `hcaptcha` | 恒 `false` |
| `recaptcha` | 恒 `false` |
| `turnstile` | 恒 `false` |
| `objectStorage` | `media.backend == "s3"` |
| `serviceWorker` | 恒 `false` |
| `miauth` | 恒 `true` |

### 错误

**没有成功路径以外的错误。** 本端点不读数据库、不鉴权、连 JSON 解码失败都忽略，所以只要路由命中就是 `200`。

### 已知限制

1. **`federation` 与 `features.globalTimeline` 读的是配置文件，不是运行时值。** 管理员通过 `POST /api/admin/federation/mode` 切换模式只写数据库 `instance_meta`（`internal/httpapi/federation_admin.go`），`meta` 的这两处仍用 `s.cfg`（`internal/httpapi/meta.go:78`、`92`），重启前一直报旧值。**而 AP 路由读的是数据库**（`internal/federation/policy.go:86`–`96`），两者会不一致。需要运行期值时应读 `GET /api/admin/federation/mode`（2026-11-06 新增，管理员端点）；普通客户端仍只能从 AP 路由是否 403 推断。
2. **`policies` 只有 2 个键。** Misskey 的 `DEFAULT_POLICIES` 有 40 多个（`gtlAvailable`、`ltlAvailable`、`canPublicNote`、`pinLimit`、`driveCapacityMb`、`maxFileSizeMb`、`uploadableFileTypes` 等，见 `packages/backend/src/core/RoleService.ts:78`–`124`）。Firefish 客户端若读 `policies.canPublicNote` 之类会拿到 `undefined`，需要前端兜底默认值。
3. **`version` 是 `"0.1.0"`，NodeInfo 的 `software.version` 是 `"0.0.1"`**，两处写死且不同。
4. **`uri` 恒为 `https://`**，而 `baseURL()`（AP 用）在 localhost 下会降级成 `http://`。本地开发时 `meta.uri` 与 actor URI 的 scheme 不同。
5. **`cacheRemoteFiles: true` 与实际行为矛盾**：远端媒体不缓存也不代理（`docs/HANDOVER.md` §12.4）。
6. 站点名、描述、maintainer、主题色全部写死，没有 `admin/meta/update` 端点。

---

## GET /api/ping

**别名**：同一 handler 同时注册了 `POST /api/ping` 与 `GET /api/ping`（`internal/httpapi/server.go:109`–`110`）。
**认证**：不需要。**联邦开关**：不受影响。
**源码**：`internal/httpapi/ping.go:13`；测试 `internal/httpapi/server_test.go:93`。

### 请求

无参数。请求体与查询串都被完全忽略（handler 连 `r` 都不读）。

### 响应

```json
{ "pong": 1762262400123 }
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `pong` | number | 服务端当前时间的 Unix **毫秒**时间戳（`time.Now().UnixMilli()`），不是请求回显 |

### 错误

无。

### 已知限制

与 Misskey 一致（Misskey 的 `ping` 同样返回服务端时间戳）。仅此一个端点支持 GET，其余 `/api/*` 只接受 POST。

---

## GET /.well-known/webfinger

**别名**：无。**认证**：不需要。**联邦开关**：`none` → `403 Forbidden`。
**源码**：`internal/httpapi/activitypub.go:85`–`149`；测试 `internal/httpapi/activitypub_test.go:90`、`128`。

### 请求

| 参数（query） | 类型 | 必填 | 说明 |
|---------------|------|------|------|
| `resource` | string | 是 | 支持 4 种形态，见下 |

`resource` 的解析顺序（`internal/httpapi/activitypub.go:125`–`149`）：

1. `<base>/users/<id>`（大小写不敏感的 base 前缀）→ 按 id 查本地用户；
2. `<base>/@<username>` → 按用户名查；
3. 去掉 `acct:` 前缀后按 `<user>@<host>` 或 `<user>` 处理；
4. 带 host 且 host 不是本机 domain → `422`；
5. 纯 `<user>`（无 host）→ 按用户名查。

`<base>` 是 `federation.public_url` 或 `https://<domain>`（localhost 下为 `http://`），见通用约定 §10。只接受本地用户（`users.host IS NULL`）。

### 响应

`Content-Type: application/jrd+json`，`Vary: Accept`，`Cache-Control: public, max-age=180`，`Access-Control-Allow-Origin: *`。

```json
{
  "subject": "acct:alice@fish.example",
  "aliases": ["https://fish.example/users/01J..."],
  "links": [
    {"rel": "self", "type": "application/activity+json", "href": "https://fish.example/users/01J..."},
    {"rel": "http://webfinger.net/rel/profile-page", "type": "text/html", "href": "https://fish.example/@alice"},
    {"rel": "http://ostatus.org/schema/1.0/subscribe", "template": "https://fish.example/authorize-follow?acct={uri}"}
  ]
}
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `subject` | string | `acct:<username>@<cfg.Domain>`（原样大小写，不 lowercase） |
| `aliases` | string[] | 只有一项：actor URI。**Misskey 不返回 `aliases`**，这是超集 |
| `links[0]` | object | `rel: self`，`type: application/activity+json`，`href: <base>/users/<id>` |
| `links[1]` | object | `rel: http://webfinger.net/rel/profile-page`，`type: text/html`，`href: <base>/@<username>` |
| `links[2]` | object | `rel: http://ostatus.org/schema/1.0/subscribe`，`template: <base>/authorize-follow?acct={uri}`（本实现没有这个路由，纯参照上游） |

### 错误

| 状态 | Body | 触发条件 |
|------|------|----------|
| `403` | `Forbidden`（纯文本） | 联邦模式为 `none`（在解析 `resource` 之前检查） |
| `400` | `Bad Request`（纯文本） | `resource` 缺失或为空串 |
| `422` | `Unprocessable Entity`（纯文本） | `resource` 指向别的主机 |
| `404` | `Not Found`（纯文本） | 本机没有这个用户 |

### 已知限制

1. **只实现 JRD，不实现 XRD 形式的 webfinger**：即使 `Accept: application/xrd+xml` 也返回 JRD（Misskey 会按 `Accept` 切换，`packages/backend/src/server/WellKnownServerService.ts:186`）。
2. **不过滤停用账户**：`users.is_suspended = true` 的用户仍能被解析出来（Misskey 的查询带 `isSuspended: false`）。
3. `subject` 用 `cfg.Domain`，而资源前缀匹配用 `baseURL()`。若 `federation.public_url` 与 `domain` 不同（例如反代域名），`acct:user@public.example` 会因为 host 比较用的是 `domain` 而返回 `422`。

---

## GET /.well-known/host-meta.json

**别名**：无。**认证**：不需要。**联邦开关**：`none` → 403。
**源码**：`internal/httpapi/activitypub.go:161`–`176`；测试 `internal/httpapi/activitypub_test.go:275`。

### 请求

无参数。

### 响应

`Content-Type: application/json`，`Cache-Control: public, max-age=180`，`Access-Control-Allow-Origin: *`。

```json
{ "links": [{ "rel": "lrdd", "type": "application/jrd+json", "template": "https://fish.example/.well-known/webfinger?resource={uri}" }] }
```

无其它字段。

### 错误

`403 Forbidden`（纯文本）当联邦模式为 `none`。

### 已知限制

无。

---

## GET /.well-known/host-meta

**别名**：XRD 版本（与 `host-meta.json` 是不同路由，不是内容协商）。
**认证**：不需要。**联邦开关**：`none` → 403。
**源码**：`internal/httpapi/activitypub.go:179`–`189`；测试 `internal/httpapi/activitypub_test.go:275`。

### 请求

无参数。

### 响应

`Content-Type: application/xrd+xml`，`Cache-Control: public, max-age=180`，`Access-Control-Allow-Origin: *`。

```xml
<?xml version="1.0" encoding="UTF-8"?><XRD xmlns="http://docs.oasis-open.org/ns/xri/xrd-1.0"><Link rel="lrdd" type="application/xrd+xml" template="https://fish.example/.well-known/webfinger?resource={uri}"/></XRD>
```

注意 XML 版的 `type` 是 `application/xrd+xml`，JSON 版是 `application/jrd+json`——与 Misskey 相同。

### 错误

`403 Forbidden`（纯文本）当联邦模式为 `none`。

### 已知限制

base URL 直接字符串拼接进 XML，没有做 XML 转义；因为值来自配置而非请求，实际不可注入。

---

## GET /.well-known/nodeinfo

**别名**：无。**认证**：不需要。**联邦开关**：`none` → 403。
**源码**：`internal/httpapi/activitypub.go:192`–`205`；测试 `internal/httpapi/activitypub_test.go:234`。

### 请求

无参数。

### 响应

`Content-Type: application/json`，`Cache-Control: public, max-age=180`，`Access-Control-Allow-Origin: *`。

```json
{ "links": [
  { "rel": "http://nodeinfo.diaspora.software/ns/schema/2.1", "href": "https://fish.example/nodeinfo/2.1" },
  { "rel": "http://nodeinfo.diaspora.software/ns/schema/2.0", "href": "https://fish.example/nodeinfo/2.0" }
] }
```

顺序固定：2.1 在前，2.0 在后。

### 错误

`403 Forbidden`（纯文本）当联邦模式为 `none`。

### 已知限制

无。

---

## GET /nodeinfo/2.1

**别名**：`GET /nodeinfo/2.0` 为同一实现，只有 `version` 字段不同。
**认证**：不需要。**联邦开关**：`none` → 403。
**源码**：`internal/httpapi/activitypub.go:208`（2.1）、`213`（2.0）、公共实现 `serveNodeInfo` 在 `217`–`253`；测试 `internal/httpapi/activitypub_test.go:234`。

### 请求

无参数。

### 响应

`Content-Type: application/json; profile="http://nodeinfo.diaspora.software/ns/schema/2.1#"`，`Cache-Control: public, max-age=600`，`Access-Control-Allow-Origin: *`。

| 字段 | 类型 | 说明 |
|------|------|------|
| `version` | string | `"2.1"`（2.0 端点为 `"2.0"`） |
| `software.name` | string | 恒 `"cinderfish"` |
| `software.version` | string | 恒 `"0.0.1"` |
| `protocols` | string[] | 恒 `["activitypub"]` |
| `services.inbound` | string[] | 恒 `[]` |
| `services.outbound` | string[] | 恒 `[]`（Misskey 是 `["atom1.0","rss2.0"]`） |
| `openRegistrations` | bool | `registration_mode == "open"` |
| `usage.users.total` | number | `SELECT count(*) FROM users WHERE host IS NULL`（含停用账户） |
| `usage.localPosts` | number | `SELECT count(*) FROM notes WHERE user_host IS NULL AND deleted_at IS NULL`（**含所有可见性**） |
| `usage.localComments` | number | 恒 `0` |
| `metadata.maxNoteTextLength` | number | 恒 `3000` |
| `metadata.disableRegistration` | bool | `registration_mode != "open"` |

`software` 对象只有 `name` 与 `version`；`metadata` 只有 `maxNoteTextLength` 与 `disableRegistration`。

### 错误

`403 Forbidden`（纯文本）当联邦模式为 `none`。

### 已知限制

1. **与 Misskey 相比缺少大量 metadata**（对照 `packages/backend/src/server/NodeinfoServerService.ts:97`–`132`）：`nodeName`、`nodeDescription`、`nodeAdmins`、`maintainer`、`langs`、`tosUrl`、`privacyPolicyUrl`、`inquiryUrl`、`impressumUrl`、`repositoryUrl`、`feedbackUrl`、`disableLocalTimeline`、`disableGlobalTimeline`、`emailRequiredForSignup`、`enableHcaptcha`/`Recaptcha`/`Mcaptcha`/`Turnstile`、`enableEmail`、`enableServiceWorker`、`proxyAccountName`、`themeColor`——全部未返回。
2. `software.homepage` / `software.repository` 未返回。
3. `usage.users.activeHalfyear` / `activeMonth` 未返回（Misskey 返回 `null`，本实现直接缺字段）。
4. `version ≥ 2.1` 时 Misskey 会补 `software.repository`/`homepage`；本实现 2.1 与 2.0 文档除 `version` 外完全相同。
5. `usage.localPosts` 不过滤可见性，也不区分回复；`usage.users.total` 包含停用账户。

---

## GET /nodeinfo/2.0

**别名**：`GET /nodeinfo/2.1` 的 2.0 版本（同一实现 `serveNodeInfo`，只有 `version` 字段不同）。
**认证**：不需要。**联邦开关**：`none` → 403。
**源码**：`internal/httpapi/activitypub.go:213`–`215`，公共实现 `217`–`253`；路由 `internal/httpapi/server.go:203`。

### 请求

无参数。

### 响应

与 `GET /nodeinfo/2.1` **完全相同**，除以下两处：

| 项 | 2.1 | 2.0 |
|----|-----|-----|
| `version` 字段 | `"2.1"` | `"2.0"` |
| `Content-Type` profile | `.../schema/2.1#` | `.../schema/2.0#` |

其余字段（`software`、`protocols`、`services`、`openRegistrations`、`usage`、`metadata`）逐字节相同，缓存都是 600 秒。

### 错误

`403 Forbidden`（纯文本）当联邦模式为 `none`。

### 已知限制

同 `GET /nodeinfo/2.1`（metadata 缺 20+ 字段、`software.homepage`/`repository` 缺失、`usage.users.activeHalfyear`/`activeMonth` 缺失）。2.0 文档没有做任何降级差异处理，例如 Misskey 在 2.1 才补的字段这里也没有。

---

## OPTIONS /.well-known/*

**别名**：匹配 `/.well-known/` 子树下任意路径（含 `/.well-known/webfinger`）。
**认证**：不需要。**联邦开关**：**不受影响**（先回 204，不查模式）。
**源码**：`internal/httpapi/activitypub.go:257`–`262`；路由 `internal/httpapi/server.go:204`。

### 请求

无参数。

### 响应

| 头 | 值 |
|----|----|
| 状态 | `204 No Content` |
| `Access-Control-Allow-Origin` | `*` |
| `Access-Control-Allow-Headers` | `Accept` |
| `Access-Control-Allow-Methods` | `GET, OPTIONS` |

无 body。

### 错误

无。

### 已知限制

只覆盖 `/.well-known/` 子树；`/users/*`、`/notes/*`、`/nodeinfo/*`、`/inbox` **没有** OPTIONS 处理，跨域预检会落到兜底 404（JSON）。AP 端点对浏览器客户端只在 GET 成功路径上可用。

---

## GET /users/{id}

**别名**：`GET /@{acct}`（见下一节）。
**认证**：不需要。**联邦开关**：`none` → 403。
**源码**：`internal/httpapi/activitypub.go:265`–`281`、`315`–`330`；测试 `internal/httpapi/activitypub_test.go:148`。

### 请求

| 参数（path） | 类型 | 说明 |
|--------------|------|------|
| `id` | string | 本地用户 ULID |

`Accept` 决定 `Content-Type`（见通用约定 §8）。

### 响应

`200` + 完整 Actor 文档（字段见“共享对象 / Actor”）。首次访问会按需生成密钥对。

响应头：`Content-Type`（协商）、`Vary: Accept`、`Access-Control-Allow-Origin: *`、`Access-Control-Allow-Headers: Accept`、`Access-Control-Allow-Methods: GET, OPTIONS`、`Access-Control-Expose-Headers: Vary`、`Cache-Control: public, max-age=180`。

### 错误

| 状态 | Body | 触发条件 |
|------|------|----------|
| `403` | `Forbidden` | 联邦模式 `none` |
| `404` | `404 page not found` | id 不存在，或该 id 属于远端用户（`users.host IS NOT NULL`） |
| `500` | `Internal Server Error` | 密钥生成或数据库失败 |

**这些错误响应都不带 CORS 头。**

### 已知限制

1. **Actor 文档缺头像、横幅、资料字段、锁定标记**：`renderLocalActor` 只填 `ID/Username/Name/Description/PublicKey/CreatedAt`（`internal/httpapi/activitypub.go:315`–`330`），于是 `icon`/`image` 恒为 `null`，`attachment` 恒缺失，`manuallyApprovesFollowers` 恒 `false`——即使 `users.is_locked = true`。远端实例因此看不到本地用户的头像与锁推状态。
2. **`type` 恒为 `Person`**：`RenderPerson` 里 `Application` 分支靠“用户名含 `.`”触发（`internal/federation/render.go:76`），而本地用户名校验是 `\w{1,20}`（`internal/auth/auth.go:150`–`169`），永远不含点；`Service` 分支依赖 `IsBot`，但调用方从不设置。
3. 不返回 `movedTo`、`alsoKnownAs`、`vcard:*`、`isCat` 等 Misskey 扩展。
4. 不做 authorized fetch：任何人不带签名就能取 actor 文档。

---

## GET /@{acct}

**别名**：`GET /users/{id}` 的 handle 形式（同一 handler 家族，不同实现）。
**认证**：不需要。**联邦开关**：`none` → 403。
**源码**：`internal/httpapi/activitypub.go:284`–`311`；路由 `internal/httpapi/server.go:217`（以 `GET /{acct}` 注册）；测试 `internal/httpapi/activitypub_test.go:192`。

### 请求

| 参数（path） | 类型 | 说明 |
|--------------|------|------|
| `acct` | string | 必须以 `@` 开头；可以是 `@alice` 或 `@alice@fish.example` |

Go 的 mux 不允许通配符出现在段中间，所以路由写成 `GET /{acct}`，由 handler 自己校验 `@` 前缀（`internal/httpapi/server.go:215`–`217`）。

### 响应

与 `GET /users/{id}` 完全相同的 Actor 文档与响应头。

### 错误

| 状态 | Body | 触发条件 |
|------|------|----------|
| `403` | `Forbidden` | 联邦模式 `none` |
| `404` | `404 page not found` | 路径不以 `@` 开头；`@user@host` 的 host 不是本机 domain；用户不存在；用户是远端账户 |
| `500` | `Internal Server Error` | 密钥生成或数据库失败 |

### 已知限制

- 由于是单段兜底路由，**任何单段 GET 路径**都会进到这里并拿到纯文本 404（见通用约定 §6），例如 `GET /nodeinfo`、`GET /.well-known`。
- 同 `GET /users/{id}` 的 Actor 字段缺失问题。

---

## GET /users/{id}/publickey

**别名**：无。**认证**：不需要。**联邦开关**：`none` → 403。
**源码**：`internal/httpapi/activitypub.go:333`–`349`、`internal/federation/render.go:402`–`409`。

### 请求

| 参数（path） | 类型 | 说明 |
|--------------|------|------|
| `id` | string | 本地用户 ULID |

### 响应

`200` + Key 文档（见“共享对象 / Key”），`Content-Type` 按 `Accept` 协商，其余头同 `writeAP`。

### 错误

| 状态 | Body | 触发条件 |
|------|------|----------|
| `403` | `Forbidden` | 联邦模式 `none` |
| `404` | `404 page not found` | 用户不存在或为远端账户 |
| `500` | `Internal Server Error` | 密钥生成失败 |

### 已知限制

无（Misskey 同样提供该路径）。

---

## GET /users/{id}/inbox

**别名**：无（注意：`POST /users/{id}/inbox` 走的是**另一个** handler，见下下节）。
**认证**：不需要。**联邦开关**：`none` → 403。
**源码**：`internal/httpapi/activitypub.go:353`–`367`。

### 请求

| 参数（path） | 类型 | 说明 |
|--------------|------|------|
| `id` | string | 本地用户 ULID（仅用于存在性校验） |

### 响应

`200`：

```json
{ "@context": [...], "type": "OrderedCollection", "totalItems": 0 }
```

只有这三个键，**没有 `id`、没有 `first`**。`Content-Type` 按 `Accept` 协商，`Cache-Control: public, max-age=180`。

### 错误

| 状态 | Body | 触发条件 |
|------|------|----------|
| `403` | `Forbidden` | 联邦模式 `none` |
| `404` | `404 page not found` | 用户不存在或为远端账户 |

### 已知限制

这是纯占位实现：ActivityPub 并不要求 GET inbox，Misskey 也不提供有意义的内容。客户端不应依赖它。

---

## POST /inbox

**别名**：`POST /users/{id}/inbox`（同一 handler，`internal/httpapi/server.go:206`、`209`）。
**认证**：HTTP Signature（RSA-SHA256）；不需要令牌。
**联邦开关**：`none` → 403。
**源码**：`internal/httpapi/activitypub.go:581`–`639`；签名校验 `internal/federation/signature.go:214`；活动分发 `internal/federation/inbox.go:40`；测试 `internal/httpapi/activitypub_federation_test.go:157`、`228`、`271`。

### 请求

| 项 | 说明 |
|----|------|
| Body | ActivityPub 活动 JSON，**上限 64 KiB**（`io.LimitReader(r.Body, 64<<10+1)`） |
| `Signature` 头 | 必须。也可用 `Authorization: Signature ...` 形式（`internal/federation/signature.go:215`–`222`） |
| 必备覆盖头 | `host`、`date`、`digest`（POST）；缺一即 401 |
| 算法 | 只接受 `rsa-sha256` |
| `Date` | 与本机时间相差超过 ±5 分钟即拒 |
| `Host` | 必须等于请求实际访问的 `r.Host`（反代场景下就是反代传来的 Host） |

校验顺序（`internal/httpapi/activitypub.go:581`–`628`）：联邦开关 → 读 body（64 KiB）→ 验签 → 解析 JSON → `actor` 字段非空 → 交给 `Inbox.Process` → `202`。

签发者公钥的解析顺序（`internal/httpapi/activitypub.go:644`–`671`）：先查本地 `users.public_key_pem`（按 actor URI），查不到再用联邦客户端抓 actor 文档里的 `publicKey.publicKeyPem`。

支持的活动类型（`internal/federation/inbox.go:56`–`98`）：`Collection`/`OrderedCollection`（递归展开）、`Follow`、`Accept`、`Reject`、`Undo`、`Create`、`Delete`、`Like`、`Announce`、`Update`、`Block`。`Update` 按对象类型分流：actor 文档更新角色信息，笔记对象更新已存远端笔记的正文/CW/敏感标记（要求 `attributedTo` 与投递 actor 一致、id 主机与 actor 主机一致；只动内容，不动可见性与受众）。其它类型（`Move`、`Flag`、`Question`、`EmojiReact` 等）记日志 `unrecognized activity type: X` 后仍回 202。

### 响应

| 状态 | Body | 说明 |
|------|------|------|
| `202 Accepted` | 空 | 验签通过即接受；`Content-Type` 按 `Accept` 协商 |

**处理结果不体现在状态码上**：语义失败（目标不存在、重复投递、未知类型）只写日志，仍回 202（`internal/httpapi/activitypub.go:628`–`638`）。

### 错误

| 状态 | Body | 头 | 触发条件 |
|------|------|----|----------|
| `403` | `Forbidden`（纯文本） | — | 联邦模式 `none`，或联邦未挂载（`federationInbox == nil`） |
| `413` | `Payload Too Large`（纯文本） | — | body 超过 64 KiB |
| `400` | `Bad Request`（纯文本） | — | 读 body 失败、JSON 非法、或活动没有 `actor` |
| `401` | 空 | `WWW-Authenticate: Signature realm="Cinderfish"` | 无签名、算法不支持、缺 `host`/`date`/`digest`、Host 不匹配、Digest 不匹配、`Date` 过期、RSA 校验失败 |

### 已知限制

1. **`POST /users/{id}/inbox` 不校验 `{id}`**：它与共享 inbox 是同一个 handler，收到什么就处理什么，`{id}` 被完全忽略。想定向投递给某个用户的行为无法区分。
2. **同步处理**：验签通过后活动在请求内处理完才回 202，没有独立队列；慢的远端 fetch 会拖长请求（`docs/HANDOVER.md` §12.4）。
3. **不校验 `Content-Type`**，任意类型都会尝试按 JSON 解析。
4. **不支持 LinkedDataSignature**（`signatureValue`），只用 HTTP Signatures。
5. **`EmojiReact` 未实现**：Firefish/Misskey 的自定义反应若走 `EmojiReact` 类型会被当作未知活动；`Like` 会读 `_misskey_reaction` → `content` → `name`，兜底 `"⭐"`（`internal/federation/inbox.go:339`–`366`）。
6. **`Move`、`Flag`、`Question`（投票）未实现**（`docs/research/2026-11-04-misskey-api-coverage.md` §6）。
7. 锁推账户的入站 `Follow` 会写入 `follow_requests`，但**不产生 `receiveFollowRequest` 通知**；本地发起关注远端锁推账户时也不会写待审行（同上文档 §6）。
8. 出站签名固定用“第一个管理员（否则第一个本地用户）”的密钥（`internal/httpapi/federation_wire.go:140`–`146`），多用户实例下签名者与活动作者不一致。

---

## GET /users/{id}/outbox

**别名**：无。**认证**：不需要。**联邦开关**：`none` → 403。
**源码**：`internal/httpapi/activitypub.go:370`–`408`；数据源 `internal/note/timeline.go:250`。

### 请求

| 参数（query） | 类型 | 必填 | 说明 |
|---------------|------|------|------|
| `page` | string | 否 | 只有字面值 `true` 才返回分页；其它值（含 `1`）返回 collection |
| `since_id` | string | 否 | 笔记 ULID，只返回更新的（`n.id > since_id`） |
| `until_id` | string | 否 | 笔记 ULID，只返回更旧的（`n.id < until_id`） |

返回的笔记范围固定为：该作者、本地、未删除、`local_only = FALSE`、可见性 `public` 或 `home`，**包含回复**（`withReplies=true`），按 id 降序，每页固定 **20** 条（`internal/httpapi/activitypub.go:390`）。

### 响应

不传 `page=true`（collection）：

```json
{ "@context": [...], "id": "<actor>/outbox", "type": "OrderedCollection", "totalItems": 42, "first": "<actor>/outbox?page=true" }
```

`totalItems` 用另一条 SQL 统计同一范围（`internal/httpapi/activitypub.go:383`–`385`）。

传 `page=true`（page）：

```json
{ "@context": [...], "id": "<actor>/outbox?page=true", "type": "OrderedCollectionPage",
  "partOf": "<actor>/outbox", "totalItems": 20, "orderedItems": [ <Create 活动>, ... ] }
```

- `orderedItems` 的每一项是 `Create` 活动，内嵌完整 `Note`（见“共享对象 / Note”）。
- 传了 `since_id` 时，page 的 `id` 会附加 `&since_id=<值>`（`internal/httpapi/activitypub.go:404`–`406`）。
- **没有 `next` / `prev`**。

### 错误

| 状态 | Body | 触发条件 |
|------|------|----------|
| `403` | `Forbidden` | 联邦模式 `none` |
| `404` | `404 page not found` | 用户不存在或为远端账户 |
| `500` | `Internal Server Error` | 查询失败 |

单个笔记渲染失败会被 `continue` 静默跳过（`internal/httpapi/activitypub.go:398`–`401`），所以 page 里的条数可能少于 20。

### 已知限制

1. **没有 `next`/`prev` 链接**，也没有把 `until_id` 回填进 page URL；客户端要翻页只能自己取最后一条笔记的 id 作为 `until_id`。
2. page 的 `totalItems` 是**本页条数**而非集合总数，与规范的用法不同。
3. 只发 `public`/`home`，`followers` 与 `specified` 的笔记永不出现在 outbox（但会通过 `Create` 投递主动推给关注者）。
4. 每页固定 20，不可配置。

---

## GET /users/{id}/followers

**别名**：无。**认证**：不需要。**联邦开关**：`none` → 403。
**源码**：`internal/httpapi/activitypub.go:411`、`420`–`464`；数据源 `internal/federation/store.go:674`、`691`。

### 请求

| 参数（query） | 类型 | 必填 | 说明 |
|---------------|------|------|------|
| `page` | string | 否 | 只有 `true` 才返回分页 |
| `cursor` | string | 否 | `followings.id`（**关注关系行的 id**），返回比它更旧的 |

每页固定 **10** 条，按 `followings.id` 降序。

### 响应

collection：

```json
{ "@context": [...], "id": "<actor>/followers", "type": "OrderedCollection", "totalItems": 7, "first": "<actor>/followers?page=true" }
```

`totalItems` 来自 `SELECT count(*) FROM followings WHERE followee_id = $1`（`internal/federation/store.go:674`–`686`）。

page：

```json
{ "@context": [...], "id": "<actor>/followers?page=true", "type": "OrderedCollectionPage",
  "partOf": "<actor>/followers", "totalItems": 7,
  "orderedItems": ["https://other.example/users/01J...", "https://third.example/@bob"] }
```

`orderedItems` 是**URI 字符串数组**，不是内嵌 Actor 对象。

### 错误

| 状态 | Body | 触发条件 |
|------|------|----------|
| `403` | `Forbidden` | 联邦模式 `none`；或该用户的 `followers_visibility = 'private'`（后者额外带 `Cache-Control: public, max-age=30`） |
| `404` | `404 page not found` | 用户不存在或为远端账户 |
| `500` | `Internal Server Error` | 查询失败 |

### 已知限制

1. **本地关注者会得到相对 URI**。`FollowRefs` 在 `users.uri` 为空（本地用户没有 uri 列值）时回退成 `"" + id`（`internal/federation/store.go:720`–`722`），实际产出是 `"/users/<ulid>"` 这种缺 scheme 与 host 的字符串，对远端无效。远端关注者正常返回完整 URI。
2. **page 里不回传 `cursor`**：`orderedItems` 只有 URI，没有 `followings.id`，客户端无法继续翻页（只能拿 collection 从头读）。这是当前实现的硬伤。
3. 只实现 `followers_visibility`，**`following_visibility` 从未被读取**（`internal/httpapi/activitypub.go:438`–`441` 只查 followers 列），所以「隐藏关注列表」的隐私设置对联邦无效。
4. 分页大小与 `cursor` 参数名是内部约定，Misskey 不使用 `cursor`（它用 `page`+`offset` 语义），**不要按 Misskey 客户端的习惯拼**。

---

## GET /users/{id}/following

**别名**：无。**认证**：不需要。**联邦开关**：`none` → 403。
**源码**：`internal/httpapi/activitypub.go:416`（复用 `serveFollowCollection`，`420`–`464`）。

### 请求

同 `followers`：`page`、`cursor`；每页 10 条，按 `followings.id` 降序，条件是 `follower_id = 该用户`。

### 响应

与 `followers` 同构，`id`/`partOf` 换成 `<actor>/following`，`totalItems` 来自 `SELECT count(*) FROM followings WHERE follower_id = $1`。

### 错误

| 状态 | Body | 触发条件 |
|------|------|----------|
| `403` | `Forbidden` | 联邦模式 `none` |
| `404` | `404 page not found` | 用户不存在或为远端账户 |
| `500` | `Internal Server Error` | 查询失败 |

注意：**following 没有隐私检查**（`following_visibility` 未实现），即使设为 `private` 也照常返回。

### 已知限制

同 `followers` 的第 1、2、3 条（本地用户产出相对 URI、无法翻页、隐私设置无效）。

---

## GET /users/{id}/collections/featured

**别名**：无。**认证**：不需要。**联邦开关**：`none` → 403。
**源码**：`internal/httpapi/activitypub.go:468`–`480`。

### 请求

| 参数（path） | 类型 | 说明 |
|--------------|------|------|
| `id` | string | 本地用户 ULID |

不接受 `page`。

### 响应

```json
{ "@context": [...], "id": "<actor>/collections/featured", "type": "OrderedCollection", "totalItems": 0 }
```

**没有 `first`**（`RenderOrderedCollection(partOf, 0, "")` 传了空串，`internal/federation/render.go:376`–`378`）。

### 错误

| 状态 | Body | 触发条件 |
|------|------|----------|
| `403` | `Forbidden` | 联邦模式 `none` |
| `404` | `404 page not found` | 用户不存在或为远端账户 |

### 已知限制

**恒为空**：置顶笔记功能未实现（`docs/HANDOVER.md` §12.4）。虽然 `i/pin` 能把笔记写进本地置顶表，这里的 featured 集合**不读它**。远端实例因此看不到任何置顶笔记。

---

## GET /notes/{id}

**别名**：无。**认证**：不需要。**联邦开关**：`none` → 403。
**源码**：`internal/httpapi/activitypub.go:483`–`505`、`536`–`565`。

### 请求

| 参数（path） | 类型 | 说明 |
|--------------|------|------|
| `id` | string | 本地笔记 ULID |

### 响应

`200` + AP `Note` 文档（字段见“共享对象 / Note”），`Content-Type` 按 `Accept` 协商，头同 `writeAP`。

可访问范围由 handler 硬编码（`internal/httpapi/activitypub.go:495`–`498`）：必须**本地作者**、`local_only = FALSE`、可见性 `public` 或 `home`。其它情况一律 404。

### 错误

| 状态 | Body | 触发条件 |
|------|------|----------|
| `403` | `Forbidden` | 联邦模式 `none` |
| `404` | `404 page not found` | 笔记不存在；远端作者；`local_only`；可见性不是 `public`/`home` |
| `500` | `Internal Server Error` | 渲染失败（作者查不到等） |

### 已知限制

1. **`followers` 与 `specified` 的笔记在 GET 上不可读**（直接 404），即使请求方是被提及者或关注者。它们只能通过活动投递到达远端。
2. **不返回附件**：`renderNote` 从不填 `NoteData.Attachments`（`internal/httpapi/activitypub.go:541`–`564`），所以带图的笔记在 AP 里**没有 `attachment` 字段**，远端只能看到正文。这是当前最影响观感的联邦缺口。
3. 不返回 `replies` 集合、`likes` 集合、`shares` 集合。
4. 不支持 authorized fetch，`followers` 可见性无法通过签名请求解锁。

---

## GET /notes/{id}/activity

**别名**：无。**认证**：不需要。**联邦开关**：`none` → 403。
**源码**：`internal/httpapi/activitypub.go:508`–`533`、`568`–`574`。

### 请求

| 参数（path） | 类型 | 说明 |
|--------------|------|------|
| `id` | string | 本地笔记 ULID |

### 响应

`200` + `Create` 活动：

```json
{ "@context": [...], "id": "https://fish.example/notes/<ulid>/activity", "type": "Create",
  "actor": "https://fish.example/users/<ulid>", "published": "2026-11-04T12:34:56Z",
  "object": { ...Note... }, "to": [...], "cc": [...] }
```

可见性限制与 `/notes/{id}` 相同。

### 错误

| 状态 | Body | 触发条件 |
|------|------|----------|
| `403` | `Forbidden` | 联邦模式 `none` |
| `404` | `404 page not found` | 笔记不存在 / 远端作者 / `local_only` / 非 `public`·`home` |
| `500` | `Internal Server Error` | 渲染失败 |

### 已知限制

同 `/notes/{id}`（不返回附件）。另外该 `Create` 的 `id` 与出站投递使用的 id 一致（`internal/federation/render.go:40`–`42`），因此远端可以用同一 URI 去重。

---

## 与 Misskey 的差异与已知缺口

下面按“会不会影响新前端”排序。

### 影响前端直接展示

| # | 缺口 | 证据 |
|---|------|------|
| 1 | `meta.policies` 只有 2 个键，Misskey 有 40+；Firefish 客户端读 `policies.*` 会拿到 `undefined` | `internal/httpapi/meta.go:121`–`126` |
| 2 | `meta.federation` 与 `features.globalTimeline` 读配置文件，运行时切换模式后不更新 | `internal/httpapi/meta.go:78`、`92` 对照 `internal/httpapi/federation_admin.go:66` |
| 3 | `/api/*` 没有 CORS，前端必须同源 | 全仓库仅 AP 路由设 ACAO |
| 4 | `WWW-Authenticate` 的 `error_description` 被覆盖，客户端只能看到 `Bearer realm="Misskey"` | `internal/httpapi/server.go:305` 与 `283`–`285` |
| 5 | 令牌来源优先级与 Misskey 相反（body `i` 优先于 Header），且不支持 GET query 里的 `i` | `internal/httpapi/auth.go:30`–`46` |
| 6 | scope 不生效：只读令牌可写 | 仅 `internal/httpapi/streaming_adapters.go:24` 检查 |
| 7 | AP 的成功响应有 CORS，403/404 没有 | `internal/httpapi/activitypub.go:46`–`63` |
| 8 | AP 的 404/403 是纯文本，只有路由未命中才是 JSON 信封 | `internal/httpapi/server.go:327` 对照 `internal/httpapi/activitypub.go:61` |

### 影响联邦互操作

| # | 缺口 | 证据 / 参照 |
|---|------|-------------|
| 9 | Actor 不含头像、横幅、资料字段、锁推标记，`type` 恒 `Person` | `internal/httpapi/activitypub.go:315`–`330` |
| 10 | 笔记不返回 `attachment`，带媒体笔记联邦后丢图 | `internal/httpapi/activitypub.go:536`–`565` |
| 11 | `featured` 恒空，置顶笔记不同步 | `internal/httpapi/activitypub.go:468`–`480` |
| 12 | followers/following 的本地条目是 `"/users/<id>"` 相对 URI | `internal/federation/store.go:720`–`722` |
| 13 | followers/following 无法翻页（page 不回传 cursor） | `internal/httpapi/activitypub.go:454`–`463` |
| 14 | `following_visibility` 未实现 | `internal/httpapi/activitypub.go:438`–`441` |
| 15 | webfinger 只做 JRD，不做 XRD | `docs/HANDOVER.md` §12.4 |
| 16 | NodeInfo metadata 缺 20+ 字段，`services.outbound` 为空 | `internal/httpapi/activitypub.go:231`–`246` 对照 `NodeinfoServerService.ts:97` |
| 17 | inbox 不支持 `Move` / `Flag` / `Question` / `EmojiReact` | `internal/federation/inbox.go:75`–`97` |
| 18 | inbox 同步处理、固定 64 KiB、语义失败仍回 202 | `internal/httpapi/activitypub.go:588`–`601`、`628`–`638` |
| 19 | 出站签名固定用第一个管理员的密钥 | `internal/httpapi/federation_wire.go:140`–`146` |
| 20 | 没有 authorized fetch、没有实例黑名单、没有 relay | `docs/HANDOVER.md` §12.4 |

### 纯占位 / 恒值

| # | 项 | 说明 |
|---|----|------|
| 21 | `GET /users/{id}/inbox` | 恒 `{type: OrderedCollection, totalItems: 0}` |
| 22 | `meta` 的站点名、描述、maintainer、主题色、图片 URL | 全部写死 |
| 23 | `meta.version` = `0.1.0` 与 NodeInfo `software.version` = `0.0.1` | 两处不一致 |
| 24 | `cacheRemoteFiles: true` | 实际不缓存远端媒体 |

---

## 附：核对方式

本文对 **Cinderfish 自身行为**的结论全部来自逐行阅读下列本仓库代码与既有测试；Misskey 源码只在“与 Misskey 的差异”章节做对照，且对照的是源码而非官方文档。

- `internal/httpapi/server.go`（路由表 107–224、错误封装 235–335）
- `internal/httpapi/meta.go`、`ping.go`、`auth.go`、`web.go`、`federation_wire.go`
- `internal/httpapi/activitypub.go`（全部 AP handler 与 `handleInbox`）
- `internal/federation/render.go`、`apobject.go`、`mfm.go`、`policy.go`、`store.go`、`inbox.go`、`keys.go`、`signature.go`
- 既有测试：`internal/httpapi/server_test.go`、`activitypub_test.go`、`activitypub_federation_test.go`、`web_test.go`
- 项目记录：`docs/HANDOVER.md` §12、`docs/research/2026-11-04-misskey-api-coverage.md` §6、`docs/research/2026-10-28-m5-federation-source-calibration.md`

上游对照文件（`tmp/refcheck/misskey`，Misskey `c05e844`）：`packages/backend/src/server/api/ApiCallService.ts`、`server/WellKnownServerService.ts`、`server/NodeinfoServerService.ts`、`core/entities/MetaEntityService.ts`、`core/RoleService.ts`、`server/api/error.ts`。

## 2026-11-06 增补：公开 stats

`POST /api/stats`（无需登录）返回：

```json
{ "usersCount": 24, "notesCount": 154, "originalNotesCount": 120,
  "originalUsersCount": 24, "reactionsCount": 0, "followingCount": 0,
  "instancesCount": 0, "driveFiles": 1, "driveUsage": 123456 }
```

`usersCount` = 未冻结本地账号；`instancesCount` = `users.host` 去重；`driveUsage` = 文件大小之和。
单条 SQL 子查询完成，没有缓存；关于页用它渲染统计。

## 2026-11-06 增补：federation/show-instance

`POST /api/federation/show-instance`（需登录）请求 `{host}`，返回 `FederationInstance` 或 `null`。

Cinderfish 没有 Misskey 的 `instances` 表，也没有在线的 nodeinfo 抓取：

- 本地主机：名称/简介来自运行时实例设置，`softwareName = "cinderfish"`、`softwareVersion = null`、
  `openRegistrations` 由注册模式推导，用户/帖子数来自实时计数；
- 其它主机：只统计本库已收录的 `users.host` 与 `notes.user_host` 行；从未出现的主机返回 `null`；
- `softwareName`/`softwareVersion`/`latestStatus` 等字段对远端恒为 `null`，`isBlocked`/`isSuspended`/
  `isNotResponding` 恒 `false`。

**不发任何网络请求**，因此该端点不能被用来探测任意地址（SSRF 风险为 0）。代价是远端信息不会自动
更新，/instance-info 页面也明确这样说明。

## 2026-11-06 增补：federation/instances

`POST /api/federation/instances`（需登录）接受 `{host?, limit?, offset?}`，返回
`FederationInstance[]`，按收录用户数降序。数据按 `users.host` 聚合，帖子数来自 `notes.user_host`；
没有 `instances` 表，所以软件字段与状态标志都是 `null`/`false`。`limit` 默认 30、上限 100。
管理台联邦页用它列出「已收录实例」。
