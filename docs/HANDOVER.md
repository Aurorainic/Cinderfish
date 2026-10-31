# Cinderfish 交接说明

- 最后更新：2026-10-31（Iceshrimp 壳层重构，重写计划 Task 2）
- 项目名 **Cinderfish**（2026-10-28 定名，Go module `github.com/Aurorainic/cinderfish`）
- 仓库：当前目录 `/home/lainsaka/文档/MilnFish`，分支 `main`；远程 `origin` 为
  `https://github.com/Aurorainic/Cinderfish.git`（GitHub 仓库已更名，本地目录改名待办）

## 1. 一句话说明

Go 单二进制的个人/极小规模联邦社交实例。后端契约是 **Misskey 原生 API**，通信层是 **ActivityPub**，前端是自建的 **Firefish 风格 Vue 3** 应用。

## 2. 里程碑

| 里程碑 | 状态 | 提交 |
|--------|------|------|
| M1 骨架 | ✅ | `883874c` |
| 源码校准（meta/ping/错误信封） | ✅ | `0ba974c` |
| M2 账户与认证 | ✅ | `1a1abed` |
| M3 笔记/时间线/反应/通知/存储 | ✅ | `f4b8508` |
| M4 Streaming（WebSocket） | ✅ | 见 git log |
| M5 联邦（ActivityPub） | ✅ | `ce7bf11` … `620435d` |
| M6 前端 | ✅ | `9281a5c` |
| M6.1 前端界面重构（Firefish/Misskey 风格、自适应 deck） | ✅ | `23ac6d7` … 见 git log |
| M7 打磨 | ✅ | 见 git log（迁移工具、账号管理、性能基线、部署文档） |

## 3. 契约以源码为准

`misskey-dev/misskey` 是 **AGPL-3.0-only**。实现任何端点前，先读源码的 `paramDef`（参数 JSON Schema）和 `res`（响应 schema），再查 `packages/misskey-js/src/api.types.ts` 的类型定义。官方文档是二手的，已发现多处与源码不一致。

本地浅克隆：`/tmp/refcheck/misskey`（不入库，重启后丢失）。若被清理，用代理重建：

```bash
export https_proxy=http://127.0.0.1:19823 http_proxy=http://127.0.0.1:19823
git clone --depth 1 --filter=blob:none --sparse https://github.com/misskey-dev/misskey.git /tmp/refcheck/misskey
cd /tmp/refcheck/misskey && git sparse-checkout set packages/backend/src packages/misskey-js/src
```

后端源码全量约 11 MB；Streaming 实现在 `packages/backend/src/server/api/stream/`。

## 4. 关键上游细节（踩过的坑）

1. **错误信封有三种**：普通 API 用 `{"error":{"message","code","id","kind"}}`（`kind` 必填）；`signin` 失败只有 `id`；OAuth 错误是扁平结构。
2. **原生 token 判据是长度 16**（`misc/token.ts`），存 `users.token`；MiAuth/OAuth token 存 `access_tokens`，两套查找路径。
3. **`/api/i` 的 secrets 只对原生 token 返回**。
4. **`meta` 用 `uri` 不是 `instanceUrl`**；`detail` 参数控制 lite/detailed。
5. **`ping` 无参数**，返回服务端 `Date.now()`。
6. **联邦模式原生三态** `all | specified | none`，与项目要求的"公开/白名单/不互联" 1:1 对应。
7. **白名单是后缀匹配**：匹配 `sub.example.com` 但不匹配 `notexample.com`。
8. **联邦拦截有六处**（`isFederationAllowedUri`/`Host` 调用点），详见 `docs/research/2026-10-26-misskey-source-calibration.md`。
9. **可见性规则**：public/home 对所有人可见；followers 需关注者或回复目标；specified 需在可见列表或被 @。
10. **LTL/GTL 只含 public 可见性**的笔记。
11. **纯 renote 没有文本是合法的**（上游 `if/then` schema 允许 renoteId 存在时无 text）。
12. **回复默认从时间线排除**，但"回复自己写的笔记"要保留（判定要用被回复笔记的作者）。

## 5. 已定案的产品决策

| 决策 | 内容 |
|------|------|
| 语言 | Go 1.26 |
| 前端 | Vue 3 + TypeScript + Vite |
| 数据库 | PostgreSQL 唯一强制依赖；Redis 可选（默认关闭） |
| Mastodon API | 明确放弃，连路由空位都不留 |
| 注册模式 | open / invite / admin 三种，配置切换 |
| 联邦模式 | public / allowlist / none 三种，运行时切换 |
| 媒体 | local + S3 兼容双实现 |
| 数据迁移 | 不迁移朋友私有项目数据，空实例起步 |
| Relay | 首版不做 |
| 参考边界 | 只参考协议与 API 契约；Pleroma/Akkoma 代码明确排除 |

## 6. 环境

- Go 1.26.8，`GOPROXY=https://mirrors.aliyun.com/goproxy/,direct`
- PostgreSQL 18：库 `cinderfish`（开发）、`cinderfish_test`（测试）；角色/密码 `cinderfish`/`cinderfish`
- Redis：**可选**，默认关闭；本机没有 `redis-server`，测试用 `miniredis` 代替
- **访问 GitHub 必须走代理** `127.0.0.1:19823`
- `dl.min.io` 被代理拦截，MinIO 二进制下不下来；S3 后端**尚未做真实服务端验证**
- S3 客户端用 `github.com/minio/minio-go/v7`（不是 AWS SDK，依赖树太大）

## 7. 怎么跑

```bash
cd /home/lainsaka/文档/MilnFish   # 目录改名待办

go vet ./...
go test ./...
CINDERFISH_TEST_DATABASE_URL="postgres://cinderfish:cinderfish@localhost:5432/cinderfish_test" go test ./...

cd web && pnpm install && pnpm build && cd ..   # 产出 web/dist
cp config.example.toml config.toml   # 按需改
go build -o cinderfish ./cmd/cinderfish
./cinderfish --config config.toml

# registration_mode = admin 时没有自助注册，用命令行建第一个管理员
go run ./cmd/cinderfish-admin --config config.toml create --username root --password-stdin --admin
```

只跑 API 时可以不构建前端：`web_dist` 目录不存在，服务器自动退回纯 API 模式。

数据库迁移在启动时自动执行（不可变的版本化脚本）。当前 schema 版本为 **4**。

数据库测试包（`internal/account`、`internal/httpapi`）都会 TRUNCATE 同一批表，因此每个包用 `internal/testdb` 在 `TestMain` 里持有 PostgreSQL advisory lock，直到该包的测试二进制退出——这样 `go test ./...` 并行跑包时不会互相清表（不加锁时会随机出现"找不到用户"之类的假失败）。

### 7.1 媒体后端迁移

`cmd/cinderfish-media-migrate` 把 `drive_files` 里的对象从当前 `[media]` 配置的后端搬到另一个后端；对象 key（内容哈希）不变，只更新行的 `storage_backend`，笔记里的链接因此不需要改写。

```bash
# 本地 → S3（S3 参数取自 config.toml 的 [media.s3]）
go run ./cmd/cinderfish-media-migrate --config config.toml --to s3 --dry-run
go run ./cmd/cinderfish-media-migrate --config config.toml --to s3 --delete-source

# S3 → 本地
go run ./cmd/cinderfish-media-migrate --config config.toml --to local --to-local-path ./data/files
```

`--dry-run` 只统计；`--limit N` 限制条数；单个对象失败不会中断其余，报告会列出失败 key，退出码为 1。`--delete-source` 只在整轮无失败时清理源对象。迁移完成后把 `config.toml` 的 `media.backend` 切到新后端再重启。

配置项可用 `CINDERFISH_*` 环境变量覆盖，如 `CINDERFISH_PORT`、`CINDERFISH_DOMAIN`、`CINDERFISH_REGISTRATION_MODE`、`CINDERFISH_FEDERATION_MODE`、`CINDERFISH_MEDIA_BACKEND`、`CINDERFISH_MEDIA_LOCAL_PATH`、`CINDERFISH_WEB_DIR`、`CINDERFISH_REDIS_ENABLED`、`CINDERFISH_REDIS_URL`、`CINDERFISH_REDIS_PREFIX`。

## 8. 代码结构

```
cmd/cinderfish/main.go      入口：配置、迁移、存储初始化、优雅关闭
internal/config/            TOML + 环境变量覆盖 + 校验
internal/db/                迁移执行器（版本化 SQL 脚本内嵌在代码里）
internal/idgen/             Misskey 风格时间有序字符串 ID
internal/auth/              bcrypt、token 生成、PKCE S256、用户名校验
internal/account/           账户服务（三种注册模式）、token 签发与解析
internal/note/              笔记服务、时间线服务、反应服务、通知服务、提及解析
internal/media/             Storage 接口 + local 实现 + S3 实现 + Drive 服务
internal/streaming/         Streaming API：事件总线、WebSocket 连接、五个频道
internal/federation/        联邦：HTTP 签名、三模式策略、AP 渲染/解析、inbox、投递队列
internal/httpapi/           路由装配与全部端点处理器、静态托管与 SPA fallback
internal/testdb/            数据库测试的 advisory lock 辅助
web/                        Vue 3 + TypeScript 前端（api/stores/components/views/layouts）
```

依赖方向单向：`httpapi → 领域包 → db/media(接口)`。`streaming` 只定义接口，实现由 `httpapi` 注入。

## 9. 已实现的端点

```
POST /api/meta                          GET  /api/ping
POST /api/signup                        POST /api/signin
POST /api/username/available            POST /api/i
POST /api/i/update                      POST /api/i/pin
POST /api/i/unpin
POST /api/users/show                    POST /api/users/notes
POST /api/miauth/gen-token
POST /api/notes/create                  POST /api/notes/show
POST /api/notes/delete                  POST /api/notes/reactions/create
POST /api/notes/reactions/delete        POST /api/notes/local-timeline
POST /api/notes/global-timeline         POST /api/notes/hybrid-timeline
POST /api/notes/timeline                POST /api/notifications
POST /api/notifications/mark-all-as-read
POST /api/drive/files/create            POST /api/drive/files
POST /api/drive/files/show              POST /api/drive/files/delete
POST /api/following/create              POST /api/following/delete
POST /api/admin/federation/mode         GET/POST /api/admin/federation/allowlist
GET  /media/{path...}
GET  /                                  （Web 前端，SPA fallback）
GET  /.well-known/oauth-authorization-server
GET  /oauth/authorize                   POST /oauth/token
GET  /streaming                         （WebSocket 升级）
```

M5 联邦路由见 §12.1。

## 10. M4 Streaming 要点

协议细节见 `docs/research/2026-10-28-m4-streaming-source-calibration.md`。要点：

- **端点**：`GET /streaming`，WebSocket 原地升级。
- **鉴权**：`Authorization: Bearer <t>` 优先，否则 query `?i=<t>`。无 token 合法（匿名）；token 无效才 401；封禁用户 403。app token 必须含 `read:account`，原生 16 位 token 不受此限。
- **频道**：`main`（需登录）、`homeTimeline`（需登录）、`localTimeline`（匿名可用）、`hybridTimeline`（需登录）、`globalTimeline`（匿名可用）。
- **客户端消息**：`connect` / `disconnect` / `ch` / `subNote` / `unsubNote` / `readNotification`。
- **上限**：每连接 32 频道、1536 条 subNote（LRU 淘汰）、8 MiB 发送积压。心跳 60s ping / 120s 无 pong 断开。
- **跨进程**：`LISTEN/NOTIFY`（channel `cinderfish_stream`）中继，为多进程部署预留。

### 10.1 已知缺口（后续里程碑范围）

- ~~**mention 永远不触发**~~：**M5 已修**。`note.ParseMentions` 解析 @user 与 @user@host，创建笔记时会解析本地提及并生成通知与事件。
- ~~**follow / followed 事件无来源**~~：**M5 已修**。`/api/following/create` 与 `/api/following/delete` 会发布 follow / followed / unfollow。
- **mute / block 过滤恒为 false**：`isNoteMutedOrBlocked` 的调用点已就位，等数据模型落地。
- **unreadNotification 提醒未实现**：需要"最近已读通知"游标。
- **events 表未写入**：M4 不做重放。

## 11. M6 前端

技术栈：Vue 3.5 + TypeScript 5.9 + Vite 8 + vue-router 4。没有 UI 组件库，也没有 Pinia——共享状态是模块级 `reactive` store（`web/src/stores/`），设计令牌在 `web/src/styles/main.css`。

### 11.1 目录

```
web/src/api/         类型（Misskey schema）与 fetch 客户端（ApiError 解析 error 信封）
web/src/stores/      session（token + /api/i）、instance（meta）、streaming（WebSocket）、
                     shell（universal/deck）、deck（列配置）、nav（导航偏好）、
                     widgets（小组件栈）、theme（主题与 meta theme-color）
web/src/lib/         MFM 渲染（先转义，再函数解析，最后生成链接）、emoji 目录、相对时间、
                     hotkeys（单条笔记热键）、focus-sibling（上下焦点跳转）
web/src/nav.ts       导航唯一真源（NavItemDef 注册表 + 默认桌面/移动顺序）
web/src/widgets/     widget 注册表（registry.ts）
web/src/os.ts        命令式 UI 总线：openPostForm / confirmDialog / openReactionPicker（单例弹层）
web/src/composables/ use-page-title（document.title 栈式覆盖）
web/src/components/  note/（Note、NoteHeader、NoteContent、NoteMedia、NoteFooter、NoteReactions、
                       RenoteBar、NoteParent、NoteDetail、use-note）、
                     composer/PostForm、deck/（DeckColumn、DeckPageWindow、deck-links）、
                     profile/（ProfileCard、ProfileSidebar）、
                     ui/（PagedFeed、PageHeader、PageWindow、EmojiPicker、MobileNav、NavDrawer）、
                     layout/（Sidebar、WidgetPanel、MobileSidebar——壳层三件套）、
                     widgets/（WidgetStack、WidgetsDrawer、WidgetCard 与五个 widget）
web/src/layouts/     AppShell（组合 Sidebar | 主列 | WidgetPanel）、DeckShell（横向列条）、AuthLayout
web/src/views/       Timeline、Note、Notifications、Profile、Explore、Settings、Admin、Login、Signup
web/src/views/settings/  设置左导航 + 8 个子页
web/src/views/admin/     管理左导航 + 4 个分区
```

### 11.2 已完成的页面

参考边界固定：**Misskey 只作后端 API 契约来源，界面以 Firefish（主要）与 Iceshrimp（补充）为参考**，
见 `docs/research/2026-10-30-firefish-ui-gap-analysis.md`、
`docs/research/2026-10-31-iceshrimp-architecture-translation.md` 与执行计划
`docs/superpowers/plans/2026-10-30-firefish-ui-replication.md`。

- **令牌层**：`styles/main.css` 是唯一的令牌来源，分表面/描边/前景/强调/状态/内容语义/焦点与滚动条/
  尺度（圆角 xs–full、间距 2–64、字号 2xs–3xl、字重、行高、动效、层级、布局宽度）几组。
  双主题走 `data-theme`，`index.html` 在首个样式表之前用同步脚本定主题，首帧不闪白；
  `stores/theme.ts` 统一负责应用、切换、跨标签同步与跟随系统，并同步 `meta theme-color`。
  主题偏好有三个取值：`auto`（跟随系统，默认）、`light`、`dark`；`auto` 时系统切换深浅色会立即重绘，
  并在窗口重新获得焦点或标签页可见时补一次同步（后台标签可能错过媒体查询事件）。
  具体选择存 `localStorage["cinderfish.theme"]`，值为 `auto`/`light`/`dark`，缺失即视为 `auto`；
  手动切换（设置页按钮或侧栏图标）会把偏好钉成显式的 `light`/`dark`。
  根节点带 `color-scheme`，原生控件与滚动条跟随主题。字号偏好写在 `views/settings/font-scale.ts`，
  由 `main.ts` 在启动时应用为根字号（rem 刻度整体缩放）。
- **双壳自适应**：`stores/shell.ts` 记住 `universal` / `deck`（`localStorage["cinderfish.ui"]`）。
  通用壳 ≥1280px 是完整侧栏 232px，1280px 以下收成图标栏 76px，1100px 以下隐藏 widget 栏并改用
  悬浮入口，720px 以下换成五项底栏（菜单/时间线/发帖/通知/探索）并由抽屉承载设置、管理、主题切换、
  壳切换与退出登录；deck 壳是 350/420px 可切宽的横向 scroll-snap 列条，窄屏单列横滑、宽屏并列多列，
  列配置存 `localStorage["cinderfish.deck.columns"]`，支持回到顶部、上移/下移与 widgets 列。
  deck 模式下点笔记/头像走 `PageWindow` 覆盖层，不改 URL。壳层高度走 flex 链，没有魔法数。
- **导航与 widget 配置**：`nav.ts` 是导航唯一真源，桌面与移动两组顺序存在 `cinderfish.ui` 的 `nav` 键；
  widget 注册表 + `cinderfish.ui.widgets` 记录有序栈，右栏与抽屉共用 `WidgetStack`。
  首版 widget：实例信息、关于链接、便签（localStorage）、通知、时间线（四源可切）。两者都能在设置页改。
- **页头**：`ui/PageHeader` 是唯一的 sticky 页头（返回、标题、副标题、图标、tab 高亮条、动作插槽），
  容器宽度 < 500px 折叠为两行；tab 状态写入 `route.query`，刷新保持、可分享、后退回上一个 tab。
- **时间线**：首页 / 本地 / 社交 / 全球四频道，未登录不显示需要认证的两个频道；统一由 `ui/PagedFeed`
  承担无限滚动、去重、日期分隔行与断线重载；流式新笔记先累计成「N 条新笔记」浮标，点击才插入。
- **发帖**：全局 `os.openPostForm()` 对话框（回复、引用、纯转贴），CW、可见性四态分段按钮
  （public/home/followers/specified 手动解析 `@user@host`）、仅本地、多文件上传、800ms 去抖草稿
  （`cinderfish.draft`）、Ctrl/Cmd+Enter 发布。
- **笔记**：单一 `<article>`（带 id/tabindex/aria-label），纯转贴不再套第二层容器；行式无卡片、
  `--divider` 分隔；反应条并入 footer 首位，单行横滚 + 两侧渐隐遮罩，自己的反应置顶，未登录或远程
  emoji 不可点；footer 常态 0.7 透明度、hover 与 focus-within 升到 1；引文在时间线是链接、在详情页是
  1px `--renote` 边框卡片；父帖上下文由 `NoteParent` 画连接线；长文 15em 折叠 + 遮罩；container query
  只有 500px 与 380px 两档，同时收字号与头像尺寸；单条笔记热键 r/e/q/j/k 与上下焦点跳转。
  笔记详情抽成 `NoteDetail`，笔记页与 deck 覆盖层共用。
- **反应**：`ui/EmojiPicker`（搜索 + 分类页签 + 最近使用，`cinderfish.recent-emojis`，键盘可导航）
  经 `os.openReactionPicker` 打开；`reactionAcceptance=likeOnly` 时只给星标按钮。
- **MFM**：x2/x3/x4、spin/shake/jump/blink/rainbow/blur、center/small、fg/bg 颜色、mono/serif、plain，
  动效在 `styles/mfm.css` 且尊重 `prefers-reduced-motion`；未知函数降级为内文，`:shortcode:` 仍是纯文本。
- **通知**：全部/反应/提及/转贴四 tab，筛选下推给后端 `includeTypes`（切 tab 重挂载，不会一边空态一边翻页），
  行式排版、反应 emoji 走 emoji 目录、全部标为已读、侧栏未读徽标与流式同步。
  被动视图（通知 widget、deck 通知列）显式传 `markAsRead: false`，不会因为渲染就清未读。
- **资料**：置顶笔记区（最多 5 条）+ 笔记/媒体/回复三 tab；副栏（文件栅格 + 动态摘要）按页面自身宽度
  < 1000px 并入主列，用 ResizeObserver 判定；关注中/关注者计数在没有对应端点时标注为暂不可用。
- **设置**：`/settings/:tab` 嵌套路由，左导航分账户/客户端/其他三组，8 个子页
  （个人资料、外观、导航、小组件、通知、隐私、账户、关于），未知 tab 回退个人资料。
  外观页管主题与字号，导航页管桌面/移动两套顺序，小组件页管有序栈。
- **管理**：`/admin?tab=` 的参数切换左导航（概览/用户/文件/设置），危险操作独立分区用
  `--danger-soft` / `--danger-border` 标出，非管理员显示门禁卡片。
- **探索**：本地/全球公开时间线，支持 `?tag=`（客户端过滤，仍能继续分页）。
- **登录/注册/404**：真实 signin/signup，注册前检查用户名可用性，邀请码字段，注册关闭提示；三页都有标题。


### 11.3 服务端托管

`web_dir`（默认 `./web/dist`，环境变量 `CINDERFISH_WEB_DIR`）指向构建产物。命中规则：

- `/api*`、`/media`、`/.well-known`、`/oauth`、`/users`、`/nodeinfo`、`/inbox`、`/streaming` 永远交给 Go mux。
- `/@user` 与 `/notes/{id}` 在 `Accept` 含 `text/html` 时给前端，否则仍走 ActivityPub。
- 其余 GET/HEAD 先找真实文件，找不到回退 `index.html`；`/assets/*` 带一年 immutable 缓存，`index.html` 每次读取（重新构建无需重启）。
- 目录不存在时服务器保持纯 API 模式。

### 11.4 Redis（可选）

默认关闭，PostgreSQL 仍是唯一必选依赖。开启后：

- Streaming 的跨进程中继从 `LISTEN/NOTIFY` 换成 Redis Pub/Sub（频道 `<prefix>:stream`），不再受 8000 字节 NOTIFY 上限约束，也不再需要分块重组。
- 联邦投递队列仍是 PostgreSQL 表；入队时通过 `<prefix>:deliver` 发布一次唤醒，本进程与其它进程的 worker 立刻检查队列，而不是等 2 秒轮询。唤醒只是提示，丢一次最多晚一个轮询周期。
- 启用后启动时会 `PING`，连不上直接退出——避免两个进程对"用哪种中继"产生分歧。
- 关闭时行为与之前完全一致；Redis 与 PostgreSQL 两种中继不能同时挂在同一个进程上。

配置：

```toml
[redis]
enabled = true
url = "redis://127.0.0.1:6379/0"
prefix = "cinderfish"
```

`prefix` 用于多实例共用一个 Redis 库时隔离频道；空值回退到 `cinderfish`。

### 11.5 实时更新

`web/src/stores/streaming.ts` 维护唯一 `/streaming` 连接：token 走 query，`connect` 带 `pong:true`，按订阅 id 分发 `channel` 帧，断线指数退避（1s→30s，6 次封顶），登录/退出时重建连接。订阅按频道引用计数：同一频道只建立一条服务端订阅，最后一个订阅者释放时才 `disconnect`；外壳自己的 `main` 订阅也计入计数，并在 `onopen` 重新确认，所以通知页进出、断线重连都不会让未读徽标失去通道。时间线各自订阅自己的频道，`__reconnected`（空频道）触发整页重载。

### 11.6 开发与构建

```bash
cd web
pnpm install
pnpm dev      # 5173，代理 /api /media /streaming 到 127.0.0.1:3000
pnpm build    # 先 vue-tsc --noEmit，再输出 web/dist
```

### 11.7 已知缺口

- **自定义 emoji 没有注册表**：`:shortcode:` 在 MFM 与通知里只按内置目录解析，解析不到就照原样显示；实例自定义 emoji 的注册、管理、picker 分组都在首版之外。后端打包的 `emojis` / `reactionEmojis` 目前恒为空数组。
- **没有搜索、没有用户列表页、没有投票接口**（前端只展示投票结果）。
- **远程资料不能浏览**：`users/show` 只接受本地用户名；远程用户点击 `@user@host` 会得到说明卡片。发帖器的 `specified` 仍允许手动输入远程 handle，后端并不会解析它。
- **`users/following` / `users/followers` / `users/search` / `notes/mentions` / `notes/children` / `notes/featured` / `notes/reactions` 都不存在**：个人页的「关注中/关注者」计数标为暂不可用，笔记页只渲染 `note.reply` 这一层上下文，没有回复树与「谁反应了」浮层。
- **改密码、封禁/静音列表、注销账户在前端与后端都未实现**：账户设置页如实标注为未开放，没有假按钮。
- **管理面板范围有限**：实例级文件列表、注册模式在线切换、用户分页都没有端点，只能在页面上说明。管理页需要管理员账号才能看到分区。
- **i18n 未做**：文案硬编码中文；主题只保留 `data-theme` 双主题 + CSS 变量，没有 Firefish 的 JSON5 主题 DSL。
- **置顶不同步联邦**：actor 文档声明了 `featured` 集合，但没有对应路由与投递，置顶只在本实例生效。
- **草稿只有一份**：`cinderfish.draft` 是单条草稿，时间线固定发帖器与全局对话框共用同一个键。
- **deck 覆盖层单层**：deck 模式的 PageWindow 打开另一个笔记会替换当前页，没有窗口栈。
- **同一条笔记跨列的状态不同步**：反应、删除、转贴的乐观更新只作用于当前组件实例，同一笔记出现在多个 deck 列时另一列不会跟着变。
- **没有前端自动化测试**：验证靠 `vue-tsc` + 构建 + 真实浏览器手工走查；没有引入 vitest/playwright。

### 11.8 复刻波次的验证证据（2026-10-30）

门禁：`gofmt -l internal/ cmd/` 空、`go vet ./...` 通过、
`CINDERFISH_TEST_DATABASE_URL=… CINDERFISH_TEST_REDIS_URL=… go test -count=1 ./...` 全绿（新增 8 个通知筛选与置读用例）、
`cd web && pnpm typecheck && pnpm build` 均退出 0。

真实实例浏览器走查（`web_dir` 指向当前 `web/dist`，独立库与端口 3213，明暗两套）：

| 场景 | 结果 |
|------|------|
| 1440 / 1280 通用壳 | 三区成立：侧栏 232px + 主列 + widget 栏 300px，`scrollWidth == clientWidth` |
| 1150 | 侧栏收成 76px 图标栏，widget 栏保留 |
| 1099 / 900 | widget 栏隐藏，改用悬浮入口；无横向溢出 |
| 720 / 500 / 380 | 侧栏隐藏，五项底栏（菜单/时间线/发帖/通知/探索）居中发帖；380px 页头折成两行 |
| 抽屉 | 380px 从底栏菜单打开，含账号、发帖、导航与设置/主题/壳切换/退出，Esc 与遮罩可关 |
| deck 1440 | 侧栏保留，两列并排 + 添加列槽位，列头菜单可回到顶部/加宽/下移 |
| deck 720 | 单列满宽，列条 `scroll-snap-type: x mandatory`，可横滑到下一列 |
| 笔记 | 无卡片边框，行间 1px 分隔；footer 常态 `opacity: .7`，`:focus-within` 实测变 1；日期分隔行正常 |
| 主题 | 明暗两套在 1440 下均正确（`body` 背景分别为 `#f7f5f1` / `#0e0d12`），`color-scheme` 跟随 |
| 小组件 | 齿轮进入编辑态：下拉添加、上移/下移/删除、恢复默认 |
| 设置 / 管理 | `/settings/appearance` 左导航 + 右内容；`/admin?tab=users` 需管理员，非管理员显示门禁卡片 |

### 11.9 全面代码审查与修复（2026-10-30）

对 `main` 合并基线到 `7146e2b` 的前后端改动做了一次分域只读审查，已修复的问题：

| 问题 | 修复 |
|------|------|
| 纯转贴把被转贴的原笔记当作操作对象，用户无法删除/置顶自己发布的转贴 | `useNote` 增加 `actionTarget`，`NoteFooter` 增加 `actionNote`；展示仍用 `appearNote`，删除与置顶落在外层笔记 |
| `users/show` 不返回关注关系，刷新个人页后已关注对象仍显示「关注」 | 后端按 viewer 返回 `isFollowing`/`isFollowed`/`hasPendingFollowRequestFromYou`/`hasPendingFollowRequestToYou`（匿名恒为 false，坏 token 不报 401）；个人页与 deck 资料窗据此初始化，待对方确认时显示「请求已发送」 |
| 并发置顶可突破 5 条上限或写入重复 id | `PinNote` 的读-检查-写放进一个事务并对 `user_profiles` 行 `FOR UPDATE`；补两个并发集成测试 |
| 取消关注不重算 `followers_count` / `following_count`，计数永久偏高 | `Unfollow` 同步重算两个计数，测试断言关注/取关后分别为 1 / 0 |
| `stop()` 后 `start()` 会重放已卸载组件的频道订阅 | `stop()` 清空 `channelSubs`/`subsByWire`/`tokens` 与 `mainToken` |
| deck 移动端断点隐式覆盖 `grid-template-columns` | 720px 处显式 `grid-template-columns: none` |

**审查中驳回的一条**：`notifications` 的 `markAsRead` 缺省为启用被判为与 Misskey 不符，实际上游 `upstream/.../i/notifications.ts` 的 `paramDef` 就是 `default: true`，当前实现正确，未改。

仍留待决定：设置路由没有统一登录守卫（账户类子页目前各自处理）；deck 移动端几何没有自动化断言；关注请求没有撤销入口（后端无对应端点）。

### 11.10 Iceshrimp 壳层重构（2026-10-31，重写计划 Task 2）

按 `docs/research/2026-10-31-iceshrimp-architecture-translation.md` 的翻译契约把壳外壳拆成三个子组件，`AppShell.vue` 只保留组合与壳级断点：

- 新增 `web/src/components/layout/Sidebar.vue`（桌面侧栏，消费 `nav.ts` 注册表；250px 全宽 / 80px 图标态，≤1279 收缩——fork `$nav-width`/`$nav-icon-only-width`）、`WidgetPanel.vue`（右列 300px，≤1090 隐藏，与 `WidgetsDrawer` 悬浮按钮的可见区间对齐为 ≥1091）、`MobileSidebar.vue`（≤500 移动开关；组合既有 `NavDrawer` 抽屉与 `MobileNav` 底栏，二者行为已满足契约故保留内部实现，仅把抽屉开关状态从 AppShell 移入）。
- `PageHeader.vue` 补齐契约项：>3 个 tab 时非活动 tab 折叠为图标（有图标者），动作区标签 ≤1090 隐藏（`.action-label` 约定，通知页已接入），移动断点随壳从 720 改为 500。
- 断点定稿（fork 主源）：`1100/1090/500/1279/400`；`DeckShell` 与 widget 抽屉的 720 随之改为 500；`.auth` 布局断点保持 720（认证页独立于壳）。底栏 ≤400 收紧按钮尺寸。
- 侧栏/壳几何样式从 `main.css` 移入各组件；`--rail-width` 改 250px、`--rail-width-collapsed` 80px、`--shell-max-width` 1280px。
- 验证：`pnpm typecheck`、`pnpm build`、`go vet`、`go test` 全绿；浏览器实测 1440/1100/840/600/375 的 `scrollWidth/clientWidth` 全部相等（无水平溢出），1440 三区（250+主列+300）、1100 图标态+widget 列、840/600 widget 隐藏、375 抽屉+底栏（含 Esc 关闭、路由变化关闭）、登录页保持裸布局。移动宽度下内建浏览器无法截图，采用 DOM 快照 + 注入式几何探针测量（探针已移除）。

## 12. M5 联邦（ActivityPub）

源码对照记录见 `docs/research/2026-10-28-m5-federation-source-calibration.md`（逐条对照上游文件，含刻意偏离项）。

### 12.1 已实现的路由

```
GET  /.well-known/webfinger            GET  /.well-known/host-meta(.json)
GET  /.well-known/nodeinfo             GET  /nodeinfo/2.1  /nodeinfo/2.0
GET  /users/{id}                       GET  /@{handle}
GET  /users/{id}/publickey             GET  /users/{id}/outbox
GET  /users/{id}/followers             GET  /users/{id}/following
GET  /users/{id}/collections/featured
GET  /notes/{id}                       GET  /notes/{id}/activity
POST /inbox                            POST /users/{id}/inbox
```

### 12.2 关键实现要点

1. **签名**：出站 POST 签 `(request-target) date host digest`，GET 签 `(request-target) date host`，`keyId` 形如 `<url>/users/<id>#main-key`。入站按上游顺序校验：解析 Signature → host/date 必须被覆盖 → Host 必须是本机 → digest 格式 → digest 值 → RSA 签名。`Date` 超出 ±5 分钟直接拒绝。
2. **内容协商**：`Accept` 含 `application/ld+json` 回 LD，否则回 `application/activity+json`；都带 `Vary: Accept` 与 `Cache-Control: public, max-age=180`。
3. **三模式**：模式存在 `instance_meta.federationMode`，每次检查都读库，所以 `POST /api/admin/federation/mode` 立即生效。白名单是 `federation_allowlist` 表，**后缀匹配**。配置里的 `federation.allowlist_hosts` 与 `federation_mode` 只做首次 seed（`ON CONFLICT DO NOTHING`），重启不会覆盖运行时改动。
4. **门禁位置**：`none` 时全部 AP 路由 403（含 well-known）；`allowlist` 时入站按来源 host、出站按目标 host 判断，出站是**入队前和发送前各查一次**，切到 `none` 会立刻停掉队列里积压的投递。
5. **投递**：`deliver_queue` 表，**按 inbox 一行**（一条笔记给多个 follower 各有一行各自的退避）。指数退避 1s→5min 上限，12 次后放弃并记 `last_error`。follower inbox 与被 @ 用户的 inbox 合并去重（优先 `shared_inbox`）。
6. **inbox 活动**：`Follow`（锁定账号进 `follow_requests` 并回 Accept，否则直接关注并投递 Accept）、`Accept`/`Reject`、`Undo`（Follow/Like/Announce）、`Create<Note>`（校验 `attributedTo` 与 host 一致、按 AP id 去重）、`Delete`（笔记或账号）、`Like`（读 `_misskey_reaction`）、`Announce`、`Update`、`Block`。返回值是 Misskey 风格的 `ok` / `skip: 原因` 字符串，写日志；**HTTP 状态一律 202**。
7. **正文渲染**：项目存 MFM、没有存 HTML，所以 `content` 由 `federation.MFMToHTML` 生成（提及/话题标签/自定义 emoji/裸链接/粗斜体/行内与围栏代码）。代码块先提取再转义，所以代码里的 URL 不会被链接化。
8. **用户名唯一性**：v2 的 `idx_users_username_lower` 是全表唯一，远程用户进同一张表后 `alice@a.example` 会和本地 `alice` 撞。v4 改成：本地用户名唯一 + `(username_lower, host)` 唯一。

### 12.3 测试覆盖

- 单元：签名签发/验证往返（POST 与 GET）、篡改正文、错误 Host、未覆盖 digest、过期 Date、`Authorization: Signature` 形式、白名单后缀匹配、MFM 转换、Person/Note 渲染、`ParseMentions`。
- 集成（真库 + httptest，`CINDERFISH_TEST_DATABASE_URL` 必需）：webfinger 三种 resource 形状与 422/404/400、actor 文档字段、`@handle` 路由、nodeinfo、两种 host-meta、`none` 下全路由 403。
- **双实例 smoke**：两个真实实例、真实 HTTP、真实 RSA 签名。alpha 的 Follow 被 beta 验签通过并记录，beta 投递签名 Accept，alpha 验签通过并记录关注。域名到监听地址的跳转由一个 rewrite transport 完成，密码学与 HTTP 环没有任何 mock。
- 管理面：非管理员改模式 403、切到 `none` 后下一次请求即 403、非法模式 400、白名单增删与后缀匹配语义。

### 12.4 已知缺口（不是 bug，是范围）

- **XRD 形式的 webfinger 不实现**：只做 JRD，现代联邦软件都只用 JRD。
- **LinkedDataSignature（`signatureValue`）不实现**：只用 HTTP Signatures。
- **inbox 同步处理**：验签通过立刻回 202，但活动在请求内处理完毕，没有单独的任务队列。
- **只有第一个管理员/第一个本地用户持有实例签名密钥**：`signingUserID` 取 `is_admin DESC, created_at ASC` 的第一人。个人实例够用，多用户实例需要改成「按活动作者签名」。
- **锁定账号**：`users.is_locked` 列存在，但还没有「锁推」设置项，本地关注一律立即完成；只有远程 Follow 会走 `follow_requests`。
- **置顶笔记**（`/collections/featured`）恒为空：置顶功能未实现。
- **Remote media 不缓存/不代理**：直接用远端 URL。
- **没有 `/api/ap/show` 与 `/api/ap/get`**。
- **没有被拉黑的实例列表、没有 relay**：Relay 按定案不做。
- **没有真实三方联调**：没有对 Mastodon/GoToSocial/真实 Misskey 实例跑过。双实例测试证明的是本实现自洽。

## 13. 文档索引

| 文档 | 内容 |
|------|------|
| `AGENTS.md` | 项目规约（**改代码前必读**） |
| `docs/superpowers/specs/2026-10-26-cinderfish-architecture-v0.md` | 架构设计 v0 |
| `docs/research/2026-10-26-misskey-fediverse-firefish.md` | 首轮调研 |
| `docs/research/2026-10-26-go-ecosystem-and-dependency-decisions.md` | Go 生态与依赖取舍 |
| `docs/research/2026-10-26-tech-stack-comparison.md` | 技术栈横向对比 |
| `docs/research/2026-10-26-misskey-source-calibration.md` | **源码校准记录（含许可证约束与踩坑清单）** |
| `docs/research/2026-10-26-m2-implementation-notes.md` | M2 实施记录 |
| `docs/research/2026-10-26-initial-scope.md` | 首版范围与已裁决项 |
| `docs/research/2026-10-28-m4-streaming-source-calibration.md` | **M4 Streaming 协议逐条对照上游源码** |
| `docs/research/2026-10-28-m5-federation-source-calibration.md` | **M5 联邦逐条对照上游源码** |
| `docs/research/2026-10-28-project-name-decision.md` | 命名决策记录 |
| `docs/research/2026-10-29-performance-baseline.md` | **性能自比基线（M7）** |
| `docs/DEPLOY.md` | **生产部署指南（反向代理、systemd、备份、媒体迁移）** |
| `deploy/` | systemd 单元、Caddyfile、生产配置模板 |
| `web/` | Vue 3 前端源码（构建到 `web/dist`） |
| `.archify/architecture-cinderfish-overall/` | 项目整体结构图（Archify 产物，随里程碑更新） |

## 14. 已知的未验证项

- **S3 后端未做真实服务端验证**：只验证了编译、接口与 URL 生成逻辑。
- **缩略图/WebP 生成未实现**：`drive_files.thumbnail_key` 字段已留，图片处理管道还没写。
- **前端没有真实第三方客户端联调**：页面已在构建产物上用浏览器走查，但没有用 misskey-js 或 Misskey Web 前端连过本实例。
- **Redis 只对 miniredis 验证过**：本机没有 `redis-server`，跨进程中继的单元测试与双进程 smoke 都用 miniredis；没有对真实 Redis 做过持久化/认证/网络闪断验证。
- **Streaming 未做真实 Misskey 客户端联调**：已用自写 WebSocket 客户端验证（握手、`connect`、`connected`、笔记投递全部通过），但**没有**用 misskey-js 或 Misskey Web 前端实际连过。
- **联邦没有任何第三方联调**：只跑了本实现的双实例测试（真签名、真 HTTP、真库），没有对 Mastodon / GoToSocial / 真实 Misskey 实例互通。这是 M5 之后最该补的验证。
- **Streaming 多进程中继只手测了单连接的 LISTEN/NOTIFY 往返**，没有做真正的双进程联调。
