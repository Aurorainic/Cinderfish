# API 对接契约：streaming

本文档记录 Cinderfish 后端 Streaming API 的真实行为，供新前端（Firefish 风格客户端）直接对接。所有结论来自通读 WebSocket 服务端、频道实现、事件发布点与相关测试，不引述 Misskey 文档；无法从代码确认的地方明确写「未实现」或「未返回」。

范围（实现文件）：

- `internal/httpapi/streaming.go`（挂载与 `GET /streaming` handler）
- `internal/httpapi/streaming_adapters.go`（token 鉴权、following 图加载）
- `internal/httpapi/streaming_pack.go`（Note / Notification 打包）
- `internal/httpapi/streaming_publish.go`（事件发布点）
- `internal/streaming/server.go`、`connection.go`、`conn.go`、`channel.go`、`channel_main.go`、`channel_timeline.go`、`channel_timelines.go`、`bus.go`、`redis.go`
- 间接证据：`internal/httpapi/notes.go`、`following.go`、`following_requests.go`、`notifications.go`、`federation_wire.go`、`internal/federation/inbox.go`、`cmd/cinderfish/main.go`

不属于本文档：REST 端点的请求/响应（见 notes-core、auth-users、timelines-notifications、following-relations 等契约），ActivityPub 出站/入站（见 meta-federation 契约）。

验证依据：`internal/streaming/*_test.go`（频道过滤矩阵、频道生命周期、subNote 引用计数与可见性、跨进程中继）、`internal/httpapi/streaming_e2e_test.go` 与 `streaming_e2e_scenarios_test.go`（401/403/匿名、笔记投递、反应通知 + `noteUpdated`、可见性不外泄、`readNotification`）、`internal/httpapi/streaming_relay_test.go`。下文标注「测试」的结论有对应用例覆盖，标注「代码」的结论由阅读代码确定、无测试直接覆盖。

## 端点总览

| 方法 | 路径 | 作用 | 鉴权 | 实现 |
|------|------|------|------|------|
| `GET` | `/streaming` | WebSocket 升级；承载全部实时事件 | 匿名可连；带 token 时校验 | `internal/httpapi/server.go:221` → `internal/httpapi/streaming.go:45` |

本命名空间**只有一个端点**，而且不是 `POST /api/...` 形状：Misskey 的上游实现同样是 `GET /streaming`（`StreamingApiServerService.attach`），用 HTTP Upgrade 就地升级。因此后文只有一个端点小节 `## GET /streaming`，其余是协议帧与事件契约（`## 通用约定` + `## GET /streaming` 内的子小节）。

没有别名路径：未注册 `/api/streaming`、`/streaming/`、SSE 或长轮询变体（`internal/httpapi/server.go:221` 是唯一注册点）。

两个容易踩坑的路由事实：

1. SPA 静态回退显式排除 `/streaming`（`internal/httpapi/web.go:45`–`internal/httpapi/web.go:52`），所以它永远进 mux，不会被 `index.html` 吃掉。
2. `Server.ServeHTTP` 先询问 SPA（`internal/httpapi/server.go:226`–`internal/httpapi/server.go:232`），再进 mux；顺序对 `/streaming` 无影响，但对 `GET /@user` 之类的共享路径有影响（不在本文范围）。

## 通用约定

### 传输

- 与 REST 共用同一个 HTTP 端口：gorilla/websocket 就地 Upgrade（`cmd/cinderfish/main.go:51`–`cmd/cinderfish/main.go:57`）。
- 不协商子协议，客户端不需要（也不应依赖）`Sec-WebSocket-Protocol`。
- **不校验 `Origin`**：`CheckOrigin` 恒返回 true（`internal/streaming/server.go:78`–`internal/streaming/server.go:83`）。跨域部署的前端可以直接连，不需要额外配置。
- 非 WebSocket 的普通 `GET /streaming` 会被 gorilla 以 400 拒绝，响应是**纯文本**而不是 JSON 错误封套——客户端不要按 JSON 解析握手失败后的 body。
- 没有 SSE / 长轮询 / HTTP 分块回退通道。

### 鉴权

token 来源，按优先级（`internal/streaming/server.go:93`–`internal/streaming/server.go:98`）：

1. `Authorization: Bearer <token>`——精确前缀匹配，`Bearer ` 大小写敏感；
2. 否则读 URL query 的 `i=<token>`。

**与 REST 端点的关键差异**：Streaming **不读**请求体里的 `i` 字段，也不读 query 之外的位置。REST 端点的 `authenticate` 读的是 body `i`（优先）与 `Authorization: Bearer`（`internal/httpapi/auth.go:30`–`internal/httpapi/auth.go:46`），**不读 query**；Streaming 是 GET、没有 body，只认 header 与 query `i`。两处的 header 前缀匹配也不同：REST 用 `strings.ToLower` 后比较 `bearer `（`internal/httpapi/auth.go:39`），Streaming 直接比较 `Bearer `（`internal/streaming/server.go:94`），所以 `bearer xxx` 这种小写写法在 Streaming 上不生效。

token 解析链路：`internal/httpapi/streaming_adapters.go:19` → `internal/account/tokens.go:88`：

- 长度为 16 的 token 视为「原生用户 token」，查 `users.token`；命中即拥有全部权限（`appToken == nil`）。
- 其他长度查 `access_tokens.token`。MiAuth token 32 位、OAuth token 128 位（`internal/auth/auth.go:96`–`internal/auth/auth.go:103`）。
- 对 `access_tokens`（app token）**要求 scopes 里存在字面量 `read:account`**（`internal/httpapi/streaming_adapters.go:24`–`internal/httpapi/streaming_adapters.go:28`、`internal/httpapi/streaming_adapters.go:36`–`internal/httpapi/streaming_adapters.go:43`），否则按认证失败处理（401）。
- 账号被封禁 → `ErrSuspended` → 403（`internal/httpapi/streaming_adapters.go:29`–`internal/httpapi/streaming_adapters.go:31`；`internal/streaming/server.go:104`–`internal/streaming/server.go:107`）。
- **没有 token 或 token 为空** → 匿名连接，合法（`internal/streaming/server.go:100`–`internal/streaming/server.go:114`）。匿名连接只能订阅 `localTimeline` 与 `globalTimeline`。

### 握手失败响应

| 条件 | 状态码 | 响应 |
|------|--------|------|
| `AttachStreaming` 未调用，或 token 服务未初始化 | 503 | JSON 错误封套 `INTERNAL_ERROR`（`internal/httpapi/streaming.go:46`–`internal/httpapi/streaming.go:52`） |
| token 存在但无效 / 缺 `read:account` | 401 | 空 body，附 `WWW-Authenticate: Bearer realm="Misskey", error="invalid_token", error_description="Failed to authenticate"`（`internal/streaming/server.go:108`–`internal/streaming/server.go:111`） |
| 账号被封禁 | 403 | 空 body，无 header（`internal/streaming/server.go:104`–`internal/streaming/server.go:107`） |
| 不是 WebSocket 请求 | 400 | gorilla 的纯文本响应，不是 JSON |
| 升级成功 | 101 | 之后全部为 WebSocket 帧 |

503 是**唯一**会返回 JSON 错误封套的路径，形状与 REST 完全一致（`writeAPIError`，`internal/httpapi/server.go:268`–`internal/httpapi/server.go:287`）：

```json
{"error":{"message":"Not available.","code":"INTERNAL_ERROR","id":"5d37dbcb-891e-41ca-a3d6-e690c97775ac","kind":"server"}}
```

401 / 403 是**裸 HTTP 响应**：没有 JSON body，没有 `code` / `kind`，客户端只能靠状态码与 `WWW-Authenticate` 判别（测试：`internal/httpapi/streaming_e2e_scenarios_test.go:18`–`internal/httpapi/streaming_e2e_scenarios_test.go:34`）。

### 帧格式与容错

- 双向帧都是 `{"type": string, "body": any}`（`internal/streaming/connection.go:123`–`internal/streaming/connection.go:131`；出站 `internal/streaming/conn.go:36`–`internal/streaming/conn.go:39`）。
- 无法反序列化的帧直接丢弃，不回报（`internal/streaming/connection.go:128`–`internal/streaming/connection.go:130`）。
- 未知的 `type` 直接忽略：分发 switch 没有 `default` 分支（`internal/streaming/connection.go:132`–`internal/streaming/connection.go:149`）。
- body 字段缺失或类型不符同样静默丢弃（例如 `connect` 缺 `id` / `channel`，`internal/streaming/connection.go:163`–`internal/streaming/connection.go:165`）。

### 订阅失败的统一语义

**所有订阅失败都不回复、不断连**（上游 Misskey 是抛错被消息队列吞掉，客户端同样看不到反馈）：

- 未知频道名（`internal/streaming/connection.go:187`–`internal/streaming/connection.go:192`）
- 需要登录的频道但连接是匿名（`internal/streaming/connection.go:193`–`internal/streaming/connection.go:195`）
- 频道 `Init` 返回 false（`internal/streaming/connection.go:221`–`internal/streaming/connection.go:229`）
- 频道数已达 32（`internal/streaming/connection.go:181`–`internal/streaming/connection.go:184`）
- `main` 重复订阅（`internal/streaming/connection.go:199`–`internal/streaming/connection.go:208`）

因此客户端**不能**用「收不到 `connected`」判定订阅失败，只能自己设超时后降级。只有 `connect` 带 `pong: true` 且订阅成功时才会回 `connected`（`internal/streaming/connection.go:231`–`internal/streaming/connection.go:233`）。

### 连接级限制

常量集中在 `internal/streaming/conn.go:10`–`internal/streaming/conn.go:33`：

| 常量 | 值 | 行为 |
|------|-----|------|
| `maxChannelsPerConnection` | 32 | 超出后新的 `connect` 静默丢弃 |
| `maxSubscribingNotesPerConnection` | 1536 | 超出后淘汰最久未用的 `subNote` 订阅（`internal/streaming/connection.go:308`–`internal/streaming/connection.go:321`） |
| `outboundQueueSize` | 256 条 | 每连接出站队列；满则直接断连（`internal/streaming/conn.go:81`–`internal/streaming/conn.go:99`） |
| `pingInterval` | 60s | 服务端发 WebSocket ping |
| `pongTimeout` | 120s | 超时判定见「已知缺口」第 1 条 |
| `writeWait` | 10s | 单次写超时 |
| `contextRefreshInterval` | 10s | 重新加载 following 图（`internal/streaming/connection.go:105`–`internal/streaming/connection.go:119`） |

### 跨进程中继

对客户端不可见，也不影响契约，但影响部署与「会不会丢事件」的预期：

- 单进程部署天然正确：`Bus.Publish` 只做本地 fanout（`internal/streaming/bus.go:140`–`internal/streaming/bus.go:158`）。
- 多进程由 `cmd/cinderfish/main.go:63`–`cmd/cinderfish/main.go:88` 决定：Redis 可用时走 Redis pub/sub（频道 `<prefix>:stream`，`internal/streaming/redis.go:25`–`internal/streaming/redis.go:30`），否则走 PostgreSQL `LISTEN/NOTIFY`，通道名 `cinderfish_stream`（`internal/streaming/bus.go:229`–`internal/streaming/bus.go:234`）。
- PG NOTIFY 单条上限 8000 字节，超限事件 base64 分片、接收端重组（`internal/streaming/bus.go:264`–`internal/streaming/bus.go:369`）；单事件上限 128 片（约 384 KiB 正文），超过只记日志并丢弃。

### 分页

本命名空间**不涉及**分页：没有 `limit` / `untilId` / `sinceId`。实时流断开后的补拉请用 `POST /api/notes/timeline`、`POST /api/notes/local-timeline` 等列表端点。

## GET /streaming

### 请求

```
GET /streaming?i=<token> HTTP/1.1
Upgrade: websocket
Connection: Upgrade
Sec-WebSocket-Key: <base64>
Sec-WebSocket-Version: 13
Authorization: Bearer <token>        // 可选，优先于 ?i=
```

无请求体。除 `i` 外的 query 参数全部忽略。响应 101 后立即进入帧模式，服务端**不发送任何初始帧**（没有 hello / welcome / 初始 `connected` 之类的握手帧），客户端必须自己发 `connect`（`internal/streaming/server.go:120`–`internal/streaming/server.go:157`）。

### 客户端 → 服务端帧

| `type` | 别名 | body | 行为 | 代码 |
|--------|------|------|------|------|
| `connect` | — | `{channel: string, id: string, params?: object, pong?: boolean}` | 订阅频道；`pong: true` 且成功时回 `connected` | `internal/streaming/connection.go:153`–`internal/streaming/connection.go:234` |
| `disconnect` | — | `{id: string}` | 退订频道，无回执 | `internal/streaming/connection.go:237`–`internal/streaming/connection.go:251` |
| `channel` | `ch` | `{id: string, type: string, body: any}` | 向频道发消息；**当前所有频道都不消费**，只校验三个字段存在 | `internal/streaming/connection.go:255`–`internal/streaming/connection.go:267` |
| `subNote` | `s`、`sr` | `{id: string}` | 订阅单条笔记的更新；同一 noteId 重复订阅按引用计数 +1 | `internal/streaming/connection.go:290`–`internal/streaming/connection.go:328` |
| `unsubNote` | `un` | `{id: string}` | 退订；计数减到 0 才真正移除 bus 监听 | `internal/streaming/connection.go:331`–`internal/streaming/connection.go:354` |
| `readNotification` | — | 忽略 | 把当前用户全部通知标记为已读 | `internal/streaming/connection.go:143`–`internal/streaming/connection.go:148` |

细节（全部读代码确认）：

- `connect.id` 由客户端自定义，服务端只把它原样回显在 `channel` 帧的 `body.id`。同一 `id` 再次 `connect` 会先销毁旧频道再建新的（`internal/streaming/connection.go:175`–`internal/streaming/connection.go:180`）。
- `connect.params` 若不是 JSON 对象 → 整帧反序列化失败而被丢弃。
- `pong` 必须是 boolean：给 `false` 或省略时订阅照常生效，只是不回 `connected`；给非 boolean 会让整帧丢弃。
- `subNote` / `unsubNote` **没有回执帧**，也不校验笔记是否存在、是否可见；不可见的笔记只是收不到事件。
- `readNotification` 只在已登录连接上生效，body 完全忽略；未登录连接发它没有任何效果。

### 频道

| channel | 需要登录 | 可重复订阅 | params（默认值） | 代码 |
|---------|----------|------------|------------------|------|
| `main` | 是 | 否（同连接第二个 `main` 静默忽略） | 无 | `internal/streaming/channel_main.go:29`–`internal/streaming/channel_main.go:79` |
| `homeTimeline` | 是 | 是 | `withRenotes`(true)、`withFiles`(false)；**无** `withReplies` | `internal/streaming/channel_timelines.go:8`–`internal/streaming/channel_timelines.go:66` |
| `localTimeline` | 否 | 是 | `withRenotes`(false)、`withReplies`(false)、`withFiles`(false) | `internal/streaming/channel_timelines.go:69`–`internal/streaming/channel_timelines.go:128` |
| `hybridTimeline` | 是 | 是 | `withRenotes`(false)、`withReplies`(false)、`withFiles`(false) | `internal/streaming/channel_timelines.go:170`–`internal/streaming/channel_timelines.go:227` |
| `globalTimeline` | 否 | 是 | `withRenotes`(false)、`withFiles`(false)；**无** `withReplies` | `internal/streaming/channel_timelines.go:131`–`internal/streaming/channel_timelines.go:167` |

频道注册表见 `internal/streaming/channel_main.go:20`–`internal/streaming/channel_main.go:26`。**未实现的频道**（订阅会被静默忽略）：`userList`、`clip`、`hashtag`、`roleTimeline`、`antenna`、`channel`、`drive`、`serverStats`、`queueStats`、`admin`、`chatUser`、`chatRoom`、`reversi`、`reversiGame` 等。依赖它们的 Firefish 功能（天线、列表、频道时间线、drive 变更推送、服务器状态面板）必须走轮询或直接降级。

param 解析规则（`internal/streaming/channel.go:222`–`internal/streaming/channel.go:241` 的 `boolParam`）：

- 缺失或 `null` → 用默认值
- boolean → 原值
- 其他任何非 null JSON 值 → 视为 `true`（对齐上游 `!!(params.x ?? default)`）

各频道的过滤规则（决定一条笔记会不会出现在该频道）：

- `homeTimeline`（`internal/streaming/channel_timelines.go:34`–`internal/streaming/channel_timelines.go:66`）：`isMe || 我关注作者` → `withFiles` → `isNoteVisibleForMe` → 回复过滤（用 per-follow 的 `withReplies`）→ 纯 renote 过滤 → mute/block（恒 false）。带 `channelId` 的笔记一律丢弃。
- `localTimeline`（`internal/streaming/channel_timelines.go:91`–`internal/streaming/channel_timelines.go:128`）：`user.host == ""` 且 `visibility == "public"` 且无 `channelId`；匿名连接**不做**回复过滤。
- `hybridTimeline`（`internal/streaming/channel_timelines.go:193`–`internal/streaming/channel_timelines.go:227`）：`isMe || 我关注作者 || (本地 && public)` → 可见性 → mute/block → 回复过滤 → renote 过滤。
- `globalTimeline`（`internal/streaming/channel_timelines.go:150`–`internal/streaming/channel_timelines.go:167`）：`visibility == "public"` 且无 `channelId`；远程笔记也走这条。
- 四个时间线频道都**不消费** `noteStream` 的 per-note 更新（`internal/streaming/channel_timeline.go:32`）。

回复过滤（`internal/streaming/channel_timeline.go:41`–`internal/streaming/channel_timeline.go:59`）与纯 renote 过滤（`internal/streaming/channel_timeline.go:63`–`internal/streaming/channel_timeline.go:81`）按 Misskey 的两段式实现，`withFiles` 要求 `fileIds` 非空（`internal/streaming/channel_timeline.go:84`–`internal/streaming/channel_timeline.go:86`）。

`isNoteVisibleForMe`（`internal/streaming/channel.go:121`–`internal/streaming/channel.go:156`）：

- `specified`：匿名 → 不可见；作者本人 → 可见；否则看 `visibleUserIds` 是否含我。
- `followers`：匿名 → 不可见；作者本人 → 可见；是回复目标 → 可见；被 `mentions` 命中 → 可见；否则看我是否关注作者。
- `public` / `home`（以及任何其他值）→ 可见。

### 服务端 → 客户端帧

| `type` | body | 触发 | 代码 |
|--------|------|------|------|
| `connected` | `{id: string}` | 仅在 `connect` 带 `pong: true` 且订阅成功时 | `internal/streaming/connection.go:231`–`internal/streaming/connection.go:233` |
| `channel` | `{id: string, type: string, body: any}` | 频道事件；`id` 是客户端在 `connect` 里给的 id | `internal/streaming/connection.go:422`–`internal/streaming/connection.go:424` |
| `noteUpdated` | `{id: string, type: string, body: any}` | `subNote` 订阅的笔记有更新 | `internal/streaming/connection.go:414`–`internal/streaming/connection.go:418` |

`noteUpdated` 的可见性门槛（`internal/streaming/connection.go:379`–`internal/streaming/connection.go:419`）：事件里的 `userId` 不是我的时候，`specified` 要求 `visibleUserIds` 含我，`followers` 要求我关注作者，否则整个事件不发。匿名连接只能收到 `public` / `home` 笔记的更新（测试：`internal/httpapi/streaming_e2e_scenarios_test.go:172`–`internal/httpapi/streaming_e2e_scenarios_test.go:196`）。

服务端不发送 `pong` 帧，也没有对客户端帧的通用 ack（`ping` 之类的 type 不存在）。

### 事件清单

下表都是 `channel` 帧内部 `body.type` 的取值。

A. 时间线频道（`homeTimeline` / `localTimeline` / `hybridTimeline` / `globalTimeline`）

| `type` | body | 来源 |
|--------|------|------|
| `note` | 打包后的 Note（见「对象 schema」） | 本地 `POST /api/notes/create` 成功后（`internal/httpapi/streaming_publish.go:27`–`internal/httpapi/streaming_publish.go:45`，调用点 `internal/httpapi/notes.go:110`） |

注意：时间线上的 note 是**以匿名视角打包**的（`packNoteFor(n, "")`，`internal/httpapi/streaming_publish.go:32`），所以**不带 `myReaction`**，poll 的每个 choice 的 `isVoted` 恒为 false。客户端必须用自己的本地状态覆盖这两个字段。

B. `main` 频道

| `type` | body | 触发点 |
|--------|------|--------|
| `notification` | Notification 对象 | 本机产生通知时（`internal/httpapi/streaming_publish.go:60`–`internal/httpapi/streaming_publish.go:69`；调用点 `internal/httpapi/notes.go:183`、`internal/httpapi/notes.go:500`） |
| `mention` | Note（detail 视角，按被提及者打包，带 `myReaction`） | 新笔记 @ 到本地用户（`internal/httpapi/streaming_publish.go:50`–`internal/httpapi/streaming_publish.go:56`；`internal/httpapi/notes.go:181`） |
| `reply` | Note（匿名视角） | 有人回复我的笔记（`internal/httpapi/streaming_publish.go:36`–`internal/httpapi/streaming_publish.go:38`） |
| `renote` | Note（匿名视角） | 有人 renote 我的笔记（`internal/httpapi/streaming_publish.go:41`–`internal/httpapi/streaming_publish.go:44`） |
| `follow` | `{id: <关注者 userId>}` | 有人关注我（`internal/httpapi/streaming_publish.go:106`–`internal/httpapi/streaming_publish.go:112`；调用点 `internal/httpapi/following.go:75`、`internal/httpapi/following_requests.go:145`） |
| `followed` | `{id: <被关注者 userId>}` | 我关注了别人（同上） |
| `unfollow` | `{id: <取关者 userId>}` | 有人取关我（`internal/httpapi/streaming_publish.go:115`–`internal/httpapi/streaming_publish.go:120`；调用点 `internal/httpapi/following.go:128`） |
| `readAllNotifications` | `null` | `POST /api/notifications/mark-all-as-read`（`internal/httpapi/notifications.go:13`–`internal/httpapi/notifications.go:22`，调用点 `internal/httpapi/notifications.go:121`） |

`mention` 还会过一次 `isNoteVisibleForMe`，不可见就不发（`internal/streaming/channel_main.go:57`–`internal/streaming/channel_main.go:67`）。其他类型原样透传，`body` 就是发布时的 JSON（`internal/streaming/channel_main.go:70`–`internal/streaming/channel_main.go:76`）。

C. `subNote` 订阅（`noteUpdated` 帧）

| `type` | body | 触发点 |
|--------|------|--------|
| `reacted` | `{reaction: string, emoji: null, userId: string}` | 添加反应（`internal/httpapi/streaming_publish.go:72`–`internal/httpapi/streaming_publish.go:86`；`internal/httpapi/notes.go:494`） |
| `unreacted` | `{reaction: string, userId: string}` | 撤销反应（同上；`internal/httpapi/notes.go:553`） |
| `deleted` | `{deletedAt: string}`（RFC3339Nano） | 删除笔记（`internal/httpapi/streaming_publish.go:89`–`internal/httpapi/streaming_publish.go:96`；`internal/httpapi/notes.go:355`） |

`noteUpdated.body.body` 就是上表对象；`PublishNoteStream` 生成的外层信封是 `{id, userId, visibility, visibleUserIds, body}`（`internal/streaming/bus.go:206`–`internal/streaming/bus.go:226`），`noteUpdated` 再把 `id` / `type` / `body` 提到帧顶层。事件里**不含**更新后的反应计数，客户端要自己 patch。

**没有 `pollVoted` 事件**：投票后不发任何流事件，客户端需重新拉取笔记（`internal/httpapi/notes_polls.go:92`–`internal/httpapi/notes_polls.go:93`）。

D. 不会出现的事件（对照 Misskey）

`unreadNotification`、`notificationFlushed`、`meUpdated`、`signin`、`driveFileCreated`、`announcementCreated`、`pollVoted`，以及 `broadcast` 一类全员广播（`SubjectBroadcast` 常量存在但没有任何发布点，`internal/streaming/bus.go:35`）。见「已知缺口」。

### 对象 schema

#### Note

时间线 `note` 事件、`mention` / `reply` / `renote` 事件的 body 都是 `packNoteDetail` 的输出（`internal/httpapi/notes_pack.go:42`–`internal/httpapi/notes_pack.go:151`）：

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | |
| `createdAt` | string | RFC3339Nano |
| `userId` | string | |
| `user` | UserLite | 作者；用户查不到时退化为 `{id}` |
| `text` | string \| null | |
| `cw` | string \| null | |
| `visibility` | string | `public` / `home` / `followers` / `specified` |
| `localOnly` | boolean | |
| `renoteCount` | number | |
| `repliesCount` | number | |
| `reactionCount` | number | 反应总数 |
| `reactions` | object | `{emoji: count}` |
| `reactionEmojis` | array | 恒 `[]` |
| `emojis` | array | 恒 `[]` |
| `fileIds` | array\<string\> | |
| `files` | array\<DriveFile\> | 按 `fileIds` 顺序；查不到的文件被跳过 |
| `replyId` | string \| null | |
| `renoteId` | string \| null | |
| `mentions` | array\<string\> | 本地用户 id；无提及为 `[]` |
| `tags` | array\<string\> | |
| `uri` | string \| null | 远程 URI，本地笔记为 null |
| `url` | string \| null | |
| `visibleUserIds` | array\<string\> | **仅** `visibility == "specified"` 时存在 |
| `reactionAcceptance` | string | **仅**数据库有值时存在，否则字段缺失 |
| `clippedCount` | number | detail 字段 |
| `reply` | Note \| null | **仅** `replyId != null` 时出现；父笔记不可见或已删除时为 `null`；内嵌对象的 `replyId` / `renoteId` 被置为 null |
| `renote` | Note \| null | **仅** `renoteId != null` 时出现；同上 |
| `hasPoll` | `true` | **仅**带 poll 的笔记 |
| `poll` | Poll \| null | **仅**带 poll 的笔记 |
| `myReaction` | string | **仅**当打包时有 viewer 且查到了反应（因此时间线事件里恒缺） |

Poll：

```json
{
  "multiple": false,
  "expiresAt": 1762300800000,
  "choices": [{"text": "A", "votes": 3, "isVoted": false}],
  "votesCount": 3
}
```

`expiresAt` 为毫秒时间戳或 `null`（`internal/httpapi/notes_pack.go:190`–`internal/httpapi/notes_pack.go:196`）。

#### Notification（`main` 频道 `notification` 事件）

`internal/httpapi/streaming_pack.go:58`–`internal/httpapi/streaming_pack.go:80`：

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | |
| `createdAt` | string | RFC3339Nano |
| `type` | string | 实际会出现：`reaction`、`reply`、`renote`、`quote`、`mention` |
| `isRead` | boolean | |
| `userId` | string \| null | 触发者（notifier） |
| `noteId` | string \| null | |
| `reaction` | string \| null | 仅 `reaction` 类型有值 |
| `user` | UserLite | 触发者；用户查不到时**字段缺失**（不是 null） |
| `note` | Note | 目标笔记，按**收件人视角**打包（带 `myReaction`）；笔记查不到时**字段缺失** |

#### UserLite（`Note.user` / `Notification.user`）

`internal/httpapi/users_pack.go:12`–`internal/httpapi/users_pack.go:41`：

`id`、`name`(string|null)、`username`、`host`(string|null，本地用户为 null)、`avatarUrl`(string|null)、`avatarBlurhash`(恒 null)、`avatarDecorations`(恒 `[]`)、`isBot`(恒 false)、`isCat`(恒 false)、`requireSigninToViewContents`(恒 false)、`emojis`(恒 `[]`)、`onlineStatus`(恒 `"unknown"`)、`badgeRoles`(恒 `[]`)、`instance`(恒 null)。

#### DriveFile（`Note.files`）

`internal/httpapi/drive.go:249`–`internal/httpapi/drive.go:264`：

`id`、`createdAt`（格式固定为 `2006-01-02T15:04:05.000Z`，**与 Note 的 createdAt 格式不同**）、`name`、`type`(MIME)、`size`(number)、`isSensitive`(boolean)、`url`、`thumbnailUrl`(null)、`webpublicUrl`(null)、`comment`(null)、`folderId`(string|null)、`blurhash`(null)。

### 错误

WebSocket 建立之后，**协议层没有任何错误帧**：订阅失败、可见性拒绝、帧解析失败、未知频道——全部静默。唯一的错误面是「握手失败响应」表里的三处（503 / 401 / 403）。

### 已知缺口与和 Misskey 的差异

1. **空闲连接会被断开（影响所有前端，优先修）**：`lastSeen` 只在读循环收到数据帧时更新（`internal/streaming/connection.go:98`），`heartbeat` 用 `lastSeen` 判 `pongTimeout`（`internal/streaming/conn.go:127`–`internal/streaming/conn.go:147`）。代码里没有任何 `SetPongHandler`（全仓 grep 无命中），所以**协议层 pong 不刷新计时**。后果：连接在约 120~180 秒内没有任何应用层帧时，会被服务端 `terminate()`，即使它一直在回 pong。`internal/streaming/conn.go:20`–`internal/streaming/conn.go:22` 的注释「Only pong frames refresh liveness」与实现不符。前端对策：每 ~60 秒发一个无害帧（例如再发一次 `subNote`，或重复 `connect` 同 id 且 `pong: false`），并实现自动重连 + 用 REST 补拉。
2. **`main` 频道不可重复订阅**：同一连接上第二次 `connect` `main`（哪怕换了 id）会被静默忽略，且不回 `connected`（`internal/streaming/connection.go:199`–`internal/streaming/connection.go:208`）。前端组件务必共享一个 `main` 订阅，不要每个组件各连一次。
3. **`readNotification` 不广播 `readAllNotifications`**：WS 帧直接调 `notifications.MarkAllRead`（`internal/httpapi/streaming.go:21`–`internal/httpapi/streaming.go:24`），而只有 `POST /api/notifications/mark-all-as-read` 才发 `readAllNotifications`（`internal/httpapi/notifications.go:13`–`internal/httpapi/notifications.go:22`）。多端同时在线时，一端清未读，其他端的未读徽章不会自动清零。
4. **没有 `follow` 通知行**：`follow` / `followed` 只是 streaming 事件，`internal/httpapi/following.go:75`、`internal/httpapi/following.go:128`、`internal/httpapi/following_requests.go:145` 都不写 `notifications` 表。通知页面拿不到「X 关注了你」，只能靠实时事件临时拼一条。
5. **入站联邦产生的通知不推送**：`federation.Store.Notification` 只写库（`internal/federation/store.go:619`–`internal/federation/store.go:626`），`Inbox` 只暴露 `OnNoteCreated` 一个回调（`internal/federation/inbox.go:27`；`internal/httpapi/federation_wire.go:46`–`internal/httpapi/federation_wire.go:52`）。远程的回复 / @ / 引用不会实时进 `main` 频道，只能轮询 `POST /api/notifications`。
6. **远程笔记的实时事件是残缺对象**：`OnNoteCreated` 只发 `{id, userId}`（`internal/httpapi/federation_wire.go:50`）。`localTimeline` / `globalTimeline` 会因为 `visibility` 为空而丢弃它；`homeTimeline` / `hybridTimeline` 在「我关注该远程作者」时会把它当成可见笔记推出，客户端会收到缺少 `text` / `user` / `createdAt` 的半成品对象。远程时间线内容请以 `POST /api/notes/global-timeline` 轮询为准。
7. **时间线事件没有 `myReaction`**：`notesStream` 一律按匿名视角打包（`internal/httpapi/streaming_publish.go:32`）。只有 `mention` 事件与 Notification 里的 `note` 带 viewer 视角。
8. **OAuth token 连不上 Streaming**：OAuth 的 `scopes_supported` 是 `read` / `write` / `follow` / `upload`（`internal/httpapi/oauth.go:44`），而 Streaming 只认字面量 `read:account`（`internal/httpapi/streaming_adapters.go:36`–`internal/httpapi/streaming_adapters.go:43`）。MiAuth token 可以（`permission` 数组原样入库，`internal/httpapi/account.go:321`–`internal/httpapi/account.go:331`），OAuth 授权的 token 一律 401。前端若走 OAuth，只能降级为轮询。
9. **回复过滤的 `withReplies` 恒为 false**：`followings` 表没有 `withReplies` 列，`LoadFollows` 固定填 `false`（`internal/httpapi/streaming_adapters.go:59`–`internal/httpapi/streaming_adapters.go:62`）。所以 `homeTimeline` 永远走严格回复分支，只能看到「回复我 / 我发的回复 / 作者自回复」。
10. **mute / block / 实例屏蔽 / `channelId` 过滤未实现**：`isNoteMutedOrBlocked` 恒 false（`internal/streaming/channel_timeline.go:99`），带 `channelId` 的笔记一律丢弃（`internal/streaming/channel_timeline.go:90`–`internal/streaming/channel_timeline.go:93`）。用户在客户端的静音、屏蔽设置对实时流**不生效**，需要前端自己过滤。
11. **`mention` 事件依赖纯文本正则**：`note.ParseMentions` 从 `text` 里提取 `@user` / `@user@host`（`internal/note/mention.go:31`–`internal/note/mention.go:56`，调用点 `internal/note/service.go:254`），不解析 MFM 结构，代码块里的 @ 也会命中。远程 `@user@host` 只解析、不建本地通知。
12. **`reacted` 事件的 `emoji` 恒为 null**：`publishNoteReaction` 只对 `reacted` 补 `body["emoji"] = nil`（`internal/httpapi/streaming_publish.go:80`–`internal/httpapi/streaming_publish.go:83`），自定义 emoji 的 shortcode 也不填。客户端若需要区分自定义 emoji，只能解析 `reaction` 字符串本身。
13. **未实现的频道**：`userList`、`clip`、`hashtag`、`roleTimeline`、`antenna`、`channel`（Misskey 的频道流，与帧类型 `channel` 同名但无关）、`drive`、`serverStats`、`queueStats`、`admin`、`chat*`、`reversi*`（`internal/streaming/channel_main.go:16`–`internal/streaming/channel_main.go:26`）。订阅它们没有任何反馈。
14. **没有 `broadcast` 展开帧**：`SubjectBroadcast` 常量存在（`internal/streaming/bus.go:35`），但无发布点，也没有对客户端的展开逻辑。
15. **跨进程对大事件有上限**：PG NOTIFY 路径单事件 8000 字节、分片上限 128 片（约 384 KiB 正文），超限只记日志丢弃（`internal/streaming/bus.go:234`–`internal/streaming/bus.go:254`、`internal/streaming/bus.go:353`–`internal/streaming/bus.go:369`）。Redis 路径无此限制（`internal/streaming/redis.go:33`–`internal/streaming/redis.go:46`）。
16. **`docs/research/2026-10-28-m4-streaming-source-calibration.md` §7.2 已过时**：该文档说 `Mentions` 恒空、mention 事件不会触发；`internal/note/mention.go` 落地后文本解析已生效，mention 事件与提及通知都会产生。
17. **没有 `unreadNotification` 延迟事件**：Misskey 在通知创建 2 秒后仍未读时补推一条，Cinderfish 只发一次 `notification`。
