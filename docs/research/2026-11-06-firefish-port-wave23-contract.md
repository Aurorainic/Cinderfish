# 第二十三波施工契约：通知类型静音（2026-11-06）

通知设置页此前只能**只读**展示账号的静音类型：`i/update` 忽略 `mutingNotificationTypes`，
通知列表和未读角标也不看它。本波把它做成真功能，避免“能保存但没效果”的假控件。

## 1. 存储与写入

- `user_profiles.settings.mutingNotificationTypes`：字符串数组，无迁移。
- `i/update` 新增 `mutingNotificationTypes`（array 或 null）：
  - 只接受本后端实际产生的 5 类：`mention`、`reply`、`renote`、`quote`、`reaction`；
  - 未知类型 → 400 `INVALID_PARAM`（`info: "unknown notification type: …"`）；
  - 去重后保存，最多 5 项；`null` 或 `[]` 清空。
- `MeDetailed.mutingNotificationTypes` 从 profile 回读，缺省 `[]`。

## 2. 过滤行为

- `POST /api/notifications`：把账号的静音类型并入 `excludeTypes` 后再查询；
  显式 `includeTypes` 仍然优先（与 Misskey 的 `NotificationService` 一致）。
- `UnreadCount(recipientID, excludeTypes)`：未读计数用 `AND NOT (type = ANY(...))` 排除静音类型，
  因此关闭某类通知后红点会同步下降。
- `i` / `i/update` / `i/pin` 等打包 `MeDetailed` 的路径都走同一个 helper，行为一致。

## 3. 前端

| 文件 | 变化 |
|------|------|
| `account/types.ts` | `ProfileUpdateRequest.mutingNotificationTypes` |
| `pages/SettingsNotifications.vue` | 5 个逐类开关；切换即调用 `i/update`，失败回滚并显示错误 |

## 4. 有意的限制

1. 只有本后端产生的 5 类通知可静音；Misskey 的 `follow`、`pollEnded`、`chat` 等类型不存在，
   传进来会报 400 而不是静默接受。
2. 静音只影响**列表与未读计数**，不会阻止通知行写入数据库。
3. `notificationRecieveConfig`（按类型选择 “全部/仅关注者/无”）仍未实现。
4. 邮件通知类型（`emailNotificationTypes`）未实现，实例没有邮件发送能力。

## 5. 验收

`gofmt -l`、`go vet ./...`、`vue-tsc --noEmit`、`check:port`、`pnpm build` 通过。按用户
2026-11-06 的指示未执行 `go test`、数据库测试与浏览器走查；静音前后的列表与角标变化留待统一验证。
