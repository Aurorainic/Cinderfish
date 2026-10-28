# MilnFish

> 暂用名，非最终命名。

Go 单二进制的个人 / 极小规模联邦社交实例。

- 后端契约：Misskey 原生 API（`POST /api/<endpoint>` RPC 风格）+ Streaming API
- 通信层：ActivityPub 联邦（WebFinger / Actor / Inbox / Outbox / HTTP Signatures）
- 前端：自建 Firefish 风格（Vue 3 + TypeScript）
- 依赖：PostgreSQL 必需；Redis 可选；媒体支持 local 与 S3 兼容存储

## 快速开始

```bash
# 依赖：Go 1.26+、PostgreSQL 14+
cp config.example.toml config.toml   # 按需修改
go build -o milnfish ./cmd/milnfish
./milnfish --config config.toml
```

服务启动时自动执行数据库迁移（迁移脚本内嵌在二进制中）。

## 当前进度

| 里程碑 | 状态 |
|--------|------|
| M1 骨架 | ✅ Go module、config、数据库迁移、`/api/meta`、`/api/ping` |
| M2 账户与认证 | ✅ 注册三模式、signin、MiAuth、OAuth 2.0 PKCE、`/api/i`、`/api/users/show` |
| M3 笔记与时间线 | ✅ 笔记 CRUD、四条时间线、反应、通知、Drive + local/S3 双存储 |
| M4 Streaming | 未开始 |
| M5 联邦 | 未开始 |
| M6 前端 | 未开始 |
| M7 打磨 | 未开始 |

## 文档

- **交接说明**：[`docs/HANDOVER.md`](docs/HANDOVER.md) — 给接手者必读
- 架构设计：[`docs/superpowers/specs/2026-10-26-milnfish-architecture-v0.md`](docs/superpowers/specs/2026-10-26-milnfish-architecture-v0.md)
- 调研记录：[`docs/research/`](docs/research/)

> 实现任何端点前，先读 Misskey 源码（`/tmp/refcheck/misskey`）中的 `paramDef` 与 `res` schema。契约以源码为准，文档是二手的。

## 测试

```bash
go vet ./...
go test ./...
```
