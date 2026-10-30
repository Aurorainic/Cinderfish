# UI 框架复刻 Firefish/Iceshrimp 执行计划（2026-10-30）

状态：执行中。差距依据 `docs/research/2026-10-30-firefish-ui-gap-analysis.md`；
设计约束见 `docs/superpowers/specs/2026-10-29-frontend-redesign-design.md`。

## 目标与边界

把 web 前端的**框架层**复刻为 Firefish/Iceshrimp 形态：壳层分区与断点、导航注册表化、
widget 体系、笔记呈现框架、页面头部与分区、主题语义令牌。

边界：
- 后端仍只按 Misskey API 契约改动，不照搬 Misskey 前端。
- 参考 Firefish（主要）与 Iceshrimp（补充），两者综合考虑。
- 保留：ember 橙、Inter、手写 SVG 图标、CSS 变量双主题、中文文案。
- 不新增 npm 依赖（不引拖拽库、不引 UI 框架、不引虚拟滚动库）。

## 文件所有权（避免并行冲突）

波次内每个任务独占一组文件，任何任务不得修改他人拥有的文件。

| 任务 | 独占文件 |
|------|----------|
| A1 | `web/src/styles/main.css`、`web/src/index.html`、`web/src/main.ts`、新建 `web/src/stores/theme.ts` |
| A2 | 新建 `web/src/nav.ts`、新建 `web/src/stores/nav.ts`、新建 `web/src/components/ui/NavDrawer.vue`、`web/src/components/ui/MobileNav.vue` |
| A3 | `web/src/components/note/*`、`web/src/components/UserAvatar.vue`、新建 `web/src/lib/hotkeys.ts`、新建 `web/src/lib/focus-sibling.ts` |
| A4 | 新建 `web/src/widgets/registry.ts`、新建 `web/src/stores/widgets.ts`、新建 `web/src/components/widgets/*` |
| A5 | 新建 `web/src/components/ui/PageHeader.vue`、新建 `web/src/composables/use-page-title.ts` |
| B1 | `web/src/layouts/AppShell.vue`、`web/src/layouts/DeckShell.vue`、`web/src/components/deck/DeckColumn.vue`、`web/src/stores/deck.ts`、`web/src/App.vue` |
| B2 | `internal/httpapi/notifications.go`、`internal/note/reaction.go`、`web/src/api/client.ts`（仅 `notifications()` 签名） |
| C1..C4 | 各自的 `web/src/views/*` 与新建的设置/管理子视图 |

## 波次

### 波次 A —— 基础层（5 个任务并行，文件互不重叠）

| ID | 任务 | 验收 |
|----|------|------|
| A1 | 主题与设计令牌：首帧内联脚本消除闪白、`color-scheme`、`stores/theme.ts` 统一控制（storage + matchMedia 同步 + meta theme-color）、尺度令牌替换裸值、补语义 token、删除 `--panel-header`/`--content-max` | `pnpm typecheck && pnpm build`；`grep -rn "var(--panel-header\|var(--content-max" web/src` 为空；唯一硬编码色 grep 通过 |
| A2 | 导航注册表与移动抽屉：单一 `navItemDef`、桌面/移动两组偏好（sanitize + 写进 `cinderfish.ui` 的 `nav` 键，合并写）、`MobileNav` 消费注册表并增加菜单键、`NavDrawer` 复用同一份数据并承载设置/管理/主题切换/通用↔deck/退出 | 类型通过；`MobileNav` 与 `NavDrawer` 不再各自硬编码列表；≤720px 时设置与管理可进入 |
| A3 | 笔记组件层：article `id`/`tabindex`/`aria-label`、反应条并入 footer、容器查询统一 500/380、头像尺寸令牌化 + `radius` prop、hover 0.7→1 且 `focus-within`、反应胶囊 nowrap+mask+`canToggle`、引文单层化、`NoteParent` 父帖上下文、热键与焦点兄弟跳转、长文折叠 | 类型通过；350px 容器内不溢出；`UserAvatar` 旧调用点仍工作 |
| A4 | widget 体系：`registry.ts`、`stores/widgets.ts`、5 个 widget（instance/links/memo/notifications/timeline）、`WidgetStack`（查看态 + 编辑态上下移/删除/添加）、`WidgetsDrawer` | 类型通过；widget 数据全部来自现有 API 或 localStorage，无假数据 |
| A5 | 共享原语：`PageHeader.vue`（sticky + tab 行 + `#actions` + 窄屏折叠 + 返回）与 `use-page-title.ts`（写入 `document.title`） | 组件可在 500px 以下折叠；类型通过 |

### 波次 B —— 壳层接线（2 个任务并行）

| ID | 任务 | 验收 |
|----|------|------|
| B1 | 壳层接线与几何：消费 A2 的注册表与抽屉、A4 的 `WidgetStack`/`WidgetsDrawer`；deck 高度改 flex 链（删 `108px`/`104px`）；`.rail` 加 `overflow-y`；deck 模式 `.main` padding 归零；`max-width` 令牌化；deck 列支持宽屏并排/窄屏横滑/点击回顶/上下移排序 + 新增 `widgets` 列类型 | 1440px 与 720px 两档无横向溢出；deck 列可在窄屏横滑吸附 |
| B2 | 通知筛选下推：后端 `includeTypes` 真正生效，前端 `notifications()` 带参数，去掉渲染层过滤；同时核对 `markAsRead` 语义是否符合 misskey 契约 | `go test ./...` 绿；通知页切换类型不再偷偷翻页 |

### 波次 C —— 页面层（4 个任务并行）

| ID | 任务 | 验收 |
|----|------|------|
| C1 | 时间线/探索/通知三页接入 `PageHeader`，tab 写入 `route.query` 并可回填 | 刷新后 tab 保持；后退回到上一个 tab |
| C2 | 个人页与笔记页：副栏、计数链接、统一翻页、`NoteDetail` 抽出供 deck 覆盖层复用 | 个人页窄屏并入主列；笔记页上下文可用 |
| C3 | 设置页拆分为 `SettingsLayout` + `SettingsMenu` + 5 个子页 + 嵌套路由 | 设置不再是一条长滚动；危险操作在独立分区 |
| C4 | 管理页拆分 + 登录/注册/404 收尾 | 管理页左导航；三页与整体风格一致 |

### 波次 D —— 验收

1. `gofmt -l`、`go vet ./...`、带 DB/Redis 的 `go test ./...`。
2. `cd web && pnpm typecheck && pnpm build`。
3. 真实实例浏览器实测：1280/1440 宽三区、1100 以下隐藏 widget、720/500 以下五项底栏 + 抽屉、
   deck 宽屏多列与窄屏横滑、明暗双主题、页面无横向溢出。
4. 截图留证，更新 `docs/HANDOVER.md`。

## 已知后端缺口（本计划内不实现，前端如实降级）

- 无 `users/following`、`users/followers`、`users/search`、`notes/mentions`、`notes/children`、
  `notes/featured`、`notes/reactions`。
- `InstanceMeta` 缺 `disableLocalTimeline` / `enableGuestTimeline`，时间线 tab 只做登录态门控。
- 自定义 emoji 注册表未实现，`:shortcode:` 未命中时按纯文本显示。

## 每个任务的通用要求

- 完成后立即提交，提交消息说明做了什么与证据。
- 提交前：后端 `go vet ./...` + 测试；前端 `cd web && pnpm typecheck && pnpm build`。
- 代码与标识符英文，文档中文；不写解释性注释。
- 引用参考实现时给出 `tmp/refcheck/...` 的具体路径，不凭印象。
