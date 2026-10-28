# Cinderfish 架构设计 v0（首版范围）

- 日期：2026-10-26
- 状态：待用户批准后生效；生效前不写产品代码
- 名称：Cinderfish（2026-10-28 定名；本文写作时为暂用名）

## 1. 定位一句话

Go 单二进制的个人/极小规模联邦社交实例：Misskey 兼容 API 为后端契约，ActivityPub 联邦为通信层，Firefish 风格（Vue 3）为自建前端的体验基线。

## 2. 已定案的核心决策

以下决定基于 `docs/research/` 研究记录，视本设计为研究结论的收敛：

| 决策 | 定案 | 一句话理由 |
|------|------|-----------|
| D1 运行形态 | 单二进制单体，进程内模块边界 | 朋友项目实测可行性 + GoToSocial/Iceshrimp.NET 先例 |
| D2 数据库 | PostgreSQL 唯一强制持久化依赖；**Redis 可选**（配置启用时接管广播/队列加速，缺省关闭时由 PostgreSQL 承载，行为一致） | 队列用 `SKIP LOCKED` 表，广播用 `LISTEN/NOTIFY`，个人实例规模足够；Redis 仅作规模化预留 |
| D3 前端 | 自建 Firefish 风格前端（**Vue 3 + TypeScript + Vite**），后端服务静态产物 | Firefish 已 EOL，构建链过时不可依赖；Vue 3 与 Misskey/Firefish 前端生态同源，可复用 misskey-js 类型与 MFM 逻辑 |
| D4 API 契约 | Misskey 原生 API（POST RPC 风格）+ MiAuth + OAuth 2.0 PKCE；Streaming API over WebSocket | 主流生态与客户端都对着这套契约 |
| D5 Mastodon API | **明确放弃**，不做、也不预留路由空位 | 用户裁决：个人实例以 Misskey 契约为唯一兼容面，避免双契约维护成本 |
| D6 数据迁移 | 不迁移朋友私有项目数据，空实例起步 | 该项目未发布、无源码可访问 |
| D7 Relay | 首版不做，列入后续路线 | 与白名单模式语义需先定义清楚 |
| D8 技术参考 | 只参考协议与 API 契约，不抄任何项目代码；Pleroma/Akkoma 代码明确排除 | 用户明确要求 |
| D9 实现语言 | **Go（已定案，2026-10-26 用户确认）**；AP 词汇库用 go-ap，HTTP Signatures 用 go-fed/httpsig 或自实现 | 调研见 `2026-10-26-tech-stack-comparison.md`：同形态先例（GoToSocial）、单二进制、协议栈完备、单人可维护 |

## 3. 模块结构

单体进程内分层，包边界即接口边界（包名为语言无关的逻辑单元，按 D9 定案后映射为 Go package 或 Rust module/crate）：

```
server            入口：配置加载、优雅关闭
config            TOML/环境变量配置（端口、域名、注册模式、联邦模式、存储后端、Redis 可选开关）
db                PostgreSQL 连接池、迁移（嵌入式 SQL 迁移文件）
httpapi           HTTP 路由装配：/api/*（Misskey RPC）、/.well-known/*、/users/*、/notes/*、/inbox、/outbox、/media/*
auth              会话、MiAuth、OAuth 2.0 PKCE、token 签发与校验、HTTP Signature 验证
account           用户、资料、关系（关注/静音/屏蔽）、注册模式实现
note              笔记 CRUD、可见性四级 + localOnly、CW、投票、引用、renote、反应
timeline          home/local/social(global) 查询与物化、游标分页
notification      通知生成与已读
drive             文件元数据、缩略图、WebP 生成
media             存储接口 Storage：local 实现 + S3 兼容实现
federation        WebFinger、Actor 文档、Inbox 处理、Outbox、签名收发、远程对象解析、三种联邦模式策略
deliver           出站投递 worker（Postgres 表队列 + SKIP LOCKED，指数退避重试）
streaming         WebSocket 端点、频道（main/homeTimeline/localTimeline/hybridTimeline/globalTimeline）、订阅管理
jobs              后台任务框架（同 deliver 的队列机制）：媒体处理、统计刷新
web               前端静态产物挂载与 SPA fallback
queue             队列/广播抽象：缺省 PostgreSQL 实现（SKIP LOCKED + LISTEN/NOTIFY）；Redis 可选实现（配置启用时替换，语义一致：持久表兜底 + Redis Pub/Sub 加速唤醒）
```

依赖方向单向：`httpapi → 各领域包 → db/media(接口)/queue(接口)`；`federation` 与 `note/account` 互相只通过接口交互，避免环。Redis 属于 `queue` 的可替换实现，业务层不感知。

## 4. 数据模型骨架（首批表）

- `users`：id(text, Misskey 风格字符串)、username、host(nullable, 本地用户为 NULL)、password_hash、is_admin、created_at
- `user_profiles`：资料、设置
- `followings` / `follow_requests` / `blocks` / `mutes`
- `notes`：id、user_id、visibility(public/home/followers/specified)、local_only、cw、text、poll(jsonb)、reply_id、renote_id、renote_user_id、uri(远程唯一标识)、tags/mentions/emojis(jsonb)
- `note_visibilities`（specified 的可见用户子表）
- `note_reactions`：emoji、user_id
- `drive_files`：元数据、hash、存储 key、缩略图 key、S3/local 标记
- `follow_activities` / 出入站活动日志（联邦审计）
- `deliver_queue`：payload jsonb、inbox_url、attempts、next_attempt_at、last_error
- `events`：流式广播事件表（streaming 消费）
- `access_tokens` / `miauth_sessions` / `oauth_flows`
- `instance_meta`：单行配置（注册模式、联邦模式、统计缓存）
- `registrations_invites`：邀请码
- `antenna/channel/list`（首版如裁剪，见 §7）

ID 生成：Misskey 风格字符串 ID（时间戳前缀 + 随机），保证字符串类型与游标排序一致。

## 5. API 首批端点集

### 5.1 Misskey RPC（`POST /api/<endpoint>`）

- 认证：`miauth/<session>` 流程、`oauth/token` 签发；请求带 `i` 参数/header
- 基础：`meta`、`stats`、`ping`、`server-info`
- 账户：`i`、`i/update`、`users/show`、`users/search`、`username/available`、`signup`（受注册模式约束）、`signin`/`signin-flow`、`signout`
- 关系：`following/create`、`following/delete`、`following/requests/*`、`blocking/*`、`mute/*`
- 笔记：`notes/create`（visibility 四级 + localOnly + CW + poll + fileIds + reply/renote）、`notes/delete`、`notes/show`、`notes/reactions/create`、`notes/reactions/delete`、`notes/children`、`notes/replies`、`notes/conversation`、`notes/state`、`notes/favorites`
- 时间线：`notes/local-timeline`、`notes/hybrid-timeline`、`notes/global-timeline`、`notes/user-list-timeline`（如列表首版保留）、`users/notes`、`notes/mentions`
- 通知：`notifications/create`(测试用)、`notifications`、`notifications/mark-all-as-read`
- Drive：`drive`、`drive/files/create`、`drive/files`、`drive/files/show`、`drive/files/update`、`drive/files/delete`、`drive/folders/*`
- 管理：`admin/meta`、`admin/accounts/create`、`admin/invite/*`、`admin/federation/*`（实例管理）
- Streaming：`streaming`（WS，频道 connect/sub/unsub 协议）

以上是"个人实例日常可用"的最小集；未列端点返回 Misskey 风格错误（`{"error":{"code":"...","message":"..."}}`），不做静默 404。

### 5.2 ActivityPub / Fediverse

- `GET /.well-known/webfinger`、`/.well-known/nodeinfo`
- `GET /users/<username>`（Actor）、`/users/<username>/outbox`、`/users/<username>/collections/followers|following`
- `POST /inbox`（共享收件箱）与 `POST /users/<username>/inbox`
- 出站：Follow/Accept/Undo(Follow)、Create(Note)/Delete/Update、Like/Undo(Like)、Announce/Undo(Announce)、Follow 拒绝
- 签名：HTTP Signatures（RSA-PSS-SHA256 与 rsa-sha256 双验签），密钥在用户创建时生成，PEM 存库
- 扩展：`_misskey_quote`、`_misskey_content`、`_misskey_reaction` 等 `_misskey_` 前缀属性，兼容接收 Mastodon `replies`/`to`/`cc` 语义
- 媒体：Image/Document 附件用 `/media/<key>` 本地路径或 S3 公共 URL，remote 媒体可配置代理/缓存

### 5.3 联邦三种模式（贯穿所有环节）

| 模式 | WebFinger 解析 | 入站 Inbox | 出站投递 | 远程媒体 |
|------|---------------|-----------|----------|---------|
| 公开互联 | 允许 | 接受（可配黑名单） | 允许 | 正常加载 |
| 白名单 | 仅白名单域 | 仅白名单域 | 仅白名单域 | 仅白名单域 |
| 完全不互联 | 拒绝远程 | 410/403 | 队列暂停并标记 | 不加载/不代理 |

实现要点：模式判断集中在 `federation.Policy` 接口；入站在 WebFinger、Actor GET、Inbox POST 三处各设闸门；出站在 deliver worker 出队后、投递前过滤。模式切换运行时生效（读 `instance_meta`，不重启）。

### 5.4 不做 Mastodon API

`/api/v1/*` 等 Mastodon 兼容路由不存在，也不预留。返回 Misskey 风格错误结构（`{"error":{"code":"...","message":"..."}}`）的未匹配端点处理就是唯一行为。OAuth 2.0 仅服务 Misskey 自身的 PKCE 流程。

## 6. 媒体存储

```text
Storage 接口（语言无关语义）:
    Put(ctx, key, reader, size, contentType) error
    Get(ctx, key) (ReadCloser, error)
    Delete(ctx, key) error
    PublicURL(key) string   // local: /media/<key>；S3: 端点+bucket+key 或预签名
```

- local 实现：内容寻址目录（`data/files/xx/yy/key`），进程内直接 serve
- S3 实现：S3 兼容 API（MinIO/R2/Backblaze B2/AWS 均可），支持自定义 endpoint；对象公开读或预签名 GET 二选一（配置项）；客户端按 D9 语言选 aws-sdk-go-v2/s3、minio-go 或 aws-sdk-rust
- 上传管道：大小限制 → MIME 嗅探 → 哈希去重 → 图像缩略图/WebP → Storage.Put
- 每个文件记录存储后端标记，支持实例级迁移工具（后续）

## 7. 首版裁剪（明确不做但记录）

- Mastodon API：明确放弃（D5），非裁剪而是路线排除
- 列表(user-list)、antenna、channel：数据模型留位，端点首版不实现
- 聊天/消息、画廊、页、游戏：不做
- 自定义 emoji 管理：本地 emoji 上传做，远程 emoji 采集首版仅存储不聚合
- 角色(role)系统、成就、flash：不做
- 搜索：仅本地基础全文（PostgreSQL 全文索引），不做 meilisearch 集成
- OAuth 2.0 只做 authorization code + PKCE（Misskey 官方流程），不做 client credentials 后台服务集成
- 对象存储不做桶自动创建；要求管理员预建桶

## 8. 质量与验证

- 单体跑 `go vet` / `go test`；API 契约测试以 Misskey 官方文档+真实响应 fixture 为基准
- ActivityPub：fixture 驱动的入站测试（Mastodon/Misskey 真实 payload 样例）+ 双实例 smoke test（本实例 ↔ 常见远端）
- 联邦模式测试矩阵：3 模式 × (WebFinger/Inbox/出站/媒体)
- 前端：TypeScript 严格模式，与后端类型契约由手写 types 包约束（Vue 3 生态，可参考 misskey-js 类型定义）
- 基准只做自比（记录每次版本的性能变化），不承诺朋友表格数值

## 9. 里程碑

1. **M1 骨架**：仓库结构（Go module）、config、db 迁移、`meta`/`ping`、单测跑通 ✅
2. **M2 账户与认证**：注册三模式、signin、MiAuth、token、`i`、users/show、OAuth 2.0 PKCE ✅
3. **M3 笔记与时间线**：notes/create/show/delete、四时间线、反应、通知、Drive+Storage 双实现
4. **M4 Streaming**：WS 端点、频道、事件广播（Postgres 事件表 + LISTEN/NOTIFY；Redis 可选实现） ✅
5. **M5 联邦**：WebFinger/Actor/Inbox/Outbox/签名、三模式策略、deliver 队列、双实例 smoke ✅
6. **M6 前端 Firefish 风格**（下一步）：Vue 3 时间线/发帖/反应/通知/资料/管理界面
7. **M7 打磨**：媒体 S3 迁移工具、admin 面板、性能自测、部署文档

里程碑顺序刻意把前端放后：API 稳定后前端才能对着契约开发，避免联调反复。

## 10. 风险登记

- Misskey API 无正式 schema，行为以官方文档+抓包 fixture 双源校准；上游破坏性变更需跟进
- HTTP Signatures 草案各家实现有差异，需用 Mastodon + Misskey + GoToSocial 三方实测
- LISTEN/NOTIFY 在连接闪断时会丢唤醒（不丢数据，靠表轮询兜底），需实现断线重订阅；Redis 模式下由 Redis Pub/Sub 替代唤醒通道，持久表兜底逻辑不变
- 自建前端工作量大，M6 前先以 API 完成度为先；可临时用现成 Misskey Web 客户端连本实例做冒烟（依赖端点兼容度）
