# API 对接契约：auth-users

本文记录 Cinderfish 后端 `auth-users` 命名空间的真实 HTTP 契约，供重写 Firefish 风格前端时直接对接。

**证据口径**：所有请求字段、响应字段、错误码、HTTP 状态码都从本仓库代码读出（引用 `internal/...:行号`），不以 Misskey 官方文档为准；与 Misskey 的差异处会单独标注。核对过的上游参考为 `tmp/refcheck/misskey`（`c05e844`）与 `tmp/refcheck/firefish`（`02c31e0`），仅用于"差异"小节，不用于推断本实现行为。

覆盖端点：`signup`、`signin`、`username/available`、`users/stats`、`i/registry/keys-with-type`、`email-address/available`、`i/update-email`、`i/email-verification`、`verify-email`、`request-reset-password`、`reset-password`、`miauth/gen-token`、`i`、`i/update`（含 `users/update` 别名）、`i/2fa/*`、`i/change-password`、`i/regenerate-token`、`i/signin-history`、`users/show`、`users/notes`、`users/followers`、`users/following`、`i/pin`、`i/unpin`，以及 OAuth 的三个端点。个人资料打包（`UserLite` / `UserDetailed` / `MeDetailed`）与 `users/notes` 用到的 `Note` 打包也在此文档化。

---

## 端点总览

| 方法 | 路径 | 认证 | Handler | 说明 |
|------|------|------|---------|------|
| POST | `/api/signup` | 匿名 | `internal/httpapi/account.go:37` | 注册；成功返回 `MeDetailed` + `token` |
| POST | `/api/signin` | 匿名 | `internal/httpapi/account.go:114` | 密码登录；启用 TOTP 时先返回空响应，带 `token` 再来一次 |
| POST | `/api/i/2fa/register` | 需登录 | `internal/httpapi/two_factor.go:22` | 校验密码后生成临时 TOTP 密钥 |
| POST | `/api/i/2fa/done` | 需登录 | `internal/httpapi/two_factor.go:70` | 用 6 位码确认并启用 TOTP |
| POST | `/api/i/2fa/unregister` | 需登录 | `internal/httpapi/two_factor.go:117` | 校验密码后停用 TOTP |
| POST | `/api/i/2fa/password-less` | 需登录 | `internal/httpapi/two_factor.go:150` | 无安全密钥，`value:true` 恒返回 `NO_SECURITY_KEY` |
| POST | `/api/i/change-password` | 需登录 | `internal/httpapi/security.go:18` | 校验当前密码后换新密码；成功 204 |
| POST | `/api/i/regenerate-token` | 需登录 | `internal/httpapi/security.go:57` | 校验密码后换 16 字符原生 token |
| POST | `/api/i/signin-history` | 需登录 | `internal/httpapi/security.go:95` | 登录历史，id 游标分页 |
| POST | `/api/username/available` | 匿名 | `internal/httpapi/account.go:174` | 用户名是否可注册 |
| POST | `/api/email-address/available` | 匿名 | `internal/httpapi/email.go:245` | 邮箱是否可用；`{available, reason?}` |
| POST | `/api/i/update-email` | 需登录 | `internal/httpapi/email.go:175` | 校验密码后换邮箱并发送验证链接 |
| POST | `/api/i/email-verification` | 需登录 | `internal/httpapi/email.go:54` | 给已登记邮箱重发验证链接 |
| POST | `/api/verify-email` | 需登录 | `internal/httpapi/email.go:104` | 用邮件里的 code 标记邮箱已验证 |
| POST | `/api/request-reset-password` | 匿名 | `internal/httpapi/email.go:281` | 发送密码重置链接；恒 204 |
| POST | `/api/reset-password` | 匿名 | `internal/httpapi/email.go:336` | 用重置链接里的 token 设置新密码 |
| POST | `/api/miauth/gen-token` | 需登录 | `internal/httpapi/account.go:290` | 为 MiAuth 会话签发 32 字符 token |
| POST | `/api/i` | 需登录 | `internal/httpapi/account.go:200` | 当前用户 `MeDetailed` |
| POST | `/api/i/update` | 需登录 | `internal/httpapi/i_update.go:66` | 修改 name / description / avatar / 两个可见性开关 |
| POST | `/api/users/update` | 需登录 | 同上（`internal/httpapi/server.go:119`） | `i/update` 的别名，行为完全相同 |
| POST | `/api/users/show` | 可选登录 | `internal/httpapi/account.go:220` | 用户详情；带 token 时补关系标志 |
| POST | `/api/users/notes` | 可选登录 | `internal/httpapi/users.go:24` | 某用户的 public/home 笔记列表 |
| POST | `/api/users/stats` | 匿名可 | `internal/httpapi/users.go:313` | 某用户的七项计数；`userId` 缺省时取调用者 |
| POST | `/api/users/followers` | 可选登录 | `internal/httpapi/users.go:104` | 粉丝列表（`Following` 信封） |
| POST | `/api/users/following` | 可选登录 | `internal/httpapi/users.go:110` | 关注列表（`Following` 信封） |
| POST | `/api/i/pin` | 需登录 | `internal/httpapi/i_pin.go:17` | 置顶自己的笔记（上限 5） |
| POST | `/api/i/unpin` | 需登录 | `internal/httpapi/i_pin.go:25` | 取消置顶 |
| GET | `/.well-known/oauth-authorization-server` | 匿名 | `internal/httpapi/oauth.go:54` | RFC 8414 元数据 |
| GET | `/oauth/authorize` | 需登录（`Authorization` 头） | `internal/httpapi/oauth.go:63` | 授权码端点（PKCE S256） |
| POST | `/oauth/token` | 匿名 | `internal/httpapi/oauth.go:139` | 用 `code` + `code_verifier` 换 access token |

路由注册位置：`internal/httpapi/server.go:112-126`（账号/用户）与 `internal/httpapi/server.go:159-161`（OAuth）。本命名空间**没有**需要管理员的端点（`requireAdmin` 仅用于 `admin/*`，见 `internal/httpapi/auth.go:74`）。

---

## 通用约定

### 请求形态

- 除 OAuth 三端点外，全部是 `POST` + JSON body。`Content-Type` **不做校验**：任何类型只要 body 是合法 JSON 就会被解析（`internal/httpapi/ping.go:23`）。
- 空 body 等价于 `{}`，不是错误（`internal/httpapi/ping.go:29`）。
- body 上限 8 MiB（`io.LimitReader(r.Body, 8<<20)`）。
- body 不是合法 JSON 时返回 `400 INVALID_PARAM`，`info` 为 `{"reason":"body is not valid JSON"}`（`internal/httpapi/ping.go:32`）。
- 未注册的路径、以及**方法不匹配**的已注册路径，都落到兜底路由 `"/"`（`internal/httpapi/server.go:223`），返回 `404 NO_SUCH_ENDPOINT` JSON，而不是 405。例如 `GET /api/i` 得到 404，而不是 405。
- 成功响应是**裸值**（对象或数组），没有 `{"result": ...}` 之类包装；响应体以换行结尾，`Content-Type: application/json; charset=utf-8`（`internal/httpapi/server.go:235`）。

### 认证（token 读取顺序）

`authenticate` 的读取顺序（`internal/httpapi/auth.go:30`）：

1. body 里的 `i` 字段（JSON string），优先级最高；
2. 否则读 `Authorization: Bearer <token>`（大小写不敏感地匹配 `bearer ` 前缀）；
3. 两者都没有 → 视为匿名（返回 `nil, nil`，由各 handler 决定是否强制）。

token 解析（`internal/account/tokens.go:88`）：

- 长度为 **16** 的 token 一律按"用户原生 token"到 `users.token` 里查，查不到即无效（`internal/auth/auth.go:112`）；
- 其它长度按 `access_tokens.token` 精确匹配（MiAuth token 32 字符、OAuth token 128 字符），命中后更新 `last_used_at`；
- 命中后若用户 `is_suspended = TRUE`，认证一律失败（`internal/httpapi/auth.go:53`），表现与无效 token 相同。

失败形态（`internal/httpapi/server.go:304`）：

- HTTP 401，`WWW-Authenticate: Bearer realm="Misskey", error="invalid_token", error_description="Authentication failed. Please ensure your token is correct."`
- body `{"error":{"message":"Authentication failed. Please ensure your token is correct.","code":"AUTHENTICATION_FAILED","id":"b0a7f5f8-dc2f-4171-b91f-de88ad238e14","kind":"client"}}`

**scope 不生效**：除 `/streaming` 会检查 app token 是否含 `read:account`（`internal/httpapi/streaming_adapters.go:24`）外，任何 `/api/*` 端点都不检查 scope；只要 token 有效就拥有账号的全部权限。

### 错误 JSON 形态

统一信封（`internal/httpapi/server.go:269`）：

```json
{
  "error": {
    "message": "Invalid param.",
    "code": "INVALID_PARAM",
    "id": "3d81ceae-475f-4600-b2a8-2bc116157532",
    "kind": "client",
    "info": { "...": "只有部分错误带 info" }
  }
}
```

- `kind` 恒存在，取值 `client` / `server` / `permission`。
- `id` 是 Misskey 的错误 UUID，客户端可据此分支（Firefish 就这么做，见 `packages/client/src/components/MkSignin.vue` 的 `loginFailed` 按 `err.id` 分支）。
- 本命名空间出现过的 `code`：`INVALID_PARAM`、`AUTHENTICATION_FAILED`、`NO_SUCH_USER`、`NO_SUCH_NOTE`、`NO_SUCH_FILE`、`ACCESS_DENIED`、`FORBIDDEN`、`INTERNAL_ERROR`、`PIN_LIMIT_EXCEEDED`、`DUPLICATED_USERNAME`、`INVALID_INVITE`、`INVALID_USERNAME`、`INVALID_PASSWORD`、`REGISTRATION_DISABLED`、`SIGNUP_FAILED`、`SIGNUP_UNAVAILABLE`、`SIGNIN_FAILED`、`SIGNIN_UNAVAILABLE`、`NO_SUCH_ENDPOINT`。

三个例外：

1. **`signin` 失败**用 Misskey 的 id-only 信封：`{"error":{"id":"..."}}`，**没有** `message`/`code`/`kind`（`internal/httpapi/account.go:22`）。
2. **OAuth 端点**用 RFC 6749 形态：`{"error":"invalid_request","error_description":"...","state":""}`，`state` 恒为字符串（未传时是空串），不是 `error` 对象（`internal/httpapi/oauth.go:250`）。
3. `/oauth/authorize` 未登录时返回的是 **HTML** 页面 + 401，不是 JSON（`internal/httpapi/oauth.go:97`）。

### 分页游标语义

所有列表端点都用同一套游标，不使用 offset：

- 游标字段：`sinceId`、`untilId`，值都是 18 字符 base36 的时间可排序 id（`internal/idgen/idgen.go`：10 位毫秒时间戳 + 8 位随机）。
- 默认方向：`ORDER BY id DESC`（新→旧）。
- 只给 `sinceId`、不给 `untilId` 时，**关注列表**（`users/followers` / `users/following`）会把方向翻转为 `ORDER BY id ASC`（"向上翻页"，返回比游标更新的最早几条），这是 Misskey `makePaginationQuery` 的语义（`internal/account/follows.go:96-104`）；**`users/notes` 不翻转**，永远是 `DESC`（`internal/note/timeline.go:270`）。
- `limit` 默认值与上限**各端点不同**，见下表；超出上限时的处理也不统一（`users/notes` 是**重置**为默认值，不是钳制）。

| 端点 | 游标主体 | `limit` 默认 | `limit` 上限 | 越界处理 |
|------|----------|--------------|--------------|----------|
| `users/notes` | `notes.id` | 20（`limit <= 0`） | 100 | `limit > 100` 时**重置为 20**（`internal/note/timeline.go:251`） |
| `users/followers` / `users/following` | **`followings.id`（关注行 id，不是用户 id）** | 10 | 100 | 超过 100 钳制为 100（`internal/account/follows.go:67-72`） |

### 时间与类型约定

- 用户 `createdAt`：`time.Time.UTC().Format(time.RFC3339Nano)` 字符串，例如 `2026-11-04T08:12:33.123456789Z`（`internal/httpapi/users_pack.go:52`）。注意 RFC3339Nano 会省略小数部分末尾的 0，位数不固定，前端按 RFC3339 解析而不是按固定长度截取。
- `Following.createdAt`：同上。
- 驱动文件 `createdAt`：固定毫秒三位格式 `2006-01-02T15:04:05.000Z`（`internal/httpapi/drive.go:252`）。
- 投票 `expiresAt`：**Unix 毫秒整数**，无期限时为 `null`（`internal/httpapi/notes_pack.go:191`）。
- 缺失的可选字符串字段通常为 `null`（`name`、`description`、`text`、`cw` 等），但 `url`/`uri` 在 `UserDetailed` 里是**空字符串**而不是 `null`（见下）。

---

## 共享对象

### `UserLite`

由 `packUserLite` 生成（`internal/httpapi/users_pack.go:12`）。出现在 `Note.user`、`Following.follower/followee`、以及 `UserDetailed`/`MeDetailed` 的全部基础字段。

| 字段 | 类型 | 实际取值 |
|------|------|----------|
| `id` | string | 用户 id（18 字符） |
| `name` | string \| null | 资料里的显示名；空则 `null` |
| `username` | string | 用户名（保留大小写） |
| `host` | string \| null | 本地用户为 `null`，远端为用户域名 |
| `avatarUrl` | string \| null | 来自 `user_profiles.avatar_url`；未设置为 `null` |
| `avatarBlurhash` | null | 恒为 `null`（未实现） |
| `avatarDecorations` | array | 恒为 `[]` |
| `isBot` | boolean | 恒为 `false`（无视 `users.is_bot`） |
| `isCat` | boolean | 恒为 `false` |
| `requireSigninToViewContents` | boolean | 恒为 `false` |
| `emojis` | array | 恒为 `[]` |
| `onlineStatus` | string | 恒为 `"unknown"` |
| `badgeRoles` | array | 恒为 `[]` |
| `instance` | null | 恒为 `null` |

### `UserDetailed`

由 `packUserDetailed` 生成（`internal/httpapi/users_pack.go:46`），在 `UserLite` 全部字段之上追加：

| 字段 | 类型 | 实际取值 |
|------|------|----------|
| `url` | string | 恒为 `""`（**空串**，不是可点击的远端主页地址） |
| `uri` | string | 恒为 `""`（未返回 ActivityPub actor id） |
| `movedTo` | null | 恒为 `null` |
| `alsoKnownAs` | null | 恒为 `null` |
| `createdAt` | string | RFC3339Nano |
| `updatedAt` | null | 恒为 `null` |
| `lastFetchedAt` | null | 恒为 `null` |
| `bannerUrl` | null | 恒为 `null`，即使数据库有 `user_profiles.banner_url` 也不读（`internal/account/profile.go:41`） |
| `bannerBlurhash` | null | 恒为 `null` |
| `isLocked` | boolean | `users.is_locked`（第四十七波起读取真值；`i/update` 可写） |
| `isModerator` | boolean | **等于 `is_admin`**（无独立 moderator 角色） |
| `isAdmin` | boolean | `users.is_admin` |
| `isSilenced` | boolean | 恒为 `false` |
| `isSuspended` | boolean | `users.is_suspended` |
| `description` | string \| null | 资料简介；空则 `null` |
| `location` | string \| null | `settings.location`；空则 `null` |
| `birthday` | string \| null | `settings.birthday`（`YYYY-MM-DD`）；空则 `null` |
| `lang` | string \| null | `settings.lang`；空则 `null` |
| `fields` | array | `settings.fields`，元素 `{name,value}`；默认 `[]` |
| `verifiedLinks` | array | 恒为 `[]` |
| `followersCount` | number | 来自 `users.followers_count`（由关注流程重算，`internal/federation/store.go:542`）；无 DB 时保持 0 |
| `followingCount` | number | 同上 |
| `notesCount` | number | 实时 `count(*)`：仅 `deleted_at IS NULL AND local_only = FALSE AND visibility IN ('public','home')`（`internal/httpapi/users.go:299`） |
| `pinnedNoteIds` | string[] | 置顶笔记 id，最多 5 条，按置顶时间倒序；默认 `[]` |
| `pinnedNotes` | Note[] | 上述 id 打包后的完整笔记；加载失败的单条会被跳过；默认 `[]` |
| `pinnedPageId` | null | 恒为 `null` |
| `pinnedPage` | null | 恒为 `null` |
| `publicReactions` | boolean | `settings.publicReactions`，默认 `false` |
| `followersVisibility` | string | `"public"` / `"followers"` / `"private"`；无资料行或空值读作 `"public"`（`internal/httpapi/users_pack.go:98`） |
| `followingVisibility` | string | 同上 |
| `twoFactorEnabled` | boolean | `user_profiles.two_factor_enabled`；登录框据此决定是否显示验证码输入框 |
| `usePasswordLessLogin` | boolean | `user_profiles.use_password_less_login`；安全密钥未实现，始终为 `false` |
| `securityKeys` | boolean | 恒为 `false`（没有可注册的安全密钥） |
| `chatScope` | string | 恒为 `"followers"` |
| `canChat` | boolean | 恒为 `false` |
| `roles` | array | 恒为 `[]`（无角色系统） |
| `memo` | null | 恒为 `null` |
| `isFollowing` | boolean | 默认 `false`；仅 `users/show` 带 token 且非本人时填充 |
| `isFollowed` | boolean | 默认 `false`；同上 |
| `hasPendingFollowRequestFromYou` | boolean | 默认 `false`；同上 |
| `hasPendingFollowRequestToYou` | boolean | 默认 `false`；同上 |

**重要**：`pinnedNoteIds`/`pinnedNotes`/`followersCount`/`followingCount`/`notesCount` 的"填充"由调用方完成（`applyUserCounts` `internal/httpapi/users.go:287`、`applyPinnedNotes` `internal/httpapi/users.go:256`）。不同端点是否填充见各自小节。

### `MeDetailed`

由 `packMeDetailed` 生成（`internal/httpapi/users_pack.go:116`），在 `UserDetailed` 全部字段之上追加（下面这些都是**常量**，没有真实存储）：

| 字段 | 类型 | 实际取值 |
|------|------|----------|
| `avatarId` | null | 恒为 `null`（**即使刚用 `avatarId` 设过头像**） |
| `bannerId` | null | 恒为 `null` |
| `followedMessage` | null | 恒为 `null` |
| `injectFeaturedNote` | boolean | `settings.injectFeaturedNote`，默认 `false` |
| `receiveAnnouncementEmail` | boolean | `settings.receiveAnnouncementEmail`（第四十八波起可写） |
| `alwaysMarkNsfw` | boolean | `settings.alwaysMarkNsfw`，默认 `false` |
| `autoSensitive` | boolean | `settings.autoSensitive`，默认 `false` |
| `carefulBot` | boolean | `settings.carefulBot`，默认 `false` |
| `autoAcceptFollowed` | boolean | `settings.autoAcceptFollowed`，默认 `false` |
| `followApprovalLocalSeconds` | number | `0` |
| `followApprovalRemoteSeconds` | number | `0` |
| `noCrawle` | boolean | `settings.noCrawle`，默认 `false` |
| `preventAiLearning` | boolean | `settings.preventAiLearning`，默认 `false` |
| `isExplorable` | boolean | `settings.isExplorable`，缺省 `true` |
| `isDeleted` | boolean | `false` |
| `twoFactorBackupCodesStock` | string | `"none"` |
| `hideOnlineStatus` | boolean | `settings.hideOnlineStatus`，默认 `false` |
| `hasUnreadSpecifiedNotes` | boolean | `false` |
| `hasUnreadMentions` | boolean | `false` |
| `hasUnreadChatMessages` | boolean | `false` |
| `hasUnreadAnnouncement` | boolean | `false` |
| `unreadAnnouncements` | array | `[]` |
| `hasUnreadAntenna` | boolean | `false` |
| `hasUnreadChannel` | boolean | `false` |
| `hasUnreadNotification` | boolean | 默认 `false`；`i` 与 `i/update` 会用真实未读数覆盖（`internal/httpapi/notifications.go:28`） |
| `hasPendingReceivedFollowRequest` | boolean | 恒为 `false`（未接入） |
| `unreadNotificationsCount` | number | 默认 `0`；`i` 与 `i/update` 会覆盖 |
| `mutedWords` | array | `[]` |
| `hardMutedWords` | array | `[]` |
| `mutedInstances` | array | `[]` |
| `mutingNotificationTypes` | array | `settings.mutingNotificationTypes`，默认 `[]`；只接受 mention/reply/renote/quote/reaction |
| `notificationRecieveConfig` | object | `{}`（字段名沿用 Misskey 的拼写错误 `Recieve`） |
| `emailNotificationTypes` | array | `settings.emailNotificationTypes`（第四十八波起可写，取值 mention / reply / follow / receiveFollowRequest） |
| `achievements` | array | `[]` |
| `loggedInDays` | number | `0` |
| `policies` | object | `{}` |

只有 `includeSecrets = true` 时**才出现**的字段：

| 字段 | 类型 | 实际取值 |
|------|------|----------|
| `email` | string \| null | `user_profiles.email`；未设置时为 `null`（2026-11-06 第三十八波起读取真实值） |
| `emailVerified` | boolean | `user_profiles.email_verified`；改邮箱后重置为 `false` |
| `securityKeysList` | array | 恒为 `[]` |

`includeSecrets` 的判定在各 handler 里：`i`、`i/update`、`i/pin`、`i/unpin` 用 `info.Token == nil`（即请求方用的是 16 字符原生 token），`signup` 恒为 `true`（`internal/httpapi/account.go:85`）。**用 MiAuth/OAuth token 调 `i` 时不会返回 `email` 字段。**

### `Note`（`users/notes` 的元素）

`packNote` + `packNoteDetail`（`internal/httpapi/notes_pack.go:28`、`:78`）。`users/notes` 返回的是**详情版**。

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | 笔记 id |
| `createdAt` | string | RFC3339Nano |
| `userId` | string | 作者 id |
| `user` | UserLite | 作者；作者加载失败时退化为 `{"id":"<userId>"}`（`internal/httpapi/notes_pack.go:160`） |
| `text` | string \| null | 正文 |
| `cw` | string \| null | 内容警告 |
| `visibility` | string | `public` / `home` / `followers` / `specified` |
| `localOnly` | boolean | 是否仅本地 |
| `renoteCount` | number | 转载数 |
| `repliesCount` | number | 回复数（存量计数，非实时） |
| `reactionCount` | number | **实时**统计的反应总数 |
| `reactions` | object | `{ "<emoji>": <count> }`，实时统计；无反应时为 `{}` |
| `reactionEmojis` | array | 恒为 `[]` |
| `emojis` | array | 恒为 `[]`（忽略笔记里的自定义表情） |
| `fileIds` | string[] | 附件 id |
| `files` | DriveFile[] | 附件对象；`null`/缺失时该元素被跳过 |
| `replyId` | string \| null | 父笔记 id |
| `renoteId` | string \| null | 被转载笔记 id |
| `mentions` | string[] | 恒为数组（不会 `null`） |
| `tags` | string[] | 恒为数组 |
| `uri` | string \| null | 远端笔记的 ActivityPub id；本地笔记为 `null` |
| `url` | string \| null | 远端笔记的原始 URL；本地为 `null` |
| `visibleUserIds` | string[] | **仅当** `visibility == "specified"` 时出现 |
| `reactionAcceptance` | string | **仅在数据库有值**时出现（如 `likeOnly`、`nonSensitiveOnly`） |
| `clippedCount` | number | 详情字段，恒为 0（无 clips） |
| `reply` | Note \| null | **仅在 `replyId != null` 时出现**；父笔记不可见/已删时为 `null`；内嵌对象里 `replyId`、`renoteId` 被置为 `null` |
| `renote` | Note \| null | **仅在 `renoteId != null` 时出现**；不可见/已删时为 `null` |
| `hasPoll` | boolean | **仅在笔记有投票时出现**，值为 `true` |
| `poll` | object | **仅在笔记有投票时出现**，见下 |
| `myReaction` | string | **仅当**带 token 且查看者自己反应过时出现 |

`poll` 对象（`internal/httpapi/notes_pack.go:135`）：

| 字段 | 类型 | 说明 |
|------|------|------|
| `multiple` | boolean | 是否多选 |
| `expiresAt` | number \| null | Unix 毫秒 |
| `choices` | array | 每项 `{"text": string, "votes": number, "isVoted": boolean}` |
| `votesCount` | number | 总票数 |

### `DriveFile`（`Note.files` 的元素）

`packDriveFile`（`internal/httpapi/drive.go:249`）：

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | 文件 id |
| `createdAt` | string | `YYYY-MM-DDTHH:MM:SS.mmmZ` |
| `name` | string | 文件名 |
| `type` | string | MIME |
| `size` | number | 字节 |
| `isSensitive` | boolean | NSFW 标记 |
| `url` | string | 可直接访问的 URL（本地为 `/media/...`） |
| `thumbnailUrl` | null | 恒为 `null`（无缩略图生成） |
| `webpublicUrl` | null | 恒为 `null` |
| `comment` | null | 恒为 `null` |
| `folderId` | string \| null | 所属文件夹 |
| `blurhash` | null | 恒为 `null` |

### `Following`（`users/followers` / `users/following` 的元素）

不是裸用户数组，而是关注关系对象（`internal/httpapi/users.go:201`）：

| 字段 | 类型 | 说明 |
|------|------|------|
| `id` | string | **关注行 id**（`followings.id`），分页游标就是它 |
| `createdAt` | string | 关注时间，RFC3339Nano |
| `followerId` | string | 关注者 id |
| `followeeId` | string | 被关注者 id |
| `follower` | UserLite | **仅 `users/followers` 返回**：这行的关注者 |
| `followee` | UserLite | **仅 `users/following` 返回**：这行的被关注者 |

也就是说 `users/followers` 的每一项只有一个 `follower`（没有 `followee`），`users/following` 的每一项只有一个 `followee`（没有 `follower`）。前端不要假设两者同时在。

---

## POST /api/signup

注册本地账号。认证：**匿名**（不需要 token）。注册策略由配置 `RegistrationMode` 决定（`internal/httpapi/account.go:105`）：`open` 直接建号；`invite` 必须带可用邀请码；`admin` 拒绝自助注册（只有 `admin/accounts/create` 能建号）。配置里出现其它值时按 `admin` 处理。

### 请求字段

| 字段 | 类型 | 必填 | 默认 | 校验 |
|------|------|------|------|------|
| `username` | string | 是 | — | `^\w{1,20}$` 且仅 ASCII 字母/数字/下划线（`internal/auth/auth.go:150`）；大小写不敏感查重 |
| `password` | string | 是 | — | 仅要求非空，**没有最小长度/强度要求**；bcrypt cost 8 |
| `invitationCode` | string | 仅 `invite` 模式 | — | 必须是未使用、未过期（`expires_at IS NULL OR expires_at > now()`）的 `registration_tickets.code`，成功后立即标记 used |

其它字段（`emailAddress`、`hcaptcha-response`、`g-recaptcha-response`、`turnstile-response` 等）**被忽略**，不报错。

### 响应 200

`MeDetailed` 对象（字段见"共享对象"），额外多一个：

| 字段 | 类型 | 说明 |
|------|------|------|
| `token` | string | 16 字符原生 API token，`users.token` |

本端点只填充了 `followersCount`/`followingCount`/`notesCount`（`applyUserCounts`）；`unreadNotificationsCount` 恒为 `0`、`pinnedNoteIds`/`pinnedNotes` 恒为 `[]`，需要真实值请随后调 `POST /api/i`。`includeSecrets` 恒为 `true`，所以一定有 `email`（值恒为 `null`）、`emailVerified`、`securityKeysList`。

### 错误

| HTTP | `code` | `kind` | 触发条件 |
|------|--------|--------|----------|
| 503 | `SIGNUP_UNAVAILABLE` | server | 服务未接数据库（`internal/httpapi/account.go:39`） |
| 400 | `INVALID_PARAM` | client | body 不是合法 JSON |
| 400 | `INVALID_USERNAME` | client | 用户名为空/超 20 字符/含非法字符 |
| 400 | `INVALID_PASSWORD` | client | 密码为空 |
| 400 | `REGISTRATION_DISABLED` | client | `admin` 模式（message：`Registration is disabled on this instance.`） |
| 400 | `INVALID_INVITE` | client | `invite` 模式缺邀请码（message：`An invitation code is required.`）或邀请码无效（message：`The invitation code is invalid.`） |
| 400 | `DUPLICATED_USERNAME` | client | 用户名已被占用（message：`Username is already taken.`） |
| 500 | `SIGNUP_FAILED` | server | 其它数据库错误 |

所有 signup 错误的 `id` 都是 `4d371c9a-3a0f-4c8b-9b8e-2f9c0a1d3e45`（`internal/httpapi/account.go:18`）。

### 与 Misskey 的差异 / 已知缺口

- Misskey 在关闭注册时返回**无 code 的裸 400**；Cinderfish 返回带 `code` 的结构化错误，前端可以用 `REGISTRATION_DISABLED` 做文案分支。
- 未实现：注册时填写邮箱（`emailRequiredForSignup` 恒 `false`，注册请求里的 `email` 字段被忽略）、已验证用户名墓碑（`usedUsernames`）、保留用户名（`preservedUsernames`）、任何 captcha 校验、邀请码的邮件绑定流程。注册后的邮箱通过 `i/update-email` 补填，见下一节。
- **远端账号会占用本地用户名**：查重只查本地行（`internal/account/service.go:202`），但 `username_lower` 的唯一索引是全表的（`internal/db/db.go:187`），所以当某个联邦远端账号已用同名时的本地注册会落到 `DUPLICATED_USERNAME`。
- 成功响应结构与 Misskey 的 `SignupApiService` 一致（`MeDetailed` + `token`），Firefish 的注册页在成功后还会再调一次 `signin`（`packages/client/src/components/MkSignup.vue:502`），这条路径可以正常工作。

代码位置：`internal/httpapi/account.go:37`、`internal/account/service.go:83`、`:113`、`:172`。

---

## POST /api/signin

密码登录。认证：**匿名**。

### 请求字段

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `username` | string | 是 | 大小写不敏感匹配**本地**账号；为空时直接 400 |
| `password` | string | 是 | bcrypt 比对；不校验长度 |
| `token` | string | 否 | TOTP 六位码；只在账号已启用两步验证时需要 |

`signature`/`credentialId`/`challengeId`（WebAuthn）、captcha 之类字段仍然忽略（安全密钥未实现）。

### 两步验证协商

账号未启用 TOTP 时行为不变：一次请求直接返回 token。启用后：

1. 只带 `username`/`password` 的请求校验密码，成功时返回 **`200` + 空对象 `{}`**（Firefish 的
   `MkSignin` 就是靠这个空响应切到验证码输入框）；
2. 客户端带上 `token` 重发，验证码正确才签发 token；错误时 403 且 `error.id` 为
   `cdf1235b-ac71-46d4-a3a6-84ccce48df6f`。

中间状态（第 1 步成功）**不写** `signins` 行；验证码错误会写一行 `success=false`。

### 响应 200

```json
{ "finished": true, "id": "<userId>", "i": "<16 字符原生 token>" }
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `finished` | boolean | 恒为 `true`；两步验证的中间响应是 `{}`，没有这个字段 |
| `id` | string | 用户 id |
| `i` | string | `users.token`（16 字符）。**不会**新建 `access_tokens` 记录 |

### 错误

分两类：

**A. 账号/密码类失败**——Misskey 的 **id-only** 信封 `{"error":{"id":"..."}}`，没有 `message`/`code`/`kind`：

| HTTP | `error.id` | 触发条件 |
|------|-----------|----------|
| 400 | `6cc579cc-885d-43d8-95c2-b8c7fc963280` | `username` 为空 |
| 404 | `6cc579cc-885d-43d8-95c2-b8c7fc963280` | 用户不存在 |
| 403 | `932c904e-9460-45b7-9ce6-7ed33be7eb2c` | 密码错误 |
| 403 | `cdf1235b-ac71-46d4-a3a6-84ccce48df6f` | 两步验证码错误或过期 |
| 403 | `e03a5f46-d309-4865-9b69-56282d94e1eb` | 账号已被暂停 |

**B. 服务级失败**——完整信封（`message`/`code`/`id`/`kind` 齐全）：

| HTTP | `code` | `kind` | `id` | 触发条件 |
|------|--------|--------|------|----------|
| 503 | `SIGNIN_UNAVAILABLE` | server | `932c904e-9460-45b7-9ce6-7ed33be7eb2c` | 服务未接数据库 |
| 500 | `SIGNIN_FAILED` | server | `932c904e-9460-45b7-9ce6-7ed33be7eb2c` | 其它数据库错误 |
| 400 | `INVALID_PARAM` | client | `3d81ceae-475f-4600-b2a8-2bc116157532` | body 不是合法 JSON |

这两个 id 正是 Firefish 登录组件分支所依赖的（`packages/client/src/components/MkSignin.vue` 的 `loginFailed`：`6cc579cc…` → no such user，`932c904e…` → incorrect password），可以直接复用。

### 与 Misskey 的差异 / 已知缺口

- 支持 TOTP 两步验证（迁移 v14）。与 Misskey 新版的分步协议不同，这里沿用 Firefish 的实现：
  缺 `token` 时返回空 `{}`，而不是 `{"finished":false,...}`；Firefish 客户端行为正确。
- 无登录失败次数限制（Misskey 有 `TOO_MANY_AUTHENTICATION_FAILURES`）。
- 无 passkey/WebAuthn 与 `passwordLessLogin`；`i/2fa/password-less` 恒报 `NO_SECURITY_KEY`。
- 无两步验证备份码（`twoFactorBackupCodesStock` 恒为 `"none"`）。
- 与 Misskey 不同的是 HTTP 状态码（Misskey 侧多用 400）；只按 `error.id` 分支的客户端不受影响。

代码位置：`internal/httpapi/account.go:114`、`internal/account/service.go:261`、
`internal/auth/totp.go`。

---

## POST /api/i/2fa/register

开始 TOTP 绑定。认证：**需登录**。

### 请求字段

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `password` | string | 是 | 当前密码；错误时 403 + id-only 信封 `932c904e…` |

### 响应 200

```json
{
  "qr": "",
  "url": "otpauth://totp/example.com:alice?secret=…&issuer=example.com&algorithm=SHA1&digits=6&period=30",
  "secret": "<base32>",
  "label": "alice",
  "issuer": "example.com"
}
```

`qr` 恒为空串：生成二维码需要图像编码依赖，前端改为展示 `secret` 与 `url` 供手动录入。
密钥写入 `user_profiles.two_factor_temp_secret`，覆盖上一次未完成的注册；此时**尚未启用**。

## POST /api/i/2fa/done

确认并启用 TOTP。认证：**需登录**。

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `token` | string | 是 | 6 位验证码；比较时忽略空格，接受 ±1 个 30 秒步长 |

成功返回 `204`。错误：

| HTTP | `code` | 触发条件 |
|------|--------|----------|
| 400 | `TWO_FACTOR_NOT_STARTED` | 没有待确认的临时密钥 |
| 400 | `TOTP_TOKEN_INVALID` | 验证码不匹配 |

## POST /api/i/2fa/unregister

停用 TOTP。认证：**需登录**；请求体只要 `password`，错误密码同样是 403 + `932c904e…`。
成功清空两个密钥列、`two_factor_enabled` 与 `use_password_less_login`，返回 `204`。

## POST /api/i/2fa/password-less

认证：**需登录**；请求体 `{"value": boolean}`。安全密钥未实现，`value:true` 恒返回
`400 NO_SECURITY_KEY`（id `f9c54d7f-d4c2-4d3c-9a8f-a70daac86512`），`value:false` 返回 `204`。
设置页的开关因此保持禁用。

## POST /api/i/change-password

认证：**需登录**。请求体 `{currentPassword, newPassword}`，`newPassword` 不能为空（空值 400
`INVALID_PARAM`）。当前密码错误返回 403 + id-only 信封 `932c904e…`；成功返回 `204`。
只替换 `users.password_hash`，不吊销既有 token。

## POST /api/i/regenerate-token

认证：**需登录**。请求体 `{password}`，密码错误同样是 403 + `932c904e…`。成功时把
`users.token` 换成新的 16 字符 token，返回 `{"token":"<new>"}`。

> 与 Misskey 的差异：Misskey 返回空响应并靠 Streaming 的 `myTokenRegenerated` 通知客户端；
> 这里额外回传新 token，让浏览器能立刻替换 localStorage 里的凭据。多出的字段对 Misskey
> 客户端无影响。

## POST /api/i/signin-history

认证：**需登录**。请求体可带 `limit`（1–100，默认 10）、`sinceId`、`untilId`；按 `signins.id`
降序返回：

```json
[{ "id": "…", "createdAt": "…", "ip": "127.0.0.1", "headers": {}, "success": true }]
```

`headers` 恒为 `{}`（表里不存请求头）；`success` 区分成功与失败尝试。

---

## POST /api/username/available

用户名能否注册。认证：**匿名**。

### 请求字段

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `username` | string | 是 | 空值/非法格式会得到 `available:false`，不报错 |

### 响应 200

```json
{ "available": true }
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `available` | boolean | 格式合法且本地未被占用为 `true`；服务未接数据库时恒为 `false`（`internal/httpapi/account.go:176`） |

无错误分支（只有 `INTERNAL_ERROR`/500 的数据库失败路径）。

### 与 Misskey 的差异 / 已知缺口

- 只按"本地账号 + 格式"判断，不看保留名、不看远端账号占用（远端占用会在真正的 signup 阶段以 `DUPLICATED_USERNAME` 暴露）。

代码位置：`internal/httpapi/account.go:174`、`internal/account/service.go:214`。

---

## POST /api/miauth/gen-token

为已确认的 MiAuth 会话签发一个 access token。认证：**需登录**（body `i` 或 `Authorization` 均可）。

### 请求字段

| 字段 | 类型 | 必填 | 默认 | 说明 |
|------|------|------|------|------|
| `session` | string | **是** | — | 空则 400 `INVALID_PARAM`（Misskey 侧是 UUID，这里不做格式校验） |
| `name` | string | 否 | 空 | 应用名，写入 `access_tokens.name` |
| `description` | string | 否 | 空 | 写入 `description` |
| `iconUrl` | string | 否 | 空 | 写入 `icon_url` |
| `permission` | string[] | 否 | `[]` | 原样存进 `access_tokens.scopes`；非字符串元素被跳过（`internal/httpapi/account.go:321`） |

### 响应 200

```json
{ "token": "<32 字符 token>" }
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `token` | string | 32 字符 MiAuth token |

签发的 token 在 `account.TokenService.Resolve` 里按"非 16 字符 → 查 access_tokens"处理，因此**用它调用 `POST /api/i` 时 `includeSecrets = false`**（响应里没有 `email`/`emailVerified`/`securityKeysList`）。

### 错误

| HTTP | `code` | `kind` | 触发条件 |
|------|--------|--------|----------|
| 401 | `AUTHENTICATION_FAILED` | client | 无 token / token 无效 / 账号被暂停 |
| 400 | `INVALID_PARAM` | client | `session` 缺失 |
| 503 | `INTERNAL_ERROR` | server | token 服务未初始化 |
| 500 | `INTERNAL_ERROR` | server | 写库失败 |

### 与 Misskey 的差异 / 已知缺口

- **MiAuth 流程不完整**：只注册了 `miauth/gen-token`，Misskey 的 `/miauth/:session` 授权页、`/api/miauth/:session/check`、`/api/auth/session/*` 均未实现。前端若要支持 MiAuth 登录，需要自己做授权页并直接调本端点。
- `permission` 在本实现里**不是必填**（Misskey 的 `paramDef` 里 `session` 与 `permission` 都是 required），且任何 scope 都不做权限校验——只有 `/streaming` 检查 `read:account`。
- 不发送 `createToken` 通知（通知服务未接入该类型）。
- 返回结构 `{token}` 与 Misskey 一致；Firefish 的 `/miauth/:session` 页面调用本端点时只等 Promise 完成、不使用返回值（`packages/client/src/pages/miauth.vue:109`），故可直接对接。

代码位置：`internal/httpapi/account.go:290`、`internal/account/tokens.go:43`。

---

## POST /api/i

当前登录用户的完整资料。认证：**需登录**。这是前端启动时"拿到 me 对象"的端点。

### 请求字段

除认证用的 `i`/`Authorization` 外**没有参数**；body 可以为空。

### 响应 200

`MeDetailed`（字段见"共享对象"），并额外填充：

| 字段 | 说明 |
|------|------|
| `followersCount` / `followingCount` / `notesCount` | 真实计数（`applyUserCounts`） |
| `unreadNotificationsCount` / `hasUnreadNotification` | 真实未读数（`applyUnreadNotifications`） |
| `pinnedNoteIds` / `pinnedNotes` | 真实置顶列表，最多 5 条（`applyPinnedNotes`，`viewerID = 自己`） |

`includeSecrets = (info.Token == nil)`：请求用的是 16 字符原生 token 时才有 `email`（`null`）、`emailVerified`（`false`）、`securityKeysList`（`[]`）。

### 错误

| HTTP | `code` | 说明 |
|------|--------|------|
| 401 | `AUTHENTICATION_FAILED` | 无 token / 无效 token / 已暂停，带 `WWW-Authenticate` |
| 500 | `INTERNAL_ERROR` | 理论上只有数据库读失败 |

注意：`i` 只有 `POST /api/i` 一条路由；`GET /api/i` 不匹配任何有效方法+路径组合，会命中兜底路由得到 404 `NO_SUCH_ENDPOINT`（不是 405）。

### 与 Misskey 的差异 / 已知缺口

- `avatarId`、`bannerId` 恒为 `null`，前端不能靠 `i.avatarId` 反查头像文件；只能用 `avatarUrl`。
- 部分设置字段仍是常量（见 `MeDetailed` 表）：`mutedWords` 恒 `[]`、`followApproval*Seconds` 恒 `0` 等。把这些字段写回 `i/update` 不会报错，但也不会保存。
- `hasPendingReceivedFollowRequest` 恒 `false`（关注请求列表由 `following/requests/list` 提供）。

代码位置：`internal/httpapi/account.go:200`。

---

## POST /api/i/update

修改当前用户的可编辑资料。认证：**需登录**。

### 别名

`POST /api/users/update` 注册到同一个 handler（`internal/httpapi/server.go:117-119`），请求/响应/错误**完全一致**。上游 Misskey 只有 `i/update`（`tmp/refcheck/misskey/.../endpoints/i/update.ts`，没有 `users/update.ts`），`users/update` 是本实现额外加的别名；`server.go` 的注释把它说成"Misskey 的规范拼写"，与实际上游不符。Firefish 客户端两端都只用 `i/update`，所以这个别名对移植没有影响。

### 请求字段

语义要点：**字段不出现 = 不改动**；出现 `null` 或 `""` = 清空该字段（`nullableField`，`internal/httpapi/i_update.go:23`）。传非字符串非 null 的值（如数字）会导致整个 body 解析失败 → 400 `INVALID_PARAM`。

| 字段 | 类型 | 必填 | 校验 | 落库目标 |
|------|------|------|------|----------|
| `name` | string \| null | 否 | 字符数（rune）≤ 50，超出 400 `INVALID_PARAM`（`info`: `"name is too long"`） | `user_profiles.name`（空串存 NULL） |
| `description` | string \| null | 否 | 字符数 ≤ 500，超出 400（`info`: `"description is too long"`） | `user_profiles.description` |
| `avatarId` | string \| null | 否 | 非空时必须是**调用者自己的** drive 文件：不存在 → 400 `NO_SUCH_FILE`，不是自己的 → 403 `ACCESS_DENIED` | 把该文件的 `url` 写入 `user_profiles.avatar_url`；`""`/`null` 清空为 NULL |
| `followersVisibility` | string \| null | 否 | 仅接受 `public` \| `followers` \| `private`，否则 400（`info`: `"followersVisibility must be public, followers or private"`） | `user_profiles.followers_visibility` |
| `followingVisibility` | string \| null | 否 | 同上（`info` 里换成 `followingVisibility`） | `user_profiles.following_visibility` |
| `location` | string \| null | 否 | 字符数 ≤ 50，超出 400 | `settings.location` |
| `birthday` | string \| null | 否 | `YYYY-MM-DD` 或不带值的日期，否则 400 | `settings.birthday` |
| `lang` | string \| null | 否 | ≤ 16 字符且只含字母/数字/`-` | `settings.lang` |
| `fields` | array \| null | 否 | 最多 16 条 `{name,value}`；name ≤ 255、value ≤ 1024；空 name 丢弃 | `settings.fields` |
| `mutingNotificationTypes` | array \| null | 否 | 只接受本后端产生的 5 类通知，最多 5 项 | `settings.mutingNotificationTypes` |
| `noCrawle` / `preventAiLearning` / `isExplorable` / `hideOnlineStatus` / `publicReactions` / `alwaysMarkNsfw` / `autoSensitive` / `carefulBot` / `autoAcceptFollowed` / `injectFeaturedNote` | boolean | 否 | 无 | `settings.<字段名>`，读回时 `isExplorable` 缺省为 `true` |
| `isLocked` | boolean | 否 | 无 | `users.is_locked`（不是 settings）；锁推后新关注进入待审 |

**被静默忽略的字段**（Firefish 的设置页会发）：`bannerId`、`followedMessage`、`isBot`、`isCat`、`speakAsCat`、`requireSigninToViewContents`、`mutedWords`、`notificationRecieveConfig`、`alsoKnownAs`、`avatarDecorations`、`chatScope`、`pinnedPageId` 等。它们不会报错，也不会保存。`mutedInstances` 单独处理（见 §`i/update` 的 `mutedInstances`）。

### 响应 200

`MeDetailed`，填充 `followersCount`/`followingCount`/`notesCount` 与未读通知计数。

**但不会填充 `pinnedNoteIds`/`pinnedNotes`**（`internal/httpapi/i_update.go:141-144` 只调了 counts 与 unread），所以响应里这两个字段是 `[]`。前端如果用整个响应替换本地 `me` 对象，会把置顶列表刷没——`i/pin` 的注释明确提到这个"整对象替换"的用法（`internal/httpapi/i_pin.go:82-86`），因此 `i/update` 这条路径需要前端自己合并字段。

### 错误

| HTTP | `code` | `kind` | 触发条件 |
|------|--------|--------|----------|
| 401 | `AUTHENTICATION_FAILED` | client | 无/无效 token、账号暂停 |
| 400 | `INVALID_PARAM` | client | body 非 JSON；`name`/`description` 超长；可见性取值非法 |
| 503 | `INTERNAL_ERROR` | server | 上传驱动未配置（设置头像时需要） |
| 400 | `NO_SUCH_FILE` | client | `avatarId` 指向不存在的文件（`id` 用的是 `INVALID_PARAM` 的 UUID） |
| 403 | `ACCESS_DENIED` | permission | `avatarId` 不是调用者的文件 |
| 500 | `INTERNAL_ERROR` | server | 写库失败 |

### 与 Misskey 的差异 / 已知缺口

- 支持 name、description、avatarId、两个 visibility，以及 location/birthday/lang/fields 与 10 个布尔开关；Misskey 的其余 30+ 个参数仍然无效。
- 没有 `bannerId`：横幅完全不可设置，`bannerUrl` 恒为 `null`。
- `isBot`/`isCat` 与发言猫语仍无写入路径（`isLocked` 自第四十七波起可写）。
- 没有邮箱修改与 `i/update-email`；改密见 `i/change-password`（第二十一波）。
- `avatarId` 回读问题：设置成功后响应里的 `avatarId` 仍是 `null`，Firefish 头像裁剪后 `me.avatarId = i.avatarId` 会拿到 `null`（`packages/client/src/pages/settings/profile.vue:254`），但 `avatarUrl` 是正确的，头像显示不受影响。

代码位置：`internal/httpapi/i_update.go:66`、`internal/account/profile.go:65`、`:112`、`internal/httpapi/i_update.go:159`。

---

## POST /api/users/show

查询任意用户的详情。认证：**可选**（匿名可读；带 token 时补充与查看者的关系标志）。

### 请求字段

两种取值方式二选一：

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `userId` | string | 与 `username` 二选一 | 优先使用；可查远端用户（`users.id` 全局唯一） |
| `username` | string | 与 `userId` 二选一 | **仅本地**账号（`host IS NULL`），大小写不敏感 |
| `host` | string | 否 | 只能为空（或 `null`）。**传非空值不会被解析**，而是落到 400 `INVALID_PARAM`；因此"用 `username` + `host` 查远端用户"在本实现里不可用（`internal/httpapi/account.go:241`） |

`userId` 与 `username` 都没给（或只给了非空 `host`）→ 400 `INVALID_PARAM`（**不带** `info`）。

### 响应 200

`UserDetailed`（**不是** `MeDetailed`，因此没有 `avatarId`、没有 `email`、没有 `mutedWords` 等），并填充：

| 字段 | 说明 |
|------|------|
| `followersCount` / `followingCount` / `notesCount` | 真实计数 |
| `pinnedNoteIds` / `pinnedNotes` | 目标用户的置顶列表，按 `viewerID` 打包（未登录时 `viewerID = ""`，`myReaction` 之类的查看者相关字段不出现） |
| `isFollowing` | `viewer` 关注了 `target` |
| `isFollowed` | `target` 关注了 `viewer` |
| `hasPendingFollowRequestFromYou` | 存在 `viewer → target` 的待审关注请求 |
| `hasPendingFollowRequestToYou` | 存在 `target → viewer` 的待审关注请求 |

关系标志的填充条件（`internal/httpapi/account.go:274-282`）：请求带**有效** token、查看者不是目标本人、且服务已挂载联邦存储（生产环境恒成立，`cmd/cinderfish` 里会 `AttachFederation`）。查看自己时四个标志恒为 `false`。**无效 token 不会导致 401**，只是退化为匿名（有测试固定在 `internal/httpapi/users_relations_test.go:39`）。

### 错误

| HTTP | `code` | `id` | 触发条件 |
|------|--------|------|----------|
| 400 | `INVALID_PARAM` | `3d81ceae-…` | 参数不足/`host` 非空 |
| 404 | `NO_SUCH_USER` | `6cc579cc-885d-43d8-95c2-b8c7fc963280` | 用户不存在 |
| 500 | `INTERNAL_ERROR` | `5d37dbcb-891e-41ca-a3d6-e690c97775ac` | 读库失败 |
| 503 | `INTERNAL_ERROR` | 同上 | 服务未接数据库 |

### 与 Misskey 的差异 / 已知缺口

- 不支持 `username` + `host` 查远端用户（Misskey 的 `paramDef` 要求这对字段同传）。
- Misskey 会按 `followersVisibility` 拒绝匿名读取"仅关注者"资料（返回 `FORBIDDEN`）；**本实现不做该检查**，匿名也能拿到完整的 `UserDetailed`（`description` 等都会返回）。这是隐私口径差异，前端如果依赖 403 做 UI 降级需要另找依据。
- 远端用户虽然可查，但 `url`/`uri` 恒为空串、`bannerUrl` 恒 `null`、`fields` 恒空、`isBot`/`isCat`/`isLocked` 恒 `false`——数据库里有这些列但打包时被写死（`internal/httpapi/users_pack.go:46`）。
- `isModerator` 与 `isAdmin` 同源，没有独立 moderator。

代码位置：`internal/httpapi/account.go:220`、`internal/httpapi/users.go:256`、`:287`。

---

## POST /api/users/notes

某用户的笔记列表（个人主页时间线）。认证：**可选**。返回 `Note` **详情**对象数组（字段见"共享对象 → `Note`"）。

### 请求字段

| 字段 | 类型 | 必填 | 默认 | 校验/行为 |
|------|------|------|------|-----------|
| `userId` | string | 与 `username` 二选一 | — | 优先；可查远端用户 |
| `username` | string | 与 `userId` 二选一 | — | **仅本地**账号 |
| `limit` | number | 否 | 20 | `limit <= 0` 或 `limit > 100` 都会变成 20（`internal/note/timeline.go:251`） |
| `sinceId` | string | 否 | — | `notes.id > sinceId` |
| `untilId` | string | 否 | — | `notes.id < untilId` |
| `withFiles` | boolean | 否 | `false` | `true` 时只返回 `file_ids <> '{}'` 的笔记 |
| `withReplies` | boolean | 否 | **`true`** | `false` 时只保留"非回复"或"回复自己"的笔记（`reply_id IS NULL OR reply_user_id = user_id`） |

排序永远是 `ORDER BY n.id DESC`（新→旧），**不会**因为只给 `sinceId` 而翻转成升序——这一点与 `users/followers`/`users/following` 不同（`internal/note/timeline.go:270`）。

**被忽略的字段**：`withRenotes`、`withChannelNotes`、`sinceDate`、`untilDate`、`allowPartial`、`includeReplies`（Firefish 的 `includeReplies`）、`fileType`、`excludeNsfw`。

### 响应 200

`Note[]`。每个元素的 `user` 是 `UserLite`；带有效 token 时，查看者自己反应过的笔记会有 `myReaction`。

### 可见性规则（重要）

查询条件固定为（`internal/note/timeline.go:255-257`）：

- `n.user_id = <目标>`
- `n.user_host IS NULL`（**只查本地笔记**）
- `n.deleted_at IS NULL`
- `n.local_only = FALSE`
- `n.visibility IN ('public','home')`

即：**`followers` 与 `specified` 笔记永远不会出现在这里——包括目标用户自己查看自己的主页时**。没有"关注者可见"分支，也没有 `isFollowing` 判断。

### 错误

| HTTP | `code` | `id` | 触发条件 |
|------|--------|------|----------|
| 400 | `INVALID_PARAM` | `3d81ceae-…` | `userId`/`username` 都没给（`info`: `"userId or username is required"`） |
| 404 | `NO_SUCH_USER` | `6cc579cc-…` | 用户不存在 |
| 500 | `INTERNAL_ERROR` | `5d37dbcb-…` | 读库失败 |

### 与 Misskey 的差异 / 已知缺口

- **`withReplies` 默认值相反**：Misskey 上游默认 `false`（`users/notes.ts` 的 `paramDef`），本实现默认 `true`。前端如果要"只看原创"，必须显式传 `withReplies:false`。
- Misskey 在 `withReplies && withFiles` 同时为真时抛 `BOTH_WITH_REPLIES_AND_WITH_FILES`；本实现允许两者同时为真（取交集）。
- Misskey 的 `userId` 是必填且只接受 id；本实现额外接受 `username`。
- `limit` 默认 20（Misskey 10），且 `limit > 100` 是重置为 20 而不是钳制到 100。
- `followers` 可见性的笔记对关注者不可见（见上），这是与 Misskey 行为差异最大的一处。
- Firefish 的个人主页图片页用 `users/notes` + `fileType`（`packages/client/src/pages/user/index.photos.vue:69`），本实现忽略 `fileType`，会返回无附件笔记；媒体页用 `withFiles:true`（`media-list.vue:24`）则可正常工作。
- Firefish 的 tab 传的是 `includeReplies`（`user/index.timeline.vue:28`），本实现忽略它——"只原创"的 tab 不会生效。

代码位置：`internal/httpapi/users.go:24`、`internal/note/timeline.go:250`、`internal/httpapi/notes_pack.go:78`。

---

## POST /api/users/followers

目标用户的粉丝列表。认证：**可选**（匿名可读，但受目标的可见性设置限制）。返回 `Following` 对象数组，每项带 `follower`（`UserLite`）。

### 请求字段

| 字段 | 类型 | 必填 | 默认 | 说明 |
|------|------|------|------|------|
| `userId` | string | 匿名时必填；带 token 时可省 | 调用者自己 | 目标用户 id。**不接受 `username`/`host`** |
| `limit` | number | 否 | 10 | `> 100` 钳制为 100 |
| `sinceId` | string | 否 | — | **关注行 id**（`followings.id`），不是用户 id |
| `untilId` | string | 否 | — | 同上 |

### 响应 200

`Following[]`，字段见"共享对象 → `Following`"：`id`（关注行 id）、`createdAt`、`followerId`、`followeeId`（恒为目标用户 id）、`follower`（UserLite）。

排序：默认 `f.id DESC`（最新关注在前）；只给 `sinceId` 时为 `f.id ASC`（向上翻页）。

### 可见性判定

`followListAllowed`（`internal/httpapi/users.go:225`）：

1. 查看者 == 目标，或是管理员（`is_admin`）→ 直接放行；
2. 读目标的 `user_profiles.followers_visibility`：`public`/空 → 放行；`private` → 拒绝（403）；`followers` → 仅当查看者**正在关注目标**时放行；
3. 其它未知取值 → 拒绝。

注意这是按 **`followers_visibility`** 判断的，与"看谁的粉丝列表"无关——用 `followingVisibility` 控制的场景是 `users/following`。

### 错误

| HTTP | `code` | `id` | 触发条件 |
|------|--------|------|----------|
| 400 | `INVALID_PARAM` | `3d81ceae-…` | 匿名且未给 `userId`（`info`: `"userId is required"`） |
| 404 | `NO_SUCH_USER` | `27fa5435-88ab-43de-9360-387de88727cd` | 目标不存在 |
| 403 | `FORBIDDEN` | `3c6a84db-d619-26af-ca14-06232a21df8a` | 可见性不允许 |
| 500 | `INTERNAL_ERROR` | `5d37dbcb-…` | 读库失败 |

### 与 Misskey 的差异 / 已知缺口

- Misskey 支持 `username` + `host` 定位目标；本实现只支持 `userId`（匿名的 MyPage 需要先知道 id）。
- 分页游标是**关注行 id**，Misskey 侧是 `following` 表 id，语义一致，但前端不要把它当成用户 id 使用。
- 响应里只嵌了 `follower`（`users/following` 则只有 `followee`，见 `internal/httpapi/users.go:208-212`）——这一点与 Misskey 一致，上游也是单侧填充（`FollowingEntityService.pack` 的 `populateFollower`/`populateFollowee`）。但**嵌入对象的 schema 更薄**：本实现是 `UserLite`，Misskey 是 `UserDetailedNotMe`（`UserDetailed`）。前端若需要被嵌入用户的 `description`、`followersCount` 等，必须另外调 `users/show`。
- `userId` 省略时默认取调用者自己，这是本实现的扩展；Misskey 的 `paramDef` 要求显式给 `userId`，或给 `username` + `host`。

代码位置：`internal/httpapi/users.go:104`、`:123`、`:225`、`internal/account/follows.go:53`、`:66`。

---

## POST /api/users/following

目标用户的关注列表。认证：**可选**。其余与 `users/followers` 对称。

### 请求字段

与 `users/followers` 相同：`userId`（匿名必填，默认自己）、`limit`（默认 10，上限 100）、`sinceId`/`untilId`（**关注行 id**）。

### 响应 200

`Following[]`，每项带 `followee`（`UserLite`）而不是 `follower`；`followerId` 恒为目标用户 id。

### 可见性判定

同上，但读的是 `user_profiles.following_visibility`（`internal/httpapi/users.go:235`）。

### 错误

| HTTP | `code` | `id` | 触发条件 |
|------|--------|------|----------|
| 400 | `INVALID_PARAM` | `3d81ceae-…` | 匿名且未给 `userId` |
| 404 | `NO_SUCH_USER` | `63e4aba4-4156-4e53-be25-c9559e42d71b` | 目标不存在 |
| 403 | `FORBIDDEN` | `f6cdb0df-c19f-ec5c-7dbb-0ba84a1f92ba` | 可见性不允许 |
| 500 | `INTERNAL_ERROR` | `5d37dbcb-…` | 读库失败 |

### 与 Misskey 的差异 / 已知缺口

与 `users/followers` 相同：不支持 `username`/`host`，响应只嵌一侧用户。

代码位置：`internal/httpapi/users.go:110`、`internal/account/follows.go:59`。

---

## POST /api/i/pin

把自己的一条笔记加入置顶列表（列表上限 5 条，最新的排在最前）。认证：**需登录**。

### 请求字段

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `noteId` | string | 是 | 必须存在、未软删除、且**作者就是调用者** |

### 响应 200

`MeDetailed`，填充 `followersCount`/`followingCount`/`notesCount`、未读计数、以及**最新的** `pinnedNoteIds`/`pinnedNotes`（`internal/httpapi/i_pin.go:79-87`）。这里刻意补齐置顶列表，因为前端会拿响应整体替换本地 `me`。

行为细节：

- 重复置顶同一条笔记是**幂等成功**（返回 200，列表不变）；
- 列表已满 5 条时再置顶新笔记 → 400 `PIN_LIMIT_EXCEEDED`；
- 只能置顶自己的笔记；别人的笔记与不存在的笔记返回**同一个** `NO_SUCH_NOTE`，不泄露存在性。

### 错误

| HTTP | `code` | `id` | 触发条件 |
|------|--------|------|----------|
| 401 | `AUTHENTICATION_FAILED` | `b0a7f5f8-…` | 无/无效 token |
| 400 | `INVALID_PARAM` | `3d81ceae-…` | `noteId` 缺失（`info`: `"noteId is required"`）或 body 非 JSON |
| 404 | `NO_SUCH_NOTE` | `24fcbfc6-2e37-42b6-8388-c29b3861a08d` | 笔记不存在/已删/不属于自己 |
| 400 | `PIN_LIMIT_EXCEEDED` | `4e1b0f8e-9a2d-4c7e-b3a1-6f5d8c2b7a94` | 已有 5 条置顶 |
| 500 | `INTERNAL_ERROR` | `5d37dbcb-…` | 写库失败 |

### 与 Misskey 的差异 / 已知缺口

- Misskey 对重复置顶返回 `ALREADY_PINNED` 错误；本实现是幂等成功，前端不需要处理该错误码。
- `PIN_LIMIT_EXCEEDED` 的 `id` 是 Cinderfish 自造的 UUID（Misskey 的该错误 id 不同），按 `code` 判断更稳。
- 置顶存在 `user_profiles.pinned_note_ids` 数组列里，没有 `pin` 关系表，所以没有 `pinnedAt` 之类的时间戳。

代码位置：`internal/httpapi/i_pin.go:17`、`:29`、`internal/account/pins.go:12`、`:53`。

---

## POST /api/i/unpin

把一条笔记从自己的置顶列表移除。认证：**需登录**。

### 请求字段

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `noteId` | string | 是 | 同 `i/pin` 的存在性与归属校验 |

### 响应 200

`MeDetailed`，填充计数、未读计数与最新的 `pinnedNoteIds`/`pinnedNotes`。

行为细节：移除一条**自己的但未置顶**的笔记是幂等成功（200）。不存在的笔记、别人的笔记 → 404 `NO_SUCH_NOTE`。

### 错误

| HTTP | `code` | 触发条件 |
|------|--------|----------|
| 401 | `AUTHENTICATION_FAILED` | 无/无效 token |
| 400 | `INVALID_PARAM` | `noteId` 缺失或 body 非 JSON |
| 404 | `NO_SUCH_NOTE` | 笔记不存在/已删/不属于自己 |
| 500 | `INTERNAL_ERROR` | 写库失败 |

（本端点不会返回 `PIN_LIMIT_EXCEEDED`。）

### 与 Misskey 的差异 / 已知缺口

同 `i/pin`：没有 `pinnedAt` 信息，错误 id 为 Cinderfish 自有值。

代码位置：`internal/httpapi/i_pin.go:25`、`:29`、`internal/account/pins.go:101`。

---

## GET /.well-known/oauth-authorization-server

RFC 8414 授权服务器元数据。认证：**匿名**。不需要数据库。

### 请求参数

无。

### 响应 200

由 `oauthMetadata()` 生成（`internal/httpapi/oauth.go:38`），`issuer`/endpoint 都基于配置的 `Domain`（拼成 `https://<Domain>`）：

| 字段 | 类型 | 值 |
|------|------|----|
| `issuer` | string | `https://<domain>` |
| `authorization_endpoint` | string | `https://<domain>/oauth/authorize` |
| `token_endpoint` | string | `https://<domain>/oauth/token` |
| `scopes_supported` | string[] | `["read","write","follow","upload"]`（声明值；实际不校验，见下） |
| `response_types_supported` | string[] | `["code"]` |
| `grant_types_supported` | string[] | `["authorization_code"]` |
| `service_documentation` | string | `https://misskey-hub.net`（直接沿用上游链接） |
| `code_challenge_methods_supported` | string[] | `["S256"]` |
| `authorization_response_iss_parameter_supported` | boolean | `true` |

未返回：`token_endpoint_auth_methods_supported`、`revocation_endpoint`、`registration_endpoint`。

### 错误

无（永远 200，除非进程崩溃）。

### 与 Misskey 的差异 / 已知缺口

- 声明的 4 个 scope 与 Misskey 的 `read:account`/`write:notes` 风格不同，是 Cinderfish 自定的粗粒度集合；而且**没有任何端点校验它们**（没有 scope 白名单，客户端请求什么就存什么），只有 `/streaming` 检查 `read:account`——所以按元数据只请求 `read`/`write` 的 OAuth token 无法连接 Streaming，必须显式把 `read:account` 写进 `scope` 参数才行。
- 没有动态客户端注册端点（RFC 7591）。

代码位置：`internal/httpapi/oauth.go:38`、`:54`。

---

## GET /oauth/authorize

OAuth 2.0 授权码端点（仅 PKCE `S256`）。认证：**需要登录**，且**只能通过 `Authorization: Bearer` 头**——这是 GET，请求体里的 `i` 字段读不到（`handleOAuthAuthorize` 调 `authenticate(..., nil)`，`internal/httpapi/oauth.go:95`）。

### 查询参数

| 参数 | 必填 | 校验 |
|------|------|------|
| `client_id` | 是 | 仅判非空，**不与任何注册表比对** |
| `redirect_uri` | 是 | 非空且能被 `url.Parse` 解析；**不做白名单校验** |
| `response_type` | 是 | 必须是 `code`，否则 `unsupported_response_type` |
| `code_challenge` | 是 | 非空 |
| `code_challenge_method` | 是 | 必须是 `S256`（`plain` 不支持） |
| `scope` | 否 | **可重复出现**；缺省为 `["read"]`；原样存库，不校验 |
| `state` | 否 | 原样回传 |

### 响应

**成功**：`302 Location: <redirect_uri>?code=<128 字符>&state=<原样>&iss=https://<domain>`。`iss` 总是附加；`state` 只在请求带了才附加。同时写一行 `oauth_grant_codes`（含 `client_id`、`user_id`、`redirect_uri`、`code_challenge`、`scopes`）。

**未登录/无效 token**：HTTP 401 + `Content-Type: text/html`，响应体是一段固定 HTML：

```html
<!doctype html><meta charset="utf-8"><title>CinderFish</title><p>Sign in to authorize this application.</p>
```

这不是 JSON，前端不能按错误信封解析。

**参数错误**：JSON `{"error":"<code>","error_description":"<描述>","state":""}`（`state` 恒为空串，即使请求给了 `state`）。

| HTTP | `error` | `error_description` |
|------|---------|---------------------|
| 400 | `invalid_request` | `client_id must be provided` |
| 400 | `invalid_request` | `redirect_uri must be provided` |
| 400 | `unsupported_response_type` | `Only the code response type is supported` |
| 400 | `invalid_request` | `code_challenge parameter is required` |
| 400 | `invalid_request` | `code_challenge_method parameter must be set as S256` |
| 400 | `invalid_request` | `Invalid redirect_uri`（`url.Parse` 失败） |
| 500 | `server_error` | `Could not issue the authorization code` |

### 与 Misskey 的差异 / 已知缺口

- **没有交互式授权页**：已登录请求会被直接签发授权码并 302，用户没有"允许/拒绝"的机会（代码注释也说明这一步留给前端，`internal/httpapi/oauth.go:104`）。对第三方客户端而言这是安全缺口，前端实现授权页时应自行加确认步骤。
- `client_id`、`redirect_uri` 都不做注册与白名单校验，任何 URI 都能收到授权码 → 存在开放重定向风险。
- 授权码**没有有效期**（`oauth_grant_codes` 只有 `used` 布尔，见 `internal/db/db.go:222`），只在兑换时被标记使用。
- 错误响应里的 `state` 不回声给客户端，前端无法按 state 关联请求。

代码位置：`internal/httpapi/oauth.go:63`、`:217`、`:250`。

---

## POST /oauth/token

用授权码 + PKCE verifier 兑换 access token。认证：**匿名**（客户端凭据即 `code` + `code_verifier`，无 `client_secret`）。

### 请求格式

**两种都接受**：

- `application/x-www-form-urlencoded`（标准做法），用 `r.ParseForm()` 解析；
- `application/json`，只读取 `grant_type`、`code`、`client_id`、`redirect_uri`、`code_verifier` 这 5 个字符串字段（`internal/httpapi/oauth.go:142-153`）。

| 字段 | 必填 | 说明 |
|------|------|------|
| `grant_type` | 是 | 只支持 `authorization_code` |
| `code` | 是 | 授权码（128 字符） |
| `client_id` | 是 | 必须与授权码里记录的一致 |
| `redirect_uri` | 是 | 必须与授权码里记录的**字符串完全一致** |
| `code_verifier` | 是 | 用它算 S256 challenge 与库里的 `code_challenge` 比对 |

### 响应 200

```json
{ "access_token": "<128 字符>", "token_type": "Bearer", "scope": "read write" }
```

| 字段 | 类型 | 说明 |
|------|------|------|
| `access_token` | string | 128 字符 token，写入 `access_tokens`（`session` 空、`name` = `client_id`、`description` = `"OAuth 2.0"`、`scopes` = 授权时的 scope 列表） |
| `token_type` | string | 恒为 `"Bearer"` |
| `scope` | string | 空格连接的 scope；无 scope 时为空串 |

未返回：`expires_in`、`refresh_token`、`id_token`。token 没有过期时间（`access_tokens` 无 `expires_at` 列）。

### 错误

统一 JSON `{"error":..., "error_description":..., "state":""}`：

| HTTP | `error` | 触发条件 |
|------|---------|----------|
| 400 | `invalid_request` | body 解析失败 / `grant_type` 缺失 |
| 400 | `unsupported_grant_type` | `grant_type != authorization_code` |
| 400 | `invalid_grant` | `code` 缺失；授权码不存在/已使用；`client_id` 或 `redirect_uri` 不匹配；`code_verifier` 缺失或校验失败 |
| 500 | `server_error` | 签发 token 写库失败 |

所有 `invalid_grant` 的 `error_description` 都是 `grant request is invalid`（不区分具体原因）。

### 与 Misskey 的差异 / 已知缺口

- 无 `client_secret`/`client_secret_basic` 校验，纯 PKCE 公共客户端模型。
- **授权码在 PKCE 校验之前就被标记 used**（`redeemGrant` 先 `UPDATE ... SET used = TRUE`，`internal/httpapi/oauth.go:239`）：一次 verifier 写错就会作废该授权码，只能重新走 authorize。
- 无 refresh token / 无过期 / 无撤销端点（`TokenService.Revoke` 存在但没有 HTTP 端点）。
- 拿到的 token 是"应用 token"，用调 `POST /api/i` 时不含 secrets；且 scope 不会被任何 `/api/*` 端点校验。

代码位置：`internal/httpapi/oauth.go:139`、`:217`、`internal/account/tokens.go:43`。

---

## 附录 A：前端对接时的硬性注意点

1. **成功响应是裸值**，`os.api` 那种"直接返回 body"的封装可以直接用；非 200 一律按 `body.error` 拒绝（Firefish 的 `os.api` 就是这么做的：只把 200 当成功、其余 `reject(body.error)`，见 `packages/client/src/os.ts:44-63`）。
2. **token 放在 `Authorization: Bearer` 头**即可，所有端点都支持；body 里的 `i` 也支持，但 OAuth 的 `/oauth/authorize`（GET）只能用头。
3. **`signin` 的失败体只有 `error.id`**，没有 `code`；其它端点才有完整信封。前端错误处理要两条路径都覆盖。
4. **`users/show` 不支持 `username` + `host`**：查远端用户必须用 `userId`。
5. **`users/notes` 的 `withReplies` 默认是 `true`**（与 Misskey 相反），"只看原创"要显式传 `false`；`includeReplies` 完全不生效。
6. **`i/update` 的响应不含置顶列表**（`pinnedNoteIds`/`pinnedNotes` 为 `[]`），整对象替换会丢置顶；`i`、`i/pin`、`i/unpin` 的响应是完整的。
7. **`avatarId` 在所有响应里恒为 `null`**，头像只能用 `avatarUrl`；`bannerUrl` 恒为 `null`，横幅不可用。
8. **分页游标**：`users/followers`/`users/following` 的 `sinceId`/`untilId` 是**关注行 id**，`users/notes` 的是**笔记 id**；两者的排序翻转规则也不同（前者 `sinceId` 单独出现会升序，后者恒降序）。
9. **错误码判断建议按 `code` + HTTP 状态**，不要只按 `id`：`signup` 系列共用一个 id，`NO_SUCH_FILE` 复用了 `INVALID_PARAM` 的 id。
10. **匿名可读**的端点：`signup`、`signin`、`username/available`、`users/show`、`users/notes`、`users/followers`、`users/following`、`/.well-known/oauth-authorization-server`、`/oauth/token`。带无效 token 访问"可选登录"端点不会 401，只会退化为匿名。

## 附录 B：本命名空间里"未返回"的 Misskey 字段

| Misskey 字段 | 本实现 |
|--------------|--------|
| `UserDetailed.url` / `.uri` | 返回空串 `""`，不是可点击地址 |
| `UserDetailed.bannerUrl` / `.bannerBlurhash` | 恒 `null` |
| `UserDetailed.fields` / `.verifiedLinks` | 恒 `[]` |
| `UserDetailed.location` / `.birthday` / `.lang` | 恒 `null` |
| `UserDetailed.isLocked` | `users.is_locked`（第四十七波起读取真值） |
| `UserLite.isBot` / `.isCat` | 恒 `false` |
| `UserLite.onlineStatus` | 恒 `"unknown"` |
| `UserLite.instance` | 恒 `null` |
| `MeDetailed.avatarId` / `.bannerId` | 恒 `null` |
| `MeDetailed.email` / `.emailVerified` | 恒 `null` / `false`（仅"原生 token"请求出现） |
| `MeDetailed.twoFactorEnabled`、`usePasswordLessLogin`、`securityKeys` | **未返回**（字段不存在） |
| `MeDetailed.mutedWords` / `.mutedInstances` / `.pinnedPageId` 等设置 | 恒空/`null` |
| `Note.emojis` / `.reactionEmojis` | 恒 `[]` |
| `Note.clippedCount` | 恒 `0` |
| `Note.channel` 等频道相关字段 | 未返回（无频道功能） |
| `DriveFile.thumbnailUrl` / `.webpublicUrl` / `.blurhash` / `.comment` | 恒 `null` |
| `Following` 的对侧用户（`users/followers` 里没有 `followee`，反之亦然） | 未返回 |
| `signin` 的 `finished:false` 中间态、`next` 字段 | 未返回（永远一次成功） |
| OAuth 的 `expires_in`、`refresh_token` | 未返回 |

## 附录 C：本文未覆盖的部分

- `notes/*`、时间线、通知、drive、emoji、registry、admin、ActivityPub、Streaming 属于其它命名空间的文档，本文只在被 `users/notes` 复用时描述了 `Note`/`DriveFile` 打包。
- `internal/account/registry.go`（`i/registry/*` 的服务层）属于 `admin-emoji-registry` 命名空间，不在本文范围。
- 未做运行时抓包验证：本文所有结论来自源码阅读；本仓库相关行为有测试覆盖（`internal/httpapi/server_db_test.go:142` 起的 signup/signin/i 测试、`i_update_test.go`、`i_pin_test.go`、`users_test.go`、`users_relations_test.go`、`following_test.go:239-429`），可作为二次核对入口。

## 2026-11-06 增补：应用授权、令牌撤销与实例静音

### POST /api/i/apps

需登录。请求 `{ sort? }`，`sort` 取 `+createdAt` / `-createdAt` / `+lastUsedAt` / `-lastUsedAt`，
默认按 id 升序。返回数组（无分页）：

```json
[{
  "id": "令牌 id（即 revoke 用的 tokenId）",
  "name": "应用名",
  "description": "描述",
  "iconUrl": null,
  "permission": ["read:notes", "…"],
  "createdAt": "…",
  "lastUsedAt": "… 或 null"
}]
```

只列 `access_tokens` 里的第三方令牌；原生 16 位 token 与会话不出现在列表里。

### POST /api/i/revoke-token

请求 `{ tokenId }`，返回 204。只能删除调用者自己的令牌；不存在或属于他人的 id 静默 no-op
（与 Misskey 一致）。删除后该令牌的后续请求返回 401。

### POST /api/i/update 的 `mutedInstances`

新增可选字段 `mutedInstances: string[] | null`：

- `null` 或 `[]` 清空；
- 每项按主机名校验：转小写、去空白、去重，拒绝带 scheme/路径/`@`/空格的项，最多 200 条；
- 存储位置是 `user_profiles.settings` 的 `mutedInstances` 键，不需要新迁移；
- `POST /api/i` 与 `i/update` 的响应回读该数组，客户端据此渲染设置页；
- 所有时间线在登录状态下按 `n.user_host` 过滤掉被静音实例的帖子。

### POST /api/users/search（见 notes-core.md 增补）

用户搜索按用户名、昵称与简介匹配，支持 `origin`（combined/local/remote）与 `limit`/`offset`，
并额外接受 `sinceId`/`untilId` 供游标分页。带凭据时排除双向屏蔽，结果带三个计数。

## 2026-11-06 增补：账号导出与注销

### POST /api/i/export

需登录。无参数。响应是 `application/json` 附件
（`Content-Disposition: attachment; filename="cinderfish-export-<user>-<timestamp>.json"`），
结构与 `cinderfish-account-export@1` 一致：

```json
{
  "exportedAt": "…",
  "format": "cinderfish-account-export@1",
  "user": { /* MeDetailed，不含 token */ },
  "notes": [{ "id", "text", "cw", "visibility", "replyId", "renoteId", "fileIds", "tags", "createdAt" }],
  "followings": [{ "username", "host", "createdAt" }],
  "followers":  [{ "username", "host", "createdAt" }],
  "favorites":  [{ "noteId", "createdAt" }],
  "driveFiles": [{ "name", "type", "size", "createdAt" }]
}
```

帖子只含本人的（含私密与关注者可见），不含他人数据；网盘只导元数据。前端通过 `apiDownload()`
读取 Blob 并触发浏览器下载。

### POST /api/i/delete-account

请求 `{ password }`，需登录。

| 结果 | 条件 |
|------|------|
| 204 | 密码正确且不是最后一个未冻结管理员；账号、帖子、关系、收藏、网盘对象随之删除 |
| 403 `INVALID_PASSWORD` | 密码错误 |
| 400 `LAST_ADMIN` | 调用者是最后一个未冻结管理员 |

注销是即时的硬删除，token 立即失效。前端在成功后调用 `signOut()` 并回到首页。

## 2026-11-06 增补：Webhook（i/webhooks/*）

迁移 v12 新增 `webhooks` 表。端点与上游同名，载荷与请求头也沿用上游的投递格式。

### POST /api/i/webhooks/create

`{ name, url, secret, on }`：

- `name` 1–100 字符；`url` 必须 `http(s)` 且 ≤1024；`secret` 1–1024；
- `on` 取自 `mention/unfollow/follow/followed/note/reply/renote/reaction`，非空、去重；
- 返回 `{ id, userId, name, url, secret, on, active, createdAt, latestSentAt, latestStatus }`。

### i/webhooks/list | show | update | delete

- `list`：当前账号的全部 Webhook，无分页；
- `show`：`{ webhookId }`，不属于当前账号返回 404 `NO_SUCH_WEBHOOK`；
- `update`：`{ webhookId, name, url, secret, on, active }`（全量字段）；
- `delete`：`{ webhookId }`，204；别人的 id 也是 204（无副作用）。

### 投递

发帖成功后异步 POST 到订阅了该事件的启用 hook（5 秒超时）：

```json
{ "hookId": "…", "userId": "…", "eventId": "…", "createdAt": "…", "type": "note|reply|renote", "body": { /* Note */ } }
```

请求头：`Content-Type: application/json`、`User-Agent: Cinderfish-Hooks`、
`X-Firefish-Hook-Id`、`X-Firefish-Hook-Secret`。结果回写 `latestSentAt` 与 `latestStatus`
（网络错误为 1）。失败不重试，也不影响发帖。

## 2026-11-06 增补：MiAuth 轮询（miauth/{session}/check）

### POST /api/miauth/gen-token（登录）

批准会话并签发 token；请求 `{ session, name?, description?, iconUrl?, permission? }`，返回
`{ token }`。token 关联 `session`，供应用轮询取走。

### POST /api/miauth/{session}/check（无需登录）

- 首次：`{ "ok": true, "token": "<明文 token>", "user": { /* UserDetailed */ } }`，
  同时把该令牌标记为 `fetched`；
- 已取走 / 从未批准的 session：`{ "ok": false }`（HTTP 200，不是错误）。

会话字符串本身就是能力凭据，端点只应答一次；迁移 v13 的 `access_tokens.fetched` 保证第二次轮询
不会再次泄露 token。

## 2026-11-06 增补：公开用户目录（POST /api/users）

匿名可读；带凭据时每行补 `isFollowing` / `isFollowed`。

| 参数 | 取值 | 说明 |
|------|------|------|
| `limit` / `offset` | | 默认 10 / 0，上限 100 |
| `sort` | `+follower` `-follower` `+createdAt` `-createdAt` `+updatedAt` `-updatedAt` | `+` 为倒序 |
| `state` | `all` `alive` `admin` `moderator` `adminOrModerator` | 无 moderator 标记，按 admin 处理 |
| `origin` | `combined` `local` `remote` | 缺省 local |
| `hostname` | string | 精确匹配 |

冻结账号不返回。返回 `UserDetailed[]`（含三个计数）。非法 `state`/`origin` 400。
**分页只支持 offset**：`sinceId`/`untilId` 会被忽略。

## 2026-11-06 增补：账号导入（POST /api/i/import）

需登录。请求体 `{fileId}`：文件必须是当前账号云盘里的 JSON，且 `format` 为
`cinderfish-account-export@1`（`i/export` 的输出）。文件上限定为 32 MiB。

行为（单事务）：

1. **笔记**：保留原 id；已存在同 id 时跳过；`replyId`/`renoteId` 只在目标已在库中或本次已导入时保留，
   否则置空；正文、标签、附件 id 与 `createdAt` 按导出恢复。
2. **关注**：按 `username + host` 查找本库用户，找不到则跳过；已关注不重复。
3. **收藏**：笔记存在才写入，重复收藏不重复。

响应 200（Misskey 返回空体，这里额外给计数）：

```json
{ "notes": 12, "followings": 3, "favorites": 5,
  "skipped": 1, "skippedFollowings": 0, "skippedFavorites": 2 }
```

错误：`NO_SUCH_FILE`（文件不存在/不属于自己）、`INVALID_PARAM`（不是 JSON、格式不符、缺 `fileId`）、
`AUTHENTICATION_FAILED`。导入是增量操作，不删除任何现有数据；没有限流。

## 2026-11-06 增补：邮箱与找回密码（第三十八波）

迁移 v18 新增 `user_profiles.email_verified` 与 `email_challenges`（`id`、`user_id`、`purpose`、
`email`、`code`、`created_at`、`expires_at`、`used_at`，`code` 唯一索引）。两种用途互不通用：
`verify` 用于邮箱验证，`reset` 用于密码重置——一条重置链接不能当验证链接用，反之亦然。

邮件发送走 `internal/mail`：配置了 `mail.host` 时用 SMTP（`starttls` / `tls` / `none` 三种），
没配置时把整封邮件打到进程日志，运营者可以从日志里复制链接完成初始化。链接格式为
`<实例根 URL>/verify-email/<code>` 与 `<实例根 URL>/reset-password/<code>`，code 是 32 字节
随机数的十六进制（64 字符）。

### POST /api/email-address/available（匿名）

`{ email }`。格式非法返回 `{"available": false, "reason": "format"}`；被其它账号占用返回
`{"available": false, "reason": "used"}`；可用返回 `{"available": true}`（不带 `reason`）。
从不返回错误码，便于注册表单边输边查。

### POST /api/i/update-email（需登录）

| 参数 | 校验 |
|------|------|
| `password` | 必填；错误时 403 + id-only 信封 `932c904e…`（与 signin 同一 id） |
| `email` | 必填；`account.ValidateEmail`：单 `@`、域名含点、无空白与 `,;<>`、≤254 字符 |

成功后写入 `user_profiles.email` 并把 `email_verified` 置 `false`，然后给新地址发验证链接
（`verify` 用途，1 小时有效），返回 `MeDetailed`（`includeSecrets` 恒 true，所以能看到新的
`email` 与 `emailVerified:false`）。地址被占用返回 400 `EMAIL_ALREADY_REGISTERED`。
**地址已保存但发信失败**时返回 500，且不回滚地址——客户端可以重发。

### POST /api/i/email-verification（需登录）

`{ email? }`。给已登记地址重发验证链接。没有登记地址 → 400 `INVALID_PARAM`；给了 `email`
但与登记值不匹配（忽略大小写）→ 400 `INVALID_PARAM`。成功 204。同一用途 60 秒内重复请求
返回 429 `RATE_LIMIT_EXCEEDED`。

### POST /api/verify-email（需登录）

`{ code }`。消费 `verify` 用途的链接并把当前账号的 `email_verified` 置 `true`，返回 204。
链接失效/已用过 → 400 `INVALID_EMAIL_CODE`；链接属于另一个账号 → 同样 400（code 已被消费）；
链接发出后地址又被改过 → 400 `INVALID_EMAIL_CODE`（旧链接不能证明新邮箱）。

### POST /api/request-reset-password（匿名）

`{ username, email }`。**恒 204**：账号不存在、地址不匹配、被节流、甚至发信失败都不暴露差别，
避免枚举。命中时发 `reset` 用途链接（24 小时有效）。同一账号 60 秒内只发一封。

### POST /api/reset-password（匿名）

| 参数 | 校验 |
|------|------|
| `token` | 必填；即邮件链接里的 code |
| `password` | 必填；仅要求非空（与 `i/change-password` 一致），bcrypt cost 8 |

成功：换密码、把 `email_verified` 置 `true`（收到链接本身就是控制权证明）、返回 204。
链接失效/已用过 → 400 `INVALID_TOKEN`。不吊销已签发的 access token（与 `i/change-password`
行为一致），吊销靠 `i/revoke-token`。

### 已知限制

- 注册流程不接受邮箱（`emailRequiredForSignup` 恒 `false`），邮箱只能在登录后补填。
- 没有「重新发送」冷却以外的限流，也没有按 IP 的发送配额。
- 不支持 HTML 邮件与多语言模板：正文是固定中文纯文本。

## 2026-11-06 增补：users/stats（第四十三波）

`{userId?}` → `{notesCount, repliesCount, renotesCount, repliedCount, renotedCount, pollVotesCount, pollVotedCount}`。

三个笔记桶**互斥**：`notesCount` 只数既不是回复也不是转贴的笔记，`repliesCount` 数回复，
`renotesCount` 数转贴，所以三者相加等于该账号的全部笔记（未删除）。`repliedCount` 是「自己的笔记
被多少条不同的笔记回复过」，`renotedCount` 同理；`pollVotesCount` 是自己投出的票数，
`pollVotedCount` 是自己的投票收到的票数。全部从现有表聚合，删除笔记后立刻反映。

`userId` 省略时取调用者（需要凭据），这是给账号信息页用的便利；上游要求必填。
其它所有数字都是公开信息，因此端点不要求登录。

## 2026-11-06 增补：邮件通知（第四十八波）

`i/update` 现在接受两个邮件通知字段（都存进 `user_profiles.settings`）：

| 字段 | 校验 | 效果 |
|------|------|------|
| `emailNotificationTypes` | 数组，取值 `mention` / `reply` / `follow` / `receiveFollowRequest`，其它 400 | 命中的通知同时发邮件 |
| `receiveAnnouncementEmail` | 布尔 | 管理员发布公告时给该账号发一封 |

发送规则：只有**邮箱已验证**且启用了对应类型时才发。邮件在后台 goroutine 里发送（单封 30 秒超时、
公告群发 2 分钟），失败只记日志，绝不影响产生通知的请求。站内通知比邮件更宽——转发、引用、反应
只进站内列表。公告群发在 `admin/announcements/create` 成功后触发，收件人由
`account.AnnouncementMailRecipients` 从 settings 里筛出（已验证 + 已订阅）。
