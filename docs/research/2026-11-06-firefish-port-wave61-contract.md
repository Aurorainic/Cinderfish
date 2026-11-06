# 第六十一波施工契约（hashtags/show、hashtags/list、hashtags/users 与标签页统计）

日期：2026-11-06
状态：已落地，证据见 `docs/HANDOVER.md` §12BZ
上游参考：Misskey `4ff2edf`：

- `packages/backend/src/server/api/endpoints/hashtags/{show,list,users}.ts`
- `packages/backend/src/core/entities/HashtagEntityService.ts`（Hashtag 实体字段）
- `packages/backend/src/misc/normalize-tag.ts`（`normalizeTag`）

## 1. 这一波补什么

三个 `hashtags/*` 端点，并让标签页的统计不再靠扫描 `hashtags/trend`：

| 端点 | 用途 |
|------|------|
| `hashtags/show` | 单个标签的计数；标签页头部要显示「被 N 人提到」 |
| `hashtags/list` | 按计数排序的标签榜（此前只能用 trend 的 7 天窗口） |
| `hashtags/users` | 上游按「用户资料上的标签」列出用户，Cinderfish 没有这个模型，返回空数组 |

## 2. 实现

Misskey 用一张 `hashtags` 表保存增量计数；Cinderfish 没有该表，改为**从 `notes.tags` 直接聚合**
（与 `hashtags/trend` 同一套口径）：

```sql
SELECT count(DISTINCT n.user_id),
       count(DISTINCT n.user_id) FILTER (WHERE n.user_host IS NULL),
       count(DISTINCT n.user_id) FILTER (WHERE n.user_host IS NOT NULL)
FROM notes n, jsonb_array_elements_text(n.tags) AS tag
WHERE n.deleted_at IS NULL AND lower(tag) = $1
```

- `normalizeHashtag` 与 Misskey 的 `normalizeTag` 一致：去 `#` / `＃`、trim、小写
- `hashtags/show` 计数为 0 时返回 404 `NO_SUCH_HASHTAG`（id 照抄）
- `hashtags/list` 支持 `limit` 与全部 12 个排序枚举（2026-11-07 第六十七波补齐）：
  `mentionedUsers/LocalUsers/RemoteUsers` 按各自计数排；`attached*` 计数恒为 0，
  因此 `attached*Only` 过滤直接返回空数组，`attached*` 排序回退到对应的 mentioned* 计数
  （页面保持稳定）。
- `hashtags/users` 解析参数后返回 `[]`：上游列的是资料标签，Cinderfish 不存这个数据

前端 `web/src/pages/Tag.vue` 头部改用 `hashtags/show`，并显示「全部时间」而不是「最近 7 天」。

## 3. 写作用域

| 文件 | 动作 |
|------|------|
| `internal/httpapi/hashtags.go` | 三个处理器 + `hashtagCounts` + `normalizeHashtag` |
| `internal/httpapi/server.go` | 三条路由（去掉一次 `hashtags/search` 重复注册） |
| `internal/httpapi/hashtags_test.go` | 两条用例 |
| `web/src/pages/Tag.vue` | 统计改用 `hashtags/show` |
| `web/src/api/{types,endpoints}.ts` | `Hashtag` 类型与三个端点 |
| i18n | 新增 `allTime`（zh-CN / ja-JP） |
| 台账 | 154 / 146 / 294 |

## 4. 有意的限制

- 计数按需聚合，不缓存：个人实例的笔记量下这是毫秒级查询，换来的是永远与笔记一致的数字。
- `hashtags/users` 恒为空数组，不按笔记标签凑数（那会与上游语义不符）。
- `attached*` 计数与排序没有真实数据，见上。
- 不做标签订阅（`hashtags/*` 的订阅端点在 Misskey 里属于 antennna 一类，未列入本波）。

## 5. 验收口径

- `go vet ./...` 通过；`CINDERFISH_TEST_DATABASE_URL=… go test ./...` 全绿
- `TestHashtagsShowAndList`：`#golang` 的计数为 2/2/0、未知标签 404 `NO_SUCH_HASHTAG`、
  升序与降序排序、attached 过滤为空
- `TestHashtagsUsersIsEmpty`：恒为空数组
- `vue-tsc --noEmit`、`check:port`（hardTotal 0、missingKeys 0）、`pnpm build` 全绿
