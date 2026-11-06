# 第七波施工契约：管理台用户筛选与文件盘点（2026-11-06）

第四波的管理台只做了四个页面，其中用户页只有 offset 翻页，文件页是占位。本波把这两块补成可用的
管理工具：用户列表按 Misskey 的参数筛选，文件列表跨账号盘点。

## 1. 端点

| 端点 | 变化 |
|------|------|
| `POST /api/admin/show-users` | 新增 `sort`、`state`、`origin`、`username`、`hostname`；省略参数时行为不变 |
| `POST /api/admin/drive/files`（新） | `limit`、`sinceId`、`untilId`、`userId`、`type`、`origin`、`hostname`；返回文件并附带上传者 `user` |

实现：

- `internal/account/admin_users.go`（新）：`AdminList`、`UsersByIDs`。默认排序用 `created_at ASC`
  而不是 Misskey 的 `id ASC`，因为引导管理员的 id 是手工写入的（`root1`），`id ASC` 会把他排到
  ULID 账号后面；在全部使用 ULID 的库里两者等价。
- `internal/media/admin.go`（新）：`ListAll` 跨账号查询，`type` 按前缀匹配（`image` / `image/png`）。
- `internal/httpapi/admin_drive.go`（新）：`handleAdminDriveFiles`，校验 `type` 只含字母数字与 `/ - + *`，
  `origin` 只接受 combined/local/remote。
- `internal/httpapi/admin_accounts.go`：`handleAdminShowUsers` 改用 `AdminList`。

## 2. 有意的降级

1. **没有 silenced/updated_at**：`state=silenced` 返回空集（而不是悄悄返回全部），`state=alive` 退化为
   「未冻结」，`±updatedAt` 排序退化为按 `created_at`。users 表没有这两个字段。
2. **没有总数**：响应仍是数组，分页靠 offset + 「本页取满 30 条」推断是否还有下一页。页面把这一点写在
   说明里，不画假的总数。
3. **文件页只读**：删除与批量清理需要回收站/二次确认设计，Misskey 走 `admin/drive/cleanup` 与单文件
   删除接口，本移植没有实现，页面明确写出。
4. **上传者只带 UserLite**：`user` 字段用于显示 @acct 与链接，不附完整 UserDetailed，避免每行再查资料。
5. **type 过滤是前缀匹配**：`image` 命中所有 `image/*`，`image/png` 命中精确类型；不支持逗号分隔的多值。

## 3. 前端

| 文件 | 变化 |
|------|------|
| `web/src/pages/AdminUsers.vue` | 排序/状态/来源三个下拉 + 用户名前缀/实例主机两个输入；应用/重置；说明改为真实行为 |
| `web/src/pages/AdminFiles.vue` | 占位页改为真实文件清单：来源/类型/上传者/主机筛选，缩略图、大小、上传者与时间，游标加载更多 |
| `web/src/pages/AdminIndex.vue` | 菜单新增「文件」 |
| `web/src/api/endpoints.ts`、`types.ts` | `AdminShowUsersParams` 扩参、`adminDriveFiles` 与 `AdminDriveFile` |

上传者筛选先把用户名解析成 `userId`（`users/show`），解析失败就地提示，不会退化成「显示全部」。

## 4. 验收

已执行 `go vet`、`go test ./...`（新增 2 个用例）、`vue-tsc`、`check:port`、`pnpm build`；真实二进制
验证筛选结果、非法参数 400、非管理员 403；无头浏览器走查桌面/移动的用户页筛选与文件页类型筛选。
证据见 HANDOVER §12W.3。
