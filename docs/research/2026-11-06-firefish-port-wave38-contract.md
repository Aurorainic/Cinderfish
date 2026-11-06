# 第三十八波施工契约：邮件发送与邮箱验证（2026-11-06）

HANDOVER 里最大的未移植块是「邮箱」：密码重置、邮箱验证、`SettingsEmail`、
`AdminEmailSettings` 全部卡在同一件事上——既没有发信通道，也没有验证码存储。本波补这两层
和六个端点；前端页面留到第三十九波。

## 1. 配置

```toml
[mail]
host = ""            # 空 = 不投递，整封信打到日志（新实例与测试用）
port = 587           # tls="tls" 且未填时默认 465
username = ""
password = ""
from = ""            # 空 = no-reply@<domain>
tls = "starttls"     # starttls | tls | none
```

环境变量：`CINDERFISH_MAIL_{HOST,PORT,USERNAME,PASSWORD,FROM,TLS}`。`from` 的默认值在
`config.Load` 里按域名推导，因为 TOML 默认值无法引用其它字段。

## 2. `internal/mail`

| 符号 | 行为 |
|------|------|
| `Sender` | `Send(ctx, Message) error` |
| `New(Settings)` | `host` 为空返回 `LogSender`，否则返回 `*SMTPSender` |
| `LogSender` | 把 `to`/`subject`/正文写进 `log`，后续流程可继续 |
| `SMTPSender` | 10 秒拨号、30 秒整体超时；`starttls` 先升级（服务端不支持则明文继续）、`tls` 首字节即 TLS、`none` 给本地中继；`username` 非空才 AUTH PLAIN |

头部值统一去掉 CR/LF，避免主题或收件人注入额外头部；正文按 SMTP 要求转成 CRLF。

## 3. 数据模型（迁移 v18）

```sql
ALTER TABLE user_profiles ADD COLUMN email_verified BOOLEAN NOT NULL DEFAULT FALSE;

CREATE TABLE email_challenges (
    id         TEXT PRIMARY KEY,
    user_id    TEXT NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    purpose    TEXT NOT NULL,          -- 'verify' | 'reset'
    email      TEXT NOT NULL,
    code       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    used_at    TIMESTAMPTZ
);
```

`code` 是 32 字节随机数的十六进制（64 字符），唯一索引兜底；`purpose` 让验证链接与重置链接
互不通用。消费用一条条件 UPDATE（`used_at IS NULL AND expires_at > now()`）返回行，天然防重放
且并发安全。

## 4. 端点

| 端点 | 认证 | 行为 |
|------|------|------|
| `email-address/available` | 匿名 | `{email}` → `{available, reason?}`，`reason` 为 `format`/`used`，从不报错 |
| `i/update-email` | 登录 | `{password,email}`：校验密码（错密码 403 + `932c904e…`）、格式、占用；写入并清 `email_verified`，发 `verify` 链接，返回 `MeDetailed` |
| `i/email-verification` | 登录 | `{email?}`：给登记地址重发 `verify` 链接；未登记 400；与登记值不符 400；60 秒节流 429 |
| `verify-email` | 登录 | `{code}`：消费 `verify` 链接并置 `email_verified`；失效/他人/地址已变更均 400 `INVALID_EMAIL_CODE` |
| `request-reset-password` | 匿名 | `{username,email}`：恒 204，不暴露账号是否存在；命中时发 `reset` 链接（24 小时） |
| `reset-password` | 匿名 | `{token,password}`：换密码并置 `email_verified`；失效 400 `INVALID_TOKEN` |

链接形如 `<实例根 URL>/verify-email/<code>` 与 `<实例根 URL>/reset-password/<code>`，根 URL 复用
`s.baseURL()`（与 ActivityPub 标识同源，`federation.public_url` 生效）。

`MeDetailed.email` / `emailVerified` 由 `account.Profile` 携带，不再恒为 `null` / `false`。

## 5. 有意的限制

1. 注册仍不收邮箱（`emailRequiredForSignup` 恒 `false`），邮箱只能登录后补填。
2. 邮件正文是固定中文纯文本，没有 HTML 与多语言模板。
3. 限流只有「同一账号同一用途 60 秒一封」，没有按 IP 配额。
4. 重置密码不吊销已签发的 access token，与 `i/change-password` 一致。
5. 发信失败时 `i/update-email` 保留已保存的地址并返回 500，不自动回滚。

## 6. 验收

`gofmt -l`、`go vet ./...` 通过。前端本波未改动，`vue-tsc`/`check:port`/`pnpm build` 不适用。
按用户 2026-11-06 的指示未执行 `go test`、数据库测试与浏览器走查。
