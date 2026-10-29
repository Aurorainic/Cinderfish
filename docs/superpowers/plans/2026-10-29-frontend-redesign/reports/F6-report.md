# Task F6 Report: Note 组件拆件 + 去卡片化

**Status: DONE（含两处需评审确认的范围说明）**
**Commit: `feat(web): split note into composable parts, borderless rows`**

## What was implemented

1. **`web/src/components/note/use-note.ts`（新建，composable）**
   - 从 `NoteCard.vue` 搬出全部 api 调用与乐观更新：`toggleReaction(emoji)`（逻辑与旧 `:67-96`
     等价，含切换反应时先撤销旧反应）、`renote()`、`remove()`，状态 `reactions`/`myReaction`/
     `reactionCount`/`busy`/`notice`（含 2s 自动消失的 `flash()`），另导出 `isMine`、
     `sortedReactions`。
   - `requireSignin()` 保留（401 前跳登录带 redirect）。
   - `note` prop 变化时重置反应状态（旧组件用 `watch(..., { immediate: true })`，此处一致）。
   - 返回类型相对简报的两处收窄（结构上兼容，react 返回 `Promise<Note | null>` 以便 `Note.vue`
     emit `renoted`；`remove()` 返回 `Promise<boolean>` 以便 `Note.vue` emit `deleted`），
     不新增任何全局状态。

2. **`NoteHeader.vue`（新建）** 头像 42px（nested 30px）、名字、`@handle`（远端带 `@host`）、
   相对时间（`detail` 态改用 `absoluteTime` 完整时间）、可见性图标。
3. **`NoteVisibility.vue`（新建，简报未列，为 DRY 抽出）** 手写 4 个 SVG：公开/首页/关注者/指定，
   `title` 同时给出中文可见性文案与「仅本地」；`NoteHeader` 与 `RenoteBar` 共用。
4. **`NoteContent.vue`（新建）** CW 横幅（`--cw-bg`/`--cw-fg`：未展开时为整条可点横幅 +
   「点击显示」，展开后为左侧色条 + 正文）、MFM 正文（`renderMFM` + 原有 `:deep(.mfm-*)` 样式）、
   媒体网格（转交 `NoteMedia`）、非图片附件胶囊、投票、引用占位 `<slot name="quote">`、
   反应胶囊条（`emit('react')`）。
5. **`NoteMedia.vue`（新建）** 1-4 图网格（3 图时首图跨两列），`thumbnailUrl || webpublicUrl || url`
   取样；任一附件 `isSensitive` 时整组模糊（`filter: blur(22px)`）+ 覆盖层「敏感内容 · 点击显示」，
   点击后显示；不实现 blurhash 解码（后端 `blurhash` 目前为 null）。
6. **`NoteFooter.vue`（新建）** 回复/转贴/反应/删除，`opacity: 0`，通过 `Note.vue` 的
   `.note:hover`/`.note:focus-within` 显现；自身保留 `:focus-within`/`:has(:focus-visible)`，
   键盘 Tab 时可见；`@media (hover: none)` 下常显（触屏无 hover）。反应调色板仍用旧
   `quickReactions` 常量（F7 用 EmojiPicker 替换）。
7. **`RenoteBar.vue`（新建）** 纯转贴顶条：转贴者 avatar(20px)/名字 + 「转贴了」+ 时间 + 可见性，
   全部 `--renote` 色系。
8. **`Note.vue`（新建，编排层）**
   - props `{ note, nested?, detail? }`，emits `reply`/`deleted`/`renoted`。
   - 纯转贴判定：有 `renote` 且无正文/CW/附件/投票 → `RenoteBar` + 自引用渲染 `note.renote`
     （未 nested，保留操作栏）；带正文的 renote 视为引用，在 `#quote` 槽里以 `nested` 态渲染
     （虚线左边框缩进、无底部分隔线、无操作栏）。
   - 回复上下文「回复 @user」；`detail` 态：完整时间、操作栏常显、无底部分隔线。
   - 样式：行式（左右 14px、无圆角无边框）、`1px var(--divider)` 分隔；每行自身
     `container-name: note; container-type: inline-size`，`@container note` 在 580/500/450px
     三档降级（内边距、头像与时间字号、handle 截断、正文行高、媒体最大高度、按钮密度）。
9. **删除** `web/src/components/NoteCard.vue`、`web/src/components/QuotedNote.vue`。
10. **调用点迁移**：`TimelineFeed.vue`、`NoteView.vue`（`detail`）、`ProfileView.vue`、
    `ExploreView.vue`（见 Concerns 第 1 条）全部改为 `<NoteItem>`（本地导入名 `NoteItem`，
    因为四个文件都已 `import type { Note }`，同名会 `TS2300 Duplicate identifier`）；
    列表 gap 归零，由 note 自身 hairline 分隔（`:deep(.paged-feed){gap:0}`）。

## 验证

```bash
cd web && pnpm typecheck   # PASS（vue-tsc --noEmit，无输出）
cd web && pnpm build       # PASS（vue-tsc --noEmit && vite build，346ms）
```

（在删除旧组件、hover 修复、`updated` 事件清理后各重跑一轮，均 PASS。）

### 手动验证（真实后端 + 内置浏览器，非 Mock）

临时起 `go run ./cmd/cinderfish`（dev DB `cinderfish`，registration=open）+ `pnpm dev`，
用 API 建测试账号 `f6tester`、上传 3 张图（其一置 `is_sensitive`）、用 API 建 6 条笔记
（双图 MFM / CW+敏感图 / 引用 / 纯转贴 / 回复 / `specified`），逐项截图确认：

1. **时间线（680px 主列）**：纯转贴顶条「转贴了 + 时间 + 可见性」+ 原笔记完整渲染；引用笔记
   虚线缩进块；回复上下文「回复 @f6tester」；行间只有 1px divider，无卡片边框/圆角。
2. **媒体网格**：2 图两列；CW 未展开为整条横幅「内容警告：剧透警告 / 点击显示」；
   敏感图模糊 + 「敏感内容 · 点击显示」覆盖层。
3. **hover 显现操作栏**：`NoteFooter` 默认 opacity:0，鼠标悬停行后回复/转贴/反应/删除出现
   （此步骤第一次验证时发现 `Note.vue` 缺 `.note:hover .foot` 规则，已补并复测通过）。
4. **详情页**：`/notes/<id>` 显示完整时间「2026/10/29 22:25:40」、操作栏常显、无 footer 折叠。
5. **Review Focus 4（specified）**：登录作者视角的首页时间线与详情页均正常渲染 specified 笔记，
   头部显示「指定」图标，无空白/无报错。
6. **反应**：点击反应 → 调色板 → ❤️，反应胶囊即时出现（accent 描边计数 1）、footer 计数同步为 1，
   与旧组件一致的乐观更新。
7. **窄列（380px 视口，container query 档位）**：`@container note (max-width: 450px)` 生效
   （内边距/字号/媒体高度收缩），无横向滚动，布局不挤。
8. **清理**：测试账号 `f6tester` 已删除（级联清掉测试笔记/反应/文件，`notes` 计数回到 37，
   `drive_files` 回到 0），临时后端进程、`/tmp/f6-*`、浏览器标签页均已清理。

未覆盖：`renote()` 与 `remove()` 的端到端点击验证在最后一轮时 dev server 因后台任务 600s
超时被杀，未能复测；两者实现与旧 `NoteCard` 完全一致（同一 `api.createNote`/`api.deleteNote`
与 `window.confirm` 流程），且 typecheck/build/事件接线均通过。

## TDD evidence

web 无测试套件（`package.json` 无 vitest），简报也未列测试步骤；验证为 typecheck + build +
上述真实实例手动矩阵，无 RED-GREEN 流程。

## Files changed

- 新建：`web/src/components/note/{Note,NoteHeader,NoteContent,NoteMedia,NoteFooter,RenoteBar,NoteVisibility}.vue`、
  `web/src/components/note/use-note.ts`
- 删除：`web/src/components/NoteCard.vue`、`web/src/components/QuotedNote.vue`
- 修改：`web/src/components/TimelineFeed.vue`、`web/src/views/NoteView.vue`、
  `web/src/views/ProfileView.vue`、`web/src/views/ExploreView.vue`

## Concerns / notes for reviewer

1. **超出 WAVES 文件所有权的两处**（均为“必须迁移才能删旧组件”的硬约束）：
   - `web/src/views/ExploreView.vue`（WAVES 记在 F4 名下）：F4 已提交且其中 `NoteCard` 用法是
     删除旧组件的唯一阻塞点；改动为机械替换 + `.explore-feed :deep(.paged-feed){gap:0}`，
     未触碰 fetcher/tabs。
   - 新增 `web/src/components/note/NoteVisibility.vue`（`note/**` 归我，但简报文件清单未列）：
     可见性图标在 `NoteHeader` 与 `RenoteBar` 都要用，抽件避免两份 SVG 漂移。
2. **接口相对简报的扩展/收窄**：
   - `Note.vue` emits 在简报的 `reply` 之外保留 `deleted`/`renoted`——`TimelineFeed`
     （`feed.remove`）、`ProfileView`（移除/前插）、`NoteView`（删除后跳走）都依赖。
   - 删除了旧 `NoteCard` 的 `updated` 事件：旧实现里它 emit 的是同一个被就地修改的对象，
     父级 `note = $event` 实际是空操作；反应状态现在完全由 `use-note` 持有，
     `NoteView` 的 `@updated` 一并去掉。
   - `use-note.renote()` 返回 `Promise<Note | null>`（简报写 `Promise<void>`），
     否则 `Note` 无法把新笔记交给 `ProfileView` 前插。
3. **名称冲突**：调用点用 `<NoteItem>`（导入名）而非 `<Note>`，因为四个文件都已
   `import type { Note }`；`Note.vue` 内部自引用仍写 `<Note>`（文件内 `import type { Note as NoteModel }`）。
4. **纯转贴的删除语义**：纯转贴行由外层（转贴本身）与内层（被转贴笔记）两个 `<article>` 组成，
   在内层点删除会 emit 内层 id，`PagedFeed.remove()` 找不到外层 id 因而不移除；与旧组件
   （无 RenoteBar、直接渲染 `note.renote` 但不提供删除）相比不算回归，但与 Misskey 行为有差异，
   可在后续把“删除纯转贴行的内层”改为删除外层。
5. **`NoteFooter` 的 `quickReactions` 仍在组件内**（旧 `NoteCard:32` 的行号搬家），
   按 WAVES 归 F7 用 EmojiPicker 替换；F7/F11 修改 `NoteFooter.vue` 时接口是
   props `{note, reactionCount, myReaction, busy, isMine}` + emits `reply|renote|react|remove`。
6. **container query 的三档**：580（内边距/时间字号/按钮密度）、500（头像间距/正文字号）、
   450（更紧内边距、`@handle` 保留但被截断、媒体高度收缩）。`NoteHeader` 首次实现曾在 <450
   隐藏 `@handle`，窄列实测后改为截断保留（身份信息在 350px deck 列仍可读）。
