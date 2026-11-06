# 第三十波施工契约：账号导入（2026-11-06）

导入导出页此前明确写「导入没有端点」。本波实现 `i/import`，读取与 `i/export` 同一格式
（`cinderfish-account-export@1`）的 JSON。

## 1. 端点

`POST /api/i/import`（需登录，Misskey 同名端点），请求 `{fileId}`：

- 文件必须是当前账号自己的云盘文件，≤ 32 MiB，JSON 且 `format` 匹配；
- 单事务内按笔记 → 关注 → 收藏的顺序恢复；
- **笔记**保留原 id，冲突跳过；回复/转发目标不存在时置空；`createdAt`、标签、附件 id 保留；
- **关注**按 `username + host` 解析，找不到的用户跳过；已关注不重复；
- **收藏**仅在笔记存在时写入；
- 返回计数 `{notes, followings, favorites, skipped, skippedFollowings, skippedFavorites}`。

## 2. 前端

| 文件 | 变化 |
|------|------|
| `api/types.ts`、`endpoints.ts` | `ImportSummary`、`i/import` |
| `pages/SettingsImportExport.vue` | 文件选择 → 上传到云盘 → 调用 import → 展示导入/跳过计数 |

## 3. 有意的限制

1. 只接受本站导出格式，不支持 Misskey/Firefish 的 zip。
2. 不导入云盘文件字节（只有元数据在导出里），附件 id 可能指向不存在的文件。
3. 不导入他人内容、通知、注册表、私信。
4. 没有限流；大文件在内存中解析。

## 4. 验收

`gofmt -l`、`go vet ./...`、`vue-tsc --noEmit`、`check:port`、`pnpm build` 通过。按用户
2026-11-06 的指示未执行 `go test`、数据库测试与浏览器走查。
