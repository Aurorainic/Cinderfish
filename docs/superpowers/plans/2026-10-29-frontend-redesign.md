# 前端界面重构实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把 Cinderfish 前端从「AI 味 Mastodon 卡片流」重构为 Firefish/Misskey 式界面：自适应壳（universal 侧边系统 + deck 列容器）、无边框笔记行、Misskey 式反应系统、MFM 补全、全局发帖器，并配以最小后端打包改动。

**Architecture:** 前端按域拆组件（`components/note/`、`components/deck/`、`components/ui/`、`components/composer/`），统一分页组件喂所有列表流，deck 列配置持久化在 localStorage；后端只做「把已有数据打包进现有 API」级别的改动（笔记文件、用户资料、置顶笔记），不新增 schema 之外的存储。

**Tech Stack:** Go 1.26 / Vue 3 + TypeScript + Vite + vue-router（不变）；不新增 npm 依赖。

**Spec:** `docs/superpowers/specs/2026-10-29-frontend-redesign-design.md`（必读，与本计划一起执行）

## Global Constraints

- 不新增 npm 依赖；后端不新增 API 端点（仅 `i/update`、`i/pin`、`i/unpin` 三个 + `users/notes` 加 `withFiles`/`withReplies` 参数）。
- 强调色保留 ember 橙 `#ff6b35`；字体维持 `Inter, "Noto Sans SC", system-ui`；图标维持手写 SVG（`AppIcon.vue`）。
- 去 AI 味清单：删 body 径向渐变光斑、删 `--shadow: 0 8px 30px` 级大投影、卡片圆角从 20px 降到 12px 级、笔记改为无边框行。
- 数据库迁移不可变：v4 已应用，新改动进 v5。
- 文案用中文 + Misskey 术语（笔记/转贴/反应/关注者），docs 中文、代码与标识符英文。
- 每个任务完成即提交：先 `go vet ./...` + `go test ./...`（后端任务加 `CINDERFISH_TEST_DATABASE_URL="postgres://cinderfish:cinderfish@localhost:5432/cinderfish_test"`）；前端任务跑 `cd web && pnpm typecheck && pnpm build`，全绿后 `git commit`。
- 笔记组件必须在 ~350px 窄列与 ~680px 主列都可用（container query，参照 Misskey MkNote）。
- 操作按钮 hover 显隐必须保持 `:focus-visible` 可见（可访问性底线）。

## Review Focus

1. `fileIds` 指向已删除/无权访问的 drive 文件 → 该笔记仍正常渲染（无媒体），不报错。测试：Task B1。
2. deck 列配置是旧版本/空/非法 localStorage 数据 → 回退默认列，不崩溃。测试：Task F5 手动+类型守卫。
3. 流式重连后 PagedFeed 整页重载 → 无重复笔记、无丢单。测试：Task F4（重连回调路径）。
4. `specified` 可见性笔记对非收件人渲染 → 正文正常显示（后端已过滤），前端不依赖 `visibleUserIds`。测试：Task F6 视觉检查项。
5. 迁移幂等：v5 在已应用 v4 的库上执行一次成功、二次执行不再改动。测试：Task B4。

---

## 任务总览与并行分组

- 批次 1（并行）：B1、B2、F1
- 批次 2（并行）：B3、B4、F2
- 批次 3：F6（依赖 F1/B1）
- 批次 4：F4（依赖 F6）
- 批次 5：F3、F5（依赖 F2）
- 批次 6（并行）：F7、F8（依赖 F6）
- 批次 7：F9（依赖 F3/F7）
- 批次 8（并行）：F10、F11（依赖 F4/B4）
- 批次 9：F12 收尾

---

# 后端任务

### Task B1: 笔记文件打包（files 从 fileIds）

**Files:**
- Modify: `internal/media/drive.go`（新增 `GetMany`）
- Modify: `internal/httpapi/notes_pack.go`（`notePackContext` 增加 `Files []*media.File`，`packNote` 填充 `files`）
- Modify: `internal/httpapi/users_pack.go` 附近不需要
- Test: `internal/httpapi/notes_pack_test.go`（如无则新建，参照现有 httpapi 测试风格）

**Interfaces:**
- Produces: `func (s *DriveService) GetMany(ctx context.Context, ids []string) ([]*File, error)` —— 返回与 ids 对应的文件（顺序不保证），调用方按 id 索引；查不到的 id 不出现在结果中。
- Produces: `packNote` 的 `files` 字段为 `[]map[string]any`（复用 `packDriveFile`），无文件时为 `[]any{}`。

- [ ] Step 1: 写失败测试——构造带 `FileIDs` 的 note + 对应 drive 文件，断言 `packed["files"]` 含 url/type/isSensitive，缺失文件被跳过
- [ ] Step 2: `go test ./internal/httpapi/ -run TestPackNoteFiles` 确认 FAIL
- [ ] Step 3: 实现 `GetMany`（`WHERE id = ANY($1)`）与打包填充；`FileIDs` 为空时不查库
- [ ] Step 4: 测试转绿；`go vet ./...`
- [ ] Step 5: `git commit -m "feat(httpapi): pack note files from fileIds"`

### Task B2: 用户资料打包（name / description / avatarUrl）

**Files:**
- Create: `internal/account/profile.go`（Profile 类型与读写）
- Modify: `internal/account/service.go`（如需复用连接）
- Modify: `internal/httpapi/users_pack.go`（`packUserLite` 增加 profile 参数；`packUserDetailed` 填 description）
- Modify: 所有调用 `packUserLite` 的 httpapi 文件（签名变化，grep 更新）
- Test: `internal/account/profile_test.go`（DB 测试，参照 `internal/account` 现有测试）

**Interfaces:**
- Produces:
  ```go
  type Profile struct { Name, Description, AvatarURL string } // 空串表示 NULL
  func (s *Service) GetProfile(ctx context.Context, userID string) (*Profile, error) // 无行返回默认零值，不报错
  func (s *Service) UpdateProfile(ctx context.Context, userID string, name, description, avatarFileID *string) error
  // avatarFileID 非 nil 时：空串→置 avatar_url=NULL；非空→按 drive 文件 id 解析 URL 写入
  ```
- Produces: `packUserLite(u *account.User, host string, prof *account.Profile)`；`prof == nil` 时行为与现状一致（name/avatarUrl 为 nil）。`packUserDetailed` 增加 `description`（来自 profile，调用方传入）。`host` 字段来自 `u.Host`（非空则填字符串）。

- [ ] Step 1: 写失败测试——GetProfile 对无 profile 行用户返回零值；UpdateProfile 写 name/description；avatarFileID 空/非空两条路径
- [ ] Step 2: `go test ./internal/account/` 确认 FAIL
- [ ] Step 3: 实现 profile.go；更新 `packUserLite` 签名与全部调用点（`notePackContext` 里随 author 一起查 profile）
- [ ] Step 4: `go test ./internal/account/ ./internal/httpapi/` 全绿（DB 测试需 `CINDERFISH_TEST_DATABASE_URL`）
- [ ] Step 5: `git commit -m "feat(account,httpapi): pack user profile (name/description/avatarUrl)"`

### Task B3: `i/update` 端点

**Files:**
- Create: `internal/httpapi/i_update.go`
- Modify: `internal/httpapi/server.go`（注册路由）
- Test: `internal/httpapi/i_update_test.go`

**Interfaces:**
- Consumes: B2 的 `UpdateProfile`
- Produces: `POST /api/i/update`，body `{ name?, description?, avatarId? }`（均可选；avatarId 空=清除，非空=drive 文件 id）。响应为 `packMeDetailed` 结果。未登录 401；非本人不可改（无此场景，仅按 token）。

- [ ] Step 1: 写失败测试——更新 name/description 后 `api/i` 返回新值；avatarId 指向他人文件被拒
- [ ] Step 2: 确认 FAIL
- [ ] Step 3: 实现处理器（参照 `internal/httpapi/` 现有 handler 的鉴权与错误风格）
- [ ] Step 4: 测试转绿
- [ ] Step 5: `git commit -m "feat(httpapi): add i/update endpoint"`

### Task B4: 置顶笔记（迁移 v5 + `i/pin`/`i/unpin` + users/notes 过滤参数）

**Files:**
- Modify: `internal/db/db.go`（新增迁移 `5:`：`ALTER TABLE user_profiles ADD COLUMN pinned_note_ids TEXT[] NOT NULL DEFAULT '{}';`）
- Create: `internal/account/pins.go`（PinnedNoteIDs/PinNote/UnpinNote）
- Create: `internal/httpapi/i_pin.go`
- Modify: `internal/httpapi/users.go`（users/show 填 `pinnedNotes`/`pinnedNoteIds`；users/notes 支持 `withFiles`/`withReplies`）
- Test: `internal/account/pins_test.go`、`internal/httpapi/i_pin_test.go`、users 测试扩展

**Interfaces:**
- Produces:
  ```go
  func (s *Service) PinnedNoteIDs(ctx context.Context, userID string) ([]string, error) // 顺序即存储顺序
  func (s *Service) PinNote(ctx context.Context, userID, noteID string) error   // 非本人笔记 → error；已存在 → no-op
  func (s *Service) UnpinNote(ctx context.Context, userID, noteID string) error // 不存在 → no-op
  ```
- Produces: `users/show` 响应含按序 `pinnedNoteIds` 与打包好的 `pinnedNotes`（至多 5 条）；`users/notes` 新参数 `withFiles bool`（默认 false→仅该用户有文件的笔记？注意 Misskey 语义：withFiles=true 过滤出有文件的，false 不过滤）、`withReplies bool`（默认 true；false 时排除 `reply_id NOT NULL AND reply_user_id != user_id` 的笔记）。
- Consumes: `packNote`（B1 后含 files）。

- [ ] Step 1: 写失败测试——pin/unpin 顺序与幂等、非本人报错、迁移在已应用库上可重放
- [ ] Step 2: 确认 FAIL
- [ ] Step 3: 实现迁移与处理器、users/show/users/notes 参数
- [ ] Step 4: `go test ./...` 全绿 + `go vet ./...`
- [ ] Step 5: `git commit -m "feat(db,httpapi): pinned notes (schema v5) and users/notes filters"`

---

# 前端任务

### Task F1: 主题 token 词表扩充 + 全局去 AI 味

**Files:**
- Modify: `web/src/styles/main.css`（token 段与全局样式重写；组件 scoped 样式不动）

**Interfaces:**
- Produces（新增/调整的 CSS 变量，组件层从此引用）:
  ```css
  /* 保留: --accent --accent-soft --accent-strong --on-accent --bg --bg-raised --panel --panel-2
     --border --text --text-strong --muted --danger --success --link --radius-sm --radius --font */
  /* 新增语义 token（明暗两套都给值，参照 Firefish themes/_dark.json5 命名裁剪）: */
  --divider            /* 1px 分隔线颜色，行流用 */
  --panel-header       /* 列头/面板头背景 */
  --nav-bg             /* 侧栏背景（可与 panel 一致） */
  --renote             /* 转贴强调色（青绿） */
  --cw-bg / --cw-fg    /* CW 横幅 */
  --mention --hashtag  /* 与 --link 并列的语义色 */
  --radius-lg          /* 从 20px 收敛为 12px */
  /* 删除: --shadow 大投影（保留一个弱化的 --shadow-sm 用于弹层） */
  ```
- Produces: 全局工具类 `.panel`（面板底+12px 圆角+边框）、`.note-row`（无边框行基类：上下 padding、无圆角）、`.divider-h`；删除 body 的 radial-gradient 背景（纯色 `--bg`）；删除 `.button` 的全胶囊（`999px`→`var(--radius)`，仅小圆片保留胶囊）。

- [ ] Step 1: 按上述词表重写 `main.css` 的 `:root` 与 `:root[data-theme="light"]`；正文/卡片/按钮样式按去 AI 味清单调整
- [ ] Step 2: `cd web && pnpm typecheck && pnpm build` 通过（组件仍在用旧变量名处保证兼容：`--radius-lg` 保留）
- [ ] Step 3: `git commit -m "style(web): expand theme tokens, flatten global aesthetics"`

### Task F2: universal 壳重构（可折叠侧栏 + widget 栏 + 移动端底栏）

**Files:**
- Create: `web/src/stores/shell.ts`（`ShellMode = "universal" | "deck"`，ref + `setShellMode`，持久化 `localStorage["cinderfish.ui"]`）
- Create: `web/src/components/ui/MobileNav.vue`（底部栏：时间线/通知/探索/发帖/我的）
- Modify: `web/src/layouts/AppShell.vue`（侧栏可折叠：≥1280 完整、<1280 图标栏；右栏改为 widget 栏容器；删除实例卡硬编码，改挂 widget 组件）
- Create: `web/src/components/widgets/InstanceCard.vue`（把现有实例卡逻辑搬进来）
- Modify: `web/src/App.vue`（读取 shell.ts；`meta.bare` 路由仍走裸布局）
- Modify: `web/src/styles/main.css`（媒体查询段重组：720px 底栏、1080px 图标栏断点由组件承担）

**Interfaces:**
- Produces: `shellMode: Ref<ShellMode>`、`setShellMode(m)`；widget 栏容器 slot 名 `#widgets`（默认渲染 InstanceCard）。
- Consumes: 现有 `stores/session.ts`、`stores/streaming.ts` 不变。

- [ ] Step 1: shell.ts + InstanceCard + MobileNav
- [ ] Step 2: AppShell 重构（侧栏折叠态：`collapsed` 由媒体查询驱动，不用 JS 监听 resize）
- [ ] Step 3: `pnpm typecheck && pnpm build`；手动验证：720px 出现底栏、发帖入口可点（先 noop）
- [ ] Step 4: `git commit -m "feat(web): universal shell with collapsible rail, widget slot, mobile nav"`

### Task F3: os.ts 命令总线 + PageWindow

**Files:**
- Create: `web/src/os.ts`（`openPostForm`、`confirmDialog`；内部事件总线 + 单例弹层容器，挂载在 App.vue）
- Create: `web/src/components/ui/PageWindow.vue`（弹窗式页面容器：标题栏 + 关闭按钮 + slot 内容；deck 模式的「点开新页」载体）
- Modify: `web/src/App.vue`（挂载 os 弹层容器）

**Interfaces:**
- Produces:
  ```ts
  // os.ts
  export function openPostForm(opts: { reply?: Note; renote?: Note; quote?: boolean }): void;
  export function confirmDialog(message: string): Promise<boolean>;
  // PageWindow.vue: props { title: string }; emits close; slot default
  ```
- Consumes: F9 的 PostForm（本任务先渲染占位 stub，F9 接入真实组件）。

- [ ] Step 1: os.ts 事件总线 + PageWindow 组件
- [ ] Step 2: App.vue 挂载；`pnpm typecheck && pnpm build`
- [ ] Step 3: `git commit -m "feat(web): command bus (os.ts) and page window overlay"`

### Task F4: PagedFeed 统一分页组件 + TimelineFeed 重写

**Files:**
- Create: `web/src/components/ui/PagedFeed.vue`
- Modify: `web/src/components/TimelineFeed.vue`（改为 PagedFeed 的薄封装：fetcher + 流式频道映射；删除内嵌 NoteComposer、手写分页/去重）
- Modify: `web/src/views/ExploreView.vue`、`web/src/views/NotificationsView.vue` 的列表部分换用 PagedFeed（通知的 channel 为 `main`，item 渲染用 NotificationItem）

**Interfaces:**
- Produces:
  ```ts
  // PagedFeed.vue
  props: {
    fetcher: (p: { untilId?: string; sinceId?: string }) => Promise<unknown[]>;
    streamChannel?: string;   // 有则订阅并前插（带「有新笔记」浮标，点击才插入）
    pageSize?: number;        // 默认 20
    disabled?: boolean;       // 未登录等
  }
  slots: #item="{ item }"，#empty；expose: refresh()
  ```
  内置：无限滚动（IntersectionObserver）、去重、`__reconnected` 整页重载、日期分隔行（相邻 item 的 UTC 日期不同时插入，渲染为居中小字）。
- Consumes: `stores/streaming.ts` 不变。

- [ ] Step 1: 实现 PagedFeed（浮标状态：流式来新帖时不直接插入，先累计并显示「N 条新笔记」按钮）
- [ ] Step 2: TimelineFeed 改薄封装；验证：`pnpm typecheck && pnpm build`，手动——时间线正常分页、断网重连后列表无重复、新帖出浮标
- [ ] Step 3: Explore/Notifications 接入
- [ ] Step 4: `git commit -m "feat(web): unified PagedFeed with streaming prepend and date separators"`

### Task F5: deck 列容器 + PageWindow 导航

**Files:**
- Create: `web/src/stores/deck.ts`
- Create: `web/src/layouts/DeckShell.vue`
- Create: `web/src/components/deck/DeckColumn.vue`
- Modify: `web/src/App.vue`（shellMode==="deck" 时主区渲染 DeckShell）
- Modify: `web/src/components/ui/PageWindow.vue` 如需（承接 deck 模式下点开笔记/个人页）

**Interfaces:**
- Produces:
  ```ts
  // stores/deck.ts
  export type ColumnType = "home" | "local" | "hybrid" | "global" | "notifications" | "explore";
  export interface ColumnConfig { id: string; type: ColumnType; }
  export const deckColumns: Ref<ColumnConfig[]>;
  export function addColumn(t: ColumnType, index?: number): void;
  export function removeColumn(id: string): void;
  export function moveColumn(id: string, toIndex: number): void;
  // 持久化 localStorage["cinderfish.deck.columns"]；读取失败/为空 → 默认 [local, notifications]
  ```
- DeckColumn：列头 = 图标 + 名称 + 菜单（切换数据源 / 关闭列）；列体按 ColumnType 映射到 TimelineFeed/通知列表/探索。列宽 350px，横向可滚动；视口宽 ≥ 2×350 时并排多列，<1100px 单列（CSS scroll-snap 横滑）。
- Deck 模式导航规则：列内点击笔记/头像 → PageWindow 覆盖层（非路由跳转）；universal 模式 → 正常路由。

- [ ] Step 1: deck.ts + DeckColumn（列类型→内容映射表）
- [ ] Step 2: DeckShell + App.vue 接入；侧栏加「deck/universal 切换」按钮（写 shell.ts）
- [ ] Step 3: 验证：窄视口单列横滑无横向滚动条溢出（Review Focus 2 的非法配置回退）；`pnpm typecheck && pnpm build`
- [ ] Step 4: `git commit -m "feat(web): adaptive deck columns with persistent config"`

### Task F6: Note 组件拆件 + 去卡片化

**Files:**
- Create: `web/src/components/note/Note.vue`（编排层：转贴头、renote bar、引用、回复上下文）
- Create: `web/src/components/note/NoteHeader.vue`（头像 42px、名字、@handle、相对时间、可见性图标）
- Create: `web/src/components/note/NoteContent.vue`（CW 横幅 + MFM 正文 + 媒体 + 投票 + 引用占位）
- Create: `web/src/components/note/NoteMedia.vue`（1-4 图网格；`isSensitive` → 模糊遮罩点击显示）
- Create: `web/src/components/note/NoteFooter.vue`（回复/转贴/反应/删除；`opacity:0`，hover/focus-within 显现）
- Create: `web/src/components/note/RenoteBar.vue`（纯转贴的顶条：原 po + 时间 + 可见性）
- Create: `web/src/components/note/use-note.ts`（把 NoteCard.vue 的 api 调用逻辑抽成 composable）
- Delete: `web/src/components/NoteCard.vue`、`web/src/components/QuotedNote.vue`（引用改由 Note 渲染 `nested` 态）
- Modify: 全部使用 NoteCard 的文件（TimelineFeed、NoteView、ProfileView 等）

**Interfaces:**
- Produces:
  ```ts
  // use-note.ts
  export function useNote(note: Ref<Note>) {
    return {
      toggleReaction(emoji: string): Promise<void>; // 乐观更新，逻辑搬自 NoteCard.vue:67-96
      renote(): Promise<void>;
      remove(): Promise<boolean>;
      busy: Ref<boolean>;
    };
  }
  // Note.vue: props { note: Note; nested?: boolean; detail?: boolean }
  // emits: reply(note)
  ```
- 样式要求：行式布局（左右 padding 14px、无卡片边框圆角）、1px `--divider` 分隔、renote 条用 `--renote` 色、CW 用 `--cw-bg/--cw-fg` 横幅、container query 按 580/500/450px 降级。

- [ ] Step 1: use-note.ts + NoteHeader + NoteContent + NoteMedia
- [ ] Step 2: NoteFooter + RenoteBar + Note 编排；迁移现有调用点，删除旧组件
- [ ] Step 3: `pnpm typecheck && pnpm build`；手动：时间线/详情/引用/纯转贴四种渲染 + Review Focus 4（specified 笔记正常显示）
- [ ] Step 4: `git commit -m "feat(web): split note into composable parts, borderless rows"`

### Task F7: 反应系统（emoji 目录 + picker + 反应条）

**Files:**
- Create: `web/src/lib/emoji-catalog.ts`（精选 unicode 目录 ~300 个，按分类分组）
- Create: `web/src/components/ui/EmojiPicker.vue`
- Modify: `web/src/os.ts`（`openReactionPicker(cb: (emoji: string) => void)` 单例）
- Modify: `web/src/components/note/NoteFooter.vue`（反应按钮 → picker；反应条由 NoteContent 渲染）

**Interfaces:**
- Produces:
  ```ts
  // emoji-catalog.ts
  export interface EmojiEntry { char: string; name: string }
  export const EMOJI_CATEGORIES: { name: string; emojis: EmojiEntry[] }[];
  export function searchEmojis(q: string, limit?: number): EmojiEntry[];
  export function useRecentEmojis(): { recent: Ref<string[]>; push(char: string): void };
  // EmojiPicker.vue: emits select(char: string)；布局：搜索框 + 分类页签 + 最近使用置顶
  ```
- 反应条视觉（对齐 MkReactionsViewer）：32px 高胶囊、emoji 1.5em、自己的反应排最前 + accent 实心底 + `inset 1px` 描边、计数同色；新增反应 `scaleIn` 动画。

- [ ] Step 1: emoji-catalog.ts（数据内嵌，不引入依赖）+ useRecentEmojis
- [ ] Step 2: EmojiPicker + os.openReactionPicker
- [ ] Step 3: 接入 Footer/反应条；`pnpm typecheck && pnpm build`；手动：picker 搜索、最近使用、点击反应乐观更新
- [ ] Step 4: `git commit -m "feat(web): emoji picker and Misskey-style reaction pills"`

### Task F8: MFM 函数补全

**Files:**
- Modify: `web/src/lib/mfm.ts`（保持占位符 stash 安全管线与现有 escape 语义）

**Interfaces:**
- Produces: `renderMFM(text: string): string` 签名不变；新增支持：
  - `$[x2 x3 x4 text]`（font-size 缩放）、`$[spin]`（含 `alt`/`left` 参数）、`$[shake]`、`$[jump]`、`$[blink]`、`$[rainbow]`、`$[blur]`、`$[center]`、`$[small]`、`$[fg.color=hex]`、`$[bg.color=hex]`、`$[font.monospace]`/`$[font.serif]`、`$[plain]`
  - 未支持的 `$[fn]` 按其内文纯文本渲染（降级，Review 口径）
  - `:shortcode:` 维持纯文本（自定义 emoji 注册表推迟，见 spec 非目标）
- 动效用 CSS class + keyframes（如 `.mfm-spin`），尊重 `prefers-reduced-motion`。

- [ ] Step 1: 在 mfm.ts 的函数分发处按上表实现（嵌套函数复用现有递归）
- [ ] Step 2: `pnpm typecheck && pnpm build`；手动渲染样例文本核对（x2/spin/blur/rainbow/未知函数降级）
- [ ] Step 3: `git commit -m "feat(web): MFM functions (x2-x4, spin, shake, jump, blink, rainbow, blur, colors)"`

### Task F9: 发帖器全局化 + 引用/草稿/specified

**Files:**
- Create: `web/src/components/composer/PostForm.vue`（从 `NoteComposer.vue` 迁移 + 扩展）
- Delete: `web/src/components/NoteComposer.vue`
- Modify: `web/src/os.ts`（openPostForm 接入真实 PostForm）
- Modify: `web/src/components/TimelineFeed.vue`（顶部固定形态复用 PostForm `fixed` 模式）

**Interfaces:**
- Produces:
  ```ts
  // PostForm.vue: props { replyTo?: Note | null; renoteTo?: Note | null; fixed?: boolean }
  // emits: posted(note), cancel()
  ```
  - 引用：`renoteTo` 时头部显示「引用」；提交带 `renoteId` + 正文（先验证后端接受 renoteId+text，若被拒则改为 renote+text 双条策略并在任务内记录）
  - 草稿：输入去抖 800ms 存 `localStorage["cinderfish.draft"]`（text/cw/visibility/localOnly/fileIds），打开时若有草稿提示恢复
  - `specified`：可见性下拉增加「指定」；选定时显示收件人输入（@handle 逗号分隔，提交前经 `api.usersShow` 解析为 userIds；解析失败的 handle 给出行内错误）
  - 可见性控件从 `<select>` 改为图标+文字的分段按钮（public/home/followers/specified 四态）
- Consumes: F3 的 openPostForm。

- [ ] Step 1: 迁移 PostForm（逻辑不变）+ 全局对话框接入
- [ ] Step 2: 引用 + 草稿 + specified 收件人解析
- [ ] Step 3: `pnpm typecheck && pnpm build`；手动：回复/引用/草稿恢复/specified 发帖
- [ ] Step 4: `git commit -m "feat(web): global post form with quote, drafts, specified recipients"`

### Task F10: 通知页改造

**Files:**
- Modify: `web/src/views/NotificationsView.vue`（换 PagedFeed；类型筛选 tab：全部/反应/提及/转贴）
- Modify: `web/src/components/NotificationItem.vue`（行式排版对齐新视觉；反应 emoji 用 picker 同源目录渲染）

- [ ] Step 1: PagedFeed 接入（客户端筛选 tab，无新后端参数）
- [ ] Step 2: NotificationItem 重排 + `pnpm typecheck && pnpm build`
- [ ] Step 3: `git commit -m "feat(web): notification page with type filters"`

### Task F11: 个人页 + 设置页

**Files:**
- Modify: `web/src/views/ProfileView.vue`（头部：头像/姓名/@host、简介、关注按钮；置顶笔记区；tab：笔记 / 媒体（withFiles）/ 回复（withReplies=false））
- Modify: `web/src/views/SettingsView.vue`（新增「个人资料」段：name/description 编辑、头像上传 → `api.uploadFile` + `api.iUpdate`）

**Interfaces:**
- Consumes: B2/B3/B4 的 `packUserLite`（name/avatarUrl/description）、`pinnedNotes`、`users/notes withFiles/withReplies`；client.ts 需新增 `api.iUpdate(params)`、`api.pin(noteId)`、`api.unpin(noteId)`（NoteFooter 菜单加「置顶」项，本人笔记）。

- [ ] Step 1: client.ts 加三个 API 方法
- [ ] Step 2: ProfileView 改造（pinned 区 + 三 tab）
- [ ] Step 3: SettingsView 资料段；`pnpm typecheck && pnpm build`；手动：改名字/头像后个人页与笔记头部即时更新
- [ ] Step 4: `git commit -m "feat(web): profile tabs with pins, settings profile editor"`

### Task F12: 术语与收尾打磨

**Files:**
- Modify: 全局文案 sweep（「说点什么」→「写点什么吧？」Misskey 风；「转贴」「反应」「笔记」统一）
- Modify: 残留的旧视觉（AuthLayout 的渐变侧板、NotFound、PagePlaceholder 等对齐新 token）
- 验证：`go vet ./...`、`go test ./...`（含 DB/Redis 变量）、`pnpm typecheck && pnpm build`、宽窄视口与明暗主题全页走查

- [ ] Step 1: 文案与残留样式 sweep
- [ ] Step 2: 全量验证 + `git commit -m "chore(web): terminology sweep and final polish"`
- [ ] Step 3: 更新 `docs/HANDOVER.md` 前端里程碑状态
