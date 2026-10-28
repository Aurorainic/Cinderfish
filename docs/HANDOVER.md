# MilnFish 交接说明

- 最后更新：2026-10-28
- 项目暂用名 **MilnFish**（非最终命名，命名待定）
- 仓库：`/home/lainsaka/文档/MilnFish`，分支 `master`，工作区干净

## 1. 一句话说明

Go 单二进制的个人/极小规模联邦社交实例。后端契约是 **Misskey 原生 API**，通信层是 **ActivityPub**，前端是自建的 **Firefish 风格 Vue 3** 应用。

## 2. 已完成

| 里程碑 | 状态 | 提交 |
|--------|------|------|
| M1 骨架 | ✅ | `883874c` |
| 源码校准（meta/ping/错误信封） | ✅ | `0ba974c` |
| M2 账户与认证 | ✅ | `1a1abed` |
| M3 笔记/时间线/反应/通知/存储 | ✅ | `f4b8508` |
| M4 Streaming（WebSocket） | ⬜ 未开始 | — |
| M5 联邦（ActivityPub） | ⬜ 未开始 | — |
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
internal/httpapi/         路由装配与全部端点处理器
```

依赖方向单向：`httpapi → 领域包 → db/media(接口)`。

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
```

## 9. 下一步：M4 Streaming API

按架构文档 M4 的范围：

1. `GET /streaming?i=<token>` WebSocket 端点
2. 频道：`main`、`homeTimeline`、`localTimeline`、`hybridTimeline`、`globalTimeline`
3. `connect` / `disconnect` 消息协议（上游格式：`{"type":"connect","body":{"channel":"...","id":"...","params":{}}}`）
4. `subNote` / `unsubNote`
5. 事件广播：PostgreSQL 事件表（持久）+ `LISTEN/NOTIFY`（唤醒）+ 进程内 fanout；Redis 可选时替换唤醒通道
6. 事件类型：`note`、`reaction`、`follow`、`notification` 等

**先读源码**：`packages/backend/src/server/api/stream/` 目录（`channels.ts`、`channel.ts`、`streaming-api.ts`、`channels/` 下各频道实现），以及 `packages/backend/src/server/api/stream/../../ws/` 下的 WebSocket 处理。频道列表在 `packages/backend/src/server/api/stream/channels.ts`。

M4 之后是 M5 联邦（WebFinger、Actor、Inbox/Outbox、HTTP Signatures、三模式策略、deliver 队列、双实例 smoke test），再是 M6 前端。

## 10. 文档索引

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

## 11. 已知的未验证项

- **S3 后端未做真实服务端验证**：只验证了编译、接口与 URL 生成逻辑。需要一个 MinIO 或 R2 实例做端到端验证（MinIO 二进制因代理拦截未下载成功，可试 GitHub releases 或其他镜像）。
- **缩略图/WebP 生成未实现**：`drive_files.thumbnail_key` 字段已留，但图片处理管道还没写。
- **联邦模式在代码里只体现了 `meta` 字段的映射**，六个拦截点尚未实现（属于 M5）。
- **前端完全未开始**（M6）。
- **Streaming 完全未开始**（M4）。
