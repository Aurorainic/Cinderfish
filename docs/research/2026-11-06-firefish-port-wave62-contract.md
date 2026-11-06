# 第六十二波施工契约（users/relation 与 users/get-frequently-replied-users）

日期：2026-11-06
状态：已落地，证据见 `docs/HANDOVER.md` §12CA
上游参考：Misskey `4ff2edf`：

- `packages/backend/src/server/api/endpoints/users/relation.ts`（单个或批量 id，返回关系标志）
- `packages/backend/src/server/api/endpoints/users/get-frequently-replied-users.ts`
  （`{user, weight}[]`，weight 为回复次数）

## 1. 这一波补什么

两个前端会用到、数据已经齐全的 `users/*` 端点：

1. `users/relation`：一次拿到「是否关注 / 是否被关注 / 有无待审请求 / 屏蔽 / 静音」，
   免去页面连发五六个查询
2. `users/get-frequently-replied-users`：某用户最常回复的对象，用户页的推荐位用得到

## 2. 实现

### 2.1 users/relation

- `userId` 支持单值与数组；响应的形状跟随请求（单值 → 对象，数组 → 同序数组）
- 标志来自已有能力：`federationStore.HasFollowing` / `HasPendingFollowRequest`、
  `accounts.IsMuted` / `IsBlocked`（两个方向）
- `isRenoteMuted` 恒为 `false`：没有 renote-mute 表
- 查询自己时全部为 `false`，不报错（与 Misskey 行为一致）

### 2.2 users/get-frequently-replied-users

```sql
SELECT reply_user_id, count(*) AS weight
FROM notes
WHERE user_id = $1 AND reply_user_id IS NOT NULL AND deleted_at IS NULL
GROUP BY reply_user_id
ORDER BY weight DESC, reply_user_id ASC
LIMIT $2
```

按 weight 降序、id 升序打破平局（稳定输出），逐个打包成 `UserDetailed` 并附 `weight`。
已删除的回复不计入；用户行不存在的目标跳过。

## 3. 写作用域

| 文件 | 动作 |
|------|------|
| `internal/httpapi/users_relation.go` | 两个处理器 |
| `internal/httpapi/server.go` | 两条路由 |
| `internal/httpapi/users_relation_test.go` | 两条用例 |
| 台账 | 156 / 148 / 292 |

## 4. 有意的限制

- `isRenoteMuted` 恒 false，见上。
- `users/relation` 不做 Credential 校验以外的可见性检查：只返回调用者自己的关系，不泄露他人隐私。
- 频繁回复统计只看 `reply_user_id`，不区分回复的具体笔记是否仍可见（与每笔记数一致）。

## 5. 验收口径

- `go vet ./...` 通过；`CINDERFISH_TEST_DATABASE_URL=… go test ./...` 全绿
- `TestUsersRelation`：双向关注两个标志为真、静音标志正确、批量请求返回同序数组
- `TestUsersFrequentlyReplied`：bob 权重 2 排第一、carol 权重 1 第二
