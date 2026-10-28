# MilnFish 交接说明

- 最后更新：2026-10-28（M5 联邦完成）
- 项目暂用名 **MilnFish**（非最终命名，命名待定）
- 仓库：`/home/lainsaka/文档/MilnFish`，分支 `main`，工作区干净（`tmp/` 是本地产物，不入库）

## 1. 一句话说明

Go 单二进制的个人/极小规模联邦社交实例。后端契约是 **Misskey 原生 API**，通信层是 **ActivityPub**，前端是自建的 **Firefish 风格 Vue 3** 应用。

## 2. 已完成

| 里程碑 | 状态 | 提交 |
|--------|------|------|
| M1 骨架 | ✅ | `883874c` |
| 源码校准（meta/ping/错误信封） | ✅ | `0ba974c` |
| M2 账户与认证 | ✅ | `1a1abed` |
| M3 笔记/时间线/反应/通知/存储 | ✅ | `f4b8508` |
| M4 Streaming（WebSocket） | ✅ | 见 git log |
| M5 联邦（ActivityPub） | ✅ | `ce7bf11` … `620435d` |
| M6 前端 | ⬜ 未开始 | — |
| M7 打磨 | ⬜ 未开始 | — |

## 3. 必须先知道的规则（写在 AGENTS.md 里）

### 3.1 契约以源码为准，不要靠文档猜

`misskey-dev/misskey` 是 **AGPL-3.0-only**。用户已明确表示开源许可无所谓，因此：

- **实现任何端点前，先读源码**：`packages/backend/src/server/api/endpoints/<name>.ts` 的 `paramDef`（参数 JSON Schema）和 `res`（响应 schema），再查 `packages/misskey-js/src/api.types.ts` 的类型定义。
- 本地已备好浅克隆：`/tmp/refcheck/misskey`（不入库，**重启后会丢失**）。若被清理，用代理重建：
  ```bash
  export https_proxy=http://127.0.0.1:19823 http_proxy=http://127.0.0.1:19823
  git clone --depth 1 --filter=blob:none --sparse https://github.com/misskey-dev/misskey.git /tmp/refcheck/misskey
  cd /tmp/refcheck/misskey && git sparse-checkout set packages/backend/src packages/misskey-js/src
  ```
  后端源码全量约 11 MB；Streaming 实现在 `packages/backend/src/server/api/stream/`。
- **文档是二手的，源码是一手的。** 实际已发现文档与源码有多处出入（`instanceUrl` vs `uri`、`ping` 语义、错误信封结构、`features` 键集等）。

### 3.2 关键上游细节（踩过的坑，别再踩）

1. **错误信封有两种**：
   - 普通 API：`{"error":{"message","code","id","kind"}}`，`kind` 必填（client/server/permission）。
   - `signin` 失败：**只有 `id`**：`{"error":{"id":"932c904e-..."}}`。
   - OAuth 错误：**扁平**：`{"error":"invalid_grant","error_description":"..."}`。
2. **原生 token 判据是长度 16**（`misc/token.ts`），存 `users.token`；MiAuth/OAuth token 存 `access_tokens`，两套查找路径。
3. **`/api/i` 的 secrets 只对原生 token 返回**（`includeSecrets = token == null`）。
4. **`meta` 用 `uri` 不是 `instanceUrl`**；`detail` 参数控制 lite/detailed。
5. **`ping` 无参数**，返回服务端 `Date.now()`。
6. **联邦模式原生三态** `all | specified | none`，与项目要求的"公开/白名单/不互联" 1:1 对应。
7. **白名单是后缀匹配**：`.${host}`.endsWith(`.${x}`)，匹配 `sub.example.com` 但不匹配 `notexample.com`。
8. **联邦拦截有六处**（`isFederationAllowedUri`/`Host` 调用点）：AP 展示、Inbox 收件（两处）、出站投递、笔记解析、远程解析、Inbox 活动处理。详见 `docs/research/2026-10-26-misskey-source-calibration.md`。
9. **可见性规则**（`QueryService.generateVisibilityQuery`）：public/home 对所有人可见；followers 需关注者或回复目标；specified 需在可见列表或被 @。
10. **LTL/GTL 只含 public 可见性**的笔记；home 可见性的笔记不进 LTL。
11. **纯 renote 没有文本是合法的**（上游 `if/then` schema 允许 renoteId 存在时无 text）。
12. **回复默认从时间线排除**，但"回复自己写的笔记"要保留（判定要用被回复笔记的作者，不是 `reply_user_id`）。

## 4. 已定案的产品决策

| 决策 | 内容 |
|------|------|
| 语言 | Go 1.26 |
| 前端 | Vue 3 + TypeScript + Vite（与 Misskey/Firefish 生态同源，可复用 misskey-js 类型与 MFM 逻辑） |
| 数据库 | PostgreSQL 唯一强制依赖；**Redis 可选**（默认关闭，队列/广播由 PostgreSQL 承载） |
| **Mastodon API** | **明确放弃**，连路由空位都不留 |
| 注册模式 | open / invite / admin 三种，配置切换 |
| 联邦模式 | public / allowlist / none 三种，运行时切换 |
| 媒体 | local + S3 兼容双实现 |
| 数据迁移 | 不迁移朋友私有项目数据，空实例起步 |
| Relay | 首版不做 |
| 参考边界 | 只参考协议与 API 契约；Pleroma/Akkoma 代码明确排除 |

## 5. 环境

- Go 1.26.8（用户通过包管理器安装），`GOPROXY=https://mirrors.aliyun.com/goproxy/,direct`
- PostgreSQL 18（本次会话中用 sudo 装的，集群运行中）：
  - 库 `milnfish`（开发用）、`milnfish_test`（测试用）
  - 角色 `milnfish` / 密码 `milnfish`
- **访问 GitHub 必须走代理** `127.0.0.1:19823`（http 与 socks5h 均可）
- `dl.min.io` 被代理拦截，MinIO 二进制下不下来；S3 后端**尚未做真实服务端验证**（只验证了编译与接口）
- S3 客户端用的是 `github.com/minio/minio-go/v7`，不是 AWS SDK——AWS SDK v2 的依赖树太大，走现有镜像下载极慢（6 分钟只拉了 2MB 后被中止）。minio-go 对 AWS S3、Cloudflare R2、Backblaze B2、MinIO 都通用。

## 6. 怎么跑

```bash
cd /home/lainsaka/文档/MilnFish

# 编译与测试
go vet ./...
go test ./...                                    # 不带数据库的测试
MILNFISH_TEST_DATABASE_URL="postgres://milnfish:milnfish@localhost:5432/milnfish_test" go test ./...

# 运行
cp config.example.toml config.toml               # 按需改
go build -o milnfish ./cmd/milnfish
./milnfish --config config.toml
```

数据库迁移在启动时自动执行（不可变的版本化脚本，已应用版本不再改动，新功能一律追加新版本号）。当前 schema 版本为 **3**。

数据库测试包（`internal/account`、`internal/httpapi`）都会 TRUNCATE 同一批表，因此每个包用 `internal/testdb` 在 `TestMain` 里持有 PostgreSQL advisory lock，直到该包的测试二进制退出——这样 `go test ./...` 并行跑包时不会互相清表（不加锁时会随机出现"找不到用户"之类的假失败）。

配置项可用 `MILNFISH_*` 环境变量覆盖，例如 `MILNFISH_PORT`、`MILNFISH_DOMAIN`、`MILNFISH_REGISTRATION_MODE`、`MILNFISH_FEDERATION_MODE`、`MILNFISH_MEDIA_BACKEND`、`MILNFISH_MEDIA_LOCAL_PATH`。

## 7. 代码结构

```
cmd/milnfish/main.go      入口：配置、迁移、存储初始化、优雅关闭
internal/config/          TOML + 环境变量覆盖 + 校验
internal/db/              迁移执行器（版本化 SQL 脚本内嵌在代码里）
internal/idgen/           Misskey 风格时间有序字符串 ID
internal/auth/            bcrypt、token 生成、PKCE S256、用户名校验
internal/account/         账户服务（三种注册模式）、token 签发与解析
internal/note/            笔记服务、时间线服务、反应服务、通知服务
internal/media/           Storage 接口 + local 实现 + S3 实现 + Drive 服务
internal/streaming/       Streaming API：事件总线、WebSocket 连接、五个频道
internal/federation/      联邦：HTTP 签名、三模式策略、AP 渲染/解析、inbox、投递队列
internal/httpapi/         路由装配与全部端点处理器
```

依赖方向单向：`httpapi → 领域包 → db/media(接口)`。`streaming` 只定义接口（`Authenticator`、`ViewerLoader`），实现由 `httpapi` 注入，因此不反向依赖 HTTP 层。

## 8. 已实现的端点

```
POST /api/meta                          GET  /api/ping
POST /api/signup                        POST /api/signin
POST /api/username/available            POST /api/i
POST /api/users/show                    POST /api/miauth/gen-token
POST /api/notes/create                  POST /api/notes/show
POST /api/notes/delete                  POST /api/notes/reactions/create
POST /api/notes/reactions/delete        POST /api/notes/local-timeline
POST /api/notes/global-timeline         POST /api/notes/hybrid-timeline
POST /api/notes/timeline                POST /api/notifications
POST /api/notifications/mark-all-as-read
POST /api/drive/files/create            POST /api/drive/files
POST /api/drive/files/show              POST /api/drive/files/delete
GET  /media/{path...}
GET  /.well-known/oauth-authorization-server
GET  /oauth/authorize                   POST /oauth/token
GET  /streaming                         （WebSocket 升级）
```

M5 联邦路由见 §10.1。

## 9. M4 Streaming API（已完成）

协议细节见 `docs/research/2026-10-28-m4-streaming-source-calibration.md`（逐条对照上游源码，含踩坑清单）。要点：

1. **端点**：`GET /streaming`，与 HTTP 共用端口，用 gorilla/websocket 原地升级。
2. **鉴权**：`Authorization: Bearer <t>` 优先，否则 query `?i=<t>`。**无 token 合法**（匿名连接）；只有 token 无效才 401（带 `WWW-Authenticate`），封禁用户 403。app token 必须含 `read:account`，原生 16 位 token 不受此限。
3. **频道**：`main`（需登录，可共享）、`homeTimeline`（需登录）、`localTimeline`（匿名可用）、`hybridTimeline`（需登录）、`globalTimeline`（匿名可用）。
4. **客户端消息**：`connect` / `disconnect` / `ch`（`channel` 的别名）/ `subNote`（别名 `s`、`sr`）/ `unsubNote`（别名 `un`）/ `readNotification`。
5. **未实现的频道静默忽略**，与上游"构造器抛错但被消息队列吞掉"的可观测行为一致。
6. **上限**：每连接 32 频道、1536 条 subNote（LRU 淘汰）、256 条发送队列（对应上游 8 MiB bufferedAmount 阈值；溢出时关闭 socket，读取循环随之退出并释放全部订阅）。心跳 60s ping / 120s 无 pong 断开。
7. **事件**：`notesStream`（进程内 fanout 到所有时间线频道）、`mainStream:<userId>`（notification / mention / reply / renote / readAllNotifications）、`noteStream:<noteId>`（reacted / unreacted / deleted）。
8. **跨进程**：`LISTEN/NOTIFY`（channel `milnfish_stream`）中继，为多进程部署预留；单进程下语义正确。NOTIFY payload 有 8000 字节硬上限，超出的事件体先 base64 再分片、接收端重组（单事件上限约 384 KiB 正文），不会退化成"只发 subject 的空唤醒"。**转发规则**：只有产出事件的进程负责中继；监听器收到其他进程的事件后只做本地分发（`deliver`），并用事件里的 `origin` 丢弃自己发布的回声——否则同一个 bus 同时发布和监听会形成无限放大（实测一次事件 1 秒内投递上万次）。

### 9.1 M4 已知缺口（不是 bug，是后续里程碑的范围）

- ~~**mention 永远不触发**~~：**M5 已修**。`note.ParseMentions` 解析 @user 与 @user@host，创建笔记时会解析本地提及并生成通知与事件。
- ~~**follow / followed 事件无来源**~~：**M5 已修**。`/api/following/create` 与 `/api/following/delete` 会发布 follow / followed / unfollow。
- **mute / block 过滤恒为 false**：`isNoteMutedOrBlocked` 的调用点已就位，等数据模型落地。
- **unreadNotification 提醒未实现**：需要"最近已读通知"游标。
- **events 表未写入**：表存在，但 M4 不做重放。

第一轮审查提出的两条 P1 已修复并有回归测试：过载连接会真正关闭 socket 并释放订阅；跨进程大事件改为分片中继。P2（channel id 替换与上限）核对后判定不成立，已补回归测试钉住行为。详细对照见 `docs/research/2026-10-28-m4-streaming-source-calibration.md` §10。

## 10. M5 联邦（ActivityPub）

源码对照记录见 `docs/research/2026-10-28-m5-federation-source-calibration.md`（逐条对照上游文件，含刻意偏离项）。

### 10.1 已实现的路由

```
GET  /.well-known/webfinger            GET  /.well-known/host-meta(.json)
GET  /.well-known/nodeinfo             GET  /nodeinfo/2.1  /nodeinfo/2.0
GET  /users/{id}                       GET  /@{handle}
GET  /users/{id}/publickey             GET  /users/{id}/outbox
GET  /users/{id}/followers             GET  /users/{id}/following
GET  /users/{id}/collections/featured
GET  /notes/{id}                       GET  /notes/{id}/activity
POST /inbox                            POST /users/{id}/inbox
```

RPC 侧新增：

```
POST /api/following/create             POST /api/following/delete
POST /api/admin/federation/mode        GET/POST /api/admin/federation/allowlist
```

### 10.2 关键实现要点

1. **签名**：出站 POST 签 `(request-target) date host digest`，GET 签 `(request-target) date host`，`keyId` 形如 `<url>/users/<id>#main-key`。入站按上游顺序校验：解析 Signature → host/date 必须被覆盖 → Host 必须是本机 → digest 格式 → digest 值 → RSA 签名。`Date` 超出 ±5 分钟直接拒绝。
2. **内容协商**：`Accept` 含 `application/ld+json` 回 LD，否则回 `application/activity+json`；都带 `Vary: Accept` 与 `Cache-Control: public, max-age=180`。
3. **三模式**：模式存在 `instance_meta.federationMode`，每次检查都读库，所以 `POST /api/admin/federation/mode` 立即生效。白名单是 `federation_allowlist` 表，**后缀匹配**（`example.com` 匹配 `sub.example.com`，不匹配 `notexample.com`）。配置里的 `federation.allowlist_hosts` 与 `federation_mode` 只做首次 seed（`ON CONFLICT DO NOTHING`），重启不会覆盖运行时改动。
4. **门禁位置**：`none` 时全部 AP 路由 403（含 well-known）；`allowlist` 时入站按来源 host、出站按目标 host 判断，出站是**入队前和发送前各查一次**，切到 `none` 会立刻停掉队列里积压的投递。
5. **投递**：`deliver_queue` 表，**按 inbox 一行**（一条笔记给多个 follower 各有一行各自的退避）。指数退避 1s→5min 上限，12 次后放弃并记 `last_error`。follower inbox 与被 @ 用户的 inbox 合并去重（优先 `shared_inbox`）。
6. **inbox 活动**：`Follow`（锁定账号进 `follow_requests` 并回 Accept，否则直接关注并投递 Accept）、`Accept`/`Reject`（我们发出的 Follow 的回应）、`Undo`（Follow/Like/Announce）、`Create<Note>`（校验 `attributedTo` 与 host 一致、按 AP id 去重）、`Delete`（笔记或账号）、`Like`（读 `_misskey_reaction`）、`Announce`、`Update`、`Block`。返回值是 Misskey 风格的 `ok` / `skip: 原因` 字符串，写日志；**HTTP 状态一律 202**，避免远端用重试解决语义问题。
7. **正文渲染**：项目存 MFM、没有存 HTML，所以 `content` 由 `federation.MFMToHTML` 生成（提及/话题标签/自定义 emoji/裸链接/粗斜体/行内与围栏代码）。代码块先提取再转义，所以代码里的 URL 不会被链接化。
8. **用户名唯一性改了**：v2 的 `idx_users_username_lower` 是全表唯一，远程用户进同一张表后 `alice@a.example` 会和本地 `alice` 撞。v4 改成：本地用户名唯一 + `(username_lower, host)` 唯一。

### 10.3 测试覆盖

- 单元：签名签发/验证往返（POST 与 GET）、篡改正文、错误 Host、未覆盖 digest、过期 Date、`Authorization: Signature` 形式、白名单后缀匹配、MFM 转换、Person/Note 渲染、`ParseMentions`。
- 集成（真库 + httptest，`MILNFISH_TEST_DATABASE_URL` 必需）：webfinger 三种 resource 形状与 422/404/400、actor 文档字段、`@handle` 路由、nodeinfo、两种 host-meta、`none` 下全路由 403。
- **双实例 smoke**：两个真实实例、真实 HTTP、真实 RSA 签名。alpha 的 Follow 被 beta 验签通过并记录，beta 投递签名 Accept，alpha 验签通过并记录关注。域名到监听地址的跳转由一个 rewrite transport 完成，密码学与 HTTP 环没有任何 mock。
- 管理面：非管理员改模式 403、切到 `none` 后下一次请求即 403、非法模式 400、白名单增删与后缀匹配语义。

### 10.4 M5 已知缺口（不是 bug，是范围）

- **XRD 形式的 webfinger（`application/xrd+xml`）不返回 JRD 以外的格式**：只做 JSON。现代联邦软件都只用 JRD。
- **LinkedDataSignature（`signatureValue`）不实现**：只用 HTTP Signatures，Mastodon 与 GoToSocial 也一样。
- **inbox 同步处理**：验签通过立刻回 202，但活动在请求内处理完毕，没有单独的任务队列。单机个人实例够用；慢速远程在客户端超时前会完成。
- **只有第一个管理员/第一个本地用户持有实例签名密钥**：`signingUserID` 取 `is_admin DESC, created_at ASC` 的第一人。个人实例够用，多用户实例需要把出站签名改成「按活动作者签名」。
- **锁定账号**：`users.is_locked` 列存在，但项目还没有「锁推」设置项，所以本地关注一律立即完成；只有远程 Follow 会走 `follow_requests`。
- **置顶笔记**（`/collections/featured`）恒为空：置顶功能未实现。
- **Remote media 不缓存/不代理**：直接用远端 URL。
- **没有 `/api/ap/show` 与 `/api/ap/get`**：管理员手动抓取远程对象的端点还没做。
- **没有被拉黑的实例列表、没有 relay**：Relay 按定案不做。
- **没有真实三方联调**：没有对 Mastodon/GoToSocial/真实 Misskey 实例跑过。双实例测试证明的是本实现自洽，不是与上游互通。**这是 M5 结束后最该补的一项验证。**


## 11. 文档索引

| 文档 | 内容 |
|------|------|
| `AGENTS.md` | 项目规约（**改代码前必读**） |
| `docs/superpowers/specs/2026-10-26-milnfish-architecture-v0.md` | 架构设计 v0（决策、模块、数据模型、API 集、里程碑、风险） |
| `docs/research/2026-10-26-misskey-fediverse-firefish.md` | 首轮调研：Misskey API、Streaming 协议、ActivityPub、Firefish 状态 |
| `docs/research/2026-10-26-go-ecosystem-and-dependency-decisions.md` | Go 生态、依赖取舍、GoToSocial/Iceshrimp 先例 |
| `docs/research/2026-10-26-tech-stack-comparison.md` | 五种语言的横向对比与推荐 |
| `docs/research/2026-10-26-misskey-source-calibration.md` | **源码校准记录（含许可证约束与踩坑清单）** |
| `docs/research/2026-10-26-m2-implementation-notes.md` | M2 实施记录与源码对齐细节 |
| `docs/research/2026-10-26-initial-scope.md` | 首版范围与已裁决项 |
| `docs/research/2026-10-28-m4-streaming-source-calibration.md` | **M4 实施记录：Streaming 协议逐条对照上游源码** |
| `docs/research/2026-10-28-m5-federation-source-calibration.md` | **M5 实施记录：WebFinger/Actor/Inbox/签名/NodeInfo 逐条对照上游源码** |

## 12. 已知的未验证项

- **S3 后端未做真实服务端验证**：只验证了编译、接口与 URL 生成逻辑。需要一个 MinIO 或 R2 实例做端到端验证（MinIO 二进制因代理拦截未下载成功，可试 GitHub releases 或其他镜像）。
- **缩略图/WebP 生成未实现**：`drive_files.thumbnail_key` 字段已留，但图片处理管道还没写。
- **联邦模式在代码里只体现了 `meta` 字段的映射**，六个拦截点尚未实现（属于 M5）。
- **前端完全未开始**（M6）。
- **Streaming 未做真实 Misskey 客户端联调**：已用自写 WebSocket 客户端对真实二进制验证（握手、`connect`、`connected`、笔记投递全部通过），也覆盖了 misskey-js 的频道契约，但**没有**用 misskey-js 或 Misskey Web 前端实际连过。M6 前端开发时应先做这一步。
- **联邦没有任何第三方联调**：只跑了本实现的双实例测试（真签名、真 HTTP、真库），没有对 Mastodon / GoToSocial / 真实 Misskey 实例互通。这是 M5 之后最该补的验证。
- **Streaming 多进程中继只手测了单连接的 LISTEN/NOTIFY 往返**，没有做真正的双进程联调；单进程部署不受影响。
