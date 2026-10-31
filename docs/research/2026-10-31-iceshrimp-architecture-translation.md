# Iceshrimp 前端架构翻译契约（2026-10-31）

目的：为《Iceshrimp 前端架构重写实施计划》（`docs/superpowers/plans/2026-10-31-iceshrimp-architecture-rewrite.md`）冻结「参考源码路径 → Cinderfish Vue 组件」的翻译契约，供 Task 2–8 直接消费。本笔记只做架构映射与证据记录，不修改 `web/`、`internal/` 任何代码。

与既有笔记的关系：`docs/research/2026-10-29-frontend-redesign-reference-analysis.md` 与 `docs/research/2026-10-30-firefish-ui-gap-analysis.md` 记录了 Firefish（主要参考）视角的差距分析。本笔记以 Iceshrimp Vue fork（组件架构参考）为主、Iceshrimp.NET（仅布局/交互证据）为辅；Misskey 仍是后端 API 契约的唯一来源，本笔记不引用 Misskey 前端作为 UI 证据。

## 1. 证据状态与版本

| 参考源 | 版本/SHA | 位置 | 验证方式 |
|---|---|---|---|
| Iceshrimp Vue fork（`codeberg.org/banaanihillo/iceshrimp`） | `1fa829ed10f42f1511a033f5c22d01ec9efe5a41`（main，2025-07-25） | `tmp/refcheck/iceshrimp-vue/packages/{client,sw}/src` | 2026-10-31 从 Codeberg `archive/main.tar.gz` 下载完整归档（2,571,861 字节），与磁盘内容 `diff -rq` 逐字节一致（`packages/client`、`packages/sw` 均零差异） |
| Iceshrimp.NET | `2c61359de29fe3c9ea6a3b31268ceb2ce0e996a7`（2026-10-27） | `tmp/refcheck/iceshrimp/Iceshrimp.Frontend` | git sparse clone（仅 `Iceshrimp.Frontend`），HEAD 即该 SHA |
| Firefish（主要前端参考） | `02c31e0a2804de48c9d73028df525e197d8df622`（2024-12-11） | `tmp/refcheck/firefish/packages/client/src` | 既有 clone，本笔记不展开 |
| Misskey（仅 API 契约） | `c05e8443a9d1222a467617ed6019784a1f1a53e7`（2026-10-29） | `tmp/refcheck/misskey/packages/backend/src` | 既有 clone |

**重要更正**：此前认为 `tmp/refcheck/iceshrimp-vue` 已丢失（目录为空）。本次复核发现该目录已有完整 `packages/client/src`（2012 个文件）且与上游 `main` 归档逐字节一致，**Vue fork 全部路径均可用**，无需把任何证据降级为「仅 .NET 推断」。注意两点：

1. 该目录内的 `.git` 指向 `github.com/Aurorainic/Cinderfish.git`（一个 Cinderfish 克隆的残留），`packages/` 下的文件并未被其 git 跟踪（`git ls-files packages/` 为 0，`git log` 只显示 Cinderfish 提交）。以磁盘内容 + 上方归档 diff 为准，不要用该目录的 `git log`/`git remote` 判断来源。后续任务如需查看提交历史，应重新 clone 或引用上方 SHA。
2. 归档不含提交元数据，SHA 取自 Codeberg API（`/api/v1/repos/banaanihillo/iceshrimp/commits?sha=main`）。

下文所有行号均于 2026-10-31 在本工作树 `tmp/refcheck/` 中用 `sed -n`/`grep -n` 逐条核对过。引用前缀：`fork/…` = `tmp/refcheck/iceshrimp-vue/packages/client/src/…`；`net/…` = `tmp/refcheck/iceshrimp/Iceshrimp.Frontend/…`；`web/src/…` = 本工作树 `web/src/…`。

## 2. 布局层（universal / zen / deck / visitor）

### 2.1 universal（三区布局，Cinderfish 的 AppShell 对应物）

`fork/ui/universal.vue`：

- 模板结构（L2–165）：`XSidebar`（v-if="!isMobile"，L3）→ `MkStickyContainer.contents`（L5–17，header 槽放 `XStatusBars`，main 内只有 `<RouterView />`）→ `.widgets-container`（v-if="isDesktop"，L19–21）→ 浮动 widget 按钮（`!isDesktop && !isMobile`，L23–29）→ 移动端底部按钮条 `.buttons`（v-if="isMobile"，L31–115：菜单/主页/通知/messaging/widget 五个按钮，主页在 `index` 路由时改为回顶）→ 浮动发帖按钮（仅 `index`/`messaging` 路由，L117–136）→ 两个抽屉（menuDrawer L138–149、widgetsDrawer L151–162）→ `XCommon`（弹层/全局组件，L164）。
- 断点常量（L194–205）：`DESKTOP_THRESHOLD = 1100`、`MOBILE_THRESHOLD = 500`；`isMobile = deviceKind === "smartphone" || innerWidth <= 500`（resize 监听更新）。
- 默认 widget 集（L255–270）：widgets 为空时注入 `notifications` + `trends` 两枚右列 widget。
- 样式：`.widgets-container` 宽 300px、`position:sticky; top:0; max-height:100vh`（L501–514），`@media (max-width: 1090px)` 隐藏（L455 与 L494 各声明一次 `$widgets-hide-threshold: 1090px`，重复声明是 fork 的手误，翻译时取单一定义）；移动端内容区 `padding-bottom: env(safe-area-inset-bottom) + 96px`（L496–498）；底部按钮条 `position:fixed` + safe-area padding（L570–578），≤400px 时按钮高度收窄（L620–626）；menuDrawer 宽 240px 左侧滑入、widgetsDrawer 右侧滑入（L405–451、L516–532、L660–677）。

### 2.2 其余形态

- `fork/ui/zen.vue`（L1–7）：无侧栏、仅 `RouterView + XCommon` 的极简形态。
- `fork/ui/visitor.vue`（L1–19）：未登录访客形态，`DesignA/B`（`ui/visitor/a.vue`、`ui/visitor/b.vue`）二选一，同样挂 `XCommon`。
- `fork/ui/deck.vue`：`XSidebar` + `.main > .columns` 横向列带（L2–6），列按 `layout` 分组渲染（L16–30），多列文件夹（`.folder.column`）内有 `flexible` 列时 `flex:1; min-width:350px`；列 chrome 在 `fork/ui/deck/column-core.vue`，列状态在 `fork/ui/deck/deck-store.ts`，列类型文件 `tl-column.vue`、`notifications-column.vue`、`main-column.vue`、`widgets-column.vue` 等。
- 页面级窗口：`fork/components/MkModalPageWindow.vue` 与 `os.pageWindow`（`fork/ui/universal.vue:366` 的右键菜单入口）——即「同 URL 也可以作为浮窗打开」的 deck 交互。Cinderfish 的 `web/src/components/ui/PageWindow.vue` 与 `components/deck/DeckPageWindow.vue` 是其对应物。

## 3. 侧边栏与导航注册表

`fork/ui/_common_/navbar.vue`（桌面侧栏）：

- 布局（L2–141）：`.top`（banner 背景 + 头像按钮，L4–24）→ `nav.middle`（L25–121）→ `.bottom`（发帖按钮，L122–138）。
- 中段渲染逻辑（L37–72）：遍历用户可排序的 `menu`（`defaultStore.state.menu`），分隔符为 `"-"`；每个条目从 `navbarItemDef[item]` 取 `to`（渲染为 `MkA`）或 `action`（渲染为 `button`）、`show !== false`、`indicated`（红点）、`title`/`icon`。之后硬编码追加 admin 入口（`$i.isAdmin || $i.isModerator`，L74–96）、"更多"按钮（弹 `MkLaunchPad`，L97–110）、设置入口（L111–120）。
- 宽度与折叠：`$nav-width: 250px`、`$nav-icon-only-width: 80px`（L229–231）；`iconOnly` 条件 = `innerWidth <= 1279` 或用户偏好 `menuDisplay === "sideIcon"`（L167–179，含 `matchMedia("(max-width: 1279px)")` 监听）。

`fork/navbar.ts`（L13–136）：`navbarItemDef` 注册表是唯一导航真源，条目字段为 `title`、`icon`、`show`（computed）、`indicated`（computed）、`to` 或 `action`。通知项带 `indicated: computed(() => $i?.hasUnreadNotification)`（L14–20），follow-requests 仅在锁定账号/有待审关注请求时显示（L34–40）。

`fork/ui/_common_/navbar-for-mobile.vue`：移动抽屉菜单，与桌面栏消费同一个 `navbarItemDef`（L39–58 处的渲染逻辑相同）。

## 4. Widget 面板

- `fork/ui/universal.widgets.vue`（L1–110）：`<aside class="widgets">`，向 `MkWidgets` 传 `edit`/`widgets`/四个事件（`add-widget`/`remove-widget`/`update-widget`/`update-widgets`），自身实现把变更写回 `defaultStore.set("widgets", …)`（L48–81）；底部 `编辑 widget` / 退出编辑按钮（L12–27）。
- `fork/components/MkWidgets.vue`：编辑态使用 `VueDraggable`（`vue-draggable-plus`，L77）拖拽排序 + 右键菜单删除。
- `fork/widgets/index.ts`（L3–122）：widget 注册表——每个 `Mkw<Name>` 以 `defineAsyncComponent` 全局注册（L3–96），`export const widgets` 是可选名单（L98–122）。Cinderfish 不引入 `vue-draggable-plus`（全局约束），排序用上下移按钮（现状）或后续单列方案。

## 5. 页头（MkPageHeader）与 section actions

`fork/components/global/MkPageHeader.vue`：

- Props（L143–163）：`tabs?: Tab[]`（`key`/`title`/`icon`/`iconOnly`/`onClick`）、`tab?: string`、`actions?: {text, icon, handler}[]`（即 section actions）、`thin?`、`displayMyAvatar?`、`displayBackButton?`、`to?`；`defineEmits<{ "update:tab" }>`（L170–172）。
- 页面把 header 塞进 `MkStickyContainer` 的 `#header` 槽（用法见 `fork/pages/timeline.vue:2–10`、`fork/pages/notifications.vue:2–10`、`fork/pages/user/index.vue:2–9`、`fork/pages/settings/index.vue:2–8`），页面标题/图标经 `scripts/page-metadata` 的 `provide/inject` 通道传入（L174 `injectPageMetadata`）。
- 返回按钮：`displayBackButton && history.length > 1`（L165–168），`goBack()` 即 `window.history.back()`（L247–249）。
- tabs 交互：`>3` 个 tab 时进入 `collapse` 态（非活动 tab 只剩图标，L74、L510–523），活动 tab 用 `getBoundingClientRect` 计算高亮条（L262–277，注释明确 offset* 取整问题）；窄容器时点击标题区弹 tab 选择菜单 `showTabsPopup`（L201–215）。
- 响应式：容器宽 `< 500` 即 `narrow`（L285、288，ResizeObserver）；`@media (max-width: 500px)` 收窄内边距并隐藏跟随按钮文字（L312–317）；`@media (max-width: 700px)` 左侧栏限宽 40%、有 tabs 时隐藏标题（L318–343）。头高 `--height: 55px`，`thin` 时 45px（L301–303、360–367）。
- `MkSpacer`（`fork/components/global/MkSpacer.vue`）约束页面内容宽：时间线/通知 `content-max="800"`，个人页 `600`（`pages/user-info.vue:9`）或 `narrow ? 800 : 1100`（`pages/user/home.vue:2`），设置 `content-max="900"`（`pages/settings/index.vue:9`）。

## 6. Note 组件族（Task 6 的结构基准）

| fork 组件 | 关键行 | 职责 |
|---|---|---|
| `fork/components/MkNote.vue`（978 行） | L1–14 根元素（`isRenote` class、`v-size max [500,350]`）；L15–17 回复链入口 `MkNoteSub`（`detailedView`/`collapsedReply` 时不渲染）；L19–88 `.note-context` 竖线区：promotion/featured/pinned 提示（L29–44）、转贴条（L45–72：转贴者名 + 时间菜单 + 可见性图标）、折叠回复摘要（L74–86）；L97–105 主体（头像 + `MkNoteHeader`）；L105–111 正文 `MkSubNoteContent`；L133–184 footer（反应查看器、回复、`MkRenoteButton`、star/react、quote、菜单） | 时间线内普通 note；props `note/pinned/detailedView/collapsedReply/forceExpandCw`（L298–303） |
| 纯转贴判定 | L331–335：`note.renote != null && text == null && cw == null && fileIds.length === 0 && poll == null`；L345–350 `appearNote = isRenote ? note.renote : note` | 显示目标与操作目标分离（计划 Task 6 复核项 5 的上游依据） |
| `fork/components/MkNoteHeader.vue` | L2–31：用户名/`MkAcct` + 绝对时间（含 `updatedAt` 编辑标记 L14–25）+ `MkVisibility` + 实例 ticker | header 行 |
| `fork/components/MkNoteSub.vue` | L1–47：会话串中的嵌套回复（`conversation`/`depth`/竖线 + 头像列） | 回复链节点 |
| `fork/components/MkSubNoteContent.vue` | L1–75：CW 处理（`cwStyle: classic/modern` 双态、回复图标跳转 `notePage(note)#replyId`）、`isLong` 折叠 | 正文/折叠 |
| `fork/components/MkRenoteButton.vue` | L1–16 模板；L44–48 `canRenote = visibility ∈ {public,home} 或本人`（否则锁定图标）；L50–65 悬停拉取 `notes/renotes` 列表；L68–180+ 菜单按可见性提供 Renote(public/home/specified/followers) 动作 | 转贴动作；fork 不把 quote 放进主按钮，quote 由独立 `MkQuoteButton` 承担 |
| `fork/components/MkNoteSimple.vue` | L1–13 | 迷你 note（转发引用内） |
| `fork/components/MkNoteDetailed.vue` | L40–90 replies tab 与 `directReplies` 渲染（L197、233–234、346–359） | 单页详情 + 回复列表 |
| `fork/components/MkDateSeparatedList.vue` | L33–37 `getDateText` **只用 `getDate()`/`getMonth()+1`——不含年份**（与计划 Task 6 步骤 4「完整年/月/日比较」的要求相反，实现时需修正）；L55–75 在相邻条目日期变化处插入分隔符 | 日期分组（基于 `<TransitionGroup>` 的渲染函数组件） |
| `fork/components/MkPagination.vue` | L111 `SECOND_FETCH_LIMIT = 15`；L113–125 props `pagination/disableAutoLoad/displayLimit`；拉页用 `limit+1` 探测 hasMore（init 逻辑 ~L135–180） | 游标分页核心 |
| `fork/components/MkNotes.vue` | L25–29 每条 note 以 `note.id`（或 `_featuredId_/_prId_`）为 key；L64–73 `prefetchMore` 定时器按滚动距离提前拉页 | note 列表 = MkPagination + MkDateSeparatedList + MkNote |
| `fork/components/MkPostForm.vue` | L273–301 props：`reply/renote/channel/mention/specified/initialText/initialVisibility/initialFiles/initialLocalOnly/initialVisibleUsers/initialNote/editId` 等；emits `posted/cancel/esc` | 发布表单（回复/转贴目标是显式 props） |

## 7. 时间线（Task 5/6）

- `fork/pages/timeline.vue`：`MkPageHeader` 挂 `#header`（L3–10），tab 与 `src` 双向绑定；`MkSpacer :content-max="800"`（L11）；Swiper 虚拟滑动在源之间切换（L29–59，`swiper/vue` + Virtual 模块）——**Cinderfish 不引入 swiper**（全局约束），以 PageHeader tabs 直接切换替代；`timelines` 数组按实例配置裁剪（L98–111：home/local/social/recommended/global）；`headerActions` 为列表/天线两项（L204–224）；`headerTabs` 每项带 icon-only（L226–273）。
- `fork/components/MkTimeline.vue`：props `src/list/antenna/channel/sound`（L46–52）；L92–212 按 `src` 映射 `endpoint` + streaming channel（`notes/timeline`↔`homeTimeline`、`notes/local-timeline`↔`localTimeline`、`notes/hybrid-timeline`↔`hybridTimeline`、`notes/global-timeline`↔`globalTimeline`、`notes/mentions`（directs 用 `visibility: "specified"` 过滤）、`notes/user-list-timeline`、`channels/timeline`、`antennas/notes`）；L61–76 流式 note 走 `pagingComponent.prepend(note)` + 可选提示音；L237–241 `pagination = {endpoint, limit: 10, params}`。
- 关注关系变化时 `onChangeFollowing`：不在回溯态才整列 reload（L86–90）。

## 8. 通知（Task 5）

- `fork/pages/notifications.vue`：tabs = `all/unread/mentions/directNotes`（L68、145–166）；进入页面即调 `notifications/mark-all-as-read`（L74）；`headerActions` 提供「类型过滤」（`includeTypes` 单选菜单 + 清除，L98–121）与「全部已读」（L123–143）；mentions/directNotes 两个 slide 直接用 `MkNotes` 挂 `notes/mentions`（L85–96）。
- `fork/components/MkNotifications.vue`：props `includeTypes/unreadOnly`（L68–71）；pagination 参数 `includeTypes ?? undefined`、`excludeTypes = includeTypes ? undefined : $i.mutingNotificationTypes`、`unreadOnly`（L75–85）；`reply/quote/mention` 类型渲染整条 `MkNote`（`collapsedReply` 折叠其余），其余类型渲染 `MkNotification`（L21–34）；流式到达的通知在页面可见时立即标记已读（`stream.send("readNotification")`，L87–103）。
- `fork/components/MkNotification.vue`：props `notification/withTime/full`（L292+），按 `notification.type` 分支选择头像/图标。

## 9. 个人页（Task 7）

- `fork/pages/user/index.vue`：`MkPageHeader :display-back-button="true"` + `v-model:tab`（L3–9）；tab → 子内容切换（L10–28：`XHome/XReactions/XClips/XPages/XGallery`）；`headerTabs` 条件裁剪——reactions 仅本人或 `user.publicReactions`，clips/pages/gallery 仅本地用户（L85–115）。
- `fork/pages/user/home.vue`：个人页 overview 组合——banner + 头像 + `MkFollowButton`（L194）+ `XUserTimeline`（`pages/user/index.timeline.vue`，L348、367）。
- `fork/components/MkUserInfo.vue`：props `user: UserDetailed`（L96–97），姓名/`MkAcct`/简介折叠展示。
- 关注列表：`fork/pages/user/followers.vue`、`following.vue`（共用 `pages/user/follow-list.vue`）——依赖 `users/followers`/`users/following` 端点，**Cinderfish 后端目前没有这两个端点**（Task 8 分组决策），UI 需按计划给出真实的「暂不可用」状态。
- `fork/pages/user-info.vue` 是管理员的用户详情页（`MkSpacer :content-max="600"`），非普通个人页。

## 10. 设置与管理（Task 7）

- `fork/pages/settings/index.vue`：`MkPageHeader :display-back-button="true"`（L2–8）；`MkSpacer :content-max="900"`（L9）；`.nav`（`MkSuperMenu` 消费 `menuDef`）+ `section.main > RouterView` 双栏（L13–33），窄容器时只显示其一（`narrow` 由容器宽度决定）；`menuDef`（L92 起）按「基础设置/客户端设置/…」分组，每项 `{icon, text, to, active}`——与 Cinderfish `web/src/views/settings/SettingsLayout.vue` + `menu.ts` 的「侧栏 + 子路由」结构同构（Cinderfish 已就位，差异仅在窄屏表现与条目数量）。
- 管理端：`fork/pages/admin/`（`abuses.vue`、`announcements.vue`、`emojis.vue`、`database.vue`、`email-settings.vue` 等），均由路由 + 后端权限双控（与计划 Task 7 步骤 2 一致）。

## 11. Iceshrimp.NET 证据（仅布局/交互，禁止移植 C#/Blazor）

- `net/Layout/MainLayout.razor`（L23–48）：`.page` = `.sidebar`（AuthorizeView 包裹 `Sidebar`）| `.main`（`.top-bar` + `@Body`）| `.widget-panel`。`.top-bar` 内三个 `SectionOutlet`：`top-bar-icon`（L34）、`top-bar`（标题，L36）、`top-bar-actions`（L39）——这是「页面把自己的图标/标题/动作注入共享顶栏」的机制，等价于 Vue fork 的 `MkPageHeader :actions`。所有页面用 `<SectionContent SectionName=…>` 提供三段内容（`net/Pages/TimelinePage.razor:27–40`、`NotificationPage.razor:11–22`、`ProfileView.razor:17–29` 等）。
- `net/Layout/MainLayout.razor.css` 断点：`@media (840px <= width)` 隐藏移动返回键（L96–98）；`<840px` 隐藏 widget 列、动作按钮只留图标（`.action .label { display:none }`，L103–108）；`<600px` `.main` 预留 `margin-bottom: 5rem` 给底栏（L114–117）。
- `net/Layout/Sidebar.razor`：`popover="auto"` 侧栏（L13），头像 + `AccountDropdown`，`NavLink` 列表（时间线/通知/搜索/书签/关注请求/云盘…，L14–60）；`Sidebar.razor.css` 宽 `5rem`（L41–42），600px 上下切换（L163、176）。
- `net/Layout/WidgetPanel.razor`（L5–30）：实际只是「通知」widget 面板——头部 + `NotificationList` + 全部已读按钮。
- **`net/Layout/NavMenu.razor` 是 Blazor 模板残留样板**（"Iceshrimp.Frontend / Hub demo"，L1–30），不构成任何参考证据。
- .NET 侧的 840/600 断点即计划中「840/600」的出处；Vue fork 使用的是 1100/500/1090/1279 一族（见 §12）。两套证据并存时，翻译契约以 Vue fork 的数值为主（见 §12 的决策）。

## 12. 断点汇总与翻译决策

| 语义 | Iceshrimp Vue fork（证据） | Iceshrimp.NET（证据） | Cinderfish 现状 | 翻译契约 |
|---|---|---|---|---|
| 桌面/移动主分界 | 500（`universal.vue:195`，叠加 `deviceKind`） | 600（`MainLayout.razor.css:114`） | 720（`AppShell.vue:325`） | Task 2 定稿；fork 的 500 + `deviceKind` 组合是主证据，600 仅为 .NET 佐证 |
| widget 列隐藏 | 1090（`universal.vue:455/496/511`） | 840（`MainLayout.razor.css:103`） | 1099（`AppShell.vue:315`） | 保持 1090 附近数值（现状 1099 已近似），以 fork 为准 |
| 桌面内容三区成立 | 1100（`universal.vue:194`） | 840（.NET 把 widget 列与顶栏动作标签绑定在同一断点） | 1279/1099 双档 | Task 2 以 fork 的 1100 复核现状 |
| 侧栏图标态 | 1279（`navbar.vue:169`）+ 用户偏好 | 5rem 恒定 | 1279（`AppShell.vue:259`） | 数值一致，保留；用户偏好开关属可选增强（不进本计划范围） |
| 页头窄容器 | 500（`MkPageHeader.vue:285`）；媒体 500/700（L312/318） | 840 隐藏动作文字标签 | 窄容器 500（`PageHeader.vue:65`）、顶栏断点 720 | 补 .NET 的「动作文字标签隐藏」中间态：Task 2 在 `PageHeader.vue` 实现 |
| 底栏细节 | ≤400 收窄按钮（`universal.vue:620`） | — | ≤720 | 采纳 400 的细节增强 |

决策：**契约数值以 Vue fork 为主源，.NET 数值仅在 fork 未覆盖处（如动作标签中间态）补充引用**。计划中「840/600」两值来自 .NET 证据；Task 2/7 的浏览器核查清单中的 840/600 检查点保留（验证无水平溢出），但不作为 CSS 断点的定义值。

## 13. 翻译契约（面向 Task 2–8 的组件命名与边界）

计划已批准的 `web/src` 目标命名如下，本节给出其与 fork 结构的对应关系与行为契约：

| Cinderfish 组件（计划目标） | fork 对应 | 契约要点 |
|---|---|---|
| `web/src/App.vue` | `fork/ui/universal.vue` | 保留现有「deck 只接管三个路由主区」的实现（deck 是 Cinderfish 既有形态，fork 无此区分）；AppShell 继续承担 `XSidebar + contents + widgets` 三区组合 |
| `web/src/layouts/AppShell.vue` | `fork/ui/universal.vue` | 三区模板不变；断点复核按 §12；抽屉/底部条与 fork 的 menuDrawer/widgetsDrawer 对应 |
| `web/src/components/layout/Sidebar.vue`（新建） | `fork/ui/_common_/navbar.vue` | 只消费 `web/src/nav.ts` 导出；宽度 250/80 两档、≤1279 图标态；不含 fork 的 `MkLaunchPad`"更多"机制 |
| `web/src/components/layout/MobileSidebar.vue`（新建） | `fork/ui/_common_/navbar-for-mobile.vue` | 抽屉 + 底栏共用同一 nav 注册表；路由变化即关闭（对应 `universal.vue:250–253`） |
| `web/src/components/layout/WidgetPanel.vue`（新建） | `fork/ui/universal.widgets.vue` + `MkWidgets` | 渲染既有 `web/src/widgets/registry.ts` 的 stack；≤1099 隐藏；编辑态沿用上下移按钮（不引入拖拽库） |
| `web/src/components/ui/PageHeader.vue`（改造） | `fork/components/global/MkPageHeader.vue` | 保留计划接口：`title`/`back`/`tabs` + `#actions` slot（fork 用 `actions` 数组，Cinderfish 用 slot，二选一以计划为准——slot 保留）；补齐：tab 高亮条 `getBoundingClientRect` 精确测量、>3 tab 折叠、窄容器 `<500`、≤1099 动作标签隐藏 |
| `web/src/nav.ts` + `web/src/stores/nav.ts` | `fork/navbar.ts` + `defaultStore.state.menu` | 已是单一注册表（`NavKey`/`NAV_ITEM_DEFS`/badge/show）；Task 3 补 `requiresAuth`/`mobile` 字段与路由守卫 |
| `web/src/components/TimelineFeed.vue` | `fork/components/MkTimeline.vue` | 保留 `source → api + streaming` 双映射结构；Cinderfish 仅支持 `local/global/hybrid/home` 四源（无 social/recommended/antenna/list/channel） |
| `web/src/components/ui/PagedFeed.vue` | `fork/components/MkPagination.vue` + `MkNotes.vue` | 保留（fetcher 注入式设计已等价）；补日期分组与 `prepend` 队列语义对齐 fork |
| 日期分组 | `fork/components/MkDateSeparatedList.vue` | **不照抄**其 day-of-month-only 分组（L33–37），按计划实现完整年/月/日比较 |
| `web/src/components/note/*`（`Note.vue`/`NoteHeader.vue`/`NoteContent.vue`/`NoteFooter.vue`/`NoteMedia.vue`/`NoteParent.vue`/`RenoteBar.vue`/`NoteReactions.vue`/`use-note.ts`） | `MkNote.vue`/`MkNoteHeader.vue`/`MkSubNoteContent.vue`/（footer 内联）/(媒体列表)/`MkNoteSub.vue`/（renote 条 + `MkRenoteButton`）/`MkReactionsViewer.vue`/`useNoteCapture` | 保留 Cinderfish 命名；结构对齐：纯转贴判定、appearNote 分离、footer 动作顺序、转贴条可见性 |
| `web/src/components/NotificationItem.vue` + `NotificationsWidget.vue` | `MkNotification.vue` + `MkNotifications.vue` | 按 Task 5 接口重排：动作/行为者/关联笔记/已读态一行呈现，不过每行单独拉数据 |
| `web/src/views/TimelineView.vue` | `fork/pages/timeline.vue` | tab 键集裁剪为 Cinderfish 四源；`route.query` 语义保持现状 |
| `web/src/views/NotificationsView.vue` | `fork/pages/notifications.vue` | tab/过滤语义保持；Misskey 参数（`includeTypes`/`excludeTypes`/`markAsRead`）直通 |
| `web/src/views/ProfileView.vue` + `ProfileHeader.vue` | `fork/pages/user/index.vue` + `home.vue` + `MkUserInfo.vue` | tab 用 `route.query.tab`（现状已实现）；关系字段以 Task 4 的 Misskey 契约为准 |
| `web/src/views/settings/*`（`SettingsLayout.vue` + `menu.ts` + 子页） | `fork/pages/settings/index.vue` + `menuDef` | 结构同构，保留；补窄屏单栏切换 |
| `web/src/views/admin/*` | `fork/pages/admin/*` | 保留现有页面；导航/路由门控在 Task 3/7 |
| `web/src/widgets/registry.ts` + `components/widgets/*` | `fork/widgets/index.ts` + `Mkw*` | 保留异步注册 + defaultProps 模式（与 fork 同构） |

**明确不移植**（fork 中存在、本计划不引入）：`nirax.ts` 自研路由、`pizzax.ts`/`store.ts` 全局 store 体系、`os.ts` 弹层总线（Cinderfish 已有自己的 `web/src/os.ts` 轻量版）、`i18n.ts`/`MkMisskeyFlavoredMarkdown.vue`（MFM 渲染差异，Cinderfish 用 `lib/mfm.ts`）、`noteViewInterruptors` 插件管线、翻译调用、`Swiper`、`vue-draggable-plus`、`iceshrimp-sdk` 依赖（Cinderfish 用 `web/src/api/types.ts` 自有类型）。fork 中的 messaging/groups/channels/antennas/gallery/pages/clips 等功能 Cinderfish 后端未实现，对应 UI 一律不引入。

## 14. Cinderfish 现状 → 迁移决策表

下表为 Task 2–8 的边界输入；「保留」= 无需结构改动，仅按契约微调；「迁移」= 按对应 fork 结构重排。

| Cinderfish 文件 | 决策 | 依据（本笔记章节） |
|---|---|---|
| `web/src/App.vue` | 保留 | deck 路由接管逻辑（`DECK_ROUTES`）为 Cinderfish 特有且合理；fork 无等价物 |
| `web/src/layouts/AppShell.vue` | 迁移（Task 2） | 三区组合已存在，但需按 §2.1 核对断点族（1100/1090/500），拆出 Sidebar/WidgetPanel/MobileSidebar 三个子组件 |
| `web/src/layouts/DeckShell.vue` | 保留（Task 2 微调） | 列带 + 添加列选择器结构自洽；与 fork `ui/deck.vue` 的差异（贴边列、拖拽、文件夹）不进本计划 |
| `web/src/layouts/AuthLayout.vue` | 保留 | 对应 fork 未登录形态（`ui/visitor.vue`）的简化版，符合「认证布局保持裸」的约束 |
| `web/src/nav.ts` | 迁移（Task 3，小幅） | 注册表已是单一真源；补 `requiresAuth`/`mobile` 字段（fork `navbar.ts` 的 `show` 字段对应） |
| `web/src/stores/nav.ts` | 保留 | 已与 `nav.ts` 协同（desktop/mobile 两列偏好） |
| `web/src/components/ui/NavDrawer.vue` | 迁移（Task 2，职责并入 `MobileSidebar.vue`） | 与计划 Task 2 的新组件命名对齐 |
| `web/src/components/ui/MobileNav.vue` | 迁移（Task 2，并入 `MobileSidebar.vue`） | 底栏与抽屉应由同一组件/同一注册表渲染 |
| `web/src/components/widgets/WidgetStack.vue` / `WidgetsDrawer.vue` | 迁移（Task 2/7） | 拆为 `WidgetPanel.vue`（右列）+ 移动端抽屉；编辑态逻辑复用 |
| `web/src/components/ui/PageHeader.vue` | 迁移（Task 2） | 已有 tabs/back/narrow 基础；补高亮条精确测量、tab 折叠、动作标签中间态（§5、§12） |
| `web/src/components/ui/PageWindow.vue` | 保留 | 对应 fork `MkModalPageWindow`/`os.pageWindow` 的简化版 |
| `web/src/components/TimelineFeed.vue` | 保留（Task 6 微调） | 四源映射 + 客户端 tag 过滤已是现状约束；流式 prepend 语义对齐 fork |
| `web/src/components/ui/PagedFeed.vue` | 保留（Task 5/6 微调） | fetcher 注入式分页等价 `MkPagination`；补日期分组 |
| `web/src/components/note/*`（9 个文件） | 迁移（Task 6，结构对齐为主） | 命名保留；按 §6 对齐纯转贴/appearNote/footer 结构 |
| `web/src/components/NotificationItem.vue` | 迁移（Task 5） | 按 §8 的 fork 呈现规则 |
| `web/src/views/TimelineView.vue` / `NotificationsView.vue` | 保留（Task 5/6 微调） | tab 语义已与 fork 对齐 |
| `web/src/views/ProfileView.vue` / `components/ProfileHeader.vue` / `components/profile/*` | 迁移（Task 4/7） | 关系字段依赖 Task 4 的 Misskey 契约；tab 结构保留 |
| `web/src/views/SettingsView.vue` + `views/settings/*` | 保留（Task 7 微调） | 「侧栏 + 子路由」已与 fork 同构（§10） |
| `web/src/views/AdminView.vue` + `views/admin/*` | 保留（Task 7 微调） | 权限双控已就位 |
| `web/src/widgets/registry.ts` + `stores/widgets.ts` + `components/widgets/*` | 保留（Task 7 微调） | 注册表模式与 fork 同构（§4） |
| `web/src/api/*`、`web/src/stores/{session,streaming,theme,instance,deck,shell}.ts`、`web/src/lib/*`、`web/src/router/index.ts` | 保留 | API/主题/流式/路由均为 Cinderfish 自有资产；路由 `bare` 元数据机制保留 |
| `web/src/components/deck/*`（`DeckColumn.vue`、`DeckPageWindow.vue`、`deck-links.ts`、`deck-page.ts`）+ `stores/deck.ts` | 保留（Task 4 微调 `DeckPageWindow.vue`） | deck 列体系为 Cinderfish 自有形态 |

## 15. Cinderfish 与 Misskey API 的显式差异（翻译契约的一部分）

计划要求「前端不得虚构远端数据」。下列差异是 Tasks 4–8 实现 UI 时必须显式降级的点（fork 参考在此处不能直接照抄）：

1. **关系字段**：`users/show` 的 `isFollowing/isFollowed/hasPendingFollowRequestFromYou/hasPendingFollowRequestToYou` 由 Task 4 后端补齐；在此之前 UI 不得展示伪造关系态（对应 fork `MkFollowButton` 的直接消费）。
2. **通知过滤**：`includeTypes`/`excludeTypes`/`markAsRead` 的服务端过滤在 Task 5 实现；fork 的 `excludeTypes` 来自 `$i.mutingNotificationTypes`（`MkNotifications.vue:75–85`），Cinderfish 后端字段需按 Misskey `i/notifications` `paramDef` 对齐。
3. **关注/粉丝列表**：`users/followers`/`users/following` 端点尚缺（§9），个人页对应 tab 呈现真实「暂不可用」态（Task 7/8 分组决策）。
4. **时间线源**：Cinderfish 仅 `notes/local-timeline`、`notes/global-timeline`、`notes/hybrid-timeline`、`notes/timeline` 四源；fork 的 social/recommended/antenna/list/channel 列不进入 UI。
5. **转贴动作**：`MkRenoteButton` 的菜单依赖 `notes/renotes`（悬停用户列表）；Cinderfish 后端未实现该端点时，悬停列表功能降级省略，转贴本身按 Misskey `notes/create {renoteId, visibility}` 契约保留。
6. **widget 数据**：fork 的 `trends/federation/jobQueue/serverInfo` 等 widget 依赖不存在的端点；Cinderfish widget 注册表维持现有五枚（`registry.ts:23–32`）。
