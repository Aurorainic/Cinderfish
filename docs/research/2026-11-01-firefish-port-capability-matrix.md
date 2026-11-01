# Firefish 前端整体移植：能力矩阵（2026-11-01）

用途：把「Firefish 页面要什么」对回「Cinderfish 现在有什么」，给出首批白名单的端点、字段、状态与降级行为，
让后续每个移植任务都能拿到一份可验收的清单，而不是边写边发现端点不存在。

配套文档：`2026-11-01-firefish-port-source-and-license-register.md`（版本与许可证）、
`2026-11-01-runtime-adaptation-contract.md`（shim 归属）。
上游设计文档 §8.3 要求本文覆盖：Firefish 页面/组件、所需端点与响应字段、当前状态、降级与验收场景。

## 1. 输入与方法

| 输入 | 具体来源 | 本轮实测 |
|------|----------|----------|
| A. Firefish 页面与 API 引用 | `tmp/refcheck/firefish@02c31e0:packages/client/src` | `pages/` 187 个 `.vue`，客户端引用 **157 个不同 `os.api` 端点**，另有 8 处 `apiGet` |
| B. Cinderfish 现有路由 | `internal/httpapi/server.go:98-176` | 35 个 `POST /api/*` 路由；另有 `GET /api/ping`、`GET`+`POST /api/admin/federation/allowlist`、`GET /media/{path...}`、ActivityPub 路由、`GET /streaming`、OAuth 三端点 |
| C. Misskey 契约 | `tmp/refcheck/misskey@c05e844:packages/backend/src/server/api/endpoints` | 只用于核对路径与字段，本轮不新增端点 |
| D. 已完成的缺口审计 | `docs/HANDOVER.md` §11.15 | 39 条分组的端点缺口（6 已实现 / 15 未实现 / 12 需 schema / 6 上游没有） |

### 1.1 端点提取口径（数字必须带口径）

只匹配 `os.api(` 字面量会漏三类真实调用：Firefish 的 `api` 帮助函数、`endpoint = "…"` 赋值（四条时间线与
`notes/mentions` 就是这样写的）、以及 XHR/fetch 直连（上传）。本轮实测的三种口径：

| 口径 | 数值 | 命令 |
|------|------|------|
| `os.api(` 调用点 | 329 | `grep -rho 'os\.api(' … \| wc -l` |
| 不同 `os.api` 端点 | 157 | 下表大部分行用这个口径 |
| `os.api` + `apiGet` + `apiWithDialog` 去重 | 220 | 正则扩到三个成员 |
| 再加 `api(` 帮助函数、`endpoint = "…"`、`fetch(apiUrl + "/…")` | **277** | **对账用这个数** |

```bash
FF=tmp/refcheck/firefish/packages/client/src
{ grep -rhoE "(os\.)?(api|apiGet|apiWithDialog)\(\s*[\"'][^\"']+" $FF
  grep -rhoE "endpoint\s*[:=]\s*[\"'][^\"']+" $FF
  grep -rhoE "fetch\(apiUrl \+ \"/[a-zA-Z0-9/_.-]+" $FF
} | grep -oE "[a-zA-Z0-9][a-zA-Z0-9/_.-]*$" | sort -u | wc -l   # 277
grep -rn "useChannel(" $FF                        # 频道使用点
grep -n 'HandleFunc(' internal/httpapi/server.go  # Cinderfish 路由
```

两个已知的提取盲区，必须人工补：

- **三元表达式里的端点**：`MkPostForm.vue:1270` 的 `os.api(postData.editId ? "notes/edit" : "notes/create", …)`，
  任何按字面量提取的正则都抓不到 `notes/edit`。
- **XHR 直连上传**：`scripts/upload.ts:92`、`MkCropperDialog.vue:90` 用 XHR 打 `drive/files/create`，
  只因另有 `os.api` 调用点才没漏。

口径说明：正则边界不同会差 ±5（一次独立审计用略宽的写法得到 222 / 274）。对账一律用上面这条命令，
它可重复执行；差异只影响「还有多少没覆盖」的绝对值，不影响 §3 的逐行状态判定。

状态四档（与设计文档 §8.3 一致）：

| 状态 | 含义 | 处理 |
|------|------|------|
| **已支持** | Go 路由存在、响应字段够用、语义一致 | 直接接 |
| **需适配** | 路由存在，但请求/响应形态、分页、错误或字段与本项目不同 | 改客户端适配层，不动后端 |
| **需新增后端** | Firefish 页面依赖，但 Cinderfish 没有对应端点或数据 | 本轮**不做**，页面降级；另开后端任务 |
| **隐藏** | 依赖 Firefish 后端专属能力或非目标功能 | 从导航和路由中移除，不留空壳 |

## 2. 首批白名单

白名单口径来自设计文档 §8.3：登录、用户、时间线、Note、发帖、反应、通知、媒体、Profile、Settings、主题、Deck。

| # | 能力 | Firefish 入口 | 已支持 | 需适配 | 需新增后端 | 隐藏 |
|---|------|---------------|--------|--------|------------|------|
| 1 | 登录/注册 | `pages/auth.vue`、`components/MkSignin.vue`、`MkSignup.vue`、`pages/miauth.vue` | `signin`、`signup`、`username/available`、`users/show` | `miauth/gen-token` | `auth/session/show`、`auth/accept`、`auth/deny`、`miauth/{session}/check` | 通行密钥、TOTP、多账户、`email-address/available` |
| 2 | 当前用户 `me` | `me.ts`、`account.ts` | `i`、`i/update`、`i/pin`、`i/unpin` | 单账户化 | — | 账户切换菜单 |
| 3 | 时间线 | `pages/timeline.vue`、`components/MkTimeline.vue` | 四条时间线端点 + 四频道 | 分页形态 | `notes/mentions` | 推荐/列表/天线/频道 tab、`drive/files/attached-notes`、`notes/recommended-timeline` |
| 4 | Note 详情 | `pages/note.vue`、`components/MkNote*.vue` | `notes/show`、`users/notes`、`notes/reactions/*` | `notes/unrenote`（→`notes/delete`） | `notes/conversation`、`notes/replies`、`notes/renotes`、`notes/children` | `notes/edit`、`notes/clips`、`notes/state`、`notes/translate` |
| 5 | 发帖 | `components/MkPostForm*.vue` | `notes/create`、`drive/files/create`、`drive/files/show`、`drive/files` | 附件上传走 XHR | `notes/polls/vote` | 投票写入、`notes/edit`、`notes/make-private`、草稿同步 |
| 6 | 反应 | `components/MkReactionsViewer*.vue` | `notes/reactions/create`、`notes/reactions/delete` | — | `notes/reactions`（反应者列表） | 反应者浮层逐人列表 |
| 7 | 通知 | `pages/notifications.vue`、`components/MkNotifications.vue` | `notifications/mark-all-as-read` | `i/notifications` 路径（后端 alias） | — | 消息类通知、`i/read-all-*` |
| 8 | 媒体 | `components/MkDrive*.vue`、`pages/settings/drive.vue` | `drive/files`、`drive/files/create`、`drive/files/show`、`drive/files/delete`、`GET /media/*` | — | `drive/files/update`、`drive/folders/*` | `drive/files/upload-from-url`、`drive/stream` |
| 9 | Profile | `pages/user/index.vue`、`components/MkInfo.vue` | `users/show`、`users/notes`、`following/create`、`following/delete` | 计数/字段空缺 | `users/followers`、`users/following`、`users/search` | `following/requests/*`、`blocking/*`、`mute/*` |
| 10 | Settings | `pages/settings/*.vue`（39 页） | `i/update`（3 字段）、`notifications/mark-all-as-read` | 8 个 tab 中按字段裁剪 | `i/registry/*` 等 | 其余 27 页 |
| 11 | 主题 | `themes/`、`theme-store.ts` | 纯客户端（`data-theme` + token） | token 语义映射 | — | 主题编辑器、自定义 CSS |
| 12 | Deck | `ui/deck.vue`、`pages/settings/deck.vue`、`widgets/` | 纯客户端（列容器 + localStorage） | 列配置 UI | `i/registry/*`（本轮不做） | 依赖缺失端点的 widget（§3.12） |

「需新增后端」的条目在 §10 汇总，全部**不在本轮实现**，页面按 §3 的降级列处理。
两个重叠数不要混用：全量 **277** 个引用端点中，**26 个**与现有 Go 路由同名（其中 `drive/files/create` 是 XHR 直连；
去掉它、只算 `os.api` + `endpoint = "…"` 口径时为 **25 个**）；首批白名单的 **43 个**端点中，**17 个**已有精确路由，
其余 26 个缺失（其中 2 个只需路径/语义适配）。
两处会直接 404 的点必须先处理：`i/notifications`（后端兼容 alias）与 `notes/unrenote`（映射到 `notes/delete`）。

## 3. 逐项矩阵

### 3.1 登录与注册（`## 1`）

| Firefish 调用 | 位置 | Cinderfish 路由 | 状态 | 说明 |
|---|---|---|---|---|
| `signin` | `components/MkSignin.vue:224`（通行密钥）、`:254`（TOTP 第二段）、`:271`（密码） | `POST /api/signin` | 已支持（密码分支） | 后端恒返回 `{finished: true, id, i}`（`internal/httpapi/account.go:118,167`），没有 TOTP 与 WebAuthn |
| `signup` | `components/MkSignup.vue:486` | `POST /api/signup` | 已支持 | 注册模式（open/invite/admin）由 `meta` 决定，见 §3.10 |
| `username/available` | `components/MkSignup.vue:392` | `POST /api/username/available` | 已支持 | |
| `signin`（注册后自动登录） | `components/MkSignup.vue:503` | `POST /api/signin` | 已支持 | |
| `miauth/gen-token` | `pages/miauth.vue:109`、`pages/settings/api.vue:38` | `POST /api/miauth/gen-token` | 需适配 | 后端只发 token，没有 `/api/miauth/{session}/check` 轮询端点 |
| `auth/session/show` | `pages/auth.vue:69` | — | 需新增后端 | 第三方应用授权页，属 MiAuth 授权流程 |
| `auth/accept` / `auth/deny` | `pages/auth.vue:75`、`pages/auth.form.vue:53,61` | — | 需新增后端 | 同上 |
| `email-address/available` | 注册表单 | — | 隐藏 | 本项目不发邮件，字段直接不渲染 |

降级行为：登录页只保留「用户名 + 密码」；`MkSignin` 的通行密钥分支与 `MkSignup` 的邮箱验证分支整体删除；
`pages/miauth.vue`、`pages/auth.vue`、`router.ts:382` 的 `/miauth/:session` 不注册路由。
账号存储改为单 token（`account.ts:53-66` 的 `getAccounts/addAccount/removeAccount` 多账户模型不移植）。

### 3.2 当前用户 `me`（`## 2`）

| Firefish 调用 | 位置 | Cinderfish 路由 | 状态 |
|---|---|---|---|
| `i` | `account.ts` 的 `refreshAccount()`（`:123`）与 `meUpdated` 流事件 | `POST /api/i` | 已支持（MeDetailed，含 `unreadNotificationsCount`/`hasUnreadNotification`，见 HANDOVER §11.15.2） |
| `i/update` | `pages/settings/general.vue` 等 9 处 | `POST /api/i/update` | 已支持 |
| `i/pin` / `i/unpin` | 个人页置顶 | `POST /api/i/pin`、`/api/i/unpin` | 已支持 |

适配：`me.ts` 的 `reactive(JSON.parse(localStorage.account))` 多账户结构换成单账户 store；
`instance.ts` 的 `updateInstanceCache()` 调 `api("meta", { detail: true })` → 对应 `POST /api/meta`。

### 3.3 时间线（`## 3`）

`components/MkTimeline.vue` 的 tab 到 endpoint/channel 映射（行号为实测）：

| tab | 端点 | 频道 |
|-----|------|------|
| 主页 | `notes/timeline`（`:143`） | `homeTimeline`（`:231`）+ `main`（`:234`） |
| 本地 | `notes/local-timeline`（`:150`） | `localTimeline`（`:236`） |
| 推荐 | — | `recommendedTimeline`（`:240`） |
| 社交 | `notes/hybrid-timeline`（`:164`） | `hybridTimeline`（`:244`） |
| 全局 | `notes/global-timeline`（`:171`） | `globalTimeline`（`:248`） |
| 列表 | `notes/user-list-timeline` | `userList`（`:252`） |
| 频道 | `channels/*` | `channel`（`:256`） |
| 天线 | `antennas/*` | `antenna`（`:227`） |

| 端点 | Cinderfish 路由 | 状态 | 降级 |
|---|---|---|---|
| `notes/timeline` | `POST /api/notes/timeline` | 已支持 | — |
| `notes/local-timeline` | `POST /api/notes/local-timeline` | 已支持 | — |
| `notes/hybrid-timeline` | `POST /api/notes/hybrid-timeline` | 已支持 | — |
| `notes/global-timeline` | `POST /api/notes/global-timeline` | 已支持 | — |
| `antennas/list`、`users/lists/list` | — | 隐藏 | 时间线 tab 配置里去掉天线/列表，`pages/timeline.vue` 的 tab 编辑只留四类 |
| `notes/user-list-timeline` | — | 需新增后端 | 同上，tab 不出现 |
| `notes/mentions` | — | 需新增后端 | `MkTimeline.vue:178-180` 的 mentions 源。四条主时间线不受影响；通知页的 mention 过滤可用 `i/notifications {includeTypes:["mention"]}` 替代，不新增端点 |
| `drive/files/attached-notes` | — | 隐藏 | `MkTimeline.vue:195` 的文件时间线，依赖 `attached-files` 页（已隐藏） |

适配点：分页参数用 `limit` + `sinceId`/`untilId`（与 Misskey 一致，`web/src/api/types.ts:163-167` 已有
`TimelineParams`）；流式合并策略从 Firefish 的 `queue` 流改为本项目 `components/TimelineFeed.vue` 的
「有新笔记」浮标，交互语义保留，实现重写。

### 3.4 Note 详情（`## 4`）

| Firefish 调用 | 位置 | Cinderfish 路由 | 状态 | 字段/降级 |
|---|---|---|---|---|
| `notes/show` | `pages/note.vue` | `POST /api/notes/show` | 已支持 | PackedNote：`cw`/`visibility`/`localOnly`/`renoteCount`/`repliesCount`/`reactions`/`reactionEmojis`/`files`/`poll`/`myReaction`（`notes_pack.go:48-58,114-140`） |
| `users/notes` | `pages/note.vue`、`pages/user/index.timeline.vue` | `POST /api/users/notes` | 已支持 | |
| `notes/reactions/*` | 见 §3.6 | 已支持 | | |
| `notes/children` | 会话树 | — | 需新增后端 | 详情页只显示单层父笔记（HANDOVER §11.13） |
| `notes/conversation` | 祖先链 | — | 需新增后端 | 同上，`MkNoteSub` 递归祖先链改为单层 `uniqueId` 父块 |
| `notes/renotes` | 转贴列表 | — | 需新增后端 | 转贴计数可显示，列表不可展开 |
| `notes/replies` | `components/MkNoteDetailed.vue:390` 的回复 tab | — | 需新增后端 | 回复 tab 不渲染；回复仍可从时间线与通知进入 |
| `notes/unrenote` | `components/MkRenoteButton.vue:258`（取消转贴） | `POST /api/notes/delete` | 需适配 | 取消转贴必须映射到 `notes/delete`（转贴行是独立笔记），否则按钮 404。这是首批验收路径之一，不能只标「缺失」 |
| `notes/edit` | `components/MkPostForm.vue:1270`（`postData.editId ? "notes/edit" : …`） | — | 隐藏 | **Misskey 没有 `notes/edit`**（Firefish 专属）。编辑入口必须不渲染，否则照 Firefish 的 composer 做出来就是死链 |
| `notes/state` / `notes/translate` | 详情页附加操作 | — | 隐藏 | 菜单项不渲染 |
| `notes/clips` | 剪藏 | — | 隐藏 | 整体不做 |

### 3.5 发帖（`## 5`）

| Firefish 调用 | Cinderfish 路由 | 状态 | 说明 |
|---|---|---|---|
| `notes/create` | `POST /api/notes/create` | 已支持 | 支持 `text`、`cw`、`visibility`、`localOnly`、`replyId`、`renoteId`、`fileIds`、`poll`（纯 renote 允许无 text） |
| `drive/files/create` | `POST /api/drive/files/create` | 已支持 | multipart 上传，见 `internal/httpapi/drive.go:45` |
| `drive/files/show` | `POST /api/drive/files/show` | 已支持 | |
| `notes/polls/vote` | — | 需新增后端 | 投票**结果**由 `notes/show` 的 `poll` 返回可读，投票动作不可用 → 投票块只读 |
| `notes/renotes` 的引用转贴 | — | 需新增后端 | 引用转贴走 `notes/create {renoteId, text}`，已可用；「转贴菜单按可见性选择」不移植（Task 6 决策） |
| `notes/edit` | — | 隐藏 | Firefish 专属端点，Misskey 无；composer 没有编辑模式（`MkPostForm.vue:1270` 的分支整段删掉） |
| `notes/make-private` | — | 隐藏 | 菜单项不渲染 |
| 草稿 / `i/registry` 同步 | — | 隐藏 | `PostForm` 只用 localStorage 草稿 |

适配点：`MkPostForm` 的可见性下拉是 Misskey 语义（`public`/`home`/`followers`/`specified`），本项目一致；
`localOnly` 开关保留（后端字段存在）。

### 3.6 反应（`## 6`）

| Firefish 调用 | Cinderfish 路由 | 状态 |
|---|---|---|
| `notes/reactions/create` | `POST /api/notes/reactions/create` | 已支持 |
| `notes/reactions/delete` | `POST /api/notes/reactions/delete` | 已支持 |

反应清单直接来自 PackedNote 的 `reactions`（对象：emoji → 计数）与 `myReaction`，不需要单独端点。
降级：Firefish 的「反应者浮层」需要 `notes/reactions`（列表）端点，本项目没有 → 浮层只显示计数，
不提供逐人列表。自定义 emoji 目录在 `POST /api/meta` 里**没有 `emojis` 字段**（`internal/httpapi/meta.go`
不含该键；`users_pack.go` 里的 `emojis: []` 是 User 上的字段，不是实例目录），因此 picker 退化为
Unicode emoji，`reactionEmojis` 恒为 `[]`。这是字段缺失，不是「返回空数组」。

### 3.7 通知（`## 7`）

| Firefish 调用 | 位置 | Cinderfish 路由 | 状态 |
|---|---|---|---|
| `i/notifications` | `components/MkNotifications.vue:80,93` | `POST /api/notifications`（canonical）；`POST /api/i/notifications` 为待注册的兼容 alias | 需适配（后端 alias） |
| `notifications/mark-all-as-read` | `pages/notifications.vue`、`pages/settings/notifications.vue` | `POST /api/notifications/mark-all-as-read` | 已支持 |
| `i/read-all-unread-notes` / `i/read-all-messaging-messages` | `pages/settings/notifications.vue` | — | 隐藏 |

`i/notifications` 是上游路径（`endpoints/i/notifications.ts`，`paramDef` = `limit/sinceId/untilId/sinceDate/untilDate/markAsRead/includeTypes/excludeTypes`，
`res` = `Notification[]`）；本实现以 `POST /api/notifications` 为 canonical，行为已实现（HANDOVER §11.15.3）。
策略见 HANDOVER §11.16：**canonical 不动；后端把 `POST /api/i/notifications` 注册为兼容 alias，两个路径进同一个 handler，
不产生第二套实现。现有客户端继续走 canonical，移植过来的 Firefish 调用由后端 alias 直接承接，不做客户端路径映射**。

**以代码为准，不以旧笔记为准**：`2026-10-30-firefish-ui-gap-analysis.md:786,880` 说 `includeTypes`「解析后丢弃」，
已被实现推翻——`notifications.go:37-70` 解析 `includeTypes`/`excludeTypes` 并传给 `NotificationService.List`，
过滤在 SQL 内生效。按本矩阵排期时不要沿用那条旧结论。

### 3.8 媒体（`## 8`）

| Firefish 调用 | 频次 | Cinderfish 路由 | 状态 | 降级 |
|---|---|---|---|---|
| `drive/files/create` | 2 | `POST /api/drive/files/create` | 已支持 | — |
| `drive/files`（列表） | 2 | `POST /api/drive/files` | 已支持 | — |
| `drive/files/show` | 5 | `POST /api/drive/files/show` | 已支持 | — |
| `drive/files/delete` | 1 | `POST /api/drive/files/delete` | 已支持 | — |
| `drive/files/update` | **11** | — | 需新增后端 | 重命名/说明/NSFW 开关不可用 → Drive 页只读列表 + 删除 |
| `drive/folders/*` | 5 | — | 需新增后端 | 文件夹树整体移除，Drive 是平铺列表 |
| `drive/files/upload-from-url` | 2 | — | 隐藏 | 按钮不渲染 |
| `drive/stream` | 1 | — | 隐藏 | 上传进度用本地状态 |
| `GET /media/{path...}` | — | 已支持 | 媒体 URL 由 `packDriveFile` 生成，无需改写 |

### 3.9 Profile（`## 9`）

| Firefish 调用 | 位置 | Cinderfish 路由 | 状态 | 说明 |
|---|---|---|---|---|
| `users/show` | `pages/user/index.vue` 等 25 处 | `POST /api/users/show` | 已支持 | UserDetailed：`followersCount`/`followingCount`/`notesCount` 由 `applyUserCounts` 实填（`users.go:116-133`）；`followersVisibility`/`chatScope` 等为固定默认值 |
| `users/notes` | `pages/user/index.timeline.vue` | `POST /api/users/notes` | 已支持 | |
| `following/create` / `following/delete` | `components/MkFollowButton.vue` | `POST /api/following/create`、`/api/following/delete` | 已支持 | |
| `users/followers` / `users/following` | 关注/粉丝列表 | — | 需新增后端 | 计数可显示，列表「暂不可用」卡片（HANDOVER §11.15.3） |
| `users/search` / `users/search-by-username-and-host` | 搜索页、@ 自动补全 | — | 需新增后端 | 搜索框只保留本地笔记搜索（若存在），用户搜索隐藏 |
| `following/requests/*` | 关注请求页 | — | 需新增后端 | 锁定账号（`isLocked` 恒 false）→ 无关注请求流程 |

字段空缺（契约完整但值为默认，不伪造）：`avatarBlurhash: null`、`bannerUrl: null`、`fields: []`、
`birthday: null`、`location: null`、`roles: []`、自定义 emoji `emojis: []`（`users_pack.go:25-45,47-90`）。
Profile 组件必须按「字段可空」渲染，不显示空占位。

### 3.10 Settings（`## 10`）

Firefish 有 39 个设置页（`pages/settings/*.vue`）。本项目现有 8 个 tab：profile / appearance / navigation /
widgets / notifications / privacy / account / about（`web/src/router/index.ts:54-101`）。

| Firefish 设置页 | 映射到 | 状态 | 依据 |
|---|---|---|---|
| `profile.vue` | settings/profile | 已支持（受限） | `i/update` 只接受 `name`/`description`/`avatarId`（`i_update.go:66-70`），见下方字段白名单 |
| `general.vue`、`accessibility.vue` | settings/appearance | 需适配 | 语言/字体缩放是本项目已有能力；Firefish 的 `i/update` 偏好项（如 `isBot`、`ffVisibility`）不提交 |
| `privacy.vue` | settings/privacy | 需适配 | 只保留 `i/update` 能写的字段（`description` 之外无隐私字段）→ 大多为只读说明 |
| `notifications.vue` | settings/notifications | 已支持 | `notifications/mark-all-as-read` |
| `navbar.vue` | settings/navigation | 已支持 | 纯客户端 |
| `deck.vue` | settings/appearance（列配置） | 需适配 | 纯客户端，列编辑器待建（§3.12） |
| `account-info.vue` | settings/account | 需适配 | `users/stats` 不存在 → 用量块隐藏 |
| `theme.vue`、`theme.manage.vue`、`theme.install.vue` | settings/appearance（主题选择） | 需适配 | 纯客户端，见 §3.11 |
| `mute-block.vue`、`instance-mute.vue`、`word-mute.vue`、`reaction.vue` | — | 隐藏 | `mute/*`、`blocking/*` 路由不存在，且不在 §4 的保留页面名单 → 不注册页面，不提供假开关（计入 27 个隐藏设置页） |
| `security.vue`、`apps.vue`、`api.vue`、`webhook*.vue`、`2fa*.vue`、`plugin*.vue` | — | 隐藏 | 需要 `i/regenerate-token`、`i/revoke-token`、`i/webhooks/*`、TOTP、AiScript 插件 |
| `import-export.vue`、`migration.vue`、`delete-account.vue`、`email.vue`、`accounts.vue`、`drive.vue`、`custom-css.vue`、`custom-katex-macro.vue`、`statusbar*.vue`、`sounds.vue`、`preferences-backups.vue`、`other.vue` | — | 隐藏 | 依赖 `i/export-*`、`i/import-*`、`ap/show`、`drive/files/update`、`get-sounds`、`i/registry/*` |
| `index.vue` | — | 重写 | 本项目 `SettingsLayout.vue` + `menu.ts` 已有菜单结构 |

注册模式开关（open / invite / admin）由 `POST /api/meta` 的 `disableRegistration`/`emailRequiredForSignup`
驱动，Firefish 的 `MkSignup` 用同一组字段，适配层只需把它们接到注册页的可见性判断。

**`i/update` 的字段白名单**（`i_update.go:60-70`，实测）：只有 `name`（≤50 字符）、`description`（≤500）、
`avatarId` 三个字段会被接受，其余字段不会报错但会被**静默忽略**。因此设置页里下列 Firefish 控件必须隐藏或只读，
否则用户改完「保存成功」但值没变：

| 必须隐藏/只读 | 原因 |
|---|---|
| `bannerId`、`fields`、`birthday`、`location`、`lang`、`movedTo`、`alsoKnownAs` | 后端不写这些列 |
| `isBot`、`isCat`、`isLocked`、`autoSensitive`、`alwaysMarkNsfw`、`carefulBot` | 后端不写 |
| `ffVisibility`、`publicReactions`、`chatScope`、`preventAiLearning` | 后端不写 |
| `pinnedNoteIds` | 有专门的 `i/pin`/`i/unpin`，不要用 `i/update` 改 |
| `email`、`emailNotificationTypes` | 本项目不发邮件 |

这条同样适用于 Profile 编辑页与「设置 → 隐私」：**只渲染三个可写字段**，其余用只读展示。

### 3.11 主题（`## 11`）

Firefish 的 30 个 `themes/*.json5` 用 `props` 声明 token（`accent`、`bg`、`panel`、`fg`、`divider`、`popup`、
`shadow`、`header`、`navBg` 等），并支持 `:darken<10<@accent` 这类派生表达式；运行时编译成 CSS 变量。

| 项 | 本项目现状 | 决策 |
|---|---|---|
| `data-theme` + CSS token | `web/src/styles/main.css` 两套主题共 126 个不同的 `--token:` 定义（`grep -oE "^\s+--[a-zA-Z0-9-]+\s*:" web/src/styles/main.css \| sed -E 's/^\s+//' \| sort -u \| wc -l`） | 保留，作为唯一主题机制 |
| 浅/深/自动跟随 | `web/src/stores/theme.ts:8`（`prefers-color-scheme` 查询）、`:30`（读存储偏好）、`:39`（解析） | 保留 |
| Firefish token 名 | 30 个 json5 文件 | **语义映射**到本项目 token，不引入 JSON5 DSL，不移植 `theme-editor` |
| 主题市场 | `theme.install.vue`（URL 安装） | 隐藏 |

降级：主题选择器只提供「浅色 / 深色 / 跟随系统」；Firefish 的多主题列表不做（删除 `theme-store.ts` 的
自定义主题加载）。

### 3.12 Deck（`## 12`）

Firefish 的 deck 是 `ui/deck.vue` + `ui/deck/*` 列容器 + 22 个 `Mkw*` widget（`widgets/index.ts`），
列配置存在 `pages/settings/deck.vue`，无任何后端调用。

| 项 | 本项目现状 | 状态 |
|---|---|---|
| 列容器与宽高 | `web/src/components/deck/DeckColumn.vue`、`stores/deck.ts`（`cinderfish.deck.columns`） | 已支持 |
| 列内页面弹窗 | `components/ui/PageWindow.vue`、`components/deck/DeckPageWindow.vue` | 已支持 |
| 通用/Deck 模式切换 | `components/ui/NavDrawer.vue:199-201` | 已支持 |
| Deck 列编辑器 | — | 需适配（纯前端新增，无后端） |
| Widget 集 | `web/src/components/widgets/`、`settings/WidgetSettings.vue` | 需适配 |

Widget 取舍：Firefish 注册 22 个（`widgets/index.ts`：Memo、Notifications、Timeline、Calendar、Rss、RssTicker、
Trends、Clock、Photos、DigitalClock、UnixClock、Federation、PostForm、Slideshow、ServerMetric、OnlineUsers、
JobQueue、InstanceCloud、Button、Aiscript、UserList、ServerInfo）。按「依赖端点是否在白名单内」逐个判定：

| 保留 | 依赖 | 隐藏 | 缺失依赖 |
|---|---|---|---|
| Memo | 纯客户端 + `i/update`（可选） | Trends | `hashtags/trend` |
| Notifications | `i/notifications` + `main` 频道 | Rss / RssTicker | 需要 RSS 抓取代理 |
| Timeline | 四端点 + 四频道 | Photos | `drive/files`（可用但无相册聚合） |
| Calendar | 纯客户端 | Federation / InstanceCloud / ServerInfo | `federation/instances`、`stats` |
| Clock / DigitalClock / UnixClock | 纯客户端 | ServerMetric | `stats` |
| PostForm | `notes/create` | OnlineUsers | `get-online-users-count` |
| Slideshow | 纯客户端 | JobQueue | `admin/queue/*` |
| — | — | Button / Aiscript | AiScript 运行时（本轮 drop） |
| — | — | UserList | `users/lists/*` |

> 精确的 widget 逐个判定放在 Deck 移植任务里做，判定依据只看「依赖端点是否在 §2 白名单内」。

## 4. 首批保留页面名单（187 个页面中的 18 项）

这份名单是 §2 白名单的落地形式：**不在名单里的页面一律按 §5 隐藏**，不注册路由、不出现在导航，
避免「隐藏」与「还没排期」混为一谈。

| # | Firefish 页面/组件 | Cinderfish 路由 | 说明 |
|---|---|---|---|
| 1 | `pages/timeline.vue` + `components/MkTimeline.vue`（四 tab） | `/` | 主页/本地/社交/全局 |
| 2 | `pages/note.vue` + `MkNote*` | `/notes/:id` | 单层父帖 |
| 3 | `pages/user/index.vue` + `user/index.timeline.vue` + `MkInfo` | `/@:handle` | 计数与置顶可用，关注/粉丝列表降级 |
| 4 | `pages/notifications.vue` + `MkNotifications.vue` | `/notifications` | `POST /api/notifications`（canonical；`i/notifications` 为后端 alias） |
| 5 | `pages/auth.vue` 的登录部分 + `MkSignin.vue`（去通行密钥/TOTP） | `/login` | 用户名 + 密码 |
| 6 | `pages/signup-complete.vue` 的注册表单部分 + `MkSignup.vue`（去邮箱） | `/signup` | 受 `meta.disableRegistration` 控制 |
| 7 | `pages/settings/index.vue` 的菜单骨架 | `/settings` | 8 个 tab |
| 8 | `pages/settings/profile.vue` | `/settings/profile` | 3 个可写字段 |
| 9 | `pages/settings/general.vue` + `theme.vue` 的主题部分 | `/settings/appearance` | 含字体缩放、深浅/自动 |
| 10 | `pages/settings/navbar.vue` | `/settings/navigation` | 纯客户端 |
| 11 | `pages/settings/deck.vue`（列配置部分） | `/settings/appearance` 内 | 列编辑器待建 |
| 12 | widget 组合（Memo/Timeline/Notifications/Clock/Calendar/Slideshow/PostForm） | 侧栏与 deck | 见 §3.12 |
| 13 | `ui/universal.vue`、`ui/deck.vue`、`ui/visitor.vue` 的壳结构 | `AppShell` / `Sidebar` | 结构参考，实现重写 |
| 14 | `pages/settings/notifications.vue` 的全部已读部分 | `/settings/notifications` | `mark-all-as-read` |
| 15 | `pages/settings/about.vue` 类的关于页 | `/settings/about` | 本项目已有 |
| 16 | `pages/settings/privacy.vue` 的只读说明 | `/settings/privacy` | 无写接口 |
| 17 | `pages/settings/account-info.vue` 的账号信息 | `/settings/account` | 用量块隐藏 |
| 18 | `pages/not-found.vue`、`_error_.vue` 的呈现方式 | `/:pathMatch(.*)*` | 错误页 |

判断口径：**只有在这个表里的页面才进入移植队列**；`pages/` 下其余 169 项（含 `admin/*`、`messaging/*`、
`welcome*`、`gallery/*` 等）在 §5 列出分组理由。

## 5. 首批隐藏清单（白名单外）

以下 Firefish 页面在首批**不注册路由、不出现在导航**，理由写在「原因」列：

| 分组 | Firefish 页面 | 原因 |
|---|---|---|
| 天线/列表/剪辑/频道/群组 | `antenna-timeline`、`user-list-timeline`、`my-antennas`、`my-lists`、`my-clips`、`clip`、`clips`、`my-groups`、`channel*`、`channels` | 端点全缺（`antennas/*`、`users/lists/*`、`clips/*`、`channels/*`、`users/groups/*`） |
| 聊天/消息 | `messaging/*` | `messaging/*` 端点与 `messaging` 频道均缺 |
| 页面/画廊/游戏 | `pages`、`page*`、`gallery/*`、`scratchpad` | `pages/*`、`gallery/*` 端点缺；AiScript 运行时整体 drop |
| 搜索/探索 | `search`、`explore*`、`tag`、`emojis.emoji`、`hashtags` | `hashtags/*`、`users/search` 缺（用户搜索隐藏；本地笔记过滤另行评估） |
| 管理 | `admin/*`、`admin-file` | 本项目有独立管理台（`web/src/views/AdminView.vue`），不移植 Firefish 管理 UI |
| 联邦信息 | `about.federation`、`instance-info`、`federation/instances`、`federation/show-instance` | 需要 `federation/*`、`ap/show` 读端点 |
| 开发工具 | `api-console`、`theme-editor`、`registry*`、`preview`、`mfm-cheat-sheet` | 依赖 `endpoints`、`i/registry/*`，或属开发工具 |
| 引导/入口 | `welcome*`、`verify-email`、`reset-password`、`oauth`、`authorize_interaction` | 依赖邮件、OAuth 授权页、`auth/*` 端点 |
| 账户迁移 | `follow-me`、`follow`、`attached-files`、`favorites`、`note-history`、`announcements`、`drive`、`settings`（39 页中 27 页） | 依赖 `ap/show`、`i/export-*`、`i/read-announcement` 等 |

## 6. Streaming 事件矩阵

Firefish 使用的频道与频道内事件（实测行号）对照 `internal/streaming/channel_main.go:21-26`：

| 频道 | Firefish 使用点 | Cinderfish | 处理 |
|---|---|---|---|
| `main` | `MkNotifications.vue:148`、`MkTimeline.vue:234`、`init.ts:499`、`MkFollowButton.vue:102` | 已支持 | 保留 |
| `homeTimeline` | `MkTimeline.vue:231` | 已支持 | 保留 |
| `localTimeline` | `MkTimeline.vue:236` | 已支持 | 保留 |
| `hybridTimeline` | `MkTimeline.vue:244` | 已支持 | 保留 |
| `globalTimeline` | `MkTimeline.vue:248` | 已支持 | 保留 |
| `recommendedTimeline` | `MkTimeline.vue:240` | 缺 | 推荐 tab 隐藏 |
| `userList` | `MkTimeline.vue:252` | 缺 | 列表 tab 隐藏 |
| `channel` | `MkTimeline.vue:256` | 缺 | 频道 tab 隐藏 |
| `antenna` | `MkTimeline.vue:227` | 缺 | 天线 tab 隐藏 |
| `drive` | `MkDrive.vue:200` | 缺 | Drive 无实时更新 |
| `serverStats` / `queueStats` | `pages/admin/overview*` | 缺 | 管理台不接 |
| `messaging` / `messagingIndex` | 消息页 | 缺 | 消息功能隐藏 |

`main` 频道事件（Firefish 侧监听 vs 本项目发布）：

| 事件 | Firefish | Cinderfish 发布 | 处理 |
|---|---|---|---|
| `notification` | `MkNotifications.vue:149` | `streaming_publish.go:68` | 保留 |
| `mention` | `MkNotification*.vue` | `:55` | 保留 |
| `reply` | | `:37` | 保留 |
| `renote` | | `:43` | 保留 |
| `follow` / `followed` / `unfollow` | `MkFollowButton.vue:102` | `:110-119` | 保留 |
| `meUpdated` | `init.ts:503` | 缺 | shim 在 `i/update` 成功后本地更新；不依赖流事件 |
| `readAllNotifications` | `init.ts:507`、`MkNotifications.vue:150` | **已发布**：`notifications.go:19` 广播 `NullMessage("readAllNotifications")` | 保留，直接接 |
| `unreadNotification` | `init.ts:511` | 缺（客户端事件） | 由 `stores/streaming.ts` 的 `unreadNotifications` 在收到 `notification` 时本地置位 |
| `readNotifications`（按 id 置读） | `MkNotifications.vue:158` | **已处理但语义更粗**：`connection.go:143-149` 对 `readNotification` 执行「该用户全部标读」 | 保留事件名；不实现「按 id 单条置读」，UI 上只有「全部已读」 |
| `unreadMention` / `readAllUnreadMentions` / `unreadSpecifiedNote` / `readAllUnreadSpecifiedNotes` | `init.ts:517-535` | 缺 | 对应徽标隐藏 |
| `unreadMessagingMessage` / `readAllMessagingMessages` / `readAllAntennas` | `init.ts:537+` | 缺 | 消息/天线功能隐藏，事件不订阅 |

## 7. 已存在的强项（移植时不要重写）

- **认证与 token**：Bearer token、`POST /api/i` 返回 MeDetailed 并播种未读通知（`notifications_test.go` 覆盖）。
- **时间线四端点 + 四频道**：真实二进制已验证（HANDOVER §11.15.7）。
- **反应读写与可见性收敛**：`notes/reactions/*`，且转贴不会放大可见性（`9167f1e`）。
- **通知列表与全部已读**：`notifications` + `notifications/mark-all-as-read`。
- **Streaming 基础设施**：单连接 + 频道引用计数 + `main` 频道持有（`web/src/stores/streaming.ts:1-13`），
  可直接被 Firefish 的流式消费点复用。
- **主题系统**：`data-theme` + 126 个 token + 自动跟随 + 无动画偏好（`stores/theme.ts`）。
- **命令式 UI 总线**：`web/src/os.ts` 已有「打开发帖框 / 确认框 / 反应选择器 / 页面窗口」，是 Firefish `os.*` 的落点。

## 8. 验收场景（首批）

| # | 场景 | 判定 |
|---|------|------|
| 1 | 用户名密码登录、刷新后保持登录、登出 | 单账户；`localStorage` 只有一份 token |
| 2 | 主页/本地/社交/全局四条时间线可切换，滚动分页正常 | 四个端点 + 四个频道 |
| 3 | 发帖（文本、CW、可见性、附件）→ 出现在时间线与详情页 | `notes/create` + `drive/files/create` |
| 4 | 回复、纯转贴、引用转贴 | `replyId` / `renoteId` / `renoteId+text` |
| 5 | 添加与撤销反应，计数与自己的反应高亮 | `notes/reactions/*` + `myReaction` |
| 6 | 通知列表、`main` 频道实时插入、全部已读 | `POST /api/notifications`（`i/notifications` 为 alias）+ `mark-all-as-read` |
| 7 | 个人页：资料、计数、置顶、关注/取消关注 | `users/show` + `i/pin` + `following/*` |
| 8 | 设置：改资料、切主题、切导航、改 Deck 列 | 8 个 tab 中已支持项 |
| 9 | 主题自动跟随系统；刷新后保持 | `stores/theme.ts` |
| 10 | 移动端 380px 无横向溢出；宽屏 deck 三列 | 浏览器实测 |

不可验收项（本轮明确不做）：关注/粉丝列表、搜索用户、投票、文件夹、RSS/天线/列表/频道、消息、插件。

## 9. 与旧（渐进式）方案的关系

本矩阵取代 `docs/superpowers/plans/2026-10-29-frontend-redesign/` 里的任务简报口径：
旧计划的目标是「在当前页面上提高信息密度」，本矩阵的目标是「以 Firefish 页面结构为基线重写，端点按表接入」。
旧计划已完成的部分（Task 2–8，见 HANDOVER §11.10–§11.15）作为**后端与 API 侧成果**保留，
前端组件不再按旧结构演进。

## 10. 需新增后端的端点汇总（本轮不做）

按能力分组，全部在此登记以免散落：

| 能力 | 缺失端点 | 影响页面 |
|---|---|---|
| 通知路径对齐 | `POST /api/i/notifications`（canonical `/api/notifications` 的兼容 alias，**最优先**） | 通知（仅路径层面） |
| 第三方授权 | `miauth/{session}/check`、`auth/session/show`、`auth/accept`、`auth/deny` | 登录/授权页 |
| Note 关系 | `notes/children`、`notes/conversation`、`notes/renotes`、`notes/replies`、`notes/mentions`、`notes/reactions`（列表）、`notes/unrenote` | Note 详情、回复 tab、通知 mentions、反应者浮层 |
| 关注关系 | `users/followers`、`users/following`、`following/requests/*` | Profile 列表、锁定账号 |
| 社交图 | `users/search`、`users/search-by-username-and-host` | 搜索、@ 补全 |
| 媒体 | `drive/files/update`、`drive/folders/*`、`drive/files/upload-from-url`、`drive/stream`、`drive/files/attached-notes` | Drive、文件时间线 |
| 偏好与安全 | `mute/*`、`blocking/*`、`i/registry/*`、`i/regenerate-token`、`i/revoke-token`、`i/webhooks/*` | 设置 |
| 数据迁移 | `i/export-*`、`i/import-*`、`ap/show` | 设置 |
| 内容类型 | `notes/polls/vote`、`notes/translate`、`pages/*`、`gallery/*`、`channels/*`、`antennas/*`、`clips/*`、`users/groups/*`、`messaging/*` | 对应页面（已隐藏） |
| 实例信息 | `stats`、`get-online-users-count`、`hashtags/trend`、`federation/instances`、`federation/show-instance` | widget、关于页 |

`notes/unrenote` 与 `i/notifications` 是仅有的两条「不做就会在首批验收路径上 404」的项：
前者本轮用 `notes/delete` 映射，后者由后端注册 `/api/i/notifications` 兼容 alias（canonical 仍是 `/api/notifications`）；
其余都伴随页面降级，不会产生死链。

每一条都可以直接对回 HANDOVER §11.15 的分组表，避免两处口径分叉。

## 11. 字段白名单（防止 UI 显示假数据）

契约里有字段不等于值是真实的。本轮实测的四处「字段存在但值为默认/缺失」，UI 必须按此渲染：

| 端点/类型 | 真实可用 | 缺失或恒为默认 | UI 处理 |
|---|---|---|---|
| `POST /api/i/update` | `name`、`description`、`avatarId` | 其余字段被静默忽略（`i_update.go:66-70`） | 只渲染三个输入框 |
| `users/show`（UserDetailed） | `id`/`username`/`name`/`avatarUrl`/`description`/`isAdmin`/`isSuspended`、`followersCount`/`followingCount`/`notesCount`（`applyUserCounts` 实算）、`pinnedNotes`（`applyPinnedNotes`）、关系四标志（按 viewer 实填） | `avatarBlurhash: null`、`bannerUrl: null`、`bannerBlurhash: null`、`fields: []`、`birthday: null`、`location: null`、`lang: null`、`roles: []`、`chatScope: "followers"`、`canChat: false`、`isLocked: false`、`onlineStatus: "unknown"` | 空值不渲染占位；`onlineStatus` 不显示在线点；`isLocked` 不显示锁图标 |
| `POST /api/meta` | `disableRegistration`、`emailRequiredForSignup`、`maxNoteTextLength`、`miauth` 等 | **没有 `emojis` 字段**（`meta.go` 不含该键） | emoji picker 只有 Unicode；不请求实例 emoji 目录 |
| `MeDetailed` | 同 UserDetailed + `pinnedNoteIds`、`unreadNotificationsCount`、`hasUnreadNotification` | `avatarId: null`、`bannerId: null`、`email` 仅原生 token 请求时返回 | 不显示「更换横幅」入口 |

规则：**缺失字段用空值表达，不用假数据兜底**（设计文档 §8.2）。任何「为了让界面好看而编一个值」的做法都算 bug。
