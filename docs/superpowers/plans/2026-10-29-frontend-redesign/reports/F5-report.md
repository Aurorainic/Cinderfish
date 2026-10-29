# Task F5 Report: deck 列容器 + PageWindow 导航

**Status: DONE_WITH_CONCERNS**
**Commit: (见文末) `feat(web): adaptive deck columns with persistent config`**

## What was implemented

1. **`web/src/stores/deck.ts`（新建）**
   - 简报接口全部按原样提供：`ColumnType`、`ColumnConfig`、`deckColumns: Ref<ColumnConfig[]>`、
     `addColumn(type, index?)`、`removeColumn(id)`、`moveColumn(id, toIndex)`。
   - 持久化：localStorage 键 `cinderfish.deck.columns`，`watch(deckColumns, deep)` 统一写回（直接赋值
     也会被持久化，与 F2 `shell.ts` 的写法一致）。
   - 读取与回退（Review Focus 2）：`localStorage` 读不到 / JSON 解析抛错 / 解析结果不是数组 → 默认
     `[local, notifications]`；条目逐个校验（`type` 必须是六种合法值、`id` 必须是非空字符串、id 重复
     的丢弃）；校验后为空 → 同样回退默认。**部分非法**（如三项里两项类型非法）保留合法项，不整体回退。
   - 简报之外的最小增补：`setColumnType(id, type)`（列菜单「切换数据源」需要就地改类型，避免
     remove+add 改变列位置与滚动位置）、`COLUMN_TYPES`、`COLUMN_META`（列名/图标/说明，供列头与
     「添加列」菜单共用）、`ColumnIcon` 字面量联合类型（图标名写错会直接 typecheck 失败）。

2. **`web/src/components/deck/deck-page.ts`（新建，20 行）**
   - deck 模式的「页面覆盖层」状态机：`deckPage: Ref<{kind:"note",id} | {kind:"profile",handle} | null>`、
     `openDeckPage` / `closeDeckPage`。列内任何位置都能唤起，`DeckShell` 只渲染一层。

3. **`web/src/components/deck/deck-links.ts`（新建，70 行）**
   - deck 导航规则的实现：列体与覆盖层都用 `@click.capture` 挂 `interceptDeckClick`。
   - 命中 `/notes/:id`、`/@:handle` 的同源链接时 `preventDefault` + `stopPropagation` 并打开覆盖层；
     其余链接（登录、外部文件、`/explore` 等）保持原行为。带修饰键 / 中键点击不拦截（保留新标签页）。
   - 笔记正文纯文本点击也打开该笔记：DOM 里笔记行没有 id，id 从它自带的 `a[href^="/notes/"]`（时间戳）
     反查；按钮 / 表单 / 媒体 / 可编辑区一律放行，正在拖选文本时不拦截。
   - **为什么用捕获期拦截而不是改 Note/NoteHeader**：简报建议「给 Note 组件加 openInWindow 注入」，
     但 `web/src/components/note/**` 属 F6/F7，`os.ts` 属 F7/F9，本任务不允许改。捕获期拦截在
     RouterLink 之上，零侵入地实现了同一条规则，且 Note 组件在 universal 模式下仍然是普通路由链接。

4. **`web/src/components/deck/DeckColumn.vue`（新建）**
   - 列头：图标 + 名称 + 「列菜单」（`数据源` 六项单选 / 左移 / 右移 / 关闭列，边界项 disabled）。
     菜单点击外部（pointerdown）关闭、Esc 关闭、`aria-haspopup`/`aria-expanded`/`menuitemradio`。
   - 列体映射：home/local/hybrid/global → `TimelineFeed`（source 同名）；notifications →
     `PagedFeed(fetcher=api.notifications, streamChannel="main", streamType="notification")` +
     `NotificationItem`，未登录显示登录占位；explore → 列内「本地/全球」切换 + `TimelineFeed`。
   - 尺寸：`flex: 0 0 350px`，列内 `overflow-y: auto` 独立滚动（`overscroll-behavior: contain`），
     视口 < 1100px 时 `flex-basis: 100%`（一列一屏，交给 strip 的 scroll-snap 横滑）。
   - PagedFeed/TimelineFeed 完全按已提交的 API 使用，**未请求任何新 prop**。

5. **`web/src/components/deck/DeckPageWindow.vue`（新建）**
   - 用 F3 的 `PageWindow`：标题「笔记」/「@handle」+ 关闭按钮 + Esc，内容为
     - 笔记：`api.showNote` + `Note`（`detail`），删除后自动关闭；
     - 个人页：`usersShow` + `ProfileHeader`（关注/取消关注就地实现）+ `PagedFeed(usersNotes)`；
       远程 handle（`user@host`）显示「仅支持本地账户」说明。
   - 覆盖层内部同样挂 `interceptDeckClick`，在覆盖层里点别的笔记/头像会替换当前页而不是离开 deck。

6. **`web/src/layouts/DeckShell.vue`（新建）**
   - 横向 strip：`display:flex` + `overflow-x:auto` + `scroll-snap-type: x mandatory`，列 `scroll-snap-align:start`；
     高度 `calc(100dvh - 108px)`（对齐 `.shell` 24px 内边距 + `.main` 60px 下内边距，deck 模式下页面
     不再出现纵向滚动条），≤720px 用 `calc(100dvh - 104px)` 让开底部导航。
   - 末尾「添加列」竖条 + 列类型选择浮层（六种类型带说明）；无列时给一句提示。
   - 渲染唯一的 `DeckPageWindow`。

7. **`web/src/App.vue`（改）**
   - `shellMode === "deck"` 且当前路由在 `home / notifications / explore` 时主区渲染 `DeckShell`，
     否则渲染路由组件。`/notes/:id`、`/@:handle`、`/settings`、`/admin` 保持路由 —— 直接打开这类
     链接仍然可用（deck 内的点击本来就不走路由，走覆盖层）。

8. **`web/src/layouts/AppShell.vue`（改，仅切换按钮）**
   - 侧栏底部新增「Deck / 通用」切换按钮（`aria-pressed`、title 随状态变化、`:focus-visible` 描边），
     调用 `setShellMode`；收起态（<1280px）压成 40px 圆角标签，不破坏图标栏。
   - 除该按钮及其样式外未改 AppShell 任何结构。

## 验证

```bash
cd web && pnpm typecheck   # PASS
cd web && pnpm build       # PASS（vue-tsc --noEmit && vite build，409ms）
```

（首次 typecheck 报 `components/note/NoteContent.vue` 的 `watch` 未定义 —— 属 F7 正在写入的文件，
按协作规则等 20 秒重试后 PASS，未改他人文件。）

### 真实实例 + 内置浏览器手动验证

本地起真实后端（`config.example.toml`，dev DB `cinderfish`，127.0.0.1:3000）+ vite dev（5173），
浏览器实测：

1. **切换**：点「Deck」→ 主区变列容器；点「通用」→ 回到三栏页面（按钮文案随状态互换）。刷新后
   模式保持（`cinderfish.ui.shellMode`）。
2. **默认列**：首次进入是「本地（本地时间线）」+「通知（未登录占位）」+「添加列」。
3. **列内导航规则**：点笔记时间戳 → 弹出 `PageWindow`「笔记」，**URL 仍为 `/`**；关闭后点头像 →
   弹出「@f4friend」个人页（ProfileHeader + 该用户笔记），URL 仍为 `/`。
4. **universal 规则**：切回「通用」后点同一时间戳 → **URL 变为 `/notes/00mvtmj52d56wgw823`**，
   渲染 `NoteView`。
5. **列菜单**：把第一列数据源切成「探索」→ 列内出现本地/全球切换并重新加载；刷新后仍是探索列
   （localStorage 持久化生效）。
6. **添加列**：+ → 选「全球」→ 第三列出现（本地/通知/全球三列）；关闭列 → 该列消失、其余列保留。
   浏览器验证时发现此处有一处真实 bug（见下），已修复并复测。
7. **窄视口**：1000×820（Rail 收起、widgets 隐藏）→ 每次只显示一列（下一列被 snap 到屏外），
   页面本身无横向滚动条，只有 strip 自己的横向滚动条；702×897（移动布局）同样单列横滑。
8. **1280 宽**：两列 350px 并排 + 右侧 widget 列，页面无横向溢出、无纵向滚动条。
9. **deck 模式下的其它路由**：`/settings` 仍然渲染设置页（deck 不接管），deck 状态保持；
   直接打开 `/notes/:id` 渲染 `NoteView`。

### 浏览器验证中发现并修复的 bug

「添加列」浮层当时渲染在 `addRoot` 元素之外，而点击外部即关闭的监听挂在 `pointerdown` 上：
按下菜单项时先触发关闭、菜单被卸载，`click` 永远到不了菜单项 —— 添加列实际不可用。
修复：把浮层移进 `addRoot` 内部（`position: fixed` 不受 strip 的 overflow 裁剪），复测三次
六种类型均可添加。

## TDD evidence

简报未列测试步骤，web 也无测试套件（`package.json` 无 vitest/test 脚本），因此没有 RED-GREEN
流程；验证为 typecheck + build + 上述手动矩阵。非法 localStorage 回退（Review Focus 2）由
`stores/deck.ts` 的 `sanitize/readColumns` 代码路径覆盖，但**未能做运行时注入验证**（内置浏览器
无法执行任意 JS 写 localStorage），见 Concerns。

## Files changed

- `web/src/stores/deck.ts`（新建）
- `web/src/components/deck/deck-page.ts`（新建）
- `web/src/components/deck/deck-links.ts`（新建）
- `web/src/components/deck/DeckColumn.vue`（新建）
- `web/src/components/deck/DeckPageWindow.vue`（新建）
- `web/src/layouts/DeckShell.vue`（新建）
- `web/src/App.vue`（deck 路由表 → DeckShell）
- `web/src/layouts/AppShell.vue`（仅 Deck/通用 切换按钮 + 其样式）

未修改 `components/ui/PageWindow.vue`：F3 的版本（标题栏 + 关闭 + Esc + 700px/移动全屏）已满足
deck 覆盖层需要，无需改动。未修改 `PagedFeed.vue` / `TimelineFeed.vue` / `views/**` / `os.ts`。

## Concerns / notes for reviewer

- **deck 模式下右侧 widget 列仍在**：`AppShell` 只允许我加切换按钮，所以 1280px 视口下主区约
  600px，只能完整显示一列 350px + 第二列的一部分（横向滚动可见）。要让 deck 真正并排多列，
  需要一个很小的改动：deck 模式下隐藏 `.widgets`（改成 `grid-template-columns: rail 1fr`）。
  这是 F2/F12 的文件，未越界修改。
- **未做运行时注入验证的项**：非法 localStorage（如 `cinderfish.deck.columns` 写入 `"oops"` 或
  `[{type:"bogus"}]`）的回退只有代码路径保证。若评审要求实测，需要能执行 JS 的手段或加一个
  water 测试工具（本任务不允许新增依赖）。
- **点什么算「点笔记」**：当前 = 笔记行内的时间戳/头像/昵称链接 + 正文文本（排除按钮/媒体/选中文本）。
  笔记行的空白内边距与反应条不触发打开，避免误触。
- **覆盖层不支持「栈」**：在覆盖层里点另一条笔记会替换当前页而非压栈，关闭即回到列。若需要
  Misskey 式多级窗口，`deck-page.ts` 状态换成数组即可（未做，超出简报）。
- **探索列不做 tag 过滤**：`/explore?tag=` 的 tag 过滤没有搬进探索列（deck 内点击话题会走路由到
  `/explore?tag=...`，而此时 deck 接管主区，过滤结果不可见）。若要一致，需要给探索列加 tag 参数
  或把话题点击也做成覆盖层 —— 建议留给 F12 收尾判断。
- **验证用进程未清理**：本任务在 127.0.0.1:3000 起了后端、5173 起了 vite dev（配置在
  `/tmp/cinderfish-f5.toml`），同波次其它 agent 也在用，故保持运行，未 kill。
