# 第五十六波施工契约（关注关系的每关系设置：following/list、update、update-all、invalidate）

日期：2026-11-06
状态：已落地，证据见 `docs/HANDOVER.md` §12BU
上游参考：Misskey `4ff2edf`：

- `packages/backend/src/server/api/endpoints/following/{list,update,update-all,invalidate}.ts`
- `packages/backend/src/models/Following.ts`（`notify`、`withReplies` 两列）
- Firefish `packages/client/src/scripts/get-user-menu.ts`（`invalidateFollow` 菜单项）

## 1. 这一波补什么

`following/*` 是台账里缺口最小的命名空间（4 条），其中：

- `following/list`：本人关注列表 + `notification` 过滤，前端关注管理页需要
- `following/update` / `following/update-all`：单条 / 全部的「接收通知」「包含回复」开关
- `following/invalidate`：作为被关注者移除某个关注者（`following/delete` 只能取消自己的关注，
  表达不了这个方向）

## 2. 数据模型（迁移 v21）

```sql
ALTER TABLE followings
    ADD COLUMN IF NOT EXISTS notify TEXT,
    ADD COLUMN IF NOT EXISTS with_replies BOOLEAN NOT NULL DEFAULT TRUE;
```

`notify` 取 `normal` 或空（Misskey 的 `none` 落库为空）；`with_replies` 默认 true，与上游列默认值一致。

## 3. 端点

| 端点 | 凭据 | 参数 | 返回 |
|------|------|------|------|
| `following/list` | 是 | `notification?`、`limit`、`sinceId`、`untilId` | `Following[]`（带 `followee`；`notify` / `withReplies` 只在非默认时出现） |
| `following/update` | 是 | `userId`、`notify?`、`withReplies?` | 调用者自己的 `UserLite`（Misskey 就是这么返回的） |
| `following/update-all` | 是 | `notify?`、`withReplies?` | 204 |
| `following/invalidate` | 是 | `userId`（关注者） | 被移除者的 `UserLite` |

错误：`NO_SUCH_USER`、`FOLLOWEE_IS_YOURSELF`、`FOLLOWER_IS_YOURSELF`、`NOT_FOLLOWING`，
id 照抄 Misskey 源码。

## 4. 前端

`web/src/scripts/get-user-menu.ts` 增加「移除关注者」（`t("breakFollow")`）：当目标账号
`isFollowed`（对方关注我）且不是自己时出现，确认后调 `following/invalidate`，并把
`user.isFollowed` 置 false。该文件的降级注释同步删掉 `following/invalidate` 一条。

## 5. 写作用域

| 文件 | 动作 |
|------|------|
| `internal/db/db.go` | 迁移 v21 |
| `internal/account/follows.go` | `Follow.Notify/WithReplies`、`UpdateFollowing`、`UpdateAllFollowing` |
| `internal/httpapi/following_settings.go` | 四个处理器与 `packFollowingItem` |
| `internal/httpapi/server.go` | 四条路由 |
| `internal/httpapi/following_settings_test.go` | 两条用例 |
| `web/src/scripts/get-user-menu.ts` | 移除关注者 |
| `web/src/api/endpoints.ts` | 四个端点与类型 |
| 台账 | 151 路由 / 143 同名 / 297 未实现 |

## 6. 有意的限制

- `notify` / `withReplies` **只落库与回读，还没有消费方**：Misskey 用它们决定实时投递与
  「该账号每次发帖都通知」，Cinderfish 的 Streaming 目前不向关注者的 main 频道扇出新帖，
  也没有 `type: note` 通知。等这两条链路补上再接线，避免留下「开关拨了没反应」的界面
  （因此本波**没有**为这两个开关添加任何 UI）。
- `following/invalidate` 只删本地行，不向远端投递 Undo：Undo 必须以被移除者的密钥签名，
  本实例没有远端账号的私钥。本地关注者会收到 Streaming 的 unfollow 事件。
- `following/list` 只返回本人关注，不支持 `userId`（Misskey 同）。

## 7. 验收口径

- `go vet ./...` 通过；`CINDERFISH_TEST_DATABASE_URL=… go test ./...` 全绿
- `TestFollowingListAndUpdate`：关注后列表一条、默认不序列化 `withReplies`、update 后回读、
  `notification` 过滤、`update-all` 清空、非关注关系 400 `NOT_FOLLOWING`
- `TestFollowingInvalidate`：移除关注者后 `HasFollowing` 为假，再次移除 400
- `vue-tsc --noEmit`、`check:port`（hardTotal 0、missingKeys 0）、`pnpm build` 全绿

2026-11-07 补：`following/list` 与 `users/following|followers` 原先缺 `sinceDate`/`untilDate`，
第六十七波补上（`account.FollowQuery`），用例 `TestFollowListDateCursors`，见 HANDOVER §12CF。
