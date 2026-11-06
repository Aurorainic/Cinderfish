# 第四十七波施工契约：锁推生效与关注通知（2026-11-06）

排查通知缺口时发现两处比邮件通知更基础的问题：

1. `following/create` 对本地目标直接建关系统，从不看对方的 `manuallyApprovesFollowers`——
   锁推账号会被立刻关注，`users.is_locked` 只有远端方向在用。
2. 关注不产生通知行：本地关注、锁推请求、远端入站关注三条路径都只有 Streaming 事件，
   `i/notifications` 刷新后看不到「谁关注了你」。

## 1. 锁推

`following/create` 拿到本地目标后先查 `users.is_locked`（新方法
`federation.Store.LocalUserLocked`）：

- `true` → 写 `follow_requests` 行、产生 `receiveFollowRequest` 通知、返回 200（与上游一样
  不区分「已关注」和「已申请」之外的差别）；
- `false` → 建关系统、产生 `follow` 通知、推 `follow`/`followed` Streaming 事件。

`i/update` 新增 `isLocked`（布尔），写到 `users.is_locked`——与入站 `Follow` 判断、actor 文档的
`manuallyApprovesFollowers` 同一个来源，不引入第二份状态。

`UserDetailed.isLocked` / `MeDetailed.isLocked` 不再是常量 `false`：`account.Profile` 增加
`IsLocked`，由 `GetProfile` 的查询顺带用一条标量子查询读出，避免改动十处重复的 user 选择语句。

## 2. 关注通知

| 触发 | 通知类型 | 收件人 |
|------|----------|--------|
| 本地直接关注 | `follow` | 被关注者 |
| 本地锁推账号收到关注 | `receiveFollowRequest` | 被申请者 |
| 入站远端 Follow（未锁） | `follow` | 被关注者 |
| 入站远端 Follow（锁） | `receiveFollowRequest` | 被申请者 |

本地两条走 `Server.createNotification`（内部调 `NotificationService.CreateForLocal` 并推
Streaming 通知事件）；入站两条走 `Store.Notification`，并新增 `Inbox.OnNotification` 钩子让 HTTP
层触发推送。`Inbox.OnFollow` 也接上了 → 远端关注现在会推 `follow`/`followed`。

## 3. 前端

`pages/SettingsPrivacy.vue`：把七个「不可用」占位换成 `CfSwitch`：

`isLocked`（锁推）、`autoAcceptFollowed`、`publicReactions`、`hideOnlineStatus`、`noCrawle`、
`preventAiLearning`、`isExplorable`。逐个保存（切换即调用 `i/update`），失败在块内提示。
`ProfileUpdateRequest` 增加 `isLocked`。

## 4. 有意的限制

1. 不产生 `followRequestAccepted` 通知（Misskey 也没有这一类型）；请求方仍靠 `followed` 事件
   或下次 `users/show` 得到结果。
2. 接受请求时不删除之前的 `receiveFollowRequest` 通知，历史保留。
3. 锁推变更立即影响下一次关注判断，但没有向远端重发 actor 文档；远端缓存要等它自己刷新。
4. 未新增测试。

## 5. 验收

```
gofmt -l internal/ cmd/                 # 空
go vet ./...                            # exit 0
cd web && pnpm exec vue-tsc --noEmit    # exit 0
cd web && pnpm run check:port           # hardTotal 0
cd web && pnpm build                    # exit 0
```

台账不变（137/130/310，没有新增路由）。未执行 `go test`、数据库测试与浏览器走查。
