# 第二十六波施工契约：隐藏话题标签（2026-11-06）

管理台的「隐藏的话题标签」页此前是 `TODO: port`，`admin/meta` 也没有该字段。本波实现它，
并让趋势真正排除这些标签。

## 1. 存储

`instance_meta` 新键 `hiddenTags`（JSON 字符串数组），无迁移。

## 2. 端点

- `admin/meta` 新增 `hiddenTags: string[]`。
- `admin/update-meta` 接受 `hiddenTags`：
  - 每项 trim、去掉开头的 `#`、转小写、去重；
  - 单条 ≤ 100 字符、最多 200 条，超出 400 `INVALID_PARAM`；
  - `null`/空数组清空。
- 公网 `meta` 不返回该字段。

## 3. 趋势过滤

`note.HashtagService.Trend(ctx, limit, hiddenTags)` 在 `exploded` CTE 里加
`AND lower(tag) <> ALL($3)`；`httpapi` 侧从运行时实例设置读取。只影响 `hashtags/trend`，
标签页、搜索和笔记内容不受影响（与 Firefish 的 `hiddenTagsDescription` 一致）。

## 4. 前端

| 文件 | 变化 |
|------|------|
| `api/types.ts`、`endpoints.ts` | `AdminInstanceMeta.hiddenTags`、update-meta 请求字段 |
| `pages/AdminHashtags.vue` | 真实页面：加载 `admin/meta`，每行一个标签，保存 `admin/update-meta` |
| `pages/AdminIndex.vue` | 「内容」分组新增「隐藏的话题标签」入口 |

## 5. 有意的限制

1. 只影响趋势，不影响标签页、搜索、探索页的用户/帖子列表。
2. 不校验标签是否符合话题标签语法（任意字符串都接受，只做去 `#`/小写）。
3. 没有缓存失效逻辑：趋势是即时 SQL，下一次请求就生效。

## 6. 验收

`gofmt -l`、`go vet ./...`、`vue-tsc --noEmit`、`check:port`、`pnpm build` 通过。按用户
2026-11-06 的指示未执行 `go test`、数据库测试与浏览器走查；保存与趋势过滤留待统一验证。
