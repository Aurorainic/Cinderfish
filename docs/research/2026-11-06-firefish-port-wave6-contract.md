# 第六波施工契约：搜索与标签（2026-11-06）

本波取代 `Search.vue` 与 `Tag.vue` 的降级实现，补上三个 Misskey 端点。搜索是「找到内容」的最后一块
空白：没有它，个人实例只能沿时间线翻。契约仍遵守前几波的规则——端点以 Misskey 源码为准，
没有引擎就不假装有。

## 1. 端点

| 端点 | 参数 | 行为 |
|------|------|------|
| `POST /api/notes/search` | `query`（必填）、`limit`、`offset`、`sinceId`、`untilId`、`host`、`userId`、`withFiles` | 正文 `ILIKE` 包含匹配；可见性、静音、屏蔽、实例静音全部生效 |
| `POST /api/notes/search-by-tag` | `tag`（必填）、`limit`、`sinceId`、`untilId`、`withFiles` | `notes.tags` 大小写不敏感匹配；同一套过滤 |
| `POST /api/users/search` | `query`（必填）、`origin`（local/remote/combined）、`host`、`limit`、`offset` | 用户名/昵称/简介匹配；排除冻结账号与双向屏蔽；带 `notesCount` 等计数 |

实现位置：

- `internal/note/search.go`（新）：`Search`、`SearchByTag`、`escapeLike`、`scanNotes`；
  时间线的静音/屏蔽/实例静音条件抽成 `queryBuilder.addViewerRelationFilters`，两处共用。
- `internal/account/search.go`（新）：`SearchUsers`。
- `internal/httpapi/search.go`（新）：三个 handler；`viewerIDForSearch` 让匿名可用、坏 token 视为匿名。
- `server.go` 注册三条 POST 路由。

## 2. 与 Misskey 的差异（有意为之）

1. **没有全文索引**：走 Misskey 自己的 ILIKE 回退。`query` 只匹配 `notes.text`，不匹配 CW、附件描述
   与投票选项；上游配置了 Elasticsearch/Meilisearch/Sonic 时行为会更强。
2. **`notes/search-by-tag` 只支持 `tag` 单标签形式**：上游还有 `query: string[][]` 的布尔组合，
   本移植没有调用方。
3. **`users/search` 增加 `sinceId`/`untilId`**：Misskey 只有 offset；这里加上 id 游标，让前端复用
   同一个分页组件。offset 仍然可用。
4. **`users/search` 返回计数**：上游的 UserDetailed 也带 `notesCount` 等字段，但一次搜索要多跑
   2×N 条聚合查询。个人实例可接受，结果页因此不会显示「0 帖子」。
5. **标签大小写**：`notes.tags` 保留作者写法（去重时按小写），所以匹配用
   `lower(t) = lower($tag)`，而不是 jsonb 的大小写敏感 `?`。

## 3. 前端

| 文件 | 变化 |
|------|------|
| `web/src/pages/Search.vue` | 从「接口未提供」的占位页改为真实搜索页：输入框、帖子/用户页签、按 query remount 的分页列表 |
| `web/src/pages/Tag.vue` | 从「本地时间线里客户端过滤 tags」改为 `notes/search-by-tag`，因此联邦模式下也能找到远端帖子 |
| `web/src/components/MkNavbar.vue` | 把 `search` 从不可用菜单集合里移除，「更多」菜单重新出现搜索入口 |
| `web/src/api/endpoints.ts`、`types.ts` | 三个端点常量与参数类型 |

`Search.vue` 不保留上游的 swiper：页签直接切换面板，键盘与窄屏都更稳，也避免为一个页签引入手势依赖。

## 4. 验收

已执行：`go vet ./...`、`go test ./...`（含数据库，新增 5 个搜索测试）、`vue-tsc --noEmit`、
`check:port`、`pnpm build` 全绿；真实二进制 `:3991` 验证匿名/作者搜索数量、通配符转义、标签大小写、
用户搜索 origin 校验与屏蔽过滤；无头浏览器走查桌面与移动的搜索页、用户页签与标签页。
证据见 HANDOVER §12V.3。
