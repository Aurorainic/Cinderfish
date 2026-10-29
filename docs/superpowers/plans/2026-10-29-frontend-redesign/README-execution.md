# 前端重构执行手册（swarm 派发版）

配套计划：`docs/superpowers/plans/2026-10-29-frontend-redesign.md`
设计依据：`docs/superpowers/specs/2026-10-29-frontend-redesign-design.md`
任务文件：`tasks/B1.md` … `tasks/F12.md`（每个文件自包含：简报 + 上下文 + 验证 + 报告格式）

## 当前状态

| 任务 | 状态 |
|------|------|
| B1 笔记文件打包 | **已实现**（commit `1d6a54b`，含 note.Create 持久化 fileIds 的前置修复），**待评审** |

其余 15 个任务未开始。B1 的评审材料已生成：`.superpowers/sdd/2026-10-29-frontend-redesign/review-a9fb0da..1d6a54b.diff`（BASE=`a9fb0da`）。

## 执行批次（swarm 并行组）

同批次内的任务**文件互不冲突**，可并行派发；批次之间必须等待前一批全部完成并评审通过。

| 批次 | 任务 | 备注 |
|------|------|------|
| 0 | B1 评审（未完成） | 评审包已生成，直接用评审者模板 |
| 1 | B2、F1 | B2=后端 Go，F1=纯 main.css，无交集 |
| 2 | B3、F2 | B3 需 B2 的 UpdateProfile；F2 前端 |
| 3 | B4、F6 | B4 后端（含迁移 v5）；F6 消费 B1 的 files 字段 |
| 4 | F3、F4 | os.ts/PageWindow 与 PagedFeed 无交集 |
| 5 | F5 | deck 依赖 F2/F3/F4 |
| 6 | F7、F8 | picker 与 MFM 无交集 |
| 7 | F9 | 发帖器依赖 F3/F4/F7/F8 |
| 8 | F10、F11 | 通知页与个人页无交集 |
| 9 | F12 | 收尾，依赖全部 |

## 全局约束（每个任务派发时原样附带）

- 不新增 npm 依赖；后端不新增 API 端点（仅 `i/update`、`i/pin`、`i/unpin` 三个 + `users/notes` 加 `withFiles`/`withReplies` 参数）。
- 强调色保留 ember 橙 `#ff6b35`；字体维持 `Inter, "Noto Sans SC", system-ui`；图标维持手写 SVG（`AppIcon.vue`）。
- 数据库迁移不可变：v4 已应用，新改动进 v5。
- 文案用中文 + Misskey 术语（笔记/转贴/反应）；代码与标识符英文，docs 中文。
- 每个任务完成即提交；提交前必须：后端 `go vet ./...` + `go test ./...`（DB 测试带 `CINDERFISH_TEST_DATABASE_URL="postgres://cinderfish:cinderfish@localhost:5432/cinderfish_test"`，Redis 集成测试带 `CINDERFISH_TEST_REDIS_URL="redis://127.0.0.1:6379/0"`）；前端 `cd web && pnpm typecheck && pnpm build`。
- 笔记组件必须在 ~350px 窄列与 ~680px 主列都可用（container query）。
- 操作按钮 hover 显隐必须保持 `:focus-visible` 可见。

## 工作目录

所有任务在 **`/home/lainsaka/文档/MilnFish/.worktrees/frontend-ui-redesign`**（分支 `frontend-ui-redesign`）执行，不要在主仓操作。

## 实现者派发模板（每个任务复制填写）

```
You are implementing Task <任务号>: <标题>

Read your task brief first: /home/lainsaka/文档/MilnFish/.worktrees/frontend-ui-redesign/docs/superpowers/plans/2026-10-29-frontend-redesign/tasks/<任务号>.md
It is your requirements: the brief, controller context, verification commands, and report format.

## Context
[一句话：本任务在批次 N，依赖哪些已完成任务；若执行手册中该任务标了「依赖已就绪」则无需重复]

## Global Constraints
[粘贴上方「全局约束」整段]

## Working directory
/home/lainsaka/文档/MilnFish/.worktrees/frontend-ui-redesign

## Rules
- TDD when the brief lists test steps: failing test → verify FAIL → implement → verify PASS.
- Verify before committing with the commands in the brief; all must pass.
- Commit with the exact message in the brief. If `git commit` fails with `index.lock`, wait 5 seconds and retry (parallel agents may hold the index).
- Do not touch files outside your task's file list unless the brief says so; if you must, record it as a concern.
- Do not spawn subagents.
- If anything is ambiguous or you get stuck, report BLOCKED/NEEDS_CONTEXT with specifics instead of guessing.

## Report
Write your full report to /home/lainsaka/文档/MilnFish/.worktrees/frontend-ui-redesign/docs/superpowers/plans/2026-10-29-frontend-redesign/reports/<任务号>-report.md
(implemented what / tests + TDD evidence RED-GREEN / files changed / concerns),
then reply with ONLY: Status (DONE | DONE_WITH_CONCERNS | BLOCKED | NEEDS_CONTEXT), commits (SHA + subject), one-line test summary, concerns, report path.
```

## 评审者派发模板（每任务实现完成后）

1. 生成评审包（在 worktree 里）：
   ```bash
   BASE=<派发实现者之前记录的 HEAD>
   git log --oneline $BASE..HEAD > .review-$T.log
   git diff --stat $BASE..HEAD >> .review-$T.log
   git diff -U10 $BASE..HEAD >> .review-$T.diff
   ```
   （或直接用现成脚本：`bash .kimi-code/plugins/managed/superpowers/skills/subagent-driven-development/scripts/review-package docs/superpowers/plans/2026-10-29-frontend-redesign.md $BASE HEAD`）
2. 派发提示词：

```
You are reviewing one task's implementation (spec compliance + code quality). Read:
1. Brief: docs/superpowers/plans/2026-10-29-frontend-redesign/tasks/<任务号>.md
2. Report: docs/superpowers/plans/2026-10-29-frontend-redesign/reports/<任务号>-report.md
3. Diff: <评审包路径>（含 commit 列表、stat、完整 diff）

Global constraints: [粘贴全局约束]

Do not trust the report — verify claims against the diff. Do not re-run tests the implementer already ran. Read-only: do not mutate the working tree or branch state. Do not spawn subagents.

Output: 
### Spec Compliance (✅/❌ + ⚠️ cannot-verify items)
### Strengths
### Issues (Critical / Important / Minor, each with file:line)
### Assessment: Task quality [Approved | Needs fixes] + 1-2 sentence reasoning
```

## 评审后

- Critical/Important 发现 → 把发现派回**同一个实现者**（resume）修复，重新验证，然后对修复 diff 做一次小范围评审；最多 3 轮，仍不过则停在台账里记录裁决。
- 通过后在该任务文件顶部补一行：`> **状态：已完成**（commit <sha>..，评审通过）`，并在批次表打勾。

## 台账

进度记录在 `/home/lainsaka/文档/MilnFish/.worktrees/frontend-ui-redesign/.superpowers/sdd/2026-10-29-frontend-redesign/progress.md`（每完成一个任务追加 `Task <N>: complete (commits <base7>..<head7>, review clean)`）。台账文件在本目录之外被 git-ignore，如果丢了以 `git log` 为准。
