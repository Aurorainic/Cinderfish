# 第三十三波施工契约：笔记「已编辑」标记（2026-11-06）

第二十四波写入了 `notes.updated_at`，但响应没有打包、头部也没有标记。本波把这条链路补完。

## 1. 后端

- `note.Note` 新增 `UpdatedAt *time.Time`；`noteSelect`/`scanNote` 读取 `n.updated_at`；
- `packNote` 输出 `updatedAt`：RFC3339Nano 或 `null`；
- 所有走 `noteSelect` 的查询（时间线、会话、搜索、收藏、详情）自动带上该字段。

## 2. 前端

| 文件 | 变化 |
|------|------|
| `api/types.ts` | `Note.updatedAt: string \| null` |
| `components/MkNoteHeader.vue` | 编辑过时显示铅笔图标，`title` 用 `edited` 文案给出日期与时间 |

## 3. 有意的限制

1. 只有本地编辑会设置 `updatedAt`；远端 `Update` 活动仍未处理。
2. 没有「查看这次编辑」的入口，历史页在笔记菜单里。
3. `edited` 文案是 `zh-CN`/`ja-JP` 已有键，其余 locale 未生成。

## 4. 验收

`gofmt -l`、`go vet ./...`、`vue-tsc --noEmit`、`check:port`、`pnpm build` 通过。按用户
2026-11-06 的指示未执行 `go test`、数据库测试与浏览器走查。
