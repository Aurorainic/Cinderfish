# 首版研究约束与待确认范围

- 记录日期：2026-10-26
- 项目名：暂用 MilnFish，不是最终命名

## 已确认

- 目标用户：个人或极少数用户的实例。
- 产品取向：Firefish 风格体验，Misskey 兼容与 Fediverse 通信不能落后到不可用。
- Pleroma/Akkoma：仅定位对标，不参考代码。
- 注册：开放注册、邀请制、管理员创建三种模式都要保留并可切换。
- 联邦：公开互联、白名单、完全不互联三种模式都要保留并可切换。
- 媒体：local 与 S3 兼容存储都必须支持。
- 性能参考：朋友未公开 Go 重构的实测数据用于可行性参考，不直接作为本项目验收承诺。
- 网络：访问 GitHub 使用 `127.0.0.1:19823` 代理。

## 首版建议边界

首版交付应是可部署的个人实例，而不是完整 Misskey 重写：

- Go 单体服务，明确分离 API、账户、笔记、媒体、联邦、后台任务和实时事件边界；
- PostgreSQL 作为持久化基础；Redis 是否强制依赖待实现阶段结合队列/流式广播需求决定；
- 自建 Firefish 风格前端，优先覆盖登录、注册、时间线、发帖、回复、renote、反应、媒体、通知、个人资料和管理设置；
- Misskey 原生 API 首批覆盖 `meta`、认证、`i`、notes、timelines、users、following、drive、notifications 和 streaming；
- ActivityPub 首批覆盖 WebFinger、Actor、Inbox/Outbox、Follow/Accept/Undo、Create Note、Like、Announce、Delete、Update、HTTP Signatures；
- 联邦策略贯穿发现、收取、发送、远程媒体和对象解析；
- 存储接口同时实现本地和 S3 适配器；
- 建立 API contract test、ActivityPub fixture test、真实双实例联邦 smoke test 和资源基准测试。

## 非目标

- 首版不追求完整 Misskey 全端点覆盖。
- 首版不直接复用 Firefish 私有后端实现。
- 首版不以 Pleroma/Akkoma 代码或数据模型为设计基础。
- 首版不承诺朋友 Go 重构表格中的所有性能数值。
- 首版不默认实现完整 Mastodon API；若需要，应单独列为兼容层范围。

## 仍需确认的关键决定

1. 首版前端是否必须直接运行现有 Firefish 构建产物，还是允许新建 Firefish 风格前端。
2. 是否把 Mastodon API/OAuth 兼容列入首版，而不只是 Misskey API/MiAuth。
3. PostgreSQL 与 Redis 是否都必须作为正式部署依赖，还是允许单机低依赖模式。
4. 首版是否需要迁移朋友私有 Go 重构的数据，还是从空实例开始。
5. 是否需要 relay 支持；基础 ActivityPub 互联与 relay 是两个不同的工作量。

## 研究判断

Firefish 已停止维护，直接复用其前端会把项目绑定到过时的 API 与构建链；更稳妥的路线是提取其交互和视觉方向，自建前端，以 Misskey 当前 API 契约为稳定后端边界，并用兼容测试持续约束行为。
