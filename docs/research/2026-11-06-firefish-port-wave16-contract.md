# 第十六波施工契约：通知页签与提及（2026-11-06）

通知页此前只有单一列表：后端没有 `unreadOnly`，也没有 `notes/mentions`，上游的三个页签因此被整体
砍掉。本波补回「全部 / 未读 / 提及」。

## 1. 端点

| 端点 | 行为 |
|------|------|
| `POST /api/i/notifications` | 新增 `unreadOnly: boolean`；为 true 时只返回未读 |
| `POST /api/notes/mentions`（新） | 提及当前登录用户的帖子；`following`、`visibility`、`limit`、`sinceId`/`untilId` |

`notes/mentions` 的匹配：`notes.mentions` 存的是本地用户 id，直接用 jsonb 的 `?` 判断成员；
可见性、静音、屏蔽、实例静音复用与时间线相同的过滤。`following=true` 只保留关注者（及自己）的帖子。
匿名调用返回 401。

实现：`internal/note/mentions.go`（新）、`internal/httpapi/mentions.go`（新）、
`i/notifications` 的 handler 与 `NotificationService.List` 增加 `unreadOnly`。

## 2. 前端

`Notifications.vue`：

- 页签「全部 / 未读 / 提及」，用 `MkPageHeader` 的 tab 机制；
- 未读页签给 `MkNotifications` 传 `unreadOnly`；
- 提及页签用 `MkNotes` 分页 `notes/mentions`；
- **取消页面加载时的「全部标为已读」**：否则未读页签永远是空的。单个通知在展示时由
  `MkNotifications` 标记已读，另在页头提供显式的「全部标为已读」动作。

`MkNotifications` 新增 `unreadOnly` prop，并入 `CfPagination` 的参数。

## 3. 有意的限制

1. 没有 directNotes 页签：本移植没有聊天范围。
2. `notes/mentions` 的远端提及：远端账号的 mention 不进入本地 `mentions` 数组（既有的联邦取址设计），
   所以被远端账号提及时不会出现在此列表。
3. 未读页签在展示通知后会随「已读」立即减少（单个展示即标记），这是既有行为。

## 4. 验收

`go vet`、`go test ./...`（新增提及与未读过滤用例）、`vue-tsc`、`check:port`、`pnpm build` 全绿；
真实二进制走查页签与未读/提及内容。证据见 HANDOVER §12AF.3。
