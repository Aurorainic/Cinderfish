# 第五十九波施工契约（入站 Flag：远端举报落库）

日期：2026-11-06
状态：已落地，证据见 `docs/HANDOVER.md` §12BX
上游参考：Misskey `4ff2edf`：

- `packages/backend/src/core/activitypub/ApInboxService.ts`（`flag()`，L557-580：取第一个本地用户为
  举报对象，其余 URI 与 `content` 一起写进评论）

## 1. 这一波补什么

入站 `Flag` 是最后一个未处理的常用联邦活动（只剩 `Question` 投票）。没有它，远端实例举报本站账号
的行为会被当成未知活动丢弃，管理员永远看不到。

## 2. 实现

`Inbox.flag`：

1. 读取 `object`（单个 URI 或数组），逐个匹配 `<baseURL>/users/` 前缀
2. 第一个能解析成本地用户的 URI 成为举报对象（与 Misskey 一致：一个报告只对应一个 target）
3. 评论 = `content` + 换行 + 目标 URI 的 JSON 数组，这样笔记类目标与上下文不会丢失
4. 写入 `abuse_reports`，`reporter_id` 是远端账号在 `users` 表里的行，所以管理台的举报列表、
   详情与「已处理」流程都不需要改动

`Store.RecordReport` 负责插入，评论长度沿用表定义（无额外截断）。

## 3. 写作用域

| 文件 | 动作 |
|------|------|
| `internal/federation/inbox.go` | `perform` 分支 + `flag` 处理器 |
| `internal/federation/store.go` | `RecordReport` |
| `internal/httpapi/flag_test.go` | 端到端用例 |
| 台账 §6 | 未覆盖活动只剩 `Question` |

## 4. 有意的限制

- 不校验远端举报的频率，也不做「同一举报人重复举报同一账号」的去重：个人实例上管理台看得到，
  滥用时人工处理。Misskey 有报告阻塞表，Cinderfish 没有。
- 笔记级目标只作为评论保存，`abuse_reports` 没有 note 列，管理台只有链接文本。
- 不给管理员发通知/邮件：站内举报本来也没有，保持两边一致。

## 5. 验收口径

- `go vet ./...` 通过；`CINDERFISH_TEST_DATABASE_URL=… go test ./...` 全绿
- `TestInboundFlagCreatesReport`：双实例 + 真实签名。beta 的账号举报 alpha 的管理员账号后，
  alpha 的 `admin/abuse-user-reports` 出现一条未处理报告，评论含原文与目标 URI，
  reporter 是 betabob
