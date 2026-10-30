# Task F4 Report: PagedFeed 统一分页组件 + TimelineFeed 重写

**Status: DONE**
**Commits: `efc1ee2` `feat(web): unified PagedFeed with streaming prepend and date separators` +
`46c34df` `fix(web): PagedFeed hinted fetcher pagination; refcounted streaming main`**

## What was implemented

1. **`web/src/components/ui/PagedFeed.vue`（新建，通用分页组件）**
   - 接口按简报：`fetcher(p: { untilId?: string; sinceId?: string })`、`streamChannel?`、
     `pageSize?`（默认 20）、`disabled?`；slots `#item="{ item }"` / `#empty`；expose `refresh()`。
   - 内置能力：
     - 无限滚动：IntersectionObserver（rootMargin 240px）观察底部哨兵；每次加载完成后额外做一次
       getBoundingClientRect 视口内检查，避免哨兵已可见但交叉状态未变化时卡住。
     - 去重：`Set<id>`（id 取自 `item.id`；无 id 的条目不参与去重）。分页游标 `untilId` 为严格小于
       （后端 `n.id < $untilId`），全量重复页时把 `hasMore` 置 false，防止同游标无限重取。
     - `__reconnected`（channel === ""）：整页重载（fetch 第一页并整体替换），清空浮标累计。
     - 浮标：流式新条目**不直接插入**，累计进 `pending` 并显示「N 条新笔记」按钮；点击后前插、
       去重、清零并平滑滚动回顶部。
     - 日期分隔行：相邻条目的 UTC 日期（`createdAt` 前 10 位）不同时插入居中小字分隔行
       （格式「2026年10月29日」，两侧 hairline）。无 `createdAt` 的条目不打断相邻比较。
   - 对简报接口的三处**最小补充**（均为可选、默认值与现有行为一致）：
     - `streamType`（默认 `"note"`）：时间线频道事件类型为 `note`，`main` 频道为 `notification`，
         通知页接入必须能指定事件类型，否则会把 main 频道其它事件（follow 等）当成条目。
     - `newItemsLabel`（默认 `"新笔记"`）：通知页浮标文案为「N 条新通知」。
     - `filter`（显示层过滤函数）：通知页「全部/未读」客户端 tab 需要（F10 的类型 tab 同样消费）。
   - 额外 expose `remove(id)`：NoteCard 删除后从列表移除（否则视图层无法操作 PagedFeed 私有列表）。
   - emit `items`（每次列表变动后广播当前条目数组）：通知页用它与全局未读数同步。
   - 组件状态：初始加载骨架屏、错误 notice、`disabled` 变化时清空/重载、卸载时退订 + 断开 observer
     + 移除 streaming handler（handler unsubscribe 泄漏在首次浏览器验证时发现并已修复）。

2. **`web/src/components/TimelineFeed.vue`（重写为薄封装，248 行 → 71 行）**
   - 保留：source→API 映射（local/global/hybrid/home）、source→流式频道映射
     （localTimeline/globalTimeline/hybridTimeline/homeTimeline）、hybrid/home 的未登录占位。
   - 删除：内嵌 NoteComposer（F9 以 `os.openPostForm` + PostForm 接管）、手写「加载更多」、
     去重/重连逻辑（下沉进 PagedFeed）、NoteComposer 的 posted/renoted 前插逻辑。
   - tag 客户端过滤保留（后端无 tag 搜索端点，全局约束禁止新增端点）。由于过滤后 PagedFeed
     的游标来自被过滤列表，fetcher 自持原始游标：带 tag 时忽略传入 untilId、用内部 rawCursor
     翻页，重载（无 untilId）时归零，避免翻页跳过被过滤掉的条目间隙。
   - `#item` slot 渲染 NoteCard（F6 完成后可无痛换成新 Note 组件）；`@deleted` → `feed.remove(id)`。
   - NoteCard 的 `reply`/`renoted` 事件暂不消费：reply 的列内 composer 已删（F9 接管），
     renote 产生的新笔记经流式频道进浮标（行为等价，略去直接前插）。

3. **`web/src/views/ExploreView.vue`**
   - 列表直接换用 PagedFeed：本地 fetcher（local/global 映射 + 同款 tag 客户端过滤 + rawCursor）、
     `streamChannel` 按映射、`#item` 用 NoteCard、`#empty` 占位。`scope`/`tag` 变化经
     `:key` 触发 PagedFeed 重挂载即整页重载。tabs/话题 notice UI 保持不变。

4. **`web/src/views/NotificationsView.vue`**
   - 列表换用 PagedFeed：fetcher = `api.notifications`，`streamChannel="main"`、
     `streamType="notification"`、`newItemsLabel="新通知"`，`#item` 渲染 NotificationItem。
   - 保留：未登录占位、全部/未读 tab（经 PagedFeed `filter` prop 客户端过滤）、全部标为已读
     （直接改 emit 出来的条目对象并重新同步未读数）、未登录/未读徽标。
   - 未读数同步改为消费 PagedFeed 的 `items` emit（加载、重载、翻页、浮标插入后都会触发），
     替代原先手写的流式前插 + 手动 setUnreadNotifications。

## 验证

```bash
cd web && pnpm typecheck   # PASS（vue-tsc --noEmit，无输出）
cd web && pnpm build       # PASS（vue-tsc --noEmit && vite build，338ms）
```

（F1/F3 提交后重跑过一轮；F4 修复 handler 泄漏后再次 build 通过，build 脚本内含 vue-tsc。）

### 手动验证（真实实例 + 内置浏览器）

本地起真实后端（dev DB `cinderfish`，3861 端口，Redis 开启），seed 26 条笔记 + 既有历史数据：

1. **分页**：时间线滚动到底，无限滚动依次加载，滚动 6000px 后列表到 #1 并继续显示历史笔记，
   停止位置正确（无重复、无跳条）。
2. **日期分隔**：今日笔记与 10-28 / 10-26 的历史笔记之间出现居中分隔行「2026年10月28日」
   「2026年10月26日」（截图确认）。
3. **浮标累计**：页面打开期间 curl 发新笔记 → 浮标「1 条新笔记」出现且列表顶部**未变化**；
   再发一条 → 「2 条新笔记」。
4. **浮标插入**：点击浮标 → 新笔记前插到顶部、浮标消失、无重复。
5. **重连重载**：kill 后端进程再重启（WS 断开→自动重连→`__reconnected`）→ 列表整页重载，
   顶部为最新笔记，无重复。
6. **通知页**：另一账号对 f4tester 的笔记反应 → 通知页浮标「1 条新通知」，列表未变；
   点击后插入，头部未读数同步为「2 条未读」、tab 徽标为 2；NotificationItem 经 `#item` 正常渲染。
7. **探索页**：local/global tab 正常；`/explore?tag=26` 只显示含 `#26` 的笔记。
8. **鉴权门**：未登录时 hybrid/home 显示「需要登录」占位卡（guest 视图确认）。

## TDD evidence

简报未列出测试步骤（web 无测试套件，package.json 无 vitest），验证为 typecheck + build + 上述手动
矩阵，无 RED-GREEN 流程。

## Files changed

- `web/src/components/ui/PagedFeed.vue`（新建）
- `web/src/components/TimelineFeed.vue`（重写为薄封装）
- `web/src/views/ExploreView.vue`（列表换 PagedFeed）
- `web/src/views/NotificationsView.vue`（列表换 PagedFeed）

## Concerns / notes for reviewer

- **接口补充**：PagedFeed 在简报接口之上新增三个可选 prop（`streamType`、`newItemsLabel`、
  `filter`）与 `remove(id)`、`items` emit。动机见上；默认值下行为与简报一致。若评审认为超出
  接口范围，`filter`/`newItemsLabel` 可等价挪到消费方（代价是 PagedFeed 需要把条目数组交还
  给视图层，反而更绕）。
- **tag 过滤翻页**：客户端 tag 过滤依赖 fetcher 内部 rawCursor（TimelineFeed 与 ExploreView 各有
  一份 ~15 行的实现）。过滤后某页为空时停止无限滚动（与改动前行为一致）；接后端 tag 搜索
  端点后可以整体删掉。
- **NoteCard reply 暂时无响应**：内嵌 NoteComposer 按 F4 简报删除，回复按钮暂无动作，F9 的
  `os.openPostForm` 接管。renote 的新笔记不再直接前插，改走流式浮标（与 Misskey 行为一致）。
- **重连重载对滚动位置的影响**：`__reconnected` 整页替换，条目 id 不变时 DOM 复用，滚动位置
  基本保持；若首屏内容变化（服务端有新帖），位置会有轻微偏移，属预期。
- 验证用的临时后端/配置/二进制已清理（/tmp/cinderfish-f4*），未触碰 dev DB 数据结构。

---

## 评审修复（fix commit）

评审发现两个 Important 缺陷 + 一组 Minor，全部修复：

### Important 1 — tag 过滤页停止翻页（修复）

PagedFeed 原来 `hasMore = page.length >= pageSize`：ExploreView 的 fetcher 返回过滤后数组，
tag 命中少于 pageSize 时第 1 页后 hasMore 即为 false，翻页中止；过滤后空页同样误停。

修复：fetcher 可返回 `{ items, hasMore }` 提示对象（纯数组 fetcher 继续工作，完全兼容）：
- `PagedFeedResult = unknown[] | { items: unknown[]; hasMore?: boolean }`。
- 提示模式下 hasMore 直接采用（raw 页长度判断）；空过滤页不再中止。
- 游标语义澄清：load 传 `{}`（untilId === undefined = 重载），loadMore 传
  `{ untilId: cursor }`（含 "" = 无可用游标），过滤型 fetcher 以 `undefined` 重置内部
  rawCursor、以 `""` 续页（不重置）——此前 `cursor || undefined` 会把两者混淆。
- 游标改为组件内部维护（`advanceCursor`），不再要求 items 非空才能翻页。

验证（真实实例 + 浏览器）：seed 25 条 `#26` 笔记后 `/explore?tag=26`：初始 raw 页 20 条
（全部命中）→ 滚动 → 第 2 raw 页补齐其余 6 条（25 新 + 1 旧 = 26 条全部呈现），止于最老
笔记，无重复请求循环；`/explore?tag=ZZZ`（0 命中）流式来的无标签笔记被 `streamFilter`
拒绝（两次验证，浮标不出现）。

### Important 2 — 通知页对 main 频道的二次订阅杀死 store 的订阅（修复）

后端对 shareable 频道（main）同连接二次 connect 静默拒绝（`internal/streaming/connection.go`
`spec.Shareable` 去重路径已确认）：NotificationsView 的 PagedFeed 抢占了 main 槽位，
页面卸载发 disconnect 后 store 的 main 订阅随之死亡——离开 /notifications 或重连后全局
未读徽标冻结。

修复（`stores/streaming.ts` 重写为按频道引用计数）：
- `subscribe(channel)` 返回调用方 token；同一频道在线上只 connect 一次
  （`channelSubs` 按 channel 共享，token→共享子引用计数）。
- `unsubscribe(token)` 减计数，归零才发 disconnect 并拆除线上订阅。
- store 自持 main：`mainToken` 走同一 refcount；`dropMain()` 是真实 release（归零才拆），
  onopen 时 `if (token && !mainToken) mainToken = subscribe("main")` 重新断言——若组件仍
  持有则加入现有引用（不发重复 connect），若已归零则重建。
- 消息路由改由 wire-id → sub 映射（删除旧的 mainID 特判 fallback）。

验证（真实实例 + 浏览器）：登录态下 (a) /notifications 上流式反应 → 浮标 + 全局徽标 +1；
(b) 导航离开通知页后再来一条反应 → 徽标照常 +1（store 的 main 存活）；(c) kill 后端强制
断线 → 自动重连后再来一条 → 徽标照常 +1（重连后 main 重新断言生效）。

### Minor 修复

- 哨兵观察前先 `unobserve`（hasMore 翻转时的观察器重复注册）。
- 删除 PagedFeed 死样式 `.empty`。
- 删除 NotificationsView 死 `ref="feed"`。
- 并发 `load()` 序列化：`loadSeq` 序号守卫，重叠调用仅最新生效（重连/换 fetcher 竞态）。
- ExploreView + TimelineFeed 新增 `streamFilter`：tag 激活时流式浮标条目同样经客户端
  标签过滤（带 tag 页的浮标不再混入无标签笔记）。

### 回归确认

`pnpm typecheck` + `pnpm build` 通过；无标签时间线（/explore 本地/全球、/ 时间线）加载、
翻页、浮标、日期分隔行为与修复前一致。
