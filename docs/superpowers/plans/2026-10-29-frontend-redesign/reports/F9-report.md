# 任务 F9 完成报告：发帖器全局化 + 引用/草稿/specified

日期：2026-10-30 · 提交：`feat(web): global post form with quote, drafts, specified recipients`

## 实现内容

### 1. `web/src/components/composer/PostForm.vue`（新建，770 行）

从 `NoteComposer.vue`（317 行）迁移并扩展：

- props `{ replyTo?: Note | null; renoteTo?: Note | null; fixed?: boolean }`，emits `posted(note)` / `cancel()`。
- **引用**：`renoteTo` 时头部显示「引用 @user 的笔记」，正文下方渲染引用预览卡
  （`--renote` 左边框 + 头像 + 名字 + 两行截断摘要，无正文时显示「（仅附件）/（无正文）」），
  提交时一次 POST 带 `text` + `renoteId`。正文留空即退化为纯转贴（后端允许 text 为空）。
- **草稿**：`localStorage["cinderfish.draft"]`，`{ text, cw, visibility, localOnly, fileIds, files }`
  （`files` 是附件预览快照，提交只认 `fileIds`）。输入去抖 800ms 保存；空草稿删除键；
  卸载时 flush。打开时若有草稿，显示「有未发送的草稿 / 恢复 / 丢弃」横幅；发送成功即清。
  仅独立发帖器（无 reply/quote）读写草稿，回复/引用不会覆盖主草稿。
- **specified**：可见性改为图标+文字的四态分段控件（公开/首页/关注者/指定，`role="radiogroup"`
  + `role="radio"` + `aria-checked`），四个手写 SVG（AppIcon 没有 lock，四枚图标统一放在组件内）。
  选「指定」时出现收件人输入（`@handle, @user@host`，支持中英文逗号/顿号/空白分隔、去重、
  自动剥离开头 `@`）。提交前逐个 `api.usersShow({ username, host })` 解析为 userIds，
  **失败的行内红色报错**（如 `@nobody@remote.example Invalid param.`），并拒绝提交；
  成功后 `visibleUserIds` 传给 `createNote`。
- 其余保留：CW 折叠输入、附件上传/预览/移除、剩余字数、Ctrl/Cmd+Enter 提交、
  `@handle` 自动注入（**改为 `{ immediate: true }`**：对话框以 `replyTo` 已就绪的形态挂载，
  原先的非 immediate watcher 永不触发，导致回复对话框不带 @ 前缀 —— 实测发现并修复）。
- `fixed` 形态：作为时间线顶部固定条（去圆角/边框、底部分隔线、紧凑头像 34px、rows=3、
  不显示取消按钮）。组件自带 `container-type: inline-size`，`@container (max-width: 420px)`
  下分段按钮只留图标、缩略图 72px，350px deck 列可用。

### 2. `web/src/os.ts`（修改）

- stub 面板替换为真实 `PostForm`：`h(PostForm, { replyTo, renoteTo, onPosted, onCancel })`
  仍包在 `PageWindow` 中，窗口标题按上下文取「回复 / 引用 / 发帖」。
- `openPostForm({ reply?, renote?, quote? })` 签名与 `confirmDialog` 完全未动；
  `handleEscape` 的优先级保持 confirm → reaction picker → post form（逐字未改）。
- 删除 `stubBodyStyle` / `contextLines`（stub 专用），新增 `postFormTitle`。
- **`quote` 标志的语义**：`opts.renote` 一律作为 `renoteTo` 传入；后端「带正文=引用、
  无正文=纯转贴」是同一条 POST，所以 `quote` 不再改变行为（纯转贴走 NoteFooter 的直接转贴，
  F11 加引用入口时传 `quote: true` 即可）。这是对简报「quote=true 时带上正文」的等价实现。

### 3. `web/src/components/TimelineFeed.vue`（修改）

顶部插入 `<PostForm v-if="isLoggedIn" fixed />`；isLoggedIn 时新笔记经 streaming 频道回流，
无需手动 prepend（PagedFeed 也没有 prepend 接口）。内嵌 inline composer 早在 F4 已删除。

### 4. `web/src/components/NoteComposer.vue`（删除）

### 5. `web/src/views/NoteView.vue`（修改，**所有权外的最小必要改动，见 Concerns 1**）

删除 `NoteComposer` 引用（否则删组件后 `pnpm typecheck` 失败），回复改走
`@reply="openPostForm({ reply: $event })"`，移除 `replyTo` ref / `isLoggedIn` /
`.reply-composer` 样式。

## 后端验证：`renoteId + text` 被接受（无退回双条策略）

`internal/httpapi/notes.go:handleNotesCreate` 同时解码 `text` 与 `renoteId` 并一并交给
`note.Create`；`internal/note/service.go:132` 的空内容判定是
`p.Text == nil && len(p.FileIDs)==0 && p.Poll==nil && p.RenoteID==nil`，即「有 renote 目标时正文可选」；
唯一拒绝是 `target.RenoteID != nil`（不能转贴纯转贴，`ErrCannotRenoteToPureRenote`）。
**单条 POST 即引用，无需两条笔记策略。**

真实服务实测（`CINDERFISH_PORT=3117 go run ./cmd/cinderfish --config config.example.toml`，
本机 dev 库）：

```
POST /api/notes/create {"text":"引用评论","renoteId":"00mvtqqung9m230uqm"}
→ text: 引用评论 | renoteId: 00mvtqqung9m230uqm | packed renote: True
POST /api/users/show {"username":"f9target"} → 00mvtqqirjg5ywds3b f9target None
POST /api/notes/create {"visibility":"specified","visibleUserIds":["00mvtqqirjg5ywds3b"]}
→ vis: specified visibleUserIds: ['00mvtqqirjg5ywds3b']
```

## 验证

```bash
cd web && pnpm typecheck   # PASS（vue-tsc --noEmit，无输出）
cd web && pnpm build       # PASS（vue-tsc --noEmit && vite build，~410ms）
```

### 手动验证（真实后端 + 内置浏览器，非 Mock）

`f9tester` 为本次新建的临时用户（保留，与既有 `f4tester`/`f7tester` 一致），
`f9target` 用作指定收件人。测试笔记验证后已通过 API 软删除。

1. **固定形态**：首页时间线顶部出现发帖面板（头像 / 文本框 / CW / 四态可见性 / 仅本地 /
   计数 3000 / 发布），笔记流下移；`380px` 视口截图确认分段控件收为图标、无横向溢出。✅
2. **草稿**：在顶部面板输入「草稿测试：这条不该发出去」→ 等 2s → 刷新页面 → 出现
   「有未发送的草稿 / 恢复 / 丢弃」→ 点「恢复」→ 文本原样回到输入框。✅
3. **specified 解析**：切到「指定」→ 收件人填 `@f9target, @nobody@remote.example` →
   两行分别显示「将仅对这些用户可见」→ 点发布 → 行内变红 `@nobody@remote.example Invalid param.`
   + 顶部「部分收件人无法解析，请检查后重试。」，**未发帖**；改成 `@f9target` 后发布成功，
   DB 核对 `notes.visibility = specified` 且 `note_visibilities` 写入 `00mvtqqirjg5ywds3b`。✅
4. **回复**：笔记详情页点「回复」→ os 弹窗（PageWindow 标题「回复」）→ 文本框已带
   `@f9tester `、CW 展开 → 发布后弹窗关闭，DB 中 `reply_id` = 原笔记。✅
5. **引用**：为驱动 UI 临时在 NoteView 加了一个按钮调 `openPostForm({ renote, quote: true })`
   （验证后已还原，diff 中不含该按钮）→ 弹窗标题「引用」、上下文「引用 @f9tester 的笔记」、
   预览卡渲染 → 填写「F9 引用评论测试」发布 → DB 一条笔记同时有 `text` 与 `renote_id`，
   弹窗关闭，时间线以引用形态渲染该笔记。✅
6. **Esc**：全局发帖弹窗按 Esc 关闭；`os.ts` 的 confirm → picker → post form 顺序未改动。✅

> 过程中修掉两个只有跑起来才能发现的缺陷：回复对话框不带 @ 前缀（watcher 非 immediate）、
> 发布成功后弹窗不关闭（`h(PostForm, { posted })` 应为 `onPosted`，Vue 只认 `onX` 键）。

## 变更文件

- 新建 `web/src/components/composer/PostForm.vue`
- 修改 `web/src/os.ts`（stub → PostForm，其余不动）
- 修改 `web/src/components/TimelineFeed.vue`（顶部 `<PostForm fixed>`）
- 修改 `web/src/views/NoteView.vue`（所有权外，最小改动，见 Concerns 1）
- 删除 `web/src/components/NoteComposer.vue`

## Concerns

1. **删掉 `NoteComposer.vue` 必须同步改 `NoteView.vue`（`views/**` 不在我的所有权内）**：
   现有 `NoteView.vue` 直接 `import NoteComposer`，删组件后 typecheck 必红，且没有其他 agent
   在本波次拥有该文件（F6 已收工、F11 只管 Profile/Settings）。因此做了 2 行语义改动：
   回复改为 `openPostForm({ reply })`、移除 composer 与其样式。**请评审确认这个越界**；
   若判定不可接受，回退方式是把回复入口改回 inline composer，但简报要求删除该组件。
2. **草稿由多个实例共享一个键**：时间线固定发帖器和全局弹窗都是独立组件实例，都会读写
   `cinderfish.draft`。极端情况下（固定面板有草稿时再打开全局弹窗）会出现两个「恢复草稿」横幅，
   且后写入者覆盖前者。单实例场景（每次只用一个发帖入口）行为正确；若要彻底消除，
   需要把草稿状态提到 store 或给固定面板加「主发帖器」标记——都不在本任务文件所有权内。
3. **deck 每列都有发帖器**：`DeckColumn` 复用 `TimelineFeed`，因此每个时间线列顶部都会出现固定
   发帖面板（Misskey 只在主列）。简报要求「TimelineFeed 中用 `<PostForm fixed>`」，未加列级开关；
   如觉得吵，可在 F12 或后续给 `TimelineFeed` 加 `composer?: boolean` prop，由 `DeckColumn`
   只对首页列开启。
4. **`quote` 标志不再改变行为**（见实现说明 2）：能力上不缺失（空正文=纯转贴、有正文=引用），
   但若后续有调用方依赖 `quote: false` 表示「不要正文」则需要注意；真正要靠的是调用方是否传
   `renote`。函数签名未改动，向后兼容。
5. **指定收件人未做实时解析**：按简报只在提交前解析，输入过程中每行仅提示「将仅对这些用户可见」；
   若要输入即校验（debounce 调 users/show）属增强，未做（后端无 users/search，逐个解析成本高）。
6. **`posts` 通知/流转**：发布成功后固定面板不手动 prepend，依赖 streaming；在 WebSocket 断开
   或流式频道未匹配（如 tag 过滤的探索页）时新笔记不会立刻出现，需刷新或等「N 条新笔记」。
