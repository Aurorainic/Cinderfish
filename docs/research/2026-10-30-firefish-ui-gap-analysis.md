# Firefish / Iceshrimp UI 框架差距分析（2026-10-30）

目的：为「UI 框架复刻 Firefish/Iceshrimp」定位现有实现与参考的具体差距。

参考边界：Misskey 只提供后端 API 契约；前端以 Firefish（主要）与 Iceshrimp（补充）为准。
参考源码位于 `tmp/refcheck/`（`.gitignore` 覆盖）。

分析由只读子代理完成，五份报告按维度归档。所有行号可在工作树与 `tmp/refcheck/` 中复核。

## 壳层（AppShell / DeckShell / 导航）

已完成只读比对。两个壳层文件、Firefish 四个参考文件、Iceshrimp.NET 的 `Layout/` 全部读完，另补读了 deck 列组件、widget 注册表、`navbarItemDef` 与 `deck-store`。

> 引用前缀说明：`web/src/…` 指 `.worktrees/frontend-ui-redesign/web/src/…`；`firefish/…` 指 `/home/lainsaka/文档/MilnFish/tmp/refcheck/firefish/packages/client/src/…`；`iceshrimp/…` 指 `/home/lainsaka/文档/MilnFish/tmp/refcheck/iceshrimp/Iceshrimp.Frontend/Layout/…`。

### 结论摘要

1. **移动端丢功能**：≤720px 时 `.rail` 直接 `display:none`（`web/src/layouts/AppShell.vue:305-307`），而设置/管理/主题切换/通用↔Deck 切换/退出登录全在 `.rail-foot`（`AppShell.vue:86-106`）里。Cinderfish 没有抽屉，这三类操作在手机上没有任何入口。
2. **导航有两个互不一致的硬编码真源**：rail 是 `AppShell.vue:25-33`（时间线/通知/探索/设置/管理），底栏是 `MobileNav.vue:16-40`（时间线/通知/探索/发帖/我的）。Firefish 只有一份 `navbarItemDef` 注册表 + 用户可排序的 `menu`。
3. **deck 高度是三个文件的魔法数之差**：`100dvh - 108px`（`DeckShell.vue:104`）= 24×2 shell padding（`main.css:150-157`）+ 60px `.main` padding-bottom（`main.css:242-245`）；移动端又换成 `- 104px`（`DeckShell.vue:224`），与 shell 预留的 72px+safe-area（`AppShell.vue:302`）对不上。
4. **widget 栏只是硬编码一张卡片**：`#widgets` slot 从来没人填（全仓只有 `InstanceCard.vue:9` 的注释提到），没有注册表、编辑模式、抽屉，deck 里还被强制隐藏（`AppShell.vue:321-323`）。
5. **列容器缺 Firefish 的一半能力**：无拖拽排序、无点击回顶、无 per-column 宽度/`flexible`、无堆叠文件夹、无 `main` 列、无 profile 多套布局。

---

### 1. 布局分区

| Firefish / Iceshrimp 的做法 | Cinderfish 现状 | 差距 | 建议动作 |
|---|---|---|---|
| universal：`XSidebar` + `.contents`（`MkStickyContainer`，header 放 `XStatusBars`）+ `main#maincontent` + `.widgets-container`，另挂 `XCommon` 与各抽屉（`firefish/ui/universal.vue:2-208`）；整体 `display:flex; min-block-size:100dvb`，内容列 `inline-size:100%` 不被封顶（`universal.vue:542-679`）；classic 模式单列 750px 居中（`universal.vue:579-660`） | `.shell` 用 grid `232px 1fr 300px`，gap 28，`max-width:1240px` 居中（`web/src/styles/main.css:150-157`）；模板四块：`aside.rail` / `main.main` / `aside.widgets` / `MobileNav`（`web/src/layouts/AppShell.vue:60-118`） | ①没有跨页 sticky 顶栏与 page-metadata 通道，页面标题由每个视图自己画 `header.page-title`（`web/src/views/TimelineView.vue:40-43`，样式在 `main.css:401-411`）；②宽度封顶 1240px，宽屏两侧空；③无 classic 单列形态 | ①新增 `web/src/components/ui/PageHeader.vue`（`position:sticky; top:0; z-index:20`，`title` + `#actions`），替换 6 个视图里的 `header.page-title`：`TimelineView.vue:40`、`NotificationsView.vue`、`ExploreView.vue`、`SettingsView.vue`、`AdminView.vue`、`NoteView.vue`（grep `.page-title` 即可定位全部）；②`main.css:154` 的 `max-width: 1240px` 改为 `max-width: var(--shell-max, 1440px)`，并在 `AppShell.vue:317-319` 的 deck 规则里补 `max-width: none`；③classic 形态不进本次范围，但把 `stores/shell.ts:7` 的 `ShellMode` 注释写明"third mode intentionally omitted" |
| deck：`XSidebar` + `.main`（`.statusbars` + `.columns`），接管整页滚动 `documentElement.style.overflowBlock="hidden"`（`firefish/ui/deck.vue:1-13, 303, 409-437`） | deck 只替换 `main` 内容，rail 与 shell padding 全部保留（`AppShell.vue:109-111`、`App.vue:19-27`） | deck 仍被 24px 页面内边距与圆角卡片包着，信息密度低于参考；参考是列贴边、5px 分隔线（`deck.vue:457-474`） | 若保留卡片式列（当前 `.deck-column` 有 border+radius，`DeckColumn.vue:165-177`），在 deck 模式把 `.main` 的 padding 归零：`AppShell.vue:317` 区块加 `.shell[data-shell="deck"] .main { padding-bottom: 0; }`，并同步修下一个表格里的高度计算 |
| Iceshrimp：`.page` flex row = sidebar \| main（`.top-bar` + `@Body`）\| `.widget-panel`（`iceshrimp/MainLayout.razor:23-51`，`MainLayout.razor.css:1-26`）；顶栏 sticky 3rem，标题/图标/actions 由 `SectionOutlet` 注入（`MainLayout.razor:32-41`，`.css:28-48`） | 无顶栏；widget 列与 rail 平级 | 参考是"每页都有顶栏"，Cinderfish 是"每页自画标题"，模块之间没有统一的 actions 挂载点 | 同上一行的 `PageHeader.vue`：内部提供 `#actions` slot，并允许视图通过 `provide/inject` 注册标题，避免每页重复 `<header>` 标记 |

### 2. 宽度断点

| Firefish / Iceshrimp 的做法 | Cinderfish 现状 | 差距 | 建议动作 |
|---|---|---|---|
| rail 两档宽 250/80px，`iconOnly` 触发条件 = 视口 ≤1279 **或** 用户偏好 `menuDisplay==='sideIcon'`（`firefish/ui/_common_/navbar.vue:179-190, 251-252`；偏好定义 `firefish/store.ts:307-310`） | 1280 断点，232→76px，只有媒体查询一个触发源（`AppShell.vue:234-287`） | 没有用户偏好开关；两个断点值（1280 / 1100）分散在 `AppShell.vue` 两处，deck 列宽断点又在第三处 | 在 `stores/shell.ts` 的 `cinderfish.ui` 记录里加 `railMode: "full" \| "icon" \| "auto"`（该文件 18-47 行已有读写框架，直接复用），`AppShell.vue:234` 的媒体查询块外再补一组 `.shell[data-rail="icon"] .brand-name, .shell[data-rail="icon"] .nav-label { display:none }` 规则，由 `:data-rail` 驱动 |
| widget 列 `max-inline-size: 1090px` 隐藏（`universal.vue:544, 676, 700`）；500–1100 之间浮出右下角 widget 按钮（`universal.vue:26-33`）；移动断点 = ≤500 或 `deviceKind==='smartphone'`（`universal.vue:236-252`）；deck 移动端也用 500（`deck.vue:227-230`） | widget 列 ≤1099 隐藏（`AppShell.vue:294-296`）；移动端 ≤720（`AppShell.vue:299`、`MobileNav.vue:49`）；deck 列 100% ≤1099（`DeckColumn.vue:350-354`）；deck strip 高度 ≤720（`DeckShell.vue:222`） | ①720 比参考的 500/600 激进，600–720 已被当成手机；②720–1099 区间 rail 是图标态且没有 widget 入口，是"信息最贫瘠"的一档；③`1090` 在 `universal.vue:544` 与 665 重复声明（第二个覆盖第一个），不要照抄这种写法 | ①新建 `web/src/breakpoints.ts` 导出 `RAIL_BP = 1280, WIDGET_BP = 1100, MOBILE_BP = 720`，CSS 的 media query 无法读变量，所以在三个文件的 media 行尾加 `/* breakpoints.ts:MOBILE_BP */` 指向唯一登记点，JS 侧（第 7 项的抽屉、第 6 项的高度同步）用同常量；②720–1099 补 widget 入口：见第 7 项建议第 4 条 |
| Iceshrimp：840 / 600 两档——≥840 显示 widget 列与文字标签，<840 隐藏 widget 列并只留图标按钮（`MainLayout.razor.css:96-111`），<600 出底栏并把侧栏改成抽屉（`Sidebar.razor.css:163-205`） | 三档 1280 / 1100 / 720 | Cinderfish 没有"只藏文字标签、保留动作按钮"的中间态（Iceshrimp `.top-bar .action .label{display:none}`，`MainLayout.razor.css:104-106`），窄屏时动作直接消失 | 在 `PageHeader.vue` 里照此实现：`@media (max-width:1099px) .page-header .label { display:none }`，图标按钮保留 |

### 3. 侧栏折叠形态

| Firefish / Iceshrimp 的做法 | Cinderfish 现状 | 差距 | 建议动作 |
|---|---|---|---|
| `iconOnly` class 单点切换：外层 `.sidebar.iconOnly` 改 `flex-basis`/`inline-size`，`body` 内 `inline-size` 同步（`navbar.vue:2, 483-490`）；图标态下文字 `display:none`、item 改居中、hover 用圆形 `--accentedBg` 胶囊（`navbar.vue:594-648`），发帖按钮变 52px 圆形渐变 FAB（`navbar.vue:512-553`） | 媒体查询里逐条改：`.brand-name/.nav-label{display:none}`、`.nav-link{justify-content:center}`、`.account-chip` 变圆形、`.compose` 变 40px 圆（`AppShell.vue:234-287`），样式在 `main.css` 里（`main.css:160-168, 203-233`） | 折叠逻辑分居两处（结构在组件、样式在全局 main.css），新增导航项必须同时改两个文件；折叠后没有 hover 胶囊反馈 | 把 `.brand/.nav/.nav-link/.rail-foot` 这一组规则从 `main.css:160-233` 迁进 `AppShell.vue` 的 scoped 样式（组件已经用 scoped 覆盖全局 `.shell`，见 `AppShell.vue:234-237`，方式一致），折叠态加 `.nav-link:hover { background: var(--accent-soft); }` |
| `.body` 是 `sticky` + `contain: strict` + `overflow: auto`，导航项多时在栏内滚动，底部按钮区 `padding-block: 20px` 常驻（`navbar.vue:257-267, 508-582`） | `.rail` 是 flex column，`height: calc(100vh - 48px)`，**没有** `overflow`（`main.css:160-168`）；底部靠 `.rail-foot{ margin-top:auto }`（`main.css:235-241`） | 导航项增加到 7 个、或通知 badge 撑高后，`.rail` 内容会溢出到 grid 之外并被视口裁掉，footer 按钮被推出可视区 | `main.css:160-168` 加 `overflow-y: auto; overflow-x: hidden; overscroll-behavior: contain;`，`.nav{ min-height: 0 }`（现 `main.css:203-207`），并保留 `margin-top:auto`——修完第 4 项的注册表后应有至少 8 项，必须先修这个 |
| Iceshrimp：不做 icon-only，只做 off-canvas——`popover="auto"` 的 `.sidebar`，<600 时 `transform: translateX(-100%)`，`@starting-style` 负责入场，路由变化后 `HidePopoverAsync` 关闭（`Sidebar.razor:13, 160-163`；`Sidebar.razor.css:176-205`） | 无抽屉形态 | 手机端完全没有侧栏形态可用 | 见第 6 项；抽屉可实现为 `web/src/components/ui/NavDrawer.vue`，用 `Teleport` + `transition`（不用 popover API，因为要对齐 Firefish 的 `deck.vue:171-182` 动画），`App.vue` 或 `AppShell.vue` 顶层挂载一份 |

### 4. 导航数据来源与可配置性

| Firefish / Iceshrimp 的做法 | Cinderfish 现状 | 差距 | 建议动作 |
|---|---|---|---|
| 一份 `navbarItemDef` 注册表，每项 `title/icon/show(computed)/indicated(computed)/to\|action`（`firefish/navbar.ts:10-140`）：`show` 做登录门控（`:14, 21, 28, 34, 57`），`indicated` 做未读点（`:15, 23, 46`），`action` 承载非路由动作（`search:49-53`、`ui:98-132`、`reload:133-139`） | 组件内硬编码 computed 数组（`AppShell.vue:25-33`），只有 `isAdmin` 一条门控（`:32`），只有通知有 badge（`:78-80`，数据来自 `stores/streaming`） | ①无 `show`：未登录也显示"设置"（`:30`）；②无 `action` 抽象，主题/布局/退出写成三个内联按钮（`:86-106`）；③两个壳层的列表不同（见下） | 新建 `web/src/nav.ts`：`export interface NavItem { id: string; title: string; icon: IconName; to?: string; action?: () => void; show?: () => boolean; indicated?: () => boolean }` + `export const NAV_ITEMS: NavItem[]` + `export function visibleNav(): NavItem[]`。把 `AppShell.vue:26-32` 的 4 项迁移过来，`ui`/`theme`/`logout` 用 `action` 表达（参照 `navbar.ts:98-132`） |
| 用户可排序的 `menu: string[]`，默认 `menuOptions`，`"-"` 渲染成分隔线（`firefish/store.ts:29-37, 138-141`；渲染 `navbar.vue:37-38`）；不在 menu 里的项通过"更多"LaunchPad 到达，`otherMenuItemIndicated` 汇总它们的未读（`navbar.vue:106-117, 171-177, 237-246`） | 顺序写死，无隐藏/排序概念 | 无任何可配置性 | 在 `stores/shell.ts` 的 `cinderfish.ui` 里加 `menuOrder: string[]`，默认 `["home","notifications","explore","settings"]`；`visibleNav()` 先按 `menuOrder` 排序、再过滤 `show()`；`"-"` 分隔线只在 Settings 里出现时再实现（先不渲染） |
| 管理入口按 `isModerator` 分流到 `/admin` 或 `/admin/emojis`，并用 5 个实例健康信号合并成一个未读点（`navbar.vue:74-105, 201-207`） | 只有 `isAdmin ? [...items, /admin]` 一项（`AppShell.vue:32`），无未读指示 | 管理入口无任何提示；且该 URL 不在底栏 | `nav.ts` 里把 `/admin` 项写成 `show: () => isAdmin.value`、`indicated: () => pendingReports.value`；本期可先 `return false`（先不加后端查询），但 `show` 门控必须加，否则未登录用户点击 404 |
| 抽屉复用同一注册表（`navbar-for-mobile.vue:36-71`），并按 `show !== false` 过滤 | `MobileNav.vue:16-40` 另一套列表，且集合与 rail 不同（多了"我的"、少了"设置/管理"），无 `show` 过滤 | 两份真源已经跑偏：加一个页面要改两处，且手机端少了两个入口 | `MobileNav.vue:16-40` 改为消费 `visibleNav().filter(i => i.mobile !== false)` 取前 4 项，第 5 项固定为"我的"（`MobileNav.vue:37-40` 的 `meRoute` 保留），发帖按钮维持 `MobileNav.vue:33-36` |
| Iceshrimp：手写 `NavLink` 列表 + 条件渲染（`iceshrimp/Sidebar.razor:21-86`），未读靠 5–60s 轮询 `GetStatusAsync`（`:135-151`） | badge 走 streaming 推送（`AppShell.vue:78-80` 来自 `stores/streaming`） | Cinderfish 的实时性优于 Iceshrimp，这点保持 | 无需改动；`indicated` 函数直接读 `unreadNotifications` |

### 5. deck 列容器行为

| Firefish / Iceshrimp 的做法 | Cinderfish 现状 | 差距 | 建议动作 |
|---|---|---|---|
| 横向容器 `.columns`：`overflow-x:auto; overflow-y:clip`，垂直滚轮转横向 `scrollLeft += deltaY`（`firefish/ui/deck.vue:297-301, 432-437`，`.self` 限制只在空白触发 `:12`）；列间 5px 分隔条（`--deckDividerThickness`，`deck.vue:415, 457-474`）；支持 left/right/center 对齐（`deck.vue:10, 447-455`） | `.deck-strip` flex + gap 12 + `scroll-snap-type: x mandatory`，`overflow-y: hidden`（`DeckShell.vue:100-113`） | ①滚轮不能横向滚（触控板之外只能拖滚动条）；②桌面多列可见时 `scroll-snap: mandatory` 会在轻微横滚后强制吸附，参考只在单列占满时才需要；③无对齐选项、无分隔条 | `DeckShell.vue:106-112` 加 `@wheel` 处理：`function onStripWheel(e: WheelEvent){ if (e.deltaX === 0) strip.scrollLeft += e.deltaY }`（需给 `.deck-strip` 加 `ref`，模板第 42 行）；把 `scroll-snap-type` 从 `:108` 移到 `@media (max-width:1099px)` 块内（该文件已有 720 块，`:222`，新增 1099 块） |
| 每列独立 `width`，新列默认 330（`deck.vue:253-283`）；`flexible` 时 `flex:1; minWidth:350px`（`deck.vue:19-33, 52-62`）；列设置表单可改 name/width/flexible（`column.vue:154-172`） | 硬编码 `flex: 0 0 350px`（`DeckColumn.vue:169`），≤1099 变 100%（`:350-354`） | 用户不能调宽、不能改名、不能自适应 | `stores/deck.ts:9-12` 的 `ColumnConfig` 增加 `name?: string; width: number; flexible?: boolean`，`addColumn`（`:111-115`）写入 `width: 350`；`DeckColumn.vue:169` 的固定值改为内联 `:style`：`flex: column.flexible ? '1 1 350px' : \`0 0 ${column.width}px\``；列菜单（`DeckColumn.vue:87-111`）在"数据源"之前插入"列设置"分组，两个 `<input type="number">` + 一个 checkbox，直接写 `setColumnConfig(id, patch)`（在 `deck.ts` 新增，参照已有 `setColumnType:132-135`） |
| 列头是拖拽把手：`draggable="true"` + `dragover/drop` → `swapColumn`，含 Chrome 的 10ms 延迟绕坑（`column.vue:21-24, 264-306`）；点击 header 回顶（`column.vue:19-26, 257-262`）；右键/菜单含配置、四向移动、左堆叠、右弹出、删除（`column.vue:148-255`）；堆叠列可折叠成 42px 头（`column.vue:365-372, 141-146`） | 列头只有标题 + 菜单（改类型/左移/右移/关闭，`DeckColumn.vue:73-111`） | ①列多时"左移/右移"要点多次；②无点击回顶；③无折叠 | ①`DeckColumn.vue:73` 的 `<header>` 加 `@click="body?.scrollTo({top:0,behavior:'smooth'})"`（给 `.col-body`（`:114`）加 `ref`），菜单按钮用 `@click.stop`（现 `:82` 已经是独立按钮，补 stop 即可）；②排序改拖拽：`header` 加 `draggable="true" @dragstart="e => e.dataTransfer.setData('text/deck-column', column.id)"`，`section`（`:72`）加 `@dragover.prevent @drop.prevent="onDrop"`，`onDrop` 调用现成的 `moveColumn(sourceId, props.index)`（`stores/deck.ts:121-130`） |
| 列类型 9 种，含 `main`（内嵌 `RouterView` 并接收 pageMetadata，`main-column.vue:1-12`）与 `widgets`（`deck.vue:256-266`）；路由跳转由 `mainRouter.navHook` 决定"开列还是开 pageWindow"（`deck.vue:217-225`） | 6 种 home/local/hybrid/global/notifications/explore（`stores/deck.ts:25-32`）；无 `main`、无 `widgets`；路由页在 deck 下走 `DeckPageWindow` 覆盖层（`App.vue:19-27`、`components/deck/deck-page.ts:1-14`） | 无法在 deck 内跟随后续导航（只能靠覆盖层看单条笔记/资料），切换后要关覆盖层才能看别的列 | 中期：把 `main` 加进 `COLUMN_TYPES`，`DeckColumn.vue:115` 用 `<RouterView />` 渲染，`App.vue:20` 的 `DECK_ROUTES` 限制随之放宽；短期不做，但要把 `App.vue:17-19` 的注释改成明确说明"main column deferred" |
| 布局是 `id[][]`，同一横位可纵向堆叠多列（`deck.vue:14-45`），`stackLeft/popRight/swapUp/swapDown` 全在 store（`deck-store.ts:201-269`）；多套 profile 存服务端 `i/registry`（`deck-store.ts:67-134`），右缘 `sideMenu` 切换/新建/改名/删除（`deck.vue:74-120`） | 扁平数组 + `moveColumn`（`stores/deck.ts:121-130`）；单套，localStorage（`stores/deck.ts:43, 99-109`） | ①无堆叠；②布局不跨设备同步 | 堆叠本期不做。持久化：先确认后端是否实现 Misskey 的 `i/registry/get|set`（`GET/POST /api/i/registry/*`），确认后把 `stores/deck.ts:80-109` 的读写改为"localStorage 作缓存 + registry 作真源"；未实现则在 `docs/HANDOVER.md` 记为已知限制，不要先写假接口 |
| 高度靠 flex 链：`.mk-deck{block-size:100dvb; display:flex}` + `.main{flex:1}` + `.columns{flex:1}`，移动端只补 `padding-block-end:100px`（`deck.vue:303, 417-437, 422-424`） | `height: calc(100vh - 108px)`（`DeckShell.vue:104-105`），移动端 `calc(100dvh - 104px)`（`:224`） | 108 = 24×2 shell padding（`main.css:150-157`）+ 60px `.main` padding-bottom（`main.css:242-245`），任何一个改动都会让 strip 高度错位并出现双层滚动条；移动端 104 与 shell 预留的 `12px + 72px + safe-area`（`AppShell.vue:302`）以及底栏实际高度（`MobileNav.vue:58` 的 `6+6+图标标签`）三者互不相等 | 改为 flex 链，删掉两个魔法数：`main.css:242-245` 的 `.main{padding-bottom:60px}` 在 deck 模式下置 0；`AppShell.vue:317-319` 的 deck 规则补 `height: 100dvh; min-height: 0;` 与 `.shell[data-shell="deck"] .main { display:flex; flex-direction:column; min-height:0; padding-bottom:0 }`；`DeckShell.vue:92-95` 的 `.deck` 加 `display:flex; flex-direction:column; flex:1; min-height:0`；`:104-105` 与 `:224` 的 `height` 改 `flex: 1; min-height: 0` |

### 6. 移动端底栏与抽屉

| Firefish / Iceshrimp 的做法 | Cinderfish 现状 | 差距 | 建议动作 |
|---|---|---|---|
| 底栏 4–5 键固定底部，`padding: 12px 12px calc(safe-area + 12px)`，`<400px` 时高度降到 60px（`universal.vue:35-153, 754-850`）；当前项用 `translateY(-0.5em)` 的胶囊高亮（`:782-792`）；未读用左上角 `indicator` 圆点（`:85-95, 794-804`）；"菜单"键打开左侧抽屉（`:37-53`） | 5 键固定底栏，`padding: 6px 8px calc(6px + safe-area)`，`backdrop-filter: blur(12px)`（`MobileNav.vue:49-63`）；badge 与 rail 同款（`:20-28, 107-121`）；激活态只换文字颜色（`:89-91`） | ①无"菜单"键、无抽屉（致命，见下行）；②激活态只有颜色变化，无位置/背景反馈；③底栏高度与 deck strip 的高度是两套数字 | ①`MobileNav.vue:16` 前插入第 1 项 `<button class="nav-item" @click="drawer = true"><AppIcon name="list"/><span class="nav-label">菜单</span></button>`（`AppIcon` 需在 `components/AppIcon.vue` 的手写集合里加 `list`，见该文件图标表）；②激活态照 `universal.vue:786-792`：`.nav-item.router-link-active .icon { background: var(--accent-soft); border-radius: 999px; padding: 2px 10px; }` |
| 抽屉是 `navbar-for-mobile.vue`：240px 左滑入，`menu` transition `translateX(-240px)→0` + 300ms 曲线，遮罩 `_modalBg` 单独过渡，点遮罩/`touchstart` 关闭，路由变化自动关（`universal.vue:177-190, 509-540, 856-868`；`universal.vue:315-318`）；抽屉里含账号区、全部导航项、"更多"、设置、发帖、帮助（`navbar-for-mobile.vue:1-137`） | **无抽屉**。≤720 时 `.rail{display:none}`（`AppShell.vue:305-307`），设置/管理/主题/布局切换/退出登录全部失联（`:86-106`）；唯一的 `/settings` 入口在"我的主页"上（`web/src/components/ProfileHeader.vue:49`）以及被隐藏的 widget 列（`InstanceCard.vue:29`） | 手机用户无法：改主题、切到 Deck 或切回、退出登录、进设置、进管理 | ①新建 `web/src/components/ui/NavDrawer.vue`：`Teleport` 到 body，`.backdrop`（`background: rgba(0,0,0,.5)`）+ `.panel`（`width: 260px; height: 100dvh; background: var(--bg-raised); overflow-y: auto`），`<Transition name="drawer">` 的 CSS 照 `universal.vue:509-540`；内容 = 账号 chip（复用 `AppShell.vue:87-90` 的标记）+ `visibleNav()` 全量 + `rail-foot` 的动作（把 `AppShell.vue:91-105` 抽成 `components/ui/AccountActions.vue` 供 rail 与抽屉共用）；②`MobileNav.vue` 内加 `const drawer = ref(false)`，挂 `<NavDrawer v-model="drawer" />`；③抽屉里 `watch(() => route.fullPath, () => drawer.value = false)`（`useRoute` 在 `App.vue` 已用，组件内 import 即可） |
| 路由为 index 时另挂一个 FAB 发帖按钮，mobile 的 `--stickyBottom: 6rem` 避开底栏（`universal.vue:155-164, 550-553, 722-752`） | 发帖按钮在底栏正中（`MobileNav.vue:33-36`，`.compose-mark` 胶囊，`:123-137`） | 形态不同但目的达成；Cinderfish 底栏 5 项含发帖，居中更清楚 | 保持；只需检查 FAB 与 `OsOverlay` 的层级：底栏 `z-index:30`（`MobileNav.vue:56`），deck 的列菜单是固定 `z-index:60`（`DeckShell.vue:164`），重叠区已正确 |
| deck 移动端另有一套 4 键 64px 底栏，并给 deck 留 `padding-block-end: 100px`（`deck.vue:124-169, 422-424, 534-602`）；Iceshrimp 底栏 4 键，第 1 键用 `popovertarget="sidebar"` 调起抽屉（`iceshrimp/Sidebar.razor:94-121`） | deck 沿用同一个 `MobileNav`（`AppShell.vue:117`，在 `.shell` 内不受 `data-shell` 影响） | 无 gap；多套底栏是参考的重复实现，Cinderfish 单套更简单 | 无需改动 |

### 7. widget 栏

| Firefish / Iceshrimp 的做法 | Cinderfish 现状 | 差距 | 建议动作 |
|---|---|---|---|
| 右列 `position: sticky; max-block-size: 100vb; overflow-y: auto; inline-size: 300px`，`≤1090px` 隐藏（`universal.vue:681-703`）；widget 容器 `flex-direction: column`，子项间距由容器统一给（`universal.widgets.vue:89-105`）；默认 seed 3 个（calendar/notifications/trends，`universal.vue:320-341`） | `.widgets` sticky top 24 + `flex column gap 16`（`AppShell.vue:225-232`）；宽度来自 grid 的 `--side-width: 300px`（`main.css:36, 152`）；≤1099 隐藏（`AppShell.vue:294-296`） | 容器本身与参考等价，问题在内容与配置（下面三行） | 容器保留；把 `max-height` + `overflow-y:auto` 补上（参考 `universal.vue:684-685`），否则 widget 变多时右侧栏滚不动：`AppShell.vue:225-232` 加 `max-height: calc(100dvh - 48px); overflow-y: auto; overscroll-behavior: contain;` |
| widget 列表存在 `defaultStore.state.widgets`，支持编辑模式增/删/拖拽排序（`universal.widgets.vue:45-85`；`components/MkWidgets.vue:1-60` 用 `VueDraggable` + `handle=".handle"`）；实现 20+ 种 widget，注册在 `widgets/index.ts:1-80+`（memo/notifications/timeline/calendar/rss/trends/clock/photos/federation/post-form/slideshow/server-metric/online-users/job-queue/instance-cloud/button…） | `#widgets` slot 的 fallback 是 `InstanceCard`（`AppShell.vue:114`），而该 slot 全仓无人填充（唯一提及是 `InstanceCard.vue:9` 的注释，注释与实际不符）；内容为硬编码两张卡 + 页脚（`InstanceCard.vue:14-48`） | ①没有 widget 概念，新增一个侧栏模块必须改 `AppShell.vue`；②`#widgets` 是空 API，注释在说谎；③无编辑态 | ①新建 `web/src/widgets/index.ts`：`export interface WidgetDef { id: string; title: string; icon: IconName; component: Component }` + `WIDGETS` 表，先把 `InstanceCard.vue` 拆成 `AboutInstance`（`:14-36`）与 `InstanceLinks`（`:38-47`）两个条目；②新建 `web/src/stores/widgets.ts`：`widgetList = ref<{ id: string; type: string }[]>([{id:'w1',type:'about'},{id:'w2',type:'links'}])`，watch 持久化到 `cinderfish.ui`（复用 `stores/shell.ts:9-47` 的读写模式）；③`AppShell.vue:114` 改 `<div class="widget-stack"><component v-for="w in widgetList" :is="WIDGETS[w.type]?.component" :key="w.id" /></div>`，删掉 slot 或改为 `<slot name="widgets">` 包裹（二选一，并在 `InstanceCard.vue:9` 的注释里删掉"exposes the column as a slot"这句不实描述） |
| 500–1100 区间浮出一个右下角 widget 按钮打开右侧 drawer（`universal.vue:26-33, 192-205, 709-720`）；移动端底栏有一枚 widget 键（`:142-152`） | 720–1099 与 ≤720 都没有任何 widget 入口 | 窄屏彻底看不到实例信息与"关于/协议链接" | ①在 `NavDrawer`（第 6 项）底部追加 widget stack 的只读渲染；②或照参考加浮出按钮：新建 `components/widgets/WidgetButton.vue`，`position: fixed; inset-block-end: 5rem; inset-inline-end: 1rem`（避开底栏），仅在 `matchMedia('(max-width: 1099px)')` 为真时渲染，点击打开 `WidgetDrawer`（可直接复用 `NavDrawer` 的过渡与容器，`inset-inline-start: auto; inset-inline-end: 0`） |
| deck 里 widget 是一个列类型（`deck.vue:256-266`），列内 widget 独立配置并可增删改（`deck-store.ts:271-325`；`widgets-column.vue:1-30`） | deck 模式强制隐藏 widget 列（`AppShell.vue:321-323`） | deck 用户拿不到任何侧栏信息 | 依赖第 7 项第 2 行建立的注册表；先加 `widgets` 列类型（`stores/deck.ts:25-32` + `DeckColumn.vue` 的 template 分支），列内渲染 `widgetList`（全局那套），per-column 独立配置后置 |
| Iceshrimp：`.widget-panel` 28rem、sticky `100vh`、左边框，<840 隐藏（`MainLayout.razor.css:12-22, 103-111`）；内容是不可配置的通知面板 + 全部已读按钮（`iceshrimp/WidgetPanel.razor:1-24`） | — | Iceshrimp 这一档只提供"固定面板"，是 Cinderfish 当前状态的更完善版：它至少有面板头（标题 + 动作按钮，`.razor.css` 里 `.widget-panel-header`） | 在建立注册表之前，若只想做一步小改：给 `AppShell.vue:114` 的 fallback 包成 `<div class="widget-stack">` 并为每张卡补一个 `widget-header`（图标 + 标题 + 可选动作），与 Iceshrimp 的面板头一致 |

---

### 补充观察（不在 7 项内，但会影响上面每一条）

- **主题偏好不可响应**：`AppShell.vue:17` 在 `setup` 里读一次 `document.documentElement.dataset.theme`，写回也只在 `:21-22`。这导致移动端加抽屉后主题按钮的状态会与真实值不同步（多标签页同理）。改为 `stores/shell.ts` 里放 `theme = ref(...)` + `watch` 写 `dataset.theme`，`AppShell.vue` 只消费。
- **deck 覆盖规则堆在文件末**：`AppShell.vue:317-329` 靠选择器特异性（`.shell[data-shell="deck"]`）压过 `:289-297` 的 1099 断点。参考里同类问题表现为 `universal.vue:544` 与 `:665` 重复声明 `$widgets-hide-threshold`。建议把 deck 的 grid 覆盖改成 `--shell-cols` 变量：`:150-157` 用 `grid-template-columns: var(--shell-cols, var(--rail-width) minmax(0,1fr) var(--side-width))`，deck 模式只改这一个变量。
- **`min-height: 100vh` + `height: calc(100vh - 48px)` 组合在移动浏览器地址栏伸缩时会跳**（`main.css:157, 167`）。第 5 项建议改成 `100dvh` 后一并解决；注意 `html, body, #app` 目前是 `min-height: 100%`（`main.css:68-71`），改 flex 链时用 `100dvh` 直接在 `.shell` 上取值即可，不需要改这三个元素。
- **无 `XCommon` 等价物差异**：Firefish 在壳层挂 `XCommon`（`universal.vue:207`、`deck.vue:184`），Cinderfish 的 `OsOverlay` 挂在 `App.vue:29`，层级正确，无需对齐。

### 建议落地顺序

1. `web/src/nav.ts` 统一注册表（第 4 项）——它是抽屉与底栏的前置。
2. `NavDrawer.vue` + `MobileNav` 增加"菜单"键（第 6 项）——当前最严重的功能缺失。
3. deck 高度改 flex 链、删掉 108/104（第 5 项末行）——不修的话后续改 padding 必然回归。
4. `.rail` 加 `overflow-y`（第 3 项）——随注册表项数增长会立刻暴露。
5. widget 注册表 + `stores/widgets.ts`（第 7 项）——顺便清理 `#widgets` 的空 API 和不实注释。
6. deck 的列宽/拖拽/点击回顶（第 5 项第 2、3 行）——纯增强，可独立合入。
7. `PageHeader.vue` 与断点登记（第 1、2 项）——改动面最大，放最后。

---

## 笔记与列表

已完成只读比对。现状代码取自 `.worktrees/frontend-ui-redesign/web/src/`，参考取自 `tmp/refcheck/firefish` 与 `tmp/refcheck/iceshrimp-vue`（Iceshrimp Vue 版没有 `components/note/` 目录，页脚与反应条都内联在 `MkNote.vue` 里，`MkNoteHeader.vue` 是根级文件）。

## 差距表

### 1. DOM 结构

| 栏 | 内容 |
|---|---|
| 参考做法 | 根节点 `<div class="note-container" :id="appearNote.historyId \|\| appearNote.id" tabindex="-1" v-hotkey v-size>`：`tmp/refcheck/firefish/packages/client/src/components/MkNote.vue:2-14`（Iceshrimp 同构，`tmp/refcheck/iceshrimp-vue/packages/client/src/components/MkNote.vue:2-13`）。层级为 `.reply-to`(父帖) → `.note-context`(HeaderInfo+RenoteBar) → `.article > .main > .header-container`(头像+Header) / `.body`(Content+Translation) / `.info`(FooterInfo) / `.footer`(ReactionsViewer+按钮)：`MkNote.vue:15-92`。详情页是 祖先链 `MkNoteSub` → `MkNote(detailedView)` → `MkTab` → 分页：`MkNoteDetailed.vue:12-99`。另有整条 `aria-label`（作者/cw/正文/时间）：`MkNote.vue:194-219` |
| 现状 | `.note > .note-inner > header.head + p.reply-context + NoteContent + footer.foot`：`web/src/components/note/Note.vue:66-104`。根节点无 `id`、无 `tabindex`、无 `aria-label`。详情页只渲染一条笔记：`web/src/views/NoteView.vue:50-59` |
| 差距 | ① 缺 `id` → 复制出的 `#noteId` 锚点无法定位，也没有 `:target` 高亮；② 缺 `tabindex` → 笔记不可聚焦，键盘交互全部无从挂载；③ 父帖被压成一行纯文本（`Note.vue:70-73`），参考是完整可交互的子帖；④ 参考 footer 首子元素是反应条、且 footer 之上有 `.info`（绝对时间/频道）行（`MkNoteFooterInfo.vue:2-18`），当前反应条放在正文里、`.info` 行不存在 |
| 建议动作 | 1) `Note.vue:66` 的 `<article>` 加 `:id="note.id"` 与 `tabindex="-1"`；2) 在 `use-note.ts` 增加 `accessibleLabel` computed（照 `MkNote.vue:194-219`），绑到 `<article :aria-label>`；3) 把 `NoteContent.vue:87-102` 的反应条整体移动到 `NoteFooter.vue:113` 的 `<footer>` 首子元素；4) 新增 `note/NoteInfo.vue`（照 `MkNoteFooterInfo.vue`）在 detail 时渲染绝对时间与频道 |

### 2. 容器查询/断点

| 栏 | 内容 |
|---|---|
| 参考做法 | 不用原生 `@container`：`v-size="{ max: [500, 350] }"`（`MkNote.vue:8`）由 ResizeObserver 写 `max-width_500px` / `max-width_350px` class（`tmp/refcheck/firefish/packages/client/src/directives/size.ts:26-44, 93-105`）。真正生效的断点是 500 与 300：`MkNote.vue:577-611`、详情 `MkNoteDetailed.vue:581-617`、`MkNoteSub.vue:603-632`。断点内同时动字号、头像、内边距（`font-size:1.05em→.975em→.825em`、`padding-inline:32px→16px`） |
| 现状 | 原生容器：`.note { container-name: note; container-type: inline-size }`（`Note.vue:108-112`），子组件各自查询（`NoteHeader.vue:97/107/125`、`NoteContent.vue:338/345`、`NoteFooter.vue:295/306`、`NoteMedia.vue:124`、`RenoteBar.vue:82`）。断点为 580 / 500 / 450，只调 padding、字号、`.name` 的 `max-width` |
| 差距 | ① 阈值体系与参考不同（580/500/450 vs 500/300）；② Deck 列固定 350px（`web/src/components/deck/DeckColumn.vue:169` `flex: 0 0 350px`）只命中 450 档，没有 350 级专属密度，也没有任何头像缩放；③ 断点内不调头像（参考 48→46→40）；④ 因整行是 container，`.note-inner` 自身查询同一容器可用（`Note.vue:180`），但引用时 `<Note nested>` 又声明了同名容器（`Note.vue:110-112`），子级查询会重新按内层宽度判定——这是双层容器的隐式行为，参考里只有一层 |
| 建议动作 | 1) 统一为两档：`Note.vue:180-190` 改成 `@container note (max-width: 500px)` 与 `(max-width: 380px)`，并同步改 5 个子组件的 `@container` 阈值；2) 500 档加 `font-size: .975em`，380 档加 `.94em`（照 `MkNote.vue:578, 610`）；3) 引文不再复用外层容器：在 `Note.vue:170` 的 `.quote` 上声明 `container-name: quote`，引文自己的间距由 `@container quote` 决定 |

### 3. 头像尺寸

| 栏 | 内容 |
|---|---|
| 参考做法 | 全局 `--avatarSize: 48px`（`tmp/refcheck/firefish/packages/client/src/style.scss:7`）；主帖 48 → ≤500 46 → ≤300 40（`MkNote.vue:539, 579, 610`，Iceshrimp 同 `MkNote.vue:810-820, 932, 961`）。转贴条 1.2em（`MkRenoteBar.vue:128-136`）。父/子帖 38px 且**圆角 8px**（`MkNoteSub.vue:296-303`），≤500 父帖 46px（`MkNoteSub.vue:623-630`），孙级 24px（`MkNoteSub.vue:603-611`）。引文卡片 40/44/48、圆角 8px（`MkNoteSimple.vue:66-98`）。`MkAvatar` 默认正圆（`components/global/MkAvatar.vue:236`） |
| 现状 | `NoteHeader.vue:13` 写死 `props.nested ? 30 : 42`，断点内不变。引用/子帖头像与主帖同组件、正圆 + 1px 边框（`UserAvatar.vue:17-23`）。转贴条 20px（`RenoteBar.vue:20`） |
| 差距 | ① 主帖 42 vs 参考 48，且缺 ≤500/≤300 递减；② 引文头像 30px 正圆带边框，参考是 38/40 圆角 8px 无边框；③ 头像尺寸是 JS 数字，无法被媒体/容器查询调整——与上一项耦合 |
| 建议动作 | 1) 删掉 `NoteHeader.vue:13` 的数字，改 `UserAvatar` 支持 `size: number \| "inherit"`，inherit 时用 `width: var(--avatar-size)`；`Note.vue:111` 上设 `--avatar-size: 48px`，500 档 `46px`，380 档 `40px`；2) 给 `UserAvatar.vue` 加 `radius?: "50%" \| "8px"`（默认 50%，去掉 `border`，改 `outline: 1px solid var(--border)` 以免影响布局）；3) `Note.vue:84` 的引用与新的父帖组件传 `radius="8px"`、`--avatar-size: 38px`（照 `MkNoteSub.vue:296-303`）；4) `RenoteBar.vue:20` 的 `20` 改为随字号：`size` 取 `1.2em` 需 `UserAvatar` 支持 em 单位，或直接写 `Math.round(16 * 1.2)` 并在断点内同步 |

### 4. hover 显隐交互

| 栏 | 内容 |
|---|---|
| 参考做法 | 按钮**常态就可见**，`opacity: .7`，hover/focus-within 升到 1：`MkNote.vue:407-414` `MkNoteFooter.vue:240-247`；子帖 `MkNoteSub.vue:471-480`；Iceshrimp `MkNote.vue:643-652`。footer 按钮 `flex-grow: 1; max-inline-size: 3.5em` 均分整行，`::before` 做 `var(--panel)` 分段底，首尾 100px 圆角（`MkNoteFooter.vue:242-274`）。详情页 hover 时整行浮出 `--panelHighlight` 背景块（`MkNoteDetailed.vue:508-570`） |
| 现状 | `.foot` 默认 `opacity: 0`，靠 `.note:hover :deep(.foot)` 与 `.note:focus-within` 才显示：`NoteFooter.vue:170-184` + `Note.vue:150-158`；触屏用 `@media (hover: none)` 常显（`NoteFooter.vue:288-293`）。按钮左对齐 + `.spacer`（`NoteFooter.vue:113-164`） |
| 差距 | ① 参考不隐藏只变暗，当前整行消失：指针从正文移向按钮的间隙会闪一下，触屏之外的窄窗口也容易"看不见操作"；② 键盘用户 Tab 进 footer 前看不到任何东西；③ 布局语义不同：参考是横跨整行的分段条，当前是左侧图标组 + 右侧菜单 |
| 建议动作 | 1) `NoteFooter.vue:177` 的 `opacity: 0` 改 `0.7`，`Note.vue:150-158` 改为 `.note:hover :deep(.foot), .note:focus-within :deep(.foot) { opacity: 1 }` 并删掉 `@media (hover: none)` 特例；2) `.action` 改为 `flex: 1 1 0; max-width: 3.5em; min-width: max-content;`，加 `::before` 背景层（照 `MkNoteFooter.vue:253-274`），删除 `NoteFooter.vue:226-228` 的 `.spacer`；3) 详情页可另在 `Note.vue` 加 `:hover { background: var(--panel-2) }` 对应 `MkNoteDetailed.vue:508-570` 的行高亮（参考另有 `:target` 高亮，`MkNoteDetailed.vue:573-579`） |

### 5. 反应胶囊

| 栏 | 内容 |
|---|---|
| 参考做法 | 单行横向滚动、**不换行**，两端 mask 渐隐，容器 `pointer-events: none`、`scrollbar-width: none`：`MkReactionsViewer.vue:37-58`。胶囊 `block-size: 32px; padding-inline: 8px`，底色由 `::before { inset: 0 2px; border-radius: 4px }` 承担：`MkReactionsViewer.reaction.vue:105-120`。只有 `canToggle`（已登录且 reaction 不含 `@host`）才有底色与 hover：`:139-150`，条件在 `:46-48`。自己的反应 `order: -1` + `--accent` 底 + `--fgOnAccent` 计数 + 图标 drop-shadow：`:152-166`。新反应 `newlyAdded` 入场：`:121-126`。计数 `margin-inline-start: 4px`：`:168-175`。hover 100ms 弹出「谁反应了」浮层，走 `notes/reactions`：`:74-101` |
| 现状 | `.reactions` `flex-wrap: wrap; gap: 4px`：`NoteContent.vue:235-241`；胶囊 32px、`border-radius: 6px`、`color-mix(in srgb, var(--text) 7%, transparent)` 底色、**所有**胶囊都是可点 button：`:243-257`；mine 用 `--accent` 实底 + `inset 0 0 0 1px` 描边：`:275-280`；fresh 动画：`:297-303`；排序 `b[1] - a[1]` 按计数降序：`:25-29`；只有 `title`/`aria-label`：`:94-95` |
| 差距 | ① 换行 vs 单行滚动：多条反应时当前会顶高行、把正文往下推，参考是固定一行 + 渐隐；② 没有 `canToggle` 区分，远端用户的 `reaction@host` 目前也渲染成可点按钮；③ 缺反应人浮层（后端也没有该端点）；④ 前端二次排序会与服务端 `reactions` 的对象顺序不一致（参考直接用 `v-for="(count, reaction) in note.reactions"`，`MkReactionsViewer.vue:8`）；⑤ 圆角 6px + 每颗都带底色，与参考的 4px + 仅可切换时着色不同 |
| 建议动作 | 1) `NoteContent.vue:235-241` 改 `flex-wrap: nowrap; overflow-x: auto; scrollbar-width: none;` 加两侧 `mask: linear-gradient(...)`（照 `MkReactionsViewer.vue:45-56`）；2) `:243` 圆角改 4px，底色改为 `::before` 且仅在 `canToggle` 时出现；3) 新增 `canToggle` computed：`isLoggedIn.value && !/@/.test(emoji)`，不满足时渲染 `<button disabled>` 或用 `aria-disabled` + `cursor: default`；4) `:25-29` 的 `sortedReactions` 去掉 `.sort()`，只过滤 `count > 0`；5) 反应人浮层列为独立任务——需先在后端加 `/api/notes/reactions`（GET，`noteId` + `type` + `limit`），再做 tooltip |

### 6. 引文/转贴呈现

| 栏 | 内容 |
|---|---|
| 参考做法 | 纯转贴：只有一条 article，`appearNote = note.renote`（`MkNote.vue:191-193`），上方 `note-context` 只放 RenoteBar（火箭图标 + 1.2em 头像 + 名字 + 时间 + 可见性）：`MkNote.vue:29-42`、`MkRenoteBar.vue:2-60`。时间点是自己的转贴时可点出「取消转贴」菜单：`MkRenoteBar.vue:92-113`；支持同时显示最多 5 个转贴者，逗号 + `...`：`:20-41`。引文在**时间轴**只显示 `引用: ...` 链接（`note/MkNoteContent.vue:108-113`），**详情页**才渲染 `div.renote > XNoteSimple`：16px padding、`1px solid var(--renote)`、`8px` 圆角、hover 变 `--panelHighlight`（`MkNoteContent.vue:128-134, 353-365`）；引文卡片无 footer、无反应、头像圆角 8px（`MkNoteSimple.vue:1-16, 88-98`） |
| 现状 | 纯转贴：`article.note.is-renote` 里放 RenoteBar + **递归完整** `<Note :note="note.renote">`：`Note.vue:36-42, 55-64`；子 Note 自带 `container-name: note` 与自己的 `border-bottom`。引文：始终渲染完整嵌套 Note（`Note.vue:82-86`），左侧 2px 虚线 + 12px 缩进（`Note.vue:124-132`）。无「引用: …」折叠、无取消转贴入口、无多转贴者聚合 |
| 差距 | ① 纯转贴多包一层 article，出现两层同 name 容器与两条边框规则的特例（`Note.vue:136-138` 靠 `border-bottom: 0` 打补丁）；② 引文呈现与参考完全不同：虚线缩进 vs 边框卡片，且参考的圆角/内边距/hover 背景都没有；③ 时间轴与详情页行为一致（都展开），参考是时间轴折叠、详情页展开；④ 转贴者聚合与「取消转贴」缺失 |
| 建议动作 | 1) `Note.vue:55-64` 去掉外层 `<article class="note is-renote">`，改为在同一个 `.note` 内用 `appearNote = pureRenote ? note.renote : note` 驱动 header/content/footer，RenoteBar 作为 `.note-context` 子元素渲染（照 `MkNote.vue:29-42, 191-193`）——这样也自动消灭双层容器；2) `Note.vue:170-172` 的 `.quote` 换成参考卡片：`margin-top: 8px; padding: 16px; border: 1px solid var(--renote); border-radius: 8px;` + `:hover/:focus-within { background: var(--panel-2) }`，删掉 `.note.nested` 的虚线（`Note.vue:124-132`）；3) `NoteContent.vue:85` 的 `<slot name="quote">` 外层加 `v-if="detail"`，非 detail 时渲染 `<RouterLink :to="/notes/${note.renoteId}">引用: …</RouterLink>`（照 `MkNoteContent.vue:108-113`）——注意需要 `NoteContent` 新增 `detail` prop，`Note.vue:75` 传入；4) `RenoteBar.vue:26-28` 的时间改成 `<button>`，`isMine` 时弹「取消转贴」（复用 `use-note.ts:88-101` 的 `renote` 反操作：调用 `api.deleteNote(note.id)`） |

### 7. 日期分隔

| 栏 | 内容 |
|---|---|
| 参考做法 | 相邻两条 `createdAt` 的 `getDate()` 不同即插入（`MkDateSeparatedList.vue:13-18`）。呈现是**两侧指向**的边界标签：`▲ 10月3日　10月4日 ▼`，12px、`line-height: 32px`、`--dateLabelFg`、无横线：`:19-28, 117-149`。列表用 `TransitionGroup`，`direction="down"` 进入时 `translateY(-64px)`：`:2-10, 110-115`。`noGap` 模式下首尾各切一半圆角、中间 0.5px divider：`:151-169` |
| 现状 | `PagedFeed.vue:183-210` 的 `rows` computed 插入 `kind: "date"` 行，模板 `:321-324` 渲染单个 `<p class="date-sep">`。标签 `2026年10月30日`（`:190-195`），两侧 `::before/::after` 1px 横线（`:366-382`）。比较方式是把 `createdAt` 串前 10 位当 UTC 日期比对（`:185-188`）。无过渡动画 |
| 差距 | ① 语义不同：参考是跨越两天的边界标记（各自带箭头），当前是单日居中标题；② 参考样式 12px/32px 行高/无横线，当前 0.78rem + 横线；③ 用 UTC 切日，东八区 00:00–08:00 发布的笔记会被分到前一天；④ 缺进入动画 |
| 建议动作 | 1) `PagedFeed.vue:190-195` 改为 `M月d日`；模板 `:322` 改成两个 span + 上下箭头（照 `MkDateSeparatedList.vue:19-28`），需要 `row` 里同时带 `prevLabel`/`nextLabel`；2) `:185-188` 改为本地日期：`const d = new Date(createdAt)` 取 `getFullYear/getMonth/getDate` 拼 key（**不要**照抄参考只用 `getDate()` 比较——跨月同号不会分割，是参考实现的缺陷）；3) `:366-382` 去掉 `::before/::after`，改 `font-size: 12px; line-height: 32px; color: var(--date-label-fg)`（需在 `styles/main.css` 补变量名，当前没有 `--dateLabelFg`）；4) 动画需要把 `:321` 的 `<template v-for>` 换成 `<TransitionGroup>`，注意 `row.key` 已满足 key 要求 |

### 8. 线程折叠

| 栏 | 内容 |
|---|---|
| 参考做法 | 时间轴：`appearNote.reply && !detailedView && !collapsedReply` 时在笔记上方内联父帖 `MkNoteSub`（`MkNote.vue:15-21`），并用 `.reply-to + .note-context .line::before` 画竖线（`MkNote.vue:416-428`）。`collapsedReply` 模式压成一行摘要：1.2em 头像 + 用户名 + 截断正文，配拐角线（`note/MkNoteHeaderInfo.vue:19-30, 72-103`；`MkNote.vue:463-504`）。长正文折叠：>10 行 / 800 字符 / >4 个文件即折叠到 `15em + 100px` 并加 mask 渐隐 + ShowMoreButton（`note/MkNoteContent.vue:210-217, 368-396, 48-53`）。详情页：祖先链 `notes/conversation` limit 30（`MkNoteDetailed.vue:12-26, 373-379`）；回复 tab 递归 `MkNoteSub`，`REPLIES_LIMIT=10 / DEPTH_UPPERBOUND=5 / REPLY_LEVEL_UPPERBOUND=11`（`MkNoteSub.vue:132-134`），子帖靠 `notes/children` 拉取（`:188-204`）；超限显示「继续该线程」虚线拐角（`:75-89, 438-469`）；`#hash` 命中高亮（`MkNoteDetailed.vue:573-579`） |
| 现状 | 只有一行文字「回复 @username」（`Note.vue:70-73`）。无长文折叠、无回复列表（`NoteView.vue:50-59` 只渲染单条）。后端仅有 `notes/create\|delete\|reactions\|show`（`internal/httpapi/` 路由），但 `Note.reply` 已由 packer 返回（`web/src/api/types.ts:89`） |
| 差距 | ① 时间轴看不到父帖内容，必须点进去；② 长帖无限展开，滚动被单条占据；③ 详情页没有任何线程结构，参考的祖先链/回复树/深度上限/继续线程全部缺失；④ 缺 `#hash` 定位能力（与第 1 项 `id` 缺失同源） |
| 建议动作 | 分两步，先后端无关部分：1) 新增 `web/src/components/note/NoteParent.vue`，用 `note.reply` 渲染 `<NoteHeader>` + `<NoteContent>`（无 footer），`--avatar-size: 38px`、左侧 `--indent` 连接线（照 `MkNoteSub.vue:493-579`），替换 `Note.vue:70-73`；2) `Note.vue` 增加 `collapsedReply?: boolean` prop，为 true 时只渲染摘要行（照 `MkNoteHeaderInfo.vue:19-30`）；3) 长文折叠放 `NoteContent.vue:22` 附近加 `isLong`/`collapsed` computed，判据照 `MkNoteContent.vue:210-217`，样式照 `:368-396`（`max-block-size: calc(15em + 100px)` + `mask: linear-gradient(black calc(100% - 64px), transparent)`）。后端部分：4) 需要新增 `/api/notes/conversation`、`/api/notes/children`、`/api/notes/replies` 三个端点后才做 `NoteView` 的祖先链与回复 tab，在那之前不要在前端伪造线程 |

### 9. 键盘热键

| 栏 | 内容 |
|---|---|
| 参考做法 | 每条笔记 `v-hotkey="keymap"` 绑在根节点（`MkNote.vue:7`），键位在 `:155-165`：`r` 回复、`e\|a\|plus` 反应、`q` 转贴、`up\|k` / `down\|j` 上下条、`esc` 失焦、`m\|o` 菜单。详情页少 up/down（`MkNoteDetailed.vue:267-274`）；Iceshrimp 多一个 `s`（`MkNote.vue:357-366`）。指令把 `keydown` 绑在**元素自身**、非全局（`directives/hotkey.ts:13`），且 input/textarea/contenteditable 内忽略（`:67-89`）。上下条用 `focusPrev/focusNext` 遍历相邻带 `tabindex` 的兄弟（`scripts/focus.ts:1-29`）。焦点用 `:focus-visible::after` 画 1px 环（`MkNote.vue:388-405`） |
| 现状 | 完全没有笔记级热键。全仓库只有三处 keydown：`NoteFooter.vue:88-90`（Esc 关菜单）、`DeckShell.vue:25-27`（Esc 关「添加列」）、`PageWindow.vue:16`（Esc 关窗口）。根 `<article>` 无 `tabindex`、无 `id` |
| 差距 | ① 无操作热键；② 无可聚焦笔记，`j/k` 遍历与焦点环都无从谈起；③ 参考里 `focusNext` 末尾误调 `focusPrev`（`scripts/focus.ts:26`）是 bug，不可照抄 |
| 建议动作 | 1) `Note.vue:66` 加 `tabindex="-1"`，`Note.vue:110` 加 `:focus-visible::after` 环（照 `MkNote.vue:388-405`）；2) 新增 `web/src/lib/hotkeys.ts`，导出 `makeHotkey(map)`（照 `scripts/hotkey.ts:20-104`，含 `e\|a\|plus` 的 `\|` 分支与 input 忽略）；3) `Note.vue` 在 `onMounted` 把 handler 绑到根元素、`onBeforeUnmount` 解绑，键位照 `MkNote.vue:155-165`（`up/k`、`down/j` 调本地 `focus-sibling.ts`）；4) 新增 `web/src/lib/focus-sibling.ts`：`prev/next` 遍历 `element.previous/nextElementSibling`，跳过无 `tabindex` 的兄弟（日期分隔 `<p>` 会被跳过），只实现正确版本；5) 右键菜单/操作菜单打开时在 handler 内 `preventDefault + stopPropagation`（照 `hotkeys.ts:97-98`） |

## 落地顺序建议

先做不依赖后端的四项，可以一次性合并：根节点 `id`+`tabindex`+`aria-label`（第 1、9 项前置）→ 头像尺寸改 CSS 变量（第 3 项）→ 断点统一 500/380（第 2 项）→ footer 由「隐藏」改「变暗 + 分段条」（第 4 项）。这四项都在同一批文件里，改动可互相验证。

第二批是纯呈现：反应条单行滚动 + `canToggle`（第 5 项）、引文卡片 + 时间轴折叠（第 6 项）、日期分隔改双端标签并修时区（第 7 项）。

第三批需要新组件或新端点：父帖内联与长文折叠（第 8 项前 3 步，`Note.reply` 已可用）；反应人浮层、线程树（第 5、8 项后半）依赖 `notes/reactions`、`notes/conversation|children|replies`，需先补后端。热键（第 9 项）建议排在 `tabindex` 落地之后单独做，可与第三批并行。

需要注意的两处参考实现缺陷，不要照抄：`focus.ts:26` 的 `focusNext` 误调 `focusPrev`；`MkDateSeparatedList.vue:15` 只用 `getDate()` 比较导致跨月同号不能分割。

---

## 主题与设计令牌

分析完成。以下是只读对比结果，所有结论均来自源码核对；未能核实的项我已标注。

---

### 一、结论摘要

1. 现有 30 个令牌（`web/src/styles/main.css:5-40`）只覆盖「颜色 22 + 圆角 3 + 阴影 1 + 布局宽度 3 + 字体 1」，Firefish 暴露的公开语义令牌是 69 个（`tmp/refcheck/firefish/packages/client/src/scripts/theme.ts:18-20` 明确过滤掉 `X*` 私有组）。**能一一对上的只有 19 个**，缺失的大头是：覆盖层/滚动条/焦点/输入边框/导航/状态展开/前景透明度梯度这 7 组，以及**全部尺度令牌（间距、字号、字重、行高、动效、z-index）**。
2. 现有圆角语义是**反的**：`--radius-sm:8px`、`--radius:14px`、`--radius-lg:12px`（`main.css:30-32`）。大面板 `--radius-lg` 比控件 `--radius` 更方，命名与实际层级相反，且 `main.css:125` 的注释「12px rounding」与令牌名不一致。
3. 双主题由三处代码各自直接写 `document.documentElement.dataset.theme`（`main.ts:7-11`、`AppShell.vue:17-23`、`SettingsView.vue:11-27`），彼此不通信；**没有 `color-scheme`，没有滚动条配色，没有切换过渡，meta theme-color 固定 `#ff6b35`**；首帧闪白的根因是 `dist/index.html` 里 CSS 是渲染阻塞的、主题要在 defer 的 module 中才算出来，而 `:root` 默认就是深色（见第五节）。
4. `mfm.css` 只覆盖了 MFM 的 13 个静态/动画类，`lib/mfm.ts` 另外产出 `mfm-code / mfm-inline-code / mfm-link / mfm-mention / mfm-hashtag / mfm-emoji / mfm-plain` 七类无样式类，其中 code 的样式写在 `NoteContent.vue:124-137` 的 scoped 块里，正文排版因此不可复用，且 `.mfm-code` 用 `--bg` 作底色，在深色卡片的 `--panel` 上形成反向层级。
5. 可删除的现有令牌 3 个：`--content-max`（0 引用）、`--panel-header`（1 引用且色值与 `--panel-2` 几乎相同）、`--shadow-sm`（把偏移和颜色焊死，应拆为 `--shadow-color` + 组合令牌）。

---

### 二、现有令牌清单（30 个，含引用次数与判定）

引用次数为 `web/src` 全量 `var(--x)` 统计。

| 令牌 | 定义 | 引用 | 判定 |
|---|---|---|---|
| `--accent` | main.css:6 | 93 | 保留 |
| `--accent-soft` | main.css:7 | 20 | 语义过载：既是「选中底色」(`main.css:232`) 又是「焦点环」(`main.css:357`) 又是「提示底色」(`main.css:382`)。建议拆分 |
| `--accent-strong` | main.css:8 | 4 | 改名 `--accent-hover` |
| `--on-accent` | main.css:9 | 18 | 保留（对应 Firefish `fgOnAccent`） |
| `--bg` | main.css:11 / 43 | 7 | 保留 |
| `--bg-raised` | main.css:12 / 44 | 12 | 名不副实：深色下比 `--panel` 更暗（`#17151d` < `#1c1a23`），浅色下与 `--panel` 完全相同（都是 `#ffffff`）。改名为 `--surface-inset` 并修正浅色值 |
| `--panel` | main.css:13 / 45 | 24 | 保留 |
| `--panel-2` | main.css:14 / 46 | 21 | 保留（≈ Firefish `panelHighlight`） |
| `--panel-header` | main.css:15 / 47 | 1（`DeckColumn.vue:185`） | **删除**，值 `#221f2b` vs `#24212c`、`#f4f1ea` vs `#f1eee8` 与 `--panel-2` 无法区分 |
| `--border` | main.css:16 / 48 | 46 | 保留，但需与 `--divider` 划清职责（见第四节） |
| `--divider` | main.css:17 / 49 | 12 | 保留 |
| `--text` | main.css:18 / 50 | 31 | 保留 |
| `--text-strong` | main.css:19 / 51 | 14 | 保留（≈ `fgHighlighted`） |
| `--muted` | main.css:20 / 52 | 52 | 保留，建议作为 `--text-weak` 的别名或直接改名 |
| `--danger` | main.css:21 / 53 | 20 | 保留 |
| `--success` | main.css:22 / 54 | 3 | 保留 |
| `--link` | main.css:23 / 55 | 1 | 保留（仅 `NoteContent.vue:121` 用；正文外链接仍走 `a{color:inherit}`，见第七节） |
| `--renote` | main.css:24 / 56 | 6 | 保留 |
| `--mention` | main.css:25 / 57 | 4 | 保留 |
| `--hashtag` | main.css:26 / 58 | 2 | 保留 |
| `--cw-bg` / `--cw-fg` | main.css:27-28 / 59-60 | 2 / 3 | 保留（缺 `cwHoverBg`） |
| `--shadow-sm` | main.css:34 / 61 | 6 | **删除并重构**：内含偏移，无法复用到不同高度 |
| `--radius-sm` | main.css:30 | 26 | 保留，值需下调（见第五节） |
| `--radius` | main.css:31 | 23 | 保留，值需下调 |
| `--radius-lg` | main.css:32 | 13 | 保留，值需上调 |
| `--rail-width` | main.css:35 | 2 | 保留，缺折叠态令牌（3 处硬编码 `76px`：`AppShell.vue:236,291,327`） |
| `--side-width` | main.css:36 | 2 | 保留 |
| `--content-max` | main.css:37 | **0** | **删除**（`.shell` 用的是硬编码 `max-width:1240px`，`main.css:154`） |
| `--font` | main.css:38-39 | 1 | 保留，但 `"Inter"` 无 `@font-face`、无 preload，实际永远落到系统字体 |

浅色块只覆盖 19 项（`main.css:42-62`），未覆盖的 4 个颜色 `--accent / --accent-soft / --accent-strong / --on-accent` 是写死的字面量——一旦将来做「可配置主题色」，这 4 个必须改成派生值（Firefish 用 `:alpha<0.15<@accent`，`_light.json5:17`，由 `theme.ts:111-159` 编译）。

---

### 三、与 Firefish 69 个公开令牌的映射 / 差集

判定列：**对应**＝语义 1:1；**改名**＝语义同、命名异；**部分**＝语义部分重合；**缺失**＝无对应。

| # | Firefish token（`_light.json5:14-98`） | Cinderfish | 判定 |
|---|---|---|---|
| 1 | `accent`:14 | `--accent` | 对应 |
| 2 | `accentDarken`:15 | — | 缺失（`--accent-strong` 是提亮，方向相反） |
| 3 | `accentLighten`:16 | `--accent-strong` | 部分 |
| 4 | `accentedBg`:17 | `--accent-soft` | 对应 |
| 5 | `focus`:18 | — | 缺失（焦点环复用 `--accent-soft`，`main.css:357`） |
| 6 | `bg`:19 | `--bg` | 对应 |
| 7 | `acrylicBg`:20 | — | 缺失 |
| 8 | `fg`:21 | `--text` | 改名 |
| 9 | `fgTransparentWeak`:22 | `--muted` | 部分（我们是独立色，非 alpha 派生） |
| 10 | `fgTransparent`:23 | — | 缺失（各处写 `opacity:0.7`：`mfm.css:21`、`NoteContent.vue:165`） |
| 11 | `fgHighlighted`:24 | `--text-strong` | 对应 |
| 12 | `fgOnAccent`:25 | `--on-accent` | 改名 |
| 13 | `divider`:26 | `--divider` | 对应 |
| 14 | `indicator`:27 | — | 缺失（角标用 `--accent`，`AppShell.vue:216`） |
| 15 | `panel`:28 | `--panel` | 对应 |
| 16 | `panelHighlight`:29 | `--panel-2` | 部分 |
| 17 | `panelHeaderBg`:30 | `--panel-header` | 部分（且只 1 处使用） |
| 18 | `panelHeaderFg`:31 | — | 缺失 |
| 19 | `panelHeaderDivider`:32 | — | 缺失 |
| 20 | `panelBorder`:33 | — | 缺失（Firefish 是 `" solid 1px var(--divider)"` 字符串拼接） |
| 21 | `acrylicPanel`:34 | — | 缺失（`MobileNav.vue:59-61` 用 `color-mix(--bg-raised 94%)` + `blur(12px)` 字面量） |
| 22 | `windowHeader`:35 | — | 缺失 |
| 23 | `popup`:36 | — | 缺失（弹层用 `--panel`/`--panel-2`：`EmojiPicker.vue:188,211`、`PageWindow.vue:61,96`） |
| 24 | `shadow`:37 | `--shadow-sm` | 部分（我们含偏移） |
| 25 | `header`:38 | — | 缺失 |
| 26-30 | `navBg`/`navFg`/`navHoverFg`/`navActive`/`navIndicator`:39-43 | — | 缺失（`AppShell.vue:225-233` 直接用 `--panel`/`--muted`/`--accent`） |
| 31 | `link`:44 | `--link` | 对应 |
| 32 | `hashtag`:45 | `--hashtag` | 对应 |
| 33 | `mention`:46 | `--mention` | 对应 |
| 34 | `mentionMe`:47 | — | 缺失 |
| 35 | `renote`:48 | `--renote` | 对应 |
| 36 | `modalBg`:49 | — | 缺失（硬编码 `rgba(0,0,0,.5)`：`os.ts:129`、`PageWindow.vue:49`、`PostForm.vue:622`） |
| 37-38 | `scrollbarHandle`/`scrollbarHandleHover`:50-51 | — | **缺失且体感明显**：深色页面滚动条仍是系统浅色 |
| 39 | `dateLabelFg`:52 | — | 缺失（时间戳用 `--muted`） |
| 40-43 | `infoBg`/`infoFg`/`infoWarnBg`/`infoWarnFg`:53-56 | — | 缺失（`main.css:365-384` 用 `color-mix` 现算） |
| 44 | `switchBg`:57 | — | 缺失 |
| 45-46 | `cwBg`/`cwFg`:58-59 | `--cw-bg`/`--cw-fg` | 对应 |
| 47 | `cwHoverBg`:60 | — | 缺失 |
| 48-49 | `buttonBg`/`buttonHoverBg`:61-62 | — | 缺失（复用 `--panel`/`--panel-2`） |
| 50-51 | `buttonGradateA/B`:63-64 | — | 缺失（无渐变按钮，可不补） |
| 52-56 | `swutchOffBg`/`…Fg`/`OnBg`/`OnFg`:65-68 | — | 缺失（无 switch 组件） |
| 57-58 | `inputBorder`/`inputBorderHover`:69-70 | — | 缺失（`main.css:346` 复用 `--border`，悬停态完全缺失） |
| 59 | `listItemHoverBg`:71 | — | 缺失（`NoteContent.vue:251,261` 现算 `color-mix(--text 7%)`） |
| 60 | `driveFolderBg`:72 | — | 不适用 |
| 61 | `wallpaperOverlay`:73 | — | 不适用 |
| 62 | `badge`:74 | — | 缺失 |
| 63 | `messageBg`:75 | — | 缺失 |
| 64 | `success`:76 | `--success` | 对应 |
| 65 | `error`:77 | `--danger` | 改名 |
| 66 | `warn`:78 | — | **缺失**（无警告色，`.notice` 只有 error/info，`main.css:374-384`） |
| 67-69 | `codeString`/`codeNumber`/`codeBoolean`:79-81 | — | 缺失（无语法高亮） |
| 70 | `htmlThemeColor`:82 | — | 缺失（`index.html:6` 硬编码 `#ff6b35`） |

统计：对应/改名/部分合共 19 项，缺失 33 项，不适用 4 项。

**Iceshrimp.NET 的命名法差异**（`tmp/refcheck/iceshrimp/Iceshrimp.Frontend/Layout/MainLayout.razor.css`）：
- 统一 `--<角色>-color` 后缀：`--background-color:5,29`、`--foreground-color:69`、`--border-color:21`、`--hover-color:82`、`--notice-color:70`、`--danger-color:93`、`--accent-primary-color`、`--accent-secondary-color`、`--mention-me-color`、`--highlight-color`、`--font-color`、`--link-color`、`--warning-color`、`--shadow-color`、`--light-color`、`--dark-color`。我们与 Firefish 都是无后缀命名，**建议沿用 Firefish 的无后缀式**（Iceshrimp 的 `-color` 后缀在非颜色令牌 `--btn-radius`、`--size`、`--scale`、`--percentage` 上已破功）。
- Iceshrimp 尺度走 rem：`border-radius:0.5rem` ×17、`0.75rem` ×10、`1rem` ×8、`1.25rem` ×5（同一 css 文件统计），间距 `gap:0.5rem` ×32 / `0.25rem` ×7 / `1rem` ×8，边框 `0.05rem`（`MainLayout.razor.css:21`）。我们全部是 px 字面量，无任何尺度令牌。
- Iceshrimp 只用一个 `--btn-radius` 承载圆角角色（17 处直接 `0.5rem`），说明即使参考实现也没做出完整尺度层——**这一点不要照抄，应自己建层**。

---

### 四、缺失的语义分组（8 组）

| 组 | 现状 | 缺失令牌 | 证据 |
|---|---|---|---|
| 覆盖层 | 模态背景三处硬编码、模糊值字面量、弹层无独立色 | `--overlay-bg`、`--overlay-blur`、`--popup-bg`、`--window-header-bg` | `os.ts:127-129`、`PageWindow.vue:45-49`、`PostForm.vue:620-624`、`MobileNav.vue:59-61` |
| 滚动条 | 完全未接管 | `--scrollbar-thumb`、`--scrollbar-thumb-hover` | 全 `web/src` 无 `::-webkit-scrollbar` / `scrollbar-color`（Firefish：`style.scss:164-196`） |
| 焦点 / 选中 / 悬停 | 焦点环借用 `--accent-soft`；悬停底色各处现算 | `--focus-ring`、`--surface-hover`、`--surface-selected` | `main.css:354-358`（输入）、`AppShell.vue:146,199`（`outline:2px` 共 37 处分散写法）、`NoteContent.vue:251,261` |
| 输入 | 无独立底/边框/悬停/错误态 | `--input-bg`、`--input-border`、`--input-border-hover`、`--input-border-invalid` | `main.css:342-352` 全部复用 `--bg-raised` + `--border` |
| 导航 / 头部 | 直接借用表面与文字令牌，角标无令牌 | `--nav-bg`、`--nav-fg`、`--nav-fg-hover`、`--nav-active`、`--nav-indicator`、`--badge-bg`、`--badge-fg` | `AppShell.vue:203-233`、`AppShell.vue:208-222`（角标 `999px` + `--accent`） |
| 状态展开 | 只有 `--danger`/`--success` 两个纯色，弱底靠 `color-mix` 临时算；无 warn | `--danger-soft`、`--danger-border`、`--warn`、`--warn-bg`、`--warn-fg`、`--info-bg`、`--info-fg` | `main.css:374-384`（3 处 `color-mix`）、`NoteFooter.vue:223,278`、`SettingsView.vue:426`、`ProfileView.vue:486` |
| 前景透明度梯度 | 只有 `--text`/`--text-strong`/`--muted` 三级；弱化靠 `opacity` 现调（10 处 `opacity:0.4~0.7`） | `--text-weak`、`--text-faint`、`--text-on-status`、`--text-date` | `mfm.css:21`、`NoteContent.vue:165`、`main.css:386-389` |
| 尺度 | 完全不存在 | 见第五节 | — |

另外两处「命名打架」需要一并定案：`--border`（46 次，组件轮廓）与 `--divider`（12 次，流内细线）在用途上已经混用——`NoteContent.vue:198,228` 用 `--divider` 画胶囊边框，`main.css:139-146` 用 `--divider` 画 1px 分隔线。建议保留两者并补 `--border-strong`，把「胶囊/卡片边框」统一回 `--border`。

---

### 五、尺度令牌差距

| 维度 | 现状（实测分布） | Firefish | Iceshrimp | 差距 |
|---|---|---|---|---|
| 圆角 | 3 个令牌但**层级倒置**：`--radius-sm:8` < `--radius-lg:12` < `--radius:14`（`main.css:30-32`）；另有 6 种裸值：`50%`（`main.css:317`、`AppShell.vue:264`）、`999px`（`AppShell.vue:215`）、`6px`（`NoteContent.vue:250`）、`5px`（`NoteContent.vue:135`）、`calc(var(--radius)-4px)`（`NoteContent.vue:181`）、`calc(var(--radius)-6px)` | 单一 `--radius:12px`（`style.scss:4`）+ 图标按钮 `5px`（`style.scss:375`）+ beta 标 `4px`（`style.scss:692`） | `--btn-radius` + 0.5/0.75/1/1.25rem | 需重排为 xs 4 / sm 6 / md 10 / lg 14 / full，并把 6 种裸值收编 |
| 间距 | **0 令牌**。`gap` 17 种取值：8px×26、6px×15、10px×14、12px×13、4px×9、2px×9、5px×4、16px×3、3px×2、20px×2、14px×2、9/28/22/1/18px 各 1；`padding` 60+ 种组合（含 `8px 10px 8px 32px` 这种非标值） | `--marginFull:16`/`--marginHalf:10`/`--margin` 三档（`style.scss:5-9,23-25`）+ `_gaps_m/s` 用 `1.5em/0.75em`（`style.scss:456-472`） | 0.25/0.5/1rem 为主 | 缺 4 点制 `--space-*` |
| 字号 | **0 令牌**。24 种 rem 值（0.62→2.4）+ 裸 `15px`（`main.css:79`）+ `em`(9 种) + `%`(3 种，`mfm.css:5-11`) | `html{font-size:14px}`（`style.scss:81`），其余 em/%(0.8em/80%/65%) | 相对 em 为主 | 缺 8 档 type scale；`@container` 里又出现 `0.94em`、`0.92em`（`NoteContent.vue:340,347`） |
| 字重 | **0 令牌**。500×2、600×21、700×26、800×3 | 无令牌，用 `bold`/`font-weight` 直接写 | — | 4 个值可固化为 3 档；注意 `"Inter"` 未加载（`main.css:38`），800 会被合成，字号令牌化时要一并解决字体自托管（Firefish 自托管 4 个字重的 woff2，`style.scss:28-63`） |
| 行高 | **0 令牌**，同一种正文出现 3 个值：`1.55`（`main.css:80`）、`1.62`（`NoteContent.vue:115`）、`1.58`（`NoteContent.vue:341`），标题 `1.25`（`main.css:97`） | `1.6`（`style.scss:82`） | — | 需 `--leading-tight/base/relaxed` |
| 阴影 | 1 个 `--shadow-sm`（含偏移与颜色，`main.css:34`） | `--shadow` 仅颜色（`_light.json5:37`），偏移在使用处（`style.scss:322,810`） | `--shadow-color` | 拆为 `--shadow-color` + `--shadow-popover`/`--shadow-modal` |
| 动效 | **0 令牌**。时长 0.12/0.15/0.2/0.25/0.3s，其中 0.15s×33 | 0.2s/0.5s/1s + `cubic-bezier(0,.5,.5,1)`（`style.scss:820-836`） | — | 需 `--dur-fast/base/slow` + `--ease-standard/emphasized` |
| z-index | **0 令牌**，裸值 1/20/30/60/1000/1100 分散在 6 个文件（`os.ts:113`、`PageWindow.vue:45`、`MobileNav.vue:53`、`DeckShell.vue:164`、`NoteFooter.vue:240`、`DeckColumn.vue:234,321`） | — | — | 需 `--z-sticky/popover/window/overlay/toast` |
| 布局高度 | 底栏高度靠**跨文件猜值**：`AppShell.vue:302` 预留 `72px`，而 `MobileNav.vue:58,73,85` 实际约 60px；折叠轨宽 `76px` 硬编码 3 次（`AppShell.vue:236,291,327`） | — | — | 需 `--rail-width-collapsed`、`--bottom-bar-height`、`--touch-target` |

---

### 六、双主题切换与首帧闪白

**现状机制**

| 环节 | 实现 | 位置 |
|---|---|---|
| 默认值 | `:root` 即深色全套值 | `main.css:5-40` |
| 浅色覆盖 | `:root[data-theme="light"]`，只覆盖 19 项 | `main.css:42-62` |
| 初始化 | localStorage → `prefers-color-scheme` → 写 `dataset.theme` | `main.ts:7-11` |
| 切换 A | 侧栏图标按钮，直接改 `dataset.theme` + localStorage | `AppShell.vue:17-23`，按钮 `:100-102` |
| 切换 B | 设置页单选，再次直接改 | `SettingsView.vue:11-27` |
| 持久化键 | `cinderfish.theme` | 三处一致 |

**问题（按严重度）**

1. **首帧闪白/闪黑确实存在，且方向明确**：`dist/index.html` 中样式表是渲染阻塞的 `<link rel="stylesheet">`，而主题计算在 `<script type="module">`（defer）里；`:root` 默认深色。浅色用户首帧必然先看到 `#0e0d12` 深底色再跳到 `#f7f5f1`。修复方式是 `<head>` 内、样式表之前插入同步内联脚本（读取 localStorage 并写 `documentElement.dataset.theme`），仅在 `index.html` 内联，不引入新依赖。
2. **无 `color-scheme`**：全 `web/src` 检索无 `color-scheme` 声明。深色页面的原生滚动条、`<select>` 下拉、自动填充背景、`::selection` 全按浅色渲染——这是当前观感最割裂的一处。Firefish 有对应设置（`theme.ts:100`，但它写成了 `color-schema`，是拼写错误，**不要照抄这个笔误**）。
3. **状态不同步**：`AppShell.vue:17` 的 `theme` ref 只在挂载时读一次，`SettingsView` 改主题后侧栏图标不会更新（要刷新）；两标签页之间也没有 `storage` 监听，会各说各话。
4. **无切换过渡**：Firefish 用 `html._themeChanging_` 给 `background`/`border` 加 1s 过渡（`style.scss:205-214`，触发在 `theme.ts:71-75`）。我们现在是瞬切，而页面里已有 33 处 `transition: background .15s`，切换时会出现「部分元素先变、部分元素后变」的错帧。补过渡时注意 `prefers-reduced-motion`，Firefish 那条 `!important` 过渡没做降级，别复制这个缺陷。
5. **meta theme-color 不跟随主题**：`index.html:6` 固定 `#ff6b35`（橙色强调色，不是背景色）。Firefish 用 `htmlThemeColor` 令牌刷新（`theme.ts:89-94`）；Iceshrimp 也是从 `getComputedStyle('--background-color')` 取值（`Iceshrimp.Frontend/App.razor:33-38`）。
6. **不跟随系统实时变化**：`main.ts:9` 只读一次 `matchMedia`，没有 `onchange` 监听。Firefish 有 `syncDeviceDarkMode` + 监听（`init.ts:362-366`）。
7. **模型差异**：我们靠 `data-theme` 属性 + 两段静态 CSS；Firefish 靠 `theme.ts:96-98` 把编译后的 69 个属性逐个 `setProperty` 到 `documentElement.style`。后者的好处是任意主题可插拔、强调色可派生（`theme.ts:111-159` 的 `:darken/:alpha/:hue` DSL）。**当前不需要引入主题编译器**，但令牌设计要让 `--accent-soft`/`--accent-strong` 可由 `--accent` 派生，否则将来做主题色配置要返工。

未核实项：Firefish 的 SSR 模板（`packages/client/index.html` / 后端 view）不在 sparse checkout 内（该目录只有 `package.json`、`src`、`vite.config.ts` 等），因此**无法确认它在首帧前是否内联了主题**；上述对比只覆盖已在树内的 `theme.ts` / `init.ts` / `style.scss`。

---

### 七、mfm.css 与正文排版的差距

| 项 | Cinderfish | Firefish（`components/global/MkMisskeyFlavoredMarkdown.vue`） | 影响 |
|---|---|---|---|
| x3 / x4 倍率 | 300% / 400%（`mfm.css:7-12`） | 400% / 600%（`:72-78`） | 与 Misskey 观感不一致 |
| 嵌套缩放 | 无，嵌套相乘（`$[x2 $[x2 …]]` = 800%） | 三级衰减：全量 → `calc(var(--mfm-zoom-size)/2 + 50%)` → 100%（`:80-98`） | 叠用即溢出 |
| 缩放间接层 | 直接把 `font-size` 写死在类上 | 经 `--mfm-zoom-size` 中转（`:69,83`） | 无法统一钳制 |
| 动画开关 | 只有 `prefers-reduced-motion`（`mfm.css:173-180`） | `advancedMfm`（高级 MFM 开关）+ `animatedMfm`（动画开关）两层，且能压掉内联样式（`:44-65`） | 用户无法在「保留彩色/字号」的同时关掉动画 |
| 函数覆盖 | 13 个：x2/x3/x4/center/small/blur/mono/serif/spin(x/y)/shake/jump/blink/rainbow | 另有 `bounce:145`、`twitch:166`、`rubberBand:301`、`fade:334` | `lib/mfm.ts:183-235` 对 `$[bounce ]` 等直接降级为纯文本 |
| jump 幅度 | `-30%/-10%` 百分比（`mfm.css:101-117`） | `-16px/-8px` 固定（`:127-143`） | 大字号下差异明显 |
| 引用块 | **无样式** | `blockquote` 8px 外边距 + 12px 内边距 + 4px 左侧色条（`:356-363`） | 长文引用不可辨 |
| 代码块 | 样式在 `NoteContent.vue:124-137`（scoped），底色 `--bg` | `.mfm-object pre{font-size:.8em}`（`:365-367`），代码块随容器 | ①不可复用到其它渲染点；②深色卡片内 `--bg`(`#0e0d12`) 比 `--panel`(`#1c1a23`) 更暗，层级反向；③`5px` 裸圆角（`:135`） |
| 换行/空白 | `\n`→`<br />`（`lib/mfm.ts:110`）+ `white-space: normal`（`NoteContent.vue:115`） | `.mfm-object{white-space:pre-wrap}`（`:346-348`） | 连续空格与制表符丢失 |
| 单行截断 | 无 | `.nowrap`（`:349-354`） | 通知列表无法做单行摘要 |
| 弱化表达 | `opacity:0.7` 硬编码（`mfm.css:21`） | 走 `--fgTransparentWeak`（`:361`） | 令牌化后需改 |
| 模糊 | `filter:blur(6px)` + `color:transparent`，无悬停恢复（`mfm.css:24-27`） | `._blur_text` 悬停 `blur(0)`，`transition .3s`（`style.scss:1038-1045`） | 内容无法被主动揭示 |
| 字族 | `mfm-mono`/`mfm-serif` 各自写死字体栈（`mfm.css:30-36`），与 `main.css:38` 重复 | `._monospace`（`style.scss:774-782`） | 两处字体栈会漂移 |
| 正文行高 | 三值混用：1.55 / 1.62 / 1.58（见第五节） | 单一 1.6 | 长文节奏不一致 |
| 链接 | `.mfm-link/.mfm-mention/.mfm-hashtag` 统一 `--link`（`NoteContent.vue:118-122`），无下划线过渡 | `a{text-decoration-color:transparent}` → hover 显形（`style.scss:242-256`），且 hashtag/mention/renote 各有色 | 链接可发现性弱 |
| 颜色函数 | `$[fg.color]` 走内联 style（`lib/mfm.ts:196-203`），完全绕过令牌 | 同样内联，但可被 `advancedMfm` 关掉（`:52-57`） | 浅色主题下用户自选色可能不可读，且无兜底 |
| 未定义类 | `lib/mfm.ts` 产出 `mfm-emoji`、`mfm-plain`、`mfm-link`、`mfm-mention`、`mfm-hashtag`、`mfm-code`、`mfm-inline-code` 共 7 类，`mfm.css` 中 0 条规则（后两类在组件内） | — | `mfn-emoji` 目前无任何视觉处理 |

---

### 八、建议 token 词表

命名约定：颜色沿用现有无后缀式 `--角色[-变体]`（与 Firefish 对齐，不改 93 处 `--accent` 的引用）；尺度用 `--space-N`/`--text-*`/`--weight-*`/`--leading-*`/`--radius-*`/`--shadow-*`/`--z-*`/`--dur-*`/`--ease-*`。标注：**保**=保留、**改**=改名、**调**=保留名改值、**删**=删除、**新**=新增。

#### A. 表面与层级（11）

| 令牌 | 动作 | 说明 |
|---|---|---|
| `--bg` | 保 | 页面底 |
| `--panel` | 保 | 卡片/侧栏 |
| `--panel-2` | 保 | 次级表面（吸收 `--panel-header`） |
| `--surface-inset` | 改（原 `--bg-raised`） | 输入框/代码块底；浅色需给出与 `--panel` 可区分的新值 |
| `--surface-hover` | 新 | 替代 `color-mix(--text 7%)`（`NoteContent.vue:251`） |
| `--surface-selected` | 新 | 替代 `--accent-soft` 的选中语义（`main.css:232`） |
| `--popup-bg` | 新 | 弹层/选择器（`EmojiPicker.vue:188`、`PageWindow.vue:61`） |
| `--window-header-bg` | 新 | 页窗标题栏（`PageWindow.vue:79`） |
| `--overlay-bg` | 新 | 模态遮罩，收编 3 处 `rgba(0,0,0,.5)` |
| `--overlay-blur` | 新 | 替代字面量 `blur(12px)`（`MobileNav.vue:61`） |
| `--shadow-color` | 新 | 阴影色；`--shadow-sm` 删除 |

#### B. 描边与分隔（4）

| 令牌 | 动作 | 说明 |
|---|---|---|
| `--border` | 保 | 组件轮廓（46 次） |
| `--border-strong` | 新 | 强调轮廓 |
| `--divider` | 保 | 流内 1px 细线（12 次） |
| `--input-border` / `--input-border-hover` / `--input-border-invalid` | 新 | 当前全部借用 `--border`（`main.css:346`），悬停/错误态缺失 |

#### C. 前景（7）

| 令牌 | 动作 | 说明 |
|---|---|---|
| `--text` | 保 | |
| `--text-strong` | 保 | |
| `--text-weak` | 改（原 `--muted`） | 52 次引用需批量改名；或保留 `--muted` 并文档化 |
| `--text-faint` | 新 | 替代散落 `opacity:0.55~0.7` |
| `--text-date` | 新 | 时间戳，当前混用 `--muted` |
| `--text-on-accent` | 改（原 `--on-accent`） | 与 `--text-on-status` 成对 |
| `--link` / `--link-hover` | 保 / 新 | |

#### D. 强调与交互（7）

| 令牌 | 动作 | 说明 |
|---|---|---|
| `--accent` | 保 | |
| `--accent-hover` | 改（原 `--accent-strong`） | |
| `--accent-active` | 新 | 对应 `accentDarken`（缺失项 #2） |
| `--accent-soft` | 保 | 仅表示「弱强调底」，不再兼任焦点环 |
| `--focus-ring` | 新 | 收编 37 处 `outline:2px solid var(--accent)` 与 `box-shadow:0 0 0 3px`（`main.css:357`、`AdminView.vue:597`） |
| `--nav-bg` / `--nav-fg` / `--nav-fg-hover` / `--nav-active` | 新 | 侧栏/底栏成组，避免直接借用 `--panel`/`--muted` |
| `--badge-bg` / `--badge-fg` | 新 | 角标（`AppShell.vue:208-222`） |

#### E. 状态（9）

| 令牌 | 动作 | 说明 |
|---|---|---|
| `--danger` | 保 | |
| `--danger-soft` / `--danger-border` | 新 | 替代 3 处 `color-mix(--danger 12%/45%)`（`main.css:375-376`、`NoteFooter.vue:223,278`） |
| `--success` / `--success-soft` | 保 / 新 | |
| `--warn` / `--warn-bg` / `--warn-fg` | 新 | Firefish 有 `warn`，我们完全缺失 |
| `--info-bg` / `--info-fg` | 新 | |

#### F. 内容语义色（6）

`--link` / `--mention` / `--mention-me`（新）/ `--hashtag` / `--renote` / `--cw-bg` / `--cw-fg` / `--cw-hover-bg`（新）

#### G. 滚动条与选择（4）

`--scrollbar-thumb`、`--scrollbar-thumb-hover`、`--selection-bg`、`--selection-fg`（Firefish 有 `::selection`，`style.scss:65-68`，我们没有）

#### H. 尺度（建议新增 26）

| 组 | 建议值 |
|---|---|
| 间距 `--space-*` | 1:2px、2:4px、3:6px、4:8px、5:10px、6:12px、8:16px、10:20px、12:24px、16:32px（把 17 种 gap 归到 10 档） |
| 字号 `--text-*` | 2xs .68rem、xs .76rem、sm .84rem、base 1rem、md 1.05rem、lg 1.25rem、xl 1.5rem、2xl 2.2rem（把 24 种归到 8 档） |
| 字重 `--weight-*` | regular 400、medium 500、semibold 600、bold 700（弃用 800，或先自托管字体） |
| 行高 `--leading-*` | tight 1.25、base 1.55、relaxed 1.65（消灭 1.58/1.62 变体） |
| 圆角 `--radius-*` | xs 4px、sm 6px、md 10px、lg 14px、full 999px（一次性修正倒置；6 种裸值并入） |
| 阴影 | `--shadow-popover:0 2px 10px var(--shadow-color)`、`--shadow-modal:0 8px 32px var(--shadow-color)` |
| 动效 | `--dur-fast:.12s`、`--dur-base:.15s`、`--dur-slow:.3s`、`--ease-standard:cubic-bezier(.2,0,0,1)`、`--ease-emphasized:cubic-bezier(0,.5,.5,1)` |
| 层级 `--z-*` | sticky 20、popover 60、window 1000、overlay 1100、toast 1200 |
| 布局 | 保 `--rail-width`、`--side-width`；新 `--rail-width-collapsed:76px`、`--bottom-bar-height:72px`、`--touch-target:40px`；删 `--content-max` |

#### 删除清单

| 令牌 | 原因 | 替代 |
|---|---|---|
| `--content-max`（`main.css:37`） | 0 引用 | 无 |
| `--panel-header`（`main.css:15,47`） | 仅 `DeckColumn.vue:185` 一处，色值与 `--panel-2` 无法区分 | `--panel-2` |
| `--shadow-sm`（`main.css:34,61`） | 偏移与颜色焊死，只能有一种高度 | `--shadow-color` + `--shadow-popover` |

---

### 九、可执行动作清单（按依赖顺序，只给动作与定位，不改代码）

1. **先修首帧闪白**（改动最小、收益最大）：在 `web/src/index.html` 的 `<head>` 起始、首个 `<link rel="stylesheet">` 之前插入同步内联脚本，读 `localStorage["cinderfish.theme"]` → 回退 `matchMedia("(prefers-color-scheme: dark)")` → 写 `document.documentElement.dataset.theme`。同时把 `main.ts:7-11` 改为「仅当 `dataset.theme` 为空时兜底」，避免二次写入。
2. **补 `color-scheme`**：在 `main.css:5-40` 的 `:root` 加 `color-scheme: dark`，在 `:root[data-theme="light"]`（`main.css:42-62`）加 `color-scheme: light`。验证方式：深浅两主题下打开 `<select>` 与滚动条，确认原生控件跟随。
3. **收敛主题控制**：新增 `web/src/stores/theme.ts`（`theme = ref<"dark"|"light">()`、`apply()`、`toggle()`、监听 `storage` 与 `matchMedia().onchange`），把 `AppShell.vue:17-23`、`SettingsView.vue:11-27`、`main.ts:7-11` 全部改为调用它。验证：设置页切换后，侧栏图标（`AppShell.vue:100-101`）立即同步；两个标签页互相同步。
4. **meta theme-color 跟随**：把 `index.html:6` 的 `content="#ff6b35"` 改为由主题写入（新增 `--theme-color` 令牌 = 两套 `--bg` 值），在 `theme.ts` 的 `apply()` 里更新 `meta[name=theme-color]`。
5. **拆尺度层（一次性大改，建议独立提交）**：在 `main.css:30-37` 用第八节 H 组替换当前 3 个圆角 + 1 个阴影 + `--content-max`；然后按 `grep -c "font-size: 0.8\|gap: 5px"` 这类清单逐文件替换。验收：`grep -o "font-size: *0\.[0-9]*rem" -r web/src | sort -u | wc -l` 从 24 降到 ≤8；`grep -o "gap: *[0-9]*px" -r web/src | sort -u | wc -l` 从 17 降到 ≤10。
6. **修圆角倒置**：`--radius-sm 8→6`、`--radius 14→10`、`--radius-lg 12→14`，新增 `--radius-full:999px`，并把 `NoteContent.vue:135`(5px)、`:250`(6px)、`:181`(calc)、`main.css:317`(50%)、`AppShell.vue:215`(999px) 五处裸值换成令牌。
7. **补覆盖层与滚动条**：新增 `--overlay-bg`/`--scrollbar-thumb`/`--scrollbar-thumb-hover`，替换 `os.ts:129`、`PageWindow.vue:49`、`PostForm.vue:622`（同时把 `PostForm.vue:623` 的 `color:#fff` 改为 `--text-on-status`）；在 `main.css` 补 `::-webkit-scrollbar` 与 `scrollbar-color`（参考 `style.scss:164-196`，但不要照搬 12px 宽度）。
8. **拆 `--accent-soft` 的三重语义**：`main.css:232`（选中 → `--surface-selected`）、`main.css:357` + `AdminView.vue:597`（焦点 → `--focus-ring`）、`main.css:382`（提示底 → `--accent-soft` 保留）。
9. **补状态色扩展**：`warn`/`danger-soft`/`danger-border`/`info-bg`/`info-fg`，替换 `main.css:374-384`、`NoteFooter.vue:223,278`、`SettingsView.vue:426`、`ProfileView.vue:486` 的 `color-mix`。
10. **`--panel-header` 与 `--content-max` 下线**：`DeckColumn.vue:185` 改 `--panel-2`；删除两个定义。验证：`grep -rn "var(--panel-header\|var(--content-max" web/src` 返回空。
11. **底栏/折叠轨高度令牌化**：`AppShell.vue:236,291,327` 的 `76px` → `--rail-width-collapsed`；`AppShell.vue:302` 的 `72px` → `--bottom-bar-height`，并与 `MobileNav.vue:58,73,85` 实测高度对齐（当前约 60px，预留值偏大 12px）。
12. **MFM 与正文排版**（建议在尺度层落地后再做）：
    - `mfm.css:4-12` 把 x2/x3/x4 改为经 `--mfm-zoom-size` 的 200/400/600%，并按 `MkMisskeyFlavoredMarkdown.vue:80-98` 加两级嵌套衰减。
    - 把 `NoteContent.vue:124-137` 的代码块样式迁到 `mfm.css`（改为 `--surface-inset` 底、`--radius-sm` 圆角、字体走新 `--font-mono`），并在 `mfm.css` 补 `blockquote` 与 `pre`/`code` 字号（参考 `:356-367`）。
    - `NoteContent.vue:115,341` 的 1.62/1.58 统一到 `--leading-relaxed`。
    - 补 `mfm-plain`/`mfm-emoji` 两条规则（当前由 `lib/mfm.ts` 产出、零样式）。
    - 若要覆盖 Misskey 全部函数，需同时改 `lib/mfm.ts:183-235` 增加 `bounce/twitch/rubberBand/fade` 分支与对应 `@keyframes`（Firefish `:145,166,301,334`）；仅改 CSS 无效。
13. **字体自托管或删 `Inter`**：`main.css:38-39` 的 `"Inter"` 无 `@font-face`、`index.html` 无 preload，实际不会生效。二选一：自托管 400/500/600/700 woff2（参考 `style.scss:28-63` 的 `font-display:swap` 用法），或从字体栈中删掉 `Inter`，避免 800 字重被合成。

**验证命令**（第 5、10、12 步后各跑一次）：
```bash
cd web/src
grep -rn "var(--panel-header\|var(--content-max" .        # 应为空
grep -roh "font-size: *[^;]*" --include=*.vue --include=*.css . | sort -u | wc -l
grep -roh "gap: *[^;]*" --include=*.vue --include=*.css . | sort -u | wc -l
grep -rn "rgba(0, 0, 0, 0.5)\|#fff\b\|#ff5c8a" --include=*.vue --include=*.ts .  # 硬编码色清理
cd .. && pnpm build   # vue-tsc + vite
```

未改动任何文件；上述所有引用均可在工作目录 `.worktrees/frontend-ui-redesign/` 与 `tmp/refcheck/` 中按行号复核。唯一未核实的是 Firefish 的服务端首帧模板（不在 sparse checkout 内），已在第六节标注。

---

## widget 体系与导航配置

已完成只读分析。下面是参考实现、Cinderfish 现状、差距表与建议。

**路径简写**：`FF` = `/home/lainsaka/文档/MilnFish/tmp/refcheck/firefish/packages/client/src`，`IS` = `/home/lainsaka/文档/MilnFish/tmp/refcheck/iceshrimp/Iceshrimp.Frontend`，`IV` = `/home/lainsaka/文档/MilnFish/tmp/refcheck/iceshrimp-vue/packages/client/src`。Cinderfish 路径相对工作树 `.worktrees/frontend-ui-redesign`。

---

### 一、参考实现

#### 1.1 Firefish 的 widget 清单（22 个）

注册表在 `FF/widgets/index.ts:4-93`（`app.component("MkwXxx", defineAsyncComponent(...))`），名称白名单在 `FF/widgets/index.ts:95-118`。逐个功能：

| widget 名 | 文件 | 功能 | 数据来源 | 可配置项（propsDef） |
|---|---|---|---|---|
| `memo` | `memo.vue` | 便签，textarea 自动保存 | `defaultStore.memo`（本地） | `showHeader` |
| `notifications` | `notifications.vue` | 通知流，含「全部已读」「通知设置」 | streaming `main` | `showHeader`、`height=300`、`includingTypes`(hidden) |
| `timeline` | `timeline.vue` | 时间线，可切 home/local/social/global/列表/天线 | streaming + `antennas/list`、`users/lists/list` | `showHeader`、`height=300`、`src`/`antenna`/`list`(hidden) |
| `calendar` | `calendar.vue` | 日历 + 今日/本月/本年进度百分比 | 纯前端 | `transparent` |
| `userList` | `user-list.vue` | 指定列表成员 | `users/lists/list`、`users/lists/show` | `showHeader`、`listId`(hidden) |
| `rss` / `rssTicker` | `rss.vue`、`rss-ticker.vue` | RSS 列表 / 横向滚动条 | 站点 URL（经代理） | `showHeader`、`height`、`urls` |
| `trends` | `trends.vue` | 热门标签 | `hashtags/trend` | `showHeader` |
| `photos` | `photos.vue` | 全站最近图片墙，实时插入 | `drive/stream` + main channel `driveFileCreated` | `showHeader`、`transparent` |
| `clock` / `digitalClock` / `unixClock` | 同名文件 | 模拟钟 / 数字钟+时区+毫秒 / UNIX 时间戳 | 纯前端 | `size`、`label`、`thickness`、`twentyFour`、`fontSize`、`showMs`… |
| `federation` | `federation.vue` | 最近交互实例 | `federation/instances` | `showHeader` |
| `instanceCloud` | `instance-cloud.vue` | 活跃实例标签云 | `federation/instances` | `transparent` |
| `serverInfo` | `server-info.vue` | 实例信息卡（banner/favicon/版本） | `instance` 全局对象 | 无 |
| `onlineUsers` | `online-users.vue` | 在线人数 | `get-online-users-count` | `transparent` |
| `serverMetric` | `server-metric/index.vue` | CPU/内存/磁盘（管理员） | `serverStats` | `showHeader`、`transparent` |
| `jobQueue` | `job-queue.vue` | 队列积压（管理员） | `admin/queue/...` | `transparent` |
| `postForm` | `post-form.vue` | 内嵌发帖器 | `XPostForm` | 无 |
| `slideshow` | `slideshow.vue` | Drive 文件夹幻灯片 | `drive/files` | `folderId`、`height` |
| `button` / `aiscript` | 同名文件 | 可编程按钮 / AiScript 沙箱 | 本地 | `label`、`colored`、`script` |

Iceshrimp 的 Vue fork（`IV/widgets/index.ts:99-121`）在此基础上多一个 `activity`（`IV/widgets/activity.vue`，热力图/柱图两种视图，`activity.calendar.vue`、`activity.chart.vue` 为子件），其余 22 个同名同构。

#### 1.2 注册与布局机制

四层，职责清晰：

1. **注册层** `FF/widgets/index.ts`：一个 Vue 插件把 `Mkw<Name>` 注册成全局异步组件；同时导出 `widgets` 字符串数组（下拉菜单的候选项）。**这两份数据是手工同步的**，新增 widget 要改两处。
2. **渲染层** `FF/components/MkWidgets.vue`：接收 `widgets: {name,id,data,place}[]` + `edit`。
   - 查看态 `:60-70`：`<component :is="\`mkw-${widget.name}\`" :widget="widget" @updateProps="...">`，右键出配置菜单（`:139-172`）。
   - 编辑态 `:31-58`：`VueDraggable` 拖拽排序 + 每个 widget 上叠「齿轮」「叉」两个按钮，顶部是 `MkSelect`（按 i18n 排序，`:104-108`）+ 添加 / 关闭。
   - 配置调用走组件 `expose.configure()`（`:111-113`）。
3. **配置层** `FF/widgets/widget.ts:26-79`：`useWidgetPropsManager(name, propsDef, props, emit)` 给出 `widgetProps`（从 `widget.data` 深拷贝，按 propsDef 补默认值，`:40-46`）、`configure()`（`os.form` 弹表单，`:59-72`）、`save()`（3s throttle 后 `updateProps`）。**widget 的「可配置」= 一个 propsDef 对象 + 一个 expose**，没有集中式 schema 文件。
4. **数据层** `FF/store.ts:161-169`：`widgets: { where: "deviceAccount", default: [] }`，即「每设备每账户」持久化，元素形状 `{name,id,place,data}`。

容器与布局：
- universal 右栏 `FF/ui/universal.widgets.vue:2-12` 只是一个 `aside`，把 `defaultStore.widgets` 整个交给 `XWidgets`，并提供编辑开关（`:22-30`）和 add/remove/update 处理器（`:52-85`）。宽度与吸附在 `FF/ui/universal.vue:681-703`（300px、sticky、`max-block-size:100vb` 自滚）。
- deck 的 widget 是**列**：`FF/ui/deck/widgets-column.vue:21-29` 把 `column.widgets` 交给同一个 `XWidgets`，增删走 `deck-store` 的 `addColumnWidget`/`setColumnWidgets`（`FF/ui/deck/deck-store.ts:271-320`）。**同一个渲染组件被两种容器复用**，这是整条链路最关键的设计。
- 首次使用播种：`FF/ui/universal.vue:320-339`，`widgets.length === 0` 时写入 `[calendar, notifications, trends]`（注意 `place:"right"` 字段——`place` 在整个渲染链路里**从未被读取**，是历史残留，不要照抄）。
- 响应式：`FF/ui/universal.vue:236-237` 脚本阈值 `DESKTOP_THRESHOLD=1100`、`MOBILE_THRESHOLD=500`，样式在 `:700-703` 以 1090px 隐藏右栏；隐藏后由右下角按钮打开抽屉（`:203-205`）。

#### 1.3 导航项的数据来源与可配置程度

- **定义**：`FF/navbar.ts:10-139` 的 `navbarItemDef` 是一个 `reactive` 字典，键为稳定 id（`notifications`、`messaging`、`drive`、`followRequests`、`explore`、`announcements`、`search`、`lists`、`antennas`、`favorites`、`pages`、`gallery`、`clips`、`channels`、`groups`、`ui`、`reload`），每项含 `title`(i18n key)、`icon`、`to` 或 `action`、`show`(computed 布尔)、`indicated`(computed 红点)。
- **顺序与取舍由用户决定**：`FF/store.ts:29-37` 的 `menuOptions` 是默认顺序（7 项），`FF/store.ts:138-141` 存成 `menu: string[]`（`deviceAccount`）。数组里的 `"-"` 渲染成分隔线（`FF/ui/_common_/navbar.vue:38`）。
- **配置界面**：`FF/pages/settings/navbar.vue`。`:4-32` 是 `VueDraggable` 列表（拖拽排序 + 每项删除按钮），`:96-115` 是「添加项」（从 `navbarItemDef` 里扣掉已在 `menu` 的键，外加 `-` 分隔符），`:121-132` 每次变化 `defaultStore.set("menu", ...)`，`:60-63` 是「恢复默认」。另有 `menuDisplay` 单选 `sideFull` / `sideIcon`（`:47-58`）切换侧栏是否只显示图标。
- **渲染**：`FF/ui/_common_/navbar.vue:37-72` 遍历 `menu`，`navbarItemDef[item].to ? 'MkA' : 'button'`，`show` 为假则跳过，`indicated` 为真则加闪烁点。**管理员入口不走 `navbarItemDef`**，在 `:73-105` 里硬编码（根据 `isModerator` 显示控制面板或自定义 emoji），底部「发帖」「帮助」也不可配置（`:129-148`）。

结论：Firefish 的导航是「定义集中 + 顺序用户可调 + 少数固定项」，覆盖率大约 17 个可选项，**不能**自定义 URL。

#### 1.4 移动端如何复用

三条不同的路子，Cinderfish 应各取一条：

1. **导航：数据同源，模板复制**。`FF/ui/_common_/navbar-for-mobile.vue:36-71` 与桌面版 `navbar.vue:37-72` 是**逐行几乎相同的两段模板**，都读同一个 `defaultStore.state.menu` 和同一个 `navbarItemDef`（`:150`、`:142`）。好处是零耦合、坏处是改一处要改两处（这是它们真实的技术债，不该当优点抄）。
2. **widget：组件真复用**。`FF/ui/universal.vue:203-205`：`<XWidgets v-if="widgetsShowing" class="widgetsDrawer" />`——抽屉里挂的是**同一个** `universal.widgets.vue`，只是换了个 fixed 定位的容器类（样式 `:705-720`）。开关在右下角浮动按钮（`:142-152`）。
3. **Iceshrimp.NET：DOM 真复用**。`IS/Layout/Sidebar.razor:13` 侧栏本身是 `popover="auto"` 元素，移动端底部栏的按钮用 `popovertarget="sidebar"`（`:96`）把**同一个侧栏**以抽屉形式弹出；桌面靠 CSS 常驻。代价是导航项硬编码在 `Sidebar.razor:20-91`，底部栏又在 `:94-121` 重写一遍（同样是复制，不是共享）。

Iceshrimp.NET 的右栏本身**不可配置**：`IS/Layout/WidgetPanel.razor:5-16` 写死一个通知列表 + 全部已读按钮，容器 `IS/Layout/MainLayout.razor:44-50` 只给它一块区域（`MainLayout.razor.css:12-22`：28rem 宽、sticky、左框线；`:103-111` 在 <840px 直接隐藏，不提供抽屉补偿）。它是「布局取舍」参考，不是 widget 系统参考。

---

### 二、Cinderfish 现状

- `web/src/layouts/AppShell.vue:25-33`：导航是**函数内联数组**，4 项 + 管理员条件追加。`:74-84` 渲染，通知红点写死 `item.to === '/notifications'`。
- `web/src/layouts/AppShell.vue:113-115`：`<aside class="widgets"><slot name="widgets"><InstanceCard /></slot></aside>`。全仓库 `grep "#widgets"` 只命中定义本身，**没有任何视图使用这个 slot**，所以 widget 栏恒等于一张 `InstanceCard`。
- `web/src/components/widgets/InstanceCard.vue:14-47`：一块实例卡 + 一块「关于」链接列表 + 一行版权，三个 `section` 硬编码，无 id、无 props、无编辑能力。
- `web/src/components/ui/MobileNav.vue:14-41`：5 项硬编码（时间线/通知/探索/发帖/我的），与 AppShell 的 4 项**是两份独立数据**；`:23` 又复制了一遍红点判断。移动端**没有设置入口**（`InstanceCard.vue:29` 的设置按钮只在登录后、且 <1099px 时 widget 栏已被隐藏）。
- `web/src/stores/deck.ts:25-41`：`COLUMN_TYPES` + `COLUMN_META` 已经是**数据驱动的类型注册表**，`:65-78` 有白名单 `sanitize`，`:99-109` 有 watch 持久化——这是本项目已有的正确范式，widget 注册表应当照此写。
- `web/src/stores/shell.ts:9-12`：`cinderfish.ui` 存成 `UiPrefs` 扩展 record，注释明确写了「so later UI preferences can be added without a second storage key」。
- 断点：`AppShell.vue:234-287` ≤1279 侧栏收图标、`:289-297` ≤1099 widget 栏 `display:none`（**无抽屉补偿**）、`:299-312` ≤720 侧栏消失、`:317-323` deck 模式直接隐藏 widget 栏。
- 后端实际可支撑的数据：`/api/notifications`、`/api/notes/{local,hybrid,global,timeline}`、`/api/drive/files`、`/api/users/show`、`/api/users/notes`、`/api/meta`；streaming 频道 `main`/`homeTimeline`/`localTimeline`/`hybridTimeline`/`globalTimeline`（`internal/streaming/channel_main.go:21-25`），main 事件 `notification`/`reply`/`renote`/`mention`/`follow`/`followed`/`unfollow`（`internal/httpapi/streaming_publish.go:37-119`）。**没有** `hashtags/trend`、`federation/instances`（仅管理员）、`announcements`、`get-online-users-count`、`drive/stream`、`driveFileCreated` 事件。
- 设计文档已定调：`docs/superpowers/specs/2026-10-29-frontend-redesign-design.md:99`「widget 栏首版：重样式现有实例卡 + 链接列表，不新增依赖后端数据的 widget」。下面的建议在这个前提下推进，只把「零后端成本」的 widget 加进来。

---

### 三、差距表

| # | 维度 | 参考实现 | Cinderfish 现状 | 差距性质 |
|---|---|---|---|---|
| 1 | widget 类型注册 | `FF/widgets/index.ts:95-118` 名称数组 + `:4-93` 全局组件注册 | 无注册表；`web/src/components/widgets/` 下只有 `InstanceCard.vue` | 缺整层 |
| 2 | widget 实例数据模型 | `FF/store.ts:161-169`：`{name,id,place,data}[]`，`deviceAccount` 持久化 | 无；`AppShell.vue:114` 直接写死 `<InstanceCard />` | 缺整层 |
| 3 | widget 配置机制 | `FF/widgets/widget.ts:26-79` `propsDef` + `useWidgetPropsManager` + `os.form` | 无；`InstanceCard.vue` 无 props | 缺整层（`os.ts` 已有 `confirmDialog`/`openPostForm`，可扩 `openForm` 或直接在组件内联表单） |
| 4 | widget 渲染容器 | `FF/components/MkWidgets.vue:60-70` 查看 / `:31-58` 编辑+拖拽 | 无；`AppShell.vue:224-232` 只有 flex 列样式 | 缺整层 |
| 5 | widget 双容器复用 | universal 右栏 `FF/ui/universal.widgets.vue:2-12` 与 deck 列 `FF/ui/deck/widgets-column.vue:21-29` 共用 `XWidgets` | deck 列类型只有 6 种（`stores/deck.ts:25-32`），无 widget 列 | 缺 deck 侧接入 |
| 6 | 移动端 widget 抽屉 | `FF/ui/universal.vue:203-205`（同组件换容器）+ `:142-152` 入口按钮 | `AppShell.vue:289-297` 在 ≤1099px 直接 `display:none`，无补偿 | 缺口，且是当前唯一的「窄屏看不到任何 widget」问题 |
| 7 | 移动端导航数据同源 | `FF/ui/_common_/navbar-for-mobile.vue:36-71` 与 `navbar.vue:37-72` 读同一 `menu` + `navbarItemDef` | `MobileNav.vue:14-41` 与 `AppShell.vue:25-33` 两份独立硬编码，项目数也不同（5 vs 4） | 重复源，必然漂移 |
| 8 | 导航项定义集中度 | `FF/navbar.ts:10-139` 单一定义表（title/icon/to/action/show/indicated） | 无定义表；图标名、路径、文案、可见性都散在两处模板 | 缺一层 |
| 9 | 导航顺序用户可调 | `FF/pages/settings/navbar.vue:4-32` 拖拽 + `:96-115` 增删 + `:121-132` 自动保存 + `:60-63` 重置 | 无；用户只能改主题（`SettingsView.vue:318-336`）和壳模式（`AppShell.vue:91-99`） | 缺整层 |
| 10 | 导航分隔符 | `FF/ui/_common_/navbar.vue:38` `item === '-'` 渲染 `<div class="divider">` | 无分组概念 | 低优先 |
| 11 | 侧栏紧凑模式用户可选 | `FF/navbar.vue:168-190` `menuDisplay` 强制 iconOnly | `AppShell.vue:234-287` 纯媒体查询，用户不可选 | 可选 |
| 12 | 红点/指示器数据源 | `navbarItemDef[x].indicated` 为 computed，`FF/navbar.vue:66-71` 统一渲染 | `AppShell.vue:78` 与 `MobileNav.vue:23` 两处写死 `/notifications` + `unreadNotifications` | 逻辑重复 |
| 13 | 移动端「更多」抽屉 | Firefish 底部按钮开 `XDrawerMenu`（`FF/ui/universal.vue:218`、`MobileNav` 对应 `navbar-for-mobile`），Iceshrimp.NET 用 `popovertarget`（`IS/Layout/Sidebar.razor:96`） | 移动端无抽屉，设置页只能靠 `InstanceCard.vue:29` 的按钮（且 widget 栏已隐藏） | 功能性缺口 |
| 14 | 管理员入口 | 桌面 `FF/navbar.vue:73-94` 硬编码；移动 `navbar-for-mobile.vue:73-93` 再写一遍 | `AppShell.vue:32` 条件追加，`MobileNav.vue` 完全没有 | 与 #7 同源问题 |
| 15 | deck 列类型扩展点 | `FF/ui/deck/deck-store.ts:8-28` 列类型含 `widgets`，`widgets?: ColumnWidget[]` | `stores/deck.ts:7` 只有 6 种时间线/通知/探索 | 可低成本对齐 |

---

### 四、首版建议

#### 4.1 首版做 5 个 widget（全部零新增后端）

排序按「投入产出比」：

| 优先级 | widget id | 中文名 | 数据来源（现有） | 复用组件 | 可配置项 |
|---|---|---|---|---|---|
| P0 | `instance` | 实例信息 | `stores/instance.ts`（`/api/meta`） | 现有 `InstanceCard.vue:14-36` 拆出 | 无 |
| P0 | `links` | 关于 / 协议链接 | 静态 | `InstanceCard.vue:38-45` 拆出 | 无 |
| P0 | `memo` | 便签 | 纯 localStorage（对齐 `FF/widgets/memo.vue`） | 新建 | `showHeader` |
| P1 | `notifications` | 通知 | `/api/notifications` + streaming `main` 的 `notification` 事件 | `NotificationItem.vue` | `height`、`showHeader` |
| P1 | `timeline` | 时间线 | `/api/notes/{local,hybrid,global,timeline}` + 对应 streaming 频道 | `TimelineFeed.vue` / `ui/PagedFeed.vue` | `source`（home/local/hybrid/global）、`height` |

**明确不做**（后端无接口，做了就是假数据）：`trends`、`federation`、`instanceCloud`、`onlineUsers`、`announcements`、`photos`（`/api/drive/files` 只返回自己的 Drive，语义不是 Firefish 的全站照片流）、`serverMetric`、`jobQueue`、`aiscript`、`slideshow`、`rss`（需代理，且与个人实例场景无关）。

`instance` + `links` 拆开而非保留一个 `InstanceCard`，理由：Firefish 的 `serverInfo` 是独立 widget，用户应该能把「关于」关掉而只留实例卡。

#### 4.2 导航配置化方案

分三步，每步可独立提交：

**第一步：抽定义表（不改行为）**
新建 `web/src/nav.ts`：

```ts
export interface NavItemDef {
  label: string;
  icon: IconName;              // 复用 stores/deck.ts:16 ColumnIcon 的联合类型做法
  to: string;
  show?: () => boolean;        // computed 化
  badge?: () => number;        // 取代两处写死的 unreadNotifications
}
export const navItemDef: Record<string, NavItemDef> = { home, notifications, explore, settings, admin, ... };
export const DEFAULT_NAV: readonly string[] = ["home", "notifications", "explore", "settings"];
```

`AppShell.vue:25-33` 与 `MobileNav.vue:11/14-41` 都改为消费它。红点从 `AppShell.vue:78`、`MobileNav.vue:23` 收敛到 `badge`。

**第二步：用户偏好存储**
新建 `web/src/stores/nav.ts`，**复用 `stores/shell.ts:18-30` 的容错读法**（try/catch 回退默认）与 `stores/deck.ts:65-78` 的 `sanitize`（白名单过滤未知 key + 去重 + 空则回退默认）。存储位置并入已有的 `cinderfish.ui` record（`stores/shell.ts:9-12` 注释已写明这是给它留的扩展位），键 `nav`；桌面与移动两个数组（Firefish 只有一份 `menu`，但它的移动端是抽屉、项数可以多；Cinderfish 移动端是 5 格底栏，项数上限必须不同，硬套一份 `menu` 会撑破 5 格）：

```ts
type NavPrefs = { desktop: string[]; mobile: string[] };
```

`sanitize` 规则：剔除不在 `navItemDef` 的 key；`mobile` 截断到 5 项（含固定的发帖按钮，见下）；空数组回退 `DEFAULT_NAV`/`DEFAULT_MOBILE_NAV`。

**第三步：设置页 UI**
`web/src/views/SettingsView.vue` 在「外观」卡（`:318-336`）之后插入一张「导航」卡：两列（桌面 / 移动）各一个有序列表，每行 `[上移][下移][显示/隐藏]`，底部「恢复默认」。

**不要引入 `vue-draggable-plus`**：`web/package.json:13-16` 当前只有 `vue` 和 `vue-router`，Firefish 的拖拽能力（`FF/components/MkWidgets.vue:31`、`FF/pages/settings/navbar.vue:4`）在新依赖的代价下不值得复刻首版；上下移按钮在 5–6 项的列表里体验接近。

#### 4.3 具体改动点（按依赖顺序）

1. `web/src/nav.ts`（新建）——定义表 + `DEFAULT_NAV` / `DEFAULT_MOBILE_NAV`。图标名字必须落在 `web/src/components/AppIcon.vue:6-28` 的现有集合内，缺的（例如「更多/齿轮已有 settings」）先补 path 再引用，否则渲染空 SVG。
2. `web/src/stores/nav.ts`（新建）——`NavPrefs` + sanitize + watch 持久化，写进 `cinderfish.ui`。注意 `stores/shell.ts:41-47` 是把整个 record 覆盖写的，新增 `nav` 时必须走 `{...readPrefs(), nav}` 同样的合并写法，否则两个 store 会互相抹掉。
3. `web/src/layouts/AppShell.vue:25-33`、`:74-84`——改读 `nav.ts` + `stores/nav.ts`；`:91-99` 的「Deck/通用」开关留在 rail-foot 不变。
4. `web/src/components/ui/MobileNav.vue:11`、`:14-41`——改读同一份定义，按 `prefs.mobile` 渲染；发帖按钮保持固定居中不可配置（对齐 `FF/ui/universal.vue:155-164` 把发帖固定进底栏的做法）。
5. `web/src/widgets/registry.ts`（新建）——`{ name, label, icon, component: defineAsyncComponent(...), defaultProps }`，结构照 `stores/deck.ts:34-41` 的 `COLUMN_META`。`components/widgets/` 目录改为每个 widget 一个文件：`InstanceWidget.vue`、`LinksWidget.vue`、`MemoWidget.vue`、`NotificationsWidget.vue`、`TimelineWidget.vue`。
6. `web/src/stores/widgets.ts`（新建）——`{ name, id, props }[]`，`deviceAccount` 级别（用现有 token 区分账户，或首版先 device 级），键 `cinderfish.ui.widgets`。
7. `web/src/components/widgets/WidgetStack.vue`（新建）——查看态 `v-for` + 编辑态（添加下拉：可用项来自 registry、隐藏已加项；每项上移/下移/删除）。**这就是 Firefish 的 `MkWidgets.vue`，但不引拖拽、不暴露 `configure()`**：首版 widget 的配置项少，直接做成 widget 自己内部的小控件（例如 timeline 的 source 分段按钮，对齐 `FF/widgets/timeline.vue:11-39` 在头部切换而不是弹表单）。
8. `web/src/layouts/AppShell.vue:113-115`——`<slot name="widgets"><WidgetStack :editable="true" /></slot>`；`:224-232` 的容器样式保留。
9. `web/src/layouts/DeckShell.vue:70`、`web/src/stores/deck.ts:7/25-32/34-41`——新增 `ColumnType` 值 `"widgets"`，`COLUMN_META` 加一行，`DeckColumn.vue` 里加一个分支渲染 `<WidgetStack>`（对齐 `FF/ui/deck/widgets-column.vue:21-29`）。这一步把 widget 和 deck 打通，是 Firefish 结构里最值得抄的一点。
10. `web/src/components/ui/WidgetsDrawer.vue`（新建）+ `AppShell.vue:289-297`——≤1099px 隐藏右栏后，在 `MobileNav` 之上提供一个「组件」入口，抽屉里挂**同一个** `WidgetStack`（对齐 `FF/ui/universal.vue:203-205`）。若首版想缩范围，至少要在这里留一个显式的 TODO 注释，否则 720–1099px 的用户永远看不到任何 widget。
11. `web/src/views/SettingsView.vue:336`（外观卡之后）——插入导航配置卡；widget 配置卡可同一位置放（列表 + 上移/下移/删除）。

#### 4.4 两个不必照抄的参考细节

- `place: string | null`（`FF/store.ts:166`）：全链路只写不读，是 Firefish 的历史残留。Cinderfish 的 widget 模型里不要这个字段。
- Firefish 导航项在移动端是**模板复制**（`navbar-for-mobile.vue:36-71`），widget 在移动端是**组件复用**（`universal.vue:203-205`）。前者是债，后者是正解——本方案对两者分别处理，导航走数据同源单模板（Cinderfish 的移动端是 5 格底栏，本来就画不出侧栏那一长串，不需要两份模板），widget 走组件复用。

#### 4.5 未决项（需要你拍板）

- 导航偏好是否真的需要「桌面 / 移动两份数组」。Firefish 只有一份；我建议两份是因为 Cinderfish 底栏固定 5 格，而桌面侧栏可能需要 6–8 项。若接受「移动端取桌面顺序的前 4 项」，存储模型可以退化成单一数组，实现量减半。
- widget 偏好是「每设备」还是「每设备每账户」。Firefish 是 `deviceAccount`；Cinderfish 当前 `stores/shell.ts`、`stores/deck.ts` 都是纯 device。首版建议跟现有保持一致（纯 device），否则 token 变更时的键管理与迁移都要新写。
- 通知 widget 与 `/notifications` 页面在数据上是否共享（Firefish 的 widget 是独立拉取 + streaming 追加，页面是分页列表）。若两者都挂 streaming `main`，`stores/streaming.ts:76-90` 的引用计数能正确处理，但要确认 `NotificationItem` 在两个上下文里的密度样式不冲突。

无代码改动，工作树未变更（`git status` 为空）。

---

## 页面级形态

分析完成。以下是逐页差距报告（只读分析，未修改任何文件）。

---

## 页面级形态差距：Cinderfish 10 个视图 vs Firefish / Iceshrimp.NET

### 口径说明

- `MkPageHeader` 指 **Firefish 自己的** `tmp/refcheck/firefish/packages/client/src/components/global/MkPageHeader.vue`，属 Firefish 前端参考范围；未引用 Misskey 前端。
- 现状路径均相对于 `.worktrees/frontend-ui-redesign/`。
- 已核实的能力边界（影响「建议动作」可行性）：后端 `notifications` 接受 `includeTypes` 但未使用；无 `notes/mentions`、`users/following|followers`、`users/search`、`notes/featured` 端点；`users/notes` 支持 `sinceId`/`untilId`/`withFiles`/`withReplies`；全站无 `document.title` 机制。

### 跨页共性：四个结构性缺口

| 维度 | 参考 | 现状 | 影响 |
|------|------|------|------|
| 页头抽象 | `components/global/MkPageHeader.vue:2-9` 单一 sticky 组件，左侧标题、中间 tab 行、右侧动作区；`72-95` tab + 3px accent 滑动高亮条（`600-612`）；`313-321` 窄屏 <500px 折叠 | 10 个视图各自手写 `<header class="page-title">`，样式仅 `main.css:401-409`（flex + h1 1.4rem），非 sticky、无 tab 槽、无动作槽、无底部分隔线 | 每个页面都要重写一遍标题+tab，形态不一致；滚动后标题与 tab 一起消失 |
| tab 状态 | Firefish 用子路由（`pages/user/index.vue:52,61`；`pages/settings/index.vue:289-297`）；Iceshrimp 用 `@page` + `NavLink`（`Components/SettingsMenu.razor:2-8`） | Timeline/Explore/Notifications/Profile 的 tab 全是组件内 `ref`（`TimelineView.vue:8`、`ExploreView.vue:12`、`NotificationsView.vue:22`、`ProfileView.vue:27`） | 刷新丢状态、无法分享某个 tab、浏览器后退不回到上一个 tab |
| 分区承载 | 设置/管理是 nav(34%) + main 双列（`settings/index.vue:351-367`、`Layout/SettingsLayout.razor:12-18`） | Settings/Admin 都是单列纵向堆 `.card`（`SettingsView.vue:221-375`、`AdminView.vue:274-493`），靠 `SettingsView.vue:380-382` 的 `margin-top:16px` 分隔 | 设置页是一条 5 屏的长滚动；危险操作（退出）在页面最底部 |
| 筛选/分页语义 | 筛选参数交给后端（`MkNotifications.vue:88-111`）；滚动到底自动加载（`PagedFeed.vue:278-292`） | 通知类型筛选只在渲染层过滤（`PagedFeed.vue:191`），分页按未过滤页推进（`PagedFeed.vue:106,136`）；Profile 却用手动按钮（`ProfileView.vue:405-409`） | 「反应」tab 会一边显示「还没有反应通知」一边继续偷偷翻页；两个流的分页交互不一致 |

**应保留的差异（不是退化）**：`PagedFeed.vue:316-318` 的三行骨架屏优于参考的转圈（`MkPagination.vue:6`、`note.vue:64`）；`PagedFeed.vue:322` 的日期分隔行与 Firefish `MkDateSeparatedList` 一致；`NotificationItem.vue:36-47` 的分类型空态文案比参考更细。

---

### 1. TimelineView（时间线）

| 参考做法（文件:行） | 现状 | 差距 | 建议动作 |
|---|---|---|---|
| `pages/timeline.vue:2-11`：`MkStickyContainer` + `#header` 里 `MkPageHeader v-model:tab="src"`，标题、tab、动作同处一个 sticky 页头 | `TimelineView.vue:41-43` 页头只有 `<h1>时间线</h1>`，纯 block，随页面滚走 | 页头没有 tab 槽，导致 tab 必须另起一行 | 新建 `web/src/components/ui/PageHeader.vue`：`position:sticky; top:0`，`backdrop-filter: blur(15px)` + 底部 0.5px divider（照 `MkPageHeader.vue:339-341,374-388`），props `{title, subtitle?, icon?, tabs?, actions?, back?}` |
| `MkPageHeader.vue:72-95`：tab 在同一行内横向排列，`600-612` 用 3px accent 圆角块做滑动高亮 | `TimelineView.vue:45-57` tab 是页头下方独立的胶囊条（样式 `63-92`：外框 + accent 填充的 active），与页头两套视觉 | tab 与标题是两块割裂的组件，视觉层级重复 | 删掉 `TimelineView.vue:45-57` 与 `63-92`，改 `<PageHeader v-model:tab="source" :tabs="tabs" title="时间线" />`；active 态由 PageHeader 统一为「文字变 accent + 下划线块」 |
| `pages/timeline.vue:238-289`：每个 tab 带 icon（`iconOnly: true`），窄屏只留图标；`MkPageHeader.vue:549-559` >3 tab 自动折叠成 `2.7em` 图标格 | `TimelineView.vue:13-18` 四个 tab 纯文字；无 icon；窄屏无折叠或横滚（`.tabs` 未设 `overflow-x`） | 窄屏 4 个中文 tab 会挤爆或换行；无图标辨识 | `tabs` 每项加 `icon`（`AppIcon.vue` 现有 home/globe/compass/feather/user；需补 `users`/`handshake`）；`.tabs` 加 `overflow-x:auto; scrollbar-width:none`，>3 项时按 `MkPageHeader.vue:549-559` 折叠 |
| `pages/timeline.vue:96-105`：按实例开关（`disableLocalTimeline`/`enableGuestTimeline`）+ 登录态生成可用 tab | `TimelineView.vue:13-18` 硬编码四源，只有 `auth: true` 标记；`55` 用一个 `·` 提示需要登录 | 实例配置无法隐藏时间线；未登录提示形态过弱 | 短期：`tabs` 的 `auth` 项在未登录时改渲染 `lock` 图标（`AppIcon` 已有），并在 PageHeader tab 上给 `aria-disabled`。中期：`InstanceMeta`（`web/src/api/types.ts:135-149`）没有 `disableLocalTimeline`/`enableGuestTimeline` 字段，需后端补，先记 TODO |
| `pages/timeline.vue:212-236`：headerActions 放「列表/天线」两个 icon-only 动作；`199-206` `src` 持久化到 store | 无动作区；`TimelineView.vue:20-23` 的 `pick()` 只改本地 ref | 无任何页头动作位；tab 选择刷新即丢 | PageHeader 加 `actions` 槽；`pick()` 内改 `router.replace({ query: { ...route.query, tl: id } })`，并在 `watch` 里从 `route.query.tl` 回填（`TimelineView.vue:27-33`） |
| `pages/timeline.vue:14-18`：置顶发帖器在流顶部 | `TimelineFeed.vue:90` `<PostForm fixed>` 已在流顶部 ✓ | 一致 | 无需改动 |

### 2. NotificationsView（通知）

| 参考做法（文件:行） | 现状 | 差距 | 建议动作 |
|---|---|---|---|
| `pages/notifications.vue:2-10` + `147-168`：tab 进 sticky 页头，`all/reactions/mentions/directNotes` 四项且带 icon | `NotificationsView.vue:123-135` 页头（h1 + 未读计数 + 全部已读），`137-149` 另起一条筛选条 | tab 不在页头内；无 icon；tab 维度不同 | 同 Timeline：tab 上移进 `PageHeader`，动作「全部已读」进 `actions` 槽 |
| `pages/notifications.vue:73`：tab 维度是 全部/反应/提及/私信 | `NotificationsView.vue:15,24-29`：全部/反应/提及/**转贴** | 「转贴」是自造维度；缺「指定/私信」（Firefish 的 directNotes） | tab 改为 全部/反应/提及/指定。指定 tab 先用 `includeTypes:["mention"]` + 前端二次判 `note.visibility === "specified"`（`Note.visibility` 在 `types.ts:72` 存在），并在 tab tooltip 注明「近似实现」 |
| `MkNotifications.vue:88-111`：`includeTypes`/`excludeTypes` 作为请求参数交给后端；`pages/notifications.vue:102-125` 另有按任意 `notificationTypes` 多选的筛选弹层 | `NotificationsView.vue:49-51` fetcher 只发 `limit/untilId`；`31-34` 的 `itemFilter` 交给 `PagedFeed` 渲染层过滤 | **筛选语义错误**：`PagedFeed.vue:191` 只在 `rows` 过滤，`items`/`hasMore` 仍按整页推进（`106,136`），于是「反应」tab 在有数据时也会闪一次空态（`PagedFeed.vue:326`），且空态文案（`36-47`）与仍在加载的事实矛盾 | 后端：`internal/note/reaction.go:163` 的 `List` 加 `includeTypes []string`（`AND type = ANY($n)`），`internal/httpapi/notifications.go:38` 解析出的 `IncludeTypes` 一路传下去（目前解析后丢弃）。前端：`client.ts` 的 `notifications()` 加 `includeTypes?: string[]`；fetcher 传当前 tab 类型；删 `31-34` 与 `:filter`（153-167） |
| `MkNotifications.vue:25-42`：按类型换呈现 —— 折叠通知 / reply-quote-mention 直接嵌 `XNote` / 其余走 `XNotification` | `NotificationItem.vue:16-27` 所有类型渲染同一种行（badge + 头像 + 文案 + 摘要）；`60-72` 摘要是 140 字纯文本 | 回复/提及通知看不到被回复的正文与上下文，只有一行截断文本 | `NotificationItem.vue` 中 `type ∈ {reply, quote, mention}` 且 `notification.note` 存在时，改渲染紧凑 `NoteItem`（`components/note/Note.vue:11-12` 已支持 `nested`），替掉 `105` 的 `.excerpt` |
| `pages/notifications.vue:78`：进入即标记全部已读；`127-145` 动作只在 all tab 出现 | `NotificationsView.vue:80-92` 需手动点；`99-104` 订阅 `readAllNotifications` 清点 ✓ | 行为差异：参考是「打开即消费」，我们是显式按钮 | 保留显式按钮（个人实例更安全），但补一条：切到非 all tab 时禁用「全部已读」（对齐 `127-145` 的 `tab.value === "all"` 条件） |
| `MkNotifications.vue:7-16` 空态仅一句 `noNotifications` | `NotificationsView.vue:36-47` 四套分 tab 文案 ✓ | 优于参考 | 保留 |

### 3. ProfileView（个人页）

| 参考做法（文件:行） | 现状 | 差距 | 建议动作 |
|---|---|---|---|
| `pages/user/index.vue:2-10`：sticky 页头 + `MkPageHeader v-model:tab` + back 按钮；`129-147` 标题=`姓名 (@username)`、副标题=`@acct`、头像 | `ProfileView.vue:338-347` 页头即 `ProfileHeader`（一张卡内横幅+头像），tab 在 `368-380` 的正文 `.tab-row` | 没有「页级标题 + tab」这层结构，tab 与资料卡分离且都会滚走 | 资料卡之上加 `PageHeader`：`title`=显示名、`subtitle`=`@username`、`tabs` 传入现有三项、`back`；删 `368-380` 与样式 `455-487` |
| `pages/user/index.vue:83-127`：tab = 主页/媒体/反应/剪藏/页面/画廊，按 `me.id`、`publicReactions`、`instance == null` 条件生成 | `ProfileView.vue:21-25` 三项：笔记/媒体/回复；`104-115` 用 `withReplies/withFiles` 区分，`112-115` 回复 tab 靠前端 `filter(n => !!n.replyId)` | 缺少「反应」维度；「回复」tab 用客户端过滤 + `19` 的 `maxPagesPerFetch=3` 补偿 | 保留三 tab；`PageHeader` 改好后 tab 可条件生成（`isSelf`/`publicReactions` 已在 `types.ts:45`）。回复 tab 补偿逻辑保留但迁移进 `PagedFeedResult.hasMore`（`PagedFeed.vue:84-87`），删掉手写循环 |
| `pages/user/home.vue:334-336`：宽屏右副栏 `<div v-if="!narrow" class="sub"><XPhotos/></div>`；`326-328` 窄屏把照片并回正文；`2-8` 内容宽 1100 | 无副栏。`AppShell.vue` 有 `#widgets` 槽但 `App.vue:24-27` 从不传，右栏恒为 `InstanceCard` | 宽屏右侧信息密度低；媒体只能靠切 tab 才看到 | 在 `ProfileView.vue:338` 外层包 `<div class="profile-layout" :class="{ wide: wide }">`，`wide` 时 `grid-template-columns: minmax(0,1fr) 300px`；副栏渲染 `usersNotes({ username, withFiles:true, limit:9 })` 的 3×3 缩略图（新建 `web/src/components/profile/PhotoGrid.vue`）；`@media (max-width:1099px)` 隐藏，断点与 `AppShell.vue:289-296` 对齐 |
| `pages/user/home.vue:280-307`：笔记/关注/关注者三个计数是可点的分区入口；Iceshrimp `ProfileView.razor:73-94` 用 `LabeledRuler` 分出「置顶笔记」「最新笔记」两个带标签分区，`77-80` 置顶 >4 时给「跳过置顶」 | `ProfileHeader.vue:76-89` 三个计数是纯文本；`ProfileView.vue:351-366` 有置顶区，但「最新笔记」无分区标签；置顶很多时无法快进 | 计数不可点；长置顶区没有跳出口 | `ProfileHeader.vue:77-89` 的「笔记」计数包成 `RouterLink` 到 `?tab=notes`；关注/关注者保持纯文本（`users/following`、`users/followers` 端点不存在，grep 无结果，需后端补，注释记 TODO）。置顶后加 `「跳到最新笔记」` 按钮滚动到锚点，照 `ProfileView.razor:77-81` |
| `pages/user/home.vue:321-325`：本人无置顶时显示「去置顶」提示 | 无 | 空置顶区对本人没有引导 | `ProfileView.vue:351` 加 `v-else-if="isSelf"` 分支，渲染一条 `notice info`：「置顶笔记会显示在个人页顶部，在笔记菜单里选择置顶」 |
| `pages/user/home.vue:448`：窄阈值 1000，`2` 窄时内容宽降到 800 | 无任何宽度切换；`ProfileView.vue:405-409` 翻页是手动按钮，与其他流（`PagedFeed.vue:278-292` 的 IntersectionObserver）不一致 | 同一产品内两种翻页手势 | 把 `121-174` 的抓取逻辑改成 `PagedFeed` 的 `fetcher`（返回 `PagedFeedResult` 表达 `hasMore`），删 `405-409` 与样式 `499-503` |
| `pages/user/index.vue:25-26`：错误态 `MkError @retry`，加载态 `MkLoading` | `ProfileView.vue:320-321` 骨架 ✓、`323-334` 不存在卡片、`336` 错误仅一行 `notice` 无重试 | 错误态不可恢复 | `336` 改用 `PagePlaceholder`（`components/PagePlaceholder.vue:18-24` 已有 slot）并塞「重试」按钮调 `load()` |

### 4. NoteView（笔记详情）

| 参考做法（文件:行） | 现状 | 差距 | 建议动作 |
|---|---|---|---|
| `pages/note.vue:174-194`：页头标题=`「{用户名} 的笔记」`、副标题=发布时间、头像；`2-9` `display-back-button` | `NoteView.vue:39-45` 页头只有返回按钮 + 常量「笔记」，无作者、无时间 | 页头零信息量，深链进来不知道在看谁的笔记 | `PageHeader` 传 `title = `${note.user.name \|\| note.user.username} 的笔记``、`subtitle = absoluteTime(note.createdAt)`（`lib/time.ts` 已有）、`back = true` |
| `pages/note.vue:17-23,55-61`：主笔记上下各挂同作者的更新/更早笔记块；`25-53` 中间用胶囊 `MkButton` 触发「加载更新/更早」（样式 `210-227`） | `NoteView.vue:50-59` 只渲染单条笔记（`NoteItem detail`） | 完全没有上下文：看回复通知点进来后无法上下浏览 | 在 `50-59` 前后各插一个 `PagedFeed`：after 用 `{ userId: note.userId, untilId: note.id, limit: 10 }`，before 用 `sinceId: note.id`（`client.ts` 的 `usersNotes` 已支持两参数）；默认折叠成一个「加载更早/更新」按钮，照 `note.vue:26-33` |
| `pages/note.vue:35-38`：远程笔记顶部 `MkRemoteCaution` 提示去原站 | 无。`Note.user.host` 在 `types.ts:5-14` 可用 | 远程笔记没有任何来源提示 | `NoteView.vue:51` 上方加 `v-if="note.user.host"` 的 `notice info`：「这条笔记来自 {host}」，附 `note.url` 原站链接 |
| `pages/note.vue:63-64`：`MkError @retry` / `MkLoading` | `NoteView.vue:47` 固定 160px 灰块；`48` 错误 `notice` 无重试 | 错误不可恢复 | `48` 换 `PagePlaceholder` + 重试按钮调 `load()` |
| Iceshrimp `Pages/SingleNote.razor:31-60`：页头动作位放 CW 显隐按钮；正文先渲染 `ascendants`（祖先回复串） | `NoteItem detail` 只做单条渲染（`components/note/Note.vue:66-68`） | 无回复串、无 CW 开关 | 回复串依赖 `notes/children` 类端点，当前不存在 → 列为后端缺口；CW 显隐可直接在 `PageHeader.actions` 加一个按钮，控制 `NoteContent.vue` 的 `cw` 折叠 |
| — | `DeckPageWindow.vue:1-60` 在 deck 弹窗里**重新实现**了一遍笔记/资料加载与渲染（`loadNote`/`loadUser`），而不是复用 `NoteView`/`ProfileView` | 页头、上下文、tab 的改造要做两遍 | 抽出 `web/src/components/note/NoteDetail.vue`（页头以外的正文+上下文），`NoteView` 与 `DeckPageWindow` 共用；`ProfileHeader` 已在两边共用，Profile 无需改 |

### 5. ExploreView（探索）

| 参考做法（文件:行） | 现状 | 差距 | 建议动作 |
|---|---|---|---|
| `pages/explore.vue:2-8`：tab 进 sticky 页头；`53-70` tab = 用户 / 精选；`10` 内容宽 1200 | `ExploreView.vue:61-64` 页头「探索」；`68-71` tab 本地/全球 | tab 不在页头；**分区语义跑偏** —— 我们做的是「第二条时间线」，与 `/` 的时间线页重复 | 先做结构（tab 上移进 `PageHeader` + query 状态），再改语义（见下两行） |
| `pages/explore.vue:27-33`：两个 tab 分别挂 `explore.users.vue` / `explore.featured.vue` 两个子组件 | `ExploreView.vue:73-91` 只有一个 `PagedFeed`，靠 `scope` 切换 API（`19`） | 无「用户」目录，无「精选」 | 「用户」需要用户检索端点（`users/search` 不存在）→ 后端缺口，列入待办。「精选」可用现有能力实现：`api.localTimeline({limit:100})` 按 `reactionCount`/`renoteCount`（`types.ts:74-77` 已有）排序取前 30。新建 `components/explore/FeaturedFeed.vue` |
| `pages/explore.featured.vue` 之类是独立文件、按需加载 | 全在一个 125 行文件里，两个 tab 共用一个 fetcher | 后续加分区会撑爆单文件 | 拆 `components/explore/TimelineScope.vue` + `FeaturedFeed.vue`，`ExploreView.vue` 只留 PageHeader 与 tab 切换 |
| — | `ExploreView.vue:21-41` 标签筛选是客户端 `text.includes("#tag")`，自管 `rawCursor`（25-40）来防跳页；`66` 用一条 `notice info` 显示当前筛选 | 无标签端点，只能拉全量再本地过；语义上「话题页」应是独立路由 | 短期保留；把「标签」做成 Explore 的第三个 tab（从最近 200 条本地笔记抽 `tags`（`types.ts:85` 已有）词频 top 20），点击跳 `/?tl=local&tag=`，替代现在只能从 URL 手动带 `?tag=` |

### 6. SettingsView（设置）

| 参考做法（文件:行） | 现状 | 差距 | 建议动作 |
|---|---|---|---|
| `pages/settings/index.vue:2-9`：sticky 页头 + `MkSpacer content-max 900` | `SettingsView.vue:204-207` 页头「设置」，无 sticky、无内容宽度约束 | 长页面无宽度上限，宽屏一行过长 | 页头换 `PageHeader`；内容容器加 `max-width: 900px; margin-inline:auto` |
| `settings/index.vue:351-367`：`.wide` 时 `display:flex`，`> .nav` 34% + `> .main flex:1`；Iceshrimp `Layout/SettingsLayout.razor:12-18` 同样是 `sidebar` + `body` 双列 | `SettingsView.vue:221-375` 五张 `.card` 纵向堆叠，`380-382` 只靠 `margin-top:16px` | **没有左导航+右内容**，设置项无层级 | 新建 `web/src/layouts/SettingsLayout.vue`：`grid-template-columns: 240px minmax(0,1fr); gap: 32px`，`@media (max-width:900px)` 单列 |
| `settings/index.vue:81-263`：`MkSuperMenu` 三组（基本/客户端/其他）+ 第四组危险区（清除缓存、退出带 `os.confirm`）；`MkSuperMenu.vue:79` 组间 0.5px 分隔线、`82` 组标题、`91-117` 条目（padding 10/8px、`border-radius:9px`、active 用 accent 背景） | 无菜单；退出登录在最后一卡（`366-375`），`192-195` 直接 `signout()` 无确认 | 无分组语义；危险操作无二次确认 | 新建 `web/src/components/settings/SettingsMenu.vue`：组 = 账户(账户/个人资料) / 外观与显示(主题) / 其他(实例信息) / 危险(退出)，条目照 `MkSuperMenu.vue:79-117` 的度量；对齐 `AdminView.vue:186-191` 的 `window.confirm` 用法 |
| `settings/index.vue:272-274,289-297`：宽屏且无子页时 `router.replace("/settings/profile")`；`13,30` 窄屏只显示菜单 | `router/index.ts:44-48` `/settings` 单路由，无子路由 | 无深链（`/settings/profile` 不存在）；移动端与桌面端共用一条长滚动 | 路由改嵌套：`{ path:"/settings", component: SettingsLayout, children:[ {path:"", component: SettingsIndex}, {path:"profile",...}, {path:"appearance",...}, {path:"account",...}, {path:"instance",...}, {path:"session",...} ] }`；`SettingsIndex` 在窄屏是菜单页、宽屏立即 `replace("/settings/profile")` |
| 拆页 | 现 5 段：账户 `221-241`、个人资料 `243-316`、外观 `318-336`、实例 `338-364`、会话 `366-375` | 单文件 557 行，5 个互不相关的域 | 拆 `views/settings/{Account,Profile,Appearance,Instance,Session}View.vue`，逻辑按现有 script 段落切割（`105-181` → Profile；`14-31` → Appearance；`36-64` → Instance 只读） |
| `pages/settings/index.vue:17-22` / `admin/index.vue:16-57`：用 `MkInfo` 在导航上方放告警（如邮箱未配置） | `SettingsView.vue:209` 会话未就绪只有一行文字；无告警位 | 缺「需要注意」的聚合位 | `SettingsLayout` 在菜单上方留一个 `slot name="notice"`；实例信息读取失败时（`362`）把告警提到这里 |

### 7. AdminView（管理）

| 参考做法（文件:行） | 现状 | 差距 | 建议动作 |
|---|---|---|---|
| `pages/admin/index.vue:1-60`：与设置同一套 `nav(左) + main(右)`，nav 内是实例 banner + 告警 `MkInfo` 列表 + `MkSuperMenu` | `AdminView.vue:255-258` 页头「管理」；`274-493` 三段纵堆（联邦模式 `274-318`、白名单 `320-394`、账号管理 `396-493`） | 缺左导航+右内容；三个独立域挤在一页 | 新建 `web/src/layouts/AdminLayout.vue`（视觉复用 `SettingsMenu.vue`，分组=概览/联邦/账号）+ `views/admin/{Overview,Federation,Users}.vue`；路由 `/admin` 嵌套，`/admin` → Overview |
| `admin/index.vue:16-57`：导航区顶部集中放告警（未处理举报、无维护者、无 bot 防护、无邮件、有更新） | 告警散落在段内：`320-332`（联邦非 allowlist 时白名单不生效）、`431-433`（注册模式 admin 时自助注册关闭） | 关键的实例级提醒在第二/第三段，首屏看不到 | `views/admin/Overview.vue` 做告警聚合：实例名/版本/联邦模式/注册方式 + 两条可点告警（跳 Federation / Users），数据来自 `instance.meta`（`338-364` 的既有渲染可整段搬过来） |
| `admin/index.vue` 每个子页一个文件、`_header_.vue` 共用页头 | 单文件 820 行承载三个域 | 文件过大；三段各自有独立的 notice 状态机（`modeNotice`/`hostNotice`/`usersNotice`/`createNotice`） | 按域拆文件，每个子页自带一个 notice；`PageHeader` 统一页头（`subtitle` 放实例域名便于确认在操作哪个实例） |
| 分页/检索 | `AdminView.vue:85-100` 白名单全量拉；`171-182` 用户列表 `limit:50` 无分页无过滤 | 100+ 用户时看不到第 51 个 | `client.ts` 的 `adminShowUsers` 已声明 `offset`；`Users.vue` 加「加载更多」（`offset += users.length`）+ 用户名前端过滤输入框（现有 50 条内即时过滤） |
| 危险操作确认 | `toggleSuspend` 已有 `window.confirm`（`186-191`）✓ | 优于参考 | 保留 |
| 联邦模式切换 | `AdminView.vue:69-83` 立即 PUT，无二次确认；`283-303` 三个大卡片按钮（含说明）✓ | 误触即改全实例联邦策略 | 加 `window.confirm`，文案说明「切换到 X 会立即影响所有投递」 |

### 8. LoginView（登录）

| 参考做法（文件:行） | 现状 | 差距 | 建议动作 |
|---|---|---|---|
| `pages/auth.vue:11-13`：未登录直接渲染 `MkSignin`；`MkSignin.vue:1-3` 表单根是 `_monolithic_` 容器，字段用 `_formBlock` 统一块间距 | `LoginView.vue:35-67` 用 `AuthLayout` + 4 处内联样式（`37` h2 1.5rem、`38` 副标题 margin、`59` 按钮 `width:100%`、`65` 底部链接 margin） | 表单没有自己的 class，间距靠内联样式散落；改版要逐处改 | 在 `main.css` 的 `.auth` 段落（`436+`）补 `.auth-form` / `.auth-form h2` / `.auth-form .sub` / `.auth-form .submit{width:100%}`，把 `37,38,59,65` 的内联样式搬进去 |
| `MkSignin.vue:21-38`：用户名输入带 `@` 前缀与 `@host` 后缀，用户知道不用填域名 | `LoginView.vue:43` 只有 placeholder `username` | 用户不确定是否需要写 `@域名` | `LoginView.vue:41-44` 的 field 下加 `<span class="hint">@{{ username || "username" }}@{{ host }}</span>`，`host` 取 `instance.meta?.uri` 的 host 或 `window.location.hostname`（`stores/instance.ts` 已在用） |
| `MkSignin.vue:39-57`：密码框带「忘记密码」按钮 | 无 | 忘记密码无出口 | 加一个 `_textButton` 风格按钮，点击展开 `notice info`：「密码重置需要管理员在服务器处理」，文案与 `SettingsView.vue:240` 一致，不做死链（无 reset 端点） |
| `MkSignin.vue:21` `autofocus`；Iceshrimp `Pages/Login.razor:43-54` 额外给多账号快速选择器 | `LoginView.vue:43` 无 `autofocus`；无多账号 | 每次都要点一下输入框（次要）；多账号是产品级差异，个人实例不需要 | 加 `autofocus`；多账号明确不做 |

### 9. SignupView（注册）

| 参考做法（文件:行） | 现状 | 差距 | 建议动作 |
|---|---|---|---|
| Iceshrimp `Pages/Register.razor:37-53`：实例规则需先勾选接受才显示表单（`RegistrationAvailability` + `AcceptedRules` 两段式） | `SignupView.vue:66-118` 表单直接显示，规则无从表达 | 无规则/条款环节（后端也无 rules 端点 → 记录为缺口，不做） | 不做规则门；把「注册模式决定谁能加入」这句（`69`）升级为按 `instance.meta.registrationMode` 的分支说明（`open`/`invite`/`closed`/`admin` 四态，标签映射已在 `SettingsView.vue:50-64`，抽成 `lib/instance-labels.ts` 复用） |
| `Pages/Register.razor:55-58`：仅邀请模式才渲染邀请码输入 | `SignupView.vue:102-105` 邀请码始终显示，标「可选」 | 开放注册时多一个迷惑字段 | `v-if="registrationMode === 'invite' || instance.meta.disableRegistration"` 再显示邀请码 |
| `Pages/Register.razor:70-74`：关闭注册时整块替换为 `ClosedNotice` + 切到登录的链接 | `SignupView.vue:18-20` 算出 `closed`，`109` 只是 `:disabled` 按钮，`71-73` 一条 notice | 表单仍在，用户可以填完才发现不能提交 | `closed` 时整块 `v-if/v-else` 替换为关闭说明 + `<RouterLink to="/login">`，照 `Register.razor:70-74` |
| `Pages/Register.razor:123-158`：`StateButton` 有 Initial/Loading/Success/Failed 四态 | `SignupView.vue:109-111` 两态（`创建中…`/`注册`） | 无成功态反馈（`55` 直接 await 后跳转，跳转失败时无提示） | 加 `created` 态：成功后先渲染一段 `notice info`「账号已创建，正在进入…」，200ms 后再 `router.push` |
| 用户名即时可用性（参考无此功能，Firefish 也没有） | `SignupView.vue:24-38` 去抖 400ms 查 `usernameAvailable` ✓ | 优于参考 | 保留；把 `87-89` 的三态提示加 `aria-live="polite"` |

### 10. NotFoundView（404）

| 参考做法（文件:行） | 现状 | 差距 | 建议动作 |
|---|---|---|---|
| `pages/not-found.vue:2-6`：`_fullinfo` 中央态（ghost 图 + 一句 `notFoundDescription`）；`19-22` 设页面元数据（标题「未找到」+ 图标） | `NotFoundView.vue:5-8` 用 `PagePlaceholder icon="search"` + 「回到时间线」按钮 | 图标语义不符（搜索 vs 找不到）；无浏览器标题 | 图标换 `compass`（`PagePlaceholder.vue:6-15` 的 icons 白名单里有），或给 `AppIcon` 加 `warning` 更贴 `not-found.vue:21` |
| — | 全站无 `document.title` 机制（`grep document.title` 无结果；`router/index.ts:60-64` 只有 scrollBehavior） | 10 个页面浏览器标签都显示同一个 `index.html` 标题，多标签时无法区分 | 在 `router/index.ts` 给每条路由加 `meta.title`，并加 `router.afterEach((to) => { document.title = ... })`；带参数的页面（笔记/个人页）由视图内 `watchEffect` 覆盖（如 `${user.name} (@${username}) · Cinderfish`） |
| `not-found.vue` 在 Firefish 里由 page metadata 决定头部，无返回按钮（靠常驻导航） | `App.vue:23-27`：`not-found` 没有 `meta.bare`，所以 404 渲染在 AppShell 内，左侧导航仍在 ✓ | 一致 | 保留 |

---

### 建议的执行顺序（按依赖）

1. **P1 页头组件**：新建 `web/src/components/ui/PageHeader.vue`（sticky + `backdrop-filter` + tab 高亮条 + `actions`/`back`/`subtitle`/`icon`、窄屏折叠），同步把 `main.css:401-409` 的 `.page-title` 收编进去。这是 7 个页面的共同前置。
2. **P2 tab 进 URL**：`TimelineView.vue:20-23`、`ExploreView.vue:69-70`、`NotificationsView.vue:145`、`ProfileView.vue:176-180` 的 tab 切换改为写 `route.query`，并从 query 回填。
3. **P3 设置页拆分**：`SettingsLayout.vue` + `SettingsMenu.vue` + 5 个 `views/settings/*` + 嵌套路由（影响最大、改动最独立）。
4. **P4 管理页拆分**：复用 P3 的菜单组件，`AdminLayout.vue` + 3 个子页（`Overview` 承接 `AdminView.vue:338-364` 的实例信息与两处告警）。
5. **P5 通知筛选语义**：后端 `List` 加 `includeTypes`（`internal/note/reaction.go:163` 与 `internal/httpapi/notifications.go:38-45`），前端 `client.ts` 的 `notifications()` 加参数、去掉渲染层 filter；顺带确认 `notifications.go:52-55` 把 `"read"` 塞进 `exclude` 的语义是否符合 Misskey `markAsRead`（这处看起来是「只取未读」而非「返回后标记已读」，需要对照 Misskey 端点契约确认后再动）。
6. **P6 个人页副栏 + 计数链接 + 统一翻页**；**P7 笔记页上下上下文 + 抽出 `NoteDetail.vue` 供 `DeckPageWindow` 复用**；**P8 浏览器标题机制**；**P9 登录/注册/404 收尾**。

### 需要上游确认的两点

- `InstanceMeta`（`web/src/api/types.ts:135-149`）缺 `disableLocalTimeline` / `enableGuestTimeline`，所以「按实例配置隐藏时间线 tab」无法照 Firefish `pages/timeline.vue:96-105` 实现，建议先只做登录态门控。
- `users/following`、`users/followers`、`users/search`、`notes/mentions`、`notes/children`、`notes/featured` 六类端点均不存在（已 grep `internal/httpapi`），因此参考实现里的「关注/关注者子页」「Explore 用户目录」「通知页 mentions tab 用 `notes/mentions`」「笔记回复串」「精选」在本轮都不能落地，只能给出近似替代或列为后端缺口。
</agent_swarm_result>

---
