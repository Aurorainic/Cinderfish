# 第三十七波施工契约：公告编辑（2026-11-06）

公告此前只能新建和删除；改标题、改正文、下线一条公告都得删了重建，用户侧的已读记录也会跟着
`ON DELETE CASCADE` 一起消失。本波补齐 Misskey 的 `admin/announcements/update`，并把管理页的
表单改成「新建 / 编辑」双模式。

## 1. 端点

`POST /api/admin/announcements/update`（管理员）

| 参数 | 语义 |
|------|------|
| `id` | 必填；不存在 → 404 `NO_SUCH_ANNOUNCEMENT` |
| `title` | 可选；去空白后 1–256 字符 |
| `text` | 可选；去空白后 1–10000 字符 |
| `imageUrl` | 三态：未提供保持原值；`null` 清空；字符串按 `create` 的规则校验（≤500 字符，`http(s)://` 或 `/` 开头） |
| `expiresAt` | 三态：未提供保持原值；`null` 清空；数字为 Unix 毫秒，直接写入（不校验「晚于现在」，填过去的时间等于立即过期） |
| `isActive` | 可选布尔，切换上下线 |

返回 `204`。非管理员 403，未登录 401。

实现落在 `internal/announcement/service.go` 的 `UpdateParams` / `Update`：先 `Get` 读出当前行，
按「nil 指针 = 保持」合并，再整行写回并 `RETURNING` 扫描；handler 只做参数校验与三态解析，
三态用包内的 `nullableField`（`provided()` / `asUpdatePointer()`）判断，`expiresAt` 用
`json.RawMessage` 区分「缺字段 / null / 数字」。

## 2. 前端

| 文件 | 变化 |
|------|------|
| `api/endpoints.ts` | `admin/announcements/update` 与其 `EndpointMap` 条目 |
| `pages/AdminAnnouncements.vue` | 表单改走 `submit()`；`editingId` 为 null 时 `create`，否则 `update`；列表项加「编辑」按钮，编辑中显示「取消」并把表单标题改成「编辑公告」；`beginEdit` 回填字段，`toLocalInput` 把 ISO 时间转 `datetime-local` |

编辑时 `imageUrl` 与 `expiresAt` 总是显式发送（空则发 `null`），这样「清空」在 UI 上是可达的；
新建时仍省略空字段，与上一版行为一致。表单在编辑模式下要求有效期晚于现在（纯 UI 约束，
后端不拦）。

## 3. 有意的限制

1. 每次保存后重新拉一遍列表，不做乐观更新。
2. 编辑不提供 `isActive` 开关入口——下线通过删除或改有效期达成，后端字段留给将来的开关。
3. 没有「撤销编辑」以外的草稿保存。

## 4. 验收

`gofmt -l internal/ cmd/`、`go vet ./...`、`vue-tsc --noEmit`、`check:port`、`pnpm build` 通过。
按用户 2026-11-06 的指示未执行 `go test`、数据库测试与浏览器走查。
