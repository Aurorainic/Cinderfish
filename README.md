# Cinderfish

**v0.1.2** — 单二进制的个人 / 极小规模联邦社交实例。后端按 Misskey 的 API 契约实现，前端是
Firefish 客户端结构的整体移植（Vue 3 + TypeScript）。

- **API 契约**：Misskey 原生 RPC（`POST /api/<endpoint>`）+ Streaming WebSocket
- **联邦**：ActivityPub（WebFinger / Actor / Inbox / Outbox / HTTP Signatures），
  `public` / `allowlist` / `none` 三种模式，可在运行时切换
- **前端**：Firefish 客户端结构的整站移植，界面风格与交互密度对齐 Firefish，不引入第三方 UI 库
- **依赖**：PostgreSQL 必需；Redis 可选（跨进程 Streaming 中继与投递唤醒）；媒体支持本地目录与
  S3 兼容对象存储（AWS S3、Cloudflare R2、Backblaze B2、MinIO）
- **许可**：AGPL-3.0-only

## 已实现的能力

| 领域 | 内容 |
|------|------|
| 账户 | 开放 / 邀请码 / 管理员建号三种注册模式、登录、MiAuth、OAuth 2.0 + PKCE、TOTP 两步验证、邮箱验证、密码重置、会话与访问令牌管理、账号迁移导出 |
| 笔记 | 发布 / 编辑 / 删除、内容警告、投票、附件、可见性（公开 / 首页 / 关注者 / 指定）、引用与转贴、表情回应（含自定义 emoji）、书签、置顶、编辑历史、按正文与用户搜索 |
| 时间线 | 首页、本地、全局、社交、标签、用户页与详情页；流式新帖提示与实时插入 |
| 通知 | 类型筛选、同类通知合并、批量已读、未读徽标，经 Streaming 实时送达 |
| 网盘 | 本地上传与 S3 兼容后端、文件夹、附件反查笔记、浏览器端压缩、从 URL 上传、容量配额 |
| Streaming | WebSocket 多频道（`main` / `homeTimeline` / `localTimeline` / `hybridTimeline` / `globalTimeline` / `note` / `drive` / `notifications` 等）；进程内 fanout，可选 Redis Pub/Sub 跨进程中继 |
| 管理台 | 概览、用户、文件、举报（含备注与 ActivityPub `Flag` 转发）、公告、联邦模式与白名单、投递队列、emoji、标签、数据库、邮件、安全、对象存储、实例屏蔽、代理账号 |
| 联邦 | WebFinger、Actor、Inbox 验签、Outbox、HTTP Signatures（含 `hs2019` 与 digest）、投递队列与重试、远端媒体代理、实例信息与用户目录 |

能力的完整台账（端点级 已实现 / 缺失）在
[`docs/research/2026-11-04-misskey-api-coverage.md`](docs/research/2026-11-04-misskey-api-coverage.md)，
页面级对照在 [`docs/research/2026-11-01-firefish-port-capability-matrix.md`](docs/research/2026-11-01-firefish-port-capability-matrix.md)。

### 明确不做

- **Mastodon API**：不提供任何 Mastodon 端点。
- **聊天 / 频道 / 剪贴 / 分组 / 画廊 / 页面 / 天线 / 列表**：Firefish 有、本实例不做，相关页面按契约降级或隐藏。
- **Relay**：首个版本不做。
- **数据迁移**：不导入其它实例的数据，实例从空库开始。

## 快速开始

依赖：Go 1.26+、Node 20+ 与 pnpm、PostgreSQL 14+（推荐 16/18）。

```bash
# 1. 前端（产出 web/dist，二进制启动时静态托管）
cd web && pnpm install && pnpm build && cd ..

# 2. 配置
cp config.example.toml config.toml    # 至少改 domain 与 database_url

# 3. 建库（示例）
createdb cinderfish

# 4. 编译并创建第一个管理员（注册模式为 admin 时没有注册入口）
go build -o cinderfish ./cmd/cinderfish
go build -o cinderfish-admin ./cmd/cinderfish-admin
./cinderfish-admin --config config.toml create --username admin --admin --password-stdin
# 密码从标准输入读入，避免留在 shell 历史里

# 5. 启动
./cinderfish --config config.toml
```

服务启动时自动执行数据库迁移（迁移脚本内嵌在二进制中，无需额外的迁移步骤）。默认监听
`127.0.0.1:3000`；`web_dir` 指向不存在目录时退化为纯 API 模式。

`cinderfish-admin` 另有 `promote` / `demote` 两个子命令用于授予或撤销管理员。

### 配置

所有键都可用环境变量覆盖：`CINDERFISH_<UPPER_SNAKE_KEY>`，例如 `CINDERFISH_DATABASE_URL`、
`CINDERFISH_PORT`、`CINDERFISH_REGISTRATION_MODE`、`CINDERFISH_FEDERATION_MODE`、
`CINDERFISH_REDIS_ENABLED`。完整键与含义见 [`config.example.toml`](config.example.toml) 与
[`docs/DEPLOY.md`](docs/DEPLOY.md)。

不需要邮件服务器也能起步：没有配置 `[mail]` 时，验证与重置链接会打印到日志里。

### 媒体在本地与 S3 之间迁移

```bash
go run ./cmd/cinderfish-media-migrate --config config.toml --to s3 --dry-run
```

## 部署

`deploy/` 下有三个可直接改用的文件：

- [`deploy/Caddyfile`](deploy/Caddyfile) — 反向代理、WebSocket 与媒体缓存头
- [`deploy/cinderfish.service`](deploy/cinderfish.service) — systemd 单元
- [`deploy/config.production.toml`](deploy/config.production.toml) — 生产配置模板

完整的 systemd、Caddy、备份与升级步骤见 [`docs/DEPLOY.md`](docs/DEPLOY.md)。

## 测试

```bash
go vet ./...
go test ./...            # 数据库测试与 Redis 集成测试在未设变量时自动跳过

# 带数据库的完整套件
CINDERFISH_TEST_DATABASE_URL="postgres://cinderfish:cinderfish@localhost:5432/cinderfish_test" \
  go test ./... -count=1

# 再带真实 Redis（miniredis 覆盖不到的部分）
CINDERFISH_TEST_DATABASE_URL="postgres://cinderfish:cinderfish@localhost:5432/cinderfish_test" \
CINDERFISH_TEST_REDIS_URL="redis://127.0.0.1:6379/0" go test ./... -count=1

# 前端：类型检查、移植规则检查、构建
cd web && pnpm typecheck && pnpm run check:port && pnpm build

# 性能自比（结果记录在 docs/research/2026-10-29-performance-baseline.md）
CINDERFISH_TEST_DATABASE_URL="postgres://cinderfish:cinderfish@localhost:5432/cinderfish_test" \
  go test ./internal/httpapi -run '^$' -bench . -benchmem -benchtime 2s
```

数据库测试会 `TRUNCATE` 测试库，**不要指向真实实例的库**。`web/scripts/check-port.mjs` 校验移植
规则：不得直接 `fetch`、不得自己读 token、不得拼接 RPC 路径、移植文件必须保留
`Ported from <repo>@<commit>:<path>` 头。

## 架构图

- 产物：[`.archify/architecture-cinderfish-overall/cinderfish-overall.html`](.archify/architecture-cinderfish-overall/cinderfish-overall.html)
- 源：[`.archify/architecture-cinderfish-overall/candidate.json`](.archify/architecture-cinderfish-overall/candidate.json)

图随里程碑更新，维护方式见该目录下的 `README.md`。

## 文档

| 文档 | 内容 |
|------|------|
| [`docs/HANDOVER.md`](docs/HANDOVER.md) | 交接说明：现状、缺口、逐波记录与验证证据 |
| [`docs/DEPLOY.md`](docs/DEPLOY.md) | 部署：systemd、Caddy、邮件、备份、升级 |
| [`docs/research/`](docs/research/) | 调研与契约台账（Misskey 端点覆盖、Firefish 移植矩阵、性能基线） |
| [`docs/superpowers/specs/2026-10-26-cinderfish-architecture-v0.md`](docs/superpowers/specs/2026-10-26-cinderfish-architecture-v0.md) | 架构基线与模块边界 |

> 实现任何端点前，先读 Misskey 源码里该端点的 `paramDef` 与 `res` schema。契约以源码为准，
> 文档是二手的；参考仓库的重建方式见 `AGENTS.md`。

## 许可

AGPL-3.0-only。前端组件移植自 Firefish 与 Misskey（均为 AGPL-3.0），每个移植文件保留其
SPDX 头与来源标注；来源清单见
[`docs/research/2026-11-01-firefish-port-source-and-license-register.md`](docs/research/2026-11-01-firefish-port-source-and-license-register.md)。
