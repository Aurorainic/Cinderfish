# Misskey API 覆盖台账（2026-11-04）

## 0. 背景与范围变更

2026-11-04：前端移植移交给独立 workflow，本仓库的活跃工作面**只剩后端 API 与联邦通讯**。
本文台账是这条工作线的唯一进度依据：任何"补了哪些端点"的结论都以下表的差集为准，
不凭印象。

- 契约来源：`tmp/refcheck/misskey/packages/backend/src/server/api/endpoints/`（Misskey `c05e844`）。
- 本实现的路由：`internal/httpapi/server.go` 及同包各 handler 文件里的 `"POST /api/…"` 注册。
- 统计口径：按 Misskey 端点路径逐一比对（不含 HTTP 方法差异；Misskey 全部为 POST）。

## 1. 总量（2026-11-06 复核）

| 指标 | 数量 |
|------|------|
| Misskey 端点总数 | 440 |
| Cinderfish 已注册路由 | 159 |
| 其中 Misskey 同名路径 | 151 |
| 其中 Cinderfish 自有路径/别名 | 8（`notifications` 别名、`admin/federation/mode`、`admin/federation/allowlist`、`miauth/{session}/check`、`admin/object-storage`、`admin/deliver-queue`、`notes/history`、`oauth/authorize`（同意页调用）） |
| 未实现（Misskey 同名） | 289 |

统计口径：`internal/httpapi/server.go` 里 `HandleFunc("METHOD /api/…")` 的去重路径；
GET 与 POST 同名路径只算一个。

## 2. 已实现（159 条注册路由）

```
admin/accounts/create       admin/announcements/create  admin/announcements/delete
admin/abuse-user-reports    admin/deliver-queue*        admin/get-index-stats
admin/get-table-stats
admin/meta                  admin/object-storage*       admin/update-meta
admin/resolve-abuse-user-report
admin/announcements/list    admin/announcements/update  admin/emoji/add
admin/emoji/delete          admin/emoji/list            admin/federation/allowlist*
admin/drive/show-file       admin/federation/mode*      admin/show-users
admin/suspend-user
admin/unsuspend-user        announcements               ap/show
app/create                  app/show                    auth/accept
auth/session/generate       auth/session/show           auth/session/userkey
blocking/create
blocking/delete             blocking/list               drive/files
drive/files/create          drive/files/delete          drive/files/show
drive/files/upload-from-url
drive/files/update
charts/user/notes           drive                       drive/folders               drive/folders/create
drive/folders/delete        drive/folders/show          drive/folders/update
email-address/available     emojis                      federation/instances
federation/show-instance
get-online-users-count      following/create
server-info                 following/delete            following/delete
following/invalidate        following/list              following/update
following/update-all
following/requests/accept    following/requests/cancel   following/requests/list
following/requests/reject    following/requests/sent     hashtags/list              hashtags/search             hashtags/show
hashtags/trend             hashtags/users
i                           i/apps                      i/delete-account
i/export                    i/favorites                 i/import
i/2fa/done                  i/2fa/password-less         i/2fa/register
i/2fa/unregister            i/change-password           i/regenerate-token
i/signin-history
i/notifications             i/pin                       i/registry/clear
i/registry/get              i/registry/get-all          i/registry/get-detail
i/registry/keys             i/registry/keys-with-type   i/registry/remove
i/email-verification        i/update-email
i/known-as                  i/move
i/read-announcement         i/registry/set              i/revoke-token
i/unpin                     i/update                    i/webhooks/create
i/webhooks/delete           i/webhooks/list             i/webhooks/show
i/webhooks/update
meta                        mute/create
mute/delete                 mute/list
miauth/gen-token            miauth/{session}/check*            notes/children              notes/conversation
notes/create                notes/delete                notes/edit                  notes/favorites/create
notes/favorites/delete      notes/global-timeline       notes/hybrid-timeline
notes/local-timeline        notes/mentions              notes/polls/vote
notes/reactions             notes/reactions/create      notes/reactions/delete
notes/featured              notes/renotes               notes/search
notes/search-by-tag         notes/unrenote
notes/replies               notes/show                  notes/state                 notes/timeline
notifications*              notifications/mark-all-as-read
oauth/authorize*            ping                        request-reset-password
reset-password
signin                      signup                      stats
username/available          users/followers             users/following
verify-email
users                       users/notes                 users/report-abuse
users/stats
users/relation              users/search                users/show
users/get-frequently-replied-users
users/update
admin/drive/files
i/registry/scopes*
notes/history*
```

2026-11-07 复核：从 `internal/httpapi/server.go` 提取的去重路径共 159 条，与本节一致；
此前漏列了 `admin/drive/files`、`i/registry/scopes`、`notes/history`。`i/registry/scopes` 与
`notes/history` 是 Cinderfish 自有路径（Misskey 对应的是 `i/registry/scopes-with-domain`，
且没有 `notes/history`）。

`*` 为 Cinderfish 自有路径/别名：`notifications` 是 `i/notifications` 的兼容别名
（Misskey 客户端读 `i/notifications`）；`admin/federation/mode` 与
`admin/federation/allowlist` 是本项目自有的运行期联邦开关（Misskey 没有同名端点）；
`oauth/authorize`（POST）是授权同意页调用的一次性端点，返回客户端跳转地址而不是 302。
`i/registry/scopes` 对应 Firefish 客户端的调用；Misskey `c05e844` 同位置是
`i/registry/scopes-with-domain`，多一个 domain 分组。

2026-11-04 之后新增（此前台账未记）：`i/favorites`、`notes/favorites/create|delete`、
`notes/conversation`、`notes/children`、`notes/reactions/create|delete`、`i/registry/*`（全部）、
`hashtags/trend`、`drive/folders*`、`emojis`、`admin/emoji/*`、`notes/polls/vote`、
`following/requests/*`、`users/update`、`signin`、`signup`、`username/available`、
`i/registry/scopes`，以及 `admin/federation/mode` 的 GET。

2026-11-06 第十九波新增：`stats`（公开计数）；`admin/update-meta` 增加六个合规 URL 与
`serverRules`（不是新路由）。

2026-11-06 第四十五波新增：`admin/drive/show-file`。

2026-11-06 第四十三波新增：`users/stats`、`hashtags/search`、`i/registry/keys-with-type`、`notes/featured`。

2026-11-06 第四十波新增：`notes/unrenote`（取消转贴）；同时修复出站投递从未接线的问题
（`notes/create` 发 `Create`/`Announce`、`notes/delete` 发 `Delete`/`Undo(Announce)`）。

2026-11-06 第三十八波新增：`email-address/available`、`i/update-email`、`i/email-verification`、
`verify-email`、`request-reset-password`、`reset-password`（迁移 v18：`email_challenges` 与
`user_profiles.email_verified`；`MeDetailed.email`/`emailVerified` 从「恒 null/false」变成真值）。

2026-11-06 第三十七波新增：`admin/announcements/update`（部分更新，`imageUrl: null`/`expiresAt: null` 清空，404 `NO_SUCH_ANNOUNCEMENT`）。

2026-11-06 第三十六波新增：`i/read-announcement`（迁移 v17：`announcement_reads`）；公告列表按登录用户返回 `isRead`。

2026-11-06 第三十五波新增：`drive/files/upload-from-url`（仅 http(s)，拒绝私有/环回目标与重定向，30 秒超时）。

2026-11-06 第三十四波新增：`notes/state`（收藏/置顶状态；线程静音恒 false）。

2026-11-06 第三十二波新增：`federation/instances`（按库内 users.host 聚合，不联网）。

2026-11-06 第三十波新增：`i/import`（读取本站 `cinderfish-account-export@1` 格式）。

2026-11-06 第二十八波新增：`drive`（容量/用量/文件数；未配置配额时 capacity=0）。

2026-11-06 第二十七波新增：`federation/show-instance`（本地取运行时设置与计数，远端只聚合库内数据，不联网抓取）。

2026-11-06 第二十六波新增：`admin/update-meta` 增加 `hiddenTags`（不是新路由），趋势查询按它过滤。

2026-11-06 第二十五波新增：`drive/files/update`（重命名、描述、敏感标记、移入文件夹）。迁移 v16：`drive_files.comment`。

2026-11-06 第二十四波新增：`notes/edit`（Misskey 同名）与 `notes/history`（Cinderfish 自有；Misskey 固定版本没有该端点）。迁移 v15：`notes.updated_at` 与 `note_edits` 表。

2026-11-06 第二十一波新增：`i/change-password`、`i/regenerate-token`、`i/signin-history`。

2026-11-06 第二十波新增：`i/2fa/{register,done,unregister,password-less}`（迁移 v14）。
只实现 TOTP：`password-less` 在没有安全密钥时会返回 `NO_SECURITY_KEY`，与 Firefish 一致；
`i/2fa/register-key|key-done|remove-key|update-key` 仍未实现。

2026-11-06 第十八波新增：`admin/object-storage`、`admin/deliver-queue`（Cinderfish 自有、只读）。

2026-11-06 第十七波新增：`users`（公开目录；只支持 offset 分页）。

2026-11-06 第十六波新增：`notes/mentions`；`i/notifications` 增加 `unreadOnly` 参数（不是新路由）。
台账计数按「440 − 同名已实现」重新对齐（此前累计漂移）。

2026-11-06 第十五波新增：`miauth/{session}/check`（Misskey 在 api/index.ts 里实现，不在 endpoints
目录，故不计入「Misskey 同名端点」分子）；迁移 v13 增加 `access_tokens.fetched`。

2026-11-06 第十四波新增：`admin/update-meta` 增加 `blockedHosts`（不是新路由），策略层在出站与入站
按精确主机拒绝。

2026-11-06 第十三波新增：`admin/meta`、`admin/update-meta`（只覆盖名称/简介/维护者/邮箱/注册模式）。

2026-11-06 第十二波新增：`admin/get-table-stats`、`admin/get-index-stats`（只读）。

2026-11-06 第十一波新增：`i/webhooks/{create,list,show,update,delete}`（迁移 v12；只产生
note/reply/renote 事件）。

2026-11-06 第十波新增：`users/report-abuse`、`admin/abuse-user-reports`、
`admin/resolve-abuse-user-report`（迁移 v11）。

2026-11-06 第九波新增：`i/export`（自有 JSON 格式，非 Misskey zip）、`i/delete-account`。

2026-11-06 第八波新增：`announcements`、`admin/announcements/create|list|delete`（迁移 v10）。

2026-11-06 第七波新增：`admin/drive/files`；`admin/show-users` 增加 sort/state/origin/username/hostname
参数（不是新路由）。

2026-11-06 第六波新增：`notes/search`、`notes/search-by-tag`、`users/search`（均走 ILIKE 回退，
无全文索引；`search-by-tag` 只支持单 `tag` 形式）。

2026-11-06 第五波新增：`mute/create|delete|list`、`blocking/create|delete|list`、`i/apps`、
`i/revoke-token`；同时 `i/update` 增加 `mutedInstances` 字段（不是新路由）。

## 3. 缺口分布（按命名空间，未实现数）

| 命名空间 | 缺口 | 说明 |
|----------|------|------|
| `i/*` | 42 | 账号设置、导入、应用授权等 |
| `users/*` | 28 | 列表、搜索、反应、推荐、关注请求、更新资料字段 |
| `notes/*` | 22 | 会话/子帖、提及、投票、收藏、搜索、翻译、草稿 |
| `chat/*` | 25 | 私聊（首版可选） |
| `drive/*` | 13 | 去重、批量移动 |
| `channels/*` | 16 | 频道（首版可选） |
| `charts/*` | 12 | 统计图表（可用 SQL 聚合实现） |
| `clips/*` | 11 | 剪贴（首版可选） |
| `flash/*` | 10 | Play（首版可选） |
| `gallery/*` | 9 | 画廊（首版可选） |
| `following/*` | 4 | `following/list`、`following/update`、`following/update-all`、`invalidate` |
| `pages/*` | 7 | 页面（首版可选） |
| `antennas/*` | 7 | 天线（首版可选） |
| `federation/*` / `hashtags/*` / `roles/*` / `invite/*` / `auth/*` / `sw/*` | 其余 | 视首版范围取舍 |

## 4. Tier 1（下一步要补，按依赖排序）

> **2026-11-06 更新：Tier 1 的 1–9 项已全部实现**（见 §2：`users/update` 可见性、收藏、
> 会话/子帖、反应名单、趋势、云盘文件夹、自定义表情、投票、`i/registry/*`）。
> 下表保留为当时的需求记录与验收口径。

这些是 Firefish 客户端页面直接依赖、且当前会降级或空白的端点。

> 2026-11-04 已实现并移出本表：`following/requests/list`、`following/requests/sent`、
> `following/requests/accept`、`following/requests/reject`、`following/requests/cancel`
> （见第 2 节）。仍未实现的前置依赖：`following/create` 对远端关注者不会写入待审行，
> 所以 `requests/sent` 与 `cancel` 目前只能作用于库里已有的行。

| 顺序 | 端点 | 解锁的界面/行为 | 现有降级 |
|------|------|------------------|----------|
| 1 | `users/update` 的 `followersVisibility` / `followingVisibility` | 隐私设置页的两个开关 | 只能直接改库 |
| 2 | `i/favorites`、`notes/favorites/create`、`notes/favorites/delete` | 收藏页、笔记菜单的收藏项 | 无收藏功能 |
| 3 | `notes/conversation`、`notes/children` | 笔记详情页的上下会话串 | 只渲染一层父帖 |
| 4 | `notes/reactions           hashtags/trend
drive/folders             drive/folders/create     drive/folders/show
drive/folders/update      drive/folders/delete
emojis                    admin/emoji/list         admin/emoji/add
admin/emoji/delete
notes/polls/vote
i/registry/set            i/registry/get           i/registry/get-detail
i/registry/keys           i/registry/get-all       i/registry/remove
i/registry/clear` | 「谁回应了」名单浮层 | 只有计数 |
| 5 | `hashtags/trend`、`hashtags/show` | 趋势小组件、标签页 | 客户端本地统计 |
| 6 | `drive/folders*` | 云盘文件夹 | 扁平文件列表 |
| 7 | `emojis` | 自定义表情渲染 | `:shortcode:` 纯文本 |
| 8 | `notes/polls/vote
i/registry/set            i/registry/get           i/registry/get-detail
i/registry/keys           i/registry/get-all       i/registry/remove
i/registry/clear` + 投票创建 | 投票功能 | 只读、表单隐藏入口 |
| 9 | `i/registry/*` | 客户端设置云端同步 | localStorage |

## 5. Tier 2（首版可选，明确不做要记录理由）

`chat/*`、`channels/*`、`clips/*`、`flash/*`、`gallery/*`、`pages/*`、`antennas/*`、
`reversi/*`、`sw/*`、`roles/*` 等。首版范围见 `docs/research/2026-10-26-initial-scope.md`，
未列入 Tier 1 前不实现。

## 6. 联邦侧现状（与 API 台账分开记）

`internal/federation/inbox.go` 已处理入站 `Follow` / `Accept` / `Reject` / `Undo` /
`Create` / `Delete` / `Like` / `EmojiReact` / `EmojiReaction` / `Announce` / `Update` /
`Move` / `Block` / `Flag` / `Question`，出站 `Follow` 走 `following/create` 投递。
2026-11-07 第六十六波补齐 `Question`：入站把 `oneOf`/`anyOf` 存进 `polls`，出站把本地投票渲染为
`Question`（见 HANDOVER §12CE）。

2026-11-06 第五十八波修复：入站反应原先只按 `notes.ap_id/uri` 找笔记，而本地笔记这两列为空，
远端用户对本地笔记的 `Like` 一直返回「target note not found」；现在本地笔记按
`<baseURL>/notes/<id>` 解析后再写 `note_reactions`，`EmojiReact` / `EmojiReaction` 与 `Undo`
共用同一条路径。

2026-11-04 起出站侧还多了三种答复，全部走 `deliver_queue`（与 `Undo` 同一条通道，
失败只记日志不回滚本地状态）：

- `following/requests/accept` → `Accept`，内嵌远端关注者发来的原始 `Follow`（用
  `follow_requests.uri`；缺 uri 时退回 `https://<本机>/users/<id>#follows/<followee>`）；
- `following/requests/reject` → `Reject`，同样内嵌原始 `Follow`；
- `following/requests/cancel` → `Undo(Follow)`，与 `following/delete` 共用
  `sendRemoteUndoFollow`。

未实现的入站行为：`follow_requests` 里的 `receiveFollowRequest` 通知、`followRequestAccepted`
通知，以及 `following/create` 对远端关注的目标写入待审行（见第 4 节的说明）。

## 7. 维护方式

补完一批端点后：更新第 2 节列表与第 4 节的"现有降级"列，重跑
`gofmt -l internal/ cmd/`、`go vet ./...`、`go test ./...`，并在
`docs/HANDOVER.md` 追加该批次的验证证据。
