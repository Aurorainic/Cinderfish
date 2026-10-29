# 性能基线（自比）

- 日期：2026-10-29
- 代码：`internal/httpapi/bench_test.go`（M7 加入）
- 结论先行：这是本项目的**自比基线**，用于判断"同一台机器上，同一个实例是变快还是变慢"。它既不是容量承诺，也不可与其他项目或朋友的基准表直接对比。

## 怎么跑

```bash
CINDERFISH_TEST_DATABASE_URL="postgres://cinderfish:cinderfish@localhost:5432/cinderfish_test" \
  go test ./internal/httpapi -run '^$' -bench . -benchmem -benchtime 2s
```

基准走真实路由、真实 PostgreSQL 与真实 `httptest` HTTP 环，没有 mock handler。数据库为空时先建一个 `bench` 用户，时间线基准预置 100 条笔记。

## 环境

| 项 | 值 |
|----|----|
| CPU | AMD Ryzen 7 6800H with Radeon Graphics（16 逻辑核） |
| OS | Linux |
| Go | 1.26.8 |
| PostgreSQL | 18，本机，`cinderfish_test` 库 |
| 数据库连接 | 本机 loopback，未开 TLS |
| HTTP | `net/http/httptest`，本机 loopback，未开 TLS |

## 结果

`ns/op` 越小越好；`allocs/op` 是每次请求的堆分配次数。

| 基准 | 请求 | ns/op | 约合请求/秒（单核） | B/op | allocs/op |
|------|------|------:|--------------------:|-----:|----------:|
| `BenchmarkMeta` | `POST /api/meta` | 93,350 | ~10,700 | 23,347 | 260 |
| `BenchmarkLocalTimeline` | `POST /api/notes/local-timeline`（100 条库存，取 20） | 2,730,467 | ~366 | 208,772 | 3,415 |
| `BenchmarkNotesCreate` | `POST /api/notes/create` | 2,864,225 | ~349 | 38,524 | 554 |
| `BenchmarkNotifications` | `POST /api/notifications`（20 条） | 193,260 | ~5,175 | 13,168 | 164 |
| `BenchmarkActorJSON` | `GET /users/{id}`（ActivityPub JSON） | 326,328 | ~3,064 | 22,179 | 313 |
| `BenchmarkWebIndex` | `GET /timeline`（SPA fallback） | 78,251 | ~12,780 | 8,201 | 91 |

## 怎么读这些数

- **写路径（约 350 req/s）**：`notes/create` 每次要做提及解析、可见性写入、通知、Streaming 发布与联邦投递入队；2.8 ms 里大部分是 PostgreSQL 往返与事务，不是 Go 计算。
- **时间线（约 370 req/s）**：每次取 20 条笔记，且每条要打包作者、反应计数与可见性；`allocs/op` 明显高于写路径，说明**打包是主要成本**。若将来要优化，先做批量查询（作者、反应）而不是碰 SQL 索引。
- **通知与 Actor（3k–5k req/s）**：纯读 + JSON 编码，属于预期量级。
- **SPA 回退（约 12k req/s）**：每次读 `index.html`（设计如此，重建无需重启）；如果这成为热点，可以加 mtime 缓存，但个人实例规模下没有必要。
- **meta（约 10k req/s）**：无数据库，纯内存组装，可作为其它无库端点的上限参考。

## 运行时偏差与不作为验收标准的部分

- 单进程、单机、无反向代理、无 TLS、无并发限流；真实部署会叠加网络与代理开销。
- 数据库是空的或只有 100 条笔记；数据量增长后时间线会变慢，这个基线不预测拐点。
- 基准包含 `b.Fatalf` 分支与请求构造开销，属于测量噪声的一部分。
- 没有测 Streaming 长连接、联邦出站 HTTP、S3 后端——它们依赖外部服务，不适合放进这个基准。
- 没有与朋友私有项目的基准表对比，按项目规约也不应对比。

## 下次怎么比较

1. 在同一台机器、同一 PostgreSQL 配置下重跑上面的命令。
2. 只比较同一基准的 `ns/op` 与 `allocs/op`，变化超过约 10% 才值得解释。
3. 记录结果时更新本文件的日期与表格，保留旧值到"历史"小节，避免只留一个数字而看不出趋势。

## 历史

| 日期 | 代码 | meta | localTimeline | notesCreate | notifications | actor | webIndex |
|------|------|-----:|--------------:|------------:|--------------:|------:|---------:|
| 2026-10-29 | `M7 性能基线` | 93,350 | 2,730,467 | 2,864,225 | 193,260 | 326,328 | 78,251 |
