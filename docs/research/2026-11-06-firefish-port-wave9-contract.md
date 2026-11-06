# 第九波施工契约：账号数据导出与账号注销（2026-11-06）

`SettingsImportExport.vue` 与 `SettingsDeleteAccount.vue` 一直是占位页。本波补两个端点，把「带走数据」
与「彻底离开」做成真实功能。两者都是破坏性/敏感性较高的路径，所以刻意收紧。

## 1. 端点

| 端点 | 行为 |
|------|------|
| `POST /api/i/export` | 同步生成账号备份 JSON，作为附件下载 |
| `POST /api/i/delete-account` | 校验密码后**硬删除**账号；先删网盘对象再删用户行 |

实现：`internal/httpapi/account_data.go`（新），`server.go` 注册两条路由。

### i/export 的内容

`{ exportedAt, format: "cinderfish-account-export@1", user, notes, followings, followers,
favorites, driveFiles }`：

- `user` 是 `MeDetailed`（不含 token）；
- `notes` 只含本人的帖子，包含 `cw` / 可见性 / 回复与转帖 id / 附件 id / 标签 / 时间；
- 关注与粉丝是 `{username, host, createdAt}`；
- 收藏是 `{noteId, createdAt}`；
- 网盘只导出元数据（名称、类型、大小、时间），不打包文件本体。

### i/delete-account

- 请求 `{ password }`；密码错误 403 `INVALID_PASSWORD`；
- 最后一个未冻结的管理员不能删除，返回 400 `LAST_ADMIN`；
- 删除前按 100 条一页遍历并删除该用户的网盘对象，再 `DELETE FROM users`（其余表级联）；
- 返回 204，原 token 立即失效。

## 2. 与上游的差异

1. **同步 JSON 而不是异步 zip**：Firefish 把导出拆成 `i/export-notes` 等任务并通过通知投递。
   Cinderfish 的账号数据量小，同步返回一个可读 JSON 更实用；格式写明是自有格式，不冒充 Misskey zip。
2. **没有导入**：导入需要格式定义、冲突规则与限流，页面明确说明并只提供导出按钮。
3. **硬删除**：Firefish 先标记 `isDeleted` 再由任务清理；这里立即删除，页面把「无法恢复」写在最前面。
4. **没有已读/任务进度**：同步请求，前端只显示「导出中…」。

## 3. 前端

| 文件 | 变化 |
|------|------|
| `web/src/api/client.ts` | 新增 `apiDownload()`：POST JSON、解析 `Content-Disposition`、返回 Blob；错误仍抛 `ApiError` |
| `web/src/pages/SettingsImportExport.vue` | 占位页改为导出卡片（内容说明、下载、成功提示）与「导入未实现」说明 |
| `web/src/pages/SettingsDeleteAccount.vue` | 占位页改为密码 + 用户名确认，调用删除后 `signOut()` 并回首页 |
| `web/src/pages/SettingsIndex.vue` | 其它设置组新增「导入/导出数据」「删除账号」，说明同步更新 |

## 4. 验收

`go vet`、`go test ./...`（新增 3 个用例：导出内容隔离、注销流程、最后管理员保护）、`vue-tsc`、
`check:port`、`pnpm build` 全绿；真实二进制走查导出下载文件内容与注销后回到访客首页。证据见
HANDOVER §12Y.3。
