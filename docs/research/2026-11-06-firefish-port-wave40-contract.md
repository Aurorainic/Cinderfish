# 第四十波施工契约：出站投递与取消转贴（2026-11-06）

本波的目标原本只是「转贴不能取消」。读代码时发现根因更大：**本地笔记从来没有投递出去过**。

`internal/httpapi/federation_wire.go` 里的 `DeliverNote` 与 `DeliverDelete` 全仓库只有定义、
没有任何调用点，`federation.RenderAnnounce` 也从未被使用。也就是说 `notes/create` 只写库、
只推 Streaming、只投 Webhook；远端关注者收不到任何东西。这直接违背「联邦是一等目标」。

## 1. 投递语义

| 触发 | 活动 | 收件人 |
|------|------|--------|
| `notes/create`（普通笔记） | `Create` | 远端关注者 + 被提及者 |
| `notes/create`（纯转贴） | `Announce` | 同上 |
| `notes/delete`（普通笔记） | `Delete` | 同上 |
| `notes/delete`（纯转贴） | `Undo(Announce)` | 同上 |
| `notes/unrenote` | `Undo(Announce)` | 同上 |

判定沿用 `note.IsPureRenote`（无正文/CW/回复指针/附件/投票）。`Announce` 的 `object` 是原笔记的
ActivityPub id：本地笔记取 `NoteRef(id)`，远端笔记取其 `uri`。

三个辅助函数：

- `deliverable(n)`：投递前置条件（投递器与渲染器就位、非 `localOnly`、非远端作者）；
- `deliveryInboxes(ctx, n)`：关注者收件箱 + 被提及者；**直发笔记只取可见用户**，不取关注者；
- `objectRef(n)`：本地/远端统一的 AP id，取不到时返回空串。

## 2. `notes/unrenote`

请求 `{noteId}`——**原笔记**的 id。服务端用 `note.Service.PureRenoteOf` 找回调用者那条纯转贴，
找不到回 **204**（幂等），找到则软删除 + 推 Streaming `deleted` + 发 `Undo(Announce)`。
引用（quote）不会被删：它带着作者自己的话。

## 3. 前端

`MkRenoteButton.vue` 在本次会话内完成过转贴后，菜单里出现「取消转发」（`t("unrenote")`），调用
`notes/unrenote` 并把按钮的已转贴标记复位。没有「这条笔记我是否转过」的查询端点，所以跨会话
状态无法判断——这条限制照实写进组件注释，不假装它存在。

## 4. 有意的限制

1. `notes/edit` 仍不发 `Update`（第三十三波只做了已编辑标记与历史）。
2. 远端原帖缺 `uri` 时，删除降级为 `Delete` 转贴本身（`Undo` 需要原对象）。
3. 投递失败只记日志，不回滚请求——沿用既有 `Deliverer` 的入队模型。
4. 没有为投递写新测试（用户指示本阶段不跑测试）。

## 5. 验收

```
gofmt -l internal/ cmd/                 # 空
go vet ./...                            # exit 0
cd web && pnpm exec vue-tsc --noEmit    # exit 0
cd web && pnpm run check:port           # hardTotal 0
cd web && pnpm build                    # exit 0
```

未执行 `go test`、数据库测试与浏览器走查。
