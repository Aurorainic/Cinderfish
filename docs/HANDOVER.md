# Cinderfish 交接说明

- 最后更新：2026-11-01（Firefish 前端整体移植 Task 3：壳、导航与 Deck）
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
| 前端 Iceshrimp 架构重写（Task 1–8） | ✅ | `d414bd9` … `9167f1e`（分支 `frontend-ui-redesign`，见 §11.10–§11.15） |
| Firefish 前端整体移植（Task 1：参考源码/许可证/能力矩阵/shim 契约） | ✅ | 见 `git log -- docs/research/2026-11-01-*`（分支 `frontend-ui-redesign`，见 §11.16） |
| Firefish 前端整体移植（Task 2–3：运行时 shim、壳/导航/Deck 与页面窗口） | ✅ | `feat(web): rebuild Firefish-style shell and adaptive deck`（分支 `frontend-ui-redesign`，见 §11.17） |
| Firefish 前端整体移植（Task 4：流式更新、`useChannel`、通知别名、auth/timeline 语义收口） | ✅ | `ab897e6` … `422a9a8`（分支 `frontend-ui-redesign`） |
| **前端重定基：Firefish 原版复原（Iceshrimp 弃用）** | 🚧 | 决策记录 `docs/research/2026-11-01-firefish-rebaseline-decision.md`；主题色板/导航/笔记密度/操作行已复原（`2c61c51`、`438cec0`、`0ca52d1`、`459d5e7`）；jclaudan 镜像评估后仅作备份（`5e763be`）；待续：PageHeader tab、composer 面板化、widgets 栏 Firefish 化 |

- Firefish 字典来源：`firefish@02c31e0a2804de48c9d73028df525e197d8df622:locales/zh-CN.yml`；当前 `web/src/i18n/zh-CN.ts` 只保留首批白名单页面词条，Cinderfish 品牌文案允许改写，其余未移植功能不复制。

## 3. 契约以源码为准

`misskey-dev/misskey` 是 **AGPL-3.0-only**。实现任何端点前，先读源码的 `paramDef`（参数 JSON Schema）和 `res`（响应 schema），再查 `packages/misskey-js/src/api.types.ts` 的类型定义。官方文档是二手的，已发现多处与源码不一致。

本地参考克隆在本分支放在 `tmp/refcheck/`（`.gitignore` 覆盖，重启不丢）：`misskey`（**仅作 API 契约**）、`firefish`（前端主参考）、`iceshrimp`（Iceshrimp.NET，布局与交互参考）、`iceshrimp-vue`（Vue 版 fork，组件镜像来源）。重建：

```bash
export https_proxy=http://127.0.0.1:19823 http_proxy=http://127.0.0.1:19823   # 仅 GitHub 需要
git clone --depth 1 --filter=blob:none --sparse https://github.com/misskey-dev/misskey.git tmp/refcheck/misskey
cd tmp/refcheck/misskey && git sparse-checkout set packages/backend/src packages/misskey-js/src packages/frontend/src
# Firefish / Iceshrimp 走 Codeberg，不需要代理；Codeberg 偶发 504，重试或改用 archive/main.tar.gz
git clone --depth 1 --filter=blob:none --sparse https://codeberg.org/firefish/firefish.git tmp/refcheck/firefish
git clone --depth 1 --filter=blob:none --sparse https://codeberg.org/iceshrimp/Iceshrimp.NET.git tmp/refcheck/iceshrimp
git clone --depth 1 --filter=blob:none --sparse https://codeberg.org/banaanihillo/iceshrimp.git tmp/refcheck/iceshrimp-vue
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

数据库迁移在启动时自动执行（不可变的版本化脚本）。当前 schema 版本为 **5**（`internal/db/db.go` 的 `migrations` 1–5）。

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
  widget 注册表 + `cinderfish.ui.widgets` 记录有序栈，右栏（`layout/WidgetPanel` 标题+动作 + `WidgetStack`
  列表）与抽屉（同一 stack，自己的标题与关闭）共用同一份编辑态（`stores/widgets` 的 `widgetEditing`，
  不落盘）。每张卡片自带标题/动作/内容三段（`widgets/WidgetCard` + `#actions`）。
  首版 widget：实例信息（刷新动作、失败重试）、关于（探索动作）、便签（保存动作）、通知（打开与全部已读）、
  时间线（来源切换与高度在卡片动作区）。两者都能在设置页改。
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
  行式排版、反应 emoji 走 emoji 目录、全部标为已读、侧栏未读徽标与流式同步。徽标初值由会话里的
  `unreadNotificationsCount` 续上（`AppShell` 用 `setUnreadNotifications` 播种，见 §11.15），流式事件在此基础上自增。
  被动视图（通知 widget、deck 通知列）显式传 `markAsRead: false`，不会因为渲染就清未读。
- **资料**：置顶笔记区（最多 5 条）+ 笔记/媒体/回复/关注中/关注者五个 tab（`?tab=`），
  置顶区与列表区各自带分区标题（`profile/ProfileSection`）；副栏（文件栅格 + 动态摘要）按页面自身宽度
  < 1000px 并入主列，用 ResizeObserver 判定。只有帖子计数是可点的链接；关注中/关注者计数保持纯数字
  并在卡片内注明「列表接口尚未提供」，对应两个 tab 渲染真实的不可用态（见 11.14）。
- **设置**：`/settings/:tab` 嵌套路由，左导航分账户/客户端/其他三组，8 个子页
  （个人资料、外观、导航、小组件、通知、隐私、账户、关于），未知 tab 回退个人资料。
  左导航与右侧内容都渲染 `ui/SectionMenu`（fork `MkSuperMenu` 的对应物），窄容器（<600）收成一行
  横向滚动的胶囊；外观页管主题与字号，导航页管桌面/移动两套顺序，小组件页管有序栈。
- **管理**：`/admin?tab=` 的参数切换同一个 `ui/SectionMenu`（实例：概览/用户/文件；配置：设置），
  危险操作独立分区用 `--danger-soft` / `--danger-border` 标出，非管理员显示门禁卡片
  （登录态不同给出不同的说明与出口）。门禁卡片只是 UX，服务端 `requireAdmin` 才是边界。
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

**完整的 API 缺口分组表在 §11.15.2–§11.15.5**（(a) 数据够 / (b) 需 schema / (c) 上游没有）。这里是客户端侧的现状：

- **自定义 emoji 没有注册表**：`:shortcode:` 在 MFM 与通知里只按内置目录解析，解析不到就照原样显示；实例自定义 emoji 的注册、管理、picker 分组都在首版之外。后端打包的 `emojis` / `reactionEmojis` 目前恒为空数组。
- **没有搜索、没有用户列表页、没有投票接口**（前端只展示投票结果）。
- **远程资料不能浏览**：`users/show` 只接受本地用户名；远程用户点击 `@user@host` 会得到说明卡片。发帖器的 `specified` 仍允许手动输入远程 handle，后端并不会解析它。
- **关注/粉丝列表、回复树、会话串、转贴列表、反应名单没有消费端**：`users/following` / `users/followers` / `notes/replies` / `notes/children` / `notes/conversation` / `notes/renotes` / `notes/reactions` 后端都不存在（数据模型够，见 §11.15.3）。个人页的「关注中/关注者」计数保持纯数字并注明列表接口未提供，两个 tab 渲染真实的不可用态（见 11.14）；笔记页只渲染 `note.reply` 这一层上下文，没有回复树与「谁反应了」浮层，也没有「查看对话」入口。
- **改密码、封禁/静音列表、注销账户在前端与后端都未实现**：账户设置页如实标注为未开放，没有假按钮。屏蔽/静音与锁定账号的**表已存在**（`blocks`、`mutes`、`follow_requests`、`users.is_locked`），缺的是端点与 UI（§11.15.3）。
- **管理面板范围有限**：实例级文件列表与注册模式在线切换都没有端点（后者要可写 meta + 配置热加载，§11.15.4），用户列表也只在 `AdminUsers.vue:42` 取前 50 条——`admin/show-users` 其实支持 `limit`/`offset`，缺的是分页 UI。管理页需要管理员账号才能看到分区。
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

仍留待决定：deck 移动端几何没有自动化断言；关注请求没有撤销入口（后端无对应端点）。设置路由的统一登录守卫已在 11.11 落地。

### 11.10 Iceshrimp 壳层重构（2026-10-31，重写计划 Task 2）

按 `docs/research/2026-10-31-iceshrimp-architecture-translation.md` 的翻译契约把壳外壳拆成三个子组件，`AppShell.vue` 只保留组合与壳级断点：

- 新增 `web/src/components/layout/Sidebar.vue`（桌面侧栏，消费 `nav.ts` 注册表；250px 全宽 / 80px 图标态，≤1279 收缩——fork `$nav-width`/`$nav-icon-only-width`）、`WidgetPanel.vue`（右列 300px，≤1090 隐藏，与 `WidgetsDrawer` 悬浮按钮的可见区间对齐为 ≥1091）、`MobileSidebar.vue`（≤500 移动开关；组合既有 `NavDrawer` 抽屉与 `MobileNav` 底栏，二者行为已满足契约故保留内部实现，仅把抽屉开关状态从 AppShell 移入）。`AppShell` 继续挂载 `WidgetsDrawer`：≤1090px 时 widget 入口由其悬浮按钮与抽屉承担，501–1090 与 ≤500 两个区间都保持可达。
- `PageHeader.vue` 补齐契约项：>3 个 tab 时非活动 tab 折叠为图标（有图标者），动作区标签 ≤1090 隐藏（`.action-label` 约定，通知页已接入），移动断点随壳从 720 改为 500。
- 断点定稿（fork 主源）：`1100/1090/500/1279/400`；`DeckShell` 与 widget 抽屉的 720 随之改为 500；`.auth` 布局断点保持 720（认证页独立于壳）。底栏 ≤400 收紧按钮尺寸。
- 侧栏/壳几何样式从 `main.css` 移入各组件；`--rail-width` 改 250px、`--rail-width-collapsed` 80px、`--shell-max-width` 1280px。
- 验证：`pnpm typecheck`、`pnpm build`、`go vet`、`go test` 全绿；浏览器实测 1440/1100/840/600/375 的 `scrollWidth/clientWidth` 全部相等（无水平溢出），1440 三区（250+主列+300）、1100 图标态+widget 列、840/600 widget 隐藏、375 抽屉+底栏（含 Esc 关闭、路由变化关闭）、登录页保持裸布局。移动宽度下内建浏览器无法截图，采用 DOM 快照 + 注入式几何探针测量（探针已移除）。

### 11.11 导航集中与账户路由门禁（2026-10-31，重写计划 Task 3）

- `web/src/nav.ts` 的 `NavItemDef` 补齐 `id`、`requiresAuth`、`mobile` 与可选 `unreadKey`：徽标不再用零散的闭包函数，`unreadKey` 指向 `nav.ts` 内的唯一 `UNREAD_SOURCES` 映射（目前仅 `notifications`，仍读 `stores/streaming` 的 `unreadNotifications`）；`DEFAULT_MOBILE_NAV` 改由 `mobile` 标记推导，`NAV_ITEM_KEYS` 改由定义对象推导。`isNavItemVisible` 同时尊重 `requiresAuth`（未登录隐藏）。
- 路由元数据：`/settings/profile`、`/settings/notifications`、`/settings/privacy`、`/settings/account`、`/admin` 带 `meta.requiresAuth`；`/settings/appearance|navigation|widgets|about` 保持公开（纯客户端偏好）。`settings/menu.ts` 的 `privacy` 补上 `requiresAuth` 与路由一致。
- `router/index.ts` 增加 `beforeEach` 守卫：命中 `meta.requiresAuth` 时等待共享的 `refreshSession()`（无 token 立即返回），`session.me` 为空则重定向 `/login?redirect=<fullPath>`，登录成功后 LoginView 用 `redirect` 参数回到原页。守卫只管 UX，授权仍在服务端；`/admin` 路由行为除守卫元数据外不变。
- 散落的「设置」快捷入口（`ProfileHeader`、`ProfileView`、`InstanceWidget` 的 `/settings` 链接）改读 `NAV_ITEM_DEFS.settings.to`，导航单一来源原则成立。
- 已知影响：未登录时点侧栏/抽屉的「设置」会因默认子页 `settings-profile` 受门禁而落到登录页（此前是渲染一份空的资料编辑表单）；深链 `/settings/<未知tab>` 对未登录用户同样经重定向落到登录页。
- 验证：`pnpm typecheck`、`pnpm build`、`go vet` 全绿；`git diff --check` 干净。内建浏览器实测（Go 二进制 @3991，`web/dist`，开发库）：未登录深链 `/settings/account`、`/settings/privacy`、`/admin` 均跳 `/login?redirect=<原路径>`，登录后按 `redirect` 回 `/admin`；未登录 `/settings/appearance` 正常渲染且菜单带「需登录」注记；登录后侧栏五项（含管理员「管理」）与点击导航、返回键正常；流式推送后侧栏「通知」徽标出现「1」；非管理员会话侧栏无「管理」项（与改动前一致）。

### 11.12 通知契约核对与时间线 tab 图标（2026-10-31，重写计划 Task 5）

- 对照 Misskey `packages/backend/src/server/api/endpoints/i/notifications.ts` 的 `paramDef` 与 `core/QueryService.ts` 的 `makePaginationQuery` 逐条核对现有实现：`includeTypes`/`excludeTypes` 在 SQL 的 WHERE 内、`LIMIT` 之前生效（此前一个波次已修复）；显式空 `includeTypes` 返回 `[]`、默认 limit 10、上限 100，与上游一致。
- `includeTypes`/`excludeTypes` 的组合语义按上游 `core/NotificationService.ts` `getNotifications`（L302-306）的 include-wins 处理：**includeTypes 非空时生效并忽略 excludeTypes**，exclude 只在 include 为空时应用（Fix round 1 修正——此前两者按 AND 叠加，属错误偏离）。
- 修正一处真实偏差：`sinceId` 单独出现（无 `untilId`）时 Misskey 返回**升序**余页（`makePaginationQuery` L58-60），此前实现固定 `ORDER BY id DESC`。`internal/note/reaction.go` 的 `NotificationService.List` 现按 sinceId-only → `ASC`、其余 → `DESC` 排序。
- `markAsRead` 语义维持现状（缺省/`true` 把**本次返回的通知**置读，`false` 不触碰），与上游「watermark 全量置读」有意保留差异：通知页与被动视图（widget、deck 列）都显式传 `markAsRead: false`，靠「全部标为已读」按钮落库并广播 `readAllNotifications`；见 11.2 的注释。
- 测试新增（先写失败再修）：`internal/note/reaction_test.go` `TestNotificationListSinceIdAscending`（修复前失败）；`internal/httpapi/notifications_test.go` 新增 `TestNotificationsFilteredCursorPagination`（limit=2 逐页取完一条 mention 流，断言无跳过无重复）、`TestNotificationsSinceIdAscending`。组合过滤用例（`TestNotificationsCombinedIncludeAndExclude`、`TestNotificationListIncludeAndExclude`）在 Fix round 1 重写为钉住 include-wins 语义（重写后、实现修复前为失败状态）。
- 已记录的有意偏离（Fix round 1 审查确认，均为宽松而非报错方向）：`includeTypes`/`excludeTypes` 里的未知/废弃类型值按字面字符串接受（上游 paramDef 会校验枚举并返回 400）；`limit` 越界按钳制处理（上游同样应返回 400）。
- 前端：`TimelineView`、`NotificationsView` 的页头 tabs 补图标（时间线四个 tab 与 `stores/deck.ts` 列图标同源；通知四 tab 用 bell/heart/reply/renote），使 PageHeader 的 >3-tab 图标折叠首次可达；`NotificationItem` 已读行按 Iceshrimp.NET `.notification.read { opacity: 0.7 }`（`NotificationComponent.razor.css` L12-14）降透明度，未读保持强调色高亮。
- 验证：`go test -count=1 ./...`（含数据库与 Redis 集成）全绿、`gofmt -l`/`go vet` 干净、`pnpm typecheck`/`pnpm build` 通过；`git diff --check` 干净。内建浏览器实测（Go 二进制 @3991，`web/dist`，开发库，临时账号 t5alice/t5bob 已清理）：通知页 32 条种子通知分两页翻完无缺失无重复，全部→反应→全部 切换后重新挂载干净；四个时间线源 tab 各自加载对应流；流式新笔记以「1条新笔记」pill 出现、点开仅插入一次；widget 抽屉通知 widget 正常渲染。
- 已知缺口（记录，未在本任务范围）：`users_pack.go:129` 的 `unreadNotificationsCount`/`hasUnreadNotification` 是硬编码 0/false，刷新后通知徽标与「全部标为已读」按钮不可用（只有流式推送会点亮），建议后续接 `NotificationService.UnreadCount`；纯转贴（renoteId）当前不产生 `renote` 通知（Task 5 只涉及通知查询/置读文件，未动通知创建点）。**Task 8 已把这两项补齐，见 §11.15.2。**

### 11.13 Note/时间线/ composer 边界对齐（2026-10-31，重写计划 Task 6）

- 对照 Misskey `server/api/endpoints/notes/create.ts`（renoteId 参数与 `cannotRenoteDueToVisibility`）与 `core/NoteCreateService.ts` L352-356 补齐后端规则：**followers 可见性的笔记只有作者本人可以转贴，specified 笔记一律不可转贴**。此前该规则缺失——`internal/httpapi/notes.go` 里错误 ID `be9529e9-…` 早已声明但从未产生，任何人都能把 followers 笔记转贴成公开笔记（可见性泄漏）。修复走 `internal/note/service.go`（错误 `ErrCannotRenoteDueToVisibility`）+ `httpapi` 错误映射，HTTP 响应与上游同形：`CANNOT_RENOTE_DUE_TO_VISIBILITY` / `be9529e9-fe72-4de0-ae43-0b363c4938af` / 400。
- 修复一处真实前端缺陷：`/api/i/pin` 与 `/api/i/unpin` 的响应是不带 pin 列表的 MeDetailed，而 web 会话整体替换自身，导致置顶后菜单标签在下次 `/api/i` 前一直是「置顶笔记」（再点会重复 pin）。两个 handler 现在都调 `applyPinnedNotes`，响应自带最新列表（测试钉住）。
- 前端按翻译契约（`docs/research/2026-10-31-iceshrimp-architecture-translation.md` §6/§13）对齐：
  - `PagedFeed` 日期分组改为**本地时区完整年/月/日**比较（用 `Date` 解析 `createdAt`，不用 UTC 字符串前缀；本地 00:30 的笔记 UTC 前缀是前一天）。fork `MkDateSeparatedList` 的「只比日-月」分组按契约明确不照抄；分隔行 key 改用本地日期本身，分页 prepend/append 不再移位。
  - `NoteFooter` 按 fork 的 footer 动作顺序（回复、转贴、反应、**引用**、菜单）补了独立「引用」按钮（对应 fork `MkQuoteButton`，此前 UI 完全没有引用入口），并在目标不可转贴（followers 非本人 / specified）时把转贴按钮替换为锁定禁用态（对应 `MkRenoteButton` 的 canRenote）。`Note.vue` 计算 `renotable`（public/home 或本人），引用与转贴共用同一门控。
  - `TimelineFeed` 此前不转发 `@reply`，时间线里点「回复」无任何反应；现在与 NoteView 一样走 `os.openPostForm({ reply })`。纯转贴展示/操作目标分离（`useNote(note, actionTarget)`、pin 用 actionNote.id）保持不变，单行单 `<article>` 结构核对过与 fork 一致。
- composer 决策：保留 prop 驱动组件不变。`os.openPostForm` 的全局弹层（OsOverlay → PageWindow）与时间线 `fixed` 内嵌是同一组件的两种挂法，草稿/可见性/回复/引用目标/附件状态已全部内聚在组件里，API 调用都在 `api/client.ts`——契约已满足，无需再动。
- 验证：`go vet`、`gofmt -l internal/ cmd/`、`git diff --check` 全干净；`CINDERFISH_TEST_DATABASE_URL=… CINDERFISH_TEST_REDIS_URL=… go test -count=1 ./...` 全绿（新测试先失败后通过：`TestRenoteTargetVisibility` 在规则缺失时 200 泄漏、`TestIPinUnpinAndUsersShow` 的 pin 响应断言为空）；`pnpm typecheck`/`pnpm build` 通过。内建浏览器走查（Go 二进制 @3991，开发库，临时账号 t6alice/t6bob，已清理）：日期分隔线在本地 00:30 边界与 10-31/11-01 交界正确分组（修复前 UTC 前缀会错误拆分 10-31）；纯转贴行单 article 单 footer；回复/引用弹窗与发布、反应选择器、置顶/取消置顶标签翻转（pin 修复后）均正常。
- 仍缺（按契约显式降级，不虚构）：`notes/conversation`、`notes/children`、`notes/replies`、`notes/renotes`、`users/following`、`users/followers` 端点不存在——会话串只有单层父笔记（`NoteParent`），转贴悬停用户列表省略，关注/粉丝列表维持 Task 7 的「暂不可用」态。转贴按钮保持单击直接 `notes/create {renoteId}`（默认 public），fork 的按可见性转贴菜单未引入（记录为决策，后端契约已支持）。

### 11.14 个人页/设置/管理/widget 组合对齐（2026-10-31，重写计划 Task 7）

- **个人页**：新增 `components/profile/ProfileSection.vue`（分区标题：图标 + 名称 + 计数徽章 + 说明 + 动作插槽；
  `plain` 去卡片外框供 deck 资料窗复用）。置顶区与列表区都改成带标题的分区（「置顶笔记」带条数与说明、
  「最近的笔记/媒体/回复」带排序说明），tab 键表扩为 `notes|media|replies|following|followers`，全部读写
  `route.query.tab`（点 tab 写查询、点帖子计数回到裸 URL、刷新与分享保持）。>3 个 tab 时 PageHeader 的折叠
  规则生效（非活动 tab 只留图标），与 fork `MkPageHeader` 一致。
- **关注/关注者列表的真实降级**（对应翻译契约 §15.3）：两个 tab 不请求任何数据，渲染「关注/关注者列表暂不可用」，
  说明后端尚无 `users/following`/`users/followers`，并显示用户资料里的真实计数（注明计数准确、列表待接口）。
  `ProfileCard` 的三个计数里只有帖子是可点链接；另两个保持纯数字并在卡片内注明「列表接口尚未提供」。
  deck 资料窗（`ProfileHeader` + `DeckPageWindow`）同样注明，并按 `ProfileView` 补上 `NOT_FOLLOWING` 分支
  （此前取消关注遇到已失效关系只会抛「取消关注失败」）。
- **设置与管理共用一套左菜单**：新增 `components/ui/SectionMenu.vue` + `ui/section-menu.ts`（分组条目
  `{key,label,icon,to,note}`，对应 fork `MkSuperMenu`；宽屏竖排 sticky、窄容器一行横向滚动胶囊）。
  `views/settings/SettingsMenu.vue` 删除，`settings/menu.ts` 导出 `settingsMenuDef()`；管理端 `AdminView`
  用同一个组件渲染 `?tab=` 分组。两处都用容器宽度（<600）而非视口宽度判定窄态。
- **修一处真实水平溢出**：设置页窄态把 `.body` 从 grid 切成 flex 列时继承了 `align-items: start`，flex 子项
  因此按 max-content 定宽，横向滚动的菜单条把整页撑宽（600px 视口下文档 796px）。`.narrow .body` 显式
  `align-items: stretch`，`SectionMenu` 窄态补 `width: 100%`/`min-width: 0`。该 bug 在 Task 2/3 的实现里已存在。
- **管理门禁**：`/admin` 的非管理员卡片按登录态给不同文案与出口（已登录非管理员：说明服务端也会拒绝，出口
  「回到时间线」；未登录：登录）。**授权边界仍是服务端**，新增测试钉住。
- **widget 面板**：编辑态从 `WidgetStack` 局部状态提升到 `stores/widgets` 的 `widgetEditing`（UI 态，不落盘），
  新增 `widgets/WidgetEditToggle.vue` 供右栏与抽屉共用；`WidgetPanel` 拥有标题与动作插槽（默认放编辑开关）、
  `WidgetStack` 只负责有序列表（编辑态另有「在设置中管理小组件」入口）、`WidgetsDrawer` 头部为标题 + 编辑开关 + 关闭。
  每张卡片经 `WidgetCard` 的标题/`#actions`/默认内容三段：实例卡新增「刷新实例信息」（`reloadInstance()`）与
  失败行「实例信息加载失败 + 重试 + 原始错误」；关于卡把探索入口做成头部动作；通知卡新增「打开通知页」并保留
  「全部标为已读」；时间线卡把高度选择器从工具条移进头部动作区，工具条只留来源分段。卡片用 `useId()` 绑定
  `aria-labelledby`，一叠卡片在无障碍树里是带名字的区块而不是匿名 section。
- **测试**：`internal/httpapi/admin_auth_test.go` 新增 `TestAdminEndpointsRequireAdministrator`——对全部七个
  `/api/admin/*` 路由断言无凭证 401、非管理员 403 `FORBIDDEN`、管理员可通过门禁（载荷都设计成授权后才 400，
  不污染共享测试库）。`web_test.go` 的 SPA 回退用例补上 `/settings/appearance`、`/settings/account`、`/admin`、
  `/admin?tab=users`、`/@alice?tab=followers` 这些深链。反证：临时去掉 `handleAdminShowUsers` 的 `requireAdmin`
  后该测试立刻失败（`without credential = 200` / `as non-admin = 200`），恢复后通过。
- **验证**：`go vet ./...`、`gofmt -l internal/ cmd/`、`git diff --check` 干净；
  `CINDERFISH_TEST_DATABASE_URL=… CINDERFISH_TEST_REDIS_URL=… go test -count=1 ./...` 全绿；
  `pnpm typecheck`、`pnpm build` 通过。内建浏览器（Go 二进制 @3991、`web/dist`、开发库）实测：`/@uicheck`
  置顶区 + 「最近的笔记」分区标题 + 帖子计数链接；`?tab=followers` 深链与 tab 点击写法、计数链接清空查询均正常；
  `/settings/profile|appearance|account` 左菜单 + 右内容，`/settings/nope` 回退个人资料；未登录 `/settings/account`
  跳 `/login?redirect=/settings/account`，登录后回跳原页；非管理员 `/admin?tab=users` 显示门禁卡、管理员显示
  分组左菜单与用户分区；1024px 下右栏由悬浮「小组件」按钮 + 抽屉承担，抽屉头部标题/编辑/关闭齐全，编辑态删除到
  空栈显示「还没有小组件…」、恢复默认可用；停掉后端后实例卡显示失败行与重试，重启后点重试恢复；溢出探针
  （临时注入 `web/dist`，已删除）在 380/600/840/1440 × `/`、`/@uicheck`、`/settings/appearance`、
  `/settings/account`、`/admin?tab=users` 共 20 组全部 `scrollWidth == clientWidth` 且无未裁剪溢出元素。
- **记录未做**：deck 列内的资料窗仅在代码层与 `ProfileView` 对齐（`NOT_FOLLOWING` + 分区标题），未在 deck 模式
  下重跑浏览器流程；`WidgetSettings` 设置页仍保留一份独立的排序 UI（与面板编辑态同源数据，未合并）。

### 11.15 API 缺口审计、契约收尾与最终清理（2026-10-31，重写计划 Task 8）

审计口径：把前端每个页面动作对回 Misskey 端点源码（`packages/backend/src/server/api/endpoints/...` 的 `paramDef`/`res`/`errors`，以及 `core/` 里实际执行语义的服务），再看它需要的表/列。
**Misskey 只提供 API 契约；界面参考仍是 Firefish/Iceshrimp**（见 `AGENTS.md` 的参考边界）。
本分支提交：`31b9137`（契约收尾与徽标播种）、`9167f1e`（转贴可见性收敛）、文档提交（本节）；更早的 `d414bd9`…`70509eb` 是 Task 2–7。

#### 11.15.1 分组总览

| 组 | 定义 | 条数 |
|----|------|------|
| **(a) 已实现** | 上游有契约、现有数据模型直接支撑、且改动小而自洽 | 6 |
| **(a) 未实现** | 上游有契约、数据也够，但消费端是新页面/新交互，或需要一次跨端点的读路径整改 | 15 |
| **(b) 需 schema/数据模型** | 上游有契约，但缺表、缺列，或需要新的后台任务（媒体清理、联邦投递、排行数据） | 12 |
| **(c) 上游没有对应** | Misskey API 里不存在，只能是客户端行为或明确不在范围 | 6 |

#### 11.15.2 (a) 已实现（六项，均先写失败测试）

**1. MeDetailed 的未读通知状态**
- 上游：`core/entities/UserEntityService.ts` L612（`hasUnreadNotification`）与 L614（`unreadNotificationsCount`）、`models/json-schema/user.ts` L572。
- 请求/响应：`POST /api/i`、`/api/i/update`、`/api/i/pin`、`/api/i/unpin` 的响应里，`unreadNotificationsCount` 为未读行数、`hasUnreadNotification` 为 `count > 0`。
- 持久化：`notifications` 表 `WHERE recipient_id = $1 AND is_read = FALSE`，复用已有的 `NotificationService.UnreadCount`。
- 前端消费：`layouts/AppShell.vue` 在会话就绪时用 `setUnreadNotifications` 播种 `stores/streaming.ts` 的 `unreadNotifications`；`nav.ts` 的 `UNREAD_SOURCES` 与侧栏/底栏徽标、`NotificationsView`/`widgets/NotificationsWidget`/`settings/NotificationSettings` 的「全部标为已读」按钮都读它。
- 测试：`internal/httpapi/notifications_test.go` `TestIUnreadNotificationCount`（新账号 0/false → 他人反应后 1/true → `mark-all-as-read` 后 0/false）。修复前红：`unreadNotificationsCount = 0, want 1`。
- 有意保留：`hasUnreadNotification` 是布尔投影而非"最近已读游标"；上游同一字段也是投影，行为一致。

**2. 回复目标的可见性与回复的受众收敛**
- 上游：`core/NoteCreateService.ts` L374-406（顺序：存在 → 纯转贴 → `isVisibleForMe` → specified 扩权）与 `core/entities/NoteEntityService.ts` L308-359（`isVisibleForMe` 规则）；错误表在 `endpoints/notes/create.ts`。
- 请求/响应：`POST /api/notes/create { replyId, visibility, … }`；错误 `400 CANNOT_REPLY_TO_AN_INVISIBLE_NOTE` / `b98980fa-3780-406c-a935-b6d0eeee10d1`，`400 CANNOT_REPLY_TO_SPECIFIED_VISIBILITY_NOTE_WITH_EXTENDED_VISIBILITY` / `ed940410-535c-4d5e-bfa3-af798671e93c`。
- 持久化：`notes.visibility`、`note_visibilities.visible_user_id`、`followings(followee_id, follower_id)`、`notes.mentions`/`reply_user_id`；远程作者→远程读者的"假设互相关注"分支按 `notes.user_host` + `users.host` 还原（`internal/note/service.go` `noteVisibleToTx`）。
- 前端消费：`components/composer/PostForm.vue` 的回复提交、`NoteFooter` 的回复按钮；错误信封经 `api/client.ts` 的 `ApiError` 直接展示 `message`。
- 测试：`internal/httpapi/notes_test.go` `TestReplyTargetVisibility`（未列入 specified 名单的人被拒、被列入者不能把可见性放宽、followers 笔记在关注前后分别为拒/允、作者本人始终可回、公开笔记仍可回）。修复前红：`reply by unaddressed user: status 200`。
- **回复的受众收敛**（Fix round 1 补上，reviewer I1）：上游在 `core/NoteCreateService.ts` L544-573 于上述检查之后再收敛回复的可见性——public 回复 home 笔记 → home，public/home 回复 followers 笔记 → followers，`reply.localOnly` 让回复也 `localOnly`。`internal/note/service.go` 的 Create 现在照同一顺序收尾（纯转贴检查 → 可见性 → specified 扩权 → 收敛），写库前改写 `p.Visibility`/`p.LocalOnly`。这条此前缺失：**关注者可以在编辑器里选 public 回复 followers 笔记，回复被存成公开**，而 `packNoteDetail` 会把父帖内容嵌进这条更宽可见的回复里。
- 测试（先红后绿）：`TestReplyInheritsTargetVisibility`——public 回复 home → home、public 回复 followers → followers、localOnly 继承、显式更窄（followers 回复 home）保持、同宽（home 回复 home）保持、public 笔记回复仍 public。修复前红：`public reply to a home note visibility = public, want home`（另两条同理）。
- composer 决策（保持不动，记录理由）：Iceshrimp 的 `MkPostForm.vue` L479-496 会在回复时把可见性预置为父帖的取值（并对 specified 拉取收件人）。Cinderfish 的 `PostForm` 没有按用户保存的默认可见性，`specified` 收件人是一行手填 handle 而不是 id 集合，照抄需要额外的 users/show 反查并把父帖受众回填到输入框；reviewer 明确「服务端收敛是契约、composer 提示可选」。本波次只做服务端收敛，composer 仍显示用户选的可见性，实际落库更窄——记录为已知 UX 偏差（不是契约偏差）。
- 未覆盖：block 检查（`blocks` 表存在但本波次未接）。

**3. 回复 / 转贴 / 引用通知**
- 上游：`core/NoteCreateService.ts` 的 `NotificationManager`（L64-162；`push` L85-103、`notify` L105-162：自不通知、mention 被后续 reply/renote 覆盖、specified 只通知名单内、followers 不过滤）与创建点 L856-895（`reply` L861-877、`renote`/`quote` L879-892）。
- 请求/响应：`POST /api/notes/create` 的副作用（无新增请求字段）；写入的通知 `type ∈ {reply, renote, quote}`、`note_id` = 新建笔记、`notifier_id` = 作者。
- 持久化：`notifications(type, notifier_id, note_id)`；收件人必须是本地用户（`users.host IS NULL`，`NotificationService.CreateForLocal`），远程收件人由 ActivityPub 投递。
- 前端消费：`components/NotificationItem.vue` 已有 `reply`/`renote`/`quote` 三套中文文案与图标，通知页「转贴」tab 用 `includeTypes: ["renote"]`。
- 测试：`TestReplyAndRenoteNotifications`（回复通知的 noteId/userId、回复+提及只留一条 reply、自回复不通知、纯转贴 → renote、引用 → quote、自转贴不通知）、`TestNotificationRespectsNoteVisibility`（specified 回复不通知看不见它的目标）、`internal/note/reaction_test.go` `TestNotificationCreateForLocal`（远程收件人不落行）。修复前红：`reply notifications = 0, want 1`。
- 已知偏差：上游按可见性给 followers 通知不过滤（上游 TODO 亦如此）；「转贴」tab 只查 `renote`，引用通知落在「全部」里（客户端 tab 未扩，记录为后续项）。

**4. 纯转贴不可反应**
- 上游：`core/ReactionService.ts` L121-123；错误表在 `endpoints/notes/reactions/create.ts`。
- 请求/响应：`POST /api/notes/reactions/create` 对纯转贴返回 `400 CANNOT_REACT_TO_RENOTE` / `eaccdc08-ddef-43fe-908f-d108faad57f5`。
- 持久化：判定用 `internal/note/renote.go` 的 `IsPureRenote`（`renote_id` 存在且 text/cw/poll/reply/files 都为空）；引用（quote）照旧可反应。
- 前端消费：`components/note/use-note.ts` 已把反应目标指向被转贴笔记，纯转贴本身不渲染可点的反应条。
- 测试：`TestCannotReactToPureRenote`（纯转贴 400，原笔记与引用 200）。修复前红：`react to pure renote: status 200`。
- 同一 handler 的 `NO_SUCH_NOTE` 用自己的 id：`endpoints/notes/reactions/create.ts` 的 `033d0620-5bfe-4027-965d-980b0c85a3ea`，不再借用 `notes/show` 的 `24fcbfc6-…`（Fix round 1，M4；`TestReactions` 末尾断言）。

**5. 转贴不放大被转贴笔记的受众**
- 上游：`core/NoteCreateService.ts` L499-530（public 不变；home → 最多 home；自己的 followers → 最多 followers；specified 拒绝）与同段的 localOnly 继承（L526-528）。
- 请求/响应：`POST /api/notes/create { renoteId }` 无新字段；`createdNote.visibility` / `createdNote.localOnly` 现在按被转贴笔记收敛，而不是客户端传来的默认 `public`。
- 持久化：`notes.visibility`、`notes.local_only`；判定读取被转贴笔记的同名列（事务内已加载）。
- 前端消费：`components/note/use-note.ts` 的转贴按钮只发 `{renoteId}`（Task 6 的决定：不做按可见性选择的转贴菜单），因此完全依赖服务端收敛。
- 测试：`internal/httpapi/notes_test.go` `TestRenoteInheritsTargetVisibility`（home → home、自己的 followers → followers、localOnly → localOnly、引用同样收敛）。修复前红：`renote of a home note visibility = public, want home`。
- 说明：这是 Task 6 同类可见性泄漏（followers 笔记被转成公开）的剩余一半——Task 6 只挡住了「不能转贴他人 followers 笔记」，没有收敛「home 笔记被转成 public」。

#### 11.15.3 (a) 未实现：数据够，但消费端是新 UI 或需要跨端点整改

| 端点 / 行为 | 上游源码 | 现有数据 | 前端消费点 | 本波次不做的原因 |
|---|---|---|---|---|
| `users/following` | `endpoints/users/following.ts`；响应 ref `Following`（`json-schema/following.ts`：`id/createdAt/followeeId/followerId/followee/follower`） | `followings(follower_id, followee_id, created_at)` + `users`/`user_profiles` | `views/ProfileView.vue:153-171` 的 following tab、`components/profile/ProfileCard.vue:62-71` 的计数 | 后端约百行，但消费端要从「不可用卡片」变成真实列表页（Following 对象、逐行关注状态、分页）——超出 Task 8「小而自洽」 |
| `users/followers` | `endpoints/users/followers.ts` | 同上（方向相反，`followee_id` 为被看的人） | 同上 | 同上 |
| `notes/replies` | `endpoints/notes/replies.ts`（`noteId` + `limit/sinceId/untilId/sinceDate/untilDate`） | `notes.reply_id`（`idx_notes_reply`） | `views/NoteView.vue:47`、`:96` 的回复区 | 需要分页组件 + 空态 + 线程语义，属新 UI |
| `notes/children` | `endpoints/notes/children.ts` | `notes.reply_id` | 同上 | 同上 |
| `notes/conversation` | `endpoints/notes/conversation.ts`（`limit`/`offset`） | `notes.reply_id`、`reply_user_id` | 笔记详情「查看对话」（目前无入口） | 上游 `offset` 语义需要向上递归/递归 CTE，工作量不小 |
| `notes/renotes` | `endpoints/notes/renotes.ts` | `notes.renote_id`（`idx_notes_renote`） | 转贴计数点击列表（目前无入口） | 无消费端 |
| `notes/reactions`（列表） | `endpoints/notes/reactions.ts`（`type` 过滤 + 分页） | `note_reactions` + `users` | 「谁反应了」浮层（目前无） | 无消费端 |
| `i/change-password` | `endpoints/i/change-password.ts` | `users.password_hash`（bcrypt） | `views/settings/AccountSettings.vue:66` | 改密后既有 token 是否失效是产品决定，本波次不碰认证语义 |
| `blocking/create`、`blocking/delete`、`mute/create`、`mute/delete` | `endpoints/blocking/*`、`endpoints/mute/*` | `blocks`、`mutes` 表已存在（v1 schema） | `views/settings/PrivacySettings.vue:73-80` 的不可用清单 | 需要列表 UI + 时间线/通知/详情的过滤接线，属新功能 |
| `following/requests/*`（锁定账号） | `endpoints/following/requests/*` | `follow_requests`、`users.is_locked` | `views/settings/PrivacySettings.vue:79`（关注审批） | 与上一条同批 |
| `users/update` 的 `followersVisibility`/`followingVisibility` | `endpoints/users/update.ts` | `user_profiles.followers_visibility`/`following_visibility` | `views/settings/PrivacySettings.vue:55-70`（只读展示） | 要与 `users/following` 的可见性校验一起落地 |
| `admin/drive/files` | `endpoints/admin/drive/files.ts` | `drive_files` | `views/admin/AdminFiles.vue:71` 占位 | 管理端列表 + 审核语义 |
| `i/signin-history` | `endpoints/i/signin-history.ts` | `signins` | 账户设置（可选展示） | 无消费端 |
| `notes/reactions/create` 的**读可见性** | `core/ReactionService.ts` L121（`isVisibleForMe`） | `noteVisibleToTx` 已存在（§11.15.2 第 2 项） | 所有笔记详情/反应入口 | 时间线已按可见性过滤；反应入口的可见性校验尚未接线（回复入径与 `notes/show` 已在最终评审波次落地，见下） |
| `notes/search`、`users/search`、`notes/search-by-tag` | `endpoints/notes/search.ts` 等 | `notes.text`、`notes.tags`（无全文索引） | **无搜索 UI** | 需要索引 + 查询设计；前端也没有入口 |

**回复的受众收敛已在 Fix round 1 落地**（§11.15.2 第 2 项），因此不在上表：首轮只做了「回复目标可见性」的拒绝，漏掉了「回复本身按目标收敛可见性/localOnly」，reviewer 以 Important 指出后补齐（`core/NoteCreateService.ts` L544-573）。

**读路径的可见性在整分支评审波次落地**（见 §11.15.8）：`notes/show` 现在按 viewer 判定可见性，不可见时以 404 `NO_SUCH_NOTE` 回答（不确认笔记是否存在），`packNoteDetail` 嵌入的父帖/被转贴笔记同样按 viewer 判定、不可见则只保留 id 不带内容。

**已实现的端点不在这张表里，但有一处路径偏差要记住（Fix round 1，M3）**：上游的通知列表端点是 `i/notifications`（`endpoints/i/notifications.ts`，注册为 `POST /api/i/notifications`，`paramDef` 有 `includeTypes`/`excludeTypes`/`limit`/`sinceId`/`untilId`，`res` 是 `Notification[]`），本实现把它注册在 `POST /api/notifications`——**路径不同，行为已实现**（前端 `api/client.ts` 与全部测试都用 `/api/notifications`）。Task 5 的 `notifications/mark-all-as-read` 路径与上游一致。本轮不改路由（改路径会同时改动前端、全部通知测试与既有部署的客户端），记录为已知偏差；对齐方式是保留 `/api/notifications` 为 canonical，把 `/api/i/notifications` 注册为同一 handler 的兼容别名。

#### 11.15.4 (b) 上游有契约，但要动 schema 或数据模型

| 端点 | 上游源码 | 缺什么 | 前端现状 |
|---|---|---|---|
| `i/delete-account` | `endpoints/i/delete-account.ts` | 除级联删除外还要媒体清理、AP `Delete` 投递、管理员保护策略 | `AccountSettings.vue:95-100` 标注不可用 |
| `i/2fa/*` | `endpoints/i/2fa/*` | 需要 two-factor secret / 恢复码列（现无） | `AccountSettings.vue:73-74` |
| `i/registry`、`i/registry/set` | `endpoints/i/registry/*` | 需要 `registry_items` 表 | 无（客户端偏好走 localStorage） |
| `users/lists/*` | `endpoints/users/lists/*` | 需要 `user_list` / `user_list_membership` 表 | 无 |
| `notes/favorites/*`、`i/favorites` | `endpoints/notes/favorites/*` | 需要 `note_favorites` 表 | 无 |
| `announcements`、`i/announcements`、`i/read-announcement` | `endpoints/announcements.ts` 等 | 需要 `announcements` 表 + 已读游标 | 无 |
| `notes/featured`、`hashtags/trend`、`charts/*` | `endpoints/notes/featured.ts`、`endpoints/charts/*` | 需要排行/图表数据（`events` 表未写入） | `ExploreView.vue` 没有趋势区 |
| `federation/instances`、`admin/federation/instances` | `endpoints/federation/instances.ts` | 需要 `instances` 表（合并/封禁状态） | 管理端只有联邦模式与白名单 |
| `admin/meta`、`admin/meta/update` | `endpoints/admin/meta.ts`、`endpoints/admin/meta/update.ts` | 注册模式/站点名/描述要能在线改：需要可写的 meta 模型 + 配置热加载语义（现在 `registration_mode` 只读自 TOML，改后必须重启） | `views/admin/AdminSettings.vue:278-290` 只展示当前模式并说明需重启 |
| `admin/emoji/*` | `endpoints/admin/emoji/*` | 需要 `emoji` 表 | MFM 只认内置 emoji 目录 |
| `channels/*` | `endpoints/channels/*` | 需要频道表与成员关系 | 无 |
| `clips/*`、`antennas/*`、`pages/*`、`play/*`、`ads/*`、`gallery/*` | 各自目录 | 各自的表 | 无 |

上游通知模型里还有一个内部列 `targetNoteId`（`models/Notification.ts` L43-44），它不出现在打包后的响应里；本实现的转贴通知用 `note_id` 指向转贴本身，被转贴笔记经 `notes.renote_id` 取回，因此**不需要新增列**。

#### 11.15.5 (c) 上游没有对应（客户端行为或明确不做）

| 行为 | 现状 |
|---|---|
| deck 列布局、列宽、列顺序 | 纯客户端：`stores/deck.ts` + `localStorage["cinderfish.deck.columns"]`，Misskey 无对应端点 |
| 导航顺序与可视化编辑（桌面/移动两套） | 纯客户端：`stores/nav.ts` + `nav.ts` 注册表，`localStorage["cinderfish.ui"]` |
| widget 栈与编辑态 | 纯客户端：`stores/widgets.ts` + `localStorage["cinderfish.ui.widgets"]`；编辑态不落盘 |
| 主题（auto/light/dark）、字号 | 纯客户端：`stores/theme.ts`、`views/settings/font-scale.ts`，同步 `meta theme-color` |
| 通知页「未读」视图 / 只读 tab | 客户端过滤（`markAsRead: false` + `includeTypes`），上游无「未读」端点 |
| 自助找回密码（邮件重置） | Misskey 无此端点，本实现同样不做；改密属于 (a) §11.15.3，重置不在范围内 |

#### 11.15.6 清理：没有可删的壳层包装

第 3 步要求「只有在 grep 证明没有任何路由/导入/store/模板引用后才能删」。实测结论：**本波次没有可删文件**。

- 可达性分析（脚本解析 `web/src` 里全部静态与动态 `import`，从 `src/main.ts` 出发）：`src` 下 101 个 `.vue`/`.ts` 文件里，除类型声明 `src/env.d.ts` 外全部可达；没有任何孤儿组件。
- 逐个候选的存活证据（"仍被引用"即不可删）：

  | 候选 | 存活引用 |
  |---|---|
  | `components/PagePlaceholder.vue` | `views/NotFoundView.vue:2` |
  | `components/ProfileHeader.vue`（deck 资料卡） | `components/deck/DeckPageWindow.vue:9` |
  | `components/ui/MobileNav.vue` / `NavDrawer.vue` | `components/layout/MobileSidebar.vue:3-4` |
  | `components/TimelineFeed.vue` | `views/TimelineView.vue:4`、`components/deck/DeckColumn.vue:19`、`components/widgets/TimelineWidget.vue:3` |
  | `components/ui/PageWindow.vue` | `components/deck/DeckPageWindow.vue:11`（`PostForm` 也经 `os.ts` 走同一弹层） |
  | `views/SettingsView.vue` | `router/index.ts:52` 的路由组件（内部只渲染 `settings/SettingsLayout.vue`） |
  | `views/AdminView.vue` | `router/index.ts:107` |
  | `layouts/AuthLayout.vue` | `views/LoginView.vue:4`、`views/SignupView.vue:5` |
  | `views/settings/menu.ts` | `views/settings/SettingsLayout.vue:8` |

- 因此本波次**删除 0 个文件**。唯一"看起来像包装"的 `views/SettingsView.vue` 仍被路由引用，按规则不动；它的合并（路由直接指向 `settings/SettingsLayout.vue`）留给后续重构，并需同时改 `router/index.ts` 与 `web_test.go` 的深链断言。
- 后端侧也没有删除：所有 `internal/httpapi` 处理器都在 `registerRoutes` 里有路由（`notes.go`/`notifications.go` 等只新增行为，未留死代码）。

#### 11.15.7 最终门禁（2026-10-31，本分支实测输出）

```
$ gofmt -l internal/ cmd/
（空）

$ go vet ./...
（退出 0，无输出）

$ CINDERFISH_TEST_DATABASE_URL="postgres://cinderfish:cinderfish@localhost:5432/cinderfish_test" \
  CINDERFISH_TEST_REDIS_URL="redis://127.0.0.1:6379/0" go test -count=1 ./...
ok  	github.com/Aurorainic/cinderfish/internal/account	1.124s
ok  	github.com/Aurorainic/cinderfish/internal/auth	0.041s
ok  	github.com/Aurorainic/cinderfish/internal/config	0.005s
ok  	github.com/Aurorainic/cinderfish/internal/federation	0.476s
ok  	github.com/Aurorainic/cinderfish/internal/httpapi	10.650s
ok  	github.com/Aurorainic/cinderfish/internal/idgen	0.003s
ok  	github.com/Aurorainic/cinderfish/internal/media	0.269s
ok  	github.com/Aurorainic/cinderfish/internal/note	0.172s
ok  	github.com/Aurorainic/cinderfish/internal/streaming	0.709s
（cmd/* 与 internal/db、internal/testdb 为 [no test files]）

$ cd web && pnpm typecheck && pnpm build
（vue-tsc 无输出；`vite build` ✓ built in 605ms，dist 产物正常）
```

**真实二进制 API 走查**（`go build -o /tmp/cf-t8 ./cmd/cinderfish`，配置 `/tmp/cf-t8.toml`：端口 3992、
`web_dir` 指向本分支 `web/dist`、开发库 `cinderfish`、`registration_mode=admin`、`federation_mode=none`；
临时账号 `t8m2`/`t8o2`，Fix round 1 用 `t8m3`/`t8o3` 复测）实测：

| 场景 | 结果 |
|---|---|
| `/api/i` 新账号 | `unreadNotificationsCount=0, hasUnreadNotification=false` |
| `t8o2` 反应 `t8m2` 的笔记 | `/api/i` → `1 / true` |
| `t8o2` 回复 / 纯转贴 / 引用同一笔记 | `/api/notifications` 依次出现 `quote, renote, reply, reaction`（`noteId` 齐全），未读 4 |
| `notifications/mark-all-as-read` | `/api/i` → `0 / false` |
| 非关注者对 followers 笔记回复 | `400 CANNOT_REPLY_TO_AN_INVISIBLE_NOTE` / `b98980fa-…` |
| 关注后再回复 | `200` |
| 被列入 specified 名单者用 public 回复 | `400 CANNOT_REPLY_TO_SPECIFIED_VISIBILITY_NOTE_WITH_EXTENDED_VISIBILITY` / `ed940410-…`；改用 specified 回复 `200` |
| 对纯转贴反应 | `400 CANNOT_REACT_TO_RENOTE` / `eaccdc08-…` |
| `t8o3` 用 public 回复 `t8m3` 的 home 笔记 | `createdNote.visibility` → `home` |
| `t8o3` 用 public 回复 `t8m3` 的 followers 笔记（已关注） | `createdNote.visibility` → `followers` |
| 回复 localOnly 笔记 | `createdNote.localOnly` → `true`（visibility 仍 public） |
| public 回复 public 笔记 / followers 回复 home 笔记 | `public` / `followers`（不放大、显式更窄保持） |
| 对不存在的笔记反应 | `404 NO_SUCH_NOTE` / `033d0620-5bfe-4027-965d-980b0c85a3ea` |
| `t8o2` 转贴 `t8m2` 的 home 笔记 | `createdNote.visibility` → `home` |
| 转贴 localOnly 笔记 | `createdNote.localOnly` → `true` |
| 四条时间线（local/global/hybrid/home） | 均 200 且返回非空 |
| 匿名 `POST /api/meta`、`/api/notes/local-timeline` | `200`，本地时间线返回 5 条 |
| 匿名 `notes/timeline`（home）、`notifications`、`/api/i` | 均 `401`（与前端对未登录隐藏这些入口一致） |
| 匿名 `GET /`、`/notifications`、`/settings/appearance`、`/admin`（`Accept: text/html`） | 均 `200 text/html`，SPA 回退正常（deep link 可硬刷新） |

**浏览器可视化走查未执行**：本轮内建浏览器返回 `BROWSER_USER_TAKEOVER`（用户接管了浏览器面板），
工具契约要求当轮停止，因此 1440/1280/1100/840/720/600/380 的截图与溢出实测**没有做**。
本波次的 UI 改动只有「徽标初值来自会话」这一处逻辑（无 CSS、无 DOM 结构变化，徽标渲染路径在 Task 3 已实测），
但**按证据规则仍记为未验证**，见 §14。

#### 11.15.8 整分支评审修复波：读路径可见性、打包去重与置顶 viewer（2026-10-31）

整分支评审（76 提交，181 文件）给出「合并前需修」的结论，本波次处理其中的三处：

| 问题 | 处理 |
|---|---|
| `notes/show` 不判可见性，followers/specified 笔记按 id 直读即得；`packNoteDetail` 嵌入的父帖同样不过滤，公开回复会带出父帖正文 | `handleNotesShow` 取可选 viewer，按 `noteVisibleToTx`（与上游 `isVisibleForMe` 逐条一致）判定，不可见时以 404 `NO_SUCH_NOTE` 回答（不确认存在性）；`packNoteDetail` 嵌入父帖/被转贴笔记前同样判定，不可见则只留 id 不带内容。`note.Service.VisibleTo` 是新暴露的读路径入口 |
| HTTP 通知列表与流式通知各有一套打包，字段已开始漂移（HTTP 多一个 `body`） | 两条路径统一走 `s.packNotification`，并删掉上游 Notification 类型没有的 `body` 字段 |
| `applyPinnedNotes` 用匿名 viewer 打包置顶笔记，登录用户看不到自己的 `myReaction` | `applyPinnedNotes` 增加 viewerID 参数，`users/show`、`/api/i`、`i/pin`、`i/unpin` 都传入真实 viewer |

测试：`TestNotesShowVisibility`（匿名/陌生人是 404，作者、被指名的用户、关注者正常）、`TestPinnedNotesPackedForViewer`、
`TestNotificationHasNoBodyField`。上游比对：`NoteEntityService.isVisibleForMe`（L308-359）与
`core/entities/NotificationEntityService.ts` 的「不可达则返回 null」语义。

**已知遗留（本波次不做）**：`notes/reactions/create` 的读可见性仍未接线（表里保留该行）；`notes/delete` 仍借用
`notes/show` 的错误 id；提及被回复/引用吸收时不再发主通道 `mention` 事件（客户端会多收一条事件，不影响通知内容）。

### 11.16 Firefish 前端整体移植 Task 1：参考源码、许可证与能力矩阵（2026-11-01）

**方向变更**：前端不再做「在现有页面上渐进改造」，改为以 Firefish 客户端的页面结构、壳、Deck、信息密度与
交互模型为基线整体移植，API 一律接本项目的 Misskey 兼容端点。2026-11-01 的设计文档
（`docs/superpowers/specs/2026-11-01-firefish-frontend-port-design.md`）取代旧的渐进式设计；
旧计划 Task 2–8 的**后端与 API 侧**成果保留（§11.10–§11.15），前端组件不再按旧结构演进。

本任务的产物是三份 research 文档，本轮**没有改产品代码**（`web/`、`internal/`、`cmd/` 零改动）：

| 文档 | 内容 |
|------|------|
| `docs/research/2026-11-01-firefish-port-source-and-license-register.md` | 四个参考的固定 commit、来源 URL、许可证、稀疏范围、可构建性判定、复制边界 |
| `docs/research/2026-11-01-firefish-port-capability-matrix.md` | 首批 12 项能力 → Firefish 页面 → 端点 → 当前状态（已支持/需适配/需新增后端/隐藏）与降级 |
| `docs/research/2026-11-01-runtime-adaptation-contract.md` | `api`/`apiWithDialog`/`alert`/`confirm`/`popup`/`popupMenu`/`contextMenu`/`inputText`/`post`/`claimZIndex`/`me`/`i18n`/`instance`/路由/`AppIcon` 的 `shim`·`rewrite`·`drop` 归属 |

**固定的参考版本**：Firefish `02c31e0`（2024-12-11）、Iceshrimp Vue fork `1fa829ed10f4`（2025-07-25，
经 Codeberg API 复核）、Iceshrimp.NET `2c61359`（2026-10-27）、Misskey `c05e844`（2026-10-29）。

**参考源码不可直接构建**（登记为只读参考，不作为本地依赖）：

- Firefish 的稀疏范围只有 `packages/client/src` 与 `packages/sw/src`，`packages/client/index.html`、
  根 `scripts/copy-index.mjs`、`packages/firefish-js` 都不在磁盘上；`packages/client/package.json` 依赖
  `firefish-js: workspace:*`，`pnpm install` 会以 `ERR_PNPM_WORKSPACE_PKG_NOT_FOUND` 中止。
- Iceshrimp Vue fork 的本地快照只有 `packages/`（`client`/`sw`/`backend`/`iceshrimp-sdk`），没有根清单，
  上游用 yarn 4.3.1 + PnP，不是 pnpm 工作区；即使补齐也不引入——它意味着同时引入 Firefish 后端契约、
  Service Worker 与插件运行时，与「Misskey API 是唯一后端兼容边界」冲突，因此只作组件镜像参考。
- Firefish 上游 `develop` 停在 `02c31e0`（2024-12-11），仓库**未归档**（Codeberg API `archived: false`，2026-11-01 复核）；
  准确表述是「上游无新提交」，不是「已归档/已删除」。

**通知端点的 canonical 与兼容 alias**：上游通知列表端点是 `POST /api/i/notifications`
（`endpoints/i/notifications.ts`），本实现注册在 `POST /api/notifications`（行为已实现，§11.15.3 记录为路径偏差）。
决定：

1. `POST /api/notifications` 是 **canonical**，现有前端 `api/client.ts` 与全部通知测试继续使用，不改路径；
2. 后端把 `POST /api/i/notifications` 注册为**兼容 alias**，两个路径进同一个 handler，不产生第二套实现；
3. 移植过来的 Firefish 代码调用 `i/notifications` 由后端 alias 直接承接，不在客户端做路径映射；页面与组件仍不得直接拼 RPC 路径；
4. 在 alias 落地前，Firefish 通知列表会 404——这是 Task 1 之后第一个必须补的后端小改动，与前端移植的第一个任务配对提交，
   不单独留在文档里。

**schema 版本**：当前为 **5**（不可变迁移 1–5，最后一次为 `user_profiles.pinned_note_ids`）。本轮不涉及 schema 变更，
前端移植不修改数据库（设计文档 §5）。§7 里原有的「当前 schema 版本为 4」已在本次一并订正。

**只读审计后的订正**（同一天的 Task 1 审计，本地记录 `.superpowers/sdd/2026-11-01-firefish-frontend-port/task-1-audit.md`，
该目录被 `.gitignore` 覆盖、不入库）：三份文档初稿的 12 类问题已修正，值得记住的六条：

1. 端点口径从「只匹配 `os.api(`」扩到 `api(`/`endpoint = "…"`/`fetch(apiUrl + …)`，去重后 **277** 个（初稿的 157 是窄口径）。
2. `POST /api/meta` **没有 `emojis` 字段**（初稿误写「返回空数组」）；emoji picker 退化为 Unicode 的结论不变。
3. `i/update` 只接受 `name`/`description`/`avatarId`（`i_update.go:66-70`），其余字段静默忽略 → 设置页只渲染三个输入框。
4. `readAllNotifications` **已发布**（`notifications.go:19`），`readNotification` **已处理**但语义是「该用户全部标读」
   （`connection.go:143-149`），初稿标成「缺」是错的。
5. 补登 5 个真实端点：`notes/unrenote`（取消转贴 → 映射 `notes/delete`）、`notes/edit`（Firefish 专属，隐藏）、
   `notes/replies`、`notes/mentions`、`drive/files/attached-notes`。
6. 运行时契约补 6 个首批页面会调用的成员：`pageWindow`（rewrite）、`select`/`yesno`（shim）、
   `form`/`selectUser`/`selectLocalUser`/`selectInstance`/`selectDriveFile`/`selectDriveFolder`（drop）、
   `openEmojiPicker`（rewrite）。

另有三处**既有研究**已被代码推翻，本次不改旧笔记原文（保审计链），以新文档为准：
`2026-10-30-firefish-ui-gap-analysis.md:786,880` 说通知 `includeTypes`「解析后丢弃」（实际已实现）、
`2026-10-31-iceshrimp-architecture-translation.md:16` 把 2012 个文件挂在 `client/src`（实际是 `packages/` 全量）、
`2026-10-29-frontend-redesign-reference-analysis.md:44` 的「Firefish 开发已中止」（更准确：`develop` 自 2024-12 无新提交，
仓库未 archive）。

**本轮门禁（2026-11-01 实测）**：`git diff --check` 无输出；`gofmt -l internal/ cmd/` 无输出；`go vet ./...` 通过；
`go test ./...` 全绿；再带 `CINDERFISH_TEST_DATABASE_URL` 重跑，`internal/account`（0.77s）、
`internal/httpapi`（9.84s）、`internal/media`（0.81s）、`internal/note`（0.73s）同样全绿。纯文档改动不影响二进制，
但按 `AGENTS.md` 的要求仍跑一遍，避免把上一轮的红色状态当成绿。

**第二轮一致性修复（同日）**：删除能力矩阵 §1 状态表的尾随空格；统一通知端点为 `/api/notifications` canonical、
`/api/i/notifications` 兼容 alias，并同步 AGENTS、计划与能力矩阵；把运行时契约里过期的 `347`/`2400` 改为
`112+39+52=203`/`1852`；把能力矩阵的 43 个白名单端点与 25/26 个全量同名端点分开口径，隐藏设置页固定为 27；
Task 2 的文件布局改为运行时契约 §5 的 11 个 shim 文件（`os.ts` 委托 `client.ts` 的 `request()`，
token 只由 `client.ts` 读写）；Iceshrimp Vue 快照范围改为完整的 `packages/`，复现命令固定到 `1fa829ed10f4` 并只解 `packages/`。
修复后 `git show --check HEAD` 与 `git diff --check` 均无输出。

**执行顺序建议**（下一步）：先按运行时契约落地 shim（`api`、`me`、`i18n`、`instance`、`z-index`、`DialogHost`、
`PopupMenu`），再移植壳与时间线；端点缺口按能力矩阵 §10 单独立项，不与前端提交混在一起。
（Task 2–3 已完成这条顺序的前半段，见 §11.17。）

### 11.17 Firefish 前端整体移植 Task 2–3：运行时 shim、壳与 Deck（2026-11-01）

**交付**：`web/src` 的 Firefish 风格壳、导航与 Deck，以及首批页面在运行期需要的剩余 `os.*` 成员。
代码清单、越界文件与理由、逐项证据见 `docs/superpowers/reports/task-3-report.md`。

1. **Deck 偏好 ≠ 布局**（修真实缺陷）：三个可成列的路由在 `router/index.ts` 标 `meta.deck`，
   `stores/shell.ts` 的 `shellLayout(preferred, routeOwnsDeck)` 决定实际布局。此前 Deck 偏好下打开
   `/notes/:id`、`/@handle`、`/settings/*` 会被 `height:100dvh; overflow:hidden` 裁掉且无法滚动。
2. **Escape 所有权收敛**：新增 `lib/layer-stack.ts`（无 Vue 依赖、可单测），`OsOverlay` 在
   document capture 注册唯一监听，命中即 `stopImmediatePropagation` 并只关栈顶。对话框、菜单、
   发帖框、表情选择器、通用 `popup`、Deck 页面窗口全部入栈；NavDrawer 的 capture 监听与
   WidgetsDrawer/NoteFooter 的 bubble 监听不再能同一次按键关掉第二层。
3. **运行时契约 §3.5/§3.15 补完**：`openLayer` / `closeLayer` / `popup`（`markRaw`、`MaybeRef` props、
   `events`、`disposeEvent`、延后 `dispose`）、`pageWindow` / `modalPageWindow`、`openEmojiPicker`
   （Unicode 目录；在给定 textarea 的光标处插入，不移植 Firefish 的全局 focus 监听）。
4. **降级写清楚**：`pageWindow` 对 `note`/`profile` 走已有的 `DeckPageWindow`（URL 不变），其余已知
   路由导航主表面（这些页面用 `useRoute()`，窗口内渲染需要第二个路由实例，而“不引入第二路由”是硬约束）；
   未知名路径（`not-found`）显式拒绝；`modalPageWindow` 与 `pageWindow` 同实现——本项目的 `PageWindow`
   固定是模态，Firefish“可拖动 vs 模态”的差别不提供。
5. **壳与密度**：三列齐边 + 1px 分隔线，外壳 cap 1280 → 1360，侧栏/小组件列全高吸附；`PageHeader` 的
   `-24px` bleed 与主列 24px 内边距（≤500px 时 12px）成对，改一处必须改另一处。导航项补齐
   `:focus-visible`。全局等待条消费 `pendingApiRequestsCount`（承接 Task 2 审计 m-1）。
6. **共享滚动锁**：`lib/scroll-lock.ts` 引用计数，NavDrawer 与 WidgetsDrawer 不再互相解锁/互相锁死。
7. **Deck 自适应**：列菜单与“添加列”迁到 `os.popupMenu`（删掉两套本地 Escape/pointerdown 与列内
   overflow 裁切）；单列断点 1099 → 839，840/600/380 分别落在“多列/单列/单列 + 底栏”。移动底栏在登录后
   多一个“我的”键（registry 的 `to` 是静态串，账户地址取不到，故按会话生成）。

**门禁（本机实测）**：`cd web && pnpm typecheck` 退出 0；`pnpm build` 退出 0；`git diff --check` 无输出；
`tmp/task3-check.mjs`（`tmp/` 不入库）跑层栈 LIFO、`shellLayout`、`deckPageFor` 共 5 项全过。
未安装前端测试框架。

**浏览器验收（真实实例 + 临时 harness，2026-11-01）**：380×720、600×800、840×900、1280×720、1440×900
五个视口 × 6 条路由 × 通用/Deck 两种布局，`document.documentElement.scrollWidth - clientWidth` 全部为 0；
Deck 偏好 + 深链实测 `data-shell="universal"` 且可滚动；Deck 列宽 380 → 364/364（一列）、840 → 350/350、
1440 → 350/350；两端抽屉的滚动锁互不破坏；`os.*` 六个成员与 `popup`/Escape/窗口/emoji 插入全部通过。
**未覆盖**：登录态下的底栏“我的”、通知列内容（本次会话未登录）；`pageWindow` 对设置/管理路由的降级只做了
代码路径核对。完整场景表见 `.superpowers/sdd/2026-11-01-firefish-frontend-port/task-3-browser-prep.md` §4。

### 11.18 Task 4 收口与前端重定基：Firefish 原版复原（2026-11-01）

**Task 4 收口**（`ab897e6` … `422a9a8`）：

1. **per-note 流式更新**：`noteUpdated`（reacted/unreacted/deleted）分发到 `PagedFeed`，
   反应计数实时增减、删除实时移行；真实实例浏览器验收通过（点反应 +1/再点归零、删除行消失）。
2. **`useChannel`/`send` 适配器**：Firefish 形状频道句柄，仅限后端注册的 5 个频道（`4e93392`）。
3. **`defaultReaction: "⭐"`** 实例投影（`de2d547`，审计 D2-b）；**`withRenotes` 移除**（`8fd5546`，D1-b）。
4. **登录错误 id 映射中文文案 + admin 模式注册 503 文案**（`cd760a8`）；通知页无 follow tab、
   `markAsRead:false` 只走显式已读按钮（`41c85ba`，D2-d）；`users/notes` 回复默认值语义注释（`422a9a8`，D1-e）。
5. **审计 §8 六项裁决全部落地**（见 `docs/research/2026-11-01-firefish-rebaseline-decision.md`）。

**重定基**：Iceshrimp 参考弃用，Firefish 上游 `tmp/refcheck/firefish@02c31e0` 为唯一前端基准；
jclaudan/firefish GitHub 镜像经核对为 v1.0.3 时期（2023-10）旧快照，落后上游约 14 个月（196 个组件
文件差异、主题目录一致），**仅作备份不切换基准**（`5e763be`）。参考目录：`firefish`（基准）、
`firefish-jclaudan-mirror`（备份）、`misskey`（API 契约）、`iceshrimp-deprecated`（存档）。

**已复原切片**：主题色板（`2c61c51`，暗 bg `#000`/panel `#0d0d0d`/accent `#86b300`/fg `#dadada`，
亮 bg `#fff`/fg `#5f5f5f`，全部取自 `_dark.json5`/`_light.json5`）；导航项形态（`438cec0`，
`ui/_common_/navbar.vue` 的 accentedBg 药丸 + 2.85rem 行高）；笔记密度（`0ca52d1`，`MkNote.vue`
article 内边距 + 1.05em）；操作行（`459d5e7`，`MkNoteFooter.vue` 0.4em 上距 + 3.5em 封顶均分）。
浏览器走查（851×897，端口 3991）确认配色与密度已对齐；**待续**：PageHeader tab 样式、
composer 面板化、widgets 栏 Firefish 化。

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

## 12A. Firefish 前端整体移植（2026-11-01 起）

方向见 `docs/superpowers/specs/2026-11-01-firefish-frontend-port-design.md`：前端不再是"借鉴 Firefish 风格的自研页面"，
而是把 Firefish 客户端的组件体系（`firefish@02c31e0:packages/client/src`）整套搬进 `web/src`，改写为 Vue 3 + TypeScript +
不额外引入依赖的形态，接口一律经 `web/src/api/client.ts`。

### 12A.1 已落地的移植清单（全部带 `SPDX AGPL-3.0-only` 与 `Ported from firefish@02c31e0:<path>` 头）

| 层 | 文件 | 来源 |
|----|------|------|
| 设计 token / 全局样式 | `web/src/styles/main.css` | `style.scss` + `themes/_dark.json5`、`_light.json5`（`:lighten/:alpha` 编译函数改写为静态值或 `color-mix()`） |
| UI 原语 | `components/ui/`：MkButton、MkAvatar、MkUserName、MkA、MkTime、MkLoading、MkError、MkEllipsis、MkInfo、MkKeyValue、MkFolder、MkPagination、MkDateSeparatedList、MkInput、PopupMenu、DialogHost、PageHeader | `components/`、`components/global/`、`components/form/input.vue` |
| 运行时 | `store/pizzax.ts`、`store/firefish-store.ts`、`lib/os.ts`、`i18n/` | `pizzax.ts`、`store.ts`、`os.ts`、`i18n.ts` |
| 笔记集群 | `components/note/`：Note、NoteSimple、NoteSub、NoteDetailed、NoteHeader、NoteHeaderInfo、NoteFooterInfo、NoteContent、NoteMediaList、NoteFooter、RenoteButton、RenoteBar、ReactionsViewer、CwButton、Visibility、InstanceTicker、get-note-menu、use-note-capture | `MkNote*.vue`、`note/MkNote*.vue`、`MkCwButton`、`MkVisibility`、`MkInstanceTicker`、`scripts/get-note-menu.ts` |
| 投稿表单 | `components/composer/PostForm.vue` | `MkPostForm.vue`（投票隐藏：无 polls 端点，审计 D2-g） |
| 布局 | `layouts/AppShell.vue`、`components/layout/Navbar.vue`、`components/widgets/WidgetHost.vue` | `ui/universal.vue`、`ui/_common_/navbar.vue`、`ui/universal.widgets.vue` |
| 页面 | `views/`：Timeline、Notifications、Profile（`views/profile/*`）、Explore、Settings（`views/settings/*` 8 个标签页）、Login、Signup、Note、NotFound | `pages/timeline.vue`、`notifications.vue`、`user/*`、`explore*`、`settings/*`、`auth.form.vue`、`note.vue`、`not-found.vue` |

### 12A.2 与 Firefish 的关键差异（均为后端能力缺口）

- `notes/replies`、`notes/renotes`、`users/following`、`users/followers` 已补齐（见 §12B），详情页回复/转推/引用页签与个人页关注/粉丝列表走真实端点；仍缺 `notes/conversation`、`notes/children`、`notes/reactions`，对话树与「谁反应了」浮层保持降级。
- 无 polls 端点：投票只读、投稿表单隐藏投票入口。
- 无 `i/registry`：pizzax 落到 localStorage 的 per-account 桶，不做云端同步。
- 无自定义 emoji 注册表：`:shortcode:` 按纯文本渲染（与 Misskey 对未注册 emoji 的行为一致）。
- 无 `hashtags/trending`、`notes/featured`、`users/`（用户目录）、`users/following|followers` 列表：探索页退化为 `admin/show-users`（非管理员为空）、趋势小组件按本地时间线客户端统计、关注/粉丝只有计数。
- 图标：没有 Phosphor 字体，沿用项目自带 `AppIcon` 的路径图标集，尺寸与位置按 Firefish 的 `ph-*` 规则对齐；`ph-*` 类本身不移植。

### 12A.3 验证

- 每次提交前 `cd web && pnpm typecheck && pnpm build` 全绿；`go vet ./...`、`go test ./...` 未受影响。
- 内建浏览器走查（Go 二进制 @3991，`web/dist`，开发库 cinderfish）：1280×800 与 iPhone 14 Pro（393×852）两档下确认
  左导航 250px/图标态、中央 750px 面板列、右 widgets 栏、移动端底部 tab、笔记行 hairline 分隔、转贴条
  「@user 转发了」、日期分隔条、个人资料页横幅+头像+三列计数、设置页菜单列+分区面板、通知/笔记空态。
- 投稿窗口对齐 Firefish `MkPostForm`：头部只剩头像 + 剩余字数 + 仅本地标记 + 可见性图标 + 渐变发布键，
  工具行是 附件/内容警告/仅本地/更多；可见性改由锚定弹出菜单选择（不再是表单内的分段按钮），
  页面窗口按内容高度、520px 宽、`--panelHeaderBg` 标题栏。实测发布一条笔记成功（已清理）。

## 12B. 四个读端点（2026-11-02）

前端页签与关注列表所需的四个 Misskey 兼容读端点，全部按 `tmp/refcheck/misskey` 的 `paramDef`/`res` 落地：

| 路由 | 语义 | 备注 |
|------|------|------|
| `POST /api/notes/replies` | `reply_id` 下的回复，按可见性过滤 | 目标不存在时返回 `200 []`（上游没有 `NO_SUCH_NOTE`） |
| `POST /api/notes/renotes` | `renote_id` 下的转贴与引用 | 目标不存在时 404 `NO_SUCH_NOTE` |
| `POST /api/users/followers` | 关注者列表，返回 `Following` 信封（`follower`/`followee` 为 UserLite） | 受 `followers_visibility` 约束，管理员可绕过 |
| `POST /api/users/following` | 关注列表，同上 | 受 `following_visibility` 约束 |

- 可见性过滤复用从时间线抽出的 `queryBuilder.addNoteVisibility`（与 `notes/show`、四条时间线同一套规则），避免两处口径漂移。
- 新存储方法：`note.TimelineService.Replies/Renotes`、`account.Followers/Following/FollowVisibility/IsFollowing`；游标统一在 `id` 上，`sinceId` 单独出现时正序翻页。
- 前端接线：`NoteDetailed` 的 回复/转发/引用 页签走 `MkPagination`；`/@user/following`、`/@user/followers` 两个新路由由 `views/profile/UserFollowList.vue` 渲染，403 显示 `MkInfo` 说明。
- 测试：`TestNotesRepliesVisibility`（5 例）、`TestNotesRenotesVisibility`（5 例）、`TestUsersFollowListsPaginate`、`TestUsersFollowListPrivacy` 等；`go test ./...` 全绿。
- 仍未做：`users/update` 不能写 `followersVisibility`/`followingVisibility`，这两项目前只能直接改库。

## 12C. 工作面收敛到后端 API（2026-11-04）

- 前端移植移交独立 workflow；本仓库此后只做后端 API 与联邦通讯，前端目录保持冻结（不再提交界面改动）。
- 新增台账 `docs/research/2026-11-04-misskey-api-coverage.md`：按 Misskey `c05e844` 的 440 个端点逐一比对，
  当前已实现 35、未实现 405，并给出 Tier 1 的顺序（关注请求、隐私字段、收藏、会话串、反应名单、趋势、
  云盘文件夹、自定义表情、投票、注册表）。
- 联邦侧盘点：`internal/federation/inbox.go` 已处理入站 `Follow`/`Accept`/`Reject`/`Undo`/`Create`/
  `Delete`/`Like`/`Announce`/`Update`/`Block`；未覆盖 `Move`、`Flag`、`Question`、`EmojiReact`。
- 前端冻结时的状态：工作树干净，最后提交 `2700334`；上一轮未提交的两处前端微调（MkPagination 的
  `transition` 去掉 `mode="out-in"`、通知行间距对齐）已按"非后端工作全废"的要求回退，未纳入提交。

## 12D. 批次：关注请求端点（2026-11-04）

按 Misskey `following/requests/*` 落地五个端点（契约取自 `tmp/refcheck/misskey/.../following/requests/`）：

| 路由 | 语义 | 备注 |
|------|------|------|
| `POST /api/following/requests/list` | 收件箱：`followee = 我` 的待处理请求 | 返回 `{id, follower, followee}`（无 `createdAt`，与 `FollowRequestEntityService` 一致） |
| `POST /api/following/requests/sent` | 发件箱：`follower = 我` 的请求 | 同上 |
| `POST /api/following/requests/accept` | 接受：请求转 followings，远程关注者回 `Accept` | 无请求时 `NO_FOLLOW_REQUEST` |
| `POST /api/following/requests/reject` | 拒绝：删请求，远程回 `Reject` | 参考实现无 not-found 错误，重复拒绝仍 200 |
| `POST /api/following/requests/cancel` | 取消自己发出的请求 | 缺失时 `FOLLOW_REQUEST_NOT_FOUND`，返回 followee |

- 存储层：`federation.FollowRequests`（id/created_at 双游标）、`AcceptFollowRequest`（单事务：删请求 → 插 following → 刷新双方计数）、`RemoveFollowRequest`；计数刷新抽成 `refreshFollowCounts`，`Follow`/`Unfollow` 一并复用。
- 顺带修复：`account.Service.GetByID` 与 `account.listFollows` 对远程用户 `password_hash IS NULL` 的扫描失败（导致 `users/followers|following` 对含远程用户的列表 500）。
- 测试：`TestFollowingRequestsListSeparatesIncomingAndOutgoing`、`TestFollowingRequestsPagination`、`TestFollowingRequestsAccept`、`TestFollowingRequestsAcceptAnswersRemoteFollower`、`TestFollowingRequestsReject`、`TestFollowingRequestsCancel`、`TestFollowingRequestsUnknownUser`、`TestFollowingRequestsNeedCredential`。
- 门禁：`gofmt -l` 空、`go vet ./...` 干净、`go test ./... -count=1`（含数据库）全绿。
- 出站待确认已补齐（同日第二次提交）：`sendRemoteFollow` 在 Follow 入队成功后写入 `follow_requests` 行（uri = 发出的 Follow id），
  因此 `requests/sent` 能看到「等待对方确认」的关注、`requests/cancel` 能撤回；入站 `Accept` 在记录 following 后清掉该行
  （行已不存在不算错误）；`following/create` 对「已有待确认请求」同样返回 `ALREADY_FOLLOWING`，避免重复向对端投递 Follow。
  测试：`TestFollowingCreateRemoteRecordsPendingRequest`（投递队列只有一条 Follow、sent 列表出现该请求、重复 create 400、cancel 后清空）；
  `deliverQueue` 助手改为同时解析字符串 object（Follow）与对象 object（Accept/Reject/Undo）。

## 13. 文档索引

| 文档 | 内容 |
|------|------|
| `AGENTS.md` | 项目规约（**改代码前必读**） |
| `docs/superpowers/specs/2026-10-26-cinderfish-architecture-v0.md` | 架构设计 v0 |
| `docs/superpowers/specs/2026-11-01-firefish-frontend-port-design.md` | **Firefish 前端整体移植设计（取代渐进式设计）** |
| `docs/research/2026-11-01-firefish-port-source-and-license-register.md` | **移植参考源码版本、许可证与复制边界** |
| `docs/research/2026-11-01-firefish-port-capability-matrix.md` | **页面 → 端点能力矩阵与降级行为** |
| `docs/research/2026-11-01-runtime-adaptation-contract.md` | **运行时 shim 契约（shim/rewrite/drop）** |
| `docs/research/2026-10-26-misskey-fediverse-firefish.md` | 首轮调研 |
| `docs/research/2026-11-04-misskey-api-coverage.md` | **Misskey API 覆盖台账（当前工作线的进度依据）** |
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

- **浏览器可视化走查（Task 8 波次）未做**：2026-10-31 的最终轮，内建浏览器返回 `BROWSER_USER_TAKEOVER`（用户接管面板），按工具契约当轮停止。因此 1440/1280/1100/840/720/600/380 的截图、`scrollWidth` 溢出实测与登录态视觉复核**都没有执行**；同一轮的后端 API 走查（未读数、通知类型、回复可见性、纯转贴反应、四条时间线）已用真实二进制完成，见 §11.15.7。本波次没有改 CSS 或 DOM 结构，风险低，但不能当作已验证。
- **S3 后端未做真实服务端验证**：只验证了编译、接口与 URL 生成逻辑。
- **缩略图/WebP 生成未实现**：`drive_files.thumbnail_key` 字段已留，图片处理管道还没写。
- **前端没有真实第三方客户端联调**：页面已在构建产物上用浏览器走查，但没有用 misskey-js 或 Misskey Web 前端连过本实例。
- **Redis 只对 miniredis 验证过**：本机没有 `redis-server`，跨进程中继的单元测试与双进程 smoke 都用 miniredis；没有对真实 Redis 做过持久化/认证/网络闪断验证。
- **Streaming 未做真实 Misskey 客户端联调**：已用自写 WebSocket 客户端验证（握手、`connect`、`connected`、笔记投递全部通过），但**没有**用 misskey-js 或 Misskey Web 前端实际连过。
- **联邦没有任何第三方联调**：只跑了本实现的双实例测试（真签名、真 HTTP、真库），没有对 Mastodon / GoToSocial / 真实 Misskey 实例互通。这是 M5 之后最该补的验证。
- **Streaming 多进程中继只手测了单连接的 LISTEN/NOTIFY 往返**，没有做真正的双进程联调。
