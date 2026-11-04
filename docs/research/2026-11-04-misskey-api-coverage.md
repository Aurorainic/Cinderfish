# Misskey API 覆盖台账（2026-11-04）

## 0. 背景与范围变更

2026-11-04：前端移植移交给独立 workflow，本仓库的活跃工作面**只剩后端 API 与联邦通讯**。
本文台账是这条工作线的唯一进度依据：任何"补了哪些端点"的结论都以下表的差集为准，
不凭印象。

- 契约来源：`tmp/refcheck/misskey/packages/backend/src/server/api/endpoints/`（Misskey `c05e844`）。
- 本实现的路由：`internal/httpapi/server.go` 及同包各 handler 文件里的 `"POST /api/…"` 注册。
- 统计口径：按 Misskey 端点路径逐一比对（不含 HTTP 方法差异；Misskey 全部为 POST）。

## 1. 总量

| 指标 | 数量 |
|------|------|
| Misskey 端点总数 | 440 |
| Cinderfish 已实现（同名路径） | 35 |
| 未实现 | 405 |
| 仅有 Cinderfish 侧别名/自有路径 | 5（`notifications`、`notifications/mark-all-as-read` 等别名） |

## 2. 已实现（35）

```
admin/accounts/create      admin/show-users          admin/suspend-user
admin/unsuspend-user       drive/files              drive/files/create
drive/files/delete         drive/files/show         following/create
following/delete           i                        i/notifications
i/pin                      i/unpin                  i/update
meta                       miauth/gen-token          notes/create
notes/delete               notes/global-timeline     notes/hybrid-timeline
notes/local-timeline       notes/reactions/create    notes/reactions/delete
notes/renotes              notes/replies            notes/show
notes/timeline             notifications/mark-all-as-read
ping                       username/available        users/followers
users/following            users/notes               users/show
```

另有 Cinderfish 自有别名：`POST /api/notifications`（Misskey 叫 `i/notifications`，两者都注册）。

## 3. 缺口分布（按命名空间，未实现数）

| 命名空间 | 缺口 | 说明 |
|----------|------|------|
| `i/*` | 52 | 账号设置、2FA、导出/导入、注册表、应用授权等 |
| `users/*` | 29 | 列表、搜索、反应、推荐、关注请求、更新资料字段 |
| `notes/*` | 26 | 会话/子帖、提及、投票、收藏、搜索、翻译、草稿 |
| `chat/*` | 25 | 私聊（首版可选） |
| `drive/*` | 16 | 文件夹、URL 上传、去重、批量移动 |
| `channels/*` | 16 | 频道（首版可选） |
| `charts/*` | 12 | 统计图表（可用 SQL 聚合实现） |
| `clips/*` | 11 | 剪贴（首版可选） |
| `flash/*` | 10 | Play（首版可选） |
| `gallery/*` | 9 | 画廊（首版可选） |
| `following/*` | 9 | 关注请求处理、列表、invalidate |
| `pages/*` | 7 | 页面（首版可选） |
| `antennas/*` | 7 | 天线（首版可选） |
| `federation/*` / `hashtags/*` / `roles/*` / `invite/*` / `auth/*` / `sw/*` | 其余 | 视首版范围取舍 |

## 4. Tier 1（下一步要补，按依赖排序）

这些是 Firefish 客户端页面直接依赖、且当前会降级或空白的端点。

| 顺序 | 端点 | 解锁的界面/行为 | 现有降级 |
|------|------|------------------|----------|
| 1 | `following/requests/list`、`following/requests/accept`、`following/requests/reject`、`following/requests/sent` | 关注请求页、通知里的接受/拒绝按钮 | 目前只有入站请求排队，无接口处理 |
| 2 | `users/update` 的 `followersVisibility` / `followingVisibility` | 隐私设置页的两个开关 | 只能直接改库 |
| 3 | `i/favorites`、`notes/favorites/create`、`notes/favorites/delete` | 收藏页、笔记菜单的收藏项 | 无收藏功能 |
| 4 | `notes/conversation`、`notes/children` | 笔记详情页的上下会话串 | 只渲染一层父帖 |
| 5 | `notes/reactions` | 「谁回应了」名单浮层 | 只有计数 |
| 6 | `hashtags/trend`、`hashtags/show` | 趋势小组件、标签页 | 客户端本地统计 |
| 7 | `drive/folders*` | 云盘文件夹 | 扁平文件列表 |
| 8 | `emojis` | 自定义表情渲染 | `:shortcode:` 纯文本 |
| 9 | `notes/polls/vote` + 投票创建 | 投票功能 | 只读、表单隐藏入口 |
| 10 | `i/registry/*` | 客户端设置云端同步 | localStorage |

## 5. Tier 2（首版可选，明确不做要记录理由）

`chat/*`、`channels/*`、`clips/*`、`flash/*`、`gallery/*`、`pages/*`、`antennas/*`、
`reversi/*`、`sw/*`、`roles/*` 等。首版范围见 `docs/research/2026-10-26-initial-scope.md`，
未列入 Tier 1 前不实现。

## 6. 联邦侧现状（与 API 台账分开记）

`internal/federation/inbox.go` 已处理入站 `Follow` / `Accept` / `Reject` / `Undo` /
`Create` / `Delete` / `Like` / `Announce` / `Update` / `Block`，出站 `Follow` 走
`following/create` 投递。未覆盖：`Move`、`Flag`、`Question`（投票）以及
`EmojiReact`（Firefish/Misskey 的扩展反应类型，目前只认 `Like`）。

## 7. 维护方式

补完一批端点后：更新第 2 节列表与第 4 节的"现有降级"列，重跑
`gofmt -l internal/ cmd/`、`go vet ./...`、`go test ./...`，并在
`docs/HANDOVER.md` 追加该批次的验证证据。
