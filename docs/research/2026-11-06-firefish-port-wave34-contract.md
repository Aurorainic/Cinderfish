# 第三十四波施工契约：notes/state 与收藏菜单（2026-11-06）

笔记菜单此前无法知道收藏状态，只能永远显示「收藏」（重复收藏是幂等的，但不会取消）。
本波实现 `notes/state` 并让菜单先取状态再弹出。

## 1. 端点

`POST /api/notes/state`（需登录）请求 `{noteId}`，返回
`{isFavorited, isPinned, isMutedThread}`：

- 收藏查 `note_favorites`，置顶查 `user_profiles.pinned_note_ids`；
- 线程静音未实现，恒 `false`；
- 笔记不存在/已删除 404，缺 `noteId` 400。

## 2. 前端

| 文件 | 变化 |
|------|------|
| `api/types.ts`、`endpoints.ts` | `NoteState`、`notes/state` |
| `scripts/get-note-menu.ts` | 新增 `buildNoteMenu`：先取状态再生成菜单；收藏项按状态切换 create/delete，置顶项使用 `isPinned` |
| `components/MkNote.vue`、`MkNoteSub.vue`、`MkNoteDetailed.vue` | 菜单/右键菜单改用 `buildNoteMenu` |

状态请求失败或未登录时 `state` 为 `null`，菜单退回原来的行为。

## 3. 有意的限制

1. 每次打开菜单多一次 RPC；失败时静默退回。
2. 收藏状态只在打开菜单时刷新，操作后不就地更新（菜单已关闭）。
3. `isMutedThread` 恒 `false`，线程静音未实现。

## 4. 验收

`gofmt -l`、`go vet ./...`、`vue-tsc --noEmit`、`check:port`、`pnpm build` 通过。按用户
2026-11-06 的指示未执行 `go test`、数据库测试与浏览器走查。
