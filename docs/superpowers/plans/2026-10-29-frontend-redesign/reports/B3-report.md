# Task B3 报告：`i/update` 端点

## 实现内容

- 新增 `internal/httpapi/i_update.go`：`POST /api/i/update`，body `{ name?, description?, avatarId? }`（指针字段区分「未提供」与「清除」）。
- 消费 B2 的 `account.Service.UpdateProfile`：nil=未提供、空串=清除。
- 鉴权与错误风格参照现有 handler：未登录/坏 token → `writeAuthenticationError`（401）。
- avatarId 校验（`checkOwnAvatarFile`）：查 drive 文件，`File.UserID != 当前用户` → 403 `ACCESS_DENIED`（kind permission）；文件不存在 → 400 `NO_SUCH_FILE`；drive 未挂载 → 503。
- 响应体：`packMeDetailed`（含更新后的 name/description/avatarUrl，native token 请求含 secrets）+ `applyUserCounts`。
- `internal/httpapi/server.go` 注册路由 `POST /api/i/update`。

## RED → GREEN 证据

- RED：`go test ./internal/httpapi/ -run TestIUpdate -v` → 三个用例全部 FAIL（端点不存在，404 `NO_SUCH_ENDPOINT`；其中一个用例当时还暴露了测试内两次 `uniqueName(t)` 同名冲突，已改为加后缀的独立用户名）。
- GREEN：同命令 → `TestIUpdateRequiresCredential`、`TestIUpdateProfileFields`、`TestIUpdateAvatar` 全部 PASS。
- `go test ./internal/httpapi/` 全包 `ok`；`go vet ./...` 通过。

## 测试

`internal/httpapi/i_update_test.go`：
- 未认证 → 401。
- 更新 name/description 后：响应体、`api/i`、`users/show` 三处均返回新值；部分更新只动给定字段。
- avatarId：本人文件 → `avatarUrl` 等于文件 URL；他人文件 → 403；未知文件 → 4xx；空串 → `avatarUrl` 清为 nil。

## 文件变更

- 新增：`internal/httpapi/i_update.go`、`internal/httpapi/i_update_test.go`
- 修改：`internal/httpapi/server.go`（一行路由注册）

## 提交

`490b6ed feat(httpapi): add i/update endpoint`

## 顾虑

- 响应的 `avatarId` 字段仍为 null：schema 只有 `avatar_url` 列，没有存文件 id 的地方（不允许为它加迁移），前端如需 avatarId 需另行讨论。
- `NO_SUCH_FILE` 错误借用了 `errIDInvalidParam`（未引入新的编造 id）；Misskey 对该场景有专用的错误 id，可在有参考源码时对齐。
