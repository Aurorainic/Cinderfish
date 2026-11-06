# 第二十一波施工契约：账号安全操作（2026-11-06）

安全设置页在第二十波之后仍把「修改密码」「重新生成登录令牌」「登录历史」三项标为未提供。
本波补齐 `i/change-password`、`i/regenerate-token`、`i/signin-history`，让页面不再有死按钮。

## 1. 端点

| 端点 | 请求 | 响应 |
|------|------|------|
| `i/change-password` | `{currentPassword, newPassword}` | `204` |
| `i/regenerate-token` | `{password}` | `{token}` |
| `i/signin-history` | `{limit?, sinceId?, untilId?}` | `Signin[]` |

- 三个端点都需登录；当前密码错误统一返回 403 + id-only 信封 `932c904e…`。
- `change-password` 的 `newPassword` 为空返回 400 `INVALID_PARAM`，只替换 `users.password_hash`，
  不吊销既有 token（与 Misskey 一致）。
- `regenerate-token` 把 `users.token` 换成新的 16 字符 token。**与 Misskey 的差异**：额外回传
  `{token}`，因为本移植没有 `myTokenRegenerated` 流事件，浏览器需要立刻替换 localStorage 凭据；
  多余字段对 Misskey 客户端无影响。
- `signin-history` 按 `signins.id` 降序、`limit` 默认 10 上限 100；元素为
  `{id, createdAt, ip, headers:{}, success}`，`headers` 恒空（表里不存请求头）。

## 2. 账号层

`internal/account/security.go`（新）：`ChangePassword`、`RegenerateToken`、`SigninHistory`。
密码校验复用第二十波的 `CheckPassword`，哈希复用 `auth.HashPassword`（bcrypt cost 8）。

## 3. 前端

| 文件 | 变化 |
|------|------|
| `api/types.ts`、`endpoints.ts` | `SigninHistoryEntry` 与三个端点 |
| `pages/SettingsSecurity.vue` | 修改密码表单（当前/新/确认）、令牌重生成（成功后 `updateMe` + `addStoredAccount`）、登录历史列表与「加载更多」 |

常量沿用 Misskey/Firefish 的键：`currentPassword`、`newPassword`、`newPasswordRetype`、
`changePassword`、`regenerateLoginToken`、`signinHistory`。

## 4. 有意的限制

1. 修改密码不吊销其它会话的 token，也没有“退出所有设备”。
2. 没有密码强度校验（Misskey 只要求非空）。
3. 登录历史不记录请求头、User-Agent 或地理位置；`headers` 恒为 `{}`。
4. 令牌重生成的响应形状偏离 Misskey（多 `token` 字段），原因见上。
5. 登录失败限流仍是既有缺口。

## 5. 验收

`gofmt -l`、`go vet ./...`、`vue-tsc --noEmit`、`check:port`、`pnpm build` 通过。按用户
2026-11-06 的指示，本波未执行 `go test`、数据库测试与浏览器走查；改密/换令牌/历史分页的
端到端行为留待统一验证。
