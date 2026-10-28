# 公开资料调研记录

- 调研日期：2026-10-26
- 项目名：Cinderfish（记录时为暂用名）
- 范围：Misskey API、Misskey Streaming API、ActivityPub/WebFinger、Firefish 前端与维护状态
- 排除项：Pleroma/Akkoma 仅作为产品定位参照，不作为代码或架构参考

## 结论摘要

1. 首版应以 Misskey 原生 API 为主要客户端兼容面，而不是先以 Mastodon API 替代。Misskey 官方明确说明其 API 不遵循 REST 习惯，调用形式是 `POST /api/<endpoint>`，并提供独立的 Streaming API。
2. Misskey 客户端兼容至少需要 token 认证、`meta`、用户、笔记、媒体、时间线、通知、反应、关注和 WebSocket channel；时间线接口的 `sinceId`/`untilId` 语义与返回对象结构必须稳定。
3. Streaming API 不是普通的“事件 WebSocket”：连接后还要通过 `connect` 加入 `main`、`homeTimeline`、`localTimeline`、`hybridTimeline`、`globalTimeline` 等 channel，并支持 `subNote`/`unsubNote`。
4. ActivityPub 互操作的基础不是某个 Misskey 私有接口，而是 WebFinger、Actor、Inbox/Outbox、ActivityStreams JSON、HTTP Signatures 和可重试的投递处理。
5. Firefish 的 UI/UX、移动端体验、线程、通知、Mastodon 客户端兼容等方向有参考价值，但 Firefish 已停止维护，不能直接作为新项目的长期上游。应参考其可见交互和信息架构，避免复制其过时的后端耦合。
6. Firefish 风格前端与 Misskey 后端并非天然完全兼容。Firefish/各 fork 会增加或改变 API 行为，且外部客户端对字符串 ID、Mastodon API 和 instance API 的假设不同，因此必须建立兼容性测试矩阵。

## Misskey API

官方资料：

- https://misskey-hub.net/en/docs/for-developers/api
- https://misskey-hub.net/en/docs/for-developers/api/endpoints
- https://misskey-hub.net/en/docs/for-developers/api/token
- https://misskey-hub.net/en/docs/for-developers/api/streaming
- https://github.com/misskey-dev/misskey/blob/develop/packages/misskey-js/README.md
- https://github.com/misskey-dev/misskey/blob/develop/packages/backend/src/server/api/endpoints/notes/create.ts

### 认证

官方支持 MiAuth 和 OAuth 2.0；旧的应用创建 token 方式仍属于历史兼容面。API token 可通过请求体的 `i` 参数使用，官方资料也描述了 Authorization header 用法。首版应至少实现：

- 安全的 token 生成、哈希存储、撤销和权限范围；
- `/api/meta` 无登录访问；
- MiAuth 流程，便于 Misskey 客户端登录；
- 明确的 API 错误结构与权限检查；
- 速率限制、请求体大小限制和媒体上传限制。

### 核心接口分组

首版优先级建议：

- 服务器发现：`meta`；
- 当前用户：`i`、用户设置相关最小接口；
- 笔记：`notes/create`、`notes/show`、`notes/delete`、`notes/reactions/create`、`notes/reactions/delete`、回复和 renote；
- 时间线：`notes/timeline`、`notes/local-timeline`、`notes/hybrid-timeline`、`notes/global-timeline`；
- 账户关系：`users/show`、`users/search-by-username-and-host`、`following/create`、`following/delete`、关系查询；
- 媒体：`drive/files/create`、`drive/files/show`、媒体列表和删除；
- 通知：`i/notifications`、已读状态；
- 搜索：先做用户/笔记/标签的必要子集；
- 注册/登录：这是本项目自己的管理能力，但结果必须能被 Misskey 风格前端使用。

`notes/create` 的公开类型定义包含 visibility、CW、localOnly、replyId、renoteId、channelId、text、fileIds 和 poll；时间线使用 `limit`、`sinceId`、`untilId` 等游标参数。不要把 API 简化成仅有文本字段的微博接口。

### Streaming

连接格式：`wss://<host>/streaming?i=<token>`。连接后通过 JSON 消息：

```json
{
  "type": "connect",
  "body": {
    "channel": "homeTimeline",
    "id": "client-generated-id",
    "params": {}
  }
}
```

服务器推送消息包含 channel connection id，客户端依靠它区分同一 channel 的不同订阅。首版至少覆盖 `main`、`homeTimeline`、`localTimeline`，并保留 `hybridTimeline`、`globalTimeline` 的接口位置。官方实现还限制每连接 channel 数量，并对 note capture 做资源限制；这些限制应在 Go 服务中明确配置，避免长连接成为内存攻击面。

## ActivityPub 与 WebFinger

主要资料：

- https://www.w3.org/TR/activitypub/
- https://www.w3.org/community/reports/socialcg/CG-FINAL-apwf-20240608
- https://www.w3.org/community/reports/socialcg/CG-FINAL-apsig-20240607
- https://datatracker.ietf.org/doc/html/rfc7033
- https://misskey-hub.net/ns
- https://misskey-hub.net/en/docs/for-users/onboarding/get-started

### 首版互操作面

本地用户需要可被 `acct:user@example.org` 发现：

- `/.well-known/webfinger?resource=acct:user@example.org`；
- Actor JSON，支持 `application/activity+json` 与常见 `application/ld+json` 请求；
- Actor 的 `inbox`、`outbox`、`followers`、`following` 和公开密钥；
- `GET` 公开对象时的内容类型与稳定 HTTPS ID；
- 接收入站活动的签名验证；
- 出站活动按收件人去重并可重试。

首批活动类型：

- `Follow` / `Accept` / `Undo`；
- `Create` + `Note`；
- `Like`；
- `Announce`，对应 renote/boost；
- `Delete`；
- `Update`，至少用于资料与笔记编辑策略；
- `Move`、投票、复杂群组和 relay 行为可后置，但数据模型要避免封死。

ActivityPub 规范要求 inbox 投递使用 ActivityStreams 活动、服务端需要处理重复活动；实际 Fediverse 还依赖 HTTP Signature 进行请求认证。签名验证不能只校验 JSON 内容，必须校验请求方法、目标和参与签名的 headers，并处理远程 key 获取、缓存、轮换和失败重试。

### 三种联邦模式

这是本项目的产品要求，应在统一的 federation policy 层实现：

- `public`：允许正常发现、入站和出站联邦；
- `allowlist`：仅允许配置的实例域名参与入站/出站，默认拒绝；
- `none`：不主动发送联邦活动，也不接受远程活动，但本地 ActivityPub 端点应返回清晰状态，避免误导远端。

策略判断必须覆盖 WebFinger、Actor 获取、Inbox、出站投递、媒体远程拉取和搜索/对象解析，不应只在发送队列处拦截。

## Firefish 与前端边界

资料：

- https://github.com/jclaudan/firefish
- https://github.com/YunoHost-Apps/firefish_ynh
- https://misskey-hub.net/en/docs/for-users/features
- https://misskey-hub.net/en/docs/for-users/features/timeline
- https://misskey-hub.net/en/docs/for-users/features/widgets
- https://misskey-hub.net/docs/for-users/features/mfm

Firefish 公开介绍的可借鉴方向包括移动端 UI/UX、线程、通知、编辑、可访问性、搜索、Groups 和 Mastodon 客户端兼容。Misskey 自身还提供 Deck、多列布局、Widgets、MFM、实时 timeline 和 Drive，这些是 Firefish 风格体验的重要组成部分。

但公开镜像/打包资料显示 Firefish 已在 2024 年底结束生命周期，相关仓库已归档或不可作为持续上游。Firefish 的安装依赖 Node.js、PostgreSQL、Redis、构建链和大量前端运行时；这与本项目的 Go 低资源目标冲突。因此建议：

- UI 参考 Firefish 的布局、密度、移动端导航、线程和通知体验；
- API 以 Misskey 当前公开契约为准；
- 前端自建，并把 API client、MFM renderer、流式连接做成独立层；
- 不直接绑定 Firefish 的私有接口和旧数据模型；
- 对 Firefish/当前 Misskey/主流客户端分别做兼容测试。

Misskey 官方明确提示其并不兼容 Mastodon API，仅有部分例外。因此“相关 API”首版至少应明确包含 Misskey 原生 API；是否额外实现 Mastodon API 兼容层，需要作为单独范围决定，不能假设自然获得。

## 兼容性风险

- Misskey API 是 endpoint RPC 风格，不是 REST；错误地采用 REST 资源模型会导致前端和 SDK 不兼容。
- Misskey/fork 的 ID 是字符串；外部客户端可能错误地按整数解析，测试必须覆盖长字符串 ID。
- Firefish、Sharkey 和其他 fork 的 API 差异会影响直接复用前端。
- ActivityPub 的“规范正确”不等于与 Misskey/Mastodon 实例互通；需要真实双端测试。
- 远程活动是异步、可重复、乱序和可失败的，不能在 HTTP 请求中假设一次处理完成。
- `localOnly`、`followers`、`specified` 等可见性必须同时影响本地时间线、流式事件和联邦收件人。
- 媒体 URL、远程媒体缓存、S3 ACL/签名 URL、删除传播和存储迁移需要统一抽象。
