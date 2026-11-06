# 第三十九波施工契约：邮箱相关前端页面（2026-11-06）

第三十八波把后端做齐了（六个端点 + SMTP + 验证码表），但 `SettingsEmail`、`VerifyEmail`、
`ResetPassword` 三个页面还是 17 行的 `TODO: port` 占位符，登录框也没有「忘记密码」入口。
本波把这三页换成真实实现。

## 1. 端点表

`web/src/api/endpoints.ts` 新增六个常量与 `EndpointMap` 条目：

| 常量 | 路径 | 请求 → 响应 |
|------|------|-------------|
| `emailAddressAvailable` | `email-address/available` | `{email}` → `EmailAddressAvailability` |
| `iUpdateEmail` | `i/update-email` | `{password,email}` → `MeDetailed` |
| `iEmailVerification` | `i/email-verification` | `{email?}` → 空 |
| `verifyEmail` | `verify-email` | `{code}` → 空 |
| `requestResetPassword` | `request-reset-password` | `{username,email}` → 空 |
| `resetPassword` | `reset-password` | `{token,password}` → 空 |

`EmailAddressAvailability`（`web/src/api/types.ts`）= `{available, reason?: "format" | "used"}`。

## 2. 页面

| 页面 | 行为 |
|------|------|
| `pages/SettingsEmail.vue` | 当前地址 + 已验证/未验证徽标；输入新地址 → 弹密码框 → `i/update-email` → 用返回值 `updateMe()` 就地刷新；「重新发送验证邮件」调 `i/email-verification` |
| `pages/VerifyEmail.vue` | 从路由 `:code` 调 `verify-email`；未登录先提示登录再重开链接；成功后 `updateMe({emailVerified:true})` 并给返回按钮 |
| `pages/ResetPassword.vue` | 有 `:token` → 新密码 + 确认 → `reset-password` → 回首页；无 token → 用户名 + 邮箱 → `request-reset-password`，提示文案是「如果匹配则已发送」 |
| `account/components/CfSignin.vue` | 登录按钮下方加「忘记密码」链接指向 `/reset-password` |

`unverified` 是新加的 i18n 键（zh-CN「未验证」/ ja-JP「未確認」）。

## 3. 有意的限制

1. 设置页的两个邮件通知开关（`receiveAnnouncementEmail`、`emailNotificationTypes`）不渲染：
   `i/update` 不接受这两个字段，`MeDetailed` 里它们是常量。页面写了一段说明而不是假开关。
2. `i/update-email` 成功但发信失败时后端返回 500 且地址已保存；页面把它当普通错误显示，
   用户点「重新发送验证邮件」即可。
3. 重置页不带 token 时是内联表单，不是 Firefish 的弹窗；两者信息量相同，内联少一个嵌套弹窗。
4. 验证页要求已登录——code 与账号绑定，未登录时硬调会 401。

## 4. 验收

```
cd web && pnpm exec vue-tsc --noEmit    # exit 0
cd web && pnpm run check:port           # hardTotal 0，missingKeys/missingHeader 空
cd web && pnpm build                    # exit 0
```

按用户 2026-11-06 的指示未执行浏览器走查与 `go test`。
