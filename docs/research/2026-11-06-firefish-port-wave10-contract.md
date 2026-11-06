# 第十波施工契约：举报与处理（2026-11-06）

`AdminAbuses.vue` 是占位页，用户菜单里也没有举报入口。本波补上举报表与三个端点，让实例能接收、
查看与标记处理举报。举报是安全功能，宁可少做（不转发、不自动处置）也不做半套。

## 1. 数据与端点

迁移 v11：`abuse_reports(id, reporter_id, target_user_id, comment, resolved, created_at)`，
被举报者与举报者都在 `users` 外键上级联删除。

| 端点 | 权限 | 行为 |
|------|------|------|
| `POST /api/users/report-abuse` | 登录 | `userId` + `comment`（1–2048 字符）；自举报 400，未知用户 404，返回 204 |
| `POST /api/admin/abuse-user-reports` | 管理员 | `limit`/`sinceId`/`untilId`、`state`（resolved/unresolved/null）、`reporterOrigin`、`targetUserOrigin`；返回带双方 UserDetailed 的数组 |
| `POST /api/admin/resolve-abuse-user-report` | 管理员 | `reportId`，标记已处理并返回 204；未知 id 幂等 |

实现：`internal/report/service.go`（新）、`internal/httpapi/reports.go`（新），`server.go` 装配与注册。

## 2. 有意的降级

1. **不转发到远端实例**（`forward-abuse-user-report` 未实现）：响应里 `forwarded` 恒为 false，页面注明。
2. **不通知管理员**：没有邮件/推送，管理员通过管理台查看。
3. **只针对账号**：Misskey 的契约只有 `userId`，帖子菜单的举报也归到作者账号，不单独记录 noteId。
4. **没有自动处置**：标记已处理只是状态位，不冻结账号；处置仍用用户页的冻结按钮。

## 3. 前端

| 文件 | 变化 |
|------|------|
| `web/src/scripts/get-user-menu.ts` | 非管理员用户对他人多出「举报」项：`os.inputParagraph` 收集说明 → `users/report-abuse` → 成功 toast |
| `web/src/pages/AdminAbuses.vue` | 占位页改为列表：状态/举报者来源/被举报者来源筛选、双方头像与链接、说明、标记已处理、加载更多 |
| `web/src/pages/AdminIndex.vue` | 管理菜单新增「举报」 |
| `web/src/api/endpoints.ts`、`types.ts` | 三个端点与 `AbuseReport` |

管理员自己不加举报项（避免管理队列里出现管理员自己的举报），仍保留冻结入口。

## 4. 验收

`go vet`、`go test ./...`（新增 2 个用例）、`vue-tsc`、`check:port`、`pnpm build` 全绿；真实二进制
走查：用户页菜单举报 → 管理台看到并标记已处理 → 筛选与移动端。证据见 HANDOVER §12Z.3。
