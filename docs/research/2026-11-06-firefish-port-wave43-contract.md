# 第四十三波施工契约：四个兼容端点（2026-11-06）

Firefish 客户端实际调用、而我们一直缺的四个小端点，全部从已有表推导，不需要新迁移：

| 端点 | 用途 |
|------|------|
| `users/stats` | 账号信息页的计数块 |
| `hashtags/search` | 编辑器里的标签补全 |
| `i/registry/keys-with-type` | 客户端设置仓库的类型列表 |
| `notes/featured` | 探索页 / 推荐笔记 |

## 1. users/stats

`{userId?}` → `{notesCount, repliesCount, renotesCount, repliedCount, renotedCount,
pollVotesCount, pollVotedCount}`。三个笔记桶互斥：`notesCount` 只数既非回复也非转贴的笔记，
于是三者相加等于全部笔记。`repliedCount`/`renotedCount` 是被动计数（自己的笔记被别人回复/转贴
过的条数），`pollVotesCount` 是自己投出的票，`pollVotedCount` 是自己的投票收到的票。

`userId` 缺省时取调用者（需凭据）；上游要求必填，这是本实现的便利扩展。端点不要求登录，
因为返回的每个数字本来就在公开资料上。

实现在 `internal/account/stats.go`，五条聚合查询，删除笔记后立刻反映。

## 2. hashtags/search

`{query?, limit?, offset?, sort?}` → `string[]`。`query` 为空取最常用标签；非空按包含匹配。
排序固定「使用次数降序 → 标签字母升序」，`limit` 默认 10 上限 100。**`sort` 接受但忽略**：
上游的排序模式依赖 Redis 索引，这里只有使用次数一个信号，半实现一种模式比固定口径更糟。

只统计未删除且可见性为 `public`/`home` 的笔记标签，与 `hashtags/trend` 同一口径。

## 3. i/registry/keys-with-type

与 `i/registry/keys` 同参数（`scope` 生效，需登录）→ `{ key: type }`。`type` 由存储的 JSON
推断：`string`/`number`/`boolean`/`object`/`array`/`null`。上游把类型和值一起存；本实现只存值，
所以把数字存成字符串的客户端会看到 `string`，与它读回的值一致。

## 4. notes/featured

`{limit?, sinceId?, untilId?}` → `Note[]`。取近 30 天、`public`、本地、非转贴、未删除的笔记，
按 `renote_count * 2 + replies_count + 反应数` 降序，同分按 id 倒序。匿名与登录同列表。
30 天窗口是与上游的已知差异（上游用 Redis 评分索引，无窗口）。

## 5. 前端

- `api/endpoints.ts` 增加四个常量与类型条目；`api/types.ts` 增加 `UserStats`、
  `RegistryValueType`。
- `pages/SettingsAccountInfo.vue` 改为从 `users/stats` 取六个计数项，并把「修改密码」「两步验证」
  从「不可用」列表移出（它们早已有端点，且在安全页）。统计失败不弹错误，只隐藏计数块。

## 6. 有意的限制

1. `hashtags/search` 忽略 `sort`。
2. `notes/featured` 有 30 天窗口，老笔记不会出现。
3. `i/registry/keys-with-type` 的类型是推断值，不是写入时记录的。
4. 四个端点都没有新增测试。

## 7. 验收

```
gofmt -l internal/ cmd/                 # 空
go vet ./...                            # exit 0
cd web && pnpm exec vue-tsc --noEmit    # exit 0
cd web && pnpm run check:port           # hardTotal 0
cd web && pnpm build                    # exit 0
```

未执行 `go test`、数据库测试与浏览器走查。
