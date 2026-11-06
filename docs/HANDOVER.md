# Cinderfish 交接说明

- 最后更新：2026-11-07（目标模式第五十一至六十四波：账号迁移、KaTeX 宏页与集成说明页、
  OAuth 交互式同意页、`ap/show` 与远程互动页、第三方应用授权全流程（`app/*` 与 `auth/session/*`，
  schema v20）、关注关系设置与其消费方（schema v21）、入站反应修复与 `EmojiReact`、入站 `Flag`、
  范围内占位页清零（24 个剩余占位页全部在首版范围外）、hashtags 三端点、`users/relation`
  与频繁回复用户、真实二进制浏览器走查（修掉登录 token 缓存与本地 Actor URI 两个缺陷）。
  恢复目标模式时先补了 §12BO.3 的验证债，
  `go test ./...`（含数据库）已全绿，
  详见 §12BP–§12CB。第五十一至六十二波的桌面/移动浏览器走查已在第六十三波补完
  （真实二进制 + Helium 156），当时发现并修复了登录 token 缓存与本地 Actor URI 解析两个缺陷）
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
| **前端移植基线切到分支 `firefish-port`（worktree `.worktrees/firefish-port/`）** | ✅ | `e54d2d1` 重定基决策 · `9ce4a00` 删除旧前端 · `6910c41` 新骨架（§12P–§12Q） |
| **前端第一波：骨架重建**（themes/styles/api/store/router/ui/os/account/i18n + 21 个 `Cf*` 原子组件，123 个占位页） | ✅ | `6910c41`，记录见 §12Q |
| **前端第二波：Note 组件树 + 时间线/通知/Drive**（+48 个 `Mk*` 组件、MFM 渲染器、单 `defaultStore`、`check-port` 门禁、浏览器走查） | ✅ | `6007474`（文档 `a723987`），记录见 §12R |
| **前端第三波：发帖链路 + 个人页/关注列表 + 设置 + 内容页 + 导航壳/deck/widgets**（13 个并行作用域，浏览器实测发帖与设置保存落库） | ✅ | 见 §12S |

- Firefish 字典来源：`firefish@02c31e0a2804de48c9d73028df525e197d8df622:locales/zh-CN.yml`；第一波的 `web/src/i18n/zh-CN.ts` 已随旧前端删除（`9ce4a00`），现在是 `web/src/i18n/locales/{zh-CN,ja-JP}.json`（扁平键，由 `web/scripts/gen-locale.mjs` 从 Firefish locales 生成，见 §12Q）。
- **提交号的权威来源是 `git log`**：本文件按批次记录做了什么、证据是什么；M3 笔记、M4 Streaming、M5 联邦等后端批次的提交号以 `git log` 为准，前端两波移植的提交号已在 §2 里程碑与 §12Q/§12R 中列明。

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

前端单独开发（vite dev server 反代到 3000 端口，见 `web/vite.config.ts`）：

```bash
cd web
pnpm install
pnpm dev            # http://127.0.0.1:5173，/api /media /streaming 等反代到 127.0.0.1:3000
pnpm typecheck      # vue-tsc --noEmit
pnpm run check:port # 移植规约门禁（web/scripts/check-port.mjs），见 §12R.1
pnpm build          # 产出 web/dist，由 Go 二进制静态托管
```

`pnpm run check:port` 会拦下六类问题：引入新依赖、旧的 `i18n.ts.xxx` 调用、`$ref`/`$computed`、直接 `fetch`、页面自己读 token，
以及**模板里用到但未 import/未全局注册的组件**（第二波踩过：`pages/Note.vue` 漏 import `MkNoteDetailed`，`vue-tsc` 不报错、页面静默空白），
外加 `t()` 引用了 zh-CN 词典里不存在的键。

数据库迁移在启动时自动执行（不可变的版本化脚本）。当前 schema 版本为 **9**（`internal/db/db.go` 的 `migrations` 1–9）。

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
web/                        Vue 3 + TS + Vite 前端（Firefish 客户端整体移植，目录见 §8.1）
```

依赖方向单向：`httpapi → 领域包 → db/media(接口)`。`streaming` 只定义接口，实现由 `httpapi` 注入。

### 8.1 前端目录（第四波之后）

```
web/src/
  api/            typed RPC（client/endpoints/types/errors/pagination/stream）—— 页面唯一允许的出网方式
  store/          pizzax 三作用域 store；base.ts 是唯一 defaultStore（store.ts 仅为 re-export shim）
  composables/    usePagination / useNoteCapture / useTheme / useInterval / use-custom-css
  os/             弹层总线（alert/confirm/popupMenu/…）与 os.api 别名
  account/        会话层（me/$i、登录注册、MiAuth、路由守卫）
  themes/         主题引擎 + 28 个内置调色板（26 个 json5）+ 已安装主题（installed.ts，localStorage）
  styles/         全局 SCSS（reset/typography/utility/animations/scrollbar/mfm）
  ui/             四壳（universal/deck/zen/visitor）+ layouts
  router/         vue-router 表（Firefish 131 条路径全保留）
  components/     Mk* 移植组件 + global/ 15 个自动注册全局组件 + ui/ 21 个 Cf* 原子组件
                  + form/ 8 个设置表单包装 + user/ 6 个个人页子视图 + admin/ 管理台行组件
                  + MkPostForm/MkNavbar/MkWidgets 等
  widgets/        12 个首版小组件 + WidgetCard/WidgetConfigDialog/index.ts
  directives/     size/appear/tooltip/user-preview/hotkey
  filters/        user/note/bytes/number
  scripts/        前三波脚本（autocomplete/preprocess/emojilist/katex-macro/get-user-menu/langmap/…）
  pages/          133 个页面文件：64 个已实装、69 个占位（首版范围外的 chat/channels/clips/gallery/
                  pages/antennas 等与无端点的设置/管理页）
  i18n/           扁平键 t()（zh-CN / ja-JP），生成脚本 web/scripts/gen-locale.mjs
```

前端依赖只有 `vue`、`vue-router`、`mfm-js`（MFM 解析）与 `@phosphor-icons/web`（图标字体）；
Firefish 其余依赖（swiper/photoswipe/vue-plyr/katex/blurhash/chart.js/date-fns/tinycolor2 等）**故意不装**，
对应组件按 §12R.3 的降级表实现。

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

## 12E. 批次：users/update 与关注列表可见性（2026-11-04）

- 新增 `POST /api/users/update`，与既有的 `i/update` 指向同一 handler（Misskey 两者等价，`users/update` 是正名）。
- 请求体新增 `followersVisibility` / `followingVisibility`，取值 `public|followers|private`，非法值 400 `INVALID_PARAM`；
  未提供的字段保持不变（`nullableField` 的既有语义）。
- 存储：`account.Profile` 读取两列，新增 `Service.UpdateVisibility`；`packUserDetailed` 不再硬编码 `public`，
  从 profile 读取，`users/show`、`api/i`、`users/update` 响应都回显真实值。
- 测试：`TestUsersUpdateVisibility`（别名路由生效、两字段落库并在 users/show 与 api/i 回显、部分更新不动另一字段、非法值 400 且不改状态）。
- 门禁：`gofmt -l` 空、`go vet ./...` 干净、`go test ./... -count=1` 全绿。

## 12F. 批次：收藏（2026-11-04）

- 迁移 **v6** 新建 `note_favorites(id, user_id, note_id, created_at, UNIQUE(user_id,note_id))`，带 user/note 两个索引；
  前端（Firefish）的收藏页与笔记菜单依赖它。
- 端点：`POST /api/notes/favorites/create`（返回 `{id, createdAt, noteId, note}`）、`POST /api/notes/favorites/delete`（幂等）、
  `POST /api/i/favorites`（按 id/时间游标分页，最新在前，删除过的笔记跳过）。
- 创建对同一笔记重复调用返回同一行（`ON CONFLICT DO NOTHING` + 回读），与 Misskey 的幂等语义一致；目标笔记不存在答 404 `NO_SUCH_NOTE`。
- 未做：Misskey 对 create 有「每小时 20 次」限流，本实现还没有限流器，代码里留了 TODO。
- 测试：`TestNoteFavoritesRoundTrip`、`TestNoteFavoritesListPagination`、`TestNoteFavoritesErrors`。
- 门禁：`gofmt -l` 空、`go vet ./...` 干净、`go test ./... -count=1` 全绿。

## 12G. 批次：会话串与子帖（2026-11-04）

- `POST /api/notes/conversation`：从 `reply_id` 逐级向上取父帖，最近的在前，支持 `limit`（默认 10，上限 100）与 `offset`；
  遇到循环引用或不可见父帖即停止（Misskey 参考实现不做可见性检查，这里按本实例既有口径收紧，避免通过公开回复泄漏受限父帖）。
- `POST /api/notes/children`：某帖的直接回复，最新在前，支持 `limit/sinceId/untilId/sinceDate/untilDate`，
  可见性走与时间线相同的 `addNoteVisibility`（公共/首页、作者、被指定者、被提及者、关注者且是回复对象）。
- 测试：`TestNotesConversationWalksAncestors`（顺序、offset、limit）、`TestNotesConversationStopsAtPrivateParent`
  （匿名在 followers 父帖处截断、作者可见、未知笔记 404）、`TestNotesChildren`（顺序、作者/回复对象/局外人三种可见性、untilId 游标）。
- 门禁：`gofmt -l` 空、`go vet ./...` 干净、`go test ./... -count=1` 全绿。

## 12H. 批次：反应名单（2026-11-04）

- `POST /api/notes/reactions`：返回 `NoteReaction` 数组 `{id, createdAt, type, user: UserLite}`，最新在前；
  支持 `type` 过滤与 `limit/sinceId/untilId/sinceDate/untilDate` 游标；与 Misskey 一致**不要求凭证**。
- 可见性：浏览者读不到的笔记一律答 404 `NO_SUCH_NOTE`（错误 id 用该端点自己的 `263fff3d-…`），不泄漏反应名单；
  已删除用户的反应行跳过，不输出 `user: null`。
- 测试：`TestNotesReactionsList`（顺序、UserLite 形状、type 过滤、untilId 游标）、
  `TestNotesReactionsListVisibility`（局外人与匿名 404 且错误 id 正确、作者可见、未知笔记与缺参）。
- 门禁：`gofmt -l` 空、`go vet ./...` 干净、`go test ./... -count=1` 全绿。

## 12I. 批次：话题标签与趋势（2026-11-04）

- **建笔记时抽取标签**：`note.ExtractHashtags` 按 Misskey `extract-hashtags` 的规则（需要前置字符、尾随标点截断、
  大小写不敏感去重）解析正文，写入 `notes.tags`。此前该列恒为空，趋势与标签页都无从计算。
- `POST /api/hashtags/trend`：从本地笔记派生（`n.user_host IS NULL`，排除已删除），近 7 个 UTC 自然日内按
  **去重用户数**排名前 10；每项返回 `{tag, chart[7], usersCount}`，`usersCount = max(chart)`（与 Misskey 一致）。
  无需凭证。
- 说明：Misskey 用 Redis 维护每日标签统计，Cinderfish 用 SQL 从 `notes.tags` 现算；远端用户的笔记不参与排名，
  避免联邦流量带偏本地趋势。
- 测试：`TestExtractHashtags`（10 个用例：无标签、行首、多标签、大小写、标点截断、空标签体、URL fragment、日文标签）、
  `TestHashtagsTrend`（两用户同标签排在前面、单用户重复不重复计数、30 天前与已删除笔记不计入、7 个桶且今日桶正确、匿名 200）、
  `TestHashtagsTrendEmptyState`。
- 门禁：`gofmt -l` 空、`go vet ./...` 干净、`go test ./... -count=1` 全绿。

## 12J. 批次：云盘文件夹（2026-11-04）

- 迁移 **v7**：`drive_folders(id, user_id, name, parent_id, created_at)`＋`drive_files.folder_id`
  （外键 `ON DELETE SET NULL`，另有 user/parent、folder 两个索引）。
- 端点：`drive/folders`（按 folderId 列子文件夹，nil 为根）、`create`、`show`、`update`、`delete`。
  返回 Misskey `DriveFolder` 形状 `{id, createdAt, name, parentId, foldersCount, filesCount, parent}`。
- `drive/files/create` 与 `drive/files` 现在接受 `folderId`（上传用 multipart 字段，列表用 JSON 字段），
  并校验文件夹归属；打包的 DriveFile 的 `folderId` 不再是硬编码 `nil`。
- 语义：`update` 的 `parentId` 用“缺省 vs 显式 null”区分（null = 移到根）；移动到自己或后代返回 400；
  `delete` 在一个事务里把文件与子文件夹上移到父级再删行，与 Misskey 一致（不丢内容）。
- 测试：`TestDriveFoldersLifecycle`（create 嵌套、分层列表、show 计数、改名、显式 null 移到根、删除）、
  `TestDriveFoldersIsolationAndErrors`（他人不可见/不可删、自指与环拒绝、匿名 401）。
- 门禁：`gofmt -l` 空、`go vet ./...` 干净、`go test ./... -count=1` 全绿。

## 12K. 批次：自定义表情注册表（2026-11-04）

- 迁移 **v8**：`emojis(id, name, host, category, aliases, url, is_sensitive, local_only, created_at)`，
  本地名唯一（`host IS NULL` 部分唯一索引）、远端 `(name, host)` 唯一、`(category, name)` 索引。
- 端点：
  - `POST /api/emojis`（无凭证）：本地表情，按 category/name 排序，返回 `EmojiSimple` `{name, category, aliases, url, localOnly, isSensitive}`。
  - `POST /api/admin/emoji/list`：全部表情（含远端占位），支持 `query`（名称或别名 ILIKE）、`limit`、`sinceId/untilId`。
  - `POST /api/admin/emoji/add`：管理员用自己云盘里的文件注册表情；名称/别名需匹配 `^[a-zA-Z0-9_]+$`，重名答 `EMOJI_ALREADY_EXISTS`。
  - `POST /api/admin/emoji/delete`：按 id 删除。
- 顺带修复：`requireAdmin` 原来只用请求头取凭证（`authenticate(..., {})`），凡是把 token 放在请求体 `i` 字段的管理端点都会 401；
  现在与普通 handler 一致，先读头再读体。
- 测试：`TestEmojisRegistryLifecycle`（上传文件→注册→公共列表 EmojiSimple 形状与别名→管理列表→重名 400→非法名 400→删除后列表清空）、
  `TestEmojisAdminOnly`（非管理员 403、匿名可读公共列表）。
- 门禁：`gofmt -l` 空、`go vet ./...` 干净、`go test ./... -count=1` 全绿。

## 12L. 批次：投票（2026-11-04）

- `POST /api/notes/polls/vote`（需登录）：`{noteId, choice}`。笔记不存在/不可见 → 404 `NO_SUCH_NOTE`（用该端点自己的 `ecafbd2e-…`）；
  无投票 → `NO_POLL`；已过期 → `ALREADY_EXPIRED`；单选重复投票、或复选里重复同一选项 → `ALREADY_VOTED`；越界选项 400。
- 服务层：`Vote` 增加 `ErrNoPoll`/`ErrAlreadyVoted`/`ErrChoiceOutOfRange` 分类（原先只有 `ErrPollExpired` 和通用错误），
  并新增 `votedChoices`/`MyVotes`；打包时多选投票的 `isVoted` 现在按“我投过的全部选项”标记，而不是只看第一条。
- 投票创建此前已支持（`notes/create` 的 `poll` 参数），本批只补投票动作。
- 测试：`TestNotesPollsVote`——首次投票后 `votes`/`votesCount` 变化、单选二次 400 `ALREADY_VOTED`、
  多选两个不同选项成功且 `isVoted` 都为 true、重复同一选项 400、无投票笔记 `NO_POLL`、越界 400、
  过期 400 `ALREADY_EXPIRED`、匿名 401、未知笔记 404 且错误 id 正确。
- 门禁：`gofmt -l` 空、`go vet ./...` 干净、`go test ./... -count=1` 全绿。

## 12M. 批次：客户端注册表 i/registry（2026-11-04）

- 迁移 **v9**：`user_registry(user_id, domain, scope TEXT[], key, value JSONB, updated_at)`，
  主键 `(user_id, domain, scope, key)`；`domain` 以空串表示本地（PostgreSQL 主键里 NULL 不参与唯一，故不用 NULL）。
- 端点（全部需登录）：`i/registry/set`、`get`、`get-detail`、`keys`、`get-all`、`remove`、`clear`。
  取值缺失答 `NO_SUCH_KEY`（`ac3ed68a-…`）；`get` 原样返回存储的 JSON，`get-detail` 额外给 `updatedAt`。
- scope 归一化：排序 + 去重后作为键的一部分，所以客户端以不同顺序传同一批 scope 命中的是同一行。
- Misskey 的 `domain` 参数（跨实例注册表）保留在签名里但不做跨实例读写，只服务本地客户端配置同步。
- 测试：`TestRegistryRoundTrip`（set/get/get-detail/keys/get-all/覆盖写/remove/clear、scope 隔离与顺序无关）、
  `TestRegistryIsolationAndErrors`（跨用户不可见、`NO_SUCH_KEY` 错误 id、匿名 401）。
- 门禁：`gofmt -l` 空、`go vet ./...` 干净、`go test ./... -count=1` 全绿。

## 12N. 批次：API 对接契约文档（2026-11-04）

- 目的：为独立 workflow 里的新版前端提供**后端实测契约**（不是 Misskey 文档转述——逐个 handler 读码）。
- 产物：`docs/research/api-contract/` 8 份，覆盖全部已注册路由：
  `auth-users`（16 端点）、`notes-core`（14）、`timelines-notifications`（8 + Streaming）、
  `drive-media`（11）、`following-relations`（7）、`admin-emoji-registry`（18）、
  `meta-federation`（24，含 ActivityPub 全部路由）、`streaming`（协议、5 频道、17 条缺口）。
- 每份含：请求字段表、响应 JSON 形状（packer 对象各写一次全字段）、错误码与错误 id 表、
  `file:line` 引用、与 Misskey `c05e844` 的差异清单。绝大多数结论有对应测试作为行为证据。
- 前端对接最关键的差异（详文见各份差异汇总）：
  - id 是 **18 字符 base36**（非 ULID/UUID）；
  - token 读取顺序是 body `i` 优先于 `Authorization: Bearer`（与上游相反），Streaming 则相反；
  - 无 CORS，前端必须同源部署；
  - Streaming 空闲约 120–180s 会被断（pong 不刷新 liveness），客户端需自建保活+重连；
  - HTTP 时间线 `withRenotes` 无效、`users/notes` 的 `withReplies` 默认与上游相反；
  - `admin/accounts/create` 不返回 `token`；`i/update` 响应不含 pinnedNotes。
- 门禁不适用（仅新增文档，未动代码）；跨文档不一致（id 长度、`WWW-Authenticate` 细节）已订正。
- 已知残留：各文档所引 `server.go` 行号以当前提交为准，后续改动 `registerRoutes` 需同步。

## 12O. 批次：Firefish 完整参考源码就位（2026-11-04）

- 用户决定：前端移植以 **jclaudan/firefish 镜像**（GitHub）为工作参考，不是 sparse 的 Codeberg 上游检出。
- 主仓库 `tmp/refcheck/firefish-jclaudan-mirror` 已解除 sparse（`git sparse-checkout disable`）
  并 `git fetch --unshallow`（经代理 `127.0.0.1:19823`），现为完整 25784 提交、51 个 tag 的镜像，
  HEAD `b425f61`（2023-10-08 "add umbrel support"，即 Firefish 1.0.3 前后）。
- 该镜像 checkout 到 worktree `tmp/refcheck/firefish-full/`（40 MB，去 `.git`，`tmp/` 整体 gitignore）。
- **结构事实**：仓库根没有 `packages/client/index.html`，也没有 `scripts/copy-index.mjs`——
  index 由 `packages/backend/src/server/web` 的模板 + `gulpfile.js` 在构建时组装；
  Vite 入口是 `packages/client/src/init.ts`（非 `main.ts`），`base: "/assets/"`。
  前端移植时由 Cinderfish 自己的 `web/index.html` 承担模板角色即可。
- 完整包结构：`backend / client / firefish-js / megalodon / sw`，含 `locales/` 全量翻译、
  `packages/client/src/pages` 65 页、`src/ui` 五种布局（universal / deck / zen / visitor / _common_）。

## 12P. 批次：前端移植基线——技术栈替换、从零重建（2026-11-04）

- 用户裁决：**前端从头开始**——不是在现有 `web/src` 上渐进重排，而是完全的技术栈替换：
  以 Firefish 客户端源码（`tmp/refcheck/firefish-full/packages/client/src`，Vue 2 + Options API +
  nirax 自研路由 + pizzax）为蓝本，在 Cinderfish 现代栈（Vue 3 Composition API + `vue-router` +
  TypeScript + Vite + `pnpm`）上**逐模块翻译重建** `web/`。
- 保留的是 Firefish 的**结构、信息架构、视觉语言与交互**，不是它的依赖树（不需要 gulp、
  sass-rs 链、megalodon）。
- 分层约定（移植时的目录镜像）：
  - `web/src/api/` — RPC 封装，对接 `docs/research/api-contract/` 记录的真实后端契约（含
    body `i` 优先的认证、18 字符 base36 id、无 CORS 同源约束）；
  - `web/src/store|composables/` — pizzax 的 Composition API 等价物；
  - `web/src/router/` — nirax → vue-router 的路由表翻译（以 Firefish `router.ts` 的定义为准）；
  - `web/src/pages|components|ui|widgets/` — 逐组件翻译，文件名与组件层级对齐 Firefish 原名
    （`Mk*` → `Cf*` 前缀，避免与 Vue 生态组件撞名）；
  - `web/src/themes/` — Firefish 主题系统照搬（SCSS 变量 + 主题 JSON）；
  - i18n：先直连 `tmp/refcheck/firefish-full/locales/ja-JP.yml`、`zh-CN.yml`，构建期转 JSON。
- 旧 `web/src` 的组件不作为翻译来源，仅在后端契约不确定时交叉印证。
- Streaming / OAuth 的客户端行为以 `docs/research/api-contract/` 为准，其中与 Firefish 假设
  不符处（心跳保活、`withRenotes`、token 读取顺序）在 shim 层吸收，不污染页面组件。

## 12Q. 批次：前端骨架重建完成——第一波移植落地（2026-11-05）

- **旧前端已删除**（提交 `9ce4a00`）：原 `web/src`（164 文件、约 29741 行）整体移除，
  保留 `web/index.html`、`tsconfig.json`、`vite.config.ts`、`package.json` 作为种子。
- **新骨架已落地**（提交 `6910c41`，320 文件、21406 行）：8 个并行 agent 按 §12P 的分层约定
  翻译 Firefish 客户端，随后合并去重（删除 6 棵并行重复目录树
  `api-client`、`store-composables`、`os-dialog-ui`、`scaffold-theme`、`account-session`、
  `ui-primitives`、`layouts`、`router-entry`，以 `web/src/{api,store,composables,os,account,
  components/ui,themes,styles}` 为 canonical）。
- 落地清单（每层与 Firefish 原件的对应关系见各目录 `PORTING-NOTES.md`）：
  - `themes/`：主题引擎全量（值语言、色彩数学、light/dark token 表、Rosé Pine 默认主题、
    CSS 变量写入）；4 个主题 85 个变量与上游逐字节一致（用真实 tinycolor2 对照验证）。
  - `styles/`：全局样式架构 935 行，Firefish 工具类原名保留。
  - `api/`：类型化 RPC 层，对接 §12N 契约——body `i` 优先、错误信封、**全部 70 条路由的
    `endpoints.ts`**、游标分页、Streaming 客户端（30s 应用层心跳对抗 ~120s 空闲断连）。
  - `store/` + `composables/`：pizzax 三作用域 + BroadcastChannel 跨页签同步、
    defaultStore/deck/cold、useTheme（亮/暗/跟随系统）、usePagination、useNoteCapture。
  - `router/`：nirax → vue-router，Firefish **131 条路径**全保留，路由名冲突已加后缀；
    页面为占位组件。
  - `ui/`：universal/deck/zen/visitor 四 shell + `layouts/`，布局解析顺序与 Firefish 一致
    （`?zen` → 登出 visitor → `ui=deck` → universal）。
  - `os/`：popup 栈、z-index 分带（1e6/2e6/3e6）、alert/confirm/prompt/toast/menu。
  - `account/`：`$i` 会话层、登录/注册/miauth 表单、路由守卫（`meta.requiresLogin`）。
  - `i18n/`：zh-CN 1838 键 + ja-JP 1833 键（从 Firefish locales YAML 生成，生成脚本
    `web/scripts/gen-locale.mjs` 需 `yaml` devDependency——**故意未装**，JSON 已提交）。
  - `components/ui/`：21 个 `Cf*` 基础组件，类名/props 对齐 Mk* 原件。
- **验证证据**：`pnpm install` 通过；`vue-tsc --noEmit` exit 0（strict）；
  `vite build` 成功（主 chunk 284 KB / gzip 93 KB，276 个产物文件）。
  修复过的接缝错误：`Miauth`/`Oauth` 页面大小写、`PaginationState.reversed` 补齐、
  `Emitter` 泛型约束、占位组件模板中的尖括号（117 处）。
- **当前状态**：123 个页面是 `TODO: port` 占位；shell + 基础设施全部可用。
  下一波按 Firefish 页面逐个填充，建议顺序：`pages/Timeline.vue` + `components/note/` 组件树
  → `PostForm`（composer）→ `pages/UserProfile.vue` → `pages/Notifications.vue` →
  `pages/Drive.vue` → 其余按 Firefish 路由表推进。
- **对接注意**（续作 agent 必读）：
  - 页面组件一律通过 `web/src/api/client.ts` 的 `api()` 调用，不得自行 fetch；
  - Streaming 事件从 `web/src/api/stream.ts` 的 `useStream()` 取，配 `useNoteCapture`；
  - i18n 用 `web/src/i18n/index.ts` 的 `t()`（**扁平键**，`i.ts._ago.justNow` 这种嵌套取值
    不存在，用 `t("ago.justNow")`）；
  - 主题变量名与 Firefish 一致（`--panel`、`--accent`、`--divider`…），组件 SCSS 直接用；
  - 后端契约疑问查 `docs/research/api-contract/`，不要按 Misskey 文档假设
    （`withRenotes` 无效、`users/notes` 的 `withReplies` 默认值相反、`admin/accounts/create`
    不返回 token 等）。

## 12R. 批次：前端第二波——Note 组件树 + 时间线/通知/Drive 落地（2026-11-05）

- **工作方式**：Lead 先写施工契约 `docs/research/2026-11-05-firefish-port-wave2-contract.md`（模块映射、降级清单、写作用域、验收口径），再并行分派 7 个 agent（各持一个作用域）+ 1 个整合 agent，最后由 Lead 统一 typecheck/build/浏览器走查并提交。
- **单一 defaultStore 修复**：第一波留下两个 `Storage("base")` 实例（`store/base.ts` 与 `store/store.ts` 各自 new），它们共用同名 pizzax 键却互不可见。本波合并到 `store/base.ts`，`store/store.ts` 降为 re-export shim，`@/store`、`@/store/base`、`@/store/store` 已是同一对象。
- **新增依赖**（Firefish 同款）：`mfm-js@0.23.3`（MFM 解析）、`@phosphor-icons/web@2.0.3`（图标字体 bold + fill）。其余 Firefish 依赖（swiper/photoswipe/vue-plyr/katex/blurhash/vue-draggable-plus/chart.js/date-fns/tinycolor2 等）一律不装，按契约 §5 降级。

### 12R.1 落地清单

| 层 | 文件 | 说明 |
|----|------|------|
| 基础设施 | `web/src/config.ts`、`instance.ts`、`stream.ts`、`const.ts`、`components/index.ts`、`directives/{size,appear,tooltip,user-preview,hotkey}`、`scripts/{array,clone,copy-to-clipboard,please-login,reduced-motion,focus,use-tooltip,format-time-string,extract-mentions,scroll,sound,device-kind,page-metadata,upload,select-file}`、`filters/{user,note,bytes}` | `os.api`/`os.apiGet`/`os.apiForm` shim；`os/types.ts` 的 MenuItem 增加 danger/indicate/caption/target |
| 全局组件（15） | `components/global/`：MkA、MkAcct、MkAvatar、MkEmoji、MkUserName、MkEllipsis、MkTime、MkUrl、MkLoading、MkError、MkPageHeader、MkSpacer、MkStickyContainer、MkMisskeyFlavoredMarkdown（全局名 `Mfm`）+ `emoji-registry.ts` | 由 `import.meta.glob` 自动注册，模板直接用、不必 import |
| MFM | `MkMisskeyFlavoredMarkdown.vue` + `components/{MkLink,MkMention,MkCode,MkFormula,MkSparkle,MkGoogle}.vue` + `styles/mfm.scss` | 节点全覆盖：text/emoji/mention/hashtag/url/link/bold/small/italic/strike/inlineCode/blockCode/math/quote/search/fn（x2-x4、spin、jump、shake、twitch、blur、rainbow、sparkle、rotate、flip、scale、fg、bg、font）/plain；未知节点退化为原文不抛异常 |
| Note 树（11） | `MkNote`、`MkNoteSub`、`MkSubNoteContent`、`MkNoteHeader`、`MkNoteSimple`、`MkNotePreview`、`MkNoteDetailed`、`MkNotes`、`MkCwButton`、`MkPoll`、`MkPollEditor` + `scripts/get-note-menu`、`get-note-summary`、`check-word-mute`、`extract-mfm`、`reaction-picker` | 折叠/展开、CW、投票、反应 picker、子帖预览、线程递归、`useNoteCapture` 单帖流式更新 |
| 媒体与操作（19） | `MkMediaList`、`MkMedia`、`MkMediaBanner`、`MkMediaCaption`、`MkImgWithBlurhash`、`MkReactionsViewer`(+.reaction/.details)、`MkReactionIcon`、`MkReactionTooltip`、`MkStarButton(NoEmoji)`、`MkRenoteButton`、`MkQuoteButton`、`MkVisibility`、`MkInstanceTicker`、`MkAvatars`、`MkUrlPreview`、`MkFileTypeIcon` | 网格布局与 Firefish 类名一致；原生 `<video>`/`<audio>` |
| 时间线 | `MkTimeline`、`pages/Timeline.vue`、`pages/Note.vue`、`pages/IndexRoute.vue` | 四条时间线端点 + 四个 Streaming 频道；home 额外订阅 `main` 的 follow/unfollow 刷新 |
| 通知 | `MkNotification`、`MkNotifications`、`MkFollowButton`、`MkNotificationSettingWindow`、`pages/Notifications.vue`、`pages/FollowRequests.vue` | 后端只有 mention/reply/renote/quote/reaction 五种类型 |
| Drive | `MkDrive`、`MkDrive.file`、`MkDrive.folder`、`MkDrive.navFolder`、`MkDriveFileThumbnail`、`MkDriveWindow`、`MkDriveSelectDialog`、`pages/Drive.vue`、`pages/AttachedFiles.vue` + `scripts/upload`、`select-file` | `uploadFile(file,{folderId,name})` / `selectFiles({multiple,label})` 签名已冻结，下一波 PostForm 直接用 |
| 质量门禁 | `web/scripts/check-port.mjs`（`pnpm run check:port`） | 硬规则：禁止新依赖、禁止旧的 `i18n.ts.` 调用、禁止 `$ref/$computed`、禁止直接 fetch、禁止页面自读 token；另有 `unresolvedComponents`（模板里用到但未 import/未全局注册的组件，本波一次真实故障的来源）与 `missingKeys`（`t()` 的键不在 zh-CN 词典） |

### 12R.2 验证证据（Lead 统一执行）

```
cd web && pnpm exec vue-tsc --noEmit     # exit 0
cd web && pnpm run check:port            # exit 0（401 文件，全部规则零违规）
cd web && pnpm build                     # 成功：主 chunk 208 KB / gzip 70 KB，字体 woff2 入产物
```

浏览器走查方式：无头 Chromium 131 + CDP 脚本（登录态用 `localStorage.account` + `cinderfish.token` 注入），1440×900 与 390×844 两档；后端是真实 Go 二进制，数据库用测试账号 uiwave/uiwave2 灌入笔记、回复、引用、转贴、反应、CW、投票、长文、MFM 全节点样例。

| 页面 | 证据 |
|------|------|
| `/` 时间线 | home/local/social 三个 tab；首屏出帖；反应条、操作按钮、`$[x2]/$[x3]/$[fg]/$[bg]/$[blur]/$[flip]/$[spin]`、mention/hashtag/URL 卡片、引用块、MFM 代码块、`> 引用` 全部正确渲染；桌面 `scrollWidth == clientWidth == 1440`，移动端 390 无横向溢出 |
| `/notes/:id` | 正文、时间、反应、操作按钮、回复列表渲染正常 |
| `/my/notifications` | 5 种通知类型混排，文本长度 172 → 736 |
| `/my/drive`、`/my/follow-requests` | 正常空态/列表，无 JS 异常 |
| 全量 | `exceptionErrors`、`consoleErrors` 均为空；`document.fonts.check("16px 'Phosphor-Bold'")` = true |

走查中修掉的真实接缝错误（都已包含在本波提交内）：
1. 两个 `Storage("base")` 实例并行（第一波遗留）；
2. Phosphor 字体用字面量 import 时不重写 `@font-face` 的 url → 图标全是方框，改为在 `styles/index.scss` 里 `@import` 它的 CSS；
3. `pages/Note.vue` 渲染 `<MkNoteDetailed>` 却忘了 import（它不在 `components/global/`，不会被自动注册）→ 详情页空白；`vue-tsc` 不报错，已加进 `check-port.mjs` 的 `unresolvedComponents` 规则；
4. `os` 未导出 `api`/`apiGet`/`apiForm`，Firefish 的 `os.api(...)` 调用点无法编译；
5. `/static-assets/badges/info.png` 不存在导致构建失败，改用 `web/public/static-assets/badges/info.svg`；
6. 后端 `avatarUrl` 恒 null，头像满屏破图，`MkAvatar`/`CfAvatar` 加了无头像占位。

### 12R.3 已知降级与残留（续作必读）

- ~~**PostForm 尚未移植**~~：第三波已完成（§12S），回复/引用/时间线固定发帖框都走 `os.post`。
- ~~**CfTab 选项卡标签不显示**~~：第三波已改成 `{ value, label, count?, icon? }[]` 数组契约并修复（§12S.2）。
- **Streaming 共享订阅**：`useChannel("main")` 返回同一个订阅的包装，`dispose()` 会拆掉所有人的监听。现有组件一律只用 `channel.on()` 返回的 unsubscribe。
- **分页**：`usePagination/CfPagination` 只有一个 limit（Firefish 首屏 10、后续 30 两档），本波用单档。
- **媒体**：无缩略图/blurhash/PhotoSwipe/vue-plyr/媒体代理，图片直接用 `file.url`，视频音频用原生控件。
- **表情**：后端 `emojis`/`reactionEmojis` 恒 `[]`，自定义表情走 `api("emojis")` 注册表按名匹配，匹配不到渲染纯文本。
- **无端点的功能**：UrlPreview 卡片（只渲染纯链接样式）、推荐/列表/天线/频道时间线、翻译、编辑、Drive 文件改名/移动/URL 上传、通知折叠、用户预览卡、hotkey、声音。
- **声音**：`scripts/sound.ts` API 齐全但恒静音（`web/public/static-assets/sounds/` 无 mp3）。

### 12R.4 第三波建议顺序（已完成，记录见 §12S）

1. `MkPostForm` + `MkPostFormAttaches` + `MkPollEditor` 接线 + `MkVisibilityPicker` + `MkEmojiPicker`（发帖是个人实例的核心路径；`uploadFile`/`selectFiles` 已冻结可直接用）；
2. `pages/user/index*.vue`（个人页，`users/show` + `users/notes` + 关注按钮已在）；
3. `MkUserCardMini`/`MkUserList` → `pages/user/followers|following.vue`（端点已有）；
4. 设置页（`i/update` + `i/registry/*` 端点已在）；
5. 再按 Firefish 路由表推进剩余占位页（第三波后：133 个页面文件里 37 个已实装、96 个占位、共 131 条路由）。

## 12S. 批次：前端第三波——发帖链路、个人页、设置、内容页、壳层与 widgets（2026-11-05）

**工作方式**：Lead 先写施工契约 `docs/research/2026-11-05-firefish-port-wave3-contract.md`
（模块映射、强制降级表、13 个互不重叠的写作用域、跨作用域接口），再并行分派 13 个 agent；
Lead 负责合并接缝、修视觉与运行期缺陷、真实二进制走查、文档与提交。

### 12S.1 落地清单

| 层 | 文件 | 说明 |
|----|------|------|
| 发帖链路 | `components/MkPostForm.vue`、`MkPostFormAttaches.vue`、`MkVisibilityPicker.vue`、`MkEmojiPicker(.section/Dialog).vue`、`MkCheatSheetDialog.vue`、`scripts/{autocomplete,preprocess,katex-macro,emojilist}.ts`、`os/{post.ts,MkPostFormDialog.vue,open-emoji-picker.ts}` | `notes/create`（visibility/visibleUserIds/text/cw/localOnly/fileIds/replyId/renoteId/poll）、`drive/files/create`（`uploadFile`/`selectFiles`，签名沿用第二波）、`emojis`；`os.post(props)` 与 `os.openEmojiPicker(target, props, textarea)` 按契约实现 |
| 个人页 | `pages/{UserProfile,UserIndex,UserInfo}.vue`、`components/user/{UserHome,UserNotes,UserPhotos,UserActivity,UserReactions}.vue`、`components/{MkUserInfo,MkUserCardMini,MkUserList,MkUserOnlineIndicator,MkShowMoreButton,MkNumber}.vue`、`scripts/get-user-menu.ts`、`filters/number.ts` | 五页签（笔记/回复/文件/活动/回应）、置顶区、关注按钮与用户菜单；`users/show`、`users/notes`（显式 `withReplies`/`withFiles`）、`following/create|delete`；远程账号按契约显示说明，不伪造资料 |
| 关注列表 | `pages/{UserFollowing,UserFollowers}.vue`、`components/user/FollowList.vue` | `users/following|followers` 分页，403 用 `CfInfo` 说明可见性限制；行卡片用 `MkUserCardMini`（端点只给 `UserLite`） |
| 设置 | `pages/SettingsIndex.vue` + 12 个子页（Profile/Privacy/General/AccountInfo/Notifications/Reaction/Drive/Security/Other/Navbar/Sounds/Deck）、`components/MkSuperMenu.vue`、`components/form/*.vue`（8 个）、`scripts/langmap.ts` | 左菜单按容器宽度切竖排/胶囊；隐私页写 `users/update` 的两项 visibility；个人资料只做后端真实支持的 `name`/`description`/`avatarId`；其余偏好落 `defaultStore`，无端点的项明确标注「未开放」 |
| 内容页 | `pages/{Favorites,FollowRequestsSent,Explore,Tag,Search}.vue` | `i/favorites` + `notes/favorites/delete`、`following/requests/{sent,cancel}`、`hashtags/trend` + 四条时间线；搜索无端点，渲染「搜索接口尚未提供」与探索入口，不伪造结果 |
| 静态页 | `pages/{About,Announcements,MfmCheatSheet,Share,Preview,Scratchpad,ApiConsole}.vue` | 关于页读 `meta`；公告无端点走空态说明；MFM 速查展示已支持节点；API 控制台用 `api()` 真实调用 |
| 壳层 | `components/MkNavbar.vue`、`ui/layouts/{XSidebar,UniversalLayout,VisitorLayout,DeckLayout,DeckColumn}.vue`、`ui/{UniversalShell,DeckShell,VisitorShell}.vue`、`store/deck.ts` | Firefish navbar（头像菜单、`store.menu` 顺序、未读徽标、发帖按钮、图标态、移动抽屉）、移动底栏 + 发帖 FAB、访客壳登录弹窗；deck 列宽 350/420、列菜单、添加列、窄屏单列横滑、registry profile |
| widgets | `components/MkWidgets.vue`、`widgets/*`（12 个）、`widgets/index.ts`、`store/widgets.ts` | post-form / notifications / timeline / trends / memo / instance-cloud / activity / photos / user-list / clock / digital-clock / server-info；增删、上下移、配置弹窗；数据接已有端点 |

### 12S.2 本波修掉的真实缺陷

1. **`CfTab` 标签不显示（第二波遗留）**：`<option>` VNode 在 Vue 3 被编译器折叠。重写为
   `tabs: { value, label, count?, icon? }[]` 数组契约，删除 `CfTabItem.vue`；笔记详情的
   回复/转发/引用页签标签与计数恢复显示。
2. **个人页头部错位**：`.title` 原本在 `.banner-container` 外，`position:absolute; bottom:0`
   相对整张卡片定位，名字压住统计行、头像盖住简介。现在按 Firefish 结构把标题放回横幅内，
   另加一份移动端标题放在头像下方；简介去掉负 `top` 偏移。
3. **无头像占位是「+」**：`MkAvatar` 的 `noAvatar::before` 从文字加号改为
   `/static-assets/avatar.svg` 中性人形图标（导航栏 `CfAvatar` 原本就是 `ph-user`）。
4. **并行期懒加载收口**：`MkNavbar` / `UniversalLayout` 里用 `import.meta.glob` 探测
   `os/post.ts`、`MkWidgets.vue` 的写法改成直接 import；`VisitorShell.vue` 改为委派
   `VisitorLayout`，三个壳的入口方式一致。
5. **发帖提交按钮缺 `data-cy-open-post-form-submit`**：走查脚本点不到按钮才发现，已补。
6. **引用/转贴/子帖回复未接 composer**：`MkQuoteButton`/`MkRenoteButton` 原本跳
   `/share?renoteId=`（不会创建引用），`MkNoteSub` 的回复仍是提示框；现在三处都走
   `os.post({ renote })` / `os.post({ reply })`。
7. **删除 65 个未被路由引用的嵌套占位页**（`pages/settings|admin|messaging|my-lists|my-clips|my-groups|my-antennas`）。
   这些是第一波按 Firefish 原路径镜像出来的死文件，路由只使用顶层同名页面；页面文件从
   198 降到 133，剩余占位 96、已实装 37。

### 12S.3 验证证据

```
cd web && pnpm exec vue-tsc --noEmit    # exit 0
cd web && pnpm run check:port           # exit 0，390 文件，hardTotal 0
cd web && pnpm build                    # exit 0
gofmt -l internal/ cmd/                 # 空
go vet ./...                            # exit 0
CINDERFISH_TEST_DATABASE_URL=… CINDERFISH_TEST_REDIS_URL=… go test -count=1 ./...  # 全绿
```

浏览器走查：Helium/Chromium 156 无头 + CDP，真实 Go 二进制 `:3991`（schema v9、开发库、
`federation_mode=none`、`registration_mode=open`），种子账号 `w3alice`/`w3bob`（走查后已从开发库删除，
用户数回到 21）。桌面 1440×900、移动 390×844。

| 场景 | 证据 |
|------|------|
| 桌面 18 条路由 | `/`、`/@w3alice`、`/@w3alice/followers`、`/@w3alice/following`、`/notes/:id`、`/my/notifications`、`/my/drive`、`/my/favorites`、`/my/follow-requests/sent`、`/explore`、`/settings/{profile,privacy,general,reaction,navbar,deck}`、`/about`、`/mfm-cheat-sheet`：全部渲染，`scrollWidth == clientWidth`，无 JS 异常 |
| 发帖 | 点导航「帖子」打开 composer，输入后提交；`notes/timeline` 回读命中同一文本（`postedVerified: true`） |
| 移动发帖 | 390px 点 FAB 打开同一 composer，提交后回读命中（`postedVerified: true`） |
| 设置保存 | `/settings/profile` 改昵称保存后 `/api/i` 返回新名字（`saveVerified: true`） |
| 个人页 | 五页签与置顶区、注册时间、三项计数；他人资料显示「取消关注 / 关注了您」；桌面与移动头部均不重叠 |
| 笔记详情 | 回复/转发/引用页签标签与计数可见（CfTab 修复），回复列表正常 |
| deck | `localStorage.ui=deck` 后 `/` 渲染列式布局（主列 350px、列头、添加列入口、右侧 +/齿轮），无横向溢出 |
| 移动 | 390px 下底栏五项 + 发帖 FAB + 抽屉；时间线、个人页、通知、设置页均渲染 |
| 探索 | `hashtags/trend` 读到 `#wave3`、`#cinderfish` 与人数 |

### 12S.4 已知降级与残留（续作必读）

- **导航仍有未实装入口**：`聊天`、`频道` 与「更多」里的书签/列表/天线/剪辑/分组/画廊/页面等仍指向
  `TODO: port` 占位页；导航只按 Firefish 的 `store.menu` 渲染，未按后端能力裁剪。
- **设置只移植 12 页**：邮箱、整合、主题编辑、插件、导入导出、状态栏、自定义 CSS/KaTeX、webhook、
  应用、API、迁移、偏好备份、删除账号、账户、无障碍、实例静音/词过滤、关注请求设置等仍是占位。
- **无端点的页面已明确降级**：搜索（`notes/search`/`users/search` 不存在）、公告、活动/回应页签
  （无 `charts/user/notes`、`users/reactions`）。只显示说明，不伪造数据。
- **widgets 只做 12 个**：rss/rss-ticker、aiscript、job-queue、server-info 之外的 server-metric、
  online-users、calendar、federation、unix-clock、slideshow、button 等未移植；拖拽排序改为上/下移按钮。
- **deck**：列拖拽排序改为菜单上下移；widgets 列复用全局 `MkWidgets`，没有独立的列内列表编辑。
- **媒体**：仍无缩略图/blurhash/PhotoSwipe/媒体代理；图片直接用 `file.url`，视频音频用原生控件。
- **通知摘要**与 Firefish 一致，保留正文里的 MFM 原文（`get-note-summary` 只取 `text`/`cw`）。
- **未走查**：图片上传的浏览器交互（上传脚本此前由 API 验证）、投票提交、引用/转贴按钮点击后的
  实际发帖、关注按钮点击、设置页除 profile/navbar/deck 外的保存动作、通知的流式实时插入。
- `i/registry/*` 已被 deck 列与设置页使用；`notes/edit`、`i/change-password`、2FA、注销、改邮箱
  在后端不存在，页面按「未开放」显示。

## 12T. 批次：前端第四波——管理台、主题体系、注册表与本地设置（2026-11-06）

**工作方式**：Lead 先补两个后端缺口并提交（`714354a`：`i/registry/scopes`、`GET admin/federation/mode`），
写施工契约 `docs/research/2026-11-06-firefish-port-wave4-contract.md`，自己实现主题引擎核心；
再并行分派 7 个互不重叠的写作用域；最后 Lead 合并导航裁剪、修视觉缺陷、真实二进制走查并统一提交。

### 12T.1 落地清单

| 层 | 文件 | 说明 |
|----|------|------|
| 后端 | `internal/account/registry.go`、`internal/httpapi/registry.go`、`federation_admin.go`、`server.go`、两份测试 | `i/registry/scopes` 返回去重 scope；`admin/federation/mode` 支持 GET（读 `instance_meta` 运行期值，不读启动配置） |
| 主题引擎 | `web/src/themes/installed.ts`（新）、`builtin.ts`、`index.ts`、`composables/use-theme.ts` | 已安装主题存 localStorage（`cinderfish.installedThemes`），内置+已安装统一查找；light/dark 双槽 + 旧单值键迁移；`setThemeForMode()` |
| 调色板 | `web/src/themes/palettes.ts`（新）、`themes/builtin/*.json5`（26 个新文件） | 上游 26 个调色板原样复制，`?raw` + `parseJson5` 加载；id 用文件 slug，缺 author 补空串 |
| 主题页 | `SettingsTheme.vue`、`SettingsThemeInstall.vue`、`SettingsThemeManage.vue`、`ThemeEditor.vue` | 模式切换、双主题选择、JSON5 安装、导出/卸载、8 个 token 的编辑器与实时预览 |
| 管理台 | `AdminIndex.vue`、`AdminUsers.vue`、`AdminOverview.vue`、`components/admin/AdminUserRow.vue`（新） | 管理员守卫、offset 分页、冻结/解冻、创建账号、实例/联邦/最近用户概览 |
| 管理内容 | `AdminEmojis.vue`、`AdminFederation.vue` | 表情搜索/新增/删除；联邦模式切换与白名单增删 |
| 注册表 | `Registry.vue`、`RegistryKeys.vue`、`RegistryValue.vue` | scope 总览、键列表、JSON5 值编辑；路由 scope = `["system", ...path]` |
| 本地设置 | `SettingsCustomCss.vue`、`SettingsWordMute.vue`、`SettingsPreferencesBackups.vue`、`SettingsApi.vue`、`composables/use-custom-css.ts`（新）、`store/base.ts`、`main.ts` | 自定义 CSS 即时生效；软过滤词表；偏好 JSON 导出/导入/重置；原生 token 展示与复制 |
| 静态页 | `Welcome.vue`、`NotFound.vue`、`Empty.vue`、`ErrorPage.vue`、`NoGraze.vue`、`MyFavorites.vue`、`MyNotifications.vue`、`Follow.vue`、`FollowMe.vue`、`NoteHistory.vue` | 访客欢迎页复用账号层登录/注册弹窗；公开本地时间线；别名页直接渲染实现页；关注授权仅本地账号；笔记历史明示无端点 |
| 导航 | `MkNavbar.vue`、`SettingsIndex.vue` | 侧栏过滤聊天/频道/搜索；更多菜单移除列表/天线/剪辑/分组/页面/画廊，加注册表；设置菜单补主题/CSS/文字过滤/备份/API/注册表 |

### 12T.2 本波修掉的真实缺陷

1. **`CfInfo` 把内联内容拆成散列单词**：`.info` 是 `display: flex`，slot 里的文本节点和 `<code>`/链接
   各自成为 flex item，主题管理页的说明被排成竖列。改为 slot 包一层 `.content`（`flex: 1; min-width: 0`），
   图标与关闭按钮保持固定，`align-items` 改为 `flex-start`。
2. **导航指向未实装路由**：第三波遗留的聊天/频道/列表/天线/剪辑/分组/页面/画廊入口全部是 `TODO: port`；
   本波按覆盖台账 §5 从导航移除，设置菜单只列已实现页。
3. **主题安装/管理页无数据源**：第三波只有 4 个主题且没有安装存储；本波补齐 28 个可选主题、安装/
   导出/卸载与 light/dark 双槽，旧 `cinderfish.themeId` 自动迁移。
4. **管理台完全空白**：用户、表情、联邦、概览四页接真实端点；非管理员不发请求。

### 12T.3 验证证据

```
cd web && pnpm exec vue-tsc --noEmit      # exit 0
cd web && pnpm run check:port             # exit 0，394 文件，hardTotal 0
cd web && pnpm build                      # exit 0
gofmt -l internal/ cmd/                   # 空
go vet ./...                              # exit 0
CINDERFISH_TEST_DATABASE_URL=… go test -count=1 ./...   # 全绿
```

浏览器走查：无头 Helium 156（Chromium 156）+ CDP，真实 Go 二进制 `:3991`（开发库 schema v9、
`federation_mode=none`、`registration_mode=open`、本地媒体目录），种子账号 `w4admin`（CLI 创建，管理员）
与 `w4dana`；走查后账号、笔记、表情文件、白名单主机全部删除，用户数回到 23。桌面 1440×900、移动 390×844。

| 场景 | 证据 |
|------|------|
| 桌面 20 条路由 | `/`、`/admin`→`/admin/users`、`/admin/{overview,emojis,federation}`、`/settings/theme{,/manage,/install}`、`/theme-editor`、`/registry`、`/registry/keys/system`、`/settings/{custom-css,word-mute,preferences-backups,api}`、`/follow-me`、`/my/favorites`、`/my/notifications`、`/notes/:id/history`、404：全部渲染，`scrollWidth == clientWidth`，`exceptionThrown` 与 `console.error` 均为 0 |
| 移动 7 条路由 | `/`、`/admin/{users,emojis,federation}`、`/settings/theme`、`/registry`、404：无横向溢出、无 JS 异常 |
| 访客 | `/` 欢迎页（实例名/简介/注册与登录入口/实例能力）、`/public/local` 公开本地时间线、受保护页跳转 `/?redirect=…` |
| 导航 | 侧栏实际为「时间线/通知/关注请求/发现/书签/控制面板/更多/设置/帖子」，不再出现聊天、频道、搜索 |
| 主题 | 写入 `cinderfish.themeId.dark=d-nord` 后 `data-theme=dark`、body 背景 `rgb(46,52,64)`；页面内安装 W4 测试主题、管理页可见并可卸载；编辑器保存后 `cinderfish.themeId.light` 指向新 id |
| 自定义 CSS | 页面输入并保存后 `<style id="cinderfish-custom-css">` 出现，body outline 变为 `rgb(1,2,3)` |
| 文字过滤 | 切到关键词组、添加「走查过滤词」，列表计数 0→1、显示「已保存」 |
| 注册表 | UI 创建 `system/w4test = {hello:1}`，键页可见；值页改为 `{hello:2}` 后 `i/registry/get-detail` 回读 2 |
| 管理台 | 用户列表分页；对 `w4dana` 点「冻结」→「好」后行显示「已冻结/解除冻结」，再解冻恢复；创建账号表单渲染可用 |
| 联邦 | UI 切「仅白名单」→ `GET admin/federation/mode` = `allowlist`；添加 `w4-remote.example` 后 `allowlist` 回读包含；切回「关闭联邦」→ `none` |
| 其他 | `/settings/api` 显示当前原生 token 且可复制；偏好备份「导出」真实下载 `cinderfish-preferences-*.json` |

### 12T.4 已知降级与残留（续作必读）

- **管理台只实装 4 页**：用户、概览、表情、联邦。队列、文件、公告、数据库、安全、Relay、整合、
  实例屏蔽、代理账号、实验、对象存储、邮件、标签、举报、管理设置等仍无端点，路由保留占位且不再出现在导航。
- **管理用户页只有 offset 分页**：`admin/show-users` 没有排序/状态/来源筛选、搜索与总数；页面不画无效控件。
- **表情管理只能新增与删除**：`admin/emoji/update` 不存在，名称写错只能删后重加；没有批量/远程表情/打包导入。
- **主题**：已安装主题只存在本浏览器 localStorage，不同步到账号或其它设备；编辑器只编辑 8 个常用 token，
  其余继承基础主题；管理页「卸载」会清掉该主题在两种模式下的选中记录。
- **剩余设置页仍是占位**：邮箱、整合、安全（2FA）、导入导出、迁移、Webhook、插件、自定义 KaTeX、
  状态栏、静音/屏蔽、实例静音、删除账号、应用、账号切换等；菜单不列出。`mute/create`、`block/create`、
  `i/export`、`i/import`、`i/apps` 等后端端点不存在，因此没有做假控件。
- **本地设置的范围**：偏好备份只覆盖 `defaultStore`（不含 cold store / 声音 / 插件）；文字过滤是客户端
  软过滤，没有服务器侧词表；自定义 CSS 按账号存进 base store 并即时注入。
- **静态页**：远程账号不能从 `/authorize-follow` 关注（没有 `ap/show`），只提示到对方实例操作；
  `notes/history` 不存在，历史页只说明；`ErrorPage.vue` 目前未被路由使用。
- **未走查**：登录/注册表单的真实交互（仍用注入 localStorage 的方式进入）、emoji 新增的浏览器文件上传
  点击路径（新增由 API 验证、列表由 UI 验证）、通知的 Streaming 实时插入、偏好导入与重置、主题的
  26 个调色板只目视检查了 Nord；图片上传/投票/引用与转贴发帖仍沿用第三波状态。
- **未实装（首版范围外）**：chat/channels/lists/antennas/clips/groups/gallery/pages、剩余约 10 个 widget、
  其余 37 个 locale、实例级主题。

## 12U. 批次：前端第五波——静音/屏蔽、令牌管理、无障碍、多账号与 widgets 补齐（2026-11-06）

**工作方式**：本轮按用户要求「极速推进、只写代码、测试与走查留后」，所以没有并行分派，也没有浏览器
走查；Lead 一次写完后端与前端，只做编译级检查。施工依据见
`docs/research/2026-11-06-firefish-port-wave5-contract.md`。

### 12U.1 落地清单

| 层 | 文件 | 说明 |
|----|------|------|
| 关系存储 | `internal/account/relations.go`（新） | `Mute`/`Unmute`/`IsMuted`、`Block`/`Unblock`/`IsBlocked`、`Mutes`/`Blocks`（id 倒序 + sinceId/untilId） |
| 关系端点 | `internal/httpapi/relations.go`（新）、`server.go`、`account.go` | `mute/create|delete|list`、`blocking/create|delete|list`；`users/show` 补 `isMuted`/`isBlocked`；屏蔽事务内清双向关注与待审请求 |
| 时间线过滤 | `internal/note/timeline.go` | 排除被静音作者、双向屏蔽、被静音实例的主机 |
| 实例静音 | `internal/account/profile.go`、`internal/httpapi/i_update.go`、`users_pack.go` | `mutedInstances` 存 `user_profiles.settings`，`i` 回读，最多 200 条并做主机校验 |
| 令牌管理 | `internal/account/tokens.go`、`internal/httpapi/apps.go`（新）、`server.go` | `i/apps`（可排序）与 `i/revoke-token`（只能删自己的令牌） |
| 设置页 | `SettingsMuteBlock.vue`、`SettingsInstanceMute.vue`、`SettingsApps.vue`、`SettingsAccounts.vue`、`SettingsAccessibility.vue` | 静音/屏蔽名单；实例静音；应用授权与撤销；多账号切换/移除/新增；6 个有消费方的无障碍开关 |
| 接口层 | `web/src/api/endpoints.ts`、`types.ts` | 8 个新端点常量、参数/返回类型、`MeDetailed.mutedInstances: string[]` |
| widgets | `web/src/widgets/button.vue`、`calendar.vue`、`unix-clock.vue`、`slideshow.vue`、`index.ts` | 4 个新 widget，loader 表 12 → 16 |
| 导航 | `web/src/pages/SettingsIndex.vue` | 基础组加无障碍/静音与屏蔽/服务器静音；其它组加账号/应用 |

### 12U.2 明确的取舍

1. **不做状态栏设置页**：没有 `MkStatusbar` 组件与挂载点，做出来只是一个永不显示的列表。
2. **不做自定义 KaTeX 宏页**：`katex` 不是依赖，`MkFormula` 已按纯文本降级。
3. **不做 RSS / 服务器指标 / 在线用户 widget**：分别需要服务端代理与 Streaming `stats` 通道。
4. **无障碍只接有消费方的键**：模糊效果、图片加载、图标集等键没有任何组件读取，不画开关，页尾说明原因。
5. **button widget 不用 AiScript**：AiScript 不是依赖，改为打开 `url`，只允许 http(s)。
6. **slideshow 不用文件夹选择器**：取网盘最新 24 张图片轮播，`folderId` 不提供。

### 12U.3 验证证据

```
gofmt -l internal/ cmd/                 # 空
go vet ./...                            # exit 0
CINDERFISH_TEST_DATABASE_URL=… go test -count=1 ./...   # 全绿
cd web && pnpm exec vue-tsc --noEmit    # exit 0
cd web && pnpm run check:port           # 398 文件，hardTotal 0
cd web && pnpm build                    # exit 0
```

真实二进制 `:3991`（开发库、`federation_mode=none`、`registration_mode=open`）+ curl + 无头 Helium 156
（CDP，1440×900 / 390×844，独立 profile）：

| 场景 | 证据 |
|------|------|
| 静音 | dana 的帖子在 admin 本地时间线 1 → `mute/create` 后 0；`mute/list` = ["w5dana"]、`users/show.isMuted=true`；`mute/delete` 后恢复 1 |
| 屏蔽 | 先互相关注（2 行），`blocking/create` 后关注行 0；双方时间线互相看不到对方帖子；`blocking/list` 正确；解封后恢复可见；响应带 `isBlocked: true` |
| 实例静音 | `[" Remote.Example "]` 规范化为 `remote.example`；远程帖子在 admin 全局时间线 1 → 0，dana 与匿名仍为 1；`null` 清空后回到 1 |
| 令牌 | `miauth/gen-token` 签发的令牌可用；`i/apps` 列出 2 个应用（名称/权限/授权时间）；`i/revoke-token` 后旧令牌 `POST /api/i` 返回 401，原生 token 仍可用 |
| 错误码 | `MUTEE_IS_YOURSELF` / `BLOCKEE_IS_YOURSELF` / `NO_SUCH_USER` / `AUTHENTICATION_FAILED` 与 Misskey 的 id 一致 |
| 设置页 | `/settings/{mute-block,instance-mute,apps,accounts,accessibility}` 桌面与移动均渲染，`scrollWidth == clientWidth`，无 exception/console 错误；静音页切到屏蔽名单后仍显示 w5dana |
| 实例静音页交互 | 文本框预填 `remote.example`；输入两行并点「保存」后 `POST /api/i` 回读 `["remote.example","second.example"]` |
| 无障碍页交互 | 6 个开关；点击第一个后 `pizzax::base` = `{"animation":false}` |
| 应用页 | 两个应用卡片显示名称、描述、授权时间、最近使用「从未使用」、权限计数与撤销按钮 |
| 账号页 | w5admin 标「当前」，w5dana 显示「切换/移除」 |
| widgets | 首页右栏 4 个 widget：按钮、日历（今天/本月/今年 73.5/19.1/84.9%）、UNIX 时钟、幻灯片显示真实上传的 `w5-photo.png`；移动端按设计不显示右栏 |
| 设置菜单 | 基础设置组含无障碍/静音与屏蔽/服务器静音，其它组含账号/应用 |

走查数据已清理：删除 w5admin、w5dana、远程测试用户与全部测试笔记，`mutes`/`blocks` 归零，用户数回到 23；
`/tmp/w5-media`、`/tmp/w5-profile`、临时二进制与配置删除，服务停止。截图留在 `/tmp/w5-shots`（不入库）。

### 12U.4 仍未验证

- **UI 点击路径未覆盖**：应用页的「撤销」、账号页的「切换/移除/添加账号」只用 API 与列表渲染验证，
  没有真的点下去；widget 的配置弹窗没有打开过。
- **分页游标**：`mute/list` / `blocking/list` 的 `sinceId`/`untilId` 只按实现检查，没有构造多页数据实测。
- **实例静音的非法主机校验**：`/`、`@`、超长项由 `parseMutedInstances` 拒绝，但只测了合法输入与去重。
- **屏蔽的通知副作用**：Misskey 屏蔽后不投递对方通知；本实现未在通知层过滤（时间线与 `users/show` 已过滤）。
- **无需登录的注册/登录表单**、emoji 文件上传点击路径、Streaming 实时插入仍是前几波遗留的未验证项。

## 12V. 批次：前端第六波——搜索与标签（2026-11-06）

**工作方式**：目标模式下自主推进，一轮完成「代码 → 全量测试 → 真实二进制 → 无头浏览器 → 文档 → 提交」。
施工依据 `docs/research/2026-11-06-firefish-port-wave6-contract.md`。

### 12V.1 落地清单

| 层 | 文件 | 说明 |
|----|------|------|
| 笔记搜索 | `internal/note/search.go`（新） | `Search`、`SearchByTag`、`escapeLike`；时间线的静音/屏蔽/实例静音条件抽为 `addViewerRelationFilters` 共用 |
| 用户搜索 | `internal/account/search.go`（新） | 用户名/昵称/简介 ILIKE，排除冻结与双向屏蔽，支持 offset 与 id 游标 |
| 端点 | `internal/httpapi/search.go`（新）、`server.go` | `notes/search`、`notes/search-by-tag`、`users/search`；匿名可读、坏 token 视为匿名 |
| 搜索页 | `web/src/pages/Search.vue` | 输入框、帖子/用户页签、按 query remount 的分页；桌面与移动均可用 |
| 标签页 | `web/src/pages/Tag.vue` | 改走 `notes/search-by-tag`，不再只在本地时间线里客户端过滤 |
| 导航 | `web/src/components/MkNavbar.vue` | `search` 移出不可用集合，「更多」菜单恢复搜索入口 |
| 接口层 | `web/src/api/endpoints.ts`、`types.ts` | 三个端点与参数类型 |
| 测试 | `internal/httpapi/search_test.go`（新，5 个用例） | 可见性、LIKE 转义、静音过滤、标签大小写、用户搜索 origin 与屏蔽 |

### 12V.2 明确的取舍

1. 没有全文索引，走 Misskey 的 ILIKE 回退：只匹配正文，不匹配 CW、附件描述与投票选项。
2. `notes/search-by-tag` 只实现 `tag` 单标签形式；上游的 `query: string[][]` 布尔组合没有调用方。
3. `users/search` 额外接受 `sinceId`/`untilId`，让前端复用一个分页组件；offset 仍可用。
4. `users/search` 为每条结果填充 `notesCount` 等计数（每行两次聚合查询），避免卡片显示 0 帖子。
5. 标签匹配大小写不敏感：`notes.tags` 保留作者写法，用 `lower(t) = lower($1)` 而不是 jsonb 的 `?`。
6. 搜索页不保留上游 swiper：页签直接切换面板，键盘与窄屏更稳。

### 12V.3 验证证据

```
gofmt -l internal/ cmd/                 # 空
go vet ./...                            # exit 0
CINDERFISH_TEST_DATABASE_URL=… go test -count=1 ./...   # 全绿（新增 5 个搜索用例）
cd web && pnpm exec vue-tsc --noEmit    # exit 0
cd web && pnpm run check:port           # 398 文件，hardTotal 0
cd web && pnpm build                    # exit 0
```

真实二进制 `:3991`（开发库，`federation_mode=none`）：

| 场景 | 结果 |
|------|------|
| 匿名 `notes/search?query=searchable` | 2 条（alice 公开 + bob 公开） |
| alice 带 token 同一查询 | 3 条（多出她自己的 followers 帖） |
| `query=%`（LIKE 通配符） | 0 条，转义生效 |
| `notes/search-by-tag` `w6tag` / `W6Tag` | 均为 1 条 |
| `users/search?query=w6ali` | `["w6alice"]`，`notesCount=2`；`origin=remote` 为 0 |
| `origin=bogus` / 缺 `query` | 400 |
| 屏蔽后 `users/search`（带 bob token） | 0 条；匿名仍 1 条 |

无头 Helium 156（CDP，1440×900 / 390×844）：

| 页面 | 证据 |
|------|------|
| `/search?q=searchable` | 显示 alice 公开帖与 bob 帖，**不显示** alice 的 followers 帖；`scrollWidth == clientWidth`，无 JS 异常 |
| `/search?q=w6` 用户页签 | w6bob「1 帖子」、w6alice「2 帖子」，计数正确 |
| `/tags/w6tag` | 显示「被 1 人提到 · 最近 7 天」与带 `#W6Tag` 的帖子 |
| 侧栏/更多菜单 | 「搜索」作为顶级入口出现（此前被过滤） |
| 移动端 `/search?q=w6`、`/tags/w6tag` | 均渲染，无横向溢出、无异常 |

走查数据已清理：删除 w6alice、w6bob 及其全部笔记，用户数回到 23；临时二进制、媒体目录、profile 与
服务全部清理，截图留在 `/tmp/w6-shots`（不入库）。

### 12V.4 仍未验证

- `offset` 分页路径、`withFiles`、`host`、`userId` 过滤只写了实现，没有构造数据实测。
- 远端用户与远端笔记的搜索：实例当前 `federation_mode=none`，没有远程行可搜。
- 大结果集性能：`ILIKE '%…%'` 会全表扫描；个人实例可接受，但没有压测。
- 搜索页的用户页签在空结果时显示「无用户」，有结果时的卡片样式已目视，但没有测很长的简介换行。

## 12W. 批次：前端第七波——管理台用户筛选与文件盘点（2026-11-06）

**工作方式**：延续目标模式，一轮完成代码、测试、真实二进制、浏览器走查、文档与提交。施工依据
`docs/research/2026-11-06-firefish-port-wave7-contract.md`。

### 12W.1 落地清单

| 层 | 文件 | 说明 |
|----|------|------|
| 用户筛选 | `internal/account/admin_users.go`（新） | `AdminList`：sort/state/origin/username/hostname；`UsersByIDs` 批量取上传者 |
| 文件清单 | `internal/media/admin.go`（新） | `ListAll` 跨账号、按 userId/type/origin/hostname/游标过滤 |
| 端点 | `internal/httpapi/admin_drive.go`（新）、`admin_accounts.go`、`server.go` | `admin/drive/files`；`admin/show-users` 接新参数 |
| 用户页 | `web/src/pages/AdminUsers.vue` | 三个下拉 + 两个输入 + 应用/重置；说明改为真实行为 |
| 文件页 | `web/src/pages/AdminFiles.vue` | 从占位页改为真实清单：筛选、缩略图、大小、上传者、时间、加载更多 |
| 管理菜单 | `web/src/pages/AdminIndex.vue` | 新增「文件」 |
| 接口层 | `web/src/api/endpoints.ts`、`types.ts` | `AdminShowUsersParams` 扩参、`AdminDriveFile` |
| 测试 | `internal/httpapi/admin_drive_test.go`（新，2 个用例） | state/username/origin 筛选、文件来源/类型过滤、非法 type 400、非管理员 403 |

### 12W.2 有意的降级

1. `state=silenced` 返回空集（没有该字段），`alive` 退化为「未冻结」，`±updatedAt` 按 `created_at`。
2. `admin/show-users` 仍不返回总数，分页靠 offset 与「本页取满 30 条」判断。
3. 文件页只读：删除与批量清理没有端点，页面明确说明。
4. 默认排序用 `created_at ASC` 而不是 `id ASC`：引导管理员的 id 是手写的 `root1`，否则会排到 ULID 之后。
5. 上传者只带 UserLite（`user` 字段），不是完整 UserDetailed。

### 12W.3 验证证据

```
gofmt -l internal/ cmd/                 # 空
go vet ./...                            # exit 0
CINDERFISH_TEST_DATABASE_URL=… go test -count=1 ./...   # 全绿（新增 2 个管理台用例）
cd web && pnpm exec vue-tsc --noEmit    # exit 0
cd web && pnpm run check:port           # 398 文件，hardTotal 0
cd web && pnpm build                    # exit 0
```

真实二进制 `:3991`（开发库，`federation_mode=none`，管理员 w7admin）：

| 场景 | 结果 |
|------|------|
| `admin/show-users state=suspended` | 仅 `["w7dana"]` |
| `username=w7d` / `origin=remote` | `["w7dana"]` / 0 |
| `admin/drive/files origin=local` | 列出 w7-note.txt、w7-photo.png，`user.username=w7dana` |
| `type=image` / `type=text` | 分别命中图片 / 文本 |
| `type=image' OR 1=1` | 400，未进入 LIKE |
| 非管理员调用 | 403（测试里用未冻结账号验证） |

无头 Helium 156（桌面 1440×900、移动 390×844）：

| 页面 | 证据 |
|------|------|
| `/admin/users` | 排序/状态/来源/用户名前缀/实例主机五个控件存在；把状态设为「已冻结」并应用后，列表只剩 w7dana，w7admin 不在结果里 |
| `/admin/files` | 三行文件（w7-note.txt、w7-photo.png、遗留 sunset.png），上传者 @w7dana / @w4alice、类型、大小、时间正确；类型切到「图片」后文本文件消失 |
| `/admin` | 菜单出现「文件」，当前页高亮 |
| 移动端 `/admin/{files,users}` | 无横向溢出、无 JS 异常，筛选栏自动换行 |

走查数据已清理：删除 w7admin、w7dana 与其上传（行随账号级联，`/tmp/w7-media` 目录删除），用户数回到 23；
临时二进制、配置、profile 与服务全部清理，截图留在 `/tmp/w7-shots`（不入库）。

### 12W.4 仍未验证

- `sort` 的四种取值只验证了默认与 state，没有逐一对「关注者最多/最少」排序结果做断言。
- `hostname` 过滤、`userId` 直接过滤、`sinceId`/`untilId` 游标在文件列表上没有构造多页数据。
- `state=silenced` 的空集语义只按实现检查，UI 没有提供该选项（页面说明原因）。
- 文件缩略图只在 146 B 的纯白 PNG 上看过；缺失媒体文件（遗留 sunset.png）会触发一次 404，属预期。
- 远端文件（origin=remote）没有数据可看，实例当前不联邦。

## 12X. 批次：前端第八波——实例公告（2026-11-06）

**工作方式**：目标模式下的第八波，一轮完成迁移、端点、两页、测试、真实二进制与浏览器走查、提交。
施工依据 `docs/research/2026-11-06-firefish-port-wave8-contract.md`。

### 12X.1 落地清单

| 层 | 文件 | 说明 |
|----|------|------|
| 迁移 | `internal/db/db.go`（v10） | `announcements` 表：标题/正文/图片/有效期/启用位 + 两个索引 |
| 服务 | `internal/announcement/service.go`（新） | `Create` / `List(activeOnly)` / `Get` / `Delete` |
| 端点 | `internal/httpapi/announcements.go`（新）、`server.go` | 公开 `announcements`；管理端 create/list/delete |
| 公告页 | `web/src/pages/Announcements.vue` | 占位页改为真实列表：时间、图片、正文、有效期、加载更多 |
| 管理页 | `web/src/pages/AdminAnnouncements.vue` | 新建表单（标题/正文/图片/datetime-local 有效期）、列表、删除确认、过期标记 |
| 管理菜单 | `web/src/pages/AdminIndex.vue` | 内容组新增「公告」 |
| 接口层 | `web/src/api/endpoints.ts`、`types.ts` | 四个端点与 `Announcement` |
| 测试 | `internal/httpapi/announcements_test.go`（新，3 个用例） | 生命周期、校验与权限、过期隐藏 |

### 12X.2 有意的降级

1. 没有已读跟踪（`i/read-announcement` 未实现），`isRead` 恒为 false，公告页写明。
2. 没有首页横幅：横幅需要已读状态才能关闭，本波只做列表页。
3. 没有 update 端点：管理页的编辑工作流是「删除后重发」。
4. 正文按纯文本渲染并保留换行，不解析 MFM。

### 12X.3 验证证据

```
gofmt -l internal/ cmd/                 # 空
go vet ./...                            # exit 0
CINDERFISH_TEST_DATABASE_URL=… go test -count=1 ./...   # 全绿（新增 3 个公告用例）
cd web && pnpm exec vue-tsc --noEmit    # exit 0
cd web && pnpm run check:port           # 398 文件，hardTotal 0
cd web && pnpm build                    # exit 0
```

真实二进制 `:3991`：

| 场景 | 结果 |
|------|------|
| 匿名 `announcements`（无公告） | 0 |
| 管理员 API 创建 | 返回 id/title、`isRead=false`；公开列表变为 `["API 验证公告"]` |
| 匿名调用 create | 401；非管理员 token 在测试中为 403 |
| 管理页表单发布「界面发布公告」 | 列表立即出现两条，正文换行保留 |
| `/announcements` | 两条公告均显示标题、时间与正文；底部说明已读状态未实现 |
| 管理菜单 | 内容组出现「公告」 |

无头 Helium 156（1440×900 / 390×844）：管理页、公开页与移动端均无横向溢出、无 JS 异常；管理页的
`datetime-local` 有效期输入在移动端正常换行。截图在 `/tmp/w8-shots`（不入库）。

走查数据已清理：两条公告与 w8admin 删除（announcements 无用户外键，单独清理），用户数回到 23；
临时二进制、配置、媒体目录、profile 与服务停止。

### 12X.4 仍未验证

- 公告的 `imageUrl` 只在 API 层校验了协议，页面没有加载过真实图片。
- `untilId` 加载更多没有构造多页数据；`limit` 上限行为未实测。
- 过期公告的公开隐藏由测试的 400ms 有效期验证，没有在浏览器里等过期。
- 已读跟踪与首页横幅按设计未做。

## 12Y. 批次：前端第九波——账号数据导出与账号注销（2026-11-06）

**工作方式**：目标模式第九波，一轮完成端点、页面、测试、真实二进制与浏览器走查、文档与提交。
施工依据 `docs/research/2026-11-06-firefish-port-wave9-contract.md`。

### 12Y.1 落地清单

| 层 | 文件 | 说明 |
|----|------|------|
| 导出 | `internal/httpapi/account_data.go`（新） | `i/export`：资料 + 本人帖子 + 关注/粉丝 + 收藏 + 网盘元数据，附件下载 |
| 注销 | 同上 | `i/delete-account`：密码复核、最后管理员保护、先删网盘对象再硬删账号 |
| 路由 | `internal/httpapi/server.go` | 两条 POST |
| 下载 | `web/src/api/client.ts` | 新增 `apiDownload()`，错误仍走 `ApiError` |
| 导出页 | `web/src/pages/SettingsImportExport.vue` | 占位页改为导出卡片；导入明确未实现 |
| 注销页 | `web/src/pages/SettingsDeleteAccount.vue` | 密码 + 用户名确认，成功后 `signOut()` 回首页 |
| 设置菜单 | `web/src/pages/SettingsIndex.vue` | 其它设置组新增两项，说明同步 |
| 测试 | `internal/httpapi/account_data_test.go`、`account_delete_test.go`（各新） | 导出只含本人数据且不泄 token；注销错误/正确密码、token 失效；最后管理员 400 |

### 12Y.2 有意的取舍

1. 导出是同步 JSON（`cinderfish-account-export@1`），不是 Firefish 的异步 zip；不冒充 Misskey 格式。
2. 没有导入：缺少格式、冲突规则与限流，页面只放导出按钮并说明原因。
3. 注销是硬删除（Firefish 先标记后清理）；页面把「无法恢复」写在最前。
4. 最后一个未冻结管理员不可注销，避免实例失去管理入口。

### 12Y.3 验证证据

```
gofmt -l internal/ cmd/                 # 空
go vet ./...                            # exit 0
CINDERFISH_TEST_DATABASE_URL=… go test -count=1 ./...   # 全绿（新增 3 个用例）
cd web && pnpm exec vue-tsc --noEmit    # exit 0
cd web && pnpm run check:port           # 398 文件，hardTotal 0
cd web && pnpm build                    # exit 0
```

真实二进制 `:3991`（开发库，`federation_mode=none`）：

| 场景 | 结果 |
|------|------|
| `i/export`（w9dana，1 帖 1 关注 1 收藏） | 200，`Content-Disposition: attachment`；JSON 中 user=w9dana、notes=1、followings=1、favorites=1，无 `token` 字段 |
| 管理页导出按钮 | 真实下载 `cinderfish-export-w9dana-*.json`，内容与 API 一致 |
| `i/delete-account` 错误密码 | 403 `INVALID_PASSWORD`，账号仍在 |
| `i/delete-account` 正确密码 | 204；随后原 token 401、`users/show` 404 |
| 最后管理员保护（独立测试库只有 root） | 400 `LAST_ADMIN`，账号仍可用 |
| 注销页 | 密码 + 用户名填对后才能点；提交后回到访客首页 |

无头 Helium 156（1440×900 / 390×844）：导出页与注销页桌面/移动均无横向溢出、无 JS 异常；
注销发生在导出与移动端检查之后，避免会话中途失效。截图在 `/tmp/w9-shots`（不入库）。

走查数据已清理：w9admin 在验证 `LAST_ADMIN` 时被删除——开发库里还有 t3admin，它不是最后管理员，
行为符合预期（最后管理员保护另用独立测试库验证）；w9dana 与其帖子/关注/收藏全部清除，用户数回到 23；
临时二进制、配置、下载目录、profile 与服务停止。

### 12Y.4 仍未验证

- 网盘文件较多的账号导出/注销：没有构造 >100 个文件来验证分页删除循环。
- 导出在帖子很多时的内存占用没有压测（同步编码到响应）。
- 导入、邮件、2FA、Webhook、迁移等仍无端点。
- 注销后媒体对象是否真的从磁盘消失，只验证了行删除与 404，没有检查存储目录。

## 12Z. 批次：前端第十波——举报与处理（2026-11-06）

**工作方式**：目标模式第十波，一轮完成迁移、端点、菜单入口、管理页、测试、真实二进制与浏览器走查、
文档与提交。施工依据 `docs/research/2026-11-06-firefish-port-wave10-contract.md`。

### 12Z.1 落地清单

| 层 | 文件 | 说明 |
|----|------|------|
| 迁移 | `internal/db/db.go`（v11） | `abuse_reports` 表 + 未处理索引；双方外键级联 |
| 服务 | `internal/report/service.go`（新） | `Create` / `List`（state 与双方来源）/ `Resolve` / `Get` |
| 端点 | `internal/httpapi/reports.go`（新）、`server.go` | `users/report-abuse`、`admin/abuse-user-reports`、`admin/resolve-abuse-user-report` |
| 用户入口 | `web/src/scripts/get-user-menu.ts` | 非管理员对他人显示「举报」，`os.inputParagraph` 收集说明 |
| 管理页 | `web/src/pages/AdminAbuses.vue` | 占位页改为列表：筛选、双方信息、说明、标记已处理、加载更多 |
| 管理菜单 | `web/src/pages/AdminIndex.vue` | 内容组新增「举报」 |
| 接口层 | `web/src/api/endpoints.ts`、`types.ts` | 三个端点与 `AbuseReport` |
| 测试 | `internal/httpapi/reports_test.go`（新，2 个用例） | 生命周期与筛选、校验与权限 |

### 12Z.2 有意的降级

1. 不转发到远端实例，`forwarded` 恒为 false，页面注明。
2. 不通知管理员（没有邮件/推送）。
3. 只针对账号：帖子举报归到作者账号，不记录 noteId。
4. 标记已处理只是状态位，不自动冻结；处置仍用冻结按钮。

### 12Z.3 验证证据

```
gofmt -l internal/ cmd/                 # 空
go vet ./...                            # exit 0
CINDERFISH_TEST_DATABASE_URL=… go test -count=1 ./...   # 全绿（新增 2 个用例）
cd web && pnpm exec vue-tsc --noEmit    # exit 0
cd web && pnpm run check:port           # 398 文件，hardTotal 0
cd web && pnpm build                    # exit 0
```

真实二进制 `:3991`（开发库，`federation_mode=none`）：

| 场景 | 结果 |
|------|------|
| 匿名举报 / 自举报 | 401 / 400 |
| w10dana 在用户页菜单点「举报」并填写说明 | 菜单出现「举报」，对话框提交成功 |
| 管理台 `/admin/abuses` | 显示「w10dana 举报 w10admin」与说明文字；状态默认「未处理」 |
| 点「标记已处理」 | 徽标变为「已处理」；`state=unresolved` 再查为 0、`resolved` 为 1（测试覆盖） |
| `reporterOrigin=remote` | 0（双方均为本地） |
| 非管理员访问举报列表 | 403（测试覆盖） |

无头 Helium 156（1440×900 / 390×844）：管理页筛选栏与举报卡片无横向溢出、无 JS 异常；移动端
处理后列表为空并显示空态。截图在 `/tmp/w10-shots`（不入库）。

走查数据已清理：举报行与 w10admin/w10dana 删除，用户数回到 23；临时二进制、配置、媒体目录、
profile 与服务停止。

### 12Z.4 仍未验证

- 远端用户举报与 `targetUserOrigin=remote` 过滤没有数据（实例不联邦）。
- 举报分页 `untilId` 没有构造多页。
- 举报成功后的 toast 只在浏览器里看到「好」按钮关闭对话框，没有逐字核对提示文案。
- 转发、通知与自动处置按设计未做。

## 12AA. 批次：前端第十一波——Webhook（2026-11-06）

**工作方式**：目标模式第十一波，一轮完成迁移、服务、五个端点、投递接入、三个页面、测试、
真实二进制与浏览器走查、文档与提交。施工依据
`docs/research/2026-11-06-firefish-port-wave11-contract.md`。

### 12AA.1 落地清单

| 层 | 文件 | 说明 |
|----|------|------|
| 迁移 | `internal/db/db.go`（v12） | `webhooks` 表；事件列名用 `events` 避开 `ON` 保留字 |
| 服务 | `internal/webhook/service.go`（新） | CRUD + `Dispatch`（异步投递、5s 超时、回写 latestStatus） |
| 端点 | `internal/httpapi/webhooks.go`（新）、`server.go` | `i/webhooks/{create,list,show,update,delete}` |
| 投递接入 | `internal/httpapi/webhooks_dispatch.go`（新）、`notes.go` | 发帖成功后按 reply/renote/note 派发 |
| 页面 | `WebhookEditor.vue`、`SettingsWebhook*.vue` | 列表、新建、编辑共用表单 |
| 菜单 | `SettingsIndex.vue` | 客户端设置组新增 Webhook |
| 接口层 | `endpoints.ts`、`types.ts` | 五个端点与 `Webhook` |
| 测试 | `internal/httpapi/webhooks_test.go`（新，3 个用例） | CRUD/越权、校验、真实投递（含 reply 事件） |

### 12AA.2 有意的降级

1. 只产生 note / reply / renote 事件；其余枚举可存但不会触发，页面不提供勾选。
2. 没有 `i/webhooks/test`（参考快照也没有），不造假按钮。
3. 没有重试队列，失败只记录 `latestStatus`。
4. 不发送上游的 `X-Firefish-Host` 头。

### 12AA.3 验证证据

```
gofmt -l internal/ cmd/                 # 空
go vet ./...                            # exit 0
CINDERFISH_TEST_DATABASE_URL=… go test -count=1 ./...   # 全绿（新增 3 个用例）
cd web && pnpm exec vue-tsc --noEmit    # exit 0
cd web && pnpm run check:port           # 398 文件，hardTotal 0
cd web && pnpm build                    # exit 0
```

真实二进制 `:3991` + 本地接收器 `127.0.0.1:4499`（Node，记录请求）：

| 场景 | 结果 |
|------|------|
| `/settings/webhook` 空态 | 「还没有 Webhook」 |
| 表单创建「走查 Webhook」（URL 指向接收器） | 返回列表并显示名称、URL、事件 `note`、尚未投递 |
| 通过 API 发一条帖子 | 列表显示「最近投递 … HTTP 200」 |
| 接收器日志 | `X-Firefish-Hook-Id`/`Secret` 正确，body 中 `type=note`、帖子正文与作者匹配 |
| 编辑页关闭「启用」并保存 | 列表徽标变为「停用」 |
| 列表删除 + 确认 | 回到空态 |
| 移动端 `/settings/webhook`、`/webhook/new` | 无横向溢出、无 JS 异常 |

Go 测试另外覆盖 reply 事件与 `latestStatus=200` 回写。截图在 `/tmp/w11-shots`（不入库）。

走查数据已清理：w11user 与 Webhook、帖子级联删除（webhooks 计数 0），用户数回到 23；服务、
接收器、临时二进制、媒体目录与 profile 停止/删除。

### 12AA.4 仍未验证

- follow / reaction / mention 等事件没有产生方，只验证了 API 接受与页面不展示。
- 失败与超时路径（`latestStatus=1`、5s 超时）只在实现层检查，没有用挂起的接收器实测。
- 重试、批量并发与投递顺序没有测试；个人实例先接受。
- 编辑页的删除只走了列表删除路径，编辑页内的删除按钮没有点击。

## 12AB. 批次：前端第十二波——数据库概览（2026-11-06）

**工作方式**：目标模式第十二波，一轮完成两个只读端点、管理页、测试、真实二进制与浏览器走查、
文档与提交。施工依据 `docs/research/2026-11-06-firefish-port-wave12-contract.md`。

### 12AB.1 落地清单

| 层 | 文件 | 说明 |
|----|------|------|
| 端点 | `internal/httpapi/admin_database.go`（新）、`server.go` | `admin/get-table-stats`（reltuples + total size）、`admin/get-index-stats`（pg_indexes） |
| 页面 | `web/src/pages/AdminDatabase.vue` | 总大小、表按大小排序（行数/大小/占比条）、索引列表与可展开定义 |
| 菜单 | `web/src/pages/AdminIndex.vue` | 内容组新增「数据库」 |
| 接口层 | `endpoints.ts`、`types.ts` | 两个端点、`TableStat`、`IndexStat` |
| 测试 | `internal/httpapi/admin_database_test.go`（新） | 表/索引存在性、数值类型、非管理员 403、匿名 401 |

### 12AB.2 有意的限制

1. 行数是 `pg_class.reltuples` 估算而非 `COUNT(*)`，页面注明；未分析的表显示 0。
2. 只读：没有 vacuum、删除或索引重建。
3. 表与索引不分页，schema 规模决定返回量（当前 31 表 / 75 索引）。
4. 不泄露连接信息或敏感元数据。

### 12AB.3 验证证据

```
gofmt -l internal/ cmd/                 # 空
go vet ./...                            # exit 0
CINDERFISH_TEST_DATABASE_URL=… go test -count=1 ./...   # 全绿（新增 1 个用例）
cd web && pnpm exec vue-tsc --noEmit    # exit 0
cd web && pnpm run check:port           # 398 文件，hardTotal 0
cd web && pnpm build                    # exit 0
```

真实二进制 `:3991`（开发库）：

| 场景 | 结果 |
|------|------|
| `admin/get-table-stats` | 31 张表；`users` = 24 行 / 136 KB |
| `admin/get-index-stats` | 75 条索引记录 |
| `/admin/database` | 总大小 1.5 MB、31 表 75 索引；表按大小排序（notes → users → notifications …） |
| 索引「定义」展开 | 显示 `CREATE INDEX …` 原文 |
| 管理菜单 | 内容组出现「数据库」并高亮 |
| 移动端 390×844 | 表格与索引列表正常换行，无横向溢出 |
| 非管理员 / 匿名 | 403 / 401（测试覆盖） |

无头 Helium 156：桌面与移动均无 JS 异常。截图在 `/tmp/w12-shots`（不入库）。

走查数据已清理：w12admin 删除，用户数回到 23；临时二进制、配置、媒体目录、profile 与服务停止。

### 12AB.4 仍未验证

- `reltuples` 估算与实际行数的偏差没有用 `ANALYZE` 前后对比。
- 大 schema（数百表/索引）下的页面滚动与渲染性能没有测试（无分页）。
- `vacuum`、索引维护与表清理按设计未做。

## 12AC. 批次：前端第十三波——运行时实例设置（2026-11-06）

**工作方式**：目标模式第十三波，一轮完成运行时设置存储、两个管理端点、meta/nodeinfo 接线、
管理页、测试、真实二进制与浏览器走查、文档与提交。施工依据
`docs/research/2026-11-06-firefish-port-wave13-contract.md`。

### 12AC.1 落地清单

| 层 | 文件 | 说明 |
|----|------|------|
| 运行时设置 | `internal/httpapi/instance_settings.go`（新） | `liveInstanceSettings`、`liveRegistrationMode`、`saveInstanceSetting` |
| 端点 | 同上、`server.go` | `admin/meta`、`admin/update-meta`（五个字段，其余忽略） |
| 接线 | `meta.go`、`account.go`、`activitypub.go` | 名称/简介/维护者/注册模式/`disableRegistration`/nodeinfo 读运行时值 |
| 页面 | `web/src/pages/AdminSettings.vue` | 名称、简介、维护者、邮箱、注册模式表单；保存提示刷新 |
| 菜单 | `AdminIndex.vue` | 管理组新增「实例设置」 |
| 接口层 | `endpoints.ts`、`types.ts` | 两个端点与 `AdminInstanceMeta` |
| 测试 | `internal/httpapi/instance_settings_test.go`（新，3 个用例） | 设置回读、运行时注册模式切换、校验与权限 |

### 12AC.2 有意的限制

1. 只覆盖五个字段；数据库、媒体、联邦策略等仍在 `config.toml`，页面写明。
2. `admin/update-meta` 忽略未列出的上游字段，不假装保存。
3. 没有审计日志。
4. `invite` 模式依赖邀请码库存，没有邀请码时 signup 失败。

### 12AC.3 验证证据

```
gofmt -l internal/ cmd/                 # 空
go vet ./...                            # exit 0
CINDERFISH_TEST_DATABASE_URL=… go test -count=1 ./...   # 全绿（新增 3 个用例）
cd web && pnpm exec vue-tsc --noEmit    # exit 0
cd web && pnpm run check:port           # 398 文件，hardTotal 0
cd web && pnpm build                    # exit 0
```

真实二进制 `:3991`（开发库）：

| 场景 | 结果 |
|------|------|
| `admin/meta` 初始 | `name=Cinderfish`、`registrationMode=open` |
| 管理页改为「W13 实例」+ 简介 + 「仅管理员创建」并保存 | 页面显示已保存 |
| `POST /api/meta` | `name=W13 实例`、简介一致、`disableRegistration=true` |
| 切到 admin 模式后 signup | 400 |
| 切回开放注册后 signup | 200 |
| 管理菜单 | 出现「实例设置」 |
| 移动端 `/admin/settings` | 表单正常换行，无横向溢出 |

无头 Helium 156：无 JS 异常。截图在 `/tmp/w13-shots`（不入库）。

测试还暴露并修掉一个隔离问题：`instance_meta` 未加入测试库 TRUNCATE，导致运行时注册模式跨用例
泄漏；已加入，httpapi 全包恢复全绿。

走查数据已清理：w13admin/w13guest 删除，`instance_meta` 中本次写入的五个键删除（实例恢复
默认名称与配置模式），用户数回到 23；临时二进制、配置、媒体目录、profile 与服务停止。

### 12AC.4 仍未验证

- 邀请码模式下用有效邀请码完成注册没有走查（需要 CLI 生成邀请码）。
- 并发保存、审计日志、设置变更后的缓存失效（公开页需手动刷新）没有处理。
- 上游 `admin/update-meta` 其余字段按设计忽略；Firefish 管理页若直连本实例会显示“保存成功但多数
  字段未生效”，这一点只在契约里记录。

## 12AD. 批次：前端第十四波——实例屏蔽（2026-11-06）

**工作方式**：目标模式第十四波，一轮完成策略层、入站检查、管理接口扩展、管理页、测试、真实二进制
与浏览器走查、文档与提交。施工依据 `docs/research/2026-11-06-firefish-port-wave14-contract.md`。

### 12AD.1 落地清单

| 层 | 文件 | 说明 |
|----|------|------|
| 策略 | `internal/federation/policy.go` | `HostBlocked` / `BlockedHosts` / `SetBlockedHosts`，带 TTL 缓存与立即失效；`IsAllowed` 先查屏蔽 |
| 入站 | `internal/httpapi/activitypub.go` | inbox 在签名验证后按 actor 主机拒绝（403） |
| 管理接口 | `internal/httpapi/instance_settings.go` | `admin/meta` 返回 `blockedHosts`（恒为数组）；`admin/update-meta` 接受并校验 |
| 页面 | `web/src/pages/AdminInstanceBlock.vue` | 多行主机列表、去重预览、保存与数量提示、精确匹配说明 |
| 菜单 | `AdminIndex.vue` | 内容组新增「实例屏蔽」 |
| 测试 | `internal/federation/policy_test.go`、`internal/httpapi/instance_settings_test.go` | 精确匹配/大小写；blockedHosts 规范化、回读、非法值 400 |

### 12AD.2 有意的限制

1. 精确匹配：`bad.example` 不自动覆盖子域，页面提示逐条列出。
2. 不删除已收到的内容、不取消已有关注；只阻止之后的收发。
3. 入站仍不做 allowlist 检查（保持既有行为），只新增屏蔽检查。
4. 没有 `silencedHosts`、没有转发屏蔽。

### 12AD.3 验证证据

```
gofmt -l internal/ cmd/                 # 空
go vet ./...                            # exit 0
CINDERFISH_TEST_DATABASE_URL=… go test -count=1 ./...   # 全绿
cd web && pnpm exec vue-tsc --noEmit    # exit 0
cd web && pnpm run check:port           # 398 文件，hardTotal 0
cd web && pnpm build                    # exit 0
```

真实二进制 `:3991`（`federation_mode=public`）：

| 场景 | 结果 |
|------|------|
| `admin/meta` 初始 | `blockedHosts: []` |
| API 写入 `["Bad.Example","bad.example","sub.bad.example"]` | 规范化并排序为 `["bad.example","sub.bad.example"]` |
| 管理页写入三行（含重复大小写） | 保存成功，显示「共 3 个被屏蔽实例」 |
| 再读 `admin/meta` | `["bad.example","spam.example","sub.bad.example"]` |
| 管理菜单 | 出现「实例屏蔽」 |
| 移动端 | 文本域与说明正常换行，无横向溢出 |

无头 Helium 156：无 JS 异常。截图在 `/tmp/w14-shots`（不入库）。

顺带修正：`admin/meta` 的 `blockedHosts` 在空列表时返回 `[]` 而不是 `null`。

走查数据已清理：w14admin 与 `instance_meta.blockedHosts` 删除，用户数回到 23；服务与临时文件停止。

### 12AD.4 仍未验证

- 真实入站拒绝没有构造签名请求，只验证了策略与页面；出站拒绝由既有 `client.go` 路径保证。
- 与 allowlist 模式组合、子域精确匹配的运维体验没有走查。
- 已收到的远端内容不会清理，按设计保留。

## 12AE. 批次：前端第十五波——MiAuth 第三方授权（2026-11-06）

**工作方式**：目标模式第十五波，一轮完成迁移、check 端点、授权页、缺失文案、测试、真实二进制与浏览器
走查、文档与提交。施工依据 `docs/research/2026-11-06-firefish-port-wave15-contract.md`。

### 12AE.1 落地清单

| 层 | 文件 | 说明 |
|----|------|------|
| 迁移 | `internal/db/db.go`（v13） | `access_tokens.fetched`，标记 token 已被应用取走 |
| 令牌服务 | `internal/account/tokens.go` | `FindUnfetchedBySession`、`MarkFetched` |
| 端点 | `internal/httpapi/miauth.go`（新）、`server.go` | `POST /api/miauth/{session}/check`：一次取走，之后 `{ok:false}` |
| 页面 | `web/src/pages/MiAuth.vue` | 从占位页改为复用 `CfMiauth`（登录、权限展示、接受/取消、callback 跳转） |
| 文案 | `i18n/locales/{zh-CN,ja-JP}.json` | 补 `_permissions.read:notes` |
| 测试 | `internal/httpapi/miauth_test.go`（新） | 首次取走含 user、二次 ok=false、未知 session ok=false |

### 12AE.2 有意的限制

1. scope 不强制：permission 只用于展示，除 Streaming 的 `read:account` 外服务端不校验。
2. 没有拒绝回执：取消只是不签发 token，应用轮询一直是 `{ok:false}`。
3. 没有授权页过期与 token 生命周期区分。
4. OAuth 同意页仍未做（`/oauth/authorize` 仍直接签码）。

### 12AE.3 验证证据

```
gofmt -l internal/ cmd/                 # 空
go vet ./...                            # exit 0
CINDERFISH_TEST_DATABASE_URL=… go test -count=1 ./...   # 全绿（新增 MiAuth check 用例）
cd web && pnpm exec vue-tsc --noEmit    # exit 0
cd web && pnpm run check:port           # 398 文件，hardTotal 0
cd web && pnpm build                    # exit 0
```

真实二进制 `:3991`：

| 场景 | 结果 |
|------|------|
| 授权前 `miauth/w15-app/check` | `{ok:false}` |
| 打开 `/miauth/w15-app?name=W15 应用&permission=read:notes,write:notes` | 显示应用名、两项权限（「查看帖子」「撰写或删除帖子」）、取消/接受 |
| 点「接受」 | 页面显示「请返回至应用」 |
| 首次轮询 | `{ok:true, token:<32 位>, user:w15user}` |
| 第二次轮询 | `{ok:false}` |
| 移动端 390×844 | 授权卡片正常换行，无横向溢出 |

无头 Helium 156：无 JS 异常。截图在 `/tmp/w15-shots`（不入库）。

走查数据已清理：w15user 与其令牌删除，用户数回到 23；服务与临时文件停止。

### 12AE.4 仍未验证

- 未登录状态下的登录表单与授权衔接没有走查（走查注入的是已登录会话）。
- 带 `callback` 的应用跳转没有构造真实第三方应用。
- OAuth 同意页、拒绝回执、scope 强制与授权页过期按设计未做。

## 12AF. 批次：前端第十六波——通知页签与提及（2026-11-06）

**工作方式**：目标模式第十六波，一轮完成 `notes/mentions`、`unreadOnly`、通知页签、测试、真实二进制
与浏览器走查、文档与提交。施工依据 `docs/research/2026-11-06-firefish-port-wave16-contract.md`。

### 12AF.1 落地清单

| 层 | 文件 | 说明 |
|----|------|------|
| 提及 | `internal/note/mentions.go`（新） | `Mentions`：jsonb 成员匹配 + 可见性/静音/屏蔽/实例静音 + following/visibility |
| 端点 | `internal/httpapi/mentions.go`（新）、`server.go` | `POST /api/notes/mentions`，匿名 401 |
| 未读 | `internal/note/reaction.go`、`internal/httpapi/notifications.go` | `List` 增加 `unreadOnly`，handler 透传 |
| 页签 | `web/src/pages/Notifications.vue` | 全部/未读/提及；取消自动 mark-all-read，改为页头显式动作 |
| 通知列表 | `web/src/components/MkNotifications.vue` | 新增 `unreadOnly` prop |
| 接口层 | `endpoints.ts` | `notesMentions` |
| 测试 | `internal/httpapi/mentions_test.go`（新） | 提及可见性/关注过滤；未读过滤与全部已读 |

### 12AF.2 有意的限制

1. 没有 directNotes 页签（无聊天范围）。
2. 远端账号的提及不进入本地 `mentions` 数组，因此不出现在提及列表。
3. 展示通知即标记已读，未读列表会随之减少（既有行为）。

### 12AF.3 验证证据

```
gofmt -l internal/ cmd/                 # 空
go vet ./...                            # exit 0
CINDERFISH_TEST_DATABASE_URL=… go test -count=1 ./...   # 全绿（新增 2 个用例）
cd web && pnpm exec vue-tsc --noEmit    # exit 0
cd web && pnpm run check:port           # 398 文件，hardTotal 0
cd web && pnpm build                    # exit 0
```

真实二进制 `:3991`：

| 场景 | 结果 |
|------|------|
| `/my/notifications` | 页签「全部 / 未读 / 提及」都在 |
| 停在「提及」时 alice 发 `hello @w16bob mention check` | 切到「未读」能看到该通知 |
| 「提及」页签 | 显示带 @ 的帖子（含更早一条） |
| `notes/mentions` 匿名调用 | 401（测试覆盖） |
| following=true 未关注时 | 0；关注后 1（测试覆盖） |
| `unreadOnly` 全部已读后 | 未读 0、全部仍 1（测试覆盖） |
| 移动端 | 页签与列表正常，无横向溢出、无 JS 异常 |

无头 Helium 156。截图在 `/tmp/w16-shots`（不入库）。

顺带修正：页面一进来就 mark-all-as-read 会让「未读」页签永远为空，已改为显式页头动作。
另修正 API 台账自第九波起累计的算术漂移（同名声明的未实现数与 440 总数不一致），本波按
「440 − 同名已实现」重新对齐。

走查数据已清理：w16alice/w16bob 与其帖子、通知级联删除，用户数回到 23；服务与临时文件停止。

### 12AF.4 仍未验证

- 远端账号提及的展示（当前设计不进入本地列表）。
- 未读页签在流式推送到达时的实时插入没有专门验证。
- directNotes、按类型过滤与页签组合没有走查。

## 12AG. 批次：前端第十七波——用户发现（2026-11-06）

**工作方式**：目标模式第十七波，一轮完成公开用户目录、发现页用户页签、测试、真实二进制与浏览器
走查、文档与提交。施工依据 `docs/research/2026-11-06-firefish-port-wave17-contract.md`。

### 12AG.1 落地清单

| 层 | 文件 | 说明 |
|----|------|------|
| 目录 | `internal/account/users_list.go`（新） | `ListUsers`：冻结过滤、sort/state/origin/hostname、offset 分页 |
| 端点 | `internal/httpapi/users_list.go`（新）、`server.go` | `POST /api/users`；匿名可读，登录补 `isFollowing`/`isFollowed` |
| 页面 | `web/src/pages/Explore.vue` | 新增「用户」页签并设为默认；`MkUserList` + `sort=+follower` |
| 接口层 | `endpoints.ts` | `usersList` |
| 测试 | `internal/httpapi/users_list_test.go`（新） | 冻结过滤、粉丝排序、来源、关注标记、非法 state 400 |

### 12AG.2 有意的限制

1. `alive` 与 `all` 相同（均排除冻结）；没有 moderator 标记，相关 state 等价于 admin。
2. 没有随机/推荐排序。
3. 远程用户在非联邦实例上为空集。
4. **offset 分页**：`users` 只支持 offset，用户页签因此自己维护 offset 与「加载更多」
   （请求 `limit+1` 判断 hasMore），没有复用走 id 游标的 `MkUserList`。

### 12AG.3 验证证据

```
gofmt -l internal/ cmd/                 # 空
go vet ./...                            # exit 0
CINDERFISH_TEST_DATABASE_URL=… go test -count=1 ./...   # 全绿（新增目录端点用例）
cd web && pnpm exec vue-tsc --noEmit    # exit 0
cd web && pnpm run check:port           # 398 文件，hardTotal 0
cd web && pnpm build                    # exit 0
```

真实二进制 `:3991`（开发库 23 个账号 + w17alice/w17bob）：

| 场景 | 结果 |
|------|------|
| `POST /api/users`（local） | 25 个账号 |
| `sort=+follower&limit=3` | 粉丝数 1 的账号排在前面 |
| `state=bogus` | 400 |
| `/explore`（默认页签） | 页签「用户 / 热门标签 / 本地」；用户卡片显示帖子/关注/粉丝计数 |
| 点「加载更多」 | 用户卡片从 20 条增至 24 条（开发库 23 账号 + 走查账号） |
| 切到「热门标签」 | 趋势标签正常 |
| 移动端 390×844 | 用户卡片正常换行，无横向溢出、无 JS 异常 |

无头 Helium 156。截图在 `/tmp/w17-shots`（不入库）。

走查数据已清理：w17alice/w17bob 与关注关系级联删除，用户数回到 23；服务与临时文件停止。

### 12AG.4 仍未验证 / 已知偏差

- 远程用户目录、`hostname` 过滤与 `state=admin` 没有构造数据。
- 用户页签的「加载更多」没有分页到底后的空态截图（当前数据量只够翻一页）。
- 未登录访客看到的目录没有走查（走查注入的是登录会话）。

## 12AH. 批次：前端第十八波——运维可视化（2026-11-06）

**工作方式**：目标模式第十八波，一轮完成两个只读运维端点、两个管理页、测试、真实二进制与浏览器
走查、文档与提交。施工依据 `docs/research/2026-11-06-firefish-port-wave18-contract.md`。

### 12AH.1 落地清单

| 层 | 文件 | 说明 |
|----|------|------|
| 端点 | `internal/httpapi/admin_infra.go`（新）、`server.go` | `admin/object-storage`（只读、无密钥）、`admin/deliver-queue`（计数与最近 20 行） |
| 对象存储页 | `web/src/pages/AdminObjectStorage.vue` | 只读卡片；本地路径或 S3 主机/桶/区域/HTTPS/公开 URL/凭据是否配置 |
| 队列页 | `web/src/pages/AdminQueue.vue` | 待投递/已到期/超重试上限、最早时间、最近投递与最后错误、刷新 |
| 菜单 | `AdminIndex.vue` | 管理组「投递队列」，内容组「对象存储」 |
| 接口层 | `endpoints.ts`、`types.ts` | 两个端点与 `ObjectStorageInfo` / `DeliverQueueInfo` |
| 测试 | `internal/httpapi/admin_infra_test.go`（新，2 个用例） | 只读配置不含密钥；空队列形状、匿名 401、非管理员 403 |

### 12AH.2 有意的限制

1. 两个端点都是 Cinderfish 自有路径，不是 Misskey 同名端点（Bull 面板与对象存储写接口都不存在）。
2. 完全只读：没有改配置、重试、清空队列的按钮。
3. 队列页最多显示最近 20 行；`failed` 只统计 `attempts >= 12` 的行。

### 12AH.3 验证证据

```
gofmt -l internal/ cmd/                 # 空
go vet ./...                            # exit 0
CINDERFISH_TEST_DATABASE_URL=… go test -count=1 ./...   # 全绿（新增 2 个用例）
cd web && pnpm exec vue-tsc --noEmit    # exit 0
cd web && pnpm run check:port           # 398 文件，hardTotal 0
cd web && pnpm build                    # exit 0
```

真实二进制 `:3991`（本地媒体 `/tmp/w18-media`，seed 一条 `deliver_queue` 行）：

| 场景 | 结果 |
|------|------|
| `admin/object-storage` | `backend=local`、`localPath=/tmp/w18-media`、`s3=null`，无任何密钥字段 |
| `admin/deliver-queue`（seed 后） | 显示 1 条：`w18-remote.example`、attempts 2、`connection refused` |
| `/admin/object-storage` | 只读卡片显示本地路径与「修改需重启」说明 |
| `/admin/queue` | 计数与最近行；走查期间后台 worker 继续重试，行最终到 attempts 12、错误换成联邦拒绝，`failed` 变为 1——页面如实反映 |
| 管理菜单 | 「投递队列」「对象存储」都在 |
| 移动端两页 | 无横向溢出、无 JS 异常 |

无头 Helium 156。截图在 `/tmp/w18-shots`（不入库）。

走查数据已清理：w18admin 与 seed 的队列行删除（用户 23、`deliver_queue` 0）；服务与临时文件停止。

### 12AH.4 仍未验证

- 队列的正确退避曲线与 worker 并发没有压测；只观察了一条行在走查期间被重试到上限。
- S3 后端的页面展示只用单元测试检查；没有真实桶。
- 手动重试/清空、S3 配置写入按设计未做。

## 12AI. 批次：前端第十九波——实例身份、合规与统计（2026-11-06）

**工作方式**：目标模式第十九波，一轮完成设置字段扩展、公开统计端点、管理页与关于页、测试、
真实二进制与浏览器走查、文档与提交。施工依据
`docs/research/2026-11-06-firefish-port-wave19-contract.md`。

### 12AI.1 落地清单

| 层 | 文件 | 说明 |
|----|------|------|
| 设置字段 | `internal/httpapi/instance_settings.go` | 六个合规 URL + `serverRules`（JSON 数组），`admin/meta` 回读、`update-meta` 校验 |
| 公开 meta | `internal/httpapi/meta.go` | `tosUrl`/`privacyPolicyUrl`/`repositoryUrl`/`feedbackUrl`/`inquiryUrl`/`impressumUrl`/`serverRules` 改用运行时值 |
| 统计 | `internal/httpapi/stats.go`（新）、`server.go` | `POST /api/stats`：9 个计数，单条 SQL |
| 管理页 | `web/src/pages/AdminSettings.vue` | 六个 URL 输入 + 规则文本域，保存时整体提交 |
| 关于页 | `web/src/pages/About.vue` | 合规链接、规则列表、真实统计数字（替换占位说明） |
| 接口层 | `endpoints.ts`、`types.ts` | `stats`、`InstanceStats`、`AdminInstanceMeta` 扩展 |
| 测试 | `internal/httpapi/stats_test.go`（新，2 个用例） | 合规字段回读/校验、统计计数 |

### 12AI.2 有意的限制

1. 规则不进入注册流程（没有“同意规则”勾选）。
2. 统计即时 `count(*)`，无缓存。
3. `instancesCount` 只是库里有账号的远程实例数。
4. 没有 `charts/*` 历史图表。

### 12AI.3 验证证据

```
gofmt -l internal/ cmd/                 # 空
go vet ./...                            # exit 0
CINDERFISH_TEST_DATABASE_URL=… go test -count=1 ./...   # 全绿（新增 2 个用例）
cd web && pnpm exec vue-tsc --noEmit    # exit 0
cd web && pnpm run check:port           # 398 文件，hardTotal 0
cd web && pnpm build                    # exit 0
```

真实二进制 `:3991`（开发库 23 账号 + w19admin）：

| 场景 | 结果 |
|------|------|
| `stats` | usersCount 24、notesCount 154、driveFiles 1、instancesCount 0 |
| 管理页填写条款 URL 与两条规则并保存 | 保存成功 |
| `POST /api/meta` | `tosUrl=https://w19.example/tos`、`serverRules` 两条 |
| `/about` | 显示「服务条款」链接、两条实例规则、用户/帖子/网盘/实例统计数字 |
| 移动端 `/about` | 无横向溢出、无 JS 异常 |

无头 Helium 156。截图在 `/tmp/w19-shots`（不入库）。

走查数据已清理：`instance_meta` 中本次写入的所有键删除（仅保留既有的 `federationMode`），
w19admin 删除，用户数回到 23；服务与临时文件停止。

### 12AI.4 仍未验证

- 规则与合规链接没有进入注册/欢迎页流程，只验证了关于页。
- `instancesCount` 与远程账号计数在非联邦实例上为 0，没有构造远端数据。
- 统计在大量数据下的耗时没有测量。

## 12AJ. 批次：两步验证（TOTP）（2026-11-06）

**工作方式**：目标模式第二十波。按用户 2026-11-06 的指示，本轮**只写代码、不执行测试与走查**，
验证集中在编译层（`gofmt`/`go vet`/`vue-tsc`/`pnpm build`），端到端与视觉验证留到统一阶段。
施工依据 `docs/research/2026-11-06-firefish-port-wave20-contract.md`。

### 12AJ.1 落地清单

| 层 | 文件 | 说明 |
|----|------|------|
| 迁移 | `internal/db/db.go` | v14：`user_profiles` 增加 `two_factor_secret`、`two_factor_temp_secret`、`two_factor_enabled`、`use_password_less_login` |
| TOTP | `internal/auth/totp.go`（新） | 20 字节 base32 密钥、HMAC-SHA1/6 位/30 秒、±1 步长、`otpauth://` URL |
| 账号层 | `internal/account/two_factor.go`（新）、`profile.go` | `CheckPassword`、`GetTwoFactorState`、`Begin/Enable/DisableTwoFactor`；Profile 带出两个布尔 |
| 端点 | `internal/httpapi/two_factor.go`（新）、`server.go` | `i/2fa/register|done|unregister|password-less` |
| 登录 | `internal/httpapi/account.go` | 请求体收 `token`；启用 TOTP 且无码时返回 `200 {}`，错码 403 |
| 打包 | `internal/httpapi/users_pack.go` | `twoFactorEnabled`、`usePasswordLessLogin`、`securityKeys` |
| 前端类型 | `api/types.ts`、`endpoints.ts`、`account/types.ts` | `SigninStep`、`TwoFactorRegistration`、四个端点、`UserDetailed` 三字段 |
| 登录流程 | `account/session.ts`、`transport.ts`、`CfSignin.vue`、`SettingsAccounts.vue` | 两步登录、验证码错误分类、添加账号的二次验证 |
| 安全设置 | `pages/SettingsSecurity.vue` | 真实 TOTP 面板：密码确认 → 密钥/URL → 验证码启用；停用需再输密码 |
| 测试 | — | 按指示本轮未写/未跑；既有测试未改动 |

### 12AJ.2 有意的限制

1. `i/2fa/register` 的 `qr` 恒为空串（不引二维码依赖），页面展示密钥与 `otpauth://` URL。
2. 安全密钥/通行密钥未实现，`i/2fa/password-less` 恒报 `NO_SECURITY_KEY`，`securityKeys` 恒 false。
3. 无备份码，`twoFactorBackupCodesStock` 仍为 `"none"`。
4. 密钥明文存库（与 Firefish 相同）。
5. 无登录失败限流（既有缺口）。

### 12AJ.3 验证证据（编译层）

```
gofmt -l internal/ cmd/                 # 空
go vet ./...                            # exit 0
cd web && pnpm exec vue-tsc --noEmit    # exit 0
cd web && pnpm run check:port           # 398 文件，hardTotal 0
cd web && pnpm build                    # exit 0
```

### 12AJ.4 仍未验证

- 注册 → 启用 → 登出 → 带码登录的端到端链路（含错码与空码两条分支）。
- 设置页的密码确认、密钥展示、启用/停用交互与移动端布局。
- 并发登录、时钟漂移 ±1 步长之外的行为、验证码重放（同一码在有效窗口内可重复使用，与 TOTP 规范一致）。
- `go test`（含数据库用例）未执行，新增端点尚无自动化用例。

## 12AK. 批次：账号安全操作（2026-11-06）

**工作方式**：目标模式第二十一波，承接第二十波。按用户 2026-11-06 的指示仍然只写代码，
验证只到编译层。施工依据 `docs/research/2026-11-06-firefish-port-wave21-contract.md`。

### 12AK.1 落地清单

| 层 | 文件 | 说明 |
|----|------|------|
| 账号层 | `internal/account/security.go`（新） | `ChangePassword`、`RegenerateToken`、`SigninHistory` |
| 端点 | `internal/httpapi/security.go`（新）、`server.go` | `i/change-password`、`i/regenerate-token`、`i/signin-history` |
| 前端类型 | `api/types.ts`、`endpoints.ts` | `SigninHistoryEntry`、三个端点 |
| 安全设置 | `pages/SettingsSecurity.vue` | 改密表单、令牌重生成并更新本地凭据、登录历史列表与分页 |

### 12AK.2 有意的限制

1. 改密不吊销既有 token，也没有“退出所有设备”。
2. 无密码强度校验（只要求非空，与 Misskey 一致）。
3. 登录历史的 `headers` 恒为 `{}`，没有 User-Agent/地理位置。
4. `i/regenerate-token` 额外返回 `{token}`（Misskey 返回空），便于本移植立即替换本地凭据。
5. 登录失败限流仍缺。

### 12AK.3 验证证据（编译层）

```
gofmt -l internal/ cmd/                 # 空
go vet ./...                            # exit 0
cd web && pnpm exec vue-tsc --noEmit    # exit 0
cd web && pnpm run check:port           # hardTotal 0
cd web && pnpm build                    # exit 0
```

### 12AK.4 仍未验证

- 改密后旧密码失效、新密码登录成功的端到端流程。
- 令牌重生成后当前浏览器继续可用、旧令牌 401 的行为。
- 登录历史的游标分页（第二页）与失败记录展示。
- `go test` 与浏览器走查未执行，三个端点没有自动化用例。

## 12AL. 批次：资料扩展字段（2026-11-06）

**工作方式**：目标模式第二十二波。继续按用户指示只写代码，验证到编译层。施工依据
`docs/research/2026-11-06-firefish-port-wave22-contract.md`。

### 12AL.1 落地清单

| 层 | 文件 | 说明 |
|----|------|------|
| 账号层 | `internal/account/profile.go`、`profile_details.go`（新） | Profile 新增元数据与布尔字段；`UpdateProfileDetails` 合并进 `settings` JSONB |
| 端点 | `internal/httpapi/i_update.go` | 接受 location/birthday/lang/fields 与 10 个布尔开关，含长度与格式校验 |
| 打包 | `internal/httpapi/users_pack.go` | `UserDetailed` 的 location/birthday/lang/fields/publicReactions、`MeDetailed` 的布尔开关改为真实值 |
| 前端类型 | `api/types.ts`、`account/types.ts` | `ProfileField`、`ProfileUpdateRequest` 扩展 |
| 资料页 | `pages/SettingsProfile.vue` | 位置、生日、语言、附加信息编辑器 |
| 其它设置 | `pages/SettingsOther.vue` | 推荐笔记开关改为真实写入 |

### 12AL.2 有意的限制

1. 横幅、锁推、关注审批、`isBot`/`isCat`、发言猫语仍无写入路径。
2. 邮箱修改与 `i/update-email` 仍缺。
3. 元数据与布尔值存在 `settings` JSONB，不是独立列。
4. `fields` 的链接不做 URL 校验。

### 12AL.3 验证证据（编译层）

```
gofmt -l internal/ cmd/                 # 空
go vet ./...                            # exit 0
cd web && pnpm exec vue-tsc --noEmit    # exit 0
cd web && pnpm run check:port           # hardTotal 0
cd web && pnpm build                    # exit 0
```

### 12AL.4 仍未验证

- 保存资料（含附加信息增删）后 `users/show` 与 `POST /api/i` 的回读。
- 日期/语言非法值的 400 分支与清空语义。
- 推荐笔记开关的写入与时间线表现（时间线本身还没有插入推荐笔记的逻辑）。
- `go test` 与浏览器走查未执行。

## 12AM. 批次：通知类型静音（2026-11-06）

**工作方式**：目标模式第二十三波，继续只写代码、验证到编译层。施工依据
`docs/research/2026-11-06-firefish-port-wave23-contract.md`。

### 12AM.1 落地清单

| 层 | 文件 | 说明 |
|----|------|------|
| 账号层 | `internal/account/profile.go`、`profile_details.go` | `MutedNotificationTypes` 存入 `settings.mutingNotificationTypes` |
| 端点 | `internal/httpapi/i_update.go` | 接受并校验 `mutingNotificationTypes`（只允许 5 类） |
| 过滤 | `internal/httpapi/notifications.go`、`internal/note/reaction.go` | 列表并入 excludeTypes；`UnreadCount` 增加排除参数 |
| 打包 | `internal/httpapi/users_pack.go` | `MeDetailed.mutingNotificationTypes` 回读真实值 |
| 前端 | `account/types.ts`、`pages/SettingsNotifications.vue` | 5 个逐类开关，切换即保存 |

### 12AM.2 有意的限制

1. 只支持本后端产生的 5 类通知，未知类型返回 400。
2. 静音只影响列表与未读计数，不阻止通知写入。
3. `notificationRecieveConfig` 与邮件通知类型未实现。

### 12AM.3 验证证据（编译层）

```
gofmt -l internal/ cmd/                 # 空
go vet ./...                            # exit 0
cd web && pnpm exec vue-tsc --noEmit    # exit 0
cd web && pnpm run check:port           # hardTotal 0
cd web && pnpm build                    # exit 0
```

### 12AM.4 仍未验证

- 静音某一类后 `notifications` 列表与 `unreadNotificationsCount` 的同步下降。
- 非法类型与重复项的 400/去重分支。
- 设置页开关的失败回滚与移动端布局。
- `go test` 与浏览器走查未执行。

## 12AN. 批次：笔记编辑与编辑历史（2026-11-06）

**工作方式**：目标模式第二十四波，继续只写代码、验证到编译层。施工依据
`docs/research/2026-11-06-firefish-port-wave24-contract.md`。

### 12AN.1 落地清单

| 层 | 文件 | 说明 |
|----|------|------|
| 迁移 | `internal/db/db.go` | v15：`notes.updated_at`、`note_edits` 表 |
| 服务层 | `internal/note/edit.go`（新） | `Edit`（重算 tags/mentions、记录旧版本）、`Edits` |
| 端点 | `internal/httpapi/note_edit.go`（新）、`server.go` | `notes/edit`（Firefish）、`notes/history`（自有） |
| 前端类型 | `api/types.ts`、`endpoints.ts` | `NoteEditEntry`、`NoteEditResponse` |
| 路由/页面 | `router/routes.ts`、`pages/NoteEdit.vue`（新）、`pages/NoteHistory.vue` | 编辑表单；历史列表替换占位说明 |
| 菜单 | `scripts/get-note-menu.ts` | 作者菜单新增「编辑」「编辑历史」 |

### 12AN.2 有意的限制

1. 只能改正文、CW、可见性与 localOnly；附件/投票/回复目标不变。
2. 不发 Streaming `updated`、不投递 ActivityPub `Update`，远端副本不更新。
3. 响应不打包 `updatedAt`，没有「已编辑」标记。
4. `notes/history` 是自有端点，Misskey 固定版本没有。

### 12AN.3 验证证据（编译层）

```
gofmt -l internal/ cmd/                 # 空
go vet ./...                            # exit 0
cd web && pnpm exec vue-tsc --noEmit    # exit 0
cd web && pnpm run check:port           # hardTotal 0
cd web && pnpm build                    # exit 0
```

### 12AN.4 仍未验证

- 编辑保存 → 历史出现旧文本 → 回读笔记的端到端流程。
- 他人笔记 403、缺 `editId` 400、超长正文 400 的分支。
- 菜单入口在时间线与详情页的实际点击路径、移动端布局。
- `go test` 与浏览器走查未执行；`note_edits` 已加入测试库 TRUNCATE 列表。

## 12AO. 批次：云盘文件编辑（2026-11-06）

**工作方式**：目标模式第二十五波，继续只写代码、验证到编译层。施工依据
`docs/research/2026-11-06-firefish-port-wave25-contract.md`。

### 12AO.1 落地清单

| 层 | 文件 | 说明 |
|----|------|------|
| 迁移 | `internal/db/db.go` | v16：`drive_files.comment` |
| 服务层 | `internal/media/drive.go` | `FileUpdate`/`Update`：名称、描述、敏感、移动文件夹 |
| 端点 | `internal/httpapi/drive.go`、`server.go` | `drive/files/update`；`packDriveFile` 返回真实 `comment` |
| 前端类型 | `api/types.ts`、`endpoints.ts` | `comment: string \| null`、新端点 |
| 网盘 | `MkDrive.file.vue`、`MkDrive.folder.vue`、`MkDrive.vue` | 重命名/描述/敏感、拖拽移动、敏感角标、原地刷新 |
| 文档 | `api-contract/drive-media.md` | 端点小节与总览 |

### 12AO.2 有意的限制

1. 面包屑不接受文件拖放；发帖框附件不做重命名。
2. 无 URL 上传、去重查询与批量移动。

### 12AO.3 验证证据（编译层）

```
gofmt -l internal/ cmd/                 # 空
go vet ./...                            # exit 0
cd web && pnpm exec vue-tsc --noEmit    # exit 0
cd web && pnpm run check:port           # hardTotal 0
cd web && pnpm build                    # exit 0
```

### 12AO.4 仍未验证

- 重命名/描述/敏感标记的保存与回读，移动文件夹后列表的变化。
- 拖拽在真实浏览器中的 dataTransfer 类型与 folder 高亮。
- 他人文件夹 400、他人文件 404 的分支。
- `go test` 与浏览器走查未执行。

## 12AP. 批次：隐藏话题标签（2026-11-06）

**工作方式**：目标模式第二十六波，继续只写代码、验证到编译层。施工依据
`docs/research/2026-11-06-firefish-port-wave26-contract.md`。

### 12AP.1 落地清单

| 层 | 文件 | 说明 |
|----|------|------|
| 实例设置 | `internal/httpapi/instance_settings.go` | `hiddenTags` 键、admin/meta 回读、update-meta 规范化与校验 |
| 趋势 | `internal/note/hashtag_trend.go`、`internal/httpapi/hashtags.go` | `Trend` 增加排除列表，SQL 用 `<> ALL` |
| 前端 | `api/types.ts`、`endpoints.ts`、`pages/AdminHashtags.vue`、`pages/AdminIndex.vue` | 真实编辑页与菜单入口 |
| 文档 | `admin-emoji-registry.md`、`meta-federation.md` | hiddenTags 契约；serverRules 行改为真实值 |

### 12AP.2 有意的限制

1. 只过滤趋势，不影响标签页/搜索/探索列表。
2. 不做话题标签语法校验。
3. 即时 SQL，无缓存。

### 12AP.3 验证证据（编译层）

```
gofmt -l internal/ cmd/                 # 空
go vet ./...                            # exit 0
cd web && pnpm exec vue-tsc --noEmit    # exit 0
cd web && pnpm run check:port           # hardTotal 0
cd web && pnpm build                    # exit 0
```

### 12AP.4 仍未验证

- 保存隐藏标签后 `admin/meta` 回读与趋势列表的变化。
- 去 `#`、小写、去重与超长/超量 400 分支。
- 管理页在移动端的布局。
- `go test` 与浏览器走查未执行。

## 12AQ. 批次：实例信息页（2026-11-06）

**工作方式**：目标模式第二十七波，继续只写代码、验证到编译层。施工依据
`docs/research/2026-11-06-firefish-port-wave27-contract.md`。

### 12AQ.1 落地清单

| 层 | 文件 | 说明 |
|----|------|------|
| 端点 | `internal/httpapi/federation_instance.go`（新）、`server.go` | `federation/show-instance`：本地设置+计数，远端库内聚合，不联网 |
| 前端类型 | `api/types.ts`、`endpoints.ts` | `FederationInstance`、端点 |
| 页面 | `pages/InstanceInfo.vue` | 主机查询、软件/注册/计数展示、未知主机与 401 提示 |
| 文档 | `api-contract/meta-federation.md`、台账 | 端点小节与计数（118/111/329） |

### 12AQ.2 有意的限制

1. 远端信息不联网抓取，只来自本库已收录的行。
2. 远端软件字段恒 `null`，状态标志恒 `false`。
3. 页面不显示实例屏蔽/暂停状态。

### 12AQ.3 验证证据（编译层）

```
gofmt -l internal/ cmd/                 # 空
go vet ./...                            # exit 0
cd web && pnpm exec vue-tsc --noEmit    # exit 0
cd web && pnpm run check:port           # hardTotal 0
cd web && pnpm build                    # exit 0
```

### 12AQ.4 仍未验证

- 本机与远端主机的查询结果、未知主机返回 `null`、匿名 401。
- `host` 归一化（scheme/路径/大小写）的实际输入。
- 页面路由替换与移动端布局。
- `go test` 与浏览器走查未执行。

## 12AR. 批次：网盘用量端点（2026-11-06）

**工作方式**：目标模式第二十八波，继续只写代码、验证到编译层。施工依据
`docs/research/2026-11-06-firefish-port-wave28-contract.md`。

### 12AR.1 落地清单

| 层 | 文件 | 说明 |
|----|------|------|
| 媒体层 | `internal/media/drive.go` | `Usage`：`sum(size)` 与行数 |
| 端点 | `internal/httpapi/drive_usage.go`（新）、`server.go` | `drive` 返回 `{capacity:0, usage, filesCount}` |
| 前端 | `api/types.ts`、`endpoints.ts`、`pages/SettingsDrive.vue` | 一次请求显示容量/用量/文件数；两个敏感偏好改为真实开关 |
| 文档 | `drive-media.md`、台账 | `drive` 小节与计数（119/112/328） |

### 12AR.2 有意的限制

1. 无实例配额，`capacity` 恒 0。
2. 无按天统计图表。
3. `filesCount` 是额外字段。

### 12AR.3 验证证据（编译层）

```
gofmt -l internal/ cmd/                 # 空
go vet ./...                            # exit 0
cd web && pnpm exec vue-tsc --noEmit    # exit 0
cd web && pnpm run check:port           # hardTotal 0
cd web && pnpm build                    # exit 0
```

### 12AR.4 仍未验证

- 用量/文件数与 `drive/files` 实际行数的一致性。
- 容量 0 的展示与两个敏感开关的保存/回读。
- `go test` 与浏览器走查未执行。

## 12AS. 批次：关于页（2026-11-06）

**工作方式**：目标模式第二十九波，前端页面与文档，无后端改动。施工依据
`docs/research/2026-11-06-firefish-port-wave29-contract.md`。

### 12AS.1 落地清单

| 文件 | 说明 |
|------|------|
| `pages/AboutFirefish.vue` | 实例名/版本、定位说明、参考项目链接、技术栈、AGPL 许可与致谢；替换 TODO 占位 |
| `docs/research/2026-11-06-firefish-port-wave29-contract.md` | 本波施工契约 |

### 12AS.2 有意的限制

1. 不显示捐赠入口与贡献者名单（不冒充 Firefish）。
2. 源码仓库链接只在实例设置 `repositoryUrl` 非空时出现。
3. 版本号来自 `meta` 的 `0.1.0`，没有构建元数据。

### 12AS.3 验证证据（编译层）

```
cd web && pnpm exec vue-tsc --noEmit    # exit 0
cd web && pnpm run check:port           # hardTotal 0
cd web && pnpm build                    # exit 0
```

### 12AS.4 仍未验证

- 页面在桌面/移动端的布局、外链点击与返回按钮。
- 未执行浏览器走查。

## 12AT. 批次：账号导入（2026-11-06）

**工作方式**：目标模式第三十波，继续只写代码、验证到编译层。施工依据
`docs/research/2026-11-06-firefish-port-wave30-contract.md`。

### 12AT.1 落地清单

| 层 | 文件 | 说明 |
|----|------|------|
| 端点 | `internal/httpapi/account_import.go`（新）、`server.go` | `i/import`：读云盘 JSON，事务内恢复笔记/关注/收藏，返回计数 |
| 前端类型 | `api/types.ts`、`endpoints.ts` | `ImportSummary`、`i/import` |
| 页面 | `pages/SettingsImportExport.vue` | 文件选择 → 上传 → 导入 → 展示导入/跳过计数 |
| 文档 | `auth-users.md`、台账 | 端小节与计数（120/113/327） |

### 12AT.2 有意的限制

1. 只接受 `cinderfish-account-export@1`，不支持 Misskey zip。
2. 不导入文件字节，附件 id 可能悬空。
3. 不导入通知、注册表、私信；无限流。

### 12AT.3 验证证据（编译层）

```
gofmt -l internal/ cmd/                 # 空
go vet ./...                            # exit 0
cd web && pnpm exec vue-tsc --noEmit    # exit 0
cd web && pnpm run check:port           # hardTotal 0
cd web && pnpm build                    # exit 0
```

### 12AT.4 仍未验证

- 导出文件重新导入后的笔记/关注/收藏数量与内容。
- 冲突跳过、目标缺失、回复/转发目标不存在时的置空行为。
- 非法格式与超大文件的 400 分支。
- `go test` 与浏览器走查未执行。

## 12AU. 批次：实例图标、横幅与主题色（2026-11-06）

**工作方式**：目标模式第三十一波，继续只写代码、验证到编译层。施工依据
`docs/research/2026-11-06-firefish-port-wave31-contract.md`。

### 12AU.1 落地清单

| 层 | 文件 | 说明 |
|----|------|------|
| 实例设置 | `internal/httpapi/instance_settings.go` | 三个新键、admin/meta 回读、update-meta 校验 |
| 公开 meta | `internal/httpapi/meta.go` | iconUrl/bannerUrl/themeColor 改用运行时值 |
| 前端 | `api/types.ts`、`endpoints.ts`、`pages/AdminSettings.vue` | 三个输入与保存 |
| 文档 | `admin-emoji-registry.md`、`meta-federation.md` | 字段契约 |

### 12AU.2 有意的限制

1. 只接受 URL，不做上传/裁剪。
2. 主题色不改变前端主题选择。
3. 其余图片字段仍为常量。

### 12AU.3 验证证据（编译层）

```
gofmt -l internal/ cmd/                 # 空
go vet ./...                            # exit 0
cd web && pnpm exec vue-tsc --noEmit    # exit 0
cd web && pnpm run check:port           # hardTotal 0
cd web && pnpm build                    # exit 0
```

### 12AU.4 仍未验证

- 保存后 `meta`/`admin/meta` 的回读与实例图标/横幅的显示。
- 非法 URL/颜色的 400 分支。
- `go test` 与浏览器走查未执行。

## 12AV. 批次：已收录实例列表（2026-11-06）

**工作方式**：目标模式第三十二波，继续只写代码、验证到编译层。施工依据
`docs/research/2026-11-06-firefish-port-wave32-contract.md`。

### 12AV.1 落地清单

| 层 | 文件 | 说明 |
|----|------|------|
| 端点 | `internal/httpapi/federation_instance.go`、`server.go` | `federation/instances` 按 users.host 聚合 |
| 前端 | `api/endpoints.ts`、`pages/AdminFederation.vue` | 「已收录实例」表格与加载状态 |
| 文档 | `meta-federation.md`、台账 | 端点小节与计数（121/114/326） |

### 12AV.2 有意的限制

1. 只统计库内已出现的主机。
2. 无投递状态、pubSub、软件信息。
3. 无分页 UI。

### 12AV.3 验证证据（编译层）

```
gofmt -l internal/ cmd/                 # 空
go vet ./...                            # exit 0
cd web && pnpm exec vue-tsc --noEmit    # exit 0
cd web && pnpm run check:port           # hardTotal 0
cd web && pnpm build                    # exit 0
```

### 12AV.4 仍未验证

- 列表排序、host 过滤与空库表现。
- 表格在移动端与实例信息页跳转。
- `go test` 与浏览器走查未执行。

## 12AW. 批次：笔记「已编辑」标记（2026-11-06）

**工作方式**：目标模式第三十三波，继续只写代码、验证到编译层。施工依据
`docs/research/2026-11-06-firefish-port-wave33-contract.md`。

### 12AW.1 落地清单

| 层 | 文件 | 说明 |
|----|------|------|
| 服务层 | `internal/note/service.go` | `Note.UpdatedAt`、`noteSelect`/`scanNote` 读取 `updated_at` |
| 打包 | `internal/httpapi/notes_pack.go` | `updatedAt`（RFC3339Nano 或 null） |
| 前端 | `api/types.ts`、`components/MkNoteHeader.vue` | 类型字段与铅笔标记（title 显示编辑时间） |
| 文档 | `api-contract/notes-core.md` | 字段与副作用说明 |

### 12AW.2 有意的限制

1. 仅本地编辑设置 `updatedAt`，远端 Update 未处理。
2. 无单次编辑详情入口。
3. 其余 locale 未生成。

### 12AW.3 验证证据（编译层）

```
gofmt -l internal/ cmd/                 # 空
go vet ./...                            # exit 0
cd web && pnpm exec vue-tsc --noEmit    # exit 0
cd web && pnpm run check:port           # hardTotal 0
cd web && pnpm build                    # exit 0
```

### 12AW.4 仍未验证

- 编辑后列表与详情页的 `updatedAt`、铅笔标记与提示时间。
- 未编辑笔记的 `updatedAt: null`。
- `go test` 与浏览器走查未执行。

## 12AX. 批次：notes/state 与收藏菜单（2026-11-06）

**工作方式**：目标模式第三十四波，继续只写代码、验证到编译层。施工依据
`docs/research/2026-11-06-firefish-port-wave34-contract.md`。

### 12AX.1 落地清单

| 层 | 文件 | 说明 |
|----|------|------|
| 服务层 | `internal/note/state.go`（新） | `State`：收藏、置顶、线程静音 |
| 端点 | `internal/httpapi/note_state.go`（新）、`server.go` | `notes/state` |
| 前端 | `api/types.ts`、`endpoints.ts`、`scripts/get-note-menu.ts`、三个笔记组件 | `buildNoteMenu` 先取状态；收藏/置顶按真实状态渲染 |
| 文档 | `notes-core.md`、台账 | 端点小节与计数（122/115/325） |

### 12AX.2 有意的限制

1. 每次打开菜单多一次 RPC，失败静默退回。
2. 操作后不就地刷新状态。
3. 线程静音恒 `false`。

### 12AX.3 验证证据（编译层）

```
gofmt -l internal/ cmd/                 # 空
go vet ./...                            # exit 0
cd web && pnpm exec vue-tsc --noEmit    # exit 0
cd web && pnpm run check:port           # hardTotal 0
cd web && pnpm build                    # exit 0
```

### 12AX.4 仍未验证

- 收藏/取消收藏与置顶项的显示切换。
- 未登录、请求失败与笔记不存在时的退回行为。
- `go test` 与浏览器走查未执行。

## 12AY. 批次：从 URL 上传（2026-11-06）

**工作方式**：目标模式第三十五波，继续只写代码、验证到编译层。施工依据
`docs/research/2026-11-06-firefish-port-wave35-contract.md`。

### 12AY.1 落地清单

| 层 | 文件 | 说明 |
|----|------|------|
| 媒体层 | `internal/media/drive.go` | `MaxSize()` 访问器 |
| 端点 | `internal/httpapi/drive_url_upload.go`（新）、`server.go` | 带 SSRF 防护的下载上传 |
| 前端 | `api/endpoints.ts`、`components/MkDrive.vue` | 菜单「从 URL 上传」与列表插入 |
| 文档 | `drive-media.md`、台账 | 端点小节与计数（123/116/324） |

### 12AY.2 有意的限制

1. 无进度/取消。
2. 不跟随非 http(s) 重定向、不代理 Cookie。
3. 不做内容类型白名单。

### 12AY.3 验证证据（编译层）

```
gofmt -l internal/ cmd/                 # 空
go vet ./...                            # exit 0
cd web && pnpm exec vue-tsc --noEmit    # exit 0
cd web && pnpm run check:port           # hardTotal 0
cd web && pnpm build                    # exit 0
```

### 12AY.4 仍未验证

- 公网 URL 下载、私有地址与重定向到私有地址的拒绝。
- 超大响应、非 2xx 与文件夹归属错误的分支。
- 菜单输入与列表插入。
- `go test` 与浏览器走查未执行。

## 12AZ. 批次：公告已读状态（2026-11-06）

**工作方式**：目标模式第三十六波，继续只写代码、验证到编译层。施工依据
`docs/research/2026-11-06-firefish-port-wave36-contract.md`。

### 12AZ.1 落地清单

| 层 | 文件 | 说明 |
|----|------|------|
| 迁移 | `internal/db/db.go` | v17：`announcement_reads`；AGENTS schema 17；测试库 TRUNCATE |
| 服务层 | `internal/announcement/service.go` | `MarkRead`、`ReadIDs`、`UnreadCount` |
| 端点 | `internal/httpapi/announcements.go`、`server.go` | 列表填充 `isRead`、`i/read-announcement`、`hasUnreadAnnouncement` |
| 前端 | `api/endpoints.ts`、`pages/Announcements.vue` | 标记已读、未读强调线、清理占位说明 |
| 文档 | `admin-emoji-registry.md`、台账 | 端点与字段说明（124/117/323） |

### 12AZ.2 有意的限制

1. 打开页面即标记当前页，不做视口级标记。
2. 修改公告不重置已读。
3. 无「全部已读」按钮。

### 12AZ.3 验证证据（编译层）

```
gofmt -l internal/ cmd/                 # 空
go vet ./...                            # exit 0
cd web && pnpm exec vue-tsc --noEmit    # exit 0
cd web && pnpm run check:port           # hardTotal 0
cd web && pnpm build                    # exit 0
```

### 12AZ.4 仍未验证

- 已读写入、列表 `isRead` 与 `hasUnreadAnnouncement` 的联动。
- 匿名列表仍为 `isRead:false`。
- `go test` 与浏览器走查未执行。

## 12BA. 批次：公告编辑（2026-11-06）

**工作方式**：目标模式第三十七波，继续只写代码、验证到编译层。施工依据
`docs/research/2026-11-06-firefish-port-wave37-contract.md`。

### 12BA.1 落地清单

| 层 | 文件 | 说明 |
|----|------|------|
| 服务层 | `internal/announcement/service.go` | `UpdateParams`（`SetExpiry` 区分「保持 / 显式 null」）与 `Update`：读-合并-整行写回 |
| 端点 | `internal/httpapi/announcements.go`、`server.go` | `admin/announcements/update`；`imageUrl`/`expiresAt` 三态；404 `NO_SUCH_ANNOUNCEMENT` |
| 前端 | `api/endpoints.ts`、`pages/AdminAnnouncements.vue` | 表单双模式、编辑回填、列表「编辑」按钮、`.itemActions` 布局 |
| 文档 | `admin-emoji-registry.md`、台账 | update 端点契约（125/118/322） |

### 12BA.2 有意的限制

1. 保存后重拉列表，不做乐观更新。
2. UI 不暴露 `isActive`（后端已支持，等将来加开关）。
3. `expiresAt` 后端不校验「晚于现在」，过去时间等于立即过期；UI 侧拦住了。

### 12BA.3 验证证据（编译层）

```
gofmt -l internal/ cmd/                 # 空
go vet ./...                            # exit 0
cd web && pnpm exec vue-tsc --noEmit    # exit 0
cd web && pnpm run check:port           # hardTotal 0（400 文件）
cd web && pnpm build                    # exit 0
```

### 12BA.4 仍未验证

- 编辑保存后的回读、清空 `imageUrl`/`expiresAt` 的实际效果。
- 404 `NO_SUCH_ANNOUNCEMENT` 与 403 分支。
- `go test` 与浏览器走查未执行。

## 12BB. 批次：邮件发送与邮箱验证（2026-11-06）

**工作方式**：目标模式第三十八波，继续只写代码、验证到编译层。施工依据
`docs/research/2026-11-06-firefish-port-wave38-contract.md`。

### 12BB.1 落地清单

| 层 | 文件 | 说明 |
|----|------|------|
| 配置 | `internal/config/config.go`、`config.example.toml` | `[mail]` 段与环境变量；host 空即日志模式；from 按域名推导 |
| 发信 | `internal/mail/mail.go` | `Sender`、`SMTPSender`（starttls/tls/none）、`LogSender`；头部注入清理 |
| 迁移 | `internal/db/db.go` | v18：`email_challenges` + `user_profiles.email_verified`；AGENTS schema 18；测试库 TRUNCATE |
| 服务层 | `internal/account/email.go` | 校验/占用/写入/发码/消费/节流；`ResetPassword` |
| 打包 | `internal/account/profile.go`、`httpapi/users_pack.go` | `MeDetailed.email`、`emailVerified` 读真值 |
| 端点 | `internal/httpapi/email.go`、`server.go` | 6 个路由：available/update-email/email-verification/verify-email/request-reset/reset |
| 文档 | `auth-users.md`、台账 | 邮箱小节 + 契约（131/124/316） |

### 12BB.2 有意的限制

1. 注册不收邮箱，`emailRequiredForSignup` 仍为 false。
2. 邮件正文固定中文纯文本，无 HTML/多语言模板。
3. 只有「同账号同用途 60 秒」节流，没有按 IP 配额。
4. 重置密码不吊销已签发 token（与 `i/change-password` 一致）。
5. `i/update-email` 发信失败时保留新地址并返回 500。

### 12BB.3 验证证据（编译层）

```
gofmt -l internal/ cmd/                 # 空
go vet ./...                            # exit 0（含测试文件编译）
```

本波未改前端，`vue-tsc`/`check:port`/`pnpm build` 不适用。

### 12BB.4 仍未验证

- SMTP 真投递（三种 TLS 模式、AUTH、超时）从未对真实服务器执行。
- 验证链接、重置链接的端到端（点开链接 → 标记/改密）。
- 边界：他人链接、地址变更后旧链接、过期链接、重放、60 秒节流。
- `MeDetailed.email` 在原生 token 与 OAuth token 两种请求下的差异。
- `go test`（含数据库）与浏览器走查未执行。

## 12BC. 批次：邮箱前端页面（2026-11-06）

**工作方式**：目标模式第三十九波，继续只写代码、验证到编译层。施工依据
`docs/research/2026-11-06-firefish-port-wave39-contract.md`。

### 12BC.1 落地清单

| 层 | 文件 | 说明 |
|----|------|------|
| 类型 | `api/endpoints.ts`、`api/types.ts` | 六个端点与 `EmailAddressAvailability` |
| 设置页 | `pages/SettingsEmail.vue` | 改邮箱（弹密码框）、验证徽标、重发验证邮件；通知开关标注无存储 |
| 验证页 | `pages/VerifyEmail.vue` | 自动核销 code；未登录先引导登录；成功后刷新 `me` |
| 重置页 | `pages/ResetPassword.vue` | 带 token 改密 / 不带 token 内联发起重置 |
| 入口 | `account/components/CfSignin.vue` | 「忘记密码」链接 |
| 文案 | `i18n/locales/{zh-CN,ja-JP}.json` | 新增 `unverified` |
| 注释 | `pages/SettingsSecurity.vue`、`components/MkPostForm.vue` | 修掉「没有端点」的过时说明 |

### 12BC.2 有意的限制

1. 邮件通知开关不渲染（`i/update` 不存这两个字段）。
2. `i/update-email` 发信失败时地址已保存，页面按普通错误显示，可点重发。
3. 重置入口是内联表单，不是 Firefish 的弹窗。
4. 验证页要求已登录。

### 12BC.3 验证证据（编译层）

```
cd web && pnpm exec vue-tsc --noEmit    # exit 0
cd web && pnpm run check:port           # hardTotal 0（400 文件）
cd web && pnpm build                    # exit 0
```

### 12BC.4 仍未验证

- 三个页面的实际渲染与交互（桌面/移动、错误分支、节流 429 的提示）。
- 忘记密码 → 收信 → 重置的完整链路。
- 浏览器走查与 `go test` 未执行。

## 12BD. 批次：出站投递与取消转贴（2026-11-06）

**工作方式**：目标模式第四十波。本波起点是「转贴不能取消」，读代码时发现更严重的问题：
`DeliverNote` / `DeliverDelete` 只有定义没有调用点，本地笔记从未投递。施工依据
`docs/research/2026-11-06-firefish-port-wave40-contract.md`。

### 12BD.1 落地清单

| 层 | 文件 | 说明 |
|----|------|------|
| 投递 | `internal/httpapi/federation_wire.go` | `deliverable`/`deliveryInboxes`/`objectRef`；`DeliverNote` 分 `Create`/`Announce`；`DeliverDelete` 分 `Delete`/`Undo(Announce)`；直发只投可见用户 |
| 接线 | `internal/httpapi/notes.go` | `notes/create`、`notes/delete` 各调一次；新增 `handleNotesUnrenote` |
| 存储 | `internal/note/renote.go` | `PureRenoteOf`（找调用者的纯转贴，引用不算） |
| 端点 | `internal/httpapi/server.go` | 注册 `notes/unrenote` |
| 前端 | `components/MkRenoteButton.vue`、`api/endpoints.ts` | 会话内转过贴后显示「取消转发」 |
| 文档 | `notes-core.md`、台账 | 投递矩阵与 unrenote 语义（132/125/315） |

### 12BD.2 有意的限制

1. `notes/edit` 仍不发 `Update`。
2. 远端原帖缺 `uri` 时删除降级为 `Delete` 转贴本身。
3. 投递失败只记日志，不回滚请求。
4. 前端只能知道「本次会话内转过」，没有逐笔记的转贴查询。

### 12BD.3 验证证据（编译层）

```
gofmt -l internal/ cmd/                 # 空
go vet ./...                            # exit 0
cd web && pnpm exec vue-tsc --noEmit    # exit 0
cd web && pnpm run check:port           # hardTotal 0（400 文件）
cd web && pnpm build                    # exit 0
```

### 12BD.4 仍未验证

- 真实双实例：Create/Announce/Delete/Undo 的签名、收件箱去重与远端反应。
- 直发笔记不投关注者这一点没有用例保证。
- `notes/unrenote` 找到/找不到两条分支。
- `go test` 与浏览器走查未执行。

## 12BE. 批次：编辑的联邦 Update（2026-11-06）

**工作方式**：目标模式第四十一波，继续只写代码、验证到编译层。施工依据
`docs/research/2026-11-06-firefish-port-wave41-contract.md`。

### 12BE.1 落地清单

| 层 | 文件 | 说明 |
|----|------|------|
| 渲染 | `internal/federation/render.go` | `RenderUpdate`：活动 id 带毫秒时间戳，继承 to/cc |
| 出站 | `internal/httpapi/federation_wire.go`、`note_edit.go` | `DeliverUpdate`（纯转贴跳过）；`notes/edit` 成功后调用 |
| 入站 | `internal/federation/inbox.go` | `inbox.update` 按对象类型分流；新增 `updateNote`（防伪检查 + 只改内容） |
| 存储 | `internal/federation/store.go` | `UpdateRemoteNote`：按 ap_id 更新正文/CW/敏感标记/updated_at |
| 文档 | `notes-core.md`、`meta-federation.md` | 出站 Update 与入站分流说明 |

### 12BE.2 有意的限制

1. 入站只更新正文/CW/敏感标记/时间，不重算标签、提及、表情。
2. 入站编辑不推 Streaming。
3. Mastodon 忽略 status `Update`，双向不完全对称。
4. 未新增测试。

### 12BE.3 验证证据（编译层）

```
gofmt -l internal/ cmd/   # 空
go vet ./...              # exit 0
```

本波未改前端。

### 12BE.4 仍未验证

- 双实例：编辑后对端正文是否真的更新、第二次编辑是否被去重。
- 入站 `attributedTo`/主机不匹配的拒绝分支。
- 远端已软删除笔记收到 Update 时的 `skip` 分支。
- `go test` 与浏览器走查未执行。

## 12BF. 批次：按活动作者签名（2026-11-06）

**工作方式**：目标模式第四十二波。施工依据
`docs/research/2026-11-06-firefish-port-wave42-contract.md`；修复 `meta-federation.md`
「已知限制」第 8 条。

### 12BF.1 落地清单

| 层 | 文件 | 说明 |
|----|------|------|
| 客户端 | `internal/federation/client.go` | `KeySource` 带 `userID`；`PostActivity` 带 `asUserID`；`sign` 透传 |
| 投递 | `internal/federation/deliver.go` | 用队列行的 `dl.UserID` 签名 |
| 入站 | `internal/federation/inbox.go` | `Accept` 用接受者密钥 |
| 装配 | `internal/httpapi/federation_wire.go` | 按用户取密钥；空 userID 回落实例密钥 |
| 文档 | `meta-federation.md` | 第 8 条限制改为已修 |

### 12BF.2 有意的限制

1. 实例密钥仍是第一个管理员，没有独立 instance actor 文档。
2. `KeyID` 格式不变。
3. 未新增测试。

### 12BF.3 验证证据（编译层）

```
gofmt -l internal/ cmd/   # 空
go vet ./...              # exit 0
```

### 12BF.4 仍未验证

- 双实例：第二个用户发帖是否被对端接受（签名密钥与 actor 一致）。
- 入站 `Accept` 的签名方是否为接受者。
- `go test` 与浏览器走查未执行。

## 12BG. 批次：四个兼容端点（2026-11-06）

**工作方式**：目标模式第四十三波，继续只写代码、验证到编译层。施工依据
`docs/research/2026-11-06-firefish-port-wave43-contract.md`。

### 12BG.1 落地清单

| 层 | 文件 | 说明 |
|----|------|------|
| 统计 | `internal/account/stats.go` | `Stats`：五条聚合查询（笔记分桶、被回复/被转贴、投票两向） |
| 端点 | `internal/httpapi/users.go`、`hashtags.go`、`registry.go`、`notes.go` | `users/stats`、`hashtags/search`、`i/registry/keys-with-type`、`notes/featured` |
| 查询 | `internal/note/hashtag_search.go`、`featured.go` | 标签搜索与推荐排序 |
| 前端 | `api/endpoints.ts`、`api/types.ts`、`pages/SettingsAccountInfo.vue` | 四个端点类型；账号信息页改用真实统计，移除过时的「不可用」项 |
| 文档 | `notes-core.md`、`timelines-notifications.md`、`admin-emoji-registry.md`、`auth-users.md`、台账 | 契约与计数（136/129/311） |

### 12BG.2 有意的限制

1. `hashtags/search` 忽略 `sort`。
2. `notes/featured` 有 30 天窗口。
3. `keys-with-type` 的类型是读时推断。
4. 四个端点均未新增测试。

### 12BG.3 验证证据（编译层）

```
gofmt -l internal/ cmd/                 # 空
go vet ./...                            # exit 0
cd web && pnpm exec vue-tsc --noEmit    # exit 0
cd web && pnpm run check:port           # hardTotal 0（400 文件）
cd web && pnpm build                    # exit 0
```

### 12BG.4 仍未验证

- 四个端点的返回值（计数一致性、标签排序、类型推断、推荐排序）。
- 账号信息页的实际渲染与统计失败时的降级。
- `go test` 与浏览器走查未执行。

## 12BH. 批次：实例 CSS 与邮件设置页（2026-11-06）

**工作方式**：目标模式第四十四波，继续只写代码、验证到编译层。施工依据
`docs/research/2026-11-06-firefish-port-wave44-contract.md`。

### 12BH.1 落地清单

| 层 | 文件 | 说明 |
|----|------|------|
| 实例设置 | `internal/httpapi/instance_settings.go` | `CustomCSS` 字段、`customCss` 键、`admin/update-meta` 接受（≤100 KiB）、`admin/meta` 返回 `customCss`/`emailServer` |
| meta | `internal/httpapi/meta.go` | `customCss`；`enableEmail` 按配置计算；详细 meta 的 `emailServer` |
| 前端注入 | `composables/use-custom-css.ts` | 实例级 `<style>`，排在账号级之前 |
| 页面 | `pages/AdminCustomCss.vue`、`pages/AdminEmailSettings.vue` | 实例 CSS 编辑器；邮件状态只读页 |
| 路由/菜单 | `router/routes.ts`、`pages/AdminIndex.vue` | `/admin/custom-css` 与两个菜单项 |
| 类型 | `api/types.ts`、`api/endpoints.ts` | `MetaDetailed.customCss`/`emailServer`、`AdminInstanceMeta` 同名字段、`customCss` 请求参数 |

### 12BH.2 有意的限制

1. 实例 CSS 无语法校验、无版本历史。
2. 实例 CSS 对未登录访客生效（已在页面警示）。
3. 邮件设置只读，没有运行期 SMTP 重载。
4. 未新增后端路由。

### 12BH.3 验证证据（编译层）

```
gofmt -l internal/ cmd/                 # 空
go vet ./...                            # exit 0
cd web && pnpm exec vue-tsc --noEmit    # exit 0
cd web && pnpm run check:port           # hardTotal 0（400 文件）
cd web && pnpm build                    # exit 0
```

### 12BH.4 仍未验证

- 实例 CSS 保存 → 下发 → 注入 → 账号样式覆盖的完整链路。
- 100 KiB 上限与空串清除两个分支。
- 邮件状态页在已配置/未配置两种实例下的呈现。
- `go test` 与浏览器走查未执行。

## 12BI. 批次：管理台文件详情（2026-11-06）

**工作方式**：目标模式第四十五波，继续只写代码、验证到编译层。施工依据
`docs/research/2026-11-06-firefish-port-wave45-contract.md`。

### 12BI.1 落地清单

| 层 | 文件 | 说明 |
|----|------|------|
| 端点 | `internal/httpapi/admin_drive.go`、`server.go` | `admin/drive/show-file`：按 id 取任意用户文件 + 所属账号；404 `NO_SUCH_FILE` |
| 页面 | `pages/AdminFile.vue` | 类型分流预览、字段表、上传者链接、只读原因说明 |
| 入口 | `pages/AdminFiles.vue` | 每行「详情」链接 |
| 类型 | `api/endpoints.ts` | `adminDriveShowFile` |
| 文档 | `admin-emoji-registry.md`、台账 | 端点契约（137/130/310） |

### 12BI.2 有意的限制

1. 没有删除/批量清理（无端点）。
2. 不显示上传 IP、原始请求、远端缓存状态。
3. 缩略图仍为 `null`，预览用原图。

### 12BI.3 验证证据（编译层）

```
gofmt -l internal/ cmd/                 # 空
go vet ./...                            # exit 0
cd web && pnpm exec vue-tsc --noEmit    # exit 0
cd web && pnpm run check:port           # hardTotal 0（400 文件）
cd web && pnpm build                    # exit 0
```

### 12BI.4 仍未验证

- 详情页在图片/视频/音频/其它四类文件下的渲染。
- 上传者已删除账号时的 `user: null` 分支。
- `go test` 与浏览器走查未执行。

## 12BJ. 批次：缩略图管道（2026-11-06）

**工作方式**：目标模式第四十六波，继续只写代码、验证到编译层。施工依据
`docs/research/2026-11-06-firefish-port-wave46-contract.md`。

### 12BJ.1 落地清单

| 层 | 文件 | 说明 |
|----|------|------|
| 生成 | `internal/media/thumbnail.go` | 箱式滤波降采样（≤512px）、解码前限 4000 万像素、JPEG q82 / PNG |
| 存储 | `internal/media/drive.go` | `storeThumbnail`（尽力而为）、`thumbnailURL`、`Delete` 顺带删缩略图 |
| 下发 | `internal/httpapi/drive.go` | `packDriveFile` 的 `thumbnailUrl`；`/media` 按扩展名给 Content-Type |
| 前端 | `api/types.ts`、`MkDriveFileThumbnail.vue`、`MkMedia.vue`、`UserPhotos.vue` | 优先用缩略图，回退原图 |
| 文档 | `drive-media.md` | `thumbnailUrl` 语义与限制更新 |

### 12BJ.2 有意的限制

1. 没有 WebP（Go 标准库无编码器）。
2. 没有 `properties` 宽高与 `blurhash`。
3. 历史文件不补生成缩略图。
4. 未加测试。

### 12BJ.3 验证证据（编译层）

```
gofmt -l internal/ cmd/                 # 空
go vet ./...                            # exit 0
cd web && pnpm exec vue-tsc --noEmit    # exit 0
cd web && pnpm run check:port           # hardTotal 0（400 文件）
cd web && pnpm build                    # exit 0
```

### 12BJ.4 仍未验证

- 真实图片上传后的缩略图尺寸、体积与透明保留。
- 非图片、损坏图片、超大像素图三条跳过/失败分支。
- 删除文件时缩略图对象是否真的消失。
- `go test` 与浏览器走查未执行。

## 12BK. 批次：锁推生效与关注通知（2026-11-06）

**工作方式**：目标模式第四十七波，继续只写代码、验证到编译层。施工依据
`docs/research/2026-11-06-firefish-port-wave47-contract.md`。

### 12BK.1 落地清单

| 层 | 文件 | 说明 |
|----|------|------|
| 锁推 | `internal/httpapi/following.go`、`internal/federation/store.go` | `LocalUserLocked`；本地目标按锁推分流成待审请求 |
| 通知 | `internal/httpapi/following.go`、`internal/federation/inbox.go` | `Server.createNotification`；入站两条路径写通知；`Inbox.OnNotification`/`OnFollow` 接线 |
| 资料 | `internal/account/profile.go`、`internal/httpapi/i_update.go`、`users_pack.go` | `Profile.IsLocked`（标量子查询）、`SetLocked`、`i/update {isLocked}`、`isLocked` 真值打包 |
| 前端 | `pages/SettingsPrivacy.vue`、`account/types.ts` | 七个真实开关；`ProfileUpdateRequest.isLocked` |
| 文档 | `auth-users.md`、`following-relations.md` | 三条旧结论更新 |

### 12BK.2 有意的限制

1. 无 `followRequestAccepted` 通知。
2. 接受请求不清理此前的待审通知。
3. 锁推变更不主动重发 actor 文档，远端缓存需自行刷新。
4. 未新增测试。

### 12BK.3 验证证据（编译层）

```
gofmt -l internal/ cmd/                 # 空
go vet ./...                            # exit 0
cd web && pnpm exec vue-tsc --noEmit    # exit 0
cd web && pnpm run check:port           # hardTotal 0（400 文件）
cd web && pnpm build                    # exit 0
```

### 12BK.4 仍未验证

- 锁推账号被关注 → 待审 → 接受/拒绝的完整链路。
- 三类通知行的落库与 Streaming 推送。
- 隐私页七个开关的保存与回读。
- `go test` 与浏览器走查未执行。

## 12BL. 批次：邮件通知（2026-11-06）

**工作方式**：目标模式第四十八波，继续只写代码、验证到编译层。施工依据
`docs/research/2026-11-06-firefish-port-wave48-contract.md`。

### 12BL.1 落地清单

| 层 | 文件 | 说明 |
|----|------|------|
| 存储 | `internal/account/profile*.go` | `EmailNotificationTypes` / `ReceiveAnnouncementEmail`（settings）；`AnnouncementMailRecipients` |
| 入参 | `internal/httpapi/i_update.go` | 两个字段 + 类型白名单校验 |
| 打包 | `internal/httpapi/users_pack.go` | 两个字段读真值 |
| 发送 | `internal/httpapi/notification_mail.go`、`notes.go`、`following.go` | `maybeSendNotificationMail`（后台、30 秒、失败只记日志）；四条通知路径接入 |
| 公告 | `internal/httpapi/announcements.go` | `mailAnnouncement` 群发（2 分钟） |
| 前端 | `pages/SettingsEmail.vue`、`account/types.ts` | 五个真实开关；`ProfileUpdateRequest` 两字段 |
| 文档 | `auth-users.md` | 字段与发送规则 |

### 12BL.2 有意的限制

1. 纯文本中文正文，无 HTML/多语言/退订头。
2. 不发转发、引用、反应邮件。
3. 公告群发串行逐封。
4. 无发送重试队列。
5. 未新增测试。

### 12BL.3 验证证据（编译层）

```
gofmt -l internal/ cmd/                 # 空
go vet ./...                            # exit 0
cd web && pnpm exec vue-tsc --noEmit    # exit 0
cd web && pnpm run check:port           # hardTotal 0（400 文件）
cd web && pnpm build                    # exit 0
```

### 12BL.4 仍未验证

- 真实投递：提及/回复/关注/关注请求四类邮件。
- 未验证邮箱、未订阅、类型未勾选三种不发送分支。
- 公告群发的收件人筛选与并发行为。
- `go test` 与浏览器走查未执行。

## 12BM. 批次：无后端的管理页转说明页（2026-11-06）

**工作方式**：目标模式第四十九波，只改前端与文档。施工依据
`docs/research/2026-11-06-firefish-port-wave49-contract.md`。

### 12BM.1 落地清单

| 层 | 文件 | 说明 |
|----|------|------|
| 组件 | `components/admin/AdminUnavailable.vue` | 现状说明 + 已有项（可带链接）+ 缺口清单 |
| 页面 | `pages/Admin{Experiments,Integrations,ProxyAccount,Promotions,Security,OtherSettings}.vue` | 六个占位页改为说明页 |
| 菜单 | `pages/AdminIndex.vue` | 新增「未实现的实例功能」分组，六个入口 |
| 文档 | 本契约 | 六个页面的缺口结论 |

### 12BM.2 有意的限制

1. 这些页面只有说明与跳转，没有交互。
2. 缺口判定以「没有端点或表结构」为据；实现其一后应改回真实页面并移出分组。
3. 未新增测试。

### 12BM.3 验证证据（编译层）

```
cd web && pnpm exec vue-tsc --noEmit    # exit 0
cd web && pnpm run check:port           # hardTotal 0（400 文件）
cd web && pnpm build                    # exit 0
```

### 12BM.4 仍未验证

- 六个说明页的渲染与管理台菜单分组在窄屏下的表现。
- 浏览器走查未执行。

## 12BN. 批次：状态栏（2026-11-06）

**工作方式**：目标模式第五十波，只改前端。施工依据
`docs/research/2026-11-06-firefish-port-wave50-contract.md`。本波之后目标被用户暂停，
见 §12BO。

### 12BN.1 落地清单

| 层 | 文件 | 说明 |
|----|------|------|
| 条目 | `components/statusbar/CfStatusbarFederation.vue` | 拉 `federation/instances`（30 条），CSS 跑马灯滚动主机名；刷新间隔/滚动时长/反向/着色 |
| 容器 | `components/statusbar/CfStatusbars.vue` | 五档高度、深色底、名称；未知类型就地说明 |
| 设置页 | `pages/SettingsStatusbar.vue` | 增删/上下移/名称/类型/高度/深色/联邦参数，即时写 store |
| 挂载 | `ui/layouts/UniversalLayout.vue`、`pages/SettingsIndex.vue` | 底部固定条（有配置才出现）+ 设置入口 |
| 文档 | 本契约 | rss / userList 为何不做 |

### 12BN.2 有意的限制

1. 只实现 `federation` 类型；`rss` 需要媒体代理，`userList` 属首版范围外，设置页列为禁用。
2. 排序只有上移/下移，没有拖拽。
3. 只有 Universal 布局挂载；Deck / Zen 未接。
4. 存储作用域 `deviceAccount`，不跨设备同步。
5. 未新增测试。

### 12BN.3 验证证据（编译层）

```
cd web && pnpm exec vue-tsc --noEmit    # exit 0
cd web && pnpm run check:port           # hardTotal 0（400 文件）
cd web && pnpm build                    # exit 0
```

### 12BN.4 仍未验证

- 状态栏在桌面/移动下的实际高度、跑马灯速度与遮挡。
- 增删/排序/参数改动是否即时反映在底部条上。
- 浏览器走查未执行。

## 12BO. 暂停点：2026-11-06 目标模式终止

用户于第五十波结束后要求「停止目标与流程，更新 HANDOVER」。本节记录停止时的确切状态，
供下一次接手时直接继续，不需要重新推导。

### 12BO.1 停止时的仓库状态

- 工作树干净；第五十波的改动已提交（`5609384`）。
- 台账：**137 条已注册路由 / 130 条 Misskey 同名 / 310 条未实现**，Tier 1 缺口 **0**
  （`docs/research/2026-11-04-misskey-api-coverage.md`）。
- 迁移版本 **v18**（`email_challenges` + `user_profiles.email_verified`）。
- 前端占位页（`cf-page-placeholder`）**34 个**：其中 19 个属于用户已裁决的首版范围外
  （`AdminRelays`、`AntennaTimeline`、`Channel*`×3、`Clip`、`Gallery*`×3、`Messaging*`×2、
  `MyAntennas*`×3、`MyClips*`×2、`MyGroups*`×2、`MyLists*`×2、`Page*`×3、`UserListTimeline`），
  其余 15 个见 §12BO.3。

### 12BO.2 本轮（§12BA–§12BN，第三十七至五十波）实际完成的内容

| 波 | 主题 | 关键证据 |
|----|------|----------|
| 37 | `admin/announcements/update` + 公告编辑 UI | 端点 + 页面，编译层 |
| 38 | SMTP 发送、`email_challenges`（v18）、六个邮箱端点 | 编译层 |
| 39 | 邮箱设置/验证/重置三页 + 忘记密码入口 | 编译层 |
| 40 | **修复出站投递从未接线** + `notes/unrenote` | 编译层，双实例未验证 |
| 41 | 编辑的双向 `Update` | 编译层 |
| 42 | **按活动作者签名**（原为固定管理员密钥） | 编译层 |
| 43 | `users/stats`、`hashtags/search`、`registry/keys-with-type`、`notes/featured` | 编译层 |
| 44 | 实例级自定义 CSS + 邮件状态页 | 编译层 |
| 45 | `admin/drive/show-file` + 管理台文件详情 | 编译层 |
| 46 | 缩略图生成管道（`thumbnailUrl` 不再恒 null） | 编译层 |
| 47 | 锁推生效 + 关注类通知落库 + 隐私页七个真开关 | 编译层 |
| 48 | 邮件通知（四类触发 + 公告群发） | 编译层 |
| 49 | 六个无后端管理页转说明页 | 编译层 |
| 50 | 状态栏（federation 类型） | 编译层 |

### 12BO.3 停止时仍待办的事项

**必须先做的验证**（自第十九波起全部欠账，这是恢复工作后第一优先级）：
`go test ./...`、`CINDERFISH_TEST_DATABASE_URL=… go test ./...`、真实二进制手工走查、
无头浏览器截图与交互。近期波次的高风险未验证点：SMTP 真投递、双实例
Create/Announce/Undo/Update 投递与按作者签名、缩略图生成与删除、锁推审批链路、邮件通知触发、
状态栏渲染。

**未移植且无后端**（15 个占位页）：
`Auth`、`OAuth`、`AuthorizeInteraction`、`SignupComplete`（应用授权 / Signup 确认流程，后端缺
`auth/session/*` 与邮箱确认注册）、`SettingsIntegration`（第三方登录，明确不做）、
`SettingsPlugin`、`SettingsPluginInstall`（缺 AiScript 运行时）、`SettingsCustomKatexMacro`
（缺 `katex` 依赖）、`SettingsMigration`（缺 `i/move`）、`EmptyPage`（内部空页）。

**下一波的候选顺序**（按价值/风险排序）：
1. `i/move` + `SettingsMigration`（ActivityPub Move，最后一处有联邦影响的设置）；
2. OAuth 交互式同意页（需后端把 token 从请求头改到表单，属行为变更，需先确认）；
3. AiScript 插件运行时（工作量大，需引依赖）；
4. KaTeX 自定义宏（需引 `katex`）；
5. `SettingsIntegration` 等无后端页转说明页（与第四十九波同法）。

### 12BO.4 接手清单（给下一个 agent 的第一小时）

按顺序做完这几步，就能在不需要历史对话的情况下继续：

1. **读三份文件**（其余可选）：本文件 §12BO 与「已知未验证/未移植」清单、`AGENTS.md`（项目
   规约、参考源码版本、代理与命名规则）、`docs/research/2026-11-04-misskey-api-coverage.md`
   （API 台账，判定「还缺什么」只认这张表）。
2. **确认环境**：Go 1.26；PostgreSQL 18；`GOPROXY=https://mirrors.aliyun.com/goproxy/,direct`；
   访问 GitHub 需代理 `127.0.0.1:19823`。数据库测试用
   `CINDERFISH_TEST_DATABASE_URL="postgres://cinderfish:cinderfish@localhost:5432/cinderfish_test"`
   开启，未设置时数据库用例自动跳过（**这会让「测试通过」成为假象，务必设置**）。
3. **先补验证债**，不要先写新代码：
   ```bash
   go vet ./... && go test ./...
   CINDERFISH_TEST_DATABASE_URL="postgres://cinderfish:cinderfish@localhost:5432/cinderfish_test" \
     go test ./...
   cd web && pnpm exec vue-tsc --noEmit && pnpm run check:port && pnpm build
   ```
   预期：编译层三项（vet / vue-tsc / check:port / build）应当全绿；**`go test` 是未知数**——
   第三十七至五十波只做了编译层验证，数据库用例很可能出问题（迁移到 v18、`i/update` 增字段、
   通知落库、缩略图写入、签名参数变化都可能撞到旧断言）。把失败逐个修掉，或至少记录清楚。
4. **手工走查**（用户此前要求暂停，接手时按需执行）：起真实二进制 + 无头浏览器，重点看
   §12BO.3 列出的高风险点；`docs/HANDOVER.md` §12R/§12S/§12T 有现成的走查方法。
5. **参考源码**：`tmp/refcheck/` 当前只有 `firefish-full`（前端参考）与 `miuno`（**私有实现，
   不得作为实现参考**，AGENTS.md 已禁止）。Misskey 契约源码**不在本地**，需要按 `AGENTS.md`
   里的 clone 命令（带代理）重建，否则只能依赖 `docs/research/api-contract/` 里已记录的契约。
6. **动手前记住三条硬规则**：迁移版本不可回改（当前 v18，新改动开 v19）；每波结束必须
   `go vet` + 测试 + `vue-tsc` + `check:port` + `build` 全绿再提交；不得为没有后端的控件造页面
   （做法见 §12BM：说明页 + 缺口清单）。

### 12BO.5 恢复方式

本节点是为**交接给另一个 agent** 写的，不是为原会话续跑写的。接手方从 §12BO.4 的第一条开始
即可，不需要历史对话：判定缺什么只看 API 台账与本文档的「仍未验证/未移植」清单，判定某处行为
只看 `docs/research/api-contract/`。

继续推进时仍按项目既有节奏：每波一个主题 → 代码 → 五道检查全绿 → 独立提交 → 更新契约与本
文件（新增一节 §12BP …，并把未验证项写进去）。若用户要求先执行测试/浏览器走查，先把
§12BO.3 的验证欠账补完再开新波。

## 12BP. 第五十一波：账号迁移（2026-11-06）

用户恢复目标模式后先补验证债，再开这一波。契约见
`docs/research/2026-11-06-firefish-port-wave51-contract.md`。

### 12BP.0 验证债（恢复后的第一件事）

- `go vet ./...` 通过。
- `go test ./...` 通过；带 `CINDERFISH_TEST_DATABASE_URL` 的数据库测试**只挂一条**：
  `TestHashtagsTrend`（`trends = [], want two tags`）。根因：未配置 `hiddenTags` 时 nil 切片被当作
  `NULL` 传给 Postgres，`lower(tag) <> ALL(NULL)` 对每行都是 NULL，整张表被过滤掉。修法：函数内把
  nil 归一成空切片，并显式 `$3::text[]`。提交 `a74b55d`。
- 前端 `vue-tsc --noEmit`、`check:port`（hardTotal 0）、`pnpm build` 全绿。
- 结论：§12BO.3 列出的「自第十九波起未跑测试」欠账，在**单元/数据库测试层面已补完**；
  真实二进制手工走查与无头浏览器走查仍按用户要求留到后续。

### 12BP.1 落地清单

| 层 | 内容 | 文件 |
|----|------|------|
| 迁移 | v19：`users.moved_to_uri`、`users.also_known_as`（jsonb）+ 部分索引 | `internal/db/db.go` |
| 账号 | `User.MovedToURI` / `AlsoKnownAs`、统一投影常量、`GetByURI`、`SetKnownAs`、`MarkMoved` | `internal/account/service.go`、`migration.go` |
| 联邦 | `RenderPerson` 输出 `movedTo` / `alsoKnownAs`；`RenderMove`；入站 `Move`（校验目标别名 → 标记 → `OnMoved` 回调）；`ResolveAcct` / `FetchActor`；`Store.MarkMoved`、`LocalFollowerIDs` | `internal/federation/{render,inbox,store}.go` |
| 端点 | `POST /api/i/known-as`、`POST /api/i/move`、`users/show` 的 `userIds` 与 URI 回退 | `internal/httpapi/migration.go`、`account.go` |
| 打包 | Lite 带 `alsoKnownAs`；详细用户按 URI 反查 `movedToUri` | `internal/httpapi/users_pack.go` |
| 前端 | `SettingsMigration` 真页面、`MkMoved` 横幅、迁移菜单项、类型与端点表 | `web/src/pages/SettingsMigration.vue`、`web/src/components/MkMoved.vue`、`web/src/components/user/UserHome.vue`、`web/src/pages/SettingsIndex.vue`、`web/src/api/{types,endpoints}.ts` |

台账更新：**139 条已注册路由 / 132 条 Misskey 同名 / 308 条未实现**，Tier 1 缺口仍为 0。

### 12BP.2 有意的限制

- 迁移页按别名逐个调 `users/show`，没用上游的批量 `userIds`（后端支持批量，但批量响应会改变
  端点返回类型；逐个调用在 10 条上限下没有实际代价）。
- 只迁移 `followings` 里 `host IS NULL` 的关注者；远端关注者由对端实例自己处理。
- `movedToUri` 只在详情端点填充，列表（UserLite）保持 `null`，避免列表查询的 N+1。
- 入站 `Move` 不验证 `object` 指向的账号是否本地存在（上游也不验证），只验证目标别名。

### 12BP.3 验证证据

- `gofmt -l internal/ cmd/` 无输出。
- `go vet ./...` 通过。
- `CINDERFISH_TEST_DATABASE_URL=… go test ./...` 全绿，含新增 4 条用例：
  - `TestIKnownAsLocalAlias`：别名落库、Actor 文档与 `users/show` 都带 `alsoKnownAs`、自指别名被拒
  - `TestIMoveRefusesLocalAndAdminTarget`：本地目标 `NOT_REMOTE`、管理员 `NOT_ADMIN_FORBIDDEN`
  - `TestTwoInstanceMove`：无别名被拒 → 目标设置别名 → 迁移成功（`movedToUri.username=betabob`）→
    Actor 文档出现 `movedTo` → 二次迁移 `ALREADY_MOVED`
  - `TestInboundMove`：目标未声明别名时 `skip` 且不落库；声明后接受，`moved_to_uri` 等于目标 URI
- 前端：`vue-tsc --noEmit` 通过、`check:port` hardTotal 0（404 文件）、`pnpm build` 成功。

### 12BP.4 仍未验证

- 迁移页与横幅**没有走查**：表单禁用态、确认对话框、别名回显、已迁移账号的只读状态都只过了编译。
- `i/move` 的**真实双实例投递**没有起真实二进制验证：投递队列只验证到「入队」，没有验证对端收到
  `Move` 后展示；测试里的两实例共用同一个数据库，`LocalFollowerIDs` 的实例边界没有被真正区分。
- 远端关注者不会收到「改关新账号」的 `Follow`（设计如此），但因此**没有测试覆盖**跨实例的关注者迁移。
- 锁定账号（`manuallyApprovesFollowers`）作为目标时的 `Follow` 请求链路未验证。
- `users/show` 的 `userIds` 批量参数没有前端调用方，只有后端支持。
- 占位页现状（2026-11-06 复核，§12BO.1 的 19/15 拆分是估算）：**33 个**，其中首版范围外 24 个，
  范围内剩 9 个：`Auth`、`OAuth`、`AuthorizeInteraction`、`SignupComplete`、`SettingsIntegration`、
  `SettingsPlugin`、`SettingsPluginInstall`、`SettingsCustomKatexMacro`、`EmptyPage`。

## 12BQ. 第五十二波：设置页收尾（2026-11-06）

契约见 `docs/research/2026-11-06-firefish-port-wave52-contract.md`。

### 12BQ.1 落地清单

| 文件 | 内容 |
|------|------|
| `web/src/pages/SettingsCustomKatexMacro.vue` | 宏源文编辑、`parseKaTeXMacros` 写盘、启用开关、重新载入 |
| `web/src/pages/SettingsIntegration.vue` | 第三方登录说明页（已有 MiAuth 授权、缺 Discord/GitHub/Twitter/Matrix） |
| `web/src/pages/SettingsIndex.vue` | 菜单新增「整合」 |
| `web/src/pages/EmptyPage.vue` | 删除（与 `Empty.vue` 重复且无引用） |

### 12BQ.2 有意的限制

- **不引入 `katex`**：宏展开发生在发帖前（`preprocess()`），在 Misskey / Firefish 客户端能被排版；
  本站仍以代码显示公式，页面已写明。加依赖只为本站排版，与首版取舍不符。
- 宏页改为「保存」按钮写盘，而不是上游的每次输入即写。
- 集成页不发任何请求，也没有外部授权跳转。

### 12BQ.3 验证证据

- `vue-tsc --noEmit` 通过。
- `check:port` hardTotal 0（402 文件），`missingKeys` / `unresolvedComponents` 均空。
- `pnpm build` 成功。
- 本波只改前端，后端测试沿用第五十一波的全绿结果。

### 12BQ.4 仍未验证与剩余缺口

- 两个页面**没有浏览器走查**：宏保存后的 `location.reload()` 生效路径、开关持久化、
  说明页在窄屏下的排版都只过了编译。
- 占位页剩余 **30 个**，其中范围内 6 个：`Auth`、`OAuth`、`AuthorizeInteraction`、
  `SignupComplete`（缺 `auth/session/*`、OAuth 同意页、邮箱确认注册）、
  `SettingsPlugin`、`SettingsPluginInstall`（缺 AiScript 运行时）。

## 12BR. 第五十三波：OAuth 交互式同意页（2026-11-06）

契约见 `docs/research/2026-11-06-firefish-port-wave53-contract.md`。

### 12BR.1 落地清单

| 层 | 内容 |
|----|------|
| 后端 | `GET /oauth/authorize` 无凭据时返回 SPA；抽出 `parseOAuthAuthorize` / `issueOAuthGrant` / `oauthRedirectURL`；新增 `POST /api/oauth/authorize`（返回跳转地址） |
| 前端 | `OAuth.vue` 占位页 → 同意页（登录、应用主机、scope、接受/取消） |
| 台账 | 140 条注册路由；自有路径 8 条（新增 `oauth/authorize`） |
| 测试 | `oauth_consent_test.go` 三条用例 |

### 12BR.2 有意的限制

- 取消授权不回跳客户端（不发 `access_denied`），只显示已拒绝。
- 不做「记住此应用」与 scope 勾选：本站 scope 是粗粒度四类，按请求整体授予/拒绝。
- `client_id` 不做 URL 校验；同意页只在能解析时显示主机。

### 12BR.3 验证证据

- `gofmt -l internal/ cmd/` 无输出；`go vet ./...` 通过。
- `CINDERFISH_TEST_DATABASE_URL=… go test ./...` 全绿，含三条新用例；既有 OAuth 三件套未受影响。
- `vue-tsc --noEmit`、`check:port`（hardTotal 0）、`pnpm build` 全绿。

### 12BR.4 仍未验证

- 同意页**没有浏览器走查**：登录内嵌、接受后的 `location.assign`、缺参提示都只过了编译。
- `GET /oauth/authorize` 无凭据分支在**没有前端构建**时保持 401 的旧行为，没有专门用例。
- 没有真实第三方客户端（misskey-js / Firefish Web）跑完整 OAuth 登录。

## 12BS. 第五十四波：ap/show 与远程互动页（2026-11-06）

契约见 `docs/research/2026-11-06-firefish-port-wave54-contract.md`。

### 12BS.1 落地清单

| 层 | 内容 |
|----|------|
| 联邦 | `Inbox.ImportObject`：签名抓取任意 AP 对象 → Actor 走 `upsertActor`、Note 走 `NoteDataFromAP` + `InsertRemoteNote`；`Store.NoteIDByAPID` |
| 端点 | `POST /api/ap/show` → `{type:"Note"|"User", object}`，打包与 `notes/show` / `users/show` 同一套 |
| 前端 | `AuthorizeInteraction.vue` 占位页 → 真页面（账号 + 关注按钮 / 笔记 + `MkNoteDetailed` / 原实例外链） |
| 台账 | 141 路由 / 133 同名 / 307 未实现 |

### 12BS.2 有意的限制

- 导入的笔记不产生通知（对象不是发给本实例的），也不解析提及。
- 只处理 Note 类与 Actor 类对象；其余一律 400 `NO_SUCH_OBJECT`。
- 不实现 Mastodon 的 `/authorize_interaction` 语义。

### 12BS.3 验证证据

- `gofmt -l internal/ cmd/` 无输出；`go vet ./...` 通过。
- `CINDERFISH_TEST_DATABASE_URL=… go test ./...` 全绿，含
  `TestAPShowResolvesRemoteObjects`（双实例：笔记与账号两种类型）与 `TestAPShowRequiresCredential`。
- `vue-tsc --noEmit`、`check:port`（hardTotal 0）、`pnpm build` 全绿。

### 12BS.4 仍未验证

- 远程互动页**没有浏览器走查**：加载态、笔记渲染、关注按钮点击都只过了编译。
- `ImportObject` 的拒绝路径（attributedTo 与 host 不匹配、非 Note/Actor 对象）没有构造对端用例。
- 真实 Mastodon / Misskey 实例的 `/authorize_interaction` 链接没有实测。

## 12BT. 第五十五波：第三方应用授权（2026-11-06）

契约见 `docs/research/2026-11-06-firefish-port-wave55-contract.md`。
本轮把 Misskey `4ff2edf` 的后端源码稀疏克隆到 `tmp/refcheck/misskey`（此前只有 `/tmp` 抓取），
端点契约以源码为准。

### 12BT.1 落地清单

| 层 | 内容 |
|----|------|
| 迁移 | v20：`apps`、`auth_sessions`、`access_tokens.app_id` |
| 账号 | `account/apps.go`：应用注册/查询、会话生成/批准/取回、按应用发令牌 |
| 端点 | `app/create`、`app/show`、`auth/session/generate|show|userkey`、`auth/accept` |
| 兼容 | `i/apps` 在有 `app_id` 时回退到应用名称/权限（Misskey 规则） |
| 前端 | `Auth.vue` 同意页（未登录嵌入登录、已授权自动放行、回调协议白名单） |
| i18n | 补 `_auth.pleaseLogin` / `_auth.accepted` / `_auth.alreadyAuthorized`（zh-CN + ja-JP） |
| 台账 | 147 路由 / 139 同名 / 301 未实现 |

### 12BT.2 有意的限制

- 应用密钥不过期、不轮换；同一 `(app,user)` 只有一条访问令牌。
- 不校验回调域名所有权，只做跳转时的协议白名单。
- `permission` 不校验取值，仅展示与记录。

### 12BT.3 验证证据

- `gofmt -l internal/ cmd/` 无输出；`go vet ./...` 通过。
- `CINDERFISH_TEST_DATABASE_URL=… go test ./...` 全绿，含
  `TestAppSessionFlow`（注册→会话→待批→批准→取 token→`/api/i` 生效→会话只能取一次→`i/apps`）
  与 `TestAuthSessionGenerateRejectsUnknownSecret`。
- `vue-tsc --noEmit`、`check:port`（hardTotal 0、missingKeys 0）、`pnpm build` 全绿。

### 12BT.4 仍未验证

- `Auth.vue` **没有浏览器走查**：未登录嵌入登录、已授权自动放行、回调跳转都只过了编译。
- 真实第三方客户端（第三方 Misskey 客户端 / Firefish 的 auth.vue）没有跑过这条流程。
- 应用所有者的 `app/show` 才返回 secret 这条分支没有用例（匿名注册的应用没有所有者）。

## 12BU. 第五十六波：关注关系设置（2026-11-06）

契约见 `docs/research/2026-11-06-firefish-port-wave56-contract.md`。

### 12BU.1 落地清单

| 层 | 内容 |
|----|------|
| 迁移 | v21：`followings.notify`、`followings.with_replies` |
| 账号 | `Follow.Notify/WithReplies`；`UpdateFollowing`、`UpdateAllFollowing`（动态 SET 与参数一起构造） |
| 端点 | `following/list`（`notification` 过滤）、`following/update`、`following/update-all`、`following/invalidate` |
| 前端 | 用户菜单「移除关注者」（`following/invalidate`），同步删掉该文件的降级注释条目 |
| 台账 | 151 路由 / 143 同名 / 297 未实现 |

### 12BU.2 有意的限制

- `notify` / `withReplies` **只落库、未消费**：Streaming 不向关注者 main 频道扇出新帖，
  也没有 `type: note` 通知，所以本波没有给这两个开关做 UI（避免假控件）。接线排在下一波。
- `following/invalidate` 不向远端投递 Undo（需要被移除者的私钥），本地关注者收到 unfollow 事件。

### 12BU.3 验证证据

- `gofmt -l internal/ cmd/` 无输出；`go vet ./...` 通过。
- `CINDERFISH_TEST_DATABASE_URL=… go test ./...` 全绿，含 `TestFollowingListAndUpdate`
  （默认不序列化 `withReplies`、update 回读、notification 过滤、update-all 清空、非关注 400）
  与 `TestFollowingInvalidate`（移除后关系消失、再次移除 400）。
- `vue-tsc --noEmit`、`check:port`（hardTotal 0、missingKeys 0）、`pnpm build` 全绿。

### 12BU.4 仍未验证

- 「移除关注者」菜单项**没有浏览器走查**：出现条件（`isFollowed`）、确认对话框、置位回写都只过了编译。
- `following/list` 的 `sinceDate`/`untilDate` 参数没有实现（台账里未列，但 Misskey 的 paramDef 有）：
  只支持 id 游标。
- 远端关注者在 `following/invalidate` 之后不会收到任何协议信号，只有本地状态改变。

## 12BV. 第五十七波：接线 notify / withReplies（2026-11-06）

契约见 `docs/research/2026-11-06-firefish-port-wave57-contract.md`。
第五十六波落库的两个开关在这一波接上了消费方，同时补掉一个既有 UI 缺口。

### 12BV.1 落地清单

| 层 | 内容 |
|----|------|
| 通知 | `notifyFollowersOfNote`：`notify=normal` 的关注者在作者发帖时收到 `type: note`；本地发帖与远端导入两条路径都挂上；直发笔记跳过 |
| Streaming | `LoadFollows` 读 `with_replies`，`homeTimeline` 的回复过滤真正生效 |
| 前端 | `MkNotification` 支持 `note` / `follow` / `receiveFollowRequest`（此前关注类通知没有文字）；类型过滤窗口与 `NotificationType` 同步；新增 `_notification.newNote` |
| 台账 | 无新增路由（151 / 143 / 297 不变） |

### 12BV.2 有意的限制

- `notify` 只发站内通知，不发邮件（邮件触发仍是四类）。
- `withReplies` 只影响 Streaming 的 `homeTimeline`；HTTP 拉取的 `notes/timeline` 不过滤回复，
  与 Misskey 的 SQL 语义一致。
- 通知不聚合，一条笔记一条通知。

### 12BV.3 验证证据

- `gofmt -l internal/ cmd/` 无输出；`go vet ./...` 通过。
- `CINDERFISH_TEST_DATABASE_URL=… go test ./...` 全绿，含
  `TestNotifyFollowNotifications`（开通知的关注者收到 `note` 且 `noteId` 正确，普通关注者收不到）
  与 `TestLoadFollowsCarriesWithReplies`（默认 true，改 false 后读回 false）。
- `vue-tsc --noEmit`、`check:port`（hardTotal 0、missingKeys 0）、`pnpm build` 全绿。

### 12BV.4 仍未验证

- 通知中心里新的三种类型**没有浏览器走查**：图标、文案、点击跳转、类型过滤都只过了编译。
- 远端导入触发通知走的是 `OnNoteCreated`，双实例用例没有覆盖到（本波只用了本地发帖触发）。
- Streaming `homeTimeline` 的实际 WS 过滤行为没有用真实连接验证，只有 `LoadFollows` 的单元断言。

## 12BW. 第五十八波：入站反应修复与 EmojiReact（2026-11-06）

契约见 `docs/research/2026-11-06-firefish-port-wave58-contract.md`。
本波是**缺陷修复**：写用例时发现远端用户对本地笔记的反应一直没落库。

### 12BW.1 问题与修法

| 问题 | 根因 | 修法 |
|------|------|------|
| 远端 `Like` 对本地笔记一直 `skip: target note not found` | `Store.React` 只按 `notes.ap_id/uri` 找笔记，本地笔记这两列为空 | 新增 `Store.NoteIDByAPIDOrURL`，本地笔记按 `<baseURL>/notes/<id>` 解析 |
| `EmojiReact` / `EmojiReaction` 被当成未知活动 | `perform` / `undo` 只判断 `Like` | 三种类型走同一条反应路径 |
| 反应写入绕了一层 AP 查询 | 只有 `React(noteAPID)` | 新增 `ReactNoteID` / `UnreactNoteID`，删除已无调用方的 `React` / `Unreact` |

### 12BW.2 有意的限制

- 出站反应仍是 `Like` + `content`（与 Misskey `renderLike` 一致），没有改发 `EmojiReact`。
- 不处理 `EmojiReact` 的 `tag` 自定义表情定义（还没有远端 emoji 表）。
- `Flag`、`Question`（投票）仍未处理。

### 12BW.3 验证证据

- `gofmt -l internal/ cmd/` 无输出；`go vet ./...` 通过。
- `CINDERFISH_TEST_DATABASE_URL=… go test ./...` 全绿。
- `TestInboundEmojiReact`：双实例 + 真实签名，beta 对 alpha 的本地笔记发 `EmojiReact` →
  `notes/reactions` 出现 🍮 → `Undo(EmojiReact)` 后清空。修复前这条用例在第一步就失败。

### 12BW.4 仍未验证

- 没有对真实 Misskey / Firefish 实例验证 `EmojiReact` 的收发。
- 自定义表情反应（`:name:` 形式）没有远端 emoji 解析，仅按文本存库。
- `Flag`（举报）与 `Question`（投票）入站仍未实现（`Flag` 已在第五十九波补上）。

## 12BX. 第五十九波：入站 Flag（2026-11-06）

契约见 `docs/research/2026-11-06-firefish-port-wave59-contract.md`。

### 12BX.1 落地清单

| 层 | 内容 |
|----|------|
| 联邦 | `Inbox.flag`：`object` 中第一个 `<baseURL>/users/<id>` 作为举报对象，`content` + 目标 URI 列表写入评论 |
| 存储 | `Store.RecordReport` 插入 `abuse_reports`，reporter 是远端账号行 |
| 台账 | 入站活动清单更新，只剩 `Question`（投票）未覆盖 |

### 12BX.2 有意的限制

- 不做重复举报去重与频率限制（Misskey 有报告阻塞表）。
- 笔记级目标只进评论文本，`abuse_reports` 没有 note 列。
- 不给管理员发通知或邮件。

### 12BX.3 验证证据

- `gofmt -l internal/ cmd/` 无输出；`go vet ./...` 通过。
- `CINDERFISH_TEST_DATABASE_URL=… go test ./...` 全绿。
- `TestInboundFlagCreatesReport`：双实例 + 真实签名，beta 举报 alpha 的账号后，
  `admin/abuse-user-reports` 出现未处理报告，评论含原文与目标 URI，reporter 为 betabob。

### 12BX.4 仍未验证

- 没有对真实 Misskey/Firefish 实例验证远端举报。
- 管理台重复举报、笔记级目标的展示没有走查。

## 12BY. 第六十波：范围内占位页清零（2026-11-06）

契约见 `docs/research/2026-11-06-firefish-port-wave60-contract.md`。

### 12BY.1 落地清单

| 文件 | 内容 |
|------|------|
| `web/src/components/CfFeatureGap.vue` | 新增：说明页共享组件（title / summary / available / missing） |
| `web/src/pages/SignupComplete.vue` | 说明「无邮箱确认注册模式」+ 登录/注册入口 |
| `web/src/pages/SettingsPlugin.vue` | 说明「无 AiScript 运行时」+ 自定义 CSS / KaTeX / API 入口 |
| `web/src/pages/SettingsPluginInstall.vue` | 说明「不接受 .aiscript 上传」 |

占位页从 30 个降到 **24 个**，全部属于用户已裁决的首版范围外页面；**范围内占位页 = 0**。

### 12BY.2 有意的限制

- 这三页都不是功能，只是把缺口写清楚：不伪造安装表单、不伪造注册确认。
- 插件页没有加入设置菜单（与第四十九波的管理台说明页一致，未链接）。
- `SignupComplete` 不做 `code` 参数校验：没有对应的后端流程，校验也无从谈起。

### 12BY.3 验证证据

- `vue-tsc --noEmit` 通过。
- `check:port`：hardTotal 0，`unresolvedComponents` / `missingKeys` 均为空。
- `pnpm build` 成功。
- 本波只改前端；后端测试沿用第五十九波的全绿结果。

### 12BY.4 仍未验证

- 三个说明页**没有浏览器走查**：窄屏排版、外链跳转、组件复用后的样式都只过了编译。
- 占位页清单的「24 个全部在范围外」是按 `grep cf-page-placeholder` 与 §12BO.1 的范围清单
  对照得出的，没有逐页复核用户裁决记录。

## 12BZ. 第六十一波：hashtags 三端点与标签页统计（2026-11-06）

契约见 `docs/research/2026-11-06-firefish-port-wave61-contract.md`。

### 12BZ.1 落地清单

| 层 | 内容 |
|----|------|
| 端点 | `hashtags/show`（单标签计数，空标签 404 `NO_SUCH_HASHTAG`）、`hashtags/list`（排序 + attached 过滤）、`hashtags/users`（恒空） |
| 实现 | 计数从 `notes.tags` 对 `notes` 聚合，不建 hashtags 表；`normalizeHashtag` 对齐 Misskey 的 trim/lower/去 `#` |
| 前端 | `Tag.vue` 头部改用 `hashtags/show`，文案由「最近 7 天」改为「全部时间」 |
| 台账 | 154 路由 / 146 同名 / 294 未实现 |

### 12BZ.2 有意的限制

- 计数不缓存（个人实例的数据量下是毫秒级），换来永远与笔记一致。
- `hashtags/users` 恒为空：上游按用户资料标签列人，本站没有这个模型，不拿笔记标签凑数。
- `attached*` 计数恒为 0，`attached*Only` 过滤返回空数组，`attached*` 排序退化为按提及人数排序。

### 12BZ.3 验证证据

- `gofmt -l internal/ cmd/` 无输出；`go vet ./...` 通过。
- `CINDERFISH_TEST_DATABASE_URL=… go test ./...` 全绿，含
  `TestHashtagsShowAndList`（计数 2/2/0、未知标签 404、升降序、attached 过滤）与
  `TestHashtagsUsersIsEmpty`。
- `vue-tsc --noEmit`、`check:port`（hardTotal 0、missingKeys 0）、`pnpm build` 全绿。

### 12BZ.4 仍未验证

- 标签页的新统计**没有浏览器走查**：404 时隐藏、切换标签、窄屏排版都只过了编译。
- `hashtags/list` 的 12 个排序枚举只验证了 `+/-mentionedUsers` 两个。
- 大表下聚合查询的耗时没有测。

## 12CA. 第六十二波：users/relation 与频繁回复用户（2026-11-06）

契约见 `docs/research/2026-11-06-firefish-port-wave62-contract.md`。

### 12CA.1 落地清单

| 层 | 内容 |
|----|------|
| 端点 | `users/relation`（单值或批量 id，返回关注/待审/屏蔽/静音标志）、`users/get-frequently-replied-users`（`{user, weight}[]`） |
| 实现 | 标志复用既有查询；频繁回复按 `reply_user_id` 分组计数，weight 降序、id 升序 |
| 台账 | 156 路由 / 148 同名 / 292 未实现 |

### 12CA.2 有意的限制

- `isRenoteMuted` 恒为 false（没有 renote-mute 表）。
- `users/relation` 只返回调用者自己的关系，没有额外的可见性检查。
- 频繁回复统计不按单条笔记可见性过滤。

### 12CA.3 验证证据

- `gofmt -l internal/ cmd/` 无输出；`go vet ./...` 通过。
- `CINDERFISH_TEST_DATABASE_URL=… go test ./...` 全绿，含 `TestUsersRelation`
  （双向关注、静音、批量同序）与 `TestUsersFrequentlyReplied`（权重 2/1 排序）。
- 本波只改后端，前端三件套沿用第六十一波的通过结果。

### 12CA.4 仍未验证

- 没有前端调用方接入这两个端点（用户页目前用 `users/show` 自带的 `isFollowing` 等字段），
  所以没有对应的界面走查。
- 批量 `users/relation` 的上限没有压测；Misskey 的 paramDef 也没有给数组长度上限。




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
| `docs/research/api-contract/` | **后端 API 对接契约（8 份，新前端对接的唯一依据，见 §12N）** |
| `tmp/refcheck/firefish-full/` | **Firefish 完整源码快照（jclaudan 镜像 `b425f61`，2023-10-08，去 `.git`，gitignore 内）——前端移植的工作参考，见 §12O** |
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
| `docs/research/2026-11-05-firefish-port-wave2-contract.md` | **第二波施工契约（模块映射、降级清单、写作用域、验收口径）——续作移植任务的模板，见 §12R** |
| `docs/research/2026-11-05-firefish-port-wave3-contract.md` | **第三波施工契约（发帖链路、个人页、设置、内容页、壳层/widgets 的模块映射与 13 个写作用域），见 §12S** |
| `docs/research/2026-11-06-firefish-port-wave4-contract.md` | **第四波施工契约（管理台、主题体系、注册表、本地设置、静态页的模块映射与 7 个写作用域），见 §12T** |
| `docs/research/2026-11-06-firefish-port-wave5-contract.md` | **第五波施工契约（静音/屏蔽、令牌管理、实例静音、无障碍、多账号、4 个 widget 与明确不做的项），见 §12U** |
| `docs/research/2026-11-06-firefish-port-wave6-contract.md` | **第六波施工契约（notes/search、notes/search-by-tag、users/search 与搜索/标签页），见 §12V** |
| `docs/research/2026-11-06-firefish-port-wave7-contract.md` | **第七波施工契约（admin/show-users 筛选、admin/drive/files 与用户/文件管理页），见 §12W** |
| `docs/research/2026-11-06-firefish-port-wave8-contract.md` | **第八波施工契约（公告表 v10、四个公告端点、公告页与管理页），见 §12X** |
| `docs/research/2026-11-06-firefish-port-wave9-contract.md` | **第九波施工契约（i/export、i/delete-account、导出页与注销页），见 §12Y** |
| `docs/research/2026-11-06-firefish-port-wave10-contract.md` | **第十波施工契约（举报表 v11、三个举报端点、用户菜单入口与管理页），见 §12Z** |
| `docs/research/2026-11-06-firefish-port-wave11-contract.md` | **第十一波施工契约（webhooks 表 v12、五个端点、异步投递与设置页），见 §12AA** |
| `docs/research/2026-11-06-firefish-port-wave12-contract.md` | **第十二波施工契约（admin/get-table-stats、admin/get-index-stats 与数据库页），见 §12AB** |
| `docs/research/2026-11-06-firefish-port-wave13-contract.md` | **第十三波施工契约（运行时实例设置、admin/meta 与 update-meta 子集），见 §12AC** |
| `docs/research/2026-11-06-firefish-port-wave14-contract.md` | **第十四波施工契约（blockedHosts、策略与入站拒绝、实例屏蔽页），见 §12AD** |
| `docs/research/2026-11-06-firefish-port-wave15-contract.md` | **第十五波施工契约（MiAuth check、fetched 标记与授权页），见 §12AE** |
| `docs/research/2026-11-06-firefish-port-wave16-contract.md` | **第十六波施工契约（notes/mentions、unreadOnly 与通知页签），见 §12AF** |
| `docs/research/2026-11-06-firefish-port-wave17-contract.md` | **第十七波施工契约（公开 users 目录与发现页用户页签），见 §12AG** |
| `docs/research/2026-11-06-firefish-port-wave18-contract.md` | **第十八波施工契约（对象存储只读页与投递队列概览），见 §12AH** |
| `docs/research/2026-11-06-firefish-port-wave19-contract.md` | **第十九波施工契约（合规链接、serverRules、公共 stats 与关于页），见 §12AI** |
| `docs/research/2026-11-06-firefish-port-wave20-contract.md` | **第二十波施工契约（TOTP 两步验证、登录协商与安全设置页），见 §12AJ** |
| `docs/research/2026-11-06-firefish-port-wave21-contract.md` | **第二十一波施工契约（改密、令牌重生成与登录历史），见 §12AK** |
| `docs/research/2026-11-06-firefish-port-wave22-contract.md` | **第二十二波施工契约（资料扩展字段与推荐笔记开关），见 §12AL** |
| `docs/research/2026-11-06-firefish-port-wave23-contract.md` | **第二十三波施工契约（通知类型静音与未读过滤），见 §12AM** |
| `docs/research/2026-11-06-firefish-port-wave24-contract.md` | **第二十四波施工契约（笔记编辑与编辑历史），见 §12AN** |
| `docs/research/2026-11-06-firefish-port-wave25-contract.md` | **第二十五波施工契约（云盘文件重命名/描述/敏感/移动），见 §12AO** |
| `docs/research/2026-11-06-firefish-port-wave26-contract.md` | **第二十六波施工契约（隐藏话题标签与趋势过滤），见 §12AP** |
| `docs/research/2026-11-06-firefish-port-wave27-contract.md` | **第二十七波施工契约（实例信息页与 show-instance），见 §12AQ** |
| `docs/research/2026-11-06-firefish-port-wave28-contract.md` | **第二十八波施工契约（网盘用量端点与设置页接线），见 §12AR** |
| `docs/research/2026-11-06-firefish-port-wave29-contract.md` | **第二十九波施工契约（关于页），见 §12AS** |
| `docs/research/2026-11-06-firefish-port-wave30-contract.md` | **第三十波施工契约（账号导入），见 §12AT** |
| `docs/research/2026-11-06-firefish-port-wave31-contract.md` | **第三十一波施工契约（实例图标/横幅/主题色），见 §12AU** |
| `docs/research/2026-11-06-firefish-port-wave32-contract.md` | **第三十二波施工契约（已收录实例列表），见 §12AV** |
| `docs/research/2026-11-06-firefish-port-wave33-contract.md` | **第三十三波施工契约（笔记已编辑标记），见 §12AW** |
| `docs/research/2026-11-06-firefish-port-wave34-contract.md` | **第三十四波施工契约（notes/state 与收藏菜单），见 §12AX** |
| `docs/research/2026-11-06-firefish-port-wave35-contract.md` | **第三十五波施工契约（从 URL 上传），见 §12AY** |
| `docs/research/2026-11-06-firefish-port-wave36-contract.md` | **第三十六波施工契约（公告已读状态），见 §12AZ** |
| `docs/research/2026-11-06-firefish-port-wave37-contract.md` | **第三十七波施工契约（公告编辑），见 §12BA** |
| `docs/research/2026-11-06-firefish-port-wave38-contract.md` | **第三十八波施工契约（邮件与邮箱验证），见 §12BB** |
| `docs/research/2026-11-06-firefish-port-wave39-contract.md` | **第三十九波施工契约（邮箱前端页面），见 §12BC** |
| `docs/research/2026-11-06-firefish-port-wave40-contract.md` | **第四十波施工契约（出站投递与取消转贴），见 §12BD** |
| `docs/research/2026-11-06-firefish-port-wave41-contract.md` | **第四十一波施工契约（编辑的联邦 Update），见 §12BE** |
| `docs/research/2026-11-06-firefish-port-wave42-contract.md` | **第四十二波施工契约（按活动作者签名），见 §12BF** |
| `docs/research/2026-11-06-firefish-port-wave43-contract.md` | **第四十三波施工契约（四个兼容端点），见 §12BG** |
| `docs/research/2026-11-06-firefish-port-wave44-contract.md` | **第四十四波施工契约（实例 CSS 与邮件设置页），见 §12BH** |
| `docs/research/2026-11-06-firefish-port-wave45-contract.md` | **第四十五波施工契约（管理台文件详情），见 §12BI** |
| `docs/research/2026-11-06-firefish-port-wave46-contract.md` | **第四十六波施工契约（缩略图管道），见 §12BJ** |
| `docs/research/2026-11-06-firefish-port-wave47-contract.md` | **第四十七波施工契约（锁推生效与关注通知），见 §12BK** |
| `docs/research/2026-11-06-firefish-port-wave48-contract.md` | **第四十八波施工契约（邮件通知），见 §12BL** |
| `docs/research/2026-11-06-firefish-port-wave49-contract.md` | **第四十九波施工契约（无后端管理页转说明页），见 §12BM** |
| `docs/research/2026-11-06-firefish-port-wave50-contract.md` | **第五十波施工契约（状态栏），见 §12BN** |
| `docs/research/2026-11-06-firefish-port-wave51-contract.md` | **第五十一波施工契约（账号迁移），见 §12BP** |
| `docs/research/2026-11-06-firefish-port-wave52-contract.md` | **第五十二波施工契约（设置页收尾），见 §12BQ** |
| `docs/research/2026-11-06-firefish-port-wave53-contract.md` | **第五十三波施工契约（OAuth 同意页），见 §12BR** |
| `docs/research/2026-11-06-firefish-port-wave54-contract.md` | **第五十四波施工契约（ap/show 与远程互动），见 §12BS** |
| `docs/research/2026-11-06-firefish-port-wave55-contract.md` | **第五十五波施工契约（应用授权 app/* 与 auth/session/*），见 §12BT** |
| `docs/research/2026-11-06-firefish-port-wave56-contract.md` | **第五十六波施工契约（关注关系设置），见 §12BU** |
| `docs/research/2026-11-06-firefish-port-wave57-contract.md` | **第五十七波施工契约（接线 notify / withReplies），见 §12BV** |
| `docs/research/2026-11-06-firefish-port-wave58-contract.md` | **第五十八波施工契约（入站反应修复与 EmojiReact），见 §12BW** |
| `docs/research/2026-11-06-firefish-port-wave59-contract.md` | **第五十九波施工契约（入站 Flag），见 §12BX** |
| `docs/research/2026-11-06-firefish-port-wave60-contract.md` | **第六十波施工契约（范围内占位页清零），见 §12BY** |
| `docs/research/2026-11-06-firefish-port-wave61-contract.md` | **第六十一波施工契约（hashtags 三端点），见 §12BZ** |
| `docs/research/2026-11-06-firefish-port-wave62-contract.md` | **第六十二波施工契约（users/relation 与频繁回复），见 §12CA** |
| `docs/research/2026-11-06-firefish-port-wave63-contract.md` | **第六十三波施工契约（真实二进制与浏览器走查），见 §12CB** |

## 14. 已知的未验证项

- **第五波的 API 与页面渲染已验证（2026-11-06，见 §12U.3）**：静音/屏蔽/实例静音/令牌撤销都做了
  真实二进制回读，五个设置页与四个 widget 在桌面/移动无头浏览器里走查过；仍未验证的是 UI 点击路径
  （撤销、切换账号、widget 配置弹窗）、列表分页游标与实例静音非法输入，见 §12U.4。
- ~~**第六波的搜索过滤组合未全测**~~：2026-11-07 补了 `TestNotesSearchFilters`，覆盖
  `offset`、`userId`、`host`（空 / 非空）与 `withFiles`（`notes/search` 与 `notes/search-by-tag`）。
  仍然没有构造数据的：远端作者的搜索结果；`ILIKE '%…%'` 的性能没有压测。
- ~~**第七波的排序与游标组合未全测**~~：2026-11-07 补了 `TestAdminDriveFilters`，用构造的远端账号与
  文件覆盖 `userId`、`origin=remote`、`hostname`（大小写折叠）与 `sinceId`/`untilId`/双游标。
  `admin/show-users` 的四种 sort 也在 2026-11-07 由 `TestAdminShowUsersSorts` 覆盖
  （+/-follower、+/-createdAt；`state=silenced` 仍在实现层面为空集，没有数据可造）。
- **第八波的部分边角（2026-11-06，见 §12X.4）**：`untilId` 多页与 `limit` 上限已由
  `TestAnnouncementsPagination` 覆盖（2026-11-07，含 100 条上限）。仍未构造数据的：公告图片渲染、
  浏览器内的过期隐藏；已读跟踪与首页横幅按设计未做。
- **第十波的部分边角（2026-11-06，见 §12Z.4）**：远端举报已由 `TestInboundFlagCreatesReport`
  覆盖（第五十九波），分页游标与 `state` 过滤已由 `TestAbuseReportPagination` 覆盖（2026-11-07）。
  仍未验证的：toast 文案、转发（forward）；通知与自动处置按设计未做。
- **第十一波的部分边角（2026-11-06，见 §12AA.4）**：失败路径已由
  `TestWebhookDeliveryFailureStatus` 覆盖（接收端 500 → `latestStatus=500`；
  连接失败 → `1`），未产生的事件类型由 `TestWebhookIgnoresUnproducedEvents` 覆盖
  （订阅 follow 的 webhook 在关注发生时收不到投递）。仍未验证的：投递重试与顺序
  （实现上不重试，一条事件一条独立请求）、编辑页内删除。
- **第十二波未测边角（2026-11-06，见 §12AB.4）**：reltuples 估算与实际行数的偏差、大 schema 渲染性能
  未验证；vacuum 与索引维护按设计未做。
- **第十三波的部分边角（2026-11-06，见 §12AC.4）**：邀请码注册已由
  `TestSignupWithInviteCode` 覆盖（有效码建号、单次使用、失败后释放），账户层另有
  `TestRegisterInviteMode` 的释放断言。注册页**界面**仍未走查；没有审计日志、公开页缓存需刷新；
  上游 update-meta 的其余字段按设计忽略。
- **第十四波的部分边角（2026-11-06，见 §12AD.4）**：与 allowlist 的组合已由
  `TestAllowlistGatesInboundInbox` 覆盖（2026-11-07）：未列出的主机连签名都无法验证（401），
  列出后同一 Follow 得到 202 并落库；签名拒绝本身有 `TestInboxRejects*` 三件套。
  子域匹配的**运维体验**（真实反代下的配置）仍未验证。
- **第十五波的部分边角（2026-11-06，见 §12AE.4）**：MiAuth 的未登录衔接与回调已在
  2026-11-07 走查（`/miauth/<session>` 未登录时内嵌登录 → 登录后出现「查看账号信息」授权页 →
  接受后跳到 `https://…/cb?session=<session>`，`access_tokens` 出现 session 绑定的令牌）。
  OAuth 同意页见 §12CB。拒绝回执与 scope 强制按设计未做。
- **第十六波的部分边角（2026-11-06，见 §12AF.4）**：远端提及已由
  `TestRemoteMentionIsDeliveredAndNotifies` 覆盖（2026-11-07，见 §12CD）：本地笔记 @ 远端账号会
  解析成 Mention 标签、经投递队列送到对端并产生 `mention` 通知（顺带修掉了入站提及被写成
  `reply` 的缺陷）。仍未验证的：流式推送下的未读实时插入、directNotes 与类型过滤组合。
- **第十七波的部分边角（2026-11-06，见 §12AG.4）**：`users` 目录的未登录视角、`state=admin`、
  `origin=remote`、`hostname`（需与 `origin=combined` 组合，与 Misskey 一致）与两种 createdAt 排序
  已由 `TestUsersDirectoryFilters` 覆盖（2026-11-07），并确认封禁账号从目录消失。
  `state=alive` 因为没有 `users.updatedAt` 列而与 `all` 等价（用例固定了这一点）。
  **界面**上的发现页用户页签仍未走查。
- **第十八波的部分边角（2026-11-06，见 §12AH.4）**：投递队列的终态已由
  `TestDelivererOutcomes` / `TestDelivererGivesUp` / `TestDelivererAbandonsDisallowedHost`
  覆盖（2026-11-07）：成功标记、500 后退避重排、达到 `MaxAttempts` 放弃、策略拒绝立即放弃。
  仍未验证的：并发压测（多 worker 抢同一条）与 S3 真实桶展示；手动重试与配置写入按设计未做。
- **第十九波的部分边角（2026-11-06，见 §12AI.4）**：规则与合规链接现在进入注册流程（2026-11-07）：
  `CfSignup` 在管理员配置了 `tosUrl` / `serverRules` 时显示「注册即表示你同意 本实例的服务条款与
  实例规则」，浏览器实测两个链接（外链 `_blank`、规则链到 `/about`）、移动端无溢出；清空配置后
  该行消失。仍未验证的：远端实例计数与大表统计耗时。
- **第二十波未测（2026-11-06，见 §12AJ.4）**：按用户指示本轮只写代码，TOTP 端到端登录（空码/错码/
  正确码）、设置页交互、移动端布局与数据库用例都未执行；`qr` 为空串、安全密钥与备份码按设计未做。
- **第二十一波已补验证（2026-11-06，见 §12AK.4）**：三个端点现在有数据库用例
  （`TestPasswordChange` 与 `TestRegenerateToken`），并且 2026-11-07 用真实二进制 + 浏览器走查了
  `/settings/security`：改密后页面提示「密码已更新」，旧密码登录 403、新密码 200；
  「重新生成令牌」显示了新令牌，旧令牌 `/api/i` 401、新令牌 200；登录历史区渲染正常；移动端无溢出。
  分页（每页 10 条）仍未构造满页数据。
- **第二十二波的部分边角（2026-11-06，见 §12AL.4）**：资料扩展字段（location / birthday / lang /
  fields）与 `injectFeaturedNote` 的保存、`users/show` 回读、四种非法值（日期、语言标签、位置过长、
  字段过多）与 null 清空已由 `TestIUpdateProfileDetails` 覆盖（2026-11-07）。仍未验证的：
  「推荐笔记」开关对时间线的实际影响（时间线里注入了 featured note 的展示没有走查）。
- **第二十三波已补验证并修缺陷（2026-11-06，见 §12AM.4）**：`TestNotificationTypeMuting` 覆盖了
  静音后列表与未读数联动、显式 `includeTypes` 优先、未知类型 400、重复项去重与清空。
  用例暴露了缺陷：`GetProfile` 从不读取 `settings.mutingNotificationTypes`，所以静音写入后
  **完全没有效果**；已修复。设置页的**界面**交互仍未走查。
- **第二十四波的部分边角（2026-11-06，见 §12AN.4）**：编辑链路已由 `TestNotesEditAndHistory`
  覆盖（2026-11-07）：作者改文本/CW/可见性后回读一致，`notes/history` 记录一条修订，非作者
  编辑/读历史 403，缺 editId 400 `NEEDS_EDIT_ID`，未知 id 404，超长文本 400。
  仍未验证的：笔记菜单的编辑入口**界面**、远端对 Update 的处理（设计上只做尽力投递）。
- **第二十五波的部分边角（2026-11-06，见 §12AO.4）**：文件编辑已由 `TestDriveFilesUpdate` 覆盖
  （2026-11-07）：改名/备注/敏感标记/移动文件夹一次完成并回读，`folderId: null` 移回根目录，
  他人文件 404、未知文件 404、未知文件夹 400、空文件名 400。仍未验证的：**拖拽移动**的界面交互。
- **第二十六波已补验证（2026-11-06，见 §12AP.4）**：`TestHiddenTags` 覆盖保存与回读
  （trim、去 `#`、小写、去重）、趋势不再包含隐藏标签、`hashtags/show` 仍可回答、超长标签 400、
  清空后标签回到趋势（2026-11-07）。
- **第二十七波已补验证（2026-11-06，见 §12AQ.4）**：`TestFederationShowInstance` 覆盖本地主机
  （实时计数、softwareName）、已收录远端主机（按存储行聚合 users/notes）、未见主机返回 null、
  匿名 401 与空 host 400（2026-11-07）。
- **第二十八波的部分边角（2026-11-06，见 §12AR.4）**：`TestDriveUsage` 覆盖容量恒 0（无限）、
  两次上传后字节数/文件数一致、删除后同步下降，以及 `alwaysMarkNsfw` / `autoSensitive` 两个开关的
  写入与回读（2026-11-07）。仍未验证的：设置页里「容量 0」的**界面**文案。
- **第二十九波已补验证（2026-11-06，见 §12AS.4）**：真实二进制 + 浏览器走查 `/about`
  （2026-11-07）：实例名、描述、两条 serverRules 都渲染；服务条款/隐私政策/源码仓库/反馈四个外链
  均为 `_blank`；桌面与 390 宽均无横向溢出。
- **第三十波的部分边角（2026-11-06，见 §12AT.4）**：`TestAccountExportImportRoundTrip` 覆盖
  导出 → 删除笔记/收藏/关注行 → 上传导入文件 → 三者恢复、文本与 ID 一致；重复导入不新增；
  非 JSON 与错误格式返回 400（2026-11-07）。仍未验证的：导出文件在**界面**上的下载/上传交互。
- **第三十一波的部分边角（2026-11-06，见 §12AU.4）**：`TestInstanceBranding` 覆盖三者的保存、
  公共 `/api/meta` 回读、非法主题色 400（2026-11-07）。用例暴露了一个缺陷：`admin/update-meta`
  把 `*string` 当作「未提供」，客户端发送 `null` 清空时会被忽略；三字段改为 `nullableField`，
  现在 `null` 与空串都能清空。仍未验证的：图标/横幅/主题色在页面上的**实际展示**。
- **第三十二波已补验证（2026-11-06，见 §12AV.4）**：`TestFederationInstances` 覆盖空库返回空数组、
  按用户数/主机名排序、`host` 过滤（大小写折叠）、`limit`/`offset` 分页与匿名 401（2026-11-07）。
- **第三十三波已补验证（2026-11-06，见 §12AW.4）**：后端断言已加进 `TestNotesEditAndHistory`
  （新笔记 `updatedAt` 为 null，编辑后非 null，这是客户端标记的依据）；浏览器走查 `/notes/<id>`
  看到铅笔图标与提示「于 2026/11/7 01:28:35 编辑」，移动端无溢出（2026-11-07）。
- **第三十四波的部分边角（2026-11-06，见 §12AX.4）**：`TestNotesState` 覆盖收藏与置顶两个标志
  随真实行变化、取消后归零、`isMutedThread` 恒 false（线程静音未实现）、未知笔记 404、缺参数 400、
  匿名 401（2026-11-07）。仍未验证的：笔记菜单里切换失败时的**回退**界面行为。
- **第三十五波的部分边角（2026-11-06，见 §12AY.4）**：`TestDriveUploadFromURLRejectsUnsafeTargets`
  覆盖 SSRF 守卫的每条拒绝路径（非 http(s)、无主机、回环、localhost、私网、无法解析）、
  缺 url 400、匿名 401 与他人文件夹被拒（2026-11-07）。**下载成功分支未验证**：成功抓取需要一个
  公网地址，测试环境里不可构造；`FETCH_FAILED`（远端非 2xx）与 `FILE_TOO_LARGE` 分支同样未构造。
- **第三十六波未测（2026-11-06，见 §12AZ.4）**：公告已读只过了编译层，写入/回读与角标联动未验证。
- **第三十七波未测（2026-11-06，见 §12BA.4）**：公告编辑只过了编译层，保存回读与清空语义未验证。
- **第三十八波未测（2026-11-06，见 §12BB.4）**：SMTP 真投递与验证/重置端到端未执行，只有编译层证据。
- **第三十九波未测（2026-11-06，见 §12BC.4）**：邮箱三页只有编译层证据，渲染与完整链路未走查。
- **第四十波未测（2026-11-06，见 §12BD.4）**：出站投递与取消转贴只有编译层证据；真实双实例投递未验证。
- **第四十一波未测（2026-11-06，见 §12BE.4）**：编辑的 Update 双向都只有编译层证据。
- **第四十二波未测（2026-11-06，见 §12BF.4）**：按作者签名只有编译层证据，双实例未验证。
- **第四十三波未测（2026-11-06，见 §12BG.4）**：四个兼容端点与账号信息页只有编译层证据。
- **第四十四波未测（2026-11-06，见 §12BH.4）**：实例 CSS 链路与邮件状态页只有编译层证据。
- **第四十五波未测（2026-11-06，见 §12BI.4）**：文件详情的四类预览与 `user: null` 分支未走查。
- **第四十六波未测（2026-11-06，见 §12BJ.4）**：缩略图生成与删除只有编译层证据。
- **第四十七波未测（2026-11-06，见 §12BK.4）**：锁推与关注通知只有编译层证据。
- **第四十八波未测（2026-11-06，见 §12BL.4）**：邮件通知与公告群发只有编译层证据。
- **第四十九波未测（2026-11-06，见 §12BM.4）**：六个说明页与菜单分组未走查。
- **第五十波未测（2026-11-06，见 §12BN.4）**：状态栏的渲染与交互未走查。
- **第五十一波未测（2026-11-06，见 §12BP.4）**：迁移页表单与 `MkMoved` 横幅只过编译；`i/move` 的
  真实双实例投递只验证到入队；跨实例的「关注者改关」没有端到端用例；`users/show` 的 `userIds`
  批量参数没有前端调用方。
- **第五十二至五十三波未测（2026-11-06，见 §12BQ.4 / §12BR.4）**：KaTeX 宏页、集成说明页、OAuth
  同意页都只过编译；`GET /oauth/authorize` 无凭据且无前端构建的 401 分支没有用例；没有真实第三方
  客户端跑完整 OAuth 登录。
- **第五十四波未测（2026-11-06，见 §12BS.4）**：远程互动页未走查；`ImportObject` 的拒绝路径
  没有对端用例；真实 Mastodon/Misskey 的 `/authorize_interaction` 链接未实测。
- **第五十五波未测（2026-11-06，见 §12BT.4）**：`Auth.vue` 未走查；真实第三方客户端没跑过这条
  授权流程；`app/show` 返回 secret 的所有者分支没有用例。
- **第五十六波未测（2026-11-06，见 §12BU.4）**：「移除关注者」未走查；`following/list` 没有
  `sinceDate`/`untilDate`。
- **第五十七波未测（2026-11-06，见 §12BV.4）**：新通知类型未走查；远端导入触发通知无双实例用例；
  `homeTimeline` 的 WS 过滤只有单元断言。
- **第五十八波未测（2026-11-06，见 §12BW.4）**：没有对真实 Misskey/Firefish 验证 `EmojiReact`；
  远端自定义表情未解析；`Question` 入站仍未实现。
- **第五十九波未测（2026-11-06，见 §12BX.4）**：远端举报没有对真实实例验证；重复举报与笔记级目标
  的展示未走查。
- **第六十波未测（2026-11-06，见 §12BY.4）**：三个说明页未走查；「24 个占位页全部在范围外」未经
  逐页复核。
- **第六十一波未测（2026-11-06，见 §12BZ.4）**：标签页统计未走查；`hashtags/list` 只验证了两个
  排序；大表聚合耗时未测。
- **第六十二波未测（2026-11-06，见 §12CA.4）**：两个端点没有前端调用方，没有对应走查。

## 12CB. 第六十三波：真实二进制与浏览器走查（2026-11-06）

契约见 `docs/research/2026-11-06-firefish-port-wave63-contract.md`。
这是第五十一至六十二波欠下的视觉验证：真实二进制 + Helium 156（CDP），单实例，schema v21。

### 12CB.1 走查结论

| 页面 | 结论 |
|------|------|
| 登录 → 首页时间线 | **修掉缺陷**：登录前缓存了空 token，登录后仍匿名请求，时间线 401 直到刷新 |
| `/settings/migration` | 表单与说明正常；本地目标被拒并回显 `User is not remote…` |
| 迁移页别名保存与回填 | **修掉缺陷**：本地账号没有 `users.uri`，保存后解析不回昵称；修复后回填 `@w63bob` |
| `/oauth/authorize` | 应用主机与权限正确；接受后 `oauth_grant_codes` 出现 `["read","write"]` 的授权码 |
| `/auth/<token>` | 应用名与权限正确；接受后会话置位、app 绑定令牌生成（2 scope） |
| `/authorize_interaction` | 远端不可达时显示错误态与外链 |
| `/tags/<tag>` | 「被 2 人提到 · 全部时间」，笔记列表正确 |
| `/my/notifications` | 「关注了您」与「发布了新笔记」都渲染出来（第五十七波的 `note` 类型生效） |
| KaTeX 宏页 | 保存写入 `customKaTeXMacro` 与 `customKaTeXMacroParsed`，提示「已保存」 |
| 三个说明页 + `SignupComplete` | 结构与文案正常 |
| 移动端 390×844（7 页） | 全部 `scrollWidth = clientWidth`，无 JS 异常 |

截图：`/tmp/w63-shots`（42 张，不入库）；服务与临时二进制已停止/删除。

### 12CB.2 修复

1. `mirrorToken` 改用 `setToken()`，让 RPC 客户端的内存 token 与 localStorage 同步（登录后无需刷新）。
2. 新增 `Server.userByAPID`：本地账号按 `<baseURL>/users/<id>` 解析（`users/show` 单值/批量、
   `packMovedTo` 共用）；回归断言加进 `TestIKnownAsLocalAlias`。

### 12CB.3 验证证据

- `gofmt -l internal/ cmd/` 无输出；`go vet ./...` 通过。
- `CINDERFISH_TEST_DATABASE_URL=… go test ./...` 全绿（含新增的 URI 回查断言）。
- `vue-tsc --noEmit`、`check:port`（hardTotal 0）、`pnpm build` 全绿。
- 浏览器：无 `Runtime.exceptionThrown`、无 console error；移动端无横向溢出。

### 12CB.4 仍未验证

- `MkMoved` 横幅无法在单实例构造（需要远端账号声明本实例账号为其别名）。
- 双因素登录、移除关注者菜单、远端举报的管理台展示未走查。
- 真实联邦联调仍缺（全部联邦证据来自双实例测试）。

## 12CC. 第六十四波：走查补漏（横幅、移除关注者、举报页、2FA 提示）

第六十三波留下的四项，这一轮用数据库造数据 + 浏览器补测。

### 12CC.1 结果

| 项 | 结果 |
|----|------|
| `MkMoved` 横幅 | 用 SQL 把 bob 的 `moved_to_uri` 指向本地 alice 的 Actor URI，`/@w63bob` 显示「用户已迁移至新账号：@w63alice」。**发现缺陷**：本地目标 `host` 为空时 `MkMention` 会多渲染一个 `@`（显示为 `@w63alice@`），已修 |
| 「移除关注者」菜单 | 在关注者资料页点菜单 → 出现「移除关注者」→ 确认「确定要移除关注者吗？」→ 好；`followings` 中 bob→alice 行消失 |
| 管理台举报页 | `/admin/abuses` 显示举报者 w63bob、被举报者 w63carol、内容「w63 走查举报内容」与「标记已处理」按钮 |
| 双因素登录提示 | carol 开启 TOTP 后，用密码登录会切到 `#cf-signin-token` 输入框（两步验证）。自动填码后完成登录的那一步没有在无头环境里跑通（表单点击时序问题），后端登录链路由测试覆盖 |

### 12CC.2 修复

1. `web/src/components/MkMention.vue`：`host === ""` 时不渲染 `@host` 片段。空的 host 表示本地账号，
   原先只有 `host !== localHost` 的判断会让它渲染成一个孤立的 `@`。
2. `internal/httpapi/web.go`：`/assets/*` 缺文件时返回 404，不再回落到 `index.html`。
   回落会把 HTML 当作 JavaScript 交给浏览器（MIME 错误），也让旧 `index.html` 引用已删除的 chunk
   时表现为白屏。新增用例 `TestWebMissingAssetIsNotFound`。
3. 2FA 复核：服务端 `POST /api/signin`（password + TOTP）→ 200；随后用真实浏览器补测，
   密码提交后切到「2FA 令牌 / 两步验证」输入框，填入验证码提交 → 请求体含
   `{"username","password","token"}`、响应 200、`localStorage.account` 写入、页面进入时间线。
   之前的失败是走查脚本自身的问题（Node 侧把 base32 当 base64 解码，且测试库被 `go test` 清空），
   不是应用缺陷。

### 12CC.3 验证证据

- `vue-tsc --noEmit`、`check:port`（hardTotal 0）、`pnpm build` 全绿。
- 浏览器：横幅文案为「用户已迁移至新账号： @w63alice」（无多余 `@`）；
  `followings` 查询确认移除生效；举报页文案与按钮齐全；2FA 输入框出现。
- 截图追加在 `/tmp/w63-shots`（`desktop-moved-banner-fixed.png`、`desktop-remove-follower.png`、
  `desktop-admin-abuses.png`、`desktop-2fa-signin.png`）。

### 12CC.4 仍未验证

- ~~2FA 完成登录未跑通~~：已在后续补测中跑通（见上）。
- 同一轮里，账号登出后重新登录的 UI 路径没有单独测。
- 无头环境在冷启动时偶发「页面已挂载但 shell 异步组件尚未渲染」的空白期，等待逻辑已改为
  `#app` 有内容后再交互；真实用户不受影响。
- **仍未移植的设置页**：邮箱、整合、导入导出、迁移、Webhook、插件、自定义 KaTeX、状态栏、
  删除账号、应用授权以外的账号操作；对应后端端点不存在或前端缺少依赖（见第五波契约 §3.3）。
- **仍缺的 widgets**：rss、rss-ticker、server-metric、online-users、federation、job-queue、aiscript、
  以及 activity 的子图表；缺服务端代理、Streaming stats 通道或 AiScript 运行时。

- **浏览器可视化走查（Task 8 波次）未做**：2026-10-31 的最终轮，内建浏览器返回 `BROWSER_USER_TAKEOVER`（用户接管面板），按工具契约当轮停止。因此 1440/1280/1100/840/720/600/380 的截图、`scrollWidth` 溢出实测与登录态视觉复核**都没有执行**；同一轮的后端 API 走查（未读数、通知类型、回复可见性、纯转贴反应、四条时间线）已用真实二进制完成，见 §11.15.7。本波次没有改 CSS 或 DOM 结构，风险低，但不能当作已验证。
- **新前端骨架的浏览器验证已部分补上（2026-11-05，第二波）**：§12R 用无头 Chromium + CDP 对
  `/`、`/notes/:id`、`/my/notifications`、`/my/drive`、`/my/follow-requests` 做了桌面（1440）与移动（390）
  两档走查，shell 渲染、主题应用、图标字体、时间线出帖都看过；但**登录/注册表单、设置页、Deck 壳、
  访客壳，以及构建产物（`web/dist` 经 Go 静态托管）仍未走查**，Streaming 只验证了首屏 REST 拉取，
  实时推送（新帖插入、通知红点）没有实测。
- ~~**CfTab 选项卡标签不显示**~~：第三波已修（§12S.2 第 1 条），笔记详情页签实测可见。
- ~~**30 个 Firefish 内置主题只移植了 4 个**~~：第四波移植了剩余 26 个调色板（`themes/builtin/*.json5`
  经 `?raw` 加载），当前可选主题 28 个；只目视检查了 Nord，其余调色板仅验证了解析、校验与打包。
- **主题安装只在本浏览器生效**：`cinderfish.installedThemes` 存 localStorage，没有走 `i/registry`，换设备不同步。
- **i18n 只覆盖 zh-CN / ja-JP**，其余 37 个 locale 未生成；`yaml` devDependency 未装
  （重新生成 locale 前要先 `pnpm add -D yaml`）。
- **S3 后端未做真实服务端验证**：只验证了编译、接口与 URL 生成逻辑。
- ~~**缩略图/WebP 生成未实现**~~：第四十六波实现了缩略图（≤512px，JPEG/PNG）；WebP 仍未做。
- **前端没有真实第三方客户端联调**：页面已在构建产物上用浏览器走查，但没有用 misskey-js 或 Misskey Web 前端连过本实例。
- **Redis 只对 miniredis 验证过**：本机没有 `redis-server`，跨进程中继的单元测试与双进程 smoke 都用 miniredis；没有对真实 Redis 做过持久化/认证/网络闪断验证。
- **Streaming 未做真实 Misskey 客户端联调**：已用自写 WebSocket 客户端验证（握手、`connect`、`connected`、笔记投递全部通过），但**没有**用 misskey-js 或 Misskey Web 前端实际连过。
- **联邦没有任何第三方联调**：只跑了本实现的双实例测试（真签名、真 HTTP、真库），没有对 Mastodon / GoToSocial / 真实 Misskey 实例互通。这是 M5 之后最该补的验证。
- **Streaming 多进程中继只手测了单连接的 LISTEN/NOTIFY 往返**，没有做真正的双进程联调。
- **前端移植第三波已完成的验证（2026-11-05，见 §12S.3）**：发帖链路、个人页五页签、关注/粉丝列表、
  12 个设置子页、收藏/探索/标签/搜索、导航壳、deck 列与 12 个 widget 已落地；桌面 18 条路由与
  移动 4 条路由实测无溢出、无 JS 异常，浏览器真实发帖与设置保存均回读 API 确认。
- **前端移植第四波已完成的验证（2026-11-06，见 §12T.3）**：管理台 4 页、主题选择/安装/管理/编辑器、
  注册表 3 页、自定义 CSS、文字过滤、偏好导出、API 页、欢迎页、公开本地时间线、404 已用真实二进制
  在桌面 20 条 + 移动 7 条路由走查；冻结/解冻、联邦模式切换、白名单增删、注册表建键/改值、主题安装与
  编辑器保存、偏好导出均完成真实 UI 交互并回读 API。
- **第四波仍未验证/未实装（2026-11-06，逐条见 §12T.4）**：登录/注册表单真实交互、emoji 新增的文件上传
  点击路径、Streaming 实时插入、偏好导入/重置、其余 25 个调色板的目视检查；管理台其余 20+ 页、设置页
  约 18 页、widgets 约 10 个、37 个 locale 仍未移植；chat/channels/lists/antennas/clips/groups/gallery/pages
  仍按首版范围不做，导航已不再指向它们。

## 12CD. 第六十五波：远端 @提及（2026-11-07）

契约见 `docs/research/2026-11-07-firefish-port-wave65-contract.md`。

### 12CD.1 落地清单

| 层 | 内容 |
|----|------|
| 迁移 | v22：`notes.remote_mentions`（acct 字符串，创建与编辑都写） |
| 渲染 | `renderNote` 解析远端提及 → `MentionRefs`（进 `to`/`cc` 与 `tag: Mention`）；先查 `users`（`Store.UserByHandle`），再 WebFinger + Actor 抓取 |
| 投递 | `deliveryInboxes` 追加每个远端提及 actor 的 inbox，解析失败只记日志 |
| 入站 | `notifyForNote` 按最强理由定类型（mention 优先于 reply），修掉「提及被记成 reply」 |
| 台账 | 无新增端点（156 / 148 / 292 不变） |

### 12CD.2 有意的限制

- 解析放在渲染与投递时，发帖请求不做网络调用；失败只记日志，笔记照常发布。
- 纯文本 `@user@host` 没有 Mention 标签时对端不会通知（与 Misskey 一致）。

### 12CD.3 验证证据

- `gofmt -l internal/ cmd/` 无输出；`go vet ./...` 通过；`CINDERFISH_TEST_DATABASE_URL=… go test ./...` 全绿。
- `TestRemoteMentionIsDeliveredAndNotifies`：双实例 + 真实投递 worker。beta 的 bob 发
  「hello @alphaalice@alpha.example」，alpha 落库远端副本，alice 收到 `mention` 通知，
  发起者是 betabob。修复前该用例在「对端收不到笔记」处失败。

### 12CD.4 仍未验证

- 真实跨实例（不同软件）的提及互认没有验证。
- 大文本里大量提及的解析开销没有测。
