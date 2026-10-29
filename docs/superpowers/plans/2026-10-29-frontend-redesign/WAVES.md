# 并行波次派发表（swarm 执行）

取代 README-execution.md 的串行批次表。原则：**同一波内任务的文件所有权互不重叠**，可真正并行；波与波之间按依赖等待上一波合并。

所有任务仍以 `tasks/<任务号>.md` 为需求来源；本表只补充「波次、依赖、文件所有权、并行协作规则」。

## 波次

| 波 | 任务 | 说明 |
|----|------|------|
| 1 | B2→B3→B4（单个 agent 串行做三个后端任务）、F1、F3、F4、F8 | 后端与前端、以及四个前端任务文件互不相交 |
| 2 | F2、F6 | F2 独占 `main.css`/`App.vue`/`AppShell`；F6 独占笔记组件与调用点 |
| 3 | F5、F7 | F5 独占 deck 壳与 `App.vue`/`PageWindow`；F7 独占 `os.ts` 与 `NoteFooter` |
| 4 | F9、F10、F11 | F9 独占 `os.ts`/`PostForm`；F10 独占通知页；F11 独占个人页/设置页/client.ts |
| 5 | F12 | 全局文案与收尾，独占全仓 sweep |

## 文件所有权（同波内不得越界）

- B2/B3/B4：`internal/**`（含 `db.go` 迁移 v5）
- F1：`web/src/styles/main.css`（token + 全局类段）
- F3：`web/src/os.ts`、`web/src/components/ui/PageWindow.vue`、`web/src/App.vue`（仅挂载 os 容器）
- F4：`web/src/components/ui/PagedFeed.vue`、`web/src/components/TimelineFeed.vue`、`web/src/views/ExploreView.vue`、`web/src/views/NotificationsView.vue`
- F8：`web/src/lib/mfm.ts`、`web/src/styles/mfm.css`（新建，动效 keyframes 放这里）、`web/src/main.ts`（仅加 import）
- F2：`web/src/stores/shell.ts`、`web/src/components/ui/MobileNav.vue`、`web/src/components/widgets/InstanceCard.vue`、`web/src/layouts/AppShell.vue`、`web/src/App.vue`、`web/src/styles/main.css`（仅媒体查询段）
- F6：`web/src/components/note/**`、删除 `NoteCard.vue`/`QuotedNote.vue`、`web/src/components/TimelineFeed.vue`、`web/src/views/NoteView.vue`、`web/src/views/ProfileView.vue`
- F5：`web/src/stores/deck.ts`、`web/src/layouts/DeckShell.vue`、`web/src/components/deck/**`、`web/src/App.vue`、`web/src/components/ui/PageWindow.vue`、`web/src/layouts/AppShell.vue`（仅加切换按钮）
- F7：`web/src/lib/emoji-catalog.ts`、`web/src/components/ui/EmojiPicker.vue`、`web/src/os.ts`、`web/src/components/note/NoteFooter.vue`、`web/src/components/note/NoteContent.vue`（反应条）
- F9：`web/src/components/composer/PostForm.vue`、删除 `NoteComposer.vue`、`web/src/os.ts`、`web/src/components/TimelineFeed.vue`
- F10：`web/src/views/NotificationsView.vue`、`web/src/components/NotificationItem.vue`
- F11：`web/src/views/ProfileView.vue`、`web/src/views/SettingsView.vue`、`web/src/api/client.ts`、`web/src/components/note/NoteFooter.vue`
- F12：全仓 sweep（文案、残留视觉）+ `docs/HANDOVER.md`

## 并行协作规则（每个 agent 必读）

- 只在 `/home/lainsaka/文档/MilnFish/.worktrees/frontend-ui-redesign` 工作。
- 与其它 agent 共享同一工作树：**只 `git add` 你文件所有权内的具体路径，禁止 `git add -A` / `git commit -a`**；`git commit` 遇 `index.lock` 等 5 秒重试。
- 验证时若 `pnpm typecheck` 报的错在你文件所有权之外（别的 agent 正在写），等 20 秒重试，最多 3 次；仍失败照实上报，不要改别人的文件。
- `pnpm build` 并发写 `web/dist` 可能互相踩：失败且错误与本任务无关时 `rm -rf web/dist` 重试一次。
- 不新增 npm 依赖；不派发子代理。

## 评审

每波结束后按 `README-execution.md` 的评审模板并行派发评审者；实现者报告在 `reports/<任务号>-report.md`。
