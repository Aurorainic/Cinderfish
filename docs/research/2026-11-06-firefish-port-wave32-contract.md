# 第三十二波施工契约：已收录实例列表（2026-11-06）

管理台联邦页只管理模式与白名单，`federation/instances` 未实现。本波按库内数据聚合实例列表。

## 1. 端点

`POST /api/federation/instances`（需登录）请求 `{host?, limit?, offset?}`：

```sql
SELECT u.host, count(DISTINCT u.id) users,
       (SELECT count(*) FROM notes n WHERE n.user_host = u.host AND n.deleted_at IS NULL) notes
FROM users u WHERE u.host IS NOT NULL AND ($1 = '' OR u.host = $1)
GROUP BY u.host ORDER BY users DESC, u.host ASC LIMIT $2 OFFSET $3
```

- `limit` 默认 30、上限 100；`host` 走 `normalizeHost`；
- 返回 `FederationInstance[]`，软件与状态字段为 `null`/`false`（没有 `instances` 表、不联网）。

## 2. 前端

| 文件 | 变化 |
|------|------|
| `api/endpoints.ts` | `federation/instances` |
| `pages/AdminFederation.vue` | 「已收录实例」表格：主机（链到实例信息页）、用户数、帖子数；加载/空/错误状态 |

## 3. 有意的限制

1. 只统计库内已出现的远端主机，不是完整联邦清单。
2. 没有投递状态、pubSub、软件信息。
3. 没有分页 UI（一次 30 条），接口支持 offset。

## 4. 验收

`gofmt -l`、`go vet ./...`、`vue-tsc --noEmit`、`check:port`、`pnpm build` 通过。按用户
2026-11-06 的指示未执行 `go test`、数据库测试与浏览器走查。
