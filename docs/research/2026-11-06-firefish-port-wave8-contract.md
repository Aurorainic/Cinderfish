# 第八波施工契约：实例公告（2026-11-06）

`Announcements.vue` 与 `AdminAnnouncements.vue` 一直是「接口未提供」的占位页。本波补上公告的数据表与
四个端点，把两页做成真实功能，并补一条 v10 迁移。

## 1. 数据与端点

迁移 v10（`internal/db/db.go`）：`announcements(id, title, text, image_url, is_active, created_at,
updated_at, expires_at)`，两个索引。

| 端点 | 权限 | 行为 |
|------|------|------|
| `POST /api/announcements` | 公开 | 有效公告（`is_active` 且未过期），id 倒序，`limit` + `untilId` |
| `POST /api/admin/announcements/create` | 管理员 | `title`(≤256) / `text`(≤10000) / `imageUrl`(http(s) 或 `/…`) / `expiresAt`（Unix ms，必须未来） |
| `POST /api/admin/announcements/list` | 管理员 | 全部公告，含已过期 |
| `POST /api/admin/announcements/delete` | 管理员 | 按 id 删除，幂等返回 204 |

实现：`internal/announcement/service.go`（新）、`internal/httpapi/announcements.go`（新），
`Server` 增加 `announcements` 字段并在 `NewServerWithDB` 里装配。

## 2. 有意的降级

1. **没有已读跟踪**：`i/read-announcement` 未实现，`isRead` 恒为 false；公告页明确写出这一点，
   不假装记得用户点过「知道了」。
2. **没有首页横幅**：横幅需要已读状态与未读计数，否则会永远挂着无法关闭；本波只做列表页。
3. **没有 update 端点**：管理页是「删除后重发」，代码注释与 HANDOVER 记录该工作流。
4. **正文按纯文本渲染**：保留换行，不解析 MFM；上游公告也以纯文本为主。

## 3. 前端

| 文件 | 变化 |
|------|------|
| `web/src/pages/Announcements.vue` | 占位页改为真实列表：时间、图片、正文、有效期、加载更多 |
| `web/src/pages/AdminAnnouncements.vue` | 占位页改为管理页：新建表单（标题/正文/图片/有效期）、列表、删除确认、过期标记 |
| `web/src/pages/AdminIndex.vue` | 菜单新增「公告」 |
| `web/src/api/endpoints.ts`、`types.ts` | 四个端点与 `Announcement` 类型 |

## 4. 验收

`go vet`、`go test ./...`（新增 3 个用例，含过期隐藏）、`vue-tsc`、`check:port`、`pnpm build` 全绿；
真实二进制走查管理页表单发布、公开页展示与移动端。证据见 HANDOVER §12X.3。
