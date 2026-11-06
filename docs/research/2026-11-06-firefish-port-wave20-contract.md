# 第二十波施工契约：两步验证（TOTP）（2026-11-06）

安全设置页此前把 2FA 标为「未提供」，登录也只有密码一步。本波实现 TOTP 的两步验证：
生成密钥、确认启用、登录时校验验证码、停用，并把状态暴露给登录框。

## 1. 数据模型（迁移 v14）

`user_profiles` 新增四列：

| 列 | 类型 | 说明 |
|----|------|------|
| `two_factor_secret` | TEXT NULL | 已生效的 base32 密钥 |
| `two_factor_temp_secret` | TEXT NULL | 注册中、尚未确认的密钥 |
| `two_factor_enabled` | BOOLEAN NOT NULL DEFAULT FALSE | 登录是否走第二因子 |
| `use_password_less_login` | BOOLEAN NOT NULL DEFAULT FALSE | 仅为 schema 对齐；安全密钥未实现，恒 false |

## 2. TOTP 实现

`internal/auth/totp.go`，不引入第三方依赖：

- 密钥：20 字节随机数，base32（无 padding）；
- 算法：HMAC-SHA1、6 位、30 秒步长，动态截断；
- 校验：接受当前步长与前后各一步（`window = 1`），忽略空格；
- `otpauth://` URL 带 `issuer`、`algorithm=SHA1`、`digits=6`、`period=30`。

## 3. 端点

| 端点 | 请求 | 响应 |
|------|------|------|
| `i/2fa/register` | `{password}` | `{qr:"",url,secret,label,issuer}` |
| `i/2fa/done` | `{token}` | `204`；错误 `TWO_FACTOR_NOT_STARTED` / `TOTP_TOKEN_INVALID` |
| `i/2fa/unregister` | `{password}` | `204` |
| `i/2fa/password-less` | `{value}` | `value:true` → `400 NO_SECURITY_KEY`；`false` → `204` |

`register`/`unregister` 的密码错误沿用登录的 id-only 信封 `932c904e…`。`done` 成功时把临时密钥
提升为正式密钥并清空临时列。

## 4. 登录协商

`POST /api/signin` 的请求体新增可选 `token`：

1. 密码正确且账号启用了 TOTP：不带 `token` 时返回 `200 {}`，不签发 token；
2. 带 `token` 时校验，错误返回 `403` + id `cdf1235b-ac71-46d4-a3a6-84ccce48df6f`，
   并写一行失败登录记录；
3. 通过后返回原来的 `{finished:true,id,i}`。

未启用 TOTP 的账号行为完全不变。

## 5. 打包字段

`packUserDetailed` 追加 `twoFactorEnabled`、`usePasswordLessLogin`、`securityKeys`（恒 false）。
登录框通过 `users/show` 读 `twoFactorEnabled` 预判是否需要验证码；`users/show` 因此对匿名请求
也会暴露某账号是否启用了两步验证，与 Firefish/Misskey 一致。

## 6. 前端

| 文件 | 变化 |
|------|------|
| `api/types.ts`、`endpoints.ts` | `UserDetailed` 三个字段、`SigninStep`、`TwoFactorRegistration`、四个端点 |
| `account/session.ts` | `signIn(username,password,token?)` 返回 `{account,token}` 或 `{twoFactorRequired:true}` |
| `account/transport.ts` | 识别验证码错误 id |
| `account/components/CfSignin.vue` | 两步表单：输入用户名后按 `users/show` 预判，或首次提交后切换到验证码输入 |
| `pages/SettingsAccounts.vue` | 添加账号时支持第二步验证码 |
| `pages/SettingsSecurity.vue` | 真实 TOTP 面板：密码确认 → 展示密钥/URL → 输入验证码启用；停用需再输密码 |

## 7. 有意的限制

1. `qr` 恒为空串：不引入二维码编码依赖，页面展示密钥与 `otpauth://` URL 供手动录入。
2. 不实现安全密钥/通行密钥（`i/2fa/register-key`、`key-done`、`remove-key`、`update-key`），
   因此 `password-less` 恒不可启用，`securityKeys` 恒 false。
3. 没有备份码，`twoFactorBackupCodesStock` 仍为 `"none"`。
4. 密钥明文存库（与 Firefish 相同）。
5. 没有登录失败限流（这是既有缺口，不在本波范围）。

## 8. 验收

`gofmt -l`、`go vet ./...`、`vue-tsc --noEmit`、`pnpm build` 通过。按用户 2026-11-06 的指示，
本波**未执行** `go test`、数据库测试与浏览器走查；端到端登录链路与设置页交互留待统一验证。
