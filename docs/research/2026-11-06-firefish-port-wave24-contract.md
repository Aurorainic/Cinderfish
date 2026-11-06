# 第二十四波施工契约：笔记编辑与编辑历史（2026-11-06）

笔记菜单此前没有编辑入口，`/notes/:id/history` 是空白占位页。本波实现 `notes/edit`（Firefish
端点）与 Cinderfish 自有的 `notes/history`，并补上编辑页与历史页。

## 1. 数据模型（迁移 v15）

```sql
ALTER TABLE notes ADD COLUMN updated_at TIMESTAMPTZ;
CREATE TABLE note_edits (
    id TEXT PRIMARY KEY,
    note_id TEXT NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    text TEXT, cw TEXT, file_ids TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

`note_edits` 保存**保存前**的文本/CW/附件 id；`notes.updated_at` 标记最近一次编辑
（响应里暂不打包，客户端的「已编辑」标记仍不显示）。

## 2. 端点的行为

`POST /api/notes/edit`（Firefish 契约）：

- 参数 `editId`（必填，缺 → 400 `NEEDS_EDIT_ID`）、`text`、`cw`、`visibility`、`localOnly`；
  其余字段静默忽略。
- 仅作者可改：他人笔记 → 403 `YOU_ARE_NOT_THE_AUTHOR`；不存在/已删 → 404 `NO_SUCH_NOTE`。
- 文本 trim 后为空 → `null`；超 3000 字符 → 400 `INVALID_PARAM`。
- `tags` 与本地 `mentions` 按新正文重算，`emojis` 清空；写一行 `note_edits`。
- 返回 `{"createdNote": Note}`。
- **不**发 Streaming `updated` 事件，**不**投递 ActivityPub `Update`。

`POST /api/notes/history`（Cinderfish 自有）：`noteId` + 可选 `limit`（默认 20、上限 100），
按 id 降序返回 `NoteEdit[]`；仅作者可读。

## 3. 前端

| 文件 | 变化 |
|------|------|
| `api/types.ts`、`endpoints.ts` | `NoteEditEntry`、`NoteEditResponse`、两个端点 |
| `router/routes.ts` | 新增 `/notes/:noteId/edit` |
| `pages/NoteEdit.vue`（新） | 正文 / 内容警告 / 可见性表单，保存后回到笔记页 |
| `pages/NoteHistory.vue` | 真实请求 `notes/history`，渲染正文、CW、时间；403 显示作者限制 |
| `scripts/get-note-menu.ts` | 作者菜单新增「编辑」与「编辑历史」 |

## 4. 有意的限制

1. 不能编辑附件、投票、回复/转发目标、指定可见用户。
2. 远端关注者不会收到 `Update`，其副本保持旧文本；本地转发也不会自动刷新。
3. 笔记响应不打包 `updatedAt`，因此界面上没有「已编辑」标记。
4. 没有 Misskey 的 300 次/小时限流。
5. `notes/history` 不是 Misskey 端点，第三方客户端不会调用它；它只为本站历史页服务。

## 5. 验收

`gofmt -l`、`go vet ./...`、`vue-tsc --noEmit`、`check:port`、`pnpm build` 通过。按用户
2026-11-06 的指示未执行 `go test`、数据库测试与浏览器走查；编辑→历史→回读链路留待统一验证。
