# Cinderfish

Go 单二进制的个人 / 极小规模联邦社交实例。

- 后端契约：Misskey 原生 API（`POST /api/<endpoint>` RPC 风格）+ Streaming API
- 通信层：ActivityPub 联邦（WebFinger / Actor / Inbox / Outbox / HTTP Signatures）
- 前端：自建 Firefish 风格（Vue 3 + TypeScript）
- 依赖：PostgreSQL 必需；Redis 可选；媒体支持 local 与 S3 兼容存储

## 快速开始

```bash
# 依赖：Go 1.26+、Node 20+（构建前端）、PostgreSQL 14+
cd web && npm install && npm run build && cd ..   # 产出 web/dist
cp config.example.toml config.toml   # 按需修改
go build -o cinderfish ./cmd/cinderfish
./cinderfish --config config.toml
```

服务启动时自动执行数据库迁移（迁移脚本内嵌在二进制中）。

媒体在 local 与 S3 之间迁移：

```bash
go run ./cmd/cinderfish-media-migrate --config config.toml --to s3 --dry-run
```

## 当前进度

| 里程碑 | 状态 |
|--------|------|
| M1 骨架 | ✅ Go module、config、数据库迁移、`/api/meta`、`/api/ping` |
| M2 账户与认证 | ✅ 注册三模式、signin、MiAuth、OAuth 2.0 PKCE、`/api/i`、`/api/users/show` |
| M3 笔记与时间线 | ✅ 笔记 CRUD、四条时间线、反应、通知、Drive + local/S3 双存储 |
| M4 Streaming | ✅ WebSocket 端点、五个频道、事件广播（进程内 fanout + LISTEN/NOTIFY 跨进程中继） |
| M5 联邦 | ✅ WebFinger、Actor、Inbox/Outbox、HTTP Signatures、三模式策略、投递队列、双实例 smoke test |
| M6 前端 | ✅ Vue 3 时间线、发帖、反应、通知、资料、设置、管理页；WebSocket 实时更新 |
| M7 打磨 | ✅ 媒体迁移工具、账号管理、性能基线、部署文档 |

## 架构图

项目整体结构图（包结构、数据流、联邦链路、Streaming 中继）：

- 产物：[`.archify/architecture-cinderfish-overall/cinderfish-overall.html`](.archify/architecture-cinderfish-overall/cinderfish-overall.html)
- 源：[`.archify/architecture-cinderfish-overall/candidate.json`](.archify/architecture-cinderfish-overall/candidate.json)

图随里程碑更新，维护方式见该目录下的 `README.md`。

## 文档

- **交接说明**：[`docs/HANDOVER.md`](docs/HANDOVER.md) — 给接手者必读
- **部署指南**：[`docs/DEPLOY.md`](docs/DEPLOY.md) — systemd、Caddy、备份与升级
- **性能基线**：[`docs/research/2026-10-29-performance-baseline.md`](docs/research/2026-10-29-performance-baseline.md)
- 架构设计：[`docs/superpowers/specs/2026-10-26-cinderfish-architecture-v0.md`](docs/superpowers/specs/2026-10-26-cinderfish-architecture-v0.md)
- 调研记录：[`docs/research/`](docs/research/)

> 实现任何端点前，先读 Misskey 源码（`/tmp/refcheck/misskey`）中的 `paramDef` 与 `res` schema。契约以源码为准，文档是二手的。

## 测试

```bash
go vet ./...
go test ./...
CINDERFISH_TEST_DATABASE_URL="postgres://cinderfish:cinderfish@localhost:5432/cinderfish_test" go test ./...

# 性能自比（结果记录在 docs/research/2026-10-29-performance-baseline.md）
CINDERFISH_TEST_DATABASE_URL="postgres://cinderfish:cinderfish@localhost:5432/cinderfish_test" \
  go test ./internal/httpapi -run '^$' -bench . -benchmem -benchtime 2s
```
