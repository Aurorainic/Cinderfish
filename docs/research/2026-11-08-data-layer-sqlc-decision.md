# 2026-11-08 数据层优化决策:不迁移 sqlc,落地轻量门禁

来源:`.workbuddy/memory/2026-11-07.md` 与
`docs/research/2026-11-07-miuno-borrowable-practices.md` L3 节提出的问题
「是否把 Cinderfish 的 store 层迁移到 sqlc」。本文是实测数据后的拍板记录。

## 现状实测(2026-11-08)

| 指标 | 值 | 说明 |
|------|-----|------|
| 生产 Go 代码(internal/) | 16,446 行 | `find internal -name '*.go' ! -name '*_test*'` |
| SQL 调用点(Query/Exec/tx.*) | 324 处 | `grep pool.Query/Exec、tx.Query/Exec` |
| 动态 SQL(fmt.Sprintf 拼 SQL) | 4 处 | 全部在 `queryBuilder`(conds/args 成对)收拢 |
| 迁移组织 | `internal/db/db.go` map[int]string | v1–v25,单二进制部署,无外部迁移文件 |
| store 抽象 | 8 个窄接口,struct 持 `*pgxpool.Pool` | 无生成物、无列登记表 |

## 决策:不迁移 sqlc

维持 2026-11-07 调研结论,并补充实测定量依据:

1. **收益面太窄**。sqlc 的核心价值是「查询编译期类型检查 + 防手改生成物 + 防列漂移」。
   本仓库 324 个调用点里只有 4 处动态 SQL 且已统一走 `queryBuilder`(占位符与 args 成对推进,
   正是 AGENTS.md 那条「placeholder 静默错位」教训的固化)。其余 320 处是静态 SQL,
   `go vet` + 测试已覆盖,sqlc 的类型检查增量很小。
2. **成本是一次 store 层重写**。Miuno 的 64 个 `.sql` 文件 + 2,456 处 `sqlc.arg` 是从第一天
   就按 sqlc 组织的结果;反向迁移 16k 行代码需要重排全部查询文件、命名映射与回归金标准,
   换不来用户可见的行为改进。
3. **迁移仍内嵌在 Go**。sqlc 假设 `sql/` 目录 + `schema.sql` 终态;本仓库刻意保持
   「单二进制部署」(migrations 是 `map[int]string`)。引入 sqlc 会连带动摇迁移组织,
   违反「部署只有一个二进制」的设计意图。

## 替代落地(本次一并完成)

借 Miuno 的轻量部分,不引工具链:

1. **迁移不可变门禁**:`scripts/check-migrations.sh` 以 HEAD 比对 `internal/db/db.go` 里
   已发布的版本条目——已出现过的版本号被修改或删除即失败,新增版本必须严格递增。
   (替代 sqlc 的 `check-migrations.sh`,成本 ~40 行 bash。)
2. **AGENTS.md schema 版本失配修复**:22 → 25(实际以 `internal/db/db.go` 为准)。
3. **CI Go 检查**:新增 `.github/workflows/ci.yml`(gofmt / go vet / go test,含数据库
   service),补上「CI 完全不跑 Go 检查」的缺口。
4. **性能补漏**(与 sqlc 无关但同属数据层):`pgxpool.MaxConns` 配置化(`database_max_conns`),
   `users/show` 批量 userIds 上限 100,`notes.file_ids` GIN 索引(v26,服务 v25 触发器的
   `ANY(file_ids)` 全表扫)。

## 不做的事(记录在案)

- 列语义登记表(P2):仅当未来做 schema 重构/数据迁移时再引入。
- 终态 schema dump 比对(P1):暂缓;迁移门禁先行,若迁移链出过一次事故再上。
- Mastodon 层、完整 fork 前端:维持 2026-11-07 结论。
