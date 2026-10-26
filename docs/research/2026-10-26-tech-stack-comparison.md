# 技术栈选型调研：语言与运行时横向对比

- 记录日期：2026-10-26
- 目的：回应"是否使用 Go 为商榷问题"的要求，广泛调研后给出带依据的推荐。
- 前提约束（来自用户已确认的产品要求）：个人/极少数用户实例；PostgreSQL 必需；Redis 可选（非强制）；媒体 local+S3 双实现；WebSocket Streaming API；ActivityPub 收发（HTTP Signatures）；单二进制/低依赖部署是加分项；性能表仅作可行性参考。

## 1. 选型判据（按本项目权重排序）

1. **Fediverse 领域先例**：有没有同形态（个人实例、联邦、API 服务器）的成功项目验证过这套技术。
2. **协议栈完备度**：WebSocket 长连接、HTTP Signatures（RSA）、S3 客户端、图像缩略图/WebP——这些是 Misskey 兼容 + AP 通信的硬需求。
3. **资源画像**：空载内存、长连接内存、二进制/部署形态。
4. **开发速度与可维护性**：本项目是重 CRUD + 协议胶水的大型单体应用，不是性能敏感的算法系统；单人可维护性权重高。
5. **前端协同**：API 类型契约、MFM 渲染等与 TS 前端的协同成本。

## 2. 候选路线与证据

### 2.1 Go

先例（判据 1 满分）：
- **GoToSocial**：Go 单二进制 AP 服务器，官方定位就是轻量个人实例；数据库仅 SQLite/PostgreSQL，无独立队列服务，后台任务进程内 worker（https://docs.gotosocial.org/en/latest/configuration/database ）。与本项目形态几乎完全重合。
- go-ap（Activity 词汇库，Gitea/Forgejo 联邦在用，持续维护，https://github.com/go-ap/activitypub ）；go-fed 已停摆不再考虑。
- 用户朋友的重构（性能表 33-55MB RSS、210ms 冷启动）证明该形态在 Misskey 兼容场景可达。

协议栈（判据 2 满分）：`gorilla/websocket`（事实标准）、`crypto/rsa` + `crypto/sha256` 原生覆盖 HTTP Signatures（go-fed/httpsig 亦可）、`aws-sdk-go-v2/s3` 或 `minio-go`、标准库 `image` + `golang.org/x/image` 覆盖缩略图与 WebP（需要引入少量第三方编码器）。

资源画像（判据 3 良）：GC 带来比 Rust 高一些的空载内存，但个人实例规模在几十 MB 量级；静态编译二进制约 15-45MB；启动毫秒级。

开发效率（判据 4 良）：静态类型 + 显式错误处理，无借用检查心智负担；goroutine 天然匹配 WebSocket fanout 与 worker 模型。

前端协同（判据 5 中）：类型契约需手写 TS types 或用代码生成维护。

### 2.2 Rust（axum + tokio）

先例（判据 1 良）：
- **Lemmy** 是最大的 Rust AP 项目，其抽出的 **activitypub-federation** crate 是高层框架，封装取送、收件、HTTP 签名（https://github.com/LemmyNet/activitypub-federation-rust ），文档完整、维护活跃。但 Lemmy 是论坛形态，与 Misskey 式 SNS 的笔记/时间线/反应模型差异大，能借的是协议层而非产品层。
- Rustodon 等同形态项目规模和成熟度都不足为凭。

协议栈（判据 2 良）：HTTP Signatures 有 `http-signature-normalization-reqwest`（https://lib.rs/cryptography 可查）；S3 用官方 aws-sdk-rust；图像处理 image/ ravif/ webp 库齐全但拼装成本高于 Go；WebSocket 走 axum 内建。

资源画像（判据 3 满分）：无 GC，内存最低；基准测试普遍显示同等负载下内存低于 Go（LogRocket 对比文 https://blog.logrocket.com/go-vs-rust-when-use-rust-when-use-go ）。

开发效率（判据 4 中下）：borrow checker + 异步生命周期对快速迭代 CRUD 单体不友好；编译时间长；单人项目"改动-验证"循环明显慢于 Go。

结论：性能天花板最高、资源最低，但开发效率代价对本项目（重 CRUD、协议胶水、单人）不划算。

### 2.3 C#/.NET（Iceshrimp.NET 路线）

先例（判据 1 良，且同源最强）：**Iceshrimp.NET 正是 Firefish 的 .NET 重写**，官方强调"resource-efficient backend"（https://iceshrimp.dev ），Hostinger 一键部署页也以低内存 VPS 为卖点（https://www.hostinger.com/applications/iceshrimp ）。它验证了"重写 Firefish 系 + .NET + 低资源"可行——但注意它实现的是 Mastodon API 兼容层（本项目已明确放弃该方向）。

协议栈（判据 2 满分）：SignalR/原生 WebSocket、System.Security Cryptography、AWSSDK.S3、ImageSharp（注意 ImageSharp 许可证为 Six Labors Split License，商用条款需阅读）。

资源画像（判据 3 良）：常规运行约几十至一百多 MB；.NET 8 起 Native AOT minimal API 可产单文件（https://learn.microsoft.com/en-us/aspnet/core/fundamentals/native-aot ），但 ASP.NET 生态中不少库不满足 trim/AOT 要求，全功能 API 服务器走 AOT 的现实成本高；不 AOT 则部署需要 .NET 运行时（自包含发布则二进制 60-100MB+）。

开发效率（判据 4 满分）：C# 语言表达力 + 生态成熟度是候选中最高的。

结论：合理路线，尤其适合团队已熟 .NET；但"单静态二进制 + 极简部署"目标下不如 Go 干净，且本项目与其先例的关键差异（放弃 Mastodon API）削弱了直接借鉴的价值。

### 2.4 TypeScript / Node.js

先例（判据 1 满分但方向相反）：Misskey 本体、Sharkey、Forkey 等 fork 全是 Node——本项目要解决的核心痛点（个人实例下 Node 栈的资源占用、上千 node_modules、官方标注构建需 2GB 内存）正是这个栈本身。选择它等于回到原点。

协议栈与生态（判据 2、5 满分）：misskey-js 类型契约可直接对齐、MFM 解析器现成、前端同语言。

资源画像（判据 3 差）：官方文档明写构建至少 2GB 内存、运行需 node_modules 上千包（首次调研已核，见 misskey-fediverse-firefish.md）。

结论：除非目标是"魔改 Misskey 本体"，否则与项目动机相悖。**排除**。

### 2.5 Elixir / BEAM（Pleroma/Akkoma 路线）

先例（判据 1 良）：Pleroma/Akkoma 正是"低资源个人实例 AP 服务器"，BEAM 的进程模型对 WebSocket fanout 极为优雅；fediverse.party 生态页也列出多支 Elixir AP 工具（https://fediverse.party/en/miscellaneous ）。

协议栈（判据 2 中）：HTTP 签名、WebSocket 均有库；图像处理生态明显弱于 Go/Rust/.NET。

资源画像（判据 3 良）：Pleroma 以低资源著称，但部署依赖 Erlang 运行时与 mix 构建链，"单二进制"目标做不到。

开发效率（判据 4 中）：函数式 + OTP 心智模型独特，单人维护的社区风险高（招不到接手者）。

结论：形态匹配但生态偏门；且用户明确"别参考 Pleroma 代码"，团队也需现学。**不推荐**。

### 2.6 JVM / Kotlin

单就空载内存与部署体积而言与本项目目标冲突（JVM 基础开销百 MB 级；GraalVM native-image 对反射重的框架成本高）。**排除**。

## 3. 对比矩阵

| 判据 | Go | Rust | .NET | Node/TS | Elixir |
|------|----|----|----|----|----|
| 领域先例（同形态） | ★★★ GoToSocial | ★★ Lemmy（形态不同） | ★★★ Iceshrimp.NET | ★★★ Misskey 本体（反向） | ★★ Pleroma（勿参考代码） |
| 协议栈完备度 | ★★★ | ★★☆ | ★★★ | ★★★ | ★★ |
| 资源画像 | ★★☆ | ★★★ | ★★☆ | ★ | ★★☆ |
| 单二进制/低依赖部署 | ★★★ | ★★★ | ★★ | ★ | ★ |
| 开发/单人维护效率 | ★★★ | ★★ | ★★★ | ★★★ | ★★ |
| 前端协同 | ★★ | ★★ | ★★ | ★★★ | ★★ |

## 4. 推荐

**推荐 Go**：在"同形态先例 + 单二进制部署 + 协议栈完备 + 单人可维护"四个本项目最重的判据上同时占优，资源画像略逊于 Rust 但在个人实例规模无实际差异。

**备选 Rust**：如果最终把资源占用放在第一位、接受更慢的迭代速度，axum + activitypub-federation 是唯一值得认真考虑的替代路线。

**语言定案状态**：待用户在 Go / Rust 之间拍板后写入架构文档生效。以下分叉影响架构文档中仅两处：实现语言与 AP 词汇库（go-ap vs activitypub-federation-rust），模块结构、数据模型、API 契约、联邦策略设计均与语言无关。

## 5. 附带结论：前端栈

用户要求前端贴近 Firefish。Firefish/Misskey 前端均为 **Vue 3**，MFM 渲染、misskey-js 客户端、组件生态都是 Vue 系。自建前端推荐 **Vue 3 + TypeScript + Vite**：可直接移植/参考 misskey-js 的 API 类型与 MFM 解析逻辑（MIT/AGPL 注意许可），并对齐 Firefish 的信息架构。React 路线无任何额外收益，除非用户有明确偏好。
