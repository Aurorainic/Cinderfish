# 补充调研：Go 生态、依赖取舍与先例项目

- 记录日期：2026-10-26
- 目的：为 initial-scope.md 中五个待确认决定提供证据，重点是"单机低依赖是否成立"与"Go 侧 ActivityPub 库选型"。

## 1. Go 侧 ActivityPub 库现状

- go-fed/activity（生成式、完整 AP 库）已基本停摆，社区评价其为"只是一个库，目前没有东西直接用它"（Hacker News 讨论，https://news.ycombinator.com/item?id=33735101）。
- 现役方案是 **go-ap**（https://github.com/go-ap/activitypub ，MIT，1173+ commits，持续维护）：Activity Vocabulary 的 Go 类型 + 序列化，是 GoActivityPub 套件的一部分。
- go-ap 被 Forgejo/Gitea 的联邦功能选用（forgejo issue #59，https://codeberg.org/forgejo/forgejo/issues/59），可作为生产可用性的旁证。
- HTTP Signatures 可用 go-fed/httpsig（独立小库，无重依赖），或按 draft-cavage-http-signatures 自实现（工作量不大，Misskey 自身也是手写序列化）。

结论：词汇与序列化可借 go-ap；收件分发、签名验证等核心逻辑自建。Misskey 本身不使用生成式 AP 库，手写 payload 反而更贴近 `_misskey_` 扩展属性的处理方式。

## 2. 先例项目：Iceshrimp.NET 与 GoToSocial

### Iceshrimp.NET（Firefish 的 C# 重写继任者）

- 全新代码库重写，官方目标："performance, stability and maintainability"，反目标是"flashy marketing"（https://iceshrimp.dev ）。
- 关键启示：Firefish 原版（Node/TypeScript）已进入仅安全补丁状态，开发力量全部转向重写；重写版实现了完整 Mastodon API 兼容层，使大量 Mastodon 客户端可用（https://thefedi.wiki/iceshrimp/home ）。
- 对本项目启示：①"旧壳新核"是 Firefish 系社区的既定路线；②Mastodon API 兼容层是真实需求但工程量大，单独成层。

### GoToSocial（Go 语言个人实例先例）

- Go 单二进制，数据库仅 SQLite 或 PostgreSQL（https://docs.gotosocial.org/en/latest/configuration/database ），**没有 Redis 或任何独立队列服务**；后台任务用内置 worker。
- 官方推荐小型实例直接用 SQLite；发行版覆盖 Linux/FreeBSD 多架构，单板机可跑。
- 对本项目启示：Go 单二进制 + 单数据库 + 进程内 worker 承载联邦收发，在个人实例规模是被验证过的形态。

## 3. 用 PostgreSQL 替代 Redis 的可行性

- 队列：`SELECT ... FOR UPDATE SKIP LOCKED` 是 PostgreSQL 官方文档记载的队列用法；事务表 + 轮询 worker 在中等负载单节点下是成熟模式（Redisson 对比文 https://redisson.pro/blog/redis-vs-postgres-when-you-need-both.html ；DEV 社区综述 https://dev.to/philip_mcclarence_2ef9475/postgres-vs-redis-mongo-and-rabbitmq-do-you-really-need-all-four-5382 ）。
- 事件广播：`LISTEN/NOTIFY` 提供事务提交后的 pub/sub；限制是每个监听连接独占一个后端进程、监听方断线期间消息丢失（Dimitri Fontaine 的分析 https://tapoueh.org/blog/2018/07/postgresql-listen/notify ）。
- 适用边界：消息丢失不可接受的场景需要配持久队列表兜底——对本项目即"NOTIFY 只做唤醒信号，状态以数据库表为准"。
- 规模判断：个人实例（≤10 用户、≤几百长连接）远在 LISTEN/NOTIFY 舒适区内；HN 讨论（https://news.ycombinator.com/item?id=44490510 ）也确认小数据量下该方案足够。

结论：**首版零 Redis**。流式广播 = 数据库事件表（持久）+ LISTEN/NOTIFY（实时唤醒）+ WebSocket 进程内 fanout；投递队列 = Postgres 表 + SKIP LOCKED。接口抽象保留未来接入 Redis 的位置。

## 4. Misskey API 细节补证

- 认证：MiAuth 与 OAuth 2.0（PKCE，code_verifier 43–128 字符、SHA256+base64url challenge）双轨并行，官方文档 https://misskey-hub.net/en/docs/for-developers/api/token/oauth ；另有 legacy app 方式。首版实现 MiAuth + OAuth2 PKCE。
- 可见性：public/home/followers/specified 四级 + localOnly 开关（https://misskey-hub.net/en/docs/for-users/features/note ）；时间线 home/local/social/global 语义见 https://misskey-hub.net/en/docs/for-users/features/timeline 。Home 可见性的时间线行为（不进他人 LTL/GTL）是时间线查询的核心规则。
- Firefish 与 Misskey 的枚举差异（antenna 源类型、notification 类型等）在迁移分析文中有完整对照（https://blog.outv.im/2024/firefish-to-misskey ），可作为 API 契约测试的输入。

## 5. 五个未定项的裁决建议

| # | 待定项 | 建议 | 依据 |
|---|--------|------|------|
| 1 | 前端路线 | 自建 Firefish 风格前端（已倾向，正式定案） | Firefish EOL、构建链过时；Iceshrimp.NET 同样重写前端 |
| 2 | Mastodon API 兼容 | 首版不做；OAuth 2.0 (PKCE) 仍实现（Misskey 官方认证方式之一），兼容层留接口边界 | Sharkey/Iceshrimp.NET 均视其为独立大工程；个人实例优先 Misskey 原生 API |
| 3 | Redis | 不强制，首版零 Redis；PostgreSQL 承载队列与广播 | GoToSocial 先例 + §3 分析 |
| 4 | 朋友项目数据迁移 | 不做，从空实例开始；其项目未发布、无源码可访问 | 用户聊天记录明确"还没公布" |
| 5 | ActivityPub relay | 首版不做；三种联邦模式已覆盖个人实例场景；relay 列入后续路线 | relay 是独立工作量，与白名单模式有语义冲突需先定义 |
