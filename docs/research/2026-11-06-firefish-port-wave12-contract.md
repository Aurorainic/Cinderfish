# 第十二波施工契约：数据库概览（2026-11-06）

`AdminDatabase.vue` 是占位页。本波接两个只读端点，把表大小与索引定义做成管理页——个人实例
排查「谁占了空间」「哪个索引没跟上」时最需要的视图。

## 1. 端点

| 端点 | 返回 |
|------|------|
| `POST /api/admin/get-table-stats` | `{ [table]: { count, size } }` |
| `POST /api/admin/get-index-stats` | `[{ schemaname, tablename, indexname, tablespace, indexdef }]` |

实现：`internal/httpapi/admin_database.go`（新），`server.go` 注册。

SQL 与上游一致：

- 表：`pg_class` + `pg_namespace`，`relkind IN ('r','p')`，排除 `pg_catalog`、`information_schema`、
  `pg_toast`；`size = pg_total_relation_size(oid)`，`count = GREATEST(reltuples, 0)`。
- 索引：`pg_indexes` 直接映射，`tablespace` 的空值转 `null`。

## 2. 有意的限制

1. **行数是估算**：`reltuples` 是规划器统计，不是 `COUNT(*)`。从未 `ANALYZE` 的表显示 0；
   页面底部写明。
2. **只读**：没有 vacuum、删除、索引重建；这些需要单独设计确认流程。
3. **不返回数据库连接串或敏感元数据**：只有表名、大小、估算行数与索引定义。
4. **没有分页**：表与索引数量由 schema 决定（当前 31 表 / 75 索引），一次返回即可。

## 3. 前端

| 文件 | 变化 |
|------|------|
| `web/src/pages/AdminDatabase.vue` | 占位页改为概览：总大小、表数量与索引数、按大小排序的表（行数/大小/占比条）、索引列表与可展开定义 |
| `web/src/pages/AdminIndex.vue` | 内容组新增「数据库」 |
| `web/src/api/endpoints.ts`、`types.ts` | 两个端点常量、`TableStat`、`IndexStat` |

## 4. 验收

`go vet`、`go test ./...`（新增 1 个用例：表/索引存在性与权限）、`vue-tsc`、`check:port`、
`pnpm build` 全绿；真实二进制走查桌面/移动页面与索引展开。证据见 HANDOVER §12AB.3。
