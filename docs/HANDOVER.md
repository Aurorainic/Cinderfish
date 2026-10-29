# Cinderfish 交接说明

- 最后更新：2026-10-29（M6 前端完成）
- 项目名 **Cinderfish**（2026-10-28 定名，Go module `github.com/Aurorainic/cinderfish`）
- 仓库：当前目录 `/home/lainsaka/文档/MilnFish`，分支 `main`；远程 `origin` 为
  `https://github.com/Aurorainic/Cinderfish.git`（GitHub 仓库已更名，本地目录改名待办）

## 1. 一句话说明

Go 单二进制的个人/极小规模联邦社交实例。后端契约是 **Misskey 原生 API**，通信层是 **ActivityPub**，前端是自建的 **Firefish 风格 Vue 3** 应用。

## 2. 里程碑

| 里程碑 | 状态 | 提交 |
|--------|------|------|
| M1 骨架 | ✅ | `883874c` |
| 源码校准（meta/ping/错误信封） | ✅ | `0ba974c` |
| M2 账户与认证 | ✅ | `1a1abed` |
| M3 笔记/时间线/反应/通知/存储 | ✅ | `f4b8508` |
| M4 Streaming（WebSocket） | ✅ | 见 git log |
| M5 联邦（ActivityPub） | ✅ | `ce7bf11` … `620435d` |
| M6 前端 | ✅ | `9281a5c` |
| M7 打磨 | ✅ | 见 git log（迁移工具、账号管理、性能基线、部署文档） |

## 3. 契约以源码为准

`misskey-dev/misskey` 是 **AGPL-3.0-only**。实现任何端点前，先读源码的 `paramDef`（参数 JSON Schema）和 `res`（响应 schema），再查 `packages/misskey-js/src/api.types.ts` 的类型定义。官方文档是二手的，已发现多处与源码不一致。

本地浅克隆：`/tmp/refcheck/misskey`（不入库，重启后丢失）。若被清理，用代理重建：

```bash
export https_proxy=http://127.0.0.1:19823 http_proxy=http://127.0.0.1:19823
git clone --depth 1 --filter=blob:none --sparse https://github.com/misskey-dev/misskey.git /tmp/refcheck/misskey
cd /tmp/refcheck/misskey && git sparse-checkout set packages/backend/src packages/misskey-js/src
```

后端源码全量约 11 MB；Streaming 实现在 `packages/backend/src/server/api/stream/`。

## 4. 关键上游细节（踩过的坑）

1. **错误信封有三种**：普通 API 用 `{"error":{"message","code","id","kind"}}`（`kind` 必填）；`signin` 失败只有 `id`；OAuth 错误是扁平结构。
2. **原生 token 判据是长度 16**（`misc/token.ts`），存 `users.token`；MiAuth/OAuth token 存 `access_tokens`，两套查找路径。
3. **`/api/i` 的 secrets 只对原生 token 返回**。
4. **`meta` 用 `uri` 不是 `instanceUrl`**；`detail` 参数控制 lite/detailed。
5. **`ping` 无参数**，返回服务端 `Date.now()`。
6. **联邦模式原生三态** `all | specified | none`，与项目要求的"公开/白名单/不互联" 1:1 对应。
7. **白名单是后缀匹配**：匹配 `sub.example.com` 但不匹配 `notexample.com`。
8. **联邦拦截有六处**（`isFederationAllowedUri`/`Host` 调用点），详见 `docs/research/2026-10-26-misskey-source-calibration.md`。
9. **可见性规则**：public/home 对所有人可见；followers 需关注者或回复目标；specified 需在可见列表或被 @。
10. **LTL/GTL 只含 public 可见性**的笔记。
11. **纯 renote 没有文本是合法的**（上游 `if/then` schema 允许 renoteId 存在时无 text）。
12. **回复默认从时间线排除**，但"回复自己写的笔记"要保留（判定要用被回复笔记的作者）。

## 5. 已定案的产品决策

| 决策 | 内容 |
|------|------|
| 语言 | Go 1.26 |
| 前端 | Vue 3 + TypeScript + Vite |
| 数据库 | PostgreSQL 唯一强制依赖；Redis 可选（默认关闭） |
| Mastodon API | 明确放弃，连路由空位都不留 |
| 注册模式 | open / invite / admin 三种，配置切换 |
| 联邦模式 | public / allowlist / none 三种，运行时切换 |
| 媒体 | local + S3 兼容双实现 |
| 数据迁移 | 不迁移朋友私有项目数据，空实例起步 |
| Relay | 首版不做 |
| 参考边界 | 只参考协议与 API 契约；Pleroma/Akkoma 代码明确排除 |

## 6. 环境

- Go 1.26.8，`GOPROXY=https://mirrors.aliyun.com/goproxy/,direct`
- PostgreSQL 18：库 `cinderfish`（开发）、`cinderfish_test`（测试）；角色/密码 `cinderfish`/`cinderfish`
- **访问 GitHub 必须走代理** `127.0.0.1:19823`
- `dl.min.io` 被代理拦截，MinIO 二进制下不下来；S3 后端**尚未做真实服务端验证**
- S3 客户端用 `github.com/minio/minio-go/v7`（不是 AWS SDK，依赖树太大）

## 7. 怎么跑

```bash
cd /home/lainsaka/文档/MilnFish   # 目录改名待办

go vet ./...
go test ./...
CINDERFISH_TEST_DATABASE_URL="postgres://cinderfish:cinderfish@localhost:5432/cinderfish_test" go test ./...

cd web && pnpm install && pnpm build && cd ..   # 产出 web/dist
cp config.example.toml config.toml   # 按需改
go build -o cinderfish ./cmd/cinderfish
./cinderfish --config config.toml

# registration_mode = admin 时没有自助注册，用命令行建第一个管理员
go run ./cmd/cinderfish-admin --config config.toml create --username root --password-stdin --admin
```

只跑 API 时可以不构建前端：`web_dist` 目录不存在，服务器自动退回纯 API 模式。

数据库迁移在启动时自动执行（不可变的版本化脚本）。当前 schema 版本为 **4**。

数据库测试包（`internal/account`、`internal/httpapi`）都会 TRUNCATE 同一批表，因此每个包用 `internal/testdb` 在 `TestMain` 里持有 PostgreSQL advisory lock，直到该包的测试二进制退出——这样 `go test ./...` 并行跑包时不会互相清表（不加锁时会随机出现"找不到用户"之类的假失败）。

### 7.1 媒体后端迁移

`cmd/cinderfish-media-migrate` 把 `drive_files` 里的对象从当前 `[media]` 配置的后端搬到另一个后端；对象 key（内容哈希）不变，只更新行的 `storage_backend`，笔记里的链接因此不需要改写。

```bash
# 本地 → S3（S3 参数取自 config.toml 的 [media.s3]）
go run ./cmd/cinderfish-media-migrate --config config.toml --to s3 --dry-run
go run ./cmd/cinderfish-media-migrate --config config.toml --to s3 --delete-source

# S3 → 本地
go run ./cmd/cinderfish-media-migrate --config config.toml --to local --to-local-path ./data/files
```

`--dry-run` 只统计；`--limit N` 限制条数；单个对象失败不会中断其余，报告会列出失败 key，退出码为 1。`--delete-source` 只在整轮无失败时清理源对象。迁移完成后把 `config.toml` 的 `media.backend` 切到新后端再重启。

配置项可用 `CINDERFISH_*` 环境变量覆盖，如 `CINDERFISH_PORT`、`CINDERFISH_DOMAIN`、`CINDERFISH_REGISTRATION_MODE`、`CINDERFISH_FEDERATION_MODE`、`CINDERFISH_MEDIA_BACKEND`、`CINDERFISH_MEDIA_LOCAL_PATH`、`CINDERFISH_WEB_DIR`。

## 8. 代码结构

```
cmd/cinderfish/main.go      入口：配置、迁移、存储初始化、优雅关闭
internal/config/            TOML + 环境变量覆盖 + 校验
internal/db/                迁移执行器（版本化 SQL 脚本内嵌在代码里）
internal/idgen/             Misskey 风格时间有序字符串 ID
internal/auth/              bcrypt、token 生成、PKCE S256、用户名校验
internal/account/           账户服务（三种注册模式）、token 签发与解析
internal/note/              笔记服务、时间线服务、反应服务、通知服务、提及解析
internal/media/             Storage 接口 + local 实现 + S3 实现 + Drive 服务
internal/streaming/         Streaming API：事件总线、WebSocket 连接、五个频道
internal/federation/        联邦：HTTP 签名、三模式策略、AP 渲染/解析、inbox、投递队列
internal/httpapi/           路由装配与全部端点处理器、静态托管与 SPA fallback
internal/testdb/            数据库测试的 advisory lock 辅助
web/                        Vue 3 + TypeScript 前端（api/stores/components/views/layouts）
```

依赖方向单向：`httpapi → 领域包 → db/media(接口)`。`streaming` 只定义接口，实现由 `httpapi` 注入。

## 9. 已实现的端点

```
POST /api/meta                          GET  /api/ping
POST /api/signup                        POST /api/signin
POST /api/username/available            POST /api/i
POST /api/users/show                    POST /api/users/notes
POST /api/miauth/gen-token
POST /api/notes/create                  POST /api/notes/show
POST /api/notes/delete                  POST /api/notes/reactions/create
POST /api/notes/reactions/delete        POST /api/notes/local-timeline
POST /api/notes/global-timeline         POST /api/notes/hybrid-timeline
POST /api/notes/timeline                POST /api/notifications
POST /api/notifications/mark-all-as-read
POST /api/drive/files/create            POST /api/drive/files
POST /api/drive/files/show              POST /api/drive/files/delete
POST /api/following/create              POST /api/following/delete
POST /api/admin/federation/mode         GET/POST /api/admin/federation/allowlist
GET  /media/{path...}
GET  /                                  （Web 前端，SPA fallback）
GET  /.well-known/oauth-authorization-server
GET  /oauth/authorize                   POST /oauth/token
GET  /streaming                         （WebSocket 升级）
```

M5 联邦路由见 §12.1。

## 10. M4 Streaming 要点

协议细节见 `docs/research/2026-10-28-m4-streaming-source-calibration.md`。要点：

- **端点**：`GET /streaming`，WebSocket 原地升级。
- **鉴权**：`Authorization: Bearer <t>` 优先，否则 query `?i=<t>`。无 token 合法（匿名）；token 无效才 401；封禁用户 403。app token 必须含 `read:account`，原生 16 位 token 不受此限。
- **频道**：`main`（需登录）、`homeTimeline`（需登录）、`localTimeline`（匿名可用）、`hybridTimeline`（需登录）、`globalTimeline`（匿名可用）。
- **客户端消息**：`connect` / `disconnect` / `ch` / `subNote` / `unsubNote` / `readNotification`。
- **上限**：每连接 32 频道、1536 条 subNote（LRU 淘汰）、8 MiB 发送积压。心跳 60s ping / 120s 无 pong 断开。
- **跨进程**：`LISTEN/NOTIFY`（channel `cinderfish_stream`）中继，为多进程部署预留。

### 10.1 已知缺口（后续里程碑范围）

- ~~**mention 永远不触发**~~：**M5 已修**。`note.ParseMentions` 解析 @user 与 @user@host，创建笔记时会解析本地提及并生成通知与事件。
- ~~**follow / followed 事件无来源**~~：**M5 已修**。`/api/following/create` 与 `/api/following/delete` 会发布 follow / followed / unfollow。
- **mute / block 过滤恒为 false**：`isNoteMutedOrBlocked` 的调用点已就位，等数据模型落地。
- **unreadNotification 提醒未实现**：需要"最近已读通知"游标。
- **events 表未写入**：M4 不做重放。

## 11. M6 前端

技术栈：Vue 3.5 + TypeScript 5.9 + Vite 8 + vue-router 4。没有 UI 组件库，也没有 Pinia——共享状态是模块级 `reactive` store（`web/src/stores/`），设计令牌在 `web/src/styles/main.css`。

### 11.1 目录

```
web/src/api/         类型（Misskey schema）与 fetch 客户端（ApiError 解析 error 信封）
web/src/stores/      session（token + /api/i）、instance（meta）、streaming（WebSocket）
web/src/lib/         MFM 渲染（先转义再生成标记）、相对时间
web/src/components/  NoteCard、NoteComposer、TimelineFeed、QuotedNote、UserAvatar、AppIcon 等
web/src/layouts/     AppShell（三栏 + 移动底栏）、AuthLayout
web/src/views/       Timeline、Note、Notifications、Profile、Explore、Settings、Admin、Login、Signup
```

### 11.2 已完成的页面

- **时间线**：关注 / 本地 / 混合 / 全球四频道，`untilId` 分页，WebSocket 实时插入新笔记，加载更多按页大小判断。
- **发帖**：CW、可见性（public/home/followers）、仅本地、多文件上传、回复上下文、Ctrl/Cmd+Enter 发布；发布后直接插入列表。
- **笔记卡片**：MFM 渲染（代码块、行内代码、提及、话题标签、链接、粗斜体/删除线、自定义 emoji 占位）、图片与附件、引用转贴、投票展示、反应（快捷表情 + 取消）、转贴、回复、删除自己的笔记。
- **笔记详情**：单条笔记 + 回复框。
- **通知**：全部/未读筛选，按类型生成中文文案，显示相关笔记摘要，全部标为已读，实时新通知置顶，侧栏未读徽标。
- **资料**：真实关注/被关注/笔记计数，加入时间，关注/取消关注，用户笔记分页；远程资料与 404 有明确说明。
- **设置**：账户信息、主题（浅色/深色，本机持久化）、实例信息、退出登录；明确说明资料编辑尚无后端接口。
- **管理**：联邦模式三选一（public/allowlist/none，立即生效）、白名单增删（后缀匹配说明），非管理员显示门禁卡片。
- **探索**：本地/全球公开时间线，支持 `?tag=` 过滤。
- **登录/注册**：真实 signin/signup，注册前检查用户名可用性，邀请码字段，注册关闭提示。

### 11.3 服务端托管

`web_dir`（默认 `./web/dist`，环境变量 `CINDERFISH_WEB_DIR`）指向构建产物。命中规则：

- `/api*`、`/media`、`/.well-known`、`/oauth`、`/users`、`/nodeinfo`、`/inbox`、`/streaming` 永远交给 Go mux。
- `/@user` 与 `/notes/{id}` 在 `Accept` 含 `text/html` 时给前端，否则仍走 ActivityPub。
- 其余 GET/HEAD 先找真实文件，找不到回退 `index.html`；`/assets/*` 带一年 immutable 缓存，`index.html` 每次读取（重新构建无需重启）。
- 目录不存在时服务器保持纯 API 模式。

### 11.4 实时更新

`web/src/stores/streaming.ts` 维护唯一 `/streaming` 连接：token 走 query，`connect` 带 `pong:true`，按订阅 id 分发 `channel` 帧，断线指数退避（1s→30s，6 次封顶），登录/退出时重建连接。`main` 频道只由外壳订阅一次（服务端每连接只允许一个），时间线各自订阅自己的频道，重连后重新拉取。

### 11.5 开发与构建

```bash
cd web
pnpm install
pnpm dev      # 5173，代理 /api /media /streaming 到 127.0.0.1:3000
pnpm build    # 先 vue-tsc --noEmit，再输出 web/dist
```

### 11.6 已知缺口

- **资料不可编辑**：后端没有 `i/update`，昵称、简介、头像都只读。
- **没有搜索、没有用户列表页、没有投票接口**（前端只展示投票结果）。
- **远程资料不能浏览**：`users/show` 只接受本地用户名；远程用户点击 `@user@host` 会得到说明卡片。
- **管理面板只有联邦设置**：账户管理、封禁、邀请码管理尚未实现。
- **没有前端自动化测试**：验证靠 `vue-tsc` + 构建 + 真实浏览器手工走查；没有引入 vitest/playwright。

## 12. M5 联邦（ActivityPub）

源码对照记录见 `docs/research/2026-10-28-m5-federation-source-calibration.md`（逐条对照上游文件，含刻意偏离项）。

### 12.1 已实现的路由

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

### 12.2 关键实现要点

1. **签名**：出站 POST 签 `(request-target) date host digest`，GET 签 `(request-target) date host`，`keyId` 形如 `<url>/users/<id>#main-key`。入站按上游顺序校验：解析 Signature → host/date 必须被覆盖 → Host 必须是本机 → digest 格式 → digest 值 → RSA 签名。`Date` 超出 ±5 分钟直接拒绝。
2. **内容协商**：`Accept` 含 `application/ld+json` 回 LD，否则回 `application/activity+json`；都带 `Vary: Accept` 与 `Cache-Control: public, max-age=180`。
3. **三模式**：模式存在 `instance_meta.federationMode`，每次检查都读库，所以 `POST /api/admin/federation/mode` 立即生效。白名单是 `federation_allowlist` 表，**后缀匹配**。配置里的 `federation.allowlist_hosts` 与 `federation_mode` 只做首次 seed（`ON CONFLICT DO NOTHING`），重启不会覆盖运行时改动。
4. **门禁位置**：`none` 时全部 AP 路由 403（含 well-known）；`allowlist` 时入站按来源 host、出站按目标 host 判断，出站是**入队前和发送前各查一次**，切到 `none` 会立刻停掉队列里积压的投递。
5. **投递**：`deliver_queue` 表，**按 inbox 一行**（一条笔记给多个 follower 各有一行各自的退避）。指数退避 1s→5min 上限，12 次后放弃并记 `last_error`。follower inbox 与被 @ 用户的 inbox 合并去重（优先 `shared_inbox`）。
6. **inbox 活动**：`Follow`（锁定账号进 `follow_requests` 并回 Accept，否则直接关注并投递 Accept）、`Accept`/`Reject`、`Undo`（Follow/Like/Announce）、`Create<Note>`（校验 `attributedTo` 与 host 一致、按 AP id 去重）、`Delete`（笔记或账号）、`Like`（读 `_misskey_reaction`）、`Announce`、`Update`、`Block`。返回值是 Misskey 风格的 `ok` / `skip: 原因` 字符串，写日志；**HTTP 状态一律 202**。
7. **正文渲染**：项目存 MFM、没有存 HTML，所以 `content` 由 `federation.MFMToHTML` 生成（提及/话题标签/自定义 emoji/裸链接/粗斜体/行内与围栏代码）。代码块先提取再转义，所以代码里的 URL 不会被链接化。
8. **用户名唯一性**：v2 的 `idx_users_username_lower` 是全表唯一，远程用户进同一张表后 `alice@a.example` 会和本地 `alice` 撞。v4 改成：本地用户名唯一 + `(username_lower, host)` 唯一。

### 12.3 测试覆盖

- 单元：签名签发/验证往返（POST 与 GET）、篡改正文、错误 Host、未覆盖 digest、过期 Date、`Authorization: Signature` 形式、白名单后缀匹配、MFM 转换、Person/Note 渲染、`ParseMentions`。
- 集成（真库 + httptest，`CINDERFISH_TEST_DATABASE_URL` 必需）：webfinger 三种 resource 形状与 422/404/400、actor 文档字段、`@handle` 路由、nodeinfo、两种 host-meta、`none` 下全路由 403。
- **双实例 smoke**：两个真实实例、真实 HTTP、真实 RSA 签名。alpha 的 Follow 被 beta 验签通过并记录，beta 投递签名 Accept，alpha 验签通过并记录关注。域名到监听地址的跳转由一个 rewrite transport 完成，密码学与 HTTP 环没有任何 mock。
- 管理面：非管理员改模式 403、切到 `none` 后下一次请求即 403、非法模式 400、白名单增删与后缀匹配语义。

### 12.4 已知缺口（不是 bug，是范围）

- **XRD 形式的 webfinger 不实现**：只做 JRD，现代联邦软件都只用 JRD。
- **LinkedDataSignature（`signatureValue`）不实现**：只用 HTTP Signatures。
- **inbox 同步处理**：验签通过立刻回 202，但活动在请求内处理完毕，没有单独的任务队列。
- **只有第一个管理员/第一个本地用户持有实例签名密钥**：`signingUserID` 取 `is_admin DESC, created_at ASC` 的第一人。个人实例够用，多用户实例需要改成「按活动作者签名」。
- **锁定账号**：`users.is_locked` 列存在，但还没有「锁推」设置项，本地关注一律立即完成；只有远程 Follow 会走 `follow_requests`。
- **置顶笔记**（`/collections/featured`）恒为空：置顶功能未实现。
- **Remote media 不缓存/不代理**：直接用远端 URL。
- **没有 `/api/ap/show` 与 `/api/ap/get`**。
- **没有被拉黑的实例列表、没有 relay**：Relay 按定案不做。
- **没有真实三方联调**：没有对 Mastodon/GoToSocial/真实 Misskey 实例跑过。双实例测试证明的是本实现自洽。

## 13. 文档索引

| 文档 | 内容 |
|------|------|
| `AGENTS.md` | 项目规约（**改代码前必读**） |
| `docs/superpowers/specs/2026-10-26-cinderfish-architecture-v0.md` | 架构设计 v0 |
| `docs/research/2026-10-26-misskey-fediverse-firefish.md` | 首轮调研 |
| `docs/research/2026-10-26-go-ecosystem-and-dependency-decisions.md` | Go 生态与依赖取舍 |
| `docs/research/2026-10-26-tech-stack-comparison.md` | 技术栈横向对比 |
| `docs/research/2026-10-26-misskey-source-calibration.md` | **源码校准记录（含许可证约束与踩坑清单）** |
| `docs/research/2026-10-26-m2-implementation-notes.md` | M2 实施记录 |
| `docs/research/2026-10-26-initial-scope.md` | 首版范围与已裁决项 |
| `docs/research/2026-10-28-m4-streaming-source-calibration.md` | **M4 Streaming 协议逐条对照上游源码** |
| `docs/research/2026-10-28-m5-federation-source-calibration.md` | **M5 联邦逐条对照上游源码** |
| `docs/research/2026-10-28-project-name-decision.md` | 命名决策记录 |
| `docs/research/2026-10-29-performance-baseline.md` | **性能自比基线（M7）** |
| `docs/DEPLOY.md` | **生产部署指南（反向代理、systemd、备份、媒体迁移）** |
| `deploy/` | systemd 单元、Caddyfile、生产配置模板 |
| `web/` | Vue 3 前端源码（构建到 `web/dist`） |
| `.archify/architecture-cinderfish-overall/` | 项目整体结构图（Archify 产物，随里程碑更新） |

## 14. 已知的未验证项

- **S3 后端未做真实服务端验证**：只验证了编译、接口与 URL 生成逻辑。
- **缩略图/WebP 生成未实现**：`drive_files.thumbnail_key` 字段已留，图片处理管道还没写。
- **前端没有真实第三方客户端联调**：页面已在构建产物上用浏览器走查，但没有用 misskey-js 或 Misskey Web 前端连过本实例。
- **Streaming 未做真实 Misskey 客户端联调**：已用自写 WebSocket 客户端验证（握手、`connect`、`connected`、笔记投递全部通过），但**没有**用 misskey-js 或 Misskey Web 前端实际连过。
- **联邦没有任何第三方联调**：只跑了本实现的双实例测试（真签名、真 HTTP、真库），没有对 Mastodon / GoToSocial / 真实 Misskey 实例互通。这是 M5 之后最该补的验证。
- **Streaming 多进程中继只手测了单连接的 LISTEN/NOTIFY 往返**，没有做真正的双进程联调。
