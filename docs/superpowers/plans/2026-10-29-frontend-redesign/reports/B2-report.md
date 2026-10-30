# Task B2 报告：用户资料打包（name / description / avatarUrl）

## 实现内容

- 新增 `internal/account/profile.go`：
  - `Profile{Name, Description, AvatarURL string}`（空串表示 NULL）；
  - `GetProfile(ctx, userID)`：无 profile 行时返回零值而非报错；
  - `UpdateProfile(ctx, userID, name, description, avatarFileID *string)`：nil 表示未提供（保持不变），非 nil 空串表示清除；`avatarFileID` 非空时经 drive 解析 URL 写入 `avatar_url`，查不到文件返回 `ErrNoSuchAvatarFile`。动态 UPDATE 的 SET 子句与参数按序一起收集（AGENTS.md 规则）。
- `account.Service` 新增 `drive` 字段与 `AttachDrive(*media.DriveService)`；`httpapi.Server.SetStorage` 里把 drive 挂到 accounts。
- `internal/httpapi/users_pack.go`：
  - `packUserLite(u, host, prof *account.Profile)`：`prof` 非 nil 时填 `name`/`avatarUrl`；`u.Host` 非空时填 `host` 字段（原硬编码 nil）；
  - `packUserDetailed(u, createdAt, isMe, prof)`：新增 `description`（由调用方传入 profile）；
  - `packMeDetailed` 同步加 `prof` 参数。
- 全部调用点更新（grep 逐一）：`account.go`（signup/i/users/show，新增 `loadProfile` 辅助）、`admin_accounts.go`（两处）、`notes_pack.go`（`notePackContext` 随 author 一起取 profile）、`streaming_pack.go`、`notifications.go`、`following.go`（传 nil）。
- 无迁移（user_profiles 列在 v1/v4 已存在，符合预期）。

## RED → GREEN 证据

- RED：`go test ./internal/account/` → `[build failed]`，`profile_test.go` 报 `GetProfile undefined`、`UpdateProfile undefined`、`account.Profile undefined` 等（未实现时编译失败即测试失败）。
- GREEN：`CINDERFISH_TEST_DATABASE_URL=... go test ./internal/account/ ./internal/httpapi/` → 两个包均 `ok`（account 0.522s、httpapi 7.097s）。
- `go vet ./...` 通过。

## 测试

`internal/account/profile_test.go`：
- `TestGetProfileDefaults`：新注册用户 profile 为零值；删除 profile 行后仍返回零值不报错。
- `TestUpdateProfileTextFields`：写 name/description；部分更新（nil 保持）；空串清除。
- `TestUpdateProfileAvatar`：挂 drive 后按文件 id 解析 URL；空串清除；合并更新保持其它字段；未知文件 id 报错；他人 profile 未被改动。

## 文件变更

- 新增：`internal/account/profile.go`、`internal/account/profile_test.go`
- 修改：`internal/account/service.go`、`internal/httpapi/users_pack.go`、`account.go`、`admin_accounts.go`、`notes_pack.go`、`streaming_pack.go`、`notifications.go`、`following.go`、`server.go`

## 提交

`7a21e1f feat(account,httpapi): pack user profile (name/description/avatarUrl)`

## 顾虑

- `packUserLite` 的 `host` 参数继续保留但实际 `host` 字段改由 `u.Host` 驱动（按 brief 要求）；调用方传入的 `s.cfg.Domain` 未使用。
- `notifications.go`/`streaming_pack.go`/`following.go` 的 `packUserLite` 传 `nil` profile，这些场景的显示名/头像暂为 nil，brief 未要求查询。
- account 包现在依赖 `internal/media`（仅用于 avatar URL 解析），无循环依赖（media 只依赖 idgen/pgx）。
